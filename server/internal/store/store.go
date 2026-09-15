package store

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/weilai1949/s3clinet/server/internal/model"
)

// Store 是 driver=json 的账号存储：内存状态与 CRUD 由 fileStore 提供，
// 本类型只负责选择落盘编解码策略。
// 当 S3C_STORE_KEY 非空时按 S3C2 格式加密落盘，同时仍能读取历史明文文件。
type Store struct {
	*fileStore
}

// New 创建 store 并从 path 加载已有数据。
// 若环境变量 S3C_STORE_KEY 非空，SecretKey 落盘时加密。
func New(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	s := &Store{fileStore: &fileStore{
		path:     path,
		accounts: make(map[string]*model.Account),
		codec:    &storeCodec{encrypted: keyFromEnv() != nil},
	}}
	if err := s.fileStore.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// keyFromEnv 从 S3C_STORE_KEY 派生 AES-256 密钥（Argon2id，与 EncryptedStore 一致）。
// 返回值仅用于判断是否启用加密落盘；实际加解密密钥按文件内盐值派生。
func keyFromEnv() []byte {
	pw := os.Getenv("S3C_STORE_KEY")
	if pw == "" {
		return nil
	}
	salt := make([]byte, encSaltLen)
	// 盐用固定派生值（非随机），确保同一密钥始终得到同一 AES 密钥。
	// 安全假设：S3C_STORE_KEY 本身是强密钥。
	dk := deriveKey(pw, salt)
	return dk
}

// storeCodec 是 json 驱动的落盘策略：无 key 时写明文 JSON，有 key 时写 S3C2 加密文件；
// 读取时两者都支持，保证旧文件无缝迁移。
type storeCodec struct {
	encrypted bool
}

func (c *storeCodec) missing() error { return nil }

func (c *storeCodec) readError(err error) error {
	return fmt.Errorf("read account file: %w", err)
}

func (c *storeCodec) decode(data []byte) ([]*model.Account, error) {
	// 支持加密文件（S3C2 魔数头）与明文文件（向后兼容）。
	if c.encrypted && strings.HasPrefix(string(data), string(encMagicV2)) {
		if len(data) < len(encMagicV2)+encSaltLen {
			return nil, errors.New("encrypted account file too short")
		}
		salt := make([]byte, encSaltLen)
		copy(salt, data[len(encMagicV2):len(encMagicV2)+encSaltLen])
		key := deriveKey(os.Getenv("S3C_STORE_KEY"), salt)
		plain, derr := decryptAESGCM(key, data[len(encMagicV2)+encSaltLen:])
		if derr != nil {
			return nil, fmt.Errorf("decrypt account file: %w", derr)
		}
		data = plain
	}
	return unmarshalAccounts(data)
}

func (c *storeCodec) encode(list []*model.Account) []byte {
	plain := marshalAccounts(list)
	if !c.encrypted {
		return plain
	}
	// 加密落盘：AES-256-GCM，格式 S3C2|salt|ciphertext（每次写盘换新盐）。
	// crypto/rand 在 Go 1.24+ 不会失败；AES-256-GCM 对 32 字节合法密钥不会失败。
	salt := make([]byte, encSaltLen)
	_, _ = rand.Read(salt)
	enc, _ := encryptAESGCM(deriveKey(os.Getenv("S3C_STORE_KEY"), salt), plain)
	return envelope(salt, enc)
}
