package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteFile 通过注入 OS 操作覆盖原子写全部分支。
// 真实 OS 上无法触发「写一半失败 / close 报错 / rename 失败」等错误路径，
// 借助包级钩子在测试里精确制造这些场景。
func TestWriteFile(t *testing.T) {
	dir := t.TempDir()

	// 备份并恢复注入点
	origOpen, origWrite, origSync, origClose, origChmod, origRename, origRemove :=
		atomicOpenTmp, atomicWrite, atomicSync, atomicClose, atomicChmod, atomicRename, atomicRemove
	t.Cleanup(func() {
		atomicOpenTmp, atomicWrite, atomicSync, atomicClose, atomicChmod, atomicRename, atomicRemove =
			origOpen, origWrite, origSync, origClose, origChmod, origRename, origRemove
	})

	t.Run("happy", func(t *testing.T) {
		p := filepath.Join(dir, "happy.bin")
		if err := WriteFile(p, []byte("hello")); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(p)
		if string(got) != "hello" {
			t.Fatalf("got %q", got)
		}
		if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("perm = %v err %v", fi.Mode().Perm(), err)
		}
	})

	t.Run("open fail", func(t *testing.T) {
		atomicOpenTmp = func(string) (*os.File, error) { return nil, errors.New("boom") }
		if err := WriteFile(filepath.Join(dir, "x"), []byte("a")); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("write fail", func(t *testing.T) {
		atomicOpenTmp = origOpen
		atomicWrite = func(*os.File, []byte) (int, error) { return 0, errors.New("disk full") }
		atomicRemove = origRemove
		p := filepath.Join(dir, "wf.bin")
		if err := WriteFile(p, []byte("x")); err == nil {
			t.Fatal("expected write error")
		}
		if _, err := os.Stat(p + ".tmp"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tmp should be cleaned, stat err = %v", err)
		}
	})

	t.Run("short write", func(t *testing.T) {
		atomicOpenTmp = origOpen
		atomicWrite = func(_ *os.File, data []byte) (int, error) { return len(data) - 1, nil }
		if err := WriteFile(filepath.Join(dir, "sw.bin"), []byte("xyz")); err == nil {
			t.Fatal("expected short write")
		}
	})

	t.Run("close fail", func(t *testing.T) {
		atomicOpenTmp = origOpen
		atomicWrite = origWrite
		atomicClose = func(*os.File) error { return errors.New("close err") }
		if err := WriteFile(filepath.Join(dir, "cf.bin"), []byte("x")); err == nil {
			t.Fatal("expected close error")
		}
		if _, err := os.Stat(filepath.Join(dir, "cf.bin.tmp")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tmp should be cleaned, stat err = %v", err)
		}
	})

	t.Run("rename fail", func(t *testing.T) {
		atomicOpenTmp = origOpen
		atomicWrite = origWrite
		atomicClose = origClose
		atomicRename = func(_, _ string) error { return errors.New("rename fail") }
		if err := WriteFile(filepath.Join(dir, "rf.bin"), []byte("x")); err == nil {
			t.Fatal("expected rename error")
		}
		if _, err := os.Stat(filepath.Join(dir, "rf.bin.tmp")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tmp should be cleaned, stat err = %v", err)
		}
	})

	// R11:数据必须在 rename 前真正落盘。写 hook 先关掉 fd 模拟 Sync 失败——
	// WriteFile 必须上抛错误并清理 tmp,而不是带着页缓存数据继续 rename。
	t.Run("sync fail", func(t *testing.T) {
		atomicOpenTmp = origOpen
		atomicClose = origClose
		atomicRename = origRename
		atomicRemove = origRemove
		atomicWrite = func(f *os.File, data []byte) (int, error) {
			if err := f.Close(); err != nil {
				return 0, err
			}
			return len(data), nil
		}
		p := filepath.Join(dir, "sync-fail.bin")
		if err := WriteFile(p, []byte("x")); err == nil {
			t.Fatal("expected sync error on closed fd")
		}
		if _, err := os.Stat(p + ".tmp"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tmp should be cleaned, stat err = %v", err)
		}
		atomicWrite = origWrite
	})

	// R11:rename 成功后必须 fsync 父目录;hook 在真 rename 后删掉目录,
	// 使 fsyncDir 打不开目录 → WriteFile 必须把该错误上抛。
	t.Run("fsync dir fail", func(t *testing.T) {
		atomicOpenTmp = origOpen
		atomicWrite = origWrite
		atomicClose = origClose
		atomicRemove = origRemove
		sub := filepath.Join(dir, "fsyncdir")
		if err := os.Mkdir(sub, 0o700); err != nil {
			t.Fatal(err)
		}
		atomicRename = func(tmp, dst string) error {
			if err := os.Rename(tmp, dst); err != nil {
				return err
			}
			return os.RemoveAll(filepath.Dir(dst))
		}
		if err := WriteFile(filepath.Join(sub, "fd.bin"), []byte("x")); err == nil {
			t.Fatal("expected fsync dir error")
		}
		atomicRename = origRename
	})

	// R11:chmod(tmp) 失败必须中止写入并清理 tmp——不能把权限失控的文件
	// rename 成正式账号文件(正式文件权限一旦是 0644,SecretKey 即对其他用户可读)。
	t.Run("chmod fail", func(t *testing.T) {
		atomicOpenTmp = origOpen
		atomicWrite = origWrite
		atomicSync = origSync
		atomicClose = origClose
		atomicRename = origRename
		atomicRemove = origRemove
		atomicChmod = func(string, os.FileMode) error { return errors.New("chmod err") }
		p := filepath.Join(dir, "chmod-fail.bin")
		if err := WriteFile(p, []byte("x")); err == nil {
			t.Fatal("expected chmod error")
		}
		if _, err := os.Stat(p + ".tmp"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tmp should be cleaned, stat err = %v", err)
		}
		atomicChmod = origChmod
	})
}

// TestWriteFileContentAndNoStale R11:写入后读回内容必须完整一致,且旧内容不残留
// (先写 64KiB+ 再写短内容,任何截断/复用旧文件的实现都会读到旧尾巴)。
func TestWriteFileContentAndNoStale(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.bin")
	long := make([]byte, 64*1024+7) // 超过一页,覆盖跨页写
	for i := range long {
		long[i] = byte('a' + i%26)
	}
	if err := WriteFile(p, long); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteFile(p, []byte("short")); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "short" {
		t.Fatalf("read back %d bytes %q, want exact new content %q", len(got), got, "short")
	}
	if _, err := os.Stat(p + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tmp residue left behind, stat err = %v", err)
	}
	if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %v err %v, want 0600", fi, err)
	}
}
