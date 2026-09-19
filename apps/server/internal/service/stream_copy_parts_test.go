package service

// stream_copy_parts_test.go —— review-2026-09-19.md §B10①：段号 10000 是合法段，
// 上限判断必须在**上传前**做，否则「正好 10000 段」的对象会被误判超限并 abort。

import (
	"context"
	"strings"
	"testing"
)

// TestMaxMultipartPartsIsProtocolLimit 上限的默认值就是 S3 协议上限（防止被测试值泄漏到生产）。
func TestMaxMultipartPartsIsProtocolLimit(t *testing.T) {
	if maxMultipartParts != 10_000 {
		t.Fatalf("maxMultipartParts = %d, want 10000", maxMultipartParts)
	}
}

// shrinkParts 把段上限/段大小临时改成小值，用小数据精确覆盖段号边界。
func shrinkParts(t *testing.T, limit int32) {
	t.Helper()
	oldLimit, oldSize := maxMultipartParts, multipartPartSize
	maxMultipartParts, multipartPartSize = limit, 1
	t.Cleanup(func() { maxMultipartParts, multipartPartSize = oldLimit, oldSize })
}

// TestMultipartStreamCopyAcceptsExactlyMaxParts 段号等于上限的那一段是合法的，必须完成上传。
func TestMultipartStreamCopyAcceptsExactlyMaxParts(t *testing.T) {
	shrinkParts(t, 3)
	f := newSvcFake(t)
	if err := MultipartStreamCopy(context.Background(), f.dstClient(t), "dst", "m.bin", "text/plain", strings.NewReader("xxx")); err != nil {
		t.Fatalf("MultipartStreamCopy(恰好 3 段): %v", err)
	}
	if f.parts != 3 || f.comps != 1 || f.aborts != 0 {
		t.Fatalf("parts=%d comps=%d aborts=%d, want 3/1/0", f.parts, f.comps, f.aborts)
	}
}

// TestMultipartStreamCopyRejectsPartOverLimit 超出上限的段必须在上传前被拒绝并 abort。
func TestMultipartStreamCopyRejectsPartOverLimit(t *testing.T) {
	shrinkParts(t, 3)
	f := newSvcFake(t)
	err := MultipartStreamCopy(context.Background(), f.dstClient(t), "dst", "m.bin", "text/plain", strings.NewReader("xxxx"))
	if err == nil || !strings.Contains(err.Error(), "3 parts") {
		t.Fatalf("err = %v, want part-limit error", err)
	}
	if f.parts != 3 {
		t.Fatalf("parts = %d, want 3（第 4 段不应发出）", f.parts)
	}
	if f.aborts != 1 {
		t.Fatalf("aborts = %d, want 1", f.aborts)
	}
}
