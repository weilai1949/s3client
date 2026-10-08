# 兼容性说明（Compatibility）

> 本文件汇总 s3client 的**对外兼容承诺**：版本命名与支持窗口、`/api/*` 契约如何演进、账号库加密
> 文件格式的向后兼容、配置项（`S3C_*`）的增删口径、S3 服务端兼容矩阵与**客户端（浏览器 /
> 操作系统）支持矩阵**、弃用政策，以及桌面端分发与签名的现状。
>
> 每条承诺都指出**真值来源**（既有文档 / 源码 / 机械门禁），本文件**不制造第二份真相**；
> 当前没有承诺的地方写明「当前没有」与将来的演进方式，不做粉饰。
> 支持版本表与漏洞披露见 [`../.github/SECURITY.md`](../.github/SECURITY.md)；
> 配置项 SSOT 见 [`CONFIGURATION.md`](CONFIGURATION.md)；接口参考见 [`api.md`](api.md)。

## 1. 版本命名与发布节奏

命名约定以 [`../README.md`](../README.md) 与 [`../CHANGELOG.md`](../CHANGELOG.md) 为准：

| 形态 | 用途 | 示例 |
|---|---|---|
| `vMAJOR.MINOR.PATCH` | **稳定里程碑**（已打 tag 的对外可用版本） | `v1.0.0`（当前版本） |
| `v1.0.0-YYYYMMDDHHmmss` | 稳定里程碑之后**日常发版**的时间戳版本 | `v1.0.0-20260902120000` |
| `v1.0.0-rcN`（亦接受 `-alphaN` / `-betaN`） | **预发布**，用于在稳定版前收口 | `v1.0.0-rc1` |

- 版本号由 `scripts/release-version.sh` 一次性同步到仓库内所有约定文件（Makefile / Go main /
  Dockerfile / openapi / 两套 compose / npm ×2 / Cargo.toml / tauri.conf.json / Cargo.lock 本包 /
  README / docs / issue 模板），步骤见 [`../.github/CONTRIBUTING.md`](../.github/CONTRIBUTING.md) §发布流程
  与 [`DEPLOYMENT.md`](DEPLOYMENT.md) §5。
- **发布节奏**：项目**不做固定周期发布**（当前没有「每 N 周发一版」这类承诺）。实际节奏是按里程碑与
  修复推进：稳定里程碑（`rcN` → `vX.Y.0`）之后转入时间戳版滚动发布，逐字历史见
  [`../CHANGELOG.md`](../CHANGELOG.md)，版本级规划见 [`ROADMAP.md`](ROADMAP.md)。
- 运行时可核对版本：`GET /api/health` 的 `version` 字段（[`api.md`](api.md) 健康检查）。
- **发布硬前提**：质量门禁全绿、P0 未清零不发布（[`ROADMAP.md`](ROADMAP.md) §二 / §四）。

## 2. 支持窗口

**当前只维护最新版本**，安全修复随下一个版本发布；支持状态表（含 `v1.0.0-rc*` 与 `0.x` 的历史状态）
**唯一来源**是 [`../.github/SECURITY.md`](../.github/SECURITY.md) §支持的版本——本文件不复制该表，
以免两处漂移。

对使用者的含义：

- 升级到最新版即可获得全部安全修复；**当前没有 LTS 分支或长期维护的旧版本线**。
- 超出支持窗口的版本遇到问题，标准答复是「先升级到最新版再复现」（见
  [`../.github/SUPPORT.md`](../.github/SUPPORT.md) §2 自查清单第 1 条）。

## 3. API 兼容性承诺（`/api/*`）

### 3.1 契约的真值来源

- 接口与请求 / 响应字段：[`api.md`](api.md)（与 [`../apps/server/internal/handler/`](../apps/server/internal/handler)
  的 `openapi_register_*.go` 注册表一致，可经 `GET /api/openapi.json` 取 OpenAPI 3.0 契约，
  默认 404，需 `S3C_EXPOSE_OPENAPI=1`）。
- **门禁保证的部分**（[`DEVELOPMENT.md`](DEVELOPMENT.md) §3「契约与文档门禁的落点」）：端点集合与
  `docs/api.md` 双向一致、请求体字段集与 handler 解码结构体字段集全量一致、输入源（body / query /
  path）声明一致、query 与 path 参数双向一致、类型 / required / 枚举语义一致、共享 schema 与
  **端点级**响应字段双向一致。

