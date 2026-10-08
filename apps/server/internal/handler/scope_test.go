package handler

// scope_test.go —— S3C_TOKEN_SCOPES 的 HTTP 行为测试（ROADMAP §三 #13）。
//
// 断言外部可见行为：状态码、审计事件、响应体/日志中无 token 明文。
// 不触碰私有判定函数。

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/config"
	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

const (
	scopeTokReadonly = "tok-readonly-aaaaaaaa"
	scopeTokUnlisted = "tok-unlisted-bbbbbbbb"
	scopeTokSecret   = "tok-secret-cccccccccc"
)

// scopeEnv 是作用域测试的运行环境：真实 Handler + 假 S3 + 一个已建账号。
type scopeEnv struct {
	t    *testing.T
	h    *Handler
	http http.Handler
	acc  *model.Account
	logs *bytes.Buffer
}

// newScopeEnv 构造带固定 token 列表的 Handler，并按 scopes 注入作用域查询函数；
// scopes == nil 表示完全不启用作用域（保持历史全权行为）。
func newScopeEnv(t *testing.T, scopes map[string]config.TokenScope) *scopeEnv {
	t.Helper()
	fake := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<ok/>`)
	})
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "acc", Endpoint: fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "default-bucket", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := New(st, logger, t.TempDir(), nil, scopeTokReadonly+","+scopeTokUnlisted+","+scopeTokSecret, "test", false, false)
	if scopes != nil {
		h.SetTokenScopes(func(tok string) (config.TokenScope, bool) {
			s, ok := scopes[tok]
			return s, ok
		})
	}
	t.Cleanup(h.Shutdown)
	return &scopeEnv{t: t, h: h, http: h.Routes(), acc: acc, logs: buf}
}

// request 发一个带 Bearer token 的请求（body 为 "" 时 ContentLength=0）。
func (e *scopeEnv) request(token, method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	e.http.ServeHTTP(rr, req)
	return rr
}

// auditEvents 解析日志中的审计事件行。
func (e *scopeEnv) auditEvents() []map[string]any {
	var out []map[string]any
	for _, line := range strings.Split(e.logs.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["audit"] != "" {
			out = append(out, m)
		}
	}
	return out
}

// ---- 向后兼容：未登记 / 未启用作用域 ----

func TestScopeNotConfiguredKeepsFullAccess(t *testing.T) {
	e := newScopeEnv(t, nil)
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts", ""); rr.Code != http.StatusOK {
		t.Fatalf("未启用作用域 GET /api/accounts = %d, want 200", rr.Code)
	}
	body := `{"method":"put","bucket":"任意桶","key":"任意/键"}`
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", body); rr.Code != http.StatusOK {
		t.Fatalf("未启用作用域 POST presign = %d, want 200", rr.Code)
	}
}

func TestScopeUnlistedTokenKeepsFullAccess(t *testing.T) {
	// 只登记了 readonly 之外的另一个 token；当前 token 未登记 = 全权。
	e := newScopeEnv(t, map[string]config.TokenScope{
		"other-token-zzzzzzzzzz": {Readonly: true},
	})
	body := `{"method":"put","bucket":"任意桶","key":"任意/键"}`
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", body); rr.Code != http.StatusOK {
		t.Fatalf("未登记 token POST presign = %d, want 200（向后兼容）", rr.Code)
	}
	if rr := e.request(scopeTokUnlisted, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", body); rr.Code != http.StatusOK {
		t.Fatalf("未登记 token2 POST presign = %d, want 200", rr.Code)
	}
}

// ---- readonly ----

func TestScopeReadonlyOnlyAllowsSafeMethods(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Readonly: true},
	})
	for _, m := range []string{http.MethodGet, http.MethodHead} {
		if rr := e.request(scopeTokReadonly, m, "/api/accounts", ""); rr.Code != http.StatusOK {
			t.Errorf("readonly %s /api/accounts = %d, want 200", m, rr.Code)
		}
	}
	cases := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/accounts/" + e.acc.ID + "/presign", `{"method":"put","bucket":"b","key":"k"}`},
		{http.MethodPut, "/api/accounts/" + e.acc.ID, `{}`},
		{http.MethodDelete, "/api/accounts/" + e.acc.ID, ""},
		{http.MethodPatch, "/api/accounts", `{}`},
	}
	for _, c := range cases {
		rr := e.request(scopeTokReadonly, c.method, c.path, c.body)
		if rr.Code != http.StatusForbidden {
			t.Errorf("readonly %s %s = %d, want 403", c.method, c.path, rr.Code)
		}
	}
}

// ---- 过期 ----

func TestScopeExpiredTokenIs401(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {ExpiresAt: time.Now().Add(-time.Minute)},
	})
	rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts", "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("过期 token = %d, want 401", rr.Code)
	}
	reasons := map[string]bool{}
	for _, ev := range e.auditEvents() {
		if ev["audit"] == "auth.denied" {
			reasons[ev["reason"].(string)] = true
		}
	}
	if !reasons["token_expired"] {
		t.Errorf("审计缺少 token_expired，实得 %v", e.auditEvents())
	}
}

// ---- accounts ----

func TestScopeAccountsRestrictPathID(t *testing.T) {
	e := newScopeEnv(t, nil)
	e.h.SetTokenScopes(func(tok string) (config.TokenScope, bool) {
		if tok == scopeTokReadonly {
			return config.TokenScope{Accounts: []string{e.acc.ID}}, true
		}
		return config.TokenScope{}, false
	})
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts/"+e.acc.ID, ""); rr.Code != http.StatusOK {
		t.Errorf("界内账号 = %d, want 200", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts/other-acc", ""); rr.Code != http.StatusForbidden {
		t.Errorf("越界账号 = %d, want 403", rr.Code)
	}
	// 列表端点无路径 {id}：不受 accounts 约束（account 越界由 {id} 判定）。
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts", ""); rr.Code != http.StatusOK {
		t.Errorf("GET /api/accounts = %d, want 200", rr.Code)
	}
	// preview-buckets 不是账号 id，不得被误判为越界。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/preview-buckets", `{}`); rr.Code == http.StatusForbidden {
		t.Errorf("preview-buckets 被误判为越界账号：%d", rr.Code)
	}
}

// ---- prefixes：界内 ----

func TestScopePrefixAllowsInScopeRequests(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a/logs/", "bucket-b"}},
	})
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts/"+e.acc.ID+"/head?bucket=bucket-a&key=logs/x", ""); rr.Code != http.StatusOK {
		t.Errorf("界内 key = %d, want 200", rr.Code)
	}
	body := `{"method":"put","bucket":"bucket-a","key":"logs/y"}`
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", body); rr.Code != http.StatusOK {
		t.Errorf("界内 presign = %d, want 200", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts/"+e.acc.ID+"/head?bucket=bucket-b&key=任意", ""); rr.Code != http.StatusOK {
		t.Errorf("裸桶授权（整桶） = %d, want 200", rr.Code)
	}
	// 与桶/键无关的端点不受 prefixes 约束。
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/accounts", ""); rr.Code != http.StatusOK {
		t.Errorf("无桶引用端点 = %d, want 200", rr.Code)
	}
}

// ---- prefixes：越界 ----

func TestScopePrefixDeniesOutOfScopeRequests(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a/logs/", "bucket-b"}},
	})
	cases := []struct {
		name, method, path, body string
	}{
		{"bucket 越界", http.MethodGet, "/api/accounts/" + e.acc.ID + "/head?bucket=bucket-c&key=logs/x", ""},
		{"key 越界", http.MethodGet, "/api/accounts/" + e.acc.ID + "/head?bucket=bucket-a&key=secret/x", ""},
		{"带键引用但键越界", http.MethodGet, "/api/accounts/" + e.acc.ID + "/object-acl?bucket=bucket-a&key=secret/x", ""},
		{"桶级引用（无 key）", http.MethodGet, "/api/accounts/" + e.acc.ID + "/bucket-info?bucket=bucket-a", ""},
		{"presign 无 key", http.MethodPost, "/api/accounts/" + e.acc.ID + "/presign", `{"method":"put","bucket":"bucket-a"}`},
		{"版本删除越界 key", http.MethodDelete, "/api/accounts/" + e.acc.ID + "/version?bucket=bucket-a&key=secret", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if rr := e.request(scopeTokReadonly, c.method, c.path, c.body); rr.Code != http.StatusForbidden {
				t.Fatalf("%s %s = %d, want 403", c.method, c.path, rr.Code)
			}
		})
	}
}

// ---- prefixes：body / query 多种输入源 ----

func TestScopePrefixAppliesToBodyAndQuerySources(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a/logs/", "bucket-b/logs/"}},
	})
	base := "/api/accounts/" + e.acc.ID

	// body keys[]：界内放行（非 403），越界 403；非字符串元素被忽略。
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/delete", `{"bucket":"bucket-a","keys":["logs/a","logs/b"]}`); rr.Code == http.StatusForbidden {
		t.Errorf("keys[] 界内被拒：%d", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/delete", `{"bucket":"bucket-a","keys":["logs/a","secret"]}`); rr.Code != http.StatusForbidden {
		t.Errorf("keys[] 越界 = %d, want 403", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/delete", `{"bucket":"bucket-a","keys":["logs/a",42]}`); rr.Code == http.StatusForbidden {
		t.Errorf("keys[] 含非字符串元素不应误拒：%d", rr.Code)
	}
	// query prefix（列表类）：界内放行，越界 403。
	if rr := e.request(scopeTokReadonly, http.MethodGet, base+"/objects?bucket=bucket-a&prefix=logs/", ""); rr.Code == http.StatusForbidden {
		t.Errorf("query prefix 界内被拒：%d", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodGet, base+"/objects?bucket=bucket-a&prefix=secret/", ""); rr.Code != http.StatusForbidden {
		t.Errorf("query prefix 越界 = %d, want 403", rr.Code)
	}
	// body prefix（按前缀删除/复制）。
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/delete-prefix", `{"bucket":"bucket-a","prefix":"logs/"}`); rr.Code == http.StatusForbidden {
		t.Errorf("body prefix 界内被拒：%d", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/delete-prefix", `{"bucket":"bucket-a","prefix":"secret/"}`); rr.Code != http.StatusForbidden {
		t.Errorf("body prefix 越界 = %d, want 403", rr.Code)
	}
	// 非字符串 bucket：无法解析出桶名 → 桶引用为空 → 落到默认桶后仍越界。
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/presign", `{"method":"put","bucket":123,"key":"logs/x"}`); rr.Code != http.StatusForbidden {
		t.Errorf("非字符串 bucket = %d, want 403（默认桶不在许可内）", rr.Code)
	}
}

// ---- prefixes：query 与 body 的桶引用必须各自校验（防「只查一处」绕过） ----

func TestScopePrefixChecksQueryAndBodyBucketsIndependently(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a/logs/"}},
	})
	base := "/api/accounts/" + e.acc.ID
	// body 越界桶（query 是界内桶）→ 403。
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/presign?bucket=bucket-a", `{"method":"put","bucket":"bucket-c","key":"logs/x"}`); rr.Code != http.StatusForbidden {
		t.Errorf("body 越界桶 = %d, want 403", rr.Code)
	}
	// query 越界桶（body 是界内桶）→ 403。
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/presign?bucket=bucket-c", `{"method":"put","bucket":"bucket-a","key":"logs/x"}`); rr.Code != http.StatusForbidden {
		t.Errorf("query 越界桶 = %d, want 403", rr.Code)
	}
	// 两处一致（去重）→ 放行。
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/presign?bucket=bucket-a", `{"method":"put","bucket":"bucket-a","key":"logs/x"}`); rr.Code != http.StatusOK {
		t.Errorf("query/body 桶一致 = %d, want 200", rr.Code)
	}
}

// ---- prefixes：默认桶解析 ----

func TestScopePrefixDefaultBucketAndUnresolvable(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"default-bucket/logs/"}},
	})
	// 省略 bucket → handler 回退账号默认桶；作用域按默认桶校验。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", `{"method":"put","key":"logs/x"}`); rr.Code != http.StatusOK {
		t.Errorf("默认桶界内 = %d, want 200", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", `{"method":"put","key":"secret/x"}`); rr.Code != http.StatusForbidden {
		t.Errorf("默认桶越界 = %d, want 403", rr.Code)
	}
	// 账号不存在 → 无法解析默认桶 → fail-closed。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/missing-acc/presign", `{"method":"put","key":"logs/x"}`); rr.Code != http.StatusForbidden {
		t.Errorf("账号不存在时无法解析默认桶 = %d, want 403", rr.Code)
	}
	// 非账号路径且桶引用为空 → 无法解析 → 403。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/migrate", `{"targetPrefix":"logs/"}`); rr.Code != http.StatusForbidden {
		t.Errorf("非账号路径的空桶引用 = %d, want 403", rr.Code)
	}
}

// ---- prefixes：body 不可判定时 fail-closed ----

func TestScopeUnparsableBodyIsDenied(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a"}},
	})
	for _, body := range []string{"{not json", `[1,2]`} {
		if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", body); rr.Code != http.StatusForbidden {
			t.Errorf("body=%q = %d, want 403（无法判定范围）", body, rr.Code)
		}
	}
}

func TestScopeOversizedBodyIsDenied(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a"}},
	})
	body := strings.Repeat("a", maxBody+1)
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", body); rr.Code != http.StatusForbidden {
		t.Fatalf("超大 body = %d, want 403", rr.Code)
	}
}

func TestScopeBodyEdgeCases(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"default-bucket"}},
	})
	// body 为 JSON null：无桶/键引用 → 放行到 handler（handler 报 400，而非 403）。
	if rr := e.request(scopeTokReadonly, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", "null"); rr.Code != http.StatusBadRequest {
		t.Errorf("body=null = %d, want 400（作用域放行、业务校验失败）", rr.Code)
	}
	// ContentLength=-1 且 body 为空（chunked 空体）：无引用 → 放行。
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+scopeTokReadonly)
	req.ContentLength = -1
	req.Body = http.NoBody
	rr := httptest.NewRecorder()
	e.http.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 chunked body = %d, want 400（作用域放行）", rr.Code)
	}
	// r.Body == nil：读取不可判定但无 body 内容 → 按空处理；桶引用越界 → 403。
	req2 := httptest.NewRequest(http.MethodGet, "/api/accounts/"+e.acc.ID+"/objects?bucket=bucket-x", nil)
	req2.Header.Set("Authorization", "Bearer "+scopeTokReadonly)
	req2.ContentLength = -1
	req2.Body = nil
	rr2 := httptest.NewRecorder()
	e.http.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Errorf("nil body + 越界桶 = %d, want 403", rr2.Code)
	}
	// 读取 body 出错：fail-closed。
	req3 := httptest.NewRequest(http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", nil)
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+scopeTokReadonly)
	req3.ContentLength = -1
	req3.Body = scopeErrReader{}
	rr3 := httptest.NewRecorder()
	e.http.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusForbidden {
		t.Errorf("body 读取失败 = %d, want 403", rr3.Code)
	}
}

// scopeErrReader 是一个 Read 恒失败的 body。
type scopeErrReader struct{}

func (scopeErrReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (scopeErrReader) Close() error             { return nil }

// ---- 桶名端点（POST/DELETE .../bucket） ----

func TestScopeBucketNameEndpoints(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a"}},
	})
	base := "/api/accounts/" + e.acc.ID
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/bucket", `{"name":"bucket-a"}`); rr.Code == http.StatusForbidden {
		t.Errorf("界内建桶被拒：%d", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodPost, base+"/bucket", `{"name":"bucket-c"}`); rr.Code != http.StatusForbidden {
		t.Errorf("越界建桶 = %d, want 403", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodDelete, base+"/bucket?name=bucket-a", ""); rr.Code == http.StatusForbidden {
		t.Errorf("界内删桶被拒：%d", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodDelete, base+"/bucket?name=bucket-c", ""); rr.Code != http.StatusForbidden {
		t.Errorf("越界删桶 = %d, want 403", rr.Code)
	}
	// 非 POST/DELETE 的 .../bucket 不按桶名解析（交回路由 405），不得误判越界。
	if rr := e.request(scopeTokReadonly, http.MethodGet, base+"/bucket", ""); rr.Code == http.StatusForbidden {
		t.Errorf("GET .../bucket 不应被作用域 403：%d", rr.Code)
	}
}

// ---- readonly + 前缀授权不能做桶级操作 ----

func TestScopeKeyedGrantDeniesBucketLevelOps(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a/logs/"}},
	})
	if rr := e.request(scopeTokReadonly, http.MethodDelete, "/api/accounts/"+e.acc.ID+"/bucket?name=bucket-a", ""); rr.Code != http.StatusForbidden {
		t.Errorf("仅键前缀授权删整桶 = %d, want 403", rr.Code)
	}
}

// ---- 复制 / 迁移的多桶（source/target/new） ----

func TestScopeMultiBucketGroups(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Prefixes: []string{"bucket-a/logs/", "bucket-b/logs/"}},
	})
	base := "/api/accounts/" + e.acc.ID

	cases := []struct {
		name, path, body string
		want             int
	}{
		{"copy-object 双桶界内", base + "/copy-object", `{"bucket":"bucket-a","key":"logs/x","newBucket":"bucket-b","newKey":"logs/y"}`, 0},
		{"copy-object 目标桶越界", base + "/copy-object", `{"bucket":"bucket-a","key":"logs/x","newBucket":"bucket-c","newKey":"logs/y"}`, http.StatusForbidden},
		{"rename 同桶 newKey 界内", base + "/rename", `{"bucket":"bucket-a","key":"logs/x","newKey":"logs/z"}`, 0},
		{"rename 同桶 newKey 越界", base + "/rename", `{"bucket":"bucket-a","key":"logs/x","newKey":"secret"}`, http.StatusForbidden},
		{"copy-objects 源键越界", base + "/copy-objects", `{"bucket":"bucket-a","keys":["secret"],"targetBucket":"bucket-b","targetPrefix":"logs/"}`, http.StatusForbidden},
		{"copy-objects 目标前缀越界", base + "/copy-objects", `{"bucket":"bucket-a","keys":["logs/x"],"targetBucket":"bucket-b","targetPrefix":"secret/"}`, http.StatusForbidden},
		{"copy-objects 界内", base + "/copy-objects", `{"bucket":"bucket-a","keys":["logs/x"],"targetBucket":"bucket-b","targetPrefix":"logs/"}`, 0},
		{"migrate 源前缀越界", "/api/migrate", `{"sourceBucket":"bucket-a","sourcePrefix":"secret/","targetBucket":"bucket-b","targetPrefix":"logs/"}`, http.StatusForbidden},
		{"migrate 目标桶越界", "/api/migrate", `{"sourceBucket":"bucket-a","sourcePrefix":"logs/","targetBucket":"bucket-c","targetPrefix":"logs/"}`, http.StatusForbidden},
		{"migrate 界内", "/api/migrate", `{"sourceBucket":"bucket-a","sourcePrefix":"logs/","targetBucket":"bucket-b","targetPrefix":"logs/"}`, 0},
		{"migrate sourceKeys 越界", "/api/migrate", `{"sourceBucket":"bucket-a","sourceKeys":["secret"],"targetBucket":"bucket-b","targetPrefix":"logs/"}`, http.StatusForbidden},
		{"migrate sourceKeys 界内", "/api/migrate", `{"sourceBucket":"bucket-a","sourceKeys":["logs/x"],"targetBucket":"bucket-b","targetPrefix":"logs/"}`, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := e.request(scopeTokReadonly, http.MethodPost, c.path, c.body)
			if c.want != 0 {
				if rr.Code != c.want {
					t.Fatalf("%s = %d, want %d", c.path, rr.Code, c.want)
				}
				return
			}
			if rr.Code == http.StatusForbidden {
				t.Fatalf("%s 界内被作用域拒绝（403）", c.path)
			}
		})
	}
}

// ---- 豁免端点 ----

func TestScopeExemptEndpointsUnchanged(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokReadonly: {Readonly: true, ExpiresAt: time.Now().Add(-time.Minute)},
	})
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/health", ""); rr.Code != http.StatusOK {
		t.Errorf("过期 token 访问 /api/health = %d, want 200（豁免）", rr.Code)
	}
	if rr := e.request(scopeTokReadonly, http.MethodGet, "/api/metrics", ""); rr.Code != http.StatusNotFound {
		t.Errorf("/api/metrics（未开启）= %d, want 404", rr.Code)
	}
}

// ---- 安全：拒绝响应与审计不得含 token 明文 ----

func TestScopeDenialDoesNotLeakToken(t *testing.T) {
	e := newScopeEnv(t, map[string]config.TokenScope{
		scopeTokSecret: {Prefixes: []string{"bucket-a/logs/"}},
	})
	rr := e.request(scopeTokSecret, http.MethodPost, "/api/accounts/"+e.acc.ID+"/presign", `{"method":"put","bucket":"bucket-c","key":"x"}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("越界 presign = %d, want 403", rr.Code)
	}
	if strings.Contains(rr.Body.String(), scopeTokSecret) {
		t.Errorf("403 响应体泄露 token 明文：%s", rr.Body.String())
	}
	if strings.Contains(e.logs.String(), scopeTokSecret) {
		t.Errorf("审计日志泄露 token 明文")
	}
	var reason string
	for _, ev := range e.auditEvents() {
		if ev["audit"] == "auth.scope_denied" {
			reason, _ = ev["reason"].(string)
		}
	}
	if reason == "" {
		t.Fatalf("缺少 auth.scope_denied 审计事件：%v", e.auditEvents())
	}
}
