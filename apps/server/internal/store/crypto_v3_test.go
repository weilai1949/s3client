package store

// crypto_v3_test.go —— 加密文件格式版本化（ASSESSMENT M2）。
//
// 背景：S3C2 信封只存 magic + salt，Argon2 参数硬编码在代码里（t=1），
// 直接调参会让既有加密库无法解密。因此新格式 S3C3 必须把 KDF 参数写进文件头，
// 读取时按文件里的参数派生密钥，并保留对 S3C2 旧库的读取能力（升级路径兼容）。

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

// buildLegacyS3C2 手工构造一个旧格式（S3C2，Argon2 t=1）加密文件，
// 用于验证新代码仍能读取升级前的既有加密库。
func buildLegacyS3C2(t *testing.T, password string, list []*model.Account) []byte {
	t.Helper()
	salt := randomSalt()
	plain := marshalAccounts(list)
	enc, err := encryptAESGCM(deriveKeyLegacy(password, salt), plain)
	if err != nil {
		t.Fatalf("encrypt legacy: %v", err)
	}
	return envelope(encMagicV2, salt, enc)
}

func TestEncryptedWritesS3C3WithParams(t *testing.T) {
	p := filepath.Join(t.TempDir(), "accounts.json.enc")
	s, err := NewEncrypted(p, "test-secret-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(encAtRestAcct()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 4 || string(raw[:4]) != string(encMagicV3) {
		t.Fatalf("expected S3C3 magic, got %q", raw[:min(4, len(raw))])
	}
	params, _, _, err := parseEnvelope(raw)
	if err != nil {
		t.Fatalf("parseEnvelope: %v", err)
	}
	if params.time < argonTimeV3 {
		t.Fatalf("S3C3 time cost = %d, want >= %d", params.time, argonTimeV3)
	}
	if params.memory != argonMemoryV3 || params.threads != argonThreadsV3 {
		t.Fatalf("S3C3 params = %+v, want memory=%d threads=%d", params, argonMemoryV3, argonThreadsV3)
	}
}

func TestEncryptedReadsLegacyS3C2(t *testing.T) {
	acct := encAtRestAcct()
	p := filepath.Join(t.TempDir(), "accounts.json.enc")
	blob := buildLegacyS3C2(t, "legacy-pw", []*model.Account{acct})
	if err := os.WriteFile(p, blob, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := NewEncrypted(p, "legacy-pw")
	if err != nil {
		t.Fatalf("NewEncrypted(legacy S3C2): %v", err)
	}
	got, err := s.Get(acct.ID)
	if err != nil || got.SecretKey != acct.SecretKey {
		t.Fatalf("get = %+v err=%v, want legacy secret preserved", got, err)
	}
}

func TestEncryptedReopenAfterV3Upgrade(t *testing.T) {
	p := filepath.Join(t.TempDir(), "accounts.json.enc")
	s, err := NewEncrypted(p, "pw")
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.Create(encAtRestAcct())
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewEncrypted(p, "pw")
	if err != nil {
		t.Fatalf("reopen S3C3: %v", err)
	}
	if got, _ := s2.Get(created.ID); got == nil || got.SecretKey != "sk-secret" {
		t.Fatalf("reopened store lost secret: %+v", got)
	}
}

// TestEnvelopeRoundTripParams 校验 parseEnvelope 对两种格式的解析与字段布局。
func TestEnvelopeRoundTripParams(t *testing.T) {
	salt := randomSalt()
	ct := []byte("ciphertext")
	v3 := envelope(encMagicV3, salt, ct)
	if !isEncryptedBlob(v3) {
		t.Fatal("v3 envelope not recognized")
	}
	params, gotSalt, _, err := parseEnvelope(v3)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotSalt, salt) {
		t.Fatalf("salt mismatch: %x vs %x", gotSalt, salt)
	}
	if params.time != argonTimeV3 || params.memory != argonMemoryV3 || params.threads != argonThreadsV3 {
		t.Fatalf("params = %+v", params)
	}
	// 头部长度固定：magic(4)+time(4)+memory(4)+threads(1)+salt(16)
	if want := 4 + 4 + 4 + 1 + encSaltLen; len(v3)-len(ct) != want {
		t.Fatalf("v3 header len = %d, want %d", len(v3)-len(ct), want)
	}
	// 用文件头里的参数应能派生出可解密的密钥。
	key := deriveKey("pw", gotSalt, params)
	if _, err := decryptAESGCM(key, ct[:0]); err == nil {
		// 空密文必然失败，这里只确认 deriveKey 可调用且长度正确。
		t.Fatal("unexpected")
	}
	if len(key) != keyLen {
		t.Fatalf("derived key len = %d, want %d", len(key), keyLen)
	}
}

