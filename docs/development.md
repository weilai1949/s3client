# 开发指南

> 面向贡献者与 AI 代理的开发规范。本文件合并了原 `agents.md` 的 TDD 约定与 `CONTRIBUTING.md` 的开发环境部分。
> 贡献流程、提交规范、环境搭建见 [CONTRIBUTING.md](../.github/CONTRIBUTING.md)；代码架构见 [architecture.md](architecture.md)。

## 1. 核心原则（TDD 优先）

1. **先写会失败的测试，再写让它通过的实现。**
2. **小步提交**：一个逻辑改动一个 commit；改动前后都要 `go test ./...` 与 `pnpm build` 全绿。
3. **行为驱动，不测内部实现**：断言「外部可见的行为 / 返回值 / HTTP 状态码」，不要断言私有函数或内部变量。
4. **红灯-绿灯-重构（红绿蓝）**：先看到测试因缺实现而失败（红），再让实现通过（绿），最后在测试保护下优化结构（重构）。
5. **改完代码必须同步文档**：任何一次**修复 bug** 或**新增功能**完成之后，必须更新相关文档；文档未同步 = 改动未完成，不得提交 / 合并。文档与代码属于同一个 commit（或同一 PR）。

## 2. 三类测试（按项目现有基建落地）

| 层 | 位置 | 运行方式 | 目的 |
|----|------|----------|------|
| **单元 / 行为测试** | `apps/server/internal/.../*_test.go` | `cd apps/server && go test ./...` | 用 `httptest.NewServer` 的**假 S3** 验证 handler 层逻辑（路由、参数校验、正/反例、错误码映射） |
| **真实对端 E2E** | `apps/server/internal/s3wrap/e2e_test.go` | `S3CLINET_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v` | 验证最硬核路径：**SigV4 签名 / 预签名直传 / 分段 Multipart 组装 / 跨 bucket 复制 / 标签 / 版本控制**。默认指向本地 RustFS |
| **真实联调浏览器 E2E** | `apps/web/e2e-real/real-backend.spec.ts` | `make e2e-real` | 真实 Go 后端（托管真实 `vite build` 产物）+ 真实 RustFS + 真实浏览器，**不 mock `/api`**：账号落库、建桶列桶、**浏览器直传**（预签名 PUT 跨源）。mock 版 `apps/web/e2e/*.spec.ts` 覆盖不到的结合部 |
| **前端类型 + 构建** | `web` | `cd apps/web && pnpm build`（含 `vue-tsc --noEmit`） | 类型安全与可构建性；UI 改动同时保留手测/截图证据 |

### 假 S3 模式（handler 测试）
- 用 `httptest.NewServer` 模拟 S3，采用 **path-style**（请求落在 `/{bucket}/{key}`）。
- 依 `r.URL.Query().Has("acl"|"tagging"|"versions"|"location"|"versioning")` 与 HTTP 方法分发返回的 XML。
- 返回标准 S3 XML（`<ListVersionsResult>`、`<AccessControlPolicy>`、`<Tagging>` 等），并按需返回特定错误码（如 `NoSuchTagSet`）验证容错。

### 真实 RustFS 联调
- 需要时用 `docker compose up -d rustfs`（默认 `rustfsadmin/rustfsadmin`，S3 API 9000、控制台 9001）。
- E2E 测试用 `S3CLINET_E2E=1` 门控，普通 `go test ./...` 不会执行，CI 因此不受影响。

