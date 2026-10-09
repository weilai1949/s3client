package handler

// conditional_write_test.go —— 条件写（If-Match / If-None-Match）与校验和物化在 HTTP 边界的契约：
// 请求字段校验（400 固定文案、不回显输入）、预签名回显头、服务端写路径透传、
// S3 412 → HTTP 412 透出、copy-object 的 checksumAlgorithm 透传与枚举校验。
//
// 注：本文件曾于 2026-10-08 21:05–21:07 的并行开发窗口内被工作区外部进程删除，此处为
// 同内容重建（含全部后续修订：copy-source 记录、CopyObject 线格式断言、checksumAlgorithm 用例）。

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// condFake 假 S3 观察到的请求要素。
type condFake struct {
	ifMatch     string
	ifNoneMatch string
	copySource  string
	method      string
	path        string
}

// startCondFake 起假 S3：默认 200；status>0 时按给定状态回 S3 风格错误体。
func startCondFake(t *testing.T, status int, code string) (*condFake, string) {
	t.Helper()
	got := &condFake{}
	srv := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		got.ifMatch = r.Header.Get("If-Match")
		got.ifNoneMatch = r.Header.Get("If-None-Match")
		got.copySource = r.Header.Get("x-amz-copy-source")
		if status != 0 {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `<?xml version="1.0"?><Error><Code>`+code+`</Code><Message>`+code+`</Message></Error>`)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return got, srv.URL
}

// TestPresignPutConditionsEcho presign PUT 带条件 → 响应回显 headers（浏览器直传必须携带）；
// 无条件 → headers 为空对象（响应体形状稳定）。
func TestPresignPutConditionsEcho(t *testing.T) {
	got, url := startCondFake(t, 0, "")
	env := accNewEnv(t, url, "b")
	id := env.acc.ID

	rr := env.accDoRec("POST", "/api/accounts/"+id+"/presign",
		`{"key":"k.txt","method":"put","ifNoneMatch":"*"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("presign put status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"headers":{"If-None-Match":"*"}`) {
		t.Fatalf("response must echo If-None-Match header: %s", rr.Body.String())
	}

	rr = env.accDoRec("POST", "/api/accounts/"+id+"/presign",
		`{"key":"k.txt","method":"put","ifMatch":"\"e1\""}`)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"If-Match":"\"e1\""`) {
		t.Fatalf("ifMatch echo missing: %d %s", rr.Code, rr.Body.String())
	}

	rr = env.accDoRec("POST", "/api/accounts/"+id+"/presign", `{"key":"k.txt","method":"put"}`)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"headers":{}`) {
		t.Fatalf("unconditional presign must return empty headers: %d %s", rr.Code, rr.Body.String())
	}
	if got.method != "" {
		t.Fatalf("presign must not hit the S3 endpoint, saw %s %s", got.method, got.path)
	}
}

