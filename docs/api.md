# REST API 参考

后端默认监听 `127.0.0.1:8080`。所有 `/api/*` 响应均为 JSON（`/api/metrics` 除外）。若设置 `S3C_TOKEN`，除 `/api/health` 与 `/api/metrics` 外，所有请求需携带 `Authorization: Bearer <token>`（机器可读契约中的同一事实见下文「API 契约」，由顶层 `security` 与豁免端点的 `security: []` 表达）。

> **最小权限（`S3C_TOKEN_SCOPES`，ROADMAP §三 #13）**：可为单个 token 声明 `readonly` / `prefixes` / `accounts` / `expiresAt`（字段缺省即不限制）。`readonly` 下仅放行 GET / HEAD，其余方法（含能铸造写 URL 的预签名 `POST`）一律 `403`；`prefixes`（`"<bucket>"` 整桶或 `"<bucket>/<key前缀>"`）下请求涉及的桶/键（query 或 JSON body）与列表 `prefix` 越界返回 `403`，桶级操作需该桶的整桶授权；`accounts` 下路径 `{id}` 越界返回 `403`；`expiresAt` 过期返回 `401`。**未在 `S3C_TOKEN_SCOPES` 中登记的 token 保持全权**（向后兼容）。越权写审计事件 `auth.scope_denied`（`reason` = `readonly` / `prefix` / `account` / `unparsable_body`），过期写 `auth.denied`（`reason=token_expired`）；审计与响应均**不含 token 明文**。配置格式与 fail-closed 规则见 [`CONFIGURATION.md`](CONFIGURATION.md)。

> **`/api/metrics` 有意不受 `S3C_TOKEN` 保护**：即使配置了 token，只要设置 `S3C_EXPOSE_METRICS=1`，`GET /api/metrics` 无需 `Authorization` 头即返回 200（`withAuth` 只豁免 `/api/health` 与 `/api/metrics`，见 `middleware.go`）。这是为了让内网 Prometheus 直接 scrape 而无需分发 token；代价是该端点一旦暴露即**匿名可读**（含版本、存储可达性、S3 上游调用统计等运行信息）。因此**不要**把开启 metrics 的实例直接暴露到公网，应仅在内网 / 反向代理鉴权之后放行。
所有响应带 `X-Request-ID`（客户端可传入，否则服务端生成）；访问日志字段 `req` 与之对应。

错误格式：`{"error": "..."}`  
通用码：`400`（请求错误）、`401`（未鉴权 / token 过期）、`403`（token 作用域越权 / Origin 不允许）、`404`（未找到）、`500`（服务端错误）。  
S3 错误码与用户消息对照见 [`docs/errors.md`](./errors.md)。

## 健康检查

```
GET /api/health
```
```json
200 {"status":"ok","version":"v1.0.0","time":"...","store":{"ok":true}}
```
`version` 为服务端版本号（构建时经 ldflags 注入），可用来核对前后端版本是否匹配。store 探测失败时返回 `503` + `"status":"error"`（不做降级；不健康即失败）。

## 指标

```
GET /api/metrics
```
Prometheus 文本格式。**默认返回 404**（不暴露端点），仅当设置 `S3C_EXPOSE_METRICS=1` 时返回 200；含 HTTP 计数与延迟直方图、uptime、goroutine、内存、`s3c_build_info`，以及：
- `s3c_store_up`：账号存储可达性（1 / 0）。store 掉线时 `/api/health` 返回 503 且本指标为 0——硬失败不降级（ADR-002），建议据此告警。
- `s3c_store_write_failures_total`：账号库写入失败次数（落盘 / SQL 写入出错，业务拒绝不计数）——`json` / `encrypted` 驱动唯一的主动存储故障信号。
- `s3c_volume_size_bytes` / `s3c_volume_free_bytes`：`S3C_DATA_DIR` 所在文件系统总容量与可用字节；取不到时**不输出该序列**（平台不支持 / statfs 失败）。
- `s3c_jobs_active`：在册（未终结）异步任务数，上限 256（与 `JobRegistry` 同口径）。
- `s3c_last_shutdown_duration_seconds`：上一次优雅关停耗时（启动时从 `data/shutdown.json` 载入；0 = 尚无记录）。
- `s3c_http_request_duration_seconds`：HTTP 请求延迟直方图（含流式端点）；`+Inf` 桶恒等于 `s3c_http_requests_total`。
- `s3c_ssrf_deny_private`：SSRF 生效策略（1 = 拒绝私网 / 回环 S3 端点，0 = 默认放行）。用于核对 `S3C_SSRF_DENY_PRIVATE` 是否真的生效（ADR-003）。
- `s3c_stream_interrupted_total`：流式传输在完成前中断的次数（上游读失败 / 写超时 / 客户端断开）。
- `s3c_s3_calls_total` / `s3c_s3_call_errors_total{code=...}` / `s3c_s3_call_duration_seconds`：S3 上游调用总数、按错误码分类的失败数、耗时直方图（非 API 错误归入 `transport`/`canceled`/`timeout`）。
- `s3c_s3_stream_bytes_total`：经本服务从 S3 流式读出的字节数。
- `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total` / `s3c_zip_failed_total`：ZIP 打包部分失败次数、失败对象累计数、整体失败次数。

> 全量逐字清单（含类型与建议告警用法）见 [`OPERATIONS.md`](OPERATIONS.md) §3.2——改指标时两处同改。

## 账号

### 列出
```
GET /api/accounts
```
```json
200 {"accounts":[{"id":"...","name":"...","endpoint":"...","accessKey":"...","secretSet":true,"bucket":"...","pathStyle":true,"useSSL":false}]}
```
响应为 `AccountView`：**不回传 `secretKey`**，用 `secretSet`（boolean）表示是否已设置密钥，客户端无法从响应中区分占位与真实密钥。

### 创建
```
POST /api/accounts
```
```json
{"name":"minio","endpoint":"http://localhost:9000","publicEndpoint":"https://s3.example.com","region":"us-east-1","accessKey":"ak","secretKey":"sk","bucket":"b","pathStyle":true,"useSSL":false}
```
必填：`name`、`endpoint`、`accessKey`、`secretKey`。`pathStyle` 用于 MinIO/OSS 等第三方；`useSSL` 启用 TLS；`publicEndpoint` 是浏览器直传/预签名使用的对外端点（留空则用 `endpoint`）。

### 获取
```
GET /api/accounts/{id}
```

### 更新
```
PUT /api/accounts/{id}
```
```json
{"name":"minio","endpoint":"http://localhost:9000","publicEndpoint":"https://s3.example.com","region":"us-east-1","accessKey":"ak","secretKey":"sk","bucket":"b","pathStyle":true,"useSSL":false}
```
必填：`name`、`endpoint`、`accessKey`（`secretKey` 可省略，传空或不传则保留原值）；其余字段同创建。
响应为 `AccountView`，`secretSet` 反映更新后是否仍有密钥。

