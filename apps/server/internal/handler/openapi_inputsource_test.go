package handler

// openapi_inputsource_test.go —— 注册表「请求体声明」与 handler「实际输入源」的机械对齐门禁。
//
// 背景：2026-09-19 的字段级门禁发现三处客户端可见失真（mkdir 注册表写 prefix、copy-objects 写
// items、DELETE /version 把 query 参数误声明为 requestBody）。`openapi_contract_test.go` 的
// 逐条断言只能覆盖「已经发现过」的端点，新端点写错输入源仍会漏。
//
// 本门禁把它做成通用比对：解析 routes.go 得到「路由 → handler 方法」，沿 handler 方法之间的
// 调用闭包找 `readJSON` / `json.NewDecoder`，与注册表每个 operation 是否声明 `Request:` 双向比对。
// 委托解码（如 parseMigrateRequest / parseCopyPrefix）通过调用闭包自然覆盖。
//
// 刻意不做的事：不校验字段名（由 TestAPIDocDocumentsRequestBodyFields 与契约测试负责）、
// 不校验类型与 required 语义。
//
// 判定实现走 **AST**（见 decodesBody / parseBodyCalls）：早先用裸字符串匹配 `readJSON(`，
// 结果注释与字符串字面量都能骗过门禁——删掉真实解码、只在注释里留一句 `readJSON(` 即假绿。
// 口径由 TestDecodesBodyIgnoresCommentsAndStrings 用合成源码钉住。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var (
	// routeRe 匹配 routes.go 的注册行，兼容 h.method 与 h.withStreamLimit(h.method) 两种写法。
	routeRe = regexp.MustCompile(`mux\.HandleFunc\("([A-Z]+) (/api\S*)",\s*h\.(\w+)(?:\(h\.(\w+)\))?\)`)
	// methodRe 匹配 handler 方法定义，用于切出方法体。
	methodRe = regexp.MustCompile(`(?m)^func \(h \*Handler\) (\w+)\(`)
	// opRe 匹配注册表的 operation 声明。
	opRe = regexp.MustCompile(`r\.Operation\("([A-Z]+)", "([^"]+)"`)
)

// handlerSourceDir 返回本测试所在目录（internal/handler）。
func handlerSourceDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位 handler 源码目录")
	}
	return filepath.Dir(thisFile)
}

// readHandlerSources 读取目录下所有非测试 Go 源码。
func readHandlerSources(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			rel = path
		}
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s: %v", dir, err)
	}
	return out
}

// parseHandlerMethods 返回 方法名 -> 方法体。
func parseHandlerMethods(sources map[string]string) map[string]string {
	out := map[string]string{}
	for _, text := range sources {
		ms := methodRe.FindAllStringSubmatchIndex(text, -1)
		for i, m := range ms {
			name := text[m[2]:m[3]]
			start := m[0]
			end := len(text)
			if i+1 < len(ms) {
				end = ms[i+1][0]
			}
			out[name] = text[start:end]
		}
	}
	return out
}

// parseRoutes 返回 "METHOD /path" -> handler 方法名。
func parseRoutes(t *testing.T, sources map[string]string) map[string]string {
	t.Helper()
	text, ok := sources["routes.go"]
	if !ok {
		t.Fatal("未找到 routes.go")
	}
	out := map[string]string{}
	for _, m := range routeRe.FindAllStringSubmatch(text, -1) {
		method := m[3]
		if m[4] != "" { // h.withStreamLimit(h.inner)
			method = m[4]
		}
		out[m[1]+" "+m[2]] = method
	}
	return out
}

// parseRegistryBodyDecl 返回 "METHOD /path" -> 该 operation 是否声明了 requestBody。
func parseRegistryBodyDecl(sources map[string]string) map[string]bool {
	out := map[string]bool{}
	for name, text := range sources {
		if !strings.HasPrefix(name, "openapi_register_") || !strings.HasSuffix(name, ".go") {
			continue
		}
		ops := opRe.FindAllStringSubmatchIndex(text, -1)
		for i, m := range ops {
			key := text[m[2]:m[3]] + " " + text[m[4]:m[5]]
			end := len(text)
			if i+1 < len(ops) {
				end = ops[i+1][0]
			}
			out[key] = strings.Contains(text[m[0]:end], "Request:")
		}
	}
	return out
}

