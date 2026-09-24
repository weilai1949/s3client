package handler

// openapi_shape_test.go —— /api/openapi.json 的序列化与顶层形状契约（分节 5–6）：
//   - MarshalJSON 确定（含并发）且 paths/operation 的 JSON key 有序、method 小写；
//   - 顶层形状（openapi 版本 / info / servers / securitySchemes）。
//
// 夹具（newOpenAPIServer / buildOpenAPIBytes 等）与分节 1–4 见 openapi_contract_test.go；
// requestBody ↔ handler 对齐（分节 7）见 openapi_requestbody_test.go。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// ---- 5. 序列化确定性 + key 有序 ----

// TestOpenAPI_ContractMarshalDeterministic 验证两次（以及并发 8×8 次）MarshalJSON 输出字节完全一致。
// 并发部分在 -race 下同时覆盖 Registry 的 RWMutex 保护。
func TestOpenAPI_ContractMarshalDeterministic(t *testing.T) {
	t.Parallel()
	h := newContractHandler(t)

	first, err := h.openapi.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON #1: %v", err)
	}
	second, err := h.openapi.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON #2: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("两次 MarshalJSON 输出不一致：len %d vs %d", len(first), len(second))
	}
	if len(first) == 0 {
		t.Fatal("MarshalJSON 输出为空")
	}

	const workers, rounds = 8, 8
	var wg sync.WaitGroup
	failures := make(chan string, workers*rounds)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < rounds; j++ {
				b, err := h.openapi.MarshalJSON()
				if err != nil {
					failures <- fmt.Sprintf("并发 MarshalJSON: %v", err)
					return
				}
				if !bytes.Equal(b, first) {
					failures <- "并发 MarshalJSON 输出与串行结果不一致"
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for msg := range failures {
		t.Error(msg)
	}
}

// TestOpenAPI_ContractJSONKeysSorted 用 token 流读取原始 JSON（而非 map），
// 验证 paths 对象及其下每个 operation 对象的 key 按字典序输出、method 为小写合法 HTTP 方法。
func TestOpenAPI_ContractJSONKeysSorted(t *testing.T) {
	t.Parallel()
	raw := fetchOpenAPIJSON(t)

	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("decode openapi.json: %v", err)
	}
	pathsRaw, ok := top["paths"]
	if !ok {
		t.Fatal("顶层缺少 paths")
	}
	pathKeys := objectKeyOrder(t, pathsRaw)
	assertSortedKeys(t, "paths", pathKeys)

	var pathValues map[string]json.RawMessage
	if err := json.Unmarshal(pathsRaw, &pathValues); err != nil {
		t.Fatalf("decode paths: %v", err)
	}
	methodCount := 0
	for _, p := range pathKeys {
		opKeys := objectKeyOrder(t, pathValues[p])
		assertSortedKeys(t, "paths."+p, opKeys)
		for _, m := range opKeys {
			methodCount++
			if m != strings.ToLower(m) {
				t.Errorf("paths.%s 的 method %q 必须小写", p, m)
			}
			switch m {
			case "get", "put", "post", "delete", "options", "head", "patch", "trace":
			default:
				t.Errorf("paths.%s 的 method %q 不是合法 HTTP 方法", p, m)
			}
		}
	}
	if methodCount != apiRouteCount {
		t.Errorf("规范 method 数 = %d, want %d", methodCount, apiRouteCount)
	}
}

// objectKeyOrder 返回一个 JSON object 顶层 key 的**出现顺序**。
func objectKeyOrder(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("读取 object 起始 token: %v", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		t.Fatalf("期望 JSON object，实际起始 token = %v", tok)
	}
	var keys []string
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			t.Fatalf("读取 object key: %v", err)
		}
		key, ok := kt.(string)
		if !ok {
			t.Fatalf("object key 类型 = %T, want string", kt)
		}
		keys = append(keys, key)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatalf("跳过 key %q 的值: %v", key, err)
		}
	}
	if _, err := dec.Token(); err != nil { // 消费收尾 '}'
		t.Fatalf("读取 object 收尾 token: %v", err)
	}
	return keys
}

func assertSortedKeys(t *testing.T, where string, keys []string) {
	t.Helper()
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Errorf("%s 的 JSON key 未按字典序输出：%q 出现在 %q 之后（%v）", where, keys[i], keys[i-1], keys)
			return
		}
	}
}

// ---- 6. 顶层形状 ----

// TestOpenAPI_ContractTopLevelShape 验证 OpenAPI 版本、info、servers、components 与 bearerAuth 齐全。
func TestOpenAPI_ContractTopLevelShape(t *testing.T) {
	t.Parallel()
	doc := openAPIDoc(t)

	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v, want 3.0.3", doc["openapi"])
	}

	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatalf("info 类型 = %T, want object", doc["info"])
	}
	for _, k := range []string{"title", "version"} {
		s, _ := info[k].(string)
		if strings.TrimSpace(s) == "" {
			t.Errorf("info.%s 为空（OpenAPI 3.0 要求非空）", k)
		}
	}

	servers, ok := doc["servers"].([]any)
	if !ok || len(servers) == 0 {
		t.Errorf("servers = %v, want 非空数组", doc["servers"])
	}
	for i, raw := range servers {
		sm, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("servers[%d] 类型 = %T, want object", i, raw)
			continue
		}
		if url, _ := sm["url"].(string); strings.TrimSpace(url) == "" {
			t.Errorf("servers[%d].url 为空", i)
		}
	}

	comps, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("components 类型 = %T, want object", doc["components"])
	}
	for _, k := range []string{"schemas", "parameters", "responses", "securitySchemes"} {
		m, ok := comps[k].(map[string]any)
		if !ok || len(m) == 0 {
			t.Errorf("components.%s 缺失或为空", k)
		}
	}
	sec, _ := comps["securitySchemes"].(map[string]any)
	bearer, ok := sec["bearerAuth"].(map[string]any)
	if !ok {
		t.Fatalf("components.securitySchemes.bearerAuth 缺失")
	}
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Errorf("bearerAuth = %v, want {type: http, scheme: bearer}", bearer)
	}
}
