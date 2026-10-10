package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/service"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

func TestMetricsEndpointExposed(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// 显式开启 metrics 暴露。
	handler := New(st, logger, t.TempDir(), nil, "", "test", true, false)
	// 卷容量指标取 S3C_DATA_DIR 的 statfs 结果：不设目录则该序列不输出（口径见下）。
	handler.SetDataDir(t.TempDir())
	h := handler.Routes()
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
		// 上游与 ZIP 可观测性（已闭环：FEATURES.md §M）。
		"s3c_s3_calls_total", "s3c_s3_call_duration_seconds_bucket", "s3c_s3_stream_bytes_total",
		"s3c_zip_partial_failures_total", "s3c_zip_failed_keys_total", "s3c_zip_failed_total",
		// R8：存储硬失败与 SSRF 生效策略的可观测面。
		"s3c_store_up", "s3c_ssrf_deny_private",
		// ROADMAP #18 补齐的五个观测缺口（2026-09-30）。
		"s3c_store_write_failures_total", "s3c_jobs_active",
		"s3c_http_request_duration_seconds_bucket", "s3c_http_request_duration_seconds_sum",
		"s3c_volume_size_bytes", "s3c_volume_free_bytes", "s3c_last_shutdown_duration_seconds",
		// 数据卷 inode（OPERATIONS.md §4.3 原「无内置指标」项，取不到时不发序列）。
		"s3c_volume_inode_total", "s3c_volume_inode_free",
		// KNOWN_ISSUES #83：计划 / 任务清单落盘失败（原为静默）。
		"s3c_persist_failures_total",
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

// TestMetricsEndpointUnauthenticatedEvenWithToken 固定「设计如此」的行为（KNOWN_ISSUES #30）：
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
// _count=3，`histogram_quantile()` 结果全错（docs/archive/review-2026-09-19.md §7.3 D1）。
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

// ---- ROADMAP #18：五个缺失指标（2026-09-30 补齐，观测缺口声明同步撤除） ----

// metricsBody 请求一次 /api/metrics 并返回正文（要求 200）。
func metricsBody(t *testing.T, h http.Handler) string {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/metrics = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	return rr.Body.String()
}

// metricValue 取 Prometheus 文本里**无标签**序列的当前值；序列缺失返回 found=false。
func metricValue(t *testing.T, body, name string) (v float64, found bool) {
	t.Helper()
	m := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + ` (\S+)$`).FindStringSubmatch(body)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("%s 的取值 %q 不是数字：%v", name, m[1], err)
	}
	return v, true
}

// mustMetric 要求序列存在并返回其值（缺失即 Fatal）。
func mustMetric(t *testing.T, body, name string) float64 {
	t.Helper()
	v, found := metricValue(t, body, name)
	if !found {
		t.Fatalf("缺少指标 %s：\n%s", name, body)
	}
	return v
}

// httpDurBucketRe 匹配 HTTP 延迟直方图的桶行（保持输出顺序）。
var httpDurBucketRe = regexp.MustCompile(`(?m)^s3c_http_request_duration_seconds_bucket\{le="([^"]+)"\} (\d+)$`)

