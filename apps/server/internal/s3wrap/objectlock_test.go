package s3wrap

// objectlock_test.go —— Object Lock（桶级配置 / 对象保留期 / 法定保留）在 s3wrap 边界的行为。
// 断言外部可见行为：XML 解析出的 DTO、请求参数转发、未配置时的降级（nil 而非错误）、
// 以及锁定相关错误码的 HTTP 状态与用户文案。

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestGetObjectLockConfigurationParses 解析桶 Object Lock 配置。
func TestGetObjectLockConfigurationParses(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.Contains(r.URL.RawQuery, "object-lock") {
			t.Errorf("want GET <bucket>?object-lock, got %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0"?><ObjectLockConfiguration>
<ObjectLockEnabled>ENABLED</ObjectLockEnabled>
<Rule><DefaultRetention><Mode>GOVERNANCE</Mode><Days>7</Days></DefaultRetention></Rule>
</ObjectLockConfiguration>`)
	}))
	cfg, err := c.GetObjectLockConfiguration(context.Background(), "bkt")
	if err != nil {
		t.Fatalf("GetObjectLockConfiguration: %v", err)
	}
	if !cfg.Enabled || cfg.DefaultRetentionMode != "GOVERNANCE" ||
		cfg.DefaultRetentionDays != 7 || cfg.DefaultRetentionYears != 0 {
		t.Fatalf("config = %+v", cfg)
	}
}

// TestGetObjectLockConfigurationYears DefaultRetention 用 Years 表达时同样解析。
func TestGetObjectLockConfigurationYears(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0"?><ObjectLockConfiguration>
<ObjectLockEnabled>ENABLED</ObjectLockEnabled>
<Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Years>3</Years></DefaultRetention></Rule>
</ObjectLockConfiguration>`)
	}))
	cfg, err := c.GetObjectLockConfiguration(context.Background(), "bkt")
	if err != nil {
		t.Fatalf("GetObjectLockConfiguration: %v", err)
	}
	if !cfg.Enabled || cfg.DefaultRetentionMode != "COMPLIANCE" ||
		cfg.DefaultRetentionYears != 3 || cfg.DefaultRetentionDays != 0 {
		t.Fatalf("config = %+v", cfg)
	}
}

// TestGetObjectLockConfigurationNotConfigured 未启用 Object Lock 的桶 → Enabled=false 且无错误
// （厂商不支持/未启用的降级口径；NoSuchBucket 仍须报错，不得被吞成「未配置」）。
func TestGetObjectLockConfigurationNotConfigured(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "no-such-bucket") {
			writeS3Error(w, http.StatusNotFound, "NoSuchBucket", "gone")
			return
		}
		writeS3Error(w, http.StatusNotFound, "ObjectLockConfigurationNotFoundError", "no lock")
	}))
	cfg, err := c.GetObjectLockConfiguration(context.Background(), "plain-bkt")
	if err != nil {
		t.Fatalf("not-configured should not error: %v", err)
	}
	if cfg == nil || cfg.Enabled {
		t.Fatalf("config = %+v, want Enabled=false", cfg)
	}
	if _, err := c.GetObjectLockConfiguration(context.Background(), "no-such-bucket"); err == nil {
		t.Fatal("NoSuchBucket must surface as error")
	}
}

// TestPutObjectLockConfigurationForwards 请求体必须带上模式与默认保留期（天 / 年两种形态）。
func TestPutObjectLockConfigurationForwards(t *testing.T) {
	cases := []struct {
		name     string
		cfg      ObjectLockConfig
		wantElem string
	}{
		{"days", ObjectLockConfig{DefaultRetentionMode: "GOVERNANCE", DefaultRetentionDays: 30}, "<Days>30</Days>"},
		{"years", ObjectLockConfig{DefaultRetentionMode: "COMPLIANCE", DefaultRetentionYears: 3}, "<Years>3</Years>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg
			var body string
			c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				body = string(b)
				w.WriteHeader(http.StatusOK)
			}))
			if err := c.PutObjectLockConfiguration(context.Background(), "bkt", cfg); err != nil {
				t.Fatalf("PutObjectLockConfiguration: %v", err)
			}
			if !strings.Contains(body, "<Mode>"+cfg.DefaultRetentionMode+"</Mode>") || !strings.Contains(body, tc.wantElem) {
				t.Fatalf("put body = %q", body)
			}
			if !strings.Contains(body, "<ObjectLockEnabled>") {
				t.Fatalf("put body missing enabled flag: %q", body)
			}
		})
	}
}

// TestGetObjectLockConfigurationEmptyBody 厂商 200 空响应 → 未启用（不是错误）。
func TestGetObjectLockConfigurationEmptyBody(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	cfg, err := c.GetObjectLockConfiguration(context.Background(), "bkt")
	if err != nil {
		t.Fatalf("empty body should not error: %v", err)
	}
	if cfg == nil || cfg.Enabled {
		t.Fatalf("config = %+v, want Enabled=false", cfg)
	}
}

