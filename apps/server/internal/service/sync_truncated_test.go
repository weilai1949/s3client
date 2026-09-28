package service

// sync_truncated_test.go —— docs/archive/code-review-2026-09-24.md R6：
// 列举命中安全上限（listMaxTotal）时，SyncKeys 必须透出 truncated，否则
// 「没枚举完」与「对端确实没有更多对象」不可区分——源侧第 100_001 个对象
// 会静默漏拷、超限的目标对象每次同步被误判缺失而重拷。
//
// 断言的都是外部可见行为（SyncResult 字段），不碰 listAll/indexDst 内部。

import (
	"context"
	"strconv"
	"testing"
)

// syncCapFixture 起一个「src/dst 各有 total 个同名同大小同 ETag 对象」的单页列举
// fake：两侧全部判等（不触发任何复制），用例只考察列举截断的透出。
// listFake 的 Size=len(key)、ETag=hash(key)，同名 key 在两个 bucket 下必然相等。
func syncCapFixture(t *testing.T, total int) *listFake {
	t.Helper()
	f := newListFake(t)
	keys := make([]string, total)
	for i := range keys {
		keys[i] = "obj" + strconv.Itoa(i)
	}
	f.keys["src-bucket"] = keys
	f.keys["dst-bucket"] = keys
	f.pageSize = total // 单页返回全部
	return f
}

// TestSync_TruncatedWhenListExceedsCap 源/目标列举都超过安全上限 ⇒ truncated=true，
// 且已枚举的 listMaxTotal 个对象正常比对（全部已存在则跳过）。
func TestSync_TruncatedWhenListExceedsCap(t *testing.T) {
	f := syncCapFixture(t, listMaxTotal+1)
	c := f.client(t)

	out, err := SyncKeys(context.Background(), c, c, "src-bucket", "", "dst-bucket", "", CompareETag, 2, nil)
	if err != nil {
		t.Fatalf("SyncKeys: %v", err)
	}
	if !out.Truncated {
		t.Fatal("列举超过安全上限必须透出 truncated=true（review R6：旧实现静默截断）")
	}
	if out.Scanned != listMaxTotal || out.Skipped != listMaxTotal || out.Copied != 0 || out.Failed != 0 {
		t.Fatalf("result = %+v, want scanned/skipped=%d copied=0 failed=0", out, listMaxTotal)
	}
}

// TestSync_TruncatedFalseWhenListsComplete 正好收满上限且对端声明列举完成 ⇒
// 全部枚举成功，truncated 必须保持 false（截断只在「确实有对象没取到」时置位）。
func TestSync_TruncatedFalseWhenListsComplete(t *testing.T) {
	f := syncCapFixture(t, listMaxTotal)
	c := f.client(t)

	out, err := SyncKeys(context.Background(), c, c, "src-bucket", "", "dst-bucket", "", CompareETag, 2, nil)
	if err != nil {
		t.Fatalf("SyncKeys: %v", err)
	}
	if out.Truncated {
		t.Fatalf("正好收满且列举完成不得标记 truncated: %+v", out)
	}
	if out.Scanned != listMaxTotal || out.Skipped != listMaxTotal || out.Copied != 0 {
		t.Fatalf("result = %+v, want scanned/skipped=%d copied=0", out, listMaxTotal)
	}
}
