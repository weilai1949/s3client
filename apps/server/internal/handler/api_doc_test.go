package handler

// api_doc_test.go —— docs/api.md 与 routes.go 的漂移门禁（ASSESSMENT I2）。
//
// 既有契约测试（openapi_contract_test.go）只保证 routes.go ↔ OpenAPI 注册表一致；
// 手写的 docs/api.md 长期没有自动化校验，删改路由时容易留下陈旧条目或漏记新端点。
// 本测试把 api.md 也当作契约事实来源之一：
//   - routes.go 注册的每条 API 路由必须在 api.md 中至少出现一次（漏记即红灯）；
//   - api.md 中出现的每条 /api 路由必须真实注册（陈旧条目即红灯）。
//
// 解析口径：只认「行首即为 METHOD /api...」的行（api.md 的既有排版约定），
// 查询串（`?bucket=B`）归一化掉；反引号包裹的行内引用不以方法名开头，天然忽略。
// 它随 `go test ./...` 在 CI 中执行，因此「文档即契约」不再依赖人工维护。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// apiDocRouteRe 匹配 api.md 中「行首 METHOD /api...」的路由声明行。
var apiDocRouteRe = regexp.MustCompile(`^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS) (/api\S*)`)

// apiDocRoutes 解析 docs/api.md，返回 path -> method 集合。
func apiDocRoutes(t *testing.T) map[string]map[string]bool {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位 docs/api.md")
	}
	// apps/server/internal/handler -> 仓库根 -> docs/api.md
	docPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "docs", "api.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("读取 %s: %v", docPath, err)
	}

	out := map[string]map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		m := apiDocRouteRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		method, path := m[1], m[2]
		if i := strings.IndexByte(path, '?'); i >= 0 {
			path = path[:i]
		}
		// 允许行尾反引号（`POST /api/x` 的收尾）被 \S* 吞入。
		path = strings.TrimRight(path, "`")
		if out[path] == nil {
			out[path] = map[string]bool{}
		}
		out[path][method] = true
	}
	if len(out) == 0 {
		t.Fatal("docs/api.md: 未解析到任何「METHOD /api」路由行")
	}
	return out
}

// TestAPIDocMatchesRoutes 双向校验 docs/api.md ↔ routes.go：
// 漏记（代码有、文档无）与陈旧（文档有、代码无）都会让测试红灯。
func TestAPIDocMatchesRoutes(t *testing.T) {
	t.Parallel()
	routes := routesFromSource(t)
	doc := apiDocRoutes(t)

	var missing, stale []string
	for p, methods := range routes {
		for m := range methods {
			if !doc[p][m] {
				missing = append(missing, m+" "+p)
			}
		}
	}
	for p, methods := range doc {
		for m := range methods {
			if !routes[p][m] {
				stale = append(stale, m+" "+p)
			}
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	for _, s := range missing {
		t.Errorf("routes.go 注册了 %s，但 docs/api.md 未记录（请补文档）", s)
	}
	for _, s := range stale {
		t.Errorf("docs/api.md 记录了 %s，但 routes.go 未注册（陈旧条目，请删文档）", s)
	}
}

// apiDocSectionRe 匹配 api.md 里的「请求体与 `METHOD /api/x` 相同」别名声明。
var apiDocSectionRe = regexp.MustCompile("请求体与\\s*`?(GET|POST|PUT|DELETE|PATCH)\\s+(/api\\S+?)`?\\s*相同")

// apiDocSections 解析 api.md，返回 "METHOD /path" -> 该路由条目到下一个路由条目之间的正文。
// 行首路由行是既有排版约定（见 apiDocRoutes 的解析口径）。
func apiDocSections(t *testing.T) map[string]string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位 docs/api.md")
	}
	docPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "docs", "api.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("读取 %s: %v", docPath, err)
	}

	sections := map[string]string{}
	var key string
	var buf []string
	flush := func() {
		if key != "" {
			sections[key] = strings.Join(buf, "\n")
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		m := apiDocRouteRe.FindStringSubmatch(line)
		if m == nil {
			if key != "" {
				buf = append(buf, line)
			}
			continue
		}
		flush()
		path := m[2]
		if i := strings.IndexByte(path, '?'); i >= 0 {
			path = path[:i]
		}
		key = m[1] + " " + strings.TrimRight(path, "`")
		buf = nil
	}
	flush()
	if len(sections) == 0 {
		t.Fatal("docs/api.md: 未解析到任何「METHOD /api」正文段")
	}
	return sections
}

