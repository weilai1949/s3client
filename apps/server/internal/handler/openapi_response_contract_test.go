package handler

// openapi_response_contract_test.go —— **响应**契约门禁：components.schemas ⇔ 真实响应 DTO，
// 以及**端点级**响应 schema ⇔ handler 真实写出的键。
//
// 背景（docs/review-2026-09-19.md §7.2 / §4.3 / docs/todolist.md #28）：此前的门禁只覆盖**请求**方向
// （`openapi_inputsource_test.go` 比对「是否解码请求体」、`openapi_contract_test.go` 断言
// 请求体字段集、`api_doc_test.go` 比对文档字段）。响应方向原先只覆盖 `components.schemas`
// 的共享 schema，各端点内联 / `map[string]any` 拼装的响应没有机械门禁。
//
// 本文件有两层：
//  1. TestOpenAPI_ResponseSchemasMatchDTOs：共享 schema ⇔ Go DTO 结构体 json tag 双向比对；
//  2. TestOpenAPI_EndpointResponseSchemasMatchHandlers：端点级响应 schema ⇔ handler 真实键集合。
//     机械抽取 handler 方法内 `h.writeJSON(w, <status>, <expr>)` / `h.writePresignResult(...)` 的
//     <expr>：字面量 map 的字符串键、命名 struct 的 json tag（omitempty 记为可选）、
//     变量（含 `var x T`）与同包 helper 函数返回值（`copyBatchJSON` / `migrateBatchJSON` /
//     `migrateResultJSON`）递归解析；`resp["k"] = ...` 的条件键记为可选。
//
// 端点级门禁语义：注册表 properties 必须 ⊆ handler「可能写出的键」（present ∪ optional），
// 且 handler「恒写出的键」（present）必须 ⊆ 注册表 properties。
//
// 断言范围（刻意不做的残留，明确记录而非静默跳过）：
//   - **自由体 `openapi.Obj()`**：注册表未声明**顶层** properties 的端点无从比对，跳过。**2026-09-22 收敛**：
//     全部注册表的顶层自由体响应已升级为具体 properties，端点级门禁从 15 个覆盖到 61 个成功响应；
//     顶层自由体仅剩 `/api/openapi.json` 一个——它的响应体就是 OpenAPI 规范本身，由
//     `Registry.HTTPHandler()` 直接写出而非 `writeJSON`，机械抽取无意义。注意：properties 内部嵌套的
//     `metadata` / `fields` 仍可为 `openapi.Obj()`，本门禁只比对顶层键，不计入 `untyped` 自检。
//     自检要求「带具体 properties 的成功响应」达到下限（60）且顶层自由体不超过 1 个，
//     防止整体退回自由体后静默变绿。
//   - **helper 内部直接 `writeJSON`**：本门禁只扫描 handler 方法体内的 `h.writeJSON` 写出点；
//     helper 返回表达式会被递归解析，但 helper 自身直接 `writeJSON` 的写法不追踪
//     （当前生产代码无此形状，新增时须在本文件登记或改为返回表达式）。
//   - **动态拼装**：handler 从 `[]map` 追加、跨包调用、或键来自运行时字符串的响应无法机械抽取。
//     若这类端点声明了具体 properties，门禁会红灯要求改用字面量 map / 具名 struct——不得静默放过。
//     （`listTrash` / `listObjectVersions` 的数组元素由 `deleteMarkerSchema` / `versionEntrySchema`
//     显式声明，元素键仍由本门禁顶层抽取口径覆盖。）
//   - **$ref 共享响应**（Account / ListObjectsResp / Error）：由第 1 层门禁覆盖，端点级跳过。
//   - 不做响应 required / 类型 / 枚举语义比对（类型语义见 `openapi_semantics_test.go`，且仅请求方向）。
//   - **嵌套对象/数组的深层字段**（如 `store` / `progress` / `result` / `rules[]` 元素）只比对到
//     顶层键；元素形状由注册表共享 schema 构造器（`openapi_register.go`）统一维护。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// responseDTO 描述一个共享 schema 对应的 Go DTO 来源。
type responseDTO struct {
	// file 相对 internal/handler；structName 非空时机械抽取其 json tag。
	file       string
	structName string
	// literalFields 用于没有命名结构体的响应（当前仅 Error：writeErr 的 map 字面量）。
	literalFields []string
}

