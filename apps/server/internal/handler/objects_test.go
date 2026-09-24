package handler

// objects_test.go —— 对象基础操作（HEAD 详情 / 建目录 / 重命名 / 列表与批量删除）的契约测试。
// 复制与前缀复制族见 objects_copy_test.go；下载 ZIP、代理下载与文件名消毒见 objects_download_test.go。
import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// TestHeadObject 用假 S3 验证 HeadObject 详情返回与 404 语义。
func TestHeadObject(t *testing.T) {
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if strings.Contains(r.URL.Path, "missing") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `<?xml version="1.0"?><Error><Code>NoSuchKey</Code><Message>no such key</Message></Error>`)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Content-Length", "5")
		w.Header().Set("Last-Modified", "Mon, 01 Jan 2026 00:00:00 GMT")
		w.Header().Set("x-amz-meta-owner", "alice")
		w.Header().Set("x-amz-storage-class", "STANDARD_IA")
		w.WriteHeader(http.StatusOK)
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

	rr := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/head?bucket=b&key=hello.txt", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("head status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("head body: %v", err)
	}
	if m["contentType"] != "text/plain" || m["etag"] != `"abc123"` || m["size"] != float64(5) {
		t.Fatalf("unexpected head fields: %+v", m)
	}
	if m["storageClass"] != "STANDARD_IA" {
		t.Fatalf("storageClass = %v, want STANDARD_IA", m["storageClass"])
	}
	if md, ok := m["metadata"].(map[string]any); !ok || md["owner"] != "alice" {
		t.Fatalf("metadata = %v, want owner=alice", m["metadata"])
	}

	// 对象不存在 → 404
	rr2 := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/head?bucket=b&key=missing.txt", "")
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("head missing status = %d, want 404, body=%s", rr2.Code, rr2.Body.String())
	}
	// key 缺失 → 400
	if rr3 := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/head?bucket=b", ""); rr3.Code != http.StatusBadRequest {
		t.Fatalf("head without key = %d, want 400", rr3.Code)
	}
}

