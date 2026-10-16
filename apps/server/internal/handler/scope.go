package handler

// scope.go —— Token 作用域与最小权限（ROADMAP §三 #13）。
//
// 在 withAuth 完成常量时间 token 比较之后，按该 token 的 S3C_TOKEN_SCOPES 声明判定：
//   - 未登记的 token = 全权（向后兼容，零配置行为与历史完全一致）；
//   - readonly：仅放行 GET/HEAD，其余方法（含能铸造写 URL 的预签名 POST）一律 403；
//   - prefixes：请求涉及的桶/键（query 或 JSON body）必须落在许可前缀内；桶级操作
//     （无键引用）需要该桶的「整桶」授权；列表类 prefix 同样按此校验；
//     **零引用 = 不可判定 = 403**（评审 2026-10-15 P-1）——请求不携带任何可判定桶/键
//     引用时一律拒绝（`reason=unjudgeable`），只有少数「与桶/键无关」的只读端点豁免
//     （见 scopeRefLessPaths）。放行零引用请求等于把最小权限变成全权：handler 会回退
//     到路径账号的默认桶，而作用域判定在这条链上完全缺席；
//   - accounts：仅允许路径 {id} 命中的账号，**以及请求引用的账号 id**（迁移 / 计划
//     body 的 sourceAccountId / targetAccountId，与 {id} 计划路径上已存计划的两个
//     账号 id）——否则 accounts:[A] 的 token 能让服务端用 B 的存储凭据执行迁移；
//   - expiresAt：过期 → 401（审计 reason token_expired）。
//
// 另有**端点级**闸（某些端点的请求不携带可判定的桶/账号引用，通用判定对它全部落空，
// 必须整体拒绝受限 token）：
//   - **账号写入面**（`POST /api/accounts`、`PUT|DELETE /api/accounts/{id}`）：创建账号
//     凭空铸造凭据、改写账号把 endpoint/凭据指向他处、删除账号让凭据与配置全损——都不
//     是「访问某个桶/键」而是**凭据面**，与 preview-buckets 同类，对任何资源受限 token
//     一律 403（`reason=account_write`，评审 2026-10-15 P-1 取代原 `accounts_create`）；
//   - POST /api/accounts/preview-buckets：调用方自带 endpoint/凭据由服务端拨号
//     （SSRF 拨号面），已设任何资源作用域（readonly/prefixes/accounts）→ 403；
//   - /api/migrate/jobs*（列取/状态/取消/事件流）：任务记录不带归属，无法按桶细判，
//     已声明前缀/账号作用域 → 403（readonly 读不受限：本就可读全量桶）。
//
// 拒绝一律写审计事件（auth.scope_denied + reason）并回统一 JSON 错误；
// **不记录 token 明文**（凭证不落日志）。
//
// 说明：body 内的桶/键/账号 id 需要完整 JSON 才能判定，因此对配置了 prefixes 或
// accounts 的 token 会在鉴权层读取（并原样还原）请求体；body 无法解析 / 超限 /
// **含重复键（归一化后同名）** 时 fail-closed（403）。

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/config"
)

// auditScopeDenied 是「token 作用域越权」的审计事件名（与 auth.denied 分开便于单独告警）。
const auditScopeDenied = "auth.scope_denied"

// 端点级作用域闸钉住的路径（与 routes.go 的注册名一致）。
const (
	accountsPath          = "/api/accounts"
	previewBucketsPath    = "/api/accounts/preview-buckets"
	migrateJobsPathPrefix = "/api/migrate/jobs"
)

// scopeHasRestriction 报告该作用域是否声明了资源/能力限制（readonly / 前缀 / 账号
// 任一）。仅 expiresAt 的声明没有限制语义（到点失效而已），不触发端点级闸。
func scopeHasRestriction(s config.TokenScope) bool {
	return s.Readonly || len(s.Prefixes) > 0 || len(s.Accounts) > 0
}

// scopeRefLessPaths 是「与桶/键引用无关」的只读端点：受限 token 访问它们时不产生任何
// 可判定的资源引用，因此不适用「零引用即拒绝」的兜底（评审 2026-10-15 P-1）。逐项理由：
//   - /api/accounts：账号**列表**，返回不含 SecretKey 的 AccountView（文档化残留语义）；
//   - /api/schedules：计划**列表**，已由 schedulesInScope 按作用域裁剪；
//   - /api/openapi.json：静态 API 契约，不含任何业务数据。
//
// 增删本表即改变最小权限口径，必须同步 docs/threat-model.md 与 scope_test.go。
var scopeRefLessPaths = map[string]bool{
	accountsPath:        true,
	"/api/schedules":    true,
	"/api/openapi.json": true,
}

