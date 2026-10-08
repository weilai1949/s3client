package handler

import (
	"encoding/json"
	"strings"

	"github.com/weilai1949/s3client/apps/server/internal/openapi"
)

// openapi_examples.go —— 每个 operation 的请求 / 响应示例（登记后由 applyExamples 附加到注册表）。
//
// 为什么集中在一个文件、而不是把 example 内联进 openapi_register_*.go：
//   - 示例是纯文档数据（占规范体积的大头），内联会把「路由 + 参数 + schema」的结构信息淹没在
//     几十个长 JSON 字面量里；集中一处后可以整体 review「每个端点长什么样」。
//   - 所有示例都是**合法 JSON 字面量**，`ex` 在构造期校验（非法 JSON 直接 panic），
//     门禁测试再机械校验「字段 ∈ schema properties、required 齐全、类型 / 枚举吻合」，
//     因此示例不会悄悄腐烂成幻影字段（见 apps/server/openapi_examples_gate_test.go）。
//
// 维护约定：新增 / 变更 operation 时同步此文件；`SetExamples` 对未注册的 method+path 或
// 状态码返回 error，拼错会立刻在构造期炸出来，不会静默丢示例。

// ex 是 openapi.Ex 的短别名：本文件有大量示例字面量，短名字显著提升可读性。
func ex(raw string) json.RawMessage { return openapi.Ex(raw) }