### 真实后端 + RustFS 浏览器联调（历史任务 #37，已闭环）
- #37 已于 2026-09-22 闭环并从 [`todolist.md`](todolist.md) 移除，归档证据见 [`features.md`](features.md) §V；当前待办以 [`todolist.md`](todolist.md) 为准。
- 一条命令：`make e2e-real`（脚本 [`scripts/e2e-real.sh`](../scripts/e2e-real.sh)）。它会起一份**独立** RustFS 容器、构建真实前端产物与后端、起真实后端托管产物、跑 `pnpm e2e:real`，最后 `trap` 自动清理。
- **该脚本是这套编排的唯一来源**：本地 `make e2e-real`、GitHub Actions 与 GitLab CI 都调用它，各自只负责「装工具链 / 装浏览器系统依赖」。门禁 `TestRealE2EUsesSharedScript` 断言两侧 CI 都**实际调用** `bash scripts/e2e-real.sh`（仅出现在 `paths:`/`changes:` 里不算），防止又抄一份编排而漂移。
- **必须给 RustFS 配 `RUSTFS_CORS_ALLOWED_ORIGINS`**（脚本自起时已默认配好，GitLab service 变量里也配了）：浏览器直传（预签名 PUT）是页面 → S3 的**跨源**请求，缺 CORS 会被浏览器拦下。注意 curl / Playwright `APIRequestContext` **不经 CORS**，只用它们验证会「假绿」——所以用例特意驱动真实浏览器 XHR。
- 复用外部对端（GitLab service / 已起的实例）：`RUSTFS_ENDPOINT=http://rustfs:9000 bash scripts/e2e-real.sh --no-rustfs`（此时脚本不管理容器生命周期）。
- 排查用 `make e2e-real E2E_REAL_ARGS=--keep`（保留容器与后端进程）或 `--skip-build`（复用已有产物）。端口默认 8080 / 9000，可用 `SERVER_PORT` / `RUSTFS_PORT` 覆盖。
- 机械门禁：`TestRustFSImageIsConsistentlyPinned`（脚本默认值 / compose / GitLab service 三处镜像版本必须一致）、`TestRealE2EArtifactsExist`（联调 spec 出现 `page.route(` 即红灯）、`TestE2ESourcesAreTypechecked`、`TestLocalRealE2ETargetExists`。

### E2E 源码的静态检查
- `e2e/`（mock 版）与 `e2e-real/`（真实联调版）**不在**主 `tsconfig.json` 的 `include` 内，因此长期零静态检查。现补 [`apps/web/tsconfig.e2e.json`](../apps/web/tsconfig.e2e.json) + `pnpm typecheck:e2e`，并把两个目录纳入 `pnpm lint`（`eslint src e2e e2e-real`）。
- 两套 CI 的 `web` job 与本地 `make check`（`web-typecheck-e2e`）都跑它；门禁 `TestE2ESourcesAreTypechecked` 防漏挂。

## 3. 必验门禁（每次改动提交前）

```bash
cd apps/server && go vet ./... && go test ./...   # 后端
cd apps/server && go build ./...                   # 后端可构建
cd apps/web && pnpm test && pnpm build             # 前端单测 + 类型检查 + 构建
# 涉及签名/直传/分段/复制/标签/版本时，额外跑真实 RustFS E2E
cd apps/server && S3CLINET_E2E=1 go test ./internal/s3wrap/ -run 'TestE2E' -v
# 涉及前端 / 后端接口 / 直传时，额外跑「真实后端 + 真实 RustFS + 真实产物」浏览器联调
# （docker 自动起一份 RustFS，跑完自动清理；不 mock /api）
make e2e-real
# 或 make test-all（后端 + 前端单测）
make test-cover                               # 后端覆盖率 100% 门禁（CI 同款检查）
make lint                                     # golangci-lint（errcheck / staticcheck / govet / ineffassign / unused / gosec / nolintlint），必须 0 issues
# 改到桌面端依赖时（需本机已装 cargo-audit）：RustSec 审计，CI desktop job 同命令
make rust-audit
```

> **契约与文档门禁的落点**（改接口 / 改文档数字时相关）：`internal/handler/` 下
> `openapi_request_fields_test.go`（请求体字段全量）、`openapi_query_params_test.go`（query 参数
> 四种读取口径）、`openapi_path_params_test.go`（path 参数 ⇔ handler `PathValue` 双向）、
> `openapi_semantics_test.go`（类型 / required / 枚举）、
> `openapi_response_contract_test.go`（共享 schema + **端点级**响应）、`api_doc_test.go`（docs 双向
> + 路由行必须顶格）；仓库级 `repo_infra_gate_test.go`（CI / 发布链 / 工具链 pin）、
> `doc_number_gate_test.go`（md 叙述性数字）、
> `deadcode_gate_test.go`（消音式死代码 AST 判定 + `_test.go` 导出符号）。各文件的**断言范围与残留**写在文件头。
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

### CI 双平台一致性

同一套门禁同时落在 **GitHub Actions** 与 **GitLab CI**，两边 job、命令、门禁阈值必须一致；
改任一侧都要同步另一侧（含下表），否则会漂移成「GitHub 绿 / GitLab 红」。

