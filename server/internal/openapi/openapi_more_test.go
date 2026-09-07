package openapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestSetInfo 覆盖 SetInfo 的整体路径（锁 + 赋值 + 后续序列化反映的 Info）。
func TestSetInfo(t *testing.T) {
	r := New("old", "1.0.0")
	r.SetInfo(Info{Title: "new", Version: "2.0.0", Description: "desc"})
	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatalf("info missing or wrong type: %T", doc["info"])
	}
	if info["title"] != "new" || info["version"] != "2.0.0" || info["description"] != "desc" {
		t.Errorf("info = %v", info)
	}
}

// TestRespond 覆盖 Respond 的两个分支：首次创建 Responses map 与后续追加。
func TestRespond(t *testing.T) {
	r := New("t", "1.0")
	// 首次 Respond：op.Responses 为 nil，进入创建 map 分支。
	b := r.Operation("POST", "/a", Op{Summary: "a"}).Respond("201", Response{Description: "created", JSON: Obj()})
	// 链式再次 Respond：op.Responses 已非 nil，进入追加分支。
	b.Respond("200", Response{Description: "ok"})

	out, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	paths := doc["paths"].(map[string]any)
	post := paths["/a"].(map[string]any)["post"].(map[string]any)
	resps := post["responses"].(map[string]any)
	if resps["201"] == nil {
		t.Error("first Respond swallowed: 201 missing")
	}
	if resps["200"] == nil {
		t.Error("second Respond swallowed: 200 missing")
	}
}

// TestRenderOp_Full 覆盖 renderOp 的所有非空 / 真分支。
func TestRenderOp_Full(t *testing.T) {
	op := Op{
		Summary:     "s",
		Description: "d",
		OperationID: "oid",
		Tags:        []string{"t1", "t2"},
		Deprecated:  true,
		Params: []Param{
			{Name: "id", In: "path", Required: true, Schema: Str()},
			{Name: "q", In: "query", Description: "opt"},
		},
		Request: &Request{Required: true, Content: MediaType{Schema: Str()}},
		Responses: map[string]Response{
			"200": {Description: "ok", JSON: Obj()},
		},
	}
	out := renderOp(op)
	if out["summary"] != "s" {
		t.Errorf("summary = %v", out["summary"])
	}
	if out["description"] != "d" {
		t.Errorf("description = %v", out["description"])
	}
	if out["operationId"] != "oid" {
		t.Errorf("operationId = %v", out["operationId"])
	}
	if tg := out["tags"].([]string); len(tg) != 2 {
		t.Errorf("tags = %v", out["tags"])
	}
	if out["deprecated"] != true {
		t.Errorf("deprecated = %v", out["deprecated"])
	}
	if out["parameters"] == nil {
		t.Error("parameters missing")
	}
	rb, ok := out["requestBody"].(map[string]any)
	if !ok {
		t.Fatalf("requestBody missing or wrong type: %T", out["requestBody"])
	}
	if rb["required"] != true {
		t.Errorf("requestBody.required = %v", rb["required"])
	}
	if rb["content"] == nil {
		t.Error("requestBody.content missing")
	}
	if _, ok := out["responses"].(map[string]any); !ok {
		t.Error("responses missing")
	}
}

// TestRenderOp_Empty 覆盖 renderOp 的所有空 / 假分支（空 Op、nil Request、空 Responses）。
func TestRenderOp_Empty(t *testing.T) {
	out := renderOp(Op{})
	if len(out) != 0 {
		t.Errorf("empty op produced keys: %v", out)
	}

	// Request 非 nil 但 Content 的 Schema 为 nil：覆盖 renderMedia 的空分支，同时不给 requestBody 造成 panic。
	opNilContent := Op{
		Request: &Request{Required: false},
	}
	out2 := renderOp(opNilContent)
	rb := out2["requestBody"].(map[string]any)
	if rb["required"] != false {
		t.Errorf("requestBody.required = %v", rb["required"])
	}
	if rb["content"] == nil {
		t.Error("requestBody.content key missing (should be empty map)")
	}
}

// TestRenderParams 覆盖 required 真假与 schema 有无。
func TestRenderParams(t *testing.T) {
	ps := []Param{
		{Name: "a", In: "path", Required: true, Schema: Str()},
		{Name: "b", In: "query", Description: "no schema", Required: false},
	}
	out := renderParams(ps)
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	first := out[0]
	if first["required"] != true {
		t.Errorf("first.required = %v", first["required"])
	}
	if first["schema"] == nil {
		t.Error("first.schema missing")
	}
	second := out[1]
	if _, ok := second["required"]; ok {
		t.Errorf("second.required should be omitted, got %v", second["required"])
	}
	if _, ok := second["schema"]; ok {
		t.Errorf("second.schema should be omitted, got %v", second["schema"])
	}
}

// TestRenderMedia 覆盖 schema 有无两个分支。
func TestRenderMedia(t *testing.T) {
	withSchema := renderMedia(MediaType{Schema: Obj()})
	ct, ok := withSchema["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("with schema missing application/json: %v", withSchema)
	}
	if ct["schema"] == nil {
		t.Error("application/json.schema missing")
	}

	without := renderMedia(MediaType{})
	if len(without) != 0 {
		t.Errorf("without schema should be empty, got %v", without)
	}
}

// TestHTTPHandler_Error 覆盖 MarshalJSON 出错时的 500 分支。
func TestHTTPHandler_Error(t *testing.T) {
	r := New("t", "1.0")
	// Default any 中放入不可序列化的 func，令 json.Marshal 失败。
	r.Operation("GET", "/x", Op{
		Params: []Param{{Name: "p", In: "query", Schema: &Schema{Type: "string", Default: func() {}}}},
	})
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	rr := httptest.NewRecorder()
	r.HTTPHandler().ServeHTTP(rr, req)
	if rr.Code != 500 {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Error("expected an error body")
	}
}

// TestProp 覆盖 required 有无，以及 desc 是否已被设置。
func TestProp(t *testing.T) {
	// desc 为空，无 required —— 触发 s.Description == "" 分支。
	name1, s1 := Prop("a", "aaa", Str())
	if name1 != "a" {
		t.Errorf("name1 = %q", name1)
	}
	if s1.Description != "aaa" {
		t.Errorf("s1.Description = %q", s1.Description)
	}

	// desc 已设置 —— 跳过覆盖。
	_, s2 := Prop("b", "bbb", &Schema{Type: "string", Description: "existing"})
	if s2.Description != "existing" {
		t.Errorf("s2.Description = %q", s2.Description)
	}

	// 带 required —— 触发 len(required) > 0 分支。返回值与名称不受影响。
	_, s3 := Prop("c", "ccc", Str(), true)
	if s3.Description != "ccc" {
		t.Errorf("s3.Description = %q", s3.Description)
	}
}
