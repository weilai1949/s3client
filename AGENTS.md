# AGENTS.md

> 本文件是 **AI 编码代理的仓库级入口**：支持 `AGENTS.md` 约定的工具（deepseek-harness、Codex、Cursor 等）
> 会在仓库根目录自动发现并把它注入模型上下文，无需人工指路。
> **规范正文以 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) 为准**——本文件只列不可从代码推断的硬约束与入口，
> 不复制正文，避免两处维护漂移。

## 硬约束（违反即不得提交 / 合并）

1. **TDD 优先**：先写会失败的测试，再写实现。断言**外部可见行为**（返回值 / HTTP 状态码 / 渲染结果），不要断言私有函数或内部变量；不要为凑覆盖率写与实现耦合的 gap 测试。
2. **改完代码必须同步文档**：修复 bug 或新增功能后，相关文档与代码属于**同一个 commit / PR**；文档未同步 = 改动未完成。改动类型 → 文档对照表见 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4。
3. **分层不外泄**：沿用 `store → model → s3wrap → handler` 分层；AWS SDK 类型不进 handler / 前端；单文件不超过约 1000 行，超过先拆再改；功能逻辑不进共享工具模块。
4. **安全**：用户输入在边界校验；`SecretKey` 不落地 localStorage、不写日志；禁 `v-html`；外部数据一律视为不可信。
5. **不留尾巴（死代码零容忍）**：**不得有死代码**（定义后无人引用的函数 / 类型 / 常量 / 变量 / 字段、不可达分支）；**不得有未定义就使用的标识符**；**不得有定义了却未使用的变量**（含只写不读、赋值后即被覆盖、恒真/恒假的空断言）。**枚举值除外**——枚举/常量表的成员即使暂无引用也算契约的一部分，保留。无「以后再说」的兼容 shim，提交前清干净。
   - **警惕「被覆盖率掩盖的死代码」**：测试文件不参与 instrumentation，`go test -cover` 只看生产代码，因此测试辅助里的死代码能让 100% 门禁全绿。覆盖率达标 ≠ 无死代码，两者要分别验证。
   - 语言/工具链已有的机械检查必须开启并保持零告警：Go 用 `golangci-lint`（`unused` / `staticcheck`，配置见 `apps/server/.golangci.yml`）、`go vet`；TS/JS 用 `pnpm lint` + `vue-tsc`。**门禁输出必须是 0 issues**，不接受「历史遗留」作为例外。

## 提交前门禁（必须全绿）

```bash
cd apps/server && go vet ./... && go test ./... && go build ./...
cd apps/web && pnpm test && pnpm build          # 前端单测 + vue-tsc 类型检查 + 构建
# 涉及签名 / 预签名直传 / 分段 / 复制 / 标签 / 版本控制时追加真实 RustFS E2E：
cd apps/server && S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v
# 涉及前端 / 后端接口 / 预签名直传时追加「真实后端 + 真实 RustFS + 真实产物」浏览器联调
# （docker 自动起 RustFS，跑完清理；不 mock /api）：
make e2e-real
# 或 make test-all（后端 + 前端单测）
```

## 文档入口

| 需要什么 | 看哪里 |
|---|---|
| 开发规范 / 测试分层 / 门禁 / 验收清单 / Red Flags / 技术债 | [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) |
| 文档导航（`docs/` 落地页，按「我要做什么」分组） | [`docs/README.md`](docs/README.md) |
| 改动要同步哪些文档（对照表） | [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) §4「文档同步门禁」 |
| 接口与请求/响应字段（改 handler 请求体时同步 `openapi_register_*.go` 并跑契约测试） | [`docs/api.md`](docs/api.md) |
| 问题（缺陷 / 外部阻塞 / 技术债，唯一来源） | [`docs/KNOWN_ISSUES.md`](docs/KNOWN_ISSUES.md) |
| 功能候选 / 版本级规划（唯一来源） | [`docs/ROADMAP.md`](docs/ROADMAP.md) §三 |
| 已实现 / 已修复台账 | [`docs/FEATURES.md`](docs/FEATURES.md) |
| 每个 PR 都要补发版记录（`[Unreleased]`） | [`CHANGELOG.md`](CHANGELOG.md) |
| 架构与关键决策 | [`docs/architecture.md`](docs/architecture.md) · [`docs/decisions/`](docs/decisions/index.md) |
| 提交规范 / 环境搭建 / 发版流程 | [`.github/CONTRIBUTING.md`](.github/CONTRIBUTING.md) |
| 安全策略（漏洞披露）与威胁模型 | [`.github/SECURITY.md`](.github/SECURITY.md) · [`docs/threat-model.md`](docs/threat-model.md) |
| 第三方依赖与许可证清单（**自动生成**，改依赖后必须重新生成） | [`docs/THIRD_PARTY_LICENSES.md`](docs/THIRD_PARTY_LICENSES.md) |
| 代理权限边界 / 什么必须人类确认 / AI 披露模板 / DoD | [`docs/AI_POLICY.md`](docs/AI_POLICY.md) |
| AI 代理评测与贡献度量（黄金任务集 / 评分卡 / 披露占比） | [`docs/AGENT_EVALS.md`](docs/AGENT_EVALS.md) |
| 数据模型 / 存储格式（字段 · 三驱动 · S3C2/S3C3 信封） | [`docs/data-model.md`](docs/data-model.md) |
| 配置项（全部 `S3C_*` 环境变量） | [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) |
| 机器可读 API 契约（不跑服务即可读） | [`docs/api/openapi.json`](docs/api/openapi.json)（改 handler 后必须重新生成） |
| 运维 / 告警 / 备份恢复 / 灾难恢复 / 事故复盘 | [`docs/OPERATIONS.md`](docs/OPERATIONS.md) · [`docs/POSTMORTEM_TEMPLATE.md`](docs/POSTMORTEM_TEMPLATE.md) |
| 用户可见的操作方式 / 快捷键 | [`docs/user-guide.md`](docs/user-guide.md) |
| 支持渠道 / 治理与发布权 | [`.github/SUPPORT.md`](.github/SUPPORT.md) · [`.github/GOVERNANCE.md`](.github/GOVERNANCE.md) |

