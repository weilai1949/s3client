package main

// deadcode_gate_test.go —— 测试代码与生产代码里的「消音式死代码」门禁。
//
// golangci-lint 的 `unused` 能发现「定义后无人引用」的测试辅助（实测 `run.tests: true`
// 下会同时报未使用的测试函数与方法，**含传递性**：只被另一个死函数引用的叶子函数也会
// 被一并报出），但**看不见**两类缺口：
//
//	_ = someVar        // 只写不读的变量假装被使用
//	var _ = SomeSymbol // 纯占位，对已断言符号是冗余
//
// 这类写法会让死代码在 100% 覆盖率门禁下存活（2026-09 的 accFailInjector 事件：
// 测试文件不参与 instrumentation，`_ = x` 又是 golangci-lint 的消音手段）。
// 本文件扫 apps/server 下全部 .go（含 _test.go），禁止上述两种形状。
//
// 另有一类 golangci-lint **结构上看不见**的缺口（见 TestNoUnusedExportedTestSymbols）：
// 导出符号被 `unused` 视为「可能被包外引用」而豁免，但 `_test.go` 里的导出符号**永远**
// 不可能被包外引用——测试文件不参与库构建。实测给 `_test.go` 加一个无人调用的
// 导出函数/类型，`golangci-lint run` 报 0 issues。
//
// 刻意**不**拦 `_ = f()`（丢弃返回值，如 `_ = resp.Body.Close()`）：那是显式、
// 可读的忽略，且 errcheck 与 exclude-functions 已对错误返回做统一策略。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"unicode"
)

var (
	// bareDiscardRe 匹配整行的 `_ = ident`（可带行尾注释）——**仅供口径测试引用**，
	// 生产扫描走 AST（见 findSilencingDeadCode）。保留它是为了让口径测试能断言
	// 「AST 判定的覆盖面严格大于旧正则」，把两者的差异固定下来。
	bareDiscardRe = regexp.MustCompile(`^[ \t]*_[ \t]*=[ \t]*[A-Za-z_][A-Za-z0-9_]*(?:[ \t]*//.*)?$`)
	// varBlankRe 匹配 `var _ = expr` 占位声明（同上，仅供口径测试）。
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

// silenceViolation 是一处消音式死代码。
type silenceViolation struct {
	file string // 相对 apps/server
	line int
	text string
}

// TestNoSilencingDeadCode 扫描全部 Go 源码，禁止「用空白标识符消音」的死代码。
//
// 判定走 **AST**，覆盖三类形状（旧实现用整行正则，只认第一类）：
//
//	_ = ident          // 整行丢弃一个标识符（只写不读）
//	var _ = expr       // 占位声明
//	_, _ = f(); x := 1; _ = x   // 行内多语句 / 选择器 / 下标 —— 旧正则全部逃逸
//
// 实测旧正则在 `x := 1; _ = x`、`if true { _ = x }`、`_ = x.Field`、`_ = s[0]` 上
// 全部不命中（fail-open），故改为 AST。
//
// 刻意**不**拦「丢弃函数调用结果」（`_ = f()` / `_, _ = w.Write(b)`）：那是显式、可读的
// 忽略，且 errcheck 与 exclude-functions 已对错误返回做统一策略。判据是「右侧是否为调用
// 表达式」——是调用则放行，是标识符/选择器/下标/字面量则视为消音。
func TestNoSilencingDeadCode(t *testing.T) {
	root := serverRoot(t)
	scanned := 0
	var violations []silenceViolation

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
		violations = append(violations, findSilencingDeadCode(filepath.ToSlash(rel), data)...)
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s: %v", root, err)
	}
	// 自检：路径写错时 WalkDir 会扫到 0 个文件而「静默变绿」。
	if scanned < 50 {
		t.Fatalf("只扫描到 %d 个 .go 文件，疑似根目录定位错误（%s）", scanned, root)
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].file != violations[j].file {
			return violations[i].file < violations[j].file
		}
		return violations[i].line < violations[j].line
	})
	for _, v := range violations {
		t.Errorf("消音式死代码：%s:%d: %s\n  请删除该语句；若需忽略返回值请显式写成 `_ = f()` 并说明原因",
			v.file, v.line, v.text)
	}
}

