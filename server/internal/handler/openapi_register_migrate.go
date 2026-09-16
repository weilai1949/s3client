package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/openapi"
)

func registerMigrate(r *openapi.Registry) {
	// migrateRequest 字段必须与 migrate_exec.go 的 migrateRequest 保持一致：
	// sourceAccountId/sourceBucket/sourceKeys/targetAccountId/targetBucket/targetPrefix。
	// （2026-09-16 评估 H1：此前误写 srcAccountId/srcBucket/srcPrefix/dstAccountId/...，
	//  且虚构了 deleteSource/storageClass 字段；契约测试 TestOpenAPI_ContractRequestBodyMatchesHandlers 兜底。）
	migrateReq := openapi.Request{
		Required: true,
		Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
			"sourceAccountId": openapi.Str(),
			"sourceBucket":    openapi.Str(),
			"sourceKeys":      openapi.Arr(openapi.Str()),
			"targetAccountId": openapi.Str(),
			"targetBucket":    openapi.Str(),
			"targetPrefix":    openapi.Str(),
		})},
	}

	r.Operation("POST", "/api/migrate", openapi.Op{
		Tags: []string{"migrate"}, Summary: "同步迁移（流式）", OperationID: "migrate",
		Request:   &migrateReq,
		Responses: map[string]openapi.Response{"200": {Description: "含 copied/failed/failedKeys/lastError", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/migrate/async", openapi.Op{
		Tags: []string{"migrate"}, Summary: "异步迁移（SSE 进度）", OperationID: "migrateAsync",
		Request:   &migrateReq,
		Responses: map[string]openapi.Response{"200": {Description: "含 jobId", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/migrate/sync", openapi.Op{
		Tags: []string{"migrate"}, Summary: "增量同步（按 ETag / size+mtime 比对，仅复制差异对象）", OperationID: "migrateSync",
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"sourceAccountId": openapi.Str(),
				"sourceBucket":    openapi.Str(),
				"sourcePrefix":    openapi.Str(),
				"targetAccountId": openapi.Str(),
				"targetBucket":    openapi.Str(),
				"targetPrefix":    openapi.Str(),
				"mode":            openapi.EnumStr(string("etag"), string("size_mtime"), string("always")),
			})},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "含 scanned/skipped/copied/failed/failedKeys/lastError", JSON: openapi.Obj()},
			"400": {Description: "mode 非法 / 账号缺配置", JSON: refSchema("Error")},
			"404": {Description: "账号不存在", JSON: refSchema("Error")},
		},
	})
	r.Operation("GET", "/api/migrate/jobs/{id}", openapi.Op{
		Tags: []string{"migrate"}, Summary: "查询迁移任务状态", OperationID: "migrateJobStatus",
		Params:    []openapi.Param{openapi.Param{Name: "id", In: "path", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/migrate/jobs/{id}/cancel", openapi.Op{
		Tags: []string{"migrate"}, Summary: "取消迁移任务", OperationID: "migrateJobCancel",
		Params:    []openapi.Param{openapi.Param{Name: "id", In: "path", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("GET", "/api/migrate/jobs/{id}/events", openapi.Op{
		Tags: []string{"migrate"}, Summary: "迁移任务 SSE 进度事件", OperationID: "migrateJobEvents",
		Params: []openapi.Param{openapi.Param{Name: "id", In: "path", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{
			"200": {Description: "text/event-stream", JSON: nil},
		},
	})
}

// ---- System ----
