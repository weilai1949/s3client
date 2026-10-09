package handler

// objectlock_api_test.go —— Object Lock 三组端点（桶配置 / 对象保留期 / 法定保留）的 HTTP 契约：
// 响应形状、输入校验（400 固定文案）、未启用降级（enabled=false / configured=false / OFF）、
// 锁定类错误的状态码透出（409/403）、未知账号 404。

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// olFakeResp 假 S3 对单请求的应答（status==0 视为 200 空响应）。
type olFakeResp struct {
	status int
	body   string
}

// olCapture 假 S3 观察到的请求要素。
type olCapture struct {
	method string
	query  string
	path   string
	body   string
}

// startObjectLockFake 起假 S3：route 按请求给出应答，同时记录请求要素供断言。
func startObjectLockFake(t *testing.T, route func(r *http.Request) olFakeResp) (*olCapture, string) {
	t.Helper()
	got := &olCapture{}
	srv := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got.method, got.query, got.path, got.body = r.Method, r.URL.RawQuery, r.URL.Path, string(b)
		resp := route(r)
		status := resp.status
		if status == 0 {
			status = http.StatusOK
		}
		if strings.Contains(resp.body, "<Error>") {
			w.Header().Set("Content-Type", "application/xml")
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, resp.body)
	})
	return got, srv.URL
}

// olErrResp 返回 S3 风格错误应答。
func olErrResp(status int, code string) olFakeResp {
	return olFakeResp{status: status, body: `<?xml version="1.0"?><Error><Code>` + code + `</Code><Message>` + code + `</Message></Error>`}
}

// ---- 桶级 Object Lock 配置 ----

// TestGetBucketObjectLockEnabled 启用中的桶 → enabled=true + 默认保留策略。
func TestGetBucketObjectLockEnabled(t *testing.T) {
	got, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olFakeResp{body: `<?xml version="1.0"?><ObjectLockConfiguration><ObjectLockEnabled>ENABLED</ObjectLockEnabled><Rule><DefaultRetention><Mode>GOVERNANCE</Mode><Days>7</Days></DefaultRetention></Rule></ObjectLockConfiguration>`}
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/bucket/object-lock?bucket=b", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{`"enabled":true`, `"defaultRetentionMode":"GOVERNANCE"`, `"defaultRetentionDays":7`, `"defaultRetentionYears":0`, `"bucket":"b"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	if !strings.Contains(got.query, "object-lock") {
		t.Fatalf("fake should see object-lock query, got %q", got.query)
	}
}

// TestGetBucketObjectLockDisabled 未启用（含厂商返回配置不存在）→ enabled=false，HTTP 200。
func TestGetBucketObjectLockDisabled(t *testing.T) {
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olErrResp(http.StatusNotFound, "ObjectLockConfigurationNotFoundError")
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/bucket/object-lock?bucket=plain", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 degradation", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"enabled":false`) || !strings.Contains(body, `"defaultRetentionMode":""`) {
		t.Fatalf("degraded response = %s", body)
	}
}

// TestPutBucketObjectLockValid 合法配置 → 转发给 S3 的请求体带模式与保留期。
func TestPutBucketObjectLockValid(t *testing.T) {
	got, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olFakeResp{}
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/bucket/object-lock",
		`{"bucket":"b","defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":30}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"enabled":true`) ||
		!strings.Contains(rr.Body.String(), `"defaultRetentionDays":30`) {
		t.Fatalf("response = %s", rr.Body.String())
	}
	if got.method != http.MethodPut || !strings.Contains(got.query, "object-lock") ||
		!strings.Contains(got.body, "<Mode>GOVERNANCE</Mode>") || !strings.Contains(got.body, "<Days>30</Days>") {
		t.Fatalf("fake saw method=%s query=%q body=%s", got.method, got.query, got.body)
	}
}

// TestPutBucketObjectLockValidation 输入校验表（每条都在边界 400，不打到 S3）。
func TestPutBucketObjectLockValidation(t *testing.T) {
	called := false
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		called = true
		return olFakeResp{}
	})
	env := accNewEnv(t, url, "b")
	cases := []struct {
		name string
		body string
	}{
		{"missing mode", `{"bucket":"b","defaultRetentionDays":3}`},
		{"bad mode", `{"bucket":"b","defaultRetentionMode":"WHATEVER","defaultRetentionDays":3}`},
		{"days and years", `{"bucket":"b","defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":1,"defaultRetentionYears":1}`},
		{"neither days nor years", `{"bucket":"b","defaultRetentionMode":"GOVERNANCE"}`},
		{"negative days", `{"bucket":"b","defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":-1}`},
		{"negative years", `{"bucket":"b","defaultRetentionMode":"GOVERNANCE","defaultRetentionYears":-1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/bucket/object-lock", tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
			}
			if called {
				t.Fatal("invalid input must not reach S3")
			}
		})
	}
}

