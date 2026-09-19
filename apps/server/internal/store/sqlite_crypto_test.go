package store

// sqlite_crypto_test.go —— SQLite 驱动的密钥落盘加密（ASSESSMENT M1）。
//
// 背景：sqlite 驱动的 `secret_key` 列此前明文落盘，且 docker-compose 默认就用该驱动。
// 现在：S3C_STORE_KEY 非空时，secret_key 列以 AES-256-GCM 密文（S3C3 参数）落盘；
// 空 key 时保持明文（向后兼容既有库，且不破坏无 key 的本地开发）。
//
// 加密是按「列值」做的（不是整库加密），因此既有明文行仍可读——读取时按魔数判别，
// 写回时统一加密。密钥不落库，只有 S3C_STORE_KEY 能解开。

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

// sqliteRawSecret 直接读库中 secret_key 列的原始值，用于断言落盘形态。
func sqliteRawSecret(t *testing.T, s *SQLiteStore, id string) string {
	t.Helper()
	var raw string
	if err := s.db.QueryRow(`SELECT secret_key FROM accounts WHERE id = ?`, id).Scan(&raw); err != nil {
		t.Fatalf("read raw secret_key: %v", err)
	}
	return raw
}

func TestSQLiteEncryptsSecretAtRest(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "accounts.db")
	s, err := openSQLite(dbPath, "store-key")
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.Create(&model.Account{
		Name: "enc", Endpoint: "http://127.0.0.1:9000", AccessKey: "ak",
		SecretKey: "sk-secret", Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	raw := sqliteRawSecret(t, s, created.ID)
	if raw == "sk-secret" {
		t.Fatal("secret_key stored in plaintext despite S3C_STORE_KEY")
	}
	if !strings.HasPrefix(raw, string(encMagicV3)) {
		t.Fatalf("secret_key should be an S3C3 blob, got %q", raw)
	}
	// 内存/Get 仍返回明文（服务端内部要用）。
	if got, _ := s.Get(created.ID); got.SecretKey != "sk-secret" {
		t.Fatalf("Get secret = %q, want plaintext in memory", got.SecretKey)
	}
}

func TestSQLiteDecryptsOnReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "accounts.db")
	s, err := openSQLite(dbPath, "store-key")
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.Create(&model.Account{
		Name: "enc", Endpoint: "http://127.0.0.1:9000", AccessKey: "ak", SecretKey: "sk-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	s2, err := openSQLite(dbPath, "store-key")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got, err := s2.Get(created.ID); err != nil || got.SecretKey != "sk-secret" {
		t.Fatalf("reopened Get = %+v err=%v", got, err)
	}
}

func TestSQLiteWrongKeyFailsToDecrypt(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "accounts.db")
	s, err := openSQLite(dbPath, "key-A")
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.Create(&model.Account{
		Name: "enc", Endpoint: "e", AccessKey: "ak", SecretKey: "sk-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	s2, err := openSQLite(dbPath, "key-B")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := s2.Get(created.ID); err == nil {
		t.Fatal("Get with wrong key must fail rather than return a corrupt secret")
	}
}

func TestSQLiteReadsLegacyPlaintextRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "accounts.db")
	// 先用无 key 写入（明文），模拟升级前的既有库。
	plain, err := openSQLite(dbPath, "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := plain.Create(&model.Account{
		Name: "legacy", Endpoint: "e", AccessKey: "ak", SecretKey: "sk-legacy",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := sqliteRawSecret(t, plain, created.ID); got != "sk-legacy" {
		t.Fatalf("no-key store should keep plaintext, got %q", got)
	}
	_ = plain.Close()

	// 升级：带 key 重开既有明文库，必须仍能读，并在写回时加密。
	upgraded, err := openSQLite(dbPath, "new-key")
	if err != nil {
		t.Fatalf("open legacy with key: %v", err)
	}
	if got, err := upgraded.Get(created.ID); err != nil || got.SecretKey != "sk-legacy" {
		t.Fatalf("legacy plaintext unreadable with key: %+v err=%v", got, err)
	}
	if _, err := upgraded.Update(created.ID, &model.Account{
		Name: "legacy2", Endpoint: "e", AccessKey: "ak", SecretKey: "sk-rotated",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	raw := sqliteRawSecret(t, upgraded, created.ID)
	if !strings.HasPrefix(raw, string(encMagicV3)) {
		t.Fatalf("updated row should be encrypted at rest, got %q", raw)
	}
}

func TestSQLiteEncryptedRoundTripViaOpen(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "sqlite", "top-key")
	if err != nil {
		t.Fatal(err)
	}
	sq, ok := st.(*SQLiteStore)
	if !ok {
		t.Fatalf("expected *SQLiteStore, got %T", st)
	}
	created, err := sq.Create(&model.Account{
		Name: "via-open", Endpoint: "e", AccessKey: "ak", SecretKey: "sk-open",
	})
	if err != nil {
		t.Fatal(err)
	}
	if raw := sqliteRawSecret(t, sq, created.ID); !strings.HasPrefix(raw, string(encMagicV3)) {
		t.Fatalf("Open with key must encrypt at rest, got %q", raw)
	}
}

// TestSQLiteDecryptSecretRequiresKey 库中已有密文但进程未配置 S3C_STORE_KEY 时必须报错，
// 不能把密文当明文返回（否则会拿密文去连 S3）。
func TestSQLiteDecryptSecretRequiresKey(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "accounts.db")
	enc, err := openSQLite(dbPath, "some-key")
	if err != nil {
		t.Fatal(err)
	}
	created, err := enc.Create(&model.Account{
		Name: "enc", Endpoint: "e", AccessKey: "ak", SecretKey: "sk-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = enc.Close()

	// 无 key 重开：解密路径必须失败，而不是把 S3C3 密文当明文用。
	noKey, err := openSQLite(dbPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := noKey.Get(created.ID); err == nil {
		t.Fatal("reading an encrypted row without S3C_STORE_KEY must fail")
	}
}

// TestSQLiteDecryptSecretRejectsCorruptBlob 带魔数但信封损坏的列值必须报解析错误。
func TestSQLiteDecryptSecretRejectsCorruptBlob(t *testing.T) {
	s := &SQLiteStore{storeKey: "k"}
	if _, err := s.decryptSecret(string(encMagicV3) + "xx"); err == nil {
		t.Fatal("corrupt S3C3 blob must fail to parse")
	}
}
