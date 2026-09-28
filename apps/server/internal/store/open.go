package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureDataDirPerm 把数据目录收紧为 0700，兑现 threat-model.md 边界 A「数据目录 0700
// + 文件 0600」的声明。
//
// os.MkdirAll 的 mode 只在**新建**时生效：Docker volume / systemd StateDirectory 预先
// 创建的目录多为 0755，此后永远保持 0755（umask 只会减位，不会补位）。文件侧已有
// chmodSQLitePerms 主动收紧到 0600 的先例，目录侧同样按「尽力而为的加固」处理：
// 只读挂载、目录非本进程属主等 Chmod 失败一律忽略，不影响建目录 / 拿锁。
func ensureDataDirPerm(dir string) {
	_ = os.Chmod(dir, 0o700)
}

// Open 按 driver 打开账号存储。
//   - json（默认）：明文 accounts.json（0600）；S3C_STORE_KEY 非空时 S3C3 加密
//   - sqlite：SQLite accounts.db（纯 Go modernc driver）；storeKey 非空时 secret_key 列加密
//   - encrypted：AES-256-GCM 加密 accounts.json.enc（需 S3C_STORE_KEY）
//
// 驱动名先归一化（去空白 + 小写）再分发，大小写/空白变体与小写行为一致；
// 未知驱动直接报错——此前静默落回 json 会让 S3C_STORE_DRIVER=sqlite 的部署
// 悄悄明文落盘，绕过明文落盘安全闸（R4）。空串 = 未显式指定，等价 json（文档化默认）。
func Open(dataDir, driver, storeKey string) (AccountStore, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	ensureDataDirPerm(dataDir)
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite":
		return openSQLite(filepath.Join(dataDir, "accounts.db"), storeKey)
	case "encrypted":
		return NewEncrypted(filepath.Join(dataDir, "accounts.json.enc"), storeKey)
	case "", "json":
		return New(filepath.Join(dataDir, "accounts.json"))
	default:
		return nil, fmt.Errorf("unknown store driver %q (want json|sqlite|encrypted)", driver)
	}
}