// TestMkdirObject 用假 S3 验证：PUT 空对象创建 key 以 / 结尾的"文件夹"。
func TestMkdirObject(t *testing.T) {
	var gotPath string
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("mkdir body should be empty, got %q", body)
		}
		w.Header().Set("ETag", `"d41d8cd98f00b204e9800998ecf8427e"`)
		w.WriteHeader(http.StatusOK)
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

	// key 不带斜杠 → 自动补全为目录形式
	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/mkdir", `{"key":"images"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("mkdir status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if gotPath != "/b/images/" {
		t.Fatalf("mkdir path = %q, want /b/images/", gotPath)
	}
	var resp struct {
		Created string `json:"created"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Created != "images/" {
		t.Fatalf("created = %q, want images/", resp.Created)
	}
	// key 缺失 → 400
	if rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/mkdir", `{}`); rr2.Code != http.StatusBadRequest {
		t.Fatalf("mkdir without key = %d, want 400", rr2.Code)
	}
}

// TestRenameObject 用假 S3 验证：先 CopyObject 成功后才 DeleteObject 源。
func TestRenameObject(t *testing.T) {
	var calls []string
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut: // CopyObject（同桶）与 mkdir 一样走 PUT，用 Copy-Source 区分
			if src := r.Header.Get("X-Amz-Copy-Source"); src != "" {
				calls = append(calls, "copy:"+src)
				if strings.Contains(src, "bad") {
					w.WriteHeader(http.StatusNotFound)
					io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code></Error>`)
					return
				}
				io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodDelete:
			calls = append(calls, "delete:"+r.URL.Path)
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

	// 正常重命名：copy 先于 delete，响应新 key
	rr := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/rename", `{"key":"old.txt","newKey":"dir/new.txt"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("rename status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if len(calls) != 2 || !strings.HasPrefix(calls[0], "copy:") || !strings.HasPrefix(calls[1], "delete:") {
		t.Fatalf("calls = %v, want copy then delete", calls)
	}
	var resp struct {
		Renamed string `json:"renamed"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Renamed != "dir/new.txt" {
		t.Fatalf("renamed = %q, want dir/new.txt", resp.Renamed)
	}

	// 复制失败 → 404（对象不存在，经 s3HTTPStatus 映射）且不删除源
	calls = nil
	rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/rename", `{"key":"bad.txt","newKey":"x.txt"}`)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("rename fail status = %d, want 404", rr2.Code)
	}
	if len(calls) != 1 || !strings.HasPrefix(calls[0], "copy:") {
		t.Fatalf("calls = %v, want only copy attempt", calls)
	}

	// 同桶内新 key 与旧 key 相同 → 400
	if rr3 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/rename", `{"key":"a.txt","newKey":"a.txt"}`); rr3.Code != http.StatusBadRequest {
		t.Fatalf("rename same key = %d, want 400", rr3.Code)
	}

	// 跨桶同名移动 → 允许（copy 到目标桶 + 删除源）
	calls = nil
	rr4 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/rename", `{"key":"a.txt","newBucket":"b2","newKey":"a.txt"}`)
	if rr4.Code != http.StatusOK {
		t.Fatalf("cross-bucket same-key rename = %d, want 200, body=%s", rr4.Code, rr4.Body.String())
	}
	if len(calls) != 2 || !strings.HasPrefix(calls[0], "copy:") || !strings.HasPrefix(calls[1], "delete:") {
		t.Fatalf("calls = %v, want copy then delete", calls)
	}
}

// TestListObjectsAndDeleteObjects 补测：主浏览端点（ListObjectsV2）与批量删除（DeleteObjects）。
func TestListObjectsAndDeleteObjects(t *testing.T) {
	var (
		mu        sync.Mutex
		delBodies []string
	)
	s3fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Query().Has("delete") {
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			delBodies = append(delBodies, string(b))
			mu.Unlock()
			io.WriteString(w, `<DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Deleted><Key>a.txt</Key></Deleted><Deleted><Key>b.txt</Key></Deleted></DeleteResult>`)
			return
		}
		if r.Method == http.MethodGet && r.URL.Query().Has("list-type") {
			// 含 StorageClass 的 ListObjectsV2 响应
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>b</Name><Prefix></Prefix><KeyCount>2</KeyCount><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated><Contents><Key>a.txt</Key><Size>3</Size><ETag>&quot;e1&quot;</ETag><StorageClass>STANDARD</StorageClass><LastModified>2026-01-01T00:00:00.000Z</LastModified></Contents><Contents><Key>dir/c.txt</Key><Size>4</Size><ETag>&quot;e2&quot;</ETag><StorageClass>STANDARD_IA</StorageClass><LastModified>2026-01-02T00:00:00.000Z</LastModified></Contents></ListBucketResult>`)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
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

	// 列出对象：objects + storageClass + commonPrefixes
	rr := doJSON(t, h, "GET", "/api/accounts/"+acc.ID+"/objects?bucket=b&prefix=&delimiter=/", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Objects []struct {
			Key          string `json:"key"`
			Size         int64  `json:"size"`
			StorageClass string `json:"storageClass"`
		} `json:"objects"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("list body: %v", err)
	}
	if len(resp.Objects) != 2 || resp.Objects[0].Key != "a.txt" || resp.Objects[0].StorageClass != "STANDARD" {
		t.Fatalf("objects = %+v", resp.Objects)
	}
	if resp.Objects[1].StorageClass != "STANDARD_IA" {
		t.Fatalf("objects[1].storageClass = %q, want STANDARD_IA", resp.Objects[1].StorageClass)
	}

	// 批量删除：POST ?delete（XML body）
	rr2 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/delete", `{"bucket":"b","keys":["a.txt","b.txt"]}`)
	if rr2.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var del struct {
		Deleted int `json:"deleted"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &del)
	mu.Lock()
	gotBodies := delBodies
	mu.Unlock()
	if del.Deleted != 2 || len(gotBodies) != 1 || !strings.Contains(gotBodies[0], "<Key>a.txt</Key>") {
		t.Fatalf("deleted=%d bodies=%v", del.Deleted, gotBodies)
	}
	// 缺 keys → 400
	if rr3 := doJSON(t, h, "POST", "/api/accounts/"+acc.ID+"/delete", `{"bucket":"b","keys":[]}`); rr3.Code != http.StatusBadRequest {
		t.Fatalf("delete empty keys=%d, want 400", rr3.Code)
	}
}
