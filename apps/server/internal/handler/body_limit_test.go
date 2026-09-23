package handler

// body_limit_test.go —— docs/archive/review-2026-09-19.md §B9：8 MB 请求体上限与「10 000 key 批量」
// 的公开承诺冲突，且超限被截断后报的是 400「JSON 无效」而不是 413。

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestReadJSONAcceptsDocumentedMaxBatch 10 000 个 1 KB key（≈10.3 MB）必须能被解析：
// 这是 maxBatchKeys 明确允许的批量大小。
func TestReadJSONAcceptsDocumentedMaxBatch(t *testing.T) {
	h := &Handler{log: accDiscardLogger()}
	key := strings.Repeat("k", 1024)
	keys := make([]string, maxBatchKeys)
	for i := range keys {
		keys[i] = key
	}
	body, err := json.Marshal(map[string]any{"keys": keys})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(body) <= 8<<20 {
		t.Fatalf("测试载荷 %d 字节未超过旧上限 8MB，无法复现截断", len(body))
	}
	var req struct {
		Keys []string `json:"keys"`
	}
	if err := h.readJSON(jsonReq(t, body), &req); err != nil {
		t.Fatalf("readJSON(合法 10k 批量) = %v, want nil", err)
	}
	if len(req.Keys) != maxBatchKeys {
		t.Fatalf("keys = %d, want %d", len(req.Keys), maxBatchKeys)
	}
}

// TestReadJSONRejectsOverLimitBody 真正超限时报 errBodyTooLarge（映射为 413），不是语法错误。
func TestReadJSONRejectsOverLimitBody(t *testing.T) {
	h := &Handler{log: accDiscardLogger()}
	body := []byte(`{"keys":["` + strings.Repeat("x", maxBody) + `"]}`)
	var req struct {
		Keys []string `json:"keys"`
	}
	err := h.readJSON(jsonReq(t, body), &req)
	if !errors.Is(err, errBodyTooLarge) {
		t.Fatalf("readJSON(超限) = %v, want errBodyTooLarge", err)
	}
}

// TestReadJSONRejectsMaxBodyPlusOne 合法 JSON 但整体超过上限（正好 maxBody+1 字节）也要按 413 拒绝，
// 而不是「解码成功就放行」。
func TestReadJSONRejectsMaxBodyPlusOne(t *testing.T) {
	h := &Handler{log: accDiscardLogger()}
	const head, tail = `{"key":"`, `"}`
	body := []byte(head + strings.Repeat("x", maxBody+1-len(head)-len(tail)) + tail)
	if len(body) != maxBody+1 {
		t.Fatalf("载荷 %d 字节，want %d", len(body), maxBody+1)
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := h.readJSON(jsonReq(t, body), &req); !errors.Is(err, errBodyTooLarge) {
		t.Fatalf("readJSON(maxBody+1 的合法 JSON) = %v, want errBodyTooLarge", err)
	}
}

// TestDeleteObjectsBodyTooLargeIs413 端点层：超限请求体回 413 + 明确消息。
func TestDeleteObjectsBodyTooLargeIs413(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		return olXML(http.StatusOK, `<?xml version="1.0"?><DeleteResult/>`)
	})
	env := accNewEnv(t, srv.URL, "b")
	body := `{"bucket":"b","keys":["` + strings.Repeat("x", maxBody) + `"]}`
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/delete", body)
	olExpectStatus(t, rr, http.StatusRequestEntityTooLarge, "oversized body")
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "too large") {
		t.Fatalf("error = %v, want a request-body-too-large message", got["error"])
	}
}

// jsonReq 构造带 application/json 的请求（readJSON 会校验 Content-Type）。
func jsonReq(t *testing.T, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/x", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	return req
}
