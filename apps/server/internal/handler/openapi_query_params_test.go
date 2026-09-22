package handler

// openapi_query_params_test.go —— **全量** query 参数门禁：注册表声明 ⇔ handler 实际读取。
//
// 背景（docs/todolist.md #26 / docs/review-2026-09-19.md §7.3 D7、§7.4）：
// 此前的 query 断言是**逐端点硬编码**（`openapi_contract_test.go` 只覆盖 version DELETE 的
// bucket/key/versionId 三个），新端点漏声明 query 参数不会变红。实测漏了 4 个：
// `head.versionId`、`proxy.maxBytes`、`trash.prefix`、`objects.startAfter`。
//
// 本门禁做成机械全量遍历：
//   1. 解析 routes.go：路由 → handler 方法；
//   2. 沿 handler 方法调用闭包，机械抽取方法体内全部四种 query 读取口径：
//      `r.URL.Query().Get("...")` / `r.URL.Query().Has("...")` / `r.URL.Query().Values("...")`
//      / `r.URL.Query()["..."]`，以及 `q := r.URL.Query()` 绑定后的同名形式
//      `q.Get` / `q.Has` / `q.Values` / `q["..."]`（覆盖 parseMigrateRequest 这类委托）；
//   3. 与注册表该 operation 的 `in: query` 参数集**双向**比对。
//
// 双向规则：
//   - handler 读取但注册表未声明 → 按 OpenAPI 生成的客户端不会发送该参数（漏声明）；
//   - 注册表声明但 handler 不读取 → 幻影参数（客户端发了也被忽略，属契约谎言）。
//
// 断言范围（刻意不做的事）：
//   - 只校验参数**名**，不校验类型 / required / 枚举（由 openapi_semantics_test.go 覆盖）。
//   - 参数名来自运行时字符串（如 `q.Get(name)` 的变量）无法机械抽取；生产代码不得出现，
//     一旦出现须改为字面量或在此登记残留范围（TestQueryReadExtractorCoversAllForms 兜底口径）。
//   - 内部专用参数（有意不公开）必须在 internalOnlyQueryParams 白名单登记原因，不得静默跳过；
//     当前为空（所有 handler 读取的 query 参数都属于公开契约）。
//
// 自检：解析口径写坏时不得静默变绿（routes / 方法数 / 已比对端点数均有下限断言）。

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// 本文件专用的解析正则。
var (
	// queryBindRe 匹配 `q := r.URL.Query()`，捕获 query 变量名。
	queryBindRe = regexp.MustCompile(`(\w+)\s*:=\s*r\.URL\.Query\(\)`)
	// queryMethodRe 匹配 `r.URL.Query().Get("x")` / `.Has("x")` / `.Values("x")`。
	queryDirectRe = regexp.MustCompile(`r\.URL\.Query\(\)\.(?:Get|Has|Values)\("([^"]+)"\)`)
	// queryDirectIndexRe 匹配 `r.URL.Query()["x"]`。
	queryDirectIndexRe = regexp.MustCompile(`r\.URL\.Query\(\)\["([^"]+)"\]`)
	// queryVarMethodRe 匹配 `q.Get("x")` / `q.Has("x")` / `q.Values("x")`。
	queryVarGetRe = regexp.MustCompile(`(\w+)\.(?:Get|Has|Values)\("([^"]+)"\)`)
	// queryVarIndexRe 匹配 `q["x"]`。
	queryVarIndexRe = regexp.MustCompile(`(\w+)\["([^"]+)"\]`)
	// queryDynamicRe 匹配对 query 变量使用**非字面量**键的读取（如 `q.Get(name)` / `q[k]`）。
	// 命中即说明存在无法机械抽取的读取口径，门禁自检会红灯要求改成字面量。
	queryDynamicMethodRe = regexp.MustCompile(`r\.URL\.Query\(\)\.(?:Get|Has|Values)\(\s*[^")\s]`)
	queryDynamicIndexRe  = regexp.MustCompile(`r\.URL\.Query\(\)\[\s*[^"\]]`)
)