// schemaDTOs 把 components.schemas 的每个共享 schema 映射到真实 DTO。
// 新增共享 schema 时**必须**在此登记，否则 TestOpenAPI_ResponseSchemasMatchDTOs 红灯。
var schemaDTOs = map[string]responseDTO{
	"Account":         {file: filepath.Join("..", "model", "account.go"), structName: "AccountView"},
	"Bucket":          {file: "buckets.go", structName: "bucketItem"},
	"ObjectItem":      {file: "handler.go", structName: "objectItem"},
	"ListObjectsResp": {file: "handler.go", structName: "listObjectsResponse"},
	// Error 没有命名结构体：writeErr 直接写 map[string]any{"error": msg}。
	"Error": {literalFields: []string{"error"}},
}

// structJSONFields 用 go/parser 抽取 structName 的 json tag 字段名集合。
func structJSONFields(t *testing.T, file, structName string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("解析 %s: %v", file, err)
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
	if !found {
		t.Fatalf("%s 中未找到结构体 %s（DTO 被改名/移走？请同步 schemaDTOs）", file, structName)
	}
	if len(out) == 0 {
		t.Fatalf("%s 的结构体 %s 没有任何 json tag 字段", file, structName)
	}
	return out
}

// jsonFieldName 从 struct tag 中取出 json 名（去掉 omitempty 等选项）。
func jsonFieldName(tag string) string {
	const key = `json:"`
	i := strings.Index(tag, key)
	if i < 0 {
		return ""
	}
	rest := tag[i+len(key):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return strings.Split(rest[:j], ",")[0]
}

// schemaProps 返回 components.schemas[name] 的属性名集合。
func schemaProps(t *testing.T, doc map[string]any, name string) map[string]bool {
	t.Helper()
	comps, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("components 类型 = %T", doc["components"])
	}
	schemas, ok := comps["schemas"].(map[string]any)
	if !ok {
		t.Fatalf("components.schemas 类型 = %T", comps["schemas"])
	}
	raw, ok := schemas[name]
	if !ok {
		t.Fatalf("components.schemas 缺少 %q", name)
	}
	sch, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("components.schemas.%s 类型 = %T", name, raw)
	}
	props, _ := sch["properties"].(map[string]any)
	out := map[string]bool{}
	for k := range props {
		out[k] = true
	}
	return out
}

