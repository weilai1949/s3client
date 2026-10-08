package handler

// objects_copy_test.go —— 对象复制族（单对象复制 / 批量复制 / 删除前缀 / 异步 / 前缀复制）的契约测试。
// 自 objects_test.go 拆出；文件头、夹具与跨文件 helper 仍在 objects_test.go。
import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/service"
	"github.com/weilai1949/s3client/apps/server/internal/store"
)

// TestCopyObject 用假 S3 验证：复制单对象到目标桶/键且不删除源。
func TestCopyObject(t *testing.T) {
	var (
		mu          sync.Mutex
		gotSrc      string
		deleteCalls int
	)
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			mu.Lock()
			gotSrc = r.Header.Get("X-Amz-Copy-Source")
			mu.Unlock()
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`)
		case http.MethodDelete:
			mu.Lock()
			deleteCalls++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
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

	// 复制（同桶，新 key）
	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-object", `{"key":"a.txt","newKey":"archive/a.txt"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("copy-object status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Copied string `json:"copied"`
		Bucket string `json:"bucket"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body: %v", err)
	}
	if resp.Copied != "archive/a.txt" || resp.Bucket != "b" {
		t.Fatalf("copied=%q bucket=%q, want archive/a.txt b", resp.Copied, resp.Bucket)
	}
	mu.Lock()
	src, dels := gotSrc, deleteCalls
	mu.Unlock()
	if src == "" {
		t.Fatalf("expected Copy-Source header, got empty")
	}
	if dels != 0 {
		t.Fatalf("copy-object must not delete source, got %d delete calls", dels)
	}

	// 跨桶复制
	rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-object", `{"key":"a.txt","newBucket":"b2","newKey":"a.txt"}`)
	if rr2.Code != http.StatusOK {
		t.Fatalf("cross-bucket copy-object status = %d, body=%s", rr2.Code, rr2.Body.String())
	}

	// 同桶同 key → 400
	if rr3 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-object", `{"key":"a.txt","newKey":"a.txt"}`); rr3.Code != http.StatusBadRequest {
		t.Fatalf("copy to self = %d, want 400", rr3.Code)
	}
	// key 缺失 → 400
	if rr4 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-object", `{"key":"a.txt"}`); rr4.Code != http.StatusBadRequest {
		t.Fatalf("copy without newKey = %d, want 400", rr4.Code)
	}
}

// TestCopyMany 用假 S3 验证：批量复制/移动所选文件（目标前缀 + 保留文件名，deleteSource 删除源，部分失败）。
func TestCopyMany(t *testing.T) {
	var (
		mu       sync.Mutex
		copySrcs []string
		putPaths []string
		delPaths []string
	)
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			mu.Lock()
			copySrcs = append(copySrcs, r.Header.Get("X-Amz-Copy-Source"))
			putPaths = append(putPaths, r.URL.Path)
			mu.Unlock()
			if strings.Contains(r.Header.Get("X-Amz-Copy-Source"), "bad") {
				w.WriteHeader(http.StatusNotFound)
				io.WriteString(w, `<?xml version="1.0"?><Error><Code>NoSuchKey</Code></Error>`)
				return
			}
			io.WriteString(w, `<?xml version="1.0"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`)
		case http.MethodDelete:
			mu.Lock()
			delPaths = append(delPaths, r.URL.Path)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
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

	// 复制模式：目标 = targetPrefix + basename；bad.txt 失败但不中断其余
	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-objects",
		`{"keys":["dir/a.txt","dir/b.txt","dir/bad.txt"],"targetPrefix":"archive/"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("copy-objects status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Copied     int      `json:"copied"`
		Failed     int      `json:"failed"`
		FailedKeys []string `json:"failedKeys"`
		LastError  string   `json:"lastError"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body: %v", err)
	}
	if resp.Copied != 2 || resp.Failed != 1 || resp.LastError == "" {
		t.Fatalf("copied=%d failed=%d lastError=%q, want 2/1/nonempty", resp.Copied, resp.Failed, resp.LastError)
	}
	if len(resp.FailedKeys) != 1 || resp.FailedKeys[0] != "dir/bad.txt" {
		t.Fatalf("failedKeys = %v, want [dir/bad.txt]", resp.FailedKeys)
	}
	mu.Lock()
	if len(putPaths) != 3 {
		t.Fatalf("putPaths = %v, want 3 copy attempts", putPaths)
	}
	// 目标路径含 targetPrefix + basename（path-style：/b/archive/a.txt 等）
	if !containsStr(putPaths, "/b/archive/a.txt") || !containsStr(putPaths, "/b/archive/b.txt") || !containsStr(putPaths, "/b/archive/bad.txt") {
		t.Fatalf("putPaths = %v, want /b/archive/{a,b,bad}.txt", putPaths)
	}
	if len(delPaths) != 0 {
		t.Fatalf("delPaths = %v, want no delete in copy mode", delPaths)
	}
	mu.Unlock()

	// 移动模式：复制成功后删除源（跨桶，保留文件名）
	copySrcs, putPaths, delPaths = nil, nil, nil
	rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-objects",
		`{"keys":["dir/a.txt"],"targetBucket":"b2","targetPrefix":"dest/","deleteSource":true}`)
	if rr2.Code != http.StatusOK {
		t.Fatalf("move status = %d, body=%s", rr2.Code, rr2.Body.String())
	}
	var resp2 struct {
		Copied int `json:"copied"`
		Failed int `json:"failed"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil || resp2.Copied != 1 || resp2.Failed != 0 {
		t.Fatalf("move resp = %+v, err=%v, want copied=1 failed=0", resp2, err)
	}
	mu.Lock()
	if !containsStr(putPaths, "/b2/dest/a.txt") {
		t.Fatalf("putPaths = %v, want /b2/dest/a.txt", putPaths)
	}
	if !containsStr(delPaths, "/b/dir/a.txt") {
		t.Fatalf("delPaths = %v, want source delete /b/dir/a.txt", delPaths)
	}
	mu.Unlock()

	// keys 缺失 → 400
	if rr3 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-objects", `{}`); rr3.Code != http.StatusBadRequest {
		t.Fatalf("copy-objects without keys = %d, want 400", rr3.Code)
	}
}

// TestDeletePrefix 用假 S3 验证递归删除：分页列出 + 批量删除，返回删除数。
func TestDeletePrefix(t *testing.T) {
	page1 := []string{"dir/a.txt", "dir/b.txt"}
	page2 := []string{"dir/sub/c.txt"}
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			// ListObjectsV2：第一页截断，第二页结束
			if r.URL.Query().Get("continuation-token") == "tok2" {
				io.WriteString(w, listBucketXML(page2, false, ""))
			} else {
				io.WriteString(w, listBucketXML(page1, true, "tok2"))
			}
		case r.Method == http.MethodPost && (r.URL.Query().Has("x-id") || r.URL.Query().Has("delete")):
			// DeleteObjects：返回与请求等量的 Deleted 条目
			body, _ := io.ReadAll(r.Body)
			n := strings.Count(string(body), "<Key>")
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0"?><DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			for i := 0; i < n; i++ {
				sb.WriteString("<Deleted><Key>x</Key></Deleted>")
			}
			sb.WriteString("</DeleteResult>")
			io.WriteString(w, sb.String())
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
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

	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/delete-prefix", `{"prefix":"dir/"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete-prefix status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Deleted   int  `json:"deleted"`
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body: %v", err)
	}
	if resp.Deleted != 3 || resp.Truncated {
		t.Fatalf("deleted = %d truncated=%v, want 3 false", resp.Deleted, resp.Truncated)
	}
	// 空前缀 → 400（防止误删全桶）
	if rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/delete-prefix", `{"prefix":""}`); rr2.Code != http.StatusBadRequest {
		t.Fatalf("delete-prefix empty prefix = %d, want 400", rr2.Code)
	}
}