> **只说门禁真正保证的事**：上述门禁挡住的是「文档 / 契约与实现**静默漂移**」。它**不**保证字段的
> 业务语义永远不变、也不保证客户端对未知字段的容错——那属于下面的规则，靠评审与 [`CHANGELOG.md`](../CHANGELOG.md)
> 的显式标注来兜底。

### 3.2 当前**没有**的版本机制

- `/api/*` **没有 URL 版本前缀**（不存在 `/api/v2/...`），**没有** `Accept` 协商、**没有**版本请求头、
  **没有**功能开关式的 API 降级。
- 因此下面这些规则不是「某个版本化框架的产物」，而是本仓库**当前自我约束的演进口径**；真正的护栏是
  §3.1 的机械门禁 + 评审 + `CHANGELOG` 标注。

### 3.3 非破坏性变更（按增强发布）

| 变更 | 说明 |
|---|---|
| **新增端点** | 在 `routes.go` + `openapi_register_*.go` + [`api.md`](api.md) 三处同 PR 落地（否则门禁红灯） |
| **新增响应字段** | 允许——但前提是现有前端对这**不是**破坏性变更（可忽略未知字段）；新增字段须在契约与文档里出现，并在 `CHANGELOG` 记 `Added` |
| **新增可选请求字段** | 允许——**必须**有默认值或「缺省即保持旧行为」，不得把旧请求变成 400 |
| **新增状态码 / 错误码** | 允许作为新增分支，但既有成功语义不变；错误体保持 `{"error": "..."}` 形状（[`api.md`](api.md)、[`errors.md`](errors.md)） |
| **放宽校验** | 允许（原来拒绝的输入开始被接受一般不影响既有客户端），但安全边界（[`threat-model.md`](threat-model.md)）收紧不算放宽 |

### 3.4 破坏性变更（必须显式标注与缓冲）

| 变更 | 为什么破坏性 |
|---|---|
| 删除 / 重命名端点，改方法或路径 | 调用方直接 404 / 405 |
| 删除或改名请求字段、把可选字段变必填、改字段类型 | 原请求变成 400 或被静默忽略（后者更危险） |
| 收窄已接受输入的取值集合（除安全修复外） | 原本可用的调用开始被拒 |
| 删除或改名响应字段、改字段类型或**语义** | 即使字段还在，含义变了同样破坏调用方 |
| 改变既有成功 / 失败的状态码约定 | 客户端分支判断失效 |
| 要求额外鉴权（新的必需头部 / scope） | 既有部署升级后直接 401 |
| 变更错误文案作为**判据** | 错误文案不是稳定契约的一部分；需要程序化判断请用状态码与 `errors.md` 映射 |

**破坏性变更的处置口径**：

1. 先在 [`CHANGELOG.md`](../CHANGELOG.md) 的对应版本段显式标注（**新增 / 变更 / 移除**这类分段标题），
   并在 [`api.md`](api.md) 更新真值；
2. 走 §7 的弃用流程（文档标注 + 过渡期），**能弃用就不直接删**；
3. 由于 §3.2 没有版本前缀，**缓冲只能靠发布节奏**：破坏性改动应在预发布（`-rcN`）或大版本
   （`vX.Y.0`）里出现，或在时间戳版发布说明中逐条点出，让使用者有机会不升级。
   > 这是**当前实践的约束，不是已承诺的硬保证**——真正能机械兜住的只有 §3.1 那些门禁。

## 4. 存储格式向后兼容（`S3C2` → `S3C3`）

账号库的加密文件格式承诺记在 [`ROADMAP.md`](ROADMAP.md) §5.2 **E10**（「只增版本、不改既有语义」）；
真值来源是源码 [`../apps/server/internal/store/crypto.go`](../apps/server/internal/store/crypto.go)
（信封读写与参数校验）与 [`../apps/server/internal/store/open.go`](../apps/server/internal/store/open.go)
（驱动分发与 StoreKey 契约）。

