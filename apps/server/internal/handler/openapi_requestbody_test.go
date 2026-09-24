package handler

// openapi_requestbody_test.go —— operation requestBody 与真实 handler DTO 的字段 / required 对齐契约
//（原 openapi_contract_test.go 分节 7）：读取 routes.go 的 AST 与 APIRequestBody 声明，逐 operation
// 比对 JSON tag、required 集与方法。
//
// 夹具与分节 1–6 见 openapi_contract_test.go / openapi_shape_test.go。

import (
	"testing"
)

// ---- 7. requestBody 与真实 handler DTO 契约对齐 ----

// requestBodyProps 从 operation 的 requestBody.content["application/json"].schema 提取
// 属性名集合（内联 object 或 $ref 到 components.schemas 的 object 均支持）。
func requestBodyProps(t *testing.T, doc map[string]any, path, method string) (props map[string]bool) {
	t.Helper()
	paths := openAPIPaths(t, doc)
	op, ok := paths[path][method]
	if !ok {
		t.Fatalf("openapi 无 %s %s", method, path)
	}
	rb, ok := op["requestBody"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s 无 requestBody", method, path)
	}
	content, ok := rb["content"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s requestBody 无 content", method, path)
	}
	mt, ok := content["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s requestBody 无 application/json", method, path)
	}
	schema, ok := mt["schema"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s requestBody schema 类型 = %T", method, path, mt["schema"])
	}
	if ref, _ := schema["$ref"].(string); ref != "" {
		schema, ok = derefComponent(doc, ref)
		if !ok {
			t.Fatalf("%s %s $ref 解析失败: %s", method, path, ref)
		}
	}
	probs, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s schema 无 properties", method, path)
	}
	props = make(map[string]bool, len(probs))
	for name := range probs {
		props[name] = true
	}
	return props
}

// requestQueryParams 返回某操作在 query 上的参数名集合（$ref 解析到 components.parameters）。
func requestQueryParams(t *testing.T, doc map[string]any, path, method string) map[string]bool {
	t.Helper()
	paths := openAPIPaths(t, doc)
	op, ok := paths[path][method]
	if !ok {
		t.Fatalf("openapi 无 %s %s", method, path)
	}
	out := map[string]bool{}
	list, _ := op["parameters"].([]any)
	for _, raw := range list {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if ref, _ := p["$ref"].(string); ref != "" {
			p, ok = derefComponent(doc, ref)
			if !ok {
				t.Fatalf("%s %s $ref 解析失败: %s", method, path, ref)
			}
		}
		if in, _ := p["in"].(string); in != "query" {
			continue
		}
		if name, _ := p["name"].(string); name != "" {
			out[name] = true
		}
	}
	return out
}