// scopeRefLessAllowed 报告零引用请求可否放行到业务层：仅限**只读方法**打白名单端点。
// 写方法一律不豁免（POST /api/accounts 属账号写入面，由 isAccountWriteEndpoint 更早拦下）。
func scopeRefLessAllowed(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	return scopeRefLessPaths[r.URL.Path]
}

// isAccountItemPath 报告路径是否为**账号条目本身**（`/api/accounts/{id}`，其后不再有段）。
// 必须比 accountIDFromPath 更严：后者只取首段，会把 `/api/accounts/{id}/bucket`（删桶）
// 也算成账号 {id}，从而把桶级写操作误判成账号写入面。
func isAccountItemPath(p string) bool {
	rest, ok := strings.CutPrefix(p, "/api/accounts/")
	if !ok {
		return false
	}
	id, tail, _ := strings.Cut(rest, "/")
	return id != "" && tail == ""
}

// isAccountWriteEndpoint 报告该请求是否属于「账号写入面」：创建账号（POST 集合）、
// 改写 / 删除账号（{id} 条目上的 PUT / DELETE）。三者都引入或重定向**凭据面**，对受限
// token 一律拒绝——且不看请求是否携带桶引用，body 里的界内桶名不得成为通过判定的跳板。
func isAccountWriteEndpoint(r *http.Request) bool {
	if r.URL.Path == accountsPath {
		return r.Method == http.MethodPost
	}
	if !isAccountItemPath(r.URL.Path) {
		return false
	}
	return r.Method == http.MethodPut || r.Method == http.MethodDelete
}

// SetTokenScopes 注入「token → 作用域」查询函数（来自 config.Config.ScopeFor）。
// nil = 不启用作用域（所有 token 全权）。需在 Routes() 前调用。
func (h *Handler) SetTokenScopes(lookup func(string) (config.TokenScope, bool)) {
	h.tokenScopes = lookup
}

// authorize 在 token 常量时间比较命中后做作用域判定；返回 false 表示已写出拒绝响应。
// 判定通过时返回的请求已把该 token 的作用域挂上上下文（handler 侧按最小权限过滤响应，
// 见 requestScope）；未受限（未启用 / 未登记）时原样返回。
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, token string) (*http.Request, bool) {
	if h.tokenScopes == nil {
		return r, true
	}
	scope, ok := h.tokenScopes(token)
	if !ok {
		return r, true // 未登记 = 全权（向后兼容）
	}
	if scopeExpired(scope) {
		h.audit(r, auditAuthDenied, "reason", "token_expired")
		h.writeErr(w, http.StatusUnauthorized, "unauthorized")
		return r, false
	}
	// 端点级闸（评审 2026-10-09 R2）：preview-buckets 用**调用方自带**的 endpoint/credential 由
	// 服务端去拨号（SSRF 拨号面，ADR-003 自托管放行私网）。它不是账号 {id} 路径、
	// body 也默认不带桶引用——下面的账号/前缀判定对它全部落空。已设任何资源作用域
	// 的 token 不得把它当认证后的跳板；无限制语义（仅 expiresAt）的 token 不受影响。
	if r.URL.Path == previewBucketsPath && scopeHasRestriction(scope) {
		h.denyScope(w, r, "preview_buckets")
		return r, false
	}
	// 端点级闸（评审 R1）：/api/migrate/jobs*（列取 / 状态 / 取消 / 事件流）没有任何
	// 可注入的桶引用（任务记录不带归属），也不匹配账号 {id}——前缀作用域 token 曾可
	// 枚举全量任务、读别桶 failedKeys、取消他人进行中的迁移。已声明前缀/账号作用域的
	// token 一律拒绝；readonly 不在此列（本就可读全量桶，任务清单不构成新增权限面，
	// POST 取消被下方方法闸拦）。
	if strings.HasPrefix(r.URL.Path, migrateJobsPathPrefix) && (len(scope.Prefixes) > 0 || len(scope.Accounts) > 0) {
		h.denyScope(w, r, "migrate_jobs")
		return r, false
	}
	if scope.Readonly && r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.denyScope(w, r, "readonly")
		return r, false
	}
	// 端点级闸（评审 2026-10-15 P-1）：账号写入面（创建 / 改写 / 删除）不是「访问某个
	// 桶/键」，而是引入或重定向**凭据面**——与 preview-buckets 同类。缺这道闸时，整桶
	// 授权的前缀 token 只要在 body 里写一个**界内**桶名，就能让建号请求产生合法引用、
	// 从而通过前缀判定（实测 201 建出指向攻击者的账号）。故不看引用，一律拒绝。
	if scopeHasRestriction(scope) && isAccountWriteEndpoint(r) {
		h.denyScope(w, r, "account_write")
		return r, false
	}
	if len(scope.Accounts) > 0 {
		if id := accountIDFromPath(r.URL.Path); id != "" && !scopeContains(scope.Accounts, id) {
			h.denyScope(w, r, "account")
			return r, false
		}
		// 请求引用的账号 id（迁移 / 计划 body，或 {id} 计划路径上已存计划）同样受约束。
		// 缺了这道，accounts:[A] 的 token 能让服务端用 B 的存储凭据执行迁移（越权读写）。
		ids, ok := h.requestAccountRefs(r)
		if !ok {
			h.denyScope(w, r, "unparsable_body")
			return r, false
		}
		for _, id := range ids {
			if !scopeContains(scope.Accounts, id) {
				h.denyScope(w, r, "account")
				return r, false
			}
		}
	}
	if len(scope.Prefixes) > 0 {
		refs, ok := h.requestScopeRefs(r)
		if !ok {
			h.denyScope(w, r, "unparsable_body")
			return r, false
		}
		// 零引用 = 不可判定 = 拒绝（评审 2026-10-15 P-1）：请求没带任何可判定的桶/键
		// 引用时，下面的循环一次都不执行——旧实现就此放行，而 handler 会回退到路径账号
		// 的默认桶执行真实操作（实测可越权读 / 改写 / 删除任意账号与其默认桶配置）。
		// 只有「与桶/键无关」的只读端点豁免（scopeRefLessPaths）。
		if len(refs) == 0 && !scopeRefLessAllowed(r) {
			h.denyScope(w, r, "unjudgeable")
			return r, false
		}
		for _, ref := range refs {
			if ref.bucket == "" {
				ref.bucket = h.defaultBucket(r)
			}
			if ref.bucket == "" || !scopeAllowsRef(scope, ref) {
				h.denyScope(w, r, "prefix")
				return r, false
			}
		}
	}
	// 判定通过：作用域挂上下文，供 handler 侧按最小权限过滤响应（如计划列表）。
	return r.WithContext(context.WithValue(r.Context(), scopeCtxKey{}, scope)), true
}

