package s3wrap

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// 应用层错误 sentinel：供 UserMessage 用 errors.Is 识别，避免依赖脆弱的错误文本匹配
// （SDK / 各 S3 兼容实现的文案随时可能变化）。
var (
	// ErrObjectTooLarge 表示对象超过单次 PutObject / CopyObject 上限（5GB），应改用分段上传。
	ErrObjectTooLarge = errors.New("object exceeds single-put limit")
	// ErrSourceDeleteFailed 表示复制成功但删除源对象失败（移动半成功）。
	ErrSourceDeleteFailed = errors.New("copied but failed to delete source")
)

// wrapObjectTooLarge 把 S3 的 EntityTooLarge 归一为 ErrObjectTooLarge，
// 让上层可用 errors.Is 判断「单次上传/复制超限」并给出 multipart 指引。
func wrapObjectTooLarge(err error) error {
	if err != nil && HasErrorCode(err, "EntityTooLarge") {
		return fmt.Errorf("%w: %w", ErrObjectTooLarge, err)
	}
	return err
}

// UserMessage 将 S3/SDK 错误映射为面向用户的短消息（防腐层出口）。
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrObjectTooLarge) {
		return "object exceeds 5GB single-put limit; use multipart upload"
	}
	if errors.Is(err, ErrSourceDeleteFailed) {
		return "copied but failed to delete source"
	}
	if IsNotFound(err) {
		if HasErrorCode(err, "NoSuchBucket") {
			return "bucket not found"
		}
		return "object not found"
	}
	switch ErrorCode(err) {
	case "AccessDenied":
		return "access denied"
	case "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return "invalid credentials"
	case "BucketNotEmpty":
		return "bucket not empty"
	case "InvalidRequest", "InvalidArgument", "MalformedPolicy", "MalformedXML", "InvalidStorageClass":
		return "invalid request"
	case "EntityTooLarge":
		return "entity too large"
	case "SlowDown", "ServiceUnavailable", "RequestTimeout":
		return "storage temporarily unavailable"
	case "NoSuchUpload":
		return "multipart upload not found"
	}
	return "storage operation failed"
}

// HTTPStatus 将常见 S3 错误映射为稳定 HTTP 状态。
func HTTPStatus(err error) int {
	if err == nil {
		return 500
	}
	if IsNotFound(err) {
		return 404
	}
	switch ErrorCode(err) {
	case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return 403
	case "InvalidRequest", "InvalidArgument", "MalformedPolicy", "MalformedXML", "EntityTooLarge", "InvalidStorageClass":
		return 400
	case "BucketNotEmpty":
		return 409
	case "InvalidRange":
		return 416
	case "SlowDown", "ServiceUnavailable":
		return 503
	}
	return 500
}

// ErrorCode 提取 smithy API 错误码；非 API 错误返回空串。
func ErrorCode(err error) string {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		return ae.ErrorCode()
	}
	return ""
}

// HasErrorCode 判断是否为给定错误码之一。
func HasErrorCode(err error, codes ...string) bool {
	code := ErrorCode(err)
	if code == "" {
		return false
	}
	for _, c := range codes {
		if code == c {
			return true
		}
	}
	return false
}

// IsNotFound 对象/桶不存在。
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var nf *types.NotFound
	var nsk *types.NoSuchKey
	var ncb *types.NoSuchBucket
	if errors.As(err, &nf) || errors.As(err, &nsk) || errors.As(err, &ncb) {
		return true
	}
	return HasErrorCode(err, "NoSuchBucket", "NotFound", "NoSuchKey", "NoSuchVersion")
}

// IsAPIError 是否为可识别的 S3 API 错误。
func IsAPIError(err error) bool { return ErrorCode(err) != "" }

// IsEntityTooLarge 判断是否为对象过大错误。
func IsEntityTooLarge(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrObjectTooLarge) {
		return true
	}
	if HasErrorCode(err, "EntityTooLarge") || UserMessage(err) == "entity too large" {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "EntityTooLarge") || strings.Contains(msg, "entity too large")
}
