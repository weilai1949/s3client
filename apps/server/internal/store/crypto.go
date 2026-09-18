package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

// 加密文件格式（仅 S3C2）：
//
//	magic "S3C2"(4) + salt(16) + nonce|ciphertext
//	密钥 = Argon2id(password, salt)
var encMagicV2 = []byte("S3C2")

const (
	encSaltLen   = 16
	argonTime    = 1
	argonMemory  = 64 * 1024 // KiB
	argonThreads = 4
	keyLen       = 32
)

// deriveKey 由口令与盐派生 AES-256 密钥（两个文件驱动的共同原语）。
func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, keyLen)
}

// envelope 拼装 S3C2 信封：magic || salt || ciphertext。
func envelope(salt, ciphertext []byte) []byte {
	out := make([]byte, 0, len(encMagicV2)+len(salt)+len(ciphertext))
	out = append(out, encMagicV2...)
	out = append(out, salt...)
	out = append(out, ciphertext...)
	return out
}

func encryptAESGCM(key, plain []byte) ([]byte, error) {
	// aes.NewCipher 仅在 key 长度非法时（≠16/24/32）报错——用于 caller 误用保护。
	// GCM 与 rand.Reader 在 Linux 上不会失败，省去冗余检查。
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonce)
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func decryptAESGCM(key, blob []byte) ([]byte, error) {
	// aes.NewCipher 仅在 key 长度非法时（≠16/24/32）报错——用于 caller 误用保护。
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, _ := cipher.NewGCM(block)
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("ciphertext too short")
	}
	return gcm.Open(nil, blob[:ns], blob[ns:], nil)
}
