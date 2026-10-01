//go:build windows

package handler

import (
	"syscall"
	"unsafe"
)

// kernel32 / procDiskFreeExW 是 GetDiskFreeSpaceExW 的延迟绑定：
// Windows 版 syscall 包（标准库）不提供磁盘容量 API，唯一免新依赖的取法是
// 直接调 kernel32（golang.org/x/sys 会新增依赖，违 ADR-004 / ROADMAP #18 约束）。
var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	procDiskFreeW = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// volumeUsage 返回 path 所在卷的总容量与调用方可支配的可用字节。
// 调用方保证 path 非空；API 失败返回 ok=false，调用方据此不输出容量序列。
//
// 取「available to caller」而非卷上空闲总量，与 unix 实现的 Bavail 口径对齐
// （配额 / 受限用户下两者不同，容量告警关心前者）。返回 float64 与 unix 实现一致。
func volumeUsage(path string) (size, free float64, ok bool) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, false
	}
	var avail, total uint64
	// 返回值是 BOOL：0 = 失败（路径不存在等），此时不发序列。
	r1, _, _ := procDiskFreeW.Call(
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&avail)),
		uintptr(unsafe.Pointer(&total)),
		0,
	)
	if r1 == 0 {
		return 0, 0, false
	}
	return float64(total), float64(avail), true
}
