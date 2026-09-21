package handler

import (
	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
)

func registerObjects(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/objects", openapi.Op{
		Tags: []string{"objects"}, Summary: "列对象（含公共前缀 / 分页）", OperationID: "listObjects",
		Params: []openapi.Param{
			acctIDParam(),
			refParam("Bucket"),
			refParam("Prefix"),
			openapi.Param{Name: "delimiter", In: "query", Schema: openapi.Str()},
			refParam("MaxKeys"),
			refParam("ContinuationToken"),
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: refSchema("ListObjectsResp")}},
	})
	r.Operation("GET", "/api/accounts/{id}/head", openapi.Op{
		Tags: []string{"objects"}, Summary: "对象元数据", OperationID: "headObject",
		Params: []openapi.Param{
			acctIDParam(),
			refParam("Bucket"),
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "404": refResp("NotFound")},
	})
	r.Operation("GET", "/api/accounts/{id}/proxy", openapi.Op{
		Tags: []string{"objects"}, Summary: "对象代理下载 / 预览（流式）", OperationID: "proxyObject",
		Params: []openapi.Param{
			acctIDParam(),
			refParam("Bucket"),
			openapi.Param{Name: "key", In: "query", Required: true, Schema: openapi.Str()},
			openapi.Param{Name: "mode", In: "query", Schema: openapi.EnumStr("download", "inline", "text")},
			openapi.Param{Name: "versionId", In: "query", Schema: openapi.Str()},
		},
		Responses: map[string]openapi.Response{
			"200": {Description: "二进制流 / text/plain", JSON: nil},
			"404": refResp("NotFound"),
		},
	})
	r.Operation("POST", "/api/accounts/{id}/set-headers", openapi.Op{
		Tags: []string{"objects"}, Summary: "编辑 HTTP 头 / 元数据（CopyObject+REPLACE）", OperationID: "setHeaders",
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
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "metadata 校验失败", JSON: refSchema("Error")}},
	})
	r.Operation("GET", "/api/accounts/{id}/lifecycle", openapi.Op{
		Tags: []string{"objects"}, Summary: "生命周期规则（桶级）", OperationID: "getLifecycle",
		Params:    []openapi.Param{acctIDParam(), refParam("Bucket")},
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
				"bucket": openapi.Str(),
				"key":    openapi.Str(),
				// method 枚举必须与 objects.go presign 的 switch 一致（小写 get|put|post）。
				"method":    openapi.EnumStr("get", "put", "post"),
				"expiresIn": openapi.Int(),
				"versionId": openapi.Str(),
			}, "bucket", "key", "method")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 url/expiresAt", JSON: openapi.Obj()}, "400": {Description: "method/expiresIn 非法", JSON: refSchema("Error")}},
	})
	r.Operation("POST", "/api/accounts/{id}/mkdir", openapi.Op{
		Tags: []string{"objects"}, Summary: "新建空文件夹（PUT 空对象）", OperationID: "mkdirObject",
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
	r.Operation("POST", "/api/accounts/{id}/rename", openapi.Op{
		Tags: []string{"objects"}, Summary: "重命名 / 移动（copy+delete，可跨桶）", OperationID: "renameObject",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket":    openapi.Str(),
				"key":       openapi.Str(),
				"newBucket": openapi.Str("可选；省略=同桶"),
				"newKey":    openapi.Str(),
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
	// copyObjectsBody 是同步 / 异步批量复制共用的请求体：两者共用 copy.go 的同一个 DTO
	// （bucket/targetBucket/targetPrefix/keys/deleteSource），此前同步侧误写成 `items`、
	// 异步侧写成空对象 schema，客户端按 OpenAPI 发送 `items` 会被 DisallowUnknownFields 拒绝。
	copyObjectsBody := openapi.BuildObj(map[string]*openapi.Schema{
		"bucket":       openapi.Str(),
		"targetBucket": openapi.Str(),
		"targetPrefix": openapi.Str(),
		"keys":         openapi.Arr(openapi.Str()),
		"deleteSource": desc(openapi.Bool(), "true=移动"),
	}, "keys")
	r.Operation("POST", "/api/accounts/{id}/copy-objects", openapi.Op{
		Tags: []string{"objects"}, Summary: "批量复制（同步）", OperationID: "copyObjects",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: copyObjectsBody},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 copied/failed/lastError", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/copy-objects/async", openapi.Op{
		Tags: []string{"objects"}, Summary: "批量复制（异步，SSE 进度）", OperationID: "copyObjectsAsync",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content:  openapi.MediaType{Schema: copyObjectsBody},
		},
		Responses: map[string]openapi.Response{"200": {Description: "jobId", JSON: openapi.Obj()}},
	})
	r.Operation("POST", "/api/accounts/{id}/delete", openapi.Op{
		Tags: []string{"objects"}, Summary: "批量删除（≤1000 keys）", OperationID: "deleteObjects",
		Params: []openapi.Param{acctIDParam()},
		Request: &openapi.Request{
			Required: true,
			Content: openapi.MediaType{Schema: openapi.BuildObj(map[string]*openapi.Schema{
				"bucket": openapi.Str(),
				"keys":   openapi.Arr(openapi.Str()),
				// 注意：真实 handler deleteObjects 仅解析 bucket/keys，无 versionId；
				// 指定版本删除走 DELETE /api/accounts/{id}/version。
			}, "bucket", "keys")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "含 deleted/failed/lastError；S3 逐 key 失败仍返回 200，deleted 只计成功数", JSON: openapi.Obj()}, "400": {Description: "key 数>1000", JSON: refSchema("Error")}},
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
		Responses: map[string]openapi.Response{"200": {Description: "含 deleted/failed/truncated/lastError", JSON: openapi.Obj()}},
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
			"400": {Description: "keys 为空 / >1000", JSON: refSchema("Error")},
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
				"versionId":    openapi.Str(),
				"storageClass": openapi.Str(),
			}, "bucket", "key", "storageClass")},
		},
		Responses: map[string]openapi.Response{"200": {Description: "OK", JSON: openapi.Obj()}, "400": {Description: "存储类型非法", JSON: refSchema("Error")}},
	})
}

// ---- Object Metadata (ACL/Tags) ----

func registerObjectMeta(r *openapi.Registry) {
	r.Operation("GET", "/api/accounts/{id}/object-acl", openapi.Op{
		Tags: []string{"object-meta"}, Summary: "对象 ACL", OperationID: "getObjectAcl",
		Params: []openapi.Param{
			acctIDParam(),
			refParam("Bucket"),
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
			refParam("Bucket"),
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
