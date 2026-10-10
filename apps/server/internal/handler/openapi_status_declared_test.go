package handler

// openapi_status_declared_test.go —— **响应状态码声明门禁**：handler 直接写出的
// `http.StatusXxx`（含经 `h.xxx()` 委托闭包）必须在该 operation 的注册表 Responses 中声明。
//
// 背景（docs/KNOWN_ISSUES.md #79 / 评审 2026-10-09 §5 O8）：注册表此前只校验状态码**语法**
// （`TestOpenAPI_ContractOperationsAreComplete`）与响应 **schema**（`openapi_response_contract_test.go`），
// 不校验「handler 会返回的状态码是否声明」——migrate jobs 的 404、trash 的 409、copy 的 409、
// proxy 的 400/416 等长期缺失而无人发现。
//
// 口径（只认能静态确定的量）：
//   - 抽取 handler 方法体内 `writeErr(w, http.StatusXxx, …)` / `writeJSON(w, http.StatusXxx, …)`
//     的常量，并沿 `h.xxx()` 调用闭包递归（覆盖 helper 里写死的状态码）；
//   - **动态状态码**（`writeErr(w, s3HTTPStatus(err), …)` 这类运行期求值）无法静态判定，
//     不在本门禁范围——其映射表另由 `writeInternalErr` 的 `s3HTTPStatus` 统一维护，相关端点
//     在注册表手工补齐并注释说明；
//   - 中间件 / 共享兜底注入的**通用**状态码（400 / 401 / 403 / 404 / 405 / 413 / 429 / 500）
//     由 `globalStatuses` 白名单放行，端点无需逐个声明（复用共享 `components.responses`）；
//     其余状态码（409 / 412 / 416 / 501 / 503 / 201 / 202 …）必须逐端点显式声明。

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// statusConstCodes 把源码中出现的 `http.StatusXxx` 常量映射为数字状态码。
// 新常量若不在表内，门禁以「未知状态常量」红灯提示补表（避免静默跳过）。
var statusConstCodes = map[string]int{
	"StatusOK":                           200,
	"StatusCreated":                      201,
	"StatusAccepted":                     202,
	"StatusNoContent":                    204,
	"StatusBadRequest":                   400,
	"StatusUnauthorized":                 401,
	"StatusForbidden":                    403,
	"StatusNotFound":                     404,
	"StatusConflict":                     409,
	"StatusPreconditionFailed":           412,
	"StatusRequestEntityTooLarge":        413,
	"StatusRequestedRangeNotSatisfiable": 416,
	"StatusTooManyRequests":              429,
	"StatusInternalServerError":          500,
	"StatusNotImplemented":               501,
	"StatusServiceUnavailable":           503,
}

// globalStatuses 是「全端点通用」的状态码，由中间件 / 共享兜底 / 共享 components.responses
// 覆盖，端点无需逐个声明：
//   - 400 / 404 / 500：`components.responses` 的 BadRequest / NotFound / InternalError；
//   - 401 / 429 / 405：withAuth / 限流 / mux 注入；
//   - 403：作用域中间件（`auth.scope_denied`）与 CORS 预检；
//   - 413：`http.MaxBytesReader` 超限（`maxBody`）。
//
// 其余状态码（如 409 / 412 / 416 / 501 / 503 / 201 / 202）必须逐端点显式声明。
var globalStatuses = map[int]bool{400: true, 401: true, 403: true, 404: true, 405: true, 413: true, 429: true, 500: true}

// directStatusRe 匹配 `writeErr(w, http.StatusXxx` / `writeJSON(w, http.StatusXxx`。
var directStatusRe = regexp.MustCompile(`\bwrite(?:Err|JSON)\([^,]+,\s*http\.(Status[A-Za-z]+)`)

// handlerMethodStatuses 返回某 handler 方法（含 `h.xxx()` 委托闭包）直接写出的状态码集合。
func handlerMethodStatuses(t *testing.T, methods map[string]string, name string) map[int]bool {
	t.Helper()
	out := map[int]bool{}
	seen := map[string]bool{}
	var walk func(string)
	walk = func(n string) {
		if seen[n] {
			return
		}
		seen[n] = true
		body, ok := methods[n]
		if !ok {
			return
		}
		for _, m := range directStatusRe.FindAllStringSubmatch(body, -1) {
			code, ok := statusConstCodes[m[1]]
			if !ok {
				t.Errorf("未知状态常量 http.%s —— 请补入 statusConstCodes", m[1])
				continue
			}
			out[code] = true
		}
		for _, c := range parseBodyCalls(n, body).calls {
			walk(c)
		}
	}
	walk(name)
	return out
}

