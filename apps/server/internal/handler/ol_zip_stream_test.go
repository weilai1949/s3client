package handler

// ol_zip_stream_test.go —— zip.go 校验与部分失败清单、stream.go 并发/超时保护补测。

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// TestOlZipValidation 打包接口：404 / 非法 JSON / 缺桶 / 超过 1000 keys。
func TestOlZipValidation(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	rr := env.accDoRec("POST", "/api/accounts/nope/download-zip", `{"keys":["k"]}`)
	olExpectStatus(t, rr, http.StatusNotFound, "zip 404")

	srv := olFake(t, func(r *http.Request) olResp { return olPlain(http.StatusOK) })
	nb := accNewEnv(t, srv.URL, "")
	rr = nb.accDoRec("POST", "/api/accounts/"+nb.acc.ID+"/download-zip", `{"keys":["k"]}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "zip no bucket")

	e := accNewEnv(t, srv.URL, "b")
	id := e.acc.ID
	rr = e.accDoRec("POST", "/api/accounts/"+id+"/download-zip", "{bad")
	olExpectStatus(t, rr, http.StatusBadRequest, "zip bad json")
	rr = e.accDoRec("POST", "/api/accounts/"+id+"/download-zip", `{"bucket":"b","keys":[]}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "zip no keys")

	// 超过 1000 keys → 400
	keys := make([]string, 1001)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%d", i)
	}
	b, _ := json.Marshal(map[string]any{"bucket": "b", "keys": keys})
	rr = e.accDoRec("POST", "/api/accounts/"+id+"/download-zip", string(b))
	olExpectStatus(t, rr, http.StatusBadRequest, "zip too many keys")
}

// TestOlZipPartialFailure 部分对象拉取失败：zip 内含失败清单。
func TestOlZipPartialFailure(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if strings.HasSuffix(r.URL.Path, "bad") {
			return olErr(http.StatusNotFound, "NoSuchKey")
		}
		return olResp{status: http.StatusOK, headers: map[string]string{"Content-Type": "text/plain"}, body: "data"}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/download-zip", `{"bucket":"b","keys":["good","bad"]}`)
	olExpectStatus(t, rr, http.StatusOK, "zip partial")
	zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("zip open: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["good"] || !names["_下载失败清单.txt"] || names["bad"] {
		t.Fatalf("zip entries = %v", names)
	}
}

