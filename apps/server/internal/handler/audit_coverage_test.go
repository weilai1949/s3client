package handler

// audit_coverage_test.go —— 2026-09-28 三路复审（KNOWN_ISSUES #64 Security）：
// 破坏性操作的审计覆盖必须与 threat-model.md 的 R（抵赖）缓解声明一致。
//
// threat-model.md 的 R 行写的是「对象删除与前缀删除」，但下面三处会**真实删掉数据**
// 却没有任何 objects.* 审计事件——持有效 token 的调用方可以不留痕迹地删数据：
//
//  1. 同步 POST /copy-objects + deleteSource（移动 = 复制后删源）；
//     异步版已按 review R3 补了 objects.move（review_20260924_test.go），同步版漏了；
//  2. POST /rename —— 复制成功后 DeleteObject 删源，源 key 永久消失，全程无审计；
//     把删除藏进「重命名」即可绕开已有的 objects.delete 事件；
//  3. DELETE /version —— 永久版本删除；回收站 purge（trash.purge）反而有审计。
//
// 三处全部复用 audit.go 既有的**稳定事件名**（objects.move / objects.delete），
// 不新增契约名——事件名是日志检索与告警的依赖。

import (
	"net/http"
	"testing"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
)

// auditNewAccount 起假 S3 + 建账号，返回账号 id；fake 按 copy-source / method 分流。
func auditNewAccount(t *testing.T, h *Handler, name string, route func(r *http.Request) olResp) string {
	t.Helper()
	srv := olFake(t, route)
	acc, err := h.store.Create(&model.Account{
		Name: name, Endpoint: srv.URL, Region: "us-east-1",
		AccessKey: "ak", SecretKey: "sk", Bucket: "b", PathStyle: true,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	return acc.ID
}

// auditFind 取指定事件名的最后一条审计记录，没有则返回 nil。
func auditFind(t *testing.T, rec *auditRecorder, event string) map[string]any {
	t.Helper()
	var ev map[string]any
	for _, e := range rec.auditEvents(t) {
		if e["audit"] == event {
			ev = e
		}
	}
	return ev
}

// TestCopyManySyncMoveAudits 同步移动（deleteSource=true）必须写 objects.move 审计；
// 纯复制不得写（避免把只读复制记成移动）。与异步版 TestCopyManyAsyncMoveAuditsJobStart
// 同一口径——同一动作只因走同步 / 异步两条路由就可审计性不同，是明确的漏洞。
func TestCopyManySyncMoveAudits(t *testing.T) {
	h, rec := newAuditHandler(t, "", nil)
	id := auditNewAccount(t, h, "audit-move-sync", func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK)
		}
		return olPlain(http.StatusNoContent)
	})
	routes := h.Routes()

	rr := doJSON(t, routes, "POST", "/api/accounts/"+id+"/copy-objects",
		`{"bucket":"b","keys":["a.txt"],"deleteSource":true}`)
	olExpectStatus(t, rr, http.StatusOK, "copy many move")
	if ev := auditFind(t, rec, "objects.move"); ev == nil {
		t.Fatal("同步移动未写 objects.move 审计（源被删却无痕迹，与异步版口径不一致）")
	} else if ev["bucket"] != "b" || ev["targetBucket"] != "b" {
		t.Fatalf("审计字段不符: bucket=%v targetBucket=%v", ev["bucket"], ev["targetBucket"])
	}

	// 纯复制（deleteSource 缺省）不得记 objects.move。
	before := 0
	for _, e := range rec.auditEvents(t) {
		if e["audit"] == "objects.move" {
			before++
		}
	}
	rr2 := doJSON(t, routes, "POST", "/api/accounts/"+id+"/copy-objects",
		`{"bucket":"b","keys":["a.txt"]}`)
	olExpectStatus(t, rr2, http.StatusOK, "copy many copy-only")
	after := 0
	for _, e := range rec.auditEvents(t) {
		if e["audit"] == "objects.move" {
			after++
		}
	}
	if after != before {
		t.Fatalf("纯复制不得写 objects.move 审计：%d → %d", before, after)
	}
}

// TestRenameObjectAudits 重命名 = 复制成功后永久删除源 key，必须留下 objects.move 审计，
// 否则把删除藏进「重命名」即可绕开 objects.delete 事件（threat-model R 的覆盖范围）。
func TestRenameObjectAudits(t *testing.T) {
	h, rec := newAuditHandler(t, "", nil)
	id := auditNewAccount(t, h, "audit-rename", func(r *http.Request) olResp {
		if r.Header.Get("x-amz-copy-source") != "" {
			return olPlain(http.StatusOK) // CopyObject
		}
		return olPlain(http.StatusNoContent) // DeleteObject 删源
	})

	rr := doJSON(t, h.Routes(), "POST", "/api/accounts/"+id+"/rename",
		`{"bucket":"b","key":"a.txt","newKey":"dir/b.txt"}`)
	olExpectStatus(t, rr, http.StatusOK, "rename object")

	ev := auditFind(t, rec, "objects.move")
	if ev == nil {
		t.Fatal("重命名未写 objects.move 审计（源 key 被永久删除却无痕迹，可绕开 objects.delete）")
	}
	if ev["bucket"] != "b" || ev["key"] != "a.txt" || ev["newKey"] != "dir/b.txt" {
		t.Fatalf("审计字段不符: bucket=%v key=%v newKey=%v", ev["bucket"], ev["key"], ev["newKey"])
	}
}

// TestDeleteObjectVersionAudits 永久版本删除必须写 objects.delete 审计（带 versionId），
// 与回收站 purge（trash.purge）同级——版本删除同样让数据不可恢复。
func TestDeleteObjectVersionAudits(t *testing.T) {
	h, rec := newAuditHandler(t, "", nil)
	id := auditNewAccount(t, h, "audit-version", func(r *http.Request) olResp {
		return olPlain(http.StatusNoContent)
	})

	rr := doJSON(t, h.Routes(), "DELETE", "/api/accounts/"+id+"/version?bucket=b&key=a.txt&versionId=v1", "")
	olExpectStatus(t, rr, http.StatusOK, "delete object version")

	ev := auditFind(t, rec, "objects.delete")
	if ev == nil {
		t.Fatal("永久版本删除未写 objects.delete 审计（数据不可恢复却无痕迹）")
	}
	if ev["bucket"] != "b" || ev["key"] != "a.txt" || ev["versionId"] != "v1" {
		t.Fatalf("审计字段不符: bucket=%v key=%v versionId=%v", ev["bucket"], ev["key"], ev["versionId"])
	}
}