// TestOpenAPI_ContractRequestBodyMatchesHandlers 验证 requestBody schema 与真实 handler DTO 对齐：
// 这是「注册表 ↔ 实际解析字段」三方一致性的最后一道防线（补 routes↔spec 双向检查覆盖不到的
// 字段级漂移）。已知失真点（2026-09-16 评估 H1）逐一断言：
//   - /api/migrate、/api/migrate/async：schema 曾写 srcAccountId/srcBucket/...dstAccountId 等，
//     真实 handler migrateRequest 使用 sourceAccountId/sourceBucket/sourceKeys/targetAccountId/
//     targetBucket/targetPrefix，且无 deleteSource/storageClass 字段；
//   - /api/accounts/{id}/presign：schema 曾声明 method 枚举 GET/PUT/DELETE/HEAD，真实 handler
//     仅支持 get/put/post（strings.ToLower 匹配）；
//   - /api/accounts/{id}/delete：schema 曾声明 versionId，真实 handler 无此字段；
//   - /api/accounts/{id}/multipart/part：真实 handler 额外支持 expiresIn。
func TestOpenAPI_ContractRequestBodyMatchesHandlers(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)

	// 1. migrate / migrate/async 请求体字段必须与 handler.migrateRequest 完全一致。
	wantMigrate := map[string]bool{
		"sourceAccountId": true, "sourceBucket": true, "sourceKeys": true,
		"targetAccountId": true, "targetBucket": true, "targetPrefix": true,
	}
	for _, path := range []string{"/api/migrate", "/api/migrate/async"} {
		got := requestBodyProps(t, doc, path, "post")
		for name := range wantMigrate {
			if !got[name] {
				t.Errorf("%s POST schema 缺少字段 %q（真实 handler migrateRequest 必需）", path, name)
			}
		}
		for name := range got {
			if !wantMigrate[name] {
				t.Errorf("%s POST schema 多出字段 %q（真实 handler migrateRequest 不解析；客户端按文档发送将被 DisallowUnknownFields 拒绝或忽略）", path, name)
			}
		}
	}

	// 2. presign method 枚举：真实 handler 仅 get/put/post（小写）。
	for _, method := range []string{"GET", "PUT", "DELETE", "HEAD"} {
		if docHasEnumValue(t, doc, "/api/accounts/{id}/presign", "post", "method", method) {
			t.Errorf("presign schema 声明 method=%q，真实 handler 仅支持 get/put/post（400 拒绝）", method)
		}
	}
	for _, method := range []string{"get", "put", "post"} {
		if !docHasEnumValue(t, doc, "/api/accounts/{id}/presign", "post", "method", method) {
			t.Errorf("presign schema 未声明 method=%q（真实 handler 支持）", method)
		}
	}

	// 3. delete 请求体：不得声明 versionId（真实 handler 无此字段）。
	if props := requestBodyProps(t, doc, "/api/accounts/{id}/delete", "post"); props["versionId"] {
		t.Errorf("delete POST schema 声明 versionId，真实 handler 不解析该字段")
	}

	// 4. multipart/part：应声明 expiresIn（真实 handler 支持）。
	if props := requestBodyProps(t, doc, "/api/accounts/{id}/multipart/part", "post"); !props["expiresIn"] {
		t.Errorf("multipart/part POST schema 缺少 expiresIn（真实 handler 解析）")
	}

	// 5. delete-marker/restore：必须与 handler.restoreDeleteMarker 一致，用 versionId。
	// 曾错误声明为 deleteMarkerId，客户端按文档调用会因缺 versionId 直接 400
	// （metadata.go:379-395）。此处同时断言「有 versionId」与「无 deleteMarkerId」。
	dmProps := requestBodyProps(t, doc, "/api/accounts/{id}/delete-marker/restore", "post")
	if !dmProps["versionId"] {
		t.Errorf("delete-marker/restore POST schema 缺少 versionId（真实 handler 解析）")
	}
	if dmProps["deleteMarkerId"] {
		t.Errorf("delete-marker/restore POST schema 声明 deleteMarkerId，真实 handler 不解析该字段")
	}
	for _, field := range []string{"bucket", "key"} {
		if !dmProps[field] {
			t.Errorf("delete-marker/restore POST schema 缺少字段 %q（真实 handler 解析）", field)
		}
	}

	// 6. version 删除（DELETE）与 version/restore：都以 versionId 标识版本。
	//    DELETE 走 query（handler metadata.go 只读 r.URL.Query()，曾误声明为 requestBody）；
	//    restore（POST）走 JSON body，禁止出现 deleteMarkerId 之类未解析字段。
	del := requestQueryParams(t, doc, "/api/accounts/{id}/version", "delete")
	for _, field := range []string{"bucket", "key", "versionId"} {
		if !del[field] {
			t.Errorf("version DELETE 缺少 query 参数 %q（真实 handler 从 query 读取）", field)
		}
	}
	restore := requestBodyProps(t, doc, "/api/accounts/{id}/version/restore", "post")
	if !restore["versionId"] {
		t.Errorf("version/restore POST schema 缺少 versionId（真实 handler 解析）")
	}
	if restore["deleteMarkerId"] {
		t.Errorf("version/restore POST schema 声明 deleteMarkerId，真实 handler 不解析该字段")
	}

	// 7. mkdir：真实 handler（objects.go mkdirObject）解析 key（写入时补 `/` 结尾），
	//    曾误写 prefix；copy-objects 同步 + 异步：真实 handler（copy.go）解析 keys，
	//    曾误写 items、且异步侧只声明空对象。按 OpenAPI 生成的客户端发这些字段会被
	//    DisallowUnknownFields 直接 400（2026-09-19 由 docs↔注册表字段门禁发现）。
	mk := requestBodyProps(t, doc, "/api/accounts/{id}/mkdir", "post")
	for _, field := range []string{"bucket", "key"} {
		if !mk[field] {
			t.Errorf("mkdir POST schema 缺少字段 %q（真实 handler 解析）", field)
		}
	}
	if mk["prefix"] {
		t.Errorf("mkdir POST schema 声明 prefix，真实 handler 只解析 key")
	}
	for _, path := range []string{"/api/accounts/{id}/copy-objects", "/api/accounts/{id}/copy-objects/async"} {
		props := requestBodyProps(t, doc, path, "post")
		for _, field := range []string{"bucket", "keys", "targetBucket", "targetPrefix", "deleteSource"} {
			if !props[field] {
				t.Errorf("%s POST schema 缺少字段 %q（真实 handler 解析）", path, field)
			}
		}
		if props["items"] {
			t.Errorf("%s POST schema 声明 items，真实 handler 只解析 keys", path)
		}
	}

	// 8. 契约幻影字段（2026-09-19 审查 §7.2 / P0-5）：注册表 + docs/api.md 都声明、
	//    但 handler 的请求结构体（readJSON 目标）**不含**该字段。因为 readJSON 用
	//    DisallowUnknownFields，按文档示例体调用必然 400。这里按「注册表 props 与 handler
	//    实际解析的字段集合**完全相等**」断言，两端任一侧漂移都会变红。
	//
	//    这四个端点的期望集合 = handler 的结构体 json tag，也正好等于前端 API 类型的入参
	//    （accounts 走 model.Account，故为 AccountInput 的字段集）。
	exact := []struct {
		path, method string
		fields       []string
	}{
		{"/api/accounts", "post", []string{
			"name", "endpoint", "publicEndpoint", "region", "accessKey", "secretKey", "bucket", "pathStyle", "useSSL",
		}},
		{"/api/accounts/{id}/rename", "post", []string{"bucket", "key", "newKey", "newBucket"}},
		{"/api/accounts/{id}/set-headers", "post", []string{"bucket", "key", "contentType", "metadata"}},
		{"/api/accounts/{id}/multipart/init", "post", []string{"bucket", "key", "contentType"}},
	}
	for _, tc := range exact {
		got := requestBodyProps(t, doc, tc.path, tc.method)
		want := map[string]bool{}
		for _, f := range tc.fields {
			want[f] = true
			if !got[f] {
				t.Errorf("%s %s schema 缺少字段 %q（真实 handler 解析；不声明则 OpenAPI 客户端不会发送）",
					tc.method, tc.path, f)
			}
		}
		for f := range got {
			if !want[f] {
				t.Errorf("%s %s schema 多出字段 %q：真实 handler 的结构体不解析它，按文档发送会被 DisallowUnknownFields 拒绝（400）",
					tc.method, tc.path, f)
			}
		}
	}
}

