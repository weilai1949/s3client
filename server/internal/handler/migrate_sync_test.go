package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weilai1949/s3clinet/server/internal/model"
	"github.com/weilai1949/s3clinet/server/internal/store"
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
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// 简化 fake 状态：每个 bucket 一个 map；测试不要求多实例并发安全。
var (
	syncStore = map[string]map[string]syncEntry{}
)

type syncEntry struct {
	size    int64
	etag    string
	deleted bool
}

func syncBucketStore(bucket string) (map[string]syncEntry, bool) {
	m, ok := syncStore[bucket]
	if !ok {
		m = map[string]syncEntry{}
		syncStore[bucket] = m
	}
	return m, true
}

func syncPutObject(bucket, key string, e syncEntry) error {
	m, _ := syncBucketStore(bucket)
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

	h := New(st, quietLogger(), t.TempDir(), nil, "", "test", false)
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
		Scanned int    `json:"scanned"`
		Skipped int    `json:"skipped"`
		Copied  int    `json:"copied"`
		Failed  int    `json:"failed"`
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
	h := New(st, quietLogger(), t.TempDir(), nil, "", "test", false)
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
