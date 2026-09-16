package handler

// openapi_contract_test.go —— /api/openapi.json 的「契约强度」测试。
//
// 与 openapi_handler_test.go 的冒烟测试不同，本文件把规范当成一份对外契约逐条校验：
//   - routes.go ↔ 规范 双向一致（漏登记、陈旧条目都会失败）；
//   - 每个 operation 有 responses、summary/operationId、唯一 operationId、响应有 description；
//   - 路径模板 {param} 在每条 operation 上都声明为 in:path 且 required:true；
//   - 文档中所有本地 $ref 都解析到 components 中真实存在的条目；
//   - MarshalJSON 确定（含并发）且 paths/operation 的 JSON key 有序、method 小写；
//   - 顶层形状（openapi 版本 / info / servers / securitySchemes）。
//
// routes.go 仅被当作「只读事实来源」解析（go/parser 读 AST），不修改。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/weilai1949/s3clinet/server/internal/store"
)

// apiRouteCount 是 routes.go 中 mux.HandleFunc 注册的 API 路由总数。
// 口径：69 = 68 条业务/系统端点 + GET /api/openapi.json 自指；
// SPA fallback 用 mux.Handle("/", ...) 注册，属于前端资源而非 /api 契约，不计入。
const apiRouteCount = 69

// ---- 夹具 ----

// newContractHandler 构造一个暴露 openapi.json 的 handler（不配 token，避免鉴权干扰契约读取）。
func newContractHandler(t *testing.T) *Handler {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	h := New(st, quietLogger(), t.TempDir(), nil, "", "contract-test", false, true)
	t.Cleanup(h.Shutdown)
	return h
}

// fetchOpenAPIJSON 通过真实 HTTP 端点取回规范原文（走完整中间件栈，验证的是对外可见行为）。
func fetchOpenAPIJSON(t *testing.T) []byte {
	t.Helper()
	h := newContractHandler(t)
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + "/api/openapi.json")
	if err != nil {
		t.Fatalf("GET /api/openapi.json: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/openapi.json status = %d, want 200", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read openapi.json: %v", err)
	}
	return b
}

func openAPIDoc(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(fetchOpenAPIJSON(t), &doc); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	return doc
}

// openAPIPaths 把 doc["paths"] 规整为 map[path]map[method]operation。
// 每层都做类型断言，形状不对就立刻失败，避免后续索引 panic。
func openAPIPaths(t *testing.T, doc map[string]any) map[string]map[string]map[string]any {
	t.Helper()
	raw, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("doc[\"paths\"] 类型 = %T, want object", doc["paths"])
	}
	out := make(map[string]map[string]map[string]any, len(raw))
	for p, v := range raw {
		ops, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("paths[%q] 类型 = %T, want object", p, v)
		}
		methods := make(map[string]map[string]any, len(ops))
		for m, opRaw := range ops {
			op, ok := opRaw.(map[string]any)
			if !ok {
				t.Fatalf("paths[%q].%s 类型 = %T, want object", p, m, opRaw)
			}
			methods[m] = op
		}
		out[p] = methods
	}
	return out
}

// routesFromSource 用 go/parser 读取 routes.go 的 AST，抽取 mux.HandleFunc("METHOD /path", ...) 清单。
// 按「只读事实来源」处理：测试不改 routes.go，只做文本/AST 解析。
func routesFromSource(t *testing.T) map[string]map[string]bool {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位 routes.go")
	}
	routesPath := filepath.Join(filepath.Dir(thisFile), "routes.go")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, routesPath, nil, 0)
	if err != nil {
		t.Fatalf("解析 %s: %v", routesPath, err)
	}

	out := map[string]map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "HandleFunc" || len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		pattern, err := strconv.Unquote(lit.Value)
		if err != nil {
			t.Fatalf("routes.go: 无法解析字面量 %s: %v", lit.Value, err)
		}
		method, path, found := strings.Cut(pattern, " ")
		if !found || method == "" || !strings.HasPrefix(path, "/") {
			t.Fatalf("routes.go: HandleFunc 模式 %q 不是 \"METHOD /path\" 形式", pattern)
		}
		method = strings.ToUpper(method)
		if out[path] == nil {
			out[path] = map[string]bool{}
		}
		if out[path][method] {
			t.Fatalf("routes.go: %s %s 被重复注册", method, path)
		}
		out[path][method] = true
		return true
	})
	if len(out) == 0 {
		t.Fatal("routes.go: 未解析到任何 mux.HandleFunc 路由")
	}
	return out
}

