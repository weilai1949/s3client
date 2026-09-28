# s3clinet 已知问题（Known Issues）

> 本文件是**缺陷 / 外部阻塞 / 技术债**的**唯一来源**，只收录**尚未闭环**的项。
> **功能候选与版本规划不在本文件**——见 [`roadmap.md`](roadmap.md) §三（战略层唯一来源）；
> 已修复 / 已完成台账见 [`features.md`](features.md)；发版历史见 [`CHANGELOG.md`](../CHANGELOG.md)。
> 最近一轮综合评估（2026-09-16，五维度：代码质量 / 漏洞 / 死代码 / 降级 / 自我迭代）见
> [`archive/assessment.md`](archive/assessment.md)。
>
> 状态图例：⬜ 待处理 · ⏳ 处理中（仅列剩余工作） · ➖ 已决策（不做 / 维持现状） · ⛔ 外部阻塞（外部凭证未获取等非代码工作）
>
> **两源分工（2026-09-24 起）**：本文件收**问题**（缺陷 / 阻塞 / 技术债）；
> [`roadmap.md`](roadmap.md) 收**方向**（版本规划 / 功能候选）。同一编号只在一个文件里是「当前条目」，
> 跨文件引用必须带前缀：`KNOWN_ISSUES #N` / `ROADMAP §三 #N`。
>
> **编号约定**：编号保持稳定、不因条目移除而重排——`apps/server/` 代码注释与历史提交仍以
> `KNOWN_ISSUES #N` 引用本清单（2026-09-24 迁移前写作 `todolist #N`，两者同指），重排会使这些引用失真。
> 已闭环移除的编号：#1–#24 / #26–#39 / #41 / #45 / #46；
> **从未启用（保留空号）**：#40 / #42 / #43 / #44——补登记时跳号，为保持既有编号稳定而**不回收**
> （回收会让历史提交里的 `#N` 指向不同条目）。
> **2026-09-24 迁出**：#47–#59 为 ROADMAP 派生的**功能候选（非问题）**，唯一来源改为
> [`roadmap.md`](roadmap.md) §三 3.2，本文件不再收录。故本清单编号不连续属预期，不是漏登记。
>
> 最后更新：2026-09-24

## 目录

