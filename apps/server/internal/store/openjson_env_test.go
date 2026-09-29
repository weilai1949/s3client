package store

// openjson_env_contract_test.go —— KNOWN_ISSUES #64 红灯用例文件。
// 删掉这两个测试，#64 一定假绿（fallback 路径本身也会被下面的改动误伤）。
//
// 它与 open_storekey_test.go 的分工：那边证明「入参生效」，
// 这边证明「入参为空时的 fallback 仍在」（防止有人把 PrioritizeExplicit 理解成 OnlyExplicit）。

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	envKey   = "env-key-abcdefghijklmnop"
	otherKey = "explicit-key-0123456789"
)

// TestOpenJSONEnvFallbackStillEncrypted 入参为空但环境变量非空 ⇒ 仍然加密。
func TestOpenJSONEnvFallbackStillEncrypted(t *testing.T) {
	t.Setenv("S3C_STORE_KEY", envKey)

	dir := t.TempDir()
	st, err := Open(dir, "json", "")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := st.Create(encAtRestAcct()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	assertEncryptedFile(t, filepath.Join(dir, "accounts.json"))

	if _, err := Open(dir, "json", otherKey); err == nil {
		t.Fatal("wrong key must fail to decrypt")
	}
}

// TestOpenJSONEnvFallbackKeyRecovers 环境变量那把 key 能把文件读回去。
func TestOpenJSONEnvFallbackKeyRecovers(t *testing.T) {
	t.Setenv("S3C_STORE_KEY", envKey)

	dir := t.TempDir()
	st, err := Open(dir, "json", "")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	created, err := st.Create(encAtRestAcct())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	back, err := Open(dir, "json", "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	got, err := back.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SecretKey != "sk-secret" {
		t.Fatalf("secret = %q", got.SecretKey)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "accounts.json"))
	if len(raw) == 0 {
		t.Fatal("accounts.json is empty")
	}
}