// internalOnlyQueryParams 是「handler 读取、但刻意不进入公开契约」的 query 参数白名单。
// 空表示当前不存在此类参数；新增条目必须写明「为什么不是公开契约」。
var internalOnlyQueryParams = map[string]string{}

// hasDynamicQueryRead 报告方法体内是否存在「非字面量键」的 query 读取
// （`q.Get(name)` / `q[k]` 等），这类读取无法机械抽取参数名。
//
// 直连形式（`r.URL.Query().Get(name)`）由两个正则覆盖；绑定形式必须先抽出
// `q := r.URL.Query()` 的变量名，再逐变量检查 `q.Get(name)` / `q[k]`——
// 否则绑定后的动态键会同时逃过「抽取」与「动态检测」两道关（实测 fail-open）。
func hasDynamicQueryRead(body string) bool {
	if queryDynamicMethodRe.MatchString(body) || queryDynamicIndexRe.MatchString(body) {
		return true
	}
	for _, m := range queryBindRe.FindAllStringSubmatch(body, -1) {
		if queryVarDynamicRead(body, m[1]) {
			return true
		}
	}
	return false
}

// queryVarDynamicRead 检查 `q.Get(name)` / `q.Has(name)` / `q.Values(name)` / `q[k]`
// 这类对绑定 query 变量使用非字面量键的读取。
func queryVarDynamicRead(body, varName string) bool {
	name := regexp.QuoteMeta(varName)
	methodRe := regexp.MustCompile(`\b` + name + `\.(?:Get|Has|Values)\(\s*[^")\s]`)
	indexRe := regexp.MustCompile(`\b` + name + `\[\s*[^"\]]`)
	return methodRe.MatchString(body) || indexRe.MatchString(body)
}

