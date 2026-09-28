package model

// review_20260924_test.go —— docs/archive/code-review-2026-09-24.md 分派到 model 域的
// 死代码门禁（R19c）：Account.BucketOrDefault 是恒等函数（等价 a.Bucket），
// 属生产赘余封装，已内联到调用方并删除。

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNoBucketOrDefaultInProductionFiles 源码级门禁：本包**非测试** .go 文件中
// 出现 BucketOrDefault 即红（恒等函数回潮）。空/非空桶的回退行为由 handler 侧
// bucketOr / testAccount 的用例断言。
func TestNoBucketOrDefaultInProductionFiles(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		checked++
		if strings.Contains(string(b), "BucketOrDefault") {
			t.Fatalf("%s 仍定义 BucketOrDefault：恒等函数（返回值恒等于 a.Bucket）"+
				"是生产赘余封装（review R19c），调用方应直接用 a.Bucket", name)
		}
	}
	if checked == 0 {
		t.Fatal("no production .go files scanned; gate is not working")
	}
}
