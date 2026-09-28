package s3wrap

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// S3 user-defined metadata 限制（用于在边界处直接 400，避免把非法请求打到 S3 才报 500）。
//
//   - 键：仅 US-ASCII 可打印字符（不允许非 ASCII），长度 ≤ 128。
//   - 值：UTF-8，长度 ≤ 256 字节。
//   - 整体：所有键值对的字节总数 ≤ 2KB（与 S3 限制对齐）。
//
// 这些限制与 AWS S3 文档一致；提前在 API 边界校验可避免
// 把畸形请求落到 S3 端再以 500 形式返回。
const (
	MaxUserMetaKeyLen   = 128
	MaxUserMetaValueLen = 256
	MaxUserMetaTotalLen = 2048
)

// ErrUserMetadataInvalid 在边界校验失败时返回，handler 映射为 400。
var ErrUserMetadataInvalid = errors.New("invalid user metadata")

// ValidateUserMetadata 校验 user-defined metadata 是否符合 S3 限制。
// 失败返回 [ErrUserMetadataInvalid] 包装的描述。
func ValidateUserMetadata(m map[string]string) error {
	if len(m) == 0 {
		return nil
	}
	total := 0
	for k, v := range m {
		if k == "" {
			return fmt.Errorf("%w: empty key", ErrUserMetadataInvalid)
		}
		if len(k) > MaxUserMetaKeyLen {
			return fmt.Errorf("%w: key %q length %d > %d", ErrUserMetadataInvalid, k, len(k), MaxUserMetaKeyLen)
		}
		for i := 0; i < len(k); i++ {
			c := k[i]
			if c < 0x20 || c > 0x7E {
				return fmt.Errorf("%w: key %q has non-printable ASCII at byte %d", ErrUserMetadataInvalid, k, i)
			}
		}
		if !utf8.ValidString(v) {
			return fmt.Errorf("%w: value for key %q is not valid UTF-8", ErrUserMetadataInvalid, k)
		}
		// 值最终会作为 HTTP 头（x-amz-meta-*）发出：控制字符必须在这里拒成 400，
		// 否则请求走到 Go transport 才被 `invalid header field value` 拒发，
		// 对客户端表现为 500——恰好违背本文件「提前在 API 边界校验」的自述目标。
		// 口径对齐 httpguts.ValidHeaderFieldValue：HTAB 与空格（isLWS）放行，
		// 其余 <0x20 的 CTL 与 0x7F 拒绝；>=0x80 属 obs-text，由上面的 UTF-8 校验把关。
		for i := 0; i < len(v); i++ {
			c := v[i]
			if (c < 0x20 && c != '\t') || c == 0x7F {
				return fmt.Errorf("%w: value for key %q has control character at byte %d", ErrUserMetadataInvalid, k, i)
			}
		}
		if len(v) > MaxUserMetaValueLen {
			return fmt.Errorf("%w: value for key %q length %d > %d", ErrUserMetadataInvalid, k, len(v), MaxUserMetaValueLen)
		}
		total += len(k) + len(v)
		if total > MaxUserMetaTotalLen {
			return fmt.Errorf("%w: total size %d > %d", ErrUserMetadataInvalid, total, MaxUserMetaTotalLen)
		}
	}
	return nil
}
