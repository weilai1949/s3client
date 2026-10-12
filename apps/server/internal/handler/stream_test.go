package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type deadlineStub struct {
	http.ResponseWriter
	called bool
}

func (d *deadlineStub) SetWriteDeadline(t time.Time) error {
	d.called = true
	return nil
}

// TestStatusRecorderUnwrap 确保日志包装器可被 ResponseController 穿透到底层连接。
func TestStatusRecorderUnwrap(t *testing.T) {
	inner := &deadlineStub{ResponseWriter: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: inner, status: 200}
	if got := rec.Unwrap(); got != inner {
		t.Fatalf("Unwrap() = %T, want underlying ResponseWriter", got)
	}
	rc := http.NewResponseController(rec)
	if err := rc.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetWriteDeadline via statusRecorder: %v (Unwrap missing?)", err)
	}
	if !inner.called {
		t.Fatal("SetWriteDeadline was not forwarded to underlying writer")
	}
}

// deadlineCounter 记录底层连接被要求刷新写 deadline 的次数。
// SetWriteDeadline 只在真实流式写出路径上被触发，是「写调用数」的可观测代理。
type deadlineCounter struct {
	http.ResponseWriter
	calls int
}

func (d *deadlineCounter) SetWriteDeadline(time.Time) error {
	d.calls++
	return nil
}

// TestCopyStreamThrottlesWriteDeadlineRefresh 大流量下写 deadline 刷新必须按时间节流：
// 旧实现每次 Write 都刷一次（1 MiB / 32 KiB = 32 次），节流后同一秒内只刷一次。
func TestCopyStreamThrottlesWriteDeadlineRefresh(t *testing.T) {
	const total = 1 << 20
	rec := httptest.NewRecorder()
	stub := &deadlineCounter{ResponseWriter: rec}
	n, err := copyStream(stub, httptest.NewRequest(http.MethodGet, "/", nil), opaqueReader{bytes.NewReader(make([]byte, total))})
	if err != nil {
		t.Fatalf("copyStream: %v", err)
	}
	if n != total || rec.Body.Len() != total {
		t.Fatalf("copied n=%d body=%d, want %d", n, rec.Body.Len(), total)
	}
	if stub.calls != 1 {
		t.Fatalf("写 deadline 刷新 %d 次，want 1（按 writeDeadlineRefreshInterval 节流）", stub.calls)
	}
}

// TestCopyStreamRefreshesDeadlineWhenIntervalElapses 刷新间隔为 0 时每次写都必须刷新，
// 保证慢速长传仍能持续续期（不被 5min 空闲超时误杀）。
func TestCopyStreamRefreshesDeadlineWhenIntervalElapses(t *testing.T) {
	old := writeDeadlineRefreshInterval
	writeDeadlineRefreshInterval = 0
	t.Cleanup(func() { writeDeadlineRefreshInterval = old })

	const total = 1 << 20
	stub := &deadlineCounter{ResponseWriter: httptest.NewRecorder()}
	if _, err := copyStream(stub, httptest.NewRequest(http.MethodGet, "/", nil), opaqueReader{bytes.NewReader(make([]byte, total))}); err != nil {
		t.Fatalf("copyStream: %v", err)
	}
	if stub.calls < 2 {
		t.Fatalf("写 deadline 刷新 %d 次，want >1（间隔 0 时每次写都刷新）", stub.calls)
	}
}

// opaqueReader 只暴露 Read，屏蔽底层 io.WriterTo（bytes.Reader 有），
// 确保走 io.CopyBuffer 的缓冲路径。
type opaqueReader struct{ r io.Reader }

func (o opaqueReader) Read(p []byte) (int, error) { return o.r.Read(p) }
