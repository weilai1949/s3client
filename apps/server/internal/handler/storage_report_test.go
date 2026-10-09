package handler

// storage_report_test.go —— ROADMAP §三 #7「FinOps：存储分析与成本洞察」的红灯用例。
//
// 被测端点：GET /api/accounts/{id}/storage-report?bucket=&prefix=
// 断言外部可见行为：HTTP 状态码 + 响应 JSON 的字段与取值（不触碰私有实现）。
// 覆盖：按存储类 / 顶层前缀聚合、月成本估算、低频与归档建议、
// 100k 对象硬上限截断、token 不前进截断、页数上限截断、空桶、错误分支。

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---- 假 S3：可编排的 ListObjectsV2 ----

// srObj 一条用于合成 ListBucketResult 的对象。
type srObj struct {
	key   string
	size  int64
	class string
	age   time.Duration // 距今多久未修改（LastModified = now-age）
}

// srGib GiB 字节数（成本按 GiB 计费，与 S3 口径一致）。
const srGib = int64(1) << 30

// srListXML 合成 ListObjectsV2 响应（含 StorageClass 与 LastModified）。
func srListXML(objs []srObj, truncated bool, nextToken string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	sb.WriteString(`<Name>b</Name><MaxKeys>1000</MaxKeys>`)
	if truncated {
		sb.WriteString("<IsTruncated>true</IsTruncated>")
	} else {
		sb.WriteString("<IsTruncated>false</IsTruncated>")
	}
	if nextToken != "" {
		sb.WriteString("<NextContinuationToken>" + nextToken + "</NextContinuationToken>")
	}
	now := time.Now()
	for _, o := range objs {
		sc := o.class
		if sc == "" {
			sc = "STANDARD"
		}
		lm := now.Add(-o.age).UTC().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(&sb, "<Contents><Key>%s</Key><Size>%d</Size><ETag>&quot;e&quot;</ETag>"+
			"<StorageClass>%s</StorageClass><LastModified>%s</LastModified></Contents>",
			o.key, o.size, sc, lm)
	}
	sb.WriteString("</ListBucketResult>")
	return sb.String()
}

// srFake 起一个只服务 ListObjectsV2 的假 S3；route 按 continuation-token 决定每页响应。
func srFake(t *testing.T, route func(token string) string) *accEnv {
	t.Helper()
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Method == http.MethodGet && r.URL.Query().Has("list-type") {
			return olXML(http.StatusOK, route(r.URL.Query().Get("continuation-token")))
		}
		return olResp{}
	})
	return accNewEnv(t, srv.URL, "b")
}

// ---- 响应解析 ----

// srDecode 把 recorder 的 body 解码为 map。
func srDecode(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("decode report: %v body=%s", err, body)
	}
	return m
}

// srI64 取整数字段（JSON 数字 → float64）。
func srI64(t *testing.T, m map[string]any, key string) int64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("field %q 不是数字: %#v", key, m[key])
	}
	return int64(v)
}

// srF64 取浮点字段。
func srF64(t *testing.T, m map[string]any, key string) float64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("field %q 不是数字: %#v", key, m[key])
	}
	return v
}

// srArr 取数组字段；nil（JSON null）不是数组，直接红灯。
func srArr(t *testing.T, m map[string]any, key string) []any {
	t.Helper()
	v, ok := m[key].([]any)
	if !ok {
		t.Fatalf("field %q 不是数组: %#v", key, m[key])
	}
	return v
}

// srNear 断言两个浮点在容差内相等。
func srNear(t *testing.T, what string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// srClass 按 storageClass 名在 byStorageClass 里查找条目。
func srClass(t *testing.T, report map[string]any, name string) map[string]any {
	t.Helper()
	for _, e := range srArr(t, report, "byStorageClass") {
		m, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("byStorageClass 元素不是对象: %#v", e)
		}
		if m["storageClass"] == name {
			return m
		}
	}
	t.Fatalf("byStorageClass 缺少 %q: %#v", name, report["byStorageClass"])
	return nil
}

// ---- 主用例：聚合 + 估算 + 建议 ----

