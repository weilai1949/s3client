package handler

// openapi_request_fields_test.go —— **全量**请求体字段门禁：注册表声明字段集 ⇔ handler 实际解码字段集。
//
// 背景（docs/review-2026-09-19.md §4.3）：此前的字段级断言是**逐端点族硬编码**
// （`openapi_contract_test.go` 第 1–8 项，只覆盖 2026-09 修过的那批端点），新端点把字段写错
// 仍然不会变红。本文件把它换成机械全量遍历：
//
//   1. 解析 routes.go：路由 → handler 方法；
//   2. 沿 handler 方法调用闭包（`parseBodyCalls` AST 抽取 `h.xxx()`）找到真正 `readJSON` 的方法
//      （覆盖 parseMigrateRequest / parseCopyPrefix 这类委托解码）；
//   3. 解析 readJSON 目标变量的字段集（内联匿名 struct 或命名 struct）；
//   4. 与注册表该 operation 的 requestBody schema 字段集比对。
//
// 比对规则（区分「端点专属 DTO」与「共享模型」）：
//   - **内联匿名 struct / 端点专属命名 struct**：双向完全相等。多一个 = 幻影字段
//     （按文档发送会被 `DisallowUnknownFields` 拒绝成 400）；少一个 = 客户端发不出该字段。
//   - **共享模型**（如 `model.Account`）：注册表字段集必须 ⊆ handler 字段集（幻影检查），
//     且 handler 比注册表多出的字段必须落在 `serverManagedFields` 白名单里（服务端赋值字段，
//     本就不该出现在请求契约中）。
//   - 注册表声明了 requestBody 但 schema **无 properties**（如 `openapi.Obj()` 自由体）：
//     无法比对字段集，跳过（仅记录，不静默——见 untyped 计数自检）。
//
// 断言范围（刻意不做的事）：不做 required / 类型 / 枚举语义比对；不做 query 参数比对
// （分别由 `openapi_contract_test.go` 的 presign 枚举断言与 version DELETE query 断言覆盖）。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// 本文件专用的解析正则。
var (
	// namedVarRe 匹配 `var req someType`（命名类型声明）。
	namedVarRe = regexp.MustCompile(`^var\s+(\w+)\s+([\w.]+)\s*$`)
	// inlineVarRe 匹配 `var req struct {`（内联匿名 struct 声明）。
	inlineVarRe = regexp.MustCompile(`^var\s+(\w+)\s+struct\s*\{`)
	// jsonTagRe 匹配 struct tag 中的 json 名。
	jsonTagRe = regexp.MustCompile("`json:\"([^\",]+)")
)

// serverManagedFields 是「handler 结构体有、但由服务端赋值、因此不应进请求契约」的字段。
// 仅用于共享模型（当前只有 model.Account）；端点专属 DTO 走双向完全相等，无白名单。
var serverManagedFields = map[string]bool{
	"id":        true, // 创建时服务端生成（忽略客户端 id）
	"createdAt": true,
	"updatedAt": true,
}

// decodeSite 描述一次 readJSON 解码：所在方法、目标变量、目标类型。
type decodeSite struct {
	method    string
	target    string
	typeName  string // 命名类型（含包限定，如 model.Account）；空表示内联匿名 struct
	inline    bool
	fieldLine int // 内联 struct 的起始行（用于定位字段）
}

// parseDecodeSites 扫描 handler 源码，返回 方法名 -> 该方法的解码点（含委托链上的）。
//
// 做法：先收集「方法 -> 该方法体内 readJSON 的目标变量声明」，再对每个方法沿 h.xxx() 调用
// 闭包向上找解码点（委托解码）。
func parseDecodeSites(t *testing.T, dir string) map[string]decodeSite {
	t.Helper()
	sources := readHandlerSources(t, dir)
	methods := parseHandlerMethods(sources)

	// 方法 -> 直接解码点（若该方法自己 readJSON）。
	direct := map[string]decodeSite{}
	for name, body := range methods {
		site, ok := findDecodeSite(body)
		if ok {
			site.method = name
			direct[name] = site
		}
	}

	// 沿调用闭包解析委托：返回方法最终触达的解码点。
	memo := map[string]decodeSite{}
	var visit func(name string) (decodeSite, bool)
	visit = func(name string) (decodeSite, bool) {
		if s, ok := memo[name]; ok {
			return s, s.target != ""
		}
		memo[name] = decodeSite{} // 防环
		if s, ok := direct[name]; ok {
			memo[name] = s
			return s, true
		}
		body, ok := methods[name]
		if !ok {
			return decodeSite{}, false
		}
		for _, c := range parseBodyCalls(name, body).calls {
			if s, ok := visit(c); ok {
				memo[name] = s
				return s, true
			}
		}
		return decodeSite{}, false
	}
	for name := range methods {
		visit(name)
	}
	return memo
}

