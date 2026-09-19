package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// syncFakeS3 同时处理 ListObjectsV2 / CopyObject / PutObject，
// 按 bucket 区分 src/dst；同一 host 表示「同 endpoint」。
func syncFakeS3(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := firstSegH(r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			prefix := r.URL.Query().Get("prefix")
			store, _ := syncBucketStore(bucket)
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			for k, e := range store {
				if prefix != "" && !strings.HasPrefix(k, prefix) {
					continue
				}
				if strings.HasSuffix(k, "/") {
					continue // 跳过目录占位
				}
				if e.deleted {
					continue
				}
				fmt.Fprintf(&sb, `<Contents><Key>%s</Key><Size>%d</Size><ETag>"%s"</ETag><LastModified>2024-01-01T00:00:00.000Z</LastModified><StorageClass>STANDARD</StorageClass></Contents>`,
					k, e.size, e.etag)
			}
			sb.WriteString(`<IsTruncated>false</IsTruncated></ListBucketResult>`)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(sb.String()))
		case r.Method == http.MethodPut:
			// CopyObject: PUT + X-Amz-Copy-Source；PutObject: 直接 PUT。
			if cs := r.Header.Get("X-Amz-Copy-Source"); cs != "" {
				ref := strings.ReplaceAll(cs, "%2F", "/")
				if idx := strings.Index(ref, "/"); idx >= 0 {
					srcB := ref[:idx]
					srcK := ref[idx+1:]
					srcMap, _ := syncBucketStore(srcB)
					if e, ok := srcMap[srcK]; ok {
						_ = syncPutObject(bucket, strings.TrimPrefix(r.URL.Path, "/"+bucket+"/"), e)
						w.Header().Set("Content-Type", "application/xml")
						_, _ = w.Write([]byte(`<CopyObjectResult><ETag>"` + e.etag + `"</ETag></CopyObjectResult>`))
						return
					}
				}
				http.NotFound(w, r)
				return
			}
			_ = syncPutObject(bucket, strings.TrimPrefix(r.URL.Path, "/"+bucket+"/"), syncEntry{size: 42, etag: "fake"})
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet:
			// 普通 GET object：为 StreamCopy（跨端点复制读取源 body）提供支持。
			key := strings.TrimPrefix(r.URL.Path, "/"+bucket+"/")
			m, _ := syncBucketStore(bucket)
			if e, ok := m[key]; ok && !e.deleted {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Length", fmt.Sprint(e.size))
				w.Header().Set("ETag", `"`+e.etag+`"`)
				_, _ = w.Write([]byte(strings.Repeat("x", int(e.size))))
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// 简化 fake 状态：每个 bucket 一个 map。加锁保证并行测试（t.Parallel）安全。
var (
	syncMu    sync.Mutex
	syncStore = map[string]map[string]syncEntry{}
)

type syncEntry struct {
	size    int64
	etag    string
	deleted bool
}

func syncBucketStore(bucket string) (map[string]syncEntry, bool) {
	syncMu.Lock()
	defer syncMu.Unlock()
	m, ok := syncStore[bucket]
	if !ok {
		m = map[string]syncEntry{}
		syncStore[bucket] = m
	}
	return m, true
}

func syncPutObject(bucket, key string, e syncEntry) error {
	m, _ := syncBucketStore(bucket)
	syncMu.Lock()
	defer syncMu.Unlock()
	m[key] = e
	return nil
}

func firstSegH(path string) string {
	path = strings.TrimPrefix(path, "/")
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return path
}

func TestMigrateSync_SkipsEqualByETag(t *testing.T) {
	syncStore = map[string]map[string]syncEntry{
		"src-bucket": {"a.txt": {size: 10, etag: "same"}, "b.txt": {size: 20, etag: "new"}},
		"dst-bucket": {"a.txt": {size: 10, etag: "same"}},
	}
	endpoint := syncFakeS3(t).URL

	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	src, _ := st.Create(&model.Account{
		Name: "src", Endpoint: endpoint, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "src-bucket", PathStyle: true,
	})
	dst, _ := st.Create(&model.Account{
		Name: "dst", Endpoint: endpoint, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "dst-bucket", PathStyle: true,
	})

	h := New(st, quietLogger(), t.TempDir(), nil, "", "test", false, false)
	srv := httptest.NewServer(h.Routes())
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"sourceAccountId": src.ID,
		"sourceBucket":    "src-bucket",
		"targetAccountId": dst.ID,
		"targetBucket":    "dst-bucket",
		"mode":            "etag",
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/migrate/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != 200 {
		t.Fatalf("status = %d", rr.StatusCode)
	}
	var resp struct {
		Scanned int `json:"scanned"`
		Skipped int `json:"skipped"`
		Copied  int `json:"copied"`
		Failed  int `json:"failed"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Scanned != 2 || resp.Skipped != 1 || resp.Copied != 1 || resp.Failed != 0 {
		t.Errorf("unexpected response: %+v", resp)
	}
	// b.txt 应已复制到 dst。
	if _, ok := syncStore["dst-bucket"]["b.txt"]; !ok {
		t.Errorf("b.txt missing in dst after sync")
	}
}

func TestMigrateSync_InvalidMode(t *testing.T) {
	st, _ := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	src, _ := st.Create(&model.Account{
		Name: "src", Endpoint: "http://x", AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "src", PathStyle: true,
	})
	dst, _ := st.Create(&model.Account{
		Name: "dst", Endpoint: "http://x", AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "dst", PathStyle: true,
	})
	h := New(st, quietLogger(), t.TempDir(), nil, "", "test", false, false)
	srv := httptest.NewServer(h.Routes())
	defer srv.Close()
	body, _ := json.Marshal(map[string]any{
		"sourceAccountId": src.ID,
		"targetAccountId": dst.ID,
		"mode":            "garbage",
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/migrate/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr, _ := srv.Client().Do(req)
	if rr.StatusCode != 400 {
		t.Errorf("invalid mode status = %d, want 400", rr.StatusCode)
	}
}

// newSyncEnv 创建 store + handler server，返回 src/dst 账号 id 与 handler server URL。
func newSyncEnv(t *testing.T, srcEp, dstEp string) (srcID, dstID, baseURL string) {
	t.Helper()
	st, _ := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	src, _ := st.Create(&model.Account{
		Name: "src", Endpoint: srcEp, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "src-bucket", PathStyle: true,
	})
	dst, _ := st.Create(&model.Account{
		Name: "dst", Endpoint: dstEp, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "dst-bucket", PathStyle: true,
	})
	h := New(st, quietLogger(), t.TempDir(), nil, "", "test", false, false)
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	return src.ID, dst.ID, srv.URL
}

// doSync 发起 /api/migrate/sync 请求并解析响应 JSON。
func doSync(t *testing.T, baseURL string, body any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+"/api/migrate/sync", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", rr.StatusCode)
	}
	var m map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return m
}

// TestMigrateSync_CompareSizeTime 验证 size_mtime 模式：size 相同（mtime 相同）即跳过，
// 不因 ETag 不同而复制（与 ETag 模式语义相反）。
func TestMigrateSync_CompareSizeTime(t *testing.T) {
	syncStore = map[string]map[string]syncEntry{
		"src-bucket": {
			"a.txt": {size: 10, etag: "src-a"},
			"b.txt": {size: 20, etag: "src-b"},
		},
		"dst-bucket": {
			"a.txt": {size: 10, etag: "dst-different"}}, // 同 size、不同 etag
	}
	endpoint := syncFakeS3(t).URL
	srcID, dstID, base := newSyncEnv(t, endpoint, endpoint)

	out := doSync(t, base, map[string]any{
		"sourceAccountId": srcID, "sourceBucket": "src-bucket",
		"targetAccountId": dstID, "targetBucket": "dst-bucket", "mode": "size_mtime",
	})
	if out["scanned"].(float64) != 2 || out["skipped"].(float64) != 1 || out["copied"].(float64) != 1 {
		t.Errorf("size_mtime: %+v", out)
	}
	if _, ok := syncStore["dst-bucket"]["b.txt"]; !ok {
		t.Errorf("b.txt missing in dst after size_mtime sync")
	}
}

// TestMigrateSync_PrefixFilter 验证 sourcePrefix 过滤：仅同步 prefix 下的对象，
// 且目标 key 是「targetPrefix + 相对路径」——与比对用的 key 必须同一个表达式，
// 否则增量同步永不收敛（docs/review-2026-09-19.md §B2）。
func TestMigrateSync_PrefixFilter(t *testing.T) {
	syncStore = map[string]map[string]syncEntry{
		"src-bucket": {
			"dir/a.txt":     {size: 10, etag: "a"},
			"dir/sub/b.txt": {size: 20, etag: "b"},
			"other.txt":     {size: 30, etag: "c"},
		},
		"dst-bucket": {},
	}
	endpoint := syncFakeS3(t).URL
	srcID, dstID, base := newSyncEnv(t, endpoint, endpoint)

	body := map[string]any{
		"sourceAccountId": srcID, "sourceBucket": "src-bucket", "sourcePrefix": "dir/",
		"targetAccountId": dstID, "targetBucket": "dst-bucket", "mode": "etag",
	}
	out := doSync(t, base, body)
	if out["scanned"].(float64) != 2 || out["copied"].(float64) != 2 {
		t.Errorf("prefix: %+v", out)
	}
	// 目标 key 是剥掉 sourcePrefix 后的相对路径（targetPrefix 为空 = 目标桶根目录）。
	for _, k := range []string{"a.txt", "sub/b.txt"} {
		if _, ok := syncStore["dst-bucket"][k]; !ok {
			t.Errorf("%s missing in dst after prefixed sync", k)
		}
	}
	if _, ok := syncStore["dst-bucket"]["dir/a.txt"]; ok {
		t.Errorf("dir/a.txt should NOT exist: sourcePrefix 应被剥掉（否则永不收敛）")
	}
	if _, ok := syncStore["dst-bucket"]["other.txt"]; ok {
		t.Errorf("other.txt should NOT be synced (outside prefix)")
	}

	// 验收：同一请求再跑一次必须收敛（copied == 0）。
	if second := doSync(t, base, body); second["copied"].(float64) != 0 {
		t.Errorf("二次同步 copied=%v, want 0（增量同步未收敛）", second["copied"])
	}
}

// TestMigrateSync_CrossEndpoint 验证异端点（不同 URL）时走 StreamCopy，仍能同步成功。
func TestMigrateSync_CrossEndpoint(t *testing.T) {
	syncStore = map[string]map[string]syncEntry{
		"src-bucket": {"a.txt": {size: 10, etag: "a"}, "b.txt": {size: 20, etag: "b"}},
		"dst-bucket": {},
	}
	srcEp := syncFakeS3(t).URL
	dstEp := syncFakeS3(t).URL
	srcID, dstID, base := newSyncEnv(t, srcEp, dstEp)

	out := doSync(t, base, map[string]any{
		"sourceAccountId": srcID, "sourceBucket": "src-bucket",
		"targetAccountId": dstID, "targetBucket": "dst-bucket", "mode": "etag",
	})
	if out["failed"].(float64) != 0 || out["copied"].(float64) != 2 {
		t.Errorf("cross-endpoint: %+v", out)
	}
	for _, k := range []string{"a.txt", "b.txt"} {
		if _, ok := syncStore["dst-bucket"][k]; !ok {
			t.Errorf("%s missing in dst after cross-endpoint sync", k)
		}
	}
}
