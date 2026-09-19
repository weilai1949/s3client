package handler

// migrate_sync_gate_test.go —— review-2026-09-19.md §B5：/api/migrate/sync 在源端列举失败时
// 必须回错误状态，而不是 `200 {scanned:0}`（后者会被读成「没有需要同步的内容」）。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMigrateSync_SourceListFailureIsReported 源端 ListObjectsV2 返回 AccessDenied → 403，
// 且响应体里不能出现 scanned=0 的「成功」形状。
func TestMigrateSync_SourceListFailureIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<?xml version="1.0"?><Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	srcID, dstID, base := newSyncEnv(t, srv.URL, srv.URL)
	b, _ := json.Marshal(map[string]any{
		"sourceAccountId": srcID, "sourceBucket": "src-bucket",
		"targetAccountId": dstID, "targetBucket": "dst-bucket", "mode": "etag",
	})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/migrate/sync", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403（源端列举失败不能报成功）", rr.StatusCode)
	}
	var m map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&m)
	if m["error"] != "access denied" {
		t.Fatalf("body = %v, want error=access denied", m)
	}
	if _, ok := m["scanned"]; ok {
		t.Fatalf("body = %v 不应包含 scanned（那是成功响应的形状）", m)
	}
}
