package openapi

// openapi_examples_test.go —— SetExamples / Ex 的错误分支与示例渲染（覆盖率门禁要求 100% 语句）。
//
// 背景：示例登记是「静态数据 + 注册表查找」，happy path 由 handler 的 applyExamples 覆盖；
// 但**拼错 method+path / 状态码**这类真实会发生的错误必须显式断言返回 error（而不是静默丢示例），
// Ex 的非法 JSON panic 同样要在本包钉住。

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSetExamplesRendersRequestAndResponseExamples 断言示例被渲染到 requestBody 与响应 content，
// 且 ContentTypes 能把响应示例挂到非 JSON 媒体类型下。
func TestSetExamplesRendersRequestAndResponseExamples(t *testing.T) {
	r := New("t", "1")
	r.Operation("POST", "/x", Op{
		Request:   &Request{Required: true, Content: MediaType{Schema: Obj()}},
		Responses: map[string]Response{"200": {Description: "ok"}},
	})
	if err := r.SetExamples("POST", "/x", OpExample{
		Request:      json.RawMessage(`{"a":1}`),
		Responses:    map[string]json.RawMessage{"200": json.RawMessage(`"stream"`)},
		ContentTypes: map[string]string{"200": "text/plain"},
	}); err != nil {
		t.Fatalf("SetExamples: %v", err)
	}

	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	out := string(b)
	// map 序列化 key 有序，故只断言与顺序无关的片段。
	for _, want := range []string{
		`"requestBody":{`,
		`"application/json":{"example":{"a":1},"schema":{"type":"object"}}`,
		`"text/plain":{"example":"stream"}`,
		`"description":"ok"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("规范缺少渲染片段 %s\n实际：%s", want, out)
		}
	}
}

// TestSetExamplesRejectsUnknownTargets 断言每一条「示例对不上注册表」的路径都返回 error。
func TestSetExamplesRejectsUnknownTargets(t *testing.T) {
	newRegistry := func() *Registry {
		r := New("t", "1")
		r.Operation("GET", "/a", Op{Responses: map[string]Response{"200": {Description: "ok"}}})
		return r
	}
	okResp := map[string]json.RawMessage{"200": json.RawMessage(`{}`)}

	cases := []struct {
		name       string
		method     string
		path       string
		ex         OpExample
		wantErrSub string
	}{
		{"未知 path", "GET", "/missing", OpExample{Responses: okResp}, "未注册的 path"},
		{"未知 method", "POST", "/a", OpExample{Responses: okResp}, "未注册的 operation"},
		{"无 requestBody 却登记请求示例", "GET", "/a", OpExample{Request: json.RawMessage(`{}`)}, "无 requestBody"},
		{"未知状态码", "GET", "/a", OpExample{Responses: map[string]json.RawMessage{"201": json.RawMessage(`{}`)}}, "未注册响应 201"},
		{"ContentTypes 指向未提供示例的状态码", "GET", "/a", OpExample{ContentTypes: map[string]string{"200": "text/plain"}}, "ContentTypes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := newRegistry().SetExamples(tc.method, tc.path, tc.ex)
			if err == nil {
				t.Fatalf("SetExamples(%s %s) 期望 error，实际 nil", tc.method, tc.path)
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Errorf("error = %q，期望包含 %q", err, tc.wantErrSub)
			}
		})
	}
}

// TestExPanicsOnInvalidJSON 断言非法示例在构造期立刻 panic（静态数据写错必须可见）。
func TestExPanicsOnInvalidJSON(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Ex 对非法 JSON 未 panic")
		}
	}()
	Ex("{not json")
}

// TestAddTagAndNoAuthRenderTopLevel 覆盖两条仅由 handler 注册表触达的渲染分支：
// 顶层 tags 声明与 NoAuth operation 的 `security: []` 覆盖。
// （CI 覆盖率门禁按包内测试计量，跨包调用不计入，故在本包显式覆盖。）
func TestAddTagAndNoAuthRenderTopLevel(t *testing.T) {
	r := New("t", "1")
	r.AddTag(Tag{Name: "accounts", Description: "账号"})
	r.Operation("GET", "/api/health", Op{
		NoAuth:    true,
		Tags:      []string{"accounts"},
		Responses: map[string]Response{"200": {Description: "ok", JSON: Obj()}},
	})

	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	out := string(b)
	for _, want := range []string{
		`"tags":[{"name":"accounts","description":"账号"}]`,
		`"security":[]`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("规范缺少片段 %s\n实际：%s", want, out)
		}
	}
}
