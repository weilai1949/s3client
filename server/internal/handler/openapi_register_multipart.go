package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/openapi"
)

func registerMultipart(r *openapi.Registry) {
	r.Operation("POST", "/api/accounts/{id}/multipart/init", openapi.Op{
		Tags: []string{"multipart"}, Summary: "初始化分段上传", OperationID: "multipartInit",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":      openapi.Str(),
				"key":         openapi.Str(),
				"contentType": openapi.Str(),
				"metadata":    openapi.Obj(),
			}, "bucket", "key")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 uploadId", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/multipart/part", openapi.Op{
		Tags: []string{"multipart"}, Summary: "预签名单段 PUT URL", OperationID: "multipartPart",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":     openapi.Str(),
				"key":        openapi.Str(),
				"uploadId":   openapi.Str(),
				"partNumber": openapi.Int(),
			}, "bucket", "key", "uploadId", "partNumber")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 url/expiresAt", JSON: openapi.Obj()}, "400": {Description: "partNumber 非法", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/multipart/complete", openapi.Op{
		Tags: []string{"multipart"}, Summary: "完成分段上传", OperationID: "multipartComplete",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":   openapi.Str(),
				"key":      openapi.Str(),
				"uploadId": openapi.Str(),
				"parts":    openapi.Arr(openapi.BuildObj(map[string]*openapi.Schema{"partNumber": openapi.Int(), "etag": openapi.Str()}, "partNumber", "etag")),
			}, "bucket", "key", "uploadId", "parts")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/multipart/abort", openapi.Op{
		Tags: []string{"multipart"}, Summary: "中止分段上传", OperationID: "multipartAbort",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":   openapi.Str(),
				"key":      openapi.Str(),
				"uploadId": openapi.Str(),
			}, "bucket", "key", "uploadId")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
}

// ---- Versions ----
