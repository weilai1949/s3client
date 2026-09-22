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
	"strconv"
	"strings"
	"testing"
)

// apiDocRouteRe 匹配 api.md 中「行首 METHOD /api...」的路由声明行。
var apiDocRouteRe = regexp.MustCompile(`^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS) (/api\S*)`)

// apiDocIndentedRouteRe 匹配「有缩进、但内容像路由声明」的行——这类行会被 apiDocRouteRe 漏掉。
//
// 为什么单独检测：`TestAPIDocMatchesRoutes` 是双向 diff，缩进行在**两个方向**上都漏——
// 「代码有、文档缩进写」会报 missing（fail-safe，看得见），但「文档有陈旧条目、且写成缩进」
// 两个方向都不匹配、**静默通过**（fail-open）。实测确认该 fail-open 存在，故显式拦缩进写法，
// 要求路由声明必须顶格，把排版约定变成机械约束。
var apiDocIndentedRouteRe = regexp.MustCompile(`^[ \t]+(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)[ \t]+/api\S*`)

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

// TestAPIDocRoutesAreFlushLeft 把 api.md 的排版约定变成机械约束：
// **路由声明行必须顶格**，不得缩进（含表格单元格内的写法）。
//
// 动机：`apiDocRouteRe` 只认行首，缩进行在两个方向上都被漏。实测「文档里有陈旧路由、但写成
// 缩进」时 `TestAPIDocMatchesRoutes` **静默通过**（fail-open）——陈旧条目就此永久留在文档里。
// 与其放宽解析（会误伤正文里偶然提到的方法名+路径），不如要求顶格并在违规时红灯。
func TestAPIDocRoutesAreFlushLeft(t *testing.T) {
	t.Parallel()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位 docs/api.md")
	}
	docPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "docs", "api.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("读取 %s: %v", docPath, err)
	}

	flushLeft := 0
	var indented []string
	for i, line := range strings.Split(string(data), "\n") {
		if apiDocRouteRe.MatchString(line) {
			flushLeft++
			continue
		}
		if apiDocIndentedRouteRe.MatchString(line) {
			indented = append(indented, strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
		}
	}

	// 自检：解析口径写坏时不要静默变绿。
	if flushLeft < 60 {
		t.Fatalf("只解析到 %d 条顶格路由行，疑似解析口径失效", flushLeft)
	}
	for _, s := range indented {
		t.Errorf("docs/api.md 第 %s 行是缩进的路由声明；路由行必须顶格（解析器只认行首，"+
			"缩进行会被双向 diff 同时漏掉，陈旧条目将静默留存）", s)
	}
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

// apiDocAliasRe 匹配「请求体与 `METHOD /api/x` 相同」或「请求体与 `copy-prefix` 相同」两种别名。
// 后者省略方法名与 /api 前缀（文档既有排版），故方法组可选、路径允许裸名。
var apiDocAliasRe = regexp.MustCompile("请求体与\\s*`?((?:GET|POST|PUT|DELETE|PATCH)\\s+)?`?(/api\\S+?|[a-z0-9-]+)`?\\s*相同")

// apiDocSectionRe 保留旧名（sectionText 使用），语义与 apiDocAliasRe 一致。
var apiDocSectionRe = apiDocAliasRe

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

// resolveAliasKey 把别名引用（可能省略方法名 / 带 /api 前缀）解析为真实路由 key。
// 解析不到时返回 ""。
func resolveAliasKey(sections map[string]string, method, ref string) string {
	ref = strings.TrimRight(ref, "`")
	if method = strings.TrimSpace(method); method != "" {
		return strings.ToUpper(method) + " " + ref
	}
	for k := range sections {
		path := strings.SplitN(k, " ", 2)[1]
		if path == ref || strings.HasSuffix(path, "/"+ref) {
			return k
		}
	}
	return ""
}

// sectionText 返回某路由的正文；若该段声明「请求体与 … 相同」则跟随别名
// （如 POST /api/migrate/async 复用 /api/migrate 的字段说明），seen 防环。
func sectionText(sections map[string]string, key string, seen map[string]bool) string {
	if seen[key] {
		return ""
	}
	seen[key] = true
	body := sections[key]
	if m := apiDocSectionRe.FindStringSubmatch(body); m != nil {
		ref := resolveAliasKey(sections, m[1], m[2])
		if ref == "" {
			return body
		}
		return sectionText(sections, ref, seen)
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

// ---- 文档侧请求体字段的机械抽取（双向门禁用） ----

// stripJSONComments 去掉 JSON 文本中的 `//` 行注释（docs/api.md 的示例体带注释），
// 同时保持字符串字面量内的 `//` 原样。
func stripJSONComments(s string) string {
	var b strings.Builder
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			if i < len(s) {
				b.WriteByte('\n')
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// jsonObjectSpans 返回文本中所有顶层 `{...}` 片段及其行前缀（用于区分请求体与 `200 {...}` 响应体）。
func jsonObjectSpans(s string) []struct{ obj, prefix string } {
	var out []struct{ obj, prefix string }
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		depth, inStr, esc := 0, false, false
		j := i
		for ; j < len(s); j++ {
			c := s[j]
			if inStr {
				switch {
				case esc:
					esc = false
				case c == '\\':
					esc = true
				case c == '"':
					inStr = false
				}
				continue
			}
			switch c {
			case '"':
				inStr = true
			case '{':
				depth++
			case '}':
				depth--
			}
			if depth == 0 {
				break
			}
		}
		if j >= len(s) {
			break
		}
		lineStart := strings.LastIndexByte(s[:i], '\n') + 1
		out = append(out, struct{ obj, prefix string }{s[i : j+1], strings.TrimSpace(s[lineStart:i])})
		i = j
	}
	return out
}

// responsePrefixRe 匹配响应体的行前缀（如 `200 `、`200`、`202 `）。
var responsePrefixRe = regexp.MustCompile(`^\d{3}\b`)

// topLevelJSONKeys 返回 JSON 对象顶层（depth==1）的键名。容错：注释已剥离、非法片段返回已识别的键。
func topLevelJSONKeys(obj string) []string {
	var out []string
	depth, inStr, esc := 0, false, false
	for i := 0; i < len(obj); i++ {
		c := obj[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			// 读取字符串字面量。
			j := i + 1
			var sb strings.Builder
			for j < len(obj) {
				if obj[j] == '\\' {
					sb.WriteByte(obj[j])
					if j+1 < len(obj) {
						sb.WriteByte(obj[j+1])
					}
					j += 2
					continue
				}
				if obj[j] == '"' {
					break
				}
				sb.WriteByte(obj[j])
				j++
			}
			k := j + 1
			for k < len(obj) && (obj[k] == ' ' || obj[k] == '\t' || obj[k] == '\n' || obj[k] == '\r') {
				k++
			}
			if depth == 1 && k < len(obj) && obj[k] == ':' {
				out = append(out, sb.String())
			}
			i = j
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	sort.Strings(out)
	return out
}

// apiDocRequestBodyFields 返回 docs/api.md 中每个端点的请求体顶层字段名。
// 取该端点正文（别名已解析）里第一个「非响应体」JSON 对象；没有则返回 nil。
func apiDocRequestBodyFields(t *testing.T) map[string][]string {
	t.Helper()
	sections := apiDocSections(t)
	out := map[string][]string{}
	for key := range sections {
		body := stripJSONComments(sectionText(sections, key, map[string]bool{}))
		for _, span := range jsonObjectSpans(body) {
			if responsePrefixRe.MatchString(span.prefix) {
				continue
			}
			out[key] = topLevelJSONKeys(span.obj)
			break
		}
	}
	return out
}

// TestAPIDocDocumentsRequestBodyFields docs/api.md 的请求体字段集必须与 OpenAPI 注册表**双向一致**。
//
// 旧实现用 `strings.Contains(body, name)` 单向校验（注册表 ⊆ 文档），有两处结构性盲区
// （docs/review-2026-09-19.md §4.3）：① 文档多写的幻影字段从不检查；② 子串碰撞——
// 字段 `key` 会被 `keys` / `secretKey` 满足、`newKey` 被 `newKeys` 满足。现改为机械抽取
// 文档请求体 JSON 的**顶层键**后双向比对，且不再做子串匹配。
//
// 别名段（「请求体与 `POST /api/x` 相同」/「请求体与 `copy-prefix` 相同」）按被引用端点校验。
// 注册表为自由体（`openapi.Obj()`，无 properties）的端点跳过——字段集无从比对。
func TestAPIDocDocumentsRequestBodyFields(t *testing.T) {
	t.Parallel()
	registry := openAPIRequestFields(t)
	doc := apiDocRequestBodyFields(t)

	var mismatches []string
	checked := 0
	for key, names := range registry {
		if len(names) == 0 {
			continue // 注册表自由体：无字段可比对（由 inputsource 门禁负责「该不该有 body」）。
		}
		checked++
		body := sectionText(apiDocSections(t), key, map[string]bool{})
		if strings.TrimSpace(body) == "" {
			mismatches = append(mismatches, key+"（docs/api.md 无该端点正文）")
			continue
		}
		docNames, ok := doc[key]
		if !ok {
			mismatches = append(mismatches, key+"（docs/api.md 未给出请求体 JSON，无法机械比对字段）")
			continue
		}
		want := map[string]bool{}
		for _, n := range names {
			want[n] = true
		}
		got := map[string]bool{}
		for _, n := range docNames {
			got[n] = true
		}
		for _, n := range names {
			if !got[n] {
				mismatches = append(mismatches, key+" 文档缺少请求体字段 `"+n+"`")
			}
		}
		for _, n := range docNames {
			if !want[n] {
				mismatches = append(mismatches, key+" 文档多出请求体字段 `"+n+"`（注册表/handler 不解析）")
			}
		}
	}
	if checked < 20 {
		t.Fatalf("仅比对 %d 个端点的请求体字段，疑似抽取口径失效", checked)
	}
	sort.Strings(mismatches)
	for _, s := range mismatches {
		t.Errorf("docs/api.md 与 OpenAPI 注册表字段漂移：%s", s)
	}
}