## 文档命名与存放

- **根目录只放四个约定文件**：`README.md`（社区约定）、`AGENTS.md`（工具加载器**硬性要求**在根目录）、`CHANGELOG.md`（Keep a Changelog 约定名，主流 changelog 工具默认 `./CHANGELOG.md`）、`llms.txt`（[llms.txt 约定](https://llmstxt.org/)固定查 `/llms.txt`，2026-09-29 登记——LLM 导航索引，非规范正文）。另有两个**工具固定名**文件按 GitHub 约定留在根目录：`LICENSE`（许可证）与 `CITATION.cff`（引用元数据——GitHub 的 cite 功能**只认根目录**，2026-09-30 登记），不计入上面「四个约定文件」。**社区健康文件**（`CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md`）放 `.github/`——GitHub 对这类文件的查找优先级是 `.github/` > 根目录 > `docs/`，放最高优先级位置可避免被将来的副本静默顶掉；其余文档统一放 `docs/`。
- **`docs/` 下按性质二分命名**（细则与沿革见 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) 命名约定）：
  - **大写** = 工具固定名（`README.md`、`AGENTS.md`、`CHANGELOG.md`、`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`、`SUPPORT.md`、`GOVERNANCE.md`、`LICENSE`、`CODEOWNERS`、`CITATION.cff`）+ `docs/` 下的**仓库元文档**（仓库自身如何运作）：`CONFIGURATION.md`、`DEPLOYMENT.md`、`DEVELOPMENT.md`、`OPERATIONS.md`、`PERFORMANCE.md`、`AI_POLICY.md`、`AGENT_EVALS.md`、`KNOWN_ISSUES.md`、`FEATURES.md`、`ROADMAP.md`、`POSTMORTEM_TEMPLATE.md`、`THIRD_PARTY_LICENSES.md`。其中 `docs/README.md` 属「工具固定名」一类的特殊情形——它是 GitHub 的 **docs 目录落地页**，同时是人类导航 SSOT（每个 docs 文档都必须能从它到达，见 `doc_index_gate_test.go`）。
  - **小写 kebab-case** = `docs/` 下的**产品内容文档**（产品是什么 / 怎么用）：`api.md`、`architecture.md`、`data-model.md`、`errors.md`、`threat-model.md`、`user-guide.md`、`compatibility.md`、`glossary.md`、`i18n.md`、`accessibility.md`，以及 `api/`（机器可读契约）、`archive/`、`decisions/`、`en/`（英文快照）下全部文件。**时点性批次交接快照**同用小写（先例 `assessment.md`、`handoff-20260930.md`，结论绑定日期、批次收口后归档入 `archive/`）。
  - 新增 `docs/` 文档先判性质再起名；**变更任何大写文件名必须同 PR 同步本处与该命名约定两处**（防分叉）。
- **任何两个路径不得仅大小写不同**——macOS / Windows 的大小写不敏感文件系统会让它们互相覆盖、检出即丢内容。

## 修改本文件

- 只有**仓库级**规则写在这里；`apps/server/`、`apps/web/` 等子树的专属规则应放进该子树的 `AGENTS.md`，工具会在读到该子树文件时惰性加载。
- 可推导的细节（完整表格、清单、命令全集）留在 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)，此处只留指针。
- 本文件会被**整体注入模型上下文**且受字节预算约束，超长会被截断——保持短小。
- 个人本地覆盖写 `AGENTS.local.md`（该文件不提交，已在 `.gitignore` 中排除）；不要用个人偏好覆盖团队硬约束。
