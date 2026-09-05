package handler

import (
	"github.com/weilai1949/s3clinet/server/internal/openapi"
)

// API 文档集中注册：保持路由旁登记，零注解散落 handler 内。
// 这里只描述「形状」，不复制 handler 的语义；行为真源仍是 handler.go。
func registerOpenAPI(r *openapi.Registry, version string) {
	r.SetInfo(openapi.Info{
		Title:       "s3clinet API",
		Version:     version,
		Description: "s3clinet 是面向 S3 兼容对象存储的多账号 Web 控制台。本文档为 /api/* 端点的 OpenAPI 3.0 契约，所有响应均 JSON（除 /api/health 等纯状态端点）。鉴权：Bearer Token（环境变量 S3C_TOKEN，多值逗号分隔）。",
	})
	r.AddServer(openapi.Server{URL: "/", Description: "同源（前端 Vite 代理或后端 SPA fallback）"})

	registerAccounts(r)
	registerBuckets(r)
	registerBucketSettings(r)
	registerObjects(r)
	registerObjectMeta(r)
	registerMultipart(r)
	registerVersions(r)
	registerTrash(r)
	registerMigrate(r)
	registerSystem(r)
}

// ---- Accounts ----

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

func registerBuckets(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/buckets", openapi.Op{
		Tags: []string{"buckets"}, Summary: "列出账号下全部桶", OperationID: "listBuckets",
		Params:    []openapi.Param{acctIDParam()},
		Responses: map[string]openapi.Response{"200": {Description: "桶列表", JSON: openapi.Obj()}},
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
			"200": {Description: "OK", JSON: openapi.Obj()},
			"400": {Description: "桶名非法 / acl 非法", JSON: openapi.Obj()},
		},
	})
	r.Operation("DELETE", "/api/accounts/{id}/bucket", openapi.Op{
		Tags: []string{"buckets"}, Summary: "删除空桶", OperationID: "deleteBucket",
		Params: []openapi.Param{acctIDParam(), openapi.Param{Name: "name", In: "query", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{
			"200": {Description: "OK", JSON: openapi.Obj()},
			"409": {Description: "桶非空", JSON: openapi.Obj()},
		},
	})
	r.Operation("GET", "/api/accounts/{id}/bucket-info", openapi.Op{
		Tags: []string{"buckets"}, Summary: "桶属性（区域 / 创建时间 / 版本控制）", OperationID: "getBucketInfo",
		Params: []openapi.Param{acctIDParam(), openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
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
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "status 非法", JSON: openapi.Obj()}},
	})
}

// ---- Bucket Settings ----

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

