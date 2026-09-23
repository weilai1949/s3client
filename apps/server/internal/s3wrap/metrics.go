package s3wrap

// metrics.go —— S3 上游调用指标（ASSESSMENT S3）。
//
// 采集三类信号，均为进程级聚合（无每请求标签，避免无界基数）：
//   - s3c_s3_calls_total：S3 API 调用总数（成功 + 失败）
//   - s3c_s3_call_errors_total{code=...}：失败调用按错误码分类（非 API 错误归 transport）
//   - s3c_s3_call_duration_seconds：调用耗时直方图
//   - s3c_s3_stream_bytes_total：经本服务流式读出的对象字节数
//
// 采集点在 smithy middleware（覆盖所有 SDK 调用，无需逐个方法埋点）。

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/smithy-go/middleware"
)

// latencyBuckets 是耗时直方图的桶上界（秒），最后一个为 +Inf。
var latencyBuckets = []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

// latencyBucketLabels 与 latencyBuckets 对应，供指标输出使用（+Inf 用 "+Inf"）。
var latencyBucketLabels = []string{"0.01", "0.05", "0.1", "0.25", "0.5", "1", "2.5", "5", "10", "30", "+Inf"}

type s3Metrics struct {
	errors   atomic.Int64
	streamIn atomic.Int64
	sumNanos atomic.Int64
	mu       sync.Mutex
	byCode   map[string]int64
	buckets  []int64 // 长度 len(latencyBuckets)+1
	calls    int64   // 与 buckets 同一把锁下维护：每次 observe 恰好 +1 个桶，故 sum(buckets) == calls
}

var globalS3Metrics = newS3Metrics()

func newS3Metrics() *s3Metrics {
	return &s3Metrics{byCode: map[string]int64{}, buckets: make([]int64, len(latencyBuckets)+1)}
}

// S3MetricsSnapshot 是指标的一次性只读快照（供 /api/metrics 输出）。
type S3MetricsSnapshot struct {
	Calls        int64
	Errors       int64
	StreamBytes  int64
	LatencySum   time.Duration
	ErrorsByCode map[string]int64
	// Latency 是**累积**直方图（Prometheus `_bucket{le=...}` 语义）：Latency[i].Count 是
	// 「耗时 ≤ Latency[i].UpperBound 的调用数」，因此 Count 沿下标**单调不减**，且
	// 最后一个 +Inf 桶恒等于 Calls（每次调用恰好落入一个桶，见 observe）。
	// 内部存储是每桶增量（O(1) 写入），累积在快照时计算。
	Latency []LatencyBucket
}

// LatencyBucket 是累积直方图的一个桶。
// Count 是「≤ UpperBound」（Inf 桶为「全部」）的累计调用数，而非该桶区间的增量。
type LatencyBucket struct {
	UpperBound float64
	Inf        bool
	Count      int64
}

// MetricsSnapshot 返回当前 S3 指标快照。
//
// 直方图在此处由「每桶增量」累积为 Prometheus 语义：`_bucket{le=...}` 要求 le 单调不减、
// 且 `+Inf` 等于 `_count`。此前直接把增量当累积输出，导致 `histogram_quantile()` 全错
// （docs/archive/review-2026-09-19.md §7.3 D1）。
func MetricsSnapshot() S3MetricsSnapshot {
	m := globalS3Metrics
	m.mu.Lock()
	defer m.mu.Unlock()
	codes := make(map[string]int64, len(m.byCode))
	for k, v := range m.byCode {
		codes[k] = v
	}
	lat := make([]LatencyBucket, 0, len(m.buckets))
	var cumulative int64
	for i, c := range m.buckets {
		cumulative += c
		if i == len(latencyBuckets) {
			lat = append(lat, LatencyBucket{Inf: true, Count: cumulative})
			continue
		}
		lat = append(lat, LatencyBucket{UpperBound: latencyBuckets[i], Count: cumulative})
	}
	return S3MetricsSnapshot{
		Calls:        m.calls,
		Errors:       m.errors.Load(),
		StreamBytes:  m.streamIn.Load(),
		LatencySum:   time.Duration(m.sumNanos.Load()),
		ErrorsByCode: codes,
		Latency:      lat,
	}
}

// LatencyBucketLabels 返回与快照 Latency 一一对应的桶标签（用于 Prometheus 输出）。
func LatencyBucketLabels() []string {
	out := make([]string, len(latencyBucketLabels))
	copy(out, latencyBucketLabels)
	return out
}

// RecordStreamBytes 记录一次流式读取写出的字节数（handler 在复制完成后调用）。
func RecordStreamBytes(n int64) {
	if n > 0 {
		globalS3Metrics.streamIn.Add(n)
	}
}

func (m *s3Metrics) observe(d time.Duration, err error) {
	m.sumNanos.Add(d.Nanoseconds())
	secs := d.Seconds()
	m.mu.Lock()
	m.calls++
	idx := len(latencyBuckets) // 默认落入 +Inf
	for i, ub := range latencyBuckets {
		if secs <= ub {
			idx = i
			break
		}
	}
	m.buckets[idx]++
	if err != nil {
		m.errors.Add(1)
		m.byCode[errorClass(err)]++
	}
	m.mu.Unlock()
}

// metricErrorCodes 是允许作为指标标签的错误码白名单。
//
// 服务端返回的 `<Code>` 来自用户配置的（不可信）S3 端点：直接把原始 Code 当标签会让
// `s3c_s3_call_errors_total` 的标签基数无界——恶意/不规范对端每次返回不同的 Code 即可
// 让指标内存与 Prometheus 序列数无限增长（review §B10⑥ / §S5）。
// 白名单覆盖 UserMessage / HTTPStatus / IsNotFound 已识别的码；其余一律归 "other"。
var metricErrorCodes = map[string]struct{}{
	"AccessDenied":          {},
	"BucketNotEmpty":        {},
	"EntityTooLarge":        {},
	"InvalidAccessKeyId":    {},
	"InvalidArgument":       {},
	"InvalidPartOrder":      {},
	"InvalidRange":          {},
	"InvalidRequest":        {},
	"InvalidStorageClass":   {},
	"MalformedPolicy":       {},
	"MalformedXML":          {},
	"NoSuchBucket":          {},
	"NoSuchKey":             {},
	"NoSuchUpload":          {},
	"NoSuchVersion":         {},
	"NotFound":              {},
	"RequestTimeout":        {},
	"ServiceUnavailable":    {},
	"SignatureDoesNotMatch": {},
	"SlowDown":              {},
}

// errorClass 把错误归一为有限集合的类别，避免高基数标签。
func errorClass(err error) string {
	if err == nil {
		return ""
	}
	if code := ErrorCode(err); code != "" {
		if _, ok := metricErrorCodes[code]; ok {
			return code
		}
		return "other"
	}
	// 非 API 错误：上下文取消/超时单列，其余归 transport。
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "transport"
}

// metricsMiddleware 包住每次 S3 调用，记录调用数、耗时与错误分类。
type metricsMiddleware struct{}

func (m *metricsMiddleware) ID() string { return "s3clinet:metrics" }

func (m *metricsMiddleware) HandleFinalize(
	ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler,
) (middleware.FinalizeOutput, middleware.Metadata, error) {
	start := time.Now()
	out, md, err := next.HandleFinalize(ctx, in)
	globalS3Metrics.observe(time.Since(start), err)
	return out, md, err
}
