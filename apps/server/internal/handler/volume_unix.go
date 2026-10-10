//go:build linux || darwin || freebsd

package handler

import "syscall"

// volumeUsage 返回 path 所在文件系统的总容量与可用字节（statfs）。
// 调用方保证 path 非空；statfs 失败（路径不存在、权限不足等）返回 ok=false，
// 调用方据此**不输出**容量序列——没有可信数据就不发序列，不用 0 冒充「容量为 0」。
//
// 构建标签取显式三平台而非 `unix`：OpenBSD 的 `syscall.Statfs_t` 字段名是
// `F_bsize` / `F_blocks` / `F_bavail`（与本实现不同），NetBSD / DragonFly 则连
// `modernc.org/sqlite` 都编不过（依赖侧既有缺口）。这三者归 volume_other.go
// 的「不支持」分支，运维按 OPERATIONS.md §4.3 外部采集。
//
// 值以 float64 返回：statfs 字段在各平台符号性不一（Linux 的 Bsize 是 int64、
// DragonFly 的 Blocks 是 int64），统一走浮点避免整型换算的溢出判定（gosec G115）；
// float64 整数精度到 2^53 字节（9 PB），远超单机数据卷，Prometheus gauge 本就是
// float64。块大小恒为正，负值只可能是内核异常值 → 乘积为 0，可安全相乘。
// 只用标准库 syscall，不引入新依赖（ADR-004 / ROADMAP #18 约束）。
func volumeUsage(path string) (size, free float64, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, false
	}
	blockSize := float64(st.Bsize)
	return blockSize * float64(st.Blocks), blockSize * float64(st.Bavail), true
}

// volumeInodes 返回 path 所在文件系统的 inode（i 节点）总数与空闲数（statfs 的
// Files / Ffree）。语义与 volumeUsage 一致：失败返回 ok=false，调用方**不输出**序列
// ——inode 总数为 0 时 `free/total` 是 0/0 = NaN，告警会静默不触发，故宁可不发序列。
//
// Windows 没有对等的 inode 概念（volume_windows.go 恒 ok=false），运维按
// OPERATIONS.md §4.3 用宿主 node_exporter / df -i 采集。值同样走 float64：
// 三平台的 Files / Ffree 分别为 uint64，统一浮点避免整型换算溢出判定（gosec G115），
// 而 inode 数量级远低于字节数，float64 精度绰绰有余。
func volumeInodes(path string) (total, free float64, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, false
	}
	return float64(st.Files), float64(st.Ffree), true
}
