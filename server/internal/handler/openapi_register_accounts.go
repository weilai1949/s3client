package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/openapi"
)

func registerAccounts(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts", openapi.Op{
		Tags:        []string{"accounts"},
		Summary:     "列出全部账号",
		OperationID: "listAccounts",
		Responses: map[string]openapi.Response{
			"200": {Description: "账号列表", JSON: openapi.BuildObj(map[string]*openapi.Schema{
				"accounts": openapi.Arr(openapi.Obj()),
			}, "accounts")},
		},
	})
	r.Operation("POST", "/api/accounts", openapi.Op{
		Tags:        []string{"accounts"},
		Summary:     "新建账号",
		OperationID: "createAccount",
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"name":           openapi.Str(),
				"provider":       openapi.Str(),
				"endpoint":       openapi.Str(),
				"publicEndpoint": openapi.Str(),
				"region":         openapi.Str(),
				"accessKey":      openapi.Str(),
				"secretKey":      openapi.Str(),
				"bucket":         openapi.Str(),
				"pathStyle":      openapi.Bool(),
			}, "name", "endpoint", "accessKey", "secretKey")},
		},
		Responses: map[string]openapi.Response{
			"201": {Description: "已创建（含敏感字段占位）", JSON: openapi.Obj()},
			"400": {Description: "参数错误", JSON: openapi.Obj()},
		},
	})
	r.Operation("GET", "/api/accounts/{id}", openapi.Op{
		Tags:        []string{"accounts"},
		Summary:     "获取账号详情",
		OperationID: "getAccount",
		Params:      []openapi.Param{acctIDParam()},
		Responses: map[string]openapi.Response{
			"200": {Description: "账号", JSON: openapi.Obj()},
			"404": {Description: "不存在", JSON: openapi.Obj()},
		},
	})
	r.Operation("PUT", "/api/accounts/{id}", openapi.Op{
		Tags:        []string{"accounts"},
		Summary:     "更新账号",
		OperationID: "updateAccount",
		Params:      []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: openapi.Obj()},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "更新后", JSON: openapi.Obj()},
			"404": {Description: "不存在", JSON: openapi.Obj()},
		},
	})
	r.Operation("DELETE", "/api/accounts/{id}", openapi.Op{
		Tags:        []string{"accounts"},
		Summary:     "删除账号",
		OperationID: "deleteAccount",
		Params:      []openapi.Param{acctIDParam()},
		Responses: map[string]openapi.Response{
			"200": {Description: "OK", JSON: openapi.Obj()},
			"404": {Description: "不存在", JSON: openapi.Obj()},
		},
	})
	r.Operation("POST", "/api/accounts/{id}/test", openapi.Op{
		Tags:        []string{"accounts"},
		Summary:     "连通性检测（200+ok 表示通；ok=false 含 error）",
		OperationID: "testAccount",
		Params:      []openapi.Param{acctIDParam(), openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{
			"200": {Description: "检测结果（始终 200，字段 ok 表状态）", JSON: openapi.Obj()},
		},
	})
	r.Operation("POST", "/api/accounts/preview-buckets", openapi.Op{
		Tags:        []string{"accounts"},
		Summary:     "用表单凭证预览桶（不落库）",
		OperationID: "previewBuckets",
		Request: &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: openapi.Obj()},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "桶列表", JSON: openapi.Obj()},
		},
	})
}

// ---- Buckets ----
