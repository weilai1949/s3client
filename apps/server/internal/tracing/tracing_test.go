package tracing

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMain 静音测试期日志（导出失败路径会打 WARN，属预期行为，不必污染测试输出）。
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func hexOf(b []byte) string { return hex.EncodeToString(b) }

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeCollector 是一个最小 OTLP/HTTP 接收端：记录每次导出的报文、路径、Content-Type，
// 按配置状态码响应。payload 经 buffered channel 暴露给测试。
type fakeCollector struct {
	mu       sync.Mutex
	payloads []map[string]any
	requests int
	lastPath string
	lastCT   string
	status   int
	ch       chan map[string]any
}

func newFakeCollector(status int) (*fakeCollector, *httptest.Server) {
	c := &fakeCollector{status: status, ch: make(chan map[string]any, 64)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p map[string]any
		_ = json.Unmarshal(body, &p)
		c.mu.Lock()
		c.requests++
		c.lastPath = r.URL.Path
		c.lastCT = r.Header.Get("Content-Type")
		c.payloads = append(c.payloads, p)
		c.mu.Unlock()
		select {
		case c.ch <- p:
		default:
		}
		w.WriteHeader(c.status)
	}))
	return c, srv
}

func (c *fakeCollector) requestCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requests
}

func waitPayload(t *testing.T, c *fakeCollector) map[string]any {
	t.Helper()
	select {
	case p := <-c.ch:
		return p
	case <-time.After(3 * time.Second):
		t.Fatal("collector 未在 3s 内收到导出请求")
		return nil
	}
}

// spansOf 从 OTLP JSON 报文中取出第一个 scopeSpans 的 span 列表。
func spansOf(t *testing.T, payload map[string]any) []map[string]any {
	t.Helper()
	rs, ok := payload["resourceSpans"].([]any)
	if !ok || len(rs) == 0 {
		t.Fatalf("payload 缺少 resourceSpans: %v", payload)
	}
	res, ok := rs[0].(map[string]any)
	if !ok {
		t.Fatalf("resourceSpans[0] 形状异常: %v", rs[0])
	}
	ss, ok := res["scopeSpans"].([]any)
	if !ok || len(ss) == 0 {
		t.Fatalf("resourceSpans[0] 缺少 scopeSpans: %v", res)
	}
	scope, ok := ss[0].(map[string]any)
	if !ok {
		t.Fatalf("scopeSpans[0] 形状异常: %v", ss[0])
	}
	raw, ok := scope["spans"].([]any)
	if !ok {
		t.Fatalf("scopeSpans[0] 缺少 spans: %v", scope)
	}
	out := make([]map[string]any, 0, len(raw))
	for _, s := range raw {
		out = append(out, s.(map[string]any))
	}
	return out
}

// spanAttrs 把 OTLP attributes 数组摊平成 key → 值（stringValue / intValue 都按字符串读）。
func spanAttrs(sp map[string]any) map[string]any {
	out := map[string]any{}
	raw, _ := sp["attributes"].([]any)
	for _, a := range raw {
		m := a.(map[string]any)
		v, ok := m["value"].(map[string]any)
		if !ok {
			continue
		}
		for _, k := range []string{"stringValue", "intValue"} {
			if val, ok := v[k]; ok {
				out[m["key"].(string)] = val
			}
		}
	}
	return out
}

func resourceServiceName(t *testing.T, payload map[string]any) string {
	t.Helper()
	rs := payload["resourceSpans"].([]any)
	res := rs[0].(map[string]any)["resource"].(map[string]any)
	for _, a := range res["attributes"].([]any) {
		m := a.(map[string]any)
		if m["key"] == "service.name" {
			return m["value"].(map[string]any)["stringValue"].(string)
		}
	}
	return ""
}

func newTracerForTest(t *testing.T, cfg Config) *Tracer {
	t.Helper()
	if cfg.ServiceName == "" {
		cfg.ServiceName = "test-svc"
	}
	tr, err := New(cfg)
	if err != nil {
		t.Fatalf("New(%+v): %v", cfg, err)
	}
	t.Cleanup(func() { closeTracer(t, tr) })
	return tr
}

