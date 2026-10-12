# docs 全量文档复盘总结（2026-10-12）

> **性质**：时点性**文档复盘快照**——对 `docs/` 下**全部文档**做一次通读复盘，结论绑定
> 2026-10-12 的工作树（行号 / 体量均为当日实读所得）。命名先例：
> [`code-review-2026-09-24.md`](code-review-2026-09-24.md) ·
> [`web-review-2026-10-12.md`](web-review-2026-10-12.md) 等时点性评审快照（小写 kebab-case）。
>
> **归档冻结（2026-10-12）**：本文沿用的 [`handoff-20261010.md`](handoff-20261010.md) §5 的 14 项建议，
> 经复核**已于更早的 2026-10-10 批次全部闭环**（5 道新门禁 + 9 处文档结构修复，逐条证据见
> [`../FEATURES.md`](../FEATURES.md) §CJ，门禁注释亦逐条点名「交接快照 §5 未做第 N 项」）——
> 本文 §1 / §5 / §7 采信该清单时**未逐项复核**，「仍存问题」与「下一步建议」以 §CJ 为准。
> 正文按归档纪律不改写，相对链接按新路径修正。
>
> **本文不是任何事项的 SSOT**。冲突时以唯一来源为准：
> **问题**（缺陷 / 阻塞 / 技术债）→ [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)；
> **方向**（功能候选 / 版本级）→ [`ROADMAP.md`](../ROADMAP.md) §三；
> **已实现台账** → [`FEATURES.md`](../FEATURES.md)；**规范正文** → [`DEVELOPMENT.md`](../DEVELOPMENT.md)。
>
> **证据口径**：体量用 `find` / `ls` 实测，结构与事实用 `grep` / `sed` 实读；本会话**未实跑**的项
> （真实 RustFS / 浏览器 E2E、两套 CI 远端、`pnpm audit`）在 §8 如实披露，不代偿、不美化。
>
> **生命周期**：本文的建议项（§7）多为**已有登记的文档维护项**（[`archive/handoff-20261010.md`](handoff-20261010.md) §5
> 的 14 项「未做清单」），并未新立问题编号；当这些建议被处置或明确移交后，本文按
> [`archive/index.md`](index.md)「归档操作」四步冻结入 `docs/archive/`——
> 不移除、不回写、不改写历史结论。

## 1. 复盘摘要（先读这段）

- **规模**：`docs/` 下共 **54 篇 Markdown**（顶层 23 · `archive/` 13 · `decisions/` 15 · `en/` 3；**不含本文自身**——
  本文即第 55 篇），
  合计约 **14.6k 行**；另有 2 份机器可读契约（[`api/openapi.json`](../api/openapi.json) 约 255 KB、
  [`api/accounts.schema.json`](../api/accounts.schema.json) 约 4 KB）。
- **三层入口**，分工明确、不互相复制正文：**人** → [`README.md`](../README.md)（docs 落地页 / 导航 SSOT）；
  **agent** → 根 [`AGENTS.md`](../../AGENTS.md)（硬约束 + 文档入口表，工具自动注入）；
  **LLM** → 根 [`llms.txt`](../../llms.txt)（扁平索引 + 机器可读契约路径）。
- **命名与存放有硬规则**：`docs/` 下「元文档大写 / 产品内容小写 kebab-case」二分，时点性快照用小写并
  在收口后 `git mv` 进 [`archive/`](index.md) 冻结；命名清单在 `AGENTS.md` / `DEVELOPMENT.md` §4 /
  `llms.txt` **三处同步**（防分叉）。
- **文档被大量机械化守住**：约 **30 道包根门禁**（`apps/server/*_gate_test.go`）外加子包与前端门禁，
  把契约、叙述性数字、链接与锚点、导航覆盖、命名清单、ADR 覆盖、第三方许可证、告警表、对比度、
  英文 revision、注入体积预算等钉在源码 / 生成物上——文档失真会**红灯**，而不是靠人记得。
