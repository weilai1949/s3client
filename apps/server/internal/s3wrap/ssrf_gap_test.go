package s3wrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIsBlockedIPNilIP nil IP 应视为不拦截（防御分支，不应 panic）。
func TestIsBlockedIPNilIP(t *testing.T) {
	if isBlockedIP(nil) {
		t.Fatal("nil IP should not be blocked")
	}
}

// TestValidateEndpointGaps 表驱动补齐 ValidateEndpoint 的分支：
// 无 scheme 自动补 http://；IP 字面量解析出的阿里云 IMDS 地址必须拦截。
func TestValidateEndpointGaps(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		wantErr  string
	}{
		{"no scheme adds http", "127.0.0.1:9000", ""},
		{"no scheme blocked host", "metadata.goog", "endpoint host is blocked"},
		{"aliyun imds ip literal", "http://100.100.100.200:80", "endpoint host is blocked"},
		{"volcano imds ip literal", "http://100.96.0.2", "endpoint host is blocked"},
		{"metadata.goog hostname", "http://metadata.goog", "endpoint host is blocked"},
		// 用户手填的 endpoint 常带首尾空白：此前 trim 后仅用于判空，解析仍用未 trim 的
		// 原串，导致合法地址被判为 "invalid endpoint URL"（features.md §K，原 todolist #10）。
		{"leading and trailing space", " http://127.0.0.1:9000 ", ""},
		{"uppercase scheme with spaces", " HTTP://127.0.0.1:9000/ ", ""},
		// 回归护栏：归一化后必须仍然拦到危险主机。空白原本就是 fail-closed（解析失败），
		// 这里断言的是「修复后拦到的是正确原因」——不能因为 trim 让危险地址漏过去。
		{"blocked host with spaces", "  http://metadata.goog  ", "endpoint host is blocked"},
		{"blocked ip with spaces", " http://100.100.100.200 ", "endpoint host is blocked"},
		// 退化地址必须 fail-closed：非空输入归一化后为空时不得静默放行。
		{"scheme only", "http://", "invalid endpoint URL"},
		{"slash only", "/", "invalid endpoint URL"},
		{"triple slash", "https:///", "invalid endpoint URL"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateEndpoint(c.endpoint)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateEndpoint(%q) = %v, want nil", c.endpoint, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("ValidateEndpoint(%q) = %v, want substring %q", c.endpoint, err, c.wantErr)
			}
		})
	}
}

// TestDialContextSSRFGaps 表驱动验证拨号前的 SSRF 二次校验：
// 地址缺端口、DNS 失败、命中被拦截 IP（含全被拦截后的兜底返回）、正常连通。
func TestDialContextSSRFGaps(t *testing.T) {
	// 启动一个本地监听，用于验证正常连通路径。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://") // host:port

	cases := []struct {
		name    string
		addr    string
		wantErr string
	}{
		{"missing port", "no-port", "missing port in address"},
		{"dns failure", "definitely.invalid.invalid.:1234", "no such host"},
		{"blocked imds ip", "169.254.169.254:80", "endpoint host is blocked"},
		{"aliyun imds blocked", "100.100.100.200:80", "endpoint host is blocked"},
		{"healthy loopback", host, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conn, err := dialContextSSRF(context.Background(), "tcp", c.addr)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("dial %q: %v", c.addr, err)
				}
				_ = conn.Close()
				return
			}
			if err == nil {
				if conn != nil {
					_ = conn.Close()
				}
				t.Fatalf("dial %q: expected error containing %q", c.addr, c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("dial %q err = %v, want substring %q", c.addr, err, c.wantErr)
			}
		})
	}
}

// TestDialContextSSRFLastDialError 记录未被拦截 IP 的拨号失败（closed port），验证 last 错误回传。
func TestDialContextSSRFLastDialError(t *testing.T) {
	_, err := dialContextSSRF(context.Background(), "tcp", "127.0.0.1:1")
	if err == nil {
		t.Fatal("dial closed port should fail")
	}
}

// TestCheckRedirectDenied 直接断言重定向回调始终拒绝。
func TestCheckRedirectDenied(t *testing.T) {
	if err := checkRedirectDenied(nil, nil); err != errRedirectDenied {
		t.Fatalf("checkRedirectDenied = %v, want errRedirectDenied", err)
	}
}

// TestSSRFAwareClientRejectsRedirect 端到端：HTTP 302 必须被 SSRF 防护客户端拒绝。
func TestSSRFAwareClientRejectsRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://169.254.169.254/latest/meta-data/")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	c := newHTTPClient()
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	_, err = c.client.Do(req)
	if err == nil {
		t.Fatal("redirect must be denied")
	}
	if !strings.Contains(err.Error(), "redirect denied") {
		t.Fatalf("err = %v, want redirect denied", err)
	}
}

// TestIsBlockedHostnameGaps 补齐剩余被拦主机名的表驱动用例。
func TestIsBlockedHostnameGaps(t *testing.T) {
	blocked := []string{"metadata.goog", "instance-data", "kubernetes.default", "kubernetes.default.svc", "metadata"}
	for _, h := range blocked {
		if !isBlockedHostname(h) {
			t.Errorf("isBlockedHostname(%q) = false, want true", h)
		}
	}
	if isBlockedHostname("minio.internal") {
		t.Error("normal host should not be blocked")
	}
}

// TestS3HTTPClientProxyDisabled S3 出站 HTTP 客户端必须禁用 HTTP(S)_PROXY，
// 避免环境变量把请求代理到任意主机绕过 dialContextSSRF。
func TestS3HTTPClientProxyDisabled(t *testing.T) {
	// 即便设了 HTTP_PROXY 也不应被使用。
	t.Setenv("HTTP_PROXY", "http://evil.example:9999")
	t.Setenv("HTTPS_PROXY", "http://evil.example:9999")
	t.Setenv("NO_PROXY", "")
	c := newHTTPClient()
	tr, ok := c.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", c.client.Transport)
	}
	if tr.Proxy != nil {
		t.Fatalf("S3 HTTP transport Proxy is set, want nil (HTTP(S)_PROXY must not bypass SSRF dial guard)")
	}
}
