package handler

// tracing_wiring_test.go —— #11 接线证据：handler 内的 tracing.Start 子 span 确实
// 挂在请求 server span 之下，并把既有 request id 关联进 span 属性。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/tracing"
)

// TestTracingMiddlewareExportsPresignChildSpan 真实路由 + 假 S3：presign 请求导出
// server span 与 presign 子 span，子 span 的 parentSpanId 指向 server span，且带 request id。
func TestTracingMiddlewareExportsPresignChildSpan(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp { return olPlain(http.StatusOK) })
	env := accNewEnv(t, srv.URL, "b")

	payloads := make(chan map[string]any, 4)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p map[string]any
		_ = json.Unmarshal(body, &p)
		select {
		case payloads <- p:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()

	tr, err := tracing.New(tracing.Config{Endpoint: collector.URL, ServiceName: "s3client", SampleRatio: 1})
	if err != nil {
		t.Fatalf("tracing.New: %v", err)
	}
	h := tr.Middleware(env.hnd.Routes())

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/"+env.acc.ID+"/presign",
		strings.NewReader(`{"key":"k"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("presign status = %d, body=%s", rr.Code, rr.Body.String())
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("tracing.Close: %v", err)
	}

	var payload map[string]any
	select {
	case payload = <-payloads:
	case <-time.After(3 * time.Second):
		t.Fatal("collector 未收到 OTLP 导出")
	}

	spans := tracingSpansOf(t, payload)
	var server, child map[string]any
	for _, sp := range spans {
		switch sp["name"] {
		case "HTTP POST":
			server = sp
		case "presign":
			child = sp
		}
	}
	if server == nil || child == nil {
		t.Fatalf("缺少 HTTP POST / presign span: %v", spans)
	}
	if child["parentSpanId"] != server["spanId"] {
		t.Errorf("presign parentSpanId = %v, want %v", child["parentSpanId"], server["spanId"])
	}
	if child["traceId"] != server["traceId"] {
		t.Errorf("presign traceId = %v, want %v", child["traceId"], server["traceId"])
	}
	if got := tracingSpanAttr(server, "request.id"); got == "" {
		t.Error("server span 应带 withLogging 的 request id")
	}
	if got := tracingSpanAttr(server, "http.response.status_code"); got != "200" {
		t.Errorf("http.response.status_code = %q, want 200", got)
	}
}

// tracingSpansOf 从 OTLP JSON 报文取第一个 scopeSpans 的 span 列表。
func tracingSpansOf(t *testing.T, payload map[string]any) []map[string]any {
	t.Helper()
	rs, ok := payload["resourceSpans"].([]any)
	if !ok || len(rs) == 0 {
		t.Fatalf("报文缺少 resourceSpans: %v", payload)
	}
	ss := rs[0].(map[string]any)["scopeSpans"].([]any)
	raw := ss[0].(map[string]any)["spans"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, s := range raw {
		out = append(out, s.(map[string]any))
	}
	return out
}

// tracingSpanAttr 读取 span 的 stringValue / intValue 属性。
func tracingSpanAttr(sp map[string]any, key string) string {
	for _, a := range sp["attributes"].([]any) {
		m := a.(map[string]any)
		if m["key"] != key {
			continue
		}
		v := m["value"].(map[string]any)
		for _, k := range []string{"stringValue", "intValue"} {
			if s, ok := v[k].(string); ok {
				return s
			}
		}
	}
	return ""
}
