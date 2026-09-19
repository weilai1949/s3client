package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// 加密文件格式（S3C2 旧格式 / S3C3 当前格式）：
//
//	S3C2（只读，兼容既有加密库）：magic "S3C2"(4) + salt(16) + nonce|ciphertext
//	    密钥 = Argon2id(password, salt)，参数硬编码（t=1, m=64MiB, p=4）。
//	S3C3（当前写入）：magic "S3C3"(4) + time(4,BE) + memory(4,BE) + threads(1) + salt(16) + nonce|ciphertext
//	    密钥 = Argon2id(password, salt, time, memory, threads)。
//
// 为什么要有 S3C3：S3C2 不存 KDF 参数，调参会让既有文件无法解密。S3C3 把参数写进
// 文件头，读取时按文件里的参数派生密钥，因此可以在不影响旧库的前提下逐步加强参数
// （ASSESSMENT M2）。S3C2 保持可读，新写入一律 S3C3。
var (
	encMagicV2 = []byte("S3C2")
	encMagicV3 = []byte("S3C3")
)

const (
	encSaltLen   = 16
	argonMemory  = 64 * 1024 // KiB，S3C2 遗留参数（只读路径）
	argonThreads = 4
	keyLen       = 32
)

// S3C3 当前 KDF 参数（OWASP 建议 Argon2id time cost ≥ 2）。
const (
	argonTimeV3    = 2
	argonMemoryV3  = 64 * 1024 // KiB
	argonThreadsV3 = 4
)

// kdfParams 是一次 Argon2id 派生的完整参数集。
type kdfParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

// legacyParams 是 S3C2 文件隐含使用的参数（无文件头，按历史硬编码值派生）。
var legacyParams = kdfParams{time: 1, memory: argonMemory, threads: argonThreads}

// currentParams 是 S3C3 写入时使用的参数。
var currentParams = kdfParams{time: argonTimeV3, memory: argonMemoryV3, threads: argonThreadsV3}

// deriveKey 由口令、盐与显式 KDF 参数派生 AES-256 密钥。
func deriveKey(password string, salt []byte, p kdfParams) []byte {
	return argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, keyLen)
}

// deriveKeyLegacy 按 S3C2 硬编码参数派生密钥（仅测试与旧格式读取使用）。
func deriveKeyLegacy(password string, salt []byte) []byte {
	return deriveKey(password, salt, legacyParams)
}

// isEncryptedBlob 判断磁盘字节是否为任一受支持的加密信封。
func isEncryptedBlob(data []byte) bool {
	return len(data) >= len(encMagicV2) &&
		(string(data[:4]) == string(encMagicV2) || string(data[:4]) == string(encMagicV3))
}

// envelope 拼装加密信封：magic || [params] || salt || ciphertext。
// 传入 encMagicV2 时按旧格式（无参数头）拼装，传入 encMagicV3 时写入参数头。
func envelope(magic, salt, ciphertext []byte) []byte {
	if string(magic) == string(encMagicV3) {
		out := make([]byte, 0, 4+4+4+1+len(salt)+len(ciphertext))
		out = append(out, encMagicV3...)
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], currentParams.time)
		out = append(out, buf[:]...)
		binary.BigEndian.PutUint32(buf[:], currentParams.memory)
		out = append(out, buf[:]...)
		out = append(out, currentParams.threads)
		out = append(out, salt...)
		return append(out, ciphertext...)
	}
	out := make([]byte, 0, len(magic)+len(salt)+len(ciphertext))
	out = append(out, magic...)
	out = append(out, salt...)
	return append(out, ciphertext...)
}

// parseEnvelope 解析 S3C2 / S3C3 信封，返回 KDF 参数、盐与密文。
// 未知 magic 或长度不足一律报错（不静默降级）。
func parseEnvelope(data []byte) (kdfParams, []byte, []byte, error) {
	if len(data) < len(encMagicV2) {
		return kdfParams{}, nil, nil, errors.New("encrypted account file too short")
	}
	switch string(data[:4]) {
	case string(encMagicV2):
		if len(data) < len(encMagicV2)+encSaltLen {
			return kdfParams{}, nil, nil, errors.New("encrypted account file too short")
		}
		salt := make([]byte, encSaltLen)
		copy(salt, data[len(encMagicV2):len(encMagicV2)+encSaltLen])
		return legacyParams, salt, data[len(encMagicV2)+encSaltLen:], nil
	case string(encMagicV3):
		head := 4 + 4 + 4 + 1
		if len(data) < head+encSaltLen {
			return kdfParams{}, nil, nil, errors.New("encrypted account file too short")
		}
		p := kdfParams{
			time:    binary.BigEndian.Uint32(data[4:8]),
			memory:  binary.BigEndian.Uint32(data[8:12]),
			threads: data[12],
		}
		if p.time == 0 || p.memory == 0 || p.threads == 0 {
			return kdfParams{}, nil, nil, fmt.Errorf("S3C3 file has invalid KDF params: %+v", p)
		}
		salt := make([]byte, encSaltLen)
		copy(salt, data[head:head+encSaltLen])
		return p, salt, data[head+encSaltLen:], nil
	default:
		return kdfParams{}, nil, nil, fmt.Errorf("encrypted account file magic %q is not S3C2/S3C3", string(data[:4]))
	}
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