// scopeCtxKey 是请求上下文中「该 token 作用域」的键；未写入 = 该请求未受限。
type scopeCtxKey struct{}

// requestScope 取出请求携带的作用域（判定通过的受限 token 请求才有）。
func requestScope(r *http.Request) (config.TokenScope, bool) {
	scope, ok := r.Context().Value(scopeCtxKey{}).(config.TokenScope)
	return scope, ok
}

// requestAccountRefs 汇总请求引用的账号 id：body 的 sourceAccountId / targetAccountId
// （迁移与计划请求体同名字段），以及 {id} 计划路径（PUT / DELETE / run）上**已存计划**
// 的两个账号 id。返回 ok=false 表示 body 无法判定（fail-closed，与 requestScopeRefs
// 同口径）；计划不存在时不注入，交由 handler 回 404（不因作用域状态泄露存在性）。
func (h *Handler) requestAccountRefs(r *http.Request) ([]string, bool) {
	var ids []string
	if bodyPresent(r) {
		body, ok := peekJSONObject(r)
		if !ok {
			return nil, false
		}
		for _, name := range []string{"sourceAccountId", "targetAccountId"} {
			if s, _ := scopeField(body, name).(string); s != "" {
				ids = append(ids, s)
			}
		}
	}
	if sid := scheduleIDFromPath(r.URL.Path); sid != "" {
		if s, ok := h.sched.Get(sid); ok {
			ids = append(ids, s.SourceAccountID, s.TargetAccountID)
		}
	}
	return ids, true
}

func (h *Handler) tokenExpired(token string) bool {
	if h.tokenScopes == nil {
		return false
	}
	scope, ok := h.tokenScopes(token)
	return ok && scopeExpired(scope)
}

