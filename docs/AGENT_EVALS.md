# AI 代理评测与贡献度量（Agent Evals）

> **定位**：本文件是第 11 层「AI 时代」的**效果证据层**。分工：
> [`AI_POLICY.md`](AI_POLICY.md) 管「**能不能做**」（过程约束：权限矩阵 / 披露 / DoD）；
> 本文件管「**做得怎么样**」（效果证据：把 AI 改动的质量变成**可重复、可复核的测量**）。
> 两者互链、不互相复制正文。
>
> **术语**：GT = Golden Task（黄金任务）。评测结果台账见 §六（**唯一来源**，不通过项转
> [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)）。
> 本文件的**结构不变量**（GT 集 ≥ 3 条、评测脚本非空壳、AI_POLICY 回引、披露字段集一致）由
> [`../apps/server/agent_evals_gate_test.go`](../apps/server/agent_evals_gate_test.go) 钉住；
> **评分卡打分与「某次改动算不算实质性 AI 贡献」没有机械保证——属人工评审**（单人维护：评审人即维护者本人）；
> 但**台账回填的一致性**已机械化（§四：表行数 ⇄ 状态行条数、PR 模板勾选项、发版脚本复核清单）。

## 一、为什么需要（效果证据 vs 过程约束）

[`AI_POLICY.md`](AI_POLICY.md) 的披露 / 权限矩阵 / DoD 都是**过程约束**：它们证明「流程走了」，
证明不了「输出是对的」——同一份 DoD 可以逐项打勾、门禁全绿，但把 API 契约改坏或把文档写失真。
本文件补上缺失的另一半：**用一组可复现的黄金任务和统一评分卡，让「AI 是否值得信任」从印象变成测量**。
基线事实（盘点时点 2026-10-10）：本仓库已有 **30+ 道机械门禁**（包根 `apps/server/*_gate_test.go` 29 道 +
子包 `error_echo` / `migrate_sync` / `sync_list` 等 3 道 + 前端 `a11y_gate` / `deadcode_gate` / `generated.gate` 3 道；
2026-09-30 盘点时为 20+），但**没有任何针对 AI 产出质量的评测任务集与度量口径**。

## 二、黄金任务集（Golden Tasks）

> 任务取材自本仓库真实发生过的工作类型（证据见 [`FEATURES.md`](FEATURES.md) 台账），不是虚构练习。
> 每项任务的完成判据**全部映射到真实存在的门禁**；评测时交给代理在**干净 checkout** 上重做同型任务，
> 按 §三 评分卡打分——不是 0/1 判定。总览表：

| # | 任务 | 关键门禁（判据映射） | 触碰的文档面 |
|---|---|---|---|
| GT-1 | 新增一个 `/api/*` 端点 | `TestCommittedOpenAPISpecMatchesRuntime`（golden 比对）· `api_doc_test.go` · `doc_number_gate` | [`api.md`](api.md) · [`api/openapi.json`](api/openapi.json) |
| GT-2 | 修复 store 层 bug | `go vet` · `golangci-lint`（0 issues）· `deadcode_gate` · 覆盖率 100% | [`CHANGELOG.md`](../CHANGELOG.md) · [`FEATURES.md`](FEATURES.md) |
| GT-3 | 新增一个 `S3C_*` 配置项 | `config_doc_gate`（红→绿）· `repo_infra_gate`（`.env.example` 占位口令） | [`CONFIGURATION.md`](CONFIGURATION.md) · `apps/server/.env.example` |
| GT-4 | 前端 UI 小改 | `a11y_gate.test.ts` · `deadcode_gate.test.ts` · `vue-tsc` · `pnpm lint --max-warnings 0` | [`user-guide.md`](user-guide.md) · [`accessibility.md`](accessibility.md) |

