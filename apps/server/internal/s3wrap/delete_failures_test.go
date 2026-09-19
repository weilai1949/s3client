package s3wrap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// deleteResultXML 构造 DeleteObjects 的 200 响应体：deleted 为成功 key，failed 为 (key, code) 逐 key 失败。
func deleteResultXML(deleted []string, failed map[string]string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><DeleteResult ` + xmlNS + `>`)
	for _, k := range deleted {
		sb.WriteString("<Deleted><Key>" + k + "</Key></Deleted>")
	}
	for k, code := range failed {
		sb.WriteString("<Error><Key>" + k + "</Key><Code>" + code + "</Code><Message>" + code + " msg</Message></Error>")
	}
	sb.WriteString("</DeleteResult>")
	return sb.String()
}

// TestDeleteObjectsReportsPerKeyFailures 200 响应体内的逐 key <Error> 必须上抛：
// S3 对「部分 key 删不掉」仍返回 200，只看顶层 err 会把失败当成功（review §B3）。
func TestDeleteObjectsReportsPerKeyFailures(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, deleteResultXML([]string{"b"}, map[string]string{"a": "AccessDenied", "c": "AccessDenied"}))
	}))
	failures, err := c.DeleteObjects(context.Background(), "bkt", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("DeleteObjects: %v", err)
	}
	if len(failures) != 2 {
		t.Fatalf("failures = %+v, want 2", failures)
	}
	got := map[string]string{}
	for _, f := range failures {
		got[f.Key] = f.Code
	}
	if got["a"] != "AccessDenied" || got["c"] != "AccessDenied" {
		t.Fatalf("failures = %+v", failures)
	}
	if UserMessageForCode(failures[0].Code) != "access denied" {
		t.Fatalf("UserMessageForCode(%q) = %q", failures[0].Code, UserMessageForCode(failures[0].Code))
	}
}

// TestDeleteObjectsPartialFailureAcrossBatches 前一批的逐 key 失败不丢：后续批次的传输错误
// 只应追加，不应清空已收集的失败（分批调用方要能报告「哪些已确定失败」）。
func TestDeleteObjectsPartialFailureAcrossBatches(t *testing.T) {
	calls := 0
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, deleteResultXML(nil, map[string]string{"k0": "AccessDenied"}))
			return
		}
		writeS3Error(w, http.StatusForbidden, "AccessDenied", "no")
	}))
	keys := make([]string, 0, 1001)
	for i := 0; i < 1001; i++ {
		keys = append(keys, "k"+strconv.Itoa(i))
	}
	failures, err := c.DeleteObjects(context.Background(), "bkt", keys)
	if err == nil {
		t.Fatal("expected transport error from second batch")
	}
	if HTTPStatus(err) != http.StatusForbidden {
		t.Fatalf("HTTPStatus = %d, want 403", HTTPStatus(err))
	}
	if len(failures) != 1 || failures[0].Key != "k0" {
		t.Fatalf("failures = %+v, want the first batch's k0", failures)
	}
}

// TestDeleteObjectsSuccessHasNoFailures 全成功时不得产生伪失败。
func TestDeleteObjectsSuccessHasNoFailures(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, deleteResultXML([]string{"a", "b"}, nil))
	}))
	failures, err := c.DeleteObjects(context.Background(), "bkt", []string{"a", "b"})
	if err != nil || len(failures) != 0 {
		t.Fatalf("failures = %+v, err = %v", failures, err)
	}
}

// TestPurgeObjectRefusesPartialDelete purge 中若有版本被拒（保留期/策略），
// 必须报错而不是把「没删掉」算进删除计数（review §B3 第 4 个受影响点）。
func TestPurgeObjectRefusesPartialDelete(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("versions") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><ListVersionsResult `+xmlNS+`>
<IsTruncated>false</IsTruncated>
<Version><Key>t.txt</Key><VersionId>v1</VersionId><IsLatest>true</IsLatest><LastModified>2024-01-01T00:00:00.000Z</LastModified><ETag>&quot;e&quot;</ETag><Size>1</Size></Version>
</ListVersionsResult>`)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, deleteResultXML(nil, map[string]string{"t.txt": "AccessDenied"}))
	}))
	deleted, err := c.PurgeObject(context.Background(), "bkt", "t.txt")
	if err == nil {
		t.Fatal("expected partial-delete error")
	}
	if !errors.Is(err, ErrPartialDelete) {
		t.Fatalf("err = %v, want ErrPartialDelete", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0 (the only version was refused)", deleted)
	}
	if got := UserMessage(err); got != "some objects could not be deleted" {
		t.Fatalf("UserMessage = %q", got)
	}
}
