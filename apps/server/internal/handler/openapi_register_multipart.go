package handler

import (
	"github.com/weilai1949/s3client/apps/server/internal/openapi"
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
			}, "key")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 uploadId", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"uploadId": openapi.Str(), "key": openapi.Str(), "bucket": openapi.Str(),
		})}},
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
				"expiresIn":  openapi.Int(),
			}, "key", "uploadId", "partNumber")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 url/expiresIn", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"partNumber": openapi.Int(), "url": openapi.Str(), "expiresIn": openapi.Int64(),
		})}, "400": {Description: "partNumber 非法", JSON: refSchema("Error")}},
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
			}, "key", "uploadId", "parts")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"completed": openapi.Str(),
		})}},
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
			}, "key", "uploadId")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"aborted": openapi.Bool(),
		})}},
	})
	r.Operation("GET", "/api/accounts/{id}/multipart/parts", openapi.Op{
		Tags: []string{"multipart"}, Summary: "列已上传分段（断点续传对齐）", OperationID: "multipartParts",
		Params: []openapi.Param{
			acctIDParam(),
			refParam("Bucket"),
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
			openapi.Param{Name: "uploadId", In: "query", Required: true, Schema: openapi.Str(), Description: "multipartInit 返回的 UploadID；失效时返回错误，前端据此重新 init"},
		},
		Responses: map[string]openapi.Response{"200": {Description: "已上传分段清单（服务端真实值）", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"parts": openapi.Arr(openapi.BuildObj(map[string]*openapi.Schema{
				"partNumber":   openapi.Int(),
				"etag":         openapi.Str(),
				"size":         openapi.Int64(),
				"lastModified": openapi.Str("date-time"),
			}, "partNumber", "etag", "size", "lastModified")),
		}, "parts")}, "400": {Description: "缺 key/uploadId 或 bucket", JSON: refSchema("Error")}},
	})
}

// ---- Versions ----
