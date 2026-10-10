# API / S3 错误目录

服务端通过防腐层 `s3wrap` 将 AWS SDK / S3 API 错误映射为**稳定 HTTP 状态**与**短英文用户消息**（前端可再 i18n）。  
实现：`apps/server/internal/s3wrap/errors.go`（`UserMessage` / `HTTPStatus` / `ErrorCode` / `IsNotFound`）。

## 映射表

| S3 / 条件 | HTTP | UserMessage | 备注 |
|---|---:|---|---|
| `NoSuchBucket` | 404 | `bucket not found` | `IsNotFound` |
| `NoSuchKey` / `NotFound` / `NoSuchVersion` | 404 | `object not found` | |
| `AccessDenied` | 403 | `access denied` | |
| `InvalidAccessKeyId` / `SignatureDoesNotMatch` | 403 | `invalid credentials` | |
| `InvalidRequest` / `InvalidArgument` / `MalformedPolicy` / `MalformedXML` / `InvalidStorageClass` / `InvalidPartOrder` | 400 | `invalid request` | `InvalidPartOrder` 由 handler 的段号升序校验先行拦截（`parts must be ordered by ascending partNumber`） |
| `EntityTooLarge` | 400 | `entity too large` | `PutObject`/`CopyObject` 会把该错误码归一为 `s3wrap.ErrObjectTooLarge`（见下） |
| `BucketNotEmpty` | 409 | `bucket not empty` | |
| `PreconditionFailed` | 412 | `precondition failed (object changed or already exists)` | 条件写不满足（#5：presign/mkdir/copy 的 `ifMatch`/`ifNoneMatch`） |
| `ConditionalRequestConflict` | 409 | `conditional request conflict, retry after re-reading the object` | 条件写的并发冲突（S3 建议重读后重试） |
| `ObjectLocked` | 409 | `object is locked by retention or legal hold` | Object Lock 合规保留期内的写 / 删 / 改保留被拒 |
| `RetentionPeriodTooShort` | 400 | `retention period too short (not later than current retention)` | 新保留期早于当前保留期 |
| `ObjectLockConfigurationNotFoundError` | 400 | `object lock is not enabled for this bucket` | 桶未启用 Object Lock 时的写操作（读操作在 `s3wrap` 已降级，不外抛） |
| `NotImplemented` | 501 | `not supported by this storage endpoint` | 厂商未实现该 API（按支持度降级的一部分） |
| `InvalidRange` | 416 | （proxy 专用文案） | `proxyErr` |
| `SlowDown` / `ServiceUnavailable` / `RequestTimeout` | 503 | `storage temporarily unavailable` | 两表（`HTTPStatus` / `UserMessageForCode`）对 `RequestTimeout` 同口径归 503（review §Nit） |
| `NoSuchUpload` | 500* | `multipart upload not found` | *HTTP 默认走 fallback 500；消息单独映射 |
| `errors.Is(err, s3wrap.ErrObjectTooLarge)` | 400** | `object exceeds 5GB single-put limit; use multipart upload` | 单次上传/复制 >5GB；`PutObject`/`CopyObject` 用 `%w` 包装 |
| `errors.Is(err, s3wrap.ErrSourceDeleteFailed)` | 500* | `copied but failed to delete source` | 移动半成功；handler 用 `%w` 包装 |
| `errors.Is(err, s3wrap.ErrPartialDelete)` | 409 | `some objects could not be deleted (N deleted)`（结构化分支带**已删计数**；裸 sentinel 回退基础文案 `some objects could not be deleted`） | 批量删除/S3 在 **200 响应体内**逐 key 报错（桶策略 / 保留期 / MFA Delete）；`HTTPStatus` 已收录该 sentinel → 409（review §R17，此前回落 500 使精细文案不可达、丢失已删计数）；回收站 purge 部分失败的 handler 显式响应同为 409 且带 `{"purged":…,"deleted":n,"error":…}` |
| 其他 | 500 | `storage operation failed` | |

> **已移除字符串匹配**：应用层错误（5GB 上限 / 删源失败）改用 sentinel + `errors.Is` 识别
> （`ErrObjectTooLarge` / `ErrSourceDeleteFailed`），不再依赖 `strings.Contains(err.Error(), ...)`；
> 即使 SDK/各 S3 实现的文案变化也不会静默失效。`**HTTPStatus` 对包装后的错误仍经 `errors.As` 取到
> `EntityTooLarge` 码 → 400。

> `HTTPStatus` 未单独列出的码（如 `NoSuchUpload`）回落 **500**；业务 handler 可在映射前特判。

> **逐 key 失败没有 error 值**：`DeleteObjects` 的失败只出现在 200 响应体内，拿不到 `error`，
> 因此 `s3wrap.UserMessageForCode(code)`（`service/delete.go` 逐 key 分支复用同一张映射表；handler 侧
> 委托入口为 `s3UserMessage` / `s3HTTPStatus`，见 `handler/s3_errors.go`）复用同一张映射表；
> 未收录的码回落 `storage operation failed`。同一个码在**指标标签**上另有白名单
> （`metricErrorCodes`）：不可信端点返回的任意 `<Code>` 一律归 `other`，避免标签基数无界。

