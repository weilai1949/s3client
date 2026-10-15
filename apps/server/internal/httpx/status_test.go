package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// spyWriter 记录被透传的 WriteHeader 调用。
type spyWriter struct {
	http.ResponseWriter
	codes []int
}

func (s *spyWriter) WriteHeader(code int) {
	s.codes = append(s.codes, code)
	s.ResponseWriter.WriteHeader(code)
}

// flushSpy 记录是否被调用 Flush。
type flushSpy struct {
	http.ResponseWriter
	flushed bool
}

func (f *flushSpy) Flush() { f.flushed = true }

// TestStatusRecorderRecordsFirstStatus 默认 200；显式状态码优先，重复调用被吞掉。
func TestStatusRecorderRecordsFirstStatus(t *testing.T) {
	rec := NewStatusRecorder(httptest.NewRecorder())
	if rec.Status() != http.StatusOK {
		t.Fatalf("default Status() = %d, want 200", rec.Status())
	}
	rec.WriteHeader(http.StatusCreated)
	rec.WriteHeader(http.StatusInternalServerError) // 第二次必须被忽略
	if rec.Status() != http.StatusCreated {
		t.Fatalf("Status() = %d, want 201", rec.Status())
	}
}

// TestStatusRecorderWriteMarksWritten 隐式 200（只 Write 不 WriteHeader）之后，
// 后续 WriteHeader 必须被吞掉、状态保持 200（review §B11）。
func TestStatusRecorderWriteMarksWritten(t *testing.T) {
	spy := &spyWriter{ResponseWriter: httptest.NewRecorder()}
	rec := NewStatusRecorder(spy)
	if _, err := rec.Write([]byte("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	rec.WriteHeader(http.StatusInternalServerError)
	if len(spy.codes) != 0 {
		t.Fatalf("Write 之后的 WriteHeader 不得透传，实际透传了 %v", spy.codes)
	}
	if rec.Status() != http.StatusOK {
		t.Fatalf("Status() = %d, want 200（隐式 200 不能被后续 WriteHeader 改写）", rec.Status())
	}
}

// TestStatusRecorderUnwrapAndFlush Unwrap 返回底层 writer；Flush 透传到底层 Flusher。
func TestStatusRecorderUnwrapAndFlush(t *testing.T) {
	inner := &flushSpy{ResponseWriter: httptest.NewRecorder()}
	rec := NewStatusRecorder(inner)
	if rec.Unwrap() != http.ResponseWriter(inner) {
		t.Fatalf("Unwrap() = %T, want underlying ResponseWriter", rec.Unwrap())
	}
	rec.Flush()
	if !inner.flushed {
		t.Fatal("Flush 未透传到底层 Flusher")
	}
}
