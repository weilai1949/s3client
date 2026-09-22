package handler

import (
	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
)

func registerBuckets(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/buckets", openapi.Op{
		Tags: []string{"buckets"}, Summary: "列出账号下全部桶", OperationID: "listBuckets",
		Params:    []openapi.Param{acctIDParam()},
		Responses: map[string]openapi.Response{"200": {Description: "桶列表", JSON: openapi.BuildObj(map[string]*openapi.Schema{"buckets": openapi.Arr(refSchema("Bucket"))}, "buckets")}},
	})
	r.Operation("POST", "/api/accounts/{id}/bucket", openapi.Op{
		Tags: []string{"buckets"}, Summary: "创建桶", OperationID: "createBucket",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"name":   openapi.Str(),
				"region": openapi.Str(),
				"acl":    openapi.EnumStr("", "private", "public-read", "public-read-write"),
			}, "name")},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"created": openapi.Str(),
				"region":  openapi.Str(),
				"acl":     openapi.Str(),
			})},
			"400": refResp("BadRequest"),
		},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket", openapi.Op{
		Tags: []string{"buckets"}, Summary: "删除空桶", OperationID: "deleteBucket",
		Params: []openapi.Param{acctIDParam(), openapi.Param{Name: "name", In: "query", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{
			"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"deleted": openapi.Str(),
			})},
			"409": {Description: "桶非空", JSON: refSchema("Error")},
		},
	})
	r.Operation("GET", "/api/accounts/{id}/bucket-info", openapi.Op{
		Tags: []string{"buckets"}, Summary: "桶属性（区域 / 创建时间 / 版本控制）", OperationID: "getBucketInfo",
		Params: []openapi.Param{acctIDParam(), refParam("Bucket")},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"bucket":     openapi.Str(),
			"region":     openapi.Str(),
			"createdAt":  openapi.Str("date-time"),
			"versioning": openapi.Str(),
		})}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket-versioning", openapi.Op{
		Tags: []string{"buckets"}, Summary: "开关桶版本控制（Enabled / Suspended）", OperationID: "putBucketVersioning",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"status": openapi.EnumStr("Enabled", "Suspended"),
			}, "bucket", "status")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"versioning": openapi.Str(),
		})}, "400": refResp("BadRequest")},
	})
}

// ---- Bucket Settings ----