// TestPresignConditionsValidation 条件字段校验（边界拒绝，文案固定不回显输入）。
func TestPresignConditionsValidation(t *testing.T) {
	_, url := startCondFake(t, 0, "")
	env := accNewEnv(t, url, "b")
	id := env.acc.ID
	cases := []struct {
		name string
		body string
	}{
		{"ifNoneMatch not star", `{"key":"k","method":"put","ifNoneMatch":"etag-x"}`},
		{"ifMatch with CRLF", `{"key":"k","method":"put","ifMatch":"a\r\nX-Evil: 1"}`},
		{"ifMatch with space", `{"key":"k","method":"put","ifMatch":"a b"}`},
		{"ifMatch too long", `{"key":"k","method":"put","ifMatch":"` + strings.Repeat("a", 600) + `"}`},
		{"conditions on get", `{"key":"k","method":"get","ifNoneMatch":"*"}`},
		{"conditions on post", `{"key":"k","method":"post","ifMatch":"\"e1\""}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := env.accDoRec("POST", "/api/accounts/"+id+"/presign", tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
			}
			if strings.Contains(rr.Body.String(), "a\r\nX-Evil") || strings.Contains(rr.Body.String(), strings.Repeat("a", 600)) {
				t.Fatalf("400 message must not echo user input: %s", rr.Body.String())
			}
		})
	}
}

// TestMkdirConditionalCreate mkdir 带 ifNoneMatch → 透传给 PutObject 的条件头（只创建不覆盖）。
func TestMkdirConditionalCreate(t *testing.T) {
	got, url := startCondFake(t, 0, "")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/mkdir",
		`{"key":"newdir/","ifNoneMatch":"*"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("mkdir status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got.method != http.MethodPut || got.ifNoneMatch != "*" {
		t.Fatalf("fake saw %s ifNoneMatch=%q, want PUT with *", got.method, got.ifNoneMatch)
	}
	// 非法 ifNoneMatch → 400（边界拒绝，不打到 S3）
	rr = env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/mkdir",
		`{"key":"newdir/","ifNoneMatch":"nope"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid ifNoneMatch status = %d, want 400", rr.Code)
	}
}

// TestMkdirConditionalMatch ifMatch 条件同样透传。
func TestMkdirConditionalMatch(t *testing.T) {
	got, url := startCondFake(t, 0, "")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/mkdir",
		`{"key":"d/","ifMatch":"\"e9\""}`)
	if rr.Code != http.StatusOK || got.ifMatch != "\"e9\"" {
		t.Fatalf("mkdir ifMatch: status=%d fake-saw=%q", rr.Code, got.ifMatch)
	}
}

// TestCopyObjectChecksumAlgorithm copy-object 的 checksumAlgorithm 透传与边界校验。
func TestCopyObjectChecksumAlgorithm(t *testing.T) {
	_, url := startCondFake(t, 0, "")
	env := accNewEnv(t, url, "b")
	// 单独起一个记录校验和算法头的假 S3（startCondFake 不含该头）。
	var gotAlg string
	srv2 := accStartFake(t, func(w http.ResponseWriter, r *http.Request) {
		gotAlg = r.Header.Get("x-amz-checksum-algorithm")
		w.WriteHeader(http.StatusOK)
	})
	env2 := accNewEnv(t, srv2.URL, "b")
	rr := env2.accDoRec("POST", "/api/accounts/"+env2.acc.ID+"/copy-object",
		`{"key":"s.txt","newKey":"d.txt","checksumAlgorithm":"CRC64NVME"}`)
	if rr.Code != http.StatusOK || gotAlg != "CRC64NVME" {
		t.Fatalf("checksumAlgorithm: status=%d fake-saw=%q", rr.Code, gotAlg)
	}
	// 非法枚举 → 400（边界拒绝，不打到 S3）
	rr = env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-object",
		`{"key":"s.txt","newKey":"d.txt","checksumAlgorithm":"MD5"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid checksumAlgorithm status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestCopyObjectInvalidConditions copy-object 的条件字段非法 → 400（边界拒绝）。
func TestCopyObjectInvalidConditions(t *testing.T) {
	_, url := startCondFake(t, 0, "")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-object",
		`{"key":"s.txt","newKey":"d.txt","ifNoneMatch":"nope"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestCopyObjectConditional 目标端条件复制 + 412 → HTTP 412 透出（writeInternalErr 映射）。
func TestCopyObjectConditional(t *testing.T) {
	got, url := startCondFake(t, 0, "")
	env := accNewEnv(t, url, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-object",
		`{"key":"s.txt","newKey":"d.txt","ifNoneMatch":"*"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("copy status = %d body=%s", rr.Code, rr.Body.String())
	}
	// SDK 的 CopyObject 线格式是 PUT + x-amz-copy-source（带条件时 if-none-match 参与签名）。
	if got.method != http.MethodPut || got.ifNoneMatch != "*" || !strings.Contains(got.copySource, "s.txt") {
		t.Fatalf("fake saw %s %s copy-source=%q ifNoneMatch=%q, want PUT copy of s.txt with *",
			got.method, got.path, got.copySource, got.ifNoneMatch)
	}

	// 条件不满足：S3 412 → 响应 412 + 固定 precondition 文案
	_, url412 := startCondFake(t, http.StatusPreconditionFailed, "PreconditionFailed")
	env412 := accNewEnv(t, url412, "b")
	rr = env412.accDoRec("POST", "/api/accounts/"+env412.acc.ID+"/copy-object",
		`{"key":"s.txt","newKey":"d.txt","ifNoneMatch":"*"}`)
	if rr.Code != http.StatusPreconditionFailed {
		t.Fatalf("precondition status = %d, want 412 (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "precondition") {
		t.Fatalf("412 body must mention precondition: %s", rr.Body.String())
	}
}
