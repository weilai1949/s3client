//go:build !unix

package atomicfile

// fsyncDir 在非 unix 平台是 no-op：Windows 没有「打开目录做 fsync」的语义，
// FAT/exFAT 的目录项持久化由系统管理。与 lock_unix/lock_other 同样的平台分流，
// 保证 Windows 上编译与调用方不变（参数同 lock_other 一样不具名，避免未使用告警）。
func fsyncDir(string) error { return nil }
