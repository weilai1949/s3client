package handler

// openapi_inputsource_test.go —— 注册表「请求体声明」与 handler「实际输入源」的机械对齐门禁。
//
// 背景：2026-09-19 的字段级门禁发现三处客户端可见失真（mkdir 注册表写 prefix、copy-objects 写
// items、DELETE /version 把 query 参数误声明为 requestBody）。`openapi_contract_test.go` 的
// 逐条断言只能覆盖「已经发现过」的端点，新端点写错输入源仍会漏。
//
// 本门禁把它做成通用比对：解析 routes.go 得到「路由 → handler 方法」，沿 handler 方法之间的
// 调用闭包找 `readJSON` / `json.NewDecoder`，与注册表每个 operation 是否声明 `Request:` 双向比对。
// 委托解码（如 parseMigrateRequest / parseCopyPrefix）通过调用闭包自然覆盖。
//
// 刻意不做的事：不校验字段名（由 TestAPIDocDocumentsRequestBodyFields 与契约测试负责）、
// 不校验类型与 required 语义。

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var (
	// routeRe 匹配 routes.go 的注册行，兼容 h.method 与 h.withStreamLimit(h.method) 两种写法。
	routeRe = regexp.MustCompile(`mux\.HandleFunc\("([A-Z]+) (/api\S*)",\s*h\.(\w+)(?:\(h\.(\w+)\))?\)`)
	// methodRe 匹配 handler 方法定义，用于切出方法体。
	methodRe = regexp.MustCompile(`(?m)^func \(h \*Handler\) (\w+)\(`)
	// opRe 匹配注册表的 operation 声明。
	opRe = regexp.MustCompile(`r\.Operation\("([A-Z]+)", "([^"]+)"`)
	// callRe 匹配方法体内对其它 handler 方法的调用（用于委托闭包）。
	callRe = regexp.MustCompile(`h\.(\w+)\(`)
)

// handlerSourceDir 返回本测试所在目录（internal/handler）。
func handlerSourceDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位 handler 源码目录")
	}
	return filepath.Dir(thisFile)
}

// readHandlerSources 读取目录下所有非测试 Go 源码。
func readHandlerSources(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			rel = path
		}
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s: %v", dir, err)
	}
	return out
}

// parseHandlerMethods 返回 方法名 -> 方法体。
func parseHandlerMethods(sources map[string]string) map[string]string {
	out := map[string]string{}
	for _, text := range sources {
		ms := methodRe.FindAllStringSubmatchIndex(text, -1)
		for i, m := range ms {
			name := text[m[2]:m[3]]
			start := m[0]
			end := len(text)
			if i+1 < len(ms) {
				end = ms[i+1][0]
			}
			out[name] = text[start:end]
		}
	}
	return out
}

// parseRoutes 返回 "METHOD /path" -> handler 方法名。
func parseRoutes(t *testing.T, sources map[string]string) map[string]string {
	t.Helper()
	text, ok := sources["routes.go"]
	if !ok {
		t.Fatal("未找到 routes.go")
	}
	out := map[string]string{}
	for _, m := range routeRe.FindAllStringSubmatch(text, -1) {
		method := m[3]
		if m[4] != "" { // h.withStreamLimit(h.inner)
			method = m[4]
		}
		out[m[1]+" "+m[2]] = method
	}
	return out
}

// parseRegistryBodyDecl 返回 "METHOD /path" -> 该 operation 是否声明了 requestBody。
func parseRegistryBodyDecl(sources map[string]string) map[string]bool {
	out := map[string]bool{}
	for name, text := range sources {
		if !strings.HasPrefix(name, "openapi_register_") || !strings.HasSuffix(name, ".go") {
			continue
		}
		ops := opRe.FindAllStringSubmatchIndex(text, -1)
		for i, m := range ops {
			key := text[m[2]:m[3]] + " " + text[m[4]:m[5]]
			end := len(text)
			if i+1 < len(ops) {
				end = ops[i+1][0]
			}
			out[key] = strings.Contains(text[m[0]:end], "Request:")
		}
	}
	return out
}

// decodesBody 沿 handler 方法调用闭包判断某方法是否最终会解码请求体。
// 记忆化 + 置 false 防环；结果对每个方法只算一次。
func decodesBody(methods map[string]string) map[string]bool {
	memo := map[string]bool{}
	var visit func(name string) bool
	visit = func(name string) bool {
		if v, done := memo[name]; done {
			return v
		}
		memo[name] = false // 防调用环
		body, ok := methods[name]
		if !ok {
			return false
		}
		found := strings.Contains(body, "readJSON(") || strings.Contains(body, "json.NewDecoder")
		if !found {
			for _, c := range callRe.FindAllStringSubmatch(body, -1) {
				if visit(c[1]) {
					found = true
					break
				}
			}
		}
		memo[name] = found
		return found
	}
	for name := range methods {
		visit(name)
	}
	return memo
}

// TestOpenAPIRequestDeclarationMatchesHandlerInput 双向校验：
// 注册表声明 requestBody ⇔ handler（含委托）实际解码请求体。
func TestOpenAPIRequestDeclarationMatchesHandlerInput(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	methods := parseHandlerMethods(sources)
	routes := parseRoutes(t, sources)
	decl := parseRegistryBodyDecl(sources)
	decodes := decodesBody(methods)

	// 自检：解析口径写坏时不要静默变绿。
	if len(routes) < 60 || len(methods) < 60 || len(decl) < 60 {
		t.Fatalf("解析结果异常（routes=%d methods=%d operations=%d），疑似解析口径失效",
			len(routes), len(methods), len(decl))
	}

	checked := 0
	for key, method := range routes {
		want, declared := decl[key]
		if !declared {
			t.Errorf("routes.go 注册了 %s，但注册表没有对应 operation", key)
			continue
		}
		checked++
		if got := decodes[method]; got != want {
			if want {
				t.Errorf("注册表为 %s 声明了 requestBody，但 handler %s 不解码请求体（参数会被忽略，客户端拿到 400）", key, method)
			} else {
				t.Errorf("handler %s（%s）解码请求体，但注册表未声明 requestBody（OpenAPI 客户端不会发送 body）", method, key)
			}
		}
	}
	if checked < 60 {
		t.Fatalf("仅比对 %d 条路由，疑似注册表与路由解析不匹配", checked)
	}
}