// findDecodeSite 在单个方法体内定位 `readJSON(r, &x)` 并解析 x 的声明。
// x 可能是：方法体内的 `var x T` / `var x struct {`，或函数签名里的命名返回值
// （如 `parseMigrateRequest(...) (req migrateRequest, ...)`）。
//
// readJSON 调用本身走 AST（findReadJSONTarget），避免注释/字符串里的调用被当成解码点。
func findDecodeSite(body string) (decodeSite, bool) {
	target, callLine, ok := findReadJSONTarget(body)
	if !ok {
		return decodeSite{}, false
	}
	lines := strings.Split(body, "\n")
	site := decodeSite{target: target}
	// 向上找声明：`var x T`（命名类型）或 `var x struct {`（内联）。
	for j := callLine - 2; j >= 0; j-- {
		trimmed := strings.TrimSpace(lines[j])
		if strings.HasPrefix(trimmed, "func ") {
			break
		}
		if mm := namedVarRe.FindStringSubmatch(trimmed); mm != nil && mm[1] == target {
			site.typeName = mm[2]
			return site, true
		}
		if mm := inlineVarRe.FindStringSubmatch(trimmed); mm != nil && mm[1] == target {
			site.inline = true
			site.fieldLine = j
			return site, true
		}
	}
	// 命名返回值/参数：在函数签名（首个 `{` 之前）里查找 `x Type`。
	sig := signatureText(lines)
	if mm := regexp.MustCompile(`\b` + regexp.QuoteMeta(target) + `\s+([\w.]+)`).FindStringSubmatch(sig); mm != nil {
		site.typeName = mm[1]
		return site, true
	}
	return site, true
}

// findReadJSONTarget 用 AST 定位方法体内 `h.readJSON(r, &x)` 的真实调用，
// 返回目标变量名与该调用在 body 中的 1-based 行号。注释/字符串字面量不算。
func findReadJSONTarget(body string) (string, int, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "probe.go", "package p\n\n"+body, 0)
	if err != nil {
		return "", 0, false
	}
	target, line := "", 0
	ast.Inspect(f, func(n ast.Node) bool {
		if target != "" {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "readJSON" || len(call.Args) < 2 {
			return true
		}
		if recv, ok := sel.X.(*ast.Ident); !ok || recv.Name != "h" {
			return true // 只认 h.readJSON；其它 receiver 的同名方法不是请求体解码点。
		}
		unary, ok := call.Args[1].(*ast.UnaryExpr)
		if !ok || unary.Op != token.AND {
			return true
		}
		id, ok := unary.X.(*ast.Ident)
		if !ok {
			return true
		}
		target = id.Name
		line = fset.Position(call.Pos()).Line - 2 // body 前插了 `package p` + 空行。
		return false
	})
	if target == "" || line < 1 {
		return "", 0, false
	}
	return target, line, true
}

