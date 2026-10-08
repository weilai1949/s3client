package handler

// ol_copy_test.go —— copy.go 校验/失败聚合/异步取消补测（仅新增，不改生产代码）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// TestOlCopyValidation 复制类接口：404 / 非法 JSON / 缺桶。
func TestOlCopyValidation(t *testing.T) {
	paths := func(id string) []struct{ name, method, path, body string } {
		return []struct{ name, method, path, body string }{
			{"copy-object", "POST", "/api/accounts/" + id + "/copy-object", `{"key":"a","newKey":"b"}`},
			{"copy-objects", "POST", "/api/accounts/" + id + "/copy-objects", `{"keys":["a"]}`},
			{"copy-objects-async", "POST", "/api/accounts/" + id + "/copy-objects/async", `{"keys":["a"]}`},
			{"copy-prefix", "POST", "/api/accounts/" + id + "/copy-prefix", `{"prefix":"p/","targetPrefix":"q/"}`},
			{"copy-prefix-async", "POST", "/api/accounts/" + id + "/copy-prefix/async", `{"prefix":"p/","targetPrefix":"q/"}`},
		}
	}
	// 404：账号不存在
	env := accNewEnv(t, "http://127.0.0.1:1", "b")
	for _, c := range paths("nope") {
		rr := env.accDoRec(c.method, c.path, c.body)
		olExpectStatus(t, rr, http.StatusNotFound, c.name)
	}
	// 非法 JSON
	srv := olFake(t, func(r *http.Request) olResp { return olPlain(http.StatusOK) })
	env = accNewEnv(t, srv.URL, "b")
	for _, c := range paths(env.acc.ID) {
		rr := env.accDoRec(c.method, c.path, "{bad")
		olExpectStatus(t, rr, http.StatusBadRequest, c.name)
	}
	// 缺桶
	nb := accNewEnv(t, srv.URL, "")
	for _, c := range paths(nb.acc.ID) {
		rr := nb.accDoRec(c.method, c.path, c.body)
		olExpectStatus(t, rr, http.StatusBadRequest, c.name)
	}
}

// TestOlCopyObjectErr 复制失败 → 403。
func TestOlCopyObjectErr(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olErr(http.StatusForbidden, "AccessDenied")
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-object", `{"key":"a","newKey":"b"}`)
	olExpectStatus(t, rr, http.StatusForbidden, "copy err")
}

// TestOlCopyManyBounds copyMany 超上限 → 400；空 key 被跳过。
func TestOlCopyManyBounds(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	id := env.acc.ID

	// 超过 10000 keys → 400
	keys := make([]string, 10001)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%d", i)
	}
	b, _ := json.Marshal(map[string]any{"bucket": "b", "keys": keys})
	rr := env.accDoRec("POST", "/api/accounts/"+id+"/copy-objects", string(b))
	olExpectStatus(t, rr, http.StatusBadRequest, "too many keys")

	// 空 key 跳过：["", "a.txt"] → 只复制 1 个
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-objects", `{"keys":["","a.txt"]}`)
	olExpectStatus(t, rr, http.StatusOK, "skip empty key")
	if !strings.Contains(rr.Body.String(), `"copied":1`) {
		t.Fatalf("skip empty body: %s", rr.Body.String())
	}
}

// TestOlCopyManyMoveFailureAggregation 移动（copy+delete）失败聚合：lastError + failedKeys。
func TestOlCopyManyMoveFailureAggregation(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		src := r.Header.Get("x-amz-copy-source") // 形如 b%2Fkey
		if src != "" {
			if strings.Contains(src, "del-a") {
				return olErr(http.StatusForbidden, "AccessDenied") // 复制失败
			}
			return olPlain(http.StatusOK) // 其余复制成功
		}
		if strings.Contains(r.URL.Path, "del-b") {
			return olErr(http.StatusForbidden, "AccessDenied") // 复制后删源失败
		}
		return olPlain(http.StatusNoContent) // 删除成功
	})
	env := accNewEnv(t, srv.URL, "b")
	body := `{"bucket":"b","keys":["del-a","del-b","del-c"],"deleteSource":true,"targetPrefix":"mv/"}`
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-objects", body)
	olExpectStatus(t, rr, http.StatusOK, "move aggregate")
	var m map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &m)
	if int(m["failed"].(float64)) != 2 || int(m["copied"].(float64)) != 1 {
		t.Fatalf("result = %v", m)
	}
	if m["lastError"] == "" {
		t.Fatalf("lastError missing: %v", m)
	}
	fk, _ := m["failedKeys"].([]any)
	if len(fk) != 2 {
		t.Fatalf("failedKeys = %v", fk)
	}
}

