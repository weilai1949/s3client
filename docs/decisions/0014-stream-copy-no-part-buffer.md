# ADR-014：跨 endpoint 流式复制改为无段缓冲直传

## Status

Accepted

## Date

2026-10-16

## Context

跨 endpoint 迁移的大对象走 `service.MultipartStreamCopy`（`apps/server/internal/service/stream_copy.go`）：每段 64MB 整段读入一块缓冲，再以 `bytes.NewReader` 上传。迁移并发固定为 4（`migrate.go` 默认与 handler 传参），峰值约 4 × 64MB = 256MB，而服务容器内存上限为 512MB（`docker-compose*.yml` 的 `deploy.resources.limits.memory: 512M`）。

段数上限 10000（S3 协议）× 段大小 64MB 得到单对象流式复制上限 640GB，登记于 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) #63（已决策维持现状）。S3 单对象上限为 5TiB，若要覆盖它，段大小需不小于约 550MiB；此时整段缓冲在任何合理并发下都会突破容器预算。因此问题的根源是「每段必须先完整驻留内存」，而不是段数上限本身。

2026-10-16 为评估本方案做了三项实测，结论作为前提记录在此：

- **真实 RustFS 上的流式 `UploadPart`**（pinned 镜像 `rustfs/rustfs:1.0.0-rc.3`，经 `s3wrap` 数据面客户端）：不可 seek 的流式 body **配合显式 `ContentLength`** 上传成功；不给长度则失败（`failed to rewind transport stream for retry, request stream is not seekable`）。
- **SDK 内置重试对流式 body 不可用**：`aws-sdk-go-v2/aws/retry/middleware.go` 在第二次尝试时调用 `RewindStream`，`smithy-go/transport/http/request.go` 对不可 seek 的流直接返回错误。段级重试必须由本项目自行实现。
- **载荷哈希依赖 `UNSIGNED-PAYLOAD`**：裸 SDK 客户端在不带该签名时会先计算载荷哈希并失败；本项目的 `unsignedPayloadSetter`（`s3wrap/client.go`）对带 body 请求注入 `UNSIGNED-PAYLOAD`，这是流式直传成立的前提。

## Decision

**已知大小**的跨 endpoint 复制改为**无段缓冲直传**；**大小未知**时仍走原来的有缓冲分段（上限不变）：

- `s3wrap.UploadPart` 接收**显式长度**的流式 body（新增 `size int64` 参数，写入 `ContentLength`）；段体是 `io.LimitReader(主源流, partSize)`，**不落盘、不整段缓冲**——进程内的段内存与段长解耦。
- 段长自适应：`planParts(size) = clamp(ceil(size / 10000), 64MiB, 5GiB)`；段数超过协议上限时在**建立分段上传之前**拒绝。单对象上限因此由 S3 的 5TiB 决定，不再由容器内存决定。
- 段级重试：主源流顺序供给各段；某段失败后主源流已不可信，该段及之后的段一律改用 `GetObjectStream` 的 Range 按偏移重读。单段最多尝试 3 次（`maxPartAttempts`），耗尽即 abort。**不使用 SDK 内置重试**（对不可 seek 的流无效）。
- **大小未知**（源以 chunked 返回）时保留有缓冲分段：段长固定 64MiB，必须整段驻留内存，单对象上限仍是 64MiB × 10000 = 640GiB，超限由段号判定拒绝（KNOWN_ISSUES #63 的结论对这条路径**继续成立**）。因此段缓冲池（`partBufPool` / `maxIdlePartBufs`）**保留**，只是不再被已知大小的路径使用。
- 同端点大对象（`CopyObject` 报 `EntityTooLarge`）改走 `UploadPartCopy`：各段服务端拷贝，数据不经过本进程（内存与带宽为零）；被拒（403）或不支持时回退流式。

落地位置：`apps/server/internal/service/stream_copy.go`（`planParts` / `runParts` / `directMultipartCopy` / `copyLargeSameEndpoint`）、`apps/server/internal/s3wrap/multipart.go`（`UploadPart` 长度参数、新增 `UploadPartCopy`）、`apps/server/internal/service/migrate.go`（同端点分支）。

RustFS 安全公告（CVE-2026-45042）的处置不在此列：受影响区间为 `1.0.0-alpha.1`–`1.0.0-beta.8`，`1.0.0-beta.9` 起修复，本仓锁定的 `1.0.0-rc.3` 在其外。

## Alternatives Considered

### 维持现状（640GB 上限）

- **Pros**：零改动、零风险；与已登记的 #63 决策一致。
- **Cons**：单对象上限长期停在 640GB，远低于 S3 的 5TiB；大对象迁移只能失败并 abort。

### 按比例放大分段缓冲

- **Pros**：实现最简单，只改常量。
- **Cons**：5TiB 需要约 550MiB 段，4 个并发即约 2.2GB 常驻缓冲，超出 512MB 容器预算数倍，直接 OOM。已由内存账否决。

### 分段前落盘到临时文件

- **Pros**：段可以 seek 重放，SDK 内置重试可以恢复使用。
- **Cons**：需要额外的临时磁盘容量（部署环境未约定 volume，容器可写层不适合大文件），I/O 约翻倍，还要承担清理与异常退出后的残留问题。

### 仅做「已知大小超限时提前拒绝」