| 格式 | 状态 | 信封结构 | KDF |
|---|---|---|---|
| **`S3C2`** | **只读**（兼容既有加密库） | `"S3C2"`(4) + `salt`(16) + `nonce‖ciphertext` | Argon2id，参数**硬编码**在代码里（`t=1, m=64MiB, p=4`），文件本身不携带参数 |
| **`S3C3`** | **当前唯一写入格式** | `"S3C3"`(4) + `time`(4,BE) + `memory`(4,BE) + `threads`(1) + `salt`(16) + `nonce‖ciphertext` | Argon2id，参数**随文件头保存**（当前 `t=2, m=64MiB, p=4`），读取时按文件里的参数派生 |

承诺与行为（全部可在上述两个源码文件里逐条对上）：

1. **只增版本，不改既有语义**：升级**不需要**任何手工迁移命令——旧 `S3C2` 库照常被读取，
   新写入一律由 `envelope()` 产出 `S3C3`（`S3C2` 的构造只存在于测试中）；首次写入后旧库即自然转为
   `S3C3`。升级路径已有实跑证据（[`FEATURES.md`](FEATURES.md)）。
2. **可读旧格式**：`parseEnvelope` 同时接受 `S3C2` 与 `S3C3`；`isEncryptedBlob` 认这两种信封。
3. **`S3C3` 的意义正是「不改语义地加强参数」**：`S3C2` 不存 KDF 参数，直接调参会**让既有文件无法解密**；
   `S3C3` 把参数写进文件头，因此将来加强参数（例如提高 `time`）**在新写入的文件上生效，旧文件仍按其
   自带参数解密**。若将来需要新的信封结构（例如再引入一种 KDF），按同一模式**新增版本号**（`S3C4`），
   而不是改写 `S3C2` / `S3C3` 的含义。
4. **文件头参数会校验**：`S3C3` 的 KDF 参数来自文件内容、可被篡改，故派生前必须通过 `kdfParamsValid`
   ——同时拒绝零值（弱密钥）与超上界值（`time ≤ 10`、`memory ≤ 512 MiB`、`threads ≤ 16`，合法值的
   8 倍以上），避免「把 memory 改成 4GiB」这类**读放大 DoS**。零值与越界合并为同一罚则，对外只报
   `invalid KDF params`，不泄露上界位置。
5. **未知 magic 一律报错，不静默降级**：非 `S3C2` / `S3C3` 的信封直接报错，**不会**回退成明文解析。
   账号库无法解密时属「存储不可用」——按 [`ROADMAP.md`](ROADMAP.md) 的 [ADR-002](decisions/0002-store-fail-closed.md)
   **硬失败**（`/api/health` 返回 503），不会以「假装可用」的方式继续服务。

> **升级与回滚注意**：回滚到较旧版本前请先备份 `/data` 卷；**`S3C2` 由新版本写入后不会被改回**，
> 若回滚到只认 `S3C2` 的历史版本（`0.x` 时代）可能读不了新格式。回滚步骤见
> [`DEPLOYMENT.md`](DEPLOYMENT.md) §7。

## 5. 配置兼容性（`S3C_*` 环境变量）

配置项**全量清单与默认值是 [`CONFIGURATION.md`](CONFIGURATION.md)**（SSOT，由门禁
`config_doc_gate_test.go` 保证「源码读取的 `S3C_*` 变量都被文档收录」）；本节只写**增删口径**。