// TestStorageReportAggregatesAndRecommends 一次报告同时验证：
// 按存储类聚合（体积降序）、按顶层前缀聚合（体积降序）、月成本估算、
// 低频（30–89 天）与归档（≥90 天）建议，以及默认桶回退。
func TestStorageReportAggregatesAndRecommends(t *testing.T) {
	objs := []srObj{
		{key: "photos/a.jpg", size: 10 * srGib, class: "STANDARD", age: 100 * 24 * time.Hour},
		{key: "photos/b.jpg", size: 5 * srGib, class: "STANDARD", age: 40 * 24 * time.Hour},
		{key: "photos/c.jpg", size: 1 * srGib, class: "STANDARD", age: 24 * time.Hour},
		{key: "logs/x.log", size: 8 * srGib, class: "STANDARD_IA", age: 100 * 24 * time.Hour},
		{key: "dump.bin", size: 4 * srGib, class: "GLACIER_IR", age: 200 * 24 * time.Hour},
		{key: "weird.dat", size: 2 * srGib, class: "VENDOR_X", age: 24 * time.Hour},
	}
	env := srFake(t, func(string) string { return srListXML(objs, false, "") })

	// 不带 bucket 参数 → 回退账号默认桶。
	rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
	olExpectStatus(t, rr, http.StatusOK, "storage report")
	report := srDecode(t, rr.Body.String())

	if got := report["bucket"]; got != "b" {
		t.Errorf("bucket = %v, want b", got)
	}
	if got := report["prefix"]; got != "" {
		t.Errorf("prefix = %v, want 空", got)
	}
	if got := srI64(t, report, "objectCount"); got != 6 {
		t.Errorf("objectCount = %d, want 6", got)
	}
	if got := srI64(t, report, "totalSize"); got != 30*srGib {
		t.Errorf("totalSize = %d, want %d", got, 30*srGib)
	}
	if got, _ := report["truncated"].(bool); got {
		t.Errorf("truncated = true, want false")
	}
	if got := srI64(t, report, "prefixGroupCount"); got != 3 {
		t.Errorf("prefixGroupCount = %d, want 3", got)
	}

	// 月成本：16GiB*0.023 + 8GiB*0.0125 + 4GiB*0.004 + 2GiB*0.023（未知类按 STANDARD 计）。
	srNear(t, "monthlyCost", srF64(t, report, "monthlyCost"), 0.53)

	// byStorageClass：体积降序 STANDARD > STANDARD_IA > GLACIER_IR > VENDOR_X。
	classes := srArr(t, report, "byStorageClass")
	if len(classes) != 4 {
		t.Fatalf("byStorageClass len = %d, want 4", len(classes))
	}
	wantOrder := []string{"STANDARD", "STANDARD_IA", "GLACIER_IR", "VENDOR_X"}
	for i, name := range wantOrder {
		got := classes[i].(map[string]any)["storageClass"]
		if got != name {
			t.Errorf("byStorageClass[%d] = %v, want %s", i, got, name)
		}
	}
	st := srClass(t, report, "STANDARD")
	if srI64(t, st, "count") != 3 || srI64(t, st, "size") != 16*srGib {
		t.Errorf("STANDARD = count %v size %v, want 3 / %d", st["count"], st["size"], 16*srGib)
	}
	srNear(t, "STANDARD.monthlyCost", srF64(t, st, "monthlyCost"), 16*0.023)
	srNear(t, "VENDOR_X.monthlyCost", srF64(t, srClass(t, report, "VENDOR_X"), "monthlyCost"), 2*0.023)

	// byPrefix：首层分组、体积降序 photos/(16G) > logs/(8G) > 基准层(6G)。
	prefixes := srArr(t, report, "byPrefix")
	if len(prefixes) != 3 {
		t.Fatalf("byPrefix len = %d, want 3: %#v", len(prefixes), prefixes)
	}
	wantPrefixes := []struct {
		prefix string
		count  int64
		size   int64
	}{
		{"photos/", 3, 16 * srGib},
		{"logs/", 1, 8 * srGib},
		{"", 2, 6 * srGib},
	}
	for i, w := range wantPrefixes {
		got := prefixes[i].(map[string]any)
		if got["prefix"] != w.prefix || srI64(t, got, "count") != w.count || srI64(t, got, "size") != w.size {
			t.Errorf("byPrefix[%d] = %#v, want prefix=%q count=%d size=%d", i, got, w.prefix, w.count, w.size)
		}
	}

	// 建议：低频 1 条（STANDARD 40 天）+ 归档 2 条（STANDARD / STANDARD_IA ≥90 天）。
	recs := srArr(t, report, "recommendations")
	if len(recs) != 3 {
		t.Fatalf("recommendations len = %d, want 3: %#v", len(recs), recs)
	}
	inf := recs[0].(map[string]any)
	if inf["kind"] != "infrequent" || inf["fromStorageClass"] != "STANDARD" || inf["toStorageClass"] != "STANDARD_IA" {
		t.Errorf("recommendations[0] = %#v, want infrequent STANDARD→STANDARD_IA", inf)
	}
	if srI64(t, inf, "count") != 1 || srI64(t, inf, "size") != 5*srGib {
		t.Errorf("infrequent = count %v size %v, want 1 / %d", inf["count"], inf["size"], 5*srGib)
	}
	srNear(t, "infrequent.saving", srF64(t, inf, "estimatedMonthlySaving"), 5*(0.023-0.0125))
	arch0 := recs[1].(map[string]any)
	arch1 := recs[2].(map[string]any)
	if arch0["kind"] != "archive" || arch0["fromStorageClass"] != "STANDARD" || arch1["fromStorageClass"] != "STANDARD_IA" {
		t.Errorf("archive recs = %#v / %#v, want archive STANDARD→… 与 STANDARD_IA→…", arch0, arch1)
	}
	if srI64(t, arch0, "size") != 10*srGib || srI64(t, arch1, "size") != 8*srGib {
		t.Errorf("archive sizes = %v / %v, want %d / %d", arch0["size"], arch1["size"], 10*srGib, 8*srGib)
	}
	srNear(t, "archive[0].saving", srF64(t, arch0, "estimatedMonthlySaving"), 10*(0.023-0.004))
	srNear(t, "archive[1].saving", srF64(t, arch1, "estimatedMonthlySaving"), 8*(0.0125-0.004))
}

