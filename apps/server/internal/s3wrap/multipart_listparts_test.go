package s3wrap

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// listPartsXML 构造 ListParts 响应；parts 为 `<Part>` 片段拼接。
func listPartsXML(truncated bool, nextMarker string, parts string) string {
	extra := ""
	if truncated {
		extra = "<IsTruncated>true</IsTruncated>"
	} else {
		extra = "<IsTruncated>false</IsTruncated>"
	}
	if nextMarker != "" {
		extra += "<NextPartNumberMarker>" + nextMarker + "</NextPartNumberMarker>"
	}
	return `<?xml version="1.0" encoding="UTF-8"?><ListPartsResult ` + xmlNS + `>` +
		`<Bucket>bkt</Bucket><Key>big.bin</Key><UploadId>up1</UploadId>` + extra + parts + `</ListPartsResult>`
}

func partXML(n int, etag string, size int64) string {
	return `<Part><PartNumber>` + strconv.Itoa(n) + `</PartNumber><LastModified>2026-10-08T05:00:00.000Z</LastModified>` +
		`<ETag>` + etag + `</ETag><Size>` + strconv.FormatInt(size, 10) + `</Size></Part>`
}

// TestListPartsReturnsParts ListParts：返回段号 / ETag（去引号）/ 大小 / 修改时间，并透传 uploadId。
func TestListPartsReturnsParts(t *testing.T) {
	var gotUploadID, gotMethod, gotMaxParts string
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUploadID = r.URL.Query().Get("uploadId")
		gotMethod = r.Method
		gotMaxParts = r.URL.Query().Get("max-parts")
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(listPartsXML(false, "", partXML(1, `"e1"`, 10*1024*1024)+partXML(2, "e2", 5))))
	}))
	parts, err := c.ListParts(context.Background(), "bkt", "big.bin", "up1")
	if err != nil {
		t.Fatalf("ListParts: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("len(parts) = %d, want 2", len(parts))
	}
	if parts[0].PartNumber != 1 || parts[0].ETag != "e1" || parts[0].Size != 10*1024*1024 {
		t.Fatalf("part[0] = %+v", parts[0])
	}
	if parts[1].PartNumber != 2 || parts[1].ETag != "e2" || parts[1].Size != 5 {
		t.Fatalf("part[1] = %+v", parts[1])
	}
	wantTime := time.Date(2026, 10, 8, 5, 0, 0, 0, time.UTC)
	if !parts[0].LastModified.Equal(wantTime) {
		t.Fatalf("LastModified = %v, want %v", parts[0].LastModified, wantTime)
	}
	if gotUploadID != "up1" || gotMethod != http.MethodGet {
		t.Fatalf("uploadId=%q method=%q", gotUploadID, gotMethod)
	}
	if gotMaxParts != "1000" {
		t.Fatalf("max-parts = %q, want 1000", gotMaxParts)
	}
}

// TestListPartsPaginates 截断时按 NextPartNumberMarker 自动翻页，清单完整。
func TestListPartsPaginates(t *testing.T) {
	var markers []string
	calls := 0
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		markers = append(markers, r.URL.Query().Get("part-number-marker"))
		w.Header().Set("Content-Type", "application/xml")
		if calls == 1 {
			_, _ = w.Write([]byte(listPartsXML(true, "2", partXML(1, "e1", 1)+partXML(2, "e2", 1))))
			return
		}
		_, _ = w.Write([]byte(listPartsXML(false, "", partXML(3, "e3", 1))))
	}))
	parts, err := c.ListParts(context.Background(), "bkt", "big.bin", "up1")
	if err != nil {
		t.Fatalf("ListParts: %v", err)
	}
	if len(parts) != 3 || parts[2].PartNumber != 3 {
		t.Fatalf("parts = %+v", parts)
	}
	if len(markers) != 2 || markers[0] != "" || markers[1] != "2" {
		t.Fatalf("markers = %v, want [\"\" \"2\"]", markers)
	}
}

// TestListPartsTruncatedWithoutMarkerStops 异常响应（截断但无下一页标记）不得死循环。
func TestListPartsTruncatedWithoutMarkerStops(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(listPartsXML(true, "", partXML(1, "e1", 1))))
	}))
	parts, err := c.ListParts(context.Background(), "bkt", "big.bin", "up1")
	if err != nil {
		t.Fatalf("ListParts: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("len(parts) = %d, want 1", len(parts))
	}
}

// TestListPartsEmpty 无已上传分段时返回空清单（续传方据空清单重新 init）。
func TestListPartsEmpty(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(listPartsXML(false, "", "")))
	}))
	parts, err := c.ListParts(context.Background(), "bkt", "big.bin", "up1")
	if err != nil {
		t.Fatalf("ListParts: %v", err)
	}
	if len(parts) != 0 {
		t.Fatalf("len(parts) = %d, want 0", len(parts))
	}
}

// TestListPartsError uploadId 失效（NoSuchUpload）时错误透传，供调用方回退重新 init。
func TestListPartsError(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeS3Error(w, http.StatusNotFound, "NoSuchUpload", "no such upload")
	}))
	if _, err := c.ListParts(context.Background(), "bkt", "big.bin", "gone"); err == nil {
		t.Fatal("expected NoSuchUpload error")
	}
}
