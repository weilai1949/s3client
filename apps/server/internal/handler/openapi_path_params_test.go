package handler

// openapi_path_params_test.go —— **全量** path 参数门禁：注册表声明 ⇔ handler 实际读取。
//
// 背景（docs/review-2026-09-19.md §4.3）：`openapi_contract_test.go` 的
// `TestOpenAPI_ContractPathParamsDeclared` 只校验「注册表**内部**自洽」——路径模板里的
// `{id}` 必须在同 path 的每条 operation 上声明为 `in:path & required:true`。它**不**比对
// handler 是否真的用 `r.PathValue("id")` 读了这个参数。于是这类漂移无人拦：
//
//   - 注册表把参数名从 `id` 改成 `accountId`（模板同步改了），但 handler 仍读 `PathValue("id")`
//     → 运行时拿到空串，账号路由全部 404 / 行为异常，而所有契约测试仍全绿；
//   - handler 读了模板里没有的参数名 → 永远拿到空串。
//
// 本门禁把「注册表 in:path 参数名集 ⇔ handler 沿调用闭包 PathValue 读取的参数名集」做双向比对，
// 与 query 参数门禁同构；调用闭包由 `parseBodyCalls` AST 抽取 `h.xxx()`，不受注释/字符串污染。
//
// 断言范围（刻意不做的事）：
//   - 只校验参数**名**，不校验类型 / required（required 由 `TestOpenAPI_ContractPathParamsDeclared`
//     保证；类型语义见 `openapi_semantics_test.go`）。
//   - 参数名来自运行时字符串（`PathValue(name)`）无法机械抽取；生产代码不得出现，
//     一旦出现须改为字面量或在此登记残留范围。
//   - 共享 `$ref` 参数（如 `AccountID`）经 `resolveRawParam` 解析后取其真实 `name`。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// pathParamReads 返回 方法名 -> 该方法（含调用闭包）读取的 path 参数名集合。
func pathParamReads(methods map[string]string) map[string]map[string]bool {
	direct := map[string]map[string]bool{}
	for name, body := range methods {
		direct[name] = parsePathValueReads(name, body)
	}

	memo := map[string]map[string]bool{}
	var visit func(name string) map[string]bool
	visit = func(name string) map[string]bool {
		if s, ok := memo[name]; ok {
			return s
		}
		memo[name] = map[string]bool{} // 防调用环
		body, ok := methods[name]
		if !ok {
			return memo[name]
		}
		set := map[string]bool{}
		for k := range direct[name] {
			set[k] = true
		}
		for _, c := range parseBodyCalls(name, body).calls {
			for k := range visit(c) {
				set[k] = true
			}
		}
		memo[name] = set
		return set
	}
	for name := range methods {
		visit(name)
	}
	return memo
}

// parsePathValueReads 用 AST 抽取方法体内 `r.PathValue("x")` 的字面量参数名。
// 走 AST 而非裸正则：注释与字符串字面量里的 `PathValue("x")` 不得被算作真实读取。
func parsePathValueReads(name, body string) map[string]bool {
	out := map[string]bool{}
	f, err := parser.ParseFile(token.NewFileSet(), name+".go", "package p\n\n"+body, 0)
	if err != nil {
		return out
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "PathValue" || len(call.Args) != 1 {
			return true
		}
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil {
				out[s] = true
			}
		}
		return true
	})
	return out
}

// hasDynamicPathValueRead 用 AST 判断方法体内是否存在对 PathValue 使用非字面量键的读取
// （如 `r.PathValue(name)`）。走 AST 而非裸正则：注释与字符串字面量里的
// `PathValue(name)` 不得触发红灯（旧正则会对两者误报）。
func hasDynamicPathValueRead(body string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), "probe.go", "package p\n\n"+body, 0)
	if err != nil {
		return false
	}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "PathValue" || len(call.Args) != 1 {
			return true
		}
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			return true
		}
		found = true
		return false
	})
	return found
}

// TestPathParamReadExtractorCoversForms 是抽取器自身的口径测试。
func TestPathParamReadExtractorCoversForms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want []string
	}{
		{
			"直接读取",
			"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\tid := r.PathValue(\"id\")\n\t_ = id\n}\n",
			[]string{"id"},
		},
		{
			"多个参数",
			"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t_ = r.PathValue(\"a\")\n\t_ = r.PathValue(\"b\")\n}\n",
			[]string{"a", "b"},
		},
		{
			"注释里的不算",
			"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t// r.PathValue(\"ghost\")\n}\n",
			nil,
		},
		{
			"字符串字面量里的不算",
			"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\ts := \"r.PathValue(\\\"ghost\\\")\"\n\t_ = s\n}\n",
			nil,
		},
		{
			"非字面量键抽不出来",
			"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t_ = r.PathValue(name)\n}\n",
			nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePathValueReads("m", tc.body)
			for _, w := range tc.want {
				if !got[w] {
					t.Errorf("抽取器漏掉 %q（body=%q，实得 %v）", w, tc.body, got)
				}
			}
			if len(tc.want) == 0 && len(got) != 0 {
				t.Errorf("不该抽出参数，实得 %v", got)
			}
		})
	}
}