func closeTracer(t *testing.T, tr *Tracer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := tr.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestDisabledTracerIsZeroOverhead Endpoint 为空 = 禁用：Middleware 原样透传、不注入
// 响应头、Start 返回原 ctx、Close 无副作用。
func TestDisabledTracerIsZeroOverhead(t *testing.T) {
	tr, err := New(Config{})
	if err != nil {
		t.Fatalf("New(empty): %v", err)
	}
	if tr.Enabled() {
		t.Fatal("Endpoint 为空必须禁用")
	}
	called := false
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if !called {
		t.Fatal("禁用时 Middleware 必须原样调用下游")
	}
	if got := rec.Header().Get("traceparent"); got != "" {
		t.Errorf("禁用时不得注入 traceparent，got %q", got)
	}
	ctx := context.Background()
	gotCtx, end := Start(ctx, "child")
	if gotCtx != ctx {
		t.Error("禁用（无 active span）时 Start 必须返回原 ctx")
	}
	end()
	if err := tr.Close(ctx); err != nil {
		t.Errorf("禁用时 Close = %v, want nil", err)
	}
}

// TestNewRejectsInvalidConfig New 对 Endpoint / SampleRatio 做边界校验。
func TestNewRejectsInvalidConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"无法解析的 endpoint", Config{Endpoint: "://bad"}},
		{"非 http(s) scheme", Config{Endpoint: "ftp://collector:4318"}},
		{"缺 host", Config{Endpoint: "http://"}},
		{"ratio > 1", Config{SampleRatio: 1.5}},
		{"ratio < 0", Config{SampleRatio: -0.1}},
		{"ratio NaN", Config{SampleRatio: math.NaN()}},
	}
	for _, c := range cases {
		if _, err := New(c.cfg); err == nil {
			t.Errorf("New(%s) = nil error, want error", c.name)
		}
	}
}

// TestMiddlewareExportsServerSpan 采样命中时导出一份 server span：W3C traceparent 响应头、
// OTLP JSON 结构、resource service.name、请求方法/路径/状态码与 request id 属性。
func TestMiddlewareExportsServerSpan(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, ServiceName: "s3client", SampleRatio: 1})

	var tpHeader string
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tpHeader = w.Header().Get("traceparent")
		// 模拟 handler 链里的 withLogging：最终 request id 写在响应头。
		w.Header().Set("X-Request-ID", "req-abc")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush: %v", err)
		}
		// SetWriteDeadline 需要经 statusWriter.Unwrap 触达底层 conn；httptest.ResponseRecorder
		// 不支持 deadline，返回 ErrNotSupported 即为「Unwrap 已生效」。
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(time.Second))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.test/api/accounts", nil))

	tp, ok := parseTraceparent(tpHeader)
	if !ok {
		t.Fatalf("响应 traceparent 非法: %q", tpHeader)
	}
	if !tp.sampled {
		t.Error("SampleRatio=1 的新 trace 必须采样")
	}

	payload := waitPayload(t, c)
	spans := spansOf(t, payload)
	if len(spans) != 1 {
		t.Fatalf("导出 span 数 = %d, want 1", len(spans))
	}
	sp := spans[0]
	if sp["name"] != "HTTP GET" {
		t.Errorf("span name = %v, want HTTP GET", sp["name"])
	}
	if kind, _ := sp["kind"].(float64); kind != float64(spanKindServer) {
		t.Errorf("span kind = %v, want %d", sp["kind"], spanKindServer)
	}
	if sp["traceId"] != hexOf(tp.traceID[:]) {
		t.Errorf("span traceId = %v, want %s", sp["traceId"], hexOf(tp.traceID[:]))
	}
	if sp["parentSpanId"] != nil {
		t.Errorf("根 span 不应有 parentSpanId，got %v", sp["parentSpanId"])
	}
	for _, f := range []string{"startTimeUnixNano", "endTimeUnixNano"} {
		s, _ := sp[f].(string)
		if s == "" || strings.HasPrefix(s, "-") {
			t.Errorf("%s = %q, want 非空十进制纳秒字符串", f, s)
		}
	}
	attrs := spanAttrs(sp)
	if attrs["http.request.method"] != "GET" {
		t.Errorf("http.request.method = %v, want GET", attrs["http.request.method"])
	}
	if attrs["url.path"] != "/api/accounts" {
		t.Errorf("url.path = %v, want /api/accounts", attrs["url.path"])
	}
	if attrs["http.response.status_code"] != "200" {
		t.Errorf("http.response.status_code = %v, want 200", attrs["http.response.status_code"])
	}
	if attrs["request.id"] != "req-abc" {
		t.Errorf("request.id = %v, want req-abc", attrs["request.id"])
	}
	if got := resourceServiceName(t, payload); got != "s3client" {
		t.Errorf("service.name = %q, want s3client", got)
	}
	if c.lastPath != "/v1/traces" {
		t.Errorf("导出路径 = %q, want /v1/traces", c.lastPath)
	}
	if !strings.HasPrefix(c.lastCT, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", c.lastCT)
	}
}