// TestOpenAPI_ResponseSchemasMatchDTOs 双向校验共享响应 schema ↔ 真实 Go DTO。
//
// 该门禁直接对治 §7.2：schema 声明 `provider`/`forcePathStyle`/`insecureSkipVerify` 而
// `model.AccountView` 无此字段（幻影）、且漏掉真实的 `useSSL`（反向漂移），两处都会红灯。
func TestOpenAPI_ResponseSchemasMatchDTOs(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)

	// 自检 + 防逃逸：components.schemas 的每个 schema 都必须在 schemaDTOs 登记。
	comps := doc["components"].(map[string]any)
	schemas := comps["schemas"].(map[string]any)
	for name := range schemas {
		if _, ok := schemaDTOs[name]; !ok {
			t.Errorf("components.schemas 新增了共享 schema %q，但未在 schemaDTOs 登记真实 DTO；"+
				"请补登记（否则该 schema 的响应契约不受任何门禁保护）", name)
		}
	}
	if len(schemaDTOs) < 5 {
		t.Fatalf("schemaDTOs 只有 %d 条，疑似登记表被误删", len(schemaDTOs))
	}

	checked := 0
	for name, dto := range schemaDTOs {
		var want map[string]bool
		if dto.structName != "" {
			path := filepath.Join(handlerSourceDir(t), dto.file)
			want = structJSONFields(t, path, dto.structName)
		} else {
			want = map[string]bool{}
			for _, f := range dto.literalFields {
				want[f] = true
			}
		}
		got := schemaProps(t, doc, name)
		checked++

		var phantom, missing []string
		for f := range got {
			if !want[f] {
				phantom = append(phantom, f)
			}
		}
		for f := range want {
			if !got[f] {
				missing = append(missing, f)
			}
		}
		sort.Strings(phantom)
		sort.Strings(missing)
		for _, f := range phantom {
			t.Errorf("components.schemas.%s 声明了字段 %q，但真实响应 DTO 不含该字段（幻影字段："+
				"按此 schema 生成的客户端会读到永远为 null 的字段）", name, f)
		}
		for _, f := range missing {
			t.Errorf("components.schemas.%s 漏掉真实响应 DTO 的字段 %q（客户端按 schema 解析会丢失该字段）", name, f)
		}
	}
	if checked != len(schemaDTOs) {
		t.Fatalf("只比对了 %d/%d 个 schema", checked, len(schemaDTOs))
	}
}

// TestOpenAPI_AccountSchemaIsAccountView 是对 §7.2 的**逐字段行为断言**：Account 的
// 响应字段集必须与 model.AccountView 完全一致，且明确不含 secretKey（安全契约）。
func TestOpenAPI_AccountSchemaIsAccountView(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	got := schemaProps(t, doc, "Account")

	for _, f := range []string{
		"id", "name", "endpoint", "publicEndpoint", "region", "accessKey",
		"secretSet", "bucket", "pathStyle", "useSSL", "createdAt", "updatedAt",
	} {
		if !got[f] {
			t.Errorf("Account schema 缺少真实响应字段 %q（model.AccountView 有）", f)
		}
	}
	// secretKey 绝不出现在对外视图（AccountView 用 secretSet 代替）。
	for _, f := range []string{"secretKey", "provider", "forcePathStyle", "insecureSkipVerify"} {
		if got[f] {
			t.Errorf("Account schema 声明了 %q，它既不在 model.AccountView 中，也不应出现在对外响应里", f)
		}
	}
}

// ---- 端点级响应抽取（#28） ----

// httpStatusConsts 把 handler 使用的 http.StatusXxx 常量名映射为状态码。
// 端点级门禁只比较 2xx 成功响应；错误响应用 Error schema，由第 1 层门禁覆盖。
var httpStatusConsts = map[string]int{
	"StatusOK": 200, "StatusCreated": 201, "StatusAccepted": 202,
	"StatusNoContent": 204, "StatusPartialContent": 206,
	"StatusBadRequest": 400, "StatusNotFound": 404, "StatusConflict": 409,
	"StatusRequestEntityTooLarge": 413, "StatusInternalServerError": 500,
	"StatusServiceUnavailable": 503,
}

// respKeys 描述一次响应写出：present 为恒写出的键，optional 为可能写出的键。
type respKeys struct {
	present  map[string]bool
	optional map[string]bool
}

func newRespKeys() respKeys {
	return respKeys{present: map[string]bool{}, optional: map[string]bool{}}
}

// merge 把 other 合并进 r（present 取并集，用于同一写出内多个返回表达式）。
func (r respKeys) merge(other respKeys) {
	for k := range other.present {
		r.present[k] = true
	}
	for k := range other.optional {
		r.optional[k] = true
	}
}

// statusFromExpr 解析 writeJSON 的状态参数：`http.StatusXxx` 或整数字面量。
func (a *handlerAST) statusFromExpr(e ast.Expr) (int, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.INT {
			n, err := strconv.Atoi(v.Value)
			return n, err == nil
		}
	case *ast.SelectorExpr:
		if pkg, ok := v.X.(*ast.Ident); ok && pkg.Name == "http" {
			if n, ok := httpStatusConsts[v.Sel.Name]; ok {
				return n, true
			}
		}
	}
	return 0, false
}