// TestOlCopyManyAsyncMove 异步移动：成功完成与中途取消。
func TestOlCopyManyAsyncMove(t *testing.T) {
	// 成功：复制+删除全部成功
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		return olPlain(http.StatusNoContent)
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-objects/async",
		`{"keys":["a.txt"],"deleteSource":true,"targetPrefix":"mv/"}`)
	olExpectStatus(t, rr, http.StatusAccepted, "async move start")
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	done := olWaitJobDone(t, env, start["jobId"].(string))
	res, _ := done["result"].(map[string]any)
	if int(res["migrated"].(float64)) != 1 {
		t.Fatalf("migrated = %v", res["migrated"])
	}

	// 取消：删除被阻塞 → cancel → 状态 cancelled
	release := make(chan struct{})
	seen := make(chan struct{}, 1)
	srv2 := olFake(t, func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		seen <- struct{}{}
		<-release
		return olPlain(http.StatusNoContent)
	})
	env2 := accNewEnv(t, srv2.URL, "b")
	rr = env2.accDoRec("POST", "/api/accounts/"+env2.acc.ID+"/copy-objects/async",
		`{"keys":["a.txt"],"deleteSource":true}`)
	olExpectStatus(t, rr, http.StatusAccepted, "async move cancel start")
	start = nil
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	select {
	case <-seen:
	case <-time.After(5 * time.Second):
		t.Fatal("delete never reached fake S3")
	}
	env2.accDoRec("POST", "/api/migrate/jobs/"+start["jobId"].(string)+"/cancel", "")
	close(release)
	done = olWaitJobDone(t, env2, start["jobId"].(string))
	prog, _ := done["progress"].(map[string]any)
	if prog["status"] != "cancelled" {
		t.Fatalf("progress status = %v, want cancelled", prog["status"])
	}
}

// TestOlCopyPrefixValidation copy-prefix 前后端校验：空前缀 / 同桶前缀重叠 / 列举失败。
func TestOlCopyPrefixValidation(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olErr(http.StatusForbidden, "AccessDenied")
		}
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	id := env.acc.ID

	// 空前缀 → 400
	rr := env.accDoRec("POST", "/api/accounts/"+id+"/copy-prefix", `{"bucket":"b"}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "no prefix")
	// 同桶前缀重叠 → 400
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-prefix", `{"bucket":"b","prefix":"a/","targetPrefix":"a/sub"}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "overlap")
	// 列举失败 → 403
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-prefix", `{"bucket":"b","prefix":"p/","targetPrefix":"q/"}`)
	olExpectStatus(t, rr, http.StatusForbidden, "list err")

	// 异步：空前缀 / 重叠 / 列举失败
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-prefix/async", `{"bucket":"b"}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "async no prefix")
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-prefix/async", `{"bucket":"b","prefix":"a/","targetPrefix":"a/sub"}`)
	olExpectStatus(t, rr, http.StatusBadRequest, "async overlap")
	rr = env.accDoRec("POST", "/api/accounts/"+id+"/copy-prefix/async", `{"bucket":"b","prefix":"p/","targetPrefix":"q/"}`)
	olExpectStatus(t, rr, http.StatusForbidden, "async list err")
}

// TestOlCopyPrefixAsyncSuccess 异步前缀复制成功。
func TestOlCopyPrefixAsyncSuccess(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/1", "p/2"}, false, ""))
		}
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/copy-prefix/async",
		`{"bucket":"b","prefix":"p/","targetPrefix":"q/"}`)
	olExpectStatus(t, rr, http.StatusAccepted, "async copy start")
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	if start["truncated"] != false {
		t.Fatalf("truncated = %v", start["truncated"])
	}
	done := olWaitJobDone(t, env, start["jobId"].(string))
	res, _ := done["result"].(map[string]any)
	if int(res["migrated"].(float64)) != 2 {
		t.Fatalf("migrated = %v", res)
	}
}

// TestOlListPrefixKeysLimit 白盒验证 listPrefixKeys 达到 maxCopy 即截断返回。
func TestOlListPrefixKeysLimit(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/1", "p/2", "p/3"}, true, "t"))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	client := olClient(t, env)
	keys, truncated, err := env.hnd.listPrefixKeys(t.Context(), client, "b", "p/", 2)
	if err != nil {
		t.Fatalf("listPrefixKeys: %v", err)
	}
	if !truncated || len(keys) != 2 {
		t.Fatalf("keys=%v truncated=%v, want 2 keys truncated", keys, truncated)
	}
}