// apiExamples 以 "METHOD /path" 为键（与 registerOpenAPI 的注册调用一一对应）。
var apiExamples = map[string]openapi.OpExample{
	// ---- accounts ----
	"GET /api/accounts": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"accounts":[{"id":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","name":"minio","endpoint":"http://localhost:9000","publicEndpoint":"https://s3.example.com","region":"us-east-1","accessKey":"AKIAEXAMPLE","secretSet":true,"bucket":"my-bucket","pathStyle":true,"useSSL":false,"createdAt":"2026-09-30T05:00:00Z","updatedAt":"2026-09-30T05:00:00Z"}]}`),
		},
	},
	"POST /api/accounts": {
		Request: ex(`{"name":"minio","endpoint":"http://localhost:9000","accessKey":"AKIAEXAMPLE","secretKey":"secret","bucket":"my-bucket","pathStyle":true,"useSSL":false}`),
		Responses: map[string]json.RawMessage{
			"201": ex(`{"id":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","name":"minio","endpoint":"http://localhost:9000","publicEndpoint":"https://s3.example.com","region":"us-east-1","accessKey":"AKIAEXAMPLE","secretSet":true,"bucket":"my-bucket","pathStyle":true,"useSSL":false,"createdAt":"2026-09-30T05:00:00Z","updatedAt":"2026-09-30T05:00:00Z"}`),
		},
	},
	"GET /api/accounts/{id}": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"id":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","name":"minio","endpoint":"http://localhost:9000","publicEndpoint":"https://s3.example.com","region":"us-east-1","accessKey":"AKIAEXAMPLE","secretSet":true,"bucket":"my-bucket","pathStyle":true,"useSSL":false,"createdAt":"2026-09-30T05:00:00Z","updatedAt":"2026-09-30T05:00:00Z"}`),
		},
	},
	"PUT /api/accounts/{id}": {
		Request: ex(`{"name":"minio","endpoint":"http://localhost:9000","accessKey":"AKIAEXAMPLE","bucket":"my-bucket","pathStyle":true,"useSSL":false}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"id":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","name":"minio","endpoint":"http://localhost:9000","publicEndpoint":"https://s3.example.com","region":"us-east-1","accessKey":"AKIAEXAMPLE","secretSet":true,"bucket":"my-bucket","pathStyle":true,"useSSL":false,"createdAt":"2026-09-30T05:00:00Z","updatedAt":"2026-09-30T05:00:00Z"}`),
		},
	},
	"DELETE /api/accounts/{id}": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d"}`),
		},
	},
	"POST /api/accounts/{id}/test": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"ok":true,"bucket":"my-bucket"}`),
		},
	},
	"POST /api/accounts/preview-buckets": {
		Request: ex(`{"endpoint":"http://localhost:9000","accessKey":"AKIAEXAMPLE","secretKey":"secret","pathStyle":true,"useSSL":false}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"buckets":[{"name":"my-bucket","creationDate":"2026-09-30T05:00:00Z"}]}`),
		},
	},

	// ---- buckets ----
	"GET /api/accounts/{id}/buckets": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"buckets":[{"name":"my-bucket","creationDate":"2026-09-30T05:00:00Z"}]}`),
		},
	},
	"POST /api/accounts/{id}/bucket": {
		Request: ex(`{"name":"my-bucket","region":"us-east-1","acl":"private"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"created":"my-bucket","region":"us-east-1","acl":"private"}`),
		},
	},
	"DELETE /api/accounts/{id}/bucket": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"my-bucket"}`),
		},
	},
	"GET /api/accounts/{id}/bucket-info": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"bucket":"my-bucket","region":"us-east-1","createdAt":"2026-09-30T05:00:00Z","versioning":"Enabled"}`),
		},
	},
	"PUT /api/accounts/{id}/bucket-versioning": {
		Request: ex(`{"bucket":"my-bucket","status":"Enabled"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"versioning":"Enabled"}`),
		},
	},

	// ---- bucket-settings ----
	"GET /api/accounts/{id}/bucket/encryption": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"bucket":"my-bucket","configured":true,"algorithm":"AES256","kmsKeyId":"","bucketKeyEnabled":false}`),
		},
	},
	"PUT /api/accounts/{id}/bucket/encryption": {
		Request: ex(`{"bucket":"my-bucket","algorithm":"AES256","bucketKeyEnabled":false}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"configured":true,"algorithm":"AES256"}`),
		},
	},
	"DELETE /api/accounts/{id}/bucket/encryption": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"my-bucket"}`),
		},
	},
	"GET /api/accounts/{id}/bucket/cors": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"bucket":"my-bucket","rules":[{"id":"r1","allowedMethods":["GET","PUT"],"allowedOrigins":["https://app.example.com"],"allowedHeaders":["*"],"exposeHeaders":["ETag"],"maxAgeSeconds":3600}]}`),
		},
	},
	"PUT /api/accounts/{id}/bucket/cors": {
		Request: ex(`{"bucket":"my-bucket","rules":[{"id":"r1","allowedMethods":["GET","PUT"],"allowedOrigins":["https://app.example.com"],"allowedHeaders":["*"],"exposeHeaders":["ETag"],"maxAgeSeconds":3600}]}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"updated":1}`),
		},
	},
	"DELETE /api/accounts/{id}/bucket/cors": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"my-bucket"}`),
		},
	},
	"GET /api/accounts/{id}/bucket/website": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"bucket":"my-bucket","configured":true,"indexDocument":"index.html","errorDocument":"error.html","redirectAllRequestsTo":""}`),
		},
	},
	"PUT /api/accounts/{id}/bucket/website": {
		Request: ex(`{"bucket":"my-bucket","indexDocument":"index.html","errorDocument":"error.html"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"configured":true}`),
		},
	},
	"DELETE /api/accounts/{id}/bucket/website": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"my-bucket"}`),
		},
	},
	"GET /api/accounts/{id}/bucket/policy": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"bucket":"my-bucket","configured":true,"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`),
		},
	},
	"PUT /api/accounts/{id}/bucket/policy": {
		Request: ex(`{"bucket":"my-bucket","policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"configured":true}`),
		},
	},
	"DELETE /api/accounts/{id}/bucket/policy": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"my-bucket"}`),
		},
	},
	"GET /api/accounts/{id}/bucket/tags": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"bucket":"my-bucket","tags":[{"key":"env","value":"prod"}]}`),
		},
	},
	"PUT /api/accounts/{id}/bucket/tags": {
		Request: ex(`{"bucket":"my-bucket","tags":[{"key":"env","value":"prod"}]}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"updated":1}`),
		},
	},
	"DELETE /api/accounts/{id}/bucket/tags": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"my-bucket"}`),
		},
	},

	// ---- objects ----
	"GET /api/accounts/{id}/objects": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"objects":[{"key":"docs/a.txt","size":17,"lastModified":"2026-09-30T05:00:00Z","etag":"\"9c1d2f3a4b5c6d7e\"","storageClass":"STANDARD","isDir":false}],"commonPrefixes":["docs/"],"isTruncated":false,"nextToken":""}`),
		},
	},
	"GET /api/accounts/{id}/head": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"key":"docs/a.txt","size":17,"lastModified":"2026-09-30T05:00:00Z","etag":"\"9c1d2f3a4b5c6d7e\"","contentType":"text/plain; charset=utf-8","storageClass":"STANDARD","metadata":{"owner":"alice"}}`),
		},
	},
	"GET /api/accounts/{id}/proxy": {
		Responses: map[string]json.RawMessage{
			"200": ex(`"<对象字节流；text 模式为纯文本前 1 MiB>"`),
		},
		ContentTypes: map[string]string{"200": "application/octet-stream"},
	},
	"POST /api/accounts/{id}/set-headers": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","contentType":"text/markdown","metadata":{"owner":"alice"}}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"updated":"docs/a.txt"}`),
		},
	},
	"GET /api/accounts/{id}/lifecycle": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"rules":[{"id":"expire-logs","prefix":"logs/","days":30}]}`),
		},
	},
	"PUT /api/accounts/{id}/lifecycle": {
		Request: ex(`{"bucket":"my-bucket","rules":[{"id":"expire-logs","prefix":"logs/","days":30}]}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"updated":1}`),
		},
	},
	"POST /api/accounts/{id}/presign": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","method":"get","expiresIn":3600}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"method":"get","bucket":"my-bucket","key":"docs/a.txt","url":"https://s3.example.com/my-bucket/docs/a.txt?X-Amz-Signature=...","expiresIn":3600}`),
		},
	},
	"POST /api/accounts/{id}/mkdir": {
		Request: ex(`{"bucket":"my-bucket","key":"docs"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"created":"docs/","bucket":"my-bucket"}`),
		},
	},
	"POST /api/accounts/{id}/rename": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","newKey":"archive/a.txt"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"renamed":"archive/a.txt"}`),
		},
	},
	"POST /api/accounts/{id}/copy-object": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","newBucket":"archive","newKey":"a.txt"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"copied":"a.txt","bucket":"archive"}`),
		},
	},
	"POST /api/accounts/{id}/copy-objects": {
		Request: ex(`{"bucket":"my-bucket","targetBucket":"archive","targetPrefix":"2026/","keys":["docs/a.txt","docs/b.txt"],"deleteSource":false}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"copied":2,"failed":0,"total":2}`),
		},
	},
	"POST /api/accounts/{id}/copy-objects/async": {
		Request: ex(`{"bucket":"my-bucket","targetBucket":"archive","targetPrefix":"2026/","keys":["docs/a.txt","docs/b.txt"],"deleteSource":false}`),
		Responses: map[string]json.RawMessage{
			"202": ex(`{"jobId":"6f1e2d3c-4b5a-6789-abcd-ef0123456789","total":2}`),
		},
	},
	"POST /api/accounts/{id}/delete": {
		Request: ex(`{"bucket":"my-bucket","keys":["docs/a.txt","docs/b.txt"]}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":2,"failed":0}`),
		},
	},
	"POST /api/accounts/{id}/delete-prefix": {
		Request: ex(`{"bucket":"my-bucket","prefix":"docs/"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":11,"failed":0,"truncated":false}`),
		},
	},
	"POST /api/accounts/{id}/delete-prefix/async": {
		Request: ex(`{"bucket":"my-bucket","prefix":"docs/"}`),
		Responses: map[string]json.RawMessage{
			"202": ex(`{"jobId":"6f1e2d3c-4b5a-6789-abcd-ef0123456789","total":11,"truncated":false}`),
		},
	},
	"POST /api/accounts/{id}/copy-prefix": {
		Request: ex(`{"bucket":"my-bucket","prefix":"docs/","targetBucket":"archive","targetPrefix":"2026/docs/"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"copied":12,"failed":0,"total":12,"truncated":false}`),
		},
	},
	"POST /api/accounts/{id}/copy-prefix/async": {
		Request: ex(`{"bucket":"my-bucket","prefix":"docs/","targetBucket":"archive","targetPrefix":"2026/docs/"}`),
		Responses: map[string]json.RawMessage{
			"202": ex(`{"jobId":"6f1e2d3c-4b5a-6789-abcd-ef0123456789","total":12,"truncated":false}`),
		},
	},
	"POST /api/accounts/{id}/download-zip": {
		Request: ex(`{"bucket":"my-bucket","keys":["docs/a.txt","docs/b.txt"]}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`"<ZIP 字节流（application/zip，流式打包不落盘）>"`),
		},
		ContentTypes: map[string]string{"200": "application/zip"},
	},
	"POST /api/accounts/{id}/storage-class": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","storageClass":"STANDARD_IA"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"changed":"docs/a.txt","versionId":"","storageClass":"STANDARD_IA"}`),
		},
	},

	// ---- object-meta ----
	"GET /api/accounts/{id}/object-acl": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"bucket":"my-bucket","key":"docs/a.txt","owner":"owner","public":false,"grants":[{"grantee":"AllUsers","permission":"READ"}],"url":"https://s3.example.com/my-bucket/docs/a.txt"}`),
		},
	},
	"PUT /api/accounts/{id}/object-acl": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","acl":"public-read"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"acl":"public-read"}`),
		},
	},
	"GET /api/accounts/{id}/object-tags": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"tags":[{"key":"env","value":"prod"}]}`),
		},
	},
	"PUT /api/accounts/{id}/object-tags": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","tags":[{"key":"env","value":"prod"}]}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"tags":[{"key":"env","value":"prod"}]}`),
		},
	},

	// ---- multipart ----
	"POST /api/accounts/{id}/multipart/init": {
		Request: ex(`{"bucket":"my-bucket","key":"big.bin","contentType":"application/octet-stream"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"uploadId":"UPLOAD123","key":"big.bin","bucket":"my-bucket"}`),
		},
	},
	"POST /api/accounts/{id}/multipart/part": {
		Request: ex(`{"bucket":"my-bucket","key":"big.bin","uploadId":"UPLOAD123","partNumber":1,"expiresIn":3600}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"partNumber":1,"url":"https://s3.example.com/my-bucket/big.bin?partNumber=1&X-Amz-Signature=...","expiresIn":3600}`),
		},
	},
	"POST /api/accounts/{id}/multipart/complete": {
		Request: ex(`{"bucket":"my-bucket","key":"big.bin","uploadId":"UPLOAD123","parts":[{"partNumber":1,"etag":"\"e1\""}]}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"completed":"big.bin"}`),
		},
	},
	"POST /api/accounts/{id}/multipart/abort": {
		Request: ex(`{"bucket":"my-bucket","key":"big.bin","uploadId":"UPLOAD123"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"aborted":true}`),
		},
	},

	// ---- versions ----
	"GET /api/accounts/{id}/versions": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"versions":[{"key":"docs/a.txt","versionId":"v2","isLatest":true,"lastModified":"2026-09-30T05:00:00Z","size":17,"etag":"\"9c1d2f3a4b5c6d7e\"","storageClass":"STANDARD"}],"deleteMarkers":[{"key":"docs/b.txt","versionId":"dm1","isLatest":true,"lastModified":"2026-09-30T04:00:00Z"}],"isTruncated":false,"nextKeyMarker":"","nextVersionIdMarker":""}`),
		},
	},
	"DELETE /api/accounts/{id}/version": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleted":"docs/a.txt","versionId":"v1"}`),
		},
	},
	"POST /api/accounts/{id}/version/restore": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/a.txt","versionId":"v1"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"restored":"docs/a.txt","versionId":"v3"}`),
		},
	},
	"POST /api/accounts/{id}/delete-marker/restore": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/b.txt","versionId":"dm1"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"restored":"docs/b.txt","versionId":"dm1"}`),
		},
	},

	// ---- trash ----
	"GET /api/accounts/{id}/trash": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"deleteMarkers":[{"key":"docs/b.txt","versionId":"dm1","isLatest":true,"lastModified":"2026-09-30T04:00:00Z"}],"isTruncated":false,"nextKeyMarker":"","nextVersionIdMarker":""}`),
		},
	},
	"POST /api/accounts/{id}/trash/purge": {
		Request: ex(`{"bucket":"my-bucket","key":"docs/b.txt"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"purged":"docs/b.txt","deleted":3}`),
		},
	},

	// ---- migrate ----
	"POST /api/migrate": {
		Request: ex(`{"sourceAccountId":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","sourceBucket":"src-bucket","sourceKeys":["docs/a.txt"],"targetAccountId":"2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809","targetBucket":"dst-bucket","targetPrefix":"migrated/"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"migrated":1,"failed":0,"failedKeys":[]}`),
		},
	},
	"POST /api/migrate/async": {
		Request: ex(`{"sourceAccountId":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","sourceBucket":"src-bucket","sourceKeys":["docs/a.txt"],"targetAccountId":"2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809","targetBucket":"dst-bucket","targetPrefix":"migrated/"}`),
		Responses: map[string]json.RawMessage{
			"202": ex(`{"jobId":"6f1e2d3c-4b5a-6789-abcd-ef0123456789","total":1}`),
		},
	},
	"POST /api/migrate/sync": {
		Request: ex(`{"sourceAccountId":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","sourceBucket":"src-bucket","sourcePrefix":"","targetAccountId":"2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809","targetBucket":"dst-bucket","targetPrefix":"","mode":"etag"}`),
		Responses: map[string]json.RawMessage{
			"200": ex(`{"scanned":100,"skipped":60,"copied":40,"failed":0,"failedKeys":[],"truncated":false}`),
		},
	},
	"GET /api/migrate/jobs": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"jobs":[{"id":"6f1e2d3c-4b5a-6789-abcd-ef0123456789","created":"2026-09-30T05:00:00Z","finishedAt":"2026-09-30T05:05:00Z","total":100,"status":"done","progress":{"done":100,"total":100,"migrated":98,"failed":2,"status":"done"},"result":{"migrated":98,"failed":2,"failedKeys":["bad.txt"]}}]}`),
		},
	},
	"GET /api/migrate/jobs/{id}": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"jobId":"6f1e2d3c-4b5a-6789-abcd-ef0123456789","done":true,"progress":{"done":100,"total":100,"migrated":98,"failed":2,"status":"done"},"result":{"migrated":98,"failed":2,"failedKeys":["bad.txt"]}}`),
		},
	},
	"POST /api/migrate/jobs/{id}/cancel": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"jobId":"6f1e2d3c-4b5a-6789-abcd-ef0123456789","cancelled":true,"done":false}`),
		},
	},
	"GET /api/migrate/jobs/{id}/events": {
		Responses: map[string]json.RawMessage{
			"200": ex(`"data: {\"done\":1,\"total\":100,\"migrated\":1,\"failed\":0,\"status\":\"running\"}\n\n"`),
		},
		ContentTypes: map[string]string{"200": "text/event-stream"},
	},

	// ---- system ----
	"GET /api/health": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"status":"ok","version":"v1.0.0","time":"2026-09-30T05:00:00Z","store":{"ok":true}}`),
		},
	},
	"GET /api/metrics": {
		Responses: map[string]json.RawMessage{
			"200": ex(`"# HELP s3c_build_info 构建信息\ns3c_build_info{version=\"v1.0.0\"} 1\ns3c_store_up 1\n"`),
		},
		ContentTypes: map[string]string{"200": "text/plain"},
	},
	"GET /api/openapi.json": {
		Responses: map[string]json.RawMessage{
			"200": ex(`{"openapi":"3.0.3","info":{"title":"s3clinet API","version":"v1.0.0"}}`),
		},
	},
}

// applyExamples 把 apiExamples 附加到已注册的 operation 上。
// 任何 method+path / 状态码对不上都 panic：这是静态注册错误，必须在构造期立刻暴露
// （SetExamples 返回 error 而非静默忽略，见 internal/openapi）。
func applyExamples(r *openapi.Registry) {
	for key, item := range apiExamples {
		method, path, ok := strings.Cut(key, " ")
		if !ok || method == "" || !strings.HasPrefix(path, "/") {
			panic("openapi 示例 key 必须是 \"METHOD /path\"：" + key)
		}
		if err := r.SetExamples(method, path, item); err != nil {
			panic(err.Error())
		}
	}
}
