package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestIDBoundary 验证 X-Request-ID 的边界处理（review §6 Nit）：
// 合法值原样透传；超长 / 含控制字符 / 非 ASCII 的值不再回显，改由服务端生成 UUID，
// 且无论如何响应头长度有上限，避免超长值进入日志与响应头。
func TestRequestIDBoundary(t *testing.T) {
	h := newTestHandler(t, nil, "")
	cases := []struct {
		name   string
		send   string
		echoed bool
	}{
		{"valid short", "req-abc-123", true},
		{"valid max length", strings.Repeat("a", maxRequestIDLen), true},
		{"too long", strings.Repeat("a", maxRequestIDLen+1), false},
		{"very long", strings.Repeat("a", 8192), false},
		{"newline injection", "abc\ndef", false},
		{"carriage return", "abc\rdef", false},
		{"space", "abc def", false},
		{"tab", "abc\tdef", false},
		{"non-ascii", "abc-中文", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/health", nil)
			req.Header.Set("X-Request-ID", c.send)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			got := rr.Header().Get("X-Request-ID")
			if got == "" {
				t.Fatal("X-Request-ID missing from response")
			}
			if len(got) > maxRequestIDLen {
				t.Fatalf("echoed X-Request-ID length %d exceeds max %d", len(got), maxRequestIDLen)
			}
			if c.echoed && got != c.send {
				t.Fatalf("X-Request-ID = %q, want passthrough %q", got, c.send)
			}
			if !c.echoed && got == c.send {
				t.Fatalf("invalid X-Request-ID %q must not be echoed", c.send)
			}
		})
	}
}

// TestAuthBearerSchemeCaseInsensitive RFC 7235：auth-scheme 大小写不敏感，
// `bearer`/`BEARER`/`BeArEr` 都应视为 Bearer 并通过鉴权；token 本身仍大小写敏感且常量时间比较。
func TestAuthBearerSchemeCaseInsensitive(t *testing.T) {
	h := newTestHandler(t, nil, "s3cret")
	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "BeArEr", "bEaReR"} {
		t.Run(scheme, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/accounts", nil)
			req.Header.Set("Authorization", scheme+" s3cret")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("scheme %q: status = %d, want 200", scheme, rr.Code)
			}
		})
	}

	// scheme 大小写不敏感不等于 token 大小写不敏感，也不接受非 Bearer scheme / 空凭证。
	for _, bad := range []string{
		"Bearer wrong",
		"bearer wrong",
		"Bearer S3CRET",
		"Basic s3cret",
		"s3cret",
		"Bearer",
		"Bearer ",
		"bearer  s3cret",
	} {
		t.Run("reject "+bad, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/accounts", nil)
			req.Header.Set("Authorization", bad)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("Authorization %q: status = %d, want 401", bad, rr.Code)
			}
		})
	}
}
