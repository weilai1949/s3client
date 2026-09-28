// Package atomicfile 提供崩溃安全的原子文件写：临时文件 + fsync + rename + 父目录 fsync。
//
// 为什么独立成叶子包：store（账号库落盘）与 service（任务清单落盘）都需要同一套
// 原子写语义，但 service→store 会形成分层倒置（docs/architecture.md：`handler →
// service → s3wrap`、`handler/store → model`，store 与 service 是平级而非上下层），
// 历史上因此复制了两份实现（review R11）。收敛到这个零依赖叶子包后两处共用同一份
// 经全量故障注入测试的实现，双方都只向下依赖，无倒置、无环。
package atomicfile

import (
	"io"
	"os"
	"path/filepath"
)

// 原子写辅助：临时文件 + 写后 rename，避免崩溃导致文件损坏。
// 七个 OS 操作通过包级变量暴露，供测试注入故障以覆盖错误分支（生产代码保持纯 stdlib）。
//
// 为什么用注入而不是 mount/tmpfs：故障注入是单测验证错误路径的事实标准做法，
// 路径固定由本进程命名，无并发抢占风险。
var (
	atomicOpenTmp = func(path string) (*os.File, error) {
		// 清理崩溃残骸；O_EXCL 防抢占与符号链接重定向。
		_ = os.Remove(path)
		return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	}
	atomicWrite  = func(f *os.File, data []byte) (int, error) { return f.Write(data) }
	atomicSync   = func(f *os.File) error { return f.Sync() }
	atomicClose  = func(f *os.File) error { return f.Close() }
	atomicChmod  = func(path string, mode os.FileMode) error { return os.Chmod(path, mode) }
	atomicRename = func(tmp, dst string) error { return os.Rename(tmp, dst) }
	atomicRemove = os.Remove
)

// WriteFile 原子写 path：先写 .tmp，fsync 后 rename 覆盖原文件，最后 fsync 父目录；
// rename 之前的任何错误都清理 .tmp，rename 之后只上抛（目标已是新内容，不能回滚成旧内容）。
// 最终文件权限固定 0600——调用方（账号库、任务清单）都是仅属主可读写的敏感数据。
//
// 为什么每步都要落盘（R11）：rename 只改目录项，不保证数据/新文件名进入磁盘。
// 断电时可能得到「旧内容 + 新文件名」或目录项丢失——账号库损坏。故：
// 数据先 f.Sync（rename 前落盘），rename 后再 fsync 父目录（目录项落盘）。
// chmod 放在 rename 之前对 .tmp 生效：随 f.Sync 一起落盘，且 rename 不改权限位，
// 最终文件天然是 0600；若放 rename 后再收紧，中间窗口文件权限取决于 umask（可能 0644）。
func WriteFile(path string, data []byte) error {
	tmp := path + ".tmp"
	f, err := atomicOpenTmp(tmp)
	if err != nil {
		return err
	}
	if n, werr := atomicWrite(f, data); werr != nil {
		_ = atomicClose(f)
		_ = atomicRemove(tmp)
		return werr
	} else if n != len(data) {
		_ = atomicClose(f)
		_ = atomicRemove(tmp)
		return io.ErrShortWrite
	}
	// 收紧权限：账号文件含 SecretKey/加密盐，必须仅属主可读写。在 rename 前对 tmp 生效，
	// 随 f.Sync 一起落盘；rename 不改权限位，最终文件天然是 0600（放 rename 后再收紧
	// 会让中间窗口的权限取决于 umask，可能 0644）。失败则中止写入：宁可这次保存失败，
	// 也不把权限失控的文件 rename 成正式文件。
	if cerr := atomicChmod(tmp, 0o600); cerr != nil {
		_ = atomicClose(f)
		_ = atomicRemove(tmp)
		return cerr
	}
	// 数据落盘（含上面的权限位），失败则不 rename。
	if serr := atomicSync(f); serr != nil {
		_ = atomicClose(f)
		_ = atomicRemove(tmp)
		return serr
	}
	if cerr := atomicClose(f); cerr != nil {
		_ = atomicRemove(tmp)
		return cerr
	}
	if err := atomicRename(tmp, path); err != nil {
		_ = atomicRemove(tmp)
		return err
	}
	// rename 后 fsync 父目录：把新目录项落盘。失败只上抛不清理——文件已是新内容，
	// 删掉只会更糟；调用方按「本次持久化不确定」处理。
	return fsyncDir(filepath.Dir(path))
}
