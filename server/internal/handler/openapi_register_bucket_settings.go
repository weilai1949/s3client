package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/openapi"
)

func registerBucketSettings(r *openapi.Registry) {
	bucketQ := openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()}

	// Encryption
	r.Operation("GET", "/api/accounts/{id}/bucket/encryption", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶服务端加密（SSE）", OperationID: "getBucketEncryption",
		Params:    []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置时 configured=false", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket/encryption", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "配置 SSE", OperationID: "putBucketEncryption",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":           openapi.Str(),
				"algorithm":        openapi.EnumStr("AES256", "aws:kms", "aws:kms:dsse"),
				"kmsKeyId":         openapi.Str(),
				"bucketKeyEnabled": openapi.Bool(),
			}, "bucket", "algorithm")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "algorithm 非法", JSON: openapi.Obj()}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/encryption", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除 SSE 配置", OperationID: "deleteBucketEncryption",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})

	// CORS
	r.Operation("GET", "/api/accounts/{id}/bucket/cors", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶 CORS 规则列表", OperationID: "getBucketCors",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回空数组", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket/cors", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "配置 CORS（rules 空数组=删除）", OperationID: "putBucketCors",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"rules":  openapi.Arr(openapi.Obj()),
			}, "bucket", "rules")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/cors", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除 CORS", OperationID: "deleteBucketCors",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})

	// Website
	r.Operation("GET", "/api/accounts/{id}/bucket/website", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶静态网站托管配置", OperationID: "getBucketWebsite",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回 configured=false", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket/website", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "配置静态网站托管", OperationID: "putBucketWebsite",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":                openapi.Str(),
				"indexDocument":         openapi.Str(),
				"errorDocument":         openapi.Str(),
				"redirectAllRequestsTo": openapi.Str(),
			}, "bucket")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "indexDocument/redirectAllRequestsTo 至少一个", JSON: openapi.Obj()}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/website", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除静态网站托管", OperationID: "deleteBucketWebsite",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})

	// Policy
	r.Operation("GET", "/api/accounts/{id}/bucket/policy", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶策略（JSON 字符串）", OperationID: "getBucketPolicy",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回 configured=false", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket/policy", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "配置桶策略（policy=空=删除）", OperationID: "putBucketPolicy",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"policy": openapi.Str("policy JSON 字符串；空字符串=删除"),
			}, "bucket")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "policy 不是合法 JSON", JSON: openapi.Obj()}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/policy", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除桶策略", OperationID: "deleteBucketPolicy",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})

	// Tags
	r.Operation("GET", "/api/accounts/{id}/bucket/tags", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶标签", OperationID: "getBucketTags",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回空数组", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket/tags", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "配置桶标签（空数组=删除）", OperationID: "putBucketTags",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"tags":   openapi.Arr(openapi.BuildObj(map[string]*openapi.Schema{"key": openapi.Str(), "value": openapi.Str()}, "key", "value")),
			}, "bucket", "tags")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/tags", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除桶标签", OperationID: "deleteBucketTags",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
}

// ---- Objects ----