func countOperations(paths map[string]map[string]map[string]any) int {
	n := 0
	for _, ops := range paths {
		n += len(ops)
	}
	return n
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func sortedPaths(paths map[string]map[string]map[string]any) []string {
	keys := make([]string, 0, len(paths))
	for p := range paths {
		keys = append(keys, p)
	}
	return sortedStrings(keys)
}

func sortedMethods(ops map[string]map[string]any) []string {
	keys := make([]string, 0, len(ops))
	for m := range ops {
		keys = append(keys, m)
	}
	return sortedStrings(keys)
}

// ---- 1. 路由 ↔ 规范 双向一致 ----

// TestOpenAPI_ContractRoutesMatchSpec 验证 routes.go 的每条 API 路由都在规范里有 path+method，
// 且规范里没有 routes.go 未注册的陈旧 path/method。
// 这比「method 数 >= N」强：漏登记与多登记（删除端点忘了下架）都会红灯。
func TestOpenAPI_ContractRoutesMatchSpec(t *testing.T) {
	t.Parallel()
	routes := routesFromSource(t)
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	// routes.go -> 规范
	for p, methods := range routes {
		ops, ok := paths[p]
		if !ok {
			t.Errorf("routes.go 注册了 path %s，但规范缺少该 path", p)
			continue
		}
		for m := range methods {
			if _, ok := ops[strings.ToLower(m)]; !ok {
				t.Errorf("routes.go 注册了 %s %s，但规范缺少该 operation", m, p)
			}
		}
	}
	// 规范 -> routes.go
	for p, ops := range paths {
		for m := range ops {
			if !routes[p][strings.ToUpper(m)] {
				t.Errorf("规范声明了 %s %s，但 routes.go 未注册（陈旧条目）", strings.ToUpper(m), p)
			}
		}
	}

	total := 0
	for _, methods := range routes {
		total += len(methods)
	}
	if total != apiRouteCount {
		t.Fatalf("routes.go 的 API 路由数 = %d, want %d；新增/删除路由时请同步更新 apiRouteCount 与 register*.go", total, apiRouteCount)
	}
	if got := countOperations(paths); got != total {
		t.Errorf("规范 operation 数 = %d, want %d", got, total)
	}
}

// ---- 2. operation 完整性 ----

// TestOpenAPI_ContractOperationsAreComplete 验证每个 operation 都有
// responses（且每条响应有 description、状态码合法）以及 summary/operationId 至少一个非空；
// 额外验证 operationId 全局唯一（防止复制粘贴后忘记改名）。
func TestOpenAPI_ContractOperationsAreComplete(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	statusRe := regexp.MustCompile(`^([1-5][0-9]{2}|default)$`)
	seen := map[string]string{}
	for _, p := range sortedPaths(paths) {
		for _, m := range sortedMethods(paths[p]) {
			op := paths[p][m]
			summary, _ := op["summary"].(string)
			opID, _ := op["operationId"].(string)
			if strings.TrimSpace(summary) == "" && strings.TrimSpace(opID) == "" {
				t.Errorf("%s %s: summary 与 operationId 均为空", m, p)
			}
			if opID != "" {
				if prev, dup := seen[opID]; dup {
					t.Errorf("operationId %q 重复：%s 与 %s %s", opID, prev, m, p)
				}
				seen[opID] = m + " " + p
			}
			resps, ok := op["responses"].(map[string]any)
			if !ok || len(resps) == 0 {
				t.Errorf("%s %s: 缺少 responses（OpenAPI 3.0 要求至少一个响应）", m, p)
				continue
			}
			for status, raw := range resps {
				if !statusRe.MatchString(status) {
					t.Errorf("%s %s: 响应状态码 %q 非法", m, p, status)
				}
				rm, ok := resolveRawResponse(doc, raw)
				if !ok {
					t.Errorf("%s %s: 响应 %s 类型 = %T, want object（含可解析的 $ref）", m, p, status, raw)
					continue
				}
				if desc, _ := rm["description"].(string); strings.TrimSpace(desc) == "" {
					t.Errorf("%s %s: 响应 %s 缺少 description", m, p, status)
				}
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("未解析到任何 operationId，规范疑似为空")
	}
}

// ---- 3. 路径参数 ----

var pathTemplateParamRe = regexp.MustCompile(`\{([^{}]+)\}`)

// TestOpenAPI_ContractPathParamsDeclared 验证路径模板里每个 {param} 在同 path 的每条 operation 上
// 都声明为 in:path & required:true & name 一致，且不存在模板里没有的多余 path 参数。
func TestOpenAPI_ContractPathParamsDeclared(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	validIn := map[string]bool{"path": true, "query": true, "header": true, "cookie": true}
	checkedTemplates := 0
	for _, p := range sortedPaths(paths) {
		want := map[string]bool{}
		for _, sub := range pathTemplateParamRe.FindAllStringSubmatch(p, -1) {
			want[sub[1]] = true
		}
		if len(want) > 0 {
			checkedTemplates++
		}
		for _, m := range sortedMethods(paths[p]) {
			op := paths[p][m]
			got := map[string]bool{}
			params, _ := op["parameters"].([]any)
			for _, rawParam := range params {
				pm, ok := resolveRawParam(doc, rawParam)
				if !ok {
					t.Errorf("%s %s: parameter 类型 = %T, want object（含可解析的 $ref）", m, p, rawParam)
					continue
				}
				name, _ := pm["name"].(string)
				in, _ := pm["in"].(string)
				if name == "" {
					t.Errorf("%s %s: parameter 缺少 name", m, p)
				}
				if !validIn[in] {
					t.Errorf("%s %s: parameter %q 的 in=%q 非法", m, p, name, in)
				}
				if pm["schema"] == nil {
					t.Errorf("%s %s: parameter %q 缺少 schema", m, p, name)
				}
				if in != "path" {
					continue
				}
				got[name] = true
				if pm["required"] != true {
					t.Errorf("%s %s: path 参数 %q 必须 required:true", m, p, name)
				}
				if !want[name] {
					t.Errorf("%s %s: 声明了路径模板中不存在的 path 参数 %q", m, p, name)
				}
			}
			for name := range want {
				if !got[name] {
					t.Errorf("%s %s: 路径模板参数 {%s} 未声明为 in:path / required:true", m, p, name)
				}
			}
		}
	}
	if checkedTemplates == 0 {
		t.Fatal("没有任何带 {param} 的路径，本测试失去意义")
	}
}

// ---- 4. $ref 解析 ----

// collectRefs 递归收集文档中所有 "$ref" 字符串值。
func collectRefs(v any) []string {
	var out []string
	var walk func(any)
	walk = func(node any) {
		switch t := node.(type) {
		case map[string]any:
			for k, val := range t {
				if k == "$ref" {
					if s, ok := val.(string); ok {
						out = append(out, s)
					}
					continue
				}
				walk(val)
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(v)
	return out
}

// resolveLocalRef 解析形如 #/a/b/c 的本地 JSON Pointer 引用，返回是否命中 root 中的真实条目。
// 非本地引用（外部文件、绝对 URL）一律视为无法在本文档内解析。
func resolveLocalRef(root map[string]any, ref string) bool {
	if !strings.HasPrefix(ref, "#/") {
		return false
	}
	var cur any = root
	for _, seg := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		seg = strings.ReplaceAll(seg, "~1", "/")
		seg = strings.ReplaceAll(seg, "~0", "~")
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		cur, ok = m[seg]
		if !ok {
			return false
		}
	}
	return true
}

// derefComponent 解析 doc 中 "#/components/<kind>/<name>" 引用并返回被引用的条目；
// 无法解析时返回 (nil, false)。供契约测试在 $ref 接线后仍能校验 name/in/description 等。
func derefComponent(doc map[string]any, ref string) (map[string]any, bool) {
	if !strings.HasPrefix(ref, "#/components/") {
		return nil, false
	}
	var cur any = doc
	for _, seg := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		seg = strings.ReplaceAll(seg, "~1", "/")
		seg = strings.ReplaceAll(seg, "~0", "~")
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[seg]
		if !ok {
			return nil, false
		}
	}
	out, ok := cur.(map[string]any)
	return out, ok
}

// resolveRawParam 把 operation 参数列表规整为「可校验形态」：$ref 参数解析为
// components.parameters 中的实体，内联参数原样返回。
func resolveRawParam(doc map[string]any, raw any) (map[string]any, bool) {
	pm, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	if ref, _ := pm["$ref"].(string); ref != "" {
		resolved, ok := derefComponent(doc, ref)
		return resolved, ok
	}
	return pm, true
}

// resolveRawResponse 把响应条目规整为「可校验形态」：$ref 响应解析为
// components.responses 中的实体，内联响应原样返回。
func resolveRawResponse(doc map[string]any, raw any) (map[string]any, bool) {
	rm, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	if ref, _ := rm["$ref"].(string); ref != "" {
		resolved, ok := derefComponent(doc, ref)
		return resolved, ok
	}
	return rm, true
}

// TestOpenAPI_ContractRefsResolve 验证文档中每个 $ref 都指向 components 中真实存在的条目，
// 且引用形式统一为 #/components/<kind>/<name>（契约要求复用片段集中在 components）。
func TestOpenAPI_ContractRefsResolve(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	refs := collectRefs(doc)
	if len(refs) == 0 {
		t.Fatal("规范中没有任何 $ref：components 片段未被接线（应通过 refSchema/refParam/refResp 复用）")
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		seen[ref] = true
		if !strings.HasPrefix(ref, "#/components/") {
			t.Errorf("$ref %q 未指向 components（应为 #/components/...）", ref)
			continue
		}
		if !resolveLocalRef(doc, ref) {
			t.Errorf("$ref %q 无法解析到 components 中的条目", ref)
		}
	}
	t.Logf("已校验 %d 个 $ref（%d 个唯一引用目标）", len(refs), len(seen))
}

// TestOpenAPI_ContractRefResolverSelfCheck 用合成文档证明 $ref 解析器真的能区分可解析与悬空引用，
// 避免上一条测试因「文档恰好没有 $ref」而空转。
func TestOpenAPI_ContractRefResolverSelfCheck(t *testing.T) {
	t.Parallel()
	doc := map[string]any{
		"components": map[string]any{
			"schemas":    map[string]any{"Ok": map[string]any{"type": "object"}},
			"parameters": map[string]any{"AccountID": map[string]any{"name": "id", "in": "path"}},
		},
		"paths": map[string]any{
			"/api/x": map[string]any{
				"get": map[string]any{
					"parameters": []any{map[string]any{"$ref": "#/components/parameters/AccountID"}},
					"responses": map[string]any{
						"200": map[string]any{"content": map[string]any{
							"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Ok"}},
						}},
					},
				},
			},
		},
	}

	refs := collectRefs(doc)
	if len(refs) != 2 {
		t.Fatalf("collectRefs = %v, want 2 个 $ref", refs)
	}
	for _, ref := range refs {
		if !resolveLocalRef(doc, ref) {
			t.Errorf("resolveLocalRef(%q) = false, want true", ref)
		}
	}
	for _, bad := range []string{
		"#/components/schemas/Missing",
		"#/components/nope/Ok",
		"#/other/x",
		"external.yaml#/components/schemas/Ok",
		"",
	} {
		if resolveLocalRef(doc, bad) {
			t.Errorf("resolveLocalRef(%q) = true, want false（悬空/外部引用必须被识别）", bad)
		}
	}
}

// TestOpenAPI_ContractComponentsAreReferenced 防止 components 重新变成「死代码」：
// schemas / parameters 里的每个片段必须至少被一个 $ref 引用一次（端点内联 schema 不得
// 重复造轮子，复用片段应通过 refSchema/refParam 接线）。responses 例外：Unauthorized /
// TooManyRequests / InternalError 是全局错误语义（Bearer 鉴权全局生效），作为契约错误词汇
// 保留在 components 中，即使没有单个端点显式列出它们。
func TestOpenAPI_ContractComponentsAreReferenced(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	comps, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("components 类型 = %T, want object", doc["components"])
	}

	referenced := map[string]bool{}
	for _, ref := range collectRefs(doc) {
		referenced[ref] = true
	}

	// 全局错误词汇：即使无端点显式引用，也属于对外契约的一部分。
	globalErrorResponses := map[string]bool{
		"#/components/responses/Unauthorized":    true,
		"#/components/responses/TooManyRequests": true,
		"#/components/responses/InternalError":   true,
	}

	for _, kind := range []string{"schemas", "parameters", "responses"} {
		entries, ok := comps[kind].(map[string]any)
		if !ok {
			t.Errorf("components.%s 类型 = %T, want object", kind, comps[kind])
			continue
		}
		for name := range entries {
			ref := "#/components/" + kind + "/" + name
			if referenced[ref] {
				continue
			}
			if globalErrorResponses[ref] {
				t.Logf("components.%s.%s 为全局错误词汇（未接线属预期）", kind, name)
				continue
			}
			t.Errorf("components.%s.%s 未被任何 $ref 引用（死片段；应通过 refSchema/refParam/refResp 接线）", kind, name)
		}
	}
}

// ---- 5. 序列化确定性 + key 有序 ----

// TestOpenAPI_ContractMarshalDeterministic 验证两次（以及并发 8×8 次）MarshalJSON 输出字节完全一致。
// 并发部分在 -race 下同时覆盖 Registry 的 RWMutex 保护。
func TestOpenAPI_ContractMarshalDeterministic(t *testing.T) {
	t.Parallel()
	h := newContractHandler(t)

	first, err := h.openapi.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON #1: %v", err)
	}
	second, err := h.openapi.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON #2: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("两次 MarshalJSON 输出不一致：len %d vs %d", len(first), len(second))
	}
	if len(first) == 0 {
		t.Fatal("MarshalJSON 输出为空")
	}

	const workers, rounds = 8, 8
	var wg sync.WaitGroup
	failures := make(chan string, workers*rounds)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < rounds; j++ {
				b, err := h.openapi.MarshalJSON()
				if err != nil {
					failures <- fmt.Sprintf("并发 MarshalJSON: %v", err)
					return
				}
				if !bytes.Equal(b, first) {
					failures <- "并发 MarshalJSON 输出与串行结果不一致"
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for msg := range failures {
		t.Error(msg)
	}
}

// TestOpenAPI_ContractJSONKeysSorted 用 token 流读取原始 JSON（而非 map），
// 验证 paths 对象及其下每个 operation 对象的 key 按字典序输出、method 为小写合法 HTTP 方法。
func TestOpenAPI_ContractJSONKeysSorted(t *testing.T) {
	t.Parallel()
	raw := fetchOpenAPIJSON(t)

	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	pathsRaw, ok := top["paths"]
	if !ok {
		t.Fatal("顶层缺少 paths")
	}
	pathKeys := objectKeyOrder(t, pathsRaw)
	assertSortedKeys(t, "paths", pathKeys)

	var pathValues map[string]json.RawMessage
	if err := json.Unmarshal(pathsRaw, &pathValues); err != nil {
		t.Fatalf("decode paths: %v", err)
	}
	methodCount := 0
	for _, p := range pathKeys {
		opKeys := objectKeyOrder(t, pathValues[p])
		assertSortedKeys(t, "paths."+p, opKeys)
		for _, m := range opKeys {
			methodCount++
			if m != strings.ToLower(m) {
				t.Errorf("paths.%s 的 method %q 必须小写", p, m)
			}
			switch m {
			case "get", "put", "post", "delete", "options", "head", "patch", "trace":
			default:
				t.Errorf("paths.%s 的 method %q 不是合法 HTTP 方法", p, m)
			}
		}
	}
	if methodCount != apiRouteCount {
		t.Errorf("规范 method 数 = %d, want %d", methodCount, apiRouteCount)
	}
}

// objectKeyOrder 返回一个 JSON object 顶层 key 的**出现顺序**。
func objectKeyOrder(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("读取 object 起始 token: %v", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		t.Fatalf("期望 JSON object，实际起始 token = %v", tok)
	}
	var keys []string
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			t.Fatalf("读取 object key: %v", err)
		}
		key, ok := kt.(string)
		if !ok {
			t.Fatalf("object key 类型 = %T, want string", kt)
		}
		keys = append(keys, key)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatalf("跳过 key %q 的值: %v", key, err)
		}
	}
	if _, err := dec.Token(); err != nil { // 消费收尾 '}'
		t.Fatalf("读取 object 收尾 token: %v", err)
	}
	return keys
}