> **机器可读规格（2026-09-30 起）**：上表的同一份任务集落到
> [`../scripts/evals/golden-tasks.yaml`](../scripts/evals/golden-tasks.yaml)。每项字段：
> `id` · `title` · `scope` · `prompt` · `files_likely_touched` · `verification_commands`
> （每条含 `cmd` / `desc` / 可选 `requires_path`）· `expected_artifacts` · `scoring_anchors`
> （键 = §三 五维 `id`：`gate_green` · `contract_doc_sync` · `tdd_evidence` · `deadcode_security` ·
> `readability_layering`）；顶层另有 `schema_version` / `doc` / `scorecard_dimensions`。
> 文件用 **JSON 内容 + `.yaml` 后缀**：JSON 是 YAML 1.2 的严格子集，`yaml.safe_load` 与 Go 标准库
> `encoding/json` 都能解析——门禁因此不必为评测新增 YAML 模块依赖。任务集 `id` / `title` 与本表
> **逐字一致**，由 [`../apps/server/agent_evals_gate_test.go`](../apps/server/agent_evals_gate_test.go)
> 钉住（改一处不同步另一处 → 门禁红灯）；判据实跑用
> [`../scripts/evals/run-golden-task.sh`](../scripts/evals/run-golden-task.sh)（用法见 §五）。
>
> **诚实边界**：跑判据命令**不等于**完成了一次 GT 评测——判据实跑只证明「判据可执行、当前树的
> 结果如此」；真正的评测是「代理在**干净 checkout** 上重做任务 + 人工按 §三 评分卡打分」。

### GT-1：新增一个 `/api/*` 端点

- **目标**：在既有 `store → model → s3wrap → handler` 分层下新增一个 REST 端点（如桶级设置类），契约与文档全量同步。
- **前置条件**：干净 checkout（`git status` 干净，分支自 `develop` 切出）；不预装任何构建产物；不预先改契约文件。
- **完成判据**：`TestCommittedOpenAPISpecMatchesRuntime`（golden 比对）绿；`api_doc_test.go` 的
  `TestAPIDocMatchesRoutes` 绿；`doc_number_gate` 绿；[`api.md`](api.md) 与
  [`api/openapi.json`](api/openapi.json) 已同步；行为测试（HTTP 状态码 / 响应字段）覆盖正反例。
- **预期红灯→绿灯轨迹**：① 行为测试先红（路由未注册 → 404）→ ② 实现 handler + `openapi_register_*.go`
  → ③ golden 比对红（提交版 spec 未更新）→ ④ 按提示重新生成 spec → ⑤ 全绿。
- **失败典型形态**：只改 handler 不更新注册表（契约测试红）；注册了但 `api.md` / 数字声明没改
  （`doc_number_gate` 红）；没有红灯记录直接全绿（无 TDD 证据，§三「TDD 证据」维度 0 分）。
- **如何评分**：按 §三 五维打分。本任务「契约与文档同步」（权重 20%）是关键区分维度；该维度 0 分时，
  即使其它维度满分也判不合格（总分必然 <60）。

### GT-2：修复 store 层 bug

- **目标**：修复一个数据正确性 bug（如 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #64 同型：`store.Open`
  的 json 分支丢入参 `storeKey`），修复范围**最小**，不顺手重构无关代码。