// TestParseEnvelopeRejectsUnknown 未知/损坏格式必须报错而不是静默降级。
func TestParseEnvelopeRejectsUnknown(t *testing.T) {
	for _, blob := range [][]byte{
		nil,          // 空输入：连 magic 都不足（len < 4 分支）
		[]byte("S3"), // 短于 magic
		[]byte("nope"),
		append([]byte("S3C2"), make([]byte, 4)...),  // S3C2 但盐不足
		append([]byte("S3C3"), make([]byte, 2)...),  // S3C3 参数头不足
		append([]byte("S3C3"), make([]byte, 40)...), // 头部齐全但缺密文
	} {
		if _, _, _, err := parseEnvelope(blob); err == nil {
			t.Fatalf("parseEnvelope(%q) should fail", blob)
		}
	}
}

// TestParseEnvelopeRejectsZeroKDFParams S3C3 头部参数为 0 属损坏文件，必须报错
// （Argon2 参数为 0 会派生出弱密钥，静默接受等于降级加密强度）。
func TestParseEnvelopeRejectsZeroKDFParams(t *testing.T) {
	salt := randomSalt()
	for _, p := range []kdfParams{
		{time: 0, memory: argonMemoryV3, threads: argonThreadsV3},
		{time: argonTimeV3, memory: 0, threads: argonThreadsV3},
		{time: argonTimeV3, memory: argonMemoryV3, threads: 0},
	} {
		blob := append([]byte{}, encMagicV3...)
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], p.time)
		blob = append(blob, buf[:]...)
		binary.BigEndian.PutUint32(buf[:], p.memory)
		blob = append(blob, buf[:]...)
		blob = append(blob, p.threads)
		blob = append(blob, salt...)
		blob = append(blob, "ciphertext"...)
		if _, _, _, err := parseEnvelope(blob); err == nil {
			t.Fatalf("parseEnvelope with zero KDF param %+v should fail", p)
		}
	}
}

// TestDeriveKeyParamsDiffer 不同 KDF 参数必须产生不同密钥（参数确实生效）。
func TestDeriveKeyParamsDiffer(t *testing.T) {
	salt := randomSalt()
	weak := deriveKey("pw", salt, kdfParams{time: 1, memory: argonMemoryV3, threads: argonThreadsV3})
	strong := deriveKey("pw", salt, kdfParams{time: 2, memory: argonMemoryV3, threads: argonThreadsV3})
	if bytes.Equal(weak, strong) {
		t.Fatal("time cost change must alter the derived key")
	}
}

// TestS3C3HeaderEncoding 显式锁定头部字节序（大端），避免将来平台/实现漂移。
func TestS3C3HeaderEncoding(t *testing.T) {
	salt := randomSalt()
	blob := envelope(encMagicV3, salt, nil)
	if got := binary.BigEndian.Uint32(blob[4:8]); got != argonTimeV3 {
		t.Fatalf("time field = %d, want %d", got, argonTimeV3)
	}
	if got := binary.BigEndian.Uint32(blob[8:12]); got != argonMemoryV3 {
		t.Fatalf("memory field = %d, want %d", got, argonMemoryV3)
	}
	if got := blob[12]; got != argonThreadsV3 {
		t.Fatalf("threads field = %d, want %d", got, argonThreadsV3)
	}
}
