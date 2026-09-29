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
//   - json（默认）：明文 accounts.json（0600）；storeKey 非空时 S3C3 加密
//   - sqlite：SQLite accounts.db（纯 Go modernc driver）；storeKey 非空时 secret_key 列加密
//   - encrypted：AES-256-GCM 加密 accounts.json.enc（需 storeKey）
//
// 驱动名先归一化（去空白 + 小写）再分发，大小写/空白变体与小写行为一致；
// 未知驱动直接报错——此前静默落回 json 会让 S3C_STORE_DRIVER=sqlite 的部署
// 悄悄明文落盘，绕过明文落盘安全闸（R4）。空串 = 未显式指定，等价 json（文档化默认）。
//
// StoreKey 契约：三个分支一律**优先使用显式入参 storeKey**，只有「json 分支 + 入参
// 为空」时才回退到环境变量 S3C_STORE_KEY（见 openJSON）——回退分支是 `New` 的历史
// 语义，服务的是「不传 key、靠环境配置」的调用方，不是丢弃入参。
// KNOWN_ISSUES #64 记录的原状是 json 分支**无条件**丢弃入参（任何调用方传了 key 都
// 明文落盘且无报错），与 sqlite / encrypted 两分支的契约不一致；修复后不再成立。
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
		return openJSON(filepath.Join(dataDir, "accounts.json"), storeKey)
	default:
		return nil, fmt.Errorf("unknown store driver %q (want json|sqlite|encrypted)", driver)
	}
}

// openJSON 打开 json 驱动（permissive：可写明文、可读历史明文 / S3C2 / S3C3）。
//
// storeKey 非空时用该 key 构造 codec（**入参优先于环境变量**）；为空时才走 `New`
// 的语义从 S3C_STORE_KEY 读取——保留 `New` 这条生产调用路径，兼容
// 「不传 key、靠环境配置」的用法（见 Open 注释的 StoreKey 契约）。
func openJSON(path, storeKey string) (*Store, error) {
	if storeKey != "" {
		return newStore(path, storeKey, false)
	}
	return New(path)
}