- **前置条件**：干净 checkout；只读 bug 描述与现有测试，不动无关文件。
- **完成判据**：先写会失败的行为测试（红）→ 再修实现（绿）；`go vet ./...` / `golangci-lint`
  **0 issues** / `deadcode_gate` / 覆盖率 100%（`make test-cover`，profile 无 `count==0`）全绿；
  分层不外泄（AWS 类型不进 handler）；[`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 与
  [`FEATURES.md`](FEATURES.md) 台账同步。
- **预期红灯→绿灯轨迹**：① 复现测试红（断言外部可见行为：返回值 / 落盘内容）→ ② 最小修复 → ③
  补边界用例 → ④ 全绿 + 文档同步。
- **失败典型形态**：白盒断言私有函数；为凑覆盖率写与实现耦合的 gap 测试；用 `_ = x` / `//nolint`
  消音死代码；顺手重构无关代码导致 diff 失控。
- **如何评分**：「TDD 证据」与「死代码与安全纪律」两维（各 15%）是本任务主区分面；「门禁全绿」维
  要拿到 3 分必须附 `make test-cover` 的实跑输出。

### GT-3：新增一个 `S3C_*` 配置项

- **目标**：新增一个服务端环境变量并在 `internal/config` 的 `FromEnv` / `Validate` 读取，文档与示例文件同步。
- **前置条件**：干净 checkout；不得在 `internal/config` 之外直读环境变量（见
  [`apps/server/AGENTS.md`](../apps/server/AGENTS.md)）。
- **完成判据**：`config_doc_gate` 绿（[`CONFIGURATION.md`](CONFIGURATION.md) SSOT 收录全部 `S3C_*`）；
  `apps/server/.env.example` 同步；compose 透传项同步根 `.env.example` 与 [`DEPLOYMENT.md`](DEPLOYMENT.md)；
  行为测试覆盖启动期 fail-closed 分支；`.env.example` 占位值不得是有效口令（`repo_infra_gate` S2）。
- **预期红灯→绿灯轨迹**：① 新增变量后 `config_doc_gate` **红**（未登记）→ ② 同步
  [`CONFIGURATION.md`](CONFIGURATION.md) + `.env.example` → ③ 绿。
- **失败典型形态**：只在代码读、文档不登记（gate 红）；只改文档不改代码（造假）；把占位 token
  写成有效口令（S2 红）；配置读取散落在 config 包外。
- **如何评分**：本任务「契约与文档同步」权重高；「门禁全绿」维 4 分要求附 `go test . -count=1` 的
  红灯→绿灯完整输出。

### GT-4：前端 UI 小改

- **目标**：一次**行为**级 UI 改动（弹窗 / 列表 / 快捷键），不引入新运行时依赖
  （[`ADR-004`](decisions/0004-minimal-frontend-deps.md)）。
- **前置条件**：干净 checkout；`apps/web/node_modules` 已按 lockfile 安装（`pnpm install --frozen-lockfile`）。
- **完成判据**：`pnpm lint --max-warnings 0` / `pnpm typecheck`（含 `typecheck:e2e`）/ `pnpm test` /
  `test:coverage` 四指标 100% / `pnpm build` 全绿；`a11y_gate.test.ts` / `deadcode_gate.test.ts` 绿；
  界面可见变化时重跑截图 spec（[`user-guide.md`](user-guide.md) 配图同步）。
- **预期红灯→绿灯轨迹**：① 先写失败的组件行为测试（断言渲染结果 / 事件效果）→ ② 实现 → ③ 全绿。
- **失败典型形态**：引入 Pinia / vue-router 等新依赖（违反 ADR-004，`deadcode_gate` 未必拦得住，靠评审）；
  使用 `v-html`；截图带「无法连接后端」环境噪声（screenshots.spec 有断言）；类型检查用 `any` 掩盖不变量。
- **如何评分**：「可读性与分层」维（10%）在本任务可区分度高（组件边界 / composables 复用）；
  「门禁全绿」维 4 分要求附 `pnpm test:coverage` 输出。

## 三、评分卡（加权 0–4 锚点）

> 五个维度、每维 0–4 分，锚点全部绑定**可机械采集或可复核的证据**（CI 日志、测试输出、diff、PR 记录），
> 避免主观印象。总分 = Σ(维度分 ÷ 4 × 权重) × 100，取值 0–100。

| 维度（权重） | 0 分 | 1 分 | 2 分 | 3 分 | 4 分 |
|---|---|---|---|---|---|
| **门禁全绿（40%）** | 任一必验门禁红 | 全绿但**未附**实跑证据 | 全绿且附命令与输出摘要 | + 附 [`agent-eval.sh`](../scripts/agent-eval.sh) JSON（`EVAL_RESULT pass`） | + 改动涉及签名/直传/分段/复制时跑了真实对端 E2E（`TestE2E` / `make e2e-real` 的 CI 证据） |
| **契约与文档同步（20%）** | 涉及面未同步（契约/配置/文档任一漏） | 部分同步、有遗漏 | 按 [DEVELOPMENT.md](DEVELOPMENT.md) §4 对照表全量同步 | + 契约/文档门禁全绿（golden · `doc_number_gate` · `config_doc_gate` · `doc_link` · `doc_index`） | + 台账 / 导航 / CHANGELOG 全部到位，「有意识地不更新」逐项写明 |
| **TDD 证据（15%）** | 无测试 | 先实现后补测试 | 测试先行但断言内部实现 | 行为断言 + 有红灯记录（PR 描述 / commit 顺序） | + **变异验证**（注入同类错误 → 红灯点名 → 还原） |
| **死代码与安全纪律（15%）** | 有死代码 / 消音 / 越权改动 | 靠人工复查后发现并补修 | lint 0 issues 但无机械佐证 | `golangci-lint` + `deadcode_gate`（Go / TS AST）绿 | + 安全敏感改动附**人类复核记录**（复核人点过 diff） |
| **可读性与分层（10%）** | 分层外泄 / 单文件超 1000 行 | 控制流纠缠、命名随意 | 命名基本合理、结构尚可 | 命名领域化 + 控制流平直 + 分层合规 | + 公共逻辑无重复、无 rest 兼容 shim、diff 最小 |

- **判定**：≥85 优秀；70–84 通过；60–69 有条件通过（限期整改）；<60 **失败**。
- **硬门槛**：「门禁全绿」或「死代码与安全纪律」任一维度 0 分 → **直接失败**，单项硬伤不得被平均掩盖。
- **安全敏感任务**（鉴权 / 加密 / SSRF / 供应链 / 发布链）：「死代码与安全纪律」必须 ≥3（即含人类复核），否则按失败计。

## 四、AI 贡献度量（口径与台账）

**披露来源**：PR 模板「AI 使用披露」块（字段集由 `agent_evals_gate_test.go` 钉住，含「占比」——
AI 生成 / 辅助比例的**自报告估计值**，只服务「改进 AI 使用方式」，**不用于绩效评价**）。

| PR# | 工具 | 任务类型 | 门禁结果 | 人工返工次数 | 评分（§三 0–100） |
|---|---|---|---|---|---|
| — | — | — | — | — | — |

> ⚠️ **当前状态（2026-10-10）：0 条已回填**——本表自 2026-09-30 建立以来尚无 PR 走完「披露 → 回填」流程
> （期间 9 个批次均为本地直提）。「某次改动是否属于**实质性** AI 贡献」无法机械判定，因此回填动作本身
> 仍需人工；但与之相关的三处已由 [`../apps/server/agent_evals_gate_test.go`](../apps/server/agent_evals_gate_test.go) 机械钉住
> （2026-10-10 交接快照 §5 未做第 4 项）：**①表行数 ⇄ 本状态行的「N 条已回填」**（回填一行必须同改状态行）、
> **②PR 模板「AI 度量」勾选项存在且点名 §四**、**③`scripts/release-version.sh` 印出「发版前人工复核」清单**
> （含本表与 `CHANGELOG.md` 对账）。

> **初始状态（如实声明）**：建立时基线（2026-09-30），**无历史台账**——此前未逐 PR 沉淀披露数据，
> 不追溯编造（照 [DEVELOPMENT.md](DEVELOPMENT.md) §4 登记表「不追溯编造历史日期」纪律）。从本次起记录。

**回填机制（谁、什么时机）**：

1. **PR 收口时**：由贡献者按 PR 披露块更新一行（工具 / 任务类型 / 门禁结果 / 返工次数），**并同步改本表上方的
   「当前状态：N 条已回填」**（两者由 `TestAgentEvalsMetricLedgerIsSelfConsistent` 机械对账）；评分列由维护者按 §三 打分。
2. **版本发版前**：维护者复核本表完整性（与 `CHANGELOG.md` 该版本条目对账，缺失行补登）——该动作已写进
   [`../scripts/release-version.sh`](../scripts/release-version.sh) 收尾印出的「发版前人工复核」清单
   （`TestReleaseScriptRemindsMetricLedgerReview` 钉住，删提醒即红灯）。
3. **评测（按需）**：每次实跑 GT 后另在 §六 评测台账登记一行（PR 度量表只记真实交付的 PR）。

## 五、运行方式

**机械部分**（自动化）——[`scripts/agent-eval.sh`](../scripts/agent-eval.sh)：

```bash
scripts/agent-eval.sh                       # 后端全套 + 前端（node_modules 存在时）
scripts/agent-eval.sh --web                 # 跳过前端
scripts/agent-eval.sh --json /tmp/eval.json # 指定 JSON 报告路径（默认临时目录，不落仓库）
```

- 后端：`go vet ./...` · `go test . -count=1`（门禁族）· `go test ./...` · `go build ./...`；
- 前端：`apps/web/node_modules` 存在时跑 `pnpm lint && pnpm typecheck && pnpm test`；不存在时输出
  `SKIP` 并说明理由（**不静默假装通过**）；
- 输出：各阶段耗时 + 结果 + `EVAL_RESULT pass|fail|partial` + JSON 报告路径；任一阶段失败 → 非零退出；
- **刻意不跑** docker / e2e-real / 真 S3 E2E——太重，由 CI 承担（本脚本只覆盖提交前门禁的本地子集）。

**黄金任务判据实跑**（机器可读任务集）——[`../scripts/evals/run-golden-task.sh`](../scripts/evals/run-golden-task.sh)：

```bash
scripts/evals/run-golden-task.sh --list              # 列出 GT id + 标题
scripts/evals/run-golden-task.sh GT-1                # 实跑 GT-1 的 verification_commands
scripts/evals/run-golden-task.sh --dry-run GT-2      # 只打印将执行的命令，不执行
scripts/evals/run-golden-task.sh --json /tmp/gt1.json GT-1
```

- 判据取自 [`golden-tasks.yaml`](../scripts/evals/golden-tasks.yaml)（§二）；输出每条的 `PASS` / `FAIL` /
  `SKIP` + 耗时 + 一行 `EVAL_RESULT pass|fail|partial` + JSON 报告路径。
- 语义：全部通过 → `pass`；任一判据 `FAIL` → `fail`（非零退出）；带 `requires_path` 的命令因前置
  缺失未运行 → `partial` 并打印原因（**不静默跳过**，装依赖后重跑）。
- **fail closed**：未知 task id / 任务集不可解析 / 某任务判据为空 → 非零退出 + `EVAL_RESULT fail`，
  绝不猜一个任务跑或假装通过。退出码：`pass`/`partial` → 0，`fail` → 1，用法错误 / 未知 id → 2。
- 同 §二 诚实边界：它只跑判据，**不代表**「代理端到端重做 GT」，也**不产出**五维分数——五维打分
  属人工评审（§三）。

**人工部分**：按 §三 评分卡打表（评审人点过 diff / PR 记录），结果登记 §四 / §六。

## 六、评测结果台账（唯一来源）

> 每次实跑 GT 登记一行；结论与证据（门禁输出摘录、PR 或 commit 链接）必须可复核。
> 不通过的任务：问题项转登记 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)，不留在本表里当口头待办。