- **急迫问题为零**：[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 存量仅 **#25**（桌面端签名，⛔ 外部凭证阻塞）
  与 **#63**（流式复制 640GB 上限，➖ 已决策维持现状）；产品功能缺陷为零。
- **主要风险已从「失真」转为「体量与维护成本」**：几个超大文件（[`FEATURES.md`](../FEATURES.md) 401 KB /
  2731 行、[`OPERATIONS.md`](../OPERATIONS.md) 897 行、[`api.md`](../api.md) 1133 行）尚未做结构性瘦身，是
  下一阶段唯一值得成批投入的文档工作（§5 / §7）。
- **一句话结论**：这是一套**成熟度显著高于同类项目**的文档体系——机制（唯一来源 + 机械门禁 + 诚实披露）
  已成型且被反复自我修正（三轮代码评审 + 两轮批次交接 + 三份子树评审），无需重建；
  下一阶段的边际收益在**体量治理**与**体例收尾**，不在新增机制。

## 2. 文档全景（清单与体量）

按性质分组的实测体量（行数为 `wc -l`，字节为 `ls -l`；统计时点 2026-10-12，**不含本文自身**）：

| 分组 | 篇数 | 代表文档（行数） | 角色 |
|---|---:|---|---|
| **仓库元文档（大写）** | 12 | [`FEATURES.md`](../FEATURES.md) 2731 · [`OPERATIONS.md`](../OPERATIONS.md) 897 · [`DEVELOPMENT.md`](../DEVELOPMENT.md) 416 | 描述「仓库自身如何运作」：规范、运维、台账、路线、政策 |
| **产品内容文档（小写）** | 10 | [`api.md`](../api.md) 1133 · [`user-guide.md`](../user-guide.md) 791 · [`accessibility.md`](../accessibility.md) 494 | 描述「产品是什么 / 怎么用」：接口、架构、数据、错误、安全、手册 |
| **docs 落地页** | 1 | [`README.md`](../README.md) 102 | 人类导航 SSOT（`doc_index_gate_test.go` 校验覆盖） |
| **ADR（`decisions/`）** | 15 | [`0013`](../decisions/0013-zero-dep-otlp-tracing.md) 111 · [`index.md`](../decisions/index.md) 33 | 13 篇决策 + 模板 + 索引；只增不删、被取代则新写一篇 |
| **归档（`archive/`）** | 13 | [`review-2026-09-19.md`](review-2026-09-19.md) 530 · [`code-review-summary.md`](code-review-summary.md) 305 | 冻结的时点性快照（评估 / 审查 / 事故复盘 / 批次交接 / 子树评审） |
| **英文快照（`en/`）** | 3 | [`index.md`](../en/index.md) 403 · [`architecture.md`](../en/architecture.md) 187 | 中文为 SSOT，英文页是带 `Source revision` 的翻译快照 |
| **机器可读契约（`api/`）** | 2 JSON | [`openapi.json`](../api/openapi.json) 255 KB（⚠️ 超大）· `accounts.schema.json` 4 KB | 不跑服务即可读的 API / 账号库契约 |

**体量排名前 6**（决定「按需读」策略，也是 §5 体量风险的来源）：

| 文档 | 字节 | 行数 | 备注 |
|---|---:|---:|---|
| [`FEATURES.md`](../FEATURES.md) | 401 KB | 2731 | 单一事实来源，历史批次台账全量堆叠；页头摘要为超长单行 |
| [`OPERATIONS.md`](../OPERATIONS.md) | 77 KB | 897 | Runbook R-1..R-10 + 告警 + 备份恢复；有 12 节附录 |
| [`user-guide.md`](../user-guide.md) | 63 KB | 791 | 面向终端用户，十六节 |
| [`DEVELOPMENT.md`](../DEVELOPMENT.md) | 59 KB | 416 | 规范正文 + 文档登记表 + 门禁落点长清单 |
| [`api.md`](../api.md) | 54 KB | 1133 | 无目录（handoff §5 第 8 项）；与 OpenAPI 注册表同源 |
| [`accessibility.md`](../accessibility.md) | 45 KB | 494 | 现状盘点 + 已知限制；与 §5.5 对比度表受门禁 |

## 3. 逐篇速览

### 3.1 仓库元文档（大写）

| 文档 | 一句话职责 | 复盘观察 |
|---|---|---|
| [`DEVELOPMENT.md`](../DEVELOPMENT.md) | 规范正文：TDD / 三类测试 / 门禁 / **文档同步门禁 §4** / 验收清单 / Red Flags | 本文件是「改完必须读」的中枢；§3 门禁落点段极长（约 100 行），是文档维护自身的元数据 |
| [`CONFIGURATION.md`](../CONFIGURATION.md) | 全部 `S3C_*` 环境变量的 **SSOT** + 启动期 fail-closed 清单 | 明示 `.env` **行内注释不被支持**（真实坑），两份 `.env.example` 分工写清 |
| [`DEPLOYMENT.md`](../DEPLOYMENT.md) | 部署形态 / compose / Nginx / TLS / 升级与回滚 / 桌面分发 | §6.4 升级六项验证与 §7 回滚步骤**可照做**；桌面端未签名现状如实写明 |
| [`OPERATIONS.md`](../OPERATIONS.md) | 可观测性 / SLO 与告警 / Runbook / 备份恢复 / 灾难恢复 / 事故响应 | 全篇标注证据等级（「建议值」vs「代码强制」）；告警表受 `doc_alert_drift_gate` 与真实规则文件双向钉住 |
| [`PERFORMANCE.md`](../PERFORMANCE.md) | 后端热路径基准与解读（加密写 30ms/64MiB、账号写 O(n)、预签名 33µs） | **明说这不是性能承诺**；工具链升级后基线**尚未复测**并如实标注 |
| [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md) | 事故复盘模板（含 DR 演练补充字段） | 首份实例已冻结在 [`archive/incident-20260916-presign-empty-url.md`](incident-20260916-presign-empty-url.md) |
| [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) | 缺陷 / 阻塞 / 技术债的**唯一来源** | 编号稳定不重排；编号台账逐条可追；页头更新流水偏长（handoff §5 第 3 项） |
| [`ROADMAP.md`](../ROADMAP.md) | 版本规划与里程碑；功能候选池 §三 | §3.2 候选池（#4–#16 未排期）+ §五 风险 R*/依赖 E* 登记，是「方向」的唯一来源 |
| [`FEATURES.md`](../FEATURES.md) | 产品能力总览 + 已完成修复 / 优化的**证据台账** | 单一事实来源，但历史批次全量堆叠 → 体量最大（§5） |
| [`AI_POLICY.md`](../AI_POLICY.md) | 代理五档模式 / 权限矩阵 / AI 披露 / DoD / 多代理协作 | §11「哪些声明由门禁守住」逐条标注**覆盖程度**（代码强制 / 部分 / ⚠️ 人工），不夸口 |
| [`AGENT_EVALS.md`](../AGENT_EVALS.md) | 黄金任务集 GT-1..GT-4 / 五维评分卡 / AI 贡献度量 | §六台账**如实记录**「机械基线 N/A ≠ 满分」；度量回填仍无门禁（handoff §5 第 4 项） |
| [`THIRD_PARTY_LICENSES.md`](../THIRD_PARTY_LICENSES.md) | 脚本自动生成的第三方依赖与许可证清单（Go 43 / Rust 416 / npm 1） | 页头明写「勿手工编辑」；§1 SPDX 表达式有同义分裂（handoff §5 第 6 项） |

### 3.2 产品内容文档（小写）

| 文档 | 一句话职责 | 复盘观察 |
|---|---|---|
| [`architecture.md`](../architecture.md) | 总体架构、后端分层、关键机制、关键取舍 | §2 明说「`store → model → s3wrap → handler` 是**栈序简化**，逐条 import 以本页为准」——主动纠正常见误解 |
| [`api.md`](../api.md) | REST API 参考（与 OpenAPI 注册表同源） | 长且无目录；curl 示例覆盖全部 tag 受门禁钉住 |
| [`data-model.md`](../data-model.md) | `model.Account` 字段 / 三驱动 / `S3C2`「`S3C3` 信封地图 | **自声明非 SSOT** 并给出 §0 冲突回退顺序——诚实且可维护 |
| [`errors.md`](../errors.md) | S3 错误 → HTTP 状态 → 用户文案对照 | 明确「已移除字符串匹配，改用 sentinel + `errors.Is`」，避免厂商改文案即静默失效 |
| [`threat-model.md`](../threat-model.md) | STRIDE × 边界威胁模型 + 安全默认值 + 供应链 | §7 把 GitLab SAST 的 10 项发现逐条 triage（误报 / 设计使然）——**报告型工具也有处置账** |
| [`user-guide.md`](../user-guide.md) | 终端用户手册（界面 / 上传下载 / 版本回收站 / 快捷键 / FAQ） | 十六节，含「数据与隐私」；个别工具栏行缺漏（handoff §5 第 13 项） |
| [`compatibility.md`](../compatibility.md) | 版本命名 / 支持窗口 / API 演进 / 格式兼容 / 兼容矩阵 / 弃用政策 | 明确「当前**没有**版本机制」「**没有**弃用记录」，只写**将要**遵循的口径，不粉饰 |
| [`glossary.md`](../glossary.md) | S3 领域术语 + 本项目自造词 | 把「栈序简化」「兜底 vs 死代码」「代次守卫」等内部黑话显式定义，降低接手成本 |
| [`i18n.md`](../i18n.md) | 界面文案体系 + `docs/en/` 翻译体系 | 明写「判定规则以代码为准，不一致请顺手修正本文件」；§7.4 漂移处置分清人工 / CI 强制 |
| [`accessibility.md`](../accessibility.md) | ARIA / 键盘可达性 / 主题 / **明确没做**的事 | §4「已知限制」逐条带 ✅ 修复日期或「仍未做」；§5.5 对比度表受 `contrast_gate_test` 机械重算 |

### 3.3 ADR（`decisions/`，13 篇 + 模板 + 索引）

- **口径**：轻量 ADR，H2 固定为 `Status / Date / Context / Decision / Alternatives Considered / Consequences`；
  **只增不删**，决策被取代时新写一篇并标 `Superseded`，补充事实则**追加** `## Update（YYYY-MM-DD）`。
- **覆盖面**：桌面端无 IPC（ADR-001）、存储 fail-closed（002）、SSRF 放行私网（003）、前端最小依赖（004）、
  SSE 异步任务（005）、三驱动原子写（006）、预签名直传（007）、零依赖前端栈（008）、有界并发（009）、
  ZIP 流式（010）、单实例无 HA（011）、REST 无版本前缀（012）、零依赖 OTLP tracing（013）。
- **观察**：`architecture.md` §7「关键取舍」每行都必须含 ADR 链接（`adr_coverage_gate_test.go` 守住），
  ADR 索引与文件双向等集（`TestADRIndexListsEveryADRFile`）——决策不会「写了没人知道」。
  体例级收尾项见 handoff §5 第 12 项（`0013` 与模板 / 索引的 Status 括注形态）。

### 3.4 归档（`archive/`，13 份冻结件）

| 归档件 | 冻结的结论 |
|---|---|
| [`assessment.md`](assessment.md) | 2026-09-16 五维度综合评估（78/100），P0/P1/P2 收口 |
| [`review-2026-09-19.md`](review-2026-09-19.md) | 2026-09-19 分支整体状态审查（P0/P1/P2、S/R/D 系列全闭环） |
| [`code-review-2026-09-24.md`](code-review-2026-09-24.md) | 全仓代码审查：2 Critical + 20 Required 全修；Nit 35 项处置 |
| [`code-review-summary.md`](code-review-summary.md) | 审查总结；**已移除仓内无采集来源的编造指标**（成功指标一节） |
| [`code-review-2026-10-09.md`](code-review-2026-10-09.md) | 全仓五轴评审：2 Critical + 10 Required + 12 Optional 全部闭环 |
| [`code-review-2026-10-10.md`](code-review-2026-10-10.md) | 全仓五轴评审：结论 **Approve**，3 道非阻断 Nit 同日闭环 |
| [`handoff-20260930.md`](handoff-20260930.md) · [`handoff-20261010.md`](handoff-20261010.md) | 两轮批次交接快照（含「未做清单」，是 §5 建议项的来源） |
| [`web-review-2026-10-12.md`](web-review-2026-10-12.md) · [`desktop-review-2026-10-12.md`](desktop-review-2026-10-12.md) · [`server-review-2026-10-12.md`](server-review-2026-10-12.md) | 同日三份子树评审快照（UI/交互 9 批 · 桌面 1 死代码 + 6 优化 · 服务端 5 微优化） |
| [`incident-20260916-presign-empty-url.md`](incident-20260916-presign-empty-url.md) | 首份已填写事故复盘（presign 吞错回 `200 {"url":""}`，near-miss） |

> **冻结纪律**：归档 = 冻结，不回写、不改写历史结论；归档于 2026-10-08 产品名改名**之前**的快照里
> `S3CLINET_*` 等旧环境变量名**照抄会静默 `t.Skip`**，[`archive/index.md`](index.md) 页头已显式警示。

### 3.5 英文快照（`en/`）与机器可读面

- [`en/index.md`](../en/index.md)（根 README 翻译快照）· [`en/README.md`](../en/README.md)（英文导航落地页）·
  [`en/architecture.md`](../en/architecture.md)（架构翻译快照）。**中文是 SSOT**，英文页文首必须声明
  来源链接与 `Source revision`（`en_docs_gate_test.go` 校验，且 revision 日期不得早于中文源最后提交）。
- 当前只有 `architecture.md` 一篇产品文档有英文翻译，其余中文-only——[`i18n.md`](../i18n.md) §7.5 如实写明。
- [`api/openapi.json`](../api/openapi.json)（255 KB，`llms.txt` 已标 `⚠️ 超大`）与
  [`api/accounts.schema.json`](../api/accounts.schema.json) 是**不跑服务即可读**的契约，分别由 golden 比对
  与反射门禁守住。

## 4. 交叉观察：这套文档为什么「不容易烂」

1. **唯一来源纪律 + 两源分工**：问题进 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)、方向进 [`ROADMAP.md`](../ROADMAP.md) §三，
   「同一编号只在一个文件里是当前条目」，跨文件引用必须带前缀——从机制上消灭「两处互相漂移」。
