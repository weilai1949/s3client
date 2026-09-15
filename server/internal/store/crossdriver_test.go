package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/server/internal/model"
)

// driverSpec 描述一个「文件落盘」驱动的构造方式，用于跨驱动对照测试。
// 三个驱动共享同一份 CRUD 行为契约，任何差异都应视为回归。
type driverSpec struct {
	name string
	file string
	open func(t *testing.T, dir string) (AccountStore, error)
}

func driverSpecs() []driverSpec {
	return []driverSpec{
		{
			name: "json-plaintext",
			file: "accounts.json",
			open: func(t *testing.T, dir string) (AccountStore, error) {
				setStoreKey(t, "")
				return New(filepath.Join(dir, "accounts.json"))
			},
		},
		{
			name: "json-legacy-encrypted",
			file: "accounts.json",
			open: func(t *testing.T, dir string) (AccountStore, error) {
				setStoreKey(t, "cross-driver-key")
				return New(filepath.Join(dir, "accounts.json"))
			},
		},
		{
			name: "encrypted",
			file: "accounts.json.enc",
			open: func(t *testing.T, dir string) (AccountStore, error) {
				return NewEncrypted(filepath.Join(dir, "accounts.json.enc"), "cross-driver-key")
			},
		},
	}
}

func crossDriverAccount(name string) *model.Account {
	return &model.Account{
		Name:      name,
		Endpoint:  "http://127.0.0.1:9000",
		Region:    "us-east-1",
		AccessKey: "ak",
		SecretKey: "sk",
		Bucket:    "b",
	}
}

// crudObservation 是一次完整 CRUD 序列的可观测结果，已剔除随机 ID / 时间戳，
// 使三个驱动的结果可以直接比较。
type crudObservation struct {
	createMaskedSecret  bool
	createHasID         bool
	createHasTimestamps bool
	listNames           []string
	listAllMasked       bool
	getSecret           string
	dupRejected         bool
	dupMentionsID       bool
	updateName          string
	updateSecretKept    string
	updateMaskedKept    string
	updateNewSecret     string
	updateIDKept        bool
	updateCreatedKept   bool
	updateMissingCode   string
	deleteMissingCode   string
	getAfterDelete      string
	namesAfterDelete    []string
	reopenNames         []string
	reopenSecret        string
}