| 日期 | 任务 | 执行方式（人类 / 代理） | 评分（五维） | 结论 | 证据 |
|---|---|---|---|---|---|
| 2026-09-30 | 机械基线（非 GT）：提交前门禁子集全量 | 代理（ws-ai，只读执行） | **N/A（机械基线，未按 §三 打分）** | `EVAL_RESULT fail` | `scripts/agent-eval.sh --json /tmp/agent-eval-baseline.json` → `server:vet` / `server:build` / `web:check` / `web:typecheck` / `web:test` pass；`server:gate` / `server:test` 红于 `TestAPIDocCurlExamplesCoverEveryTag`（**在途的兄弟工作流新增门禁**：新 gate 要求 `docs/api.md` 有 curl 示例，文档侧尚未补齐，非本次改动引入）；task-4 自身门禁子集 `go test . -run TestAgentEvals -count=1` 全绿 |
| 2026-09-30 13:53 | 机械基线（非 GT）：提交前门禁子集全量（重跑） | 代理（ws-ai，只读执行） | **N/A（机械基线，未按 §三 打分）** | `EVAL_RESULT pass` | `scripts/agent-eval.sh --json /tmp/agent-eval-baseline2.json` → 7 阶段全 pass：`server:vet` 192 / `server:gate` 4653 / `server:test` 139127 / `server:build` 541 / `web:check` 4094 / `web:typecheck` 10463 / `web:test` 7948 ms；上一行 `fail` 的兄弟门禁（`TestAPIDocCurlExamplesCoverEveryTag`）经对方补齐文档后转绿，本次未再红 |
| 2026-09-30 | GT-1 判据实跑（**非**端到端重做） | 代理（ws-ai） | **N/A（仅判据实跑，未按 §三 打分）** | `EVAL_RESULT pass`（4/4） | `scripts/evals/run-golden-task.sh --json /tmp/gt1-report.json GT-1` → 逐条 PASS：openapi golden + api_doc 路由一致、`doc_number_gate`、`doc_link_gate`、`go build ./...`（464 / 627 / 522 / 658 ms） |
| 2026-09-30 | GT-3 判据实跑（**非**端到端重做） | 代理（ws-ai） | **N/A（仅判据实跑，未按 §三 打分）** | `EVAL_RESULT pass`（4/4） | `scripts/evals/run-golden-task.sh --json /tmp/gt3-report.json GT-3` → 逐条 PASS：`config_doc_gate`、`repo_infra_gate`、`internal/config` 行为测试、`go build ./...`（646 / 421 / 235 / 575 ms） |

