package main

// openapi_examples_gate_test.go —— OpenAPI **示例**与 docs/api.md **curl 示例**的机械门禁。
//
// 背景：`docs/api/openapi.json` 此前有 5 个共享 schema、70 个 operation，但 `example` 字段
// 数为 0——契约能说明「有哪些字段」，却没有任何「长什么样」的样例，外部代码生成器 / Swagger UI
// 拿到的骨架全是占位符；`docs/api.md` 同样一条 curl 示例都没有，复制即用的门槛很高。
// 补示例本身是文档工作，但示例最容易「补一次就腐烂」：handler DTO 改了字段、路由被删除、
// 或示例 key 拼错，规范里就会留下指向不存在字段的样例，而没有任何测试会红。
//
// 本门禁把「示例」也钉成契约事实：
//   - 每个 operation 的 requestBody 必须有 application/json example；
//   - 每个 operation 必须至少有一个带 example（响应级或 schema 级）的 2xx 响应；
//   - 示例必须能对上 schema：未知字段 / 缺 required / 类型不符 / 枚举越界都会红灯并点名 operation；
//   - docs/api.md 的 curl 示例数达到下限，且覆盖 OpenAPI 顶层 `tags` 的**每一个**分组；
//     除 OpenAPI 显式声明 `security: []` 的端点外，每条 curl 必须带 `Authorization: Bearer`。
//
// 防「空扫描静默变绿」：规范解析出的 operation 数、api.md 解析出的 curl 数、tags 数都设下限，
// 解析口径失效时直接 Fatal，而不是 0 条比对通过。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	// openAPISpecRelPath / apiDocRelPath 相对仓库根。
	openAPISpecRelPath = "docs/api/openapi.json"
	apiDocRelPath      = "docs/api.md"

	// minScannedOperations 自检下限：规范里可解析的 operation 数不得低于此值。
	minScannedOperations = 60
	// minAPIDocCurlExamples 自检下限：api.md 里可解析的 curl 示例数不得低于此值。
	minAPIDocCurlExamples = 10
)

// specOp 是一个已注册 operation 的解析结果（只保留门禁需要的事实）。
type specOp struct {
	key     string // "METHOD /path"
	method  string
	path    string
	tags    []string
	noAuth  bool
	op      map[string]any
	request map[string]any
	// responses 已解析 $ref，key 为状态码。
	responses map[string]map[string]any
}

// loadOpenAPIExamplesSpec 读取提交版 OpenAPI 规范（docs/api/openapi.json）。
func loadOpenAPIExamplesSpec(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join(repoRoot(t), filepath.FromSlash(openAPISpecRelPath))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v（示例门禁依赖提交版规范，需先 go test ./internal/handler/ -run TestCommittedOpenAPISpecMatchesRuntime -update-openapi-spec）", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("解析 %s: %v", path, err)
	}
	return doc
}