2. **把可机械推导的事实交给门禁**：端点总数、配置项数、i18n 模块清单、gate 道数
   （`doc_number_gate`）、链接与锚点（`doc_link_gate`）、导航覆盖（`doc_index_gate`）、
   三处命名清单（`docs_naming_gate`）、ADR 覆盖、许可证全量、告警表 ⇔ 规则文件
   （`doc_alert_drift_gate`）、对比度表 ⇔ `styles.css`（`contrast_gate`）、英文 revision
   （`en_docs_gate`）、超大文件预警（`llms_size_gate`）。**文档失真 = 红灯**，不靠人自觉。
3. **诚实披露是明文要求**：多处显式写「未实测 / 未跑 / N/A ≠ 满分 / 沿用 CI」，
   并**删掉了**没有采集来源的编造指标（[`code-review-summary.md`](code-review-summary.md)）。
4. **文档同步是硬约束 #2**，且配对照表（[`DEVELOPMENT.md`](../DEVELOPMENT.md) §4「改动类型 → 必须同步的文档」）——
   不存在「代码改了文档没跟上」的灰色地带。
5. **时点性文档有明确生命周期**：活跃 SSOT 与冻结快照分开（`archive/`），归档走「四步」且**不回写历史**，
   历史结论与当前事实不互相污染。