### 删除
```
DELETE /api/accounts/{id}
```

### 预览桶（不落库）
```
POST /api/accounts/preview-buckets
```
```json
{"name":"N(可选)","endpoint":"http://minio:9000","publicEndpoint":"https://s3.example.com(可选)","region":"us-east-1","accessKey":"ak","secretKey":"sk","bucket":"B(可选)","pathStyle":true,"useSSL":false}
```
用表单临时凭据只读列出桶（不保存账号），用于新建账号时选择默认桶；仍受 endpoint 校验与拨号期 SSRF 防护。
```json
200 {"buckets":[{"name":"b1","creationDate":"..."}]}
```
缺 endpoint/accessKey/secretKey 返回 400；上游 `ListBuckets` 失败返回 500。

### 连通性测试
```
POST /api/accounts/{id}/test
POST /api/accounts/{id}/test?bucket=B
```
有默认桶（或 `?bucket=`）时对该桶 `HeadBucket`；否则用 `ListBuckets` 探测凭证与端点。
```json
200 {"ok":true,"bucket":"b"}
```
`ok:false` 时附 `error` 字段（HTTP 仍 200，便于前端展示）。

### 列出桶
```
GET /api/accounts/{id}/buckets
```
```json
200 {"buckets":[{"name":"b","creationDate":"..."}]}
```

### 创建桶
```
POST /api/accounts/{id}/bucket
```
```json
{"name":"new-bucket","region":"cn-north-1(可选)","acl":"private|public-read|public-read-write(可选)"}
```
名称须 3-63 位小写字母/数字/连字符/点；`region` 非 `us-east-1` 时附带 `LocationConstraint`（OSS/COS/TOS 需要）。
```json
200 {"created":"new-bucket","region":"cn-north-1","acl":"private"}
```

### 删除桶
```
DELETE /api/accounts/{id}/bucket?name=new-bucket
```
桶内须为空；非空时返回 `409 {"error":"bucket not empty, delete all objects first"}`。
```json
200 {"deleted":"new-bucket"}
```

## 对象

### 列出对象
```
GET /api/accounts/{id}/objects?bucket=B&prefix=P&delimiter=/&maxKeys=1000&continuationToken=CT&startAfter=K
```
- `bucket` 缺省用账号默认桶；`maxKeys` 范围 1–1000（默认 1000）。
- `delimiter=/` 时用 `commonPrefixes` 返回目录。
- `continuationToken` 用于分页。
- `startAfter` 可选，按 key 字典序从该 key **之后**开始列举（ListObjectsV2 `start-after`）；与 `continuationToken` 互斥使用，用于「从某位置继续」的场景。
```json
200 {
  "objects":[{"key":"a.txt","size":17,"lastModified":"...","etag":"\"...\"","storageClass":"STANDARD","isDir":false}],
  "commonPrefixes":["dir/"],
  "isTruncated":false,
  "nextToken":""
}
```
`objects` 各项含 `storageClass`（存储类型），可用于列表展示。

### 存储分析与成本洞察
```
GET /api/accounts/{id}/storage-report?bucket=B&prefix=P(可选)
```
列举桶（可限定 `prefix`）并聚合：按存储类与顶层前缀的用量、按 USD/GiB/月的成本估算，以及低频（30–89 天未修改）与归档（≥90 天）建议。最多列举 100 页 / 10 万个对象，超出或多页游标异常时置 `truncated` 为 `true`（报告仍可用，只是未覆盖全部对象）。未知存储类按 STANDARD 单价估算；`prefixGroupCount` 为 `byPrefix` 分组数。
```json
200 {"bucket":"my-bucket","prefix":"","objectCount":6,"totalSize":32212254720,"truncated":false,"monthlyCost":0.53,"prefixGroupCount":3,"byStorageClass":[{"storageClass":"STANDARD","count":3,"size":17179869184,"monthlyCost":0.368}],"byPrefix":[{"prefix":"photos/","count":3,"size":17179869184}],"recommendations":[{"kind":"infrequent","fromStorageClass":"STANDARD","toStorageClass":"STANDARD_IA","count":1,"size":5368709120,"estimatedMonthlySaving":0.0525}]}
```

### 对象详情
```
GET /api/accounts/{id}/head?bucket=B&key=K&versionId=V(可选)
```
`bucket` 缺省用账号默认桶；对象不存在返回 404。`versionId` 可选，指定读取某历史版本的详情。
```json
200 {"key":"dir/a.txt","size":17,"lastModified":"...","etag":"\"...\"","contentType":"text/plain","storageClass":"STANDARD_IA","metadata":{"owner":"alice"},"checksums":{"crc64nvme":"N4bktbEKNg8=","crc32c":"","sha256":"","sha1":"","type":"FULL_OBJECT"}}
```
`checksums` 为端到端校验和（服务端存储值）：对象无校验和或厂商不支持时为 `null`；`type` 为 `FULL_OBJECT`（全对象）或 `COMPOSITE_*`（分段合成，不可全对象比对）。可用 `POST /api/accounts/{id}/verify-checksum` 本地重算比对。

### 校验对象内容（端到端校验和）
```
POST /api/accounts/{id}/verify-checksum
```
```json
{"bucket":"B(可选)","key":"dir/a.txt","versionId":"V(可选)"}
```
先 `Head` 选定算法阶梯（`crc64nvme` → `crc32c` → `sha256` → `sha1` → `etag-md5`），再流式拉取全对象本地重算并与存储端比对。无校验和可用时 `method="none"`、`match=false`（如实降级，不是错误）。大对象验证会占用带宽与 CPU，响应在流读完后返回。
```json
200 {"bucket":"B","key":"dir/a.txt","versionId":"","method":"crc64nvme","local":"N4bktbEKNg8=","remote":"N4bktbEKNg8=","match":true}
```

### 新建文件夹
```
POST /api/accounts/{id}/mkdir
```
```json
{"bucket":"B(可选)","key":"images","ifMatch":"\"e1\"(可选)","ifNoneMatch":"*(可选)"}
```
S3 无真实目录：服务端 PUT 空对象，`key` 自动补全为以 `/` 结尾（`images/`）。条件写（可选）：`ifNoneMatch` 只接受 `*`（目标不存在才创建，防并发覆盖）；`ifMatch` 为 ETag 字面量（仅当目标当前 ETag 匹配才写入）；条件不满足返回 412。
```json
200 {"created":"images/","bucket":"B"}
```

