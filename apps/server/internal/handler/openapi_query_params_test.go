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
//   2. 沿 handler 方法调用闭包，机械抽取方法体内 `r.URL.Query().Get("...")` 与
//      `q := r.URL.Query()` 后的 `q.Get("...")`（覆盖 parseMigrateRequest 这类委托）；
//   3. 与注册表该 operation 的 `in: query` 参数集**双向**比对。
//
// 双向规则：
//   - handler 读取但注册表未声明 → 按 OpenAPI 生成的客户端不会发送该参数（漏声明）；
//   - 注册表声明但 handler 不读取 → 幻影参数（客户端发了也被忽略，属契约谎言）。
//
// 断言范围（刻意不做的事）：
//   - 只识别 `Get()` 读取形式；`Has()` / `Values()` / `Query()["x"]` 等其它读取口径不在本门禁内
//     （当前生产代码无此类调用，新增时须同步扩展抽取器）。
//   - 只校验参数**名**，不校验类型 / required / 枚举（由 openapi_semantics_test.go 覆盖）。
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
	// queryDirectRe 匹配 `r.URL.Query().Get("x")`。
	queryDirectRe = regexp.MustCompile(`r\.URL\.Query\(\)\.Get\("([^"]+)"\)`)
	// queryVarGetRe 匹配 `q.Get("x")`，结合 queryBindRe 判定接收者是否为 query 变量。
	queryVarGetRe = regexp.MustCompile(`(\w+)\.Get\("([^"]+)"\)`)
)

// internalOnlyQueryParams 是「handler 读取、但刻意不进入公开契约」的 query 参数白名单。
// 空表示当前不存在此类参数；新增条目必须写明「为什么不是公开契约」。
var internalOnlyQueryParams = map[string]string{}

// parseQueryReads 返回 方法名 -> 该方法（含调用闭包）读取的 query 参数名集合。
func parseQueryReads(methods map[string]string) map[string]map[string]bool {
	direct := map[string]map[string]bool{}
	for name, body := range methods {
		set := map[string]bool{}
		for _, m := range queryDirectRe.FindAllStringSubmatch(body, -1) {
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