// TestOlZipPartialFailureObservable ZIP 部分失败必须在服务端指标中可见（roadmap #4）。
func TestOlZipPartialFailureObservable(t *testing.T) {
	beforePartial := metricZipPartialFailures.Load()
	beforeKeys := metricZipFailedKeys.Load()
	srv := olFake(t, func(r *http.Request) olResp {
		if strings.HasSuffix(r.URL.Path, "bad") {
			return olErr(http.StatusNotFound, "NoSuchKey")
		}
		return olResp{status: http.StatusOK, headers: map[string]string{"Content-Type": "text/plain"}, body: "data"}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/download-zip", `{"bucket":"b","keys":["good","bad"]}`)
	olExpectStatus(t, rr, http.StatusOK, "zip partial")

	if got := metricZipPartialFailures.Load() - beforePartial; got != 1 {
		t.Fatalf("s3c_zip_partial_failures_total delta = %d, want 1", got)
	}
	if got := metricZipFailedKeys.Load() - beforeKeys; got != 1 {
		t.Fatalf("s3c_zip_failed_keys_total delta = %d, want 1", got)
	}
}

// TestOlZipAllGoodNoPartialMetric 全部成功时不计入部分失败指标（避免虚假告警）。
func TestOlZipAllGoodNoPartialMetric(t *testing.T) {
	beforePartial := metricZipPartialFailures.Load()
	srv := olFake(t, func(r *http.Request) olResp { return olPlain(http.StatusOK) })
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/download-zip", `{"bucket":"b","keys":["a","b"]}`)
	olExpectStatus(t, rr, http.StatusOK, "zip all good")
	if got := metricZipPartialFailures.Load() - beforePartial; got != 0 {
		t.Fatalf("partial failures delta = %d, want 0", got)
	}
}

// TestRecordZipOutcomeFailed 整体失败（zipErr 非空）必须计入 s3c_zip_failed_total 并留错误日志。
func TestRecordZipOutcomeFailed(t *testing.T) {
	before := metricZipFailed.Load()
	h := accNewHandler(t, mustStore(t), nil, "")
	h.recordZipOutcome("b", 3, nil, errors.New("writer exploded"))
	if got := metricZipFailed.Load() - before; got != 1 {
		t.Fatalf("s3c_zip_failed_total delta = %d, want 1", got)
	}
}

// mustStore 建一个空的临时文件 store（供只调用内部方法的白盒测试使用）。
func mustStore(t *testing.T) store.AccountStore {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	return st
}

// TestOlStreamLimit503 流式并发槽占满 → 503。
func TestOlStreamLimit503(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	// 占满全部并发槽
	for i := 0; i < maxConcurrentStreams; i++ {
		streamSlots <- struct{}{}
	}
	defer drainSlots(t)
	called := false
	next := func(w http.ResponseWriter, r *http.Request) { called = true }
	rr := httptest.NewRecorder()
	env.hnd.withStreamLimit(next)(rr, httptest.NewRequest("GET", "/x", nil))
	olExpectStatus(t, rr, http.StatusServiceUnavailable, "stream limit")
	if called {
		t.Fatal("next should not be called when saturated")
	}
}

// TestOlStreamLimitCtxDone 占满槽且请求已取消 → 静默返回（不写响应）。
func TestOlStreamLimitCtxDone(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	for i := 0; i < maxConcurrentStreams; i++ {
		streamSlots <- struct{}{}
	}
	defer drainSlots(t)
	called := false
	next := func(w http.ResponseWriter, r *http.Request) { called = true }
	rr := httptest.NewRecorder()
	env.hnd.withStreamLimit(next)(rr, olCancelReq(t, "GET", "/x", ""))
	if rr.Code != http.StatusOK || rr.Body.Len() != 0 {
		t.Fatalf("canceled request wrote response: code=%d body=%q", rr.Code, rr.Body.String())
	}
	if called {
		t.Fatal("next should not be called")
	}
}

// TestOlCopyStreamCtxCancel copyStream：上下文取消时停止读取（contextReader 分支）。
func TestOlCopyStreamCtxCancel(t *testing.T) {
	req := olCancelReq(t, "GET", "/x", "")
	rr := httptest.NewRecorder()
	copyStream(rr, req, strings.NewReader("hello"))
	// 取消后立即停止：不应复制任何内容
	if rr.Body.Len() != 0 {
		t.Fatalf("copied %d bytes after cancel", rr.Body.Len())
	}
}

// TestOlCopyStreamNormal copyStream：正常流式复制与写超时刷新（deadlineWriter 分支）。
func TestOlCopyStreamNormal(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	rr := httptest.NewRecorder()
	copyStream(rr, req, strings.NewReader("hello"))
	if rr.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", rr.Body.String())
	}
}

func drainSlots(t *testing.T) {
	t.Helper()
	for {
		select {
		case <-streamSlots:
			continue
		default:
		}
		return
	}
}

// errReader 在读取时立即返回错误，用于模拟上游读中断。
type errReader struct{ err error }

func (e *errReader) Read(p []byte) (int, error) { return 0, e.err }

// TestCopyStreamReportsUpstreamError copyStream 必须把上游读错误返回给调用方
// （此前 `_, _ = io.Copy(...)` 会吞掉，下载中断在日志/指标里无痕迹 —— todolist #20）。
func TestCopyStreamReportsUpstreamError(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	rr := httptest.NewRecorder()
	want := errors.New("upstream read failed")

	n, err := copyStream(rr, req, &errReader{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if n != 0 {
		t.Errorf("bytes = %d, want 0", n)
	}
}

// TestCopyStreamReportsBytesOnSuccess 成功路径返回实际字节数且无错误。
func TestCopyStreamReportsBytesOnSuccess(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	rr := httptest.NewRecorder()

	n, err := copyStream(rr, req, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if n != 5 {
		t.Errorf("bytes = %d, want 5", n)
	}
	if rr.Body.String() != "hello" {
		t.Errorf("body = %q, want hello", rr.Body.String())
	}
}

// TestCopyStreamContextCancelIsNotError 客户端断开（ctx 取消）也返回错误，
// 但错误必须是 ctx.Err()，调用方据此与「真实传输失败」区分、不计入中断指标。
func TestCopyStreamContextCancelIsNotError(t *testing.T) {
	req := olCancelReq(t, "GET", "/x", "")
	rr := httptest.NewRecorder()

	_, err := copyStream(rr, req, strings.NewReader("hello"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// TestRecordStreamOutcomeCountsRealInterruption 真实中断（ctx 未取消）→ 指标 +1。
func TestRecordStreamOutcomeCountsRealInterruption(t *testing.T) {
	h, _ := gapStoreHandler(t)
	before := metricStreamInterrupted.Load()

	h.recordStreamOutcome(context.Background(), "b", "k", 10, errors.New("read failed"))

	if got := metricStreamInterrupted.Load(); got != before+1 {
		t.Errorf("metricStreamInterrupted = %d, want %d", got, before+1)
	}
}

// TestRecordStreamOutcomeIgnoresClientCancel 客户端主动断开不计入中断指标
// （否则用户取消下载会污染告警）。
func TestRecordStreamOutcomeIgnoresClientCancel(t *testing.T) {
	h, _ := gapStoreHandler(t)
	before := metricStreamInterrupted.Load()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.recordStreamOutcome(ctx, "b", "k", 10, context.Canceled)

	if got := metricStreamInterrupted.Load(); got != before {
		t.Errorf("metricStreamInterrupted = %d, want unchanged %d", got, before)
	}
}

// TestRecordStreamOutcomeNilErrorNoop 无错误时不计数、不记录。
func TestRecordStreamOutcomeNilErrorNoop(t *testing.T) {
	h, _ := gapStoreHandler(t)
	before := metricStreamInterrupted.Load()

	h.recordStreamOutcome(context.Background(), "b", "k", 5, nil)

	if got := metricStreamInterrupted.Load(); got != before {
		t.Errorf("metricStreamInterrupted = %d, want unchanged %d", got, before)
	}
}