// docHasEnumValue 检查指定操作 schema 中字段的 enum 是否包含给定值。
func docHasEnumValue(t *testing.T, doc map[string]any, path, method, field, value string) bool {
	t.Helper()
	paths := openAPIPaths(t, doc)
	op, ok := paths[path][method]
	if !ok {
		t.Fatalf("openapi 无 %s %s", method, path)
	}
	rb, ok := op["requestBody"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s 无 requestBody", method, path)
	}
	content, ok := rb["content"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s requestBody 无 content", method, path)
	}
	mt, ok := content["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s requestBody 无 application/json", method, path)
	}
	schema, ok := mt["schema"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s requestBody schema 类型 = %T", method, path, mt["schema"])
	}
	if ref, _ := schema["$ref"].(string); ref != "" {
		schema, ok = derefComponent(doc, ref)
		if !ok {
			t.Fatalf("%s %s $ref 解析失败: %s", method, path, ref)
		}
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s %s schema 无 properties", method, path)
	}
	fieldSchema, ok := props[field].(map[string]any)
	if !ok {
		return false
	}
	enum, ok := fieldSchema["enum"].([]any)
	if !ok {
		return false
	}
	for _, v := range enum {
		if s, _ := v.(string); s == value {
			return true
		}
	}
	return false
}