// resolveSchemaRef 解析 `$ref: #/components/schemas/X`；非引用原样返回。
func resolveSchemaRef(t *testing.T, doc map[string]any, schema map[string]any) map[string]any {
	t.Helper()
	ref, _ := schema["$ref"].(string)
	if ref == "" {
		return schema
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		t.Fatalf("不支持的 $ref: %s", ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	comps, _ := doc["components"].(map[string]any)
	schemas, _ := comps["schemas"].(map[string]any)
	target, ok := schemas[name].(map[string]any)
	if !ok {
		t.Fatalf("$ref 指向不存在的 schema: %s", ref)
	}
	return target
}

// resolveResponseRef 解析 `$ref: #/components/responses/X`；非引用原样返回。
func resolveResponseRef(t *testing.T, doc map[string]any, resp map[string]any) map[string]any {
	t.Helper()
	ref, _ := resp["$ref"].(string)
	if ref == "" {
		return resp
	}
	const prefix = "#/components/responses/"
	if !strings.HasPrefix(ref, prefix) {
		t.Fatalf("不支持的响应 $ref: %s", ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	comps, _ := doc["components"].(map[string]any)
	responses, _ := comps["responses"].(map[string]any)
	target, ok := responses[name].(map[string]any)
	if !ok {
		t.Fatalf("$ref 指向不存在的响应: %s", ref)
	}
	return target
}

// walkOperations 遍历 paths，返回按 key 排序的 operation 列表。
func walkOperations(t *testing.T, doc map[string]any) []specOp {
	t.Helper()
	paths, _ := doc["paths"].(map[string]any)
	if len(paths) == 0 {
		t.Fatal("openapi.json 的 paths 为空，示例门禁无法比对")
	}
	var out []specOp
	for path, rawOps := range paths {
		ops, _ := rawOps.(map[string]any)
		for method, rawOp := range ops {
			op, _ := rawOp.(map[string]any)
			if op == nil {
				continue
			}
			so := specOp{
				key:    strings.ToUpper(method) + " " + path,
				method: strings.ToUpper(method),
				path:   path,
				op:     op,
			}
			if tags, ok := op["tags"].([]any); ok {
				for _, tg := range tags {
					if s, ok := tg.(string); ok {
						so.tags = append(so.tags, s)
					}
				}
			}
			if sec, ok := op["security"].([]any); ok && len(sec) == 0 {
				so.noAuth = true
			}
			if rb, ok := op["requestBody"].(map[string]any); ok {
				so.request = rb
			}
			so.responses = map[string]map[string]any{}
			if resps, ok := op["responses"].(map[string]any); ok {
				for status, rawResp := range resps {
					if m, ok := rawResp.(map[string]any); ok {
						so.responses[status] = resolveResponseRef(t, doc, m)
					}
				}
			}
			out = append(out, so)
		}
	}
	if len(out) < minScannedOperations {
		t.Fatalf("只解析到 %d 个 operation（下限 %d），疑似 paths 解析口径失效", len(out), minScannedOperations)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

// jsonMedia 取 content 下的 application/json 媒体对象；缺失返回 nil。
func jsonMedia(content map[string]any) map[string]any {
	if content == nil {
		return nil
	}
	mt, _ := content["application/json"].(map[string]any)
	return mt
}

// schemaHasExample 判断 schema 自身（或 $ref 目标）带 example。
func schemaHasExample(t *testing.T, doc map[string]any, mt map[string]any) bool {
	t.Helper()
	if mt == nil {
		return false
	}
	if mt["example"] != nil {
		return true
	}
	schema, _ := mt["schema"].(map[string]any)
	if schema == nil {
		return false
	}
	return resolveSchemaRef(t, doc, schema)["example"] != nil
}

// typeMatches 粗略校验 example 与 schema 的 type 是否吻合。
func typeMatches(schemaType string, example any) bool {
	switch schemaType {
	case "":
		return true
	case "string":
		_, ok := example.(string)
		return ok
	case "boolean":
		_, ok := example.(bool)
		return ok
	case "integer":
		f, ok := example.(float64)
		return ok && f == float64(int64(f))
	case "number":
		_, ok := example.(float64)
		return ok
	case "object":
		_, ok := example.(map[string]any)
		return ok
	case "array":
		_, ok := example.([]any)
		return ok
	default:
		return true
	}
}

// validateExampleShape 递归校验 example 是否符合 schema：未知字段、缺 required、类型、枚举。
// 只对声明了 properties 的对象做字段级校验（自由体 / 嵌套 metadata 等不误报）。
func validateExampleShape(t *testing.T, doc map[string]any, example any, schema map[string]any, where string) {
	t.Helper()
	if schema == nil {
		return
	}
	schema = resolveSchemaRef(t, doc, schema)

	typ, _ := schema["type"].(string)
	if !typeMatches(typ, example) {
		t.Errorf("%s: 示例类型与 schema 不符（schema type=%s，示例=%T）", where, typ, example)
		return
	}
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		hit := false
		for _, v := range enum {
			if fmt.Sprint(v) == fmt.Sprint(example) {
				hit = true
			}
		}
		if !hit {
			t.Errorf("%s: 示例值 %v 不在 schema 枚举 %v 内", where, example, enum)
		}
	}

	props, _ := schema["properties"].(map[string]any)
	obj, isObj := example.(map[string]any)
	if typ == "object" && len(props) > 0 && isObj {
		var unknown []string
		for k := range obj {
			if _, ok := props[k]; !ok {
				unknown = append(unknown, k)
			}
		}
		sort.Strings(unknown)
		for _, k := range unknown {
			t.Errorf("%s: 示例含 schema 未声明字段 %q（示例会指向幻影字段）", where, k)
		}
		if required, ok := schema["required"].([]any); ok {
			for _, r := range required {
				name, _ := r.(string)
				if _, ok := obj[name]; !ok {
					t.Errorf("%s: 示例缺少 required 字段 %q", where, name)
				}
			}
		}
		for k, v := range obj {
			ps, _ := props[k].(map[string]any)
			if ps == nil {
				continue
			}
			validateExampleShape(t, doc, v, ps, where+"."+k)
		}
		return
	}

	if typ == "array" {
		items, _ := schema["items"].(map[string]any)
		arr, _ := example.([]any)
		for i, v := range arr {
			validateExampleShape(t, doc, v, items, fmt.Sprintf("%s[%d]", where, i))
		}
	}
}

// TestOpenAPIExamplesCoverEveryOperation 断言每个 operation 都有可机械校验的示例。
func TestOpenAPIExamplesCoverEveryOperation(t *testing.T) {
	t.Parallel()
	doc := loadOpenAPIExamplesSpec(t)
	ops := walkOperations(t, doc)

	schemas, _ := doc["components"].(map[string]any)
	sharedSchemas, _ := schemas["schemas"].(map[string]any)
	if len(sharedSchemas) < 5 {
		t.Fatalf("components.schemas 只有 %d 个（预期 ≥5），示例门禁无法比对共享 schema", len(sharedSchemas))
	}
	withExample := 0
	for name, raw := range sharedSchemas {
		if m, ok := raw.(map[string]any); ok && m["example"] != nil {
			withExample++
			validateExampleShape(t, doc, m["example"], m, "components.schemas."+name)
		}
	}
	if withExample < len(sharedSchemas) {
		t.Errorf("components.schemas 有 %d/%d 个带 example；每个共享 schema 都应有示例", withExample, len(sharedSchemas))
	}

	missingReq, missingResp, unvalidated := 0, 0, 0
	for _, so := range ops {
		// (a) 有 requestBody 的 operation 必须有 application/json 示例。
		if so.request != nil {
			content, _ := so.request["content"].(map[string]any)
			mt := jsonMedia(content)
			if mt == nil {
				t.Errorf("%s: requestBody 缺 application/json", so.key)
				unvalidated++
			} else if !schemaHasExample(t, doc, mt) {
				missingReq++
				t.Errorf("%s: requestBody 缺 example（请求示例缺失）", so.key)
			} else {
				schema, _ := mt["schema"].(map[string]any)
				example := mt["example"]
				if example == nil {
					example = resolveSchemaRef(t, doc, schema)["example"]
				}
				validateExampleShape(t, doc, example, schema, so.key+" requestBody")
			}
		}

		// (b) 每个 operation 至少有一个带示例的 2xx 响应。
		twoXX := 0
		ok2xx := false
		for status, resp := range so.responses {
			if !strings.HasPrefix(status, "2") {
				continue
			}
			twoXX++
			content, _ := resp["content"].(map[string]any)
			for _, rawMT := range content {
				mt, _ := rawMT.(map[string]any)
				if mt == nil || !schemaHasExample(t, doc, mt) {
					continue
				}
				ok2xx = true
				schema, _ := mt["schema"].(map[string]any)
				example := mt["example"]
				if example == nil {
					example = resolveSchemaRef(t, doc, schema)["example"]
				}
				validateExampleShape(t, doc, example, schema, so.key+" "+status)
			}
		}
		if twoXX == 0 {
			t.Errorf("%s: 没有任何 2xx 响应", so.key)
		} else if !ok2xx {
			missingResp++
			t.Errorf("%s: 2xx 响应缺 example（响应级与 schema 级都没有）", so.key)
		}
	}

	if missingReq+missingResp+unvalidated > 0 {
		t.Logf("示例缺口：request=%d response=%d 无法解析=%d", missingReq, missingResp, unvalidated)
	}
	t.Logf("已校验 %d 个 operation、%d 个共享 schema 示例", len(ops), withExample)
}

// ---- docs/api.md curl 示例 ----

// apiDocBashBlockRe 匹配 api.md 中的 ```bash 代码块。
var apiDocBashBlockRe = regexp.MustCompile("(?s)```bash\n(.*?)```")

// curlMethodRe 从 curl 命令提取方法（无 -X 视作 GET）。
var curlMethodRe = regexp.MustCompile(`-X[ \t]+(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)`)

// curlPathRe 提取 curl 命令里的 /api 路径（兼容 $BASE 变量与绝对 URL）。
var curlPathRe = regexp.MustCompile(`(?:https?://[^\s"'\\]+)?(/api/[^\s"'\\?]+)`)

// specPathMatches 判断 URL 中的实际路径是否匹配规范的路径模板（{id} 段匹配任意非空段）。
func specPathMatches(template, actual string) bool {
	tp := strings.Split(strings.Trim(template, "/"), "/")
	ap := strings.Split(strings.Trim(actual, "/"), "/")
	if len(tp) != len(ap) {
		return false
	}
	for i := range tp {
		if strings.HasPrefix(tp[i], "{") && strings.HasSuffix(tp[i], "}") {
			if ap[i] == "" {
				return false
			}
			continue
		}
		if tp[i] != ap[i] {
			return false
		}
	}
	return true
}

// TestAPIDocCurlExamplesCoverEveryTag 断言 api.md 的 curl 示例覆盖全部 tag 分组且带鉴权头。
func TestAPIDocCurlExamplesCoverEveryTag(t *testing.T) {
	t.Parallel()
	doc := loadOpenAPIExamplesSpec(t)
	ops := walkOperations(t, doc)

	// tag 声明（顶层 tags）。
	var declaredTags []string
	if tags, ok := doc["tags"].([]any); ok {
		for _, tg := range tags {
			if m, ok := tg.(map[string]any); ok {
				if name, ok := m["name"].(string); ok {
					declaredTags = append(declaredTags, name)
				}
			}
		}
	}
	if len(declaredTags) < 10 {
		t.Fatalf("顶层 tags 只有 %d 个（预期 10），示例覆盖门禁无法比对", len(declaredTags))
	}

	// path -> method -> op（用于把 curl 映射回 tag / security）。
	byPath := map[string]map[string]specOp{}
	for _, so := range ops {
		if byPath[so.path] == nil {
			byPath[so.path] = map[string]specOp{}
		}
		byPath[so.path][so.method] = so
	}

	path := filepath.Join(repoRoot(t), filepath.FromSlash(apiDocRelPath))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	md := string(b)

	blocks := apiDocBashBlockRe.FindAllStringSubmatch(md, -1)
	curlCount := 0
	tagCovered := map[string]int{}
	var unmapped []string
	for _, m := range blocks {
		block := m[1]
		if !strings.Contains(block, "curl ") && !strings.Contains(block, "curl\n") {
			continue
		}
		curlCount++
		method := "GET"
		if mm := curlMethodRe.FindStringSubmatch(block); mm != nil {
			method = mm[1]
		}
		pm := curlPathRe.FindStringSubmatch(block)
		if pm == nil {
			unmapped = append(unmapped, "curl 块未含 /api 路径："+firstLine(block))
			continue
		}
		actual := pm[1]

		var matched *specOp
		for specPath, methods := range byPath {
			so, ok := methods[method]
			if !ok || !specPathMatches(specPath, actual) {
				continue
			}
			cp := so
			matched = &cp
			break
		}
		if matched == nil {
			unmapped = append(unmapped, fmt.Sprintf("%s %s", method, actual))
			continue
		}
		if len(matched.tags) == 0 {
			t.Errorf("curl 示例 %s %s 映射到无 tag 的 operation", method, matched.path)
		}
		for _, tg := range matched.tags {
			tagCovered[tg]++
		}
		// 鉴权头：OpenAPI 显式 security: [] 的端点豁免，其余必须带 Bearer。
		if !matched.noAuth && !strings.Contains(block, "Authorization: Bearer") {
			t.Errorf("%s 的 curl 示例缺 `Authorization: Bearer` 头（该端点未声明 security: []）", matched.key)
		}
	}

	if curlCount < minAPIDocCurlExamples {
		t.Fatalf("docs/api.md 只解析到 %d 条 curl 示例（下限 %d），疑似 curl 示例被删或解析口径失效",
			curlCount, minAPIDocCurlExamples)
	}
	for _, u := range unmapped {
		t.Errorf("curl 示例无法映射到任何已注册 operation（路径/方法拼错或路由已删）：%s", u)
	}
	var uncovered []string
	for _, tg := range declaredTags {
		if tagCovered[tg] == 0 {
			uncovered = append(uncovered, tg)
		}
	}
	sort.Strings(uncovered)
	if len(uncovered) > 0 {
		t.Errorf("以下 tag 分组在 docs/api.md 没有任何 curl 示例：%v", uncovered)
	}
	t.Logf("docs/api.md curl 示例 %d 条，覆盖 tag：%v", curlCount, tagCovered)
}

// firstLine 返回字符串首行（错误信息里定位 curl 块）。
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
