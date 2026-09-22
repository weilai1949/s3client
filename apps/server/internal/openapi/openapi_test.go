package openapi

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestRegistry_Roundtrip(t *testing.T) {
	r := New("s3clinet API", "1.0.0-test")
	r.AddServer(Server{URL: "/"})
	r.Operation("GET", "/api/health", Op{
		Tags:        []string{"system"},
		Summary:     "健康检查",
		OperationID: "health",
		Responses: map[string]Response{
			"200": {Description: "OK", JSON: BuildObj(map[string]*Schema{
				"ok":      Bool(),
				"version": Str(),
			}, "ok", "version")},
		},
	}).Param(Param{
		Name: "verbose", In: "query", Schema: Bool(),
	})

	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi version = %v", doc["openapi"])
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths missing or wrong type: %T", doc["paths"])
	}
	health, ok := paths["/api/health"].(map[string]any)
	if !ok {
		t.Fatalf("health path missing")
	}
	get := health["get"].(map[string]any)
	if get["operationId"] != "health" {
		t.Errorf("operationId = %v", get["operationId"])
	}
	if get["summary"] != "健康检查" {
		t.Errorf("summary = %v", get["summary"])
	}
	// param 保留
	ps := get["parameters"].([]any)
	if len(ps) != 1 {
		t.Errorf("params len = %d", len(ps))
	}
	// responses
	resps := get["responses"].(map[string]any)
	if resps["200"] == nil {
		t.Errorf("200 missing")
	}
	// 顶层含 components
	if doc["components"] == nil {
		t.Error("components missing")
	}
}

func TestRegistry_HTTPHandler(t *testing.T) {
	r := New("test", "0.0.0")
	r.Operation("GET", "/x", Op{Summary: "x"})
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	rr := httptest.NewRecorder()
	r.HTTPHandler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q", ct)
	}
	if !strings.Contains(rr.Body.String(), `"openapi":"3.0.3"`) {
		t.Errorf("body missing openapi version: %s", rr.Body.String())
	}
}

// TestRegistry_MarshalJSONCached（P3）：重复 marshal 命中缓存返回相同字节，
// 且调用方改写返回的切片不得污染缓存（否则第二个请求会读到被改坏的内容）。
func TestRegistry_MarshalJSONCached(t *testing.T) {
	r := New("test", "0.0.0")
	r.Operation("GET", "/x", Op{Summary: "x"})
	first, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON #1: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("MarshalJSON 返回空")
	}
	// 命中缓存后必须返回等值内容。
	second, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON #2: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("缓存命中时两次输出应完全一致")
	}
	// 调用方改写返回值：后续读取必须不受影响（返回的是副本）。
	first[0] = 'X'
	third, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON #3: %v", err)
	}
	if third[0] == 'X' {
		t.Error("MarshalJSON 返回的切片必须与缓存隔离（调用方改写污染了缓存）")
	}
}

// TestRegistry_MarshalJSONCacheInvalidated（P3）：任何注册动作都必须让缓存失效，
// 否则新增端点后 /api/openapi.json 会一直吐旧契约。
func TestRegistry_MarshalJSONCacheInvalidated(t *testing.T) {
	r := New("test", "0.0.0")
	r.Operation("GET", "/x", Op{Summary: "x"})
	before, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	// ① 新增 operation → 新 path 必须出现。
	r.Operation("POST", "/y", Op{Summary: "y"})
	after, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON after Operation: %v", err)
	}
	if bytes.Equal(before, after) || !bytes.Contains(after, []byte(`"/y"`)) {
		t.Error("新增 operation 后缓存未失效")
	}

	// ② Param / Respond 链式追加 → 也必须失效。
	r.Operation("GET", "/z", Op{Summary: "z"}).Param(Param{Name: "q", In: "query", Schema: Str()}).
		Respond("200", Response{Description: "OK"})
	withParam, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON after Param/Respond: %v", err)
	}
	if !bytes.Contains(withParam, []byte(`"q"`)) || !bytes.Contains(withParam, []byte(`"200"`)) {
		t.Error("Param/Respond 追加后缓存未失效")
	}

	// ③ SetInfo / AddServer → 也必须失效。
	r.SetInfo(Info{Title: "changed", Version: "9.9.9"})
	afterInfo, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON after SetInfo: %v", err)
	}
	if !bytes.Contains(afterInfo, []byte(`"9.9.9"`)) {
		t.Error("SetInfo 后缓存未失效")
	}
	r.AddServer(Server{URL: "https://example.test"})
	afterSrv, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON after AddServer: %v", err)
	}
	if !bytes.Contains(afterSrv, []byte("https://example.test")) {
		t.Error("AddServer 后缓存未失效")
	}
}

// TestRegistry_MarshalJSONConcurrent 并发 marshal 与注册交错时不得出现数据竞争，
// 且每次返回的文档都是自洽的 JSON。
func TestRegistry_MarshalJSONConcurrent(t *testing.T) {
	r := New("test", "0.0.0")
	r.Operation("GET", "/x", Op{Summary: "x"})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				b, err := r.MarshalJSON()
				if err != nil {
					t.Errorf("并发 MarshalJSON: %v", err)
					return
				}
				if !json.Valid(b) {
					t.Errorf("并发 MarshalJSON 输出非法 JSON")
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 20; j++ {
			r.Operation("GET", "/dyn", Op{Summary: "dyn"})
		}
	}()
	wg.Wait()
}

func TestBuildObj_RequiredOrdering(t *testing.T) {
	s := BuildObj(map[string]*Schema{
		"a": Str(),
		"b": Str(),
		"c": Str(),
	}, "c", "a")

	if len(s.Required) != 2 {
		t.Fatalf("required len = %d", len(s.Required))
	}
	if s.Required[0] != "a" || s.Required[1] != "c" {
		t.Errorf("required not sorted: %v", s.Required)
	}
	// required 为空时省略
	s2 := BuildObj(map[string]*Schema{"a": Str()})
	if s2.Required != nil {
		t.Errorf("required should be nil when empty, got %v", s2.Required)
	}
}

func TestEnumStr(t *testing.T) {
	s := EnumStr("AES256", "aws:kms", "aws:kms:dsse")
	if len(s.Enum) != 3 {
		t.Errorf("enum len = %d", len(s.Enum))
	}
	if s.Type != "string" {
		t.Errorf("type = %q", s.Type)
	}
}