- [一、外部阻塞](#一外部阻塞)
- [二、技术债 / 缺陷](#二技术债--缺陷)
- [三、分类归零凭证](#三分类归零凭证)
- [四、编号台账](#四编号台账)

---

## 一、外部阻塞

> 非代码工作，依赖外部凭证 / 第三方通道；代码侧已无剩余工作，故**不占仓内排期**。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 25 | 桌面端分发与签名 | ROADMAP §三 #1 / 长期 | ⛔ | **外部阻塞**：Windows 代码签名证书 / Apple Developer ID + 公证是**外部凭证**（roadmap E6，⬜ 未获取），未获取前无法完成签名 / 公证，产物被 SmartScreen / Gatekeeper 拦截且无自动更新通道（roadmap R5）。代码侧无剩余工作（发布链已收口）；未签名产物以 `SHA256SUMS` + 手动放行说明过渡（[deployment.md](deployment.md) §5） |

## 二、技术债 / 缺陷

> 与代码 / 仓库硬约束直接相关、可在仓内闭环的项。

| # | 项 | 来源 | 状态 | 说明 |
|---|----|------|------|------|
| 60 | 4 个超 1000 行的前端测试文件拆分 | AGENTS.md 单文件约束 | ⬜ | `src/api.test.ts`（1928 行）、`src/components/MigratePanel.test.ts`（1210 行）、`src/composables/useObjectActions.test.ts`（1118 行）、`src/composables/useObjectBrowser.test.ts`（1040 行）均超 AGENTS.md「单文件不超过约 1000 行」约束（Go 侧同款 2 文件已于 2026-09-24 拆分完成，证据见 [`CHANGELOG.md`](../CHANGELOG.md)）。拆分只动文件归属、不改断言；会牵动登记数——[`features.md`](features.md) §Z 等处「66 文件 / 1043 例」为时点实测值，拆分后逐处收口 |
| 61 | `SameEndpoint` 精确判定未纳入 `useSSL` | code-review-2026-09-24 Nit / `service/migrate.go` `normalizeForCompare` 注释 | ⬜ | 两侧**裸端点**（无 scheme）而账号 `UseSSL` 不同时，会被判为同端而走 `CopyObject`（写到错误 scheme）或被判异端而走流式（慢但正确）。现有归一化是对称的（裸端点恒按 http 互比，无单边错配），且失败方向偏安全；精确判定需把 `useSSL` 纳入 `SameEndpoint` 签名并改 handler/service 全部调用点（跨包契约变更），口径说明与钉死用例见 `service/migrate.go` / `service/gaps_test.go` |
| 62 | 批量删除编排仍留在 `handler`（同类批量编排在 `service`） | code-review-2026-09-24 Nit（**本轮未完成**） | ⬜ | 批量编排的同类已在 `internal/service`（`RunBatch` / `CopyKeys`），删除族却仍写在 handler：`handler/objects.go` 的 `deleteObjects` / `deletePrefix` / `deletePrefixAsync`（含 `deleteCounts` 计数与前缀递归分片）、`handler/copy.go:177` 的 `copyKeysThenDelete`（先复制后删除）。后果是 `handler → service` 分层口径不一、`objects.go` 持续膨胀、编排逻辑无法脱离 HTTP 层单测。下沉需先把响应形状（`deleteCounts` / `deletePrefixResult`）抽离 `http.ResponseWriter`，再把编排与测试搬到 `service`——**纯重构、不改外部可见行为**，可独立成 PR。审查报告见 [code-review-2026-09-24.md](code-review-2026-09-24.md) 「处置明细」B18 |
| 63 | 流式复制单对象 640GB 上限（64MB × 10000 段） | code-review-2026-09-24 Nit（刻意取舍） | ➖ | **已决策维持现状**：10000 段是 S3 协议上限，按比例放大分段缓冲会突破容器 512MB 内存预算（`docker-compose*.yml` 的 `deploy.resources.limits.memory`，一块分段缓冲即 64MB）。超出上限的对象在段号耗尽前被**明确拒绝并 abort**，绝不静默截断。口径与内存账写在 `service/stream_copy.go` 注释（段号在**上传前**判定，不误杀第 10000 段的合法对象），默认值由测试钉住；如将来要放宽，先评估内存预算再动 |

## 三、分类归零凭证

> 迁移前本文件按六类收录待办。以下**原分类当前均无开放项**——保留本表是为了让旧编号
> （`todolist #N` / `KNOWN_ISSUES #N`）仍可回溯到闭环证据，**不是待办**；
> 各分类里属于「功能候选」的条目已按两源分工迁往 [roadmap.md](roadmap.md) §三 3.2。

| 原分类 | 现状 | 闭环证据 / 迁出去向 |
|--------|------|---------------------|
| 功能 / 架构 | 无开放项 | 迁出：#47–#52 / #58 / #59 → roadmap §三 3.2 |
| API / 契约 | 无开放项 | 迁出：#53 → roadmap §三 3.2。#26 / #27 / #28 已于 2026-09-22 闭环（[features.md](features.md) §T / §W）；请求体字段、query 参数、类型 / required / 枚举语义、共享 schema 与端点级响应均有机械门禁，断言范围写在各自文件头注释 |
| 代码质量 / 死代码 | 无开放项 | 迁出：#57 → roadmap §三 3.2。#11 / #12 已闭环；#28 的残留范围已随自由体全量收敛（[features.md](features.md) §W） |
| 安全 / 供应链 | 无开放项 | 迁出：#55 / #56 → roadmap §三 3.2。#16 / #17 已闭环；#18 于 2026-09-23 复审维持现状并移出（决策见 [threat-model.md](threat-model.md) §6.2）；#29–#34 于 2026-09-20 闭环（[features.md](features.md) §T）；S1 / S2 于 2026-09-19 闭环 |
| 可靠性 / 可观测性 | 无开放项 | 迁出：#54 → roadmap §三 3.2。#21 / #22 / #23 已闭环；#35 / #36 / #38 / #39 于 2026-09-22 闭环（[features.md](features.md) §U）；#37 于 2026-09-22 闭环（[features.md](features.md) §V） |
| 文档失真 | 无开放项 | D2 / D3 / D6 / D8 / D9 随 2026-09-19 文档同步修正（[features.md](features.md) §S）；D1 / D4 / D5 / D10（对应 #35 / #45 / #41 / #46）于 2026-09-22 闭环（[features.md](features.md) §U） |

## 四、编号台账

> 历史提交与代码注释里的 `todolist #N` / `KNOWN_ISSUES #N` 均可按下表回溯；**编号不重排、不回收**。

| 编号 | 状态 | 说明 |
|------|------|------|
| #1–#24 | 已闭环移除 | 证据见 [features.md](features.md) 与 [threat-model.md](threat-model.md) §6.2 |
| #25 | **开放** ⛔ | 外部阻塞，见 §一 |
| #26–#39 | 已闭环移除 | 2026-09-19 补登记，2026-09-20 / 2026-09-22 分批闭环（[features.md](features.md) §S–§W） |
| #40 / #42 / #43 / #44 | 从未启用（保留空号） | 补登记时跳号；回收会让历史提交里的 `#N` 指向不同条目 |
| #41 / #45 / #46 | 已闭环移除 | 对应文档失真 D1 / D4 / D5 / D10，2026-09-22 闭环（[features.md](features.md) §U） |
| #47–#59 | **已迁出** | 功能候选（非问题），唯一来源改为 [roadmap.md](roadmap.md) §三 3.2 |
| #60 | **开放** ⬜ | 技术债，见 §二 |
| #61 | **开放** ⬜ | 技术债（`SameEndpoint` 未纳入 `useSSL`），见 §二 |
| #62 | **开放** ⬜ | 技术债（批量删除编排未下沉 `service`，2026-09-24 审查 Nit 本轮未完成），见 §二 |
| #63 | **已决策** ➖ | 已知限制（流式复制单对象 640GB 上限，维持现状），见 §二 |

---

> **迁移说明**：本文件原为 `docs/todolist.md`，2026-09-24 以 `git mv` 更名（保留重命名历史），
> 并同步按「问题 / 方向」两源分工重组：功能候选 #47–#59 迁至 [roadmap.md](roadmap.md) §三 3.2，
> 本文件只保留缺陷 / 外部阻塞 / 技术债。
