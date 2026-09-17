package handler

import (
	"context"
	"io"
	"net/http"
	"time"
)

// 流式响应（proxy / download-zip / migrate）并发与写超时保护。
const (
	maxConcurrentStreams = 32
	// 滚动空闲写超时：每次成功写出后刷新；慢网大文件不会因绝对截止被误杀。
	streamIdleTimeout = 5 * time.Minute
)

var streamSlots = make(chan struct{}, maxConcurrentStreams)

// withStreamLimit 限制同时进行的流式/长耗时请求数；饱和时返回 503。
func (h *Handler) withStreamLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case streamSlots <- struct{}{}:
			defer func() { <-streamSlots }()
		case <-r.Context().Done():
			return
		default:
			h.writeErr(w, http.StatusServiceUnavailable, "too many concurrent streaming requests")
			return
		}
		next(w, r)
	}
}

// beginStreamResponse 为流式响应设置初始写超时（大文件下载），在 WriteHeader 之前调用。
// 依赖 statusRecorder.Unwrap，使 ResponseController 能触达底层 conn。
func beginStreamResponse(w http.ResponseWriter) {
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(streamIdleTimeout))
}

// copyStream 流式复制：客户端断开时停止读源，并在每次写出后刷新空闲写超时。
//
// 返回本次复制的字节数与错误。错误不再被丢弃：此前 `_, _ = io.Copy(...)` 会把
// 「上游读中断 / 写超时 / 客户端断开」一律吞掉，大文件下载失败在日志与指标里不留
// 任何痕迹（todolist #20 / ASSESSMENT S2）。
//
// 客户端主动断开（ctx.Err() != nil）属正常路径，由调用方降噪处理，不计入中断指标。
func copyStream(w http.ResponseWriter, r *http.Request, src io.Reader) (int64, error) {
	rc := http.NewResponseController(w)
	dw := &deadlineWriter{w: w, rc: rc, idle: streamIdleTimeout}
	return io.Copy(dw, &contextReader{ctx: r.Context(), r: src})
}

// recordStreamOutcome 记录一次流式传输的结果。
//
// 响应头已发出，无法再改状态码，因此这里只做观测：真实中断计入
// s3c_stream_interrupted_total 并 Warn；客户端主动断开（ctx 已取消）属正常路径，
// 仅 Debug，避免用户取消下载就刷出告警（todolist #20）。
func (h *Handler) recordStreamOutcome(ctx context.Context, bucket, key string, n int64, err error) {
	if err == nil {
		return
	}
	if ctx.Err() == nil {
		metricStreamInterrupted.Add(1)
		h.log.Warn("stream interrupted", "bucket", bucket, "key", key, "bytes", n, "err", err)
		return
	}
	h.log.Debug("stream cancelled by client", "bucket", bucket, "key", key, "bytes", n)
}

type deadlineWriter struct {
	w    http.ResponseWriter
	rc   *http.ResponseController
	idle time.Duration
}

func (d *deadlineWriter) Write(p []byte) (int, error) {
	_ = d.rc.SetWriteDeadline(time.Now().Add(d.idle))
	return d.w.Write(p)
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *contextReader) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		return c.r.Read(p)
	}
}
