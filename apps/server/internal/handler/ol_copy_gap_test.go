package handler

// ol_copy_gap_test.go —— copy.go 剩余分支：failedKeys 截断 / 异步校验与空 key 跳过 / 异步取消。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/service"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// TestOlCopyManyFailKeysAll 201 个 key 全部删源失败 → 响应 200，failed=201，
// 而 failedKeys 必须裁到 maxFailKeys（200）：这是对外的承诺（review §7.3 D4）。
// 此前 copy/migrate/sync 原样回传、只有 delete-prefix 异步路径裁剪，承诺与实现不符。
func TestOlCopyManyFailKeysAll(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK) // 复制全部成功
		}
		return olErr(http.StatusForbidden, "AccessDenied") // 删源全部失败
	})
	env := accNewEnv(t, srv.URL, "b")
	keys := make([]string, 201)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%03d", i)
	}
	b, _ := json.Marshal(map[string]any{"bucket": "b", "keys": keys, "deleteSource": true})
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-objects", string(b))
	olExpectStatus(t, rr, http.StatusOK, "fail keys all")
	var m map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &m)
	if int(m["failed"].(float64)) != 201 {
		t.Fatalf("failed = %v", m["failed"])
	}
	// 计数不裁剪（failed=201 是真实失败数），只有列表裁剪。
	if fk, _ := m["failedKeys"].([]any); len(fk) != maxFailKeys {
		t.Fatalf("failedKeys len = %d, want %d（承诺上限）", len(fk), maxFailKeys)
	}
}

// TestOlMigrateSyncFailKeysCapped 增量同步同样兑现 failedKeys ≤ 200 承诺（review §7.3 D4）：
// 源端 201 个对象全部需要复制、复制一律失败 → failedKeys 必须被裁到上限。
func TestOlMigrateSyncFailKeysCapped(t *testing.T) {
	syncStore = map[string]map[string]syncEntry{}
	for i := 0; i < 201; i++ {
		syncStore["cap-src"] = syncBucketOrCreate("cap-src")
		syncStore["cap-src"][fmt.Sprintf("k%03d", i)] = syncEntry{size: 10, etag: "e"}
	}
	// 目标端只提供 GET（列举），不提供 PUT → 复制全部失败。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			bucket := firstSegH(r.URL.Path)
			m, _ := syncBucketStore(bucket)
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			if bucket == "cap-src" {
				for k, e := range m {
					fmt.Fprintf(&sb, `<Contents><Key>%s</Key><Size>%d</Size><ETag>"%s"</ETag><LastModified>2024-01-01T00:00:00.000Z</LastModified></Contents>`, k, e.size, e.etag)
				}
			}
			sb.WriteString(`<IsTruncated>false</IsTruncated></ListBucketResult>`)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(sb.String()))
			return
		}
		// HEAD（目标端比对）返回 404 → 视为不存在，需要复制；PUT 一律 403 → 复制失败。
		if r.Method == http.MethodHead {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
	}))
	t.Cleanup(srv.Close)

	st, err := store.New(filepath.Join(t.TempDir(), "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	src, _ := st.Create(&model.Account{
		Name: "src", Endpoint: srv.URL, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "cap-src", PathStyle: true,
	})
	dst, _ := st.Create(&model.Account{
		Name: "dst", Endpoint: srv.URL, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "cap-dst", PathStyle: true,
	})
	h := New(st, quietLogger(), t.TempDir(), nil, "", "test", false, false)
	hs := httptest.NewServer(h.Routes())
	defer hs.Close()

	body, _ := json.Marshal(map[string]any{
		"sourceAccountId": src.ID, "sourceBucket": "cap-src",
		"targetAccountId": dst.ID, "targetBucket": "cap-dst",
		"mode": "always",
	})
	req, _ := http.NewRequest("POST", hs.URL+"/api/migrate/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var m map[string]any
	_ = json.NewDecoder(res.Body).Decode(&m)
	if failed, _ := m["failed"].(float64); failed == 0 {
		t.Fatalf("expected 全部失败，got %v", m)
	}
	fk, _ := m["failedKeys"].([]any)
	if len(fk) == 0 {
		t.Fatalf("expected failedKeys 非空：%v", m)
	}
	if len(fk) > maxFailKeys {
		t.Fatalf("failedKeys len = %d, want ≤ %d（D4 承诺）", len(fk), maxFailKeys)
	}
}

