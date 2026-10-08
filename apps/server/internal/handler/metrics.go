package handler

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

var (
	metricHTTPTotal atomic.Int64
	metricHTTP2xx   atomic.Int64
	metricHTTP4xx   atomic.Int64
	metricHTTP5xx   atomic.Int64
	// metricStreamInterrupted 统计流式传输在写出完成前中断的次数（上游读失败、
	// 写超时、客户端断开）。此前这类失败被 io.Copy 的返回值吞掉，无从观测（#20）。
	metricStreamInterrupted atomic.Int64
	// ZIP 打包可见性（已闭环：FEATURES.md §M）：部分失败次数、失败 key 累计、整体失败次数。
	metricZipPartialFailures atomic.Int64
	metricZipFailedKeys      atomic.Int64
	metricZipFailed          atomic.Int64
	metricStartedAt          = time.Now()
)

// httpDurBucketBounds 是 HTTP 请求延迟直方图的桶上界（秒，不含 +Inf）。
// 取值对齐常见 API 分位点并留出流式端点（proxy / download / zip 会跑到分钟级）的长尾，
// 输出时补 `le="+Inf"`（恒等于请求总数）。
var httpDurBucketBounds = [...]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

// metricHTTPDurBuckets 与 httpDurBucketBounds 一一对应，存**累积**桶计数
// （耗时 ≤ 上界即 +1），故输出天然满足 Prometheus 的 le 单调不减契约。
var metricHTTPDurBuckets [len(httpDurBucketBounds)]atomic.Int64

// metricHTTPDurSumNs 累计请求耗时（纳秒）。存整数避免原子浮点加法，输出时换算秒。
var metricHTTPDurSumNs atomic.Int64

// recordHTTPMetric 记一次请求的状态类与耗时。
// 同一次请求在同一处同时进计数器与直方图，保证 `s3c_http_request_duration_seconds_count`
// 与 `s3c_http_requests_total` 恒等（ROADMAP #18）。
func recordHTTPMetric(status int, dur time.Duration) {
	metricHTTPTotal.Add(1)
	switch {
	case status >= 500:
		metricHTTP5xx.Add(1)
	case status >= 400:
		metricHTTP4xx.Add(1)
	case status >= 200 && status < 300:
		metricHTTP2xx.Add(1)
	}
	metricHTTPDurSumNs.Add(dur.Nanoseconds())
	sec := dur.Seconds()
	for i, b := range httpDurBucketBounds {
		if sec <= b {
			metricHTTPDurBuckets[i].Add(1)
		}
	}
}