| 规则 | 内容 |
|---|---|
| **变量名只增不删**（作为常规做法） | 常规演进**新增**变量、不改名。文档多写了源码不读的变量不会红灯，正是为「已废弃项说明」留的合法空间 |
| **语义变更不得静默** | 已存在变量的**含义**若变化，必须在 [`CHANGELOG.md`](../CHANGELOG.md) 显式标注并在 [`CONFIGURATION.md`](CONFIGURATION.md) 更新；改变安全默认值（如鉴权、明文落盘、SSRF）属破坏性变更，须走 §7 |
| **需要改语义时优先新增开关** | 参照既有做法：SSRF 默认放行私网，收紧用**新增** `S3C_SSRF_DENY_PRIVATE` 显式开启（[ADR-003](decisions/0003-ssrf-private-allow.md)）；明文存储默认拒绝启动，本地放行用**新增** `S3C_ALLOW_PLAINTEXT_STORE`。即「默认值变更」配一个**显式 opt-in / opt-out 开关**，而不是让老部署的行为悄悄改变 |
| **改名 = 新增 + 弃用旧名** | 先新增新名，旧名在同一过渡期内**仍被读取**并标注弃用；过渡期结束后在允许破坏性变更的版本里移除。**注意**：本仓库有「不留兼容 shim / 死代码零容忍」的硬约束（[`../AGENTS.md`](../AGENTS.md) 第 5 条），因此**已无任何读者的旧变量名不会被长期保留**——移除时必须在 `CHANGELOG` 的 `Removed` 段说明 |
| **启动期硬失败清单是契约的一部分** | 哪些配置组合会**拒绝启动**（非回环无 token、无 `S3C_STORE_KEY` 的 `json` / `sqlite`、显式 `S3C_ENV_FILE` 不可读、数据目录已被加锁等）列在 [`CONFIGURATION.md`](CONFIGURATION.md) §3。这些是**有意的 fail-closed 行为**，不是缺陷；收紧该清单会影响既有部署，按破坏性变更处理 |
| **客户端设置键** | Web / 桌面存在浏览器里的键（`s3c.apiBase` / `s3c.token` / `s3c.servers` / `s3c.currentAccountId` / `s3c.locale` / `s3c.theme` / `s3c.hintsHidden`，见 [`CONFIGURATION.md`](CONFIGURATION.md) §5）落在**用户自己的浏览器**里、生命周期不受服务端控制：读不到旧键时按默认值回退（不报错），但**当前没有**跨键迁移逻辑，也不承诺为它们提供弃用期 |

> **当前没有任何已废弃的配置项**：`S3C_*` 的历史演进全是**新增**（例如 `S3C_LOG_JSON` /
> `S3C_EXPOSE_OPENAPI` / `S3C_CSP_CONNECT_SRC` 是后补的，见 [`../CHANGELOG.md`](../CHANGELOG.md)
> 关于配置 SSOT 的那条）——本文件不为尚未发生的事编造迁移案例。

## 6. 兼容矩阵

### 6.1 S3 服务端兼容矩阵

各 S3 实现的兼容性要求（**`ETag` 暴露是分段组装的硬前提**）以
[`../README.md`](../README.md) 的「各 S3 实现的兼容性要求」表为**唯一来源**——那里逐厂商列出
「在 CORS 规则中暴露 `ETag`」与「本项目自动化覆盖」（RustFS 有真对端 E2E 与真实联调 E2E，其余为手动）。
本文件**不复制该表**，避免两处漂移。

这里只强调三条与兼容性直接相关的事实：

1. **单文件直传（< 100MB）不依赖 `ETag` 暴露**；**分段直传（≥ 100MB）依赖**——需从每段 PUT 响应的
   `ETag` 头读取指纹才能完成组装（Bucket 的 CORS 规则需含 `ExposeHeader: ETag`）。
2. **未暴露时不会静默产出损坏对象**：每段 PUT 仍返回 2xx，但前端读不到 `ETag`，会在**组装前**报
   「未读取到 ETag」并 `abort` 清理已上传分段。
3. **端点可达性是部署侧的兼容前提**：预签名直传的 S3 端点必须能被**浏览器**解析（容器化 S3 场景见
   [`../README.md`](../README.md) 的提示与 [`DEPLOYMENT.md`](DEPLOYMENT.md)）。

厂商差异属外部服务行为（[`ROADMAP.md`](ROADMAP.md) §5.2 **E8**：状态「⚠️ 因厂商而异」）；本项目只保证
自身的要求已文档化 + 对 RustFS 有自动化真对端覆盖，**不承诺**对所有厂商 / 所有区域做过实测。

### 6.2 客户端支持矩阵（浏览器与操作系统）

> 本小节只写**能从源码 / CI 文件逐条核实**的事实，核实不了的组合一律标「❓ 未验证」，不做推断。
> 证据等级沿用 [`OPERATIONS.md`](OPERATIONS.md) §1.1 的思路，并在表格里逐格标注：
> **「代码 / CI 文件」**= 有文件与行号可查；**「构建目标推导」**= 由 Vite / TypeScript 的默认构建目标
> 推得的最低版本区间，**未逐版本实测**；**「上游行为」**= Tauri / GitHub runner 的默认行为，本仓库
> 未显式配置。与源码冲突时**以源码为准**，发现漂移按 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 同步修正。

