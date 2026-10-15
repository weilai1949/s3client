package handler

import (
	"github.com/weilai1949/s3client/apps/server/internal/openapi"
)

func registerSystem(r *openapi.Registry) {
	// health 的 store 子对象：正常时仅 {ok:true}，store 不可用时追加 error。
	healthStore := openapi.BuildObj(map[string]*openapi.Schema{
		"ok":    openapi.Bool(),
		"error": openapi.Str(),
	}, "ok")
	healthBody := openapi.BuildObj(map[string]*openapi.Schema{
		"status": openapi.EnumStr("ok", "error"),
		"time":   openapi.Str("date-time"),
		"store":  healthStore,
	}, "status", "time", "store")

	// health 始终免鉴权；metrics 可由 S3C_METRICS_TOKEN 条件鉴权，
	// 契约用「metricsBearerAuth 或匿名」两个 alternatives 表达。
	r.Operation("GET", "/api/health", openapi.Op{
		NoAuth: true, Tags: []string{"system"}, Summary: "健康检查（status/store；有效 API token 可见 version）", OperationID: "health",
		Responses: map[string]openapi.Response{
			"200": {Description: "OK", JSON: healthBody},
			"503": {Description: "store 不可用", JSON: healthBody},
		},
	})
	r.Operation("GET", "/api/metrics", openapi.Op{
		Tags: []string{"system"}, Summary: "Prometheus 文本指标（默认 404，需 S3C_EXPOSE_METRICS=1；S3C_METRICS_TOKEN 可选鉴权）", OperationID: "metrics",
		Security: []map[string][]string{{"metricsBearerAuth": {}}, {}},
		Responses: map[string]openapi.Response{
			"200": {Description: "text/plain; version=0.0.4", JSON: nil},
			"404": {Description: "默认关闭", JSON: nil},
		},
	})
	// /api/openapi.json 不在豁免名单：配置 S3C_TOKEN 后仍需 Bearer（见 openapi_handler_test.go）。
	r.Operation("GET", "/api/openapi.json", openapi.Op{
		Tags: []string{"system"}, Summary: "OpenAPI 3.0 规范（本文件；默认 404，需 S3C_EXPOSE_OPENAPI=1）", OperationID: "openapi",
		Responses: map[string]openapi.Response{
			"200": {Description: "application/json", JSON: openapi.Obj()},
		},
	})
}

// ---- 共享参数构造器 ----
