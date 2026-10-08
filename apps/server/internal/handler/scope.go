package handler

// scope.go —— Token 作用域与最小权限（ROADMAP §三 #13）。
//
// 在 withAuth 完成常量时间 token 比较之后，按该 token 的 S3C_TOKEN_SCOPES 声明判定：
//   - 未登记的 token = 全权（向后兼容，零配置行为与历史完全一致）；
//   - readonly：仅放行 GET/HEAD，其余方法（含能铸造写 URL 的预签名 POST）一律 403；
//   - prefixes：请求涉及的桶/键（query 或 JSON body）必须落在许可前缀内；桶级操作
//     （无键引用）需要该桶的「整桶」授权；列表类 prefix 同样按此校验；
//   - accounts：仅允许路径 {id} 命中的账号；
//   - expiresAt：过期 → 401（审计 reason token_expired）。
//
// 拒绝一律写审计事件（auth.scope_denied + reason）并回统一 JSON 错误；
// **不记录 token 明文**（凭证不落日志）。
//
// 说明：body 内的桶/键需要完整 JSON 才能判定，因此对配置了 prefixes 的 token 会在
// 鉴权层读取（并原样还原）请求体；body 无法解析 / 超限时 fail-closed（403）。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/config"
)

// auditScopeDenied 是「token 作用域越权」的审计事件名（与 auth.denied 分开便于单独告警）。
const auditScopeDenied = "auth.scope_denied"

// SetTokenScopes 注入「token → 作用域」查询函数（来自 config.Config.ScopeFor）。
// nil = 不启用作用域（所有 token 全权）。需在 Routes() 前调用。
func (h *Handler) SetTokenScopes(lookup func(string) (config.TokenScope, bool)) {
	h.tokenScopes = lookup
}

// authorize 在 token 常量时间比较命中后做作用域判定；返回 false 表示已写出拒绝响应。
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, token string) bool {
	if h.tokenScopes == nil {
		return true
	}
	scope, ok := h.tokenScopes(token)
	if !ok {
		return true // 未登记 = 全权（向后兼容）
	}
	if !scope.ExpiresAt.IsZero() && time.Now().After(scope.ExpiresAt) {
		h.audit(r, auditAuthDenied, "reason", "token_expired")
		h.writeErr(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	if scope.Readonly && r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.denyScope(w, r, "readonly")
		return false
	}
	if len(scope.Accounts) > 0 {
		if id := accountIDFromPath(r.URL.Path); id != "" && !scopeContains(scope.Accounts, id) {
			h.denyScope(w, r, "account")
			return false
		}
	}
	if len(scope.Prefixes) > 0 {
		refs, ok := h.requestScopeRefs(r)
		if !ok {
			h.denyScope(w, r, "unparsable_body")
			return false
		}
		for _, ref := range refs {
			if ref.bucket == "" {
				ref.bucket = h.defaultBucket(r)
			}
			if ref.bucket == "" || !scopeAllowsRef(scope, ref) {
				h.denyScope(w, r, "prefix")
				return false
			}
		}
	}
	return true
}

// denyScope 写审计事件与统一 403 JSON 错误（不回显 token）。
func (h *Handler) denyScope(w http.ResponseWriter, r *http.Request, reason string) {
	h.audit(r, auditScopeDenied, "reason", reason)
	h.writeErr(w, http.StatusForbidden, "forbidden by token scope")
}

// scopeContains 报告 list 是否含 want。
func scopeContains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// accountIDFromPath 从 /api/accounts/{id}/... 提取 {id}；非账号路径返回空串。
// preview-buckets 是固定子路径、不是账号 id，按空串处理，避免被误判为越界账号。
func accountIDFromPath(p string) string {
	rest, ok := strings.CutPrefix(p, "/api/accounts/")
	if !ok {
		return ""
	}
	id, _, _ := strings.Cut(rest, "/")
	if id == "preview-buckets" {
		return ""
	}
	return id
}

// defaultBucket 解析账号默认桶（handler 省略 bucket 时会回退到它）。
// 解析不到返回空串，由调用方 fail-closed。
func (h *Handler) defaultBucket(r *http.Request) string {
	id := accountIDFromPath(r.URL.Path)
	if id == "" {
		return ""
	}
	acc, err := h.store.Get(id)
	if err != nil {
		return ""
	}
	return acc.Bucket
}

// scopeRef 是请求涉及的一个资源引用：桶 + 键（键为空表示桶级引用）。
type scopeRef struct {
	bucket string
	key    string
}

// scopeAllowsRef 判定引用是否落在许可前缀内：
//   - "<bucket>" = 整桶授权（任意键，含桶级操作）；
//   - "<bucket>/<key前缀>" = 仅该前缀下的键；桶级引用（key 为空）需整桶授权。
func scopeAllowsRef(scope config.TokenScope, ref scopeRef) bool {
	for _, p := range scope.Prefixes {
		bucket, keyPrefix, _ := strings.Cut(p, "/")
		if bucket != ref.bucket {
			continue
		}
		switch {
		case keyPrefix == "":
			return true
		case ref.key != "" && strings.HasPrefix(ref.key, keyPrefix):
			return true
		}
	}
	return false
}

// bodyPresent 报告请求是否可能带 body（ContentLength 为 0 视为无 body；-1 = chunked）。
func bodyPresent(r *http.Request) bool {
	return r.ContentLength != 0
}

// peekJSONObject 读取请求体前 maxBody 字节用于作用域判定，并把 body 原样还原
// 供后续 handler 解码。返回 ok=false 表示无法判定（读取失败 / 超限 / 非法 JSON），
// 由调用方 fail-closed。
func peekJSONObject(r *http.Request) (map[string]any, bool) {
	if r.Body == nil {
		return nil, true
	}
	buf, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return nil, false
	}
	if len(buf) > maxBody {
		return nil, false
	}
	r.Body = io.NopCloser(bytes.NewReader(buf))
	if len(bytes.TrimSpace(buf)) == 0 {
		return nil, true
	}
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, false
	}
	return m, true
}

