package handler

// accounts_plaintext_test.go —— A5（2026-10-10，威胁模型 §6.2 的「明文 http:// 载荷完整性」残留）：
//
// 残留风险本体只能靠 TLS 消除（`useSSL=true` 或反向代理终止 TLS），本测试**不要求**服务端拦截
// ——自托管刚需下拦截属行为变更。它钉的是另一半：**不能让风险无感知**。此前
// `S3C_ALLOW_PLAINTEXT_STORE=1` 有启动 WARN（`cfg.StorePlaintextWarning()`），
// 而新建 / 更新一个 `http://` 账号端点时**零提示**，运维要到读威胁模型才知道。
//
// 断言的外部可见行为：HTTP 201 / 200 之外，服务端日志必须出现一条 WARN，且
// **不含任何密钥**（根 AGENTS.md「SecretKey 不落日志」）；HTTPS 端点则不得出现该 WARN
// （否则告警噪声会让人忽略真正的明文告警）。

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3client/apps/server/internal/store"
)

// plaintextWarnMarker 是 WARN 消息里必含的稳定片段（改动文案需同步本测试）。
const plaintextWarnMarker = "载荷完整性"

// newCapturingHandler 构造带**可捕获日志**的被测 Handler 与请求体编码器。
func newCapturingHandler(t *testing.T, buf *bytes.Buffer) http.Handler {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	return New(st, slog.New(slog.NewJSONHandler(buf, nil)), t.TempDir(), nil, "", "test", false, false).Routes()
}

// accountBody 编码账号创建 / 更新请求体（`secretKey` 用可搜索的哨兵值）。
func accountBody(t *testing.T, endpoint, publicEndpoint string, useSSL bool) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"name":           "acc-plain",
		"endpoint":       endpoint,
		"publicEndpoint": publicEndpoint,
		"accessKey":      "AKIATEST",
		"secretKey":      "SECRET-SENTINEL",
		"region":         "us-east-1",
		"pathStyle":      true,
		"useSSL":         useSSL,
	})
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}
	return string(b)
}

// TestCreateAccountWarnsOnPlaintextEndpoint 断言创建明文端点账号时打 WARN、
// TLS 端点不打，且日志里绝不出现密钥。
func TestCreateAccountWarnsOnPlaintextEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		useSSL   bool
		wantWarn bool
	}{
		{"显式 http://", "http://127.0.0.1:9000", false, true},
		{"scheme 大小写不敏感", "HTTP://127.0.0.1:9000", false, true},
		{"裸端点 + useSSL=false（默认 http）", "127.0.0.1:9000", false, true},
		{"显式 https://", "https://s3.example.com", false, false},
		{"裸端点 + useSSL=true", "s3.example.com", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := newCapturingHandler(t, &buf)
			rr := doJSON(t, h, http.MethodPost, "/api/accounts",
				accountBody(t, tc.endpoint, "", tc.useSSL))
			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201（本用例只关心日志，不改行为）：\n%s", rr.Code, rr.Body.String())
			}
			got := strings.Contains(buf.String(), plaintextWarnMarker)
			if got != tc.wantWarn {
				t.Errorf("明文告警 = %v, want %v；日志：\n%s", got, tc.wantWarn, buf.String())
			}
			if strings.Contains(buf.String(), "SECRET-SENTINEL") {
				t.Errorf("日志泄漏了 secretKey：\n%s", buf.String())
			}
		})
	}
}

// TestUpdateAccountWarnsOnPlaintextEndpoint 断言**更新**路径同样有告知
// （编辑既有账号把 TLS 关掉，是明文端点最常见的进入方式）。
func TestUpdateAccountWarnsOnPlaintextEndpoint(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(t, &buf)

	create := doJSON(t, h, http.MethodPost, "/api/accounts",
		accountBody(t, "https://s3.example.com", "", false))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d：\n%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("解析创建响应 id 失败（err=%v）：\n%s", err, create.Body.String())
	}
	buf.Reset()

	rr := doJSON(t, h, http.MethodPut, "/api/accounts/"+created.ID,
		accountBody(t, "http://127.0.0.1:9000", "", false))
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200：\n%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(buf.String(), plaintextWarnMarker) {
		t.Errorf("更新为明文端点时未打 WARN：\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "SECRET-SENTINEL") {
		t.Errorf("日志泄漏了 secretKey：\n%s", buf.String())
	}
}

// TestCreateAccountWarnsOnPlaintextPublicEndpoint 断言浏览器直传用的
// `publicEndpoint` 明文同样有告知——预签名 PUT 走的是它，明文链路下直传载荷同样无完整性保护。
func TestCreateAccountWarnsOnPlaintextPublicEndpoint(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(t, &buf)
	rr := doJSON(t, h, http.MethodPost, "/api/accounts",
		accountBody(t, "https://s3.example.com", "http://127.0.0.1:9000", false))
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d：\n%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(buf.String(), plaintextWarnMarker) {
		t.Errorf("publicEndpoint 为明文时未打 WARN：\n%s", buf.String())
	}
}
