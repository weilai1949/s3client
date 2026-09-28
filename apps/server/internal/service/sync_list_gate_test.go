package service

// sync_list_gate_test.go —— docs/archive/review-2026-09-19.md §B6：列举循环必须有界。
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

// listResult 是三值返回的可传结果：条目数 + 是否截断（err 一律为 nil，
// 走到这里的用例都不期待列举失败）。
type listResult struct {
	n         int
	truncated bool
}

// TestIndexDstStopsOnNonAdvancingToken 对端反复返回同一个 NextContinuationToken 时必须终止，
// 且必须标记截断（不是「列举完了」——review R6）。
func TestIndexDstStopsOnNonAdvancingToken(t *testing.T) {
	srv := blockingListServer(t, func(int) string { return "same" })
	done := make(chan listResult, 1)
	go func() {
		idx, truncated, _ := indexDst(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
		done <- listResult{n: len(idx), truncated: truncated}
	}()
	select {
	case got := <-done:
		// 首页 token=""、次页请求带 "same" 而又返回 "same" → 判定不前进即停：共处理 2 页。
		if got.n != 2 {
			t.Fatalf("indexDst = %d entries, want 2（token 不前进后立即停止）", got.n)
		}
		if !got.truncated {
			t.Fatal("token 不前进而停 = 列举不完整，必须标记 truncated")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("indexDst 未终止：NextContinuationToken 不前进导致死循环")
	}
}

// TestListAllStopsOnNonAdvancingToken 同上，覆盖 listAll。
func TestListAllStopsOnNonAdvancingToken(t *testing.T) {
	srv := blockingListServer(t, func(int) string { return "same" })
	done := make(chan listResult, 1)
	go func() {
		items, truncated, _ := listAll(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
		done <- listResult{n: len(items), truncated: truncated}
	}()
	select {
	case got := <-done:
		if got.n != 2 {
			t.Fatalf("listAll = %d items, want 2（token 不前进后立即停止）", got.n)
		}
		if !got.truncated {
			t.Fatal("token 不前进而停 = 列举不完整，必须标记 truncated")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("listAll 未终止：NextContinuationToken 不前进导致死循环")
	}
}

// TestIndexDstStopsAtPageCap token 每次都前进时，页数上限仍然是硬边界，
// 且耗尽页数上限而对端仍称有下一页 ⇒ 截断（review R6）。
func TestIndexDstStopsAtPageCap(t *testing.T) {
	srv := blockingListServer(t, func(page int) string { return fmt.Sprintf("tok-%d", page) })
	idx, truncated, err := indexDst(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
	if err != nil {
		t.Fatalf("indexDst: %v", err)
	}
	if len(idx) != listMaxPages {
		t.Fatalf("indexDst = %d entries, want %d（页数上限）", len(idx), listMaxPages)
	}
	if !truncated {
		t.Fatal("页数上限耗尽而对端仍称有下一页，必须标记 truncated")
	}
}

// TestListAllStopsAtPageCap listAll 同样以页数上限为硬边界并标记截断。
func TestListAllStopsAtPageCap(t *testing.T) {
	srv := blockingListServer(t, func(page int) string { return fmt.Sprintf("tok-%d", page) })
	items, truncated, err := listAll(context.Background(), newTestClient(t, srv.URL), listFakeBucket, "")
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if len(items) != listMaxPages {
		t.Fatalf("listAll = %d items, want %d（页数上限）", len(items), listMaxPages)
	}
	if !truncated {
		t.Fatal("页数上限耗尽而对端仍称有下一页，必须标记 truncated")
	}
}

// TestListCapExactlyCompleteIsNotTruncated 正好收满 listMaxTotal 且对端声明列举
// 完成 ⇒ 全部枚举/收录成功，不是截断（截断只在「确实有对象没取到」时置位）。
// listAll 与 indexDst 两条路径都要成立（review R6 的边界口径）。
func TestListCapExactlyCompleteIsNotTruncated(t *testing.T) {
	const total = listMaxTotal
	f := newListFake(t)
	keys := make([]string, total)
	for i := range keys {
		keys[i] = "obj" + strconv.Itoa(i)
	}
	f.keys[listFakeBucket] = keys
	f.pageSize = total // 单页正好装满，IsTruncated=false

	items, truncated, err := listAll(context.Background(), f.client(t), listFakeBucket, "")
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if len(items) != total || truncated {
		t.Fatalf("listAll = %d items truncated=%v, want %d/false（正好收满不是截断）", len(items), truncated, total)
	}

	idx, idxTruncated, err := indexDst(context.Background(), f.client(t), listFakeBucket, "")
	if err != nil {
		t.Fatalf("indexDst: %v", err)
	}
	if len(idx) != total || idxTruncated {
		t.Fatalf("indexDst = %d entries truncated=%v, want %d/false（正好收满不是截断）", len(idx), idxTruncated, total)
	}
}
