// Package httpx 存放跨包共用的极简 HTTP 工具。它不是「功能逻辑」容器（见根 AGENTS.md
// 硬约束 #3），只放无处归属、又必须被多个包共享的基础件。
package httpx

import "net/http"

// StatusRecorder 包装 http.ResponseWriter，记录**首次**写出的状态码，并透传 Flush / Unwrap
// （SSE、Range 下载与 http.ResponseController 需要）。它收敛了此前 handler.statusRecorder 与
// tracing.statusWriter 两份职责相同、Flush 已分叉的实现（2026-10-14 评审 O3 / KNOWN_ISSUES #90）。
type StatusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

// NewStatusRecorder 构造记录器；未写出时 Status() 为 200（与 net/http 的隐式 200 一致）。
func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{ResponseWriter: w, status: http.StatusOK}
}

// Status 返回首次写出的状态码（未写出即默认 200）。
func (r *StatusRecorder) Status() int { return r.status }

// WriteHeader 记录并透传**首次**调用；后续调用忽略。Go 服务端本身也会忽略多余 WriteHeader，
// 覆写是为了让日志 / 指标里的状态与实际响应一致（review §B11）。
func (r *StatusRecorder) WriteHeader(code int) {
	if r.written {
		return
	}
	r.written = true
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Write 标记已写出：只 Write 不 WriteHeader 时 Go 隐式发 200，若不在此标记，
// 后续显式 WriteHeader 会被再次透传（Go 打 superfluous 日志，且日志状态与实际不符，review §B11）。
func (r *StatusRecorder) Write(b []byte) (int, error) {
	r.written = true
	return r.ResponseWriter.Write(b)
}

// Flush 透传到真实连接的 Flusher；底层不支持时静默忽略（与 net/http 同口径）。
func (r *StatusRecorder) Flush() {
	_ = http.NewResponseController(r.ResponseWriter).Flush()
}

// Unwrap 让 http.ResponseController / SetWriteDeadline 穿透到真实 ResponseWriter——
// 否则网络流式下载的写超时无法续期（会被静默忽略）。
func (r *StatusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
