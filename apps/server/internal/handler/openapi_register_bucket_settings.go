package handler

import (
	"github.com/weilai1949/s3client/apps/server/internal/openapi"
)

func registerBucketSettings(r *openapi.Registry) {
	bucketQ := refParam("Bucket")

	// Encryption
	r.Operation("GET", "/api/accounts/{id}/bucket/encryption", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶服务端加密（SSE）", OperationID: "getBucketEncryption",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置时 configured=false", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"bucket": openapi.Str(), "configured": openapi.Bool(), "algorithm": openapi.Str(),
			"kmsKeyId": openapi.Str(), "bucketKeyEnabled": openapi.Bool(),
		})}},
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
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"configured": openapi.Bool(), "algorithm": openapi.Str(),
		})}, "400": {Description: "algorithm 非法", JSON: refSchema("Error")}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/encryption", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除 SSE 配置", OperationID: "deleteBucketEncryption",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"deleted": openapi.Str(),
		})}},
	})

	// CORS
	r.Operation("GET", "/api/accounts/{id}/bucket/cors", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶 CORS 规则列表", OperationID: "getBucketCors",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回空数组", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"bucket": openapi.Str(),
			"rules":  openapi.Arr(corsRuleSchema()),
		})}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket/cors", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "配置 CORS（rules 空数组=删除）", OperationID: "putBucketCors",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"rules":  openapi.Arr(corsRuleSchema()),
			}, "bucket", "rules")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "rules 非空时 updated；空数组触发删除时 deleted", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"updated": openapi.Int(),
			"deleted": openapi.Str(),
		})}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/cors", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除 CORS", OperationID: "deleteBucketCors",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"deleted": openapi.Str(),
		})}},
	})

	// Website
	r.Operation("GET", "/api/accounts/{id}/bucket/website", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶静态网站托管配置", OperationID: "getBucketWebsite",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回 configured=false", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"bucket": openapi.Str(), "configured": openapi.Bool(),
			"indexDocument": openapi.Str(), "errorDocument": openapi.Str(), "redirectAllRequestsTo": openapi.Str(),
		})}},
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
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"configured": openapi.Bool(),
		})}, "400": {Description: "indexDocument/redirectAllRequestsTo 至少一个", JSON: refSchema("Error")}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/website", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除静态网站托管", OperationID: "deleteBucketWebsite",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"deleted": openapi.Str(),
		})}},
	})

	// Policy
	r.Operation("GET", "/api/accounts/{id}/bucket/policy", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶策略（JSON 字符串）", OperationID: "getBucketPolicy",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回 configured=false", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"bucket": openapi.Str(), "configured": openapi.Bool(), "policy": openapi.Str(),
		})}},
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
		Responses: map[string]openapi.Response{"200": {Description: "policy 非空时 configured；空字符串触发删除时 deleted", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"configured": openapi.Bool(),
			"deleted":    openapi.Str(),
		})}, "400": {Description: "policy 不是合法 JSON", JSON: refSchema("Error")}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/policy", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除桶策略", OperationID: "deleteBucketPolicy",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"deleted": openapi.Str(),
		})}},
	})

	// Tags
	r.Operation("GET", "/api/accounts/{id}/bucket/tags", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶标签", OperationID: "getBucketTags",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回空数组", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"bucket": openapi.Str(),
			"tags":   openapi.Arr(tagRowSchema()),
		})}},
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
		Responses: map[string]openapi.Response{"200": {Description: "tags 非空时 updated；空数组触发删除时 deleted", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"updated": openapi.Int(),
			"deleted": openapi.Str(),
		})}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket/tags", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "删除桶标签", OperationID: "deleteBucketTags",
		Params: []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.BuildObj(map[string]*openapi.Schema{
			"deleted": openapi.Str(),
		})}},
	})

	// Object Lock（WORM）：桶级默认保留策略。桶须在创建时启用 Object Lock，
	// 否则 PUT 返回 409（InvalidBucketState）；GET 未启用 → enabled=false。
	objectLockResp := openapi.BuildObj(map[string]*openapi.Schema{
		"bucket":                openapi.Str(),
		"enabled":               openapi.Bool(),
		"defaultRetentionMode":  desc(openapi.Str(), "GOVERNANCE | COMPLIANCE（未配置默认保留时为空串）"),
		"defaultRetentionDays":  openapi.Int(),
		"defaultRetentionYears": openapi.Int(),
	})
	r.Operation("GET", "/api/accounts/{id}/bucket/object-lock", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "桶 Object Lock 配置", OperationID: "getObjectLock",
		Params:    []openapi.Param{acctIDParam(), bucketQ},
		Responses: map[string]openapi.Response{"200": {Description: "未启用 Object Lock 时 enabled=false", JSON: objectLockResp}},
	})
	r.Operation("PUT", "/api/accounts/{id}/bucket/object-lock", openapi.Op{
		Tags: []string{"bucket-settings"}, Summary: "设置桶默认保留策略（桶须创建时启用 Object Lock）", OperationID: "putObjectLock",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":                openapi.Str(),
				"defaultRetentionMode":  openapi.EnumStr("GOVERNANCE", "COMPLIANCE"),
				"defaultRetentionDays":  desc(openapi.Int(), "与 defaultRetentionYears 二选一；必须 ≥1"),
				"defaultRetentionYears": desc(openapi.Int(), "与 defaultRetentionDays 二选一；必须 ≥1"),
			}, "bucket", "defaultRetentionMode")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: objectLockResp},
			"400": {Description: "输入非法 / 桶未启用 Object Lock", JSON: refSchema("Error")},
			"409": {Description: "桶未在创建时启用 Object Lock（InvalidBucketState）", JSON: refSchema("Error")},
			"501": {Description: "厂商未实现 Object Lock（NotImplemented）", JSON: refSchema("Error")}},
	})
}

// ---- Objects ----