// responseWrites 返回 handler 方法内「成功响应写出点」按状态码分组的结果。
func (a *handlerAST) responseWrites(method string) map[int][]respKeys {
	fd := a.methods[method]
	out := map[int][]respKeys{}
	if fd == nil || fd.Body == nil {
		return out
	}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); !ok || x.Name != "h" {
			return true
		}
		switch sel.Sel.Name {
		case "writeJSON":
			if len(call.Args) != 3 {
				return true
			}
			status, ok := a.statusFromExpr(call.Args[1])
			if !ok {
				return true
			}
			if k, ok := a.resolveExpr(fd, call.Args[2], 0); ok {
				out[status] = append(out[status], k)
			}
		case "writePresignResult":
			// 成功路径固定 200（失败路径走 writeErr）。
			if len(call.Args) != 3 {
				return true
			}
			if k, ok := a.resolveExpr(fd, call.Args[2], 0); ok {
				out[200] = append(out[200], k)
			}
		}
		return true
	})
	return out
}

// resolveExpr 机械解析一个响应表达式为键集合。
// 支持：字面量 map（字符串键）/ 命名 struct 字面量 / 变量（`var x T`、`x := expr`、包级 var）
// / 包级 helper 函数返回值（递归）。无法机械解析时 ok=false。
func (a *handlerAST) resolveExpr(fd *ast.FuncDecl, e ast.Expr, depth int) (respKeys, bool) {
	if fd == nil || depth > 8 {
		return newRespKeys(), false
	}
	switch v := e.(type) {
	case *ast.CompositeLit:
		return a.compositeKeys(v)
	case *ast.Ident:
		if typ, ok := localVarType(fd, v.Name); ok {
			if st, ok := a.types[typ]; ok {
				k := structTypeKeys(st)
				addDynamicKeys(k, dynamicKeys(fd, v.Name))
				return k, true
			}
			return newRespKeys(), false
		}
		if expr, ok := localVarExpr(fd, v.Name); ok {
			k, ok := a.resolveExpr(fd, expr, depth+1)
			if ok {
				addDynamicKeys(k, dynamicKeys(fd, v.Name))
			}
			return k, ok
		}
		if expr, ok := a.vars[v.Name]; ok {
			return a.resolveExpr(fd, expr, depth+1)
		}
		if st, ok := a.types[v.Name]; ok {
			return structTypeKeys(st), true
		}
		return newRespKeys(), false
	case *ast.CallExpr:
		id, ok := v.Fun.(*ast.Ident)
		if !ok {
			return newRespKeys(), false
		}
		callee := a.funcs[id.Name]
		if callee == nil || callee.Body == nil {
			return newRespKeys(), false
		}
		out := newRespKeys()
		found := false
		ast.Inspect(callee.Body, func(n ast.Node) bool {
			rs, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, r := range rs.Results {
				if k, ok := a.resolveExpr(callee, r, depth+1); ok {
					found = true
					out.merge(k)
				}
			}
			return true
		})
		return out, found
	}
	return newRespKeys(), false
}

// compositeKeys 解析复合字面量：map 的字符串键（present）或命名 struct 的字段。
func (a *handlerAST) compositeKeys(cl *ast.CompositeLit) (respKeys, bool) {
	out := newRespKeys()
	switch t := cl.Type.(type) {
	case *ast.MapType:
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				return out, false
			}
			lit, ok := kv.Key.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return out, false // 非字面量键：动态拼装，无法机械抽取。
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return out, false
			}
			out.present[s] = true
		}
		return out, true
	case *ast.Ident:
		st, ok := a.types[t.Name]
		if !ok {
			return out, false
		}
		return structTypeKeys(st), true
	}
	return out, false
}

