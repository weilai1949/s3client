# 文档导航（docs/）

> 本页是 **`docs/` 目录的落地页与人类导航入口**（GitHub 浏览 `docs/` 时按字面名渲染本文件）。
> 三套入口各有分工，**不要互相复制正文**：
> - **人**（第一次读 / 查用法）→ 本页；
> - **编码代理 / agent** → 根 [`AGENTS.md`](../AGENTS.md)（硬约束 + 文档入口表，工具会自动注入）；
> - **LLM / 工具消费** → 根 [`llms.txt`](../llms.txt)（扁平索引 + 机器可读契约路径）。
>
> **owner / 复审周期 / 最后复审**见 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4「文档登记表」；
> **命名与存放规则**见同节（元文档大写 / 产品内容小写；根目录只放 4 个约定文件）。
> 本页由 `apps/server/doc_index_gate_test.go` 校验：`docs/` 下每篇文档都必须能从本页（或所属子目录索引）到达——
> 新增文档漏登记会让门禁红灯。

## 按「我要做什么」找

### 用起来（终端用户 / 集成方）

| 我想…… | 看哪里 | 一句话 |
|---|---|---|
| 学会用这个工具 | [`user-guide.md`](user-guide.md) | 界面操作 / 首次配置 / 上传下载 / 版本与回收站 / 快捷键 / FAQ / 排障；含「数据与隐私」 |
| 读英文版 README | [`en/index.md`](en/index.md) | 根 README 的完整英文翻译（中文为 SSOT，英文页为翻译快照）；当前唯一的英文文档 |
| 读英文版架构与文档导航 | [`en/README.md`](en/README.md) · [`en/architecture.md`](en/architecture.md) | 英文文档落地页（列出已翻译与中文专属文档）+ 架构全文翻译；**中文为 SSOT，英文为翻译快照**，来源 revision 写在页头 |
| 调 API / 接自己的程序 | [`api.md`](api.md) | REST API 参考（84 个 `/api/*` 端点），与 OpenAPI 注册表同源 |
| 让工具 / AI 直接读契约 | [`api/openapi.json`](api/openapi.json) | 机器可读契约（62 paths / 84 operations，含鉴权与 tags），**不跑服务即可读** |
| 校验 / 生成账号库文件 | [`api/accounts.schema.json`](api/accounts.schema.json) | 账号库 `accounts.json` 的 JSON Schema（2020-12），与 `model.Account` 双向对齐 |
| 查某个报错是什么意思 | [`errors.md`](errors.md) | S3 错误 → HTTP 状态 → 用户文案对照 |
| 判断能不能升级 / 支持多久 / 哪些浏览器与 OS 能用 | [`compatibility.md`](compatibility.md) | 版本命名 / 支持窗口 / API 演进与弃用政策 / 存储格式兼容 / S3 厂商矩阵 / 客户端（浏览器与操作系统）支持矩阵 |
| 查某个词是什么意思 | [`glossary.md`](glossary.md) | S3 领域术语 + 本项目自造词 |
| 换语言 / 加一门语言 | [`i18n.md`](i18n.md) | 语言现状 / 新增语言的步骤 / 文案 key 与覆盖率门禁 |
| 了解无障碍现状与限制 | [`accessibility.md`](accessibility.md) | ARIA / 键盘可达性 / 主题 / **明确没做**的事（未做正式 WCAG 审计） |

### 部署与运维

