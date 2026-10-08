// Package tracing 提供零第三方依赖的最小 OTLP tracer：W3C traceparent 解析 / 生成、
// 有界队列 + 后台批量导出 OTLP/HTTP JSON，默认关闭（Endpoint 为空即禁用）。
//
// 设计口径（见 docs/decisions/0013-zero-dep-otlp-tracing.md）：
//   - 只做 tracing 不做 metrics——Prometheus 指标已是既有 SSOT，两者互补；
//   - 采样在**新建 trace** 时决定；入站 traceparent 的采样位被继承，不做尾部采样；
//   - 导出失败只记日志，绝不影响请求路径；队列满即丢弃并计数。
package tracing

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 运行时参数。声明为变量而非常量，便于测试用确定性参数验证批量阈值 / ticker 刷出 /
// 有界队列丢弃三条行为（生产值即默认值）。
var (
	queueCapacity = 512         // 待导出 span 的有界队列容量；满则丢弃（计数 + WARN）
	maxBatchSpans = 64          // 单批最大 span 数，达到即立即导出
	flushInterval = time.Second // 未达批量时的定时刷出间隔
)

const (
	defaultServiceName = "s3client"      // ServiceName 为空时的兜底
	exportTimeout      = 5 * time.Second // OTLP 导出 HTTP 超时
)

// Config 是 tracer 的配置，字段与 S3C_OTEL_* 环境变量一一对应。
type Config struct {
	Endpoint    string  // OTLP/HTTP 基址（导出到 {Endpoint}/v1/traces）；空 = 禁用
	ServiceName string  // resource attribute service.name；空 = s3client
	SampleRatio float64 // 新建 trace 的采样比例，取值 [0,1]
}

// Tracer 是一个最小自研 OTLP tracer 实例。
type Tracer struct {
	enabled    bool
	endpoint   string
	service    string
	ratio      float64
	logger     *slog.Logger
	client     *http.Client
	queue      chan *span
	stop       chan struct{}
	workerDone chan struct{}
	closeOnce  sync.Once
	dropped    atomic.Uint64
}

// activeSpanKey 是请求上下文里携带「当前 tracer + 活跃 span」的私有键。
type activeSpanKey struct{}

type activeSpan struct {
	tracer *Tracer
	span   *span
}

// New 构造 tracer。Endpoint 为空（含全空白）返回**禁用态**：不启动后台 goroutine、
// Middleware 原样透传、Start 为 no-op。Endpoint 非空时校验 URL 与 SampleRatio，
// 非法即返回错误（fail-closed，由调用方决定是否拒绝启动）。
func New(cfg Config) (*Tracer, error) {
	service := cfg.ServiceName
	if service == "" {
		service = defaultServiceName
	}
	if math.IsNaN(cfg.SampleRatio) || cfg.SampleRatio < 0 || cfg.SampleRatio > 1 {
		return nil, fmt.Errorf("tracing: SampleRatio=%v 必须在 [0,1] 区间", cfg.SampleRatio)
	}
	t := &Tracer{service: service, ratio: cfg.SampleRatio, logger: slog.Default()}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return t, nil
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("tracing: Endpoint 无法解析: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("tracing: Endpoint 必须是 http(s) URL，得到 %q", cfg.Endpoint)
	}
	t.enabled = true
	t.endpoint = strings.TrimRight(cfg.Endpoint, "/")
	t.client = &http.Client{Timeout: exportTimeout}
	t.queue = make(chan *span, queueCapacity)
	t.stop = make(chan struct{})
	t.workerDone = make(chan struct{})
	go t.run()
	return t, nil
}

// Enabled 报告 tracing 是否启用（Endpoint 非空）。
func (t *Tracer) Enabled() bool { return t.enabled }

// Middleware 为每个请求创建一个 server span，并把 trace 上下文注入 r.Context()
// 供包级 Start 创建子 span；同时向响应写入 traceparent（禁用时整段不执行）。
func (t *Tracer) Middleware(next http.Handler) http.Handler {
	if !t.enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parent, hasParent := parseTraceparent(r.Header.Get("traceparent"))
		var traceID [16]byte
		var parentSpanID [8]byte
		var sampled bool
		if hasParent {
			// 有合法上游上下文：继承 trace id / parent span id 与采样位（不做尾部采样）。
			traceID = parent.traceID
			parentSpanID = parent.spanID
			sampled = parent.sampled
		} else {
			traceID = newTraceID()
			sampled = t.sampleNewTrace(traceID)
		}
		sp := &span{
			traceID: traceID, spanID: newSpanID(), parentID: parentSpanID,
			name: "HTTP " + r.Method, kind: spanKindServer, start: time.Now(), sampled: sampled,
		}
		sp.setStr("http.request.method", r.Method)
		sp.setStr("url.path", r.URL.Path)
		w.Header().Set("traceparent", formatTraceparent(traceID, sp.spanID, sampled))

		rec := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		ctx := context.WithValue(r.Context(), activeSpanKey{}, &activeSpan{tracer: t, span: sp})
		next.ServeHTTP(rec, r.WithContext(ctx))

		sp.setInt("http.response.status_code", rec.status)
		// request id 由 handler 链的 withLogging 写在响应头（客户端提供或服务端生成）；
		// 在此读取即可把 trace 与既有日志 / 审计口径关联起来（不新增日志字段）。
		if reqID := w.Header().Get("X-Request-ID"); reqID != "" {
			sp.setStr("request.id", reqID)
		}
		sp.end = time.Now()
		t.enqueue(sp)
	})
}

