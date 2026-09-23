package handler

// openapi_semantics_test.go —— **类型语义**门禁：注册表的 enum / required 与 handler 真实行为对齐。
//
// 背景（docs/todolist.md #27 / docs/archive/review-2026-09-19.md §7.4）：此前的门禁只比对字段**名**集合
// （`openapi_request_fields_test.go` / `api_doc_test.go`），类型语义（enum / required）没有机械门禁；
// 只有 presign 的 method 枚举有逐条断言。结果是 objects ACL 的注册表 enum 少了两个 handler 真实
// 接受的取值；presign `method` / copy-object `newBucket` / copy-prefix `targetBucket` 被误标
// required（handler 对空值有默认）——都属「注册表与 handler 行为不一致」。
//
// 本文件用 go/parser 机械抽取 handler 源码：
//   - **enum**：注册表每个带 enum 的请求体字段 / query 参数，都必须登记一条 enumContract；
//     contract 从 handler 的 switch case 字符串或 `map[string]bool` 字面量键抽取真实接受值
//     （`service.CompareX` 这类跨包常量由 ../service 的 const 声明解析）。注册表 enum 与
//     handler 接受值（忽略空串）必须**双向相等**。未登记的 enum 站点直接红灯（防新 EnumStr 逃逸）。
//     当前 7 个 EnumStr 站点（presign method、proxy mode、createBucket acl、bucket-versioning
//     status、SSE algorithm、object acl、migrate/sync mode）**全部可机械抽取**，因此没有
//     「显式声明已知契约」的兜底分支；若将来出现不可抽取的 enum（值来自运行时/外部包函数），
//     应在 enumContract 增加 declared 字段并写清不可抽取原因，不得静默跳过。
//   - **required**：注册表 required 字段必须是 handler 实际解码的字段；且不得是 handler
//     「空值有默认」的字段（那会让「必填」变成契约谎言）。
//
// 断言范围（刻意不做 / 无法机械化的残留，明确记录而非静默跳过）：
//   - enum 只比较**非空**取值：`""` 是否被接受由 `required` 表达（字段可省略 = 空值走默认），
//     因此不纳入 enum 集合比较；createBucket 的 acl 注册表额外列出 `""` 属冗余但不矛盾。
//   - required 的「空值默认」只识别字面量赋值模式
//     `x := req.F; if x == "" { x = <expr> }` 与 `if req.F == "" { req.F = <expr> }`。
//     **不识别** `h.bucketOr(w, acc, req.Bucket)` 这类「调用内部兜底为账号默认桶」的模式——
//     所有请求体的 `bucket` 字段因此不在本门禁的 required 校验范围内（它是全仓库统一的
//     「缺省用账号默认桶」语义，由共享 Bucket query 参数描述「可省略」承载）。
//   - required 只覆盖 requestBody；query 参数的 required 由 `openapi_query_params_test.go`
//     的「声明 ⇔ 读取」双向门禁覆盖（读取即被接受，未读取会红灯）。

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// handlerAST 是 handler 包的机械视图：方法体、包级变量字面量、包级函数与命名 struct，
// 以及跨包（service）常量。
type handlerAST struct {
	dir       string
	fset      *token.FileSet
	methods   map[string]*ast.FuncDecl
	funcs     map[string]*ast.FuncDecl   // 包级函数（无 receiver），如 copyBatchJSON
	types     map[string]*ast.StructType // 命名 struct 类型
	vars      map[string]ast.Expr        // 包级 var 名 -> 值表达式
	svcConsts map[string]string          // service 包常量 ident -> 字符串值
}

// parseHandlerAST 解析 handler 目录下所有非测试 Go 源码，建立方法 / 函数 / 类型 / 包级变量 / service 常量索引。
func parseHandlerAST(t *testing.T, dir string) *handlerAST {
	t.Helper()
	a := &handlerAST{
		dir:       dir,
		fset:      token.NewFileSet(),
		methods:   map[string]*ast.FuncDecl{},
		funcs:     map[string]*ast.FuncDecl{},
		types:     map[string]*ast.StructType{},
		vars:      map[string]ast.Expr{},
		svcConsts: serviceConstStrings(t, filepath.Join(dir, "..", "service")),
	}
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	for _, p := range entries {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(a.fset, p, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", p, err)
		}
		for _, d := range f.Decls {
			switch decl := d.(type) {
			case *ast.FuncDecl:
				switch {
				case decl.Recv != nil && decl.Name != nil:
					a.methods[decl.Name.Name] = decl
				case decl.Name != nil:
					a.funcs[decl.Name.Name] = decl
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if st, ok := s.Type.(*ast.StructType); ok {
							a.types[s.Name.Name] = st
						}
					case *ast.ValueSpec:
						for i, id := range s.Names {
							if i < len(s.Values) {
								a.vars[id.Name] = s.Values[i]
							}
						}
					}
				}
			}
		}
	}
	if len(a.methods) < 60 {
		t.Fatalf("仅解析到 %d 个 handler 方法，疑似解析口径失效", len(a.methods))
	}
	return a
}