// TestMiddlewareWithoutRequestIDOmitsAttribute 上游没有 request id 时不写该属性。
func TestMiddlewareWithoutRequestIDOmitsAttribute(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 1})
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/health", nil))
	spans := spansOf(t, waitPayload(t, c))
	if _, ok := spanAttrs(spans[0])["request.id"]; ok {
		t.Error("无 request id 时不应写 request.id 属性")
	}
}

// TestSamplingInheritsSampledFlag 入站 traceparent 采样位为 1 时，即使 SampleRatio=0 也继承采样，
// 并沿用上游 trace id / parent span id。
func TestSamplingInheritsSampledFlag(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, ServiceName: "svc", SampleRatio: 0})

	inbound := "00-" + testTraceIDHex + "-" + testSpanIDHex + "-01"
	var respTP string
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respTP = w.Header().Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req.Header.Set("traceparent", inbound)
	h.ServeHTTP(httptest.NewRecorder(), req)

	tp, ok := parseTraceparent(respTP)
	if !ok {
		t.Fatalf("响应 traceparent 非法: %q", respTP)
	}
	if !tp.sampled {
		t.Error("入站 sampled=1 必须继承采样（即使 SampleRatio=0）")
	}
	if hexOf(tp.traceID[:]) != testTraceIDHex {
		t.Errorf("trace id = %s, want 继承 %s", hexOf(tp.traceID[:]), testTraceIDHex)
	}
	spans := spansOf(t, waitPayload(t, c))
	if len(spans) != 1 {
		t.Fatalf("导出 span 数 = %d, want 1", len(spans))
	}
	if spans[0]["traceId"] != testTraceIDHex {
		t.Errorf("span traceId = %v, want %s", spans[0]["traceId"], testTraceIDHex)
	}
	if spans[0]["parentSpanId"] != testSpanIDHex {
		t.Errorf("span parentSpanId = %v, want %s", spans[0]["parentSpanId"], testSpanIDHex)
	}
}

// TestUnsampledInboundIsNotReSampled 入站采样位为 0 时尊上游决定，不做尾部采样（ratio=1 也不导出）。
func TestUnsampledInboundIsNotReSampled(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 1})

	inbound := "00-" + testTraceIDHex + "-" + testSpanIDHex + "-00"
	var respTP string
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respTP = w.Header().Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req.Header.Set("traceparent", inbound)
	h.ServeHTTP(httptest.NewRecorder(), req)

	tp, ok := parseTraceparent(respTP)
	if !ok {
		t.Fatalf("响应 traceparent 非法: %q", respTP)
	}
	if tp.sampled {
		t.Error("入站 sampled=0 不得被重新采样")
	}
	closeTracer(t, tr)
	if n := c.requestCount(); n != 0 {
		t.Errorf("未采样请求不应导出，collector 收到 %d 次请求", n)
	}
}

// TestRatioZeroNewTraceNotExported 无入站 traceparent 时由 SampleRatio 决定：0 → 不导出。
func TestRatioZeroNewTraceNotExported(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 0})
	var respTP string
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respTP = w.Header().Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	tp, ok := parseTraceparent(respTP)
	if !ok {
		t.Fatalf("响应 traceparent 非法: %q", respTP)
	}
	if tp.sampled {
		t.Error("SampleRatio=0 且无入站上下文时不得采样")
	}
	closeTracer(t, tr)
	if n := c.requestCount(); n != 0 {
		t.Errorf("未采样请求不应导出，collector 收到 %d 次请求", n)
	}
}

// TestInvalidTraceparentStartsNewTrace 非法入站 traceparent（全 0 trace id）按「新建 trace」处理。
func TestInvalidTraceparentStartsNewTrace(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 1})
	var respTP string
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respTP = w.Header().Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req.Header.Set("traceparent", "00-00000000000000000000000000000000-"+testSpanIDHex+"-01")
	h.ServeHTTP(httptest.NewRecorder(), req)

	tp, ok := parseTraceparent(respTP)
	if !ok {
		t.Fatalf("响应 traceparent 非法: %q", respTP)
	}
	if hexOf(tp.traceID[:]) == "00000000000000000000000000000000" {
		t.Error("非法入站上下文不得被继承")
	}
	spans := spansOf(t, waitPayload(t, c))
	if spans[0]["parentSpanId"] != nil {
		t.Errorf("新建 trace 的 span 不应有 parentSpanId，got %v", spans[0]["parentSpanId"])
	}
}

