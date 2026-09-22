package handler

import (
	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
)

func registerOpenAPI(r *openapi.Registry, version string) {
	r.SetInfo(openapi.Info{
		Title:       "s3clinet API",
		Version:     version,
		Description: "s3clinet 是面向 S3 兼容对象存储的多账号 Web 控制台。本文档为 /api/* 端点的 OpenAPI 3.0 契约，所有响应均 JSON（除 /api/health 等纯状态端点）。鉴权：Bearer Token（环境变量 S3C_TOKEN，多值逗号分隔）。",
	})
	r.AddServer(openapi.Server{URL: "/", Description: "同源（前端 Vite 代理或后端 SPA fallback）"})

	registerAccounts(r)
	registerBuckets(r)
	registerBucketSettings(r)
	registerObjects(r)
	registerObjectMeta(r)
	registerMultipart(r)
	registerVersions(r)
	registerTrash(r)
	registerMigrate(r)
	registerSystem(r)
}

// ---- Accounts ----

// refParam 构造指向 components.parameters 的 $ref 参数（如 "AccountID"）。
func refParam(component string) openapi.Param {
	return openapi.Param{Ref: "#/components/parameters/" + component}
}

// refResp 构造指向 components.responses 的 $ref 响应（如 "NotFound"）。
func refResp(component string) openapi.Response {
	return openapi.Response{Ref: "#/components/responses/" + component}
}

// refSchema 构造指向 components.schemas 的 $ref schema（如 "Account"），用于响应体。
func refSchema(component string) *openapi.Schema {
	return openapi.Ref("#/components/schemas/" + component)
}

// acctIDParam 账号 UUID 路径参数（复用共享 AccountID 参数）。
func acctIDParam() openapi.Param {
	return refParam("AccountID")
}

// desc 给已有 schema 补一段描述，保持 BuildObj 一行式声明不被长串 Description 撑爆。

func desc(s *openapi.Schema, d string) *openapi.Schema {
	if s == nil {
		return nil
	}
	s.Description = d
	return s
}

// tagRowSchema 是「key/value 标签行」的共享 schema。
// 桶标签与对象标签的响应元素形状相同（handler 均写 map[string]string{"key","value"}），
// 抽出来避免两处各写一份而漂移。
func tagRowSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"key":   openapi.Str(),
		"value": openapi.Str(),
	}, "key", "value")
}

// corsRuleSchema 是桶 CORS 规则的共享 schema，字段名必须与 s3wrap.CorsRule 的 json tag
// 一致（camelCase）——前端 types.ts 的 CorsRule 按 camelCase 读取。
func corsRuleSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"id":             openapi.Str(),
		"allowedMethods": openapi.Arr(openapi.Str()),
		"allowedOrigins": openapi.Arr(openapi.Str()),
		"allowedHeaders": openapi.Arr(openapi.Str()),
		"exposeHeaders":  openapi.Arr(openapi.Str()),
		"maxAgeSeconds":  openapi.Int(),
	}, "allowedMethods", "allowedOrigins")
}

// jobProgressSchema 对应 service.JobProgress 的 json tag（异步任务进度）。
func jobProgressSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"done":     openapi.Int(),
		"total":    openapi.Int(),
		"migrated": openapi.Int(),
		"failed":   openapi.Int(),
		"key":      openapi.Str(),
		"error":    openapi.Str(),
		"status":   openapi.Str(),
	})
}

// jobResultSchema 对应 service.JobResult 的 json tag（异步任务终态汇总）。
func jobResultSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"migrated":  openapi.Int(),
		"failed":    openapi.Int(),
		"lastError": openapi.Str(),
		"failKeys":  openapi.Arr(openapi.Str()),
	})
}

// jobRecordSchema 对应 service.JobRecord 的 json tag（任务清单条目）。
func jobRecordSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"id":       openapi.Str(),
		"created":  openapi.Str("date-time"),
		"total":    openapi.Int(),
		"status":   openapi.EnumStr("running", "done", "cancelled", "interrupted"),
		"progress": jobProgressSchema(),
		"result":   jobResultSchema(),
	}, "id", "created", "total", "status", "progress", "result")
}

// versionEntrySchema 是版本列表元素（listObjectVersions 写出的键）。
func versionEntrySchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"key":          openapi.Str(),
		"versionId":    openapi.Str(),
		"isLatest":     openapi.Bool(),
		"lastModified": openapi.Str("date-time"),
		"size":         openapi.Int64(),
		"etag":         openapi.Str(),
		"storageClass": openapi.Str(),
	}, "key", "versionId", "isLatest", "lastModified", "size", "etag", "storageClass")
}

// deleteMarkerSchema 是删除标记元素（trash / versions 两个端点写出形状相同）。
func deleteMarkerSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"key":          openapi.Str(),
		"versionId":    openapi.Str(),
		"isLatest":     openapi.Bool(),
		"lastModified": openapi.Str("date-time"),
	}, "key", "versionId", "isLatest", "lastModified")
}

// lifecycleRuleSchema 是生命周期规则元素（读 / 写两侧形状一致）。
func lifecycleRuleSchema() *openapi.Schema {
	return openapi.BuildObj(map[string]*openapi.Schema{
		"id":     openapi.Str(),
		"prefix": openapi.Str(),
		"days":   openapi.Int(),
	}, "id", "prefix", "days")
}
