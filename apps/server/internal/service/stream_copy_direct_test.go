package service

// stream_copy_direct_test.go —— ADR-014：已知大小的跨 endpoint 复制改为无段缓冲直传。
// 用真实小对象（4 字节一段）精确走直传分段路径，内容由 fake 拼接后逐字节比对。

import (
	"context"
	"strings"
	"testing"
)

// shrinkDirect 把单 PUT 阈值、段长、段号上限、单段上限缩到小值，用小对象走分段路径。
func shrinkDirect(t *testing.T, single, partBytes int64, parts int32, maxPart int64) {
	t.Helper()
	oldSingle, oldPart, oldParts, oldMax := singlePutLimit, multipartPartSize, maxMultipartParts, maxPartBytes
	singlePutLimit, multipartPartSize, maxMultipartParts, maxPartBytes = single, partBytes, parts, maxPart
	t.Cleanup(func() {
		singlePutLimit, multipartPartSize, maxMultipartParts, maxPartBytes = oldSingle, oldPart, oldParts, oldMax
	})
}

// TestStreamCopyDirectKnownSizeMultipart 已知大小超过单 PUT 阈值：逐段直传，内容逐字节一致，
// 元数据随分段初始化请求带到目标端，全程只有一次源端 GET（主流顺序供给各段）。
func TestStreamCopyDirectKnownSizeMultipart(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab") // 12B / 4B = 3 段
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err != nil {
		t.Fatalf("StreamCopy: %v", err)
	}
	if got := string(f.dst["obj"]); got != "0123456789ab" {
		t.Fatalf("dst body = %q", got)
	}
	if f.parts != 3 || f.comps != 1 || f.aborts != 0 || f.creates != 1 {
		t.Fatalf("parts=%d comps=%d aborts=%d creates=%d, want 3/1/0/1", f.parts, f.comps, f.aborts, f.creates)
	}
	if f.createMeta != "red" {
		t.Fatalf("CreateMultipartUpload 元数据 = %q, want red", f.createMeta)
	}
	if f.gets != 1 || len(f.ranges) != 0 {
		t.Fatalf("gets=%d ranges=%v, want 1 次主流且无 Range 重读", f.gets, f.ranges)
	}
}

// TestStreamCopyDirectRetriesFailedPartViaRangeGet 某段上传失败：主流已读到一半无法回退，
// 该段改用 Range GET 按偏移重读并重发；失败之后的段也走 Range（主流已不可信）。
func TestStreamCopyDirectRetriesFailedPartViaRangeGet(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	f.failParts = 1 // 第一次 UploadPart 失败，之后成功
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err != nil {
		t.Fatalf("StreamCopy: %v", err)
	}
	if got := string(f.dst["obj"]); got != "0123456789ab" {
		t.Fatalf("dst body = %q", got)
	}
	want := []string{"bytes=0-3", "bytes=4-7", "bytes=8-11"}
	if strings.Join(f.ranges, ",") != strings.Join(want, ",") {
		t.Fatalf("ranges = %v, want %v", f.ranges, want)
	}
	if f.comps != 1 || f.aborts != 0 {
		t.Fatalf("comps=%d aborts=%d, want 1/0", f.comps, f.aborts)
	}
}

// TestStreamCopyDirectGivesUpAfterMaxAttempts 持续失败：单段最多尝试 maxPartAttempts 次后放弃，
// 并 abort 分段上传，绝不 Complete 一个残缺对象。
func TestStreamCopyDirectGivesUpAfterMaxAttempts(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	f.failPart = true
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err == nil {
		t.Fatal("持续失败应返回错误")
	}
	if f.parts != maxPartAttempts {
		t.Fatalf("parts = %d, want %d（单段尝试次数）", f.parts, maxPartAttempts)
	}
	if f.aborts != 1 || f.comps != 0 {
		t.Fatalf("aborts=%d comps=%d, want 1/0", f.aborts, f.comps)
	}
	if _, ok := f.dst["obj"]; ok {
		t.Fatal("失败时不得落盘残缺对象")
	}
}

// TestStreamCopyDirectShortSourceFails 源对象实际短于声明长度：末段读不满，
// 不得完成分段上传（否则会静默产生残缺对象）。
func TestStreamCopyDirectShortSourceFails(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab") // 实际 12B
	f.lieLength = 20                      // 声明 20B
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err == nil {
		t.Fatal("声明长度与实际不符必须失败")
	}
	if f.comps != 0 || f.aborts != 1 {
		t.Fatalf("comps=%d aborts=%d, want 0/1", f.comps, f.aborts)
	}
}