// parseHTTPDuration 校验 HTTP 延迟直方图满足 Prometheus 契约（le 单调不减、
// `le="+Inf"` 是最后一个桶且等于 `_count`），返回 `_count` / `_sum` / `s3c_http_requests_total`。
func parseHTTPDuration(t *testing.T, body string) (count, sum, total float64) {
	t.Helper()
	matches := httpDurBucketRe.FindAllStringSubmatch(body, -1)
	if len(matches) < 2 {
		t.Fatalf("未解析到 s3c_http_request_duration_seconds 的桶（指标缺失或口径失效）：\n%s", body)
	}
	var prev float64
	var inf float64
	for i, m := range matches {
		n, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			t.Fatalf("解析桶计数 %q: %v", m[2], err)
		}
		if n < prev {
			t.Errorf("le=%s 的桶计数 %g < 前一个 %g：违反 le 单调不减", m[1], n, prev)
		}
		prev = n
		if m[1] == "+Inf" {
			if i != len(matches)-1 {
				t.Errorf(`le="+Inf" 出现在第 %d/%d 个桶（必须是最后一个）`, i+1, len(matches))
			}
			inf = n
		}
	}
	if matches[len(matches)-1][1] != "+Inf" {
		t.Fatalf("最后一个桶是 le=%q，应为 +Inf", matches[len(matches)-1][1])
	}
	count = mustMetric(t, body, "s3c_http_request_duration_seconds_count")
	sum = mustMetric(t, body, "s3c_http_request_duration_seconds_sum")
	total = mustMetric(t, body, "s3c_http_requests_total")
	if inf != count {
		t.Errorf(`le="+Inf" 桶 = %g 但 _count = %g：Prometheus 要求两者相等`, inf, count)
	}
	if count != total {
		t.Errorf("_count = %g 但 s3c_http_requests_total = %g：两者必须同一次请求同步增长", count, total)
	}
	return count, sum, total
}

// TestMetricsHTTPDurationHistogram ROADMAP #18 指标③：HTTP 请求延迟直方图。
// 每个请求都要进桶并累加 _sum / _count，且与 s3c_http_requests_total 恒等。
func TestMetricsHTTPDurationHistogram(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", true, false).Routes()

	c1, sum1, _ := parseHTTPDuration(t, metricsBody(t, h))
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/health", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("health = %d, want 200", rr.Code)
		}
	}
	c2, sum2, _ := parseHTTPDuration(t, metricsBody(t, h))
	// 3 次 health + 中间那次 metrics scrape 本身也计入（scrape 的响应先写出、后计数）。
	if c2-c1 < 3 {
		t.Errorf("_count 增量 = %g，至少应有 3 次 health 请求入桶", c2-c1)
	}
	if sum2 <= sum1 {
		t.Errorf("_sum 未随请求增长：%g -> %g", sum1, sum2)
	}
}

// TestMetricsJobsActive ROADMAP #18 指标②：在册（未终结）异步任务数。
// 上限语义与 JobRegistry.TryCreate 相同——只数未终结任务，故任务终态后必须回落。
func TestMetricsJobsActive(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(st, logger, t.TempDir(), nil, "", "test", true, false)
	t.Cleanup(handler.Shutdown)
	h := handler.Routes()

	if v := mustMetric(t, metricsBody(t, h), "s3c_jobs_active"); v != 0 {
		t.Fatalf("无在册任务时 s3c_jobs_active = %g, want 0", v)
	}
	_, cancel := context.WithCancel(context.Background())
	job, err := handler.migrateJobs.TryCreate(2, cancel)
	if err != nil {
		t.Fatalf("注册测试任务: %v", err)
	}
	if v := mustMetric(t, metricsBody(t, h), "s3c_jobs_active"); v != 1 {
		t.Errorf("在册 1 个任务时 s3c_jobs_active = %g, want 1", v)
	}
	job.Finish(service.JobResult{Migrated: 2}, service.JobStatusDone)
	if v := mustMetric(t, metricsBody(t, h), "s3c_jobs_active"); v != 0 {
		t.Errorf("任务进入终态后 s3c_jobs_active = %g, want 0（终态任务不占在册名额）", v)
	}
}