#### Web 端浏览器

| 平台 / 产品形态 | 支持状态 | 依据（文件与行号） | 备注 |
|---|---|---|---|
| Chromium 系桌面浏览器（Chrome / Edge） | ✅ 已实现并验证 | 两套 Playwright **仅**跑 `chromium`（[`../apps/web/playwright.config.ts`](../apps/web/playwright.config.ts) 21–26 行、[`../apps/web/playwright.real.config.ts`](../apps/web/playwright.real.config.ts) 32–37 行）；`pnpm e2e:install` 只装 chromium（[`../apps/web/package.json`](../apps/web/package.json) 17 行） | **唯一有自动化覆盖的浏览器**。`showSaveFilePicker` 可用（[`../apps/web/src/api/download.ts`](../apps/web/src/api/download.ts) 53–65 行）→ ZIP 打包下载走**流式落盘**，不受 500MB 兜底上限约束 |
| Firefox（桌面） | ⚠️ 部分（全站未实测） | 无 Playwright firefox 项目；`showSaveFilePicker` 缺失时走 blob 兜底（[`../apps/web/src/api/download.ts`](../apps/web/src/api/download.ts) 67–84 行），且 >500MB 或大小未知且 >50 个对象时**拒绝**（同文件 4、68–74 行）；其余依赖均为通用能力：XHR 上传（[`../apps/web/src/api/upload.ts`](../apps/web/src/api/upload.ts) 12 行）、`fetch` 流读 SSE（[`../apps/web/src/api/jobs.ts`](../apps/web/src/api/jobs.ts) 58–65 行）、`sessionStorage` / `localStorage`（[`../apps/web/src/api/storage.ts`](../apps/web/src/api/storage.ts) 30–89 行） | **全站功能没有 Firefox 实测证据**；ZIP 下载能力有降级路径，语法基线见下方「构建目标推导」段 |
| Safari（macOS / iOS） | ⚠️ 部分（全站未实测） | 无 Playwright webkit 项目；ZIP 下载走 blob 兜底（同上）；`:focus-visible` 需 Safari ≥15.4（[`../apps/web/src/styles.css`](../apps/web/src/styles.css) 163–168 行，旧版无键盘焦点轮廓——可达性降级而非功能故障）；flex `gap` 需 Safari ≥14.1（同文件多处使用） | **未实测**；Safari <14.1 / <15.4 属「构建目标推导」之外的不保证 |
| 移动端浏览器（iOS / Android） | ❓ 未验证 | 有 `viewport` meta（[`../apps/web/index.html`](../apps/web/index.html) 5 行）与 `@media (max-width: 900px)` 断点（[`../apps/web/src/styles.css`](../apps/web/src/styles.css) 579 行），但 E2E 只用 Desktop Chrome 视口、无触屏用例 | **不承诺移动端可用**；按桌面浏览器对待属推测 |
| 旧浏览器 / IE / 禁用 JS | ❌ 不支持 | 入口是 `<script type="module">`（[`../apps/web/index.html`](../apps/web/index.html) 13 行）；Vue 3 依赖原生 `Proxy`（IE 无）；构建配置没有 legacy / polyfill 插件（[`../apps/web/vite.config.ts`](../apps/web/vite.config.ts)） | 构建产物为 ES2020 语法基线（见下），旧浏览器无法解析 / 运行 |
| 浏览器存储被禁（隐私模式 / 站点数据被清） | ⚠️ 部分 | 存储读写全部走 try/catch 降级（[`../apps/web/src/api/storage.ts`](../apps/web/src/api/storage.ts) 22–28、64–89、280–288 行）；Token 默认 `sessionStorage`（同文件 6–8 行头注释） | 界面可用、请求可发；「跨会话保留」失效、设置不持久 |

