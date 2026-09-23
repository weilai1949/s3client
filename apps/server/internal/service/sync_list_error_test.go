package service

// sync_list_error_test.go —— docs/archive/review-2026-09-19.md §B5/§B6：列举错误必须上抛（源/目标两侧），
// 而 ctx 取消是「客户端放弃」，不上报为错误；取消后 indexDst 必须立刻停。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// listErrorServer 返回一个「列举一律 403」的假 S3。
func listErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestSync_SourceListErrorIsReturned 源端列举失败（403）必须作为错误返回，
// 而不是 `{scanned:0}` 的「无事可做」。
func TestSync_SourceListErrorIsReturned(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{"src-bucket": {"x.txt"}, "dst-bucket": {}}
	s3FakeMu.Unlock()

	src := newTestClient(t, listErrorServer(t).URL)
	dst, _, closer := makeFakePair(t)
	defer closer()

	if _, err := SyncKeys(context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2, nil); err == nil {
		t.Fatal("源端列举失败必须上抛（否则用户被告知「没有需要同步的对象」）")
	}
}

// TestSync_DstListErrorIsReturned 目标端列举失败同样必须上抛，且保留源侧已扫到的计数。
func TestSync_DstListErrorIsReturned(t *testing.T) {
	s3FakeMu.Lock()
	s3FakeStore = map[string][]string{"src-bucket": {"x.txt"}, "dst-bucket": {}}
	s3FakeMu.Unlock()

	src, _, closer := makeFakePair(t)
	defer closer()
	dst := newTestClient(t, listErrorServer(t).URL)

	out, err := SyncKeys(context.Background(), src, dst, "src-bucket", "", "dst-bucket", "", CompareETag, 2, nil)
	if err == nil {
		t.Fatal("目标端列举失败必须上抛")
	}
	if out.Scanned != 1 {
		t.Fatalf("Scanned = %d, want 1（源侧已扫到的数量应保留在部分结果里）", out.Scanned)
	}
}

// TestIndexDstStopsOnCancelledContext 已取消的 ctx 必须在循环开头就退出，连请求都不发：
// 旧实现的循环体不查 ctx，对端持续返回 truncated 时任务会一直挂到 2h 超时（review §B6）。
func TestIndexDstStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reqs := 0
	srv := blockingListServer(t, func(int) string { return "next" })
	base := srv.Config.Handler
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		base.ServeHTTP(w, r)
	})

	idx, err := indexDst(ctx, newTestClient(t, srv.URL), listFakeBucket, "")
	if err == nil {
		t.Fatal("取消后 indexDst 必须返回错误，而不是继续列举")
	}
	if len(idx) != 0 {
		t.Fatalf("已取回的条目数 = %d, want 0", len(idx))
	}
	if reqs != 0 {
		t.Fatalf("ctx 已取消时不应发起列举请求，实际发了 %d 次", reqs)
	}
}