func assertSortedKeys(t *testing.T, where string, keys []string) {
	t.Helper()
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Errorf("%s 的 JSON key 未按字典序输出：%q 出现在 %q 之后（%v）", where, keys[i], keys[i-1], keys)
			return
		}
	}
}

// ---- 6. 顶层形状 ----

// TestOpenAPI_ContractTopLevelShape 验证 OpenAPI 版本、info、servers、components 与 bearerAuth 齐全。
func TestOpenAPI_ContractTopLevelShape(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)

	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v, want 3.0.3", doc["openapi"])
	}

	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatalf("info 类型 = %T, want object", doc["info"])
	}
	for _, k := range []string{"title", "version"} {
		s, _ := info[k].(string)
		if strings.TrimSpace(s) == "" {
			t.Errorf("info.%s 为空（OpenAPI 3.0 要求非空）", k)
		}
	}

	servers, ok := doc["servers"].([]any)
	if !ok || len(servers) == 0 {
		t.Errorf("servers = %v, want 非空数组", doc["servers"])
	}
	for i, raw := range servers {
		sm, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("servers[%d] 类型 = %T, want object", i, raw)
			continue
		}
		if url, _ := sm["url"].(string); strings.TrimSpace(url) == "" {
			t.Errorf("servers[%d].url 为空", i)
		}
	}

	comps, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("components 类型 = %T, want object", doc["components"])
	}
	for _, k := range []string{"schemas", "parameters", "responses", "securitySchemes"} {
		m, ok := comps[k].(map[string]any)
		if !ok || len(m) == 0 {
			t.Errorf("components.%s 缺失或为空", k)
		}
	}
	sec, _ := comps["securitySchemes"].(map[string]any)
	bearer, ok := sec["bearerAuth"].(map[string]any)
	if !ok {
		t.Fatalf("components.securitySchemes.bearerAuth 缺失")
	}
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Errorf("bearerAuth = %v, want {type: http, scheme: bearer}", bearer)
	}
}