- **Pros**：改动小，只省掉超限对象白传全部段的流量。
- **Cons**：上限不变，仅改善失败路径；不解决「上限低于 S3 协议」的问题。已作为独立改动先行落地，不构成本决策的替代。

### 同端点改用 UploadPartCopy（服务端复制）

- **Pros**：数据不经过本进程，内存与带宽都为零；同端点大对象可直接覆盖到 5TiB。
- **Cons**：只适用于同端点，不能替代跨 endpoint 的流式路径。**已采纳**，作为同端点分支与本决策一并落地（见 Decision）。

## Consequences

- **正面**：已知大小的路径**完全不再分配段缓冲**（HTTP 传输层直接消费限定长度的流），进程内存与段长解耦；单对象上限从 640GB 提升到 S3 协议的 5TiB；同端点大对象走 `UploadPartCopy` 后连带宽都省下。
- **代价 / 风险**：
  - 段级重试改为 Range GET 重读，源端 GET 次数最多为「段数 × 尝试次数」，计费与源端压力上升。
  - **大小未知**的对象仍受 640GiB 上限约束、段长无法自适应，且仍需 64MiB 段缓冲——该路径的取舍与 #63 一致，未变。
  - `s3wrap.UploadPart` 签名改变（增加 `size`），所有调用点已同步。
  - `UploadPartCopy` 用**目标端**凭证签名，源桶私有时会 403：迁移里必须保留回退流式的分支，否则跨账号整批失败（与 review R10 同因）。
  - 载荷完整性依赖 `UNSIGNED-PAYLOAD`：对 `http://` endpoint，完整性依赖 TLS，残留风险已在 [`threat-model.md`](../threat-model.md) 与 ADR-003 记录。
- **后续**：已同步 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) #63（限定为「大小未知」路径）、根 `README.md` 的「已知限制」、[`user-guide.md`](../user-guide.md) 与 `stream_copy.go` 注释。5TiB 量级无法在 CI 中直接测试，边界以注入的小段长覆盖（`stream_copy_direct_test.go` / `copy_large_test.go`），真实对端由 `internal/s3wrap/e2e_multipart_copy_test.go` 覆盖（流式 `UploadPart` 与 `UploadPartCopy`）。大对象压测与内存实测仍未做，已登记为 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) **#92**（含闭环方式：补基准与内存实测 + 真实大对象跑一次迁移）。

## Update（2026-10-16）

**决策不变**（无段缓冲直传、段级 Range 重试、`UploadPartCopy` 同端点分支均维持）。本篇 Context / Decision / Consequences 中的 640GB / 640GiB / 64MiB / 512MB 均为**变更前原文**（不改写历史结论），现行口径以本节为准。本节记录同日的三项后续变更：

1. **「大小未知」路径段长 64MiB → 105MiB，单对象上限 640GiB → 1025GiB**（105MiB × 10000 段 = 1,101,004,800,000 B ≥ 1TiB，用户口径「1024GB」）。105MiB 是**覆盖 1TiB 所需的最小整数 MiB 段长**（1TiB ÷ 10000 = 104.86MiB，向上取整到整 MiB），故实际能力是 1025GiB 而非恰好 1024GiB。`multipartPartSize` 同时是 `planParts` 的段长下限，已知大小路径的下限随之 64MiB → 105MiB（只影响「`ceil(size/10000)` 低于下限」的小分段，5TiB 上限不受影响）。
2. **容器内存预算 512M → 1G**（`docker-compose.yml` / `docker-compose.prod.yml` 的 server `deploy.resources.limits.memory`）。段缓冲内存账重算：单块分段缓冲 105MiB ≈ 预算 1/10；迁移并发 4 路最坏在飞 4 × 105MiB = 420MiB + 进程基线 < 1G（原 512M 预算下 420MiB 占 82%，会逼近 OOM）；`maxIdlePartBufs = 1` 不变（池常驻 105MiB）。Prometheus `S3ClientMemoryNearLimit` 阈值随之 410MB → 820MB（仍为容器上限的 80%），Grafana 红线与 [`OPERATIONS.md`](../OPERATIONS.md) §4.2 同步；ADR-009「调整内存预算前必须先重算常量」的义务已按其 `Update（2026-10-16）` 逐条复核。
3. **[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) #63 闭环移除**：640GB 上限已提升至 1024GB 口径，#63 计入该文件页头「已闭环编号」（编号不回收）。钉值证据为 `TestMultipartStreamCopyPartSizeIs105MB`（段长）+ `TestMaxMultipartPartsIsProtocolLimit`（10000 段协议上限）；同步面：根 `README.md`「已知限制」、[`user-guide.md`](../user-guide.md)、`docs/en/index.md`、[`OPERATIONS.md`](../OPERATIONS.md) §4.2 / §10.2、迁移面板 `migrate.limitHint`（中英）、[`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`、[`BATCHES.md`](../BATCHES.md) §CZ。

### 本轮备选

- **只抬容器上限、段长维持 64MiB**：被否——单对象上限 = 段长 × 10000，抬预算不改变上限。
- **段长取 256MiB（上限约 2500GiB）**：被否——最坏在飞 4 × 256MiB = 1GiB 顶满新预算，仍需再抬到 2G+，运维与告警面成本高于 105MiB 方案；105MiB 既覆盖 1TiB 又是内存增量最小的选择。
