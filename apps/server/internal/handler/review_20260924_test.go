package handler

// review_20260924_test.go —— docs/code-review-2026-09-24.md 分派到 handler 域的
// R1/R2/R3/R5/R15c/R18/R19a/R20 与 4 个 Nit（migrate 误报 404、copyMany 响应漂移、
// CSP 字面量两处维护、getBucketInfo 串行 3 次调用）的回归测试。
// 全部断言**外部可见行为**（HTTP 状态 / 响应体 / 审计日志 / 对假 S3 的调用次数 /
// expvar 注册表 / 源码级门禁），先红后绿。

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

// ---- R1：限速与审计必须按可信代理链「最后一段」XFF 计数（首段可被上游伪造） ----

// TestRateLimitUsesLastXFFHop 端到端：XFF 首段每次换值伪造、末段固定为真实出口时，
// 限速仍应按末段聚合计数直至 429，且审计事件记录的 ip 是末段。
// 旧实现取首段 → 10 个伪造 IP 各分一桶 → 永不 429（红）。
func TestRateLimitUsesLastXFFHop(t *testing.T) {
	h, rec := newAuditHandler(t, "", []string{"127.0.0.1"})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	status := 0
	for i := 0; i < rateLimitBurst+5; i++ {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/accounts", nil)
		if err != nil {
			t.Fatal(err)
		}
		// 首段由不可信的上游客户端伪造（每次不同），末段是可信代理追加的真实对端。
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.9.9.%d, 198.51.100.7", i%10))
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		status = resp.StatusCode
		resp.Body.Close()
		if status == http.StatusTooManyRequests {
			break
		}
	}
	if status != http.StatusTooManyRequests {
		t.Fatalf("伪造首段绕过了限速（按首段分桶）：last status = %d, want 429", status)
	}
	var last map[string]any
	for _, ev := range rec.auditEvents(t) {
		if ev["audit"] == "rate_limit.exceeded" {
			last = ev
		}
	}
	if last == nil {
		t.Fatal("缺少 rate_limit.exceeded 审计事件")
	}
	if last["ip"] != "198.51.100.7" {
		t.Fatalf("审计 ip = %v, want 198.51.100.7（可信代理链最后一段）", last["ip"])
	}
}

// ---- R2：限速必须在鉴权之外层（未鉴权请求同样消耗限速额度） ----

// TestRateLimitAppliesBeforeAuth 配置了 token 的 handler 收到无 Bearer 的连续请求时，
// 除 401 外必须出现 429。旧实现 rateLimit 在 auth 内层 → 全部 401、永不 429（红）。
func TestRateLimitAppliesBeforeAuth(t *testing.T) {
	h := accNewHandler(t, &accStubStore{}, nil, "unit-test-token-0123456789")
	routes := h.Routes()
	saw429 := false
	for i := 0; i < rateLimitBurst+5; i++ {
		rr := httptest.NewRecorder()
		routes.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
		switch rr.Code {
		case http.StatusTooManyRequests:
			saw429 = true
		case http.StatusUnauthorized:
			// 鉴权拒绝在限速额度耗尽前是预期状态。
		default:
			t.Fatalf("第 %d 次未鉴权请求 status = %d, want 401/429", i+1, rr.Code)
		}
		if saw429 {
			break
		}
	}
	if !saw429 {
		t.Fatalf("连续 %d 次未鉴权请求均未被限速（rateLimit 在 auth 内层）", rateLimitBurst+5)
	}
}

// ---- R3：异步批量任务启动必须写审计（与同步路径的审计对齐） ----