// serviceConstStrings 解析 service 包的非测试源码，返回「常量 ident -> 字符串值」。
// 用于解析 switch case 里的 `service.CompareETag` 这类跨包常量。
func serviceConstStrings(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	for _, p := range entries {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", p, err)
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, id := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					lit, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					if s, err := strconv.Unquote(lit.Value); err == nil {
						out[id.Name] = s
					}
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s 未解析到任何字符串常量", dir)
	}
	return out
}

// exprText 用 go/printer 还原 AST 节点文本（用于匹配 switch 的被判别表达式 / 方法体调用）。
func (a *handlerAST) exprText(node any) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, a.fset, node); err != nil {
		return ""
	}
	return buf.String()
}

// constString 把 case 表达式解析为字符串值：字符串字面量，或 service 包常量标识符。
func (a *handlerAST) constString(e ast.Expr) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(v.Value)
		return s, err == nil
	case *ast.SelectorExpr:
		if pkg, ok := v.X.(*ast.Ident); ok && pkg.Name == "service" {
			if s, ok := a.svcConsts[v.Sel.Name]; ok {
				return s, true
			}
		}
	}
	return "", false
}

// switchCaseStrings 返回 method 内被判别表达式含 scrutinee 的 switch 的全部 case 字符串值。
func (a *handlerAST) switchCaseStrings(method, scrutinee string) []string {
	fd := a.methods[method]
	if fd == nil || fd.Body == nil {
		return nil
	}
	var out []string
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok || sw.Tag == nil {
			return true
		}
		if !strings.Contains(a.exprText(sw.Tag), scrutinee) {
			return true
		}
		for _, stmt := range sw.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}
			for _, e := range cc.List {
				if s, ok := a.constString(e); ok {
					out = append(out, s)
				}
			}
		}
		return false
	})
	return out
}

// mapKeys 返回包级 `map[string]bool{...}` 变量 varName 的字面量键。
func (a *handlerAST) mapKeys(varName string) []string {
	cl, ok := a.vars[varName].(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var out []string
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if lit, ok := kv.Key.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil {
				out = append(out, s)
			}
		}
	}
	return out
}

// ---- enum 门禁 ----

// enumContract 描述注册表一个 enum 站点在 handler 侧的机械抽取方式：
// switch（method + scrutinee）或 map（mapVar 字面量键）。二者当前覆盖全部 7 个 EnumStr 站点。
type enumContract struct {
	method    string
	scrutinee string
	mapVar    string
}

// extract 返回 handler 真实接受的 enum 取值（含空串；比较时忽略空串）。
func (c enumContract) extract(a *handlerAST) []string {
	if c.mapVar != "" {
		return a.mapKeys(c.mapVar)
	}
	return a.switchCaseStrings(c.method, c.scrutinee)
}

// enumContracts 把注册表每个 enum 站点映射到 handler 的机械抽取契约。
// 新增 `openapi.EnumStr` 时必须在此登记，否则 TestOpenAPISemanticsEnumsMatchHandlers 红灯。
var enumContracts = map[string]enumContract{
	// objects.go presign：switch strings.ToLower(req.Method) { get / post / put }。
	"POST /api/accounts/{id}/presign method": {method: "presign", scrutinee: "req.Method"},
	// proxy.go proxyObject：switch mode { text / inline,download }。
	"GET /api/accounts/{id}/proxy mode": {method: "proxyObject", scrutinee: "mode"},
	// buckets.go createBucket：switch req.ACL { "",private,public-read,public-read-write }。
	"POST /api/accounts/{id}/bucket acl": {method: "createBucket", scrutinee: "req.ACL"},
	// buckets.go putBucketVersioning：switch req.Status { Enabled,Suspended }。
	"PUT /api/accounts/{id}/bucket-versioning status": {method: "putBucketVersioning", scrutinee: "req.Status"},
	// bucket_settings.go putBucketEncryption：map allowedSSEAlgorithms。
	"PUT /api/accounts/{id}/bucket/encryption algorithm": {mapVar: "allowedSSEAlgorithms"},
	// metadata.go putObjectAcl：switch req.ACL { private,public-read,public-read-write,
	// authenticated-read,aws-exec-read }（注册表此前漏了后两个，已同步补齐）。
	"PUT /api/accounts/{id}/object-acl acl": {method: "putObjectAcl", scrutinee: "req.ACL"},
	// migrate_sync.go syncHandler：switch mode { "",service.CompareETag/SizeTime/Always }。
	"POST /api/migrate/sync mode": {method: "syncHandler", scrutinee: "mode"},
}

