package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
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

// TestMetricsEndpointUnauthenticatedEvenWithToken 固定「设计如此」的行为（todolist #30）：
// 即使配置了 S3C_TOKEN，/api/metrics 也**免鉴权**（内部 Prometheus 需在无 Bearer 的情况下
// scrape），仅由 S3C_EXPOSE_METRICS 门控；其余 /api/* 仍强制鉴权。若未来要改鉴权行为，
// 本测试必须先被有意识地改写——不要为「修安全项」而悄悄给 metrics 加鉴权。
func TestMetricsEndpointUnauthenticatedEvenWithToken(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "unit-test-token-0123456789", "test", true, false).Routes()

	// 无 Authorization：/api/metrics 仍 200（设计如此：内部 scrape 免鉴权）。
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/metrics without token = %d, want 200（设计：metrics 免鉴权）", rr.Code)
	}

	// 对照：同 token 下 /api/accounts 无 Authorization 必须 401。
	rrAcc := httptest.NewRecorder()
	h.ServeHTTP(rrAcc, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	if rrAcc.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/accounts without token = %d, want 401", rrAcc.Code)
	}

	// 对照：/api/health 同样免鉴权。
	rrHealth := httptest.NewRecorder()
	h.ServeHTTP(rrHealth, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rrHealth.Code != http.StatusOK {
		t.Fatalf("GET /api/health without token = %d, want 200", rrHealth.Code)
	}
}

// TestMetricsHistogramSatisfiesPrometheusContract 端到端固定 D1 的修复：
// /api/metrics 输出的 `_bucket{le=...}` 必须满足 Prometheus 直方图契约——le 单调不减、
// `+Inf` 等于 `_count`。旧实现把「每桶增量」当累积输出，实测 le="0.01"=3 / le="+Inf"=0 /
// _count=3，`histogram_quantile()` 结果全错（review-2026-09-19.md §7.3 D1）。
func TestMetricsHistogramSatisfiesPrometheusContract(t *testing.T) {
	// 假 S3：列出桶成功 → 触发一次上游调用，使直方图非空。
	srv := olFake(t, func(r *http.Request) olResp {
		return olXML(http.StatusOK, `<?xml version="1.0"?><ListAllMyBucketsResult><Buckets><Bucket><Name>b1</Name><CreationDate>2026-09-01T00:00:00Z</CreationDate></Bucket></Buckets></ListAllMyBucketsResult>`)
	})
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	acc, err := st.Create(&model.Account{
		Name: "acc-h", Endpoint: srv.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b1", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", true, false).Routes()
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/accounts/"+acc.ID+"/buckets", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("list buckets = %d, want 200: %s", rr.Code, rr.Body.String())
		}
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/metrics", nil))
	body := rr.Body.String()

	// 解析所有 `s3c_s3_call_duration_seconds_bucket{le="X"} N`，按输出顺序校验单调不减。
	bucketRe := regexp.MustCompile(`s3c_s3_call_duration_seconds_bucket\{le="([^"]+)"\} (\d+)`)
	matches := bucketRe.FindAllStringSubmatch(body, -1)
	if len(matches) < 2 {
		t.Fatalf("未解析到直方图桶（口径需同步）：\n%s", body)
	}
	var prev int64
	var infCount, sawInf int64
	for _, m := range matches {
		n, err := strconv.ParseInt(m[2], 10, 64)
		if err != nil {
			t.Fatalf("解析桶计数 %q: %v", m[2], err)
		}
		if n < prev {
			t.Errorf("le=%s 的桶计数 %d < 前一个 %d：违反 le 单调不减", m[1], n, prev)
		}
		prev = n
		if m[1] == "+Inf" {
			infCount, sawInf = n, 1
		}
	}
	if sawInf == 0 {
		t.Fatal("缺少 le=\"+Inf\" 桶")
	}
	countRe := regexp.MustCompile(`s3c_s3_call_duration_seconds_count (\d+)`)
	cm := countRe.FindStringSubmatch(body)
	if cm == nil {
		t.Fatalf("缺少 _count（口径需同步）：\n%s", body)
	}
	count, err := strconv.ParseInt(cm[1], 10, 64)
	if err != nil {
		t.Fatalf("解析 _count: %v", err)
	}
	if count == 0 {
		t.Fatal("_count 为 0：本用例已触发请求，解析口径可能失效")
	}
	if infCount != count {
		t.Errorf("+Inf 桶 = %d 但 _count = %d：Prometheus 要求两者相等（D1）", infCount, count)
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
