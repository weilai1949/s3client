package handler

// openapi_response_contract_test.go —— **响应**契约门禁：components.schemas ⇔ 真实响应 DTO。
//
// 背景（docs/review-2026-09-19.md §7.2 / §4.3）：此前的门禁只覆盖**请求**方向
// （`openapi_inputsource_test.go` 比对「是否解码请求体」、`openapi_contract_test.go` 断言
// 请求体字段集、`api_doc_test.go` 比对文档字段）。**没有任何门禁把「响应 schema」与
// handler 真实 DTO 做比对**，于是 `components.schemas.Account` 长期声明 `provider` /
// `forcePathStyle` / `insecureSkipVerify` 三个幻影字段、并漏掉真实的 `useSSL`——
// 而它是全部账号端点 200/201 响应的契约。
//
// 本门禁对 `internal/openapi/sharedSchemas()` 里的每个共享 schema，机械抽取对应 Go DTO
// 结构体的 `json` tag，做**双向**比对（schema 多写 = 幻影字段；schema 漏写 = 客户端拿不到
// 已返回的字段）。
//
// 断言范围（刻意不做的事，避免把「有门禁」误读为「响应契约已全量收敛」）：
//   - 只覆盖 components.schemas 中的**共享** schema（Account / Bucket / ObjectItem /
//     ListObjectsResp / Error）。各端点内联的响应体（多数注册为无属性的 `Obj()`）不在内。
//   - 不做 required / 类型 / 枚举语义比对，只比对字段名集合。
//   - handler 用 `map[string]any` 手工拼装的响应（绝大多数端点）无法机械抽取，
//     因此不在本门禁范围内；`Error` 是唯一例外（固定单字段，见下）。
//   新增共享 schema 必须同时在 schemaDTOs 登记，否则本门禁红灯（防止新 schema 静默逃逸）。

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
