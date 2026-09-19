//go:build !unix

package store

// acquireDataDirLock 在非 unix 平台不提供跨进程文件锁：Windows 需要 LockFileEx，
// 本项目当前不为单一用途引入额外依赖，故降级为 no-op（roadmap §5.1 R4 的已知残留），
// 单副本约束由部署方式保证（compose 不编排副本）。
func acquireDataDirLock(string) (func(), error) { return func() {}, nil }