// registryEnumSites 返回 "METHOD /path field" -> 注册表声明的 enum 取值（字符串）。
func registryEnumSites(t *testing.T, doc map[string]any) map[string][]string {
	t.Helper()
	paths := openAPIPaths(t, doc)
	out := map[string][]string{}
	for path, ops := range paths {
		for method, op := range ops {
			key := strings.ToUpper(method) + " " + path
			if rb, ok := op["requestBody"].(map[string]any); ok {
				if schema := requestBodySchema(t, doc, rb); schema != nil {
					if props, ok := schema["properties"].(map[string]any); ok {
						for name, raw := range props {
							if vals, ok := schemaEnum(raw); ok {
								out[key+" "+name] = vals
							}
						}
					}
				}
			}
			if params, ok := op["parameters"].([]any); ok {
				for _, raw := range params {
					pm, ok := resolveRawParam(doc, raw)
					if !ok {
						continue
					}
					if in, _ := pm["in"].(string); in != "query" {
						continue
					}
					name, _ := pm["name"].(string)
					if name == "" {
						continue
					}
					if vals, ok := schemaEnum(pm["schema"]); ok {
						out[key+" "+name] = vals
					}
				}
			}
		}
	}
	return out
}

// requestBodySchema 返回 requestBody 的 application/json schema（$ref 已解析）；无则 nil。
func requestBodySchema(t *testing.T, doc map[string]any, rb map[string]any) map[string]any {
	t.Helper()
	content, ok := rb["content"].(map[string]any)
	if !ok {
		return nil
	}
	mt, ok := content["application/json"].(map[string]any)
	if !ok {
		return nil
	}
	schema, ok := mt["schema"].(map[string]any)
	if !ok {
		return nil
	}
	if ref, _ := schema["$ref"].(string); ref != "" {
		resolved, ok := derefComponent(doc, ref)
		if !ok {
			t.Fatalf("$ref 解析失败: %s", ref)
		}
		return resolved
	}
	return schema
}

// schemaEnum 从 schema 中取出字符串 enum 值。
func schemaEnum(raw any) ([]string, bool) {
	sch, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	enum, ok := sch["enum"].([]any)
	if !ok || len(enum) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(enum))
	for _, v := range enum {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out, len(out) > 0
}

// nonEmpty 去掉空串并排序。
func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// TestOpenAPISemanticsEnumsMatchHandlers 双向校验注册表 enum ⇔ handler 真实接受值（忽略空串）。
func TestOpenAPISemanticsEnumsMatchHandlers(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	sites := registryEnumSites(t, doc)
	if len(sites) < 7 {
		t.Fatalf("注册表仅解析到 %d 个 enum 站点，疑似解析口径失效", len(sites))
	}
	a := parseHandlerAST(t, handlerSourceDir(t))

	checked := 0
	for key, regVals := range sites {
		c, ok := enumContracts[key]
		if !ok {
			t.Errorf("注册表 enum 站点 %q 未在 enumContracts 登记 handler 抽取契约："+
				"新增 EnumStr 必须同步登记（否则该 enum 不受任何门禁保护）", key)
			continue
		}
		checked++
		got := nonEmpty(c.extract(a))
		want := nonEmpty(regVals)
		if len(got) == 0 {
			t.Errorf("enumContracts[%q] 未能从 handler 抽取任何取值（抽取口径失效？）", key)
			continue
		}
		if strings.Join(got, ",") == strings.Join(want, ",") {
			continue
		}
		t.Errorf("注册表 enum 与 handler 真实接受值不一致：%s\n  注册表 = %v\n  handler = %v", key, want, got)
	}
	for key := range enumContracts {
		if _, ok := sites[key]; !ok {
			t.Errorf("enumContracts 条目 %q 在注册表中没有对应 enum（陈旧登记，请删除）", key)
		}
	}
	if checked != len(sites) {
		t.Fatalf("只校验了 %d/%d 个 enum 站点", checked, len(sites))
	}
}

// ---- required 门禁 ----

