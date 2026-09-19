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
	calls    atomic.Int64
	errors   atomic.Int64
	streamIn atomic.Int64
	sumNanos atomic.Int64
	mu       sync.Mutex
	byCode   map[string]int64
	buckets  []int64 // 长度 len(latencyBuckets)+1
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
	// Latency 为累积直方图：Latency[i].UpperBound 是桶上界（+Inf 用 0 之外的哨兵由调用方区分）。
	Latency []LatencyBucket
}

// LatencyBucket 是直方图的一个桶。
type LatencyBucket struct {
	UpperBound float64
	Inf        bool
	Count      int64
}

// MetricsSnapshot 返回当前 S3 指标快照。
func MetricsSnapshot() S3MetricsSnapshot {
	m := globalS3Metrics
	m.mu.Lock()
	defer m.mu.Unlock()
	codes := make(map[string]int64, len(m.byCode))
	for k, v := range m.byCode {
		codes[k] = v
	}
	lat := make([]LatencyBucket, 0, len(m.buckets))
	for i, c := range m.buckets {
		if i == len(latencyBuckets) {
			lat = append(lat, LatencyBucket{Inf: true, Count: c})
			continue
		}
		lat = append(lat, LatencyBucket{UpperBound: latencyBuckets[i], Count: c})
	}
	return S3MetricsSnapshot{
		Calls:        m.calls.Load(),
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

// ResetMetrics 清零所有 S3 指标（仅测试使用）。
func ResetMetrics() {
	globalS3Metrics = newS3Metrics()
}

func (m *s3Metrics) observe(d time.Duration, err error) {
	m.calls.Add(1)
	m.sumNanos.Add(d.Nanoseconds())
	secs := d.Seconds()
	m.mu.Lock()
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

// errorClass 把错误归一为有限集合的类别，避免高基数标签。
func errorClass(err error) string {
	if err == nil {
		return ""
	}
	if code := ErrorCode(err); code != "" {
		return code
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
