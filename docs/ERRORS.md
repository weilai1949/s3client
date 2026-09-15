# API / S3 错误目录

服务端通过防腐层 `s3wrap` 将 AWS SDK / S3 API 错误映射为**稳定 HTTP 状态**与**短英文用户消息**（前端可再 i18n）。  
实现：`server/internal/s3wrap/errors.go`（`UserMessage` / `HTTPStatus` / `ErrorCode` / `IsNotFound`）。

## 映射表

| S3 / 条件 | HTTP | UserMessage | 备注 |
|---|---:|---|---|
| `NoSuchBucket` | 404 | `bucket not found` | `IsNotFound` |
| `NoSuchKey` / `NotFound` / `NoSuchVersion` | 404 | `object not found` | |
| `AccessDenied` | 403 | `access denied` | |
| `InvalidAccessKeyId` / `SignatureDoesNotMatch` | 403 | `invalid credentials` | |
| `InvalidRequest` / `InvalidArgument` / `MalformedPolicy` / `MalformedXML` / `InvalidStorageClass` | 400 | `invalid request` | |
| `EntityTooLarge` | 400 | `entity too large` | `PutObject`/`CopyObject` 会把该错误码归一为 `s3wrap.ErrObjectTooLarge`（见下） |
| `BucketNotEmpty` | 409 | `bucket not empty` | |
| `InvalidRange` | 416 | （proxy 专用文案） | `proxyErr` |
| `SlowDown` / `ServiceUnavailable` | 503 | `storage temporarily unavailable` | |
| `NoSuchUpload` | 500* | `multipart upload not found` | *HTTP 默认走 fallback 500；消息单独映射 |
| `errors.Is(err, s3wrap.ErrObjectTooLarge)` | 400** | `object exceeds 5GB single-put limit; use multipart upload` | 单次上传/复制 >5GB；`PutObject`/`CopyObject` 用 `%w` 包装 |
| `errors.Is(err, s3wrap.ErrSourceDeleteFailed)` | 500* | `copied but failed to delete source` | 移动半成功；handler 用 `%w` 包装 |
| 其他 | 500 | `storage operation failed` | |

> **已移除字符串匹配**：应用层错误（5GB 上限 / 删源失败）改用 sentinel + `errors.Is` 识别
> （`ErrObjectTooLarge` / `ErrSourceDeleteFailed`），不再依赖 `strings.Contains(err.Error(), ...)`；
> 即使 SDK/各 S3 实现的文案变化也不会静默失效。`**HTTPStatus` 对包装后的错误仍经 `errors.As` 取到
> `EntityTooLarge` 码 → 400。

> `HTTPStatus` 未单独列出的码（如 `NoSuchUpload`）回落 **500**；业务 handler 可在映射前特判。

## Handler 约定

- `writeInternalErr`：可识别 S3 错误 → `s3HTTPStatus` + `s3UserMessage`；否则 500 + 通用文案。
- 批量操作：`lastError` / `failedKeys` 使用 `failed at {key}: {UserMessage}`。
- 响应 JSON：`{"error":"..."}`（见 `docs/API.md`）。
- `POST /api/migrate/sync`：`mode` 非法 → 400 `mode must be etag, size_mtime or always`；账号不存在 → 404；账号配置无效 → 400 `invalid ... account configuration`；成功返回 `scanned/skipped/copied/failed/failedKeys/lastError`（见 `API.md`）。
- `GET /api/openapi.json`：默认（未设置 `S3C_EXPOSE_OPENAPI=1`）→ 404（不暴露 API 契约）；开启后配置了 `S3C_TOKEN` 时需 Bearer 鉴权，否则 401 `unauthorized`。

## 前端

- `web/src/errors.ts`（若存在）或 `toErrorMessage` 展示后端 `error` 字段；勿依赖 SDK 原文。
- 鉴权失败：`401 unauthorized`（与 S3 映射无关）。
