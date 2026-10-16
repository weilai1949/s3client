package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.store.List()
	if err != nil {
		h.writeInternalErr(w, err, "failed to load accounts")
		return
	}
	views := make([]*model.AccountView, 0, len(accounts))
	for _, a := range accounts {
		views = append(views, a.View())
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"accounts": views})
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	var a model.Account
	if err := h.readJSON(r, &a); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	// 账号 id 由服务端生成；忽略客户端提交的 id，避免覆盖/污染已有账号。
	a.ID = ""
	if a.Name == "" {
		h.writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if a.Endpoint == "" || a.AccessKey == "" || a.SecretKey == "" {
		h.writeErr(w, http.StatusBadRequest, "endpoint/accessKey/secretKey are required")
		return
	}
	if err := validateAccountEndpoints(&a); err != nil {
		h.log.Debug("account endpoint rejected", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid endpoint configuration")
		return
	}
	created, err := h.store.Create(&a)
	if err != nil {
		h.writeInternalErr(w, err, "failed to create account")
		return
	}
	h.warnPlaintextEndpoints(created)
	h.audit(r, auditAccountCreate, "id", created.ID, "name", created.Name)
	h.writeJSON(w, http.StatusCreated, created.View())
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.writeErr(w, http.StatusNotFound, "account not found")
			return
		}
		h.writeInternalErr(w, err, "failed to load account")
		return
	}
	h.writeJSON(w, http.StatusOK, a.View())
}

func (h *Handler) updateAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var a model.Account
	if err := h.readJSON(r, &a); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	// 账号 id 由服务端生成；忽略客户端提交的 id，避免覆盖/污染已有账号。
	a.ID = ""
	if a.Name == "" {
		h.writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if a.Endpoint == "" || a.AccessKey == "" {
		h.writeErr(w, http.StatusBadRequest, "endpoint/accessKey are required")
		return
	}
	if err := validateAccountEndpoints(&a); err != nil {
		h.log.Debug("account endpoint rejected", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid endpoint configuration")
		return
	}
	updated, err := h.store.Update(id, &a)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.writeErr(w, http.StatusNotFound, "account not found")
			return
		}
		h.writeInternalErr(w, err, "failed to update account")
		return
	}
	h.warnPlaintextEndpoints(updated)
	h.clients.evict(id)
	h.audit(r, auditAccountUpdate, "id", id)
	h.writeJSON(w, http.StatusOK, updated.View())
}

// warnPlaintextEndpoints 对**明文 http://** 的账号端点打一条 WARN（A5，威胁模型 §6.2）。
//
// 背景：数据面签名对带 body 的 `PutObject` / `UploadPart` 用 `UNSIGNED-PAYLOAD`
// （流式 body 无法预读哈希），明文链路上这些载荷可被中间人改写——残留风险本体只能靠
// TLS 消除，本服务**不拦截**（自托管刚需，ADR-003 同口径），但不能让它**无感知**：
// 与 `main.go` 里 `cfg.StorePlaintextWarning()` 的「明文落盘告警」同一口径——只提示、
// 不改变行为。前端在「使用 HTTPS/TLS」勾选框旁同步给出同一句提示（docs/threat-model.md §6.2）。
//
// 判定走 `s3wrap.NormalizeEndpoint`（端点归一化的**唯一**实现）：裸端点按 `useSSL` 补全
// scheme，故「没写 scheme + 未勾 TLS」同样算明文。`publicEndpoint` 单独判——浏览器直传的
// 预签名 PUT 走它，明文链路下同样无完整性保护。日志只记 id / name / 端点，且端点先经
// `redactEndpoint` 去掉 userinfo（P-3）——**不记任何密钥**。
func (h *Handler) warnPlaintextEndpoints(a *model.Account) {
	var plain []string
	if isPlaintextEndpoint(a.Endpoint, a.UseSSL) {
		plain = append(plain, "endpoint")
	}
	if a.PublicEndpoint != "" && isPlaintextEndpoint(a.PublicEndpoint, a.UseSSL) {
		plain = append(plain, "publicEndpoint")
	}
	if len(plain) == 0 {
		return
	}
	h.log.Warn("明文 http:// 端点：带 body 的请求缺少 SigV4 载荷完整性（threat-model §6.2）——建议勾选「使用 HTTPS/TLS」或在反向代理处终止 TLS",
		"fields", strings.Join(plain, ","),
		"id", a.ID, "name", a.Name,
		"endpoint", redactEndpoint(a.Endpoint), "publicEndpoint", redactEndpoint(a.PublicEndpoint))
}

// isPlaintextEndpoint 判定归一化后的端点是否为明文 http（scheme 大小写不敏感）。
func isPlaintextEndpoint(endpoint string, useSSL bool) bool {
	return strings.HasPrefix(s3wrap.NormalizeEndpoint(endpoint, useSSL), "http://")
}