// handlerDecodedFields 返回 handler 方法（含委托闭包）解码出的请求字段集。
func handlerDecodedFields(t *testing.T, dir string, sources map[string]string, sites map[string]decodeSite, method string) (map[string]bool, bool) {
	t.Helper()
	site, ok := sites[method]
	if !ok || site.target == "" {
		return nil, false
	}
	if site.inline {
		body := parseHandlerMethods(sources)[site.method]
		return inlineStructFields(body, site.fieldLine), true
	}
	return namedStructFields(t, dir, site.typeName), true
}

// handlerFieldNames 返回 handler 方法解码结构体的 Go 字段名 -> JSON 名映射。
// 用于把 defaultedFields 抽出的 Go 字段名（如 NewBucket）换算成注册表使用的 JSON 名（newBucket）。
func handlerFieldNames(t *testing.T, dir string, sources map[string]string, sites map[string]decodeSite, method string) map[string]string {
	t.Helper()
	site, ok := sites[method]
	if !ok || site.target == "" {
		return nil
	}
	if site.inline {
		body := parseHandlerMethods(sources)[site.method]
		return inlineStructFieldNames(body, site.fieldLine)
	}
	base := site.typeName
	if i := strings.LastIndex(base, "."); i >= 0 {
		pkg, name := base[:i], base[i+1:]
		return structFieldNamesInFile(t, filepath.Join(dir, "..", pkg, "account.go"), name)
	}
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	for _, p := range entries {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		if m, ok := tryStructFieldNames(t, p, base); ok {
			return m
		}
	}
	t.Fatalf("未找到命名类型 %s 的 struct 定义（dir=%s）", site.typeName, dir)
	return nil
}

// inlineStructFieldNames 抽取内联匿名 struct 顶层字段的 Go 名 -> JSON 名映射。
func inlineStructFieldNames(body string, fieldLine int) map[string]string {
	lines := strings.Split(body, "\n")
	depth, started := 0, false
	out := map[string]string{}
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
			if m := structFieldRe.FindStringSubmatch(line); m != nil {
				out[m[1]] = m[2]
			}
		}
		if started && depth == 0 {
			break
		}
	}
	return out
}

// structFieldRe 匹配 `Name Type `json:"name..."“ 的字段行。
var structFieldRe = regexp.MustCompile("^\\s*(\\w+)\\s+[^`]+`json:\"([^\",]+)")

// structFieldNamesInFile 抽取指定文件中 structName 的 Go 名 -> JSON 名映射（找不到即失败）。
func structFieldNamesInFile(t *testing.T, path, structName string) map[string]string {
	t.Helper()
	m, ok := tryStructFieldNames(t, path, structName)
	if !ok {
		t.Fatalf("%s 中未找到结构体 %s", path, structName)
	}
	return m
}

// tryStructFieldNames 抽取 structName 的 Go 名 -> JSON 名映射；第二个返回值表示是否找到该类型。
func tryStructFieldNames(t *testing.T, path, structName string) (map[string]string, bool) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, false
	}
	out := map[string]string{}
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
			if field.Tag == nil || len(field.Names) == 0 {
				continue
			}
			tag, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				continue
			}
			name := jsonFieldName(tag)
			if name != "" && name != "-" {
				out[field.Names[0].Name] = name
			}
		}
		return false
	})
	return out, found
}

// registryRequestBodyRequired 返回某 operation requestBody 的 required 字段集；
// 第二个返回值表示该 operation 是否声明了带 properties 的请求体。
func registryRequestBodyRequired(t *testing.T, doc map[string]any, key string) ([]string, bool) {
	t.Helper()
	method, path, found := strings.Cut(key, " ")
	if !found {
		t.Fatalf("路由 key %q 不是 \"METHOD /path\" 形式", key)
	}
	op, ok := openAPIPaths(t, doc)[path][strings.ToLower(method)]
	if !ok {
		t.Fatalf("openapi 无 %s", key)
	}
	rb, ok := op["requestBody"].(map[string]any)
	if !ok {
		return nil, false
	}
	schema := requestBodySchema(t, doc, rb)
	if schema == nil {
		return nil, true
	}
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return nil, true
	}
	var out []string
	if reqs, ok := schema["required"].([]any); ok {
		for _, r := range reqs {
			if s, ok := r.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out, true
}

// defaultedFields 返回 handler 方法（含 h.xxx() 调用闭包上的委托方法）内「空值会被赋予默认值」
// 的请求字段集。识别两种模式：`x := req.F; if x == "" { x = <expr> }` 与
// `if req.F == "" { req.F = <expr> }`。沿调用闭包是为了覆盖 parseCopyPrefix 这类委托解码。
func (a *handlerAST) defaultedFields(method string) map[string]bool {
	out := map[string]bool{}
	seen := map[string]bool{}
	var visit func(name string)
	visit = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		fd := a.methods[name]
		if fd == nil || fd.Body == nil {
			return
		}
		collectDefaults(fd.Body, out)
		for _, c := range handlerMethodCalls(fd) {
			visit(c)
		}
	}
	visit(method)
	return out
}

