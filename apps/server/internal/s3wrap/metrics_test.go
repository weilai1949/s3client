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
	"sync"
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
	assertPrometheusHistogramContract(t, snap)
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

// bucketIncrements 把累积直方图还原为每桶增量：increments[i] 是恰好落在第 i 个桶区间的调用数。
// 快照对外是累积语义（Prometheus `_bucket{le=}`），而「每次调用只计入一个桶」要靠增量验证。
func bucketIncrements(s S3MetricsSnapshot) []int64 {
	out := make([]int64, 0, len(s.Latency))
	var prev int64
	for _, b := range s.Latency {
		out = append(out, b.Count-prev)
		prev = b.Count
	}
	return out
}

// assertPrometheusHistogramContract 断言直方图满足 Prometheus 契约：
// le 单调不减、+Inf == _count、且每次调用恰好落进一个桶（增量之和 == _count）。
func assertPrometheusHistogramContract(t *testing.T, s S3MetricsSnapshot) {
	t.Helper()
	if len(s.Latency) == 0 {
		t.Fatal("Latency 为空：直方图必须至少含 +Inf 桶")
	}
	var prev int64
	var sumInc int64
	for i, b := range s.Latency {
		if b.Count < prev {
			t.Fatalf("桶 %d 违反 le 单调不减：Count=%d < 前一个 %d（histogram_quantile 会全错）", i, b.Count, prev)
		}
		inc := b.Count - prev
		if inc < 0 {
			t.Fatalf("桶 %d 增量为负：%d", i, inc)
		}
		sumInc += inc
		prev = b.Count
	}
	if last := s.Latency[len(s.Latency)-1]; !last.Inf {
		t.Fatal("最后一个桶必须是 +Inf")
	}
	if got := s.Latency[len(s.Latency)-1].Count; got != s.Calls {
		t.Fatalf("+Inf 桶 = %d，_count = %d：Prometheus 要求两者相等", got, s.Calls)
	}
	if sumInc != s.Calls {
		t.Fatalf("各桶增量之和 = %d，_count = %d：每次调用必须恰好落入一个桶", sumInc, s.Calls)
	}
}

// TestS3MetricsHistogramConcurrentContract 并发 observe 下快照仍满足 Prometheus 契约。
// 旧实现把 calls 与 buckets 分属 atomic / mutex 两处更新，快照可能读到「+Inf 桶 0、
// _count 3」这类不一致（review §7.3 D1 的实测现象）；现在两者同锁维护。
func TestS3MetricsHistogramConcurrentContract(t *testing.T) {
	resetMetrics()
	const goroutines, perG = 8, 50
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				globalS3Metrics.observe(time.Duration(n*i)*time.Millisecond, nil)
			}
		}(g)
	}
	wg.Wait()
	snap := MetricsSnapshot()
	assertPrometheusHistogramContract(t, snap)
	if want := int64(goroutines * perG); snap.Calls != want {
		t.Fatalf("Calls = %d, want %d", snap.Calls, want)
	}
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

// TestS3MetricsLatencyBuckets 每次调用必须落入恰好一个桶，且桶标签与快照一一对应；
// 快照对外是**累积**语义（Prometheus `_bucket{le=}`）：le 单调不减、+Inf == _count。
// 旧实现把「每桶增量」直接当累积输出，`histogram_quantile()` 全错（review §7.3 D1）。
func TestS3MetricsLatencyBuckets(t *testing.T) {
	resetMetrics()
	globalS3Metrics.observe(15*time.Millisecond, nil) // 落在 0.05 桶
	snap := MetricsSnapshot()
	assertPrometheusHistogramContract(t, snap)
	if snap.Calls != 1 {
		t.Fatalf("Calls = %d, want 1", snap.Calls)
	}
	// 15ms 落在 0.05 桶（下标 1）：前两个桶累积为 1，之后保持 1。
	inc := bucketIncrements(snap)
	if inc[1] != 1 {
		t.Fatalf("15ms 应恰好落入 0.05 桶，增量 = %v", inc)
	}
	if snap.Latency[1].Count != 1 || snap.Latency[0].Count != 0 {
		t.Fatalf("累积值错误：le=0.01 → %d（want 0），le=0.05 → %d（want 1）",
			snap.Latency[0].Count, snap.Latency[1].Count)
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
	assertPrometheusHistogramContract(t, snap)
	// 标签副本不可被调用方改写内部状态。
	labels[0] = "mutated"
	if LatencyBucketLabels()[0] != "0.01" {
		t.Fatal("LatencyBucketLabels must return a copy")
	}
}

// TestS3MetricsHistogramIsCumulativeAcrossBuckets 多个不同耗时的调用累积后，
// 每个 le 桶必须是「≤ 该上界」的**累计**数，而不是只统计该区间。
func TestS3MetricsHistogramIsCumulativeAcrossBuckets(t *testing.T) {
	resetMetrics()
	globalS3Metrics.observe(5*time.Millisecond, nil)  // 0.01
	globalS3Metrics.observe(20*time.Millisecond, nil) // 0.05
	globalS3Metrics.observe(3*time.Second, nil)       // 5
	snap := MetricsSnapshot()
	assertPrometheusHistogramContract(t, snap)
	if snap.Calls != 3 {
		t.Fatalf("Calls = %d, want 3", snap.Calls)
	}
	// 5ms→0.01 桶、20ms→0.05 桶、3s→5 桶；累积后每个 le 是「≤ 上界」的总数。
	want := []int64{1, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3}
	labels := LatencyBucketLabels()
	if len(snap.Latency) != len(want) {
		t.Fatalf("桶数 = %d, want %d", len(snap.Latency), len(want))
	}
	for i, b := range snap.Latency {
		if b.Count != want[i] {
			t.Errorf("le=%s 累积 = %d, want %d（全量快照：%v）", labels[i], b.Count, want[i], snap.Latency)
		}
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
