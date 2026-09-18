package handler

import (
	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
)

func registerTrash(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/trash", openapi.Op{
		Tags: []string{"trash"}, Summary: "列出桶内全部删除标记（分页游标）", OperationID: "listTrash",
		Params: []openapi.Param{
			acctIDParam(),
			refParam("Bucket"),
			openapi.Param{Name: "keyMarker", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "versionIdMarker", In: "query", Schema: openapi.Str()},
			refParam("MaxKeys"),
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/trash/purge", openapi.Op{
		Tags: []string{"trash"}, Summary: "彻底清除某 key 的全部版本+标记", OperationID: "purgeTrashObject",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"key":    openapi.Str(),
			}, "bucket", "key")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
}

// ---- Migrate ----
