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
> **编号约定**：编号保持稳定、不因条目移除而重排——`apps/server/` 代码注释与历史提交仍以
> `KNOWN_ISSUES #N` 引用本清单（2026-09-24 迁移前写作 `todolist #N`，两者同指），重排会使这些引用失真。
> **已闭环编号索引（2026-10-16 精简：闭环细节不再堆叠于本文件，按编号去 [`FEATURES.md`](FEATURES.md)
> 与 [`CHANGELOG.md`](../CHANGELOG.md) 检索证据）**：
> `#1–#24` / `#26–#39` / `#41` / `#45` / `#46` / `#60–#62` / `#64–#90`；
> **从未启用（保留空号）**：`#40` / `#42` / `#43` / `#44`——补登记时跳号，为保持既有编号稳定而**不回收**
> （回收会让历史提交里的 `#N` 指向不同条目）。
> **2026-09-24 迁出**：`#47–#59` 为 ROADMAP 派生的**功能候选（非问题）**，唯一来源改为
> [`ROADMAP.md`](ROADMAP.md) §三 3.2，本文件不再收录。故本清单编号不连续属预期，不是漏登记。
>
> 最后更新：2026-10-16（**2026-10-15 全仓对抗性渗透测试 P-1..P-15 全部闭环**：最小权限作用域两条 High
> 与端点凭据 / CSP / CORS / 探针限流 / 密钥注入 / 供应链等 13 项中低危全部处置；同时本文件按
> 「只留未闭环项」精简，改为上面的编号索引 + 下表两个未闭环项。逐条证据见
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
| 63 | 流式复制单对象 640GB 上限（64MB × 10000 段） | code-review-2026-09-24 Nit（刻意取舍） | ➖ | **已决策维持现状**（2026-09-28 复核并补齐证据）：10000 段是 S3 协议上限，按比例放大分段缓冲会突破容器 512MB 内存预算（`docker-compose.yml` / `docker-compose.prod.yml` 的 server 服务 `deploy.resources.limits.memory: 512M`，一块分段缓冲即 64MB）。超出上限的对象在段号耗尽前被**明确拒绝并 abort**，绝不静默截断。口径与内存账写在 `service/stream_copy.go` 注释（段号在**上传前**判定，不误杀第 10000 段的合法对象）；两个默认值分别由 `TestMultipartStreamCopyPartSizeIs64MB`（分段 64MB）与 `TestMaxMultipartPartsIsProtocolLimit`（段数 10000）钉住，边界行为由 `TestMultipartStreamCopyAcceptsExactlyMaxParts` / `TestMultipartStreamCopyRejectsPartOverLimit` / `TestMultipartStreamCopyByteCeiling` 覆盖。如将来要放宽，先评估内存预算再动。**2026-10-08 补登**：该限制此前只写在 `stream_copy.go` 注释 / 本文件 / [`OPERATIONS.md`](OPERATIONS.md) §性能表，**根 `README.md` 新增「已知限制」小节**把它推到用户可见面 |
| 91 | nginx 基座 digest 需按上游刷新节奏滚动更新 | 2026-10-16 渗透测试 P-4 处置的**维护期义务** | ➖ | **这不是「未闭环的缺陷」，而是已决策方案（仓库内构建边缘镜像）的维护期义务**（避免后来人误判成漏修）：截至 2026-10-16 实测**所有**官方 nginx alpine tag（1.28.3 / 1.29.8 / 1.30.6 / 1.31 / mainline / slim / unprivileged）仍 ship `zlib-1.3.2-r0`（CVE-2026-85091 HIGH），而修复版 `1.3.2-r1` 已发布——因此边缘镜像**必须**保持「[`deploy/nginx/Dockerfile`](../deploy/nginx/Dockerfile) 仓库内构建（基座固定 `nginx:1.30.6-alpine@sha256:97036422…`，Alpine 3.24.2）+ 构建期 `apk upgrade`」路径（P-4 的结构性解法；纯 digest pin 吃不到补丁）。**复核动作**：定期 `docker pull nginx:<newest>-alpine` + `docker run --rm <img> apk info -v \| grep zlib`；**即使上游某天已带 r1 也不能去掉 `apk upgrade`**（后续 CVE 同理）。**到期硬约束**：Alpine 3.24 于 **2028-06-01 EOL**，届时必须抬到下一个受支持分支并重解 digest。退化由门禁 `TestRuntimeStageUpgradesAlpinePackages` / `TestNginxEdgeDockerfilePinsBaseAndAppliesPatches` 挡住；技术与口径在 [`threat-model.md`](threat-model.md) §5 / §5.4 |

**当前存量**：§一 **#25（⛔ 外部阻塞）** · §二 **#63（➖ 已决策）· #91（➖ 已决策 / 维护期义务）**；产品功能缺陷为零、**仓内技术债归零**
（#91 是已决策方案的**维护期义务**、不是缺陷）。

---

> **迁移说明**：本文件原为 `docs/todolist.md`，2026-09-24 以 `git mv` 更名（保留重命名历史），
> 并同步按「问题 / 方向」两源分工重组：功能候选 #47–#59 迁至 [ROADMAP.md](ROADMAP.md) §三 3.2，
> 本文件只保留缺陷 / 外部阻塞 / 技术债。
> **2026-10-16 精简**：原「三、分类归零凭证」与「四、编号台账」两节、闭环批次索引表与历轮叙述一并移除
> （闭环细节与证据本就在 [`FEATURES.md`](FEATURES.md) / [`CHANGELOG.md`](../CHANGELOG.md)，此处只保留编号索引）——
> 编号**不重排、不回收**，代码注释与历史提交里的旧引用仍可按上面的索引回溯。
