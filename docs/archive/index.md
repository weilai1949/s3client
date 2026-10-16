# 文档归档（Archive）

> 本目录存放**时点性文档**的历史快照：它们记录某次评估 / 审查 / 迁移在**当时**的结论，
> 结论被后续工作取代后即**冻结**，不再随代码演进更新。归档 = 冻结——**不删除、不回写、
> 不改写历史结论**；后续处置写进 [`FEATURES.md`](../FEATURES.md) 与
> [`CHANGELOG.md`](../../CHANGELOG.md)，本目录只保留原文。
>
> 命名与存放约定见 [`DEVELOPMENT.md`](../DEVELOPMENT.md) §4；当前**已归档 16 份**（见下表）。
> **归档触发规则**：时点性文档在引用收敛后即按下方「归档操作」`git mv` 归档；**已填写的事故复盘**
> 在事件闭环、行动项登记完成后同样按「四步」冻结——**首例即当时唯一待归档项，已于 2026-09-30 执行**
> （[`incident-20260916-presign-empty-url.md`](incident-20260916-presign-empty-url.md)，即
> [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md) 的首份已填写实例）；
> **批次交接快照**（`handoff-YYYYMMDD.md`）在批次全部收口后同 PR 冻结——首例
> [`handoff-20260930.md`](handoff-20260930.md) 已于 2026-10-08 执行，
> [`handoff-20261010.md`](handoff-20261010.md) 已于 2026-10-10 执行（批次提交 `dd9ab88`，
> 其 §5 未做清单 14 项随后收口于 `55102a9`，逐条证据见 [`FEATURES.md`](../FEATURES.md) §CJ）；
> **代码评审快照**（`code-review-YYYY-MM-DD.md`）在处置状态表**全部闭环**后同 PR 冻结——
> 先例 [`code-review-2026-09-24.md`](code-review-2026-09-24.md) 已于 2026-09-28 执行，
> [`code-review-2026-10-09.md`](code-review-2026-10-09.md) 已于 2026-10-10 执行。
> [`code-review-2026-10-10.md`](code-review-2026-10-10.md) 亦已于 2026-10-10 执行（结论 **Approve**：无 Critical / Required，三道非阻断 Nit #84–#86 同日闭环）。
> [`code-review-2026-10-14.md`](code-review-2026-10-14.md) 已于 2026-10-14 执行（结论 **Approve**：F1–F7 + A1–A2 + C2 同日闭环、C1 有意维持，三项可维护性 O1–O3 转登记 `KNOWN_ISSUES` #88–#90；**#88–#90 已于 2026-10-15 在有测试环境闭环**，门禁全量实跑全绿，见 [`../FEATURES.md`](../FEATURES.md) §CR）。
> [`code-review-2026-10-15.md`](code-review-2026-10-15.md) 已于 2026-10-16 执行（处置状态表**全部闭环** + **R3a 人类专家复核通过**，见 [`../FEATURES.md`](../FEATURES.md) §CS；⚠️ 其「Approve」结论**已被同日 [`pentest-2026-10-15.md`](pentest-2026-10-15.md) 证伪**，只能作评审时点事实引用）。
> **子树评审快照**（`<subtree>-review-YYYY-MM-DD.md`）在条目全部闭环后同 PR 冻结——首例
> [`web-review-2026-10-12.md`](web-review-2026-10-12.md) / [`desktop-review-2026-10-12.md`](desktop-review-2026-10-12.md) /
> [`server-review-2026-10-12.md`](server-review-2026-10-12.md) 均已同日执行（2026-10-12）。
> **文档复盘快照**（`docs-review-YYYY-MM-DD.md`）在建议项被处置或移交后冻结——首例
> [`docs-review-2026-10-12.md`](docs-review-2026-10-12.md) 已于 2026-10-12 执行
> （其沿用的 14 项建议经复核已被更早的 2026-10-10 批次全部闭环，见 [`FEATURES.md`](../FEATURES.md) §CJ）。
> **渗透测试 / 对抗性验证快照**（`pentest-YYYY-MM-DD.md`）在发现项全部闭环后同 PR 冻结——首例
> [`pentest-2026-10-15.md`](pentest-2026-10-15.md) 已于 2026-10-16 执行（P-1..P-15 全部闭环，逐条处置见
> [`FEATURES.md`](../FEATURES.md) §CV / §CW / §CX）。
> 当前**无待归档例外**。
>
> ⚠️ **冻结件里的旧写法**：归档于 2026-10-08 产品名改名（`s3clinet → s3client`）**之前**的快照
> （`review-2026-09-19.md` / `code-review-2026-09-24.md` / `code-review-summary.md` / `assessment.md`）
> 内的可复现命令与项目名保留当时写法——其中 `S3CLINET_E2E` / `S3CLINET_ENDPOINT` 等
> **环境变量名已改** `S3CLIENT_*`，照抄执行会因变量不匹配而**静默 `t.Skip`**（看似绿、实为 0 用例）；
> 复现时请替换为 `S3CLIENT_*`。正文按冻结纪律不回写。