**语法与特性基线（构建目标推导，未逐版本实测）**：`vite.config.ts` 未覆盖 `build.target` → Vite 6 默认
`'modules'`（≈ES2020 语法基线）；`tsconfig.json` 的 `target` / `lib` 均为 ES2021（[`../apps/web/tsconfig.json`](../apps/web/tsconfig.json)
3、14 行）。据此**建议**的最低版本：Chrome ≥87 / Edge ≥88 / Firefox ≥78 / Safari ≥14——**建议值**，
由构建默认值推导，**不是逐版本实测结论**。代码里可核实的运行期特性：`String.prototype.replaceAll`
（ES2021，[`../apps/web/src/i18n/index.ts`](../apps/web/src/i18n/index.ts) 65 行）、`crypto.randomUUID`
（**有降级兜底**，[`../apps/web/src/api/storage.ts`](../apps/web/src/api/storage.ts) 110 行）、
`matchMedia('(prefers-color-scheme: dark)')`（[`../apps/web/src/theme.ts`](../apps/web/src/theme.ts) 9 行）。

#### 桌面端操作系统

| 平台 / 产品形态 | 支持状态 | 依据（文件与行号） | 备注 |
|---|---|---|---|
| Windows（NSIS `.exe`） | ✅ 已实现（构建产物） | 发布矩阵 `windows-latest` + `--bundles nsis`（[`../.github/workflows/release-desktop.yml`](../.github/workflows/release-desktop.yml) 38–40、146 行）；窗口尺寸下限见 [`../apps/desktop/src-tauri/tauri.conf.json`](../apps/desktop/src-tauri/tauri.conf.json) 的 `windows.minWidth/minHeight` | 产物架构 = runner 原生架构，**未显式 pin Rust target**（「上游行为」，以 Release 页 `SHA256SUMS` 为准）；**未签名** → SmartScreen 拦截（现状见 §8 与 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) §一 #25） |
| macOS（`.dmg`） | ✅ 已实现（构建产物） | 发布矩阵 `macos-latest` + `--bundles dmg`（[`../.github/workflows/release-desktop.yml`](../.github/workflows/release-desktop.yml) 44–46、146 行） | 产物架构 = `macos-latest` 当时指向的原生架构（「上游行为」，未 pin）；**未签名、未公证** → Gatekeeper（§8）；**Intel x64 是否被覆盖 ❓ 未验证**（workflow 没有独立的 x64 目标） |
| Linux x64 桌面（`.deb`，Debian 系） | ✅ 已实现（构建产物） | 发布矩阵 `ubuntu-22.04` + `--bundles deb`（[`../.github/workflows/release-desktop.yml`](../.github/workflows/release-desktop.yml) 41–43、146 行）；构建期安装 `libwebkit2gtk-4.1-dev`（[`../.github/workflows/ci.yml`](../.github/workflows/ci.yml) 347 行） | Tauri 2 运行时依赖 **WebKitGTK 4.1**（「上游要求」）→ Ubuntu 22.04+ / Debian 12+ 预期可用，**未在矩阵实测**；只产 `.deb`，无 rpm / AppImage；Fedora / Arch / ARM ❓ 未验证 |

> 桌面端**功能面与 Web 端完全一致**（同一套前端、无 Tauri IPC，见 [`user-guide.md`](user-guide.md) 十、桌面端）；
> 桌面壳内嵌浏览器引擎为 Tauri 2 平台默认（Windows WebView2 / macOS WKWebView / Linux WebKitGTK），
> 其版本由运行环境决定（「上游行为」，本仓库未 pin、未逐版本实测）。桌面产物未签名 / 未公证、无自动
> 更新通道的现状与处置见 §8，不在此重复。

## 7. 弃用政策

**当前状态**：项目**尚未弃用任何端点、请求 / 响应字段或配置项**——[`CHANGELOG.md`](../CHANGELOG.md)
里没有弃用相关的分段，也没有任何一条标注过弃用。因此下面是**将要遵循的口径**，而不是已经履行的记录。

标注弃用需要**同时**做三件事：

| 位置 | 怎么标 |
|---|---|
| 文档 | 在 [`api.md`](api.md)（端点 / 字段）或 [`CONFIGURATION.md`](CONFIGURATION.md)（配置项）就地标注「已弃用」+ **替代方案** + 预定移除的版本 |
| 变更记录 | [`CHANGELOG.md`](../CHANGELOG.md) 对应版本段新增一条**弃用**分段并逐条列出（体例同该文件的「新增 / 变更 / 移除」分段） |
| 本文件 | 在本节登记一行：弃用什么、替代什么、何时引入、**预定移除版本** |

**提前多久**：