// metrics 暴露 Prometheus 文本格式指标（无需鉴权，便于内网 scrape）。
func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP s3c_http_requests_total Total HTTP requests handled\n")
	fmt.Fprintf(w, "# TYPE s3c_http_requests_total counter\n")
	fmt.Fprintf(w, "s3c_http_requests_total %d\n", metricHTTPTotal.Load())
	fmt.Fprintf(w, "# HELP s3c_http_responses_total HTTP responses by class\n")
	fmt.Fprintf(w, "# TYPE s3c_http_responses_total counter\n")
	fmt.Fprintf(w, "s3c_http_responses_total{class=\"2xx\"} %d\n", metricHTTP2xx.Load())
	fmt.Fprintf(w, "s3c_http_responses_total{class=\"4xx\"} %d\n", metricHTTP4xx.Load())
	fmt.Fprintf(w, "s3c_http_responses_total{class=\"5xx\"} %d\n", metricHTTP5xx.Load())
	writeHTTPDurationMetrics(w)
	fmt.Fprintf(w, "# HELP s3c_stream_interrupted_total Streams interrupted before completion (upstream read failure, write timeout, or client disconnect)\n")
	fmt.Fprintf(w, "# TYPE s3c_stream_interrupted_total counter\n")
	fmt.Fprintf(w, "s3c_stream_interrupted_total %d\n", metricStreamInterrupted.Load())
	fmt.Fprintf(w, "# HELP s3c_zip_partial_failures_total ZIP downloads that completed with at least one failed object\n")
	fmt.Fprintf(w, "# TYPE s3c_zip_partial_failures_total counter\n")
	fmt.Fprintf(w, "s3c_zip_partial_failures_total %d\n", metricZipPartialFailures.Load())
	fmt.Fprintf(w, "# HELP s3c_zip_failed_keys_total Objects that failed to be fetched during ZIP downloads\n")
	fmt.Fprintf(w, "# TYPE s3c_zip_failed_keys_total counter\n")
	fmt.Fprintf(w, "s3c_zip_failed_keys_total %d\n", metricZipFailedKeys.Load())
	fmt.Fprintf(w, "# HELP s3c_zip_failed_total ZIP downloads that failed to complete\n")
	fmt.Fprintf(w, "# TYPE s3c_zip_failed_total counter\n")
	fmt.Fprintf(w, "s3c_zip_failed_total %d\n", metricZipFailed.Load())
	fmt.Fprintf(w, "# HELP s3c_uptime_seconds Process uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE s3c_uptime_seconds gauge\n")
	fmt.Fprintf(w, "s3c_uptime_seconds %s\n", strconv.FormatInt(int64(time.Since(metricStartedAt).Seconds()), 10))
	fmt.Fprintf(w, "# HELP s3c_go_goroutines Number of goroutines\n")
	fmt.Fprintf(w, "# TYPE s3c_go_goroutines gauge\n")
	fmt.Fprintf(w, "s3c_go_goroutines %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "# HELP s3c_go_memstats_alloc_bytes Bytes allocated and still in use\n")
	fmt.Fprintf(w, "# TYPE s3c_go_memstats_alloc_bytes gauge\n")
	fmt.Fprintf(w, "s3c_go_memstats_alloc_bytes %d\n", ms.Alloc)
	fmt.Fprintf(w, "# HELP s3c_build_info Build version\n")
	fmt.Fprintf(w, "# TYPE s3c_build_info gauge\n")
	fmt.Fprintf(w, "s3c_build_info{version=%q} 1\n", h.version)
	// 存储硬失败（ADR-002）的可观测面：健康检查之外再给一个可告警的 gauge，
	// 让「store 掉线」不必等到业务 5xx 才被发现（roadmap §5.1 R8）。
	storeUp := 1
	if err := h.store.Ping(); err != nil {
		storeUp = 0
	}
	fmt.Fprintf(w, "# HELP s3c_store_up Account store reachable (1) or failing (0); store failure is hard-fail, not degraded (ADR-002)\n")
	fmt.Fprintf(w, "# TYPE s3c_store_up gauge\n")
	fmt.Fprintf(w, "s3c_store_up %d\n", storeUp)
	// 反映 S3C_SSRF_DENY_PRIVATE 的生效值：默认 0（ADR-003 放行私网/回环），置 1 时拒绝。
	fmt.Fprintf(w, "# HELP s3c_ssrf_deny_private Effective SSRF policy: 1 = private/loopback S3 endpoints rejected\n")
	fmt.Fprintf(w, "# TYPE s3c_ssrf_deny_private gauge\n")
	fmt.Fprintf(w, "s3c_ssrf_deny_private %d\n", boolToInt(s3wrap.DenyPrivateNetworks()))
	// 账号库写入失败（ROADMAP #18 指标①）：json / encrypted 驱动的 Ping() 恒为 nil、
	// s3c_store_up 对它们永远是 1，落盘失败是这两类驱动暴露「磁盘满 / 只读」的主动信号。
	fmt.Fprintf(w, "# HELP s3c_store_write_failures_total Account store write failures (persist/SQL write error; business rejections not counted)\n")
	fmt.Fprintf(w, "# TYPE s3c_store_write_failures_total counter\n")
	fmt.Fprintf(w, "s3c_store_write_failures_total %d\n", store.WriteFailureCount())
	// 数据卷容量（ROADMAP #18 指标④）：S3C_DATA_DIR 所在文件系统。
	// 未配置目录 / 平台不支持 / statfs 失败时不输出序列——不用 0 冒充「容量为 0」。
	if h.dataDir != "" {
		if size, free, ok := volumeUsage(h.dataDir); ok {
			fmt.Fprintf(w, "# HELP s3c_volume_size_bytes Filesystem size in bytes for the volume holding S3C_DATA_DIR\n")
			fmt.Fprintf(w, "# TYPE s3c_volume_size_bytes gauge\n")
			fmt.Fprintf(w, "s3c_volume_size_bytes %s\n", strconv.FormatFloat(size, 'f', -1, 64))
			fmt.Fprintf(w, "# HELP s3c_volume_free_bytes Filesystem bytes available to this process on the volume holding S3C_DATA_DIR\n")
			fmt.Fprintf(w, "# TYPE s3c_volume_free_bytes gauge\n")
			fmt.Fprintf(w, "s3c_volume_free_bytes %s\n", strconv.FormatFloat(free, 'f', -1, 64))
		}
	}
	// 在册异步任务数（ROADMAP #18 指标②）：与 JobRegistry 的在册上限同口径
	// （只数未终结任务），逼近上限时按 OPERATIONS.md §4.2 告警。
	fmt.Fprintf(w, "# HELP s3c_jobs_active Non-terminal async jobs currently registered (registry cap 256)\n")
	fmt.Fprintf(w, "# TYPE s3c_jobs_active gauge\n")
	fmt.Fprintf(w, "s3c_jobs_active %d\n", h.migrateJobs.ActiveCount())
	// 最近一次优雅关停耗时（ROADMAP #18 指标⑤）：关停发生在进程退出前，scrape 赶不上，
	// 故由下一次启动从 shutdown.json 载入后暴露（值为 0 表示尚无记录 / 首次启动）。
	fmt.Fprintf(w, "# HELP s3c_last_shutdown_duration_seconds Duration of the most recent graceful shutdown, loaded from data/shutdown.json at startup (0 = no record yet)\n")
	fmt.Fprintf(w, "# TYPE s3c_last_shutdown_duration_seconds gauge\n")
	fmt.Fprintf(w, "s3c_last_shutdown_duration_seconds %s\n",
		strconv.FormatFloat(h.lastShutdown.Seconds(), 'f', -1, 64))
	writeS3UpstreamMetrics(w)
}

