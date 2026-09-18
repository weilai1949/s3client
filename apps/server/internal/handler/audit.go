package handler

// audit.go —— 安全审计日志（roadmap #3 / ASSESSMENT M3）。
//
// 与通用 access log 的区别：审计日志只记录**安全敏感动作**（鉴权失败、账号 CRUD、
// 策略与删除变更），带固定的 `audit` 事件名与操作者 IP，便于集中检索与告警。
// 不记录任何密钥/凭据值；失败也不影响主流程。

import "net/http"

// 审计事件名（稳定契约，勿随意改名：日志检索与告警依赖它们）。
const (
	auditAuthDenied        = "auth.denied"
	auditAccountCreate     = "account.create"
	auditAccountUpdate     = "account.update"
	auditAccountDelete     = "account.delete"
	auditBucketPolicySet   = "bucket.policy.update"
	auditBucketPolicyClear = "bucket.policy.delete"
	auditObjectsDelete     = "objects.delete"
	auditDeletePrefix      = "objects.delete_prefix"
	auditTrashPurge        = "trash.purge"
	auditRateLimited       = "rate_limit.exceeded"
)

// audit 写一条安全审计事件。ip 由调用方传入（统一走可信代理解析），
// extra 为附加字段（如账号 id、bucket）；不接受密钥类字段。
func (h *Handler) audit(r *http.Request, event string, extra ...any) {
	attrs := []any{
		"audit", event,
		"ip", clientIPWithProxies(r, h.trustedProxies),
		"method", r.Method,
		"path", r.URL.Path,
	}
	attrs = append(attrs, extra...)
	h.log.Info("audit", attrs...)
}
