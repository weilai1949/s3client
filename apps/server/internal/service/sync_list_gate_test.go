package service

// sync_list_gate_test.go —— review-2026-09-19.md §B6：列举循环必须有界。
//
// indexDst 旧实现的循环条件是「已收集数 < 10 万」且循环内不查 ctx：对端只要返回
// 不前进的 NextContinuationToken，这个循环就永不退出（同步端点在任务超时前一直挂着）。

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// blockingListServer 返回一个「每页 1 个 key、永远 truncated」的假 S3：
// token 由 nextToken 决定下一页的 NextContinuationToken（返回空串表示不前进）。
func blockingListServer(t *testing.T, nextToken func(page int) string) *httptest.Server {
	t.Helper()
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := nextToken(page)
		page++
		w.Header().Set("Content-Type", "application/xml")
		body := `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<Contents><Key>obj` + strconv.Itoa(page) + `</Key><Size>1</Size><ETag>"e"</ETag>` +
			`<LastModified>2024-01-01T00:00:00.000Z</LastModified><StorageClass>STANDARD</StorageClass></Contents>` +
			`<IsTruncated>true</IsTruncated>`
		if token != "" {
			body += `<NextContinuationToken>` + token + `</NextContinuationToken>`
		}
		body += `</ListBucketResult>`
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestIndexDstStopsOnNonAdvancingToken 对端反复返回同一个 NextContinuationToken 时必须终止。
func TestIndexDstStopsOnNonAdvancingToken(t *testing.T) {
	srv := blockingListServer(t, func(int) string { return "same" })
	done := make(chan int, 1)
	go func() {
		idx, _ := indexDst(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
		done <- len(idx)
	}()
	select {
	case n := <-done:
		// 首页 token=""、次页请求带 "same" 而又返回 "same" → 判定不前进即停：共处理 2 页。
		if n != 2 {
			t.Fatalf("indexDst = %d entries, want 2（token 不前进后立即停止）", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("indexDst 未终止：NextContinuationToken 不前进导致死循环")
	}
}

// TestListAllStopsOnNonAdvancingToken 同上，覆盖 listAll。
func TestListAllStopsOnNonAdvancingToken(t *testing.T) {
	srv := blockingListServer(t, func(int) string { return "same" })
	done := make(chan int, 1)
	go func() {
		items, _ := listAll(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
		done <- len(items)
	}()
	select {
	case n := <-done:
		if n != 2 {
			t.Fatalf("listAll = %d items, want 2（token 不前进后立即停止）", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("listAll 未终止：NextContinuationToken 不前进导致死循环")
	}
}

// TestIndexDstStopsAtPageCap token 每次都前进时，页数上限仍然是硬边界。
func TestIndexDstStopsAtPageCap(t *testing.T) {
	srv := blockingListServer(t, func(page int) string { return fmt.Sprintf("tok-%d", page) })
	idx, err := indexDst(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
	if err != nil {
		t.Fatalf("indexDst: %v", err)
	}
	if len(idx) != listMaxPages {
		t.Fatalf("indexDst = %d entries, want %d（页数上限）", len(idx), listMaxPages)
	}
}