- **没有固定的时间窗或版本数承诺**（当前不存在「至少保留 N 个版本 / N 个月」这类规则）。可依赖的
  缓冲只有发布节奏：弃用标注**至少在移除前一个发布周期**进入 `CHANGELOG`，并在**支持窗口内**的
  版本上可见（支持窗口见 §2 / [`../.github/SECURITY.md`](../.github/SECURITY.md)）。
- 若将来要给出硬性窗口，应当先按 §3.2 的路径把它写成明确承诺（并同步本节与 `CHANGELOG` 口径），
  而不是在此处先写一个做不到的数字。

**超出支持窗口的版本怎么办**：按 §2 —— 不修复、不回溯移植；遇到问题先升级到最新版再复现
（[`../.github/SUPPORT.md`](../.github/SUPPORT.md) §2 / §4）。历史分支（`0.x` 线、`v1.0.0-rc*`）
**不存在**独立维护通道。

**何时可以「无过渡期」直接移除**：仅限安全修复（例如某字段本身就是漏洞面）——此时仍须在
`CHANGELOG` 与本节说明理由与影响面。

## 8. 桌面端分发与签名现状

三个平台的产物（Windows NSIS `.exe` / Linux `.deb` / macOS `.dmg`）由推送 `v*` tag 触发
[`../.github/workflows/release-desktop.yml`](../.github/workflows/release-desktop.yml) 构建并挂到
GitHub Release，流程见 [`DEPLOYMENT.md`](DEPLOYMENT.md) §5。

**如实说明现状（不粉饰）**：

| 项 | 现状 |
|---|---|
| **Windows 代码签名** | ❌ **未签名**——需要代码签名证书（外部凭证，[`ROADMAP.md`](ROADMAP.md) §5.2 **E6** ⬜ 未获取），产物会被 **SmartScreen** 拦截 |
| **macOS 签名与公证** | ❌ **未签名、未公证**——需要 Apple Developer ID + 公证（同为 E6），会被 **Gatekeeper** 拦截，用户需**手动允许打开** |
| **Linux** | `.deb` 不做签名（无签名链） |
| **自动更新通道** | ❌ **当前没有**——无 Tauri updater 通道（其前提也是签名产物），修复版**无法自动触达**已安装的旧客户端 |
| **完整性校验的过渡手段** | 每个平台内唯一 `SHA256SUMS-<bundle>.txt` + `aggregate-checksums` 聚合 job，随 Release 发布，由 `repo_infra_gate_test.go` 守住；配合手动放行说明 |
| **代码层面剩余工作** | **无**——打包与发布链（tag ↔ `tauri.conf.json` / `Cargo.toml` 清单校验、平台内唯一 checksum、聚合 job）已收口 |

风险登记与跟踪：**R5**（桌面产物未签名 / 未公证、无自动更新通道，🟡 开放）见
[`ROADMAP.md`](ROADMAP.md) §5.1；阻塞项 **KNOWN_ISSUES #25** 见
[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) §一。**这不是兼容性缺陷而是分发缺口**：未签名不会导致运行错误，
但会让「安全修复触达存量用户」这条链路断掉——因此如实写明，并把它列为长期项
（[`ROADMAP.md`](ROADMAP.md) §三 #1）。

## 9. 相关文档

- 支持版本表与漏洞披露：[`../.github/SECURITY.md`](../.github/SECURITY.md)
- 客户端操作方式与桌面端差异：[`user-guide.md`](user-guide.md)
- 接口真值与错误映射：[`api.md`](api.md) · [`errors.md`](errors.md)
- 配置项 SSOT 与启动期硬失败：[`CONFIGURATION.md`](CONFIGURATION.md)
- 部署形态、升级与回滚：[`DEPLOYMENT.md`](DEPLOYMENT.md)
- 风险登记与依赖清单（E8 / E10 / E6 / R5）：[`ROADMAP.md`](ROADMAP.md) §五
- 外部阻塞与技术债：[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)
- 关键架构决策：[`decisions/index.md`](decisions/index.md)
- 契约与文档门禁的落点：[`DEVELOPMENT.md`](DEVELOPMENT.md) §3
- 版本命名与功能总览：[`../README.md`](../README.md) · [`FEATURES.md`](FEATURES.md)