// TestPutBucketObjectLockBucketState 桶未在创建时启用 Object Lock（RustFS: InvalidBucketState 409）
// → 409 + 固定文案（说明只能建桶时启用）。
func TestPutBucketObjectLockBucketState(t *testing.T) {
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olErrResp(http.StatusConflict, "InvalidBucketState")
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/bucket/object-lock",
		`{"bucket":"b","defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":30}`)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "object lock") {
		t.Fatalf("409 body must explain object lock enablement: %s", rr.Body.String())
	}
}

// ---- 对象保留期 ----

// TestGetObjectRetentionConfigured 有保留期 → configured=true + RFC3339 到期时间 + versionId 透传。
func TestGetObjectRetentionConfigured(t *testing.T) {
	got, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olFakeResp{body: `<?xml version="1.0"?><ObjectRetention><Mode>COMPLIANCE</Mode><RetainUntilDate>2030-01-02T03:04:05.000Z</RetainUntilDate></ObjectRetention>`}
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/object-retention?bucket=b&key=k.txt&versionId=v-1", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{`"configured":true`, `"mode":"COMPLIANCE"`, `"retainUntilDate":"2030-01-02T03:04:05Z"`, `"versionId":"v-1"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
	if !strings.Contains(got.query, "retention") || !strings.Contains(got.query, "versionId=v-1") {
		t.Fatalf("fake query = %q, want retention + versionId", got.query)
	}
}

// TestGetObjectRetentionNotConfigured 无保留期 → configured=false（形状稳定：空串字段）。
func TestGetObjectRetentionNotConfigured(t *testing.T) {
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olFakeResp{}
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/object-retention?bucket=b&key=k.txt", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"configured":false`) || !strings.Contains(body, `"mode":""`) ||
		!strings.Contains(body, `"retainUntilDate":""`) {
		t.Fatalf("degraded response = %s", body)
	}
}

// TestPutObjectRetentionValid 合法保留期 → 转发 S3（模式 / RFC3339 / versionId）。
func TestPutObjectRetentionValid(t *testing.T) {
	got, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olFakeResp{}
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/object-retention",
		`{"bucket":"b","key":"k.txt","versionId":"v-9","mode":"GOVERNANCE","retainUntilDate":"2031-02-03T04:05:06Z"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"configured":true`) ||
		!strings.Contains(rr.Body.String(), `"retainUntilDate":"2031-02-03T04:05:06Z"`) {
		t.Fatalf("response = %s", rr.Body.String())
	}
	if got.method != http.MethodPut || !strings.Contains(got.query, "retention") ||
		!strings.Contains(got.query, "versionId=v-9") ||
		!strings.Contains(got.body, "<Mode>GOVERNANCE</Mode>") ||
		!strings.Contains(got.body, "2031-02-03T04:05:06") {
		t.Fatalf("fake saw method=%s query=%q body=%s", got.method, got.query, got.body)
	}
}

