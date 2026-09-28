package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
