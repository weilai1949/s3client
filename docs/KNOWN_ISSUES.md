# s3client 已知问题（Known Issues）

> 本文件是**缺陷 / 外部阻塞 / 技术债**的**唯一来源**，只收录**尚未闭环**的项。
> **功能候选与版本规划不在本文件**——见 [`ROADMAP.md`](ROADMAP.md) §三（战略层唯一来源）；
> 已修复 / 已完成台账见 [`FEATURES.md`](FEATURES.md)；发版历史见 [`CHANGELOG.md`](../CHANGELOG.md)。
> 最近一轮综合评估（2026-09-16，五维度：代码质量 / 漏洞 / 死代码 / 降级 / 自我迭代）见
> [`archive/assessment.md`](archive/assessment.md)。
>
> 状态图例：⬜ 待处理 · ⏳ 处理中（仅列剩余工作） · ➖ 已决策（不做 / 维持现状） · ⛔ 外部阻塞（外部凭证未获取等非代码工作）
>
> **两源分工（2026-09-24 起）**：本文件收**问题**（缺陷 / 阻塞 / 技术债）；
> [`ROADMAP.md`](ROADMAP.md) 收**方向**（版本规划 / 功能候选）。同一编号只在一个文件里是「当前条目」，
> 跨文件引用必须带前缀：`KNOWN_ISSUES #N` / `ROADMAP §三 #N`。
>
> **编号约定**：编号稳定——不因条目移除而重排、空号不回收。`apps/server/` 代码注释与历史提交
> 仍以 `KNOWN_ISSUES #N` 引用本清单（2026-09-24 前写作 `todolist #N`，两者同指），重排会使引用失真。
> **已闭环编号**（细节按编号去 [`FEATURES.md`](FEATURES.md) / [`CHANGELOG.md`](../CHANGELOG.md) 检索）：
> `#1–#24` / `#26–#39` / `#41` / `#45` / `#46` / `#60–#62` / `#64–#90`；
> **从未启用**：`#40` / `#42–#44`；**已迁出**：`#47–#59` 属 [ROADMAP.md](ROADMAP.md) §三 3.2 的
> 功能候选（非问题）。故编号不连续属预期，不是漏登记。
>
> 最后更新：2026-10-16（2026-10-15 全仓对抗性渗透测试 P-1..P-15 全部闭环，逐条证据见
> [`FEATURES.md`](FEATURES.md) §CX 与 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`，
> 口径见 [`threat-model.md`](threat-model.md)。）

---

## 一、外部阻塞

> 非代码工作，依赖外部凭证 / 第三方通道；代码侧已无剩余工作，故**不占仓内排期**。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 25 | 桌面端分发与签名 | ROADMAP §三 #1 / 长期 | ⛔ | **外部阻塞**：Windows 代码签名证书 / Apple Developer ID + 公证是**外部凭证**（roadmap E6，⬜ 未获取），未获取前无法完成签名 / 公证，产物被 SmartScreen / Gatekeeper 拦截且无自动更新通道（roadmap R5）。代码侧无剩余工作（发布链已收口）；未签名产物以 `SHA256SUMS` + 手动放行说明过渡（[DEPLOYMENT.md](DEPLOYMENT.md) §5） |

## 二、技术债 / 缺陷

> 与代码 / 仓库硬约束直接相关、可在仓内闭环的项。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 63 | 流式复制单对象 640GB 上限（64MB × 10000 段）——**已收窄为「源对象大小未知」路径** | code-review-2026-09-24 Nit（刻意取舍） | ➖ | **2026-10-16 范围收窄**（[ADR-014](decisions/0014-stream-copy-no-part-buffer.md)）：已知大小的跨 endpoint 复制改为**无段缓冲直传**（`service/directMultipartCopy`），段长随对象放大到 5GiB，上限由 S3 单对象 5TiB 决定，**不再受本条约束**；`s3wrap.UploadPart` 增加显式 `size` 参数写入 `Content-Length`（不可 seek 的流必须显式给长度），某段失败后按偏移 Range 重读（单段最多 3 次）。本条限制**只对源以 chunked 返回、拿不到 `Content-Length` 的对象**继续成立：该路径保留有缓冲分段（段长固定 64MiB、整段驻留内存），上限 64MiB × 10000 = 640GiB，超限由段号判定拒绝并 abort（段号在**上传前**判定，不误杀第 10000 段）。因此段缓冲池（`partBufPool` / `maxIdlePartBufs`）**保留**——内存账（512MB 容器预算、一块缓冲 64MB）对这条路径依旧适用。同端点大对象（`CopyObject` 报 `EntityTooLarge`）改走 `UploadPartCopy` 分段落拷，数据不经过本进程，也**不受本条约束**；被拒（403）时仍逐 key 回退流式。默认值由 `TestMultipartStreamCopyPartSizeIs64MB` / `TestMaxMultipartPartsIsProtocolLimit` 钉住，边界由 `TestMultipartStreamCopyAcceptsExactlyMaxParts` / `TestMultipartStreamCopyRejectsPartOverLimit` / `TestMultipartStreamCopyByteCeiling` 覆盖；直传路径由 `stream_copy_direct_test.go` 与 `copy_large_test.go` 覆盖，真实对端由 `s3wrap/e2e_multipart_copy_test.go` 覆盖。**2026-10-08 补登**：该限制此前只写在 `stream_copy.go` 注释 / 本文件 / [`OPERATIONS.md`](OPERATIONS.md) §性能表，根 `README.md`「已知限制」把它推到用户可见面 |
| 91 | nginx 基座 digest 需按上游刷新节奏滚动更新 | 2026-10-16 渗透测试 P-4 处置的**维护期义务** | ➖ | **这不是「未闭环的缺陷」，而是已决策方案（仓库内构建边缘镜像）的维护期义务**（避免后来人误判成漏修）：截至 2026-10-16 实测**所有**官方 nginx alpine tag（1.28.3 / 1.29.8 / 1.30.6 / 1.31 / mainline / slim / unprivileged）仍 ship `zlib-1.3.2-r0`（CVE-2026-85091 HIGH），而修复版 `1.3.2-r1` 已发布——因此边缘镜像**必须**保持「[`deploy/nginx/Dockerfile`](../deploy/nginx/Dockerfile) 仓库内构建（基座固定 `nginx:1.30.6-alpine@sha256:97036422…`，Alpine 3.24.2）+ 构建期 `apk upgrade`」路径（P-4 的结构性解法；纯 digest pin 吃不到补丁）。**滚动机制（2026-10-16 修复）**：Dependabot `docker` 规则（`/deploy/nginx` 与 `/apps/server`，月度）在上游基座 digest 变化时提 PR → CI 重新构建（`apk upgrade` 拿到构建时的补丁）+ Trivy 扫描构建产物。**此前 `docker` 规则的 directory 写成 `/`（仓库根无 Dockerfile），该规则从未生效**，已由 `TestDependabotDockerCoversEveryDockerfile` 钉住。**仍开放的缺口**：Dependabot 只在 digest 变化时开 PR，**没有定时重建**——Alpine 索引在 digest 不变时出的新补丁，要等下一次 push 才会被构建拿到。**复核动作**（人工兜底）：`docker pull nginx:<newest>-alpine` + `docker run --rm <img> apk info -v \| grep zlib`；**即使上游某天已带 r1 也不能去掉 `apk upgrade`**（后续 CVE 同理）。**到期硬约束**：Alpine 3.24 于 **2028-06-01 EOL**，届时必须抬到下一个受支持分支并重解 digest。退化由门禁 `TestRuntimeStageUpgradesAlpinePackages` / `TestNginxEdgeDockerfilePinsBaseAndAppliesPatches` 挡住；技术与口径在 [`threat-model.md`](threat-model.md) §5 / §5.4 |
| 92 | 大对象迁移缺真实压测：直传路径的内存收益未实测、5TiB 量级无端到端验证 | [ADR-014](decisions/0014-stream-copy-no-part-buffer.md) 落地时的自留验证缺口 | ⬜ | **两项都是「已实现但未实测」，不是已知缺陷**——[ADR-014](decisions/0014-stream-copy-no-part-buffer.md) 的结论靠设计推断与结构化测试支撑，尚无量化证据：① **内存收益未实测**：无段缓冲直传（`service.directMultipartCopy`）不再分配整段缓冲，这是代码层面可读的事实，但**没有实测 RSS 数字**——峰值内存与「在飞复制数 × 传输层缓冲」的实际关系未量化，[`PERFORMANCE.md`](PERFORMANCE.md) 也未加对应基准。② **大文件量级无端到端验证**：CI 无法承载 5TiB 对象，边界靠注入小段长覆盖（`stream_copy_direct_test.go` / `copy_large_test.go`），真实对端验证只有 12MiB 级（`s3wrap/e2e_multipart_copy_test.go`）——**数百 GB 以上对象的真实迁移从未跑过**，段长自适应放大到 5GiB 后的行为（超时、断连重试、源端压力）因此未经检验。闭环方式：按 [`PERFORMANCE.md`](PERFORMANCE.md) 补基准与内存实测；用真实大对象跑一次跨 endpoint 迁移与一次同端点 `UploadPartCopy`，记录耗时、峰值 RSS 与 Range 重读次数。**在此之前，不要把「上限已提到 5TiB」当作已验证的容量承诺** |
**当前存量**：§一 **#25（⛔ 外部阻塞）** · §二 **#63（➖ 已决策）· #91（➖ 已决策 / 维护期义务）**；产品功能缺陷为零、**仓内技术债归零**
（#91 是已决策方案的**维护期义务**、不是缺陷）。
