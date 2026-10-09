package handler

// storage_report_extra_test.go —— storage_report_test.go 之外的边界补测：
// 单页超过 100k 上限时必须在页内裁断并标记 truncated（collectStorageReportObjects
// 的 dropped 分支；正常 1000/页 的分片永远走不到该分支，只有超额单页才会触发）。

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestStorageReportTruncatedWhenPageExceedsLimit(t *testing.T) {
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Method == http.MethodGet && r.URL.Query().Has("list-type") {
			page := 0
			if token := r.URL.Query().Get("continuation-token"); token != "" {
				page, _ = strconv.Atoi(token)
			}
			if page < 99 {
				objs := make([]srObj, 1000)
				for i := range objs {
					objs[i] = srObj{key: fmt.Sprintf("p%03d-%04d", page, i), size: 1, class: "STANDARD", age: time.Hour}
				}
				return olXML(http.StatusOK, srListXML(objs, true, strconv.Itoa(page+1)))
			}
			// 最后一页 1500 个：前 1000 个刚好补满 100k，其后触发「页内超额」裁断。
			objs := make([]srObj, 1500)
			for i := range objs {
				objs[i] = srObj{key: fmt.Sprintf("last-%04d", i), size: 1, class: "STANDARD", age: time.Hour}
			}
			return olXML(http.StatusOK, srListXML(objs, false, ""))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")

	rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
	olExpectStatus(t, rr, http.StatusOK, "over-large page report")
	report := srDecode(t, rr.Body.String())
	if got := srI64(t, report, "objectCount"); got != 100_000 {
		t.Errorf("objectCount = %d, want 100000（页内超额须裁到上限）", got)
	}
	if got, _ := report["truncated"].(bool); !got {
		t.Errorf("truncated = false, want true（页内超额即截断）")
	}
}