### 重命名 / 移动
```
POST /api/accounts/{id}/rename
```
```json
{"bucket":"B(可选)","key":"a.txt","newKey":"dir/b.txt","newBucket":"B2(可选)"}
```
先 `CopyObject` 到新 key，成功后才删除源（复制失败不丢数据）；`newBucket` 缺省同桶；同桶内 `newKey` 与 `key` 相同拒绝，跨桶同名移动允许。
```json
200 {"renamed":"dir/b.txt"}
```

### 复制对象（单文件，跨桶）
```
POST /api/accounts/{id}/copy-object
```
```json
{"bucket":"B(可选)","key":"a.txt","newKey":"archive/a.txt","newBucket":"B2(可选)","ifMatch":"\"e1\"(可选)","ifNoneMatch":"*(可选)","checksumAlgorithm":"CRC64NVME|SHA256|CRC32C|SHA1(可选)"}
```
复制单个对象到目标桶/目标 key，**不删除源**（区别于 `rename`）；`newBucket` 缺省同桶；同桶内 `newKey` 与 `key` 相同则拒绝。条件写（可选）作用于**目标**对象：`ifNoneMatch="*"` 仅当目标不存在才复制、`ifMatch` 为目标 ETag；条件不满足返回 **412**（`PreconditionFailed`），并发冲突返回 **409**（`ConditionalRequestConflict`）。`checksumAlgorithm` 非空时服务端计算并**存储全对象校验和**（复制后即可被 `verify-checksum` 以对应算法端到端比对）；缺省走服务端默认（不改变历史行为）。
```json
200 {"copied":"archive/a.txt","bucket":"B2"}
```

### 批量复制 / 移动（所选文件）
```
POST /api/accounts/{id}/copy-objects
```
```json
{"bucket":"B(可选)","targetBucket":"B2(可选)","targetPrefix":"archive/","keys":["dir/a.txt","dir/b.txt"],"deleteSource":false}
```
把多个文件复制到目标桶 + 前缀（`targetPrefix + 文件名`，保留文件名）；`targetPrefix` 留空 = 目标桶根目录；`deleteSource=true` 时复制成功后再删除源（移动）。逐个处理、失败不中断。`failedKeys` **上限 200 条**（`failed` 计数不受裁剪影响）。
```json
200 {"copied":1,"failed":1,"lastError":"(失败时才有)","failedKeys":["dir/b.txt"]}
```

```
POST /api/accounts/{id}/copy-objects/async
```
请求体与 `POST /api/accounts/{id}/copy-objects` 相同。立即返回 `jobId`，进度通过既有迁移任务接口查询（`progress.migrated` = 已复制/移动成功数）。
```json
202 {"jobId":"uuid","total":2}
```

### 删除文件夹（递归）
```
POST /api/accounts/{id}/delete-prefix
```
```json
{"bucket":"B(可选)","prefix":"dir/"}
```
循环 `ListObjectsV2` + 批量 `DeleteObjects` 删除前缀下全部对象；`prefix` 必填（空前缀拒绝，防误删全桶）；上限 10 万个对象。`deleted` 只计成功数：S3 对逐 key 失败仍返回 200，被桶策略/保留期拒绝的对象计入 `failed` 并在 `lastError` 给出原因（截断时 `truncated=true`）。
```json
200 {"deleted":11,"failed":1,"truncated":false,"lastError":"access denied"}
```

```
POST /api/accounts/{id}/delete-prefix/async
```
请求体与 `delete-prefix` 相同。先列举再异步批量删除；`progress.migrated` 表示已删除数，`progress.failed` 表示被逐 key 拒绝的数量（终态 `result.failedKeys` 列出前 200 个失败 key）。
```json
202 {"jobId":"uuid","total":12,"truncated":false}
```

### 复制文件夹（递归）
```
POST /api/accounts/{id}/copy-prefix
```
```json
{"bucket":"B","prefix":"dir/","targetBucket":"B2(可选)","targetPrefix":"archive/"}
```
同账号内逐 key `CopyObject`，`targetPrefix` 直接前置到相对 key；同桶目标前缀与源重叠时拒绝（防无限复制）。
```json
200 {"copied":12,"failed":0,"total":12,"lastError":"(失败时才有)"}
```

### 设置对象 HTTP 头
```
POST /api/accounts/{id}/set-headers
```
```json
{"bucket":"B(可选)","key":"a.txt","contentType":"text/markdown","metadata":{"owner":"alice"}}
```
`CopyObject` 复制到自己并 `MetadataDirective: REPLACE`；`contentType` 留空不修改，`metadata` 整体覆盖（传空对象则清空）。
```json
200 {"updated":"a.txt"}
```

### 对象权限（ACL）
```
GET /api/accounts/{id}/object-acl?bucket=B(可选)&key=K
```
返回所有者、是否公有、授权列表与公开链接（`url` 按账号 `publicEndpoint`/path-style/useSSL 构造）。
```json
200 {"bucket":"B","key":"a.txt","owner":"owner","public":true,"grants":[{"grantee":"所有用户 (AllUsers)","permission":"READ"}],"url":"http://host/B/a.txt"}
```
```
PUT /api/accounts/{id}/object-acl
```
```json
{"bucket":"B(可选)","key":"a.txt","acl":"public-read"}
```
`acl` 取值：`private`（默认）、`public-read`、`public-read-write`、`authenticated-read`、`aws-exec-read`。
```json
200 {"acl":"public-read"}
```

### 对象标签（Tagging）
```
GET /api/accounts/{id}/object-tags?bucket=B(可选)&key=K
```
无标签时（S3 返回 `NoSuchTagSet` 等）视为空列表。
```json
200 {"tags":[{"key":"env","value":"prod"}]}
```
```
PUT /api/accounts/{id}/object-tags
```
```json
{"bucket":"B(可选)","key":"K","tags":[{"key":"env","value":"prod"}]}
```
覆盖写入；`tags` 传空数组则删除全部标签；`key` 非空且不可重复。
```json
200 {"tags":[{"key":"env","value":"prod"}]}
```

### 对象保留期（Object Lock 保留）
```
GET /api/accounts/{id}/object-retention?bucket=B(可选)&key=K&versionId=V(可选)
```
无保留期（或桶未启用 Object Lock）时 `configured=false`。
```json
200 {"bucket":"B","key":"a.txt","versionId":"","configured":true,"mode":"COMPLIANCE","retainUntilDate":"2030-01-02T03:04:05Z"}
```
```
PUT /api/accounts/{id}/object-retention
```
```json
{"bucket":"B(可选)","key":"a.txt","versionId":"V(可选)","mode":"GOVERNANCE|COMPLIANCE","retainUntilDate":"2031-02-03T04:05:06Z"}
```
`retainUntilDate` 为 RFC3339 且必须是未来时刻（否则 400）。错误映射：输入非法 / 保留期短于当前 → **400**；GOVERNANCE 保留期内拒绝（含权限不足）→ **403**；COMPLIANCE 锁定 → **409**（`ObjectLocked`）。
```json
200 {"bucket":"B","key":"a.txt","versionId":"","configured":true,"mode":"GOVERNANCE","retainUntilDate":"2031-02-03T04:05:06Z"}
```

