//go:build unix

package atomicfile

import "os"

// fsyncDir fsync 目录本身，把 rename 产生的目录项落盘（R11）。
// 不做这一步，断电后可能丢失刚 rename 出去的文件名（数据在、目录项没了）。
// unix 上以只读方式打开目录是合法的，Sync 即 fsync(dirfd)。
// Sync 错误直接作为返回值上抛，不额外分叉分支（每个分支都要有测试覆盖）。
func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	serr := d.Sync()
	_ = d.Close()
	return serr
}