// findSilencingDeadCode 用 AST 找出「用空白标识符消音」的语句。
//
// 覆盖：`_ = x`（x 为标识符 / 选择器 / 下标 / 字面量）、`var _ = expr`、`_, _ = x, y`
// 等任意组合；**放行**右侧为函数调用（显式丢弃返回值）与 `var _ T = expr`（编译期接口断言）。
func findSilencingDeadCode(file string, data []byte) []silenceViolation {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, data, 0)
	if err != nil {
		return nil // 语法错误交给 go build / vet 报错，不在此重复。
	}
	var out []silenceViolation
	record := func(pos token.Pos, text string) {
		p := fset.Position(pos)
		out = append(out, silenceViolation{file: file, line: p.Line, text: strings.TrimSpace(text)})
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.AssignStmt:
			// 判定规则（关键）：只有当**左侧全部为 `_`** 时才算消音。
			//   - `_, ok := m[k]`（comma-ok 惯用法）左侧有真实变量 `ok`，语句有实际用途 → 放行；
			//   - `_ = x` / `_, _ = x, y` 左侧全空 → 纯粹为丢弃而写 → 拦。
			// 若左侧全空但右侧**全是调用**（`_, _ = w.Write(b)`），属显式丢弃返回值 → 放行。
			allBlank := len(v.Lhs) > 0
			for _, lhs := range v.Lhs {
				if id, ok := lhs.(*ast.Ident); !ok || id.Name != "_" {
					allBlank = false
					break
				}
			}
			if !allBlank {
				return true
			}
			allCalls := len(v.Rhs) > 0
			for _, rhs := range v.Rhs {
				if !isCallExpr(rhs) {
					allCalls = false
					break
				}
			}
			if allCalls {
				return true // 显式丢弃返回值：放行。
			}
			record(v.Pos(), exprLine(fset, data, v))
		case *ast.ValueSpec:
			// `var _ = expr` 是占位；`var _ Iface = (*T)(nil)` 是编译期接口断言，放行。
			if v.Type == nil {
				for _, name := range v.Names {
					if name.Name == "_" {
						record(v.Pos(), exprLine(fset, data, v))
						break
					}
				}
			}
		}
		return true
	})
	return out
}

// isCallExpr 报告表达式是否为函数/方法调用（`f()` / `x.M()` / `pkg.F()`）。
func isCallExpr(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.CallExpr:
		return true
	case *ast.UnaryExpr:
		// `_ = &x` 之类仍属消音，不放行；只有调用才放行。
		return false
	case *ast.TypeAssertExpr:
		// `_ = x.(T)` 的右侧可能含调用，但形式上是断言，按消音处理。
		return false
	case *ast.ParenExpr:
		return isCallExpr(v.X)
	default:
		return false
	}
}

// exprLine 取该节点所在行的源码文本（用于报错定位）。
func exprLine(fset *token.FileSet, data []byte, n ast.Node) string {
	start := fset.Position(n.Pos())
	end := fset.Position(n.End())
	if start.Line != end.Line {
		return "（多行语句）"
	}
	lines := strings.Split(string(data), "\n")
	if start.Line-1 < 0 || start.Line-1 >= len(lines) {
		return ""
	}
	return lines[start.Line-1]
}