### 法定保留（Legal Hold）
```
GET /api/accounts/{id}/object-legal-hold?bucket=B(可选)&key=K&versionId=V(可选)
```
未设置时 `status="OFF"`。
```json
200 {"bucket":"B","key":"a.txt","versionId":"","status":"ON"}
```
```
PUT /api/accounts/{id}/object-legal-hold
```
```json
{"bucket":"B(可选)","key":"a.txt","versionId":"V(可选)","status":"ON|OFF"}
```
`status` 只接受 `ON` / `OFF`；拒绝（含权限不足）→ 403，对象被锁定 → 409。
```json
200 {"bucket":"B","key":"a.txt","versionId":"","status":"ON"}
```

### 桶属性（区域 / 创建时间 / 版本控制）
```
GET /api/accounts/{id}/bucket-info?bucket=B(可选)
```
```json
200 {"bucket":"B","region":"cn-north-1","createdAt":"2026-09-01T00:00:00Z","versioning":"Enabled"}
```
`region` 为 `GetBucketLocation` 结果（空视为 `us-east-1`）；`versioning` 取值 `""`（未配置）/ `Enabled` / `Suspended`。

### 桶版本控制开关
```
PUT /api/accounts/{id}/bucket-versioning
```
```json
{"bucket":"B(可选)","status":"Enabled"}
```
`status` 取值 `Enabled` 或 `Suspended`。
```json
200 {"versioning":"Enabled"}
```

### 桶服务端加密（SSE）
```
GET /api/accounts/{id}/bucket/encryption?bucket=B
```
```json
200 {"bucket":"B","configured":true,"algorithm":"AES256","kmsKeyId":"","bucketKeyEnabled":true}
```
未配置时返回 `configured:false`。

```
PUT /api/accounts/{id}/bucket/encryption
```
```json
{"bucket":"B(可选)","algorithm":"AES256","kmsKeyId":"(可选)","bucketKeyEnabled":true}
```
`algorithm` 取值 `AES256` / `aws:kms` / `aws:kms:dsse`。
```
DELETE /api/accounts/{id}/bucket/encryption?bucket=B
200 {"deleted":"B"}
```

### 桶 CORS 规则
```
GET /api/accounts/{id}/bucket/cors?bucket=B
200 {"bucket":"B","rules":[{"id":"r1","allowedMethods":["GET"],"allowedOrigins":["*"],"allowedHeaders":["*"],"exposeHeaders":["ETag"],"maxAgeSeconds":3600}]}
```
```
PUT /api/accounts/{id}/bucket/cors
{"bucket":"B(可选)","rules":[{...}]}
```
`rules` 传空数组时删除全部规则。
```
DELETE /api/accounts/{id}/bucket/cors?bucket=B
200 {"deleted":"B"}
```

### 桶静态网站托管
```
GET /api/accounts/{id}/bucket/website?bucket=B
200 {"bucket":"B","configured":true,"indexDocument":"index.html","errorDocument":"error.html","redirectAllRequestsTo":""}
```
```
PUT /api/accounts/{id}/bucket/website
{"bucket":"B(可选)","indexDocument":"index.html","errorDocument":"error.html","redirectAllRequestsTo":""}
```
`indexDocument` 或 `redirectAllRequestsTo` 至少填其一。
```
DELETE /api/accounts/{id}/bucket/website?bucket=B
200 {"deleted":"B"}
```

### 桶策略
```
GET /api/accounts/{id}/bucket/policy?bucket=B
200 {"bucket":"B","configured":true,"policy":"{...}"}
```
```
PUT /api/accounts/{id}/bucket/policy
{"bucket":"B(可选)","policy":"{\"Version\":\"2012-10-17\",...}"}
```
`policy` 必须是合法 JSON；传空字符串时删除桶策略。
```
DELETE /api/accounts/{id}/bucket/policy?bucket=B
200 {"deleted":"B"}
```

### 桶标签
```
GET /api/accounts/{id}/bucket/tags?bucket=B
200 {"bucket":"B","tags":[{"key":"env","value":"prod"}]}
```
```
PUT /api/accounts/{id}/bucket/tags
{"bucket":"B(可选)","tags":[{"key":"env","value":"prod"}]}
```
`tags` 传空数组时删除全部标签。
```
DELETE /api/accounts/{id}/bucket/tags?bucket=B
200 {"deleted":"B"}
```

### 桶 Object Lock 配置（WORM 默认保留）
```
GET /api/accounts/{id}/bucket/object-lock?bucket=B(可选)
```
桶未在创建时启用 Object Lock（或厂商不支持）时 `enabled=false`，不是错误。
```json
200 {"bucket":"B","enabled":true,"defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":30,"defaultRetentionYears":0}
```
```
PUT /api/accounts/{id}/bucket/object-lock
```
```json
{"bucket":"B(可选)","defaultRetentionMode":"GOVERNANCE|COMPLIANCE","defaultRetentionDays":30,"defaultRetentionYears":0}
```
`defaultRetentionMode` 必填；`defaultRetentionDays` 与 `defaultRetentionYears` **二选一**且 ≥1。桶必须**在创建时**启用 Object Lock：既有桶上设置返回 **409**（`InvalidBucketState`）；厂商未实现返回 **501**（`NotImplemented`）。
```json
200 {"bucket":"B","enabled":true,"defaultRetentionMode":"GOVERNANCE","defaultRetentionDays":30,"defaultRetentionYears":0}
```

### 对象版本列表（ListObjectVersions）
```
GET /api/accounts/{id}/versions?bucket=B(可选)&prefix=P&keyMarker=K&versionIdMarker=V
```
```json
200 {
  "versions":[{"key":"a.txt","versionId":"v2","isLatest":true,"lastModified":"2026-09-01T01:00:00Z","size":2,"etag":"\"e2\"","storageClass":"STANDARD"}],
  "deleteMarkers":[{"key":"a.txt","versionId":"dv1","isLatest":true,"lastModified":"2026-09-01T00:00:00Z"}],
  "isTruncated":false,"nextKeyMarker":"","nextVersionIdMarker":""
}
```
需桶版本控制开启后才保留多版本；`isLatest` 标记当前版本，`deleteMarkers` 为删除标记，`storageClass` 为该版本的存储类型。

### 删除指定版本
```
DELETE /api/accounts/{id}/version?bucket=B(可选)&key=K&versionId=V
```
`versionId` 必填；可删除普通版本或删除标记（版本控制下删除对象不传 versionId 只会新增删除标记）。
```json
200 {"deleted":"K","versionId":"V"}
```

