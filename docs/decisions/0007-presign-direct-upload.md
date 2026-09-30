# ADR-007：浏览器预签名直传，对象字节不经过本服务

## Status

Accepted

## Date

2026-09-18（回溯记录；`s3wrap/presign.go` 于 2026-09-18 首次引入）

## Context

对象上传 / 下载若经 Go 服务代理，带宽与内存成本会集中到单进程（容器内存限额 512MB），
大文件还需要额外缓冲或落盘；同时 `secretKey` 只存服务端、**绝不能下发前端**。S3 生态的
标准做法是 Signature V4 预签名 URL，由客户端直接与 S3 交互。

现状（2026-09-30 回读源码核实）：

- [`apps/server/internal/s3wrap/presign.go`](../../apps/server/internal/s3wrap/presign.go)：
  `PresignPut` / `PresignPost` / `PresignGetVersion` / `PresignUploadPart`，均校验
  `expires > 0`（`errInvalidExpiry`）。
- 时长策略（[`apps/server/internal/handler/objects.go`](../../apps/server/internal/handler/objects.go)
  与 [`multipart.go`](../../apps/server/internal/handler/multipart.go)）：`expiresIn` ≤ 0 时
  默认 **1h**；> 24h 钳到 **24h**；无 1h 下限。（「S3 协议预签名上限 7 天」沿用
  [`architecture.md`](../architecture.md) §2 既有表述，**未独立核对 AWS 文档**。）
- 前端 [`apps/web/src/api/upload.ts`](../../apps/web/src/api/upload.ts)：用 **XHR** 直传
  （只有 XHR 提供 `upload.onprogress`）；[`apps/web/src/upload.ts`](../../apps/web/src/upload.ts)：
  文件 < 100MB 走单次预签名 PUT（`expiresIn: 3600`），≥ 100MB 自动分段（每段 10MB、
  **4 路并发**、退避重试 500/1500/3500ms、任一段最终失败即 `abort` 清理会话）；
  上传队列同时最多 **2 个文件**（`useUploadQueue.ts` 的 `UPLOAD_CONCURRENCY = 2`）。
- 后端在签发 URL 后不接触对象字节；multipart 仅逐段签发 `PresignUploadPart`。

## Decision

**对象读 / 写全部走浏览器直传预签名 URL，服务端只生成短时签名；`expiresIn` 默认 1h、
钳制 24h；≥ 100MB 自动走 multipart 分段直传。** 密钥永不回传前端。

## Alternatives Considered

### 服务端代理上传 / 下载（字节过 Go 服务）
- Pros：鉴权集中、可统一审计与限流。
- Cons：单进程带宽 / 内存瓶颈，大文件需落盘或大缓冲；与 512MB 容器限额冲突。
- 被拒：[`architecture.md`](../architecture.md) §1 核心设计决策 #1——大流量不经过 Go 服务。

### 预签名 POST 表单 vs 预签名 PUT
- 两者**都实现**了（`PresignPost` 保留为 API 能力，供外部调用方做表单直传）；
- 前端主路径选 **PUT**：更简单、可逐请求带 `Content-Type`，配合 XHR 取进度。

### 分段阈值 100MB / 段 10MB / 4 路并发
- S3 约束：单 PUT 上限 5GB，除最后一段外每段最小 5MB；
- 取舍：取 10MB「平衡并发数与段数量」（`upload.ts` 注释原文），4 路并发与重试退避覆盖弱网。

## Consequences

- 带宽成本转移到 S3 与客户端，服务端内存预算大幅下降。
- 新增失败面：CORS（分段需 `ExposeHeader: ETag`）、签名过期、分段会话残留——
  对应 Runbook R-7（[`OPERATIONS.md`](../OPERATIONS.md)）。
- 密钥安全边界成形：前端只见签名 URL，不见密钥（[`threat-model.md`](../threat-model.md) 边界 B）。