// TestPutObjectRetentionValidation 输入校验表（缺 key / 缺或非法模式 / 缺或非法或过期日期）。
func TestPutObjectRetentionValidation(t *testing.T) {
	called := false
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		called = true
		return olFakeResp{}
	})
	env := accNewEnv(t, url, "b")
	future := "2031-02-03T04:05:06Z"
	cases := []struct {
		name string
		body string
	}{
		{"missing key", `{"bucket":"b","mode":"GOVERNANCE","retainUntilDate":"` + future + `"}`},
		{"missing mode", `{"bucket":"b","key":"k","retainUntilDate":"` + future + `"}`},
		{"bad mode", `{"bucket":"b","key":"k","mode":"WHATEVER","retainUntilDate":"` + future + `"}`},
		{"missing date", `{"bucket":"b","key":"k","mode":"GOVERNANCE"}`},
		{"bad date", `{"bucket":"b","key":"k","mode":"GOVERNANCE","retainUntilDate":"tomorrow"}`},
		{"past date", `{"bucket":"b","key":"k","mode":"GOVERNANCE","retainUntilDate":"2001-01-01T00:00:00Z"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/object-retention", tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
			}
			if called {
				t.Fatal("invalid input must not reach S3")
			}
		})
	}
}

// TestPutObjectRetentionLocked 合规锁定拒绝修改（S3: ObjectLocked 409）→ 409 + 文案。
func TestPutObjectRetentionLocked(t *testing.T) {
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olErrResp(http.StatusConflict, "ObjectLocked")
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/object-retention",
		`{"bucket":"b","key":"k.txt","mode":"GOVERNANCE","retainUntilDate":"2031-02-03T04:05:06Z"}`)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "locked") {
		t.Fatalf("409 body must mention locked: %s", rr.Body.String())
	}
}

// ---- 法定保留 ----

// TestGetObjectLegalHoldOn 状态读取 + versionId 透传。
func TestGetObjectLegalHoldOn(t *testing.T) {
	got, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olFakeResp{body: `<?xml version="1.0"?><LegalHold><Status>ON</Status></LegalHold>`}
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/object-legal-hold?bucket=b&key=k.txt&versionId=v-2", "")
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"ON"`) {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(got.query, "legal-hold") || !strings.Contains(got.query, "versionId=v-2") {
		t.Fatalf("fake query = %q", got.query)
	}
}

// TestGetObjectLegalHoldOff 未设置（S3: LegalHoldNotFoundError）→ status=OFF。
func TestGetObjectLegalHoldOff(t *testing.T) {
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olErrResp(http.StatusNotFound, "LegalHoldNotFoundError")
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/object-legal-hold?bucket=b&key=k.txt", "")
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"OFF"`) {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestPutObjectLegalHoldValid ON/OFF 写入转发。
func TestPutObjectLegalHoldValid(t *testing.T) {
	got, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olFakeResp{}
	})
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/object-legal-hold",
		`{"bucket":"b","key":"k.txt","versionId":"v-3","status":"ON"}`)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"ON"`) {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got.method != http.MethodPut || !strings.Contains(got.query, "legal-hold") ||
		!strings.Contains(got.query, "versionId=v-3") ||
		!strings.Contains(got.body, "<Status>ON</Status>") {
		t.Fatalf("fake saw method=%s query=%q body=%s", got.method, got.query, got.body)
	}
}

// TestPutObjectLegalHoldValidation 输入校验（缺 key / 缺状态 / 非法状态）。
func TestPutObjectLegalHoldValidation(t *testing.T) {
	called := false
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		called = true
		return olFakeResp{}
	})
	env := accNewEnv(t, url, "b")
	cases := []struct {
		name string
		body string
	}{
		{"missing key", `{"bucket":"b","status":"ON"}`},
		{"missing status", `{"bucket":"b","key":"k"}`},
		{"bad status", `{"bucket":"b","key":"k","status":"MAYBE"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			rr := env.accDoRec("PUT", "/api/accounts/"+env.acc.ID+"/object-legal-hold", tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
			}
			if called {
				t.Fatal("invalid input must not reach S3")
			}
		})
	}
}

// ---- 边界分支：坏 JSON / 缺桶 / 缺 key / S3 真实错误透传 ----

// TestObjectLockBoundaryPaths 六个 Object Lock 端点的边界分支（覆盖率口径）：
// readJSON 失败、无默认桶且未传 bucket、GET 缺 key、S3 真实错误经 writeInternalErr 透传。
func TestObjectLockBoundaryPaths(t *testing.T) {
	_, url := startObjectLockFake(t, func(r *http.Request) olFakeResp {
		return olErrResp(http.StatusNotFound, "NoSuchBucket")
	})
	env := accNewEnv(t, url, "b")     // 有默认桶
	noBucket := accNewEnv(t, url, "") // 无默认桶（触发 bucketOr 失败）
	id, nid := env.acc.ID, noBucket.acc.ID

	cases := []struct {
		name, method, path, body string
		noDefault                bool // true = 该账号无默认桶（走 noBucket 的 handler）
		want                     int
	}{
		// bucket/object-lock
		{"put-lock bad json", "PUT", "/api/accounts/" + id + "/bucket/object-lock", `{`, false, 400},
		{"put-lock no bucket", "PUT", "/api/accounts/" + nid + "/bucket/object-lock",
			`{"defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":1}`, true, 400},
		{"put-lock s3 error", "PUT", "/api/accounts/" + id + "/bucket/object-lock",
			`{"bucket":"b","defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":1}`, false, 404},
		{"get-lock no bucket", "GET", "/api/accounts/" + nid + "/bucket/object-lock", "", true, 400},
		{"get-lock s3 error", "GET", "/api/accounts/" + id + "/bucket/object-lock?bucket=b", "", false, 404},
		// object-retention
		{"get-retention no bucket", "GET", "/api/accounts/" + nid + "/object-retention?key=k", "", true, 400},
		{"get-retention no key", "GET", "/api/accounts/" + id + "/object-retention?bucket=b", "", false, 400},
		{"get-retention s3 error", "GET", "/api/accounts/" + id + "/object-retention?bucket=b&key=k", "", false, 404},
		{"put-retention bad json", "PUT", "/api/accounts/" + id + "/object-retention", `{`, false, 400},
		{"put-retention no bucket", "PUT", "/api/accounts/" + nid + "/object-retention",
			`{"key":"k","mode":"GOVERNANCE","retainUntilDate":"2031-01-01T00:00:00Z"}`, true, 400},
		// object-legal-hold
		{"get-hold no bucket", "GET", "/api/accounts/" + nid + "/object-legal-hold?key=k", "", true, 400},
		{"get-hold no key", "GET", "/api/accounts/" + id + "/object-legal-hold?bucket=b", "", false, 400},
		{"get-hold s3 error", "GET", "/api/accounts/" + id + "/object-legal-hold?bucket=b&key=k", "", false, 404},
		{"put-hold bad json", "PUT", "/api/accounts/" + id + "/object-legal-hold", `{`, false, 400},
		{"put-hold no bucket", "PUT", "/api/accounts/" + nid + "/object-legal-hold",
			`{"key":"k","status":"ON"}`, true, 400},
		{"put-hold s3 error", "PUT", "/api/accounts/" + id + "/object-legal-hold",
			`{"bucket":"b","key":"k","status":"ON"}`, false, 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := env
			if tc.noDefault {
				e = noBucket
			}
			rr := e.accDoRec(tc.method, tc.path, tc.body)
			if rr.Code != tc.want {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

// ---- 账号解析 ----

// TestObjectLockEndpointsUnknownAccount 六个端点对不存在账号统一 404。
func TestObjectLockEndpointsUnknownAccount(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	cases := []struct {
		name, method, path, body string
	}{
		{"get lock cfg", "GET", "/api/accounts/nope/bucket/object-lock?bucket=b", ""},
		{"put lock cfg", "PUT", "/api/accounts/nope/bucket/object-lock", `{"bucket":"b","defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":1}`},
		{"get retention", "GET", "/api/accounts/nope/object-retention?bucket=b&key=k", ""},
		{"put retention", "PUT", "/api/accounts/nope/object-retention", `{"bucket":"b","key":"k","mode":"GOVERNANCE","retainUntilDate":"2031-01-01T00:00:00Z"}`},
		{"get legal hold", "GET", "/api/accounts/nope/object-legal-hold?bucket=b&key=k", ""},
		{"put legal hold", "PUT", "/api/accounts/nope/object-legal-hold", `{"bucket":"b","key":"k","status":"ON"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := env.accDoRec(tc.method, tc.path, tc.body)
			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rr.Code)
			}
		})
	}
}
