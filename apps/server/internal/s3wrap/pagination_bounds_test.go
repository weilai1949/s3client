package s3wrap

// pagination_bounds_test.go —— s3wrap 的**列举循环必须有界**（口径同 service/sync_list_gate_test.go）。
//
// 列举纪律要求「对端始终 truncated / 游标原地打转即停」，此前这条纪律只在 internal/service
// 层有门禁；s3wrap 的 ListParts 与 PurgeObject 同样会逐页请求对端，故障 / 恶意端点必须同样有界，
// 否则断点续传清单查询与回收站「彻底清除」会永久挂住（阻塞请求直至超时）。
//
// 两个用例都用「始终 truncated + 游标不前进」的假 S3 驱动守卫分支，并以 5s 超时兜底：
// 若守卫被删，循环将永不返回，用例超时红灯（而不是静默挂住整个测试进程）。

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestListPartsNonAdvancingMarkerStops：对端始终 truncated 且 NextPartNumberMarker 原地打转时，
// ListParts 必须终止（守卫命中）而不是无限翻页。
func TestListPartsNonAdvancingMarkerStops(t *testing.T) {
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		// 始终 truncated，NextPartNumberMarker 恒为 "1"（不前进）。
		_, _ = w.Write([]byte(listPartsXML(true, "1", partXML(1, "e1", 1))))
	}))
	done := make(chan int, 1)
	go func() {
		parts, err := c.ListParts(context.Background(), "bkt", "big.bin", "up1")
		if err != nil {
			done <- -1
			return
		}
		done <- len(parts)
	}()
	select {
	case n := <-done:
		if n <= 0 {
			t.Fatalf("ListParts returned %d（<0 表示报错），want > 0", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ListParts 未在游标不前进时终止（守卫缺失 → 死循环）")
	}
}

// TestPurgeObjectNonAdvancingMarkerStops：对端始终 truncated 且版本游标原地打转时，
// PurgeObject 必须终止。首页返回一个版本（触发一次批量删除），其后页码不再返回版本。
func TestPurgeObjectNonAdvancingMarkerStops(t *testing.T) {
	pages := 0
	c, _ := newFakeS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Has("versions"):
			pages++
			w.Header().Set("Content-Type", "application/xml")
			version := ""
			if pages == 1 {
				version = `<Version><Key>k</Key><VersionId>v1</VersionId><IsLatest>true</IsLatest></Version>`
			}
			// 始终 truncated，游标恒为同一组值（不前进）。
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListVersionsResult ` + xmlNS + `>
<IsTruncated>true</IsTruncated><NextKeyMarker>k</NextKeyMarker><NextVersionIdMarker>v-next</NextVersionIdMarker>
` + version + `
</ListVersionsResult>`))
		case q.Has("delete"):
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><DeleteResult ` + xmlNS + `></DeleteResult>`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	done := make(chan int, 1)
	go func() {
		n, err := c.PurgeObject(context.Background(), "bkt", "k")
		if err != nil {
			done <- -1
			return
		}
		done <- n
	}()
	select {
	case n := <-done:
		if n != 1 {
			t.Fatalf("PurgeObject returned %d, want 1（只删首页的 v1，游标不前进即停）", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("PurgeObject 未在游标不前进时终止（守卫缺失 → 死循环）")
	}
}
