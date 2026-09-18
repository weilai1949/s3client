package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

// setStoreKey 将 S3C_STORE_KEY 设为 key，并在测试结束时恢复原始值。
func setStoreKey(t *testing.T, key string) {
	t.Helper()
	t.Setenv("S3C_STORE_KEY", key)
}

// assertEncryptedFile 断言 path 的磁盘内容以 S3C2 魔数开头。
func assertEncryptedFile(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(raw) < 4 || string(raw[:4]) != string(encMagicV2) {
		t.Fatalf("file %s should start with S3C2 magic, got %q", path, raw)
	}
}

// assertPlaintextFile 断言 path 的磁盘内容不是加密文件（向后兼容明文落盘）。
func assertPlaintextFile(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.HasPrefix(string(raw), string(encMagicV2)) {
		t.Fatalf("file %s should be plaintext JSON, got S3C2 header", path)
	}
}

func encAtRestAcct() *model.Account {
	return &model.Account{
		ID:        "enc-id",
		Name:      "enc",
		Endpoint:  "http://127.0.0.1:9000",
		Region:    "us-east-1",
		AccessKey: "ak",
		SecretKey: "sk-secret",
		Bucket:    "b",
	}
}

// ---- newStoreCodec：严格模式决定盐策略与加密启停 ----

func TestStoreCodecConfig(t *testing.T) {
	t.Run("json permissive derives no key dependency", func(t *testing.T) {
		// json 无 key：明文 codec；missing 不生成盐。
		c := newStoreCodec("", false)
		if c.strict {
			t.Fatal("json codec must not be strict")
		}
		if err := c.missing(); err != nil {
			t.Fatalf("missing: %v", err)
		}
		if c.salt != nil {
			t.Fatal("json codec must not pre-generate salt")
		}
	})

	t.Run("encrypted strict generates salt on missing", func(t *testing.T) {
		c := newStoreCodec("pw", true)
		if !c.strict {
			t.Fatal("encrypted codec must be strict")
		}
		if err := c.missing(); err != nil {
			t.Fatalf("missing: %v", err)
		}
		if len(c.salt) != encSaltLen {
			t.Fatalf("encrypted codec salt len = %d, want %d", len(c.salt), encSaltLen)
		}
	})
}

// ---- Store.load：加密文件（S3C2）与明文向后兼容的检测与解密路径 ----

func TestStoreLoadEncryptAtRest(t *testing.T) {
	acct := encAtRestAcct()

	t.Run("file not exist with key set", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "accounts.json")
		setStoreKey(t, "key-A")
		s, err := New(p)
		if err != nil {
			t.Fatalf("New(nonexistent): %v", err)
		}
		if list, _ := s.List(); len(list) != 0 {
			t.Fatalf("expected empty store, got %d", len(list))
		}
	})

	t.Run("empty file with key set", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "accounts.json")
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		setStoreKey(t, "key-A")
		s, err := New(p)
		if err != nil {
			t.Fatalf("New(empty): %v", err)
		}
		if list, _ := s.List(); len(list) != 0 {
			t.Fatalf("expected empty store, got %d", len(list))
		}
	})

	t.Run("valid encrypted roundtrip", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "accounts.json")
		setStoreKey(t, "key-A")
		s, err := New(p)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if _, err := s.Create(acct); err != nil {
			t.Fatalf("Create: %v", err)
		}
		assertEncryptedFile(t, p)
		// 用同一 key 重开：应能解密成明文 JSON 并回填内存。
		s2, err := New(p)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		list, err := s2.List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 1 || list[0].Name != "enc" {
			t.Fatalf("list = %+v, want one account named enc", list)
		}
		full, err := s2.Get(acct.ID)
		if err != nil || full.SecretKey != "sk-secret" {
			t.Fatalf("get = %+v err=%v, want secret preserved", full, err)
		}
	})

	t.Run("wrong password fails decrypt", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "accounts.json")
		setStoreKey(t, "key-A")
		s, err := New(p)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if _, err := s.Create(acct); err != nil {
			t.Fatalf("Create: %v", err)
		}
		// 换成错误的 S3C_STORE_KEY 重开 → 派生密钥不同 → 解密失败。
		setStoreKey(t, "key-B")
		if _, err := New(p); err == nil || !strings.Contains(err.Error(), "decrypt account file") {
			t.Fatalf("New with wrong key = %v, want decrypt error", err)
		}
	})

	t.Run("too-short encrypted file", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "accounts.json")
		// 前缀合法但总长 < 4+encSaltLen。
		if err := os.WriteFile(p, []byte("S3C2"), 0o600); err != nil {
			t.Fatal(err)
		}
		setStoreKey(t, "key-A")
		if _, err := New(p); err == nil || !strings.Contains(err.Error(), "too short") {
			t.Fatalf("New(short) = %v, want too-short error", err)
		}
	})

	t.Run("plaintext stays plaintext despite key", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "accounts.json")
		// 明文 JSON（非 S3C2 魔数）在设置了 key 时必须按向后兼容明文解析。
		plain := `[{"id":"p1","name":"plain","endpoint":"e","accessKey":"ak","secretKey":"sk","region":"r"}]`
		if err := os.WriteFile(p, []byte(plain), 0o600); err != nil {
			t.Fatal(err)
		}
		setStoreKey(t, "key-A")
		s, err := New(p)
		if err != nil {
			t.Fatalf("New(plaintext): %v", err)
		}
		list, err := s.List()
		if err != nil || len(list) != 1 || list[0].ID != "p1" {
			t.Fatalf("list = %+v err=%v, want one plaintext account p1", list, err)
		}
	})
}

// ---- Store.persistLocked：key 非空时加密落盘、key 为空时明文落盘 ----

func TestStorePersistEncryptOnDisk(t *testing.T) {
	p := filepath.Join(t.TempDir(), "accounts.json")
	setStoreKey(t, "key")
	s, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	acct := encAtRestAcct()
	created, err := s.Create(acct)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertEncryptedFile(t, p)
	// 内部读回仍应看到真实密钥（内存不脱敏）。
	if full, _ := s.Get(created.ID); full.SecretKey != "sk-secret" {
		t.Fatalf("Create: mem secret = %q", full.SecretKey)
	}

	upd := &model.Account{ID: created.ID, Name: "enc2", Endpoint: "e", AccessKey: "ak"}
	if _, err := s.Update(created.ID, upd); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertEncryptedFile(t, p)

	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertEncryptedFile(t, p)
}

func TestStorePersistPlaintextOnDisk(t *testing.T) {
	p := filepath.Join(t.TempDir(), "accounts.json")
	setStoreKey(t, "")
	s, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(encAtRestAcct()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// 无 key → 明文 JSON 落盘（不出现 S3C2 头）。
	assertPlaintextFile(t, p)
}

// ---- sqlite Update 的 Exec 错误分支 ----
// getLocked(SELECT) 成功但 UPDATE 失败：BEFORE UPDATE 触发器 RAISE(FAIL)，
// 使 Exec 返回错误，命中 Update 的「sqlite update」错误分支。
func TestSQLiteUpdateExecError(t *testing.T) {
	st, err := openSQLite(filepath.Join(t.TempDir(), "accounts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	a, err := st.Create(gapAcc("u"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := st.db.Exec(`CREATE TRIGGER block_update BEFORE UPDATE ON accounts
		BEGIN SELECT RAISE(FAIL, 'blocked'); END`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	if _, err := st.Update(a.ID, gapAcc("u2")); err == nil {
		t.Fatal("Update should error when the UPDATE statement fails")
	}
}
