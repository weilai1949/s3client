package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

func TestMetricsEndpointExposed(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// 显式开启 metrics 暴露。
	h := New(st, logger, t.TempDir(), nil, "", "test", true, false).Routes()
	// 触发一次请求以累计计数
	rr0 := httptest.NewRecorder()
	h.ServeHTTP(rr0, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"s3c_http_requests_total", "s3c_uptime_seconds", "s3c_build_info", "s3c_stream_interrupted_total",
		// 上游与 ZIP 可观测性（已闭环：features.md §M）。
		"s3c_s3_calls_total", "s3c_s3_call_duration_seconds_bucket", "s3c_s3_stream_bytes_total",
		"s3c_zip_partial_failures_total", "s3c_zip_failed_keys_total", "s3c_zip_failed_total",
		// R8：存储硬失败与 SSRF 生效策略的可观测面。
		"s3c_store_up", "s3c_ssrf_deny_private",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID")
	}
}

// TestMetricsStoreUpAndSSRFPolicy R8 的可观测面：store 可达性与 SSRF 生效策略都要能
// 从 /api/metrics 读到，前者用于告警「store 掉线」（ADR-002 硬失败），后者用于核对
// S3C_SSRF_DENY_PRIVATE 是否真的生效（ADR-003 可选加固）。
func TestMetricsStoreUpAndSSRFPolicy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 存储不可达 + SSRF 严格模式。
	s3wrap.SetDenyPrivateNetworks(true)
	t.Cleanup(func() { s3wrap.SetDenyPrivateNetworks(false) })
	h := New(&failPingStore{}, logger, t.TempDir(), nil, "", "test", true, false).Routes()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/metrics", nil))
	body := rr.Body.String()
	if !strings.Contains(body, "s3c_store_up 0") {
		t.Fatalf("store 不可达时应输出 s3c_store_up 0，got:\n%s", body)
	}
	if !strings.Contains(body, "s3c_ssrf_deny_private 1") {
		t.Fatalf("S3C_SSRF_DENY_PRIVATE 生效时应输出 s3c_ssrf_deny_private 1，got:\n%s", body)
	}

	// 存储可达 + 默认（放行私网）。
	s3wrap.SetDenyPrivateNetworks(false)
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	h2 := New(st, logger, t.TempDir(), nil, "", "test", true, false).Routes()
	rr2 := httptest.NewRecorder()
	h2.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/api/metrics", nil))
	body2 := rr2.Body.String()
	for _, want := range []string{"s3c_store_up 1", "s3c_ssrf_deny_private 0"} {
		if !strings.Contains(body2, want) {
			t.Fatalf("默认配置应输出 %q，got:\n%s", want, body2)
		}
	}
}

// 默认（exposeMetrics=false）下 /api/metrics 须返回 404，假装端点不存在，
// 避免公网被 scrape 运行指标（PROBE 探测也只看到 404）。
func TestMetricsEndpointHiddenByDefault(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/metrics", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rr.Code)
	}
}
