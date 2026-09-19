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
| `InvalidRange` | 416 | （proxy 专用文案） | `proxyErr` |
| `SlowDown` / `ServiceUnavailable` | 503 | `storage temporarily unavailable` | |
| `NoSuchUpload` | 500* | `multipart upload not found` | *HTTP 默认走 fallback 500；消息单独映射 |
| `errors.Is(err, s3wrap.ErrObjectTooLarge)` | 400** | `object exceeds 5GB single-put limit; use multipart upload` | 单次上传/复制 >5GB；`PutObject`/`CopyObject` 用 `%w` 包装 |
| `errors.Is(err, s3wrap.ErrSourceDeleteFailed)` | 500* | `copied but failed to delete source` | 移动半成功；handler 用 `%w` 包装 |
| `errors.Is(err, s3wrap.ErrPartialDelete)` | 409† | `some objects could not be deleted` | 批量删除/S3 在 **200 响应体内**逐 key 报错（桶策略 / 保留期 / MFA Delete）；† `HTTPStatus` 未收录该 sentinel（回落 500），409 来自回收站 purge handler 的显式响应（`{"purged":…,"deleted":n,"error":…}`） |
| 其他 | 500 | `storage operation failed` | |

> **已移除字符串匹配**：应用层错误（5GB 上限 / 删源失败）改用 sentinel + `errors.Is` 识别
> （`ErrObjectTooLarge` / `ErrSourceDeleteFailed`），不再依赖 `strings.Contains(err.Error(), ...)`；
> 即使 SDK/各 S3 实现的文案变化也不会静默失效。`**HTTPStatus` 对包装后的错误仍经 `errors.As` 取到
> `EntityTooLarge` 码 → 400。

> `HTTPStatus` 未单独列出的码（如 `NoSuchUpload`）回落 **500**；业务 handler 可在映射前特判。

> **逐 key 失败没有 error 值**：`DeleteObjects` 的失败只出现在 200 响应体内，拿不到 `error`，
> 因此 `s3wrap.UserMessageForCode(code)`（handler 侧 `s3UserMessageForCode`）复用同一张映射表；
> 未收录的码回落 `storage operation failed`。同一个码在**指标标签**上另有白名单
> （`metricErrorCodes`）：不可信端点返回的任意 `<Code>` 一律归 `other`，避免标签基数无界。

## 非 S3 来源的固定文案

以下错误不来自上游 S3，而是本服务的本地校验 / 资源约束，文案固定、不经 `s3wrap` 映射：

| 条件 | HTTP | 文案 | 备注 |
|---|---:|---|---|
| 预签名生成失败（取凭证、输入序列化等） | 500 | `failed to create presigned url` | 此前被 `u, _ :=` 吞掉，返回 `200 {"url":""}`（todolist #23） |
| 在册异步任务数达上限 | 503 | `too many running jobs; retry later` | 上限 256 个未终结任务（todolist #17） |
| 并发流式请求数达上限 | 503 | `too many concurrent streaming requests` | `withStreamLimit`，上限 32 |
| 单任务 SSE 订阅数达上限 | 503 | `too many subscribers for this job` | 每任务上限 16，防止终态关闭耗时随订阅数线性增长 |
| 请求体超过上限（16MB） | 413 | `request body too large (max 16MB)` | 与「JSON 无效」的 400 区分开（此前被 `LimitReader` 截断成 400） |
| download-zip 传入空 key | 400 | `keys must not contain empty entries` | 空 key 会被 S3 当成「列举桶」，把 ListBucket XML 塞进 ZIP |
| 分段顺序不是升序 | 400 | `parts must be ordered by ascending partNumber` | S3 要求 `CompleteMultipartUpload` 的 Parts 升序 |

## Handler 约定

- `writeInternalErr`：可识别 S3 错误 → `s3HTTPStatus` + `s3UserMessage`；否则 500 + 通用文案。
- 批量操作：`lastError` / `failedKeys` 使用 `failed at {key}: {UserMessage}`。
- 响应 JSON：`{"error":"..."}`（见 `docs/api.md`）。
- `POST /api/migrate/sync`：`mode` 非法 → 400 `mode must be etag, size_mtime or always`；账号不存在 → 404；账号配置无效 → 400 `invalid ... account configuration`；成功返回 `scanned/skipped/copied/failed/failedKeys/lastError`（见 `api.md`）。
- `GET /api/openapi.json`：默认（未设置 `S3C_EXPOSE_OPENAPI=1`）→ 404（不暴露 API 契约）；开启后配置了 `S3C_TOKEN` 时需 Bearer 鉴权，否则 401 `unauthorized`。

## 前端

- `apps/web/src/errors.ts`（若存在）或 `toErrorMessage` 展示后端 `error` 字段；勿依赖 SDK 原文。
- 鉴权失败：`401 unauthorized`（与 S3 映射无关）。
