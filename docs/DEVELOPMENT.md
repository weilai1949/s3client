# 开发指南

> 面向贡献者与 AI 代理的开发规范。本文件合并了原 `agents.md` 的 TDD 约定与 `CONTRIBUTING.md` 的开发环境部分。
> 贡献流程、提交规范、环境搭建见 [CONTRIBUTING.md](../.github/CONTRIBUTING.md)；代码架构见 [architecture.md](architecture.md)。

## 1. 核心原则（TDD 优先）

1. **先写会失败的测试，再写让它通过的实现。**
2. **小步提交**：一个逻辑改动一个 commit；改动前后都要 `go test ./...` 与 `pnpm build` 全绿。**成文例外**：同日并行、且共享同一组生成物 / 文档（如 `docs/api/openapi.json`、`docs/FEATURES.md`、计数类文档）的多个批次，可合并为一次提交——强拆会产生互相依赖的半成品 commit；但该 commit 的提交信息**必须按批切片**（逐批列出改动点），评审据此按批阅读。
3. **行为驱动，不测内部实现**：断言「外部可见的行为 / 返回值 / HTTP 状态码」，不要断言私有函数或内部变量。
4. **红灯-绿灯-重构（红绿蓝）**：先看到测试因缺实现而失败（红），再让实现通过（绿），最后在测试保护下优化结构（重构）。
5. **改完代码必须同步文档**：任何一次**修复 bug** 或**新增功能**完成之后，必须更新相关文档；文档未同步 = 改动未完成，不得提交 / 合并。文档与代码属于同一个 commit（或同一 PR）。

## 2. 三类测试（按项目现有基建落地）

| 层 | 位置 | 运行方式 | 目的 |
|----|------|----------|------|
| **单元 / 行为测试** | `apps/server/internal/.../*_test.go` | `cd apps/server && go test ./...` | 用 `httptest.NewServer` 的**假 S3** 验证 handler 层逻辑（路由、参数校验、正/反例、错误码映射） |
| **真实对端 E2E** | `apps/server/internal/s3wrap/e2e_test.go` | `S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v` | 验证最硬核路径：**SigV4 签名 / 预签名直传 / 分段 Multipart 组装 / 跨 bucket 复制 / 标签 / 版本控制**。默认指向本地 RustFS |
| **真实联调浏览器 E2E** | `apps/web/e2e-real/real-backend.spec.ts` | `make e2e-real` | 真实 Go 后端（托管真实 `vite build` 产物）+ 真实 RustFS + 真实浏览器，**不 mock `/api`**：账号落库、建桶列桶、**浏览器直传**（预签名 PUT 跨源）、**OpenAPI 规范真断言**（后端带 `S3C_EXPOSE_OPENAPI=1`，生产默认 404）。mock 版 `apps/web/e2e/*.spec.ts` 覆盖不到的结合部 |
| **前端类型 + 构建** | `web` | `cd apps/web && pnpm build`（含 `vue-tsc --noEmit`） | 类型安全与可构建性；UI 改动同时保留手测/截图证据 |

### 假 S3 模式（handler 测试）
- 用 `httptest.NewServer` 模拟 S3，采用 **path-style**（请求落在 `/{bucket}/{key}`）。
- 依 `r.URL.Query().Has("acl"|"tagging"|"versions"|"location"|"versioning")` 与 HTTP 方法分发返回的 XML。
- 返回标准 S3 XML（`<ListVersionsResult>`、`<AccessControlPolicy>`、`<Tagging>` 等），并按需返回特定错误码（如 `NoSuchTagSet`）验证容错。

### 真实 RustFS 联调
- 需要时用 `docker compose up -d rustfs`（默认 `rustfsadmin/rustfsadmin`，S3 API 9000、控制台 9001）。
- E2E 测试用 `S3CLIENT_E2E=1` 门控，普通 `go test ./...` 不会执行，CI 因此不受影响。

### 真实后端 + RustFS 浏览器联调（历史任务 #37，已闭环）
- #37 已于 2026-09-22 闭环并从 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) 移除，归档证据见 [`FEATURES.md`](FEATURES.md) §V；当前待办以 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（问题）与 [`ROADMAP.md`](ROADMAP.md)（方向）为准。
- 一条命令：`make e2e-real`（脚本 [`scripts/e2e-real.sh`](../scripts/e2e-real.sh)）。它会起一份**独立** RustFS 容器、构建真实前端产物与后端、起真实后端托管产物（env 显式带 `S3C_EXPOSE_OPENAPI=1`——生产默认不暴露 `/api/openapi.json`，评审 R9 迁入的契约真断言需要显式打开）、跑 `pnpm e2e:real`，最后 `trap` 自动清理。
- **该脚本是这套编排的唯一来源**：本地 `make e2e-real`、GitHub Actions 与 GitLab CI 都调用它，各自只负责「装工具链 / 装浏览器系统依赖」。门禁 `TestRealE2EUsesSharedScript` 断言两侧 CI 都**实际调用** `bash scripts/e2e-real.sh`（仅出现在 `paths:`/`changes:` 里不算），防止又抄一份编排而漂移。
- **必须给 RustFS 配 `RUSTFS_CORS_ALLOWED_ORIGINS`**（脚本自起时已默认配好，GitLab service 变量里也配了）：浏览器直传（预签名 PUT）是页面 → S3 的**跨源**请求，缺 CORS 会被浏览器拦下。注意 curl / Playwright `APIRequestContext` **不经 CORS**，只用它们验证会「假绿」——所以用例特意驱动真实浏览器 XHR。
- 复用外部对端（GitLab service / 已起的实例）：`RUSTFS_ENDPOINT=http://rustfs:9000 bash scripts/e2e-real.sh --no-rustfs`（此时脚本不管理容器生命周期）。
- 排查用 `make e2e-real E2E_REAL_ARGS=--keep`（保留容器与后端进程）或 `--skip-build`（复用已有产物）。端口默认 5000 / 9000，可用 `SERVER_PORT` / `RUSTFS_PORT` 覆盖。
- 机械门禁：`TestRustFSImageIsConsistentlyPinned`（脚本默认值 / compose / GitLab service 三处镜像版本必须一致）、`TestRealE2EArtifactsExist`（联调 spec 出现 `page.route(` 即红灯）、`TestE2ESourcesAreTypechecked`、`TestLocalRealE2ETargetExists`。

### E2E 源码的静态检查
- `e2e/`（mock 版）与 `e2e-real/`（真实联调版）**不在**主 `tsconfig.json` 的 `include` 内，因此长期零静态检查。现补 [`apps/web/tsconfig.e2e.json`](../apps/web/tsconfig.e2e.json) + `pnpm typecheck:e2e`，并把两个目录纳入 `pnpm lint`（`eslint src e2e e2e-real`）。
- 两套 CI 的 `web` job 与本地 `make check`（`web-typecheck-e2e`）都跑它；门禁 `TestE2ESourcesAreTypechecked` 防漏挂。

### 原生 fuzz（有界，**不是** PR 门禁）

解析面的输入空间探索用 Go stdlib `testing.F`（不引入新依赖）：`internal/s3wrap`（端点归一化 / SSRF）、
`internal/store`（`S3C2` / `S3C3` 信封——磁盘字节不可信）、`internal/handler`（桶策略 JSON、桶名与
下载文件名边界）。**PR 只跑种子语料**（`go test ./...` 内的 `-run Fuzz`），保证语料不退化成死代码；
真正的探索在 [`.github/workflows/fuzz.yml`](../.github/workflows/fuzz.yml)（周一 03:00 UTC + 手动，
`-fuzztime` 有界，崩溃上传语料）。覆盖率门禁看不见非法输入空间，两者互补。

## 3. 必验门禁（每次改动提交前）

