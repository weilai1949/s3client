//go:build unix

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// dataDirLockFile 是数据目录下的锁文件；内容无关紧要，flock 由内核持有并在进程退出时释放。
const dataDirLockFile = ".s3clinet.lock"

// acquireDataDirLock 用 flock(LOCK_EX|LOCK_NB) 抢锁：第二个进程立刻失败而非阻塞等待。
func acquireDataDirLock(dataDir string) (func(), error) {
	f, err := os.OpenFile(filepath.Join(dataDir, dataDirLockFile), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open data dir lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("data dir %s is already in use by another s3clinet instance: %w", dataDir, err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