// TestStorageReportPrefixParamForwardedAndGrouped 指定 prefix 时：
// 参数透传给 S3，且分组标签落在该前缀之下的首层。
func TestStorageReportPrefixParamForwardedAndGrouped(t *testing.T) {
	var gotPrefix string
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Method == http.MethodGet && r.URL.Query().Has("list-type") {
			gotPrefix = r.URL.Query().Get("prefix")
			return olXML(http.StatusOK, srListXML([]srObj{
				{key: "logs/a.txt", size: 1 * srGib, age: time.Hour},
				{key: "logs/2024/x.log", size: 4 * srGib, age: time.Hour},
				{key: "logs/2024/y.log", size: 2 * srGib, age: time.Hour},
			}, false, ""))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")

	rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report?prefix=logs%2F", "")
	olExpectStatus(t, rr, http.StatusOK, "storage report with prefix")
	if gotPrefix != "logs/" {
		t.Errorf("S3 收到的 prefix = %q, want logs/", gotPrefix)
	}
	report := srDecode(t, rr.Body.String())
	if report["prefix"] != "logs/" {
		t.Errorf("prefix = %v, want logs/", report["prefix"])
	}
	prefixes := srArr(t, report, "byPrefix")
	if len(prefixes) != 2 {
		t.Fatalf("byPrefix len = %d, want 2: %#v", len(prefixes), prefixes)
	}
	if got := prefixes[0].(map[string]any); got["prefix"] != "logs/2024/" || srI64(t, got, "size") != 6*srGib {
		t.Errorf("byPrefix[0] = %#v, want logs/2024/ 6GiB", got)
	}
	if got := prefixes[1].(map[string]any); got["prefix"] != "logs/" || srI64(t, got, "count") != 1 {
		t.Errorf("byPrefix[1] = %#v, want logs/ count=1", got)
	}
}

// TestStorageReportEmptyBucket 空桶：全零报告，数组必须是 [] 而不是 null。
func TestStorageReportEmptyBucket(t *testing.T) {
	env := srFake(t, func(string) string { return srListXML(nil, false, "") })
	rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
	olExpectStatus(t, rr, http.StatusOK, "empty report")
	report := srDecode(t, rr.Body.String())
	if srI64(t, report, "objectCount") != 0 || srI64(t, report, "totalSize") != 0 {
		t.Errorf("objectCount/totalSize = %v/%v, want 0/0", report["objectCount"], report["totalSize"])
	}
	srNear(t, "monthlyCost", srF64(t, report, "monthlyCost"), 0)
	for _, k := range []string{"byStorageClass", "byPrefix", "recommendations"} {
		if got := srArr(t, report, k); len(got) != 0 {
			t.Errorf("%s = %#v, want 空数组", k, got)
		}
	}
}

// TestStorageReportErrors 错误分支：未知账号 404 / 缺桶 400 / S3 侧错误 500。
func TestStorageReportErrors(t *testing.T) {
	t.Run("unknown account", func(t *testing.T) {
		env := srFake(t, func(string) string { return srListXML(nil, false, "") })
		rr := env.accDoRec(http.MethodGet, "/api/accounts/nope/storage-report?bucket=b", "")
		olExpectStatus(t, rr, http.StatusNotFound, "unknown account")
	})

	t.Run("missing bucket", func(t *testing.T) {
		srv := olFake(t, func(r *http.Request) olResp { return olResp{} })
		env := accNewEnv(t, srv.URL, "") // 账号无默认桶
		rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
		olExpectStatus(t, rr, http.StatusBadRequest, "missing bucket")
	})

	t.Run("s3 error", func(t *testing.T) {
		srv := olFake(t, func(r *http.Request) olResp {
			if r.Method == http.MethodGet && r.URL.Query().Has("list-type") {
				return olErr(http.StatusInternalServerError, "InternalError")
			}
			return olResp{}
		})
		env := accNewEnv(t, srv.URL, "b")
		rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
		olExpectStatus(t, rr, http.StatusInternalServerError, "s3 list error")
	})
}

// TestStorageReportTruncatedWhenTokenStops 对端声称还有下一页却不给 token：
// 报告标记截断（与 delete-prefix / copy-prefix 的防空转口径一致）。
func TestStorageReportTruncatedWhenTokenStops(t *testing.T) {
	env := srFake(t, func(string) string {
		return srListXML([]srObj{{key: "a.txt", size: 1, age: time.Hour}}, true, "")
	})
	rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
	olExpectStatus(t, rr, http.StatusOK, "token stop report")
	report := srDecode(t, rr.Body.String())
	if got, _ := report["truncated"].(bool); !got {
		t.Errorf("truncated = false, want true")
	}
	if got := srI64(t, report, "objectCount"); got != 1 {
		t.Errorf("objectCount = %d, want 1", got)
	}
}

// TestStorageReportTruncatedAtPageLimit 对端永远给「下一页」（token 持续前进但无对象）：
// 页数到顶必须停并标记截断，不得无限翻页。
func TestStorageReportTruncatedAtPageLimit(t *testing.T) {
	calls := 0
	srv := olFake(t, func(r *http.Request) olResp {
		if r.Method == http.MethodGet && r.URL.Query().Has("list-type") {
			calls++
			next := strconv.Itoa(calls)
			return olXML(http.StatusOK, srListXML(nil, true, next))
		}
		return olResp{}
	})
	env := accNewEnv(t, srv.URL, "b")
	rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
	olExpectStatus(t, rr, http.StatusOK, "page limit report")
	if calls != storageReportMaxPages {
		t.Errorf("列举页数 = %d, want %d（页数上限必须生效）", calls, storageReportMaxPages)
	}
	report := srDecode(t, rr.Body.String())
	if got, _ := report["truncated"].(bool); !got {
		t.Errorf("truncated = false, want true")
	}
}

// TestStorageReportTruncatedAt100kObjects 100k 对象硬上限：
// 第 100001 个起被丢弃并标记截断（与 delete-prefix / sync 的 100_000 口径一致）。
func TestStorageReportTruncatedAt100kObjects(t *testing.T) {
	env := srFake(t, func(token string) string {
		page := 0
		if token != "" {
			page, _ = strconv.Atoi(token)
		}
		objs := make([]srObj, 0, 1000)
		for i := 0; i < 1000; i++ {
			objs = append(objs, srObj{
				key:   fmt.Sprintf("p%03d-%04d", page, i),
				size:  1024,
				class: "STANDARD",
				age:   time.Hour,
			})
		}
		// 第 0..100 页共 101 页：第 100 页的对象将超出 100k 上限。
		return srListXML(objs, page < 100, strconv.Itoa(page+1))
	})
	rr := env.accDoRec(http.MethodGet, "/api/accounts/"+env.acc.ID+"/storage-report", "")
	olExpectStatus(t, rr, http.StatusOK, "100k report")
	report := srDecode(t, rr.Body.String())
	if got := srI64(t, report, "objectCount"); got != 100_000 {
		t.Errorf("objectCount = %d, want 100000", got)
	}
	if got, _ := report["truncated"].(bool); !got {
		t.Errorf("truncated = false, want true")
	}
	if got := srI64(t, report, "totalSize"); got != 100_000*1024 {
		t.Errorf("totalSize = %d, want %d", got, 100_000*1024)
	}
}
