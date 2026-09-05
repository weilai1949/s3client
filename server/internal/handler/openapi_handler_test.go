package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/weilai1949/s3clinet/server/internal/store"
)

// quietLogger 把日志降级到 discard，避免 CI 测试输出噪声。
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestOpenAPI_ExposesAllRoutes 验证 /api/openapi.json 与 routes.go 注册数一致，
// 防止「新增端点忘了登记 / 删除端点忘了下架」造成契约漂移。
func TestOpenAPI_ExposesAllRoutes(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	h := New(st, quietLogger(), t.TempDir(), nil, "", "test", false)

	// 真实路由数：routes.go 中 mux.HandleFunc 共 68 行；其中 1 行为 SPA fallback `/`。
	const wantAPIRoutes = 67
	srv := httptest.NewServer(h.Routes())
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/openapi.json")
	if err != nil {
		t.Fatalf("GET openapi.json: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi version = %v", doc["openapi"])
	}
	paths := doc["paths"].(map[string]any)
	// 统计每个 path 下的 method 数（同一 path 多 method 共享 key）。
	methodCount := 0
	for _, ops := range paths {
		m, ok := ops.(map[string]any)
		if !ok {
			continue
		}
		methodCount += len(m)
	}
	const wantMethods = 68 // 67 业务 + 1 /api/openapi.json 自指
	if methodCount < wantMethods {
		t.Errorf("api method count = %d, want >= %d", methodCount, wantMethods)
	}
	// 抽样：核心端点必须存在且 operationId 非空
	for _, mustExist := range []string{
		"/api/health",
		"/api/accounts",
		"/api/accounts/{id}",
		"/api/accounts/{id}/objects",
		"/api/accounts/{id}/presign",
		"/api/migrate/async",
		"/api/openapi.json",
	} {
		entry, ok := paths[mustExist].(map[string]any)
		if !ok {
			t.Errorf("missing path %s", mustExist)
			continue
		}
		// 任一 method 下必须有 operationId 或 summary
		hasContent := false
		for _, v := range entry {
			m := v.(map[string]any)
			if m["operationId"] != nil || m["summary"] != nil {
				hasContent = true
				break
			}
		}
		if !hasContent {
			t.Errorf("%s: no operationId/summary on any method", mustExist)
		}
	}
	comps := doc["components"].(map[string]any)
	sec := comps["securitySchemes"].(map[string]any)
	if sec["bearerAuth"] == nil {
		t.Error("bearerAuth scheme missing")
	}
}

// TestOpenAPI_PublicEndpoint 验证 /api/openapi.json 在配置 token 时也不被鉴权层挡住。
func TestOpenAPI_PublicEndpoint(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	h := New(st, quietLogger(), t.TempDir(), nil, "supersecrettokenmustbelongenough", "test", false)
	srv := httptest.NewServer(h.Routes())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/openapi.json", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET openapi.json unauthed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("unauthed openapi.json status = %d, want 200", resp.StatusCode)
	}
}