### 版本回滚（恢复某版本为当前）
```
POST /api/accounts/{id}/version/restore
```
```json
{"bucket":"B(可选)","key":"K","versionId":"V"}
```
把该版本复制回当前 key（`CopyObject` 带 `?versionId=`）；版本控制下会写出一条新的当前版本，`versionId` 为新版本号。
```json
200 {"restored":"K","versionId":"<新版本号>"}
```

### 一键还原已删除对象（恢复删除标记）
```
POST /api/accounts/{id}/delete-marker/restore
```
```json
{"bucket":"B(可选)","key":"K","versionId":"V"}
```
`versionId` 必须是删除标记的版本号。S3 语义：删除标记是一个无数据版本，删除（`DeleteObject` 带 `versionId`）该标记即完成「撤销删除」，对象回到被删除前的状态。
```json
200 {"restored":"K","versionId":"V"}
```

### 切换对象存储类型（StorageClass）
```
POST /api/accounts/{id}/storage-class
```
```json
{"bucket":"B(可选)","key":"K","versionId":"V(可选)","storageClass":"STANDARD_IA"}
```
通过 `CopyObject` 副本写入自身并携带新 `x-amz-storage-class`（`versionId` 为空切换当前对象，否则切换指定版本）。版本控制桶下写出一条新版本；`storageClass` 支持 `STANDARD` / `STANDARD_IA` / `ONEZONE_IA` / `INTELLIGENT_TIERING` / `GLACIER` / `GLACIER_IR` / `DEEP_ARCHIVE` / `REDUCED_REDUNDANCY` / `EXPRESS_ONEZONE`。
```json
200 {"changed":"K","versionId":"<新版本号>","storageClass":"STANDARD_IA"}
```

### 回收站（列出删除标记）
```
GET /api/accounts/{id}/trash?bucket=B(可选)&prefix=P&keyMarker=K&versionIdMarker=V&maxKeys=1000
```
返回桶内删除标记（已删除对象），带分页游标（ListObjectVersions 过滤为删除标记）。逐页加载时若某页无删除标记，调用方继续用游标向后翻页即可。
```json
200 {
  "deleteMarkers":[{"key":"a.txt","versionId":"dv1","isLatest":true,"lastModified":"2026-09-01T00:00:00Z"}],
  "isTruncated":false,"nextKeyMarker":"","nextVersionIdMarker":""
}
```

### 彻底清除（永久删除对象）
```
POST /api/accounts/{id}/trash/purge
```
```json
{"bucket":"B(可选)","key":"K"}
```
删除该 key 的全部版本与删除标记（不可再还原）；非版本控制桶兜底删除当前对象。
```json
200 {"purged":"K","deleted":3}
```

### 生命周期规则（前缀过期删除）
```
GET /api/accounts/{id}/lifecycle?bucket=B
```
```json
200 {"rules":[{"id":"r1","prefix":"logs/","days":30}]}
```
未配置规则返回空列表。基于 S3 兼容生命周期 API（MinIO/AWS 支持，部分厂商兼容性有限）。

```
PUT /api/accounts/{id}/lifecycle
```
```json
{"bucket":"B","rules":[{"id":"r1","prefix":"logs/","days":30}]}
```
`rules` 传空数组时删除全部规则（`DeleteBucketLifecycle`，空 PUT 会被多数实现拒绝）。每条规则需 `id` 唯一且 `days >= 1`。
```json
200 {"updated":1}
```

### 批量下载（ZIP 打包）
```
POST /api/accounts/{id}/download-zip
```
```json
{"bucket":"B(可选)","keys":["a.txt","dir/b.txt"]}
```
服务端流式打包 ZIP（不落盘），响应 `Content-Type: application/zip`；获取失败的对象写入包内 `_下载失败清单.txt`，同时在服务端记录 Warn 日志并计入 `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total`（失败率可观测）。

### 安全代理（下载 / 预览）
```
GET /api/accounts/{id}/proxy?bucket=B&key=K&mode=download|inline|text&maxBytes=N
```
统一走服务端转发，避免签名 URL 暴露与恶意内容渲染：

- `mode=download`（默认）：强制 `Content-Disposition: attachment` 流式转发（浏览器直接保存，**内容不进渲染管道**），支持 `Range` 请求头透传。
- `mode=inline`：透传源 `Content-Type` 流式转发（图片 / PDF / 媒体预览），支持 `Range`。
- `mode=text`：读取前 `maxBytes` 字节（默认 1MB，上限 2MB），**强制** `text/plain; charset=utf-8` + `X-Content-Type-Options: nosniff`，超限时响应头 `X-Preview-Truncated: 1`（杜绝 HTML/JS 注入）。

对象不存在返回 404；文件名经清洗（去路径分隔符/引号）后写入 `Content-Disposition`，防止头注入。

### 生成签名
```
POST /api/accounts/{id}/presign
```
```json
{"method":"get|put|post","key":"dir/a.txt","bucket":"B(可选)","versionId":"V(可选,仅get)","expiresIn":3600,"ifMatch":"\"e1\"(可选,仅put)","ifNoneMatch":"*(可选,仅put)"}
```
- `expiresIn` 单位秒；缺省（或 ≤0）时**默认 1 小时**，超过 **24 小时**会被钳到 24 小时（S3 协议上限为 7 天，控制台场景收紧到 24h；见 `s3wrap/presign.go` 与 `objects.go` 的钳制）。
- `get` / `put` 返回 `{method,url,expiresIn,...}`；`post` 额外返回 `{url,fields}`（multipart 表单字段）。
- `get` 可传 `versionId` 生成指向指定历史版本的签名 GET（用于「版本比较/详情」拉取某个版本内容）。
- 条件写（可选，仅 `put`）：`ifNoneMatch` 只接受 `*`、`ifMatch` 为 ETag 字面量；条件不在 URL 里——**浏览器 PUT 时必须携带响应 `headers` 回显的请求头**（S3 服务端求值，且条件头参与签名），条件不满足返回 412。`get`/`post` 携带条件字段直接 400。浏览器直传场景要求桶 CORS 的 **AllowedHeaders 放行 `If-Match` / `If-None-Match`**（条件头必须原样回传，否则预检失败或签名校验不过）。
```json
get: {"method":"get","bucket":"B","key":"k","url":"https://...X-Amz-Signature=...","expiresIn":3600}
post: {"method":"post","bucket":"B","key":"k","url":"https://...","fields":{"X-Amz-Signature":"...","key":"k"},...}
put: {"method":"put","bucket":"B","key":"k","url":"https://...","expiresIn":3600,"headers":{"If-None-Match":"*"}}
```