func observeCRUD(t *testing.T, spec driverSpec) crudObservation {
	t.Helper()
	dir := t.TempDir()
	st, err := spec.open(t, dir)
	if err != nil {
		t.Fatalf("%s: open: %v", spec.name, err)
	}

	var o crudObservation

	created, err := st.Create(crossDriverAccount("alpha"))
	if err != nil {
		t.Fatalf("%s: create alpha: %v", spec.name, err)
	}
	o.createMaskedSecret = created.SecretKey == model.MaskedSecret
	o.createHasID = created.ID != ""
	o.createHasTimestamps = !created.CreatedAt.IsZero() && !created.UpdatedAt.IsZero()

	second, err := st.Create(crossDriverAccount("beta"))
	if err != nil {
		t.Fatalf("%s: create beta: %v", spec.name, err)
	}

	list, err := st.List()
	if err != nil {
		t.Fatalf("%s: list: %v", spec.name, err)
	}
	o.listAllMasked = true
	for _, a := range list {
		o.listNames = append(o.listNames, a.Name)
		if a.SecretKey != model.MaskedSecret {
			o.listAllMasked = false
		}
	}

	full, err := st.Get(created.ID)
	if err != nil {
		t.Fatalf("%s: get: %v", spec.name, err)
	}
	o.getSecret = full.SecretKey

	dup := crossDriverAccount("alpha")
	dup.ID = created.ID
	_, dupErr := st.Create(dup)
	if dupErr != nil {
		o.dupRejected = true
		o.dupMentionsID = strings.Contains(dupErr.Error(), created.ID)
	}

	// 不提供 SecretKey 的更新：密钥保留、id/createdAt 不变。
	updated, err := st.Update(created.ID, &model.Account{Name: "alpha-renamed", Endpoint: "http://127.0.0.1:9000", AccessKey: "ak"})
	if err != nil {
		t.Fatalf("%s: update: %v", spec.name, err)
	}
	o.updateName = updated.Name
	afterUpdate, err := st.Get(created.ID)
	if err != nil {
		t.Fatalf("%s: get after update: %v", spec.name, err)
	}
	o.updateSecretKept = afterUpdate.SecretKey
	o.updateIDKept = afterUpdate.ID == created.ID
	o.updateCreatedKept = afterUpdate.CreatedAt.Equal(created.CreatedAt)

	// 掩码更新：不得覆盖真实密钥。
	if _, err := st.Update(created.ID, &model.Account{Name: "alpha-renamed", SecretKey: model.MaskedSecret, AccessKey: "ak"}); err != nil {
		t.Fatalf("%s: masked update: %v", spec.name, err)
	}
	masked, _ := st.Get(created.ID)
	o.updateMaskedKept = masked.SecretKey

	// 显式新密钥：应被采纳。
	if _, err := st.Update(created.ID, &model.Account{Name: "alpha-renamed", SecretKey: "sk-new", AccessKey: "ak"}); err != nil {
		t.Fatalf("%s: secret update: %v", spec.name, err)
	}
	rotated, _ := st.Get(created.ID)
	o.updateNewSecret = rotated.SecretKey

	if _, err := st.Update("missing-id", crossDriverAccount("x")); errors.Is(err, ErrNotFound) {
		o.updateMissingCode = "not_found"
	}
	if err := st.Delete("missing-id"); errors.Is(err, ErrNotFound) {
		o.deleteMissingCode = "not_found"
	}

	if err := st.Delete(second.ID); err != nil {
		t.Fatalf("%s: delete: %v", spec.name, err)
	}
	if _, err := st.Get(second.ID); errors.Is(err, ErrNotFound) {
		o.getAfterDelete = "not_found"
	}
	rest, _ := st.List()
	for _, a := range rest {
		o.namesAfterDelete = append(o.namesAfterDelete, a.Name)
	}

	// 重开：磁盘格式可被同配置再次解析，顺序保持。
	reopened, err := spec.open(t, dir)
	if err != nil {
		t.Fatalf("%s: reopen: %v", spec.name, err)
	}
	reList, err := reopened.List()
	if err != nil {
		t.Fatalf("%s: reopen list: %v", spec.name, err)
	}
	for _, a := range reList {
		o.reopenNames = append(o.reopenNames, a.Name)
	}
	reFull, err := reopened.Get(created.ID)
	if err != nil {
		t.Fatalf("%s: reopen get: %v", spec.name, err)
	}
	o.reopenSecret = reFull.SecretKey

	switch spec.name {
	case "json-plaintext":
		assertPlaintextFile(t, filepath.Join(dir, spec.file))
	default:
		assertEncryptedFile(t, filepath.Join(dir, spec.file))
	}
	return o
}

// TestCrossDriverCRUDEquivalence 同一 CRUD 序列在三个文件驱动上必须产生完全一致的可观测结果。
func TestCrossDriverCRUDEquivalence(t *testing.T) {
	specs := driverSpecs()
	observations := make([]crudObservation, len(specs))
	for i, spec := range specs {
		observations[i] = observeCRUD(t, spec)
	}
	for i := 1; i < len(specs); i++ {
		if !reflect.DeepEqual(observations[0], observations[i]) {
			t.Fatalf("driver %s diverges from %s:\n got  %+v\n want %+v",
				specs[i].name, specs[0].name, observations[i], observations[0])
		}
	}
}

// TestCrossDriverCreateIsolatesCallerInput Create 之后调用方再修改入参对象，
// 不得改变已落库账号（三个驱动语义必须一致）。Store 原先把调用方指针直接存进内存，
// 与 encrypted 驱动的「存副本」行为不一致——此测试即为该差异的回归保护。
func TestCrossDriverCreateIsolatesCallerInput(t *testing.T) {
	for _, spec := range driverSpecs() {
		t.Run(spec.name, func(t *testing.T) {
			dir := t.TempDir()
			st, err := spec.open(t, dir)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			input := crossDriverAccount("demo")
			created, err := st.Create(input)
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			// 调用方在 Create 之后复用/修改入参。
			input.Name = "mutated-after-create"
			input.SecretKey = "leaked-after-create"

			got, err := st.Get(created.ID)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if got.Name != "demo" || got.SecretKey != "sk" {
				t.Fatalf("stored account changed by caller mutation: name=%q secret=%q", got.Name, got.SecretKey)
			}
			list, _ := st.List()
			if len(list) != 1 || list[0].Name != "demo" {
				t.Fatalf("list affected by caller mutation: %+v", list)
			}
		})
	}
}