// parseQueryReads 返回 方法名 -> 该方法（含调用闭包）读取的 query 参数名集合。
func parseQueryReads(methods map[string]string) map[string]map[string]bool {
	direct := map[string]map[string]bool{}
	for name, body := range methods {
		set := map[string]bool{}
		for _, m := range queryDirectRe.FindAllStringSubmatch(body, -1) {
			set[m[1]] = true
		}
		for _, m := range queryDirectIndexRe.FindAllStringSubmatch(body, -1) {
			set[m[1]] = true
		}
		var vars []string
		for _, m := range queryBindRe.FindAllStringSubmatch(body, -1) {
			vars = append(vars, m[1])
		}
		for _, m := range queryVarGetRe.FindAllStringSubmatch(body, -1) {
			for _, v := range vars {
				if m[1] == v {
					set[m[2]] = true
				}
			}
		}
		for _, m := range queryVarIndexRe.FindAllStringSubmatch(body, -1) {
			for _, v := range vars {
				if m[1] == v {
					set[m[2]] = true
				}
			}
		}
		direct[name] = set
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
		for _, c := range callRe.FindAllStringSubmatch(body, -1) {
			for k := range visit(c[1]) {
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

// TestQueryReadExtractorCoversAllForms 是抽取器自身的口径测试：
// 四种字面量读取形式都必须被识别，**非字面量键**必须被识别为「无法机械抽取」而红灯。
// 没有它，抽取正则写坏（例如漏掉 Has/Values）会让门禁静默变绿。
func TestQueryReadExtractorCoversAllForms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"Get 直取", `v := r.URL.Query().Get("alpha")`, []string{"alpha"}},
		{"Has 直取", `if r.URL.Query().Has("beta") {}`, []string{"beta"}},
		{"Values 直取", `vs := r.URL.Query().Values("gamma")`, []string{"gamma"}},
		{"下标直取", `vs := r.URL.Query()["delta"]`, []string{"delta"}},
		{"绑定后 Get", "q := r.URL.Query()\nv := q.Get(\"eps\")", []string{"eps"}},
		{"绑定后 Has", "q := r.URL.Query()\nif q.Has(\"zeta\") {}", []string{"zeta"}},
		{"绑定后 Values", "q := r.URL.Query()\nvs := q.Values(\"eta\")", []string{"eta"}},
		{"绑定后下标", "q := r.URL.Query()\nvs := q[\"theta\"]", []string{"theta"}},
		{"无读取", `v := "x"`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseQueryReads(map[string]string{"m": tc.body})["m"]
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

	// 非字面量键：必须被 detectDynamicQueryReads 标记，否则门禁会静默漏检。
	dynamic := []string{
		`v := r.URL.Query().Get(name)`,
		`vs := r.URL.Query()[k]`,
		"q := r.URL.Query()\nv := q.Get(name)",
		"q := r.URL.Query()\nvs := q[k]",
	}
	for _, body := range dynamic {
		if !hasDynamicQueryRead(body) {
			t.Errorf("非字面量键读取未被识别为动态读取（body=%q）：门禁会静默漏检", body)
		}
	}
	if hasDynamicQueryRead(`v := r.URL.Query().Get("literal")`) {
		t.Error("字面量读取被误判为动态读取")
	}
	for _, body := range []string{
		"q := r.URL.Query()\nv := q.Get(\"literal\")",
		"q := r.URL.Query()\nv := q[\"literal\"]",
	} {
		if hasDynamicQueryRead(body) {
			t.Errorf("绑定后的字面量读取被误判为动态读取（body=%q）", body)
		}
	}
}

// TestOpenAPIQueryParamsMatchHandlerReads 双向校验：
// 注册表 query 参数集 ⇔ handler（含委托闭包）实际读取的 query 参数集。
func TestOpenAPIQueryParamsMatchHandlerReads(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	methods := parseHandlerMethods(sources)
	routes := parseRoutes(t, sources)
	reads := parseQueryReads(methods)
	doc := openAPIDoc(t)

	// 自检：解析口径写坏时不要静默变绿。
	if len(routes) < 60 || len(methods) < 60 {
		t.Fatalf("解析结果异常（routes=%d methods=%d），疑似解析口径失效", len(routes), len(methods))
	}

	// 非字面量键读取无法机械抽取，必须显式失败而非静默跳过。
	for name, body := range methods {
		if hasDynamicQueryRead(body) {
			t.Errorf("handler %s 用非字面量键读取 query 参数（如 q.Get(name) / q[k]）；"+
				"本门禁无法机械抽取，请改为字面量键或扩展抽取器", name)
		}
	}

	checked := 0
	for key, method := range routes {
		httpMethod, path, found := strings.Cut(key, " ")
		if !found {
			t.Fatalf("路由 key %q 不是 \"METHOD /path\" 形式", key)
		}
		declared := requestQueryParams(t, doc, path, strings.ToLower(httpMethod))
		read := reads[method]
		checked++

		var missing, phantom []string
		for name := range read {
			if declared[name] {
				continue
			}
			if reason, ok := internalOnlyQueryParams[name]; ok {
				t.Logf("%s 的 query 参数 %q 为内部专用（%s），不公开", key, name, reason)
				continue
			}
			missing = append(missing, name)
		}
		for name := range declared {
			if !read[name] {
				phantom = append(phantom, name)
			}
		}
		sort.Strings(missing)
		sort.Strings(phantom)
		for _, name := range missing {
			t.Errorf("handler %s（%s）读取 query 参数 %q，但注册表未声明该 in:query 参数："+
				"按 OpenAPI 生成的客户端不会发送它", method, key, name)
		}
		for _, name := range phantom {
			t.Errorf("注册表为 %s 声明 query 参数 %q，但 handler %s 不读取它（幻影参数："+
				"客户端发送后会被忽略）", key, name, method)
		}
	}
	if checked < 60 {
		t.Fatalf("仅比对 %d 条路由，疑似遍历口径失效", checked)
	}
	t.Logf("query 参数全量比对：%d 个端点", checked)
}