| 我想…… | 看哪里 | 一句话 |
|---|---|---|
| 部署这个服务 | [`DEPLOYMENT.md`](DEPLOYMENT.md) | 部署形态 / Docker Compose / Nginx / TLS / 桌面端分发 / 升级与回滚 |
| 日常运维、排障、备份 | [`OPERATIONS.md`](OPERATIONS.md) | 可观测性 / SLO 与告警（**建议值**）/ Runbook R-1..R-10 / 备份恢复 / 灾难恢复 / 容量 |
| 查 / 改某个配置项 | [`CONFIGURATION.md`](CONFIGURATION.md) | 全部 `S3C_*` 环境变量的 **SSOT** + 启动期 fail-closed 清单 + 客户端设置 |
| 出事后写复盘 / 做 DR 演练 | [`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md) | 事故复盘模板（取证 / 时间线 / 根因 / 行动项）+ §7.1 灾难恢复演练字段 |
| 看性能基线、解释热路径数字 | [`PERFORMANCE.md`](PERFORMANCE.md) | 基准复现命令与三条实测结论（加密写入 / 账号写入 O(n) / 预签名） |
| 做依赖 / 许可证合规审查 | [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md) | **脚本自动生成**的第三方依赖与许可证清单（Go 43 模块 / Rust 428 crates / npm 1 包；含覆盖门禁） |

### 开发与贡献

| 我想…… | 看哪里 | 一句话 |
|---|---|---|
| 按规范改代码 | [`DEVELOPMENT.md`](DEVELOPMENT.md) | 规范正文：TDD / 三类测试 / 门禁 / **文档同步门禁 §4** / 验收清单 / Red Flags |
| 提交 PR / 起分支 / 发版 | [`../.github/CONTRIBUTING.md`](../.github/CONTRIBUTING.md) | 分支与提交规范 / 开发环境 / 发布流程 |
| 用 AI 代理改代码（或其权限边界） | [`AI_POLICY.md`](AI_POLICY.md) | 五档代理模式 / 权限矩阵（允许·需确认·禁止）/ AI 披露 / DoD / 多代理协作 |
| 看 AI 代理怎么被评测 / 度量 | [`AGENT_EVALS.md`](AGENT_EVALS.md) | 黄金任务集 GT-1..GT-4 / 五维评分卡 / AI 贡献度量口径（效果证据层，与 AI_POLICY 互补）；机械部分 [`../scripts/agent-eval.sh`](../scripts/agent-eval.sh) |
| 看懂整体架构与分层 | [`architecture.md`](architecture.md) | B/S 架构、`store → model → s3wrap → handler` 分层、关键机制与取舍 |
| 查数据怎么落盘 / 三驱动与信封 | [`data-model.md`](data-model.md) | `model.Account` 12 字段 / `json`·`sqlite`·`encrypted` 三驱动 / `S3C3` 信封的**地图**（非 SSOT，冲突裁决顺序见其 §0） |
| 查「当初为什么这么定」 | [`decisions/index.md`](decisions/index.md) | ADR 索引（12 篇 + [`0000-template.md`](decisions/0000-template.md) 新篇模板；architecture §7 取舍表每行必须有 ADR 链接，由 adr_coverage_gate 守住） |
| 看安全边界与威胁模型 | [`threat-model.md`](threat-model.md) | STRIDE × 5 条边界 / 安全默认值 / 已接受的风险 / SAST triage |
| 报告漏洞 / 看安全策略 | [`../.github/SECURITY.md`](../.github/SECURITY.md) | 支持的版本 / **私有**漏洞报告渠道 / 部署加固建议 |
| 问问题 / 提 issue | [`../.github/SUPPORT.md`](../.github/SUPPORT.md) | 支持渠道 / 提问前自查 / **本仓库不提供什么** |
| 了解谁说了算 / 怎么成为维护者 | [`../.github/GOVERNANCE.md`](../.github/GOVERNANCE.md) | 治理现状（单人维护）/ 角色与权限 / 决策与发布权 |
| 了解行为准则 | [`../.github/CODE_OF_CONDUCT.md`](../.github/CODE_OF_CONDUCT.md) | 社区行为准则与执行 |

### 状态、规划与历史

| 我想…… | 看哪里 | 一句话 |
|---|---|---|
| 看已经做了什么 | [`FEATURES.md`](FEATURES.md) | 产品能力总览 + 已完成修复 / 优化的证据台账（**单一事实来源**） |
| 看还有哪些问题 | [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) | 缺陷 / 外部阻塞 / 技术债（**唯一来源**，含编号台账与闭环凭证） |
| 看下一步做什么 | [`ROADMAP.md`](ROADMAP.md) | 版本规划与里程碑；功能候选池在 §三 |
| 看发版历史 | [`../CHANGELOG.md`](../CHANGELOG.md) | Keep a Changelog 格式的逐条发布记录 |
| 找冻结的历史快照 | [`archive/index.md`](archive/index.md) | 评估 / 审查 / 事故复盘 / 批次交接类**时点性文档**的归档索引（只读、不回写；含首份已填写复盘 [`incident-20260916-presign-empty-url.md`](archive/incident-20260916-presign-empty-url.md) 与已收口批次快照 [`handoff-20260930.md`](archive/handoff-20260930.md)） |

## 机器可读面（给工具与 AI）

| 产物 | 位置 | 由什么门禁守住 |
|---|---|---|
| API 契约 | [`api/openapi.json`](api/openapi.json) | golden 比对（`TestCommittedOpenAPISpecMatchesRuntime`）+ 鉴权表达（`openapi_auth_test.go`） |
| 账号库格式 | [`api/accounts.schema.json`](api/accounts.schema.json) | 反射比对 `model.Account`（`TestAccountStoreSchemaMatchesModel`） |
| 告警规则 | [`../deploy/prometheus/s3client.rules.yml`](../deploy/prometheus/s3client.rules.yml) | 指标 / `code` 取值真实性（`TestPrometheusRulesReferenceRealMetrics`） |
| 供应链评分 / PR 依赖审查 | [`../.github/workflows/scorecard.yml`](../.github/workflows/scorecard.yml) · [`dependency-review.yml`](../.github/workflows/dependency-review.yml) | 全 SHA pin 由 `TestWorkflowActionsAreShaPinned` 守住；口径见 [`threat-model.md`](threat-model.md) §5.5 |
| 仓库导航（LLM） | [`../llms.txt`](../llms.txt) | 链接可达性（`doc_link_gate_test.go`）+ 超大文档体量预警（`llms_size_gate_test.go`：目标 >200 KB 必须就地标 `⚠️ 超大`） |
| 代理硬约束 | [`../AGENTS.md`](../AGENTS.md) | 本页 + `AGENTS.md` 命名约定两处同步 |

| 漏洞披露（机器可读） | [`../.well-known/security.txt`](../.well-known/security.txt) | RFC 9116 必填字段 + `Expires` 未过期（`security_txt_gate_test.go`） |
| SLI 仪表盘 | [`../deploy/grafana/s3client.dashboard.json`](../deploy/grafana/s3client.dashboard.json) | 指标 / `code` / recording rule 真实性（`grafana_dashboard_gate_test.go`） |
| AI 评测黄金任务集 | [`../scripts/evals/golden-tasks.yaml`](../scripts/evals/golden-tasks.yaml) | 与 [`AGENT_EVALS.md`](AGENT_EVALS.md) 的 id / 标题逐字一致（`agent_evals_gate_test.go`） |
| 性能预算 | [`../apps/server/bench_budget_test.go`](../apps/server/bench_budget_test.go) | 分配确定性断言 + 时间兜底（`bench_budget_test.go`）；数字解读见 [`PERFORMANCE.md`](PERFORMANCE.md) §4 |

## 文档维护（改文档前先读这节）

| 事项 | 规则 |
|---|---|
| 新增 / 改名 / 删除文档 | 同一 PR 内同步：本页导航 + [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 命名约定与「文档登记表」+ 根 [`llms.txt`](../llms.txt) + 引用它的全部文档；并在 [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 记一条 |
| 命名 | `docs/` 下**元文档大写**（描述仓库自身如何运作）、**产品内容小写 kebab-case**（描述产品是什么 / 怎么用）；本页 `README.md` 属「工具固定名」（GitHub 按字面名渲染目录落地页）。细则见 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 |
| 归档 | 结论绑定在某个 commit / 日期上的时点性文档（评估、审查、事故复盘）`git mv` 进 [`archive/`](archive/index.md) **冻结**——不移除、不回写、不改写历史结论；**已填写的事故复盘在事件闭环后按该页「归档操作」四步冻结**（首例 [`incident-20260916-presign-empty-url.md`](archive/incident-20260916-presign-empty-url.md) 已于 2026-09-30 执行） |
| 链接与锚点 | 全仓 md 的相对链接与页内锚点由 `doc_link_gate_test.go` 机械校验（含扫描面自检阈值） |
| 导航覆盖 | `docs/` 下每篇文档必须有导航入口，由 `doc_index_gate_test.go` 校验 |
| 文档里的数字 | 「N 个 `/api/*` 端点」由 `doc_number_gate_test.go` 钉在 `routes.go` 上；配置项由 `config_doc_gate_test.go` 钉在 `CONFIGURATION.md` 上 |
| owner / 复审周期 | [`DEVELOPMENT.md`](DEVELOPMENT.md) §4「文档登记表」（当前 owner 为单人维护者；复审周期是**建议值**，但带月数的到期由 `doc_review_gate_test.go` 机械红灯） |
| AI 工具入口（根/子树 `AGENTS.md`、Copilot 指针、`llms.txt`） | 体积预算、子树覆盖与回指由 `apps/server/ai_governance_gate_test.go` 断言；政策与门禁的对应关系见 [`AI_POLICY.md`](AI_POLICY.md) §11 |
