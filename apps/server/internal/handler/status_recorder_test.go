package handler

// status_recorder_test.go —— docs/archive/review-2026-09-19.md §B11：statusRecorder 未覆写 Write，
// 隐式 200 之后 written 仍为 false，后续显式 WriteHeader 会被再次透传
//（Go 会打 superfluous 日志，且日志里的状态可能被改写成与实际响应不符的值）。

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

// TestStatusRecorderWriteMarksWritten 隐式 200（只 Write 不 WriteHeader）之后，
// 后续 WriteHeader 必须被吞掉、状态保持 200。
func TestStatusRecorderWriteMarksWritten(t *testing.T) {
	spy := &spyWriter{ResponseWriter: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: spy, status: http.StatusOK}
	if _, err := rec.Write([]byte("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	rec.WriteHeader(http.StatusInternalServerError)
	if len(spy.codes) != 0 {
		t.Fatalf("Write 之后的 WriteHeader 不得透传，实际透传了 %v", spy.codes)
	}
	if rec.status != http.StatusOK {
		t.Fatalf("status = %d, want 200（隐式 200 不能被后续 WriteHeader 改写）", rec.status)
	}
}

// TestStatusRecorderWriteHeaderFirst 显式状态码仍然优先，且重复调用被吞掉。
func TestStatusRecorderWriteHeaderFirst(t *testing.T) {
	spy := &spyWriter{ResponseWriter: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: spy, status: http.StatusOK}
	rec.WriteHeader(http.StatusCreated)
	rec.WriteHeader(http.StatusInternalServerError)
	if len(spy.codes) != 1 || spy.codes[0] != http.StatusCreated {
		t.Fatalf("透传的 WriteHeader = %v, want [201]", spy.codes)
	}
	if rec.status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.status)
	}
}