| GitHub Actions | GitLab CI job | 门禁内容 |
|---|---|---|
| `ci.yml` · `server` | `server` | gofmt / go vet / govulncheck / golangci-lint v2.13.2（**0 issues**）/ `go test -race` + 覆盖率 100% / build |
| `ci.yml` · `web` | `web` | `pnpm lint`（`--max-warnings 0`，含 E2E 源码）/ typecheck（src + E2E 两份）/ `test:coverage`（100%）/ build |
| `ci.yml` · `docker` | `docker` | `docker build` + Trivy CRITICAL/HIGH 失败门禁（`.trivyignore`） |
| `ci.yml` · `desktop` | `desktop` | `cargo check --locked` + `cargo audit`（RustSec，有漏洞即红灯；webkit/gtk 系统依赖） |
| `ci.yml` · `desktop-build`（仅 `workflow_dispatch`） | `desktop-build`（`when: manual`，仅 `web` 源） | `tauri build --no-bundle` |
| `e2e.yml` | `rustfs-e2e` | 真 RustFS 对端 `TestE2E`（GitLab service 容器替代 compose） |
| `e2e-playwright.yml` | `playwright-e2e` | 构建产物 + vite preview + Playwright chromium |
| `e2e-real.yml` | `e2e-real` | **真实 Go 后端（托管真实构建产物）+ 真实 RustFS + 真实浏览器**，不 mock `/api`（含浏览器直传）；本地与两套 CI 共用 `scripts/e2e-real.sh` |
| `release-desktop.yml` | **不镜像** | 发布目标是 GitHub Release（tauri-action + `gh release upload`），需 Windows/macOS runner 与 `GITHUB_TOKEN` |

**触发事件也要对齐**（三套 workflow 的 `on:` 并不相同，别假设一致）：

| 事件 | GitHub 会跑 | GitLab 会跑 |
|---|---|---|
| push 到 main/develop | `ci.yml` 四个 job | `server` / `web` / `docker` / `desktop` |
| pull_request | `ci.yml` + 路径命中时的三个 E2E | 同上 + 命中的 E2E |
| workflow_dispatch / web | 全部四套 | 全部 8 个（`desktop-build` 为手动） |
| schedule | **只有三个 E2E** | 只有 `rustfs-e2e` / `playwright-e2e` / `e2e-real` |

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
> 前端 `i18n/**` 被排除统计。**写测试请以行为价值为先**（断言外部行为），不要为凑覆盖率而写
> 与实现耦合的测试；遇到确实不可达的分支，正确做法是**删除死代码**，而不是写 gap 测试把它
> 「测活」（2026-09 后端冲 100% 时即据此清掉了 `jobsList` 的 nil 兜底）；反之，若某条兜底
> 能由公开 API 触发（如 `JobProgress.Status` 是 `omitempty`，`Emit` 允许不带状态），则应
> 保留并用行为断言覆盖，不要为数字把它删掉。

## 4. 文档同步门禁（改完代码必须更新文档）

任何「修复 bug」或「新增功能」完成之后，必须更新对应文档，否则视为改动未完成。动手前先按下表确认本次改动会触碰哪些文档面，与代码一起改。

| 改动类型 | 必须同步的文档 |
|----------|----------------|
| 前端使用方式 / 界面 / 快捷键 / 截图 | [`README.md`](../README.md)（必要时补截图） |
| 后端接口、请求体、响应字段、状态码 | [`api.md`](api.md) + `apps/server/internal/handler/openapi_register_*.go`（并跑契约测试） |
| **md 正文里「N 个 `/api/*` 端点」这类数字** | 由 `apps/server/doc_number_gate_test.go` 机械校验（真值取自 `routes.go` 的 `mux.HandleFunc` 注册数）；新增此类声明时在 `docNumberClaims` 登记 |
| 错误码 / 错误文案 | [`errors.md`](errors.md) |
| **任何**新功能或 bug 修复 | [`CHANGELOG.md`](../CHANGELOG.md) 的 `[Unreleased]` 段（Keep a Changelog：Added / Fixed / Changed） |
| 已实现 / 已修复能力的台账 | [`features.md`](features.md) |
| 待办事项状态变化 | [`todolist.md`](todolist.md)（单一待办来源） |
| 版本级规划 / 优先级 | [`roadmap.md`](roadmap.md)（不做逐条流水账） |
| 分层 / 模块边界 / 目录结构 | [`architecture.md`](architecture.md)；重大决策另加 [`decisions/`](decisions/index.md) ADR |
| 环境变量 / 配置项 | [`.env.example`](../.env.example) + [`deployment.md`](deployment.md) + `README.md` |
| 部署 / 镜像 / compose / 发布流程 | [`deployment.md`](deployment.md) |
| 安全策略 / 威胁模型 / 加固 | [`threat-model.md`](threat-model.md) 与 [`SECURITY.md`](../.github/SECURITY.md) |
| 开发流程 / 门禁 / 测试命令 | [`AGENTS.md`](../AGENTS.md)（代理入口）+ 本文件 + [`CONTRIBUTING.md`](../.github/CONTRIBUTING.md) |
| 文档命名 / 存放位置 / 归档 | 本文件 §4 + [`README.md`](../README.md)「文档」段 + [`archive/index.md`](archive/index.md) |