```bash
cd apps/server && go vet ./... && go test ./...   # 后端
cd apps/server && go build ./...                   # 后端可构建
cd apps/web && pnpm test && pnpm build             # 前端单测 + 类型检查 + 构建
# 涉及签名/直传/分段/复制/标签/版本时，额外跑真实 RustFS E2E
cd apps/server && S3CLIENT_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v
# 涉及前端 / 后端接口 / 直传时，额外跑「真实后端 + 真实 RustFS + 真实产物」浏览器联调
# （docker 自动起一份 RustFS，跑完自动清理；不 mock /api）
make e2e-real
# 或 make test-all（后端 + 前端单测）
make test-cover                               # 后端覆盖率 100% 门禁（CI 同款检查）
make lint                                     # golangci-lint（errcheck / staticcheck / govet / ineffassign / unused / gosec / nolintlint），必须 0 issues
# 改到桌面端依赖时（需本机已装 cargo-audit）：RustSec 审计，CI desktop job 同命令
make rust-audit
```

> **编辑器约定**：仓根 [`.editorconfig`](../.editorconfig) 声明**已被工具链强制**的最小格式规则
> （Go / Makefile = tab，TS / Vue / YAML / shell = 2 空格，Rust = 4 空格，统一 LF + 末行换行；
> `*.md` 关闭行尾空格裁剪以保留两空格硬换行）。它只影响编辑器内即时行为、**不新增 CI 门禁**，
> 也不与 `gofmt` / `rustfmt` 打架。
>
> **契约与文档门禁的落点**（改接口 / 改文档数字时相关）：`internal/handler/` 下
> `openapi_request_fields_test.go`（请求体字段全量）、`openapi_query_params_test.go`（query 参数
> 四种读取口径）、`openapi_path_params_test.go`（path 参数 ⇔ handler `PathValue` 双向）、
> `openapi_semantics_test.go`（类型 / required / 枚举）、
> `openapi_response_contract_test.go`（共享 schema + **端点级**响应）、`api_doc_test.go`（docs 双向
> + 路由行必须顶格）、`openapi_spec_file_test.go`（提交版 [`api/openapi.json`](api/openapi.json)
> ⇔ 运行时规范语义一致，含 `-update-openapi-spec` 再生成）；仓库级 `repo_infra_gate_test.go`
> （CI / 发布链 / 工具链 pin / **所有 action 必须 pin 完整 commit SHA** /
> **nginx 访问日志必须带请求 ID**，见 `TestNginxAccessLogCarriesRequestID` /
> **告警规则引用的指标与错误码必须真实存在**，见 `TestPrometheusRulesReferenceRealMetrics` /
> **桌面发布必须产出并上传 SBOM**，见 `TestReleaseWorkflowProducesDesktopSBOM` /
> **cosign 签名必须接线且两处 pin 同一 SHA**，见 `TestCosignSigningIsWiredAndConsistentlyPinned` /
> **账号库 JSON Schema 必须与 `model.Account` 双向一致**，见 `TestAccountStoreSchemaMatchesModel` /
> **`make gcl GCL_JOBS=…` 必须真能跑**（Makefile 的 gcl recipe 需用 `env -u` 摘掉 `GCL_*`
> 环境变量，否则 gcl 的 yargs `.env('GCL')` 前缀映射会把它当成同名 CLI 选项而报
> `Unknown argument`），见 `TestMakefileGclStripsEnvPrefixVars`）、
> `doc_number_gate_test.go`（md 叙述性数字）、`config_doc_gate_test.go`（配置 SSOT
> [`CONFIGURATION.md`](CONFIGURATION.md) ⇔ `internal/config` 读取的 `S3C_*` 变量全量）、
> `env_example_gate_test.go`（根 `.env.example` ⇔ 三个 compose 文件 `${VAR}` 透传面双向一致 +
> 两份 `.env.example` 不示范「值后行内注释」——`#` 只在行首才算注释）、
> `dev_scripts_gate_test.go`（① 受管 web dev server 的 `.run/web.pid` 必须指向 vite 本身 +
> 启动带 `--strictPort`；② 启动流程不得执行 `go mod tidy`——`make dev` 曾每次改写 `go.sum`，
> 依赖同步只留显式 `make tidy`）、
> `deadcode_gate_test.go`（消音式死代码 AST 判定 + `_test.go` 导出符号 + **生产代码导出符号零引用**）、
> `ci_consistency_gate_test.go`（两套 CI 的 job / 命令 / 版本 pin 逐项一致 + checkout 深克隆与前端依赖
> 前置——评审 2026-10-09 C1/C2，见下节；并钉真 RustFS Go E2E 的 PR 触发面覆盖整个后端，
> `TestRustFSE2ETriggersCoverWholeBackend`——O11 #82）、
> `doc_ci_drift_gate_test.go`（文档里 **`<workflow>` × `<job id>`** 形式的引用其 job id 必须真实存在——`perf-budget` 事故；
> 覆盖率排除项必须完整写出——O11 #82）、
> `doc_status_drift_gate_test.go`（**状态陈述**三连：带 ⬜/⏳ 的 `§三 #N` 引用必须指向 ROADMAP §3.1/§3.2 现存条目；
> `OPERATIONS.md` §4.3 trace 行不得与同文件 §3.4 OTel 实现自相矛盾；`FEATURES.md` 里「仍开放 / 仍未处置」
> 必须带日期时点锚——2026-10-10 实测发现的悬空编号 + 自相矛盾两处漂移）；
> `doc_alert_drift_gate_test.go`（**告警表 ⇔ 规则文件**：`OPERATIONS.md` §4.2 表行第一列的
> `S3Client*` 名与 `deploy/prometheus/s3client.rules.yml` 的 `- alert:` 集合**双向相等**——
> 实测两边各 13 条纯属巧合、集合并不相同，见 A3）、
> 前端半边落点是四份：`apps/web/src/deadcode_gate.test.ts`（API 公开面 + **非 API 模块运行期导出 /
> 孤儿模块 + 类型导出零引用**（`export type` / `interface`——评审 R7 后半区从声明的盲区升级为真断言），
> 引用计数走 **TS AST**——注释与字符串字面量不算引用）、
> `apps/web/src/a11y_gate.test.ts`（**源码形态**：`:focus-visible` 列表必须含 `textarea`、
> `prefers-reduced-motion` 媒体查询必须关闭 animation / transition）、
> `apps/web/src/vite_env_guard.test.ts`（宿主 `NODE_ENV` 隔离）、
> `apps/web/src/api/generated.gate.test.ts`（**生成物新鲜度**：`pnpm gen:api --check` 直接跑仓库自己的
> `scripts/gen-api.mjs`，[`api/openapi.json`](api/openapi.json) 与提交的 `src/api/schema.d.ts` /
> `src/api/operations.ts` 不逐字节一致即红灯，另有结构自检防「空表永远绿」——改 spec 忘了重跑
> `pnpm gen:api` 就是红的；见 ROADMAP §三 3.2 #10 / FEATURES §BN）。
> 各文件的**断言范围与残留**写在文件头。
>
> 另有三道门禁落在**子包** `apps/server/internal/**`（不在根包，故不属上面「包根 `*_gate_test.go` 全量」口径）：
> `internal/handler/error_echo_gate_test.go`（**错误文案不回显用户输入**——生产代码里
> `writeErr(..., 400, "字面量"+变量)` 这类「固定前缀 + 用户输入拼接」一律红灯，确需回显须改固定文案并落日志）、
> `internal/handler/migrate_sync_gate_test.go`（`/api/migrate/sync` 源端列举失败必须回错误状态，
> 不得回 `200 {scanned:0}`——否则会被读成「没有需要同步的内容」）、
> `internal/service/sync_list_gate_test.go`（**列举循环必须有界**：`NextContinuationToken` 不前进
> 或触页上限即停，否则同步端点在任务超时前一直挂着；见 [`archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §B5 / §B6）。
>
> **门禁自身也用 AST 而非裸正则**：判断「handler 是否解码请求体」「是否读取 path 参数」、
> `PathValue` 的动态键与 **handler 调用闭包**（`h.xxx()`）时，裸字符串匹配会被注释与字符串字面量
> 骗过（实测），故这些判定走 `go/parser` 语法树（闭包遍历统一走 `parseBodyCalls` /
> `handlerMethodCalls`）。query 参数的字段抽取仍为正则，但绑定变量后的非字面量键
> （`q := r.URL.Query(); q.Get(name)` / `q[k]`）已由 `hasDynamicQueryRead` 显式红灯，不再静默逃逸。
> 每条判定都配一个用**合成源码**写的口径测试（如 `TestDecodesBodyIgnoresCommentsAndStrings`、
> `TestParseBodyCallsIgnoresCommentsAndStrings`、`TestPathParamDynamicReadUsesAST`、
> `TestFindReadJSONTargetIgnoresCommentsAndStrings`），不依赖「仓库里正好有一个反例」来证明门禁有效。
> `//nolint` 消音由 `nolintlint` 拦截（必须写明具体 linter 与理由）。
>
> **2026-09-30 新增的门禁落点**（AI 时代文档补强的机械保证，全部经变异验证）：
> `openapi_examples_gate_test.go`（有请求体的 operation 必须有请求示例、每个 operation 必须有 2xx 示例、
> 示例字段须过 schema 形状校验，且 [`api.md`](api.md) 的 curl 示例覆盖全部 tag）、
> `data_model_gate_test.go`（[`data-model.md`](data-model.md) ⇔ 反射 `model.Account` + `store.Open` 的 switch 驱动名）、
> `contrast_gate_test.go`（[`accessibility.md`](accessibility.md) §5.5 表值与计数 ⇔ `styles.css` token 重算）、
> `grafana_dashboard_gate_test.go`（[`../deploy/grafana/s3client.dashboard.json`](../deploy/grafana/s3client.dashboard.json)
> 的指标 / `code` / recording rule 必须真实存在）、`bench_budget_test.go`（PresignPut / 加密写 / 账号写 O(n)
> 的分配与耗时预算）、`security_txt_gate_test.go`（[`../.well-known/security.txt`](../.well-known/security.txt)
> 必填字段 + `Expires` 未过期）、`en_docs_gate_test.go`（[`en/README.md`](en/README.md) 每篇须声明中文
> SSOT 来源与 revision，且可从英文导航到达）、`agent_evals_gate_test.go`（黄金任务集机器可读规格
> [`../scripts/evals/golden-tasks.yaml`](../scripts/evals/golden-tasks.yaml) 与
> [`AGENT_EVALS.md`](AGENT_EVALS.md) 的 id / 标题逐字一致）。追加两道**文档维护自身**的门禁
> （2026-09-30，ROADMAP §三 3.2 #19 落地）：`llms_size_gate_test.go`（[`../llms.txt`](../llms.txt)
> 里目标 >200 KB 的链接必须就地标 `⚠️ 超大` 体量预警，防 LLM 按索引整读被截断）、
> `doc_review_gate_test.go`（本文件 §4「文档登记表」带「N 个月」周期的行超过
> `最后复审 + 周期` 即红灯——复审从「建议值」升级为**有机械提醒**，见该节说明）。前端**渲染态**无障碍由
> [`../apps/web/e2e/a11y.spec.ts`](../apps/web/e2e/a11y.spec.ts)（axe，4 状态 + 1 条有效性自检）承担，
> 见 [`accessibility.md`](accessibility.md) §5.2。

### CI 双平台一致性

同一套门禁同时落在 **GitHub Actions** 与 **GitLab CI**，两边 job、命令、门禁阈值必须一致；
改任一侧都要同步另一侧（含下表），否则会漂移成「GitHub 绿 / GitLab 红」。
双侧一致性由 `apps/server/ci_consistency_gate_test.go` 机械钉住（评审 2026-10-09 C1）。两个易碎前提同为该批评审点名，两侧都已落地：
**① 所有 checkout 必须 `fetch-depth: 0`**（`changelog_tag` 门禁从 `.git` 读 v* tag，浅克隆读不到即必红——C2）；
**② Go job 必须先真装前端依赖**（GitHub `server` job 加 pnpm/setup-node + `pnpm install --frozen-lockfile`，
GitLab `server` 用 nodejs tarball + `npm install -g pnpm@9.15.0` + `.pnpm-store/` cache）——根包 `agent_evals_gate_test.go`
的判据前置是 `apps/web/node_modules` 存在（gitignore 构建产物，**不得 `mkdir` 伪造**——C1）。

| GitHub Actions | GitLab CI job | 门禁内容 |
|---|---|---|
| `ci.yml` · `server` | `server` | gofmt / go vet / govulncheck / golangci-lint v2.13.2（**0 issues**）/ `go test -race` + 覆盖率 100% / build |
| `ci.yml` · `web` | `web` | `pnpm lint`（`--max-warnings 0`，含 E2E 源码）/ typecheck（src + E2E 两份）/ `test:coverage`（100%）/ build |
| `ci.yml` · `docker` | `docker` | `docker build` + Trivy CRITICAL/HIGH 失败门禁（`.trivyignore`） |
| `ci.yml` · `desktop` | `desktop` | `cargo check --locked` + `cargo audit`（RustSec，有漏洞即红灯；webkit/gtk 系统依赖） |
| `ci.yml` · `desktop-build`（仅 `workflow_dispatch`） | `desktop-build`（`when: manual`，仅 `web` 源） | `tauri build --no-bundle` |
| `ci.yml` · `publish` | **不镜像** | 推送镜像到 GHCR（`needs: docker`，Trivy 通过才推；`if: != 'pull_request'` 即 push / dispatch 才推），并产出 **CycloneDX SBOM 文件（作为 artifact）+ BuildKit provenance/SBOM attestation + GitHub 产物证明**。GitLab 侧未配置 registry，故无对应 job；门禁 `TestGitHubWorkflowPushesImage` |
| `codeql.yml` · `analyze` | `semgrep-sast`（GitLab 原生 SAST） | 两侧语言覆盖一致（Go + JS/TS）。**都是报告型、非阈值门禁**：CodeQL 用 `security-and-quality` 查询集；GitLab 侧由 `include: template: Jobs/SAST.gitlab-ci.yml` 引入，`semgrep-sast` 继承 `.sast-analyzer` 的 `allow_failure: true`。逐 `uses:` 的 SHA pin 由 `TestWorkflowActionsAreShaPinned` 守住 |
| `scorecard.yml` · `analysis` | **不镜像** | OpenSSF Scorecard 仓库健康度评分：schedule 周六 02:00 UTC + `workflow_dispatch`；顶层 `permissions: read-all`，仅 analysis job 持 `security-events: write` + `id-token: write`（发布到 api.scorecard.dev 所需 OIDC）。GitLab 无等价原生产品（其 SAST / Dependency Scanning 模板分别扫仓库内代码与发布时点全量依赖，均非本检查）——差异登记见 [`threat-model.md`](threat-model.md) §5.5 |
| `dependency-review.yml` · `dependency-review` | **不镜像** | PR 期依赖 diff 审查（GitHub 依赖图 API）：每个 `pull_request`（含 dependabot PR），`permissions: contents: read`。GitLab 无等价原生产品；未镜像登记见 [`threat-model.md`](threat-model.md) §5.5 |
| `perf.yml` · `bench` | **不镜像** | 性能预算门禁（`bench_budget_test.go`：分配数 / 字节的确定性断言为主 + 极宽的耗时兜底）+ 原始 benchmark artifact 留存；`push`/`pull_request` 路径命中 + 周一 03:00 UTC schedule + `workflow_dispatch`。GitLab 无等价原生产品；口径与「测什么 / 不测什么」见 [`PERFORMANCE.md`](PERFORMANCE.md) §4.2 |
| `fuzz.yml` · `fuzz` | **不镜像** | 原生 fuzz 有界轮跑（stdlib `testing.F`，`-fuzztime` 可配，崩溃上传语料）；周一 03:00 UTC schedule + `workflow_dispatch`。**刻意不进 PR 门禁**（不定长会拖合并），PR 侧只跑种子语料。GitLab 无等价原生产品 |
| `e2e.yml` | `rustfs-e2e` | 真 RustFS 对端 `TestE2E`（GitLab service 容器替代 compose） |
| `e2e-playwright.yml` | `playwright-e2e` | 构建产物 + vite preview + Playwright chromium |
| `e2e-real.yml` | `e2e-real` | **真实 Go 后端（托管真实构建产物）+ 真实 RustFS + 真实浏览器**，不 mock `/api`（含浏览器直传）；本地与两套 CI 共用 `scripts/e2e-real.sh` |
| `release-desktop.yml` | **不镜像** | 发布目标是 GitHub Release（tauri-action + `gh release upload`），需 Windows/macOS runner 与 `GITHUB_TOKEN`；同时为三平台安装包出具**产物证明**（`attest-build-provenance`） |

> **GitLab 侧 SAST 的落地方式与取舍**（2026-09-29 收口 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #66）：
> GitLab 用官方 `Jobs/SAST.gitlab-ci.yml` 模板（开源分析器，**Free 档即可用**；
> Advanced SAST / MR 内联标注属 Ultimate）。三条落地约束：
> ① 本文件自定义了 `stages`，**必须把 `test` 列进去**，否则模板里 `stage: test` 的 job
>    会因「stage 不存在」直接配置报错；
> ② Go 与 TypeScript/JavaScript 都由 semgrep 分析器覆盖，实测只新增 `semgrep-sast`
>    **一个** job（其余分析器由各自 `exists:` 规则决定，本仓库没有对应语言文件）；
> ③ `allow_failure: true` 继承自模板的 `.sast-analyzer`——分析器自身崩溃不拖垮流水线，
>    发现项也从不置非零退出码。**这与 GitHub 侧 CodeQL 同构**：两边都是「报告」而非
>    「阈值门禁」，真正的阈值门禁仍是两套 CI 各自的 Trivy。
>
> **此前「不镜像」的理由已作废**：原记「另一套工具链，且无法在 `make gcl` 本地执行器里验证」。
> 实测**可验证**——`make gcl-list` 能正常解析并列出 `semgrep-sast`（stage `test`，见上方
> §「本地验证 GitLab 流水线」）。故该理由不再成立，改为**已镜像**。

**触发事件也要对齐**（各 workflow 的 `on:` 并不相同，别假设一致）：

| 事件 | GitHub 会跑 | GitLab 会跑 |
|---|---|---|
| push 到 main/develop | `ci.yml` 五个 job（`server`/`web`/`docker`/`desktop`/**`publish`**）+ `codeql.yml` | `server` / `web` / `docker` / `desktop` / **`semgrep-sast`** |
| pull_request | `ci.yml` 四个 job（**`publish` 不跑**，`if: != 'pull_request'`）+ `codeql.yml` + `dependency-review.yml` + 路径命中时的三个 E2E | 同上 + 命中的 E2E |
| workflow_dispatch / web | `ci.yml` 全部 **6 个 job**（含 `desktop-build` 与 `publish`）+ `codeql.yml` / `e2e.yml` / `e2e-playwright.yml` / `e2e-real.yml` + `scorecard.yml` / `perf.yml` / `fuzz.yml`（可带 `fuzztime` 输入）；`release-desktop.yml` 需 `tag` 输入 | `server` / `web` / `docker` / `desktop` / `rustfs-e2e` / `playwright-e2e` / `e2e-real`（**7 个自动 job**）+ **`semgrep-sast`**；`desktop-build` 为 manual |
| schedule | `codeql.yml`（周日）+ `scorecard.yml`（周六）+ 三个 E2E（周一/三/五）+ `perf.yml` / `fuzz.yml`（周一 03:00） | 三个 E2E + **`semgrep-sast`**（周一/三/五） |

> **`semgrep-sast` 不进上表的「本文件各 job 的 rules」体系**：它的 `rules` 来自被 `include` 的
> GitLab 官方模板，**只受「流水线是否创建」约束**（即 `workflow:` 块），不受本文件里
> `.ci-trigger` / `.e2e-*-trigger` 那套控制。因此**只要流水线建起来它就跑**——上表按这个口径列。
> 这也是它与 GitHub 侧 `codeql.yml`（有自己独立的 `on:`）的一处结构性差异：
> 那边是独立 workflow、有自己的触发事件；这边是同一流水线内的一个 job。

本地验证 GitLab 流水线（无需 GitLab 实例，用 [gitlab-ci-local](https://github.com/firecow/gitlab-ci-local) 的 docker executor 跑真实 job；版本 pin 在 `Makefile` 的 `GCL`）：

```bash
make gcl-list                         # 列出将要运行的 job
make gcl GCL_JOBS=web                 # 跑单个 job（web / server / rustfs-e2e …）
make gcl GCL_JOBS='--stage ci'        # 跑一个 stage
make gcl-docker                       # docker job（.gitlab-ci-local-env 已挂宿主 docker.sock）
# 国内网络：复制变量示例，或一次性覆盖 Trivy DB / Go / npm 源
cp .gitlab-ci-local-variables.yml.example .gitlab-ci-local-variables.yml
# 或：make gcl-docker GCL_EXTRA='--variable TRIVY_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-db:2'
```

执行器差异需知：**①** gitlab-ci-local 只把**已跟踪（tracked）**文件同步进容器，新增被 job 读取的文件要先 `git add`，否则本地结果与线上不一致；**②** `docker` job 走宿主 daemon——正式 runner 已在 `runners.docker.volumes` 挂 socket，本地由已提交的 [`.gitlab-ci-local-env`](../.gitlab-ci-local-env) 默认 `VOLUME=/var/run/docker.sock:...`（`make gcl-docker` / 裸 `npx` 都会读到）。`rustfs-e2e` 无需任何额外变量：gitlab-ci-local 会把 service 作为**独立容器**放进同一 docker network 并注册别名，`rustfs:9000` 直接可用（**不要**改成 `127.0.0.1`，回环在 job 容器内指向自己，实测不可达）。

> **Trivy DB 超时**：`docker` job 里 Trivy 默认拉 `mirror.gcr.io/aquasec/trivy-db:2`，国内常连不上。设环境变量 / CI 变量 `TRIVY_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-db:2` 即可（trivy 原生读取，script 不用改）。团队示例见 [`.gitlab-ci-local-variables.yml.example`](../.gitlab-ci-local-variables.yml.example)；复制为 `.gitlab-ci-local-variables.yml`（已 gitignore）后本地自动生效。

> ⚠️ **别把 GitHub 的 `&& exit 0` 直接搬到 GitLab**：GitLab 把整个 `script` 拼成**一个 shell 脚本**执行，`exit 0` 结束的是**整个 job**。`rustfs-e2e` / `playwright-e2e` 的就绪轮询若照搬 GitHub 的 `curl … && exit 0`，会在服务就绪后立刻退出——**测试一条没跑却报 PASS**（GitHub 每个 `run:` 是独立 step，所以那边写法没问题）。本仓库统一用「置标志位 + `break`」跳出循环。

> **Trivy 镜像名坑**：Docker Hub 上的仓库是 `aquasec/trivy`，`aquasecurity/trivy` **只存在于 `ghcr.io`**。写成 `aquasecurity/trivy` 时 `docker run` 拉不到镜像会以 **exit 125** 退出，看起来像「扫出漏洞导致失败」（漏洞门禁是 exit 1），实际是镜像名错。两套 CI 现均用 `aquasec/trivy:0.74.0@sha256:62b1e65e8869bc4b4c6aa4fa2b21595256c7c2f6018a9d9ad61caf87187c1969`（tag + digest 双 pin；`repo_infra_gate_test.go` 的 `TestTrivyImageIsVersionAndDigestPinned` 守住两套 CI 一致且必须带 digest）。`--vuln-type` 已 deprecated，改用 `--pkg-types os,library`（0.74.0 的 `trivy image --help` 与 `pkg/flag/package_flags.go` 已核实）。

> ⚠️ **关于覆盖率 100% 门禁**：项目 CI 对后端与前端均设有 100% 覆盖率门禁。后端直接检查
> profile 中是否存在 `count==0` 的语句块，而不是比较 `total` 百分比——后者只有 1 位小数，
> 99.96% 会被四舍五入显示成 100.0% 而漏过回退。注意 100% 是「达到」而非「自然覆盖」——
> 前端排除项为 `src/main.ts` / `src/env.d.ts` / `src/**/*.test.ts` / `src/i18n/messages/**`
> （纯数据字典，另有键完整性门禁）/ `src/assets/**`，见 [`apps/web/AGENTS.md`](../apps/web/AGENTS.md)
> 与 `apps/web/vite.config.ts`；`src/i18n/index.ts` 的读写与回退逻辑**纳入统计**（由
> `doc_ci_drift_gate_test.go` 的 `TestDevelopmentDocumentsAllCoverageExclusions` 钉住不漏写）。
> **写测试请以行为价值为先**（断言外部行为），不要为凑覆盖率而写
> 与实现耦合的测试；遇到确实不可达的分支，正确做法是**删除死代码**，而不是写 gap 测试把它
> 「测活」（2026-09 后端冲 100% 时即据此清掉了 `jobsList` 的 nil 兜底）；反之，若某条兜底
> 能由公开 API 触发（如 `JobProgress.Status` 是 `omitempty`，`Emit` 允许不带状态），则应
> 保留并用行为断言覆盖，不要为数字把它删掉。

## 4. 文档同步门禁（改完代码必须更新文档）

任何「修复 bug」或「新增功能」完成之后，必须更新对应文档，否则视为改动未完成。动手前先按下表确认本次改动会触碰哪些文档面，与代码一起改。

| 改动类型 | 必须同步的文档 |
|----------|----------------|
| 前端使用方式 / 界面 / 快捷键 / 截图 | [`README.md`](../README.md)（必要时补截图）+ [`en/index.md`](en/index.md)（英文版同 PR 同步） |
| 后端接口、请求体、响应字段、状态码 | [`api.md`](api.md) + `apps/server/internal/handler/openapi_register_*.go`（并跑契约测试） |
| **md 正文里「N 个 `/api/*` 端点」这类数字** | 由 `apps/server/doc_number_gate_test.go` 机械校验（真值取自 `routes.go` 的 `mux.HandleFunc` 注册数）；新增此类声明时在 `docNumberClaims` 登记 |
| 错误码 / 错误文案 | [`errors.md`](errors.md) |
| **任何**新功能或 bug 修复 | [`CHANGELOG.md`](../CHANGELOG.md) 的 `[Unreleased]` 段（Keep a Changelog：Added / Fixed / Changed） |
| 发版 / 打 tag / 改名版本段 | [`CHANGELOG.md`](../CHANGELOG.md) 顶部「tag ↔ 版本段对应关系（唯一台账）」（快照 tag / 快照段登记；正式版本段由 `changelog_tag_gate_test.go` 正向校验）+ `scripts/release-version.sh`（缺段即 exit 1） |
| 已实现 / 已修复能力的台账 | [`FEATURES.md`](FEATURES.md) |
| 待办事项状态变化 | **问题**（缺陷 / 阻塞 / 技术债）→ [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)；**方向**（功能候选 / 版本级）→ [`ROADMAP.md`](ROADMAP.md) §三。两处各自唯一来源，同一事项只登记一处 |
| 版本级规划 / 优先级 | [`ROADMAP.md`](ROADMAP.md)（不做逐条流水账） |
| 分层 / 模块边界 / 目录结构 | [`architecture.md`](architecture.md)；重大决策另加 [`decisions/`](decisions/index.md) ADR；architecture §7「关键取舍」表每行必须含 ADR 链接（由 `apps/server/adr_coverage_gate_test.go` 守住） |
| 环境变量 / 配置项 | [`CONFIGURATION.md`](CONFIGURATION.md)（**SSOT**）+ 相应示例文件：compose 透传项改根 [`.env.example`](../.env.example)，服务端可选项改 [`apps/server/.env.example`](../apps/server/.env.example)（分工口径见 CONFIGURATION.md 开头）+ [`DEPLOYMENT.md`](DEPLOYMENT.md) + `README.md` 摘要 |
| 部署 / 镜像 / compose / 发布流程 | [`DEPLOYMENT.md`](DEPLOYMENT.md) |
| 安全策略 / 威胁模型 / 加固 / 依赖与许可证 | [`threat-model.md`](threat-model.md) 与 [`SECURITY.md`](../.github/SECURITY.md)；依赖增删改后跑 [`../scripts/gen-third-party-licenses.sh`](../scripts/gen-third-party-licenses.sh) 重新生成 [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md)（门禁 `TestThirdPartyLicensesAreComplete`） |
| 代理权限边界 / 什么必须人类确认 / MCP 工具权限 / AI 披露 | [`AI_POLICY.md`](AI_POLICY.md) |
| AI 评测任务集 / 披露占比口径 / 贡献度量 | [`AGENT_EVALS.md`](AGENT_EVALS.md)（黄金任务集 / 评分卡 / 台账）+ PR 模板「AI 使用披露」块（两处字段集由 `agent_evals_gate_test.go` 钉住一致） |
| 仓库导航（新增 / 改名核心文档时） | [`llms.txt`](../llms.txt) |
| 开发流程 / 门禁 / 测试命令 | [`AGENTS.md`](../AGENTS.md)（代理入口）+ 本文件 + [`CONTRIBUTING.md`](../.github/CONTRIBUTING.md) |
| 接口与请求/响应字段 | [`api.md`](api.md) + `apps/server/internal/handler/openapi_register_*.go`（并跑契约测试）+ **重新生成 [`api/openapi.json`](api/openapi.json)**（见下行） |
| 提交版 OpenAPI 规范 | [`api/openapi.json`](api/openapi.json)——改 handler / 注册表后必须 `go test ./internal/handler/ -run TestCommittedOpenAPISpecMatchesRuntime -update-openapi-spec` 重新生成，否则门禁红灯 |
| 运维 / 告警 / 备份恢复 / 容量 / 事故响应 / 事故复盘 | [`OPERATIONS.md`](OPERATIONS.md) + [`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md)（复盘格式；填写完成的记录按下方「归档」条冻结进 `archive/`） |
| 告警阈值 / SLI 表达式 / 指标名 / `code` 标签取值 | [`OPERATIONS.md`](OPERATIONS.md) §4 **与** [`../deploy/prometheus/s3client.rules.yml`](../deploy/prometheus/s3client.rules.yml)（**必须同改**；规则文件由 `TestPrometheusRulesReferenceRealMetrics` 校验指标与错误码真实存在） |
| 账号存储格式（`model.Account` 字段增删改） | [`api/accounts.schema.json`](api/accounts.schema.json)（由 `TestAccountStoreSchemaMatchesModel` 反射比对，漂移即红灯）+ [`compatibility.md`](compatibility.md) §4（S3C2 / S3C3 信封字节布局）+ [`data-model.md`](data-model.md)（汇总地图；与代码不一致时按其 §0 回退权威来源） |
| 性能特征 / 热路径 / 新增基准 | [`PERFORMANCE.md`](PERFORMANCE.md) + 对应包的 `bench_test.go` |
| 用户可见的操作方式 / 界面用法 / 快捷键 | [`user-guide.md`](user-guide.md) + [`README.md`](../README.md)（界面改动需重新生成截图：`cd apps/web && pnpm build && pnpm exec playwright test screenshots.spec.ts`，产物落在 [`images/`](images/)） |
| 版本兼容性 / 弃用 / 支持窗口 / 客户端支持矩阵 | [`compatibility.md`](compatibility.md) §6.2（矩阵每行状态带源码 / CI 依据；改构建目标（vite `build.target` / tsconfig `target`）、浏览器特性依赖（File System Access / `:focus-visible` 等）、Playwright 浏览器项目或发布矩阵（release-desktop.yml）时必须同步复核对应行） |
| 领域术语 / 内部自造词 | [`glossary.md`](glossary.md) |
| 翻译 / 新增语言 / 文案 key | [`i18n.md`](i18n.md) |
| 无障碍（ARIA / 键盘 / 焦点 / 主题） | [`accessibility.md`](accessibility.md) |
| 支持渠道 / 治理 / 决策与发布权 | [`../.github/SUPPORT.md`](../.github/SUPPORT.md) · [`../.github/GOVERNANCE.md`](../.github/GOVERNANCE.md) |
| 文档命名 / 存放位置 / 归档 / 导航 | 本文件 §4 + [`README.md`](README.md)（docs 导航 SSOT）+ [`../README.md`](../README.md)「文档」段 + [`../llms.txt`](../llms.txt) + [`archive/index.md`](archive/index.md) |
| **文档登记表（owner / 复审周期 / 最后复审）** | 本文件 §4 下方「文档登记表」——**新增 / 改名 / 归档任何文档时同 PR 登记一行** |
| 数据模型 / 存储格式（字段 / 驱动 / 信封） | [`data-model.md`](data-model.md)（**地图**；冲突时按其 §0 回退到 schema / ADR / 代码权威来源） |
| 告警 / SLI 仪表盘（Grafana） | [`../deploy/grafana/s3client.dashboard.json`](../deploy/grafana/s3client.dashboard.json)（与 [`OPERATIONS.md`](OPERATIONS.md) §4 **和** `deploy/prometheus/s3client.rules.yml` 同改；指标 / `code` / recording rule 真实性由 `grafana_dashboard_gate_test.go` 校验） |
| 性能预算 / 热路径 | [`PERFORMANCE.md`](PERFORMANCE.md) §4 + `apps/server/bench_budget_test.go`（改预算须同改两处；原始基准见 [`.github/workflows/perf.yml`](../.github/workflows/perf.yml)） |
| 漏洞披露渠道 / `security.txt` | [`../.well-known/security.txt`](../.well-known/security.txt)（`Expires` 到期前必须续期，门禁 `security_txt_gate_test.go`）+ [`../.github/SECURITY.md`](../.github/SECURITY.md) |
| 原生 fuzz 目标与语料 | `apps/server/internal/*/*_fuzz_test.go` + 本文件 §2（新增解析面须同补 fuzz 目标；语料入库防回归） |
| AI 评测黄金任务集 / 评分锚点 | [`AGENT_EVALS.md`](AGENT_EVALS.md) + [`../scripts/evals/golden-tasks.yaml`](../scripts/evals/golden-tasks.yaml)（两处 id / 标题逐字一致，由 `agent_evals_gate_test.go` 钉住）+ [`../scripts/evals/run-golden-task.sh`](../scripts/evals/run-golden-task.sh) |
| 可访问性渲染态扫描 / 对比度记录 | [`accessibility.md`](accessibility.md) §4 / §5.2 / §5.5 + [`../apps/web/e2e/a11y.spec.ts`](../apps/web/e2e/a11y.spec.ts) + `apps/server/contrast_gate_test.go`（改 `styles.css` 必须同改 §5.5 表值） |
| 英文文档（新增 / 更新翻译） | [`en/README.md`](en/README.md) · [`i18n.md`](i18n.md) §7（中文为 SSOT；`en_docs_gate_test.go` 校验来源声明、revision 与可达性） |

**文档登记表**（2026-09-29 建立）：

> - **owner**：全仓 owner 均为单人维护者 **@weilai1949**。新增协作者 / 转移所有权时同 PR 改本表。
> - **复审周期**：按文档性质给出的**建议值**（证据等级「建议值」——**带「N 个月」的到期由
>   [`doc_review_gate_test.go`](../apps/server/doc_review_gate_test.go) 机械检查**：超过
>   `最后复审 + 周期` 即红灯点名该行，复核后把日期更新为复核当天即绿；无数值周期的口径
>   （「每个版本发版前」「由门禁强制」「每次归档操作时同 PR」）仍靠人工。它只说明
>   「多久不看就该有人看」，不是 SLA）。
> - **最后复审**：一律填 `登记时基线（2026-09-29）`——本表建立时**没有**一份文档有过可核实的
>   「逐份复审」记录，因此**不追溯编造**历史日期。只有真正做过一次逐份复核（不只是改错别字 /
>   跑门禁），才把该行更新为**那天的日期**。本列因此是「自本表建立起是否被复审过」的凭据，
>   不是「文档有多新」（最后改动时间看 `git log`）。

| 文档 | 复审周期（**建议**） | 最后复审 |
|---|---|---|
| 根 `README.md` · `AGENTS.md` · `CHANGELOG.md` · `llms.txt` | 3 个月 | 登记时基线（2026-09-29） |
| [`docs/README.md`](README.md)（docs 导航落地页 = 人类导航 SSOT） | 3 个月，或新增 / 改名文档时同 PR | 登记时基线（2026-09-29） |
| 子树 `AGENTS.md`：[`apps/server`](../apps/server/AGENTS.md) · [`apps/web`](../apps/web/AGENTS.md) · [`apps/desktop`](../apps/desktop/AGENTS.md) | 6 个月，或子树规则变更时同 PR | 登记时基线（2026-09-29） |
| [`.github/copilot-instructions.md`](../.github/copilot-instructions.md)（AI 工具**指针**，非规则本体） | 6 个月，或 AI 工具入口调整时同 PR | 登记时基线（2026-09-29） |
| `docs/DEVELOPMENT.md` · `CONFIGURATION.md` | 3 个月 | 登记时基线（2026-09-29） |
| `docs/DEPLOYMENT.md` · `OPERATIONS.md` · `POSTMORTEM_TEMPLATE.md` | 3 个月 | 登记时基线（2026-09-29） |
| `docs/FEATURES.md` · `KNOWN_ISSUES.md` · `ROADMAP.md` | 每个版本发版前 | 登记时基线（2026-09-29） |
| `docs/PERFORMANCE.md` · `AI_POLICY.md` | 6 个月 | 登记时基线（2026-09-29） |
| [`AGENT_EVALS.md`](AGENT_EVALS.md)（AI 评测与贡献度量） | 6 个月，或评测口径 / 披露字段集变更时同 PR | 登记时基线（2026-09-30） |
| [`.github/`](../.github/) 社区健康文件：`CONTRIBUTING.md` · `SECURITY.md` · `SUPPORT.md` · `GOVERNANCE.md` · `CODE_OF_CONDUCT.md` | 6 个月 | 登记时基线（2026-09-29） |
| 产品内容文档：[`api.md`](api.md) · [`architecture.md`](architecture.md) · [`data-model.md`](data-model.md) · [`errors.md`](errors.md) · [`threat-model.md`](threat-model.md) · [`user-guide.md`](user-guide.md) · [`compatibility.md`](compatibility.md) · [`glossary.md`](glossary.md) · [`i18n.md`](i18n.md) · [`accessibility.md`](accessibility.md) | 6 个月，或对应功能变更时同 PR | 登记时基线（2026-09-29） |
| [`en/index.md`](en/index.md)（英文 README 入口） | 3 个月，或根 `README.md` 变更时同 PR | 登记时基线（2026-09-30） |
| [`docs/api/`](api/openapi.json)（机器可读契约） | 由门禁强制，无需人肉周期 | 登记时基线（2026-09-29） |
| [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md)（**脚本自动生成，勿手工编辑**） | 依赖增删改时同 PR 重新生成 | 登记时基线（2026-09-29） |
| [`docs/decisions/`](decisions/index.md)（ADR + 模板） | 决策变化 / 新增 ADR 时同 PR | 登记时基线（2026-09-29） |
| [`docs/archive/`](archive/index.md)（冻结归档 + 索引） | 每次归档操作时同 PR | 登记时基线（2026-09-29） |
| [`docs/archive/incident-20260916-presign-empty-url.md`](archive/incident-20260916-presign-empty-url.md)（首份已填写事故复盘，**冻结件**） | 不复审——归档 = 冻结，不回写、不改写 | 2026-09-30（归档登记） |
| [`en/README.md`](en/README.md) · [`en/architecture.md`](en/architecture.md)（英文快照） | 3 个月，或中文源文档变更时同 PR | 登记时基线（2026-09-30） |
| [`../.well-known/security.txt`](../.well-known/security.txt)（机器可读漏洞披露） | 到期前续期（≤ 12 个月，门禁强制） | 登记时基线（2026-09-30） |
| [`../CITATION.cff`](../CITATION.cff)（工具固定名：GitHub 引用元数据） | 6 个月，或作者 / 许可变化时同 PR | 登记时基线（2026-09-30） |
| [`../deploy/grafana/`](../deploy/grafana/) · [`../scripts/evals/`](../scripts/evals/)（机器可读运维 / 评测资产） | 由门禁强制，无需人肉周期 | 登记时基线（2026-09-30） |
| [`archive/handoff-20260930.md`](archive/handoff-20260930.md)（批次交接**时点快照**，**冻结件**，2026-10-01 批次收口） | 不复审——归档 = 冻结，不回写、不改写 | 2026-10-08（归档登记） |
| [`archive/code-review-2026-10-09.md`](archive/code-review-2026-10-09.md)（全仓代码质量评审**时点快照**，**冻结件**，2026-10-09 评审 O1–O12 全部收口） | 不复审——归档 = 冻结，不回写、不改写 | 2026-10-10（归档登记） |

落地要求：

- **写清「为什么」**：文档记录用户可见行为、边界与决策依据，不复述 diff。
- **链接不悬空**：新增 / 移动文档时同步修复引用。
- **有意识地不更新**：若某类文档确实无需改动，在 PR 描述中写明原因。
- **文档改动也要验证**：命令、路径、示例需与实际一致（能跑就跑一遍）。

文档命名与存放约定：

- **位置**：根目录只保留四个**约定文件**——`README.md`（社区约定）、`AGENTS.md`（agent 工具加载器**硬性要求**在根目录，放在 `docs/` 下不会被自动加载）、`CHANGELOG.md`（Keep a Changelog 约定名，release-please / semantic-release / standard-version / git-cliff 等工具默认 `./CHANGELOG.md`）、`llms.txt`（[llms.txt 约定](https://llmstxt.org/)把位置固定为 `/llms.txt`，2026-09-29 登记——它是**给 LLM 的仓库导航索引**，只列入口不复述规范，规范正文仍以本文件与 [`AI_POLICY.md`](AI_POLICY.md) 为准）。**社区健康文件**（`CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md` / `SUPPORT.md` / `GOVERNANCE.md`）放 `.github/`——GitHub 对这类文件的查找优先级是 `.github/` > 根目录 > `docs/`，放在最高优先级位置可避免被将来某个副本静默顶掉（`.github/SUPPORT.md` 已于 2026-09-29 落地，不再是「将来若加」的假设）；除上述根目录约定文件与 `.github/` 社区健康文件外的其余文档统一放 `docs/`。另有**工具固定名**留在根目录：`LICENSE` 与 `CITATION.cff`（GitHub 的 cite 功能只认根目录，2026-09-30 登记）。另有一类**工具固定名**放在 `.github/`：[`.github/copilot-instructions.md`](../.github/copilot-instructions.md)（GitHub Copilot 的仓库指令文件，位置与字面名由 Copilot 固定；本仓库只放**指针**，规则本体仍在根 `AGENTS.md`）。
- **命名**：`docs/` 下按**文档性质**二分，外加工具固定名：
  - **大写** = ① 名字被外部工具固定的：`README.md`（含 [`docs/README.md`](README.md)——GitHub 按字面名渲染的**目录落地页**，同时是人类导航 SSOT）、`AGENTS.md`、`CHANGELOG.md`、`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`、`LICENSE`、`CODEOWNERS`（GitHub 按字面名在 `CODEOWNERS` / `.github/CODEOWNERS` / `docs/CODEOWNERS` 三处查找，小写不生效）、`SUPPORT.md`、`GOVERNANCE.md`（社区健康文件固定名）、`CITATION.cff`（GitHub 引用元数据，只认根目录）；② `docs/` 下的**仓库元文档**——描述「**仓库自身如何运作**」（配置 / 部署 / 开发规范 / 运维 / 性能 / 政策 / 台账 / 规划）：`CONFIGURATION.md`、`DEPLOYMENT.md`、`DEVELOPMENT.md`、`OPERATIONS.md`、`PERFORMANCE.md`、`AI_POLICY.md`、`AGENT_EVALS.md`、`KNOWN_ISSUES.md`、`FEATURES.md`、`ROADMAP.md`、`POSTMORTEM_TEMPLATE.md`、`THIRD_PARTY_LICENSES.md`。
  - **小写 kebab-case** = `docs/` 下的**产品内容文档**——描述「**产品是什么 / 怎么用**」（接口 / 架构 / 错误码 / 安全设计 / 用户手册 / 兼容 / 术语 / 翻译 / 无障碍）：`api.md`、`architecture.md`、`data-model.md`、`errors.md`、`threat-model.md`、`user-guide.md`、`compatibility.md`、`glossary.md`、`i18n.md`、`accessibility.md`，以及 `docs/api/`（机器可读契约，如 `openapi.json`）、`docs/archive/`、`docs/decisions/`、`docs/en/` 下的全部文件。**时点性批次交接快照**同用小写（先例 `assessment.md`、`handoff-20260930.md`，结论绑定日期、批次收口后归档入 `docs/archive/`——两者均已归档）；时点性**代码评审快照**同样小写（先例 `code-review-2026-09-24.md`、`code-review-2026-10-09.md`，结论绑定评审日期、全部闭环后归档入 `docs/archive/`——两者均已归档）。
  - **新增 `docs/` 文档时先判性质再起名**：属「仓库怎么运作」→ 大写；属「产品是什么」→ 小写。**新增或变更任何大写文件名，必须在同一个 PR 里同时改本处与 [`AGENTS.md`](../AGENTS.md)**（防两处分叉）。
  - **沿革——不要凭直觉把某一类「修正」回去**：2026-09-17 曾把当时按「台账类大写」习惯命名的 `API.md` / `ASSESSMENT.md` / `ERRORS.md` / `FEATURES.md` / `ROADMAP.md` **全部小写化**（理由：无任何工具按文件名匹配）；2026-09-24 登记 `KNOWN_ISSUES.md` 为例外；**2026-09-29 改为现行的「元文档大写 / 内容文档小写」二分**，即元文档恢复大写、内容文档维持小写。三代规则的取舍逐条记在 [`CHANGELOG.md`](../CHANGELOG.md)，历史条目按惯例不改写。
- **目录**：一律小写（`docs/`、`docs/decisions/`、`docs/archive/`、`docs/en/`、`apps/server/`、`apps/web/`）。**本条只约束目录名**（文件名规则见上一条）——目录不存在「约定大写」这一说，大写只由工具强制决定：`.github/` 与 `.github/ISSUE_TEMPLATE/`（GitHub 按字面名查找，小写不生效）已符合；若将来引入 REUSE 规范的逐文件许可证全文，则用 `LICENSES/`。把 `docs/` 改成 `Docs/` 会让 GitHub 的社区健康文件查找（以及将来的 Pages 发布源）失效。`docs/en/` 为英文文档目录（当前唯一文件 [`en/index.md`](en/index.md)）：中文为 SSOT、英文页是翻译快照，根 `README.md` 变更时同 PR 同步。
- **归档**：时点性文档（综合评估、分支 / 版本审查、迁移对照、已填写的事故复盘等，结论绑定在某个 commit 或日期上）在结论被后续工作取代后，用 `git mv` 移入 [`docs/archive/`](archive/index.md) **冻结**——**不移除、不回写、不改写历史结论**，并在该目录索引登记一行、修正全仓引用。判断标准与操作步骤见 [archive/index.md](archive/index.md)。`decisions/` 的 ADR **不归档、不删除**：决策变化时新写一篇 ADR 引用旧篇并标 `Superseded`。**归档触发规则**：**已填写的事故复盘**（`incident-YYYYMMDD-<短名>.md`）在事件闭环、行动项登记完成后按同一「四步」冻结——首例 [`archive/incident-20260916-presign-empty-url.md`](archive/incident-20260916-presign-empty-url.md) 已于 2026-09-30 执行；**批次交接快照**（`handoff-YYYYMMDD.md`）在批次全部收口后同 PR 冻结并从三处命名清单 + `docs/README.md` 导航行摘除——首例 [`archive/handoff-20260930.md`](archive/handoff-20260930.md) 已于 2026-10-08 执行。**当前无待归档例外**——`assessment.md`（2026-09-16 综合评估）已于 2026-09-24 完成引用收敛并归档至 `docs/archive/`，`review-2026-09-19.md` 已于 2026-09-23 同样归档；归档清单见 [archive/index.md](archive/index.md)。
- **禁止大小写冲突**：任何两个路径不得仅大小写不同——macOS / Windows 的大小写不敏感文件系统会让它们互相覆盖、检出即丢内容。重命名后自检一次全仓。

### 4.1 Agent 指令文件（`AGENTS.md`）的加载机制

支持该约定的工具（deepseek-harness 等）按以下规则发现并注入指令，改文件时不要破坏这些前提：

- **候选名精确匹配、区分大小写**：项目级为 `AGENTS.md`（次选 `CLAUDE.md`），本地覆盖为 `AGENTS.local.md`（已进 `.gitignore`），用户级为 `$DSH_HOME/AGENTS.md` 或 `~/.dsh/AGENTS.md`。
- **项目根由 `.git` 标记向上查找确定**；根目录与用户级同名文件会去重。
- **子树 `AGENTS.md` 惰性加载**：`apps/server/`、`apps/web/` 等子目录的 `AGENTS.md` 在工具读到该子树文件时才注入，用于子树专属规则；仓库级规则才放根文件。
- **内容整体注入且受字节预算约束**：超长会被省略 / 截断，因此根 `AGENTS.md` 保持短小，只留硬约束与指针，可推导的细节留在本文件。
- **其它 AI 工具入口是「指针」而不是副本**：GitHub Copilot 读 [`.github/copilot-instructions.md`](../.github/copilot-instructions.md)——本仓库提交的是纯指针（指向根 `AGENTS.md` / 本文件 / [`AI_POLICY.md`](AI_POLICY.md)）。**刻意不提交** `CLAUDE.md`（候选名已覆盖该工具链）与 `llms-full.txt`（会把全部文档复制一份，与单一事实源 + 链接门禁冲突），理由与门禁见 [`AI_POLICY.md`](AI_POLICY.md) §11。
- **本节的这些前提有门禁守着**：根 `AGENTS.md` ≤ 10 KiB 且必须指向本文件、每个 `apps/*` 子树必须有**回指根文件**的 `AGENTS.md`、AI 工具指针文件 ≤ 2 KiB —— 由 `apps/server/ai_governance_gate_test.go` 断言。


## 5. 验收清单（按 review 五轴）

提交前逐项核对并在 PR 描述中说明：

- **正确性**：符合需求；边界（空值/空列表/截断/大文件）覆盖；错误路径有测试；测试断言的是行为不是实现。
- **可读性**：命名贴近领域且与项目一致；控制流平直；无死代码 / 无 rest 兼容 shim。
- **无死代码 / 无未使用变量**（详见 §6 前两条）：定义后无人引用的函数 / 类型 / 常量 / 变量 / 字段、不可达分支、未定义即使用的标识符、定义了却未使用的变量（含只写不读、赋值后即被覆盖、恒真/恒假的空断言）**一律不得残留**；**枚举值除外**（枚举/常量表成员即使暂无引用也属契约，保留）。
- **架构**：沿用 `store → model → s3wrap → handler` 分层；`AWS` 类型不外泄到 handler/前端；**单文件不超过约 1000 行**，超过先拆再改；Feature 逻辑不进共享模块。
- **账号存储**：`S3C_STORE_DRIVER` 支持 `json`（默认）/ `sqlite` / `encrypted`；`store.Open` 统一入口。
- **安全**：用户输入在边界校验；敏感字段（`SecretKey`）不落地 localStorage、不写日志；输出转义（禁 `v-html`）；外部数据视为不可信。
- **性能**：列表端点分页；分段上传有界并发；无 N+1 / 无界循环。
- **文档**：本次改动涉及的文档已按 §4「文档同步门禁」更新（README / api.md / CHANGELOG / KNOWN_ISSUES / roadmap 等），链接无死链，命令与路径实测一致。
- **验证**：测试绿 + 构建绿 + 涉及 UI 的保留截图/手测记录。

## 6. Red Flags（遇到即停下修正）

- 没看到红灯就直接写实现；
- 大而全的单文件改动（>1000 行）；
- 用 `any`/断言私有成员「掩盖」不清晰的不变量；
- 一次升级一批依赖 / 手改 lockfile；
- 功能逻辑渗入共享工具模块；
- 「以后再说」的清理不会发生——提交前就清干净；
- 改完代码不更新文档（README / api.md / CHANGELOG / KNOWN_ISSUES / roadmap 等相关文档漏更，见 §4）；
- 为凑覆盖率而写与实现耦合的测试（gap 测试模式）；
- **死代码 / 未使用变量**：定义后无人引用的函数 / 类型 / 常量 / 变量 / 字段、不可达分支、未定义即使用的标识符、定义了却未使用的变量（含只写不读、赋值后即被覆盖、恒真/恒假的空断言）——**枚举值除外**。这类问题必须由 `golangci-lint`（`unused` + `staticcheck`）/ `go vet` / `pnpm lint` / `vue-tsc` 机械拦住，**门禁输出必须 0 issues**；不要用 `_ = x`、`var _ = f`、`//nolint` 或导出为 `_test` 辅助来「消音」，那只是把死代码藏起来。
- **以为覆盖率达标就等于没死代码**：测试文件不参与 instrumentation（`go test -cover` 只统计生产代码），测试辅助里的死代码可以让 100% 门禁全绿——两者必须分别验证。本仓库就曾因此让一套失效的错误注入器长期存活（见 `CHANGELOG.md`）。

## 7. 历史技术债的现行守卫（2026-09-16 综合评估三条，均已闭环）

> **登记问题的唯一来源是 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)**，本节不登记问题。
> 原文三条来自 [`archive/assessment.md`](archive/assessment.md)（H1 / S7 / D5），当时记作「已知技术债」；
> 三条现已全部闭环，故改写为**闭环状态 + 现行守卫 + 开发规则**——它们已经修好，但
> 「什么动作会把它改坏」仍是每次改动都要知道的规则。台账见 [`FEATURES.md`](FEATURES.md)。

| 曾登记的技术债 | 闭环状态与守卫（回归即红灯） | 开发时仍须遵守 |
|---|---|---|
| **H1** OpenAPI 注册表与真实 handler 字段不一致（SSOT 失真） | 人工比对已换成**机械门禁**：`openapi_request_fields_test.go` 的 `TestOpenAPIRequestFieldsMatchHandlerDTOs`（注册表 ⇔ handler 字段集全量遍历）、`openapi_requestbody_test.go` 的 `TestOpenAPI_ContractRequestBodyMatchesHandlers`、`openapi_inputsource_test.go` 的 `TestOpenAPIRequestDeclarationMatchesHandlerInput`、`api_doc_test.go` 的 `TestAPIDocMatchesRoutes` / `TestAPIDocDocumentsRequestBodyFields`、`openapi_response_contract_test.go` 的 `TestOpenAPI_EndpointResponseSchemasMatchHandlers` | 改 handler 请求 / 响应体时**同 PR** 同步 `openapi_register_*.go`（见 §4 文档同步门禁），别等红灯才发现 |
| **S7 / P0-4** SSE 终态检测分叉（MigratePanel 与 `useObjectActions` / `DestDialog` 各写一份 → 流以 EOF 结束时 Promise 悬挂、`opsBusy` 永不复位、按钮永久禁用） | 收敛为**单一实现** `apps/web/src/api/jobs.ts` 的 `subscribeMigrateEvents`（EOF 后轮询回读直到终态 + 合成终态事件 + 心跳空闲超时 + 连续回读失败快速 `onError`），三个调用方共用；用例见 `api.transfer.test.ts` / `api.gaps.test.ts` 的 `subscribeMigrateEvents …` 组 | 新增异步任务消费方**只调 `subscribeMigrateEvents`**，不得自己开 SSE 流、自己判终态、自己加超时——三者都会把分叉再带回来 |
| **D5** endpoint 归一化多份实现（行为不一致） | 收敛为**单一 helper** `s3wrap.NormalizeEndpoint`，四处共用：建 client（`s3wrap/client.go`）、预签名（`s3wrap/presign.go`）、SSRF 拨号校验（`s3wrap/ssrf.go`）、同端判定（`service/migrate.go`）；用例 `s3wrap_test.go` 的 `TestNormalizeEndpoint` / `TestNormalizeEndpointNeverDoubleScheme` | 改端点逻辑一律走 `NormalizeEndpoint`，不要在调用点重写 scheme 补全 / 去尾斜杠 / 大小写归一——这正是当年分叉的来源 |
