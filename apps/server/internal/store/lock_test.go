//go:build unix

package store

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDataDirLockExcludesSecondInstance 同一 DataDir 的第二个实例必须立刻失败，
// 释放后可以重新加锁（roadmap §5.1 R4：文件型 store + 内存 JobRegistry 只支持单副本）。
func TestDataDirLockExcludesSecondInstance(t *testing.T) {
	dir := t.TempDir()
	release, err := AcquireDataDirLock(dir)
	if err != nil {
		t.Fatalf("首次加锁失败: %v", err)
	}
	if _, err := AcquireDataDirLock(dir); err == nil {
		t.Fatal("第二次加锁应失败（同一 DataDir 只允许一个实例）")
	}
	release()

	release2, err := AcquireDataDirLock(dir)
	if err != nil {
		t.Fatalf("释放后应可重新加锁: %v", err)
	}
	release2()
}

// TestDataDirLockErrors 覆盖两条失败路径：数据目录不可创建、锁文件无法打开。
func TestDataDirLockErrors(t *testing.T) {
	// 父路径是普通文件 → MkdirAll 失败。
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireDataDirLock(filepath.Join(file, "data")); err == nil {
		t.Fatal("父路径非目录时 AcquireDataDirLock 应失败")
	}

	// 锁文件路径被同名目录占用 → OpenFile 失败。
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, dataDirLockFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireDataDirLock(dir); err == nil {
		t.Fatal("锁文件路径被目录占用时 AcquireDataDirLock 应失败")
	}
}