落地要求：

- **写清「为什么」**：文档记录用户可见行为、边界与决策依据，不复述 diff。
- **链接不悬空**：新增 / 移动文档时同步修复引用。
- **有意识地不更新**：若某类文档确实无需改动，在 PR 描述中写明原因。
- **文档改动也要验证**：命令、路径、示例需与实际一致（能跑就跑一遍）。

文档命名与存放约定：

- **位置**：根目录只保留三个**约定文件**——`README.md`（社区约定）、`AGENTS.md`（agent 工具加载器**硬性要求**在根目录，放在 `docs/` 下不会被自动加载）、`CHANGELOG.md`（Keep a Changelog 约定名，release-please / semantic-release / standard-version / git-cliff 等工具默认 `./CHANGELOG.md`）。**社区健康文件**（`CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md`，将来若加 `SUPPORT.md` 同理）放 `.github/`——GitHub 对这类文件的查找优先级是 `.github/` > 根目录 > `docs/`，放在最高优先级位置可避免被将来某个副本静默顶掉；除上述根目录约定文件与 `.github/` 社区健康文件外的其余文档统一放 `docs/`。
- **命名**：普通文档用小写 kebab-case（如 `threat-model.md`、`todolist.md`）；只有名字被外部约定固定的才用大写（`README.md`、`AGENTS.md`、`CHANGELOG.md`、`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`、`LICENSE`）。**仓库内不存在白名单之外的大写文件名**——历史上按「台账类大写」习惯命名的 `API.md` / `ASSESSMENT.md` / `ERRORS.md` / `FEATURES.md` / `ROADMAP.md` 已于 2026-09-17 小写化。
- **目录**：一律小写（`docs/`、`docs/decisions/`、`docs/archive/`、`apps/server/`、`apps/web/`）。目录不存在「约定大写」这一说——大写只由工具强制决定：`.github/` 与 `.github/ISSUE_TEMPLATE/`（GitHub 按字面名查找，小写不生效）已符合；若将来引入 REUSE 规范的逐文件许可证全文，则用 `LICENSES/`。把 `docs/` 改成 `Docs/` 会让 GitHub 的社区健康文件查找（以及将来的 Pages 发布源）失效。
- **归档**：时点性文档（综合评估、分支 / 版本审查、迁移对照等，结论绑定在某个 commit 或日期上）在结论被后续工作取代后，用 `git mv` 移入 [`docs/archive/`](archive/index.md) **冻结**——**不移除、不回写、不改写历史结论**，并在该目录索引登记一行、修正全仓引用。判断标准与操作步骤见 [archive/index.md](archive/index.md)。`decisions/` 的 ADR **不归档、不删除**：决策变化时新写一篇 ADR 引用旧篇并标 `Superseded`。**当前例外**：`assessment.md` 仍被代码注释与活跃文档大量引用，暂按活跃文档维护（可追加状态更新块），待引用收敛后再归档；见 [archive/index.md](archive/index.md) 的「待归档」。（`review-2026-09-19.md` 已于 2026-09-23 完成引用收敛并归档至 `docs/archive/`。）
- **禁止大小写冲突**：任何两个路径不得仅大小写不同——macOS / Windows 的大小写不敏感文件系统会让它们互相覆盖、检出即丢内容。重命名后自检一次全仓。

### 4.1 Agent 指令文件（`AGENTS.md`）的加载机制

支持该约定的工具（deepseek-harness 等）按以下规则发现并注入指令，改文件时不要破坏这些前提：

