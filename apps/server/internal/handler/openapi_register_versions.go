package handler

import (
	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
)

func registerVersions(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/versions", openapi.Op{
		Tags: []string{"versions"}, Summary: "对象版本列表（含删除标记）", OperationID: "listObjectVersions",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			// 注意：真实 handler listObjectVersions（metadata.go）只读 bucket/prefix/keyMarker/
			// versionIdMarker/maxKeys，**不读 key**。此前误声明 key 为幻影参数（客户端发送后被
			// 忽略），2026-09-19 由 openapi_query_params_test.go 全量门禁发现并移除。
			openapi.Param{Name: "prefix", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "keyMarker", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "versionIdMarker", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "maxKeys", In: "query", Schema: openapi.Int()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 versions/deleteMarkers/isTruncated", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"versions":            openapi.Arr(openapi.Obj()),
			"deleteMarkers":       openapi.Arr(openapi.Obj()),
			"isTruncated":         openapi.Bool(),
			"nextKeyMarker":       openapi.Str(),
			"nextVersionIdMarker": openapi.Str(),
		})}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/version", openapi.Op{
		Tags: []string{"versions"}, Summary: "删除指定版本", OperationID: "deleteObjectVersion",
		// 该端点从 query 读参数（metadata.go deleteObjectVersion），此前误声明为 requestBody：
		// 按 OpenAPI 生成的客户端会把参数放进 DELETE body，handler 收不到 key/versionId 直接 400。
		Params: []openapi.Param{
			acctIDParam(),
			refParam("Bucket"),
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
			openapi.Param{Name: "versionId", In: "query", Required: true, Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"deleted":   openapi.Str(),
			"versionId": openapi.Str(),
		})}, "400": {Description: "缺 key/versionId", JSON: refSchema("Error")}},
	})
	r.Operation("POST", "/api/accounts/{id}/version/restore", openapi.Op{
		Tags: []string{"versions"}, Summary: "把历史版本恢复为当前（复制回 key）", OperationID: "restoreObjectVersion",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":    openapi.Str(),
				"key":       openapi.Str(),
				"versionId": openapi.Str(),
			}, "bucket", "key", "versionId")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"restored":  openapi.Str(),
			"versionId": openapi.Str(),
		})}},
	})
	r.Operation("POST", "/api/accounts/{id}/delete-marker/restore", openapi.Op{
		Tags: []string{"versions"}, Summary: "撤销删除标记（一键还原已删除对象）", OperationID: "restoreDeleteMarker",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":    openapi.Str(),
				"key":       openapi.Str(),
				"versionId": openapi.Str(),
			}, "bucket", "key", "versionId")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"restored":  openapi.Str(),
			"versionId": openapi.Str(),
		})}},
	})
}

// ---- Trash ----