// TestFindSilencingDeadCodeCoverage 是 AST 判定的口径测试：
// 钉住「哪些形状必须拦、哪些必须放行」，并断言 AST 覆盖面**严格大于**旧整行正则——
// 旧正则在行内多语句 / 选择器 / 下标上全部逃逸（实测 fail-open）。
func TestFindSilencingDeadCodeCoverage(t *testing.T) {
	t.Parallel()
	src := `package p

func f() {
	_ = someVar                 // 整行标识符：拦
	var unused = 1
	x := 1
	_ = x                       // 行内/普通：拦
	_, _ = x, unused            // 多值非调用：拦
	_ = x.Field                 // 选择器：拦（旧正则逃逸）
	_ = xs[0]                   // 下标：拦（旧正则逃逸）
	if true {
		_ = x                   // 块内：拦
	}
	_, _ = w.Write(b)           // 丢弃返回值：放行
	_ = closeIt()               // 丢弃返回值：放行
	_ = g().Field               // 右侧非纯调用：拦
	if _, ok := m[k]; ok {      // comma-ok 惯用法：放行（左侧有真实变量）
		_ = ok
	}
}

var _ = placeholder             // 占位声明：拦

var _ Iface = (*T)(nil)         // 编译期接口断言：放行
`
	got := findSilencingDeadCode("probe.go", []byte(src))
	// 统计被拦的行号集合，便于精确断言。
	gotLines := map[int]bool{}
	for _, v := range got {
		gotLines[v.line] = true
	}
	lines := strings.Split(src, "\n")
	lineOf := func(needle string) int {
		for i, l := range lines {
			if strings.Contains(l, needle) {
				return i + 1
			}
		}
		t.Fatalf("源码里找不到 %q", needle)
		return 0
	}

	mustCatch := []string{
		"_ = someVar",
		"_, _ = x, unused",
		"_ = x.Field",
		"_ = xs[0]",
		"_ = g().Field",
		"var _ = placeholder",
	}
	for _, s := range mustCatch {
		if !gotLines[lineOf(s)] {
			t.Errorf("漏报消音死代码 %q（第 %d 行）", s, lineOf(s))
		}
	}
	mustPass := []string{
		"_, _ = w.Write(b)",
		"_ = closeIt()",
		"var _ Iface = (*T)(nil)",
		"if _, ok := m[k]; ok {",
	}
	for _, s := range mustPass {
		if gotLines[lineOf(s)] {
			t.Errorf("误报合法写法 %q（第 %d 行）：右侧为调用或属编译期接口断言", s, lineOf(s))
		}
	}

	// 关键断言：AST 必须比旧整行正则多抓到「行内多语句 / 选择器 / 下标」。
	oldRegexHits := 0
	for _, l := range lines {
		if bareDiscardRe.MatchString(l) || varBlankRe.MatchString(l) {
			oldRegexHits++
		}
	}
	if len(got) <= oldRegexHits {
		t.Errorf("AST 判定命中 %d 处，未超过旧整行正则的 %d 处——覆盖面没有扩大，"+
			"旧正则的 fail-open（行内多语句 / 选择器 / 下标）仍未封堵", len(got), oldRegexHits)
	}
	t.Logf("AST 命中 %d 处 vs 旧正则 %d 处", len(got), oldRegexHits)
}

// testExportedDecl 是一个在 _test.go 中声明的导出包级符号。
type testExportedDecl struct {
	file string // 相对 apps/server
	pkg  string // 归一化包名（`foo_test` 与 `foo` 视为同一作用域）
	name string
	kind string
}

// TestNoUnusedExportedTestSymbols 拦截 golangci-lint `unused` 结构上看不见的死代码：
// **`_test.go` 里声明、却从未被引用的导出函数 / 类型 / 变量 / 常量**。
//
// 为什么 `unused` 不管：导出符号默认视为「可能被包外引用」而豁免。但 `_test.go` 只在
// `go test` 编译本包时参与构建，**永远**不会被包外引用——导出与否对测试文件毫无意义。
// 实测：往 `internal/handler/zz_probe_test.go` 加一个无人调用的 `ZzExportedUnusedHelper`
// 与 `ZzExportedUnusedType`，`golangci-lint run ./internal/handler/` 报 **0 issues**。
//
// 例外：`export_test.go` 是 Go 社区约定的**测试接缝**文件（只在测试构建中参与，
// 供同包外的 `_test` 包或跨包测试引用），其导出符号按约定合法，整体豁免。
//
// 本仓库当前该集合为空（无死代码），故检测逻辑另由 TestFindUnusedExportedTestSymbols
// 用合成源码做单元测试——**不**依赖「仓库里正好有一个死符号」来证明门禁有效。
func TestNoUnusedExportedTestSymbols(t *testing.T) {
	root := serverRoot(t)
	sources := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		sources[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s: %v", root, err)
	}
	// 自检：路径写错时不要静默变绿。
	if len(sources) < 50 {
		t.Fatalf("只读取到 %d 个 .go 文件，疑似根目录定位错误（%s）", len(sources), root)
	}

	dead := findUnusedExportedTestSymbols(sources)
	for _, d := range dead {
		t.Errorf("%s: _test.go 中的导出%s %q 从未被引用（golangci-lint `unused` 因「可能被包外引用」"+
			"而豁免导出符号，但测试文件不参与库构建，永远不会有包外引用）——请删除，或改为非导出",
			d.file, d.kind, d.name)
	}
	t.Logf("扫描 %d 个文件，_test.go 中无引用的导出符号 %d 个", len(sources), len(dead))
}

