package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/openapi"
)

func registerSystem(r *openapi.Registry) {
	r.Operation("GET", "/api/health", openapi.Op{
		Tags: []string{"system"}, Summary: "健康检查（含 store 状态与版本）", OperationID: "health",
		Responses: map[string]openapi.Response{
			"200": {Description: "OK", JSON: openapi.Obj()},
			"503": {Description: "store 不可用", JSON: openapi.Obj()},
		},
	})
	r.Operation("GET", "/api/metrics", openapi.Op{
		Tags: []string{"system"}, Summary: "Prometheus 文本指标（默认 404，需 S3C_EXPOSE_METRICS=1）", OperationID: "metrics",
		Responses: map[string]openapi.Response{
			"200": {Description: "text/plain; version=0.0.4", JSON: nil},
			"404": {Description: "默认关闭", JSON: nil},
		},
	})
	r.Operation("GET", "/api/openapi.json", openapi.Op{
		Tags: []string{"system"}, Summary: "OpenAPI 3.0 规范（本文件）", OperationID: "openapi",
		Responses: map[string]openapi.Response{
			"200": {Description: "application/json", JSON: openapi.Obj()},
		},
	})
}

// ---- 共享参数构造器 ----
