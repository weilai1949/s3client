package s3wrap

// metrics_test.go —— S3 上游指标（ASSESSMENT S3）。
//
// 指标口径：每次 S3 API 调用计入调用数、延迟直方图与（失败时）错误码分类；
// 流式读取的字节数由 handler 在复制完成后回报，统一计入 s3c_s3_stream_bytes_total。
// 这些指标让「S3 上游变慢/挂掉」不再只能靠 5xx 间接推断。

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aws/smithy-go/middleware"
)

// resetMetrics 清零进程级 S3 指标，避免用例之间相互污染。
// 定义在测试包内：清零是纯测试需求，不应作为生产 API 暴露（原 `ResetMetrics`
// 见 docs/review-2026-09-19.md §A2）。
func resetMetrics() {
	globalS3Metrics = newS3Metrics()
}

func TestS3MetricsRecordsSuccess(t *testing.T) {
	resetMetrics()
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if _, err := c.ListBuckets(context.Background()); err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}
	snap := MetricsSnapshot()
	if snap.Calls != 1 {
		t.Fatalf("Calls = %d, want 1", snap.Calls)
	}
	if snap.Errors != 0 {
		t.Fatalf("Errors = %d, want 0", snap.Errors)
	}
	if total := bucketTotal(snap); total != 1 {
		t.Fatalf("latency bucket total = %d, want 1", total)
	}
}

func TestS3MetricsRecordsErrorCode(t *testing.T) {
	resetMetrics()
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeS3Error(w, http.StatusForbidden, "AccessDenied", "denied")
	}))
	if _, err := c.ListBuckets(context.Background()); err == nil {
		t.Fatal("expected ListBuckets to fail")
	}
	snap := MetricsSnapshot()
	if snap.Calls != 1 || snap.Errors != 1 {
		t.Fatalf("Calls=%d Errors=%d, want 1/1", snap.Calls, snap.Errors)
	}
	if got := snap.ErrorsByCode["AccessDenied"]; got != 1 {
		t.Fatalf("ErrorsByCode[AccessDenied] = %d, want 1 (%v)", got, snap.ErrorsByCode)
	}
}

// TestS3MetricsTransportError 非 API 错误（连接失败）归入 transport 桶，避免无界基数。
func TestS3MetricsTransportError(t *testing.T) {
	resetMetrics()
	c, srv := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.Close() // 关掉对端 → 传输层错误
	if _, err := c.ListBuckets(context.Background()); err == nil {
		t.Fatal("expected transport failure")
	}
	snap := MetricsSnapshot()
	if snap.ErrorsByCode["transport"] != 1 {
		t.Fatalf("ErrorsByCode = %v, want transport=1", snap.ErrorsByCode)
	}
}

func TestS3MetricsStreamBytes(t *testing.T) {
	resetMetrics()
	RecordStreamBytes(1024)
	RecordStreamBytes(24)
	if got := MetricsSnapshot().StreamBytes; got != 1048 {
		t.Fatalf("StreamBytes = %d, want 1048", got)
	}
}

func TestS3MetricsReset(t *testing.T) {
	resetMetrics()
	RecordStreamBytes(5)
	resetMetrics()
	snap := MetricsSnapshot()
	if snap.Calls != 0 || snap.StreamBytes != 0 || len(snap.ErrorsByCode) != 0 {
		t.Fatalf("snapshot after reset = %+v, want zeroed", snap)
	}
}

// bucketTotal 汇总所有直方图桶的计数。
func bucketTotal(s S3MetricsSnapshot) int64 {
	var n int64
	for _, b := range s.Latency {
		n += b.Count
	}
	return n
}

// TestS3MetricsCanceledAndTimeoutClass 非 API 错误按上下文取消/超时单列，
// 其余归 transport——保证标签基数有界且可区分「上游挂掉」与「客户端放弃」。
func TestS3MetricsCanceledAndTimeoutClass(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"canceled", context.Canceled, "canceled"},
		{"timeout", context.DeadlineExceeded, "timeout"},
		{"transport", errors.New("dial tcp: refused"), "transport"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := errorClass(c.err); got != c.want {
				t.Fatalf("errorClass(%v) = %q, want %q", c.err, got, c.want)
			}
		})
	}
	// 包装后的 context 错误也应被识别（errors.Is 语义）。
	wrapped := fmt.Errorf("request failed: %w", context.DeadlineExceeded)
	if got := errorClass(wrapped); got != "timeout" {
		t.Fatalf("errorClass(wrapped timeout) = %q, want timeout", got)
	}
}