// validateAccountEndpoints 校验账号写入（建号 / 更新）时的两个端点。
//
// P-3：带 userinfo 的端点必须 fail-closed（`s3wrap.ValidateEndpoint` 拒绝）——不落库、不回显；
// 同一道闸顺带把云元数据 / 非 ASCII 主机等「建得成但永远连不上」的端点在写入侧就挡住
// （此前只有 `preview-buckets` 走 `s3wrap.New` 时会校验）。调用方只回固定文案，不转发 error
// 文本（响应体同样不得回显凭据）。`PublicEndpoint` 留空 = 与 endpoint 相同，空串合法豁免。
func validateAccountEndpoints(a *model.Account) error {
	if err := s3wrap.ValidateEndpoint(a.Endpoint); err != nil {
		return err
	}
	return s3wrap.ValidateEndpoint(a.PublicEndpoint)
}

// redactEndpoint 抹掉端点 URL 的 userinfo（`http://key:secret@host:9000` → `http://host:9000`）供
// 日志使用。P-3 之后脱敏的**唯一实现**是 model.RedactEndpointUserinfo（handler 侧不再保留
// 重复实现）：s3wrap 的归一化入口会在建 client / 预签名前剥离 userinfo，历史落盘数据则由
// AccountView 脱敏，日志侧这一道是同一实现的第三个出口（评审 2026-10-10 N1 / KNOWN_ISSUES #84）。
func redactEndpoint(endpoint string) string { return model.RedactEndpointUserinfo(endpoint) }

// deleteAccount 删除账号。
func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.writeErr(w, http.StatusNotFound, "account not found")
			return
		}
		h.writeInternalErr(w, err, "failed to delete account")
		return
	}
	h.clients.evict(id)
	h.audit(r, auditAccountDelete, "id", id)
	h.writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// testAccount 检测连通性：有默认桶时 HeadBucket，否则 ListBuckets。
func (h *Handler) testAccount(w http.ResponseWriter, r *http.Request) {
	client, acc, ok := h.accountClient(w, r)
	if !ok {
		return
	}
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		bucket = acc.Bucket
	}
	if bucket == "" {
		if _, err := client.ListBuckets(r.Context()); err != nil {
			h.log.Debug("list buckets failed", "err", err)
			h.writeJSON(w, http.StatusOK, map[string]any{"ok": false, "bucket": "", "error": s3UserMessage(err)})
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": ""})
		return
	}
	if err := client.HeadBucket(r.Context(), bucket); err != nil {
		h.log.Debug("head bucket failed", "bucket", bucket, "err", err)
		h.writeJSON(w, http.StatusOK, map[string]any{"ok": false, "bucket": bucket, "error": s3UserMessage(err)})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": bucket})
}

// previewBuckets 用表单凭证临时列出 Bucket（不落库），便于新建账号时选择默认桶。
func (h *Handler) previewBuckets(w http.ResponseWriter, r *http.Request) {
	var a model.Account
	if err := h.readJSON(r, &a); err != nil {
		h.writeBadJSON(w, err)
		return
	}
	if a.Endpoint == "" || a.AccessKey == "" || a.SecretKey == "" {
		h.writeErr(w, http.StatusBadRequest, "endpoint/accessKey/secretKey are required")
		return
	}
	client, err := s3wrap.New(&a)
	if err != nil {
		h.log.Debug("preview buckets client init", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid account configuration")
		return
	}
	items, err := client.ListBuckets(r.Context())
	if err != nil {
		h.writeInternalErr(w, err, "failed to list buckets")
		return
	}
	buckets := make([]bucketItem, 0, len(items))
	for _, b := range items {
		buckets = append(buckets, bucketItem{Name: b.Name, CreationDate: b.CreationDate})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"buckets": buckets})
}

// validBucketName 校验 S3 桶命名规则（DNS 合规的子集）。
//   - 3–63 字符
//   - 仅小写字母、数字、连字符、点
//   - 首尾必须是小写字母或数字
//   - 不允许连续两个点（保留 ..）
//   - 不允许以连字符相邻（保留 -. / .-）
//
// 完整 S3 规则还有"禁止 IP 形式"和"禁止 xn-- 前缀"等更细边界，违反会由 S3 端拒绝。
func validBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	for i, r := range name {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.'
		if !allowed {
			return false
		}
		if i == 0 || i == len(name)-1 {
			if r == '-' || r == '.' {
				return false
			}
		}
		if i > 0 && (r == '.' || r == '-') && (name[i-1] == '.' || name[i-1] == '-') {
			return false
		}
	}
	return true
}