// Start 在当前请求的活跃 span 下创建子 span，返回携带子 span 的 ctx 与结束函数。
// 无活跃 span（未启用 / 未经过 Middleware）时原样返回 ctx 与 no-op 结束函数。
func Start(ctx context.Context, name string) (context.Context, func()) {
	active, ok := ctx.Value(activeSpanKey{}).(*activeSpan)
	if !ok {
		return ctx, func() {}
	}
	return active.tracer.startSpan(ctx, active.span, name)
}

// startSpan 创建子 span；结束函数幂等性不做额外保证（调用方 defer 一次）。
func (t *Tracer) startSpan(ctx context.Context, parent *span, name string) (context.Context, func()) {
	sp := &span{
		traceID: parent.traceID, spanID: newSpanID(), parentID: parent.spanID,
		name: name, kind: spanKindInternal, start: time.Now(), sampled: parent.sampled,
	}
	childCtx := context.WithValue(ctx, activeSpanKey{}, &activeSpan{tracer: t, span: sp})
	return childCtx, func() {
		sp.end = time.Now()
		t.enqueue(sp)
	}
}

// sampleNewTrace 决定**新建 trace** 是否采样。ratio 为 1 / 0 时直接短路，中间值按
// trace id 低 63 位做确定性映射（同一 trace id 的判定稳定，便于复现）。
func (t *Tracer) sampleNewTrace(traceID [16]byte) bool {
	if t.ratio >= 1 {
		return true
	}
	if t.ratio <= 0 {
		return false
	}
	v := binary.BigEndian.Uint64(traceID[8:]) >> 1
	return float64(v) < t.ratio*float64(uint64(1)<<63)
}

// enqueue 把已结束的 span 放入有界队列；未采样直接丢弃，队列满则计数并 WARN，
// 绝不阻塞请求路径。
func (t *Tracer) enqueue(sp *span) {
	if !sp.sampled {
		return
	}
	select {
	case t.queue <- sp:
	default:
		n := t.dropped.Add(1)
		t.logger.Warn("tracing: span 队列已满，丢弃 span", "droppedTotal", n, "span", sp.name)
	}
}

// drain 在后台停止时把队列剩余 span 一次性刷出。
func (t *Tracer) drain(batch []*span) {
	for {
		select {
		case sp := <-t.queue:
			batch = append(batch, sp)
		default:
			if len(batch) > 0 {
				t.flush(batch)
			}
			return
		}
	}
}

// run 是后台导出循环：达到批量阈值立即导出，否则按 flushInterval 定时导出，
// 收到 stop 后刷出剩余队列并退出。
func (t *Tracer) run() {
	defer close(t.workerDone)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	batch := make([]*span, 0, maxBatchSpans)
	for {
		select {
		case sp := <-t.queue:
			batch = append(batch, sp)
			if len(batch) >= maxBatchSpans {
				t.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				t.flush(batch)
				batch = batch[:0]
			}
		case <-t.stop:
			t.drain(batch)
			return
		}
	}
}

// Close 停止后台导出并刷出剩余队列。ctx 用于限定等待时间；禁用态直接返回 nil。
// 重复调用安全（closeOnce）。
func (t *Tracer) Close(ctx context.Context) error {
	if !t.enabled {
		return nil
	}
	t.closeOnce.Do(func() { close(t.stop) })
	select {
	case <-t.workerDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// flush 把一批 span 以 OTLP/HTTP JSON POST 到 {Endpoint}/v1/traces。
// 任何失败（序列化 / 构造请求 / 网络 / 非 2xx）只记日志，不影响调用方。
func (t *Tracer) flush(spans []*span) {
	body, err := json.Marshal(buildPayload(t.service, spans))
	if err != nil {
		t.logger.Warn("tracing: 序列化 OTLP 报文失败", "err", err)
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		t.endpoint+"/v1/traces", bytes.NewReader(body))
	if err != nil {
		t.logger.Warn("tracing: 构造 OTLP 导出请求失败", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Warn("tracing: OTLP 导出失败", "err", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		t.logger.Warn("tracing: OTLP 采集端返回非 2xx", "status", resp.StatusCode)
	}
}

// statusWriter 记录响应状态码并保持流式能力（Flush / Unwrap 供 SSE、Range 下载使用）。
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader 记录状态码后透传。
func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush 透传到真实连接的 Flusher；底层不支持时静默忽略（与 net/http 同口径）。
func (w *statusWriter) Flush() {
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap 让 http.ResponseController / SetWriteDeadline 穿透到真实 ResponseWriter，
// 与 handler.statusRecorder 的既有口径一致。
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