## 归档清单

| 归档文件 | 原路径 | 归档日期 | 冻结的结论 |
|---|---|---|---|
| [review-2026-09-19.md](review-2026-09-19.md) | `docs/review-2026-09-19.md` | 2026-09-23 | 2026-09-19 分支整体状态审查（七维度实跑复核）：P0 / P1 / P2、S1–S6、R2–R10、D1–D10 全部闭环 |
| [assessment.md](assessment.md) | `docs/assessment.md` | 2026-09-24 | 2026-09-16 五维度综合评估（代码质量 / 漏洞 / 死代码 / 降级 / 自我迭代）：评分 78/100，P0 / P1 / P2 与路线图 v1.0.0–v1.1.0 收口证据见 [`FEATURES.md`](../FEATURES.md) §H 起 |
| [code-review-2026-09-24.md](code-review-2026-09-24.md) | `docs/code-review-2026-09-24.md` | 2026-09-28 | 2026-09-24 全仓代码审查（六路五轴）：2 Critical + 20 Required **全部修复**，Nit 35 项中 31 闭环 / 3 转登记（→ `KNOWN_ISSUES` #61 #62 #63）/ 1 判定不成立；转登记的 #61 #62 已于 2026-09-28 闭环（见 [`FEATURES.md`](../FEATURES.md) §AB） |
| [code-review-summary.md](code-review-summary.md) | `docs/code-review-summary.md` | 2026-09-28 | 代码审查总结与改进建议（总体评分、质量指标、待解决项、改进建议）；头部「状态更新」活块 ①–⑥ 冻结 2026-09-24 → 2026-09-28 的收口事实，正文数字均为**审查时点值**（如「1042 用例 / 66 文件」）。其「⚠️ 待解决的技术问题」在归档前已清空到只剩 #25——该清单是在办事项、已闭环项直接移除 |
| [incident-20260916-presign-empty-url.md](incident-20260916-presign-empty-url.md) | `docs/incident-20260916-presign-empty-url.md` | 2026-09-30 | **首份已填写的事故复盘**（near-miss 示例，按 [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md) §1–§10 填写）：`b3b287c`（2026-09-05 为达 100% 覆盖率删除被误判「不可达」的预签名错误分支）→ 签名失败回 `200 {"url":""}`；2026-09-16 评估实测发现（`ASSESSMENT` L1 / `KNOWN_ISSUES` #23），2026-09-17 `5954bfa` 统一 `writePresignResult` + 源码门禁 `TestPresignErrorsNotSwallowed` 闭环，2026-09-19 `0fbd560` 收口 #23 余项；正文字段全挂 git 取证，**未编造运行期数据** |
| [handoff-20260930.md](handoff-20260930.md) | `docs/handoff-20260930.md` | 2026-10-08 | **首份批次交接快照**（2026-09-30 中断优化批次的断点记录）：所记四条目 `ROADMAP` #19 / `KNOWN_ISSUES` #70 / `ROADMAP` #18 / #17 已于 2026-10-01 **全部收口**（§2 表），快照冻结为「执行到哪、还剩什么」的历史原文（含当时实跑的门禁数字与踩坑记录）；当前待办一律以 [`ROADMAP.md`](../ROADMAP.md) §三 3.2 与 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 为准 |
| [code-review-2026-10-09.md](code-review-2026-10-09.md) | `docs/code-review-2026-10-09.md` | 2026-10-10 | **2026-10-09 全仓五轴代码评审**（结论 `Request changes` → 收口）：C1 / C2（CI 两个平台的 Go job 构造性红灯——缺 `apps/web/node_modules` 前置、浅克隆致 `changelog_tag` 必红）与 R1–R10 **全部修复**，O1–O12 **全部闭环**（转登记 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) #72–#83，逐条证据见 [`FEATURES.md`](../FEATURES.md) §BZ–§CD）；§1 门禁实测数字为**评审时点快照**，正文不回写。**读正文注意**：其 §8 末条停在同日**中间态**「仅剩 O1 / O3–O8 开放」——终态（全部闭环）以本行与 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 编号台账为准 |
| [handoff-20261010.md](handoff-20261010.md) | `docs/handoff-20261010.md` | 2026-10-10 | **第二份批次交接快照**（2026-10-10 全 docs 通读问题清单收口批次的断点记录）：该批 24 组问题已于当日全部执行完毕并提交 `dd9ab88`（唯一 Go 生产变更 = OpenAPI 通用状态码逐 operation 接线，另含三道门禁扩面），快照冻结「执行到哪、还剩什么」的原文（§5 未做清单 14 项 + 5 条踩坑），正文不回写；其 §5 的 14 项已于 2026-10-10 **全部收口**（提交 `55102a9`：5 道新门禁 + 三份文档结构瘦身 + 契约/体例修复 + 覆盖门禁由红转绿，见 [`FEATURES.md`](../FEATURES.md) §CJ）；§CI 台账与 §CJ 收口台账并列，当前待办一律以 [`ROADMAP.md`](../ROADMAP.md) §三 3.2 与 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 为准 |
| [web-review-2026-10-12.md](web-review-2026-10-12.md) | `docs/web-review-2026-10-12.md` | 2026-10-12 | **2026-10-12 apps/web 子树评审快照**：页面 UI / 交互高 / 中 / 低三档 **9 批全部实施**（H1 任务中心 / H2 拖拽上传 / H3 软刷新 / M1 过滤防抖高亮 / M2 键盘行导航 / M3 Toast 三改 / L1 偏好持久化 / L2 移动视口扫描）+ 新增扫描**实测抓到对比度缺陷 D1**（`.sc-badge` 4.39:1 → 新增 `--badge-text` token）同批闭环；用户可见行为已搬进 [`user-guide.md`](../user-guide.md)、台账 [`FEATURES.md`](../FEATURES.md) §CL、发版草稿 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`；§9 如实披露未跑项（全量 `pnpm e2e`、`make e2e-real`、真机 / 触摸）。**读正文注意**：文首「未同步写入 …」是生成时点事实，其偏离已解除（见冻结横幅） |
| [desktop-review-2026-10-12.md](desktop-review-2026-10-12.md) | `docs/desktop-review-2026-10-12.md` | 2026-10-12 | **2026-10-12 apps/desktop 子树评审快照**：Tauri 2 无 IPC 桌面壳，1 项死代码 + 6 项优化 + 1 道防复发门禁**全部闭环**——删死依赖 `serde` / `serde_json`(+O1 release profile 体积优化 / O2 收窄 `bundle.targets` 至 `nsis`·`deb`·`dmg` / O3 两套 CI 的 tauri-cli pin 至 `2.12.2` / O4 `make desktop-dev` 补 web 依赖 / O5 打包元数据+窗口 `label`+CSP 收紧 / O6 tauri `2.11.5→2.12.3`·tauri-build `2.6.3→2.7.2`·CLI `2.11.4→2.12.2`（连带 audit 告警 **7→2 条**、许可证清单 Rust **428→416**）/ G1 `RUSTFLAGS=-D unused_crate_dependencies` 防复发门禁)。证据见 [`FEATURES.md`](../FEATURES.md) §CM、发版记录 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]`。**读正文注意**：O5 的 **CSP 收紧**按 [`AI_POLICY.md`](../AI_POLICY.md) §3 须**人类专家复核**，改动已备、确认前不得推送（见冻结横幅） |
| [server-review-2026-10-12.md](server-review-2026-10-12.md) | `docs/server-review-2026-10-12.md` | 2026-10-12 | **2026-10-12 apps/server 子树评审快照**：结论**无结构性 / 算法级优化缺口**——S1–S5 五项微优化落地（有界 256 KiB 复用缓冲 + 写 deadline 按 30s 节流 / 限流器 16 分片并补总量硬上限 / S3 调用指标桶改 atomic、Calls 由桶求和 / CSP 整串预计算 / `/api/metrics` 去 STW 且整包一次写出）、S6 维持现状、S0 跨子树 i18n 门禁由同日 web 批次解除；证据见 [`FEATURES.md`](../FEATURES.md) §CN 与 [`PERFORMANCE.md`](../PERFORMANCE.md) §3.5 |
| [code-review-2026-10-10.md](code-review-2026-10-10.md) | `docs/code-review-2026-10-10.md` | 2026-10-10 | **2026-10-10 全仓五轴代码质量评审**（结论 **Approve**）：机械化门禁独立复跑全绿（10/10 包、覆盖率 100.0% 零未覆盖块、`golangci-lint` 0 issues、`pnpm test` 83 文件 1292 例）+ 逐文件五轴复核 2026-10-09 闭环批次 delta，**未发现 Critical / Required**；登记的**三道非阻断 Nit #84–#86 同日全部闭环**（`redactEndpoint()` 凭据脱敏 / 指标改名 `s3c_volume_inode_capacity` / 注释去重），证据见 [`FEATURES.md`](../FEATURES.md) §CK；§1 门禁实测数字为**评审时点快照**，正文不回写 |
| [code-review-2026-10-14.md](code-review-2026-10-14.md) | `docs/code-review-2026-10-14.md` | 2026-10-14 | **2026-10-14 全仓代码质量评审**（结论 **Approve**）：F1–F7（前端 `loadBuckets`/`load` 共用计数桶竞态 · `s3wrap` `ListParts`/`PurgeObject` 无界列举 · 4 道门禁「断言被文案/词满足」假阴性 · 孤儿 `.PHONY`）+ A1–A2（图标按钮 / 进度条可访问名）+ C2（六个 CI / compose 基础镜像 digest pin + 新门禁 `TestCIBaseImagesAreDigestPinned`）**同日闭环**；C1（Tauri `connect-src`）复核**有意维持**并记录取舍；三项可维护性发现 **O1–O3 转登记** [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) #88–#90。§1 门禁实测为**评审时点快照**且**因环境离线未含「全绿」结论**（诚实披露），正文不回写 |
| [docs-review-2026-10-12.md](docs-review-2026-10-12.md) | `docs/docs-review-2026-10-12.md` | 2026-10-12 | **2026-10-12 docs 全量通读复盘快照**（54 篇清单与体量 / 三层入口 / 唯一来源与机械门禁的交叉观察）；其 §7 建议项（沿用 [`handoff-20261010.md`](handoff-20261010.md) §5 的 14 项）经复核**已于更早的 2026-10-10 批次全部闭环**（[`FEATURES.md`](../FEATURES.md) §CJ）——本文采信该清单时未逐项复核，相关「仍存问题」表述以 §CJ 为准（见冻结横幅），正文不回写 |
| [pentest-2026-10-15.md](pentest-2026-10-15.md) | `docs/pentest-2026-10-15.md` | 2026-10-16 | **2026-10-15 全仓对抗性渗透测试快照**：2 条 High（最小权限作用域「零引用即放行」fail-open / 授权解析器 ≠ 执行解析器）+ 13 条中低危与信息项（P-3..P-15）**全部闭环**；含 60+ 项负结果清单、对抗性复核与未验证项披露。正文保留**发现时点**的原始利用链与修复建议，不回写；逐条处置见 [`FEATURES.md`](../FEATURES.md) §CV（P-1/P-2）· §CW（账号维度残留）· §CX（P-3..P-15），SAST 台账刷新见 [`../threat-model.md`](../threat-model.md) §7 |
| [code-review-2026-10-15.md](code-review-2026-10-15.md) | `docs/code-review-2026-10-15.md` | 2026-10-16 | **2026-10-15 全仓安全评审快照**（结论 **Approve**：未发现高危 / 中危；R1–R5 为低危 / 观察项）：处置状态表 **R1 / R4 / R5 · R2a / R2b / R3a · 配置边界（CSP 启动期校验）· 限流可观测 · R6–R10（含两项低风险加固）全部闭环**；**R3a 的人类专家复核于 2026-10-16 通过**（复核人 = 仓库负责人（用户）；依据 = `middleware.go:32` CSP 单源字面量 + `TestServerCSPSharesTauriDirectives` + `make e2e-real` 5/5 PASS），见 [`FEATURES.md`](../FEATURES.md) §CS。⚠️ **其「Approve」结论已被同日 [`pentest-2026-10-15.md`](pentest-2026-10-15.md) 证伪**（P-1 / P-2 两条 High）——本件只作「评审时点事实」引用，**不得**当作系统无高危的证据；正文（含 §4 第 4 条与 §5 未复跑披露）按冻结纪律原文保留、不回写 |