// TestOpenAPIStatusCodesAreDeclared 全量遍历：handler 直接写出的状态码必须已声明。
func TestOpenAPIStatusCodesAreDeclared(t *testing.T) {
	t.Parallel()
	dir := handlerSourceDir(t)
	sources := readHandlerSources(t, dir)
	methods := parseHandlerMethods(sources)
	routes := parseRoutes(t, sources)
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	if len(routes) < 60 || len(methods) < 60 {
		t.Fatalf("解析结果异常（routes=%d methods=%d），疑似口径失效", len(routes), len(methods))
	}

	checked := 0
	for key, method := range routes {
		m, path, _ := strings.Cut(key, " ")
		op, ok := paths[path][strings.ToLower(m)]
		if !ok {
			t.Fatalf("openapi 无 %s", key)
		}
		resps, _ := op["responses"].(map[string]any)
		declared := map[int]bool{}
		for s := range resps {
			if n, err := strconv.Atoi(s); err == nil {
				declared[n] = true
			}
		}
		checked++
		for code := range handlerMethodStatuses(t, methods, method) {
			if globalStatuses[code] || declared[code] {
				continue
			}
			t.Errorf("%s 的 handler %s 直接写出状态码 %d，但注册表 Responses 未声明", key, method, code)
		}
	}
	if checked < 60 {
		t.Fatalf("仅检查 %d 个端点，疑似遍历口径失效", checked)
	}
}

// TestOpenAPIUniversalResponsesAreWiredPerOperation 断言中间件级通用状态码已逐 operation 接线，
// 且 components.responses 无零引用孤儿：
//   - 429（限流）/ 500（兜底）→ 全部端点；
//   - 401（withAuth / scope）→ 除 `security: []` 豁免端点外全部端点；
//   - 413（16 MB 请求体上限，`handler.go` decodeBody）→ 仅带 requestBody 的端点。
//
// 背景：`Unauthorized` / `TooManyRequests` / `InternalError` 三个共享响应此前**定义后 0 引用**，
// 84 个 operation 无一声明 401 / 413 / 429 / 500——代码生成器与 Swagger UI 看不到这些状态，
// 而 docs/api.md 把 401 / 500 列为通用码、「共享 responses 已全部接线为 $ref」。
// 通用码由中间件统一产生，逐条手写 84×4 必然漂移，故在注册完成后统一接线并由本门禁钉住。
func TestOpenAPIUniversalResponsesAreWiredPerOperation(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	comps, _ := doc["components"].(map[string]any)
	respComps, _ := comps["responses"].(map[string]any)
	if len(respComps) < 5 {
		t.Fatalf("components.responses 只有 %d 个，疑似口径失效", len(respComps))
	}

	refCount := map[string]int{}
	checked := 0
	for path, methods := range paths {
		for m, op := range methods {
			checked++
			resps, _ := op["responses"].(map[string]any)
			if resps == nil {
				t.Errorf("%s %s 缺少 responses", m, path)
				continue
			}
			noAuth := false
			if s, ok := op["security"].([]any); ok {
				noAuth = len(s) == 0
			}
			want := []string{"429", "500"}
			if !noAuth {
				want = append(want, "401")
			}
			if _, hasBody := op["requestBody"]; hasBody {
				want = append(want, "413")
			}
			for _, s := range want {
				if _, ok := resps[s]; !ok {
					t.Errorf("%s %s 未声明通用状态码 %s（中间件会产生该响应）", m, path, s)
				}
			}
			for _, r := range resps {
				if rm, _ := r.(map[string]any); rm != nil {
					if ref, _ := rm["$ref"].(string); ref != "" {
						refCount[ref]++
					}
				}
			}
		}
	}
	if checked < 60 {
		t.Fatalf("仅检查 %d 个 operation，疑似遍历口径失效", checked)
	}
	for name := range respComps {
		if refCount["#/components/responses/"+name] == 0 {
			t.Errorf("components.responses.%s 零引用——孤儿共享响应，契约消费者（codegen / Swagger UI）看不到它",
				name)
		}
	}
}

// TestOpenAPIBucketNotRequiredInRequestBody 所有请求体的 `bucket` 都必须可选：handler 经
// `bucketOr` 回退账号默认桶（全仓统一语义），共享 `Bucket` query 参数也文档为「可省略」。
// 注册表若把 `bucket` 标 required 即契约谎言（KNOWN_ISSUES #79）。
func TestOpenAPIBucketNotRequiredInRequestBody(t *testing.T) {
	t.Parallel()
	sources := readHandlerSources(t, handlerSourceDir(t))
	routes := parseRoutes(t, sources)
	doc := openAPIDoc(t)

	checked := 0
	for key := range routes {
		required, hasBody := registryRequestBodyRequired(t, doc, key)
		if !hasBody {
			continue
		}
		checked++
		for _, f := range required {
			if f == "bucket" {
				t.Errorf("%s 把可回退账号默认桶的 bucket 标为 required（应为可选，与共享 Bucket query 参数语义一致）", key)
			}
		}
	}
	if checked < 20 {
		t.Fatalf("仅检查 %d 个请求体，疑似口径失效", checked)
	}
}
