package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

func gapLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func gapStoreHandler(t *testing.T) (*Handler, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	return New(st, gapLogger(), t.TempDir(), nil, "", "test", false, false), st
}

// SetCSPConnectSrc 覆盖分支（handler.go 70-73）＋ withSecurityHeaders 默认值分支（middleware.go 24-26）。
func TestCSPOverrideAndDefaultBranch(t *testing.T) {
	h, _ := gapStoreHandler(t)

	// 非空 src → 覆盖生效（覆盖 handler.go 70-73）
	h.SetCSPConnectSrc("https://oss.example.com")
	srv := httptest.NewServer(h.Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	_ = body
	if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "https://oss.example.com") {
		t.Fatalf("CSP after override = %q, want custom src", csp)
	}

	// 置空 → 走默认 connect-src 分支（middleware.go 24-26）
	h.cspConnectSrc = ""
	resp2, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("get2: %v", err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	if csp := resp2.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "'self' http://127.0.0.1:* http://localhost:*") {
		t.Fatalf("CSP default = %q, want built-in default", csp)
	}
}

// desc：nil 直接返回 nil，非 nil 设置 Description（openapi_register.go 40-42）。
func TestDescNilAndSet(t *testing.T) {
	if d := desc(nil, "x"); d != nil {
		t.Fatalf("desc(nil) = %v, want nil", d)
	}
	s := openapi.Str("string")
	got := desc(s, "hello")
	if got != s || s.Description != "hello" {
		t.Fatalf("desc set: got %+v (description %q)", got, s.Description)
	}
}

// clientIP XFF 分支（ratelimit.go）：可信代理时带逗号取第一个、无逗号整体 trim。
func TestClientIPXFFBranches(t *testing.T) {
	trusted := []string{"10.0.0.1"}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if got := clientIPWithProxies(r, trusted); got != "1.2.3.4" {
		t.Fatalf("clientIP(comma) = %q, want 1.2.3.4", got)
	}

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.RemoteAddr = "10.0.0.1:1234"
	r2.Header.Set("X-Forwarded-For", "  1.2.3.4  ")
	if got := clientIPWithProxies(r2, trusted); got != "1.2.3.4" {
		t.Fatalf("clientIP(single) = %q, want 1.2.3.4", got)
	}
}

// proxyObject 拒绝含控制字符的 key（proxy.go 37-40）。
func TestProxyRejectsControlChars(t *testing.T) {
	h, st := gapStoreHandler(t)
	acc, err := st.Create(&model.Account{
		Name: "a", Endpoint: "http://127.0.0.1:1", AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	req := httptest.NewRequest("GET", "/api/accounts/"+acc.ID+"/proxy?bucket=b&key=bad%0Akey&mode=download", nil)
	rr := httptest.NewRecorder()
	h.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("control-char key = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "control characters") {
		t.Fatalf("body = %q, want control-character message", rr.Body.String())
	}
}

// syncHandler 全部错误分支（migrate_sync.go 35-71）。
func TestSyncHandlerErrorPaths(t *testing.T) {
	h, st := gapStoreHandler(t)
	okAcc, err := st.Create(&model.Account{
		Name: "ok", Endpoint: "http://127.0.0.1:1", AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create ok: %v", err)
	}
	// 无密钥 → clients.get 失败 → 400 invalid account configuration
	badAcc, err := st.Create(&model.Account{Name: "bad"})
	if err != nil {
		t.Fatalf("create bad: %v", err)
	}
	// 无桶 → bucketOr 失败 → 400 bucket is required
	noBucketAcc, err := st.Create(&model.Account{
		Name: "nb", Endpoint: "http://127.0.0.1:1", AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create nobucket: %v", err)
	}

	body := func(s string) func() *http.Request {
		return func() *http.Request {
			req := httptest.NewRequest("POST", "/api/migrate/sync", strings.NewReader(s))
			req.Header.Set("Content-Type", "application/json")
			return req
		}
	}

	cases := []struct {
		name string
		req  *http.Request
		want int
		msg  string
	}{
		{"bad json", body(`{`)(), 400, "invalid request body"},
		{"missing accounts", body(`{"sourceBucket":"a","targetBucket":"b"}`)(), 400, "sourceAccountId and targetAccountId are required"},
		{"source not found", body(`{"sourceAccountId":"missing","targetAccountId":"` + okAcc.ID + `"}`)(), 404, "source account not found"},
		{"target not found", body(`{"sourceAccountId":"` + okAcc.ID + `","targetAccountId":"missing"}`)(), 404, "target account not found"},
		{"bad source config", body(`{"sourceAccountId":"` + badAcc.ID + `","targetAccountId":"` + okAcc.ID + `"}`)(), 400, "invalid source account configuration"},
		{"bad target config", body(`{"sourceAccountId":"` + okAcc.ID + `","targetAccountId":"` + badAcc.ID + `"}`)(), 400, "invalid target account configuration"},
		{"src bucket required", body(`{"sourceAccountId":"` + noBucketAcc.ID + `","targetAccountId":"` + okAcc.ID + `"}`)(), 400, "bucket is required"},
		{"dst bucket required", body(`{"sourceAccountId":"` + okAcc.ID + `","targetAccountId":"` + noBucketAcc.ID + `"}`)(), 400, "bucket is required"},
	}
	for _, c := range cases {
		rr := httptest.NewRecorder()
		h.syncHandler(rr, c.req)
		if rr.Code != c.want {
			t.Errorf("%s: status = %d, want %d", c.name, rr.Code, c.want)
		}
		if !strings.Contains(rr.Body.String(), c.msg) {
			t.Errorf("%s: body = %q, want contains %q", c.name, rr.Body.String(), c.msg)
		}
	}
}
