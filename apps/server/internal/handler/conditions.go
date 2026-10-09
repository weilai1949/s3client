package handler

// conditions.go —— 条件写（If-Match / If-None-Match）请求字段的边界校验。
//
// 这些值最终作为 HTTP 头发给 S3（服务端写路径）或由浏览器回传（预签名直传），
// 必须在边界拦下注入类输入（CRLF / 空格 / 控制字符）与超长值；校验文案固定，
// 不回显用户输入（error_echo_gate 机械把关）。

// maxConditionValueLen 条件值字节上限（真实 ETag 约 34 字符、"*" 1 字符，余量充足）。
const maxConditionValueLen = 512

// allowedIfNoneMatchValues If-None-Match 的唯一合法非空取值（S3 条件写：仅当目标不存在时写入）。
// 暴露为 map[string]bool 字面量：openapi_semantics 门禁据此机械抽取注册表 enum 的真实值集。
var allowedIfNoneMatchValues = map[string]bool{"*": true}

// validateConditions 校验条件写谓词；返回固定 400 文案，空串表示合法。
//   - ifNoneMatch：S3 条件写只接受 "*"（仅当目标不存在时写入）；
//   - ifMatch：ETag 字面量（可含引号），要求可见 ASCII、无空格、长度受限。
func validateConditions(ifMatch, ifNoneMatch string) string {
	if ifNoneMatch != "" && !allowedIfNoneMatchValues[ifNoneMatch] {
		return "ifNoneMatch must be *"
	}
	if len(ifMatch) > maxConditionValueLen {
		return "ifMatch is too long"
	}
	for i := 0; i < len(ifMatch); i++ {
		if c := ifMatch[i]; c < 0x21 || c > 0x7E {
			return "ifMatch must be printable ASCII without spaces"
		}
	}
	return ""
}