## 非 S3 来源的固定文案

以下错误不来自上游 S3，而是本服务的本地校验 / 资源约束，文案固定、不经 `s3wrap` 映射：

| 条件 | HTTP | 文案 | 备注 |
|---|---:|---|---|
| 预签名生成失败（取凭证、输入序列化等） | 500 | `failed to create presigned url` | 此前被 `u, _ :=` 吞掉，返回 `200 {"url":""}`（KNOWN_ISSUES #23） |
| 在册异步任务数达上限 | 503 | `too many running jobs; retry later` | 上限 256 个未终结任务（KNOWN_ISSUES #17） |
| 并发流式请求数达上限 | 503 | `too many concurrent streaming requests` | `withStreamLimit`，上限 32 |
| 单任务 SSE 订阅数达上限 | 503 | `too many subscribers for this job` | 每任务上限 16，防止终态关闭耗时随订阅数线性增长 |
| 请求体超过上限（16MB） | 413 | `request body too large (max 16MB)` | 与「JSON 无效」的 400 区分开（此前被 `LimitReader` 截断成 400） |
| 每 IP 令牌桶超限 | 429 | `rate limit exceeded` | `withRateLimit`，120 次/分钟/IP，作用于**全部** `/api/*`（在 `withAuth` 外层，未鉴权也受限）；`Retry-After: 5`；仅 `S3C_TRUSTED_PROXIES` 配置下才采信 `X-Forwarded-For` |
| download-zip 传入空 key | 400 | `keys must not contain empty entries` | 空 key 会被 S3 当成「列举桶」，把 ListBucket XML 塞进 ZIP |
| 分段顺序不是升序 | 400 | `parts must be ordered by ascending partNumber` | S3 要求 `CompleteMultipartUpload` 的 Parts 升序 |
| `ifNoneMatch` 非 `*` | 400 | `ifNoneMatch must be *` | 条件写边界校验（`handler/conditions.go`；S3 只接受「仅当不存在」语义） |
| `ifMatch` 超长（>512B） | 400 | `ifMatch is too long` | 条件值边界校验 |
| `ifMatch` 含非可见 ASCII / 空格 | 400 | `ifMatch must be printable ASCII without spaces` | 条件值作为 HTTP 头发出，CRLF / 空格在边界拦截（防头注入） |
| presign 的 get/post 携带条件字段 | 400 | `conditional write is only supported for method put` | 条件写只作用于 PUT；显式拒绝而非静默忽略 |
| `checksumAlgorithm` 非法 | 400 | `checksumAlgorithm must be CRC64NVME, SHA256, CRC32C or SHA1` | `POST copy-object` 复制时物化校验和的算法枚举 |
| Object Lock 默认保留输入非法 | 400 | `defaultRetentionMode must be GOVERNANCE or COMPLIANCE` / `default retention days and years must not be negative` / `specify only one of defaultRetentionDays or defaultRetentionYears` / `defaultRetentionDays or defaultRetentionYears is required` | `PUT bucket/object-lock` 边界校验（模式必填、天 / 年二选一且 ≥1） |
| 在既有桶上启用 Object Lock | 409 | `object lock cannot be enabled on an existing bucket` | RustFS 返回 `InvalidBucketState`；Object Lock 只能在建桶时启用 |
| 保留期输入非法 | 400 | `mode must be GOVERNANCE or COMPLIANCE` / `retainUntilDate is required` / `retainUntilDate must be RFC3339, e.g. 2031-02-03T04:05:06Z` / `retainUntilDate must be in the future` | `PUT object-retention` 边界校验 |
| 法定保留状态非法 | 400 | `status must be ON or OFF` | `PUT object-legal-hold` 边界校验 |

## Handler 约定

- `writeInternalErr`：可识别 S3 错误 → `s3HTTPStatus` + `s3UserMessage`；否则 500 + 通用文案。
- 账号读取（`accountClient` 与 migrate 系列）：仅 `store.ErrNotFound` → 404 `account not found`；其余 store 读取故障 → 500 `failed to load … account`（误报 404 会让用户去查一个本来存在的账号；review §Nit）。
- 批量操作：`lastError` / `failedKeys` 使用 `failed at {key}: {UserMessage}`。
- 响应 JSON：`{"error":"..."}`（见 `docs/api.md`）。
- `POST /api/migrate/sync`：`mode` 非法 → 400 `mode must be etag, size_mtime or always`；账号不存在 → 404（store 读取故障 → 500 `failed to load … account`）；账号配置无效 → 400 `invalid ... account configuration`；成功返回 `scanned/skipped/copied/failed/failedKeys/truncated/lastError`（见 `api.md`）。
- `GET /api/openapi.json`：默认（未设置 `S3C_EXPOSE_OPENAPI=1`）→ 404（不暴露 API 契约）；开启后配置了 `S3C_TOKEN` 时需 Bearer 鉴权，否则 401 `unauthorized`。

## 前端

- `apps/web/src/errors.ts` 的 `toErrorMessage` 展示后端 `error` 字段；勿依赖 SDK 原文。
- 鉴权失败：`401 unauthorized`（与 S3 映射无关）。
