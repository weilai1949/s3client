package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// quietLogger 把日志降级到 discard，避免 CI 测试输出噪声。
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestOpenAPI_ExposesAllRoutes 验证 /api/openapi.json 能取到、路由总数与 routes.go 一致、
// 核心端点已登记且带 operationId/summary、bearerAuth 存在。
// 完整的 routes.go ↔ 规范双向比对见 openapi_contract_test.go 的 TestOpenAPI_ContractRoutesMatchSpec。
func TestOpenAPI_ExposesAllRoutes(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)
	paths := openAPIPaths(t, doc)

	// 路由总数以 routes.go 的实际注册为准（解析 AST），杜绝注释/常量漂移。
	routes := routesFromSource(t)
	total := 0
	for _, methods := range routes {
		total += len(methods)
	}
	if total != apiRouteCount {
		t.Fatalf("routes.go API 路由数 = %d, want %d", total, apiRouteCount)
	}
	if got := countOperations(paths); got != total {
		t.Errorf("规范 operation 数 = %d, want %d", got, total)
	}

	// 抽样：核心端点必须存在且 operationId/summary 至少一个非空。
	for _, mustExist := range []string{
		"/api/health",
		"/api/accounts",
		"/api/accounts/{id}",
		"/api/accounts/{id}/objects",
		"/api/accounts/{id}/presign",
		"/api/migrate/async",
		"/api/openapi.json",
	} {
		entry, ok := paths[mustExist]
		if !ok {
			t.Errorf("missing path %s", mustExist)
			continue
		}
		hasContent := false
		for _, m := range entry {
			if s, _ := m["operationId"].(string); s != "" {
				hasContent = true
				break
			}
			if s, _ := m["summary"].(string); s != "" {
				hasContent = true
				break
			}
		}
		if !hasContent {
			t.Errorf("%s: no operationId/summary on any method", mustExist)
		}
	}

	comps, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("components 类型 = %T, want object", doc["components"])
	}
	sec, ok := comps["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatalf("securitySchemes 类型 = %T, want object", comps["securitySchemes"])
	}
	if sec["bearerAuth"] == nil {
		t.Error("bearerAuth scheme missing")
	}
}

// TestOpenAPI_DescNilSafe 直测注册辅助 desc 的 nil 保护与就地补描述语义。
// desc 的所有生产调用点都传非 nil schema，nil 分支是纯防御性守卫；
// 这里显式覆盖，证明守卫语义正确（而不是仅为了数字）。
func TestOpenAPI_DescNilSafe(t *testing.T) {
	t.Parallel()
	if got := desc(nil, "ignored"); got != nil {
		t.Errorf("desc(nil, ...) = %v, want nil", got)
	}
	s := openapi.Str()
	if got := desc(s, "说明"); got != s {
		t.Errorf("desc 应返回同一 *Schema 指针，got %p want %p", got, s)
	} else if s.Description != "说明" {
		t.Errorf("desc 未补描述: %q", s.Description)
	}
}

// TestOpenAPI_GateAndAuth 验证 openapi.json 默认 404（S3C_EXPOSE_OPENAPI 未开）；
// 显式开启后需 Bearer 鉴权（配置 token 时），否则 401。
func TestOpenAPI_GateAndAuth(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	// 1) 默认（未开启）：一律 404，不泄露端点信息。
	hHidden := New(st, quietLogger(), t.TempDir(), nil, "", "test", false, false)
	t.Cleanup(hHidden.Shutdown)
	srvHidden := httptest.NewServer(hHidden.Routes())
	defer srvHidden.Close()
	resp, err := srvHidden.Client().Get(srvHidden.URL + "/api/openapi.json")
	if err != nil {
		t.Fatalf("GET openapi.json: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("hidden openapi.json status = %d, want 404", resp.StatusCode)
	}

	// 2) 显式开启 + 配置 token：未带 token 应 401，带 token 应 200。
	h := New(st, quietLogger(), t.TempDir(), nil, "supersecrettokenmustbelongenough", "test", false, true)
	t.Cleanup(h.Shutdown)
	srv := httptest.NewServer(h.Routes())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/openapi.json", nil)
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET openapi.json unauthed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("unauthed openapi.json status = %d, want 401", resp.StatusCode)
	}

	req, _ = http.NewRequest("GET", srv.URL+"/api/openapi.json", nil)
	req.Header.Set("Authorization", "Bearer supersecrettokenmustbelongenough")
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET openapi.json authed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("authed openapi.json status = %d, want 200", resp.StatusCode)
	}
}