func registerObjects(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/objects", openapi.Op{
		Tags: []string{"objects"}, Summary: "列对象（含公共前缀 / 分页）", OperationID: "listObjects",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "prefix", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "delimiter", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "maxKeys", In: "query", Schema: openapi.Int()},
			openapi.Param{Name: "continuationToken", In: "query", Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("GET", "/api/accounts/{id}/head", openapi.Op{
		Tags: []string{"objects"}, Summary: "对象元数据", OperationID: "headObject",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "404": {Description: "对象不存在", JSON: openapi.Obj()}},
	})
	r.Operation("GET", "/api/accounts/{id}/proxy", openapi.Op{
		Tags: []string{"objects"}, Summary: "对象代理下载 / 预览（流式）", OperationID: "proxyObject",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
			openapi.Param{Name: "mode", In: "query", Schema: openapi.EnumStr("download", "inline", "text")},
			openapi.Param{Name: "versionId", In: "query", Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "二进制流 / text/plain", JSON: nil},
			"404": {Description: "对象不存在", JSON: openapi.Obj()},
		},
	})
	r.Operation("POST", "/api/accounts/{id}/set-headers", openapi.Op{
		Tags: []string{"objects"}, Summary: "编辑 HTTP 头 / 元数据（CopyObject+REPLACE）", OperationID: "setHeaders",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":       openapi.Str(),
				"key":          openapi.Str(),
				"contentType":  openapi.Str(),
				"contentLang":  openapi.Str(),
				"contentEnc":   openapi.Str(),
				"cacheControl": openapi.Str(),
				"disposition":  openapi.Str(),
				"metadata":     openapi.Obj(),
			}, "bucket", "key")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "metadata 校验失败", JSON: openapi.Obj()}},
	})
	r.Operation("GET", "/api/accounts/{id}/lifecycle", openapi.Op{
		Tags: []string{"objects"}, Summary: "生命周期规则（桶级）", OperationID: "getLifecycle",
		Params: []openapi.Param{acctIDParam(), openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回空数组", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/lifecycle", openapi.Op{
		Tags: []string{"objects"}, Summary: "写入生命周期规则（空规则=删除）", OperationID: "putLifecycle",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"rules":  openapi.Arr(openapi.Obj()),
			}, "bucket")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/presign", openapi.Op{
		Tags: []string{"objects"}, Summary: "生成预签名 URL", OperationID: "presign",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":    openapi.Str(),
				"key":       openapi.Str(),
				"method":    openapi.EnumStr("GET", "PUT", "DELETE", "HEAD"),
				"expiresIn": openapi.Int(),
				"versionId": openapi.Str(),
			}, "bucket", "key", "method")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 url/expiresAt", JSON: openapi.Obj()}, "400": {Description: "method/expiresIn 非法", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/mkdir", openapi.Op{
		Tags: []string{"objects"}, Summary: "新建空文件夹（PUT 空对象）", OperationID: "mkdirObject",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"prefix": openapi.Str(),
			}, "bucket", "prefix")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/rename", openapi.Op{
		Tags: []string{"objects"}, Summary: "重命名 / 移动（copy+delete，可跨桶）", OperationID: "renameObject",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":     openapi.Str(),
				"key":        openapi.Str(),
				"newBucket":  openapi.Str("可选；省略=同桶"),
				"newKey":     openapi.Str(),
				"replaceTags":   desc(openapi.Bool(), "保留标签"),
			}, "bucket", "key", "newKey")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/copy-object", openapi.Op{
		Tags: []string{"objects"}, Summary: "单文件复制（不删源）", OperationID: "copyObject",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":    openapi.Str(),
				"key":       openapi.Str(),
				"newBucket": openapi.Str(),
				"newKey":    openapi.Str(),
			}, "bucket", "key", "newBucket", "newKey")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/copy-objects", openapi.Op{
		Tags: []string{"objects"}, Summary: "批量复制（同步）", OperationID: "copyObjects",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"items":        openapi.Arr(openapi.Obj()),
				"targetBucket": openapi.Str(),
				"targetPrefix": openapi.Str(),
				"deleteSource": desc(openapi.Bool(), "true=移动"),
			}, "items", "targetBucket")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 copied/failed/lastError", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/copy-objects/async", openapi.Op{
		Tags: []string{"objects"}, Summary: "批量复制（异步，SSE 进度）", OperationID: "copyObjectsAsync",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: openapi.Obj()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "jobId", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/delete", openapi.Op{
		Tags: []string{"objects"}, Summary: "批量删除（≤1000 keys）", OperationID: "deleteObjects",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":    openapi.Str(),
				"keys":      openapi.Arr(openapi.Str()),
				"versionId": openapi.Str("可选：仅删该版本"),
			}, "bucket", "keys")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 deleted/failed/lastError", JSON: openapi.Obj()}, "400": {Description: "key 数>1000", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/delete-prefix", openapi.Op{
		Tags: []string{"objects"}, Summary: "递归删除前缀（同步流式）", OperationID: "deletePrefix",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"prefix": openapi.Str(),
			}, "bucket", "prefix")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 deleted/lastError", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/delete-prefix/async", openapi.Op{
		Tags: []string{"objects"}, Summary: "递归删除前缀（异步）", OperationID: "deletePrefixAsync",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: openapi.Obj()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "jobId", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/copy-prefix", openapi.Op{
		Tags: []string{"objects"}, Summary: "递归复制前缀（同步流式）", OperationID: "copyPrefix",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":       openapi.Str(),
				"prefix":       openapi.Str(),
				"targetBucket": openapi.Str(),
				"targetPrefix": openapi.Str(),
			}, "bucket", "prefix", "targetBucket", "targetPrefix")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 copied/failed/lastError", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/copy-prefix/async", openapi.Op{
		Tags: []string{"objects"}, Summary: "递归复制前缀（异步）", OperationID: "copyPrefixAsync",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: openapi.Obj()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "jobId", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/download-zip", openapi.Op{
		Tags: []string{"objects"}, Summary: "流式 ZIP 打包下载（≤1000 个）", OperationID: "downloadZip",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"keys":   openapi.Arr(openapi.Str()),
			}, "bucket", "keys")},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "application/zip 流", JSON: nil},
			"400": {Description: "keys 为空 / >1000", JSON: openapi.Obj()},
		},
	})
	r.Operation("POST", "/api/accounts/{id}/storage-class", openapi.Op{
		Tags: []string{"objects"}, Summary: "切换对象存储类型", OperationID: "changeStorageClass",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":       openapi.Str(),
				"key":          openapi.Str(),
				"storageClass": openapi.Str(),
			}, "bucket", "key", "storageClass")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "存储类型非法", JSON: openapi.Obj()}},
	})
}