### 分段上传（大文件直传）
用于大文件（前端 `≥100MB` 自动使用；<100MB 走单 PUT）。四步：
```
POST /api/accounts/{id}/multipart/init
{"bucket":"B(可选)","key":"big.bin","contentType":"application/octet-stream(可选)"}
200 {"uploadId":"UPLOAD123","key":"big.bin","bucket":"B"}
```
```
POST /api/accounts/{id}/multipart/part
{"bucket":"B(可选)","key":"big.bin","uploadId":"UPLOAD123","partNumber":1,"expiresIn":3600}
200 {"partNumber":1,"url":"https://...X-Amz-Signature=...","expiresIn":3600}
```
浏览器 PUT 到 `url` 后需读取响应头 `ETag`（要求 Bucket CORS 暴露 `ETag`）。
```
POST /api/accounts/{id}/multipart/complete
{"bucket":"B(可选)","key":"big.bin","uploadId":"UPLOAD123","parts":[{"partNumber":1,"etag":"\"e1\""}]}
200 {"completed":"big.bin"}
```
```
POST /api/accounts/{id}/multipart/abort
{"bucket":"B(可选)","key":"big.bin","uploadId":"UPLOAD123"}
200 {"aborted":true}
```
`partNumber` 范围 1–10000；`parts` 需按段号对应各自 `etag`；失败时应调用 `abort` 清理。

续传前对齐服务端真实清单（只读；刷新 / 断电 / 重选同一文件后，前端据此跳过已上传段、只补缺段）：
```
GET /api/accounts/{id}/multipart/parts?bucket=B&key=big.bin&uploadId=UPLOAD123
```
```json
200 {"parts":[{"partNumber":1,"etag":"e1","size":10485760,"lastModified":"2026-10-08T05:00:00Z"}]}
```
`bucket` 可选（缺省回退账号默认桶）；`key` / `uploadId` 必填（缺失 400）。`uploadId` 已失效或清单为空时前端重新 `init`，不以上传方本地记录为准。

### 删除对象（批量）
```
POST /api/accounts/{id}/delete
```
```json
{"bucket":"B(可选)","keys":["a.txt","b.txt"]}
```
SDK 单次最多 1000，服务端自动分批。`deleted` 只计成功数：S3 对逐 key 失败仍返回 200，被桶策略/保留期拒绝的对象计入 `failed` 并在 `lastError` 给出原因。
```json
200 {"deleted":1,"failed":1,"lastError":"access denied"}
```

### 跨账号迁移
```
POST /api/migrate
```
```json
{
  "sourceAccountId":"a","sourceBucket":"B","sourceKeys":["x.txt"],
  "targetAccountId":"b","targetBucket":"B2","targetPrefix":"migrated/"
}
```
- 源、目标 endpoint 一致时用 `CopyObject`（服务端复制）；否则 `GetObject`→`PutObject` 流式转发（保留 Content-Type/元数据）。
- 逐个对象迁移，任一失败继续其余；失败对象 key 通过 `failedKeys` 返回（便于前端展示失败清单，**上限 200 条**——`failed` 计数不受裁剪影响）。
```json
200 {"migrated":1,"failed":1,"lastError":"(失败时才有)","failedKeys":["bad.txt"]}
```

#### 异步迁移（SSE 进度）
```
POST /api/migrate/async
```
请求体与 `POST /api/migrate` 相同。
```json
202 {"jobId":"uuid","total":100}
```

```
GET /api/migrate/jobs
```
返回异步任务清单（按创建时间倒序，最新在前），含进程重启后恢复的任务。
`status` 取值：`running` | `done` | `cancelled` | `interrupted`。
`interrupted` 表示服务重启导致任务中断，需人工对账——**移动（`deleteSource:true`）任务可能已复制但源未删除**。
`finishedAt` 为**完成时刻**（Reap TTL 从此刻起算 30 分钟，而非 `created`——跑超 30 分钟的任务不会在完成后 ≤5 分钟即被清掉）；运行中任务与旧版落盘记录省略该字段。
```json
200 {"jobs":[{"id":"uuid","created":"2026-09-16T10:00:00Z","total":100,"status":"interrupted","progress":{"done":42,"total":100,"migrated":42,"failed":0,"status":"interrupted"},"result":{"migrated":42,"failed":0,"failedKeys":["..."]}}]}
```

```
GET /api/migrate/jobs/{id}
```
```json
200 {"jobId":"uuid","done":true,"progress":{"done":100,"total":100,"migrated":98,"failed":2,"status":"done"},"result":{"migrated":98,"failed":2,"failedKeys":["..."]}}
```

```
POST /api/accounts/{id}/copy-prefix/async
```
请求体与 `copy-prefix` 相同。立即返回 `jobId`，进度通过既有迁移任务接口查询：
`GET /api/migrate/jobs/{id}` / `.../events` / `POST .../cancel`（`progress.migrated` 表示已复制数）。

```
GET /api/migrate/jobs/{id}/events
```
`Content-Type: text/event-stream`。每行 `data: {"done":1,"total":100,...}`，结束时 `status:"done"` 或 `status:"cancelled"`；另有 `event: ping` 心跳。

```
POST /api/migrate/jobs/{id}/cancel
```
取消运行中的异步迁移/复制任务（已完成则 `cancelled:false`）。
```json
200 {"jobId":"uuid","cancelled":true}
```

### 增量同步（按 ETag / size+mtime 比对，仅复制差异对象）

```
POST /api/migrate/sync
```

请求体：
```json
{
  "sourceAccountId": "uuid",
  "sourceBucket": "src-bucket",
  "sourcePrefix": "",        // 可选；空=整桶
  "targetAccountId": "uuid",
  "targetBucket": "dst-bucket",
  "targetPrefix": "",        // 可选
  "mode": "etag"             // etag（默认）| size_mtime | always
}
```

行为：
- 列举源 prefix 全部对象（递归，硬上限 100k 防卡死；命中上限时响应 `truncated=true`）；
- 列举目标 prefix 全部元数据；
- **目标 key 映射**：`targetPrefix + (源 key 去掉 sourcePrefix)`。`sourcePrefix` 必须落在 `/` 段边界上
  才算命中（`p` 不会命中 `prefix/x.txt`）；`sourcePrefix` 留空 = 整桶、目标 key 原样保留。
  比对与复制共用同一个映射表达式，因此重复执行必然收敛（第二次 `copied` 为 0）；
- 按 mode 比对源/目标：相等则跳过（计入 `skipped`），不等或目标缺失则复制（计入 `copied`）；
- 实际复制复用 `MigrateKeys` 的复制内核（同/异端点自动适配 CopyObject / StreamCopy）。

