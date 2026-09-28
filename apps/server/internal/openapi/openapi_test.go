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
		Params: []Param{{
			Name: "verbose", In: "query", Schema: Bool(),
		}},
		Responses: map[string]Response{
			"200": {Description: "OK", JSON: BuildObj(map[string]*Schema{
				"ok":      Bool(),
				"version": Str(),
			}, "ok", "version")},
		},
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

	// ② 覆盖已存在的 path/method（带 params / responses）→ 渲染进文档且缓存失效。
	r.Operation("POST", "/y", Op{
		Summary:   "y",
		Params:    []Param{{Name: "q", In: "query", Schema: Str()}},
		Responses: map[string]Response{"200": {Description: "OK"}},
	})
	withParam, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON after overwrite: %v", err)
	}
	if !bytes.Contains(withParam, []byte(`"q"`)) || !bytes.Contains(withParam, []byte(`"200"`)) {
		t.Error("覆盖注册后缓存未失效")
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

// resolveJSONPointer 测试内的本地 $ref 解析器：沿 "#/..." 逐段下钻，证明引用真实可解析。
func resolveJSONPointer(t *testing.T, doc map[string]any, ref string) map[string]any {
	t.Helper()
	if !strings.HasPrefix(ref, "#/") {
		t.Fatalf("ref %q 不是本地引用", ref)
	}
	cur := any(doc)
	for _, seg := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("ref %q 的段 %q 父节点不是对象（%T）", ref, seg, cur)
		}
		next, ok := m[seg]
		if !ok {
			t.Fatalf("ref %q 无法解析：段 %q 不存在", ref, seg)
		}
		cur = next
	}
	out, ok := cur.(map[string]any)
	if !ok {
		t.Fatalf("ref %q 解析结果不是对象（%T）", ref, cur)
	}
	return out
}

// responseJSONSchema 从响应实体取出 application/json 的 schema 对象。
func responseJSONSchema(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	content, ok := resp["content"].(map[string]any)
	if !ok {
		t.Fatalf("响应实体缺少 content：%v", resp)
	}
	mt, ok := content["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("content 缺少 application/json：%v", content)
	}
	sch, ok := mt["schema"].(map[string]any)
	if !ok {
		t.Fatalf("application/json 缺少 schema：%v", mt)
	}
	return sch
}

// TestComponentsResponsesKeepSchema（R15b）：components.responses 的实体必须携带
// description 与 content/ schema——Response.JSON / Ref 字段曾被 `json:"-"` 静默丢弃，
// 端点级 $ref 解析到的共享响应是无 schema 空壳，契约 SSOT 失效。
// 断言走「序列化 → 解析 JSON → 沿 $ref 解析到真实 schema」的完整消费路径，不检查 Go 结构体。
func TestComponentsResponsesKeepSchema(t *testing.T) {
	r := New("s3clinet API", "1.0.0-test")
	r.Operation("GET", "/api/x", Op{
		Summary:   "x",
		Responses: map[string]Response{"404": {Ref: "#/components/responses/NotFound"}},
	})
	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// 端点级 $ref 原样输出（复用共享响应，不内联）。
	op := doc["paths"].(map[string]any)["/api/x"].(map[string]any)["get"].(map[string]any)
	entry := op["responses"].(map[string]any)["404"].(map[string]any)
	if ref, _ := entry["$ref"].(string); ref != "#/components/responses/NotFound" {
		t.Fatalf("端点级 404 = %v, want $ref", entry["$ref"])
	}

	// 沿 $ref 解析共享响应实体：有 description，且 JSON schema 是对 Error 组件的引用。
	nf := resolveJSONPointer(t, doc, entry["$ref"].(string))
	if desc, _ := nf["description"].(string); desc == "" {
		t.Errorf("共享响应实体缺少 description：%v", nf)
	}
	schemaRef, _ := responseJSONSchema(t, nf)["$ref"].(string)
	if schemaRef != "#/components/schemas/Error" {
		t.Fatalf("NotFound.schema = %v, want $ref #/components/schemas/Error", responseJSONSchema(t, nf))
	}
	// 再解析一层：schema 引用必须落到真实存在的组件（含 error 属性），不是空壳。
	errSchema := resolveJSONPointer(t, doc, schemaRef)
	if typ, _ := errSchema["type"].(string); typ != "object" {
		t.Errorf("Error schema type = %v, want object", errSchema["type"])
	}
	props, _ := errSchema["properties"].(map[string]any)
	if props["error"] == nil {
		t.Errorf("Error schema 缺少 error 属性：%v", errSchema)
	}

	// 内联 JSON schema 的共享响应（BadRequest）同样带 schema。
	bad := resolveJSONPointer(t, doc, "#/components/responses/BadRequest")
	if typ, _ := responseJSONSchema(t, bad)["type"].(string); typ != "object" {
		t.Errorf("BadRequest.schema type = %v, want object", responseJSONSchema(t, bad)["type"])
	}
	// 未声明 body 的共享响应（Unauthorized）保留 description，不产生空 content。
	un := resolveJSONPointer(t, doc, "#/components/responses/Unauthorized")
	if desc, _ := un["description"].(string); desc == "" {
		t.Errorf("Unauthorized 缺少 description：%v", un)
	}
	if _, has := un["content"]; has {
		t.Errorf("Unauthorized 不应有 content：%v", un)
	}
}
