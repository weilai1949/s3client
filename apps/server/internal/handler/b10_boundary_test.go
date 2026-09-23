package handler

// b10_boundary_test.go —— docs/archive/review-2026-09-19.md §B10 的四个端点级边界缺陷：
// ② filename* 编码、③ ZIP 空 key、④ text 预览读错误、⑤ 分段顺序。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOlProxyContentDispositionRFC5987 filename* 必须用 RFC 5987 百分号编码：
// 空格要编成 %20，而不是 url.QueryEscape 的 "+"（浏览器会存成 my+file.txt）。
func TestOlProxyContentDispositionRFC5987(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		return olResp{
			status:  http.StatusOK,
			headers: map[string]string{"Content-Type": "text/plain"},
			body:    "hi",
		}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/proxy?bucket=b&key=docs/my%20file%20文档.txt&mode=download", "")
	olExpectStatus(t, rr, http.StatusOK, "download")
	disp := rr.Header().Get("Content-Disposition")
	const want = `filename*=UTF-8''my%20file%20%E6%96%87%E6%A1%A3.txt`
	if !strings.Contains(disp, want) {
		t.Fatalf("Content-Disposition = %q, want it to contain %q", disp, want)
	}
	if strings.Contains(disp, "my+file") {
		t.Fatalf("Content-Disposition = %q 不得用 '+' 表示空格", disp)
	}
}

// TestOlProxyTextShortBodyIsNot200 上游声明的 Content-Length 大于实际返回的字节数时，
// 不能把截断的预览当完整内容回 200。
func TestOlProxyTextShortBodyIsNot200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("hijack unsupported")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer conn.Close()
		// 声明 100 字节却只写 10 字节后断开 → 客户端读到 ErrUnexpectedEOF。
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 100\r\n\r\n0123456789")
		_ = buf.Flush()
	}))
	t.Cleanup(srv.Close)

	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/proxy?bucket=b&key=k&mode=text", "")
	if rr.Code == http.StatusOK {
		t.Fatalf("status = 200，截断的预览不得当作完整内容返回（body=%q）", rr.Body.String())
	}
}

// TestOlProxyTextMalformedChunkIsNot200 传输层读取错误（畸形 chunked 编码）不能被当成
// 「内容只有这么多」——它既不是 EOF 也不是短读，必须回错误状态（review §B10④）。
func TestOlProxyTextMalformedChunkIsNot200(t *testing.T) {
	srv := hijackResponse(t, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nTransfer-Encoding: chunked\r\n\r\n"+
		"a\r\n0123456789\r\n"+ // 合法的第一个 chunk
		"ZZZZ\r\n") // 非法 chunk 长度 → 客户端读取报错
	t.Cleanup(srv.Close)

	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("GET", "/api/accounts/"+env.acc.ID+"/proxy?bucket=b&key=k&mode=text", "")
	if rr.Code == http.StatusOK {
		t.Fatalf("status = 200，读取失败不得当作完整预览返回（body=%q）", rr.Body.String())
	}
}

// hijackResponse 起一个假 S3：把 raw 逐字写回并关闭连接（用于构造畸形/截断的响应）。
func hijackResponse(t *testing.T, raw string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("hijack unsupported")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer conn.Close()
		_, _ = buf.WriteString(raw)
		_ = buf.Flush()
	}))
}

// TestOlDownloadZipRejectsEmptyKey 空 key 会被 S3 当成「列举桶」，把 ListBucket XML 塞进 ZIP。
func TestOlDownloadZipRejectsEmptyKey(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		return olXML(http.StatusOK, listBucketXML([]string{"leaked.txt"}, false, ""))
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/download-zip", `{"bucket":"b","keys":["","a.txt"]}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "empty key in zip")
}

// TestOlMultipartCompleteRequiresAscendingParts 段号乱序 → 400（而不是让 S3 回 InvalidPartOrder 变 500）。
func TestOlMultipartCompleteRequiresAscendingParts(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		return olXML(http.StatusOK, `<?xml version="1.0"?><CompleteMultipartUploadResult><ETag>"e"</ETag></CompleteMultipartUploadResult>`)
	})
	env := accNewEnv(t, srv.URL, "b")
	body := `{"bucket":"b","key":"big.bin","uploadId":"U1","parts":[{"partNumber":2,"etag":"e2"},{"partNumber":1,"etag":"e1"}]}`
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/multipart/complete", body)
	olExpectStatus(t, rr, http.StatusBadRequest, "unordered parts")

	// 升序则放行（同一 fake）。
	ok := `{"bucket":"b","key":"big.bin","uploadId":"U1","parts":[{"partNumber":1,"etag":"e1"},{"partNumber":2,"etag":"e2"}]}`
	rr = env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/multipart/complete", ok)
	olExpectStatus(t, rr, http.StatusOK, "ordered parts")
}