响应：
```json
200 {
  "scanned": 100,           // 源侧扫描数
  "skipped": 60,            // 因 equal 跳过
  "copied": 38,             // 实际复制数
  "failed": 2,              // 复制失败数
  "failedKeys": ["bad.txt"],// 上限 200
  "truncated": false        // 源侧列举命中 100k 硬上限（第 100001 个起未参与本次同步）
}
```
`truncated=true` 时必须继续处理剩余对象（再次同步会从同一位置继续列举），否则超出上限的源对象**永不同步**且无任何信号。

### 计划任务（cron 定时增量同步）

```
GET /api/schedules
```
返回全部计划任务（按创建时间倒序，最新在前）；空清单为 `[]`。
```json
200 {"schedules":[{"id":"9c2f1a44-...","sourceAccountId":"a","sourceBucket":"src-bucket","sourcePrefix":"data/","targetAccountId":"b","targetBucket":"dst-bucket","targetPrefix":"backup/","mode":"etag","cron":"0 2 * * *","enabled":true,"createdAt":"2026-10-08T10:00:00Z","nextRunAt":"2026-10-09T02:00:00Z","lastRunAt":"2026-10-08T02:00:00Z","lastJobId":"uuid","lastError":""}]}
```
`lastRunAt`（从未运行时省略）、`lastJobId`（最近一次触发的任务 id，前端跳转进度用）与 `lastError`
（最近一次触发失败原因，成功或修复后更新计划时清除）为运行态；`enabled:false` 只冻结自动调度，不删除计划。

```
POST /api/schedules
```
```json
{
  "sourceAccountId":"a","sourceBucket":"src-bucket","sourcePrefix":"data/",
  "targetAccountId":"b","targetBucket":"dst-bucket","targetPrefix":"backup/",
  "mode":"etag","cron":"0 2 * * *","enabled":true
}
```
- `cron` 为**标准 5 字段**（分 时 日 月 周），仅数字与 `*`/`-`/`,`/`/`；不支持 `MON`/`JAN` 名字、
  每字段仅一个 `a/b` 步进。`0 0 30 2 *` 这类**永不触发**的表达式拒收（400）。
- `mode`：`etag`（默认）/ `size_mtime` / `always`，空串按 `etag`；`enabled` 缺省 `true`；
  桶缺省回退账号默认桶（两端都为空则 400）。引用的账号不存在回 404、配置缺密钥回 400。
- 触发语义（均有测试钉住）：停机错过的多个槽**只补跑一次**；上一轮未结束**不叠加**（手动 run 回 409）；
  触发失败**记录 `lastError` 且排期照常前移**（坏配置不会每分钟重试轰炸）。
- 计划落盘 `S3C_DATA_DIR/schedules.json`（0600，原子写），重启自动恢复；手动 run 不改自动排期。
```json
201 {"schedule":{"id":"9c2f1a44-...","sourceAccountId":"a","sourceBucket":"src-bucket","sourcePrefix":"data/","targetAccountId":"b","targetBucket":"dst-bucket","targetPrefix":"backup/","mode":"etag","cron":"0 2 * * *","enabled":true,"createdAt":"2026-10-08T10:00:00Z","nextRunAt":"2026-10-09T02:00:00Z"}}
```

```
PUT /api/schedules/{id}
```
请求体与 `POST /api/schedules` 相同（整体替换，保留 `id`/`createdAt`/运行态）；`cron` 变更时重算排期。
```json
200 {"schedule":{"id":"9c2f1a44-...","sourceAccountId":"a","sourceBucket":"src-bucket","sourcePrefix":"data/","targetAccountId":"b","targetBucket":"dst-bucket","targetPrefix":"backup/","mode":"size_mtime","cron":"30 3 * * *","enabled":false,"createdAt":"2026-10-08T10:00:00Z","nextRunAt":"2026-10-09T03:30:00Z"}}
```

```
DELETE /api/schedules/{id}
```
```json
200 {"deleted":"9c2f1a44-..."}
```

```
POST /api/schedules/{id}/run
```
立即触发一次（不改自动排期），返回异步任务 id；进度经 `GET /api/migrate/jobs/{id}` 与 SSE 订阅
（与迁移任务同链路）。上一轮未结束回 409；计划或账号不存在回 404；账号配置已损坏回 400；在册任务满回 503。
```json
202 {"jobId":"uuid","scheduleId":"9c2f1a44-..."}
```

### API 契约（OpenAPI 3.0）

```
GET /api/openapi.json
```

OpenAPI 3.0 规范，作为 84 个 `/api/*` 端点的契约单一来源；**经过鉴权层**——配置了 `S3C_TOKEN` 时无 token 访问返回 401，且需 `S3C_EXPOSE_OPENAPI=1` 才暴露（否则 404）。
前端**已经**以它为源生成 TypeScript 类型与端点封装：`cd apps/web && pnpm gen:api` 从提交的
黄金契约生成 `src/api/schema.d.ts`（类型）与 `src/api/operations.ts`（`operationId → method / path /
路径参数`），`endpoints.ts` 的 URL 与 method 全部经 `opPath()` 取自生成物；`pnpm gen:api --check`
（由 `src/api/generated.gate.test.ts` 在 `pnpm test` 内调用）钉住「spec 改了而忘了重新生成」。
同一份契约也可用于 Swagger UI / 契约测试。
共享 `components.schemas` / `parameters` / `responses` 已全部接线为 `$ref`（`refSchema` / `refParam` / `refResp`）。
鉴权与分组已机器可读：顶层 `security: [{bearerAuth: []}]` 要求 `components.securitySchemes.bearerAuth`（`type: http`、`scheme: bearer`），真实豁免鉴权的 `/api/health` 与 `/api/metrics` 逐 operation 显式声明 `security: []`（`middleware.go` 的 `withAuth` 是唯一真值来源；`/api/openapi.json` 不豁免）；顶层 `tags` 声明全部 11 个分组（`accounts` / `buckets` / `bucket-settings` / `objects` / `object-meta` / `multipart` / `versions` / `trash` / `migrate` / `schedules` / `system`），每个 operation 至少归入其中一个。该不变式由 `openapi_auth_test.go` 机械校验。

## 请求示例（curl）

以下示例假设后端监听 `127.0.0.1:8080`，且已设置 `S3C_TOKEN`（未设置 token 时无需 `Authorization` 头）。先导出变量：

```bash
export BASE="http://127.0.0.1:8080"
export S3C_TOKEN="<Bearer Token>"
export ACCOUNT_ID="<账号 UUID，见 GET /api/accounts 返回的 id>"
```

### accounts

```bash
# 列出全部账号
curl -sS -X GET "$BASE/api/accounts" \
  -H "Authorization: Bearer $S3C_TOKEN"
```
```json
200 {"accounts":[{"id":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","name":"minio","endpoint":"http://localhost:9000","accessKey":"AKIAEXAMPLE","secretSet":true,"bucket":"my-bucket","pathStyle":true,"useSSL":false}]}
```

