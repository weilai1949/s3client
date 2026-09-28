package service

// delete_test.go —— 删除族编排（service/delete.go）的行为测试：
// 记账、前缀递归分页与截断、按批删除与进度、移动（复制后删源）。
// 此前这些用例住在 handler 包（直调 runDeletePrefix），随编排下沉一并搬到 service。

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- 可编程假 S3：ListObjectsV2 / DeleteObjects / DeleteObject / CopyObject ----

type deleteFake struct {
	mu            sync.Mutex
	pages         []string          // list 页按调用次序轮转返回
	listCall      int               // 已发出的 list 请求数
	listFail      bool              // 所有 list 请求回 403
	deleteFail    bool              // 所有 DeleteObjects 回 500
	failKeys      map[string]string // DeleteObjects 逐 key 失败（key → S3 错误码）
	copyFail      bool              // CopyObject 回 403
	srcDeleteFail bool              // DeleteObject 回 500
	copies        []string          // 收到的 CopySource 头
	srcDeletes    []string          // 被删除的源 key
}

func newDeleteFake(t *testing.T, f *deleteFake) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodGet && q.Get("list-type") == "2":
			if f.listFail {
				writeS3Err(w, http.StatusForbidden, "AccessDenied")
				return
			}
			f.mu.Lock()
			i := f.listCall
			f.listCall++
			page := f.pages[i%len(f.pages)]
			f.mu.Unlock()
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, page)
		case r.Method == http.MethodPost && q.Has("delete"):
			if f.deleteFail {
				writeS3Err(w, http.StatusInternalServerError, "InternalError")
				return
			}
			raw, _ := io.ReadAll(r.Body)
			var sb strings.Builder
			sb.WriteString(`<DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			for _, k := range xmlTagValues(string(raw), "Key") {
				if code, bad := f.failKeys[k]; bad {
					fmt.Fprintf(&sb, "<Error><Key>%s</Key><Code>%s</Code><Message>%s</Message></Error>", k, code, code)
				} else {
					fmt.Fprintf(&sb, "<Deleted><Key>%s</Key></Deleted>", k)
				}
			}
			sb.WriteString("</DeleteResult>")
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, sb.String())
		case r.Method == http.MethodDelete:
			if f.srcDeleteFail {
				writeS3Err(w, http.StatusInternalServerError, "InternalError")
				return
			}
			key := strings.TrimPrefix(r.URL.Path, "/")
			if i := strings.Index(key, "/"); i >= 0 {
				key = key[i+1:]
			}
			f.mu.Lock()
			f.srcDeletes = append(f.srcDeletes, key)
			f.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.Header.Get("X-Amz-Copy-Source") != "":
			if f.copyFail {
				writeS3Err(w, http.StatusForbidden, "AccessDenied")
				return
			}
			f.mu.Lock()
			f.copies = append(f.copies, r.Header.Get("X-Amz-Copy-Source"))
			f.mu.Unlock()
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `<?xml version="1.0"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`)
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.String(), http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// snapshotMove 返回（已发出的 CopyObject 次数, 已删除的源 key 数）。
func (f *deleteFake) snapshotMove() (copies, deletes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.copies), len(f.srcDeletes)
}

func writeS3Err(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, fmt.Sprintf(
		`<?xml version="1.0"?><Error><Code>%s</Code><Message>%s</Message></Error>`, code, code))
}

// xmlTagValues 从请求体里抽出所有 <tag>…</tag> 值。
func xmlTagValues(raw, tag string) []string {
	open, closeTag := "<"+tag+">", "</"+tag+">"
	var out []string
	for {
		i := strings.Index(raw, open)
		if i < 0 {
			return out
		}
		raw = raw[i+len(open):]
		j := strings.Index(raw, closeTag)
		if j < 0 {
			return out
		}
		out = append(out, raw[:j])
		raw = raw[j+len(closeTag):]
	}
}

// deleteListXML 构造 ListObjectsV2 响应。
func deleteListXML(keys []string, truncated bool, token string) string {
	isTruncated := "false"
	if truncated {
		isTruncated = "true"
	}
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	sb.WriteString("<IsTruncated>" + isTruncated + "</IsTruncated>")
	if token != "" {
		sb.WriteString("<NextContinuationToken>" + token + "</NextContinuationToken>")
	}
	for _, k := range keys {
		sb.WriteString("<Contents><Key>" + k + "</Key><Size>1</Size><ETag>\"e\"</ETag>" +
			"<LastModified>2026-01-01T00:00:00.000Z</LastModified></Contents>")
	}
	sb.WriteString("</ListBucketResult>")
	return sb.String()
}

// deleteKeysN 构造 n 个稳定排序的 key。
func deleteKeysN(n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("k%06d", i))
	}
	return out
}

func allFailKeys(keys []string) map[string]string {
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = "AccessDenied"
	}
	return out
}

// ---- DeleteKeys ----

func TestDeleteKeysAccountsPerKeyFailures(t *testing.T) {
	f := &deleteFake{failKeys: map[string]string{"b.txt": "AccessDenied"}}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	counts, err := DeleteKeys(context.Background(), client, "bkt", []string{"a.txt", "b.txt", "c.txt"})
	if err != nil {
		t.Fatalf("DeleteKeys: %v", err)
	}
	if counts.Deleted != 2 || counts.Failed != 1 {
		t.Fatalf("counts = %+v, want deleted 2 failed 1", counts)
	}
	if counts.LastError != "access denied" {
		t.Fatalf("lastError = %q, want %q", counts.LastError, "access denied")
	}
}

func TestDeleteKeysSuccessKeepsEmptyLastError(t *testing.T) {
	client := newTestClient(t, newDeleteFake(t, &deleteFake{}).URL)
	counts, err := DeleteKeys(context.Background(), client, "bkt", []string{"a.txt"})
	if err != nil {
		t.Fatalf("DeleteKeys: %v", err)
	}
	if counts.Deleted != 1 || counts.Failed != 0 || counts.LastError != "" {
		t.Fatalf("counts = %+v, want deleted 1 failed 0 empty lastError", counts)
	}
}

func TestDeleteKeysTransportError(t *testing.T) {
	client := newTestClient(t, newDeleteFake(t, &deleteFake{deleteFail: true}).URL)
	if _, err := DeleteKeys(context.Background(), client, "bkt", []string{"a.txt"}); err == nil {
		t.Fatal("整批失败必须返回错误，由调用方映射状态码")
	}
}

// ---- RunDeletePrefix ----

func TestRunDeletePrefixPaginates(t *testing.T) {
	f := &deleteFake{pages: []string{
		deleteListXML(deleteKeysN(1000), true, "t"),
		deleteListXML([]string{"p/last.txt"}, false, ""),
	}}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	res, err := RunDeletePrefix(context.Background(), client, "bkt", "p/")
	if err != nil {
		t.Fatalf("RunDeletePrefix: %v", err)
	}
	if res.Deleted != 1001 || res.Failed != 0 || res.Truncated || res.LastError != "" {
		t.Fatalf("result = %+v, want deleted 1001 not truncated", res)
	}
}

func TestRunDeletePrefixEmptyPageFinishes(t *testing.T) {
	f := &deleteFake{pages: []string{deleteListXML(nil, false, "")}}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	res, err := RunDeletePrefix(context.Background(), client, "bkt", "p/")
	if err != nil {
		t.Fatalf("RunDeletePrefix: %v", err)
	}
	if res.Deleted != 0 || res.Truncated {
		t.Fatalf("result = %+v, want zero not truncated", res)
	}
}

// TestRunDeletePrefixTruncateExact 达到上限即截断（跨页 guard）：
// 100 页 ×1000 key，每页声明 truncated → 第 101 轮触发上限 guard。
func TestRunDeletePrefixTruncateExact(t *testing.T) {
	pages := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		// token 必须逐页前进：真实的对端不会重复同一个 token，而「不前进」正是
		// deletePrefix 的守卫对象（见 TestRunDeletePrefixStopsOnNonAdvancingToken）。
		pages = append(pages, deleteListXML(deleteKeysN(1000), true, fmt.Sprintf("t%d", i)))
	}
	client := newTestClient(t, newDeleteFake(t, &deleteFake{pages: pages}).URL)
	res, err := RunDeletePrefix(context.Background(), client, "bkt", "p/")
	if err != nil {
		t.Fatalf("RunDeletePrefix: %v", err)
	}
	if res.Deleted != 100_000 || res.Failed != 0 || !res.Truncated {
		t.Fatalf("result = %+v, want deleted 100000 failed 0 truncated", res)
	}
}

// TestRunDeletePrefixStopsOnNonAdvancingToken 对端反复返回同一个 NextContinuationToken
// 且每页都声明「还有下一页」时，循环必须终止并标记截断——review §B6 同款守卫。
//
// deletePrefix 此前的退出条件只有「maxDelete 计数到顶」与「token 为空」，而空页 /
// 不前进的 token 都不会推进计数，循环会一直空转到客户端断连（同步端点还会占住
// withStreamLimit 的流槽位）。
func TestRunDeletePrefixStopsOnNonAdvancingToken(t *testing.T) {
	f := &deleteFake{pages: []string{deleteListXML(deleteKeysN(1000), true, "t")}}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	type got struct {
		deleted   int
		truncated bool
		err       error
	}
	done := make(chan got, 1)
	go func() {
		r, err := RunDeletePrefix(context.Background(), client, "bkt", "p/")
		done <- got{deleted: r.Deleted, truncated: r.Truncated, err: err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("RunDeletePrefix: %v", r.err)
		}
		// 首页 token=""→"t" 前进；次页请求带 "t" 而回的仍是 "t" ⇒ 判定不前进即停，
		// 共处理 2 页 ×1000。
		if r.deleted != 2000 {
			t.Fatalf("deleted = %d, want 2000（token 不前进后立即停止）", r.deleted)
		}
		if !r.truncated {
			t.Fatal("token 不前进而停 = 列举不完整，必须标记 truncated")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunDeletePrefix 未终止：NextContinuationToken 不前进导致死循环")
	}
}

// TestRunDeletePrefixStopsAtPageCap token 每次都前进时，页数上限仍是硬边界：
// 否则对端给「空页 + 前进 token」时计数永不增长，maxDelete 守卫永远不触发。
func TestRunDeletePrefixStopsAtPageCap(t *testing.T) {
	// 每页都空、但 token 逐页前进：maxDelete 的计数永不增长，只有页数上限停得住。
	pages := make([]string, 0, listMaxPages+1)
	for i := 0; i <= listMaxPages; i++ {
		pages = append(pages, deleteListXML(nil, true, fmt.Sprintf("t%d", i)))
	}
	f := &deleteFake{pages: pages}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	type got struct {
		deleted   int
		truncated bool
		err       error
	}
	done := make(chan got, 1)
	go func() {
		r, err := RunDeletePrefix(context.Background(), client, "bkt", "p/")
		done <- got{deleted: r.Deleted, truncated: r.Truncated, err: err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("RunDeletePrefix: %v", r.err)
		}
		if r.deleted != 0 {
			t.Fatalf("deleted = %d, want 0（空页）", r.deleted)
		}
		if !r.truncated {
			t.Fatal("页数上限耗尽而对端仍称有下一页，必须标记 truncated")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunDeletePrefix 未终止：空页 + 前进 token 导致死循环")
	}
}

// TestRunDeletePrefixReturnsOnCancelledContext 循环首行就查 ctx：不依赖 SDK 调用
// 间接响应取消（守卫是 100% 覆盖的一部分，不许用「反正 SDK 也会报」跳过）。
func TestRunDeletePrefixReturnsOnCancelledContext(t *testing.T) {
	f := &deleteFake{pages: []string{deleteListXML(deleteKeysN(1), true, "t")}}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunDeletePrefix(ctx, client, "bkt", "p/"); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunDeletePrefix(cancelled) = %v, want context.Canceled", err)
	}
}

// TestRunDeletePrefixCrossLimit 单页跨越上限时裁剪 keys 并截断：
// 99 页 ×1000 + 1 页 ×1500 → 最后一页只取 1000 并截断。
func TestRunDeletePrefixCrossLimit(t *testing.T) {
	pages := make([]string, 0, 100)
	for i := 0; i < 99; i++ {
		pages = append(pages, deleteListXML(deleteKeysN(1000), true, fmt.Sprintf("t%d", i)))
	}
	pages = append(pages, deleteListXML(deleteKeysN(1500), false, ""))
	client := newTestClient(t, newDeleteFake(t, &deleteFake{pages: pages}).URL)
	res, err := RunDeletePrefix(context.Background(), client, "bkt", "p/")
	if err != nil {
		t.Fatalf("RunDeletePrefix: %v", err)
	}
	if res.Deleted != 100_000 || res.Failed != 0 || !res.Truncated {
		t.Fatalf("result = %+v, want deleted 100000 failed 0 truncated", res)
	}
}

// TestRunDeletePrefixProgressesOnAllFailures 全部删除都失败时循环仍必须前进：
// 进度只看「已删除数」会让同一页被反复列出并反复失败，直到任务超时（review §B3 连带缺陷）。
func TestRunDeletePrefixProgressesOnAllFailures(t *testing.T) {
	// 100 页逐页前进的 token，每页都是同一批 1000 个 key 且全部删失败：
	// 循环必须靠「已处理数」而不是「已删除数」前进，一路跑到 10 万上限。
	pages := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		pages = append(pages, deleteListXML(deleteKeysN(1000), true, fmt.Sprintf("t%d", i)))
	}
	f := &deleteFake{
		pages:    pages,
		failKeys: allFailKeys(deleteKeysN(1000)),
	}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	res, err := RunDeletePrefix(context.Background(), client, "bkt", "p/")
	if err != nil {
		t.Fatalf("RunDeletePrefix: %v", err)
	}
	if res.Deleted != 0 || res.Failed != 100_000 || !res.Truncated {
		t.Fatalf("result = %+v, want failed 100000 truncated", res)
	}
	if res.LastError != "access denied" {
		t.Fatalf("lastError = %q", res.LastError)
	}
}

func TestRunDeletePrefixListError(t *testing.T) {
	client := newTestClient(t, newDeleteFake(t, &deleteFake{listFail: true}).URL)
	if _, err := RunDeletePrefix(context.Background(), client, "bkt", "p/"); err == nil {
		t.Fatal("列举失败必须返回错误")
	}
}

func TestRunDeletePrefixDeleteError(t *testing.T) {
	f := &deleteFake{
		pages:      []string{deleteListXML([]string{"p/a.txt"}, false, "")},
		deleteFail: true,
	}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	if _, err := RunDeletePrefix(context.Background(), client, "bkt", "p/"); err == nil {
		t.Fatal("删除失败必须返回错误")
	}
}

// ---- DeleteKeysBatched（异步删除任务体） ----

func TestDeleteKeysBatchedChunksAndReportsProgress(t *testing.T) {
	client := newTestClient(t, newDeleteFake(t, &deleteFake{}).URL)
	keys := deleteKeysN(1500)
	var got []Progress
	counts, failKeys := DeleteKeysBatched(context.Background(), client, "bkt", keys, 200,
		func(p Progress) { got = append(got, p) })
	if counts.Deleted != 1500 || counts.Failed != 0 || counts.LastError != "" {
		t.Fatalf("counts = %+v, want deleted 1500", counts)
	}
	if len(failKeys) != 0 {
		t.Fatalf("failKeys = %v, want none", failKeys)
	}
	if len(got) != 2 {
		t.Fatalf("progress frames = %d, want 2（1000 + 500 两批）", len(got))
	}
	if got[0].Done != 1000 || got[0].Total != 1500 || got[0].Status != "running" {
		t.Fatalf("progress[0] = %+v", got[0])
	}
	if got[1].Done != 1500 || got[1].OK != 1500 || got[1].Error != "" {
		t.Fatalf("progress[1] = %+v", got[1])
	}
}

func TestDeleteKeysBatchedTransportErrorCountsWholeChunk(t *testing.T) {
	client := newTestClient(t, newDeleteFake(t, &deleteFake{deleteFail: true}).URL)
	counts, failKeys := DeleteKeysBatched(context.Background(), client, "bkt", deleteKeysN(3), 200,
		func(Progress) {})
	if counts.Deleted != 0 || counts.Failed != 3 {
		t.Fatalf("counts = %+v, want failed 3（结果未知按整批失败记账）", counts)
	}
	if counts.LastError == "" {
		t.Fatal("整批失败必须留下首个错误文案")
	}
	if len(failKeys) != 3 || failKeys[0] != "k000000" {
		t.Fatalf("failKeys = %v", failKeys)
	}
}

// TestDeleteKeysBatchedCapsFailKeysKeepsFirstError 逐 key 失败时按 API 层上限截断
// failedKeys，且 lastError 只取第一条（后续批次不得覆盖）。
func TestDeleteKeysBatchedCapsFailKeysKeepsFirstError(t *testing.T) {
	keys := deleteKeysN(1500)
	f := &deleteFake{failKeys: allFailKeys(keys)}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	counts, failKeys := DeleteKeysBatched(context.Background(), client, "bkt", keys, 2, func(Progress) {})
	if counts.Deleted != 0 || counts.Failed != 1500 {
		t.Fatalf("counts = %+v, want failed 1500", counts)
	}
	if len(failKeys) != 2 {
		t.Fatalf("failKeys = %v, want 2（按上限截断）", failKeys)
	}
	if counts.LastError != "access denied" {
		t.Fatalf("lastError = %q", counts.LastError)
	}
}

func TestDeleteKeysBatchedWithoutProgressCallback(t *testing.T) {
	client := newTestClient(t, newDeleteFake(t, &deleteFake{}).URL)
	counts, failKeys := DeleteKeysBatched(context.Background(), client, "bkt", []string{"a.txt"}, 200, nil)
	if counts.Deleted != 1 || len(failKeys) != 0 {
		t.Fatalf("counts = %+v failKeys = %v", counts, failKeys)
	}
}

func TestDeleteKeysBatchedStopsWhenCancelled(t *testing.T) {
	client := newTestClient(t, newDeleteFake(t, &deleteFake{}).URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var got []Progress
	counts, failKeys := DeleteKeysBatched(ctx, client, "bkt", deleteKeysN(3), 200,
		func(p Progress) { got = append(got, p) })
	if counts.Deleted != 0 || counts.Failed != 0 {
		t.Fatalf("counts = %+v, want untouched after cancel", counts)
	}
	if len(failKeys) != 0 || len(got) != 0 {
		t.Fatalf("failKeys = %v progress = %v, want none", failKeys, got)
	}
}

// ---- MoveKeys（复制成功后删源） ----

func TestMoveKeysCopiesThenDeletesSource(t *testing.T) {
	f := &deleteFake{}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	out := MoveKeys(context.Background(), client, "bkt", "dst",
		[][2]string{{"a.txt", "a.txt"}, {"b.txt", "b.txt"}}, 2, nil)
	if out.OK != 2 || out.Failed != 0 || out.FirstError != "" {
		t.Fatalf("result = %+v, want ok 2", out)
	}
	copies, deletes := f.snapshotMove()
	if copies != 2 || deletes != 2 {
		t.Fatalf("copies = %d deletes = %d, want 2/2", copies, deletes)
	}
}

func TestMoveKeysSourceDeleteFailureAggregates(t *testing.T) {
	f := &deleteFake{srcDeleteFail: true}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	out := MoveKeys(context.Background(), client, "bkt", "dst",
		[][2]string{{"a.txt", "a.txt"}, {"b.txt", "b.txt"}}, 2, nil)
	if out.OK != 0 || out.Failed != 2 {
		t.Fatalf("result = %+v, want failed 2", out)
	}
	if !strings.Contains(out.FirstError, "copied but failed to delete source") {
		t.Fatalf("firstError = %q, want 移动半成功文案", out.FirstError)
	}
	if len(out.FailKeys) != 2 {
		t.Fatalf("failKeys = %v", out.FailKeys)
	}
	copies, deletes := f.snapshotMove()
	if copies != 2 || deletes != 0 {
		t.Fatalf("copies = %d deletes = %d, want 复制成功但未删源", copies, deletes)
	}
}

func TestMoveKeysCopyFailureSkipsDelete(t *testing.T) {
	f := &deleteFake{copyFail: true}
	client := newTestClient(t, newDeleteFake(t, f).URL)
	out := MoveKeys(context.Background(), client, "bkt", "dst",
		[][2]string{{"a.txt", "a.txt"}}, 1, nil)
	if out.OK != 0 || out.Failed != 1 || out.FirstError == "" {
		t.Fatalf("result = %+v, want failed 1", out)
	}
	if copies, deletes := f.snapshotMove(); copies != 0 || deletes != 0 {
		t.Fatalf("copies = %d deletes = %d, want 0/0（复制失败不得删源）", copies, deletes)
	}
}