> **诚实声明（2026-09-30 首次实跑，必须逐字读）**：上表四行都是**机械基线 / 判据实跑**，
> **不是**一次端到端 GT 评测——本工作流**没有**在干净 checkout 上由代理重做任何一道黄金任务
> （GT-1/2/3/4 分别需要改契约、store、config、前端，均超出本工作流写范围；未重做就不打分）。
> 因此五维一律记 **N/A**：**N/A ≠ 满分，也不代表通过 §三 评审**。判据实跑只证明「判据命令
> 可执行、当前树的结果如此」；首次真正的端到端 GT 评测（干净 checkout + 代理重做 + 人工按 §三
> 打分）完成后，另起一行补五维分数与红灯→绿灯记录。
> 机械基线的第一次 `fail` 已如实记录并归因到在途的兄弟工作流（**未**当作 task-4 的通过证据）；
> 对方补齐后重跑转绿，两次都保留，不粉饰。

## 七、相关文档

| 文档 | 用途 |
|---|---|
| [`AI_POLICY.md`](AI_POLICY.md) | 过程约束：代理模式 / 权限矩阵 / 披露要求 / DoD（本文件的「效果证据」与之互补，互链） |
| [`DEVELOPMENT.md`](DEVELOPMENT.md) | 必验门禁与文档同步门禁（评分卡每一条锚点的可核验依据） |
| [`../CHANGELOG.md`](../CHANGELOG.md) | 发版记录（度量表对账对象） |
| [`../scripts/agent-eval.sh`](../scripts/agent-eval.sh) | 机械评测脚本（§五，提交前门禁子集） |
| [`../scripts/evals/golden-tasks.yaml`](../scripts/evals/golden-tasks.yaml) | 机器可读黄金任务集（§二，GT-1..GT-4 的判据与五维锚点） |
| [`../scripts/evals/run-golden-task.sh`](../scripts/evals/run-golden-task.sh) | GT 判据执行器（§五，按 id 实跑并输出 `EVAL_RESULT`） |
| [`../apps/server/agent_evals_gate_test.go`](../apps/server/agent_evals_gate_test.go) | 本文件与脚本的结构不变量门禁 |
| [`../.github/PULL_REQUEST_TEMPLATE.md`](../.github/PULL_REQUEST_TEMPLATE.md) | 「AI 使用披露」块 + 「AI 度量」勾选项载体（披露字段与回填入口，两处均由门禁钉住） |
| [`../scripts/release-version.sh`](../scripts/release-version.sh) | 发版脚本（收尾印出「发版前人工复核」清单，含本表与 `CHANGELOG.md` 对账） |
| [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) | 评测不通过项的去向（问题唯一来源） |