- **候选名精确匹配、区分大小写**：项目级为 `AGENTS.md`（次选 `CLAUDE.md`），本地覆盖为 `AGENTS.local.md`（已进 `.gitignore`），用户级为 `$DSH_HOME/AGENTS.md` 或 `~/.dsh/AGENTS.md`。
- **项目根由 `.git` 标记向上查找确定**；根目录与用户级同名文件会去重。
- **子树 `AGENTS.md` 惰性加载**：`apps/server/`、`apps/web/` 等子目录的 `AGENTS.md` 在工具读到该子树文件时才注入，用于子树专属规则；仓库级规则才放根文件。
- **内容整体注入且受字节预算约束**：超长会被省略 / 截断，因此根 `AGENTS.md` 保持短小，只留硬约束与指针，可推导的细节留在本文件。


## 5. 验收清单（按 review 五轴）

提交前逐项核对并在 PR 描述中说明：

- **正确性**：符合需求；边界（空值/空列表/截断/大文件）覆盖；错误路径有测试；测试断言的是行为不是实现。
- **可读性**：命名贴近领域且与项目一致；控制流平直；无死代码 / 无 rest 兼容 shim。
- **无死代码 / 无未使用变量**（详见 §6 前两条）：定义后无人引用的函数 / 类型 / 常量 / 变量 / 字段、不可达分支、未定义即使用的标识符、定义了却未使用的变量（含只写不读、赋值后即被覆盖、恒真/恒假的空断言）**一律不得残留**；**枚举值除外**（枚举/常量表成员即使暂无引用也属契约，保留）。
- **架构**：沿用 `store → model → s3wrap → handler` 分层；`AWS` 类型不外泄到 handler/前端；**单文件不超过约 1000 行**，超过先拆再改；Feature 逻辑不进共享模块。
- **账号存储**：`S3C_STORE_DRIVER` 支持 `json`（默认）/ `sqlite` / `encrypted`；`store.Open` 统一入口。
- **安全**：用户输入在边界校验；敏感字段（`SecretKey`）不落地 localStorage、不写日志；输出转义（禁 `v-html`）；外部数据视为不可信。
- **性能**：列表端点分页；分段上传有界并发；无 N+1 / 无界循环。
- **文档**：本次改动涉及的文档已按 §4「文档同步门禁」更新（README / api.md / CHANGELOG / todolist 等），链接无死链，命令与路径实测一致。
- **验证**：测试绿 + 构建绿 + 涉及 UI 的保留截图/手测记录。

## 6. Red Flags（遇到即停下修正）

- 没看到红灯就直接写实现；
- 大而全的单文件改动（>1000 行）；
- 用 `any`/断言私有成员「掩盖」不清晰的不变量；
- 一次升级一批依赖 / 手改 lockfile；
- 功能逻辑渗入共享工具模块；
- 「以后再说」的清理不会发生——提交前就清干净；
- 改完代码不更新文档（README / api.md / CHANGELOG / todolist 等相关文档漏更，见 §4）；
- 为凑覆盖率而写与实现耦合的测试（gap 测试模式）；
- **死代码 / 未使用变量**：定义后无人引用的函数 / 类型 / 常量 / 变量 / 字段、不可达分支、未定义即使用的标识符、定义了却未使用的变量（含只写不读、赋值后即被覆盖、恒真/恒假的空断言）——**枚举值除外**。这类问题必须由 `golangci-lint`（`unused` + `staticcheck`）/ `go vet` / `pnpm lint` / `vue-tsc` 机械拦住，**门禁输出必须 0 issues**；不要用 `_ = x`、`var _ = f`、`//nolint` 或导出为 `_test` 辅助来「消音」，那只是把死代码藏起来。
- **以为覆盖率达标就等于没死代码**：测试文件不参与 instrumentation（`go test -cover` 只统计生产代码），测试辅助里的死代码可以让 100% 门禁全绿——两者必须分别验证。本仓库就曾因此让一套失效的错误注入器长期存活（见 `CHANGELOG.md`）。

## 7. 已知技术债（来自 2026-09-16 综合评估）

> 详见 [assessment.md](assessment.md) 与 [todolist.md](todolist.md)。开发时**避免扩大**以下模式：

- OpenAPI 注册表与真实 handler 字段不一致（改 handler 请求体时同步更新 `openapi_register_*.go`）。
- 复制粘贴式逻辑分叉（如 SSE 终态检测在 MigratePanel 与 useObjectActions 各一份）——优先收敛为共享实现。
- endpoint 归一化多份实现——改端点逻辑时统一到单一 helper。