// TestGetObjectLockConfigurationRuleVariants 有配置但 Rule / DefaultRetention 缺失 → 启用但默认保留为空。
func TestGetObjectLockConfigurationRuleVariants(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"no-rule", `<?xml version="1.0"?><ObjectLockConfiguration><ObjectLockEnabled>ENABLED</ObjectLockEnabled></ObjectLockConfiguration>`},
		{"rule-no-retention", `<?xml version="1.0"?><ObjectLockConfiguration><ObjectLockEnabled>ENABLED</ObjectLockEnabled><Rule></Rule></ObjectLockConfiguration>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body
			c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = io.WriteString(w, body)
			}))
			cfg, err := c.GetObjectLockConfiguration(context.Background(), "bkt")
			if err != nil {
				t.Fatalf("GetObjectLockConfiguration: %v", err)
			}
			if !cfg.Enabled || cfg.DefaultRetentionMode != "" ||
				cfg.DefaultRetentionDays != 0 || cfg.DefaultRetentionYears != 0 {
				t.Fatalf("config = %+v, want enabled with empty default retention", cfg)
			}
		})
	}
}

// TestGetObjectRetentionEmptyBody 厂商 200 空响应 → 无保留期（nil，非错误）。
func TestGetObjectRetentionEmptyBody(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ret, err := c.GetObjectRetention(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("empty body should not error: %v", err)
	}
	if ret != nil {
		t.Fatalf("retention = %+v, want nil", ret)
	}
}

// TestGetObjectRetentionParses 解析对象保留期。
func TestGetObjectRetentionParses(t *testing.T) {
	var gotQuery string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0"?><ObjectRetention>
<Mode>COMPLIANCE</Mode><RetainUntilDate>2030-01-02T03:04:05.000Z</RetainUntilDate>
</ObjectRetention>`)
	}))
	ret, err := c.GetObjectRetention(context.Background(), "bkt", "k.txt", "v-1")
	if err != nil {
		t.Fatalf("GetObjectRetention: %v", err)
	}
	if ret == nil || ret.Mode != "COMPLIANCE" {
		t.Fatalf("retention = %+v", ret)
	}
	if !ret.RetainUntil.Equal(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("retainUntil = %s", ret.RetainUntil)
	}
	if !strings.Contains(gotQuery, "versionId=v-1") {
		t.Fatalf("versionId not forwarded: %q", gotQuery)
	}
}

// TestGetObjectRetentionNone 无保留期（或厂商不支持）→ nil 且不报错（降级口径）。
func TestGetObjectRetentionNone(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusNotFound, "ObjectLockConfigurationNotFoundError", "no lock")
	}))
	ret, err := c.GetObjectRetention(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("should degrade, not error: %v", err)
	}
	if ret != nil {
		t.Fatalf("retention = %+v, want nil", ret)
	}
}

// TestGetObjectRetentionInvalidRequestDegrade RustFS 非锁定桶读保留期返回 InvalidRequest
// （对象详情逐对象读，必须降级为「无保留期」而不是全线 400）。
func TestGetObjectRetentionInvalidRequestDegrade(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "Bucket is missing ObjectLockConfiguration")
	}))
	ret, err := c.GetObjectRetention(context.Background(), "bkt", "k.txt", "")
	if err != nil || ret != nil {
		t.Fatalf("ret=%+v err=%v, want (nil,nil) degrade", ret, err)
	}
}

// TestGetObjectLegalHoldInvalidRequestDegrade 同上，法定保留读的降级口径。
func TestGetObjectLegalHoldInvalidRequestDegrade(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusBadRequest, "InvalidRequest", "Bucket is missing ObjectLockConfiguration")
	}))
	status, err := c.GetObjectLegalHold(context.Background(), "bkt", "k.txt", "")
	if err != nil || status != "OFF" {
		t.Fatalf("status=%q err=%v, want (OFF,nil) degrade", status, err)
	}
}

// TestGetObjectRetentionEmptyElement RustFS 对锁桶上无保留对象返回空 <ObjectRetention>
// 元素（非 nil、字段全空）→ 同样归一为「未设置」。
func TestGetObjectRetentionEmptyElement(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0"?><ObjectRetention></ObjectRetention>`)
	}))
	ret, err := c.GetObjectRetention(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("GetObjectRetention: %v", err)
	}
	if ret != nil {
		t.Fatalf("retention = %+v, want nil (empty element means unset)", ret)
	}
}

// TestGetObjectRetentionFailure 非「未配置」类错误（如桶不存在）必须原样上抛，不得吞成 nil。
func TestGetObjectRetentionFailure(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusNotFound, "NoSuchBucket", "gone")
	}))
	if _, err := c.GetObjectRetention(context.Background(), "bkt", "k.txt", ""); err == nil {
		t.Fatal("real errors must surface")
	}
}

// TestGetObjectLegalHoldFailure 非「未设置」类错误必须原样上抛。
func TestGetObjectLegalHoldFailure(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusForbidden, "AccessDenied", "no")
	}))
	if _, err := c.GetObjectLegalHold(context.Background(), "bkt", "k.txt", ""); err == nil {
		t.Fatal("real errors must surface")
	}
}