> `docs/` 下（本目录之外）的文档均为**活跃文档**，须随代码同步维护
> （§4 文档同步门禁）。本目录约定随归档实践持续生效。

## 什么该归档

- **时点性快照**：综合评估、分支 / 版本审查、迁移对照表、已完成的一次性方案——结论绑定在某个
  commit 或日期上，不随后续代码变化而更新。
- **已填写的事故复盘**：`incident-YYYYMMDD-<短名>.md`（空模板见
  [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md)）——事件闭环、行动项已登记进
  [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 后即冻结；正文是**当时**的取证与判断，不随后续修复回写。
- **已被取代、且不再作为当前事实来源**的文档：仍有追溯价值（证据链、当时的判断依据），
  但其数字与结论已不代表现状。
- 归档后如需引用，**只作为历史证据**引用；当前事实一律以活跃文档为准。

## 什么不该归档

- **活跃 SSOT**：[`api.md`](../api.md) / [`architecture.md`](../architecture.md) /
  [`errors.md`](../errors.md) / [`FEATURES.md`](../FEATURES.md) / [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) /
  [`ROADMAP.md`](../ROADMAP.md) / [`DEPLOYMENT.md`](../DEPLOYMENT.md) /
  [`DEVELOPMENT.md`](../DEVELOPMENT.md) / [`threat-model.md`](../threat-model.md)，
  以及根目录的 [`README.md`](../../README.md) / [`CHANGELOG.md`](../../CHANGELOG.md) /
  [`AGENTS.md`](../../AGENTS.md)——这些必须随代码同步更新。
- **ADR**：见 [`decisions/`](../decisions/index.md)。ADR 同样**不删除、不归档**——决策被取代时
  **新写一篇** ADR 引用旧篇，并把旧篇状态标为 `Superseded`，保留「当时为什么这么定」的原始理由。

## 归档操作

1. 用 `git mv` 移动（保留重命名历史，**不复制**、不删了重加）；
2. 修正全仓指向旧路径的引用（含源码注释里的 `docs/xxx.md` 路径），**链接不悬空**；
3. 在本文件「归档清单」补一行（归档文件 / 原路径 / 归档日期 / 冻结的结论）；
4. 在 [`CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]` 记一条 `变更`。