// decodesBody 沿 handler 方法调用闭包判断某方法是否最终会解码请求体。
// 记忆化 + 置 false 防环；结果对每个方法只算一次。
//
// 判定走 **AST** 而非裸字符串匹配：裸匹配会把注释（`// 这里提到 readJSON(`）与字符串
// 字面量（`s := "readJSON("`）也算作「真的解码了请求体」——删掉真实解码、只在注释里留一句
// 就能让门禁假绿。实测该缺口存在，故改为解析语法树，只认真实的调用表达式。
func decodesBody(methods map[string]string) map[string]bool {
	// 先把每个方法体解析成 AST，并抽出它调用的其它 handler 方法名。
	parsed := make(map[string]bodyCalls, len(methods))
	for name, body := range methods {
		parsed[name] = parseBodyCalls(name, body)
	}

	memo := map[string]bool{}
	var visit func(name string) bool
	visit = func(name string) bool {
		if v, done := memo[name]; done {
			return v
		}
		memo[name] = false // 防调用环
		bc, ok := parsed[name]
		if !ok {
			return false
		}
		found := bc.decodes
		if !found {
			for _, callee := range bc.calls {
				if visit(callee) {
					found = true
					break
				}
			}
		}
		memo[name] = found
		return found
	}
	for name := range methods {
		visit(name)
	}
	return memo
}

// bodyCalls 是一个 handler 方法体的 AST 抽取结果。
type bodyCalls struct {
	decodes bool     // 是否真实调用了 readJSON / json.NewDecoder（AST 判定，不受注释影响）
	calls   []string // 调用的其它 handler 方法名（用于委托闭包）
}

// parseBodyCalls 把方法体包成一个函数声明后解析，抽取真实调用。
// go/parser 只做语法解析、不做类型检查，故无需为 `json.NewDecoder` 等选择器补 import。
func parseBodyCalls(name, body string) bodyCalls {
	out := bodyCalls{}
	src := "package p\n\n" + body
	f, err := parser.ParseFile(token.NewFileSet(), name+".go", src, 0)
	if err != nil {
		return out
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		// 只认真实请求体解码：`h.readJSON(...)` 与 `json.NewDecoder(...)`。
		// 其它 receiver 的 `readJSON` / `NewDecoder` 不得算作请求体解码（receiver 负例见
		// TestParseBodyCallsDecodeReceivers）。
		switch sel.Sel.Name {
		case "readJSON":
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "h" {
				out.decodes = true
			}
		case "NewDecoder":
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "json" {
				out.decodes = true
			}
		}
		// 收集 h.<method>(...) 形式的委托调用（用于沿闭包查找解码点）。
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "h" {
			out.calls = append(out.calls, sel.Sel.Name)
		}
		return true
	})
	return out
}

// handlerMethodCalls 返回函数声明体内所有 `h.<method>(...)` 形式的真实调用（AST 判定）。
// 供各契约门禁的调用闭包遍历使用，替代会命中注释/字符串的裸正则 `h\.(\w+)\(`。
func handlerMethodCalls(fd *ast.FuncDecl) []string {
	if fd == nil || fd.Body == nil {
		return nil
	}
	var out []string
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "h" {
			out = append(out, sel.Sel.Name)
		}
		return true
	})
	return out
}

// TestDecodesBodyIgnoresCommentsAndStrings 是 decodesBody 的口径测试：
// 只有**真实的调用表达式**才算「解码请求体」；注释与字符串字面量里的 `readJSON(` 不算。
//
// 没有它，判定实现退回裸字符串匹配时门禁会静默假绿（删掉真解码、只在注释留一句即可通过）。
func TestDecodesBodyIgnoresCommentsAndStrings(t *testing.T) {
	t.Parallel()
	methods := map[string]string{
		// 只有注释提到 readJSON( —— 不得算作解码。
		"onlyComment": "func (h *Handler) onlyComment(w http.ResponseWriter, r *http.Request) {\n" +
			"\t// 这里提到 readJSON( 但并没有真的解码\n\t_ = 1\n}\n",
		// 只有字符串字面量 —— 不得算作解码。
		"stringLit": "func (h *Handler) stringLit(w http.ResponseWriter, r *http.Request) {\n" +
			"\ts := \"readJSON(\"\n\t_ = s\n}\n",
		// 真实调用 —— 必须算作解码。
		"realDecode": "func (h *Handler) realDecode(w http.ResponseWriter, r *http.Request) {\n" +
			"\tvar x struct{}\n\tif err := h.readJSON(r, &x); err != nil {\n\t\treturn\n\t}\n}\n",
		// 委托：自身不解码，但调用了解码的方法 —— 必须算作解码。
		"delegates": "func (h *Handler) delegates(w http.ResponseWriter, r *http.Request) {\n" +
			"\th.realDecode(w, r)\n}\n",
		// 什么都不做。
		"nothing": "func (h *Handler) nothing(w http.ResponseWriter, r *http.Request) {\n}\n",
	}
	got := decodesBody(methods)

	want := map[string]bool{
		"onlyComment": false,
		"stringLit":   false,
		"realDecode":  true,
		"delegates":   true,
		"nothing":     false,
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("decodesBody(%s) = %v, want %v", name, got[name], w)
		}
	}
}