// structTypeKeys 返回 struct 类型的 json 键：omitempty 记为 optional，其余 present。
func structTypeKeys(st *ast.StructType) respKeys {
	out := newRespKeys()
	for _, f := range st.Fields.List {
		if f.Tag == nil {
			continue
		}
		tag, err := strconv.Unquote(f.Tag.Value)
		if err != nil {
			continue
		}
		name, opt := jsonFieldNameOpt(tag)
		if name == "" || name == "-" {
			continue
		}
		if opt {
			out.optional[name] = true
		} else {
			out.present[name] = true
		}
	}
	return out
}

// jsonFieldNameOpt 从 struct tag 取 json 名，并返回是否带 omitempty。
func jsonFieldNameOpt(tag string) (string, bool) {
	const key = `json:"`
	i := strings.Index(tag, key)
	if i < 0 {
		return "", false
	}
	rest := tag[i+len(key):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return "", false
	}
	parts := strings.Split(rest[:j], ",")
	name := parts[0]
	opt := false
	for _, p := range parts[1:] {
		if p == "omitempty" {
			opt = true
		}
	}
	return name, opt
}

// addDynamicKeys 把 `x["lit"] = ...` 形式的运行时键加入 optional。
func addDynamicKeys(k respKeys, keys []string) {
	for _, key := range keys {
		k.optional[key] = true
	}
}

// dynamicKeys 返回方法/函数体内对变量 name 的 `name["lit"] = ...` 键。
func dynamicKeys(fd *ast.FuncDecl, name string) []string {
	var out []string
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, l := range as.Lhs {
			idx, ok := l.(*ast.IndexExpr)
			if !ok {
				continue
			}
			id, ok := idx.X.(*ast.Ident)
			if !ok || id.Name != name {
				continue
			}
			if lit, ok := idx.Index.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil {
					out = append(out, s)
				}
			}
		}
		return true
	})
	return out
}

// localVarType 返回函数体内 `var x T`（无初值）的命名类型 T。
func localVarType(fd *ast.FuncDecl, name string) (string, bool) {
	var typ string
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if typ != "" {
			return false
		}
		vs, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, id := range vs.Names {
			if id.Name != name || i < len(vs.Values) {
				continue
			}
			if idt, ok := vs.Type.(*ast.Ident); ok {
				typ = idt.Name
				return false
			}
		}
		return true
	})
	return typ, typ != ""
}

// localVarExpr 返回函数体内对 name 的赋值表达式（`x := expr` / `x = expr` / `var x = expr`）。
func localVarExpr(fd *ast.FuncDecl, name string) (ast.Expr, bool) {
	var found ast.Expr
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		switch s := n.(type) {
		case *ast.AssignStmt:
			if s.Tok != token.DEFINE && s.Tok != token.ASSIGN {
				return true
			}
			for i, l := range s.Lhs {
				if id, ok := l.(*ast.Ident); ok && id.Name == name && i < len(s.Rhs) {
					found = s.Rhs[i]
					return false
				}
			}
		case *ast.ValueSpec:
			for i, id := range s.Names {
				if id.Name == name && i < len(s.Values) {
					found = s.Values[i]
					return false
				}
			}
		}
		return true
	})
	return found, found != nil
}

// combineSites 把同一状态码的多个写出点合并：present = 各写出点 present 的交集；
// optional = 各写出点 present∪optional 的并集减去 present。
func combineSites(sites []respKeys) (present, optional map[string]bool) {
	present = map[string]bool{}
	possible := map[string]bool{}
	for i, s := range sites {
		for k := range s.present {
			possible[k] = true
		}
		for k := range s.optional {
			possible[k] = true
		}
		if i == 0 {
			for k := range s.present {
				present[k] = true
			}
			continue
		}
		for k := range present {
			if !s.present[k] {
				delete(present, k)
			}
		}
	}
	optional = possible
	for k := range present {
		delete(optional, k)
	}
	return present, optional
}

