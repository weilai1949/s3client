package handler

// objects_download_test.go —— 下载 ZIP、代理下载（inline / attachment）、文本截断、文件名消毒与
// SetHeaders 的契约测试。自 objects_test.go 拆出；夹具与跨文件 helper 仍在 objects_test.go。
import (
	"archive/zip"
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// TestDownloadZip 用假 S3 验证：多个对象流式打包为 ZIP，失败对象写入清单。
func TestDownloadZip(t *testing.T) {
	// key -> 内容；bad.txt 返回 404
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		switch {
		case strings.Contains(r.URL.Path, "bad.txt"):
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `<?xml version="1.0"?><Error><Code>NoSuchKey</Code></Error>`)
		case strings.Contains(r.URL.Path, "hello.txt"):
			w.Header().Set("Content-Type", "text/plain")
			io.WriteString(w, "hello")
		case strings.Contains(r.URL.Path, "dir/nested.txt"):
			w.Header().Set("Content-Type", "text/plain")
			io.WriteString(w, "nested!")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s3fake.Close()

	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "fake", Endpoint: s3fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()

	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/download-zip",
		`{"keys":["hello.txt","dir/nested.txt","bad.txt"]}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("zip status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("Content-Type = %q, want application/zip", ct)
	}
	zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("parse zip: %v", err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(b)
	}
	if got["hello.txt"] != "hello" || got["dir/nested.txt"] != "nested!" {
		t.Fatalf("zip contents = %v", got)
	}
	if !strings.Contains(got["_下载失败清单.txt"], "bad.txt") {
		t.Fatalf("fail list missing bad.txt: %v", got)
	}
	// keys 缺失 → 400
	if rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/download-zip", `{}`); rr2.Code != http.StatusBadRequest {
		t.Fatalf("zip without keys = %d, want 400", rr2.Code)
	}
}

// TestProxyDownload 用假 S3 验证代理下载：attachment 头、Content-Type 透传、Range 转发。
func TestProxyDownload(t *testing.T) {
	var gotRange string
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		gotRange = r.Header.Get("Range")
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Length", "5")
		io.WriteString(w, "a,b,c")
	}))
	defer s3fake.Close()

	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "fake", Endpoint: s3fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()

	req := httptest.NewRequest("GET", "/api/accounts/"+acc.ID+"/proxy?bucket=b&key=report.csv&mode=download", nil)
	req.Header.Set("Range", "bytes=0-99")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("proxy status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if gotRange != "bytes=0-99" {
		t.Fatalf("Range forwarded = %q, want bytes=0-99", gotRange)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("Content-Type = %q, want text/csv", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment;") {
		t.Fatalf("Content-Disposition = %q, want attachment", cd)
	}
	if rr.Body.String() != "a,b,c" {
		t.Fatalf("body = %q", rr.Body.String())
	}
	// key 缺失 → 400；无效 mode → 400
	if rr2 := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/proxy?bucket=b", ""); rr2.Code != http.StatusBadRequest {
		t.Fatalf("proxy without key = %d, want 400", rr2.Code)
	}
	if rr3 := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/proxy?bucket=b&key=x&mode=bad", ""); rr3.Code != http.StatusBadRequest {
		t.Fatalf("proxy bad mode = %d, want 400", rr3.Code)
	}
	// 对象不存在 → 404
	s3fake2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `<?xml version="1.0"?><Error><Code>NoSuchKey</Code></Error>`)
	}))
	defer s3fake2.Close()
	st2, _ := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	acc2, _ := st2.Create(&model.Account{
		Name: "fake", Endpoint: s3fake2.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	h2 := New(st2, logger, t.TempDir(), nil, "", "test", false, false).Routes()
	if rr4 := doJSON(t, h2, "GET", "/api/accounts/"+acc2.ID+"/proxy?bucket=b&key=missing", ""); rr4.Code != http.StatusNotFound {
		t.Fatalf("proxy missing = %d, want 404", rr4.Code)
	}
}

// TestProxyInline 验证 inline 模式透传 Content-Disposition: inline。
func TestProxyInline(t *testing.T) {
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		io.WriteString(w, "PNGDATA")
	}))
	defer s3fake.Close()
	st, _ := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	acc, _ := st.Create(&model.Account{
		Name: "fake", Endpoint: s3fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()
	rr := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/proxy?bucket=b&key=a.png&mode=inline", "")
	if cd := rr.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "inline;") {
		t.Fatalf("Content-Disposition = %q, want inline", cd)
	}
	if rr.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", rr.Header().Get("Content-Type"))
	}
}

// TestProxyTextTruncate 验证文本预览：强制 text/plain + nosniff，超限截断并标记。
func TestProxyTextTruncate(t *testing.T) {
	big := strings.Repeat("x", 4096)
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html") // 源是 HTML，代理必须降级为纯文本
		io.WriteString(w, big)
	}))
	defer s3fake.Close()
	st, _ := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	acc, _ := st.Create(&model.Account{
		Name: "fake", Endpoint: s3fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()

	rr := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/proxy?bucket=b&key=page.html&mode=text&maxBytes=1024", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("text status = %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain", ct)
	}
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff")
	}
	if rr.Header().Get("X-Preview-Truncated") != "1" {
		t.Fatalf("missing X-Preview-Truncated")
	}
	if len(rr.Body.Bytes()) != 1024 {
		t.Fatalf("body len = %d, want 1024", len(rr.Body.Bytes()))
	}

	// 小文件不截断
	rr2 := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/proxy?bucket=b&key=small.txt&mode=text", "")
	if rr2.Header().Get("X-Preview-Truncated") != "" {
		t.Fatalf("small file should not be truncated")
	}
	if len(rr2.Body.Bytes()) != 4096 {
		t.Fatalf("small body len = %d, want 4096", len(rr2.Body.Bytes()))
	}
}

// TestSanitizeFilename 验证 Content-Disposition 文件名清洗。
func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"a.txt":         "a.txt",
		`../"evil".txt`: "..__evil_.txt",
		"a/b.txt":       "a_b.txt",
		"":              "download",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSetHeaders 用假 S3 验证设置对象 HTTP 头：metadata-directive=REPLACE + Content-Type/元数据。
func TestSetHeaders(t *testing.T) {
	var (
		gotDirective string
		gotCT        string
		gotMeta      string
	)
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		gotDirective = r.Header.Get("X-Amz-Metadata-Directive")
		gotCT = r.Header.Get("Content-Type")
		gotMeta = r.Header.Get("X-Amz-Meta-Owner")
		io.WriteString(w, `<?xml version="1.0"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`)
	}))
	defer s3fake.Close()

	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "fake", Endpoint: s3fake.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()

	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/set-headers",
		`{"key":"a.txt","contentType":"text/markdown","metadata":{"owner":"alice"}}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("set-headers status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if gotDirective != "REPLACE" {
		t.Fatalf("directive = %q, want REPLACE", gotDirective)
	}
	if gotCT != "text/markdown" {
		t.Fatalf("content-type = %q, want text/markdown", gotCT)
	}
	if gotMeta != "alice" {
		t.Fatalf("meta = %q, want alice", gotMeta)
	}
	// key 缺失 → 400
	if rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/set-headers", `{}`); rr2.Code != http.StatusBadRequest {
		t.Fatalf("set-headers without key = %d, want 400", rr2.Code)
	}
}