// ---- Object Metadata (ACL/Tags) ----

func registerObjectMeta(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/object-acl", openapi.Op{
		Tags: []string{"object-meta"}, Summary: "对象 ACL", OperationID: "getObjectAcl",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/object-acl", openapi.Op{
		Tags: []string{"object-meta"}, Summary: "设置对象 ACL", OperationID: "putObjectAcl",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"key":    openapi.Str(),
				"acl":    openapi.EnumStr("private", "public-read", "public-read-write"),
			}, "bucket", "key", "acl")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("GET", "/api/accounts/{id}/object-tags", openapi.Op{
		Tags: []string{"object-meta"}, Summary: "对象标签", OperationID: "getObjectTags",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "未配置返回空数组", JSON: openapi.Obj()}},
	})
	r.Operation("PUT", "/api/accounts/{id}/object-tags", openapi.Op{
		Tags: []string{"object-meta"}, Summary: "设置对象标签（空数组=清空）", OperationID: "putObjectTags",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"key":    openapi.Str(),
				"tags":   openapi.Arr(openapi.BuildObj(map[string]*openapi.Schema{"key": openapi.Str(), "value": openapi.Str()}, "key", "value")),
			}, "bucket", "key", "tags")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
}

// ---- Multipart ----

func registerMultipart(r *openapi.Registry) {
	r.Operation("POST", "/api/accounts/{id}/multipart/init", openapi.Op{
		Tags: []string{"multipart"}, Summary: "初始化分段上传", OperationID: "multipartInit",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":       openapi.Str(),
				"key":          openapi.Str(),
				"contentType":  openapi.Str(),
				"metadata":     openapi.Obj(),
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
				"bucket":   openapi.Str(),
				"key":      openapi.Str(),
				"uploadId": openapi.Str(),
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

func registerVersions(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/versions", openapi.Op{
		Tags: []string{"versions"}, Summary: "对象版本列表（含删除标记）", OperationID: "listObjectVersions",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "key", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "prefix", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "keyMarker", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "versionIdMarker", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "maxKeys", In: "query", Schema: openapi.Int()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 versions/deleteMarkers/isTruncated", JSON: openapi.Obj()}},
	})
	r.Operation("DELETE", "/api/accounts/{id}/version", openapi.Op{
		Tags: []string{"versions"}, Summary: "删除指定版本", OperationID: "deleteObjectVersion",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":    openapi.Str(),
				"key":       openapi.Str(),
				"versionId": openapi.Str(),
			}, "bucket", "key", "versionId")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "缺 key/versionId", JSON: openapi.Obj()}},
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
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/delete-marker/restore", openapi.Op{
		Tags: []string{"versions"}, Summary: "撤销删除标记（一键还原已删除对象）", OperationID: "restoreDeleteMarker",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":         openapi.Str(),
				"key":            openapi.Str(),
				"deleteMarkerId": openapi.Str(),
			}, "bucket", "key", "deleteMarkerId")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
}