// TestCopyManyAsyncAndDeletePrefixAsync 验证批量复制/删除前缀异步任务复用 migrate job SSE。
func TestCopyManyAsyncAndDeletePrefixAsync(t *testing.T) {
	s3copy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.Header.Get("X-Amz-Copy-Source") != "" {
			io.WriteString(w, `<?xml version="1.0"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer s3copy.Close()

	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	acc, err := st.Create(&model.Account{
		Name: "fake", Endpoint: s3copy.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(st, logger, t.TempDir(), nil, "", "test", false, false).Routes()

	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-objects/async",
		`{"keys":["a.txt","b.txt"],"targetPrefix":"out/"}`)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("copy async = %d body=%s", rr.Code, rr.Body.String())
	}
	var start struct {
		JobID string `json:"jobId"`
		Total int    `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &start); err != nil || start.JobID == "" || start.Total != 2 {
		t.Fatalf("start=%+v err=%v", start, err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/migrate/jobs/"+start.JobID+"/events", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	sc := bufio.NewScanner(w.Body)
	var last service.JobProgress
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "data: ") {
			_ = json.Unmarshal([]byte(line[6:]), &last)
		}
	}
	if last.Status != "done" || last.Migrated != 2 {
		t.Fatalf("copy progress=%+v", last)
	}

	// delete-prefix/async
	s3del := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			io.WriteString(w, listBucketXML([]string{"dir/a.txt", "dir/b.txt"}, false, ""))
		case r.Method == http.MethodPost && r.URL.Query().Has("delete"):
			io.WriteString(w, `<?xml version="1.0"?><DeleteResult><Deleted><Key>dir/a.txt</Key></Deleted><Deleted><Key>dir/b.txt</Key></Deleted></DeleteResult>`)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer s3del.Close()
	acc2, err := st.Create(&model.Account{
		Name: "del", Endpoint: s3del.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create2: %v", err)
	}
	rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc2.ID+"/delete-prefix/async", `{"prefix":"dir/"}`)
	if rr2.Code != http.StatusAccepted {
		t.Fatalf("delete async = %d body=%s", rr2.Code, rr2.Body.String())
	}
	var dstart struct {
		JobID string `json:"jobId"`
		Total int    `json:"total"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &dstart); err != nil || dstart.Total != 2 {
		t.Fatalf("dstart=%+v err=%v", dstart, err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/migrate/jobs/"+dstart.JobID+"/events", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	sc2 := bufio.NewScanner(w2.Body)
	var dlast service.JobProgress
	for sc2.Scan() {
		line := sc2.Text()
		if strings.HasPrefix(line, "data: ") {
			_ = json.Unmarshal([]byte(line[6:]), &dlast)
		}
	}
	if dlast.Status != "done" || dlast.Migrated != 2 {
		t.Fatalf("delete progress=%+v", dlast)
	}
}

// TestCopyPrefix 用假 S3 验证递归复制：分页列出 + 逐 key CopyObject，目标 key 前缀拼接正确。
func TestCopyPrefix(t *testing.T) {
	page1 := []string{"src/a.txt", "src/b.txt"}
	page2 := []string{"src/sub/c.txt"}
	var (
		copyMu      sync.Mutex
		copySources []string
	)
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			if r.URL.Query().Get("continuation-token") == "tok2" {
				io.WriteString(w, listBucketXML(page2, false, ""))
			} else {
				io.WriteString(w, listBucketXML(page1, true, "tok2"))
			}
		case r.Method == http.MethodPut && r.Header.Get("X-Amz-Copy-Source") != "":
			copyMu.Lock()
			copySources = append(copySources, r.Header.Get("X-Amz-Copy-Source"))
			copyMu.Unlock()
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
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

	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-prefix",
		`{"prefix":"src/","targetPrefix":"dst/"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("copy-prefix status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Copied int `json:"copied"`
		Failed int `json:"failed"`
		Total  int `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body: %v", err)
	}
	if resp.Copied != 3 || resp.Failed != 0 || resp.Total != 3 {
		t.Fatalf("copied=%d failed=%d total=%d, want 3/0/3", resp.Copied, resp.Failed, resp.Total)
	}
	// 同桶目标前缀与源重叠 → 400（防无限复制）
	if rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-prefix",
		`{"prefix":"src/","targetPrefix":"src/archive/"}`); rr2.Code != http.StatusBadRequest {
		t.Fatalf("overlap copy = %d, want 400, body=%s", rr2.Code, rr2.Body.String())
	}
	// 跨桶同前缀不重叠 → 允许（不同桶无循环风险）
	if rr3 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/copy-prefix",
		`{"prefix":"src/","targetBucket":"b2","targetPrefix":"src/"}`); rr3.Code != http.StatusOK {
		t.Fatalf("cross-bucket same prefix = %d, want 200, body=%s", rr3.Code, rr3.Body.String())
	}
}