// TestMetricsVolumeCapacity ROADMAP #18 指标④：卷 / 磁盘容量取自 S3C_DATA_DIR 的 statfs。
// 未配置目录或 statfs 失败时**不输出**该序列（没有可信数据就不发序列，不用哨兵值冒充）。
func TestMetricsVolumeCapacity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// ① 未 SetDataDir：容量序列缺失。
	noDir := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	noDirBody := metricsBody(t, noDir.Routes())
	if _, ok := metricValue(t, noDirBody, "s3c_volume_size_bytes"); ok {
		t.Errorf("未配置数据目录时不应输出 s3c_volume_size_bytes：\n%s", noDirBody)
	}

	// ② 配置真实目录：总容量 > 0 且可用 ≤ 总量。
	okHandler := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	okHandler.SetDataDir(t.TempDir())
	okBody := metricsBody(t, okHandler.Routes())
	size := mustMetric(t, okBody, "s3c_volume_size_bytes")
	free := mustMetric(t, okBody, "s3c_volume_free_bytes")
	if size <= 0 {
		t.Errorf("s3c_volume_size_bytes = %g, want > 0：\n%s", size, okBody)
	}
	if free < 0 || free > size {
		t.Errorf("s3c_volume_free_bytes = %g，应在 [0, %g] 内", free, size)
	}

	// ③ 目录不存在：statfs 失败 → 序列缺失（而非 0）。
	missing := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	missing.SetDataDir(filepath.Join(t.TempDir(), "does-not-exist"))
	missingBody := metricsBody(t, missing.Routes())
	if _, ok := metricValue(t, missingBody, "s3c_volume_size_bytes"); ok {
		t.Errorf("statfs 失败时不应输出容量序列：\n%s", missingBody)
	}
}

// TestMetricsVolumeInodes 断言数据卷 **inode** 序列（OPERATIONS.md §4.3 原「无内置指标」项）：
// 取自同一 statfs 的 Files / Ffree，口径与容量序列一致——**取不到就不发序列**，
// 不用 0 冒充「inode 总数为 0」（0 会让 `free/total` 变成 0/0=NaN，告警静默失效）。
func TestMetricsVolumeInodes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// ① 未 SetDataDir：inode 序列同样缺失（与容量序列同一开关）。
	noDir := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	noDirBody := metricsBody(t, noDir.Routes())
	if _, ok := metricValue(t, noDirBody, "s3c_volume_inode_total"); ok {
		t.Errorf("未配置数据目录时不应输出 s3c_volume_inode_total：\n%s", noDirBody)
	}

	// ② 配置真实目录：总数 > 0，空闲 ∈ [0, 总数]。
	okHandler := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	okHandler.SetDataDir(t.TempDir())
	okBody := metricsBody(t, okHandler.Routes())
	total := mustMetric(t, okBody, "s3c_volume_inode_total")
	free := mustMetric(t, okBody, "s3c_volume_inode_free")
	if total <= 0 {
		t.Errorf("s3c_volume_inode_total = %g, want > 0：\n%s", total, okBody)
	}
	if free < 0 || free > total {
		t.Errorf("s3c_volume_inode_free = %g，应在 [0, %g] 内", free, total)
	}

	// ③ 目录不存在：statfs 失败 → 两条 inode 序列都不发（而非 0）。
	missing := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	missing.SetDataDir(filepath.Join(t.TempDir(), "does-not-exist"))
	missingBody := metricsBody(t, missing.Routes())
	if _, ok := metricValue(t, missingBody, "s3c_volume_inode_total"); ok {
		t.Errorf("statfs 失败时不应输出 inode 序列：\n%s", missingBody)
	}
	if _, ok := metricValue(t, missingBody, "s3c_volume_inode_free"); ok {
		t.Errorf("statfs 失败时不应输出 inode 序列：\n%s", missingBody)
	}
}

// TestMetricsStoreWriteFailures ROADMAP #18 指标①：账号库写入失败次数。
// 只统计**真实写入失败**（落盘 / SQL 写失败，写操作已回滚），成功写入与业务性拒绝
// （重复 ID、NotFound）不计数——否则告警会被客户端的 4xx 触发。
func TestMetricsStoreWriteFailures(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "accounts.json")
	st, err := store.New(p)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", true, false).Routes()

	before := mustMetric(t, metricsBody(t, h), "s3c_store_write_failures_total")
	if _, err := st.Create(&model.Account{Name: "ok"}); err != nil {
		t.Fatalf("正常写入: %v", err)
	}
	afterOK := mustMetric(t, metricsBody(t, h), "s3c_store_write_failures_total")
	if afterOK != before {
		t.Errorf("成功写入后计数 %g -> %g，不应计数", before, afterOK)
	}

	// 落盘失败注入：把目标文件换成目录 → rename 失败 → 写操作回滚并计数。
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Create(&model.Account{Name: "boom"}); err == nil {
		t.Fatal("目标路径是目录时 Create 必须失败")
	}
	afterFail := mustMetric(t, metricsBody(t, h), "s3c_store_write_failures_total")
	if afterFail <= afterOK {
		t.Errorf("写入失败后计数 %g -> %g，应递增", afterOK, afterFail)
	}
}