// signatureText 返回方法签名的文本（从 `func` 行到首个含 `{` 的行之前）。
func signatureText(lines []string) string {
	var out []string
	for _, ln := range lines {
		if strings.Contains(ln, "{") {
			break
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// inlineStructFields 抽取 body 中第 fieldLine 行开始的匿名 struct 的**顶层** json tag。
// 嵌套 struct（如 tags 的元素）只取其外层字段名。
func inlineStructFields(body string, fieldLine int) map[string]bool {
	lines := strings.Split(body, "\n")
	depth := 0
	out := map[string]bool{}
	started := false
	for i := fieldLine; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			switch ch {
			case '{':
				depth++
				started = true
			case '}':
				depth--
			}
		}
		if started && depth == 1 {
			if m := jsonTagRe.FindStringSubmatch(line); m != nil {
				out[m[1]] = true
			}
		}
		if started && depth == 0 {
			break
		}
	}
	return out
}

// namedStructFields 在 dir 下解析命名类型的 struct json tag（顶层）。
// typeName 可带包限定（model.Account）；外部包按 `../<pkg>/<file>.go` 约定查找。
func namedStructFields(t *testing.T, dir, typeName string) map[string]bool {
	t.Helper()
	base := typeName
	if i := strings.LastIndex(typeName, "."); i >= 0 {
		pkg, name := typeName[:i], typeName[i+1:]
		// 本仓库约定：internal/<pkg>/<pkg>.go 或 internal/model/account.go 等；
		// 这里只解析 model 包（当前唯一的跨包解码目标）。
		path := filepath.Join(dir, "..", pkg, "account.go")
		return structJSONFieldsInFile(t, path, name)
	}
	// 同包命名类型：在 dir 下所有 .go 中查找。
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	for _, p := range entries {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		if fields, ok := tryStructJSONFields(t, p, base); ok {
			return fields
		}
	}
	t.Fatalf("未找到命名类型 %s 的 struct 定义（dir=%s）", typeName, dir)
	return nil
}

// structJSONFieldsInFile 抽取指定文件中 structName 的顶层 json tag（找不到即失败）。
func structJSONFieldsInFile(t *testing.T, path, structName string) map[string]bool {
	t.Helper()
	fields, ok := tryStructJSONFields(t, path, structName)
	if !ok {
		t.Fatalf("%s 中未找到结构体 %s", path, structName)
	}
	return fields
}

// tryStructJSONFields 抽取 structName 的顶层 json tag；第二个返回值表示是否找到该类型。
func tryStructJSONFields(t *testing.T, path, structName string) (map[string]bool, bool) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, false
	}
	out := map[string]bool{}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != structName {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		found = true
		for _, field := range st.Fields.List {
			if field.Tag == nil {
				continue
			}
			tag, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				continue
			}
			name := jsonFieldName(tag)
			if name != "" && name != "-" {
				out[name] = true
			}
		}
		return false
	})
	return out, found
}

// TestFindReadJSONTargetIgnoresCommentsAndStrings 是 findReadJSONTarget 的口径测试：
// 只有真实的 `h.readJSON(r, &x)` 调用表达式才算解码；注释与字符串字面量里的
// `readJSON(...)` 不得被当成解码点（旧实现用逐行正则，实测会被两者骗过）。
func TestFindReadJSONTargetIgnoresCommentsAndStrings(t *testing.T) {
	t.Parallel()
	realBody := "func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n" +
		"\tvar req struct {\n\t\tBucket string `json:\"bucket\"`\n\t}\n" +
		"\tif err := h.readJSON(r, &req); err != nil {\n\t\treturn\n\t}\n}\n"
	target, line, ok := findReadJSONTarget(realBody)
	if !ok || target != "req" || line != 5 {
		t.Fatalf("真实 readJSON 识别错误：target=%q line=%d ok=%v（want req / 5 / true）", target, line, ok)
	}
	for _, body := range []string{
		"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\t// h.readJSON(r, &ghost)\n}\n",
		"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\ts := \"h.readJSON(r, &ghost)\"\n\t_ = s\n}\n",
		"func (h *Handler) m(w http.ResponseWriter, r *http.Request) {\n\tvar req struct{}\n\t_ = other.readJSON(r, &req)\n}\n",
	} {
		if got, _, ok := findReadJSONTarget(body); ok {
			t.Errorf("注释/字符串里的 readJSON 被误判为真实解码（body=%q，target=%q）", body, got)
		}
	}
}

