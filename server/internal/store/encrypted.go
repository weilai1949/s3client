package store

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/weilai1949/s3clinet/server/internal/model"
)

// EncryptedStore 是 driver=encrypted 的账号存储：内存状态与 CRUD 由 fileStore 提供，
// 本类型只负责随机盐 + AES-256-GCM 的落盘编解码（静态保护 SecretKey）。
type EncryptedStore struct {
	*fileStore
}

// NewEncrypted 创建加密存储。storeKey 非空；新文件用 Argon2id + 随机盐。
func NewEncrypted(encPath, storeKey string) (*EncryptedStore, error) {
	if storeKey == "" {
		return nil, errors.New("S3C_STORE_KEY is required for encrypted store")
	}
	s := &EncryptedStore{fileStore: &fileStore{
		path:     encPath,
		accounts: make(map[string]*model.Account),
		codec:    &encryptedCodec{password: storeKey},
	}}
	if err := s.fileStore.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// encryptedCodec 是 encrypted 驱动的落盘策略：S3C2|salt|ciphertext。
// 盐在文件创建（或首次读取已有文件）时确定，并在该文件生命周期内复用。
type encryptedCodec struct {
	password string
	key      []byte
	salt     []byte
}

// missing 在文件不存在或为空时生成新的随机盐与派生密钥。
func (c *encryptedCodec) missing() error { return c.initFreshKey() }

func (c *encryptedCodec) readError(err error) error {
	return fmt.Errorf("read encrypted account file: %w", err)
}

func (c *encryptedCodec) decode(data []byte) ([]*model.Account, error) {
	if len(data) < len(encMagicV2)+encSaltLen {
		return nil, errors.New("encrypted account file too short or not S3C2")
	}
	if string(data[:len(encMagicV2)]) != string(encMagicV2) {
		return nil, fmt.Errorf("encrypted account file magic %q is not S3C2", string(data[:len(encMagicV2)]))
	}
	salt := make([]byte, encSaltLen)
	copy(salt, data[len(encMagicV2):len(encMagicV2)+encSaltLen])
	c.salt = salt
	c.key = deriveKey(c.password, salt)
	plain, err := decryptAESGCM(c.key, data[len(encMagicV2)+encSaltLen:])
	if err != nil {
		return nil, fmt.Errorf("decrypt accounts: %w", err)
	}
	return unmarshalAccounts(plain)
}

func (c *encryptedCodec) encode(list []*model.Account) []byte {
	// AES-256 GCM 对合法 32 字节 key 不会失败。
	enc, _ := encryptAESGCM(c.key, marshalAccounts(list))
	return envelope(c.salt, enc)
}

func (c *encryptedCodec) initFreshKey() error {
	// crypto/rand 在 Linux 上不会失败；省略防御性检查。
	salt := make([]byte, encSaltLen)
	_, _ = io.ReadFull(rand.Reader, salt)
	c.salt = salt
	c.key = deriveKey(c.password, salt)
	return nil
}