// TestDeletePrefixAsyncAuditsJobStart delete-prefix/async 202 成功后必须留下
// objects.delete_prefix 审计事件（含 jobId/bucket/prefix/total/truncated）。
// 旧实现只在同步路径写审计（红：查不到事件）。
func TestDeletePrefixAsyncAuditsJobStart(t *testing.T) {
	h, rec := newAuditHandler(t, "", nil)
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/1", "p/2"}, false, ""))
		}
		return olPlain(http.StatusNoContent)
	})
	acc, err := h.store.Create(&model.Account{
		Name: "audit-del", Endpoint: srv.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	rr := doJSON(t, h.Routes(), "POST", "/api/accounts/"+acc.ID+"/delete-prefix/async", `{"bucket":"b","prefix":"p/"}`)
	olExpectStatus(t, rr, http.StatusAccepted, "delete-prefix async")
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)

	var ev map[string]any
	for _, e := range rec.auditEvents(t) {
		if e["audit"] == "objects.delete_prefix" {
			ev = e
		}
	}
	if ev == nil {
		t.Fatalf("异步删除前缀未写 objects.delete_prefix 审计事件（破坏「操作可追溯」契约）")
	}
	if ev["jobId"] != start["jobId"] || ev["bucket"] != "b" || ev["prefix"] != "p/" {
		t.Fatalf("审计字段不符: jobId=%v bucket=%v prefix=%v", ev["jobId"], ev["bucket"], ev["prefix"])
	}
	if ev["total"] != float64(2) || ev["truncated"] != false {
		t.Fatalf("审计 total/truncated = %v/%v, want 2/false", ev["total"], ev["truncated"])
	}
}

// TestCopyManyAsyncMoveAuditsJobStart 移动（deleteSource）异步批量必须写 objects.move 审计；
// 纯复制则不得写（避免把只读复制记成移动）。
func TestCopyManyAsyncMoveAuditsJobStart(t *testing.T) {
	h, rec := newAuditHandler(t, "", nil)
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		return olPlain(http.StatusNoContent)
	})
	acc, err := h.store.Create(&model.Account{
		Name: "audit-move", Endpoint: srv.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	routes := h.Routes()
	countMove := func() int {
		n := 0
		for _, e := range rec.auditEvents(t) {
			if e["audit"] == "objects.move" {
				n++
			}
		}
		return n
	}

	rr := doJSON(t, routes, "POST", "/api/accounts/"+acc.ID+"/copy-objects/async",
		`{"bucket":"b","keys":["a.txt"],"deleteSource":true}`)
	olExpectStatus(t, rr, http.StatusAccepted, "copy many async move")
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)

	var ev map[string]any
	for _, e := range rec.auditEvents(t) {
		if e["audit"] == "objects.move" {
			ev = e
		}
	}
	if ev == nil {
		t.Fatalf("移动型异步批量未写 objects.move 审计事件（复制半成功后源被删，无审计无法对账）")
	}
	if ev["jobId"] != start["jobId"] || ev["bucket"] != "b" || ev["targetBucket"] != "b" {
		t.Fatalf("审计字段不符: jobId=%v bucket=%v targetBucket=%v", ev["jobId"], ev["bucket"], ev["targetBucket"])
	}
	if ev["total"] != float64(1) {
		t.Fatalf("审计 total = %v, want 1", ev["total"])
	}

	// 纯复制（deleteSource=false）不得记 objects.move。
	before := countMove()
	rr2 := doJSON(t, routes, "POST", "/api/accounts/"+acc.ID+"/copy-objects/async",
		`{"bucket":"b","keys":["a.txt"]}`)
	olExpectStatus(t, rr2, http.StatusAccepted, "copy many async copy-only")
	if got := countMove(); got != before {
		t.Fatalf("纯复制不得写 objects.move 审计：%d → %d", before, got)
	}
}

// ---- R5：mode=text 预览必须透传 versionId（历史版本文本预览） ----

