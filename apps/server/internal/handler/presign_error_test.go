package handler

// presign_error_test.go —— 预签名失败不得被静默吞掉（todolist #23 / ASSESSMENT L1）。
//
// 背景：原实现三处写成 `u, _ := client.PresignXxx(...)`，失败时 url 为空字符串
// 却仍返回 200，调用方无法区分「服务端出错」与「前端没渲染」。
//
// 为什么不靠「取消请求上下文」注入失败：AWS SDK 会在 client 内缓存已解析的凭证，
// 首次调用失败后同一 client 的后续调用不再失败——实测同一进程内 put/post/part
// 的成功与失败取决于调用顺序，据此写的测试会 flaky。故改为：
//   1) 直接单测统一的错误处理函数（行为确定）；
//   2) 用源码级门禁 TestPresignErrorsNotSwallowed 防止 `_, _ := Presign...` 复发。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestWritePresignResultError 预签名失败 → 500，且响应体不含空 url。
func TestWritePresignResultError(t *testing.T) {
	h, _ := gapStoreHandler(t)

	rr := httptest.NewRecorder()
	ok := h.writePresignResult(rr, errTestPresign, map[string]any{
		"method": "put", "bucket": "b", "key": "k", "url": "", "expiresIn": 3600,
	})

	if ok {
		t.Error("writePresignResult should report failure")
	}
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rr.Code, rr.Body)
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("500 body is not JSON: %s", rr.Body)
	}
	// 关键：不得把「空 url 的 200」泄露成任何形式的成功响应。
	if _, hasURL := m["url"]; hasURL {
		t.Errorf("error response must not carry a url field: %s", rr.Body)
	}
	if !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("error response should describe the failure: %s", rr.Body)
	}
}

// TestWritePresignResultSuccess 成功路径不被错误处理破坏：原样写 200 与给定字段。
func TestWritePresignResultSuccess(t *testing.T) {
	h, _ := gapStoreHandler(t)

	rr := httptest.NewRecorder()
	ok := h.writePresignResult(rr, nil, map[string]any{
		"method": "put", "bucket": "b", "key": "k", "url": "https://example.com/sig", "expiresIn": 3600,
	})

	if !ok {
		t.Error("writePresignResult should report success")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body)
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if u, _ := m["url"].(string); u != "https://example.com/sig" {
		t.Errorf("url = %v, want the presigned URL", m["url"])
	}
}

// TestPresignErrorsNotSwallowed 源码级门禁：handler 生产代码中不得再出现
// `u, _ := client.PresignXxx(...)` 这类吞错写法（#23 的复发防护）。
func TestPresignErrorsNotSwallowed(t *testing.T) {
	// 匹配「忽略 error 的预签名调用」：形如 `x, _ := <something>Presign`。
	swallowed := regexp.MustCompile(`,\s*_\s*:=\s*[A-Za-z0-9_.]*Presign`)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		// 只查生产代码：跳过测试文件（测试里允许构造假数据）。
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		checked++
		// 逐行剔除注释后再匹配：文档/注释里说明该反模式属正常，
		// 若直接扫全文会把「注释里举例说明」误判为真实违规。
		for i, line := range strings.Split(string(b), "\n") {
			code := line
			if idx := strings.Index(code, "//"); idx >= 0 {
				code = code[:idx]
			}
			if m := swallowed.FindString(code); m != "" {
				t.Errorf("%s:%d 忽略了预签名错误（会返回 200 + 空 url）：%s", name, i+1, strings.TrimSpace(line))
			}
		}
	}
	if checked == 0 {
		t.Fatal("no production .go files scanned; gate is not working")
	}
}