// listKeysResult 是 listPrefixKeys 三值返回的可传载体。
type listKeysResult struct {
	keys      []string
	truncated bool
	err       error
}

// runListPrefixKeys 在 goroutine 里跑 listPrefixKeys：5 秒不返回即判列举循环空转（红灯），
// 被测函数拿到的 ctx 由 t.Context() 提供，测试结束会自动取消，残留 goroutine 随之退出。
func runListPrefixKeys(t *testing.T, ctx context.Context, env *accEnv, client *s3wrap.Client, limit int) listKeysResult {
	t.Helper()
	done := make(chan listKeysResult, 1)
	go func() {
		keys, truncated, err := env.hnd.listPrefixKeys(ctx, client, "b", "p/", limit)
		done <- listKeysResult{keys: keys, truncated: truncated, err: err}
	}()
	select {
	case r := <-done:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("listPrefixKeys 未终止：列举循环空转")
		return listKeysResult{}
	}
}

// TestOlListPrefixKeysStopsOnNonAdvancingToken 对端反复返回同一个 NextContinuationToken
// 且每页都声明还有下一页时必须终止并标记截断（与 service.deletePrefix 同款守卫，review §B6）。
// limit 取大值是为了让「靠 maxCopy 计数停住」这条路走不通——空转只能由守卫拦下。
func TestOlListPrefixKeysStopsOnNonAdvancingToken(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/x"}, true, "t"))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	got := runListPrefixKeys(t, t.Context(), env, olClient(t, env), 1_000_000)
	if got.err != nil {
		t.Fatalf("listPrefixKeys: %v", got.err)
	}
	// 首页 token=""→"t" 前进；次页请求带 "t" 而回的仍是 "t" ⇒ 判定不前进即停，共 2 页。
	if len(got.keys) != 2 {
		t.Fatalf("keys = %d, want 2（token 不前进后立即停止）", len(got.keys))
	}
	if !got.truncated {
		t.Fatal("token 不前进而停 = 列举不完整，必须标记 truncated")
	}
}

// TestOlListPrefixKeysStopsAtPageCap 空页 + 每次都前进的 token：maxCopy 的计数永不增长，
// 只有页数上限拦得住（否则同步 copy-prefix / delete-prefix 会一直占着 withStreamLimit 的槽位）。
func TestOlListPrefixKeysStopsAtPageCap(t *testing.T) {
	var calls atomic.Int64
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			page := calls.Add(1)
			return olXML(http.StatusOK, listBucketXML(nil, true, fmt.Sprintf("t%d", page)))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	got := runListPrefixKeys(t, t.Context(), env, olClient(t, env), 1_000_000)
	if got.err != nil {
		t.Fatalf("listPrefixKeys: %v", got.err)
	}
	if len(got.keys) != 0 {
		t.Fatalf("keys = %d, want 0（空页）", len(got.keys))
	}
	if !got.truncated {
		t.Fatal("页数上限耗尽而对端仍称有下一页，必须标记 truncated")
	}
}

// TestOlListPrefixKeysExactlyLimitIsNotTruncated 正好收满 limit 且对端声明列举完成
// ⇒ 全部枚举成功，不是截断。docs/api.md 把 truncated 定义为「第 limit+1 个起未参与本次
// 操作」，正好收满时根本没有第 limit+1 个，误报会让客户端去重试一个已经完成的操作。
func TestOlListPrefixKeysExactlyLimitIsNotTruncated(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/1", "p/2", "p/3"}, false, ""))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	got := runListPrefixKeys(t, t.Context(), env, olClient(t, env), 3)
	if got.err != nil {
		t.Fatalf("listPrefixKeys: %v", got.err)
	}
	if len(got.keys) != 3 {
		t.Fatalf("keys = %d, want 3", len(got.keys))
	}
	if got.truncated {
		t.Fatal("正好收满 limit 且对端声明列举完成 ⇒ 不是截断")
	}
}

// TestOlListPrefixKeysReturnsOnCancelledContext 循环首行就查 ctx：不依赖 SDK 调用
// 间接响应取消（守卫是覆盖率的一部分，不许用「反正 SDK 也会报」跳过）。
func TestOlListPrefixKeysReturnsOnCancelledContext(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, listBucketXML([]string{"p/1"}, true, "t"))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := runListPrefixKeys(t, ctx, env, olClient(t, env), 100)
	if !errors.Is(got.err, context.Canceled) {
		t.Fatalf("listPrefixKeys(cancelled) = %v, want context.Canceled", got.err)
	}
}