// collectDefaults 把 body 内「空值赋默认」的字段加入 out。
// 别名检测同时接受 `x := req.F` 与 `x = req.F`（后者覆盖命名返回值，如 parseCopyPrefix 的 targetBucket）。
func collectDefaults(body *ast.BlockStmt, out map[string]bool) {
	alias := map[string]string{}
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || (as.Tok != token.DEFINE && as.Tok != token.ASSIGN) || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		if sel, ok := as.Rhs[0].(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "req" {
				alias[id.Name] = sel.Sel.Name
			}
		}
		return true
	})
	ast.Inspect(body, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		field, varName, ok := emptyCompare(ifs.Cond)
		if !ok {
			return true
		}
		if field == "" {
			field = alias[varName]
		}
		if field == "" {
			return true
		}
		if bodyAssignsNonEmpty(ifs.Body, varName, field) {
			out[field] = true
		}
		return true
	})
}

// emptyCompare 识别 `x == ""` / `req.F == ""`，返回 (字段名, 局部变量名, 是否命中)。
func emptyCompare(cond ast.Expr) (field, varName string, ok bool) {
	be, ok := cond.(*ast.BinaryExpr)
	if !ok || be.Op != token.EQL {
		return "", "", false
	}
	var other ast.Expr
	switch {
	case isEmptyLit(be.X):
		other = be.Y
	case isEmptyLit(be.Y):
		other = be.X
	default:
		return "", "", false
	}
	switch v := other.(type) {
	case *ast.Ident:
		return "", v.Name, true
	case *ast.SelectorExpr:
		if x, ok := v.X.(*ast.Ident); ok && x.Name == "req" {
			return v.Sel.Name, "", true
		}
	}
	return "", "", false
}

// bodyAssignsNonEmpty 判断 body 是否给 varName（或 req.field）赋了非空字面量/表达式。
func bodyAssignsNonEmpty(body *ast.BlockStmt, varName, field string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, l := range as.Lhs {
			if i >= len(as.Rhs) || isEmptyLit(as.Rhs[i]) {
				continue
			}
			switch v := l.(type) {
			case *ast.Ident:
				if v.Name == varName {
					found = true
				}
			case *ast.SelectorExpr:
				if x, ok := v.X.(*ast.Ident); ok && x.Name == "req" && v.Sel.Name == field {
					found = true
				}
			}
		}
		return true
	})
	return found
}

// isEmptyLit 判断表达式是否为空字符串字面量。
func isEmptyLit(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && lit.Value == `""`
}

// TestOpenAPISemanticsRequiredFieldsAreDecoded 校验注册表 required 字段：
//  1. 必须是 handler 实际解码的字段（否则「必填」字段被 handler 忽略）；
//  2. 不得是 handler 空值有默认的字段（否则「必填」与 handler 行为矛盾）。
func TestOpenAPISemanticsRequiredFieldsAreDecoded(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	routes := parseRoutes(t, sources)
	sites := parseDecodeSites(t, dir)
	a := parseHandlerAST(t, dir)
	doc := openAPIDoc(t)

	checked := 0
	for key, method := range routes {
		required, hasBody := registryRequestBodyRequired(t, doc, key)
		if !hasBody || len(required) == 0 {
			continue
		}
		fields, ok := handlerDecodedFields(t, dir, sources, sites, method)
		if !ok {
			t.Errorf("注册表为 %s 声明了 required 字段，但 handler %s 不解码请求体", key, method)
			continue
		}
		checked++
		defaults := a.defaultedFields(method)
		goNames := handlerFieldNames(t, dir, sources, sites, method)
		for _, f := range required {
			if !fields[f] {
				t.Errorf("注册表把 %s 的字段 %q 标为 required，但 handler %s 的解码结构体不含它"+
					"（契约谎言：必填字段被忽略）", key, f, method)
			}
			for goName, jsonName := range goNames {
				if jsonName == f && defaults[goName] {
					t.Errorf("注册表把 %s 的字段 %q 标为 required，但 handler %s 对空值有默认"+
						"（空值等价于省略，应改为可选）", key, f, method)
				}
			}
		}
	}
	if checked < 15 {
		t.Fatalf("仅校验了 %d 个带 required 的请求体，疑似遍历口径失效", checked)
	}
	t.Logf("required 语义全量比对：%d 个请求体", checked)
}