// TestParseBodyCallsIgnoresCommentsAndStrings 钉住「调用闭包」的 AST 口径：
// 注释与字符串字面量里的 `h.ghost()` 不得进入闭包，只有真实调用表达式才算。
// 各契约门禁（path / query / request fields / semantics）都消费 parseBodyCalls 的 calls，
// 若退回裸正则，注释即可污染闭包并掩盖漂移。
func TestParseBodyCallsIgnoresCommentsAndStrings(t *testing.T) {
	t.Parallel()
	body := "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n" +
		"\t// h.ghost()\n" +
		"\ts := \"h.alsoGhost()\"\n\t_ = s\n" +
		"\th.real(w, r)\n}\n"
	bc := parseBodyCalls("m", body)
	want := map[string]bool{"real": true}
	got := map[string]bool{}
	for _, c := range bc.calls {
		got[c] = true
	}
	if len(got) != len(want) || !got["real"] {
		t.Errorf("parseBodyCalls calls = %v, want 仅 [real]（注释/字符串里的 h.x() 不得进入闭包）", bc.calls)
	}
	if bc.decodes {
		t.Error("parseBodyCalls 不应把该函数判为解码请求体")
	}
}

// TestParseBodyCallsDecodeReceivers 钉住解码判定的 receiver：
// 只有 `h.readJSON(...)` 与 `json.NewDecoder(...)` 算请求体解码；
// `other.readJSON(...)` / `other.NewDecoder(...)` 不得误报。
func TestParseBodyCallsDecodeReceivers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"h.readJSON", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\tvar x struct{}\n\t_ = h.readJSON(r, &x)\n}\n", true},
		{"json.NewDecoder", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t_ = json.NewDecoder(r.Body)\n}\n", true},
		{"other.readJSON", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\tvar x struct{}\n\t_ = other.readJSON(r, &x)\n}\n", false},
		{"other.NewDecoder", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t_ = other.NewDecoder(r.Body)\n}\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseBodyCalls("m", tc.body).decodes; got != tc.want {
				t.Errorf("parseBodyCalls.decodes = %v, want %v（body=%q）", got, tc.want, tc.body)
			}
		})
	}
}

// TestOpenAPIRequestDeclarationMatchesHandlerInput 双向校验：
// 注册表声明 requestBody ⇔ handler（含委托）实际解码请求体。
func TestOpenAPIRequestDeclarationMatchesHandlerInput(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	methods := parseHandlerMethods(sources)
	routes := parseRoutes(t, sources)
	decl := parseRegistryBodyDecl(sources)
	decodes := decodesBody(methods)

	// 自检：解析口径写坏时不要静默变绿。
	if len(routes) < 60 || len(methods) < 60 || len(decl) < 60 {
		t.Fatalf("解析结果异常（routes=%d methods=%d operations=%d），疑似解析口径失效",
			len(routes), len(methods), len(decl))
	}

	checked := 0
	for key, method := range routes {
		want, declared := decl[key]
		if !declared {
			t.Errorf("routes.go 注册了 %s，但注册表没有对应 operation", key)
			continue
		}
		checked++
		if got := decodes[method]; got != want {
			if want {
				t.Errorf("注册表为 %s 声明了 requestBody，但 handler %s 不解码请求体（参数会被忽略，客户端拿到 400）", key, method)
			} else {
				t.Errorf("handler %s（%s）解码请求体，但注册表未声明 requestBody（OpenAPI 客户端不会发送 body）", method, key)
			}
		}
	}
	if checked < 60 {
		t.Fatalf("仅比对 %d 条路由，疑似注册表与路由解析不匹配", checked)
	}
}
