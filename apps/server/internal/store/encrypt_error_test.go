package store

// encrypt_error_test.go —— 覆盖 KNOWN_ISSUES #83：加密编码失败必须上抛，
// 绝不能吞掉后产出损坏信封/列值。encryptAESGCM 的失败只可能来自密钥长度非法
// （deriveKey 恒 32 字节，生产不可达），故经 encryptAESGCMFn 注入制造。

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestStoreCodecEncodeErrorPropagates 两种驱动的 encode 都必须把加密错误上抛。
func TestStoreCodecEncodeErrorPropagates(t *testing.T) {
	orig := encryptAESGCMFn
	t.Cleanup(func() { encryptAESGCMFn = orig })
	boom := errors.New("encrypt boom")
	encryptAESGCMFn = func([]byte, []byte) ([]byte, error) { return nil, boom }

	t.Run("strict（encrypted）", func(t *testing.T) {
		c := newStoreCodec("key", true)
		if err := c.missing(); err != nil {
			t.Fatalf("missing: %v", err)
		}
		if _, err := c.encode(nil); !errors.Is(err, boom) {
			t.Fatalf("encode err = %v, want boom", err)
		}
	})
	t.Run("permissive（json + key）", func(t *testing.T) {
		if _, err := newStoreCodec("key", false).encode(nil); !errors.Is(err, boom) {
			t.Fatalf("encode err = %v, want boom", err)
		}
	})
}

// TestFileStorePersistPropagatesEncodeError Create 落盘时 encode 失败必须回滚并报错。
func TestFileStorePersistPropagatesEncodeError(t *testing.T) {
	orig := encryptAESGCMFn
	t.Cleanup(func() { encryptAESGCMFn = orig })
	s, err := NewEncrypted(filepath.Join(t.TempDir(), "accounts.json.enc"), "key")
	if err != nil {
		t.Fatalf("NewEncrypted: %v", err)
	}
	boom := errors.New("encode boom")
	encryptAESGCMFn = func([]byte, []byte) ([]byte, error) { return nil, boom }
	if _, err := s.Create(sample()); !errors.Is(err, boom) {
		t.Fatalf("Create err = %v, want boom", err)
	}
	if got, _ := s.List(); len(got) != 0 {
		t.Fatalf("落盘失败必须回滚内存状态, got %d", len(got))
	}
}

// TestSQLiteEncryptSecretErrorPropagates Create / Update 的列加密失败必须上抛。
func TestSQLiteEncryptSecretErrorPropagates(t *testing.T) {
	orig := encryptAESGCMFn
	t.Cleanup(func() { encryptAESGCMFn = orig })
	s, err := openSQLite(filepath.Join(t.TempDir(), "accounts.db"), "key")
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	boom := errors.New("encrypt boom")
	encryptAESGCMFn = func([]byte, []byte) ([]byte, error) { return nil, boom }
	if _, err := s.Create(sample()); !errors.Is(err, boom) {
		t.Fatalf("Create err = %v, want boom", err)
	}

	// 先正常建一条，再让 Update 的加密失败。
	encryptAESGCMFn = orig
	created, err := s.Create(sample())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	encryptAESGCMFn = func([]byte, []byte) ([]byte, error) { return nil, boom }
	if _, err := s.Update(created.ID, sample()); !errors.Is(err, boom) {
		t.Fatalf("Update err = %v, want boom", err)
	}
}
