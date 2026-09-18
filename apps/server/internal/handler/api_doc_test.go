package handler

// api_doc_test.go —— docs/api.md 与 routes.go 的漂移门禁（roadmap #1 / ASSESSMENT I2）。
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