// isBucketNameEndpoint 报告该请求是否用 name 字段表示桶名（建桶 / 删桶）。
func isBucketNameEndpoint(r *http.Request) bool {
	if !strings.HasSuffix(r.URL.Path, "/bucket") {
		return false
	}
	return r.Method == http.MethodPost || r.Method == http.MethodDelete
}

// requestScopeRefs 汇总请求涉及的桶/键引用（query 优先，其次 JSON body）。
// 返回 ok=false 表示 body 无法判定（fail-closed）。
//
// 字段与 routes.go 的真实端点一一对应：
//   - 主桶：query bucket/key/prefix；body bucket/key/prefix/keys；rename 的 newKey 同桶；
//   - 建桶 / 删桶：body.name / query.name；
//   - 复制 / 迁移：sourceBucket+sourceKeys/sourcePrefix、targetBucket+targetPrefix、
//     newBucket+newKey。
func (h *Handler) requestScopeRefs(r *http.Request) ([]scopeRef, bool) {
	q := r.URL.Query()
	var body map[string]any
	if bodyPresent(r) {
		m, ok := peekJSONObject(r)
		if !ok {
			return nil, false
		}
		body = m
	}
	str := func(v any) string {
		s, _ := v.(string)
		return s
	}
	keys := func(v any) []string {
		arr, _ := v.([]any)
		out := make([]string, 0, len(arr))
		for _, e := range arr {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	addKey := func(dst []string, k string) []string {
		if k == "" {
			return dst
		}
		return append(dst, k)
	}

	// 主桶 + 主键。
	var primaryKeys []string
	primaryKeys = addKey(primaryKeys, q.Get("key"))
	primaryKeys = addKey(primaryKeys, q.Get("prefix"))
	primaryKeys = addKey(primaryKeys, str(body["key"]))
	primaryKeys = addKey(primaryKeys, str(body["prefix"]))
	primaryKeys = append(primaryKeys, keys(body["keys"])...)
	// 桶引用可能有多个来源（query / body）。二者不一致时**都要**校验：handler 读哪一处
	// 由端点决定，只取其一会让越界桶从另一处溜过（fail-closed）。
	var primaryBuckets []string
	addBucket := func(b string) {
		if b == "" {
			return
		}
		for _, x := range primaryBuckets {
			if x == b {
				return
			}
		}
		primaryBuckets = append(primaryBuckets, b)
	}
	addBucket(q.Get("bucket"))
	addBucket(str(body["bucket"]))
	// rename / copy-object 的 newKey：给了 newBucket 才算跨桶（否则同主桶）。
	newBucket := str(body["newBucket"])
	newKey := str(body["newKey"])
	if newBucket == "" {
		primaryKeys = addKey(primaryKeys, newKey)
	}
	if len(primaryBuckets) == 0 && isBucketNameEndpoint(r) {
		addBucket(q.Get("name"))
		addBucket(str(body["name"]))
	}

	var refs []scopeRef
	add := func(bucket string, refKeys []string) {
		switch {
		case bucket == "" && len(refKeys) == 0:
			return
		case len(refKeys) == 0:
			refs = append(refs, scopeRef{bucket: bucket})
		default:
			for _, k := range refKeys {
				refs = append(refs, scopeRef{bucket: bucket, key: k})
			}
		}
	}
	if len(primaryBuckets) == 0 {
		add("", primaryKeys) // 桶缺省 → 由默认桶解析（解析不到即 fail-closed）
	}
	for _, b := range primaryBuckets {
		add(b, primaryKeys)
	}
	if newBucket != "" {
		add(newBucket, addKey(nil, newKey))
	}
	add(str(body["sourceBucket"]), append(keys(body["sourceKeys"]), addKey(nil, str(body["sourcePrefix"]))...))
	add(str(body["targetBucket"]), addKey(nil, str(body["targetPrefix"])))
	return refs, true
}