// TestStreamCopyDirectRejectsOverCeilingBeforeCreate 已知大小超出对象上限：建立分段上传之前拒绝。
func TestStreamCopyDirectRejectsOverCeilingBeforeCreate(t *testing.T) {
	shrinkDirect(t, 4, 4, 2, 4) // 上限 2 段 × 4B = 8B < 12B
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj")
	if err == nil || !strings.Contains(err.Error(), "2 parts") {
		t.Fatalf("err = %v, want part-limit error", err)
	}
	if f.creates != 0 || f.parts != 0 {
		t.Fatalf("creates=%d parts=%d, want 0/0（建立前即拒绝）", f.creates, f.parts)
	}
}

// TestStreamCopyDirectPartSizeAdaptsToSegmentCap 对象超过 64MiB×10000 时段长随之放大：
// 12B 对象、段数上限 2、段长下限 4B → 段长 = ceil(12/2) = 6B，恰好 2 段。
func TestStreamCopyDirectPartSizeAdaptsToSegmentCap(t *testing.T) {
	shrinkDirect(t, 4, 4, 2, 100)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err != nil {
		t.Fatalf("StreamCopy: %v", err)
	}
	if got := string(f.dst["obj"]); got != "0123456789ab" {
		t.Fatalf("dst body = %q", got)
	}
	if f.parts != 2 {
		t.Fatalf("parts = %d, want 2（段长随段数上限放大）", f.parts)
	}
}

// TestStreamCopyDirectRangeGetFailureAborts 主流失效后重读也失败：耗尽尝试次数后放弃并 abort。
func TestStreamCopyDirectRangeGetFailureAborts(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	f.failParts = 1       // 首段上传失败 → 主流失效
	f.failRangeGet = true // 重读一律失败
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err == nil {
		t.Fatal("重读失败应返回错误")
	}
	if f.aborts != 1 || f.comps != 0 {
		t.Fatalf("aborts=%d comps=%d, want 1/0", f.aborts, f.comps)
	}
}

// TestStreamCopyDirectCtxCancelledDuringPart ctx 在段上传过程中被取消：立即停止并 abort，不再重试。
func TestStreamCopyDirectCtxCancelledDuringPart(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.onPart = cancel
	if err := StreamCopy(ctx, f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err == nil {
		t.Fatal("ctx 取消应返回错误")
	}
	if f.aborts != 1 || f.comps != 0 {
		t.Fatalf("aborts=%d comps=%d, want 1/0", f.aborts, f.comps)
	}
	if f.parts != 1 {
		t.Fatalf("parts = %d, want 1（取消后不得继续重试）", f.parts)
	}
}

// TestStreamCopyDirectKeepsPartialLastPart 段长不能整除对象长度：末段短于段长，内容仍逐字节一致。
func TestStreamCopyDirectKeepsPartialLastPart(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789") // 10B / 4B = 4+4+2
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err != nil {
		t.Fatalf("StreamCopy: %v", err)
	}
	if got := string(f.dst["obj"]); got != "0123456789" {
		t.Fatalf("dst body = %q", got)
	}
	if f.parts != 3 {
		t.Fatalf("parts = %d, want 3", f.parts)
	}
}

// TestStreamCopyDirectCompleteFailureAborts 完成分段上传失败：abort 且不落盘残缺对象。
func TestStreamCopyDirectCompleteFailureAborts(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	f.failComplete = true
	if err := StreamCopy(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err == nil {
		t.Fatal("complete 失败应返回错误")
	}
	// comps 会被 SDK 对 5xx 的内置重试抬高，故只断言 abort 与「未落盘」。
	if f.aborts != 1 {
		t.Fatalf("aborts = %d, want 1", f.aborts)
	}
	if _, ok := f.dst["obj"]; ok {
		t.Fatal("complete 失败不得落盘对象")
	}
}

// TestPlanPartsFallsBackToDefaultForUnknownSize 未知大小用默认段长（未知大小不再依赖对象大小）。
func TestPlanPartsFallsBackToDefaultForUnknownSize(t *testing.T) {
	got, err := planParts(0)
	if err != nil {
		t.Fatalf("planParts(0): %v", err)
	}
	if got != multipartPartSize {
		t.Fatalf("planParts(0) = %d, want %d", got, multipartPartSize)
	}
}

// TestStreamCopyDirectCtxCancelledDuringRangeGet 重读时 ctx 已取消：立即返回取消错误，不再重试。
func TestStreamCopyDirectCtxCancelledDuringRangeGet(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.src["obj"] = []byte("0123456789ab")
	f.failParts = 1
	f.failRangeGet = true
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.onRangeGet = cancel
	if err := StreamCopy(ctx, f.srcClient(t), f.dstClient(t), "bkt", "obj", "dst", "obj"); err == nil {
		t.Fatal("ctx 取消应返回错误")
	}
	if f.aborts != 1 {
		t.Fatalf("aborts = %d, want 1", f.aborts)
	}
}
