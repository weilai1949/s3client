package handler

// ol_delete_partial_test.go —— docs/archive/review-2026-09-19.md §B3：200 响应体内的逐 key 删除失败
// 必须体现在「已删除」计数里（S3 对部分失败仍返回 200）。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// deleteWithErrorXML 构造 DeleteObjects 的 200 响应：failed 中的 key 逐条报错，其余视为已删除。
func deleteWithErrorXML(deleted []string, failed map[string]string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0"?><DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	for _, k := range deleted {
		fmt.Fprintf(&sb, "<Deleted><Key>%s</Key></Deleted>", k)
	}
	for k, code := range failed {
		fmt.Fprintf(&sb, "<Error><Key>%s</Key><Code>%s</Code><Message>%s</Message></Error>", k, code, code)
	}
	sb.WriteString("</DeleteResult>")
	return sb.String()
}

// TestOlDeleteObjectsReportsPartialFailure POST /delete：一个 key 被拒 → deleted 不包含它。
func TestOlDeleteObjectsReportsPartialFailure(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Method == http.MethodPost && r.URL.Query().Has("delete") {
			return olXML(http.StatusOK, deleteWithErrorXML(
				[]string{"a.txt", "c.txt"}, map[string]string{"b.txt": "AccessDenied"}))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/delete",
		`{"bucket":"b","keys":["a.txt","b.txt","c.txt"]}`)
	olExpectStatus(t, rr, http.StatusOK, "partial delete")
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if got["deleted"] != float64(2) || got["failed"] != float64(1) {
		t.Fatalf("response = %v, want deleted 2 failed 1", got)
	}
	if got["lastError"] != "access denied" {
		t.Fatalf("lastError = %v, want access denied", got["lastError"])
	}
}

// TestOlDeletePrefixReportsPartialFailure 同步 delete-prefix：逐 key 失败计入 failed 而不是 deleted。
func TestOlDeletePrefixReportsPartialFailure(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodGet && q.Has("list-type"):
			return olXML(http.StatusOK, listBucketXML([]string{"p/1", "p/2"}, false, ""))
		case r.Method == http.MethodPost && q.Has("delete"):
			return olXML(http.StatusOK, deleteWithErrorXML([]string{"p/1"}, map[string]string{"p/2": "AccessDenied"}))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/delete-prefix", `{"prefix":"p/"}`)
	olExpectStatus(t, rr, http.StatusOK, "partial delete-prefix")
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if got["deleted"] != float64(1) || got["failed"] != float64(1) || got["lastError"] != "access denied" {
		t.Fatalf("response = %v, want deleted 1 failed 1 lastError access denied", got)
	}
}

// TestOlDeletePrefixAsyncReportsPartialFailure 异步 delete-prefix：任务结果区分 migrated/failed 并记失败 key。
func TestOlDeletePrefixAsyncReportsPartialFailure(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodGet && q.Has("list-type"):
			return olXML(http.StatusOK, listBucketXML([]string{"p/1", "p/2"}, false, ""))
		case r.Method == http.MethodPost && q.Has("delete"):
			return olXML(http.StatusOK, deleteWithErrorXML([]string{"p/1"}, map[string]string{"p/2": "AccessDenied"}))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec("POST", "/api/accounts/"+env.acc.ID+"/delete-prefix/async", `{"prefix":"p/"}`)
	olExpectStatus(t, rr, http.StatusAccepted, "async start")
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	jobID, _ := start["jobId"].(string)
	if jobID == "" {
		t.Fatalf("missing jobId: %s", rr.Body.String())
	}
	done := olWaitJobDone(t, env, jobID)
	res, _ := done["result"].(map[string]any)
	if res["migrated"] != float64(1) || res["failed"] != float64(1) {
		t.Fatalf("result = %v, want migrated 1 failed 1", res)
	}
	keys, _ := res["failedKeys"].([]any)
	if len(keys) != 1 || keys[0] != "p/2" {
		t.Fatalf("failedKeys = %v, want [p/2]", res["failedKeys"])
	}
}
