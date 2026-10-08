package tracing

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

// TestEnqueueDropsWhenQueueFull 有界队列满时丢弃并计数，绝不阻塞请求路径；
// 未采样的 span 直接丢弃（不占队列）。
func TestEnqueueDropsWhenQueueFull(t *testing.T) {
	tr := &Tracer{logger: quietLogger(), queue: make(chan *span, 1)}
	tr.enqueue(&span{name: "a", sampled: true})
	tr.enqueue(&span{name: "b", sampled: true})
	if n := tr.dropped.Load(); n != 1 {
		t.Fatalf("dropped = %d, want 1", n)
	}
	tr.enqueue(&span{name: "c", sampled: false})
	if n := tr.dropped.Load(); n != 1 {
		t.Fatalf("未采样 span 不应触发丢弃计数，dropped = %d", n)
	}
}

// TestDrainFlushesQueuedSpans Stop 时把剩余队列一次性刷出；空队列不发请求。
func TestDrainFlushesQueuedSpans(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := &Tracer{
		endpoint: srv.URL,
		service:  "svc",
		client:   srv.Client(),
		logger:   quietLogger(),
		queue:    make(chan *span, 4),
	}
	start := time.Now()
	tr.queue <- &span{name: "child", kind: spanKindInternal, start: start, end: start.Add(time.Millisecond), sampled: true}
	tr.drain(nil)
	if n := c.requestCount(); n != 1 {
		t.Fatalf("drain 应刷出 1 次导出，实际 %d", n)
	}
	if spans := spansOf(t, <-c.ch); len(spans) != 1 || spans[0]["name"] != "child" {
		t.Fatalf("导出内容不符: %v", spans)
	}
	tr.drain(nil)
	if n := c.requestCount(); n != 1 {
		t.Errorf("空队列不应触发导出，实际 %d", n)
	}
}

// TestFlushMarshalFailureIsLogged 序列化失败（未来若加入不可 JSON 化的属性类型）只记日志、
// 不发出请求、不影响调用方。
func TestFlushMarshalFailureIsLogged(t *testing.T) {
	c, srv := newFakeCollector(http.StatusOK)
	defer srv.Close()
	tr := &Tracer{endpoint: srv.URL, service: "svc", client: srv.Client(), logger: quietLogger()}
	now := time.Now()
	sp := &span{
		name: "bad", kind: spanKindInternal, start: now, end: now, sampled: true,
		attrs: []attribute{{key: "bad", value: make(chan int)}},
	}
	tr.flush([]*span{sp})
	if n := c.requestCount(); n != 0 {
		t.Errorf("序列化失败不应发出请求，实际 %d", n)
	}
}

// TestFlushInvalidEndpointIsLogged 构造请求失败（endpoint 非法）只记日志。
func TestFlushInvalidEndpointIsLogged(t *testing.T) {
	tr := &Tracer{endpoint: "://bad", service: "svc", client: http.DefaultClient, logger: quietLogger()}
	now := time.Now()
	tr.flush([]*span{{name: "x", kind: spanKindInternal, start: now, end: now, sampled: true}})
}

// TestCloseHonorsCanceledContext Close 在 ctx 已取消且后台未结束时返回 ctx.Err()。
func TestCloseHonorsCanceledContext(t *testing.T) {
	tr := &Tracer{enabled: true, stop: make(chan struct{}), workerDone: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tr.Close(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Close(canceled) = %v, want context.Canceled", err)
	}
}

// TestSampleNewTraceThreshold 新建 trace 的采样判定：阈值区间按 trace id 低 63 位确定性映射。
func TestSampleNewTraceThreshold(t *testing.T) {
	var zero [16]byte
	var ones [16]byte
	for i := range ones {
		ones[i] = 0xff
	}
	if !(&Tracer{ratio: 0.5}).sampleNewTrace(zero) {
		t.Error("trace id 低 63 位为 0、ratio=0.5 应采样")
	}
	if (&Tracer{ratio: 0.5}).sampleNewTrace(ones) {
		t.Error("trace id 全 1、ratio=0.5 不应采样")
	}
	if !(&Tracer{ratio: 1}).sampleNewTrace(ones) {
		t.Error("ratio=1 必须采样")
	}
	if (&Tracer{ratio: 0}).sampleNewTrace(zero) {
		t.Error("ratio=0 不得采样")
	}
}
