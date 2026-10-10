//go:build !linux && !darwin && !freebsd && !windows

package handler

// volumeUsage 在本平台没有免依赖的容量查询实现（Linux / macOS / FreeBSD 走 statfs、
// Windows 走 kernel32 的 GetDiskFreeSpaceExW，见各自实现），恒返回 ok=false：
// 调用方不输出容量序列，运维按 OPERATIONS.md §4.3 的外部采集口径（node_exporter / df）
// 补齐——没有可信数据就不发序列，不用 0 冒充「容量为 0」。
func volumeUsage(string) (size, free float64, ok bool) {
	return 0, 0, false
}

// volumeInodes 同 volumeUsage：本平台无免依赖实现，恒 ok=false，调用方不发 inode 序列
// （运维按 OPERATIONS.md §4.3 外部采集 inode 用量）。
func volumeInodes(string) (total, free float64, ok bool) {
	return 0, 0, false
}