// sectionText 返回某路由的正文；若该段声明「请求体与 `METHOD /path` 相同」则跟随别名
// （如 POST /api/migrate/async 复用 /api/migrate 的字段说明），seen 防环。
func sectionText(sections map[string]string, key string, seen map[string]bool) string {
	if seen[key] {
		return ""
	}
	seen[key] = true
	body := sections[key]
	if m := apiDocSectionRe.FindStringSubmatch(body); m != nil {
		return sectionText(sections, m[1]+" "+strings.TrimRight(m[2], "`"), seen)
	}
	return body
}

// openAPIRequestFields 从运行中的规范取回每个 operation 的 application/json 请求体字段名。
// 值为 nil 表示该端点没有请求体。
func openAPIRequestFields(t *testing.T) map[string][]string {
	t.Helper()
	var spec struct {
		Paths map[string]map[string]struct {
			RequestBody *struct {
				Content map[string]struct {
					Schema json.RawMessage `json:"schema"`
				} `json:"content"`
			} `json:"requestBody"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(fetchOpenAPIJSON(t), &spec); err != nil {
		t.Fatalf("解析 openapi.json: %v", err)
	}

	deref := func(raw json.RawMessage) map[string]any {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("解析 schema: %v", err)
		}
		if ref, ok := m["$ref"].(string); ok {
			name := ref[strings.LastIndex(ref, "/")+1:]
			raw, ok := spec.Components.Schemas[name]
			if !ok {
				t.Fatalf("$ref 指向不存在的 schema: %s", ref)
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("解析 $ref %s: %v", ref, err)
			}
		}
		return m
	}

	out := map[string][]string{}
	for path, ops := range spec.Paths {
		for method, op := range ops {
			if op.RequestBody == nil {
				continue
			}
			mt, ok := op.RequestBody.Content["application/json"]
			if !ok {
				continue
			}
			schema := deref(mt.Schema)
			props, _ := schema["properties"].(map[string]any)
			var fields []string
			for name := range props {
				fields = append(fields, name)
			}
			sort.Strings(fields)
			out[strings.ToUpper(method)+" "+path] = fields
		}
	}
	return out
}

// TestAPIDocDocumentsRequestBodyFields docs/api.md 必须出现 OpenAPI 注册表里每个
// 请求体字段名（ASSESSMENT H1 的残留面：端点漂移已有 TestAPIDocMatchesRoutes 兜底，
// 字段级漂移——文档写旧字段名、代码改新字段名——此前无人校验）。
// 别名段（「请求体与 `POST /api/x` 相同」）按被引用端点校验。
func TestAPIDocDocumentsRequestBodyFields(t *testing.T) {
	t.Parallel()
	fields := openAPIRequestFields(t)
	sections := apiDocSections(t)

	var missing []string
	for key, names := range fields {
		body := sectionText(sections, key, map[string]bool{})
		if body == "" {
			missing = append(missing, key+"（docs/api.md 无该端点正文）")
			continue
		}
		for _, name := range names {
			if !strings.Contains(body, name) {
				missing = append(missing, key+" 缺少请求体字段 `"+name+"`")
			}
		}
	}
	sort.Strings(missing)
	for _, s := range missing {
		t.Errorf("docs/api.md 与 OpenAPI 注册表字段漂移：%s", s)
	}
}
