package service

// review_20260924_test.go —— docs/archive/code-review-2026-09-24.md 分派到 service 域的
// 死代码门禁（R19b）：JobRegistry.Create 生产零引用（生产统一走 handler newJob →
// TryCreate），只允许作为测试接缝存在于 export_test.go。

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestJobRegistryCreateNotInProductionFiles 源码级门禁：本包**非测试** .go 文件中
// 出现 JobRegistry.Create 的定义即红——生产零引用的测试便利封装不得进生产文件。
func TestJobRegistryCreateNotInProductionFiles(t *testing.T) {
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
		if strings.Contains(string(b), "func (r *JobRegistry) Create(") {
			t.Fatalf("%s 定义了 JobRegistry.Create：生产零引用的测试便利封装"+
				"不得进生产文件（review R19b），应移入 export_test.go", name)
		}
	}
	if checked == 0 {
		t.Fatal("no production .go files scanned; gate is not working")
	}
}