// findUnusedExportedTestSymbols 在给定的「相对路径 → 源码」集合里，找出 `_test.go` 中
// 声明却从未被引用的导出包级符号。抽成纯函数以便用合成源码做单元测试（见下）。
//
// 判定口径：
//   - 只统计**包级**声明（函数 / 类型 / 变量 / 常量），跳过方法与 Test/Benchmark/Example/Fuzz 入口；
//   - 跳过 `export_test.go`（约定合法的测试接缝）；
//   - 引用计数按**归一化包名 + 符号名**分组：`foo_test` 与 `foo` 视为同一作用域，
//     不同包中的同名符号互不抵消；标识符出现次数 ≤ 1（仅声明处）即视为无引用。
func findUnusedExportedTestSymbols(sources map[string][]byte) []testExportedDecl {
	fset := token.NewFileSet()
	type parsedFile struct {
		file *ast.File
		pkg  string
	}
	parsed := map[string]parsedFile{}
	refs := map[string]int{}

	for rel, data := range sources {
		f, err := parser.ParseFile(fset, rel, data, 0)
		if err != nil {
			continue // 解析失败交给 go build / vet 报错，不在此重复。
		}
		pkg := strings.TrimSuffix(f.Name.Name, "_test")
		parsed[rel] = parsedFile{file: f, pkg: pkg}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				refs[pkg+"."+id.Name]++
			}
			return true
		})
	}

	var decls []testExportedDecl
	for rel, pf := range parsed {
		if !strings.HasSuffix(rel, "_test.go") || filepath.Base(rel) == "export_test.go" {
			continue
		}
		for _, dc := range pf.file.Decls {
			switch v := dc.(type) {
			case *ast.FuncDecl:
				if v.Recv != nil {
					continue // 方法：unused 会按接收者类型可达性判定。
				}
				n := v.Name.Name
				if !isExported(n) || isTestEntrypoint(n) {
					continue
				}
				decls = append(decls, testExportedDecl{file: rel, pkg: pf.pkg, name: n, kind: "func"})
			case *ast.GenDecl:
				for _, sp := range v.Specs {
					switch s := sp.(type) {
					case *ast.TypeSpec:
						if isExported(s.Name.Name) {
							decls = append(decls, testExportedDecl{file: rel, pkg: pf.pkg, name: s.Name.Name, kind: "type"})
						}
					case *ast.ValueSpec:
						for _, nm := range s.Names {
							if nm.Name != "_" && isExported(nm.Name) {
								decls = append(decls, testExportedDecl{file: rel, pkg: pf.pkg, name: nm.Name, kind: "var/const"})
							}
						}
					}
				}
			}
		}
	}

	var dead []testExportedDecl
	for _, d := range decls {
		if refs[d.pkg+"."+d.name] <= 1 {
			dead = append(dead, d)
		}
	}
	sort.Slice(dead, func(i, j int) bool {
		if dead[i].file != dead[j].file {
			return dead[i].file < dead[j].file
		}
		return dead[i].name < dead[j].name
	})
	return dead
}

