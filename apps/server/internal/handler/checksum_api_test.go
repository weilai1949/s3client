package handler

// checksum_api_test.go —— 端到端校验和在 HTTP 边界的契约：
// head 响应新增 checksums 字段（有值 / null 降级）、verify-checksum 端点的校验结果与错误路径。

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// startChecksumFake 起假 S3：HEAD 带出存储端校验和头，GET 返回对象内容。
func startChecksumFake(t *testing.T, head func(h http.Header), body string) string {
	t.Helper()
	srv := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			if head != nil {
				head(w.Header())
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
	})
	return srv.URL
}

// TestHeadObjectExposesChecksums head 响应包含 checksums 对象。
func TestHeadObjectExposesChecksums(t *testing.T) {
	url := startChecksumFake(t, func(h http.Header) {
		h.Set("x-amz-checksum-crc64nvme", "N4bktbEKNg8=")
		h.Set("x-amz-checksum-sha256", "r+tj1ERvRP2f7jYwQzQIixedBQG2kUn3ZrRI1O6CdCU=")
		h.Set("x-amz-checksum-crc32c", "OL0iOA==")
		h.Set("x-amz-checksum-sha1", "nY7HDn3gD4GCCi0zieizVEU5TTY=")
		h.Set("x-amz-checksum-type", "FULL_OBJECT")
		h.Set("ETag", `"abc"`)
	}, "")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/head?bucket=b&key=k.txt", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"checksums":{`) ||
		!strings.Contains(body, `"crc64nvme":"N4bktbEKNg8="`) ||
		!strings.Contains(body, `"type":"FULL_OBJECT"`) {
		t.Fatalf("head response missing checksums: %s", body)
	}
}

// TestHeadObjectWithoutChecksums 无校验和（厂商不支持 / 对象未存）→ checksums:null。
func TestHeadObjectWithoutChecksums(t *testing.T) {
	url := startChecksumFake(t, func(h http.Header) {
		h.Set("ETag", `"abc"`)
	}, "")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/head?bucket=b&key=k.txt", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"checksums":null`) {
		t.Fatalf("head response must degrade to null: %s", rr.Body.String())
	}
}

// TestVerifyChecksumMatch 校验通过：method=crc64nvme、local==remote、match=true。
func TestVerifyChecksumMatch(t *testing.T) {
	url := startChecksumFake(t, func(h http.Header) {
		h.Set("x-amz-checksum-crc64nvme", "N4bktbEKNg8=")
		h.Set("x-amz-checksum-type", "FULL_OBJECT")
	}, "checksum probe payload")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/verify-checksum",
		`{"bucket":"b","key":"k.txt"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`"method":"crc64nvme"`, `"match":true`,
		`"local":"N4bktbEKNg8="`, `"remote":"N4bktbEKNg8="`, `"key":"k.txt"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
}

// TestVerifyChecksumNone 无可验证来源 → method=none、match=false（如实降级，不是错误）。
func TestVerifyChecksumNone(t *testing.T) {
	url := startChecksumFake(t, func(h http.Header) {
		h.Set("ETag", `"0123456789abcdef0123456789abcdef-3"`)
	}, "payload")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/verify-checksum",
		`{"bucket":"b","key":"big.bin"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"method":"none"`) ||
		!strings.Contains(rr.Body.String(), `"match":false`) {
		t.Fatalf("none-method response = %s", rr.Body.String())
	}
}

// TestVerifyChecksumInputValidation 缺 key → 400（边界拒绝）。
func TestVerifyChecksumInputValidation(t *testing.T) {
	url := startChecksumFake(t, nil, "")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/verify-checksum", `{"bucket":"b"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestVerifyChecksumNotFound 对象不存在 → 404。
func TestVerifyChecksumNotFound(t *testing.T) {
	srv := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `<?xml version="1.0"?><Error><Code>NoSuchKey</Code><Message>NoSuchKey</Message></Error>`)
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/verify-checksum",
		`{"bucket":"b","key":"gone.txt"}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestVerifyChecksumBoundary 坏 JSON 与无默认桶（覆盖率口径）。
func TestVerifyChecksumBoundary(t *testing.T) {
	url := startChecksumFake(t, nil, "")
	env := accNewEnv(t, url, "b")
	noBucket := accNewEnv(t, url, "")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/verify-checksum", `{`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d, want 400", rr.Code)
	}
	rr = noBucket.accDoRec("POST", "/api/accounts/"+noBucket.acc.ID+"/verify-checksum", `{"key":"k"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("no bucket status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestVerifyChecksumUnknownAccount 不存在账号 → 404。
func TestVerifyChecksumUnknownAccount(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	rr := env.accDoRec("POST", "/api/accounts/nope/verify-checksum", `{"bucket":"b","key":"k"}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

// TestHeadObjectUnknownAccount head 对不存在账号 → 404。
func TestHeadObjectUnknownAccount(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	rr := env.accDoRec("GET", "/api/accounts/nope/head?bucket=b&key=k", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
