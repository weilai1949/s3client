package s3wrap

// checksum.go —— 端到端校验和：Head 暴露存储端校验和 + VerifyObjectChecksum 本地重算比对。
//
// 设计（ROADMAP §三 #5「端到端校验和」）：
//   - 读侧：HeadObject(ChecksumMode=ENABLED) 把服务端存储的校验和带回（厂商不支持 → nil）；
//   - 验证：先 Head 选定算法阶梯 CRC64NVME → CRC32C → SHA256 → SHA1 → ETag-MD5，
//     再流式 GET 全对象本地重算、与存储端逐字节比对；
//   - 降级：分段合成校验和（COMPOSITE / 值内 "-" 后缀）不可全对象比对 → 跳过；
//     既无校验和又非单段 MD5 ETag → method="none"（不是错误，厂商支持度差异如实报告）。

import (
	"context"
	"crypto/md5"  //nolint:gosec // G501：仅复现 S3 单段 ETag（存储校验和）用于比对，非密码学安全用途
	"crypto/sha1" //nolint:gosec // G505：仅复现 S3 存储的 SHA1 校验和（阶梯第 4 位），非安全用途
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"hash/crc32"
	"io"
	"regexp"
	"strings"
)

// 校验和方法常量（verify 阶梯与响应体共用，唯一来源）。
const (
	methodCRC64NVME = "crc64nvme"
	methodCRC32C    = "crc32c"
	methodSHA256    = "sha256"
	methodSHA1      = "sha1"
	methodETagMD5   = "etag-md5"
	methodNone      = "none"
)

// ObjectChecksums 是对象存储端校验和（Head 的 ChecksumMode=ENABLED 返回；零值字段 = 该算法无值）。
type ObjectChecksums struct {
	CRC64NVME string
	CRC32C    string
	SHA256    string
	SHA1      string
	Type      string // "" | FULL_OBJECT | COMPOSITE_*（分段合成）
}

// ChecksumVerify 是 VerifyObjectChecksum 的结果：method=none 表示无可验证来源（降级，非错误）。
type ChecksumVerify struct {
	Method string
	Local  string // 本地全量重算值（与 remote 同编码：校验和为大端 base64、etag-md5 为小写 hex）
	Remote string // 存储端值
	Match  bool
}

// checksumAlgo 把「构造哈希器」与「按算法编码结果」绑在同一条目上：
// pick 阶梯只产出这里的键，map 查找避免 switch 分支在不可达路径上留下未覆盖语句。
type checksumAlgo struct {
	new    func() hash.Hash
	encode func(hash.Hash) string
}

var crc32Castagnoli = crc32.MakeTable(crc32.Castagnoli)

func base64BigEndian64(v uint64) string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	return base64.StdEncoding.EncodeToString(b[:])
}

func base64BigEndian32(v uint32) string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	return base64.StdEncoding.EncodeToString(b[:])
}

var checksumAlgos = map[string]checksumAlgo{
	methodCRC64NVME: {
		new: func() hash.Hash { return newCRC64NVME() },
		encode: func(h hash.Hash) string {
			return base64BigEndian64(h.(hash.Hash64).Sum64())
		},
	},
	methodCRC32C: {
		new: func() hash.Hash { return crc32.New(crc32Castagnoli) },
		encode: func(h hash.Hash) string {
			return base64BigEndian32(h.(hash.Hash32).Sum32())
		},
	},
	methodSHA256: {
		new:    func() hash.Hash { return sha256.New() },
		encode: func(h hash.Hash) string { return base64.StdEncoding.EncodeToString(h.Sum(nil)) },
	},
	methodSHA1: {
		//nolint:gosec // G401：SHA1 仅复现 S3 存储的 SHA1 校验和，不作安全用途
		new:    func() hash.Hash { return sha1.New() },
		encode: func(h hash.Hash) string { return base64.StdEncoding.EncodeToString(h.Sum(nil)) },
	},
	methodETagMD5: {
		//nolint:gosec // G401：MD5 仅复现 S3 单段 ETag 的存储校验和，不作安全用途
		new:    func() hash.Hash { return md5.New() },
		encode: func(h hash.Hash) string { return hex.EncodeToString(h.Sum(nil)) },
	},
}

// md5ETagPattern 单段上传的 ETag（= 内容 MD5 hex）；含 "-" 是分段合成，不可当 MD5 用。
var md5ETagPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

// isCompositeChecksum 分段合成校验和判定：类型字段 COMPOSITE_* 前缀，或值内含 "-" 后缀。
func isCompositeChecksum(typ, value string) bool {
	return strings.HasPrefix(typ, "COMPOSITE") || strings.Contains(value, "-")
}

// pickChecksum 从 Head 结果里选定可验证的算法与存储端值（阶梯顺序见文件头）。
func pickChecksum(meta *ObjectMeta) (method, remote string) {
	if cs := meta.Checksums; cs != nil {
		if v := cs.CRC64NVME; v != "" && !isCompositeChecksum(cs.Type, v) {
			return methodCRC64NVME, v
		}
		if v := cs.CRC32C; v != "" && !isCompositeChecksum(cs.Type, v) {
			return methodCRC32C, v
		}
		if v := cs.SHA256; v != "" && !isCompositeChecksum(cs.Type, v) {
			return methodSHA256, v
		}
		if v := cs.SHA1; v != "" && !isCompositeChecksum(cs.Type, v) {
			return methodSHA1, v
		}
	}
	etag := strings.Trim(meta.ETag, `"`)
	if md5ETagPattern.MatchString(etag) {
		return methodETagMD5, strings.ToLower(etag)
	}
	return methodNone, ""
}

// VerifyObjectChecksum 流式拉取全对象本地重算，与存储端校验和比对。
// 对象不存在等真实错误上抛；无可验证来源 → {Method:"none"}（由上层如实转述，不视为失败）。
func (c *Client) VerifyObjectChecksum(ctx context.Context, bucket, key, versionID string) (*ChecksumVerify, error) {
	meta, err := c.HeadObjectMeta(ctx, bucket, key, versionID)
	if err != nil {
		return nil, err
	}
	method, remote := pickChecksum(meta)
	if method == methodNone {
		return &ChecksumVerify{Method: methodNone}, nil
	}
	out, err := c.getObjectIn(ctx, bucket, key, versionID, "")
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	algo := checksumAlgos[method]
	h := algo.new()
	if _, err := io.Copy(h, out.Body); err != nil {
		return nil, err
	}
	local := algo.encode(h)
	return &ChecksumVerify{Method: method, Local: local, Remote: remote, Match: local == remote}, nil
}