// TestMetricsLastShutdownDuration ROADMAP #18 指标⑤：优雅关停耗时。
// 关停发生在进程退出前，指标无法被进程内 scrape 捕获 → 落盘到 data/shutdown.json，
// 下一次启动载入后作为「最近一次关停耗时」暴露（跨重启可读，口径见 OPERATIONS §3.2）。
func TestMetricsLastShutdownDuration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// ① 记录 → 落盘 → 新进程载入 → 指标可读。
	dir := t.TempDir()
	rec := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	rec.SetDataDir(dir)
	rec.RecordShutdown(150 * time.Millisecond)
	raw, err := os.ReadFile(filepath.Join(dir, "shutdown.json"))
	if err != nil {
		t.Fatalf("关停耗时未落盘: %v", err)
	}
	if !strings.Contains(string(raw), `"durationUs":150000`) {
		t.Errorf("shutdown.json = %s，应记录 150000µs（150ms）", raw)
	}
	next := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	next.SetDataDir(dir)
	next.LoadLastShutdown()
	if v := mustMetric(t, metricsBody(t, next.Routes()), "s3c_last_shutdown_duration_seconds"); v != 0.15 {
		t.Errorf("s3c_last_shutdown_duration_seconds = %g, want 0.15", v)
	}

	// ② 无历史记录（首次启动）→ 恒为 0。
	fresh := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	fresh.SetDataDir(t.TempDir())
	fresh.LoadLastShutdown()
	if v := mustMetric(t, metricsBody(t, fresh.Routes()), "s3c_last_shutdown_duration_seconds"); v != 0 {
		t.Errorf("无关停记录时 s3c_last_shutdown_duration_seconds = %g, want 0", v)
	}

	// ③ 未 SetDataDir：只更新内存值，不得把文件写进当前工作目录。
	noDir := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	noDir.RecordShutdown(10 * time.Millisecond)
	if _, err := os.Stat("shutdown.json"); !os.IsNotExist(err) {
		t.Errorf("未配置数据目录时不应写 shutdown.json（当前工作目录残留），stat err=%v", err)
	}
	if v := mustMetric(t, metricsBody(t, noDir.Routes()), "s3c_last_shutdown_duration_seconds"); v != 0.01 {
		t.Errorf("内存态关停耗时 = %g, want 0.01", v)
	}
	// 同样未设置目录时载入是空操作（不会去读相对当前工作目录的文件）。
	noDir.LoadLastShutdown()
	if v := mustMetric(t, metricsBody(t, noDir.Routes()), "s3c_last_shutdown_duration_seconds"); v != 0.01 {
		t.Errorf("载入后内存值被清掉 = %g, want 0.01", v)
	}

	// ④ 文件损坏：回退 0 而不是带着半截值启动（关停耗时绝不阻断启动）。
	corruptDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(corruptDir, "shutdown.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt := New(mustStore(t), logger, t.TempDir(), nil, "", "test", true, false)
	corrupt.SetDataDir(corruptDir)
	corrupt.LoadLastShutdown()
	if v := mustMetric(t, metricsBody(t, corrupt.Routes()), "s3c_last_shutdown_duration_seconds"); v != 0 {
		t.Errorf("shutdown.json 损坏时应读作 0，got %g", v)
	}
}