// TestFindUnusedExportedTestSymbols 是上面门禁的**口径测试**：用合成源码证明
// 「无引用的导出测试符号」会被抓到，而下列合法形态不会被误伤——
// 被引用的导出符号、非导出符号、Test 入口、`export_test.go`、方法。
func TestFindUnusedExportedTestSymbols(t *testing.T) {
	t.Parallel()
	sources := map[string][]byte{
		// 死符号：导出函数 / 类型 / 常量，全无引用。
		"pkg/aaa_test.go": []byte(`package pkg

import "testing"

func DeadExportedFunc() int { return 1 }

type DeadExportedType struct{ X int }

const DeadExportedConst = 3

var DeadExportedVar = 4

func TestSomething(t *testing.T) {}
`),
		// 合法：被引用的导出符号、非导出符号、Test 入口。
		"pkg/bbb_test.go": []byte(`package pkg

func LiveExportedFunc() int { return 1 }

func liveUnexported() int { return 2 }

func TestUsesLive(t *testing.T) {
	_ = LiveExportedFunc()
	_ = liveUnexported()
}
`),
		// 合法：export_test.go 是约定的测试接缝，整体豁免。
		"pkg/export_test.go": []byte(`package pkg

func SeamForOtherPackage() int { return 1 }
`),
	}
	dead := findUnusedExportedTestSymbols(sources)

	got := map[string]bool{}
	for _, d := range dead {
		got[d.name] = true
	}
	for _, want := range []string{"DeadExportedFunc", "DeadExportedType", "DeadExportedConst", "DeadExportedVar"} {
		if !got[want] {
			t.Errorf("漏报死符号 %q（实得 %v）", want, got)
		}
	}
	for _, mustNot := range []string{"LiveExportedFunc", "liveUnexported", "TestUsesLive", "TestSomething", "SeamForOtherPackage"} {
		if got[mustNot] {
			t.Errorf("误报合法符号 %q", mustNot)
		}
	}
}

// TestFindUnusedExportedTestSymbolsScopesByPackage 钉住引用计数的作用域：
// 不同包中的同名导出符号不得互相「引用」；`foo_test` 外部测试包与 `foo` 视为同一作用域。
func TestFindUnusedExportedTestSymbolsScopesByPackage(t *testing.T) {
	t.Parallel()
	sources := map[string][]byte{
		// alpha 包：SameName 被同包测试引用 → 合法。
		"alpha/aaa_test.go": []byte(`package alpha

func SameName() int { return 1 }
`),
		"alpha/bbb_test.go": []byte(`package alpha

import "testing"

func TestUsesSameName(t *testing.T) { _ = SameName() }
`),
		// beta 包：同名 SameName 无任何引用 → 死符号；不能因为 alpha 用过就漏报。
		"beta/aaa_test.go": []byte(`package beta

func SameName() int { return 2 }
`),
		// gamma_test 外部测试包引用 gamma 的导出符号 → 视为同作用域，不得误报。
		"gamma/aaa_test.go": []byte(`package gamma

func ExportedSeam() int { return 3 }
`),
		"gamma/external_test.go": []byte(`package gamma_test

import "testing"

func TestUsesSeam(t *testing.T) { _ = ExportedSeam() }
`),
	}
	dead := findUnusedExportedTestSymbols(sources)
	got := map[string]bool{}
	for _, d := range dead {
		got[d.file+"|"+d.name] = true
	}
	if !got["beta/aaa_test.go|SameName"] {
		t.Errorf("漏报 beta 包中无引用的 SameName（跨包同名引用不应互相抵消）；实得 %v", got)
	}
	if got["alpha/aaa_test.go|SameName"] {
		t.Errorf("误报 alpha 包中被同包引用的 SameName")
	}
	if got["gamma/aaa_test.go|ExportedSeam"] {
		t.Errorf("误报 gamma 包中被 gamma_test 外部测试包引用的 ExportedSeam")
	}
}

// isExported 报告标识符是否导出（首字母大写）。
func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

// isTestEntrypoint 报告是否为 go test 框架识别的入口名（TestXxx / BenchmarkXxx / ExampleXxx / FuzzXxx）。
func isTestEntrypoint(name string) bool {
	for _, p := range []string{"Test", "Benchmark", "Example", "Fuzz"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