// TestProxyTextModeForwardsVersionID 假 S3 仅在 versionId=v1 时放行：
// 旧实现 text 分支固定传空 versionId → 403（红）；修复后 200 且内容正确。
func TestProxyTextModeForwardsVersionID(t *testing.T) {
	const body = "hello versioned"
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Get("versionId") != "v1" {
			return olErr(http.StatusForbidden, "AccessDenied")
		}
		return olResp{
			status: http.StatusOK,
			headers: map[string]string{
				"Content-Type":   "text/plain; charset=utf-8",
				"Content-Length": strconv.Itoa(len(body)),
			},
			body: body,
		}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/proxy?bucket=b&key=k&mode=text&versionId=v1", "")
	olExpectStatus(t, rr, http.StatusOK, "text 模式带 versionId")
	if rr.Body.String() != body {
		t.Fatalf("body = %q, want %q", rr.Body.String(), body)
	}
}

// ---- R15c：列表项 / ObjectItem schema 不得带恒为空的 contentType 幻影字段 ----

// TestObjectListOmitsContentType GET /objects 的列表项 JSON 不含 contentType 键
// （head 详情接口仍返回真实 contentType，不在本条范围）。
func TestObjectListOmitsContentType(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		return olXML(http.StatusOK, listBucketXML([]string{"a.txt"}, false, ""))
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/objects?bucket=b", "")
	olExpectStatus(t, rr, http.StatusOK, "list objects")
	var resp struct {
		Objects []map[string]any `json:"objects"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Objects) != 1 {
		t.Fatalf("objects = %d, want 1", len(resp.Objects))
	}
	if _, has := resp.Objects[0]["contentType"]; has {
		t.Fatalf("列表项含幻影字段 contentType（值恒为空串）: %v", resp.Objects[0])
	}
	for _, f := range []string{"key", "size", "lastModified", "etag", "storageClass", "isDir"} {
		if _, ok := resp.Objects[0][f]; !ok {
			t.Errorf("列表项缺少字段 %s: %v", f, resp.Objects[0])
		}
	}
}

// TestOpenAPIObjectItemSchemaOmitsContentType ObjectItem 共享 schema 与真实 DTO
// 同步删除 contentType（两端同删，TestOpenAPI_ResponseSchemasMatchDTOs 双向比对保持绿）。
func TestOpenAPIObjectItemSchemaOmitsContentType(t *testing.T) {
	props := schemaProps(t, openAPIDoc(t), "ObjectItem")
	if props["contentType"] {
		t.Fatal("components.schemas.ObjectItem 仍声明 contentType（幻影字段）")
	}
	for _, f := range []string{"key", "size", "lastModified", "etag", "storageClass", "isDir"} {
		if !props[f] {
			t.Errorf("ObjectItem schema 缺少字段 %s", f)
		}
	}
}

// ---- R18：/api/metrics 是唯一指标出口，包内不得再向 expvar 发布 ----

// 见 acc_core_test.go 的 TestAccMetricsPublishesNoExpvar（负向断言：两键必须未发布）。

// ---- R19a：presign.go 不得保留仅测试引用的哨兵错误（死代码） ----

// TestPresignGoHasNoTestSentinel 源码级门禁：生产文件 presign.go 出现 errTestPresign 即红，
// 测试侧改为内联 errors.New 构造错误。
func TestPresignGoHasNoTestSentinel(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(handlerSourceDir(t), "presign.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "errTestPresign") {
		t.Fatal("presign.go 仍定义 errTestPresign（生产代码死代码；测试应内联构造错误）")
	}
}

// ---- R20：copy-objects 是长耗时批量端点，必须挂流式并发限（饱和 503） ----

// TestCopyManyStreamLimit503 并发槽占满时 POST /copy-objects 回 503；
// 释放槽后进入业务校验（缺 keys → 400）。旧实现未挂载 → 饱和时直接 400（红）。
func TestCopyManyStreamLimit503(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	for i := 0; i < maxConcurrentStreams; i++ {
		streamSlots <- struct{}{}
	}
	defer drainSlots(t)
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-objects", "{}")
	olExpectStatus(t, rr, http.StatusServiceUnavailable, "copy-objects 饱和时应 503")

	drainSlots(t)
	rr2 := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-objects", "{}")
	olExpectStatus(t, rr2, http.StatusBadRequest, "释放槽后进入业务校验")
}

// ---- Nit：migrate 系列把 store 故障误报 404（应 500；ErrNotFound 仍是 404） ----

// TestMigrateStoreFailureReturns500 三个迁移端点 × 源/目标 store 读取故障 = 6 案，
// 全部必须 500 且带载明原因的错误消息。旧实现一律 404（红）。
// 既有「账号不存在 → 404」用例（真实 store 返回 ErrNotFound）另行保持绿。
func TestMigrateStoreFailureReturns500(t *testing.T) {
	good := &model.Account{
		Name: "good", Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	}
	// /api/migrate 与 /async 的请求体要求非空 sourceKeys；/sync 的 DTO 无该字段
	//（DisallowUnknownFields，带上会 400），故两者体不同。
	migrateBody := `{"sourceAccountId":"src","targetAccountId":"dst","sourceKeys":["a.txt"]}`
	syncBody := `{"sourceAccountId":"src","targetAccountId":"dst"}`
	cases := []struct {
		name, path, body, failID, wantMsg string
	}{
		{"migrate source", "/api/migrate", migrateBody, "src", "failed to load source account"},
		{"migrate target", "/api/migrate", migrateBody, "dst", "failed to load target account"},
		{"migrate async source", "/api/migrate/async", migrateBody, "src", "failed to load source account"},
		{"migrate async target", "/api/migrate/async", migrateBody, "dst", "failed to load target account"},
		{"migrate sync source", "/api/migrate/sync", syncBody, "src", "failed to load source account"},
		{"migrate sync target", "/api/migrate/sync", syncBody, "dst", "failed to load target account"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := &accStubStore{getAcc: good, getErrs: map[string]error{c.failID: io.EOF}}
			h := accNewHandler(t, st, nil, "")
			rr := doJSON(t, h.Routes(), "POST", c.path, c.body)
			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("store 故障 status = %d, want 500（误报 404 会让用户去查不存在的账号）, body=%s",
					rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), c.wantMsg) {
				t.Fatalf("body = %s, want contains %q", rr.Body.String(), c.wantMsg)
			}
		})
	}
}

// TestAccountClientStoreFailureReturns500 accountClient 是对象/桶/代理等一大批端点的
// 公共前缀：store 读取故障必须 500（与 migrate 系列同一缺陷同一修复），
// 只有 ErrNotFound 才是 404。旧实现对任何错误一律 404（红）。
func TestAccountClientStoreFailureReturns500(t *testing.T) {
	// getAcc 留空：acc-nope 走 ErrNotFound 分支（404），acc-1 走故障分支（500）。
	st := &accStubStore{getErrs: map[string]error{"acc-1": io.EOF}}
	h := accNewHandler(t, st, nil, "")

	rr := doJSON(t, h.Routes(), "GET", "/api/accounts/acc-1/buckets", "")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("store 故障 status = %d, want 500（误报 404 会让用户去查不存在的账号）, body=%s",
			rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "failed to load account") {
		t.Fatalf("body = %s, want contains %q", rr.Body.String(), "failed to load account")
	}

	// 真正不存在（ErrNotFound）→ 仍是 404。
	rr2 := doJSON(t, h.Routes(), "GET", "/api/accounts/acc-nope/buckets", "")
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("账号不存在 status = %d, want 404, body=%s", rr2.Code, rr2.Body.String())
	}
}

// ---- Nit：copyMany 移动分支响应形状与 copyBatchJSON 漂移（缺 truncated） ----

// TestCopyManyBranchResponseShapeParity 两分支都带同一个失败样例时，响应 JSON 键集
// 必须完全一致（否则 openapi 契约「failed 处必有 truncated」只兑现一半）。
func TestCopyManyBranchResponseShapeParity(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			// 目标路径含 bad 的对象复制失败；其余成功。
			if strings.Contains(r.URL.Path, "bad") {
				return olErr(http.StatusForbidden, "AccessDenied")
			}
			return olPlain(http.StatusOK)
		}
		return olPlain(http.StatusNoContent)
	})
	env := accNewEnv(t, srv.URL, "b")
	id := env.acc.ID
	body := `{"bucket":"b","keys":["dir/good.txt","dir/bad.txt"]}`

	copyRR := env.accDoRec("POST", "/api/accounts/"+id+"/copy-objects", body)
	olExpectStatus(t, copyRR, http.StatusOK, "copy 分支")
	moveRR := env.accDoRec("POST", "/api/accounts/"+id+"/copy-objects", body[:len(body)-1]+`,"deleteSource":true}`)
	olExpectStatus(t, moveRR, http.StatusOK, "move 分支")

	var copyM, moveM map[string]any
	_ = json.Unmarshal(copyRR.Body.Bytes(), &copyM)
	_ = json.Unmarshal(moveRR.Body.Bytes(), &moveM)
	// 失败样例必须真的失败，否则两分支都「全成功」，键集比对无意义。
	if copyM["failed"] != float64(1) || moveM["failed"] != float64(1) {
		t.Fatalf("failed: copy=%v move=%v, want 1（失败样例未生效）", copyM["failed"], moveM["failed"])
	}
	if !sameJSONKeys(copyM, moveM) {
		t.Fatalf("copy/move 分支响应键集漂移: copy=%v move=%v（map 输出按键排序）", copyM, moveM)
	}
}

// sameJSONKeys 比较两个 JSON 对象的键集合是否完全一致。
func sameJSONKeys(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// ---- Nit：CSP connect-src 默认值字面量只允许维护一处 ----

// TestCSPConnectSrcLiteralHasSingleSource 源码级门禁：默认 connect-src 字面量在
// 非测试 .go 文件中必须恰好出现 1 次（抽常量共享；两处维护必漂移）。
func TestCSPConnectSrcLiteralHasSingleSource(t *testing.T) {
	const lit = "'self' http://127.0.0.1:* http://localhost:*"
	dir := handlerSourceDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	checked, total := 0, 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		checked++
		total += strings.Count(string(b), lit)
	}
	if checked == 0 {
		t.Fatal("no production .go files scanned; gate is not working")
	}
	if total != 1 {
		t.Fatalf("默认 connect-src 字面量在生产代码出现 %d 次，want 1（多处字面量维护会漂移，应抽常量共享）", total)
	}
}

// TestServerCSPSharesTauriDirectives 纯守卫（预期恒绿）：server CSP 与桌面壳
// tauri.conf.json 的 CSP 在**共享指令**上取值必须一致（connect-src 除外：
// server 收紧到本地回环，壳侧允许 http:/https:），防止两侧静默漂移。
func TestServerCSPSharesTauriDirectives(t *testing.T) {
	h, _ := gapStoreHandler(t)
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	resp, err := srv.Client().Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	serverCSP := resp.Header.Get("Content-Security-Policy")
	if serverCSP == "" {
		t.Fatal("server 响应缺少 Content-Security-Policy")
	}

	confPath := filepath.Join(handlerSourceDir(t), "..", "..", "..", "..",
		"apps", "desktop", "src-tauri", "tauri.conf.json")
	raw, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("读取 tauri.conf.json: %v", err)
	}
	var conf struct {
		App struct {
			Security struct {
				CSP string `json:"csp"`
			} `json:"security"`
		} `json:"app"`
	}
	if err := json.Unmarshal(raw, &conf); err != nil {
		t.Fatalf("解析 tauri.conf.json: %v", err)
	}
	if conf.App.Security.CSP == "" {
		t.Fatal("tauri.conf.json 缺少 app.security.csp")
	}

	serverD := parseCSPDirectives(serverCSP)
	tauriD := parseCSPDirectives(conf.App.Security.CSP)
	if _, ok := serverD["connect-src"]; !ok {
		t.Fatal("server CSP 缺少 connect-src")
	}
	if _, ok := tauriD["connect-src"]; !ok {
		t.Fatal("tauri CSP 缺少 connect-src")
	}
	shared := 0
	for d, tv := range tauriD {
		if d == "connect-src" {
			continue // 两侧取值刻意不同（server 收紧、壳侧放宽），不参与比对。
		}
		sv, ok := serverD[d]
		if !ok {
			continue // 非共享指令不比对（如 server 独有的 form-action）。
		}
		shared++
		if sv != tv {
			t.Errorf("共享 CSP 指令 %s 漂移: server=%q tauri=%q", d, sv, tv)
		}
	}
	if shared < 8 {
		t.Fatalf("只比对了 %d 条共享指令（want ≥8），防漂移门禁疑似失效", shared)
	}
}

// parseCSPDirectives 把 CSP 字符串解析为「指令 → 值（不含指令名）」。
func parseCSPDirectives(csp string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(csp, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, val, _ := strings.Cut(part, " ")
		out[name] = strings.TrimSpace(val)
	}
	return out
}

// ---- Nit：getBucketInfo 串行 3 次上游调用 ----

// TestBucketInfoNoSuchBucketShortCircuits location 返回桶不存在时：
// 直接 200 + 空属性，且**只**打 1 次上游（旧实现 3 次 → 红）。
func TestBucketInfoNoSuchBucketShortCircuits(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	srv := olFake(t, func(r *http.Request) olResp {
		op := accBucketsOp(r)
		mu.Lock()
		calls[op]++
		mu.Unlock()
		switch op {
		case "location":
			return olErr(http.StatusNotFound, "NoSuchBucket")
		case "versioning":
			return olXML(http.StatusOK, `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`)
		case "list":
			return olXML(http.StatusOK, accListBucketsXML)
		default:
			return olPlain(http.StatusOK)
		}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/bucket-info?bucket=b", "")
	olExpectStatus(t, rr, http.StatusOK, "桶不存在仍回 200（属性未知置空）")
	var info map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &info)
	if info["region"] != "" || info["versioning"] != "" || info["createdAt"] != "" {
		t.Fatalf("桶不存在时属性应置空: %v", info)
	}
	mu.Lock()
	total := calls["location"] + calls["versioning"] + calls["list"]
	snapshot := fmt.Sprintf("%v", calls)
	mu.Unlock()
	if total != 1 {
		t.Fatalf("上游共调用 %d 次（%s），桶不存在时应在 location 短路只调 1 次", total, snapshot)
	}
}

// TestBucketInfoVersioningAndListInParallel versioning 与 ListBuckets 必须**并行**发起：
// 两者都阻塞在 release 上，若实现串行则第二个上游调用 2s 内不会到达（红，约 2s）；
// 并行实现下两个调用几乎同时到达，测试瞬时完成。
func TestBucketInfoVersioningAndListInParallel(t *testing.T) {
	entered := make(chan string, 2)
	release := make(chan struct{})
	var once sync.Once
	srv := olFake(t, func(r *http.Request) olResp {
		switch accBucketsOp(r) {
		case "location":
			return olXML(http.StatusOK, `<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`)
		case "versioning":
			entered <- "versioning"
			<-release
			return olXML(http.StatusOK, `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></VersioningConfiguration>`)
		case "list":
			entered <- "list"
			<-release
			return olXML(http.StatusOK, accListBucketsXML)
		default:
			return olPlain(http.StatusOK)
		}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		env.h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
			"/api/accounts/"+env.acc.ID+"/bucket-info?bucket=b", nil))
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			// 串行实现会死锁在这里：先放行已阻塞的上游调用，让 handler 正常收尾，
			// 再报错（不留挂起的 goroutine / httptest 连接）。
			once.Do(func() { close(release) })
			<-done
			t.Fatalf("versioning 与 list 未并行到达上游（第 %d 个调用 2s 未开始，串行实现会互相等待）", i+1)
		}
	}
	once.Do(func() { close(release) })
	<-done
	olExpectStatus(t, rr, http.StatusOK, "bucket-info 并行拉取")
}
