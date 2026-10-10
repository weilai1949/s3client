# 文档归档（Archive）

> 本目录存放**时点性文档**的历史快照：它们记录某次评估 / 审查 / 迁移在**当时**的结论，
> 结论被后续工作取代后即**冻结**，不再随代码演进更新。归档 = 冻结——**不删除、不回写、
> 不改写历史结论**；后续处置写进 [`FEATURES.md`](../FEATURES.md) 与
> [`CHANGELOG.md`](../../CHANGELOG.md)，本目录只保留原文。
>
> 命名与存放约定见 [`DEVELOPMENT.md`](../DEVELOPMENT.md) §4；当前**已归档 8 份**（见下表）。
> **归档触发规则**：时点性文档在引用收敛后即按下方「归档操作」`git mv` 归档；**已填写的事故复盘**
> 在事件闭环、行动项登记完成后同样按「四步」冻结——**首例即当时唯一待归档项，已于 2026-09-30 执行**
> （[`incident-20260916-presign-empty-url.md`](incident-20260916-presign-empty-url.md)，即
> [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md) 的首份已填写实例）；
> **批次交接快照**（`handoff-YYYYMMDD.md`）在批次全部收口后同 PR 冻结——首例
> [`handoff-20260930.md`](handoff-20260930.md) 已于 2026-10-08 执行，
> [`handoff-20261010.md`](handoff-20261010.md) 已于 2026-10-10 执行（收口提交 `dd9ab88` 之后）；
> **代码评审快照**（`code-review-YYYY-MM-DD.md`）在处置状态表**全部闭环**后同 PR 冻结——
> 先例 [`code-review-2026-09-24.md`](code-review-2026-09-24.md) 已于 2026-09-28 执行，
> [`code-review-2026-10-09.md`](code-review-2026-10-09.md) 已于 2026-10-10 执行。
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
| [handoff-20261010.md](handoff-20261010.md) | `docs/handoff-20261010.md` | 2026-10-10 | **第二份批次交接快照**（2026-10-10 全 docs 通读问题清单收口批次的断点记录）：该批 24 组问题已于当日全部执行完毕并提交 `dd9ab88`（唯一 Go 生产变更 = OpenAPI 通用状态码逐 operation 接线，另含三道门禁扩面），快照冻结「执行到哪、还剩什么」的原文（§5 未做清单 14 项 + 5 条踩坑），正文不回写；台账见 [`FEATURES.md`](../FEATURES.md) §CI，当前待办一律以 [`ROADMAP.md`](../ROADMAP.md) §三 3.2 与 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 为准 |

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
