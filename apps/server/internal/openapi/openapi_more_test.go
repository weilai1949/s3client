package openapi

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
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
	out := renderOp(op, nil)
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
	out := renderOp(Op{}, nil)
	if len(out) != 0 {
		t.Errorf("empty op produced keys: %v", out)
	}

	// Request 非 nil 但 Content 的 Schema 为 nil：覆盖 renderMedia 的空分支，同时不给 requestBody 造成 panic。
	opNilContent := Op{
		Request: &Request{Required: false},
	}
	out2 := renderOp(opNilContent, nil)
	rb := out2["requestBody"].(map[string]any)
	if rb["required"] != false {
		t.Errorf("requestBody.required = %v", rb["required"])
	}
	if rb["content"] == nil {
		t.Error("requestBody.content key missing (should be empty map)")
	}
}

// TestDescribeOp 覆盖合成说明的各分支：显式 description 优先、摘要 / 分组 / 2xx 三个来源的组合，
// 以及「三处都为空 → 不产生 description 键」的兜底。
func TestDescribeOp(t *testing.T) {
	tagDesc := map[string]string{"objects": "对象：列举 / 复制 / 删除（docs/api.md「对象」）"}

	// 显式 Description 优先，不被合成内容覆盖。
	explicit := renderOp(Op{Summary: "s", Description: "手写说明", Tags: []string{"objects"}}, tagDesc)
	if explicit["description"] != "手写说明" {
		t.Errorf("显式 description 被覆盖：%v", explicit["description"])
	}

	// 摘要 + 分组 + 2xx（201 与 200 升序；500 / default 不计入）。
	full := renderOp(Op{
		Summary: "上传对象",
		Tags:    []string{"objects"},
		Responses: map[string]Response{
			"201": {Description: "created"}, "200": {Description: "ok"},
			"500": {Description: "err"}, "default": {Description: "other"},
		},
	}, tagDesc)
	want := "上传对象。分组：对象：列举 / 复制 / 删除（docs/api.md「对象」）。成功状态码：200 / 201。"
	if full["description"] != want {
		t.Errorf("合成 description = %q, want %q", full["description"], want)
	}

	// 无 tag、只有摘要：不因缺分组说明而失败。
	if got := describeOp(Op{Summary: "仅摘要"}, nil); got != "仅摘要。" {
		t.Errorf("summary-only = %q", got)
	}
	// tag 存在但未声明说明 / 只有非 2xx 响应：只剩摘要。
	if got := describeOp(Op{Summary: "s", Tags: []string{"objects"}, Responses: map[string]Response{"404": {}}}, nil); got != "s。" {
		t.Errorf("tag-without-description = %q", got)
	}
	// 三处皆空：返回空串（renderOp 因此不写 description 键）。
	if got := describeOp(Op{}, nil); got != "" {
		t.Errorf("empty op description = %q, want empty", got)
	}
	if _, ok := renderOp(Op{Tags: []string{"objects"}}, nil)["description"]; ok {
		t.Error("空 summary / 无 responses 的 op 不应产生 description 键")
	}
	// successCodes 的非数字状态码分支。
	if codes := successCodes(map[string]Response{"default": {}}); len(codes) != 0 {
		t.Errorf("successCodes(default) = %v, want empty", codes)
	}
}

// TestForEachOperationRewritesEveryOp 覆盖 ForEachOperation 的整体路径：遍历全部已注册
// operation（多 path × 多 method）、把 f 的返回值写回、并使已缓存的规范失效。
// 该函数是 handler.applyUniversalResponses 的唯一入口（84 个 operation 的通用状态码靠它统一补挂），
// 此前包内无测试覆盖其函数体——`make test-cover` 的 `count==0` 检查会红灯。
func TestForEachOperationRewritesEveryOp(t *testing.T) {
	r := New("t", "1.0.0")
	r.Operation("get", "/a", Op{Summary: "a"})
	r.Operation("post", "/a", Op{Summary: "a2"})
	r.Operation("get", "/b", Op{Summary: "b"})

	// 先 marshal 一次，让规范进入缓存，验证 ForEachOperation 会置空缓存。
	if _, err := r.MarshalJSON(); err != nil {
		t.Fatalf("marshal: %v", err)
	}

	seen := map[string]bool{}
	r.ForEachOperation(func(method, path string, op Op) Op {
		seen[method+" "+path] = true
		op.Summary += "!"
		return op
	})
	if len(seen) != 3 {
		t.Fatalf("遍历到 %d 个 operation，期望 3（%v）", len(seen), seen)
	}
	for _, want := range []string{"GET /a", "POST /a", "GET /b"} {
		if !seen[want] {
			t.Errorf("未遍历到 %s（method 应已大写归一）", want)
		}
	}

	// 返回值必须写回且缓存已失效：重新 marshal 能读到改写结果。
	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		t.Fatal("规范缺少 paths")
	}
	for _, p := range []string{"/a", "/b"} {
		item, _ := paths[p].(map[string]any)
		if item == nil {
			t.Fatalf("paths.%s 缺失", p)
		}
		for _, m := range []string{"get", "post"} {
			op, _ := item[m].(map[string]any)
			if op == nil {
				continue
			}
			if s, _ := op["summary"].(string); !strings.HasSuffix(s, "!") {
				t.Errorf("%s %s 的 summary = %q，未写回 f 的返回值", m, p, s)
			}
		}
	}
}

// TestRenderParams 覆盖 required 真假、schema 有无，以及 $ref 参数分支。
func TestRenderParams(t *testing.T) {
	ps := []Param{
		{Name: "a", In: "path", Required: true, Schema: Str()},
		{Name: "b", In: "query", Description: "no schema", Required: false},
		{Ref: "#/components/parameters/AccountID"},
	}
	out := renderParams(ps)
	if len(out) != 3 {
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
	third := out[2]
	if ref, _ := third["$ref"].(string); ref != "#/components/parameters/AccountID" {
		t.Errorf("third.$ref = %q", ref)
	}
	if len(third) != 1 {
		t.Errorf("third 应只含 $ref，got %v", third)
	}
}

// TestRenderResponsesRef 覆盖 renderResponses 的 $ref 与内联两个分支。
func TestRenderResponsesRef(t *testing.T) {
	rs := map[string]Response{
		"404": {Ref: "#/components/responses/NotFound"},
		"200": {Description: "OK", JSON: Obj()},
	}
	out := renderResponses(rs)
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	nf := out["404"].(map[string]any)
	if ref, _ := nf["$ref"].(string); ref != "#/components/responses/NotFound" {
		t.Errorf("404.$ref = %q", ref)
	}
	if len(nf) != 1 {
		t.Errorf("404 应只含 $ref，got %v", nf)
	}
	ok := out["200"].(map[string]any)
	if desc, _ := ok["description"].(string); desc != "OK" {
		t.Errorf("200.description = %q", desc)
	}
	if ok["content"] == nil {
		t.Error("200.content missing")
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
