package store

// datadir_perm_test.go —— threat-model.md 边界 A 的 Info disclosure 承诺
// 「数据目录 0700 + 文件 0600」必须对**预建目录**也成立。
//
// os.MkdirAll(dir, 0o700) 的 mode 只在**新建**时生效：Docker volume /
// systemd StateDirectory 预先创建的目录多为 0755，此后永远保持 0755，
// 与文档声明不符（文件侧已有 chmodSQLitePerms 主动收紧到 0600 的先例）。

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirPermTightenedTo0700ForPrecreatedDir(t *testing.T) {
	makers := map[string]func(dir string) error{
		"Open": func(dir string) error {
			_, err := Open(dir, "json", "")
			return err
		},
		"AcquireDataDirLock": func(dir string) error {
			release, err := AcquireDataDirLock(dir)
			if err == nil {
				release()
			}
			return err
		},
	}
	for name, make := range makers {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "data")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatalf("mkdir precreated dir: %v", err)
			}
			if err := make(dir); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			fi, err := os.Stat(dir)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if fi.Mode().Perm() != 0o700 {
				t.Fatalf("%s 后 data dir perm = %04o, want 0700（MkdirAll 只在新建时生效，预建 0755 必须被收紧）",
					name, fi.Mode().Perm())
			}
		})
	}
}
