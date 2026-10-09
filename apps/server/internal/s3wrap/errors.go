package s3wrap

import (
	"errors"
	"fmt"

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
	// ErrPartialDelete 表示批量删除的 200 响应体内有逐 key 被服务端拒绝的条目（其余已删除）。
	ErrPartialDelete = errors.New("partially deleted")
)

// partialDeleteError 是 ErrPartialDelete 的结构化载体：sentinel 只表达「部分删除」，
// 已删计数与首个失败 key/code 必须放进结构化字段，UserMessage 才能向用户透出计数
// 而不必解析错误文本（review §R17：部分删除曾被报成通用 500，精细文案与计数一并丢失）。
type partialDeleteError struct {
	deleted   int
	firstKey  string
	firstCode string
}

func (e *partialDeleteError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", ErrPartialDelete.Error(), e.firstKey, e.firstCode)
}

// Unwrap 指向 sentinel：errors.Is(err, ErrPartialDelete) 与 HTTPStatus 的 409 判定据此成立。
func (e *partialDeleteError) Unwrap() error { return ErrPartialDelete }

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
	if errors.Is(err, ErrPartialDelete) {
		// 优先透出已删计数（结构化字段，不解析文本）；裸 sentinel 包装回退基础文案。
		var pde *partialDeleteError
		if errors.As(err, &pde) {
			return fmt.Sprintf("some objects could not be deleted (%d deleted)", pde.deleted)
		}
		return "some objects could not be deleted"
	}
	if IsNotFound(err) {
		if HasErrorCode(err, "NoSuchBucket") {
			return "bucket not found"
		}
		return "object not found"
	}
	return UserMessageForCode(ErrorCode(err))
}

// UserMessageForCode 把 S3 错误码映射为面向用户的短消息，与 UserMessage 共用同一张映射表。
//
// 用于「错误只以响应体形式出现、拿不到 error 值」的场景——例如 DeleteObjects 在 200 响应体内
// 逐 key 返回的 <Error>（review §B3）。未收录的码统一归为 "storage operation failed"，
// 与 UserMessage 的兜底一致。
func UserMessageForCode(code string) string {
	switch code {
	case "AccessDenied":
		return "access denied"
	case "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return "invalid credentials"
	case "BucketNotEmpty":
		return "bucket not empty"
	case "InvalidRequest", "InvalidArgument", "MalformedPolicy", "MalformedXML", "InvalidStorageClass", "InvalidPartOrder":
		return "invalid request"
	case "EntityTooLarge":
		return "entity too large"
	case "SlowDown", "ServiceUnavailable", "RequestTimeout":
		return "storage temporarily unavailable"
	case "NoSuchUpload":
		return "multipart upload not found"
	case "PreconditionFailed":
		return "precondition failed (object changed or already exists)"
	case "ConditionalRequestConflict":
		return "conditional request conflict, retry after re-reading the object"
	case "ObjectLocked":
		return "object is locked by retention or legal hold"
	case "RetentionPeriodTooShort":
		return "retention period too short (not later than current retention)"
	case "ObjectLockConfigurationNotFoundError":
		return "object lock is not enabled for this bucket"
	case "NotImplemented":
		return "not supported by this storage endpoint"
	}
	return "storage operation failed"
}

// HTTPStatus 将常见 S3 错误映射为稳定 HTTP 状态；与 UserMessage 对同一错误的归类必须
// 同口径（「暂不可用」→ 503、冲突 → 409），否则客户端状态码与用户文案互相矛盾。
func HTTPStatus(err error) int {
	if err == nil {
		return 500
	}
	// 应用层 sentinel 先于错误码判定：部分删除的错误体不含 API 错误码，但语义是
	// 「请求已完成、部分条目冲突」→ 409；回落 500 会让 UserMessage 的精细文案不可达
	// 且丢失已删计数（review §R17）。
	if errors.Is(err, ErrPartialDelete) {
		return 409
	}
	if IsNotFound(err) {
		return 404
	}
	switch ErrorCode(err) {
	case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return 403
	case "InvalidRequest", "InvalidArgument", "MalformedPolicy", "MalformedXML", "EntityTooLarge", "InvalidStorageClass", "InvalidPartOrder":
		return 400
	case "BucketNotEmpty":
		return 409
	case "InvalidRange":
		return 416
	case "PreconditionFailed":
		return 412
	case "ConditionalRequestConflict":
		return 409
	case "ObjectLocked":
		return 409
	case "RetentionPeriodTooShort":
		return 400
	case "ObjectLockConfigurationNotFoundError":
		return 400
	case "NotImplemented":
		return 501
	case "SlowDown", "ServiceUnavailable", "RequestTimeout":
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

// IsEntityTooLarge 判断是否为对象过大错误：只做结构化判定（应用层 sentinel 或 S3
// 错误码），不匹配错误文案——文案会被上游措辞 / 本地化改变，字符串子串匹配会静默失效
// （review Nit：冗余文案匹配已删除）。
func IsEntityTooLarge(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrObjectTooLarge) || HasErrorCode(err, "EntityTooLarge")
}