// TestPathParamDynamicReadUsesAST 钉住动态键检测走 AST：注释与字符串字面量里的
// `PathValue(...)` 不得被当成读到了动态参数，真实调用 `r.PathValue(name)` 必须命中。
func TestPathParamDynamicReadUsesAST(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"真实动态键", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t_ = r.PathValue(name)\n}\n", true},
		{"真实字面量键", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t_ = r.PathValue(\"id\")\n}\n", false},
		{"注释里的不算", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t// r.PathValue(ghost)\n}\n", false},
		{"字符串字面量里的不算", "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\ts := \"r.PathValue(ghost)\"\n\t_ = s\n}\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasDynamicPathValueRead(tc.body); got != tc.want {
				t.Errorf("hasDynamicPathValueRead(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

// requestPathParams 返回某操作在 path 上的参数名集合（$ref 解析到 components.parameters）。
func requestPathParams(t *testing.T, doc map[string]any, path, method string) map[string]bool {
	t.Helper()
	paths := openAPIPaths(t, doc)
	op, ok := paths[path][method]
	if !ok {
		t.Fatalf("openapi 无 %s %s", method, path)
	}
	out := map[string]bool{}
	list, _ := op["parameters"].([]any)
	for _, raw := range list {
		p, ok := resolveRawParam(doc, raw)
		if !ok {
			t.Fatalf("%s %s: parameter 无法解析", method, path)
		}
		if in, _ := p["in"].(string); in != "path" {
			continue
		}
		if name, _ := p["name"].(string); name != "" {
			out[name] = true
		}
	}
	return out
}

// TestOpenAPIPathParamsMatchHandlerReads 双向校验：
// 注册表 in:path 参数集 ⇔ handler（含委托闭包）实际 PathValue 读取的参数集。
func TestOpenAPIPathParamsMatchHandlerReads(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	methods := parseHandlerMethods(sources)
	routes := parseRoutes(t, sources)
	reads := pathParamReads(methods)
	doc := openAPIDoc(t)

	// 自检：解析口径写坏时不要静默变绿。
	if len(routes) < 60 || len(methods) < 60 {
		t.Fatalf("解析结果异常（routes=%d methods=%d），疑似解析口径失效", len(routes), len(methods))
	}

	// 非字面量键读取无法机械抽取，必须显式失败而非静默跳过。
	for name, body := range methods {
		if hasDynamicPathValueRead(body) {
			t.Errorf("handler %s 用非字面量键读取 path 参数（如 r.PathValue(name)）；"+
				"本门禁无法机械抽取，请改为字面量键或扩展抽取器", name)
		}
	}

	checked, withPath := 0, 0
	for key, method := range routes {
		httpMethod, path, found := strings.Cut(key, " ")
		if !found {
			t.Fatalf("路由 key %q 不是 \"METHOD /path\" 形式", key)
		}
		declared := requestPathParams(t, doc, path, strings.ToLower(httpMethod))
		read := reads[method]
		checked++
		if len(declared) > 0 {
			withPath++
		}

		var missing, phantom []string
		for name := range read {
			if !declared[name] {
				missing = append(missing, name)
			}
		}
		for name := range declared {
			if !read[name] {
				phantom = append(phantom, name)
			}
		}
		sort.Strings(missing)
		sort.Strings(phantom)
		for _, name := range missing {
			t.Errorf("handler %s（%s）读取 path 参数 %q，但注册表未声明该 in:path 参数："+
				"按 OpenAPI 生成的客户端不会把它放进 URL 模板", method, key, name)
		}
		for _, name := range phantom {
			t.Errorf("注册表为 %s 声明 path 参数 %q，但 handler %s 不读取它（幻影参数："+
				"handler 拿不到该值，行为会与契约不符）", key, name, method)
		}
	}
	if checked < 60 {
		t.Fatalf("仅比对 %d 条路由，疑似遍历口径失效", checked)
	}
	// 自检：带 path 参数的路由数不得为 0，否则门禁形同虚设。
	if withPath == 0 {
		t.Fatal("没有任何路由声明 in:path 参数，疑似解析口径失效（门禁形同虚设）")
	}
	t.Logf("path 参数全量比对：%d 个端点（其中 %d 个带 path 参数）", checked, withPath)
}
