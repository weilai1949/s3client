package main

// deadcode_gate_test.go —— 测试代码与生产代码里的「消音式死代码」门禁。
//
// golangci-lint 的 `unused` 能发现「定义后无人引用」的测试辅助（实测 `run.tests: true`
// 下会同时报未使用的测试函数与方法），但**看不见显式消音**：
//
//	_ = someVar        // 只写不读的变量假装被使用
//	var _ = SomeSymbol // 纯占位，对已断言符号是冗余
//
// 这类写法会让死代码在 100% 覆盖率门禁下存活（2026-09 的 accFailInjector 事件：
// 测试文件不参与 instrumentation，`_ = x` 又是 golangci-lint 的消音手段）。
// 本门禁扫 apps/server 下全部 .go（含 _test.go），禁止上述两种形状。
//
// 刻意**不**拦 `_ = f()`（丢弃返回值，如 `_ = resp.Body.Close()`）：那是显式、
// 可读的忽略，且 errcheck 与 exclude-functions 已对错误返回做统一策略。

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var (
	// bareDiscardRe 匹配整行的 `_ = ident`（可带行尾注释）。
	bareDiscardRe = regexp.MustCompile(`^[ \t]*_[ \t]*=[ \t]*[A-Za-z_][A-Za-z0-9_]*(?:[ \t]*//.*)?$`)
	// varBlankRe 匹配 `var _ = expr` 占位声明。
	varBlankRe = regexp.MustCompile(`^[ \t]*var[ \t]+_[ \t]*=`)
)

// serverRoot 返回 apps/server 目录（本文件所在目录）。
func serverRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位 server 根目录")
	}
	return filepath.Dir(thisFile)
}

// TestNoSilencingDeadCode 扫描全部 Go 源码，禁止 `_ = ident` 与 `var _ = expr`。
func TestNoSilencingDeadCode(t *testing.T) {
	root := serverRoot(t)
	scanned := 0
	var violations []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		scanned++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		for i, line := range strings.Split(string(data), "\n") {
			if bareDiscardRe.MatchString(line) || varBlankRe.MatchString(line) {
				violations = append(violations,
					filepath.ToSlash(rel)+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s: %v", root, err)
	}
	// 自检：路径写错时 WalkDir 会扫到 0 个文件而「静默变绿」。
	if scanned < 50 {
		t.Fatalf("只扫描到 %d 个 .go 文件，疑似根目录定位错误（%s）", scanned, root)
	}
	for _, v := range violations {
		t.Errorf("消音式死代码：%s\n  请删除该语句；若需忽略返回值请显式写成 `_ = f()` 并说明原因", v)
	}
}
