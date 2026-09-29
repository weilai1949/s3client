# 项目治理（Governance）

> 本文件说明 s3clinet **当前实际如何运作**：谁做决定、改动怎么被审查、谁能发版、分歧怎么收场。
> 它**不描述理想组织**——本仓库目前是**单人维护 + 社区贡献**的项目（见 §1），因此这里如实写明
> 「哪些角色目前由同一个人承担」「哪些机制当前没有」以及将来怎么演进。
> 支持渠道见 [`SUPPORT.md`](SUPPORT.md)；行为规范见 [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)；
> 安全决策口径见 [`SECURITY.md`](SECURITY.md)。

## 1. 治理现状

| 项 | 现状 |
|---|---|
| 维护者 | **1 人**：[`@weilai1949`](https://github.com/weilai1949) |
| 决策机构 | **无委员会、无选举、无投票机制、无指导委员会（TSC）**，也不存在多维护者梯队 |
| 贡献来源 | 社区通过 Issue / PR 参与；维护者本人开发 |
| 分支保护 | 代码侧声明见 [`.github/CODEOWNERS`](CODEOWNERS) 头部说明：该文件生效前提是仓库分支保护里开启「Require review from Code Owners」，否则它只是**提示**而非强制。**分支保护开关本身在 GitHub 仓库设置里，仓内没有可审计的副本**——本文件不假装它一定开着 |
| 资金来源 | 无商业实体、无赞助协议、无付费支持（见 [`SUPPORT.md`](SUPPORT.md) §4） |
| 规则变更 | 本文件与 [`docs/ROADMAP.md`](../docs/ROADMAP.md)、[`AGENTS.md`](../AGENTS.md) 记录的规则，改动一律走 §3 的同一流程（PR + 人类复核）；治理结构本身要变（例如引入第二个维护者角色）须先写 ADR（§3.2）再改本文件 |

> **为什么现在不建委员会**：项目由单人长期维护、无商业实体、无资金与法务主体，设立委员会 /
> 选举 / 多维护者梯队**既无实际职能可分配，也会产生无人履行的空转角色**。将来若出现**行为主体**
> （例如：有第二位持续参与并承担审查的维护者、或出现需要代表项目对外发声的场景），再按
> §6 的路径引入角色与规则，届时应新写一篇 ADR（见 §3）并同步改本文件——而不是先立空架子。

## 2. 角色与权限

角色按「实际有人在做」列出；**同一人可同时持有多个角色**（当前维护者即全部持有者）。

| 角色 | 当前持有者 | 能做什么 | 不能做什么 |
|---|---|---|---|
| **贡献者（Contributor）** | 任何提交 Issue / PR 的人 | 提 Issue、提 PR、参与讨论、评审他人 PR（评论） | 不能合并、不能发版、不能改分支保护与仓库设置 |
| **审查者（Reviewer）** | [`@weilai1949`](https://github.com/weilai1949) | 审查 PR、要求修改、批准 | 无独立于维护者的否决权（见 §6） |
| **维护者（Maintainer）** | [`@weilai1949`](https://github.com/weilai1949) | 合并 PR、发版与打 tag、改仓库设置、处置行为准则事件、裁定分歧 | 受 [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) 与 [`docs/AI_POLICY.md`](../docs/AI_POLICY.md) 约束；对架构级变更须先有 ADR（§3） |

**CODEOWNERS 的作用**：`.github/CODEOWNERS` 声明「哪些路径的改动应由谁审查」。当前**全部条目都指向
同一位维护者**——这不是多维护者分工，而是把**敏感面显式点名**，让审查时不会漏看。它同时是
GitHub 侧「Require review from Code Owners」的输入。

被点名要求人类专家复核的敏感面（与 [`docs/AI_POLICY.md`](../docs/AI_POLICY.md) §3 权限矩阵一致）：

| 路径 | 为什么单独点名 |
|---|---|
| `/.github/SECURITY.md`、`/docs/threat-model.md` | 安全策略与威胁模型是安全口径的唯一来源 |
| `/apps/server/internal/store/` | 账号库与加密信封（`S3C2` / `S3C3`），错误改动会造成不可逆的数据不可解密 |
| `/apps/server/internal/s3wrap/` | AWS SDK 唯一边界（签名 / 预签名 / SSRF / 端点归一化） |
| `/apps/server/internal/handler/middleware.go` | 鉴权、CORS、限速、审计的公共入口 |
| `/.github/workflows/`、`/.gitlab-ci.yml`、`/apps/server/Dockerfile` | 供应链与流水线，改一侧必须同步另一侧（[`docs/DEVELOPMENT.md`](../docs/DEVELOPMENT.md) §3） |
| `/AGENTS.md`、`/llms.txt`、`/docs/AI_POLICY.md` | Agent 治理与仓库级硬约束 |

## 3. 决策流程

### 3.1 日常改动（缺陷修复、小功能、文档、依赖维护）

1. 在 Issue 里说明问题或需求（可复现、可验证；见 [`SUPPORT.md`](SUPPORT.md) §3）。
2. 从 `develop` 切分支（`feat/…`、`fix/…`）→ 按 TDD 开发 → 本地跑门禁
   （[`CONTRIBUTING.md`](CONTRIBUTING.md) §开发环境与提交规范、[`AGENTS.md`](../AGENTS.md) 提交前门禁）。
3. 提 PR；改动涉及 CODEOWNERS 点名的路径时，审查必须覆盖该路径。
4. **审查通过 + 门禁全绿**才合并；合并进 `develop`。
5. 每个改动必须同 PR 更新对应文档与 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`
   （对照表见 [`docs/DEVELOPMENT.md`](../docs/DEVELOPMENT.md) §4）——**文档未同步视为改动未完成**。

> **单人维护下的「审查」是什么**：当前审查者与 PR 作者是同一人，因此对外部贡献的审查是**真实审查**，
> 对维护者自己的改动则依靠**机械门禁 + 自审**兜底（Go `vet` / `golangci-lint` 0 issues / 覆盖率 100% /
> 契约与文档门禁 / RustFS 真对端 E2E / 真实后端浏览器联调，清单见 [`AGENTS.md`](../AGENTS.md)）。
> 这一点不粉饰：**不存在「第二双人类眼睛」**，所以宁可把可机械验证的部分做成门禁。
> 使用 AI 代理时按 [`docs/AI_POLICY.md`](../docs/AI_POLICY.md) 的权限矩阵与披露模板执行。

### 3.2 架构级变更（必须先有 ADR）

**架构级变更在动代码之前，先写 ADR**，落在 [`docs/decisions/`](../docs/decisions/index.md)
（格式：状态 / 日期 / 背景 / 决策 / 替代方案 / 后果，见该目录 `index.md` 的索引表）。

| 什么算架构级 | 例子（已归档决策） |
|---|---|
| 分层 / 模块边界 / 目录结构变化 | [`ADR-001`](../docs/decisions/0001-desktop-no-ipc.md) 桌面端 B/S、不用 Tauri IPC |
| 可用性语义取舍（失败时降级还是硬失败） | [`ADR-002`](../docs/decisions/0002-store-fail-closed.md) 存储不可用硬失败、不降级只读 |
| 安全策略默认值 | [`ADR-003`](../docs/decisions/0003-ssrf-private-allow.md) SSRF 默认放行私网 / 回环（含 `S3C_SSRF_DENY_PRIVATE` 加固开关） |
| 依赖面 / 供应链口径 | [`ADR-004`](../docs/decisions/0004-minimal-frontend-deps.md) 前端生产依赖仅保留 `vue` |
| **推翻既有决策** | 决策变化时**新写一篇 ADR** 引用旧篇并标 `Superseded`；ADR **不归档、不删除**（[`docs/DEVELOPMENT.md`](../docs/DEVELOPMENT.md) §4） |

ADR 本身也走 PR 审查；**未记录 ADR 的架构级改动不予合并**。改动若同时触及安全面或存储格式，
还需满足：安全面须人类专家复核（[`docs/AI_POLICY.md`](../docs/AI_POLICY.md) §3），
存储格式须保持既有账号库仍可解密（[`docs/ROADMAP.md`](../docs/ROADMAP.md) §5.2 E10、
[`docs/compatibility.md`](../docs/compatibility.md) §4）。

## 4. 发布权

| 动作 | 谁能做 | 怎么做 |
|---|---|---|
| 同步版本号 | **仅人类维护者** | `scripts/release-version.sh`（自动生成 `v1.0.0-YYYYMMDDHHmmss`，或显式指定 `v1.0.0` / `v1.0.0-rcN`） |
| 更新 `[Unreleased]` → 版本段 | **仅人类维护者** | [`CHANGELOG.md`](../CHANGELOG.md)（Keep a Changelog） |
| 打 tag / 触发发布 | **仅人类维护者** | 推送 `v*` tag 触发 [`.github/workflows/release-desktop.yml`](workflows/release-desktop.yml)，产物挂到 GitHub Release；流程见 [`docs/DEPLOYMENT.md`](../docs/DEPLOYMENT.md) §5 与 [`CONTRIBUTING.md`](CONTRIBUTING.md) §发布流程 |
| 桌面产物签名 / 公证 | **当前无人能做** | 缺外部凭证（代码签名证书 / Apple Developer ID，见 [`docs/ROADMAP.md`](../docs/ROADMAP.md) §5.2 E6），现状见 [`docs/compatibility.md`](../docs/compatibility.md) §8 |

> **代理与自动化不得自动发版**：AI 代理 / 机器人可准备版本号改动、可跑门禁、可开 PR，
> 但**不得打 tag、不得发布、不得合并**——[`docs/AI_POLICY.md`](../docs/AI_POLICY.md) §3 把
> 「发布版本 / 打 tag」列为**需确认**且由人类执行，§4 把 `release.create` 与 `pr.auto-merge`
> 一并按需确认 / 禁止处理。「需确认」的含义是：代理只能在本地工作区准备改动与证据，落地动作由人执行。

**发布的硬前提**：任一版本发布前质量门禁必须全绿、P0 未清零不得发布、里程碑验收标准须已满足
（[`docs/ROADMAP.md`](../docs/ROADMAP.md) §二 / §四）。发布通道与回滚见 [`docs/DEPLOYMENT.md`](../docs/DEPLOYMENT.md) §6.4、§7。
## 5. 分支模型

```
feat/* fix/* (topic)  ──PR──▶  develop  ──发版时合并──▶  main (稳定发布，打 v* tag)
```

| 分支 | 定位 | 规则 |
|---|---|---|
| `develop` | **主开发分支**（默认工作分支） | 日常 PR 合入此处；门禁全绿 + 审查通过才能合并 |
| `main` | **稳定发布分支** | 只在发版时从 `develop` 合并；对外可用的稳定版本以 `main` 上的 tag 为准 |
| `feat/*`、`fix/*` … | 主题分支 | 从 `develop` 切出，一个分支只做一件事（[`CONTRIBUTING.md`](CONTRIBUTING.md) §分支与提交流程） |

## 6. 冲突与升级路径

分歧在 PR / Issue 评论里**先对话**，按以下顺序推进：

1. **回到事实**：以可复现证据（测试、门禁输出、实测日志）与仓库既有文档为准；契约与门禁是共同基准，
   不是某一方的偏好。
2. **回到文档与 ADR**：若分歧源于规则不清，先判断是「文档缺失」还是「规则缺失」——前者补文档，
   后者按 §3.2 走 ADR。
3. **行为问题**：涉及人身攻击、骚扰、歧视等，按 [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) 处理，
   以该准则为**唯一准绳**（投诉走私密渠道，维护者有保密义务）。
4. **维护者裁定**：技术与流程分歧最终由维护者决定——**并且必须说明理由**（在 Issue / PR 里写明
   决策依据与被拒方案的理由），而不是只给结论。若理由充分，任何人都可以据此再提新证据推动复议。
5. **不接受裁定时**：你有权 fork（MIT 许可，见 [`LICENSE`](../LICENSE)）——这是本项目的最终兜底，
   也是「单人维护 + 无委员会」结构下唯一诚实的答案。

> **当前没有**正式的申诉委员会、仲裁人或投票机制；第 4 步的「说明理由」是唯一的问责形式。

## 7. 如何成为维护者

**当前路径**：长期、高质量的贡献 → 维护者邀请。

| 阶段 | 实际含义 |
|---|---|
| 持续贡献 | 反复提交被合并的高质量 PR（含测试与文档同步），并**在审查中表现出对仓库硬约束的理解**（TDD、文档同步、死代码零容忍、安全边界） |
| 承担审查 | 在 Issue / PR 里给出有依据的技术意见，帮维护者分担判断成本 |
| 邀请 | 由维护者**主动邀请**成为维护者 / 审查者，并同步改 [`.github/CODEOWNERS`](CODEOWNERS) 与本文档 §1、§2 |

**明确不承诺**：没有固定的贡献数量、时长或时间表门槛（「再提交 N 个 PR 就晋升」这种规则**当前没有**），
也不保证一定会有第二个维护者席位。理由与 §1 的「为什么现在不建委员会」相同：**席位必须对应真实职能**，
在能真正分担审查与发布责任之前，授予名义权限对项目没有帮助。

## 8. 与其它政策的分工

| 主题 | 归属文件 | 边界 |
|---|---|---|
| 支持渠道、提问自查、不提供什么 | [`SUPPORT.md`](SUPPORT.md) | 「去哪问、能期待什么」 |
| 贡献流程、提交规范、开发环境、发布步骤细节 | [`CONTRIBUTING.md`](CONTRIBUTING.md) | 「怎么提改动」 |
| **谁决策 / 谁审查 / 谁发版 / 分歧怎么收场** | **本文件** | 「权力与流程」 |
| 漏洞披露、支持版本表、安全默认值 | [`SECURITY.md`](SECURITY.md) | 安全决策的**唯一来源**；本文件不重复其内容 |
| 可接受行为、执行与举报 | [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) | 行为是**唯一准绳**；本文件 §6 只引用不解释 |
| 代理权限矩阵、必须人类确认的动作、AI 披露 | [`docs/AI_POLICY.md`](../docs/AI_POLICY.md) | 代理行为的边界 |
| 开发硬约束、门禁、文档同步、验收清单 | [`AGENTS.md`](../AGENTS.md) · [`docs/DEVELOPMENT.md`](../docs/DEVELOPMENT.md) | 工程规范正文 |
| 版本规划与功能候选 | [`docs/ROADMAP.md`](../docs/ROADMAP.md) | 方向（不含缺陷） |
| 缺陷 / 阻塞 / 技术债 | [`docs/KNOWN_ISSUES.md`](../docs/KNOWN_ISSUES.md) | 问题（不含方向） |
| 兼容性承诺与弃用政策 | [`docs/compatibility.md`](../docs/compatibility.md) | 对外契约的时间维度 |