// TestPutObjectRetentionForwards 保留期请求参数转发（模式 / 到期时间 / 版本）。
func TestPutObjectRetentionForwards(t *testing.T) {
	var body, gotQuery string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body, gotQuery = string(b), r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	until := time.Date(2031, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := c.PutObjectRetention(context.Background(), "bkt", "k.txt", "v-9", "GOVERNANCE", until); err != nil {
		t.Fatalf("PutObjectRetention: %v", err)
	}
	if !strings.Contains(body, "<Mode>GOVERNANCE</Mode>") {
		t.Fatalf("body missing mode: %q", body)
	}
	if !strings.Contains(body, "2031-02-03T04:05:06") {
		t.Fatalf("body missing retainUntil: %q", body)
	}
	if !strings.Contains(gotQuery, "versionId=v-9") {
		t.Fatalf("versionId not forwarded: %q", gotQuery)
	}
}

// TestGetObjectLegalHoldParses 法定保留状态解析。
func TestGetObjectLegalHoldParses(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0"?><LegalHold><Status>ON</Status></LegalHold>`)
	}))
	status, err := c.GetObjectLegalHold(context.Background(), "bkt", "k.txt", "v-5")
	if err != nil {
		t.Fatalf("GetObjectLegalHold: %v", err)
	}
	if status != "ON" {
		t.Fatalf("status = %q, want ON", status)
	}
}

// TestGetObjectLegalHoldOff 未设法定保留 → OFF（空状态归一为 OFF，响应形状稳定）。
func TestGetObjectLegalHoldOff(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusNotFound, "LegalHoldNotFoundError", "no hold")
	}))
	status, err := c.GetObjectLegalHold(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("should degrade, not error: %v", err)
	}
	if status != "OFF" {
		t.Fatalf("status = %q, want OFF", status)
	}
}

// TestGetObjectLegalHoldEmptyElement 200 返回空 LegalHold 元素 → 归一为 OFF（响应形状稳定）。
func TestGetObjectLegalHoldEmptyElement(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0"?><LegalHold></LegalHold>`)
	}))
	status, err := c.GetObjectLegalHold(context.Background(), "bkt", "k.txt", "")
	if err != nil {
		t.Fatalf("GetObjectLegalHold: %v", err)
	}
	if status != "OFF" {
		t.Fatalf("status = %q, want OFF", status)
	}
}

// TestPutObjectLegalHoldForwards 法定保留写入参数转发。
func TestPutObjectLegalHoldForwards(t *testing.T) {
	var body, gotQuery string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body, gotQuery = string(b), r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	if err := c.PutObjectLegalHold(context.Background(), "bkt", "k.txt", "v-3", "ON"); err != nil {
		t.Fatalf("PutObjectLegalHold: %v", err)
	}
	if !strings.Contains(body, "<Status>ON</Status>") {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(gotQuery, "versionId=v-3") {
		t.Fatalf("versionId not forwarded: %q", gotQuery)
	}
}

// TestObjectLockUnavailableMapping 厂商不支持 / 桶未启用的错误码契约：
// ObjectLockConfigurationNotFoundError（AWS 口径）→ 400 + 明确说明未启用；NotImplemented → 501。
func TestObjectLockUnavailableMapping(t *testing.T) {
	err := fakeAPIError{code: "ObjectLockConfigurationNotFoundError"}
	if got := HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(ObjectLockConfigurationNotFoundError) = %d, want 400", got)
	}
	if msg := UserMessage(err); !strings.Contains(msg, "object lock") {
		t.Fatalf("UserMessage = %q, want mention object lock", msg)
	}
	ni := fakeAPIError{code: "NotImplemented"}
	if got := HTTPStatus(ni); got != http.StatusNotImplemented {
		t.Fatalf("HTTPStatus(NotImplemented) = %d, want 501", got)
	}
	if msg := UserMessage(ni); !strings.Contains(msg, "not supported") {
		t.Fatalf("UserMessage = %q, want mention not supported", msg)
	}
}

// TestObjectLockErrorMapping 锁定相关错误码的对外契约（状态 + 文案）。
func TestObjectLockErrorMapping(t *testing.T) {
	locked := fakeAPIError{code: "ObjectLocked"}
	if got := HTTPStatus(locked); got != http.StatusConflict {
		t.Fatalf("HTTPStatus(ObjectLocked) = %d, want 409", got)
	}
	if msg := UserMessage(locked); !strings.Contains(msg, "locked") {
		t.Fatalf("UserMessage(ObjectLocked) = %q, want mention locked", msg)
	}
	short := fakeAPIError{code: "RetentionPeriodTooShort"}
	if got := HTTPStatus(short); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(RetentionPeriodTooShort) = %d, want 400", got)
	}
	if msg := UserMessage(short); !strings.Contains(msg, "retention") {
		t.Fatalf("UserMessage(RetentionPeriodTooShort) = %q, want mention retention", msg)
	}
}
