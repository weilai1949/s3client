package openapi

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
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