// TestCrossDriverFilesOnDisk 固定各驱动的落盘文件名与加密格式（供 Open 保持兼容）。
func TestCrossDriverFilesOnDisk(t *testing.T) {
	dir := t.TempDir()
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}

	setStoreKey(t, "")
	st, err := New(filepath.Join(dir, "accounts.json"))
	if err != nil {
		t.Fatalf("plaintext open: %v", err)
	}
	if _, err := st.Create(crossDriverAccount("p")); err != nil {
		t.Fatalf("plaintext create: %v", err)
	}
	if !exists("accounts.json") {
		t.Fatal("json driver must write accounts.json")
	}
	if _, err := os.Stat(filepath.Join(dir, "accounts.json.enc")); !os.IsNotExist(err) {
		t.Fatal("json driver must not create accounts.json.enc")
	}

	enc, err := NewEncrypted(filepath.Join(dir, "accounts.json.enc"), "k")
	if err != nil {
		t.Fatalf("encrypted open: %v", err)
	}
	if _, err := enc.Create(crossDriverAccount("e")); err != nil {
		t.Fatalf("encrypted create: %v", err)
	}
	assertEncryptedFile(t, filepath.Join(dir, "accounts.json.enc"))
}

// TestJSONDriverReadsLegacyS3C2Envelope 固定 json 驱动的向后兼容：历史版本写出的
// S3C2 加密文件（magic || salt || GCM(nonce||ct)，密钥 = Argon2id(S3C_STORE_KEY, 文件盐)）
// 必须仍能被 New 读取，不得因重构而静默丢弃旧数据。
func TestJSONDriverReadsLegacyS3C2Envelope(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "accounts.json")
	plain := []byte(`[{"id":"legacy-1","name":"old","endpoint":"http://127.0.0.1:9000","accessKey":"ak","secretKey":"sk","region":"us-east-1"}]`)
	writeEnvelope(t, p, "key-A", plain, false)

	setStoreKey(t, "key-A")
	s, err := New(p)
	if err != nil {
		t.Fatalf("New(legacy S3C2): %v", err)
	}
	got, err := s.Get("legacy-1")
	if err != nil || got.Name != "old" || got.SecretKey != "sk" {
		t.Fatalf("legacy read = %+v err=%v, want name=old secret=sk", got, err)
	}
}

// TestJSONDriverPlaintextByteFormat 固定无 key 时的落盘字节：缩进 JSON 数组，无信封头。
func TestJSONDriverPlaintextByteFormat(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "accounts.json")
	setStoreKey(t, "")
	s, err := New(p)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	created, err := s.Create(crossDriverAccount("fmt"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	full, _ := s.Get(created.ID)
	want, _ := json.MarshalIndent([]*model.Account{full}, "", "  ")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(raw) != string(want) {
		t.Fatalf("plaintext bytes mismatch:\n got %s\nwant %s", raw, want)
	}
}

// TestJSONDriverEncryptedByteFormat 固定有 key 时的信封布局与密钥派生：
// magic(4) || salt(16) || GCM(nonce||ct)，key = Argon2id(S3C_STORE_KEY, 文件盐)。
func TestJSONDriverEncryptedByteFormat(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "accounts.json")
	setStoreKey(t, "key-A")
	s, err := New(p)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	created, err := s.Create(crossDriverAccount("fmt"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(raw) < 4+encSaltLen || string(raw[:4]) != string(encMagicV2) {
		t.Fatalf("missing S3C2 envelope: %q", raw)
	}
	salt := raw[4 : 4+encSaltLen]
	plain, err := decryptAESGCM(deriveKey("key-A", salt), raw[4+encSaltLen:])
	if err != nil {
		t.Fatalf("envelope does not decrypt with file salt: %v", err)
	}
	full, _ := s.Get(created.ID)
	want, _ := json.MarshalIndent([]*model.Account{full}, "", "  ")
	if string(plain) != string(want) {
		t.Fatalf("decrypted payload mismatch:\n got %s\nwant %s", plain, want)
	}
}

// TestEncryptedDriverReusesFileSalt 固定 encrypted 驱动的「随机每文件盐」：
// 同一文件多次写盘复用同一盐（只在建文件时随机一次）。
func TestEncryptedDriverReusesFileSalt(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "accounts.json.enc")
	s, err := NewEncrypted(p, "pw-long")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	a, err := s.Create(crossDriverAccount("a"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	first, _ := os.ReadFile(p)
	if _, err := s.Update(a.ID, &model.Account{Name: "b", AccessKey: "ak"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	second, _ := os.ReadFile(p)
	if string(first[4:4+encSaltLen]) != string(second[4:4+encSaltLen]) {
		t.Fatal("encrypted driver must reuse the file salt across writes")
	}
}