```bash
# 新建账号（secretKey 只在此请求体内出现，响应不回传）
curl -sS -X POST "$BASE/api/accounts" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"minio","endpoint":"http://localhost:9000","accessKey":"AKIAEXAMPLE","secretKey":"secret","bucket":"my-bucket","pathStyle":true,"useSSL":false}'
```
```json
201 {"id":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","name":"minio","endpoint":"http://localhost:9000","accessKey":"AKIAEXAMPLE","secretSet":true,"bucket":"my-bucket","pathStyle":true,"useSSL":false}
```

### buckets

```bash
# 列出账号下的桶
curl -sS -X GET "$BASE/api/accounts/$ACCOUNT_ID/buckets" \
  -H "Authorization: Bearer $S3C_TOKEN"
```
```json
200 {"buckets":[{"name":"my-bucket","creationDate":"2026-09-30T05:00:00Z"}]}
```

### bucket-settings

```bash
# 配置桶 CORS（rules 传空数组 = 删除全部规则）
curl -sS -X PUT "$BASE/api/accounts/$ACCOUNT_ID/bucket/cors" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bucket":"my-bucket","rules":[{"id":"r1","allowedMethods":["GET","PUT"],"allowedOrigins":["https://app.example.com"],"allowedHeaders":["*"],"exposeHeaders":["ETag"],"maxAgeSeconds":3600}]}'
```
```json
200 {"updated":1}
```

### objects

```bash
# 生成预签名 URL（method=get|put|post；expiresIn 秒，默认 3600，上限 86400）
curl -sS -X POST "$BASE/api/accounts/$ACCOUNT_ID/presign" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bucket":"my-bucket","key":"docs/a.txt","method":"get","expiresIn":3600}'
```
```json
200 {"method":"get","bucket":"my-bucket","key":"docs/a.txt","url":"https://s3.example.com/my-bucket/docs/a.txt?X-Amz-Signature=...","expiresIn":3600}
```

```bash
# 存储分析与成本洞察（可选 prefix；返回按存储类 / 前缀的用量与建议）
curl -sS -X GET "$BASE/api/accounts/$ACCOUNT_ID/storage-report?bucket=my-bucket&prefix=logs/" \
  -H "Authorization: Bearer $S3C_TOKEN"
```
```json
200 {"bucket":"my-bucket","prefix":"logs/","objectCount":6,"totalSize":32212254720,"truncated":false,"monthlyCost":0.53,"prefixGroupCount":3,"byStorageClass":[],"byPrefix":[],"recommendations":[]}
```

### object-meta

```bash
# 覆盖写入对象标签（tags 传空数组 = 清空）
curl -sS -X PUT "$BASE/api/accounts/$ACCOUNT_ID/object-tags" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bucket":"my-bucket","key":"docs/a.txt","tags":[{"key":"env","value":"prod"}]}'
```
```json
200 {"tags":[{"key":"env","value":"prod"}]}
```

### multipart

```bash
# 初始化分段上传
curl -sS -X POST "$BASE/api/accounts/$ACCOUNT_ID/multipart/init" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bucket":"my-bucket","key":"big.bin","contentType":"application/octet-stream"}'
```
```json
200 {"uploadId":"UPLOAD123","key":"big.bin","bucket":"my-bucket"}
```

### versions

```bash
# 列出版本与删除标记（需桶版本控制已开启）
curl -sS -X GET "$BASE/api/accounts/$ACCOUNT_ID/versions?bucket=my-bucket&prefix=docs/" \
  -H "Authorization: Bearer $S3C_TOKEN"
```
```json
200 {"versions":[{"key":"docs/a.txt","versionId":"v2","isLatest":true,"lastModified":"2026-09-30T05:00:00Z","size":17,"etag":"\"9c1d2f3a4b5c6d7e\"","storageClass":"STANDARD"}],"deleteMarkers":[],"isTruncated":false,"nextKeyMarker":"","nextVersionIdMarker":""}
```

### trash

```bash
# 彻底清除某 key 的全部版本与删除标记
curl -sS -X POST "$BASE/api/accounts/$ACCOUNT_ID/trash/purge" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bucket":"my-bucket","key":"docs/b.txt"}'
```
```json
200 {"purged":"docs/b.txt","deleted":3}
```

### migrate

```bash
# 增量同步（mode=etag|size_mtime|always）
curl -sS -X POST "$BASE/api/migrate/sync" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sourceAccountId":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","sourceBucket":"src-bucket","sourcePrefix":"","targetAccountId":"2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809","targetBucket":"dst-bucket","targetPrefix":"","mode":"etag"}'
```
```json
200 {"scanned":100,"skipped":60,"copied":40,"failed":0,"failedKeys":[],"truncated":false}
```

### schedules

```bash
# 创建计划任务（cron 定时增量同步）
curl -sS -X POST "$BASE/api/schedules" \
  -H "Authorization: Bearer $S3C_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sourceAccountId":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","sourceBucket":"src-bucket","sourcePrefix":"data/","targetAccountId":"2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809","targetBucket":"dst-bucket","targetPrefix":"backup/","mode":"etag","cron":"0 2 * * *","enabled":true}'
```
```json
201 {"schedule":{"id":"9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","sourceAccountId":"1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d","sourceBucket":"src-bucket","sourcePrefix":"data/","targetAccountId":"2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809","targetBucket":"dst-bucket","targetPrefix":"backup/","mode":"etag","cron":"0 2 * * *","enabled":true,"createdAt":"2026-10-08T10:00:00Z","nextRunAt":"2026-10-09T02:00:00Z"}}
```

```bash
# 立即触发一次（不改自动排期）
curl -sS -X POST "$BASE/api/schedules/9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d/run" \
  -H "Authorization: Bearer $S3C_TOKEN"
```
```json
202 {"jobId":"e5d4c3b2-...","scheduleId":"9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d"}
```

### system

```bash
# 健康检查（withAuth 豁免：无需 Authorization 头）
curl -sS -X GET "$BASE/api/health"
```
```json
200 {"status":"ok","version":"v1.0.0","time":"2026-09-30T05:00:00Z","store":{"ok":true}}
```

```bash
# 拉取 OpenAPI 规范（需 S3C_EXPOSE_OPENAPI=1；该端点**不**豁免鉴权）
curl -sS -X GET "$BASE/api/openapi.json" \
  -H "Authorization: Bearer $S3C_TOKEN"
```
```json
200 {"openapi":"3.0.3","info":{"title":"s3client API","version":"v1.0.0"}}
```

## 静态资源

- 非 `/api` 路径由 Go 托管 `apps/web/dist`；未命中的页面路由（无扩展名）回退到 `index.html`（SPA），带扩展名的缺失资源返回 404。
- 所有响应（含静态资源）携带基础安全头：`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: no-referrer`。
