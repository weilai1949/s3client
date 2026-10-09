package handler

import (
	"github.com/weilai1949/s3client/apps/server/internal/openapi"
)

// openapi_register_schedules.go —— 计划任务 5 端点的契约登记（ROADMAP §三 #6）。
//
// scheduleSchema 用**内联 builder**（同 jobRecordSchema 先例）而非 components 共享
// schema：计划响应只在本组端点出现，内联避免 openapi.go 共享表与 schemaDTOs 登记
// 两处联动；嵌套字段仍由示例门禁递归校验（未知字段 / 缺 required / 类型 / 枚举）。

// scheduleSchema 对应 service.Schedule 的 json tag（15 字段）。
// required 取「无 omitempty/omitzero、恒出现」的字段；sourcePrefix/targetPrefix
// （omitempty）与 lastRunAt（omitzero）、lastJobId/lastError（omitempty）为可选。
func scheduleSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"id":              openapi.Str(),
		"sourceAccountId": openapi.Str(),
		"sourceBucket":    openapi.Str(),
		"sourcePrefix":    openapi.Str(),
		"targetAccountId": openapi.Str(),
		"targetBucket":    openapi.Str(),
		"targetPrefix":    openapi.Str(),
		"mode":            openapi.EnumStr("etag", "size_mtime", "always"),
		"cron":            openapi.Str(),
		"enabled":         openapi.Bool(),
		"createdAt":       openapi.Str("date-time"),
		"nextRunAt":       openapi.Str("date-time"),
		"lastRunAt":       openapi.Str("date-time"),
		"lastJobId":       openapi.Str(),
		"lastError":       openapi.Str(),
	},
		"id", "sourceAccountId", "sourceBucket", "targetAccountId", "targetBucket",
		"mode", "cron", "enabled", "createdAt", "nextRunAt")
}

// scheduleRequestSchema 是 POST/PUT 共用的请求体（字段集与 scheduleRequest DTO
// 由 openapi_request_fields 门禁双向钉住）。required 取 handler 无默认、空值即 400
// 的字段：sourceAccountId/targetAccountId（显式空值校验）、cron（service.Validate 拒空）。
// 桶不标 required（可省略回退账号默认桶），mode 不标（空值归一 etag），enabled 不标（缺省 true）。
func scheduleRequestSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"sourceAccountId": openapi.Str(),
		"sourceBucket":    openapi.Str(),
		"sourcePrefix":    openapi.Str(),
		"targetAccountId": openapi.Str(),
		"targetBucket":    openapi.Str(),
		"targetPrefix":    openapi.Str(),
		"mode":            openapi.EnumStr("etag", "size_mtime", "always"),
		"cron":            openapi.Str(),
		"enabled":         openapi.Bool(),
	},
		"sourceAccountId", "targetAccountId", "cron")
}

// registerSchedules 登记计划任务 5 端点。
func registerSchedules(r *openapi.Registry) {
	// newSchedReq 每次调用返回**独立**的 Request——POST /schedules 与 PUT /schedules/{id}
	// 绝不能共享同一个 *Request：SetExamples 经指针写入 Content.Example，共享指针会让两个
	// operation 的请求示例互相覆盖（apiExamples 是 map，遍历顺序按进程随机 → 提交版规范
	// 与运行时字节级抖动，TestCommittedOpenAPISpecMatchesRuntime 时红时绿；2026-10-08 实测定位）。
	newSchedReq := func() *openapi.Request {
		return &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: scheduleRequestSchema()},
		}
	}
	schedIDParam := []openapi.Param{{Name: "id", In: "path", Required: true, Schema: openapi.Str()}}

	r.Operation("GET", "/api/schedules", openapi.Op{
		Tags: []string{"schedules"}, Summary: "计划任务清单（最新在前）", OperationID: "listSchedules",
		Responses: map[string]openapi.Response{
			"200": {Description: "计划数组（空清单为 []）", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"schedules": openapi.Arr(scheduleSchema()),
			})},
		},
	})
	r.Operation("POST", "/api/schedules", openapi.Op{
		Tags: []string{"schedules"}, Summary: "创建计划（cron 定时增量同步）", OperationID: "createSchedule",
		Request: newSchedReq(),
		Responses: map[string]openapi.Response{
			"201": {Description: "已创建并完成首次排期", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"schedule": scheduleSchema(),
			})},
			"400": {Description: "字段缺失 / cron 非法或永不触发 / mode 非法 / 桶不可解析 / 账号配置无效", JSON: refSchema("Error")},
			"404": {Description: "引用的账号不存在", JSON: refSchema("Error")},
		},
	})
	r.Operation("PUT", "/api/schedules/{id}", openapi.Op{
		Tags: []string{"schedules"}, Summary: "整体替换计划（保留 id/createdAt/运行态）", OperationID: "updateSchedule",
		Params: schedIDParam, Request: newSchedReq(),
		Responses: map[string]openapi.Response{
			"200": {Description: "更新后的计划（cron 变更则重算排期）", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"schedule": scheduleSchema(),
			})},
			"400": {Description: "字段缺失 / cron 或 mode 非法 / 账号配置无效", JSON: refSchema("Error")},
			"404": {Description: "计划或引用的账号不存在", JSON: refSchema("Error")},
		},
	})
	r.Operation("DELETE", "/api/schedules/{id}", openapi.Op{
		Tags: []string{"schedules"}, Summary: "删除计划（冻结的计划一并移除）", OperationID: "deleteSchedule",
		Params: schedIDParam,
		Responses: map[string]openapi.Response{
			"200": {Description: "回显被删计划 id", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"deleted": openapi.Str(),
			})},
			"404": {Description: "计划不存在", JSON: refSchema("Error")},
		},
	})
	r.Operation("POST", "/api/schedules/{id}/run", openapi.Op{
		Tags: []string{"schedules"}, Summary: "立即触发一次（不改自动排期；进度经 /api/migrate/jobs/{id}）", OperationID: "runScheduleNow",
		Params: schedIDParam,
		Responses: map[string]openapi.Response{
			"202": {Description: "异步任务已创建", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"jobId":      openapi.Str(),
				"scheduleId": openapi.Str(),
			})},
			"400": {Description: "引用的账号配置已损坏（缺密钥等）", JSON: refSchema("Error")},
			"404": {Description: "计划或引用的账号不存在", JSON: refSchema("Error")},
			"409": {Description: "上一轮执行尚未结束（不叠加）", JSON: refSchema("Error")},
			"503": {Description: "在册任务已满，稍后重试", JSON: refSchema("Error")},
		},
	})
}