// writeHTTPDurationMetrics 输出 HTTP 请求延迟直方图（ROADMAP #18 指标③）。
// `_count` 与 `le="+Inf"` 都取请求总数——两者与计数器同处一次 recordHTTPMetric，
// 结构上不可能漂移（Prometheus 要求 +Inf 桶等于 _count，histogram_quantile 才正确）。
func writeHTTPDurationMetrics(w io.Writer) {
	fmt.Fprintf(w, "# HELP s3c_http_request_duration_seconds HTTP request latency in seconds (includes streaming endpoints)\n")
	fmt.Fprintf(w, "# TYPE s3c_http_request_duration_seconds histogram\n")
	for i, b := range httpDurBucketBounds {
		fmt.Fprintf(w, "s3c_http_request_duration_seconds_bucket{le=%q} %d\n",
			strconv.FormatFloat(b, 'g', -1, 64), metricHTTPDurBuckets[i].Load())
	}
	fmt.Fprintf(w, "s3c_http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", metricHTTPTotal.Load())
	fmt.Fprintf(w, "s3c_http_request_duration_seconds_sum %s\n",
		strconv.FormatFloat(float64(metricHTTPDurSumNs.Load())/1e9, 'g', -1, 64))
	fmt.Fprintf(w, "s3c_http_request_duration_seconds_count %d\n", metricHTTPTotal.Load())
}

// boolToInt 把布尔策略值转成 0/1 gauge 输出。
func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// writeS3UpstreamMetrics 输出 s3wrap 采集的上游调用指标（ASSESSMENT S3）：
// 调用总数、错误按码分类、耗时直方图、流式字节数。标签值来自有限集合，基数可控。
func writeS3UpstreamMetrics(w io.Writer) {
	snap := s3wrap.MetricsSnapshot()
	fmt.Fprintf(w, "# HELP s3c_s3_calls_total Total S3 upstream API calls\n")
	fmt.Fprintf(w, "# TYPE s3c_s3_calls_total counter\n")
	fmt.Fprintf(w, "s3c_s3_calls_total %d\n", snap.Calls)
	fmt.Fprintf(w, "# HELP s3c_s3_call_errors_total S3 upstream call failures by error class\n")
	fmt.Fprintf(w, "# TYPE s3c_s3_call_errors_total counter\n")
	codes := make([]string, 0, len(snap.ErrorsByCode))
	for c := range snap.ErrorsByCode {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, c := range codes {
		fmt.Fprintf(w, "s3c_s3_call_errors_total{code=%q} %d\n", c, snap.ErrorsByCode[c])
	}
	fmt.Fprintf(w, "# HELP s3c_s3_call_duration_seconds S3 upstream call latency histogram\n")
	fmt.Fprintf(w, "# TYPE s3c_s3_call_duration_seconds histogram\n")
	labels := s3wrap.LatencyBucketLabels()
	for i, b := range snap.Latency {
		fmt.Fprintf(w, "s3c_s3_call_duration_seconds_bucket{le=%q} %d\n", labels[i], b.Count)
	}
	fmt.Fprintf(w, "s3c_s3_call_duration_seconds_sum %g\n", snap.LatencySum.Seconds())
	fmt.Fprintf(w, "s3c_s3_call_duration_seconds_count %d\n", snap.Calls)
	fmt.Fprintf(w, "# HELP s3c_s3_stream_bytes_total Bytes streamed from S3 upstream to clients\n")
	fmt.Fprintf(w, "# TYPE s3c_s3_stream_bytes_total counter\n")
	fmt.Fprintf(w, "s3c_s3_stream_bytes_total %d\n", snap.StreamBytes)
}