// ---- Trash ----

func registerTrash(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/trash", openapi.Op{
		Tags: []string{"trash"}, Summary: "列出桶内全部删除标记（分页游标）", OperationID: "listTrash",
		Params: []openapi.Param{
			acctIDParam(),
			openapi.Param{Name: "bucket", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "keyMarker", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "versionIdMarker", In: "query", Schema: openapi.Str()},
			openapi.Param{Name: "maxKeys", In: "query", Schema: openapi.Int()},
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

func registerMigrate(r *openapi.Registry) {
	migrateReq := openapi.Request{
		Required: true,
		Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
			"srcAccountId":  openapi.Str(),
			"srcBucket":     openapi.Str(),
			"srcPrefix":     openapi.Str(),
			"dstAccountId":  openapi.Str(),
			"dstBucket":     openapi.Str(),
			"dstPrefix":     openapi.Str(),
			"keys":          openapi.Arr(openapi.Str()),
			"deleteSource":  desc(openapi.Bool(), "true=移动"),
			"storageClass":  openapi.Str(),
		})},
	}

	r.Operation("POST", "/api/migrate", openapi.Op{
		Tags: []string{"migrate"}, Summary: "同步迁移（流式）", OperationID: "migrate",
		Request: &migrateReq,
		Responses: map[string]openapi.Response{"200": {Description: "含 copied/failed/failedKeys/lastError", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/migrate/async", openapi.Op{
		Tags: []string{"migrate"}, Summary: "异步迁移（SSE 进度）", OperationID: "migrateAsync",
		Request: &migrateReq,
		Responses: map[string]openapi.Response{"200": {Description: "含 jobId", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/migrate/sync", openapi.Op{
		Tags: []string{"migrate"}, Summary: "增量同步（按 ETag / size+mtime 比对，仅复制差异对象）", OperationID: "migrateSync",
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"sourceAccountId": openapi.Str(),
				"sourceBucket":    openapi.Str(),
				"sourcePrefix":    openapi.Str(),
				"targetAccountId": openapi.Str(),
				"targetBucket":    openapi.Str(),
				"targetPrefix":    openapi.Str(),
				"mode":            openapi.EnumStr(string("etag"), string("size_mtime"), string("always")),
			})},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "含 scanned/skipped/copied/failed/failedKeys/lastError", JSON: openapi.Obj()},
			"400": {Description: "mode 非法 / 账号缺配置", JSON: openapi.Obj()},
			"404": {Description: "账号不存在", JSON: openapi.Obj()},
		},
	})
	r.Operation("GET", "/api/migrate/jobs/{id}", openapi.Op{
		Tags: []string{"migrate"}, Summary: "查询迁移任务状态", OperationID: "migrateJobStatus",
		Params: []openapi.Param{openapi.Param{Name: "id", In: "path", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/migrate/jobs/{id}/cancel", openapi.Op{
		Tags: []string{"migrate"}, Summary: "取消迁移任务", OperationID: "migrateJobCancel",
		Params: []openapi.Param{openapi.Param{Name: "id", In: "path", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}},
	})
	r.Operation("GET", "/api/migrate/jobs/{id}/events", openapi.Op{
		Tags: []string{"migrate"}, Summary: "迁移任务 SSE 进度事件", OperationID: "migrateJobEvents",
		Params: []openapi.Param{openapi.Param{Name: "id", In: "path", Required: true, Schema: openapi.Str()}},
		Responses: map[string]openapi.Response{
			"200": {Description: "text/event-stream", JSON: nil},
		},
	})
}

// ---- System ----

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

func acctIDParam() openapi.Param {
	return openapi.Param{
		Name: "id", In: "path", Required: true,
		Description: "账号 UUID",
		Schema:      openapi.Str(),
	}
}

// desc 给已有 schema 补一段描述，保持 BuildObj 一行式声明不被长串 Description 撑爆。
func desc(s *openapi.Schema, d string) *openapi.Schema {
	if s == nil {
		return nil
	}
	s.Description = d
	return s
}