// TestStartCreatesChildSpan 包级 Start 在活跃 span 上下文里建子 span，parentSpanId 指向父 span。
func TestStartCreatesChildSpan(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 1})
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, end := Start(r.Context(), "presign")
		if ctx == r.Context() {
			t.Error("Start 应派生携带子 span 的 ctx")
		}
		end()
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/accounts/a/presign", nil))

	spans := spansOf(t, waitPayload(t, c))
	if len(spans) != 2 {
		t.Fatalf("导出 span 数 = %d, want 2（server + presign）", len(spans))
	}
	var server, child map[string]any
	for _, sp := range spans {
		if sp["name"] == "HTTP POST" {
			server = sp
		}
		if sp["name"] == "presign" {
			child = sp
		}
	}
	if server == nil || child == nil {
		t.Fatalf("缺少 server 或 presign span: %v", spans)
	}
	if child["parentSpanId"] != server["spanId"] {
		t.Errorf("子 span parentSpanId = %v, want %v", child["parentSpanId"], server["spanId"])
	}
	if child["traceId"] != server["traceId"] {
		t.Errorf("子 span traceId = %v, want %v", child["traceId"], server["traceId"])
	}
	if kind, _ := child["kind"].(float64); kind != float64(spanKindInternal) {
		t.Errorf("子 span kind = %v, want %d", child["kind"], spanKindInternal)
	}
}

// TestStartUnderUnsampledParentNotExported 父 span 未采样时子 span 同样不导出。
func TestStartUnderUnsampledParentNotExported(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 0})
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, end := Start(r.Context(), "presign")
		end()
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/accounts/a/presign", nil))
	closeTracer(t, tr)
	if n := c.requestCount(); n != 0 {
		t.Errorf("未采样 trace 不应导出，collector 收到 %d 次请求", n)
	}
}

// TestExportFailureIsNonFatal 采集端不可达时请求照常完成，Close 不报错（失败只记日志）。
func TestExportFailureIsNonFatal(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := dead.URL
	dead.Close()
	tr := newTracerForTest(t, Config{Endpoint: endpoint, SampleRatio: 1})
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("采集端不可达不应影响请求，status = %d", rec.Code)
	}
	closeTracer(t, tr)
}

// TestExportNon2xxIsNonFatal 采集端返回 5xx 时同样不影响请求。
func TestExportNon2xxIsNonFatal(t *testing.T) {
	c, srv := newFakeCollector(http.StatusInternalServerError)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 1})
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("采集端 5xx 不应影响请求，status = %d", rec.Code)
	}
	closeTracer(t, tr)
	if n := c.requestCount(); n != 1 {
		t.Errorf("应尝试导出 1 次，实际 %d", n)
	}
}

// TestCloseFlushesQueueAndIsIdempotent Close 刷出队列，重复调用安全。
func TestCloseFlushesQueueAndIsIdempotent(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := newTracerForTest(t, Config{Endpoint: srv.URL, SampleRatio: 1})
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/health", nil))
	closeTracer(t, tr)
	if n := c.requestCount(); n != 1 {
		t.Fatalf("Close 应刷出 1 次导出，实际 %d", n)
	}
	closeTracer(t, tr) // 幂等
}

// TestTickerFlushesPendingSpans 后台 ticker 会把不足批量的 span 按时刷出（不依赖 Close）。
func TestTickerFlushesPendingSpans(t *testing.T) {
	old := flushInterval
	flushInterval = 20 * time.Millisecond
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr, err := New(Config{Endpoint: srv.URL, ServiceName: "svc", SampleRatio: 1})
	if err != nil {
		flushInterval = old
		t.Fatalf("New: %v", err)
	}
	// 空批量的 ticker 分支：先空转几拍。
	time.Sleep(3 * flushInterval)
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if spans := spansOf(t, waitPayload(t, c)); len(spans) != 1 {
		t.Fatalf("ticker 应刷出 1 个 span，实际 %d", len(spans))
	}
	closeTracer(t, tr)
	flushInterval = old
}

// TestBatchFlushesAtThreshold 达到批量阈值立即导出（不等 ticker）。
func TestBatchFlushesAtThreshold(t *testing.T) {
	oldQueue, oldBatch, oldInterval := queueCapacity, maxBatchSpans, flushInterval
	queueCapacity, maxBatchSpans, flushInterval = 16, 2, time.Hour
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr, err := New(Config{Endpoint: srv.URL, ServiceName: "svc", SampleRatio: 1})
	if err != nil {
		queueCapacity, maxBatchSpans, flushInterval = oldQueue, oldBatch, oldInterval
		t.Fatalf("New: %v", err)
	}
	h := tr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < maxBatchSpans; i++ {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/health", nil))
	}
	if spans := spansOf(t, waitPayload(t, c)); len(spans) != maxBatchSpans {
		t.Fatalf("达到阈值应导出 %d 个 span，实际 %d", maxBatchSpans, len(spans))
	}
	closeTracer(t, tr)
	queueCapacity, maxBatchSpans, flushInterval = oldQueue, oldBatch, oldInterval
}