// TestOpenAPIRequestFieldsMatchHandlerDTOs 全量遍历：注册表请求体字段集 ⇔ handler 解码字段集。
func TestOpenAPIRequestFieldsMatchHandlerDTOs(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	routes := parseRoutes(t, sources)
	sites := parseDecodeSites(t, dir)
	doc := openAPIDoc(t)

	// 自检：解析口径写坏时不要静默变绿。
	if len(routes) < 60 || len(sites) < 60 {
		t.Fatalf("解析结果异常（routes=%d decodeSites=%d），疑似解析口径失效", len(routes), len(sites))
	}

	checked, untyped, skipped := 0, 0, 0
	for key, method := range routes {
		regProps, hasBody := registryRequestBodyProps(t, doc, key)
		if !hasBody {
			continue // 无请求体：由 openapi_inputsource_test.go 负责「该不该有」。
		}
		site, ok := sites[method]
		if !ok || site.target == "" {
			// 注册表声明了请求体，但 handler 不解码 → 由 inputsource 门禁负责。
			continue
		}
		var handlerFields map[string]bool
		if site.inline {
			body := parseHandlerMethods(sources)[site.method]
			handlerFields = inlineStructFields(body, site.fieldLine)
		} else {
			handlerFields = namedStructFields(t, dir, site.typeName)
		}
		if len(regProps) == 0 {
			untyped++ // 注册表是自由体（openapi.Obj()），字段集无从比对。
			continue
		}
		checked++

		var phantom, missing []string
		for f := range regProps {
			if !handlerFields[f] {
				phantom = append(phantom, f)
			}
		}
		if site.inline || isEndpointOwnedType(site.typeName) {
			// 端点专属 DTO：双向完全相等。
			for f := range handlerFields {
				if !regProps[f] {
					missing = append(missing, f)
				}
			}
		} else {
			// 共享模型：handler 多出的字段必须在服务端赋值白名单内。
			for f := range handlerFields {
				if !regProps[f] && !serverManagedFields[f] {
					missing = append(missing, f)
				}
			}
		}
		sort.Strings(phantom)
		sort.Strings(missing)
		for _, f := range phantom {
			t.Errorf("注册表为 %s 声明了请求字段 %q，但 handler %s 的解码结构体不含它："+
				"readJSON 用 DisallowUnknownFields，按文档发送会被拒绝（400）", key, f, method)
		}
		for _, f := range missing {
			t.Errorf("handler %s（%s）解码字段 %q，但注册表未声明：OpenAPI 客户端不会发送该字段", method, key, f)
		}
	}
	if checked < 20 {
		t.Fatalf("仅比对 %d 个端点（untyped=%d skipped=%d），疑似遍历口径失效", checked, untyped, skipped)
	}
	t.Logf("请求体字段全量比对：%d 个端点（另有 %d 个注册表自由体跳过字段集比对）", checked, untyped)
}

// isEndpointOwnedType 判断命名类型是否为「端点专属请求 DTO」（非共享模型）。
// 共享模型以 `pkg.Type` 形式出现（当前仅 model.Account）；其余同包命名类型均为端点专属。
func isEndpointOwnedType(typeName string) bool {
	return !strings.Contains(typeName, ".")
}

// registryRequestBodyProps 返回注册表某 operation 的请求体字段集；第二个返回值表示是否声明了请求体。
func registryRequestBodyProps(t *testing.T, doc map[string]any, key string) (map[string]bool, bool) {
	t.Helper()
	method, path, found := strings.Cut(key, " ")
	if !found {
		t.Fatalf("路由 key %q 不是 \"METHOD /path\" 形式", key)
	}
	paths := openAPIPaths(t, doc)
	op, ok := paths[path][strings.ToLower(method)]
	if !ok {
		t.Fatalf("openapi 无 %s", key)
	}
	rb, ok := op["requestBody"].(map[string]any)
	if !ok {
		return nil, false
	}
	content, ok := rb["content"].(map[string]any)
	if !ok {
		return map[string]bool{}, true
	}
	mt, ok := content["application/json"].(map[string]any)
	if !ok {
		return map[string]bool{}, true
	}
	schema, ok := mt["schema"].(map[string]any)
	if !ok {
		return map[string]bool{}, true
	}
	if ref, _ := schema["$ref"].(string); ref != "" {
		schema, ok = derefComponent(doc, ref)
		if !ok {
			t.Fatalf("%s $ref 解析失败: %s", key, ref)
		}
	}
	props, _ := schema["properties"].(map[string]any)
	out := map[string]bool{}
	for name := range props {
		out[name] = true
	}
	return out, true
}