func scopeExpired(scope config.TokenScope) bool {
	return !scope.ExpiresAt.IsZero() && time.Now().After(scope.ExpiresAt)
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
// preview-buckets 等固定子路径不会走到这里：该端点在 authorize 的端点级作用域闸
// 已先行拦截（见头注释 R2），账号判定只看真实 {id}。
func accountIDFromPath(p string) string {
	rest, ok := strings.CutPrefix(p, "/api/accounts/")
	if !ok {
		return ""
	}
	id, _, _ := strings.Cut(rest, "/")
	return id
}

// scheduleIDFromPath 从 /api/schedules/{id}[/...] 提取 {id}；非计划路径返回空串。
func scheduleIDFromPath(p string) string {
	rest, ok := strings.CutPrefix(p, "/api/schedules/")
	if !ok {
		return ""
	}
	id, _, _ := strings.Cut(rest, "/")
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

// scopeField 按「键名大小写不敏感」取 body 字段：peekJSONObject 已把键名归一化为
// 小写，这里把查询名一并归一化，于是调用点仍可照 handler DTO 的驼峰写法书写。
// body 为 nil（无请求体）时返回 nil，与读空 map 同语义。
func scopeField(body map[string]any, name string) any {
	return body[strings.ToLower(name)]
}

// peekJSONObject 读取请求体前 maxBody 字节用于作用域判定，并把 body 原样还原
// 供后续 handler 解码。返回 ok=false 表示无法判定，由调用方 fail-closed。
//
// 解析口径必须与 handler 侧 `encoding/json` 解进 struct 完全一致——否则同一份 body
// 两侧结论不同就是授权绕过（评审 2026-10-15 P-2：只把 `bucket` 写成 `Bucket`，作用域
// 侧看到「零引用」而 handler 侧照常执行）。因此这里：
//   - 键名**归一化为小写**（`encoding/json` 对 struct 字段做 case-fold 匹配，
//     `{"Bucket":…}` 会命中 `json:"bucket"`，而 map 的键查找没有这个回退）；
//   - **拒绝重复键**：归一化后同名即视为不可判定——map 与 struct 对重复键的语义不同
//     （前者末值覆盖，后者顺序赋值且 `null` 对标量字段是 no-op），无法归一就先失败关闭。
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
	dec := json.NewDecoder(bytes.NewReader(buf))
	tok, err := dec.Token()
	if err != nil {
		return nil, false
	}
	if tok == nil {
		return nil, true // JSON null：与「解进 map」的语义一致，视为无请求体
	}
	if tok != json.Delim('{') {
		return nil, false // 顶层必须是对象；数组 / 标量无法按键判定
	}
	out := make(map[string]any, 8)
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, false
		}
		key, ok := kt.(string)
		if !ok {
			return nil, false
		}
		norm := strings.ToLower(key)
		if _, dup := out[norm]; dup {
			return nil, false
		}
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, false
		}
		out[norm] = v
	}
	if _, err := dec.Token(); err != nil { // 收尾 '}'
		return nil, false
	}
	if _, err := dec.Token(); err != io.EOF { // 对象之后不得再有多余数据
		return nil, false
	}
	return out, true
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
	str := func(name string) string {
		s, _ := scopeField(body, name).(string)
		return s
	}
	keys := func(name string) []string {
		arr, _ := scopeField(body, name).([]any)
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
	primaryKeys = addKey(primaryKeys, str("key"))
	primaryKeys = addKey(primaryKeys, str("prefix"))
	primaryKeys = append(primaryKeys, keys("keys")...)
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
	addBucket(str("bucket"))
	// rename / copy-object 的 newKey：给了 newBucket 才算跨桶（否则同主桶）。
	newBucket := str("newBucket")
	newKey := str("newKey")
	if newBucket == "" {
		primaryKeys = addKey(primaryKeys, newKey)
	}
	if len(primaryBuckets) == 0 && isBucketNameEndpoint(r) {
		addBucket(q.Get("name"))
		addBucket(str("name"))
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
	add(str("sourceBucket"), append(keys("sourceKeys"), addKey(nil, str("sourcePrefix"))...))
	add(str("targetBucket"), addKey(nil, str("targetPrefix")))
	// 计划任务的 {id} 路径（PUT/DELETE/run）：从已存计划注入桶/前缀引用——DELETE 与 run
	// 无请求体，不注入的话前缀作用域 token 可触发/删除越界计划（POST 建计划仍走上方 body refs）。
	// 计划不存在时不注入，交给 handler 回 404（不因越权状态泄露存在性）。
	if sid := scheduleIDFromPath(r.URL.Path); sid != "" {
		if s, ok := h.sched.Get(sid); ok {
			refs = append(refs,
				scopeRef{bucket: s.SourceBucket, key: s.SourcePrefix},
				scopeRef{bucket: s.TargetBucket, key: s.TargetPrefix})
		}
	}
	return refs, true
}