// TestS3MetricsLatencyBuckets 每次调用必须落入恰好一个桶，且桶标签与快照一一对应。
func TestS3MetricsLatencyBuckets(t *testing.T) {
	resetMetrics()
	globalS3Metrics.observe(15*time.Millisecond, nil) // 落在 0.05 桶
	snap := MetricsSnapshot()
	if total := bucketTotal(snap); total != 1 {
		t.Fatalf("bucket total = %d, want 1", total)
	}
	if snap.LatencySum != 15*time.Millisecond {
		t.Fatalf("LatencySum = %v, want 15ms", snap.LatencySum)
	}
	labels := LatencyBucketLabels()
	if len(labels) != len(snap.Latency) {
		t.Fatalf("labels(%d) and buckets(%d) length mismatch", len(labels), len(snap.Latency))
	}
	if labels[len(labels)-1] != "+Inf" || !snap.Latency[len(snap.Latency)-1].Inf {
		t.Fatalf("last bucket must be +Inf: label=%q inf=%v", labels[len(labels)-1], snap.Latency[len(snap.Latency)-1].Inf)
	}
	// 超过最大上界的耗时必须落入 +Inf 桶。
	resetMetrics()
	globalS3Metrics.observe(time.Minute, nil)
	snap = MetricsSnapshot()
	if got := snap.Latency[len(snap.Latency)-1].Count; got != 1 {
		t.Fatalf("+Inf bucket count = %d, want 1", got)
	}
	// 标签副本不可被调用方改写内部状态。
	labels[0] = "mutated"
	if LatencyBucketLabels()[0] != "0.01" {
		t.Fatal("LatencyBucketLabels must return a copy")
	}
}

// TestRegisterMiddlewares 中间件注册到 Finalize 步骤，且两次调用互不干扰。
func TestRegisterMiddlewares(t *testing.T) {
	stack := middleware.NewStack("test", func() any { return nil })
	stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("ResolveEndpointV2", nil), middleware.After)
	if err := registerMiddlewares(stack); err != nil {
		t.Fatalf("registerMiddlewares: %v", err)
	}
	ids := stack.List()
	if !containsStr(ids, "s3clinet:unsigned-payload") || !containsStr(ids, "s3clinet:metrics") {
		t.Fatalf("middlewares not registered: %v", ids)
	}
	// 再次注册不应报错（同 ID 中间件可重复加入，列表中出现两次）。
	if err := registerMiddlewares(stack); err != nil {
		t.Fatalf("second registerMiddlewares: %v", err)
	}
}

// TestRegisterMiddlewaresAnchorMissing 缺少锚点中间件时 Insert 失败，错误必须上抛
// （静默忽略会让 UNSIGNED-PAYLOAD 注入消失，签名行为随 SDK 版本漂移）。
func TestRegisterMiddlewaresAnchorMissing(t *testing.T) {
	stack := middleware.NewStack("test", func() any { return nil })
	if err := registerMiddlewares(stack); err == nil {
		t.Fatal("registerMiddlewares must fail when the ResolveEndpointV2 anchor is absent")
	}
}

func containsStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// TestRecordStreamBytesIgnoresNonPositive 非正字节数不计入（避免负数污染累计值）。
func TestRecordStreamBytesIgnoresNonPositive(t *testing.T) {
	resetMetrics()
	RecordStreamBytes(0)
	RecordStreamBytes(-5)
	if got := MetricsSnapshot().StreamBytes; got != 0 {
		t.Fatalf("StreamBytes = %d, want 0", got)
	}
}

// TestMetricsSnapshotIsCopy 快照与内部状态隔离：修改快照的 map 不影响后续读取。
func TestMetricsSnapshotIsCopy(t *testing.T) {
	resetMetrics()
	globalS3Metrics.observe(time.Millisecond, errors.New("boom"))
	snap := MetricsSnapshot()
	snap.ErrorsByCode["injected"] = 99
	if _, ok := MetricsSnapshot().ErrorsByCode["injected"]; ok {
		t.Fatal("snapshot map must be a copy")
	}
}

// TestErrorClassUnknownCodeFoldsToOther 白名单之外的服务端错误码必须归入 "other"：
// 不可信对端可以用任意 Code 撑爆指标标签基数（review §B10⑥ / §S5）。
func TestErrorClassUnknownCodeFoldsToOther(t *testing.T) {
	resetMetrics()
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeS3Error(w, http.StatusBadRequest, "TotallyMadeUpCode123", "x")
	}))
	if _, err := c.ListBuckets(context.Background()); err == nil {
		t.Fatal("expected ListBuckets to fail")
	}
	snap := MetricsSnapshot()
	if _, ok := snap.ErrorsByCode["TotallyMadeUpCode123"]; ok {
		t.Fatalf("原始错误码不得成为指标标签：%v", snap.ErrorsByCode)
	}
	if snap.ErrorsByCode["other"] != 1 {
		t.Fatalf("ErrorsByCode = %v, want other=1", snap.ErrorsByCode)
	}
}

// TestErrorClassNil 无错误时返回空串（成功调用不写错误标签）。
func TestErrorClassNil(t *testing.T) {
	if got := errorClass(nil); got != "" {
		t.Fatalf("errorClass(nil) = %q, want empty", got)
	}
}