// TestOlMigrateAsyncFailKeysCappedBeforePersist 异步迁移：failedKeys 必须在 Finish（落盘
// jobs.json）之前裁到上限。这是 D4 的原始风险点——10 万个 key 全失败时未裁剪的 FailKeys
// 约 10 MB，会同时撑大内存、SSE 帧与持久化文件。
func TestOlMigrateAsyncFailKeysCappedBeforePersist(t *testing.T) {
	// 源端有 201 个对象；目标端一律 403 → 全部复制失败。
	syncStore = map[string]map[string]syncEntry{}
	srcMap := syncBucketOrCreate("acap-src")
	for i := 0; i < 201; i++ {
		srcMap[fmt.Sprintf("k%03d", i)] = syncEntry{size: 10, etag: "e"}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			m, _ := syncBucketStore(firstSegH(r.URL.Path))
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
			for k, e := range m {
				fmt.Fprintf(&sb, `<Contents><Key>%s</Key><Size>%d</Size><ETag>"%s"</ETag><LastModified>2024-01-01T00:00:00.000Z</LastModified></Contents>`, k, e.size, e.etag)
			}
			sb.WriteString(`<IsTruncated>false</IsTruncated></ListBucketResult>`)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(sb.String()))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "accounts.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	src, _ := st.Create(&model.Account{
		Name: "src", Endpoint: srv.URL, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "acap-src", PathStyle: true,
	})
	dst, _ := st.Create(&model.Account{
		Name: "dst", Endpoint: srv.URL, AccessKey: "ak", SecretKey: "sk",
		Region: "us-east-1", Bucket: "acap-dst", PathStyle: true,
	})
	persistPath := filepath.Join(dir, "jobs.json")
	h := New(st, quietLogger(), dir, nil, "", "test", false, false)
	h.SetJobPersister(service.NewFileJobPersister(persistPath))
	t.Cleanup(h.Shutdown)
	hs := httptest.NewServer(h.Routes())
	defer hs.Close()

	body, _ := json.Marshal(map[string]any{
		"sourceAccountId": src.ID, "sourceBucket": "acap-src", "sourceKeys": syncKeysOf(srcMap),
		"targetAccountId": dst.ID, "targetBucket": "acap-dst", "targetPrefix": "t/",
	})
	req, _ := http.NewRequest("POST", hs.URL+"/api/migrate/async", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	var start map[string]any
	_ = json.NewDecoder(res.Body).Decode(&start)
	res.Body.Close()
	jobID, _ := start["jobId"].(string)
	if jobID == "" {
		t.Fatalf("missing jobId: %v", start)
	}

	// 轮询状态直到终态。
	deadline := time.Now().Add(30 * time.Second)
	var result map[string]any
	for time.Now().Before(deadline) {
		sr, err := hs.Client().Get(hs.URL + "/api/migrate/jobs/" + jobID)
		if err != nil {
			t.Fatalf("GET job: %v", err)
		}
		var st map[string]any
		_ = json.NewDecoder(sr.Body).Decode(&st)
		sr.Body.Close()
		if done, _ := st["done"].(bool); done {
			result, _ = st["result"].(map[string]any)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if result == nil {
		t.Fatal("任务未在期限内完成")
	}
	if failed, _ := result["failed"].(float64); failed != 201 {
		t.Fatalf("failed = %v, want 201（计数不裁剪）", result["failed"])
	}
	keys, _ := result["failedKeys"].([]any)
	if len(keys) != maxFailKeys {
		t.Fatalf("failedKeys len = %d, want %d（落盘前必须裁剪）", len(keys), maxFailKeys)
	}

	// 直接检查落盘文件：不得出现第 201 个 key，证明裁剪发生在持久化之前。
	raw, err := os.ReadFile(persistPath)
	if err != nil {
		t.Fatalf("读取 jobs.json: %v", err)
	}
	if bytes.Count(raw, []byte(`"k`)) > maxFailKeys {
		t.Errorf("jobs.json 中失败 key 数超过上限（裁剪未发生在落盘前）")
	}
}

// syncBucketOrCreate 返回（必要时创建）某 bucket 的对象表。
func syncBucketOrCreate(bucket string) map[string]syncEntry {
	syncMu.Lock()
	defer syncMu.Unlock()
	if syncStore[bucket] == nil {
		syncStore[bucket] = map[string]syncEntry{}
	}
	return syncStore[bucket]
}

// syncKeysOf 返回 map 中的 key 列表（顺序不重要，服务端按集合处理）。
func syncKeysOf(m map[string]syncEntry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestOlCopyManyAsyncValidation 异步批量复制：缺 keys / 超上限 / 空 key 跳过。
func TestOlCopyManyAsyncValidation(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		return olPlain(http.StatusNoContent)
	})
	env := accNewEnv(t, srv.URL, "b")
	id := env.acc.ID

	// 缺 keys → 400
	rr := env.accDoRec("POST", "/api/accounts/"+id+"/copy-objects/async", `{"bucket":"b"}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "no keys")

	// 超过上限 → 400
	keys := make([]string, 10001)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%d", i)
	}
	b, _ := json.Marshal(map[string]any{"bucket": "b", "keys": keys})
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-objects/async", string(b))
	olExpectStatus(t, rr, http.StatusBadRequest, "too many keys")

	// 空 key 被跳过：仅复制 1 个
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-objects/async", `{"keys":["","a.txt"]}`)
	olExpectStatus(t, rr, http.StatusAccepted, "skip empty key")
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	if int(start["total"].(float64)) != 1 {
		t.Fatalf("total = %v, want 1", start["total"])
	}
	done := olWaitJobDone(t, env, start["jobId"].(string))
	res, _ := done["result"].(map[string]any)
	if int(res["migrated"].(float64)) != 1 {
		t.Fatalf("migrated = %v", res)
	}
}

// TestOlCopyPrefixAsyncCancel 异步前缀复制被取消：复制阻塞 → cancel → 状态 cancelled。
func TestOlCopyPrefixAsyncCancel(t *testing.T) {
	release := make(chan struct{})
	seen := make(chan struct{}, 1)
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/1", "p/2"}, false, ""))
		}
		if r.Header.Get("x-amz-copy-source") != "" {
			seen <- struct{}{}
			<-release
			return olPlain(http.StatusOK)
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-prefix/async",
		`{"bucket":"b","prefix":"p/","targetPrefix":"q/"}`)
	olExpectStatus(t, rr, http.StatusAccepted, "async copy cancel start")
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	select {
	case <-seen:
	case <-time.After(5 * time.Second):
		t.Fatal("copy never reached fake S3")
	}
	env.accDoRec("POST", "/api/migrate/jobs/"+start["jobId"].(string)+"/cancel", "")
	close(release)
	done := olWaitJobDone(t, env, start["jobId"].(string))
	prog, _ := done["progress"].(map[string]any)
	if prog["status"] != "cancelled" {
		t.Fatalf("progress status = %v, want cancelled", prog["status"])
	}
}
