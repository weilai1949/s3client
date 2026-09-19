package handler

// audit_test.go —— 安全审计日志与可信代理 XFF（ASSESSMENT M3/M4）。
//
// 审计日志只记录「谁在何时对什么做了什么」，不记录密钥等敏感值；
// XFF 只有在直连对端是已配置的可信代理时才被采信，否则回退 RemoteAddr，
// 避免直连部署时伪造 X-Forwarded-For 绕过限速。

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// auditRecorder 收集 JSON 日志行，便于断言审计事件字段。
type auditRecorder struct {
	buf *bytes.Buffer
}

func newAuditHandler(t *testing.T, token string, trustedProxies []string) (*Handler, *auditRecorder) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := New(st, logger, t.TempDir(), nil, token, "test", false, false)
	h.SetTrustedProxies(trustedProxies)
	t.Cleanup(h.Shutdown)
	return h, &auditRecorder{buf: buf}
}

// auditEvents 返回日志中所有 audit=... 的事件（key -> 该行 JSON）。
func (a *auditRecorder) auditEvents(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(a.buf.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line is not JSON: %s", line)
		}
		if ev, ok := m["audit"].(string); ok && ev != "" {
			out = append(out, m)
		}
	}
	return out
}

func TestAuditLogsUnauthorized(t *testing.T) {
	h, rec := newAuditHandler(t, "unit-test-token-0123456789", nil)
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/accounts", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	events := rec.auditEvents(t)
	if len(events) == 0 {
		t.Fatal("expected an audit event for the rejected request")
	}
	ev := events[len(events)-1]
	if ev["audit"] != "auth.denied" {
		t.Fatalf("audit = %v, want auth.denied", ev["audit"])
	}
	if ev["path"] != "/api/accounts" {
		t.Fatalf("path = %v, want /api/accounts", ev["path"])
	}
	// 审计日志不得记录 token 本身。
	if strings.Contains(rec.buf.String(), "wrong-token") {
		t.Fatal("audit log leaked the presented token")
	}
}

func TestAuditLogsAccountCRUD(t *testing.T) {
	h, rec := newAuditHandler(t, "", nil)
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	body := `{"name":"audited","endpoint":"http://127.0.0.1:1","accessKey":"ak","secretKey":"sk-secret"}`
	resp, err := srv.Client().Post(srv.URL+"/api/accounts", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}

	events := rec.auditEvents(t)
	var created map[string]any
	for _, ev := range events {
		if ev["audit"] == "account.create" {
			created = ev
		}
	}
	if created == nil {
		t.Fatalf("expected account.create audit event, got %v", events)
	}
	if created["id"] == "" || created["id"] == nil {
		t.Fatalf("account.create event missing id: %v", created)
	}
	// 审计日志不得记录密钥。
	if strings.Contains(rec.buf.String(), "sk-secret") {
		t.Fatal("audit log leaked the account secret key")
	}
}

// TestAuditLogsDeleteAndPolicyChanges 覆盖删除与桶策略变更两类安全敏感操作。
func TestAuditLogsDeleteAndPolicyChanges(t *testing.T) {
	h, rec := newAuditHandler(t, "", nil)
	st := h.store
	acc, err := st.Create(newAccountForAudit())
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	// 桶策略变更：假 S3 返回 200。
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fake.Close()
	acc.Endpoint = fake.URL
	if _, err := st.Update(acc.ID, acc); err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/accounts/"+acc.ID+"/bucket/policy",
		strings.NewReader(`{"bucket":"b","policy":"{\"Version\":\"2012-10-17\"}"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	events := rec.auditEvents(t)
	found := false
	for _, ev := range events {
		if ev["audit"] == "bucket.policy.update" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected bucket.policy.update audit event, got %v", events)
	}
}

func newAccountForAudit() *model.Account {
	return &model.Account{
		Name: "audit", Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	}
}

// ---- clientIP / 可信代理 ----

func TestClientIPTrustedProxy(t *testing.T) {
	cases := []struct {
		name       string
		trusted    []string
		remoteAddr string
		xff        string
		want       string
	}{
		{"no xff uses remote", nil, "203.0.113.9:1234", "", "203.0.113.9"},
		{"untrusted peer ignores xff", []string{"10.0.0.1"}, "203.0.113.9:1234", "1.2.3.4", "203.0.113.9"},
		{"trusted peer honors xff", []string{"10.0.0.1"}, "10.0.0.1:5555", "1.2.3.4", "1.2.3.4"},
		{"trusted peer honors first xff hop", []string{"10.0.0.1"}, "10.0.0.1:5555", "1.2.3.4, 5.6.7.8", "1.2.3.4"},
		{"trusted peer without xff falls back to remote", []string{"10.0.0.1"}, "10.0.0.1:5555", "", "10.0.0.1"},
		{"empty trusted list ignores xff", nil, "10.0.0.1:5555", "1.2.3.4", "10.0.0.1"},
		{"malformed remote falls back", []string{"10.0.0.1"}, "not-an-addr", "1.2.3.4", "not-an-addr"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
			r.RemoteAddr = c.remoteAddr
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := clientIPWithProxies(r, c.trusted); got != c.want {
				t.Fatalf("clientIP = %q, want %q", got, c.want)
			}
		})
	}
}

// TestRateLimitNotBypassedBySpoofedXFF 直连客户端伪造 XFF 不能绕过限速：
// 无论 XFF 怎么变，都按同一个 RemoteAddr 计数，超过突发即 429。
func TestRateLimitNotBypassedBySpoofedXFF(t *testing.T) {
	h, _ := newAuditHandler(t, "", nil)
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	client := srv.Client()
	status := 0
	for i := 0; i < rateLimitBurst+5; i++ {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/accounts", nil)
		// 每次伪造不同的 XFF：若被采信则每个 IP 都是新桶，永远不会 429。
		req.Header.Set("X-Forwarded-For", "10.9.9."+string(rune('0'+i%10)))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		status = resp.StatusCode
		resp.Body.Close()
		if status == http.StatusTooManyRequests {
			break
		}
	}
	if status != http.StatusTooManyRequests {
		t.Fatalf("spoofed XFF bypassed rate limiting: last status = %d, want 429", status)
	}
}
