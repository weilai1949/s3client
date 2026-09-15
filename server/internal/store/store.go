package store

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/weilai1949/s3clinet/server/internal/model"
)

// Store 是 json 与 encrypted 两个文件驱动的统一账号存储：内存状态与 CRUD 由
// fileStore 提供，本类型只负责选择落盘编解码策略（codec 配置）。
//
//   - driver=json：路径 accounts.json。S3C_STORE_KEY 非空时按 S3C2 加密落盘，
//     同时仍能读取历史明文文件（permissive）。
//   - driver=encrypted：路径 accounts.json.enc。必须 S3C_STORE_KEY，严格只读写
//     S3C2（strict），文件盐在建文件时随机并复用。
//
// 两驱动共享同一 fileStore + 同一 storeCodec（仅配置不同），消除历史重复的
// EncryptedStore / encryptedCodec。
type Store struct {
	*fileStore
}

// New 创建 json 驱动 store 并从 path 加载已有数据。
// 若环境变量 S3C_STORE_KEY 非空，SecretKey 落盘时加密（S3C2，每次写盘换新盐）。
func New(path string) (*Store, error) {
	return newStore(path, os.Getenv("S3C_STORE_KEY"), false)
}

// NewEncrypted 创建 encrypted 驱动 store（严格 S3C2，复用文件盐）。
// storeKey 必填；文件不存在或为空时生成随机盐并派生密钥。
func NewEncrypted(path, storeKey string) (*Store, error) {
	return newStore(path, storeKey, true)
}

// newStore 构造 *Store：内存状态由 fileStore 提供，codec 按 strict 区分两种驱动。
func newStore(path, storeKey string, strict bool) (*Store, error) {
	if strict && storeKey == "" {
		return nil, errors.New("S3C_STORE_KEY is required for encrypted store")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	s := &Store{fileStore: &fileStore{
		path:     path,
		accounts: make(map[string]*model.Account),
		codec:    newStoreCodec(storeKey, strict),
	}}
	if err := s.fileStore.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// storeCodec 是文件驱动的落盘策略（json / encrypted 共用，仅配置不同）：
//
//	strict=false（json）：无 key 写明文 JSON，有 key 写 S3C2 加密；读取两者都
//	                    支持（向后兼容旧明文文件），写盘每次换新盐。
//	strict=true（encrypted）：只读写 S3C2；文件盐在建文件时随机一次并复用。
type storeCodec struct {
	password string
	strict   bool
	salt     []byte // strict 模式下复用的文件盐
}

// newStoreCodec 按严格模式构造 codec（盐策略随之确定，见 encode/decode）。
func newStoreCodec(storeKey string, strict bool) *storeCodec {
	return &storeCodec{password: storeKey, strict: strict}
}

// missing 在文件不存在或为空时调用：encrypted 借此生成新盐；json 无操作。
func (c *storeCodec) missing() error {
	if c.strict {
		c.salt = randomSalt()
	}
	return nil
}

func (c *storeCodec) readError(err error) error {
	if c.strict {
		return fmt.Errorf("read encrypted account file: %w", err)
	}
	return fmt.Errorf("read account file: %w", err)
}

func (c *storeCodec) decode(data []byte) ([]*model.Account, error) {
	// strict（encrypted）：必须 S3C2 信封；错误文案保持历史语义。
	if c.strict {
		if len(data) < len(encMagicV2)+encSaltLen {
			return nil, errors.New("encrypted account file too short or not S3C2")
		}
		if string(data[:len(encMagicV2)]) != string(encMagicV2) {
			return nil, fmt.Errorf("encrypted account file magic %q is not S3C2", string(data[:len(encMagicV2)]))
		}
		salt := make([]byte, encSaltLen)
		copy(salt, data[len(encMagicV2):len(encMagicV2)+encSaltLen])
		c.salt = salt
		plain, err := decryptAESGCM(deriveKey(c.password, c.salt), data[len(encMagicV2)+encSaltLen:])
		if err != nil {
			return nil, fmt.Errorf("decrypt accounts: %w", err)
		}
		return unmarshalAccounts(plain)
	}

	// permissive（json）：无 key 时视为明文；有 key 时按 S3C2 魔数判别，
	// 兼容历史明文文件（向后兼容）。
	if c.password != "" && strings.HasPrefix(string(data), string(encMagicV2)) {
		if len(data) < len(encMagicV2)+encSaltLen {
			return nil, errors.New("encrypted account file too short")
		}
		salt := make([]byte, encSaltLen)
		copy(salt, data[len(encMagicV2):len(encMagicV2)+encSaltLen])
		plain, derr := decryptAESGCM(deriveKey(c.password, salt), data[len(encMagicV2)+encSaltLen:])
		if derr != nil {
			return nil, fmt.Errorf("decrypt account file: %w", derr)
		}
		data = plain
	}
	return unmarshalAccounts(data)
}

func (c *storeCodec) encode(list []*model.Account) []byte {
	plain := marshalAccounts(list)
	// strict（encrypted）：复用文件盐（missing/decode 时确定）。
	if c.strict {
		enc, _ := encryptAESGCM(deriveKey(c.password, c.salt), plain)
		return envelope(c.salt, enc)
	}
	// permissive（json）：无 key 时明文落盘；有 key 时每次写盘换新盐。
	if c.password == "" {
		return plain
	}
	salt := randomSalt()
	enc, _ := encryptAESGCM(deriveKey(c.password, salt), plain)
	return envelope(salt, enc)
}

// randomSalt 生成 encSaltLen 字节的随机盐（crypto/rand 在 Linux 上不会失败）。
func randomSalt() []byte {
	salt := make([]byte, encSaltLen)
	_, _ = io.ReadFull(rand.Reader, salt)
	return salt
}