// TestOpenAPI_EndpointResponseSchemasMatchHandlers 端点级响应契约：
// 注册表声明的响应 properties ⇔ handler 真实写出的键集合。
//
//   - 注册表 properties ⊆ handler「可能写出的键」（present ∪ optional）——多写即幻影字段；
//   - handler「恒写出的键」（present）⊆ 注册表 properties——漏写则客户端解析丢失字段。
//
// $ref 共享响应与自由体（无 properties）跳过；带具体 properties 的成功响应若无法从 handler
// 机械抽取（动态拼装）则红灯，必须显式登记而不是静默放过。
func TestOpenAPI_EndpointResponseSchemasMatchHandlers(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	routes := parseRoutes(t, sources)
	a := parseHandlerAST(t, dir)
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	checked, untyped := 0, 0
	for key, method := range routes {
		httpMethod, path, found := strings.Cut(key, " ")
		if !found {
			t.Fatalf("路由 key %q 不是 \"METHOD /path\" 形式", key)
		}
		op, ok := paths[path][strings.ToLower(httpMethod)]
		if !ok {
			t.Fatalf("openapi 无 %s", key)
		}
		resps, _ := op["responses"].(map[string]any)
		writes := a.responseWrites(method)
		for status, raw := range resps {
			code, err := strconv.Atoi(status)
			if err != nil || code < 200 || code >= 300 {
				continue
			}
			rm, ok := resolveRawResponse(doc, raw)
			if !ok {
				continue
			}
			content, ok := rm["content"].(map[string]any)
			if !ok {
				continue
			}
			mt, ok := content["application/json"].(map[string]any)
			if !ok {
				continue
			}
			sch, _ := mt["schema"].(map[string]any)
			if ref, _ := sch["$ref"].(string); ref != "" {
				continue // 共享 schema 由 TestOpenAPI_ResponseSchemasMatchDTOs 覆盖。
			}
			props, _ := sch["properties"].(map[string]any)
			if len(props) == 0 {
				untyped++ // 自由体（openapi.Obj()）：字段集无从比对（残留范围，见文件头）。
				continue
			}
			sites := writes[code]
			if len(sites) == 0 {
				t.Errorf("%s 的 %s 响应声明了具体 properties，但无法从 handler %s 机械抽取响应键"+
					"（动态拼装？请改用具名 struct / 字面量 map，或在文件头登记残留范围）", key, status, method)
				continue
			}
			checked++
			present, optional := combineSites(sites)

			var phantom, missing []string
			for f := range props {
				if !present[f] && !optional[f] {
					phantom = append(phantom, f)
				}
			}
			for f := range present {
				if _, ok := props[f]; !ok {
					missing = append(missing, f)
				}
			}
			sort.Strings(phantom)
			sort.Strings(missing)
			for _, f := range phantom {
				t.Errorf("%s 的 %s 响应 schema 声明了字段 %q，但 handler %s 从不写出它（幻影字段："+
					"客户端会读到永远为 null 的字段）", key, status, f, method)
			}
			for _, f := range missing {
				t.Errorf("%s 的 %s 响应 schema 漏掉 handler %s 恒写出的字段 %q"+
					"（客户端按 schema 解析会丢失该字段）", key, status, method, f)
			}
		}
	}
	if checked < 60 {
		t.Fatalf("仅比对 %d 个端点级响应（untyped=%d），疑似注册表回退成自由体或抽取口径失效",
			checked, untyped)
	}
	if untyped > 1 {
		t.Fatalf("有 %d 个 2xx 响应仍是自由体（openapi.Obj()），超过已登记的 1 个残留"+
			"（仅 /api/openapi.json：响应体即 OpenAPI 规范本身，由 HTTPHandler 直接写，非 writeJSON）；"+
			"新增自由体须先升级为具体 properties 或在此登记原因", untyped)
	}
	t.Logf("端点级响应全量比对：%d 个带具体 properties 的成功响应（另有 %d 个自由体跳过）", checked, untyped)
}