6. **自我纠错留痕**：编号稳定不重排（`#N` 与历史提交里的注释同指）、沿革段落记录「不要凭直觉改回去」
   （如 `API.md` ↔ `api.md` 的三代命名规则），让后续接手者不会好心办坏事。

## 5. 复盘发现：仍存在的问题与风险

> 均为**文档层面**的观察，不新开问题编号；可机械可维护的项见 §7，需要决策的项保留在本文供交接。

**中（体量 / 维护成本）**

- **超大文件尚未做结构性瘦身**：[`FEATURES.md`](../FEATURES.md)（401 KB / 2731 行）页头为超长单行摘要 + 6000+ 字
  目录行；[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 页头「上一轮更新」流水偏长且与台账三重重复；
  [`DEVELOPMENT.md`](../DEVELOPMENT.md) §3 门禁落点段可读性下降。它们是 handoff §5 第 1–3 项的遗留，
  **单纯瘦身但需一次专门校对**，故被反复押后。
- **`api.md` 无目录**（1133 行，handoff §5 第 8 项）；operation 级 `description` 在 OpenAPI 中全缺。

**低（体例 / 内容收尾）**

- [`THIRD_PARTY_LICENSES.md`](../THIRD_PARTY_LICENSES.md) §1 SPDX 表达式同义分裂（`MIT OR Apache-2.0` 与
  `Apache-2.0/MIT` 等约 8 行）——需改生成脚本并重生成（handoff §5 第 6 项）。
- [`compatibility.md`](../compatibility.md) §6.2 仍以 `file:line` 引用（行号会漂移），根治是改**符号锚点**。
- [`POSTMORTEM_TEMPLATE.md`](../POSTMORTEM_TEMPLATE.md) §0 / §11 有重复解释；[`OPERATIONS.md`](../OPERATIONS.md) §3.2
  白名单 / 桶边界可归附录；[`user-guide.md`](../user-guide.md) §三 工具栏表缺「桶属性」「视图切换」两行、
  FinOps 看板未单列成节；[`decisions/0013`](../decisions/0013-zero-dep-otlp-tracing.md) 体例与
  [`0000-template.md`](../decisions/0000-template.md) / 索引的 Status 括注未统一（handoff §5 第 11–13 项）。
- **两道「建议但未立项」的门禁**：[`AGENT_EVALS.md`](../AGENT_EVALS.md) 度量表回填无强制（第 4 项）；
  `i18n.md` §2 表行数 ⇄ `coverage.test.ts` 模块数的弱断言（第 9 项）；`doc_status_drift` 可扩面为
  「页头『最后更新』≤ 最近正文日期」（第 10 项）。

**接受 / 不构成缺陷**

- [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 编号不连续（保留空号 #40/#42/#43/#44）——**有意如此**，
  为保持历史引用稳定，不是漏登记。
- [`en/`](../en/README.md) 仅 3 页、多数文档中文-only——[`i18n.md`](../i18n.md) §7.5 已如实说明，无「英文覆盖 X%」门禁。

## 6. 经验与教训

**正面（值得固化）**

- **「不编造数据」是可执行纪律，不是口号**：台账数字来自实跑；跑不动的写「沿用 CI」；没有采集来源的
  指标直接删除。
- **报告型工具也要有处置账**：GitLab SAST 报告（10 项）逐条 triage 记在
  [`threat-model.md`](../threat-model.md) §7，避免「跑一次没人看」。
- **主动纠正常见误解**：[`architecture.md`](../architecture.md) §2 说明分层是「栈序简化」、
  [`data-model.md`](../data-model.md) 自声明非 SSOT、[`glossary.md`](../glossary.md) 把内部黑话定义化——
  降低「照着错的直觉改坏」的概率。

**反面教材（已闭环，留作警示）**

- **为冲覆盖率删「不可达」分支 → 真实缺陷**：[`incident-20260916-presign-empty-url.md`](incident-20260916-presign-empty-url.md)
  记录 `b3b287c` 为把语句覆盖率推到 100% 删掉测试触达不到的预签名错误分支，导致签名失败仍返回
  `200 {"url":""}`（在库 11 天 23 小时，near-miss）。**教训**：100% 是「达到」而非「自然覆盖」，
  不可达分支该删死代码、可能触发的兜底该保留并用行为断言覆盖（[`DEVELOPMENT.md`](../DEVELOPMENT.md) §4 已写入）。
- **成套现状表两处维护必然漂移**：FEATURES §三 与 ROADMAP §四 曾各存一份现状表而互相矛盾，
  处置是**收敛为指针**（唯一来源在 ROADMAP §四）。
- **门禁也会停在旧数字**：`doc_number_gate` 实测发现「配置项数停在 18、真值 22」、
  「gate 道数停在 27」——**可机械推导的数字必须钉在源码上**，否则文档静默过期。

## 7. 下一步建议（按性价比排序）

1. **超大文件结构瘦身**（[`FEATURES.md`](../FEATURES.md) 页头/目录、[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 页头、
   [`DEVELOPMENT.md`](../DEVELOPMENT.md) §3）——收益最大、纯体力需一次专门校对（handoff §5 第 1–3 项）。
2. **给 [`api.md`](../api.md) 补目录**，并统一 `multipart/complete` 的 etag 引号口径（第 8 项）。
3. **`compatibility.md` §6.2 从 `file:line` 改为符号锚点**，根治行号漂移（第 7 项）。
4. **补两道建议门禁**：`i18n.md` §2 表 ⇄ 模块数弱断言 + `doc_status_drift` 扩面「页头日期 ≤ 最近正文日期」
   （第 9–10 项）；并给 `AGENT_EVALS` 度量回填接上 PR 模板 checkbox（第 4 项）。
5. **体例收尾批次**：`decisions/0013` 与模板 / 索引 Status 形态、`POSTMORTEM_TEMPLATE` §0/§11 合并、
   `OPERATIONS` §3.2 归附录、`user-guide` 工具栏补行 + FinOps 单列（第 11–13 项）。
6. **再生 `THIRD_PARTY_LICENSES.md`** 以消除 §1 SPDX 同义分裂（改生成脚本，第 6 项）。
7. **按需补 `en/` 覆盖**（先点名「刻意不译」清单，再给 revision 过期加机械提醒，第 14 项）。

> 以上均**沿用** [`archive/handoff-20261010.md`](handoff-20261010.md) §5 的既有建议编号，**不新增**
> [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 条目——它们此前已被判定为「低优先级 / 需改脚本 / 需新立项」。
> 若本批要把其中任一项立项，按两源分工登记进 [`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md)（技术债）或
> [`ROADMAP.md`](../ROADMAP.md) §三（方向）即可。

## 8. 未覆盖 / 未实跑（如实披露）

- **本复盘只做「通读」（结构 + 事实锚点 + 交叉观察），未复跑全量门禁**：文档门禁的绿灯状态取自
  最新提交的既有记录（[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) 页头 #87 收口 + [`CHANGELOG.md`](../../CHANGELOG.md)
  `[Unreleased]`），本会话未逐条重跑 `apps/server` 的全部 `*_gate_test.go`。
- **未实跑真实对端 / 浏览器联调**：`S3CLIENT_E2E=1` 的 RustFS E2E、`make e2e-real` 本会话**未执行**
  （环境无 docker / 外网），其结论以既有台账为准。
- **未跑两套 CI 远端**：GitHub Actions / GitLab CI 的实际通过与否以推送后结果为准。
- **`pnpm audit` 无 audit 端点**，任何数字都应写「沿用 CI」（沿用历史各批口径）。

## 9. 本文件的登记与归档

- **登记**（文档门禁要求，同批完成）：[`README.md`](../README.md)（docs 导航 SSOT）「状态、规划与历史」表一行
  （`doc_index_gate_test.go`）；`AGENTS.md` 命名段 + 本目录 `DEVELOPMENT.md` §4 命名段 + `llms.txt` 目录段
  **三处反引号登记**（`docs_naming_gate_test.go`）；[`DEVELOPMENT.md`](../DEVELOPMENT.md) §4「文档登记表」一行
  （`doc_review_gate_test.go`）；[`../CHANGELOG.md`](../../CHANGELOG.md) `[Unreleased]` 记一条。
- **归档条件**：本文 §7 建议项被处置或明确移交（登记进 `KNOWN_ISSUES` / `ROADMAP`）后，按
  [`archive/index.md`](index.md)「归档操作」四步 `git mv` 入 `docs/archive/` 并冻结——
  届时同步从三处命名清单与 [`README.md`](../README.md) 导航行摘除，登记表行改为「冻结件 · 不复审」口径。
- **复核命令**：`cd apps/server && go test . -count=1`。

## 10. 相关文档

- 文档导航（人类入口）：[`README.md`](../README.md) · 归档索引：[`archive/index.md`](index.md) ·
  ADR 索引：[`decisions/index.md`](../decisions/index.md)
- 规范正文与文档同步门禁：[`DEVELOPMENT.md`](../DEVELOPMENT.md) · 代理硬约束：[`../AGENTS.md`](../../AGENTS.md) ·
  LLM 索引：[`../llms.txt`](../../llms.txt)
- 问题 / 方向 / 台账：[`KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) · [`ROADMAP.md`](../ROADMAP.md) · [`FEATURES.md`](../FEATURES.md)
- 上一轮全 docs 通读的批次交接：[`archive/handoff-20261010.md`](handoff-20261010.md)（§5 即 §7 建议项来源）
- 最近一次全仓代码评审：[`archive/code-review-2026-10-10.md`](code-review-2026-10-10.md)
