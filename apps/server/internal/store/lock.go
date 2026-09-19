package store

import (
	"fmt"
	"os"
)

// AcquireDataDirLock 对数据目录加**单写者锁**：同一 DataDir 同时只允许一个 s3clinet 进程。
//
// 决定依据（docs/roadmap.md §5.1 R4）：store 是文件型的（json / sqlite），JobRegistry 在内存中，
// 单 token 模型没有租约或选主；两个副本共享同一数据卷会静默互相覆盖写入、重复执行迁移任务。
// 锁由内核在进程退出（含 panic / SIGKILL）时释放，不会留下需要手工清理的陈旧锁文件。
//
// 非 unix 平台当前降级为 no-op（见 lock_other.go），单副本约束仍由部署方式保证。
// 返回的 release 必须由调用方 defer；重复调用是安全的。
func AcquireDataDirLock(dataDir string) (release func(), err error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return acquireDataDirLock(dataDir)
}
