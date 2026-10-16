package service

// copy_large_test.go —— 同端点大对象（CopyObject 报 EntityTooLarge）改走 UploadPartCopy：
// 数据不经过本进程（无源端 GET），各段以 Range 服务端拷贝。

import (
	"context"
	"strings"
	"testing"
)

// TestMigrateKeysEntityTooLargeUsesUploadPartCopy 同端点 CopyObject 超限 → 服务端分段复制，
// 不发源端 GET，段号与 Range 正确，内容逐字节一致。
func TestMigrateKeysEntityTooLargeUsesUploadPartCopy(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.copyTooLarge = true
	f.src["big"] = []byte("0123456789ab")
	out := MigrateKeys(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "dst", []string{"big"}, "", true, 1, nil)
	if out.OK != 1 || out.Failed != 0 {
		t.Fatalf("result = %+v", out)
	}
	if got := string(f.dst["big"]); got != "0123456789ab" {
		t.Fatalf("dst body = %q", got)
	}
	if f.gets != 0 {
		t.Fatalf("gets = %d, want 0（数据不经过本进程）", f.gets)
	}
	want := "bytes=0-3,bytes=4-7,bytes=8-11"
	if strings.Join(f.copyRanges, ",") != want {
		t.Fatalf("copy ranges = %v, want %s", f.copyRanges, want)
	}
	if f.creates != 1 || f.comps != 1 || f.aborts != 0 {
		t.Fatalf("creates=%d comps=%d aborts=%d, want 1/1/0", f.creates, f.comps, f.aborts)
	}
}

// TestMigrateKeysCopyLargeAccessDeniedFallsBackToStream UploadPartCopy 被拒（403）时
// 回退流式复制：分段上传 abort，对象仍然迁移成功（与 R10 的 AccessDenied 回退同口径）。
func TestMigrateKeysCopyLargeAccessDeniedFallsBackToStream(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.copyTooLarge = true
	f.copyPartDenied = true
	f.src["big"] = []byte("0123456789ab")
	out := MigrateKeys(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "dst", []string{"big"}, "", true, 1, nil)
	if out.OK != 1 || out.Failed != 0 {
		t.Fatalf("result = %+v", out)
	}
	if got := string(f.dst["big"]); got != "0123456789ab" {
		t.Fatalf("dst body = %q", got)
	}
	if f.aborts != 1 {
		t.Fatalf("aborts = %d, want 1（被拒的服务端分段复制必须 abort）", f.aborts)
	}
	if f.gets == 0 {
		t.Fatal("回退流式路径应发生源端 GET")
	}
}

// TestMigrateKeysCopyLargeEmptyObject 空对象走服务端分段复制：无段可传，退回单次 PUT 落地。
func TestMigrateKeysCopyLargeEmptyObject(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.copyTooLarge = true
	f.src["empty"] = []byte{}
	out := MigrateKeys(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "dst", []string{"empty"}, "", true, 1, nil)
	if out.OK != 1 || out.Failed != 0 {
		t.Fatalf("result = %+v", out)
	}
	if _, ok := f.dst["empty"]; !ok {
		t.Fatal("空对象应落到目标端")
	}
	if f.comps != 0 {
		t.Fatalf("comps = %d, want 0（空对象走 PUT）", f.comps)
	}
}

// TestMigrateKeysCopyLargeCreateFails 分段初始化失败：回退流式同样失败，逐 key 记为失败且不 panic。
func TestMigrateKeysCopyLargeCreateFails(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.copyTooLarge = true
	f.failCreate = true
	f.src["big"] = []byte("0123456789ab")
	out := MigrateKeys(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "dst", []string{"big"}, "", true, 1, nil)
	if out.OK != 0 || out.Failed != 1 || out.FirstError == "" {
		t.Fatalf("result = %+v", out)
	}
}

// TestMigrateKeysCopyLargeOverCeilingRejected 对象超出段数上限：建立上传之前拒绝，不产生 upload。
func TestMigrateKeysCopyLargeOverCeilingRejected(t *testing.T) {
	shrinkDirect(t, 4, 4, 2, 4) // 上限 2 段 × 4B = 8B < 12B
	f := newSvcFake(t)
	f.copyTooLarge = true
	f.src["big"] = []byte("0123456789ab")
	out := MigrateKeys(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "dst", []string{"big"}, "", true, 1, nil)
	if out.Failed != 1 {
		t.Fatalf("超限对象应失败: %+v", out)
	}
	if f.creates != 0 {
		t.Fatalf("creates = %d, want 0（建立前即拒绝）", f.creates)
	}
}

// TestMigrateKeysCopyLargeHeadFails HEAD 失败：服务端分段复制拿不到大小，回退流式。
func TestMigrateKeysCopyLargeHeadFails(t *testing.T) {
	shrinkDirect(t, 4, 4, 10000, 8)
	f := newSvcFake(t)
	f.copyTooLarge = true
	f.failHead = true
	f.src["big"] = []byte("0123456789ab")
	out := MigrateKeys(context.Background(), f.srcClient(t), f.dstClient(t), "bkt", "dst", []string{"big"}, "", true, 1, nil)
	if out.OK != 1 || out.Failed != 0 {
		t.Fatalf("HEAD 失败应回退流式并成功: %+v", out)
	}
	if got := string(f.dst["big"]); got != "0123456789ab" {
		t.Fatalf("dst body = %q", got)
	}
}
