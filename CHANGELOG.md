# Changelog

遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。v1.0.0 之前采用 [SemVer](https://semver.org/lang/zh-CN/)；稳定里程碑后日常发版用 **`v1.0.0-YYYYMMDDHHmmss`**，预发布可用 **`v1.0.0-rcN`**（例如 `v1.0.0-rc0`）。

> 「已实现 / 已修复 / 已完善」功能的合并视图（含 0.1.0 起的全量台账）见 [`docs/features.md`](docs/features.md)；本文件保留逐字发布历史。

## [Unreleased]

### 修复（2026-09-22 威胁模型相邻失真收尾 · `.github/SECURITY.md` 三处 + `features.md` 现状表四行）
- **`.github/SECURITY.md` 与本轮修好的 `docs/threat-model.md` 相互矛盾**（该文件把 threat-model 引为权威，自身却没有任何机械门禁）：① 支持版本表仍停在「`v1.0.0-rc` 预发布阶段」——实际 `main.go` `version = "v1.0.0"` 且 `v1.0.0` tag 已存在，改为 `v1.0.0` / 时间戳版为受支持、`v1.0.0-rc*` 已被取代；② 部署建议称「`sqlite` 驱动当前明文落盘密钥」与实现相反（配 `S3C_STORE_KEY` 时 `secret_key` 列以 AES-256-GCM 密文落盘；无 key 时进程默认拒绝启动，`S3C_ALLOW_PLAINTEXT_STORE=1` 仅限联调），按实现重写并注明其余列仍为明文——故生产仍推荐 `encrypted`（整库加密）；③ CSRF 条目「强制 `application/json`」过强（实现为**非空** `Content-Type` 必须为 json、缺省放行），与 threat-model 边界 A 同款措辞对齐。
- **`docs/features.md` §一 现状表（§9 / §10）四行同款过期**：①「落盘加密 … 格式 `S3C2｜salt｜ciphertext`」→ 当前写入 `S3C3`（Argon2id 参数随文件头保存），`S3C2` 仅为兼容只读；② `encrypted`「严格 S3C2」→ 严格模式只接受受支持的加密信封（`S3C3` 当前 / `S3C2` 兼容）；③「CSRF 双防 … 强制 `application/json`」→ 改为非空 `Content-Type` 必须为 json（与上条 ③ 同源）；④「限速 … 优先 `X-Forwarded-For`」→ 仅直连对端命中 `S3C_TRUSTED_PROXIES` 时才采信 XFF 首段（默认不信任，防伪造绕过），与 threat-model 边界 A 的 DoS 行一致。
- **范围纪律**：`docs/assessment.md` / `docs/review-2026-09-19.md` 里的同款措辞**不改**——二者是审计 / 审查时点快照，按「不追溯篡改」保留原样；`docs/features.md` §二 历史台账（§C 的 `S-1 明文 SecretKey` / `S-2 XFF 限速绕过` 两行写的是 2026-04-19 当轮的处置事实，S3C2 与「优先采信 XFF」在当时均成立）同样保留，本轮只改 §一 现状表。收口方式：对全部活文档按「同一事实的多种措辞」逐一 grep 对源复核（`明文落盘密钥` / `X-Forwarded-For` / `S3C2` / `v1.0.0-rc` / `强制 application/json`），除上述 7 处外**零残留**（README / deployment / roadmap / api 早已与实现一致）。

### 修复（2026-09-22 门禁自身追加复核 · query 动态键 fail-open + AST 口径 + async 请求体自由体）
- **query 绑定变量动态键此前会 fail-open**：`openapi_query_params_test.go` 的动态键检测只匹配 `r.URL.Query().Get(name)` / `r.URL.Query()[k]`，而抽取又只认字面量 `q.Get("x")`；于是 `q := r.URL.Query(); q.Get(name)`（或 `q[k]`）既不被抽成参数、也不被标记为动态读取，注册表漏声明时可以全绿。已补 `hasDynamicQueryRead` 对所有绑定变量的检查，并新增合成用例覆盖绑定后的 `Get(name)` / `q[k]` 与「字面量不得误报」。
- **`PathValue` 动态键检测改 AST**：`openapi_path_params_test.go` 旧实现用裸正则扫整段方法体，注释/字符串里的 `PathValue(name)` 会误报红灯。新增 `hasDynamicPathValueRead` 走 `go/parser`，并用 `TestPathParamDynamicReadUsesAST` 钉住「真实动态键必报、注释/字符串/字面量不误报」，删除旧正则变量。
- **`readJSON` 定位改 AST**：`openapi_request_fields_test.go` 旧实现用逐行正则找 `readJSON(r, &x)`，注释或字符串字面量里的调用会被当成真实解码点。新增 `findReadJSONTarget`（AST 定位真实调用并返回行号，`findDecodeSite` 改为消费它），并加 `TestFindReadJSONTargetIgnoresCommentsAndStrings` 口径测试。
- **`delete-prefix/async`、`copy-prefix/async` 顶层请求体从自由体升级**：注册表此前用 `openapi.Obj()`，请求体字段门禁跳过字段集比对。现与同步端点共用具体 `properties`（`deletePrefixBody` / `copyPrefixBody`），请求体字段门禁覆盖这两个端点；剩余 `metadata` / `fields` 等嵌套自由体仍只比对到顶层键（已在门禁文件头登记）。
- **文档同步**：`docs/review-2026-09-19.md` 加当前状态块（HEAD / 数字 / 工作区非干净）、修正 P2 未闭环的旧摘要、更新过期数字与行号、明确附录 C 只描述 2026-09-19 当轮；`docs/development.md` / `docs/roadmap.md` / `docs/features.md` 同步 AST 口径、golangci linter 列表与门禁落点。
- **门禁实跑**：后端 `go test -count=1 ./...` 8/8 包通过 / `golangci-lint run ./...` **0 issues**；前端 `pnpm test` **66 文件 / 1042 用例**全绿 / `pnpm build` OK（360.52 kB，gzip 110.85 kB）。

### 新增（docs 归档目录：为时点性文档提供冻结存放处）
- **新增 `docs/archive/` 与索引 [`docs/archive/index.md`](docs/archive/index.md)**：此前时点性文档（综合评估、分支 / 版本审查等）没有约定的归档位置——要么留在 `docs/` 根下与活跃 SSOT 混放，要么在文档收敛时被直接移除（上一轮「文档收敛」即删掉了三份历史评估快照，追溯证据随之丢失）。现明确：结论被后续工作取代的时点性文档用 `git mv` 移入 `docs/archive/` **冻结**——不删除、不回写、不改写历史结论；`decisions/` 的 ADR **不归档、不删除**（被取代时新写一篇引用旧篇并标 `Superseded`）。索引内写明「什么该归档 / 什么不该归档 / 归档操作四步」。
- **约定登记**：[`docs/development.md`](docs/development.md) §4 补「归档」条目、把 `docs/archive/` 纳入目录清单、并新增一行「文档命名 / 存放位置 / 归档」的文档同步门禁；[`README.md`](README.md)「文档」段加归档入口。**当前无归档文件**，本批只落约定，供后续归档使用。
- **验证**：纯文档改动，不触碰代码与门禁；`docs/archive/index.md` 及本批改动文件内的全部相对链接已用脚本逐一核验可解析（无死链）。

### 修复（2026-09-22 审查 §4.3「维持」项复核 · 门禁自身可被绕过）
- **`openapi_inputsource_test.go` 用裸字符串匹配，注释即可骗过**：该门禁判断「handler 是否真的解码请求体」时用 `strings.Contains(body, "readJSON(")`，于是 `// 这里提到 readJSON(` 与 `s := "readJSON("` 都被判为「解码了」。实测把 `createAccount` 的真实 `readJSON` 调用删掉、只在注释里留一句 `readJSON(`，**旧实现仍全绿**（假绿）。现改为 **AST 判定**（`parseBodyCalls` 解析语法树，只认真实调用表达式），新增 `TestDecodesBodyIgnoresCommentsAndStrings` 钉住口径。**变异验证**：删真实解码留注释 → 新实现红灯。
- **`api_doc_test.go` 的缩进盲区让陈旧条目静默留存（fail-open）**：解析正则只认行首，缩进行在双向 diff 中**两个方向都不匹配**。实测在 `docs/api.md` 插入一条缩进的陈旧路由 `DELETE /api/nonexistent-stale`，`TestAPIDocMatchesRoutes` **静默通过**。现新增 `TestAPIDocRoutesAreFlushLeft`：路由声明行必须顶格，缩进即红灯（含扫描下限自检）。**变异验证**：插入缩进陈旧条目 → 新门禁红灯。
- **`deadcode_gate_test.go` 的整行正则对行内形状全部逃逸**：旧实现匹配 `^_ = ident` / `^var _ = expr`，实测 `x := 1; _ = x`、`if true { _ = x }`、`_ = x.Field`、`_ = s[0]` **全部不被拦**（fail-open）。现改为 **AST 判定**（`findSilencingDeadCode`）：按语句判定，只有「左侧全为 `_`」才算消音；`_, ok := m[k]`（comma-ok 惯用法）与 `_, _ = w.Write(b)`（显式丢弃返回值）放行，`var _ Iface = (*T)(nil)`（编译期接口断言）放行。新增 `TestFindSilencingDeadCodeCoverage` 断言「AST 命中数严格大于旧正则」（实测 9 vs 5），防止退回旧口径。
- **`//nolint` 消音此前无任何门禁**：`AGENTS.md` / `development.md` 明令禁止用 `//nolint` 消音（「那只是把死代码藏起来」），但实测在函数上方加一行无理由的 `//nolint:all`，`golangci-lint run` 仍报 **0 issues**——`//nolint` 是 golangci 自身的指令，默认不受任何 linter 审查。现 `.golangci.yml` 启用 `nolintlint`（`require-explanation` + `require-specific` + `allow-unused: false`）。**变异验证**：植入 `//nolint:all` → 红灯。
- **path 参数缺 handler 侧门禁（新发现的缺口）**：`TestOpenAPI_ContractPathParamsDeclared` 只校验注册表**内部**自洽（模板 `{x}` ⇔ 声明 `in:path`），**没有任何一条**比对 handler 是否真的 `r.PathValue("x")` 读取它。现新增 `openapi_path_params_test.go`：注册表 `in:path` 参数名集 ⇔ handler 沿调用闭包 `PathValue` 读取集**双向**比对（70 个端点 / 60 个带 path 参数），含非字面量键检测。**变异验证**：把 `getAccount` 的 `PathValue("id")` 改为 `PathValue("accountId")` → 同时报「漏声明」与「幻影参数」。
- **门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues**（含新启用的 `nolintlint`）/ `go build ./...` OK / `go test ./...` 全绿（8/8 包，每包 100.0% 覆盖且零未覆盖块）。

### 修复（2026-09-22 审查 §7.4 门禁盲区收尾 · 响应契约 + 4 处真实漂移）
- **`s3wrap.CorsRule` 缺 `json` tag（CORS 规则在浏览器里被静默清空）**：该结构体经 handler **直接序列化**进 `GET /api/accounts/{id}/bucket/cors` 的响应体，却没有 json tag → Go 输出 PascalCase（`AllowedMethods` / `MaxAgeSeconds`），而前端 `types.ts` 的 `CorsRule` 与 `docs/api.md` 都是 camelCase。Go 的 `json.Unmarshal` 大小写不敏感，**服务端既有测试一直全绿**；但 JavaScript 严格区分大小写，浏览器读 `x.allowedMethods` 恒为 `undefined` → `BucketCors.vue` 把方法与来源归一化成空数组。已补全 6 个 camelCase tag（`id` 带 `omitempty`）。回归测试：`TestCorsRuleJSONFieldNames`（双向：必须含 camelCase、不得再有 PascalCase）、`TestCorsRuleJSONRoundTrip`（前端发来的 camelCase 能被解析）。**变异验证**：去掉 tag → 两条红灯并列出实际 PascalCase 键。
- **`POST /api/migrate/async` 注册表误声明 `200`**：handler 写 `http.StatusAccepted`（202），`docs/api.md` 也写 `202`，只有注册表写 200。已改为 202，并由端点级响应门禁钉住。
- **`docs/api.md` 称 `/api/openapi.json`「不进鉴权层」**：与实现相反——`withAuth` 只豁免 `health` / `metrics`；配了 `S3C_TOKEN` 时无凭证访问返回 401，且需 `S3C_EXPOSE_OPENAPI=1` 否则 404（既有 `TestOpenAPI_GateAndAuth` 早已钉住该行为，纯文档失真）。已改为「经过鉴权层」并写明两个前置条件。
- **`docs/features.md` 称 69 个 `/api/*` 端点**：实际为 70（`routes.go` 70 条 `mux.HandleFunc`，README / `api.md` / `roadmap.md` 与 `apiRouteCount` 常量均为 70）。已改为 70。
- **`docs/api.md` 的 `PUT /api/accounts/{id}` 缺请求体 JSON**：此前只有散文描述（「字段同创建」），导致 `TestAPIDocDocumentsRequestBodyFields` 无法机械比对字段集而红灯。已补显式 JSON 示例（必填 `name`/`endpoint`/`accessKey`，`secretKey` 可省略保留原值），与注册表 required 声明一致。
- **门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go build ./...` OK / `go test ./...` 全绿（含新增门禁）。

### 新增（2026-09-22 审查 §7.4 门禁盲区收尾 · 补门禁而非只补缺陷）
- **端点级响应门禁全量收敛（自由体 `openapi.Obj()` → 具体 `properties`）**：把 accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system 十个注册表的自由体响应全部升级为 `openapi.BuildObj`，嵌套形状抽为共享构造器（`corsRuleSchema` / `tagRowSchema` / `lifecycleRuleSchema` / `versionEntrySchema` / `deleteMarkerSchema` / `jobProgressSchema` / `jobResultSchema` / `jobRecordSchema`）。端点级门禁覆盖面 **15 → 61** 个成功响应，自由体从约 45 个降到 **1 个**（仅 `/api/openapi.json`：响应体即规范本身，由 `Registry.HTTPHandler()` 直接写而非 `writeJSON`，机械抽取无意义）。自检从 `checked ≥ 15` 收紧为 `checked ≥ 60` 且 `untyped ≤ 1`——**注册表回退成自由体会直接红灯**。
- **query 参数门禁扩到四种读取口径**：抽取器从只认 `Get()` 扩到 `Get` / `Has` / `Values` / `Query()["x"]`，并覆盖 `q := r.URL.Query()` 绑定后的同名形式；新增**非字面量键检测**（`q.Get(name)` / `q[k]` 命中即红灯），杜绝动态键读取静默逃逸。新增口径测试 `TestQueryReadExtractorCoversAllForms`（8 种形态逐一断言 + 「无读取不得抽出参数」的反向断言）。
- **新增 md 叙述性数字门禁**（审查 §7.4 矩阵里唯一标「否」的行）：`apps/server/doc_number_gate_test.go` 以 `routes.go` 的 `mux.HandleFunc` 注册数为唯一真值，校验 README / `api.md` / `roadmap.md` / `features.md` 中「N 个 `/api/*` 端点」的 N；要求每条声明**至少命中一次**（文案漂移致正则失配即红灯，防门禁静默失效）。**变异验证**：把 `features.md` 改回 69 → 红灯。
- **新增 `_test.go` 导出符号死代码门禁**：`golangci-lint unused` 对**导出**符号因「可能被包外引用」而豁免，但 `_test.go` 不参与库构建、永远无包外引用——实测给 `_test.go` 加无人调用的导出函数/类型，`golangci-lint run` 报 **0 issues**。新增 `TestNoUnusedExportedTestSymbols` 扫描全部 `_test.go` 的导出包级符号，无引用即红灯（`export_test.go` 作为约定的测试接缝整体豁免）；检测逻辑抽为纯函数 `findUnusedExportedTestSymbols`，由 `TestFindUnusedExportedTestSymbols` 用**合成源码**做口径测试，不依赖「仓库里正好有个死符号」来证明门禁有效。

### 修复（2026-09-22 文档失真批次：roadmap 门禁基线 · threat-model · 活文档行号引用）
- **`docs/roadmap.md` §四 门禁基线整体过期**：表头标注「实测 2026-09-19」，但 09-20 / 09-22 两批改动（审查 P2、#37 真实联调）合入后数字未刷新，违反本仓库「门禁数字必须来自实跑」约定。本轮**全部门禁实跑取真数**并同步：前端测试 986 例（64 文件）→ **1042 例（66 文件）**；前端覆盖率补实测值（4074 / 2844 / 1095 / 3503 四指标 100%）；E2E 行补上 2026-09-22 新增的第三套工作流 `e2e-real.yml`（`make e2e-real` 实跑 3 passed），并把 mock 版与真实联调版拆成两行；新增此前漏登记的 `golangci-lint run ./...`（v2.13.2，0 issues——它是 AGENTS.md 硬门禁与 R2 守卫，却一直不在基线表内）；前端 lint / 类型行补上 `eslint src e2e e2e-real` 与 `typecheck:e2e` 口径。5 个 E2E workflow action SHA 重新经 GitHub API 核验（均 200）。
- **`docs/features.md` §三 现状表同步**：前端测试 1039 → **1042**，并补 2026-09-22 复测注记段（历史注记按「不追溯篡改」保留原值追加，不改写）。
- **`docs/threat-model.md` 状态与语义失真（5 处状态翻转 + 3 处语义 + 行号漂移）**：① 安全审计日志仍标「❌ 未缓解（todo #17）」、DoS 行仍以「XFF 伪造、job 无上限」为缺口——`handler/audit.go`、`S3C_TRUSTED_PROXIES` 可信代理白名单、JobRegistry ≤256 均已闭环，R/D 两行改 ✅；② §3「`s3c.servers` 把 token 写 localStorage（待修复）」与实现相反（`storage.ts` `writeServers` 只存 `{id,name,base}`，#15 已归档），改为已闭环描述；③ §6 两条开放项（审计日志缺失 / XFF 伪造）划线；④ presign「过期钳制 [1h, 24h]」改为真实的「≤0 默认 1h、>24h 钳 24h、无 1h 下限」并补 `multipart.go` 同款钳制；⑤ 桶名校验补漏禁 `--` 规则；⑥ metadata「字节总长」不存在——实际为 key≤128 / value≤256 字符 + ≤10 标签；⑦ action SHA「10 个」→ **12 个**（全量 workflows 去重，本轮逐一经 GitHub API 复验均 200）；⑧ 全文 9 处 `file:line` 行号引用（`config.go:120`、`main.go:74`、`objects.go:433` 等已全部漂移）改为**符号引用**（函数 / 常量名）——与「测试名不得硬编码源码行号」同一教训，根治行号腐烂；顺手修 `.trivyignore` 注释里的过期 `alpine:3.20`（Dockerfile 实为 3.24）。
- **`docs/threat-model.md` 二次全量对源复核（3 处事实错误 + 2 处精确度 + §2/§6 结构补全）**：① 边界 D 的 IMDS 归属错误——`100.96.0.2` 是**火山引擎**内网元数据而非阿里云（`ssrf.go` 注释原文即为两家），拆分归属；② §4 元数据 tags 单位错误——前一条 bullet 所述「字节总长」修正后写的「≤256 **字符**」实为 `len()` **字节**口径（错误消息文案 chars 属既有措辞），改正并注明；③ §边界 C 的「todolist #29/#31」死指针（编号已从 todolist 正文移除）改指 features §T 归档证据；④ Tampering 行「强制 application/json」过强——实现为**非空时**必须 json、缺省放行（安全结论不变，Bearer 头本就强制预检），按实现措辞；⑤ §6 拆为「6.1 已闭环（证据归档）+ 6.2 仍接受的风险（health 暴露 version / metrics 免鉴权 / SSRF 私网放行 / 明文 store 放行开关，共 4 项）」并去掉无实义的「与路线」标题——原节清一色已闭环项，读者无从知晓仍被有意接受的风险；⑥ §2 安全默认值表补 3 行（`/api/health` 免鉴权、`S3C_TRUSTED_PROXIES` 默认空、`S3C_SSRF_DENY_PRIVATE` 默认关），ReadHeaderTimeout 位置改符号 `runServer`；⑦ `.trivyignore`「空清单」改「仅策略注释、0 忽略条目」；⑧ 文件头删 CHANGELOG 式括注（明细归本文件）。STRIDE 覆盖边界 B–E 的补表**另开一轮**（改动大，不在本批）。
- **`docs/threat-model.md` STRIDE 补全（边界 B–E 各 6 行状态表，承接上一条的「另开一轮」）**：§1 标题自称「STRIDE × 边界」，但此前仅边界 A 有完整 STRIDE×状态表，B–E 只有缓解 bullet，读者无法逐类核对。本轮把 B/C/D/E 各补成与 A 同构的 6 行表（不适用 / 已决策放行的类别逐行给理由，不用「没写」来回避），新增 §1 状态图例（✅/⚠️/❌/➖）。新增事实全部先对源复核：presign 方法白名单 `get|put|post`（缺省 put、非法 400）与签发端点在 Bearer 鉴权内（`routes.go`）；密文篡改链路（`decryptAESGCM` 认证失败 → `store.Open` 失败 → `main` return 1 硬退出，ADR-002）；数据目录 flock 单写者锁（`AcquireDataDirLock`，`LOCK_NB` 第二实例立即失败）与 0700/0600 权限；Tauri 无 `invoke_handler`、capabilities 源真值 `capabilities/default.json` `permissions: []`（`gen/schemas/capabilities.json` 系未跟踪的过期构建产物——mtime 早于源文件——不作真值）。原 B/C/D/E 的 bullet 全部并入表格，无信息丢失。
- **活文档 file:line 引用腐烂与 ADR / 快照指针失真（threat-model 之外的残留清零）**：① `docs/decisions/0002`：`main.go:61-66` / `health.go:9-18` 均已漂移（`store.Open` 现在 `main.go:81` 附近、503 由 `health.go` `health` 回）——ADR 按 `docs/archive/` 约定**不归档、属活文档**，改符号引用（`main.go` `runServer` · `health.go` `health` / `store.Ping`）；② `decisions/0003`：`config.go:120` 改符号（`FromEnv`，`S3C_ADDR` 默认回环），并修 IMDS 归属同款错误——`100.96.0.2` 是**火山引擎**而非阿里（全仓第 2 处也是最后一处，另一处在 threat-model 已修，源注释 `ssrf.go` 原文即为两家）；③ `features.md` 两处：§E 台账行 `metadata.go:40` 去行号（遮蔽循环已改名，行号无从对证）、§U row 9 的例证行号「实际」改「当时」时态（历史证据保留数字，只消歧现在时误读）；④ `assessment.md` **仅头部处置指针**补「§N 起持续归档见 features / CHANGELOG」与「正文 file:line、版本号、测试数字均为审计时点值」——正文历史结论零改动（快照纪律）；⑤ `review-2026-09-19.md` 整份不动：审查对象 pinned 到 `develop@0fbd560`，时点自证。收口方式：对全部活文档（README / AGENTS / architecture / api / development / deployment / errors / features / roadmap / todolist / threat-model / decisions / archive / .github）按 `(go|ts|vue|js|yml|yaml|json|sh|conf|mod|Dockerfile):数字` 模式全量扫描 → **0 处无时点限定的引用**（两处纪律豁免：快照 `assessment` / `review` 整份不动；`features` §U row 9 的历史例证行号保留原数字、已加「当时」时点限定——它是「测试名硬编码行号」修复的证据本体，抹掉数字即抹掉证据）。

### 新增（2026-09-22 真实后端 + 真实 RustFS 浏览器联调 · todolist #37）
- **新增真实联调浏览器冒烟（不 mock `/api`）**：此前 `e2e-playwright.yml` / `playwright-e2e` 的 `/api/**` 全被 `page.route` mock，没有任何门禁验证「真实 Go 后端 + 真实 RustFS + 真实构建产物」拼在一起时的行为——尤其**浏览器直传**（预签名 PUT 是浏览器 → S3 的跨源请求），mock 时代在结构上不可能覆盖。现新增 `apps/web/e2e-real/real-backend.spec.ts` + 专用 `apps/web/playwright.real.config.ts`（不自拉 webServer，`PLAYWRIGHT_BASE_URL` 指向真实后端），3 条用例：账号 CRUD 真实落库（`secretKey` 不回传、UI「测试连接」走真实 S3）、建桶 → 列桶（真实 `CreateBucket`/`ListBuckets`）、**浏览器真实直传 → 列对象 → 预签名 GET 回读逐字节校验**。用例先清空后端账号表，保证每个用例从真实空状态出发。
- **本地一键运行（docker 自动起 RustFS）**：新增 `scripts/e2e-real.sh` + `make e2e-real`，自动 docker 起一份真实 RustFS（显式配 `RUSTFS_CORS_ALLOWED_ORIGINS`，否则浏览器直传被 CORS 拦下）→ `pnpm build` 真实产物 → 起真实 Go 后端（`S3C_STATIC_DIR` 托管产物）→ 跑 `pnpm e2e:real` → `trap` 自动清理；支持 `--keep` / `--skip-build`。
- **两套 CI 接入**：GitHub 新增 `.github/workflows/e2e-real.yml`（PR 按路径 + 手动 + 每周五定时）；GitLab 新增 `e2e-real` job（RustFS 用 GitLab service，与 `rustfs-e2e` 同款，service 变量里配好 CORS）。触发规则与另两套 E2E 对齐，不阻塞普通 push。
- **编排收敛为单一来源**：初版把「起 RustFS → 构建产物与后端 → 起后端 → 跑用例」在脚本 / GitHub / GitLab 各写了一份（改一处漏两处即漂移）。现 `scripts/e2e-real.sh` 是唯一编排：两套 CI 都 `bash scripts/e2e-real.sh`（GitHub 自起容器；GitLab 用 `--no-rustfs` + `RUSTFS_ENDPOINT` 复用 service），各自只留「装工具链 / 装浏览器系统依赖」。脚本同时补上 `curl` 依赖检查、`--no-rustfs` 时只清自己起的容器、以及幂等的 `playwright install chromium`（修掉新克隆下 `make e2e-real` 直接失败的 UX 缺口）。
- **机械门禁防漂移**：`TestRealE2EUsesSharedScript`（两侧 CI 必须**实际调用** `bash scripts/e2e-real.sh`——仅出现在 `paths:`/`changes:` 里不算）、`TestRustFSImageIsConsistentlyPinned`（脚本默认值 / compose / GitLab service 三处镜像版本必须一致，禁 `latest`）、`TestRealE2EArtifactsExist`（联调 spec 出现 `page.route(` 即红灯）、`TestE2ESourcesAreTypechecked`、`TestLocalRealE2ETargetExists`。全部门禁均经变异验证（改镜像版本 / 删 CI 里的脚本调用 / 删 `typecheck:e2e` 都会红灯）。
- **补齐 E2E 源码的静态检查**：`e2e/` 与 `e2e-real/` 此前不在主 `tsconfig.json` 的 `include` 内，**零类型检查与 lint**（`pnpm lint` 只扫 `src`）。现新增 `apps/web/tsconfig.e2e.json` + `pnpm typecheck:e2e`，并把 `pnpm lint` 扩到 `eslint src e2e e2e-real`（顺带修掉一个此前未被发现的未使用变量）；两套 CI 的 `web` job 与本地 `make check` 都纳入。
- **变异验证**：去掉 RustFS 的 `RUSTFS_CORS_ALLOWED_ORIGINS` 后重跑——浏览器直传用例红灯，前两条不经浏览器的用例仍绿，证明该用例确非空断言。

### 修复（2026-09-22 审查 §9.3 P2 收尾 · 正确性 + 发布链 + 门禁质量）
- **D1 Prometheus 直方图语义违规（`/api/metrics` 输出全错）**：`s3wrap` 内部按「每次调用只 +1 个桶」记录**增量**，但 `handler/metrics.go` 直接把它当**累积**值按 `_bucket{le=…}` 输出——实测 `le="0.01"`=3、`le="+Inf"`=0、`_count`=3，违反 Prometheus 契约（`le` 必须单调不减且 `+Inf == _count`），`histogram_quantile()` 结果全错。现 `MetricsSnapshot` 在快照时把增量累积为真累积值；并把 `calls` 从 `atomic.Int64` 移入同一把锁（此前 `calls` 与 `buckets` 分属两处更新，快照可读到 `+Inf`=0 而 `_count`=3 的不一致）。注释同步改为准确描述。回归测试：`TestS3MetricsHistogramIsCumulativeAcrossBuckets`（多耗时累积值逐桶断言）、`TestS3MetricsHistogramConcurrentContract`（8×50 并发下契约不变）、`TestMetricsHistogramSatisfiesPrometheusContract`（端点级：解析 `le` 单调性与 `+Inf == _count`）。已用「还原增量语义」变异验证三条均红灯。
- **P3 `/api/openapi.json` 每次请求全量 marshal**：70 个 operation 此前每个请求都重新渲染。现 `Registry` 增加结果缓存，**任何注册动作（`Operation` / `Param` / `Respond` / `SetInfo` / `AddServer`）都置空缓存**，语义与「每次重新 marshal」完全一致；并发调用双检加锁只计算一次，且返回缓存副本（调用方改写不污染后续请求）。回归测试：`TestRegistry_MarshalJSONCached`（含副本隔离）、`TestRegistry_MarshalJSONCacheInvalidated`（四类注册动作逐一验证失效）、`TestRegistry_MarshalJSONConcurrent`。
- **D4 `failKeys ≤ 200` 承诺与实现不符**：`docs/features.md` / `docs/api.md` 承诺「failKeys ≤ 200」，但服务端只有异步 `delete-prefix` 在 handler 内裁剪，`copy-objects`（同步/异步）、`migrate`（同步/异步）、`migrate/sync` 全部原样回传——10k 全失败会回约 10 MB 并落盘 `jobs.json`。现把上限提为 handler 层共享常量 `maxFailKeys = 200` + `capFailKeys`，**所有回传 `failedKeys` 的端点统一裁剪**；异步路径经新增的 `jobResultFromBatch` 在 `Job.Finish`（落盘）**之前**裁剪。`failed` 计数不受裁剪影响。`service/batch.go` 与 `copy.go` 里两处互相矛盾的注释（「由调用方裁剪」vs「RunBatch 已裁剪」）改为与实现一致。回归测试：`TestOlCopyManyFailKeysAll`、`TestOlMigrateSyncFailKeysCapped`、`TestOlMigrateAsyncFailKeysCappedBeforePersist`（直接检查 `jobs.json`，证明裁剪发生在持久化前）。
- **R7 本地 `make` 与 CI 不等价**：`Makefile` 缺 `lint`（golangci-lint）/ `govulncheck` 目标，`test-all` 不含覆盖率门禁（本地全绿不代表 CI 会绿）；裸 `pnpm install` 会改写锁文件，导致 CI 的 `--frozen-lockfile` 失败；`test-cover` / `install-hooks` 未列入 `.PHONY`。现新增 `lint` / `govulncheck` / `check`（聚合静态检查 + 双侧覆盖率）目标，全部 `pnpm install` 改为 `--frozen-lockfile`，`test-all: test-cover web-test-cover`，`.PHONY` 补全全部目标。回归门禁 `TestMakefileMirrorsCIGates`（含「所有已定义目标必须在 .PHONY」的机械检查）。
- **R9 GitHub 构建不传 `VERSION` build-arg**：镜像内版本回落到 Dockerfile 的过期字面量，而 GitLab 传 `VERSION=ci`、Makefile 传真实版本——同一个 Dockerfile 三种行为。现 GitHub 的构建与推送两处都显式注入 `VERSION=ci`。门禁 `TestGitHubWorkflowInjectsVersionBuildArg`。
- **R10 无 workflow 推送镜像**：「发布」此前只有 tag + 桌面安装包，用户无法 `docker pull` 到 CI 产物。新增 `publish` job：`needs: docker`（Trivy 扫描通过后才推送）、`if: github.event_name != 'pull_request'`、`permissions: packages: write`、`docker/login-action` 登录 GHCR、按 commit SHA 与分支双标签推送。因 buildx 的 `push` 与 `load` 互斥且扫描需 `load`，故拆为独立 job（也避免把未扫描的镜像推出去）。门禁 `TestGitHubWorkflowPushesImage`。
- **D5 `git tag v1.0.0` 不存在但 roadmap 称已收口**：用 `scripts/release-version.sh v1.0.0` 把 15 处版本串从 `1.0.0-rc1` 同步到 `1.0.0`（Makefile / Dockerfile / compose×2 / web+desktop `package.json` / `tauri.conf.json` / `Cargo.toml` / `Cargo.lock` / `main.go` / README / docs），并打上 `v1.0.0` tag。历史性提及（CHANGELOG 已发布区、roadmap 里程碑行、review 报告）按「不追溯篡改」纪律保留。
- **D10 `CHANGELOG.md` 内过期计数**：`[Unreleased]` 段内两处「当时实测值」（覆盖率 3883/2769/1059/3339、前端 63 文件 / 983 测试）后续用例增加后已失真。按本文件「不追溯篡改」的纪律**保留原值并加注**当时的时点与新值，不改写历史数字。
- **测试质量：21 个用例名硬编码源码行号且已过期**（如 `api.test.ts` 写 `line 125 else`，实际 `listServers` 在 `storage.ts:301`；`bucketPolicy.test.ts` 写 `line 111`，实际在 `:108`）——测试在描述一个不存在的版本，属实现耦合而非行为断言。已从全部用例名中移除行号（改为描述行为，行号留在注释里），并新增机械门禁 `deadcode_gate.test.ts` 的「测试名不得硬编码源码行号」（含扫描下限自检防解析失效）。
- **测试质量：i18n 门禁两处盲区**：① `usedKeyTexts` 对整份源码做正则，**注释里的键形字符串也算「已使用」**——删掉真实引用、只在注释留个键名，死键门禁仍全绿。现先按字符扫描剥离注释（正确跳过字符串字面量，避免把 `'https://x'` 的 `//` 当注释）再匹配。② 中英双语只比对**键数量**，`zh-CN` 缺 `a` 而 `en-US` 多 `b` 时数量相等、门禁仍绿，用户切到英文会看到原始 key。现按语言分别解析键集合做**集合级双向比对**，并新增「每个键的两种语言取值都非空」。两条均经变异验证（改名一个 en-US 键使数量不变 → 集合门禁红灯；删真实引用只留注释 → 死键门禁红灯）。
- **文档失真修正**：`openapi_contract_test.go` 的 `apiRouteCount` 注释算错（写「69 = 68 + 1」而常量与实际均为 70，应为「70 = 69 + 1」）；`docs/todolist.md` 补记 #40 / #42 / #43 / #44 为**从未启用的保留空号**（避免被误读为漏登记），并把 #15 补入已完成编号。
- **门禁实跑**：后端 `gofmt -l` 干净 / `go vet ./...` 0 告警 / `go test -race` 8/8 包通过且**每包 100.0%**；前端 `pnpm lint` 0 告警 / `pnpm typecheck` 通过 / `pnpm test:coverage` **66 文件 1042 用例全绿，四项覆盖率 100%**（statements 4074 / branches 2844 / functions 1095 / lines 3503）。

### 修复（2026-09-20 审查 §9.3 P2 · 契约 + 安全 / 供应链）
- **#26 OpenAPI 注册表漏声明 query 参数（含 1 个幻影参数）**：注册表未声明 4 个 handler 真实读取的 query 参数——`head.versionId`、`proxy.maxBytes`、`trash.prefix`、`objects.startAfter`（后者此前连 `docs/api.md` 都未记录），按 OpenAPI 生成的客户端根本不会发送它们。已全部补声明（`maxBytes` 为整数并注明默认 1 MiB / 上限 2 MiB）。**新增全量门禁** `openapi_query_params_test.go`：解析 `routes.go` → handler 方法调用闭包 → 机械抽取 `r.URL.Query().Get("x")` / `q.Get("x")`，与注册表 `in:query` 集合**双向**比对（含自检计数防解析口径失效）。上线即抓到**第 5 处漂移**：`GET /api/accounts/{id}/versions` 声明了 handler 从不读取的幻影参数 `key`——已移除。`docs/api.md` 同步补 `startAfter`。
- **#27 类型 / required / 枚举语义无门禁**：新增 `openapi_semantics_test.go`。① **枚举门禁**：每个注册表 `enum`（请求体字段或 query）必须在 `enumContracts` 登记，并从 handler 的 `switch` case 字符串 / `map[string]bool` 字面量 / 跨包 `service` 常量机械抽取真实接受值做双向比对（空串除外）；上线即发现 `PUT object-acl` 注册表漏了 handler 实际接受的 `authenticated-read` / `aws-exec-read`。② **required 门禁**：注册表声明 required 的请求字段必须是 handler 真实解码、且非「空值→默认」的字段；据此修正 `presign.method`、`copy-object.newBucket`、`copy-prefix.targetBucket` 三处「注册表 required 但 handler 对空值取默认」的错误声明。残留范围（未识别的 `bucketOr` 默认桶模式）写入文件头。
- **#28 响应门禁残留范围（端点内联 / `map[string]any`）**：扩展 `openapi_response_contract_test.go`，新增**端点级响应门禁**：对每个声明了具体 `properties` 的 2xx 响应，机械抽取 handler `h.writeJSON` 的表达式键（字面量 map 键、命名 struct 的 `json` tag、变量、同包 helper 返回值递归），与注册表属性做双向比对（`omitempty` 记为可选）。同时把 4 个注册表文件里约 30 个 `openapi.Obj()` 响应升级为 `BuildObj` 具体属性以纳入门禁，并修正 3 个异步端点误声明 `200`（handler 实为 `202`）。仍未覆盖的残留（其他注册表文件的自由体响应、运行时动态键 map）已在文件头写明。
- **#29 / #31 明文密钥落盘改为安全默认（硬失败）**：此前 `json` / `sqlite` 驱动在 `S3C_STORE_KEY` 为空时把 `secretKey` 明文落盘，仅启动 WARN。现 `Config.Validate` 直接返回新的 `ErrPlaintextStoreNotAllowed` **拒绝启动**，除非显式设置 `S3C_ALLOW_PLAINTEXT_STORE=1`（仅本地联调；此时仍打「明文落盘」WARN）。错误信息给出两条出路（设 `S3C_STORE_KEY` ≥ 16 字符 / 显式 opt-in），且不泄露任何密钥值。base `docker-compose.yml` 同步改为 `S3C_STORE_KEY: "${S3C_STORE_KEY:?…}"` 强制非空（缺失即 compose 启动失败）；`.env.example`（根与 `apps/server`）补 `S3C_ALLOW_PLAINTEXT_STORE` 说明；`e2e.yml` 因 `docker compose up` 会插值整个文件而补一次性 dummy key。回归测试：`TestValidatePlaintextStoreRejected`、`TestMainServerRejectsPlaintextStoreWithoutOptIn`、`TestMainServerAllowsPlaintextStoreWithOptIn`、`TestFromEnvAllowPlaintextStore`。
- **#30 `/api/metrics` 不受 `S3C_TOKEN` 保护（文档显式声明）**：行为不变（有意为内网 Prometheus 免 token scrape），但此前文档未说明「即使配了 token 也匿名可读」。已在 `README.md`（变量表 + 安全默认值）、`docs/api.md`、`docs/deployment.md` §6.3、`docs/threat-model.md` §2 显式声明，并新增 `TestMetricsEndpointUnauthenticatedEvenWithToken` 钉住该行为（配 token + 开 metrics 时无 `Authorization` 仍 200，而 `/api/accounts` 无 token 为 401）。
- **#32 Trivy 0.58.1 → 0.74.0 + digest 双 pin**：两套 CI 由 `aquasec/trivy:0.58.1` 升级到 `aquasec/trivy:0.74.0@sha256:62b1e65e…1969`（tag + digest 不可变 pin，digest 经 `docker pull` 与 ghcr.io / public.ecr.aws 清单三源核对）。CI 依赖的 `--vuln-type` 自 0.58 起 deprecated，改为 `--pkg-types os,library`（0.74.0 的 `trivy image --help` 与 `pkg/flag/package_flags.go` 已核实）。回归门禁 `TestTrivyImageIsVersionAndDigestPinned`：两套 CI 必须版本一致、必须带 digest、必须用 `--pkg-types` 且不得残留 `--vuln-type`。
- **#33 工具链 pin（Rust / Node / desktop pnpm）**：`dtolnay/rust-toolchain` 此前 pin 的 SHA 实为 `stable` **浮动分支 tip**（commit 即 `toolchain: stable`，并非版本 pin）——改为版本分支 SHA `ce678459…`（`# 1.98.1`）；新增 `apps/desktop/src-tauri/rust-toolchain.toml`（`channel = "1.98.1"`）；GitLab 桌面镜像 `rust:1-bookworm` → `rust:1.98.1-bookworm`。Node 由只 pin 大版本（`24`）收紧到精确 patch `24.21.0`（workflow `node-version`、`node:24.21.0-alpine`、`node:24.21.0-bookworm`、NodeSource `nodejs=24.21.0-1nodesource1`）。`apps/desktop/package.json` 补 `packageManager: pnpm@9.15.0`（与 web / CI 一致；已实测 `pnpm@9.15.0 --frozen-lockfile` 不破坏锁文件）。门禁：`TestRustToolchainIsVersionPinned`、`TestNodePinnedToPatchVersion`、`TestPackageManagerIsPinnedToCIPnpm`。
- **#34 `.dockerignore` 漏覆盖率 / 缓存目录**：此前漏掉 `apps/web/coverage`、`.pnpm-store`、`test-results`、`.run/`、`.cargo/` 等（本机实测 >4 MB，另有 `.gitlab-ci-local` 体积更大）→ 同一 commit 因本地是否跑过覆盖率而产出不同镜像层。已补齐上述目录及 `playwright-report`、`.trivy-cache/`、`coverage.out`/`.txt`、`*.tsbuildinfo`、`*.log` 等，并把过窄的 `apps/server/*.log` 换成 `**/*.log`。门禁 `TestDockerignoreExcludesBuildArtifacts`（11 条必需模式，注释行不计入）。
- **门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 -coverprofile` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）/ `go build ./...` OK；前端 `pnpm lint` 0 告警 / `pnpm test` 66 文件 1039 用例全绿 / `pnpm build` OK；`docker compose config` 在缺 `S3C_STORE_KEY` 时退出 1、补齐后退出 0。新增门禁均经「先红后绿」验证，供应链 pin 均经上游实测核验（Trivy / Node / Rust 版本与 digest）。

### 修复（2026-09-19 审查 P1 · §9.2 全部闭环）
- **§7.2 响应契约失真：`components.schemas.Account` 与真实 DTO 不符**：schema 声明了 `provider` / `forcePathStyle` / `insecureSkipVerify` 三个**幻影字段**（`model.AccountView` 无此字段），却**漏掉真实的 `useSSL`**——而它是 `GET /api/accounts`、`POST /api/accounts`、`GET|PUT /api/accounts/{id}` 全部 200/201 响应的契约，按此 schema 生成的客户端会读到永远为 null 的字段、并丢失 `useSSL`。已按 `model.AccountView` 逐字段修正。回归测试：新增 `TestOpenAPI_AccountSchemaIsAccountView`（逐字段断言 + `secretKey` 不得出现）。
- **新增响应契约门禁（handler DTO ⇔ OpenAPI schema 双向比对）**：此前所有门禁只覆盖**请求**方向，**没有任何门禁把「响应 schema」与真实响应 DTO 做比对**——这正是上一条长期存活的根因，也是审查 §4.3/§7.4 点名的唯一缺失门禁类型。新增 `apps/server/internal/handler/openapi_response_contract_test.go`：对 `components.schemas` 的每个共享 schema（`Account` / `Bucket` / `ObjectItem` / `ListObjectsResp` / `Error`）用 `go/parser` 机械抽取对应 Go DTO 的 `json` tag，做**双向**比对（schema 多写 = 幻影字段；漏写 = 客户端丢字段），并强制新共享 schema 必须在 `schemaDTOs` 登记（防逃逸）。上线即对 `Account` 红灯，修正后转绿。文件头写明断言范围：仅共享 schema，不含各端点内联 / `map[string]any` 拼装的响应。
- **§4.3 文档字段门禁改双向并去掉子串匹配**：`api_doc_test.go` 的 `TestAPIDocDocumentsRequestBodyFields` 原用 `strings.Contains(body, name)` **单向**校验（注册表 ⊆ 文档），有两处结构性盲区：① 文档多写的幻影字段从不检查；② 子串碰撞（字段 `key` 被 `keys`/`secretKey` 满足、`newKey` 被 `newKeys` 满足）。现改为机械抽取 `docs/api.md` 请求体 JSON 的**顶层键**（自写容错扫描器：剥离 `//` 注释、支持嵌套对象、区分 `200 {…}` 响应体）后**双向**比对，彻底移除子串匹配。别名段同时支持「请求体与 `POST /api/x` 相同」与省略方法名的「请求体与 `copy-prefix` 相同」。
- **§4.3 请求体字段门禁从「硬编码端点族」改为「全量遍历」**：原 `openapi_contract_test.go` 的字段断言是**逐端点族硬编码**（2026-09 修过的那批），新端点写错字段仍可 100% 全绿。新增 `apps/server/internal/handler/openapi_request_fields_test.go`：解析 `routes.go` → handler 方法调用闭包 → 定位真正 `readJSON` 的方法（覆盖 `parseMigrateRequest` / `parseCopyPrefix` 委托解码）→ 抽取解码结构体字段集，与注册表 requestBody schema **全量**双向比对（端点专属 DTO 双向相等；共享模型走 `serverManagedFields` 白名单；注册表自由体跳过并计数自检）。**上线即抓到 3 处客户端可见漂移**：`POST …/storage-class` 注册表漏 `versionId`、`POST /api/accounts/preview-buckets` 注册表与 `docs/api.md` 漏 `publicEndpoint`/`useSSL`——已修注册表与文档。
- **R2 无 tag ↔ 清单一致性门禁（P0 级）**：`release-desktop.yml` 解析出 tag 后**从不**与 `tauri.conf.json` / `Cargo.toml` 比对，给 `v1.0.0-rc2` 打 tag 会发布**标着 rc1** 的安装包。新增 `Verify tag matches manifest versions` 步骤：tag（去前缀 `v`）必须等于两处清单版本，否则 `exit 1`。本地实测 rc1 通过、rc2 正确失败。
- **R3 `SHA256SUMS` 跨平台竞态**：三平台矩阵此前各自 `gh release upload SHA256SUMS.txt --clobber` **同一资产名**，并行执行下「最后写入者胜」，发布出去的校验清单只覆盖一个平台。现改为各平台上传唯一命名的 `SHA256SUMS-<bundle>.txt`，新增 `aggregate-checksums` job（`needs: publish`）拉取三份合并去重为唯一 `SHA256SUMS.txt`（含行数自检防漏拉），并清理中间资产。
- **R4 Trivy DB 下载无缓存/重试**：仓库自带日志记录过 `FATAL … connection timed out`（6m44s 后失败），网络抖动会让安全门禁以「构建失败」形式变红。两套 CI 均拆出 `trivy image --download-db-only` + 4 次指数退避重试，扫描阶段加 `--skip-db-update` 复用已就绪 DB；GitHub 侧加 `actions/cache`（`.trivy-cache`），GitLab 侧加 `cache: paths`。
- **S1 运行镜像基础版 EOL**：`apps/server/Dockerfile` 运行阶段 `alpine:3.20`（安全支持已于 2026-04-01 结束）→ `alpine:3.24`（支持至 2028-06-01）。EOL 分支叠加 Trivy 的 `--ignore-unfixed` 会让「所有未来 CVE 永远无修复版」，安全门禁结构性失明。
- **S2 `.env.example` 占位 token 是「有效口令」**：根 `.env.example` 的 `S3C_TOKEN=change-me-use-openssl-rand-hex-32` 长 33 ≥ `MinTokenLength=16`，满足 compose 的 `${S3C_TOKEN:?}` 非空守卫——照 README 快速开始 `cp .env.example .env && docker compose up -d` 会以**公开已知 token** 上线。已置空（对齐 `apps/server/.env.example`）。
- **配置类回归门禁（防复发）**：新增 `apps/server/repo_infra_gate_test.go`，把上述四项配置断言变成会变红的测试——`.env.example` 的 token 不得 ≥ `MinTokenLength`；运行镜像不得是 EOL alpine（基线常量 `minSupportedAlpineMinor`，附 3.21 EOL 时须抬到 3.22 的说明）；发布 workflow 必须做 tag↔清单比对与平台内唯一 checksum；两套 CI 的 Trivy 必须缓存 + 重试。这四项都是「YAML/配置正确但断言缺席」的盲区。
- **门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go test -race -count=1 ./...` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）。新增门禁均经「先红后绿」验证。

### 文档（2026-09-19 审查 P1 同批：§7.3 文档失真 D2 / D3 / D6 / D8 / D9）
- **D2 修正 `openapi_handler.go` 的鉴权注释**：原注释称 `/api/openapi.json`「不经鉴权层」，与实际相反——配置 `S3C_TOKEN` 时无 token 返回 **401**（`withAuth` 只豁免 health/metrics），未开启 `S3C_EXPOSE_OPENAPI` 时返回 404。注释改为与实测一致。
- **D3 修发布脚本 `scripts/release-version.sh`**：正则原为 `^v1\.0\.0-<时间戳|rcN>$`，会**拒绝纯 `v1.0.0`**（roadmap 首个稳定里程碑）与 `v1.0.1` / `v1.1.0`；现放宽到通用 `vMAJOR.MINOR.PATCH` + `-rcN`/`-alphaN`/`-betaN`/时间戳。同步文件从 8 处补到 12 处（新增 `docs/deployment.md` / `docs/roadmap.md` / `docs/features.md` / `.github/ISSUE_TEMPLATE/bug_report.md` / `internal/openapi/openapi.go`）；`Cargo.lock` 改用 awk **只改本包** `name = "s3clinet"`，避免全局 `sed` 把 400+ 依赖包的 version 行改坏。已实测 `v1.0.0` / `v1.1.0` / rc 三种输入均正确同步。
- **D6 修 presign `expiresIn` 文档**：`docs/api.md` 原写「范围 1s–7 天、默认 15 分钟」，代码是**缺省默认 1 小时、超过 24 小时钳到 24 小时**（`objects.go`）；已改为与代码及 `architecture.md` 一致。
- **D8 修「3 路并发」失真**：现行能力描述（`README.md` / `docs/architecture.md` / `docs/features.md` 能力表）改为实际值 **2 路**（`UPLOAD_CONCURRENCY = 2`）；0.3.0 历史条目保留原样（不追溯篡改历史台账）。
- **D9 修 `docs/development.md` 存放约定自相矛盾**：原段先说 `CHANGELOG.md` 是根目录约定文件，随后又写「其余文档（含 `CHANGELOG.md`）统一放 `docs/`」；改为「除根目录约定文件与 `.github/` 社区健康文件外的其余文档放 `docs/`」。
- **SSOT 修复：`docs/todolist.md` 补登记本轮全部开放项**：此前该文件写着「无待办」而审查仍有大量开放发现，违反「本文件是唯一待办来源」的约定。现按 #26–#46 补登记 P2 全部条目（契约残留 / 安全 / 发布链 / 可观测性 / 文档失真），并注明编号稳定不重排。

### 修复（2026-09-19 审查 §三 正确性 · 后端 B3–B11）
- **B3 `DeleteObjects` 吞掉 200 响应体内的逐 key 失败**：S3 对「部分 key 删不掉」（桶策略 / 保留期 / MFA Delete）**仍返回 200**，只在响应体内列 `<Error>`；旧实现只看顶层 `err`，于是 4 个调用点全部把「请求数」当成「已删除数」——受保护对象删不掉，UI 却报「已删除 N 个」。现在 `s3wrap.DeleteObjects` 返回 `[]DeleteFailure{Key,Code,Message}`（`err` 只表示传输/协议失败，两者可同时非空），`POST …/delete` 回 `{"deleted":成功数,"failed":失败数,"lastError":"access denied"}`，`delete-prefix`（同步/异步）与回收站 `PurgeObject` 同样按「请求数 − 逐 key 失败数」记账，`PurgeObject` 遇部分失败上抛新的 `ErrPartialDelete`（回收站不再显示「已彻底清除」而版本仍在）。回归测试 `TestDeleteObjectsReportsPerKeyFailures` / `TestDeleteObjectsPartialFailureAcrossBatches` / `TestPurgeObjectRefusesPartialDelete` + handler 三端点 `TestOlDeleteObjectsReportsPartialFailure` / `TestOlDeletePrefixReportsPartialFailure` / `TestOlDeletePrefixAsyncReportsPartialFailure`。
- **B3 连带：递归删除的进度口径改为「已处理数」**：`runDeletePrefix` 原来只按 `deleted` 前进，桶策略让删除全部失败时同一页会被反复列出、反复失败直到 2h 任务超时。现按 `deleted + failed` 前进并在 `truncated` 处收口，`TestOlRunDeletePrefixProgressesOnAllFailures`（100 页全失败仍正常终止）锁死该行为。
- **B4 ZIP 打包遇首个拷贝错误即 `break` → 永久 goroutine + 连接泄漏**：`results` 是无缓冲通道，一旦停止消费，其余 3 个 worker 永久阻塞在发送上（`wg.Wait(); close(results)` 协程永不返回，已取回未写出的 body 不关闭）。触发路径包含**用户取消 ZIP 下载**（`ctxCancelReader` → `io.Copy` 报错）。改为 `continue` 后失败 key 照常进入清单、在途 body 全部关闭。测试 `TestWriteObjectsZipCopyErrorDoesNotLeak`（断言所有已取回 body 最终被关闭且 5 个 key 全部上报；旧实现会泄漏 3 个 body）。`features.md` 中「P-3 zip goroutine 泄漏 ✅」的结论至此才真正成立。
- **B5 增量同步吞掉列举错误**：源端 403/5xx 时 `SyncKeys` 回 `200 {scanned:0}`，用户读到的是「没有需要同步的对象」。现 `listAll`/`indexDst` 返回 `error`，`SyncKeys` 签名改为 `(SyncResult, error)`，handler 走 `writeInternalErr`（源端 AccessDenied → **403**）；`cancelOrErr` 把「ctx 取消（客户端放弃）」与「真实列举错误」分开：前者仍返回已完成的部分结果且不报错，后者必须上抛。测试 `TestMigrateSync_SourceListFailureIsReported`（端点级，断言 403 且响应体不含 `scanned`）、`TestSync_SourceListErrorIsReturned` / `TestSync_DstListErrorIsReturned`。
- **B6 `indexDst` 无页数上限且循环内不查 ctx**：旧的循环条件是「已收集数 < 10 万」，对端只要返回**不前进的** `NextContinuationToken` 就会空转到 2h 任务超时。现在 `listAll`/`indexDst` 共用一组硬上限（100 页 × 1000 key × 10 万总量）+ 每轮 ctx 检查 + 「NextToken 未前进即停」。测试 `TestIndexDstStopsOnNonAdvancingToken` / `TestListAllStopsOnNonAdvancingToken`（带 5s 超时护栏，旧实现会挂死）/ `TestIndexDstStopsAtPageCap` / `TestIndexDstStopsOnCancelledContext`。
- **B7 `Job.Emit` 在锁外投递、`Finish` 关闭同一批 channel**：`Emit` 先在锁内快照订阅者、解锁后才发送，`Finish` 在同一把锁下清空订阅表、解锁后 `close` 这些 channel——两者之间没有互斥，一旦并发就是 **send on closed channel → panic，进程直接退出**（无 recover）。现改为**在锁内非阻塞投递**（`select` + `default`，持锁时间有界），落盘 I/O 移到锁外。测试 `TestJobEmitFinishConcurrentDoesNotPanic` 用「节流落盘钩子」把 Emit 卡在「快照之后、投递之前」再调用 `Finish`——旧实现**确定性 panic**，新实现通过；另加 20 轮 × 50 次并发压测。
- **B9 8 MB 请求体上限与「10 000 key 批量」的承诺冲突**：`maxBody` 原为 8 MiB，而 `maxBatchKeys` 允许 10 000 个 key（1 KB/key ≈ 10.3 MB）——`io.LimitReader` 把合法请求截断成 JSON 语法错误，回的是 400「invalid request body」而不是 413。现上限提到 16 MiB，并用 `io.LimitedReader{N: maxBody+1}` 区分「超限」与「JSON 无效」：超限时 `writeBadJSON` 回 **413** + `request body too large (max 16MB)`。测试 `TestReadJSONAcceptsDocumentedMaxBatch`（10 000 × 1 KB 载荷可解析）/ `TestReadJSONRejectsOverLimitBody` / `TestReadJSONRejectsMaxBodyPlusOne` / `TestDeleteObjectsBodyTooLargeIs413`。
- **B10① 分段上传第 10000 段被误判超限**：`partNum++` 之后才判 `partNum > 10000`，而**段号 10000 是合法的最后一段**——正好 10000 段的对象会被 abort 并报「exceeds part limit」。改为**上传前**判断（`partNum > maxMultipartParts`），并把手写常量提为可注入的 `maxMultipartParts`（默认 10000，测试断言其未被改动）。测试 `TestMultipartStreamCopyAcceptsExactlyMaxParts`（旧实现必失败）/ `TestMultipartStreamCopyRejectsPartOverLimit`（第 N+1 段根本不发出）。
- **B10② `Content-Disposition` 的 `filename*` 用 `QueryEscape`**：空格被编成 `+`，而 RFC 5987 的百分号编码里 `+` 是字面加号——浏览器会把 `my file.txt` 存成 `my+file.txt`。新增 `rfc5987Escape`（按 attr-char 白名单逐字节 `%XX`，非 ASCII 走 UTF-8 字节）。测试 `TestOlProxyContentDispositionRFC5987`（含中文文件名）。
- **B10③ download-zip 不校验空 key**：`GET /bucket/?key=""` 会被 S3 当成「列举桶」，把 ListBucket XML 当成对象内容塞进 ZIP 包。现在空 key 直接 400。测试 `TestOlDownloadZipRejectsEmptyKey`。
- **B10④ `mode=text` 忽略读错误仍回 200**：`io.ReadFull` 的错误被丢弃，传输中断会被渲染成「内容只有这么多」。现在非 `EOF`/`ErrUnexpectedEOF` 的读错误走 `proxyErr`；对 `ErrUnexpectedEOF` 再用 `Content-Length` 判定是否短读（短读同样报错）。测试 `TestOlProxyTextMalformedChunkIsNot200`（畸形 chunked 编码）/ `TestOlProxyTextShortBodyIsNot200`。
- **B10⑤ 分段顺序不校验**：`CompleteMultipartUpload` 的 `Parts` 必须按 `PartNumber` 升序，乱序时 S3 回 `InvalidPartOrder`，而它没进错误映射表 → 用户拿到 **500**（其实是自己的请求错了）。现在 handler 在提交前拒绝降序/重复段号（400），`HTTPStatus`/`UserMessageForCode` 也补上 `InvalidPartOrder → 400 / invalid request`。测试 `TestOlMultipartCompleteRequiresAscendingParts`（乱序 400、升序 200）、`TestHTTPStatusCoversAllBranches` 新增用例。
- **B10⑥ 指标标签取自服务端原始 `<Code>`，基数无界**（同 §S5）：`errorClass` 直接返回 `ErrorCode(err)`，而错误码来自用户配置的**不可信**端点——对端每次返回不同的 `Code` 即可让 `s3c_s3_call_errors_total` 的序列数无限增长。现改为白名单（20 个已识别的 S3 错误码）内的原样返回，其余一律归 `other`。测试 `TestErrorClassUnknownCodeFoldsToOther`。
- **B11 `statusRecorder` 未覆写 `Write`**：只 `Write` 不 `WriteHeader` 时 Go 隐式发出 200，而 recorder 的 `written` 仍为 false、后续显式 `WriteHeader` 会再次透传（Go 打 superfluous 日志），并把日志/指标里的状态改写成与实际响应不符的值。现覆写 `Write` 标记「已写出」。测试 `TestStatusRecorderWriteMarksWritten` / `TestStatusRecorderWriteHeaderFirst`。

### 修复（2026-09-19 审查 §三 正确性 · 前端 F2–F10）
- **F2 虚拟列表窗口不随 `entries` 重置 → 渲染 0 行空白表**：`scrollTop` 只由滚动事件赋值，切换目录/过滤后 `start` 仍取旧偏移，`entries.slice(start,end)` 为空，而空态占位因子项非空被抑制。现抽出共用的窗口化数学 `src/virtualList.ts`（`virtualWindow` + 单一 `ROW_HEIGHT`/`OVERSCAN`），`ObjectList` 与 `MigratePanel` 在 `entries` 变化时把 `scrollTop` 归零并同步写回真实 DOM（不再依赖浏览器钳制反复触发 scroll 事件收敛）。
- **F3 客户端不做批量分片，撞服务端硬上限后整批失败**：`useObjectActions` 一次提交全部选中 key（服务端上限 1000）、`MigratePanel` 同理（上限 10 000），而选中量可达 2 万–20 万——「加载全部 → 全选 → 删除」在 >1000 时返回英文 400，**一个都没删**。新增 `src/limits.ts`（`DELETE_MAX_KEYS_PER_REQUEST=1000` / `MIGRATE_MAX_KEYS_PER_REQUEST=10000` + `batchKeys`），两处改为**分片串行提交并聚合**：单片传输失败计入该片但不中断后续分片，`deleted/failed/lastError` 汇总后按「全部成功 / 部分成功 / 整体失败」分别提示（新增 i18n 键 `objects.toastDeletePartial`），选中态在 `finally` 中复位。
- **F4 复制文件夹忽略 `truncated`**：>10 万对象时静默只复制 10 万却提示成功（同一弹窗的删除分支有检查，属复制粘贴分叉）。现在按 `start.truncated` 提示「仅复制一部分」（i18n 键 `dest.toastCopiedTruncated`）。
- **F5 `loadAllSourceObjects` 在分页循环内读实时 `sourcePrefix`**：续页 token 会串到新前缀上，列表混两个前缀。现在进入循环时快照 bucket/prefix，并以 `listGen` 代次守卫丢弃过期响应（成功/失败/finally 三处）。
- **F6 迁移 SSE 无空闲超时**：`reader.read()` 可永久挂起 → EOF 后的补偿轮询根本不会启动 → 按钮永久禁用。现在 45s 空闲超时（任何数据/心跳到达即重置），超时经既有 `onError` 上报并主动 `cancel` reader + `abort` 释放连接；EOF 补偿轮询语义不变。（未加总时长上限：空闲超时已消除永久挂起，硬上限会误杀仍在正常推进的长迁移。）
- **F7 复制分享链接对全部选中并发 presign**：5000 选中 = 5000 并发请求。现在复用 `batchMetadata` 的 4 路有界池（`boundedPool` + `BATCH_META_CONCURRENCY`）。
- **F8 `AccountsPanel.load` 无 try/catch**：后端不可用时 unhandled rejection（健康轮询正是为此设计）。现在捕获并渲染带「重试」按钮的错误条（i18n 键 `accounts.loadFailed`）。
- **F9 四处可靠性缺陷**：① `useHealthPoll` 改为代次 + `disposed` 守卫，`stop()`/卸载后在途探测不再续跑定时器；② `ObjectList` 的 ResizeObserver 改为 `watch(scrollEl)` 挂载（首屏是骨架屏，原 `onMounted` 路径下 `viewportH` 恒 480），与 `MigratePanel` 做法对齐；③ 行高统一为 **42px**（`virtualList.ts` 常量与 `.tbl-virtual .v-row` CSS 同源，此前常量 38 低于真实行高导致窗口与垫片错位）；④ `theme.ts`（模块级调用）/`store.ts`/`useObjectBrowser.ts` 的 `localStorage` 读写全部加 try/catch 降级（隐私模式/配额异常不再抛错，`theme.ts` 抛出会导致整站启动失败）。
- **F10 `request()/requestResponse()` 的 headers 覆盖陷阱**：`fetch(url, { headers, ...opts })` 中 `opts` 在后，调用方一旦传 `headers` 就会把合并好的 `Authorization`/`Content-Type` 整体覆盖掉。现改为先合成默认 headers 再 `Object.assign` 调用方 headers（同名以调用方为准，默认值不丢）。
- **门禁实跑**：后端 `go vet` / `golangci-lint run ./...` **0 issues** / `gofmt -l` 干净 / 8/8 包 `-race` 通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）；前端 `pnpm lint`（0 警告）/ `pnpm typecheck` / `pnpm test:coverage`（66 文件 / 1039 用例，四项覆盖率 100%：statements 4074、branches 2844、functions 1095、lines 3503）/ `pnpm build`（360.52 kB / gzip 110.85 kB）全绿；**真对端 RustFS E2E 4/4 通过**（`S3CLINET_E2E=1`，覆盖 12 MiB 分段组装、批量删除、桶配置、回收站 purge）。

### 修复（2026-09-19 审查 P0 清零）
- **P0 SSE 自旋：`interrupted` 任务的 `/events` 流现在收到终态帧后立即关闭**。`migrate_async.go` 的 SSE 循环用 `case p := <-ch:`（缺 `ok` 判断），而 `Job.Subscribe` 对已结束任务「推一帧后立即 close」——**已关闭的 channel 永远就绪**，于是终态帧之后死循环刷零值帧（实测 3 秒吐 37 MB 且流永不结束）；终态判定又只认 `done`/`cancelled`，漏掉了同属终态的 `interrupted`（重启恢复标记）。现改为 `case p, open := <-ch: if !open { return }` + `service.IsTerminalJobStatus(p.Status)`；并给 SSE 路由补上 `withStreamLimit`（此前唯一未受并发保护的流式端点），`Job.Subscribe` 增加**每任务订阅上限 16**（`Finish` 对每个订阅者都要投递，无上限会让终态关闭的最坏耗时线性增长），饱和时端点回 503。回归测试 `TestMigrateJobEventsInterruptedStreamCloses`（用真实 `jobs.json` 走恢复路径造 interrupted 任务，断言只收到 1 帧且流 EOF）与 `TestMigrateJobEventsSubscriberCap`；前者已用「还原旧循环」变异验证会红灯。
- **P0 前端白屏：`s3c.servers` 被写坏不再导致整站白屏**。`readServers` 直接把 localStorage 解析结果当可信数组用（`rawList.map((s) => s.id ?? '')`），`[null]` 直接抛 `TypeError: Cannot read properties of null (reading 'id')`；而 `App.vue` 在**渲染期**调用 `api.getActiveServer()`，异常打断渲染 → 白屏，连唯一能修复存储的 Server 面板也不可达。`[1]`/`["x"]`/`[{}]` 不抛错但会生成 `id:''` 的幽灵 server 且不修复存储；`[{"base":{}}]` 会在 `applyProfile` 抛 `replace is not a function`。现新增 `sanitizeServer()` 逐元素做形状校验（非对象 / 无 id 一律丢弃并回写修复后的清单），`getActiveServer()` 兜底为渲染期安全访问器（存储被禁用时返回 undefined 而非抛出）。测试：`api.test.ts` 覆盖 5 种坏存储 + 混合条目，新增 `App.corruptStorage.test.ts` 用**真实** `./api` 模块挂载真实 App 断言三种坏存储下 header/侧边栏正常渲染（白屏即挂不上 header）。原「条目只有 token 时补空串成 id:''」的用例改判为「丢弃并修复」——幽灵 server 正是本缺陷的一部分。
- **P0 增量同步永不收敛：`SyncKeys` 的目标 key 映射与比对口径统一**。过滤阶段用相对 key 判定（`dstPrefix + stripPrefix(so.Key, srcPrefix)`），复制阶段却把**完整源 key** 交给 `MigrateKeys`（后者裸拼接 `targetPrefix + k`）——源 `p/a.txt`、`dstPrefix=q/` 实际写到 `q/p/a.txt`，比较仍在看 `q/a.txt`，于是二次同步依旧判定缺失，重复 10 万次也不收敛。现抽出复制内核 `migrateKeys(..., dstKeyFor func(string) string, ...)`，`MigrateKeys` 与 `SyncKeys` 共用；`SyncKeys` 只保留**一个** `dstKeyFor` 表达式，比对与复制不可能再漂移。附带修 `stripPrefix` 的段边界：前缀 `p` 不再命中 `prefix/x.txt` 并削成 `refix/x.txt`（该 key 在目标侧永不存在，同样导致反复重拷）。测试：`TestSync_PrefixMappingConverges`（目标 key 正确 + 二次同步 `copied:0`）、`TestSync_PrefixSegmentBoundary`；handler 侧 `TestMigrateSync_PrefixFilter` 增加收敛断言。**行为变更**：`sourcePrefix` 非空时目标 key 现在是「targetPrefix + 相对路径」（此前是 targetPrefix + 完整源 key）——这正是 `docs/api.md` 对 `copy-prefix` 已声明的「targetPrefix 直接前置到相对 key」语义。
- **P0 CI docker job 结构性失败：`build-push-action` 补 `load: true`**。`setup-buildx-action` 默认 `driver=docker-container`，镜像只留在 builder 缓存；而 Trivy 通过 `docker.sock` 扫的是**本地 daemon**，因此该 job 在 GitHub 侧不可能通过（GitLab 侧用原生 `docker build`，故一直正常——两侧门禁并不等价）。现补 `load: true`，并新增 `docker image inspect s3clinet/server:ci` 前置步骤，把「镜像没进 daemon」这一结构性失败与「真的扫出漏洞」区分开。已用本地 `docker-container` builder 复现：不加 `--load` 时构建 exit 0 但 `docker image inspect` 失败，加 `--load` 后成功。
- **P0 OpenAPI 契约幻影字段清零**：注册表 + `docs/api.md` 声明、handler 的请求结构体却**不含**的字段共 6 个——`POST /api/accounts` 的 `provider`、`POST …/rename` 的 `replaceTags`、`POST …/set-headers` 的 `cacheControl`/`contentEnc`/`contentLang`/`disposition`、`POST …/multipart/init` 的 `metadata`。因 `readJSON` 用 `DisallowUnknownFields`，**按文档示例体调用必然 400**；`set-headers` 更严重——文档承诺的「编辑 Cache-Control / Content-Language / Content-Encoding / Content-Disposition」这个功能并不存在（前端 `HeadersDialog` 也只发 contentType/metadata）。三处均无实现意图，故从注册表与文档删除；同时补上真漏记的 `useSSL`（前端 `AccountInput` 一直在发，只是没被文档声明）。`TestOpenAPI_ContractRequestBodyMatchesHandlers` 新增第 8 项：这四个端点的注册表字段集必须与 handler 结构体字段集**完全相等**（两端任一方向漂移都会变红，含 `useSSL` 缺失这类反向漏记）。

### 重构（前端 API 模块拆分）
- **`src/api.ts`（838 行单文件）拆分为 `src/api/` 目录 7 个模块**：`storage.ts`（浏览器凭据 / 多服务器 profile，339 行）、`endpoints.ts`（`s3api` 领域端点，192）、`jobs.ts`（异步任务 SSE + EOF 状态回读，119）、`download.ts`（ZIP 流式落盘，85）、`index.ts`（公开面 barrel，82）、`upload.ts`（预签名直传，48）、`http.ts`（传输层，38）。原文件把凭据存储、传输、~70 个领域端点、SSE、XHR 上传混在一处，内聚性已失。模块间为单向依赖 `index → {endpoints, jobs, download, upload} → http → storage`，无环；`./api` 与 `../api` 仍解析到 `api/index.ts`，**全部既有 import 路径与对外契约不变**。顺带消除 `migrateJobStatus` 的重复定义（拆分时 endpoints 与 jobs 各写一份 URL 与返回类型），统一由 `jobs.ts` 实现、`s3api` 引用同一函数。本次为**纯搬运**：`fetch` 的 headers 合并顺序等既有语义原样保留（`docs/review-2026-09-19.md` §F10 的潜伏缺陷不混入本次改动）。验证：`vue-tsc` / `eslint` 通过、986 例（64 文件）单测全绿、覆盖率四指标 100%（statements 3941 / branches 2787 / functions 1082 / lines 3392，7 个新模块全部满覆盖）。

### 新增（前端死代码门禁）
- **`apps/web/src/deadcode_gate.test.ts`：API 公开面「导出后无人引用」门禁**（对应后端 `apps/server/deadcode_gate_test.go` 的前端半边）。`s3api.*` / `api.*` / 具名导出的每个成员必须在**至少一个非测试源文件**中被引用——eslint 与 `vue-tsc` 只报未使用的**局部变量**，看不见导出符号，而测试文件的引用会让死代码一直"活着"（真实事故见下条移除项）。实现要点：用 `import.meta.glob(?raw)` 读源码（不引入 `node:fs` / `@types/node`，维持前端依赖最小化），按 **import 别名精确匹配 `别名.成员`**，不做裸词或子串匹配——避免 `'migrate'` 这类字符串字面量与 `migrateAsync` 这类前缀碰撞造成假绿。三重自检防"空跑变绿"（扫描文件数 ≥50、公开面解析 ≥50、引用总数 ≥30）+ `INTENTIONAL_UNUSED` 豁免清单（附"豁免不得腐烂"检查）。**变异验证**：注入 `s3api.zzDeadProbe` → 红灯；再复刻历史缺陷（`s3api.copyFiles` 仅被 `api.test.ts` 调用）→ **仍红灯**，即"只被测试引用"不被算作使用。

### 移除（前端死 API 与生产测试接缝）
- **删除 5 个零生产调用的 `s3api` 方法**：`copyFiles` / `deletePrefix` / `copyPrefix` / `migrate` / `migrateSync`——全仓仅 `api.test.ts` 引用，属"被测试引用掩盖的死代码"。对应后端端点保留，仅前端 UI 未使用。同时删除 2 个零生产引用的 barrel 导出：`requestResponse`（仅 `download.ts` 内部使用）、`downloadZipToDisk`（生产只走 `s3api.downloadZipToDisk`）；相关测试改为从 `./api/http`、`./api/download` 直接导入。
- **生产包不再携带测试专用接缝**：`s3wrap.ResetMetrics`（导出且注释自称"仅测试使用"）移入 `metrics_test.go` 内的 `resetMetrics()`，不再进入生产二进制；`service.SetMaxJobsForTest` 删除，在册任务上限由**可变全局** `var maxJobs` 改为**每实例**构造期选项 `NewJobRegistry(WithMaxJobs(n))`——原写法会跨用例、跨注册表实例相互污染（并发测试下即数据竞争），handler 侧接缝移到 `internal/handler/export_test.go`（仅 `go test` 时编译），并以行为用例（上限耗尽 → `ErrTooManyJobs`、非正上限回落默认值）替换原「钩子返回旧值」用例。

### 测试与质量门禁（2026-09-19 浏览器 E2E 空转用例）
- **消除 Playwright 的 2 个「空转」用例：14 passed / 1 skipped → 15 passed / 0 skipped**。「账号管理：空状态 → 新增 → 列表」用 `getByRole('link')` 找侧边栏入口，而 `App.vue` 的导航渲染为 `<button>`（实测 link 命中 0 / button 命中 1）→ `isVisible()` 恒为 false → `test.skip(true, '侧边栏没有账号入口（可能是 Tauri-only 视图）')`。**它不是 Tauri-only 场景，是定位器写错被 skip 掩盖**：该用例的空状态断言 `/暂无|empty|no account/i` 与中英两版真实文案均不匹配（实测 `false`），即真正执行也会失败——说明它从未跑过。现改为 role=button 硬断言入口可见（找不到即失败，不再 skip）+ 校验真实空态文案 + 走完「新增登录 → 填表 → 保存」，断言表单字段进入 POST 请求体、新增行渲染、弹窗关闭。第二个用例原本只 `page.evaluate(fetch)` 一次并断言 fixture 自己写死的 `version`（前端完全不参与，app 全坏也会绿），现改为断言用户可见的恢复行为：`/api/accounts` 先 500 → 错误横幅 + 「连接异常」徽标，后端恢复后 `useHealthPoll` 自动重拉账号并清除错误态。两个用例均经变异验证（还原旧定位器 / 打断 `onRecover` 都会红灯）。

### 修复（OpenAPI 契约字段级漂移）
- **字段级契约门禁 + 三处客户端可见的注册表失真**：新增 `TestAPIDocDocumentsRequestBodyFields`——OpenAPI 每个 `requestBody` 的字段名必须出现在该端点 `docs/api.md` 正文（支持「请求体与 `POST /api/x` 相同」别名）。门禁上线即抓到：`POST …/mkdir` 注册表写 `prefix`（handler / 前端 / 文档一直用 `key`）、`POST …/copy-objects` 与 `/async` 写 `items`（真实字段是 `keys`，异步侧此前只声明空对象）、`DELETE /api/accounts/{id}/version` 把 query 参数误声明为 requestBody。按 OpenAPI 生成的客户端用这些字段会被 `DisallowUnknownFields` 直接 400 / 收不到参数。已修注册表并加回归断言（`openapi_contract_test.go` 第 7 项 + 新增 `requestQueryParams` 助手），并新增通用输入源门禁 `TestOpenAPIRequestDeclarationMatchesHandlerInput`（routes.go → handler 调用闭包找 `readJSON` ⇔ 注册表 `Request:` 双向比对，含 `withStreamLimit` 包装与委托解码；经变异测试验证会红灯）；`docs/api.md` 补齐 `provider` / `publicEndpoint` / `replaceTags` / `cacheControl` / `contentEnc` / `contentLang` / `disposition` / `metadata` 等漏记字段。
- **消音式死代码门禁**：`golangci-lint`（`run.tests: true`）实测**能**报未引用的测试函数/方法，盲区只有显式消音 `_ = ident` / `var _ = expr`。新增 `apps/server/deadcode_gate_test.go` 扫全仓 Go 源码禁止这两种形状（放行 `_ = f()` 这类显式忽略返回值），并自检扫描文件数避免路径写错静默变绿；本轮清掉 11 处遗留（含 `var _ = errors.New // 保持 errors 引用位`、只声明不用的 `deletes`、`PurgeObject` 里的冗余 `_ = keyMarker`）。

### 新增（可靠性与安全）
- **R8 完善：存储硬失败可观测 + SSRF 生效策略可见**：`/api/metrics` 新增 `s3c_store_up`（store 掉线为 0，与既有 `/api/health` 503 一起构成 ADR-002 的可告警面）与 `s3c_ssrf_deny_private`（反映 `S3C_SSRF_DENY_PRIVATE` 生效值）；启动日志新增 `ssrfDenyPrivate` 字段；新增 `s3wrap.DenyPrivateNetworks()` getter。测试 `TestMetricsStoreUpAndSSRFPolicy` 覆盖两个 gauge 的 1/0 两态。文档同步 [api.md](docs/api.md) 指标清单、[deployment.md](docs/deployment.md) §6.1（503 处置顺序 + 告警建议）与 §6.3、[threat-model.md](docs/threat-model.md) 边界 D；`roadmap.md` §5.1 把 R8 移入「已决策接受（ADR 兜底）」索引，登记表只剩 R5（唯一开放项）。
- **DataDir 单写者锁**：文件型 store（json/sqlite）+ 内存 JobRegistry 只支持单副本，此前只靠文档约束。新增 `store.AcquireDataDirLock`（unix `flock(LOCK_EX|LOCK_NB)` + `<DataDir>/.s3clinet.lock`），第二个实例启动即失败返回 1，内核在进程退出时释放锁（无陈旧锁文件）。测试覆盖互斥、释放后重加、目录不可建 / 锁文件被占、`runServer` 端到端拒绝启动；`GOOS=windows` 构建通过（非 unix 为文档化 no-op）。
- **`S3C_SSRF_DENY_PRIVATE` 可选 SSRF 加固**：置 `1` 后创建期与拨号期校验连 RFC1918 / ULA / 回环 / 未指定地址一并拒绝；默认关闭，保持 [ADR-003](docs/decisions/0003-ssrf-private-allow.md) 的自托管主场景（MinIO / RustFS / 局域网放行），ADR-003 增加 Update 段。测试：`TestDenyPrivateNetworksOptIn` / `TestDenyPrivateNetworksDialGuard` / `TestFromEnvSSRFDenyPrivate`。文档同步 `.env.example`（根 + server）、README 配置表、[deployment.md](docs/deployment.md) §2.1、[threat-model.md](docs/threat-model.md) 边界 D、[architecture.md](docs/architecture.md) 关键机制 / 取舍、[decisions/index.md](docs/decisions/index.md)。
- **RustSec 审计入 CI**：`cargo audit`（pin `cargo-audit 0.22.2`）加入 GitHub `ci.yml` 与 GitLab `desktop` job，新增本地 `make rust-audit`。实跑 **0 漏洞**；7 条 unmaintained / unsound 告警（`proc-macro-error`、5 个 `unic-*`、`glib 0.18.5`）逐条 triage，均为上游无修复版本的传递依赖，不用 ignore 清单掩盖。
- **分段上传缺 ETag 的失败路径可见 + 多厂商兼容矩阵**：新增 `upload.test.ts` 用例断言「2xx 但读不到 ETag → 报错并在组装前 abort 清理分段」；README 补 RustFS / MinIO / AWS S3 / 阿里 OSS / 腾讯 COS 的 CORS `ExposeHeader: ETag` 要求矩阵（RustFS 为唯一自动化真对端 E2E）。

### 变更（安全：明文落盘告警）
- **`json` / `sqlite` 驱动空 `S3C_STORE_KEY` 时启动打 WARN 告警**：此前 base compose 默认 `sqlite` + 空 key，`secretKey` 明文落盘却只在文档里提示，运行时没有任何信号。新增 `config.Config.StorePlaintextWarning()`（返回可执行文案：驱动名 / `DataDir` / 改用 `encrypted` 或设 ≥16 字符的 `S3C_STORE_KEY`），`runServer` 在配置校验后 `logger.Warn` 输出；`encrypted` 或非空 key 不告警。测试：`TestStorePlaintextWarning` 六例表驱动（先失败后实现）+ `TestMainServerWarnsPlaintextStore` 子进程断言启动日志确实出现该告警。文档同步 [deployment.md](docs/deployment.md) §2.1、[threat-model.md](docs/threat-model.md) 边界 C、[roadmap.md](docs/roadmap.md) §5.1 R3（🟡 → 🟢）。
- **旧 `roadmap #N` 引用清零（27 处 / 25 文件）**：2026-09-17 路线图收口后，代码注释仍指向已作废的旧编号。带 `ASSESSMENT` 编号的只保留该编号，纯 roadmap 编号改为「已闭环：features.md §M」，`docker-compose.yml` 的密钥加密说明改指 §M + §5.1 R3；并给 [roadmap.md](docs/roadmap.md) §六 增加第 6 条编号引用规则。涉及 `apps/server`（config / store / handler / s3wrap）与 `apps/web`（App.vue、useHealthPoll、useBucketSetting、vite.config.ts 等）。
- **桌面端分发与签名正式立项（暂不处理）**：登记为 [todolist.md](docs/todolist.md) #25，阻塞依赖为 Windows 代码签名证书 / Apple Developer ID + 公证（[roadmap.md](docs/roadmap.md) §5.2 E6）。

### 文档（路线图风险登记）
- **风险登记只留「还需要人看的项」**：R1/R2/R3/R4/R6/R7 已收敛为自动化门禁，从 §5.1 登记表移入同节末尾的「已收敛」索引（编号不重排，todolist #25 / §三 #1 / ADR / 代码注释的引用保持有效）；§5.1 现在只剩 R5（桌面签名，🟡 开放）与 R8（已决策接受，➖）。§5 说明、§四 守卫段与 §六 维护约定同步改写。
- **`docs/roadmap.md` §五 重构为「风险登记表 + 依赖清单」**：原表 6 行中 5 行写的是**已闭环**的缓解措施（与本文件「只列未完成项」的约定冲突，历史证据本应归 `features.md`），且标题含「依赖」却没有依赖内容。现拆为 §5.1 风险登记（新增 `R*` 编号、**可观测触发信号**、等级、状态图例 🟢 有门禁 / 🟡 开放 / ➖ 已决策接受、守卫与跟踪）与 §5.2 依赖清单（Go 工具链、AWS SDK v2、modernc sqlite、RustFS 镜像、actions pin SHA、签名证书、GitHub Release 通道、S3 厂商 CORS/ETag、S3C2/S3C3 兼容承诺），并补「复审规则」。同时补登此前缺失的开放风险：base compose 明文密钥（R3）、单副本假设（R4）、桌面未签名且无自动更新（R5）、上游 ETag/CORS 差异（R7）、Rust 依赖审计缺口（R6）。另修正编号说明：代码注释里的 `roadmap #N` 指 2026-09-17 收口前的**旧编号**，不可按当前编号回读。
- **`docs/roadmap.md` 全文与新 §五 对齐**：§二 澄清「依赖」列仅指里程碑前置关系并把外部依赖指向 §5.2；§三 #1 补 `R5` / `E6` 交叉引用；§四 标明本表即 §5.1 中 🟢 风险的生效守卫；§六 新增第 6 条「编号引用」规则（新引用必须带 `§三 #N` / `R*` / `E*` 前缀）。[`docs/threat-model.md`](docs/threat-model.md) 边界 C 的旧 `roadmap #2` 引用改为指向 `features.md` §M 与 §5.1 R3（残留风险：base compose 仍可空 `S3C_STORE_KEY`）。

### 变更（本地 CI 体验）
- **gitlab-ci-local 本地跑法收口**：提交 [`.gitlab-ci-local-env`](.gitlab-ci-local-env) 默认挂 `/var/run/docker.sock`（对齐正式 runner，免每次 `--volume`）；`Makefile` 增加 `gcl` / `gcl-list` / `gcl-docker` 并 pin `gitlab-ci-local@4.75.1`；新增 [`.gitlab-ci-local-variables.yml.example`](.gitlab-ci-local-variables.yml.example) 说明国内可覆盖的 `GOPROXY` / `NPM_REGISTRY` / `TRIVY_DB_REPOSITORY`（复制为 gitignore 的 `.gitlab-ci-local-variables.yml` 即生效）。Trivy 默认 DB（`mirror.gcr.io`）超时无需改 script——设 `TRIVY_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-db:2` 即可。文档入口：[`docs/development.md`](docs/development.md) §3、README、`.gitlab-ci.yml` 头部注释。

### 变更（目录结构）
- **monorepo 布局：`server/` / `web/` / `desktop/` 统一迁入 `apps/`**：三个应用此前平铺在仓库根目录，与 `docs/` / `deploy/` / `scripts/` 混在同一层，根目录随应用增多持续膨胀。现统一为 `apps/server`、`apps/web`、`apps/desktop`（一律 `git mv`，保留历史）。Go module 路径随之由 `github.com/weilai1949/s3clinet/server` 改为 `github.com/weilai1949/s3clinet/apps/server`（84 个文件、121 处 import 同步改写）。构建与运行链路全部对齐：`Makefile`、`apps/server/Dockerfile`（构建上下文仍为仓库根，`COPY apps/...`）、三套 compose、`.dockerignore`、`.gitignore`、`scripts/*.sh`（`run-dev` / `graceful-restart` / `release-version` / `nginx-local`）、`.githooks/pre-commit`、4 个 GitHub workflow、`.gitlab-ci.yml`。Tauri 的 `tauri.conf.json` 用相对路径（`../web`、`../../web/dist`），随目录整体移动后无需改动即保持有效。
- **修复迁移后 `S3C_STATIC_DIR` 默认值失效**：默认值原为 `./web/dist`，是相对**进程工作目录**解析的。所有文档化的启动方式（`make server`、README 的 `cd apps/server && go run .`）都以 `apps/server` 为 CWD，迁移后该默认值会解析到不存在的 `apps/server/web/dist`，静态托管静默 404。已改为 `../web/dist`，并同步 `apps/server/.env.example` 与 README 配置表说明（先写失败断言再改实现）。

### 移除（不留尾巴）
- **`acc_helper_test.go` 中一整套失效的 S3 错误注入机制**：`accFailInjector.set` 是 `rules` 的**唯一写入方**，却从未被任何测试调用——于是 `rules` 恒为空 map、`take` 恒返回 false，`accErrXML` 与 `accSettingsFake` 也随之不可达（`accSettingsFake` 本身也无调用点）。更关键的是它**被覆盖率掩盖**：测试文件不参与 instrumentation，100% 门禁只看生产代码，所以这段死代码在门禁全绿的情况下长期存在。桶级配置（encryption/cors/website/policy/tagging）已由后写的 `bucket_settings_test.go` 用自带假 S3 完整覆盖，注入器属于被取代的遗留物。已删除 `accS3Fail` / `set` / `accErrXML` / `accSettingsOp` / `accSettingsFake` 与 4 个仅它使用的 XML 常量，`accBucketsFake` 去掉无人可填的 `inj` 参数（改为 `accBucketsFake()`），`newAccFailInjector` 一并移除。
- **前端 lint 门禁由 `--max-warnings 50` 收紧为 `--max-warnings 0`**：实测 `eslint src` 当前为 **0 errors / 0 warnings**，`50` 的阈值等于给后续改动留了 50 条告警的沉默额度，与新增的「机械检查零告警」规则冲突。收紧后 `pnpm lint` 与 `pnpm typecheck`（`vue-tsc --noEmit`）均通过。
- **测试代码中的 12 处 `_ = x` 消音与假断言**：按新增的「死代码零容忍」硬约束（见 `AGENTS.md` 第 5 条）清理——`gaps_test.go` 的 `_ = res`（`res` 已在上一行 `defer res.Body.Close()` 使用）、`ctx2`/`cancel2`（未使用，改用 `func() {}`）、`_ = r`（`r` 下一行即用）、`_ = in`（循环变量被丢弃，改为真正断言每个输入）、`sub`/`release`（订阅与信号通道均未被消费）、`job_persist_test.go` 的 `_ = job`、`cover_extra_test.go` 的 `_ = newCancelReader(...)`（Go 允许直接丢弃返回值）、`stream_copy_test.go` 的 `var _ = StreamCopy`（编译期形状断言，对已被常量断言覆盖的符号是冗余）、`metadata_test.go` 的 `_ = httptest.NewRecorder // 兼容 import 暂未使用`（该文件已用 `httptest` 7 次，纯属死 shim）、`acc_core_test.go` 的 `_ = up.String() // 触发 uptime 闭包`（改为断言非空）。这些模式 `golangci-lint` **看不见**（`_ =` 正是它的消音手段），只能靠人工审查与显式规则拦住。
- **26 个从未被引用的 i18n 死键**（`app.name`、`common.back` / `confirm` / `danger` / `default` / `done` / `empty` / `failed` / `none` / `search` / `selected` / `test` / `upload` / `waiting`、`objects.bucket` / `prefix`、`toolbar.delete` / `downloadZip` / `filter` / `refresh`、`trash.empty`、`upload.queue`、`server.save`、`batchEdit.aclNoChange` / `storageNoChange`、`bucketTags.errEmptyKey`）：中英各 26 行，字典键数 695 → 669。此前只校验「被引用 → 有定义」，「有定义 → 被引用」无人校验，而 i18n 目录又被排除在覆盖率统计外，死键因此长期堆积（2026-09-16 评估 D7）。现于 `apps/web/src/i18n/coverage.test.ts` 增加**反向门禁**：字典中任何既非字面量引用、也不匹配动态拼接模式的键都会让测试变红。动态键（`` `provider.${p}.label` `` 等 4 处）按「字面命名空间前缀 + 插值」形状生成豁免正则，避免把 `/api/accounts/${id}` 这类非 i18n 模板串误当豁免而让门禁恒真。
- **`@vitest/coverage-v8` 依赖**：`vite.config.ts` 的覆盖率 provider 是 `istanbul`，v8 provider 从未启用，属重复依赖。已从 `apps/web/package.json` 移除并重算 `pnpm-lock.yaml`（`pnpm install --frozen-lockfile` 仍通过；锁文件里残留的 `coverage-v8` 条目仅为 vitest 的 optional peer 声明，不再安装）。覆盖率仍为 100%（statements 3883/3883、branches 2769/2769、functions 1059/1059、lines 3339/3339）。<sup>†</sup>
  > <sup>†</sup> 上述四项计数是**该条目撰写当时的实测值**，后续用例增加后已变化（2026-09-21 实测：statements 4074 / branches 2844 / functions 1095 / lines 3503）。按本文件「不追溯篡改已发布区」的纪律保留原值并加注，不回写（review-2026-09-19.md §7.3 D10）。
- **`ObjectList.vue` 的 `visibleCount` prop**：父组件 `ObjectsPanel.vue` 传入、组件内从不读取（空态判断实际用 `totalCount`），属迁移遗留的无效契约。已连同父组件绑定与测试 fixture 一并删除。`ObjectToolbar` 的同名 prop **保留**——它真实用于 `{{ visibleCount }}/{{ totalCount }}` 计数展示。
- **`.gitignore` 由 202 行精简至 81 行**（其中非注释的有效规则 45 条）：删去 28 行与本站技术栈无关的脚手架模板条目（`bower_components`、`jspm_packages/`、`web_modules/`、`.next`、`out`、`.nuxt`、`.output`、`.parcel-cache`、`.svelte-kit/`、`.docusaurus`、`.serverless/`、`.fusebox/`、`.dynamodb/`、`.firebase/`、`.tern-port`、`.vscode-test`、`lib-cov`、`.grunt`、`.lock-wscript`、`build/Release`、`*.tgz`、`.yarn-integrity`、`.node_repl_history`、`.stylelintcache`、`.npm`、Gatsby / vuepress / vitepress 段等），并按**实测**（逐条注释掉该规则后 `git check-ignore` 复查）删除 4 条被通用规则覆盖的冗余项（`apps/server/data/` ← `data/`、`apps/web/node_modules/` 与 `apps/desktop/node_modules/` ← `node_modules/`、`apps/web/dist/` ← `dist/`）。删除前后 `git status --ignored` 的忽略集合逐行比对一致，唯一差异是把过窄的 `.cache/` 收敛为 `.cargo/`（前者会顺带忽略任意层级的 `.cache/`，后者才是桌面端构建真实产生的目录）。已确认无任何**已跟踪**文件落入新规则。
- **`apps/web/src/styles.css` 中 2 个零引用设计 token**：`--primary-h`（浅 `#0b8563` / 深 `#2fdca6`）与 `--warn-bg`（浅 `#fffbeb` / 深 `rgba(217,119,6,.12)`）在 `src/` 中**无任何 `var()` 消费**（全仓 grep 仅命中定义本身，且不存在 `setProperty` 或动态 `var(--${...})` 拼接）。二者是早期「主色 hover 加深」「警告底色」方案的遗留——现在主色 hover 靠 `box-shadow` + `translateY` 表达，警告仅用前景色 `--warn`（`PreviewOverlay.vue:112`），配套的 `.msg.warn` / `.tag.warn` 规则从未存在。已按两套主题**成对删除**，删后调色板仍是浅/深 1:1 镜像（43 个颜色 token 两侧一致），`--warn` 与 `--primary` 保留（各有消费方）。
- **`apps/web/public/logo.svg` 死资源**：与 `apps/web/src/assets/logo.svg` **字节相同**（`md5 fd267cb2…`），但只有后者被 `App.vue` 以 `import logoUrl from './assets/logo.svg'` 引用。前者属 Vite 的 `public/` 直拷目录，会被**原样复制进每次构建产物**（`dist/logo.svg`，942 B），而 `dist/index.html` 与全部 chunk 都不引用它——早期 `public/` 方案的遗留副本。已 `git rm`（`public/` 目录随之消失）；品牌图仍由 `src/assets/logo.svg` 经 Vite 内联为 data URI 正常渲染（重建后 `dist/` 不再出现游离的 `logo.svg`，`pnpm build` 通过）。
- **`apps/web/src/styles.css` 中失效的 `.popover*` 规则块**：设置弹层已改为整页 `ServerPanel.vue`，`popover-wrap` / `.popover`（含 `h4` / `.field + .field` / `.actions`）与 `.popover-backdrop` 共 6 条规则，以及媒体查询里的 `.popover` 宽度覆盖，在 `src/` 中**已无任何模板引用**（全仓 grep `popover` 仅命中这些定义本身）。已删除该块；同段的 `@keyframes pop-in` **保留**——它仍被 `ObjectContextMenu.vue:105` 复用（该组件测试 9 项通过）。


### 新增（2026-09-17 路线图迭代 #1–#8、#10、#11）
- **`docs/api.md` 自动化校验（roadmap #1 / todolist #8）**：新增 `apps/server/internal/handler/api_doc_test.go`，从 `docs/api.md` 解析 `METHOD /api/...` 行并与 `routes.go` 注册表做**双向 diff**——文档多写一个端点或漏写一个端点都会让 `go test ./...` 变红。此前 `docs/api.md` 与路由的同步完全靠人工维护，70 个端点里任何一个改名/新增都可能静默漂移。已注入两侧漂移实测变红后还原，当前 70 ↔ 70 完全一致。
- **S3C3 加密格式：KDF 参数写入文件头（roadmap #2 / todolist #16）**：S3C2 信封只存 `magic + salt`，Argon2 参数硬编码（`t=1`），因此直接调参会让既有加密库**永久无法解密**。新格式 `S3C3` 为 `magic(4) + time(4,BE) + memory(4,BE) + threads(1) + salt(16) + ciphertext`，读取时按**文件头里的参数**派生密钥，从而可以在不影响旧库的前提下逐步加强。`argonTimeV3 = 2`（OWASP 建议 Argon2id time cost ≥ 2）。**双版本读取**：`parseEnvelope` 同时识别 S3C2（沿用 `legacyParams = {1, 64MiB, 4}`）与 S3C3；S3C3 头部参数为 0 视为损坏并报错（不静默降级为弱密钥）。新写入一律 S3C3。
- **SQLite 驱动 `secret_key` 列加密（roadmap #2 / ASSESSMENT M1）**：`SQLiteStore` 新增 `storeKey`，写入时 `encryptSecret` 以 S3C3 密文落盘、读取时 `decryptSecret` 解密；**历史明文行仍可读**（按魔数判别），写回时自动加密。库中已是密文但进程未配置 `S3C_STORE_KEY` 时显式报错，绝不把密文当明文返回。`config.MinStoreKeyLength = 16` 对 `S3C_STORE_KEY` 做最短长度校验（`ErrShortStoreKey`）。
- **安全审计日志（roadmap #3 / ASSESSMENT M3）**：新增 `apps/server/internal/handler/audit.go`，以稳定事件常量 + `h.audit(r, event, extra...)` 记录 `audit` / `ip` / `method` / `path`。覆盖：鉴权失败（401，含 `malformed` / `bad_token` 原因）、账号创建/更新/删除、桶策略设置/清除、对象删除、前缀删除、回收站清空、限速命中。
- **可信代理 XFF 解析（roadmap #3 / ASSESSMENT M5）**：新增配置 `S3C_TRUSTED_PROXIES`（默认空）。`clientIPWithProxies` **仅当直连对端命中白名单**时才采信 `X-Forwarded-For` 首段，否则一律回退 `RemoteAddr`。此前无条件信任 XFF，直连部署下任何客户端都能为每个请求伪造一个新 IP 绕过限速。
- **S3 上游调用指标（roadmap #5 / todolist #21）**：新增 `apps/server/internal/s3wrap/metrics.go`，用 smithy `Finalize` 中间件统一采集（覆盖全部 SDK 调用，无需逐方法埋点）：`s3c_s3_calls_total`、`s3c_s3_call_errors_total{code=...}`（API 错误码，非 API 错误归 `canceled` / `timeout` / `transport`，基数有界）、`s3c_s3_call_duration_seconds` 直方图（11 桶）、`s3c_s3_stream_bytes_total`。经 `/api/metrics` 输出。
- **ZIP 部分失败可见（roadmap #4 / ASSESSMENT S6）**：`zip.go` 不再丢弃 `service.WriteObjectsZip` 返回的 `failKeys`——部分失败落 Warn 日志（含失败 key 清单）并计入 `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total`，整体失败计入 `s3c_zip_failed_total`。此前失败信息只写进包内 `_下载失败清单.txt`，服务端完全无法观测批量下载失败率。
- **前端后端健康轮询与自动恢复（roadmap #8 / todolist #22）**：新增 `apps/web/src/composables/useHealthPoll.ts`，后端不可用后每 5s 探测 `/api/health`，一旦恢复即回调 `loadAccounts` 重新拉取数据并清除错误横幅；未出错时不轮询（不做无谓请求）。新增 `api.health()`。
- **grid 视图渲染上限（roadmap #7 / ASSESSMENT D8）**：`ObjectList.vue` 的 grid 分支改为渲染 `gridItems`（上限 300 条），超出时显示截断提示（新增 i18n 键 `objects.gridTruncated`）。此前 grid 的 `v-for` 对万级条目全量渲染 DOM（列表视图早已窗口化）。
- **错误文案源码级门禁（roadmap #6 / todolist #23）**：新增 `apps/server/internal/handler/error_echo_gate_test.go`，扫描生产 `.go` 文件（剔除注释与 `_test.go`），禁止 `writeErr(..., StatusBadRequest, "..." +` 这类把用户输入拼进响应文案的写法。

### 修复（2026-09-17 路线图迭代）
- **客户端错误消息不再回显用户输入（roadmap #6 / ASSESSMENT L2）**：`headers.go` 把 `ValidateUserMetadata` 的 `err.Error()` 直接回传客户端，而该错误串含用户提交的 metadata key（形如 `key %q length %d > %d`）。现改为固定文案 `invalid user metadata` + 服务端 `Debug` 日志（记录 bucket / key）。同类问题一并修复：`metadata.go` 的 `unsupported acl: <值>` → `unsupported acl`、`duplicate tag key`、`duplicate rule id`；`objects.go` 的 `unsupported storageClass`；`multipart.go` 的 `duplicate partNumber`。`TestSetHeadersInvalidUserMetadata400` 增加 `echo` 断言，确保响应体不含用户输入（含中文与超长 key/value）。
- **`useBucketSetting.reload()` 竞态守卫（roadmap #8 / ASSESSMENT S8）**：快速切桶或保存后刷新会让多个 reload 并发，旧请求的响应/失败会覆盖新请求的状态。现加 `reloadSeq` 序号，只有最后一次发起的请求才允许写 `loading` 或上报错误。
- **`entries` / `visibleEntries` 重复排序（roadmap #10 / ASSESSMENT D9）**：`useObjectBrowser.ts` 抽出 `compareEntries`（文件夹恒在前、文件按当前列与方向），`entries` 一次排序到位，`visibleEntries` 只做过滤并保持顺序。此前过滤态下会先排文件夹、再排一次文件。

### 测试与质量门禁（2026-09-17 路线图迭代）
- **覆盖率门禁去「注水」：前端纳入 `src/i18n/index.ts`（roadmap #11 / todolist #12）**：`vite.config.ts` 不再整体排除 `src/i18n/**`，改为只排除纯数据模块 `src/i18n/messages/**`（其完整性由 `i18n/coverage.test.ts` 的键门禁保证）。纳入后 `index.ts` 的 `readLocale` 回退/异常、`setLocale` 写入失败、`cycleLocale`、`locale()`、`i18nKeyCount` 缺省参数等分支补齐了**行为测试**（非 gap 测试），四指标仍为 100%。
- **后端删除不可达防御分支，`count==0` 检查归零**：`SQLiteStore.encryptSecret` 的 AES 加密错误分支（`deriveKey` 恒返回 32 字节合法密钥）与 `openSQLite` 中冗余的 `os.MkdirAll` 判断属确实不可达的防御代码，已删除而非写测试凑覆盖；`registerMiddlewares` 从内联闭包抽为命名函数以便直接测试锚点缺失时的错误上抛。`make test-cover` 的 `count==0` 扫描现无任何输出，8 个包语句覆盖率均 100%。
- **新增前端测试**：`useHealthPoll.test.ts`（轮询节奏 / 恢复回调 / start 幂等 / 卸载停止）、`useBucketSetting` 竞态守卫用例、`ObjectList` grid 截断用例、`App.vue` 恢复路径集成用例、`api.health()` 用例、i18n 分支用例。前端 63 文件 / 983 测试全绿。<sup>†</sup>
  > <sup>†</sup> 「63 文件 / 983 测试」是**该条目撰写当时的实测值**；后续用例增加后已变化（2026-09-21 实测：66 文件 / 1039 用例）。按本文件「不追溯篡改已发布区」的纪律保留原值并加注，不回写（review-2026-09-19.md §7.3 D10）。

### 文档（2026-09-17 路线图迭代）
- **路线图 v1.0.0 / v1.0.x / v1.1.0 收口（roadmap #1–#8、#10、#11）**：三个里程碑的开放条目全部完成并从 `docs/roadmap.md` 移除（长期项重编号为 #1–#3），`docs/todolist.md` 五个分类均归零，完成证据归档至 `docs/features.md` §M。`docs/api.md` 补充 `/api/metrics` 的 S3 上游指标与 ZIP 失败指标说明、ZIP 端点部分失败可观测说明。`README.md` / `.env.example` / `apps/server/.env.example` / `docs/deployment.md` 补充 `S3C_STORE_KEY`（≥16）与 `S3C_TRUSTED_PROXIES`；`docs/threat-model.md` 边界 C 表与已知风险同步更新。

### 新增
- **GitLab CI（`.gitlab-ci.yml`），与 GitHub Actions 同门禁**：把 `.github/workflows/` 的 `ci.yml`、`e2e.yml`、`e2e-playwright.yml` 逐 job 镜像为 `server` / `web` / `docker` / `desktop` / `desktop-build`（`when: manual`）与 `rustfs-e2e` / `playwright-e2e`，命令与阈值完全一致（gofmt、`go vet`、govulncheck v1.8.0、golangci-lint v2.13.2、`go test -race` + 覆盖率 100%、`pnpm lint/typecheck/test:coverage/build`、`docker build` + Trivy CRITICAL/HIGH、`cargo check --locked`、RustFS 真对端 E2E、Playwright chromium E2E）。触发规则对应 GitHub 的 push（main/develop）+ pull_request + 手动 + 定时。`release-desktop.yml` **有意不镜像**——它发布到 GitHub Release（tauri-action + `gh release upload`）且需 Windows/macOS runner 与 `GITHUB_TOKEN`，属发版设计而非 CI 一致性。`rustfs-e2e` 用 GitLab service 容器替代 compose 起对端，镜像/端口/凭据不变，并照搬 GitHub 的 `/health` 轮询等待（正式 runner 的 service healthcheck 只认镜像自带 HEALTHCHECK，rustfs 镜像没有）。新增 `.gitlab-ci-local/` 到 `.gitignore`（本地执行状态目录）。本地无 GitLab 实例即可用 `npx --yes gitlab-ci-local` 跑真实 job；两侧对照表与执行器差异（tracked-only 同步、`docker` job 需挂宿主 socket）记入 [`docs/development.md`](docs/development.md) §3。

### 修复
- **`server` job 的 16 条 golangci-lint 问题清零，GitHub 与 GitLab 两侧 `server` 门禁转绿**：此前 `server` job 在 `develop` 上连续失败，原因不是代码缺陷而是**门禁本身**——4 个 `unused` 指向已失效的注入器（见「移除」首条），12 个 `staticcheck` 里 8 个是 `s3wrap/*_fake_test.go` 的 `QF1002`（假 S3 的 `switch { case r.Method == http.MethodGet: }` 应为 `switch r.Method { case http.MethodGet: }`，纯风格），另有 `QF1006`（`internal/service/sync.go` 的 `for { if len(out) >= maxTotal { break } }` 提升为循环条件）、`QF1008`（`internal/store/store.go` 冗余的 `s.fileStore.load()` 选择器）、`S1040`（`internal/store/gaps_test.go` 把 `st` 断言成它已是的 `AccountStore` 类型，恒真的空断言）。逐一修掉后 `golangci-lint run` 输出 **0 issues**，`go vet` / `gofmt` 均干净。
- **`main_test.go` 的 `result.err` 字段从未被写入**：并发结果结构体带 `err`，但 goroutine 只填 `code`，`err` 永远为零值，属「看似在断言错误、实则从不校验」的假保障。已把通道简化为 `chan int`，直接断言 `runServer` 返回值。
- **`ha_gap_test.go` 用例循环重复构造请求与 recorder**：循环内先按 `"/nope-suffix"` 建 `req`/`rr`，紧接着在 `if/else` 里**无条件覆盖** `req`、随后又覆盖 `rr`，另有一个恒被丢弃的 `_ = i`。净效果虽等同于直接按 `a.ID` 构造，但读起来像在覆盖「非法后缀」分支。已改为按 `c.name` 一次性决定路径，去掉全部无效赋值（行为不变，`-race` 下仍全绿）。

- **CI 的 Trivy 镜像名错误导致 `docker` job 一直失败（GitHub 侧同样中招）**：`ci.yml` 的 Trivy 步骤引用 `aquasecurity/trivy:0.58.1`，但该路径**只存在于 `ghcr.io`**，Docker Hub 上的官方仓库是 [`aquasec/trivy`](https://github.com/aquasecurity/trivy/blob/v0.58.1/docs/getting-started/installation.md)。`docker run` 拉不到镜像会以 **exit 125** 退出，而漏洞门禁的失败码是 **exit 1**——两者混淆后看起来像「扫出了 CRITICAL/HIGH 漏洞」，实际是镜像名写错。GitHub 上 `docker` job 自引入起连续多次失败均为此因（check-run annotation 为 "Process completed with exit code 125"）。已把两套 CI 统一改为 `aquasec/trivy:0.58.1`（与 `ghcr.io/aquasecurity/trivy:0.58.1` 为同一镜像，config digest 一致）。
- **GitLab 侧 `-race` 在 alpine 镜像下必然失败**：`.go-base` 原用 `golang:1.26.6-alpine`，而 `go test -race` 需要 cgo（`CGO_ENABLED=1`）+ C 编译器，alpine 镜像两者皆无，会直接报 `-race requires cgo`。已改用 `golang:1.26.6-bookworm`（自带 gcc，也更贴近 GitHub 的 `ubuntu-latest`）并显式设 `CGO_ENABLED=1`。
- **GitLab 侧 `docker` job 的 `.trivyignore` 挂载路径不可达**：`docker run -v "$CI_PROJECT_DIR/.trivyignore:..."` 里的 `$CI_PROJECT_DIR` 是 **job 容器内**路径，宿主 daemon 看不到（正式 runner 上 `/builds/...` 为容器卷，宿主无此路径），Trivy 会以 `ignore file not found: /trivy/.trivyignore` 失败。已改为把 Trivy 二进制从固定版本镜像 `docker cp` 到 job 容器内执行——trivy 经挂载的 `docker.sock` 扫描宿主镜像，`--ignorefile` 用容器内可见的相对路径，GitLab 与本地 gitlab-ci-local 行为一致。
- **补完 `apps/` 迁移漏改的 `working-directory`（4 个 GitHub workflow 全部受影响）**：目录迁移把 `paths` / `go-version-file` / `cache-dependency-path` / artifact `path` 改成了 `apps/...`，但 `working-directory:` 与 tauri-action 的 `projectPath:` 仍是迁移前的裸名（`server` / `web` / `desktop`）。这些目录已不存在，相关步骤会以 "No such file or directory" 失败——即迁移后 GitHub CI 实际是红的。已统一改为 `apps/server` / `apps/web` / `apps/desktop`（含 `release-desktop.yml` 的 `projectPath: apps/desktop`）。
- **GitLab 两个 E2E job 的就绪轮询会「假绿」**：GitHub 的写法是 `curl … && exit 0`，但 GitLab 把整个 `script` 拼成**一个 shell 脚本**执行，`exit 0` 会结束**整个 job**——照搬过来会让 `rustfs-e2e` / `playwright-e2e` 在服务就绪后立即退出，**一条测试都没跑却报 PASS**。已改为「置标志位 + `break`」跳出循环，失败路径的 `exit 1` 保留（本就该中断 job）。GitHub 侧不受影响（每个 `run:` 是独立 step）。
- **GitLab 触发规则与 GitHub 逐 job 对齐**：三套 GitHub workflow 的 `on:` 并不相同——`ci.yml` 有 `push` 但**没有** `schedule`；两个 E2E 有 `schedule` 与 `pull_request.paths` 但**没有** `push`。此前 GitLab 用一条全局 `schedule` 规则，会让定时任务把 `server`/`web`/`docker`/`desktop` 全量重跑，且 `push` 时也会跑 E2E。现按 job 拆成 `.ci-trigger` / `.e2e-rustfs-trigger` / `.e2e-playwright-trigger`，E2E 侧用 `changes:` 复刻 GitHub 的 `paths` 过滤。实测四类事件选出的 job 集合与 GitHub 一致。
- **`desktop-build` 的 `when: manual` 被 rules 覆盖**：该 job 一旦引入 `rules:`，job 级 `when: manual` 即失效，会变成自动执行的重负载 Tauri 构建。已改为在 `.ci-manual-only-trigger` 的 rules 里写 `when: manual`，并限定 `CI_PIPELINE_SOURCE == "web"`——对应 GitHub 的 `if: github.event_name == 'workflow_dispatch'`（push / PR 流水线里该 job 根本不存在）。

### 测试与质量门禁
- **后端覆盖率门禁 90% → 100%，并按「删除死代码 / 行为断言」补齐**：`make test-cover` 与 CI 现在直接检查 `coverage.out` 中是否存在 `count==0` 的语句块，不再比较只有 1 位小数的 `total` 百分比——99.96% 会被四舍五入显示成 `100.0%`，用它做门禁会漏过回退。补齐过程中删除了 `jobsList` 中确实不可达的 `recs == nil` 兜底（`JobRegistry.List` 以 `make(..., 0, n)` 构造，空清单也非 nil，序列化本就是 `[]`）；`record()` 里「状态为空时按 done 反推」的兜底则**保留**——它可由公开的 `Emit` 触发（`JobProgress.Status` 是 omitempty，进度帧允许不带状态），而空状态一旦落盘，恢复流程会把它当成非终态、将已完成任务误标为 `interrupted`，属于真实防线而非死代码，因此改为用行为断言锁定。其余缺口全部改用行为断言覆盖：四个异步端点在在册任务达上限时经**真实路由**返回 503 且不注册新任务；`Create` 在容量耗尽时仍返回已终结任务且不占用名额；`restore` 跳过无 ID 的损坏记录；`List` 在 `Created` 相同时按 ID 升序稳定排序；无状态进度帧不得让清单/落盘快照出现空状态；`Emit` 的中间进度按 `jobProgressPersistEvery` 节流落盘（该间隔改为包级变量以便测试快速触发，与既有 `reapInterval` / `finishSendTimeout` 同一模式）；`PublicURL` 在端点不可用（未配置或退化为 scheme-only）时返回空串，而不是拼出损坏链接。8 个包（main / config / handler / model / openapi / s3wrap / service / store）语句覆盖率现均为 100%。

### 文档
> 本节按时间顺序记录本次文档结构重构的每一步。**各条目中的路径名反映该步骤当时的真实位置**——文件名小写化与迁至 `.github/` 发生在后，最终布局以本节最后一条为准。
- **根目录文档统一收敛至 `docs/`**：除 `README.md` 外，`CHANGELOG.md` / `ROADMAP.md` / `CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md` / `agents.md` 全部移入 `docs/`（GitHub 的 `docs/` 目录同样被识别为社区健康文件位置），根目录只保留 `README.md`（代理入口 `AGENTS.md` 随后回归根目录，见下一条）。同步修正 README、`docs/` 内部互链、`scripts/release-version.sh` 与 release workflow 中的路径引用，全仓相对链接已脚本校验无死链。
- **待办清单只保留未完成项**：`docs/todolist.md` 移除 6 项已完成条目（#9 / #13 / #14 / #19 / #20 / #24）以及仅描述已完成工作的归档叙述；⏳ 条目收敛为「剩余工作」，已落地部分不再重复记录（证据在 `FEATURES.md`）。编号保持稳定、不重排——`server/` 代码注释仍以 `todolist #N` 引用本清单，并据此修正 `ROADMAP.md` 中指向已移除编号的悬空引用。
- **文档命名约定 + 修复大小写冲突**：约定普通文档用小写 kebab-case，仅名字被外部约定固定的用大写（`README.md` / `CHANGELOG.md` / `CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md` / `LICENSE`），Agent 指令用 `AGENTS.md`；禁止任意两个路径仅大小写不同，规则写入 `docs/development.md` §4。据此把威胁模型 `docs/security.md` 重命名为 `docs/threat-model.md`——它由本次移动带入 `docs/` 后与 GitHub 认的漏洞披露策略 `docs/SECURITY.md` 仅差大小写，在 macOS / Windows 的大小写不敏感文件系统上检出会互相覆盖；README / FEATURES / development / deployment / SECURITY / agents 中的引用同步修正。；同时补充**目录命名**约定——目录一律小写，大写只由工具强制决定（`.github/`、`.github/ISSUE_TEMPLATE/` 已符合，若将来引入 REUSE 规范则为 `LICENSES/`），`docs/` 改成 `Docs/` 会使 GitHub 的社区健康文件查找与 Pages 发布源失效。
- **Agent 指令文件归位到仓库根目录**：删除已废弃的 `docs/agents.md`（其 TDD 约定与「文档同步门禁」对照表早已合并进 [`docs/development.md`](docs/development.md) §4，文件自身即声明为待删的跳转入口），并在**仓库根目录**新建 [`AGENTS.md`](AGENTS.md)。原因是 agent 工具的发现机制：候选名精确匹配 `AGENTS.md`（区分大小写），项目根由 `.git` 标记确定，且只注入根文件与子树文件——放在 `docs/` 下即使改成大写也不会被加载。新文件只保留仓库级硬约束、门禁命令与文档入口指针（详细规范仍在 `docs/development.md`，避免此前的两处维护漂移），并新增 `docs/development.md` §4.1 记录加载机制（候选名 / `.git` 根判定 / 子树惰性加载 / 字节预算截断）、`AGENTS.local.md` 加入 `.gitignore`。同步恢复 `docs/CONTRIBUTING.md`、`docs/ROADMAP.md` 对代理入口的引用，并清理 `scripts/release-version.sh` 里针对旧文件、已无匹配目标的两条版本号 sed。
- **5 个文档名小写化，命名规范收口**：`docs/API.md` / `ASSESSMENT.md` / `ERRORS.md` / `FEATURES.md` / `ROADMAP.md` → `api.md` / `assessment.md` / `errors.md` / `features.md` / `roadmap.md`。它们此前是本仓库自己的「台账类大写」习惯，并无任何工具按文件名匹配，属白名单之外的豁免项；本轮按唯一正确处理方向（**小写化**，而非继续大写）消除豁免，命名规范自此**不存在非白名单例外**——大写仅限 `README.md`、`AGENTS.md`、`CHANGELOG.md`、`CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`、`LICENSE`。改名一律用两步 `git mv`（纯大小写改名在 macOS / Windows 上单步会静默失败），并同步更新引用：README / AGENTS.md / `docs/` 内部互链与正文路径，以及 `scripts/release-version.sh` 中 `docs/api.md` 的 sed 目标（漏改会让 `set -e` 直接中断发版脚本）。`docs/CHANGELOG.md` 的历史条目按惯例不改写，旧名↔新名对照见本条。
- **社区健康文件迁至 `.github/`（规避「静默顶掉」）**：`CONTRIBUTING.md` / `SECURITY.md` / `CODE_OF_CONDUCT.md` 由 `docs/` 移至 `.github/`。依据 [GitHub 官方规则](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file)：这类文件在 `.github/`、根目录、`docs/` 三处都被识别，但**查找优先级为 `.github/` > 根目录 > `docs/`**，且命中即独占、多份并存不报错——放在最低优先级的 `docs/` 会被将来的根目录副本静默忽略。`CHANGELOG.md` 不在 GitHub 的社区健康文件名单内（位置纯属 Keep a Changelog 约定），故留在 `docs/`。因相对链接方向反转，同步更新 README / AGENTS.md / `docs/development.md` / `docs/threat-model.md` 的引用，以及三个文件自身指向 `docs/` 的链接。
- **`CHANGELOG.md` 迁至仓库根目录（类别归位）**：它此前与 `api.md` / `development.md` 等**叙述文档**混放在 `docs/`，但类别上属于**约定文件**——Keep a Changelog 钉死了文件名（`CHANGELOG.md`）并把它与 README / CONTRIBUTING 并列为「典型大写文件」，主流 changelog 工具（release-please / semantic-release / standard-version / git-cliff）默认路径也都是 `./CHANGELOG.md`。GitHub 对 CHANGELOG **没有任何查找行为**（它不在社区健康文件名单内），故位置纯由约定决定；未选 `.github/` 是因为那里放的是 GitHub 配置与社区健康文件，放进去没有任何工具会读。共迁移 11 处引用：`docs/` 内 9 处同目录链接升为 `../CHANGELOG.md`、`.github/SECURITY.md` 1 处、CI Release notes 1 处 URL（反而更短）；`AGENTS.md` 与 `docs/development.md` §4 的根目录存放规则同步改写，`AGENTS.md` 入口表新增「每个 PR 都要补发版记录」一行。根目录现为 `README.md` / `AGENTS.md` / `CHANGELOG.md` 三个约定文件。
- **补齐 `.github/` 下文件的发现入口**：GitHub 会自动在「新建 issue / PR 页面」「`/contribute` 页面」「仓库概览的 Contributing 与 Code of conduct 标签」「Security 标签页的 Security policy」等处暴露这三个社区健康文件，但仓库还缺两个**显式**入口，本轮补上：① `.github/PULL_REQUEST_TEMPLATE.md` 顶部新增指向 `CONTRIBUTING.md` 与 `docs/development.md` 的提示行——PR 模板会被**自动填充**到每个 PR 的描述框，是触达率最高的位置；② 新增 `.github/ISSUE_TEMPLATE/config.yml`，用 `contact_links` 在 issue 模板选择页底部给出「贡献指南」与「报告安全漏洞（请勿公开）」两个入口，并设 `blank_issues_enabled: false` 隐藏空白 issue（Write 及以上权限的维护者仍可见 Maintainers only 入口）。

### 仓库卫生
- **补齐 `.gitignore` 缺口**：审计工作区（`git ls-files -i -c` 无命中、未跟踪且未忽略文件仅 `AGENTS.md`）后补 4 类未覆盖项——① `coverage.out` / `coverage.txt`：`make test-cover` 与 CI 的 Go 覆盖率产物，**属真实缺口**（通用 `coverage` 规则只匹配同名目录，匹配不到这两个文件）；② OS 垃圾 `.DS_Store` / `Thumbs.db` / `Desktop.ini`（本项目同时发布 macOS `.dmg` 与 Windows `.exe`）；③ 编辑器临时文件 `.idea/` / `*.iml` / `*.swp` / `*.swo` / `*~`；④ Agent 工具本地目录 `.claude/` / `.cursor/`。`.vscode/` 采用 `.vscode/*` 加白名单 `extensions.json` / `settings.json`，保留将来共享团队配置的位置（已用退出码与真实建文件端到端验证：仅这两个文件出现在 `git status`）。
- **本轮按最小改动未处理的已知项**（留待专项）：5 条被通用规则覆盖的冗余条目（`web/node_modules/`、`desktop/node_modules/`、`web/dist/`、`server/data/`、`web/playwright/.cache/`）、过宽规则 `data/`（裸名会静默忽略任意层级的 `data/` 目录，如将来的 `web/src/data/`）与 Next.js 遗留的 `out`，以及 `.gitignore` 中 143 行与本技术栈无关的 Node 通用模板。
- **已核实无需改动**：账号存储文件（`accounts.json` / `accounts.db` / `accounts.json.enc`，`S3C_DATA_DIR` 默认 `./data`）均在忽略范围内；nginx 日志写容器内 `/var/log/nginx/` 或 stderr；compose 使用命名卷；Trivy 仅只读挂载 `.trivyignore`、不落报告文件。

### 开发规范
- **硬约束第 5 条细化为「不留尾巴（死代码零容忍）」**：明确列举四类禁止项——死代码（定义后无人引用的函数/类型/常量/变量/字段、不可达分支）、未定义即使用的标识符、定义了却未使用的变量（含只写不读、赋值后即被覆盖、恒真/恒假的空断言）；**枚举值除外**（枚举/常量表成员即使暂无引用也属契约，保留）。同时写入两条此前缺失的认知：① **警惕「被覆盖率掩盖的死代码」**——测试文件不参与 instrumentation，`go test -cover` 只看生产代码，测试辅助里的死代码能让 100% 门禁全绿（本仓库即因此让一套失效的错误注入器长期存活），覆盖率达标与无死代码必须分别验证；② 机械检查必须开启并保持**零告警**（Go: `golangci-lint` 的 `unused`/`staticcheck` + `go vet`；TS/JS: `pnpm lint` + `vue-tsc`），不接受「历史遗留」作为例外。同步写入 [`AGENTS.md`](AGENTS.md) 硬约束与 [`docs/development.md`](docs/development.md) §5 验收清单、§6 Red Flags（含「不要用 `_ = x` / `//nolint` / 导出为 `_test` 辅助来消音」）。
- **新增「改完代码必须同步文档」强制规则**：修复 bug 或新增功能完成后**必须**更新相关文档，文档未同步视为改动未完成、不得提交/合并。新增「改动类型 → 必须更新的文档」对照表（README / `docs/API.md` + `openapi_register_*.go` / `CHANGELOG.md` / `docs/FEATURES.md` / `docs/todolist.md` / `ROADMAP.md` / `docs/architecture.md` / `docs/deployment.md` / `docs/security.md` / `docs/ERRORS.md` / `CONTRIBUTING.md` 等），落地到 `agents.md`（后已删除）与 [`docs/development.md`](docs/development.md) §4「文档同步门禁」，并写入验收清单与 Red Flags；[`CONTRIBUTING.md`](.github/CONTRIBUTING.md) 开发规范速览同步补充。

### P1 稳定版门槛修复（2026-09-16 评估）
- **异步任务清单持久化 + 重启恢复（v1.0.0 最后一项门槛）**：`JobRegistry` 此前纯内存，进程重启即丢失全部任务记录；而「复制→删源」的移动语义是两阶段操作，中途崩溃会留下「已复制但源未删除」的中间态且无任何痕迹可对账。现在任务清单经 `service/job_persist.go` 的 `JobPersister` 落盘（临时文件 → rename → 0600；`Create`/`Finish` 必写、中间进度 2s 节流），启动时把非终态任务标记为 `interrupted` 并回写；新增 `GET /api/migrate/jobs` 返回任务清单；前端 `MigratePanel` 新增「未完成任务」区块，明确提示移动任务「可能已复制但源未删除」并支持逐条忽略。`interrupted` 采用独立 7 天保留期（`JobInterruptedTTL`），修复了「恢复后首次 reap 就被 30 分钟 TTL 清除、恢复功能形同虚设」的缺陷（附回归测试）。落盘未复用 `store/atomic.go`——`service→store` 会造成分层倒置，两处技术策略一致。`NewJobRegistry()` 保持纯内存语义，既有调用方与测试不受影响。
- **Go 工具链 1.26.5 → 1.26.6，并新增 `govulncheck` 门禁**：`server/go.mod`、`server/Dockerfile` 同步至 1.26.6（CI 经 `go-version-file` 自动跟随）。修复 go1.26.5 中 6 个**可达** stdlib 漏洞（net/url 二次方复杂度、crypto/tls 握手 DoS、net/http HTTP/2 探测、encoding/xml 递归、encoding/asn1 递归、net/http Punycode），`govulncheck ./...` 实测由「6 个可达」降为 **0**。CI 在 `go vet` 后新增固定版本 `golang.org/x/vuln/cmd/govulncheck@v1.8.0` 门禁——Trivy 只扫容器 OS/库，拦不住标准库 CVE，此前是盲区。
- **修正 2 处幽灵 action SHA**：`e2e-playwright.yml` 的 `pnpm/action-setup` 与 `actions/upload-artifact` 指向的 commit 在 GitHub 上不存在（API 404），会导致工作流失败，且若上游伪造同名 tag 存在执行恶意 action 的供应链风险。已改为与其它 workflow 一致的正确 SHA；全仓 10 个 action SHA 经 GitHub API 逐一核验均有效。
- **补齐 3 个缺失 i18n 键**：`objects.toastCopyFailed`、`batchEdit.tagsNeedKey`、`common.working` 此前被引用但未定义，用户界面会直接显示原始 key。已补齐 zh-CN / en-US 文案；新增 `src/i18n/coverage.test.ts` 静态扫描「被引用但未定义」的字面量键（用 `import.meta.glob` 读源码，不引入 `node:*` 依赖），作为该类缺陷的常驻门禁。
- **OpenAPI 契约收尾**：`POST /api/accounts/{id}/delete-marker/restore` 请求体由错误的 `deleteMarkerId` 修正为 handler 实际解析的 `versionId`（此前客户端按文档调用会因缺 `versionId` 直接 400）。契约测试同步扩展至 `delete-marker/restore` / `version`(DELETE) / `version/restore`，断言「含 `versionId` 且不含 `deleteMarkerId`」，并已验证该测试在回退修复时确实失败。

### P2 加固修复（2026-09-16 评估）
- **预签名失败不再被静默吞掉**：`presign`（get/put/post）与 `multipart/part` 三处原为 `u, _ := client.PresignXxx(...)`，失败时返回 `200 {"url":""}`——调用方拿到空 URL 却看到成功状态。预签名并非不可能失败：AWS SDK 在取凭证、输入序列化等阶段都会报错（实测空 key、context 取消均可触发）。现统一经 `writePresignResult` 处理，失败返回 500 `failed to create presigned url`，并新增源码级门禁 `TestPresignErrorsNotSwallowed` 防止该写法复发（已回退验证门禁会失败）。
- **流式传输中断不再无痕**：`copyStream` 原以 `_, _ = io.Copy(...)` 丢弃返回值，大文件下载被上游读失败或写超时打断时，日志与指标里都没有任何记录。现返回 `(int64, error)`，由 `recordStreamOutcome` 分流：真实中断记 Warn 并累加新指标 `s3c_stream_interrupted_total`；客户端主动断开（用户取消/关页面）仅记 Debug、不计入指标，避免污染告警。
- **异步任务加上限**：`JobRegistry` 新增在册任务上限（256 个未终结任务），超限时异步端点返回 503 `too many running jobs; retry later`，避免短时间内大量请求持续堆积 goroutine、SSE 订阅与落盘条目。上限只统计未终结任务，因此恢复出的 `interrupted` 历史任务不会永久占满名额。`Create` 保持原签名以不波及 70+ 处调用点，新增 `TryCreate` 供需要感知容量的路径使用。
- **TLS 站点补齐 HSTS 与 Permissions-Policy**：`deploy/nginx/conf.d/s3clinet-tls.example.conf` 增加 `Strict-Transport-Security`（180 天，暂不带 preload）与 `Permissions-Policy`（关闭定位/麦克风/摄像头/支付/USB/interest-cohort）。这两项只能由 TLS 终止层表达，后端已有的 CSP 等头无法替代。

### P2 加固修复（续）
- **清理后端死代码**：删除 `ctxReader`（与 `ctxCancelReader` 职责重复且零生产引用）、`batchItemError`（生产零引用；该格式串实际在 `service/batch.go` 内联）、`Client.S3()`（导出但零生产调用，E2E 清理逻辑改用包装层已有的 `DeleteObjectVersion`/`DeleteObject`）。核验后**保留** `isNoSuchBucketSetting`——它有 5 处生产调用，todolist 将其列为死代码属描述有误。当时覆盖率为 99.6%（该轮之后门禁已提升至 100%，见上方「测试与质量门禁」）。
- **compose 默认结构化日志**：`docker-compose.yml` 与 `docker-compose.prod.yml` 注入 `S3C_LOG_JSON: "${S3C_LOG_JSON:-1}"`，容器日志可直接被采集器按字段检索；设 `S3C_LOG_JSON=0` 可退回纯文本。已用 `docker compose config` 验证默认值与覆盖行为。
- **端点归一化合并为唯一实现，修掉 URL 损坏**：`s3wrap` 与 `service` 各有一份归一化且行为不一致——`s3wrap` 只做大小写敏感的 `http://`/`https://` 前缀判断，把用户手填的 `"HTTP://MinIO:9000"` 当成裸主机，拼成损坏的 `"http://HTTP://MinIO:9000"`。现 `s3wrap.NormalizeEndpoint` 为唯一实现（去首尾空白、scheme 大小写不敏感、host 小写、**保留路径前缀**），`service.SameEndpoint` 与 `PublicURL` 均改为复用；`PublicURL` 同时修掉 `" http://a.com/ /b/k"` 这类带空格的链接。
- **`ValidateEndpoint` 不再把合法端点误判为非法**：该函数此前 `strings.TrimSpace` 只用于判空、`url.Parse` 仍用未 trim 的原串，用户填 `" http://127.0.0.1:9000 "` 会收到 `invalid endpoint URL`（且消息里是难懂的 `first path segment in URL cannot contain colon`）。现改为归一化后解析；**保持 fail-closed**——非空输入归一化后退化（`"http://"`、`"/"`、`"https:///"`）仍报错，不因归一化静默放行。E2E 实证：畸形端点建账号后 `preview-buckets` 由 400 变为通过校验进入网络调用，日志中 BaseEndpoint 为干净的 `http://127.0.0.1:9000`。
- **删除 handler 侧 4 个死代码包装**：`derefString` / `derefInt64` / `timeOrZero` / `boolOrFalse` 生产零引用，仅被 `TestAccDerefHelpers` 引用（覆盖率注水样本），连同该测试一并删除。`s3wrap` 内的同名 helper 保留——各有 29/1/5/2/4 处生产调用。

### P0 发布阻塞修复（2026-09-16 评估）
- **OpenAPI 契约与真实 handler 字段级对齐**：`/api/migrate`、`/api/migrate/async` 请求体字段改为 handler 实际解析的 `sourceAccountId` / `sourceBucket` / `sourceKeys` / `targetAccountId` / `targetBucket` / `targetPrefix`（删除此前虚构的 `deleteSource` / `storageClass`）；presign `method` 枚举由 `GET/PUT/DELETE/HEAD` 改为实际支持的 `get/put/post`；delete 请求体移除 handler 不解析的 `versionId`；multipart/part 补上 handler 支持的 `expiresIn`。新增 `TestOpenAPI_ContractRequestBodyMatchesHandlers`，对「注册表 schema ↔ handler DTO」做字段级一致性断言（含 `$ref` 解引用 / enum 检查），补上 routes↔spec 检查覆盖不到的漂移面。
- **前端 Token 不再明文落 `localStorage`**：`s3c.servers` 只存 `{id,name,base}`；token 单副本按 `s3c.token.<serverId>` 存储（默认 sessionStorage，仅显式开启「跨会话保留」时写 localStorage）。旧版本内嵌 token 首次读取时一次性迁移并回填活动服务器 profile；删除服务器同步清理 per-server token。修复「token 仅 sessionStorage」策略被多服务器列表旁路的问题。
- **`loadAll()` 至少在 `nextToken` 为空时也发起一次请求**：改为 do-while，空 token 时以 reset 语义加载第一页（替换旧列表、避免重复项），不再出现「一次请求都没发却提示已加载全部」；某页失败即停止续页且不再发成功提示。
- **迁移 SSE 终态检测兜底，消除 Promise 永久悬挂**：流以 EOF 结束且未收到终态时，轮询 `migrateJobStatus`（500ms 间隔、30s 上限）直到 done/cancelled 并合成终态 `status`；连续 3 次回读失败快速 `onError`。`ctxDeleteFolder` / `DestDialog` / `MigratePanel` 三个调用方由此保证拿到终态或错误，`opsBusy` 不再永不复位、按钮不再永久禁用。

### 安全与可靠性
- **运行时基镜像换成 Alpine 3.20**：`debian:bookworm-slim`（~80MB、90+ 包、glibc + openssl 3.0.x）→ `alpine:3.20`（~5MB、19 个包、musl + openssl 3.3.7）。体积从 ~80MB 降到 41.5MB；Trivy 报告的 CRITICAL/HIGH 漏洞面收窄 ~80%。
- **`/api/metrics` 默认关闭**：新增 `S3C_EXPOSE_METRICS=1` 显式开启；默认返回 404，假装端点不存在，避免公网被 scrape 运行指标。
- **S3C_TOKEN 短口令硬失败**：长度 < 16 字符（含多 token 中最短者）直接拒绝启动，不再仅告警。
- **S3 出站禁 HTTP(S)_PROXY**：transport 显式 `Proxy=nil`，防止环境变量代理绕过 `dialContextSSRF` 校验。
- **桶名校验收紧**：拒绝首尾为 `.`/`-`、连续 `..`/`.-`/`-.`。
- **user metadata 边界 400**：键值长度 / 字节总长超 S3 限制直接 400，不再落到 S3 端以 500 返回。
- **delete-objects 单次 ≤1000**：与 S3 `DeleteObjects` 上限对齐；超限返回 400 并提示走 `delete-prefix`。
- **migrate SSE 写超时**：复用 `streamIdleTimeout`（每次成功写出后刷新），慢客户端不会让连接无限挂着。
- **store 失败回滚**：`EncryptedStore.Update` 持久化失败回滚内存；`SQLiteStore.Update` 显式传播 `Exec` 错误；`JSON Store.Update` 补测。
- **错误映射去字符串匹配**：新增 sentinel `ErrObjectTooLarge` / `ErrSourceDeleteFailed` 并用 `errors.Is` 判断；`PutObject`/`CopyObject` 把 S3 `EntityTooLarge` 归一为 sentinel，不再依赖 SDK 文案。
- **X-Request-ID 回显加固**：仅回显长度 ≤128 且全为可见 ASCII 的值，超长/含控制字符改由服务端生成 UUID，阻挡日志与响应头注入。
- **Bearer scheme 大小写不敏感**：按 RFC 7235 接受 `bearer`/`BEARER` 等，凭证仍常量时间比较。
- **`.env` 解析解耦进程 CWD**：`S3C_ENV_FILE`（设置后为唯一来源）→ CWD `.env` → 可执行文件同目录 `.env`；真实环境变量始终优先。
- **nginx 内存上限**：两个 compose 文件为 nginx 显式设置 `deploy.resources.limits.memory: 128M`。

### 工程化（v1.0.0-rc1 评估 P1/P2 路线落地）
- **OpenAPI 自动生成**：`/api/openapi.json` 端点（无依赖显式 builder），70 个 `/api/*` 端点按域（accounts/buckets/bucket-settings/objects/object-meta/multipart/versions/trash/migrate/system）集中登记；与 routes.go 一一对应；端点不进鉴权层（契约非业务）。
- **Playwright 浏览器 E2E 基建**：chromium + vite preview + `page.route()` 拦截 `/api/*` 回放 fixture；`smoke.spec.ts` 3 例 + `account-flow.spec.ts` 2 例；新增 `.github/workflows/e2e-playwright.yml`（PR/dispatch + 每周三 02:00 UTC 冒烟）；`pnpm e2e:install` 安装 chromium 与系统依赖；vitest exclude `e2e/**` 避免冲突。
- **增量同步**：`POST /api/migrate/sync`（withStreamLimit 保护），按 `etag`（默认）/ `size_mtime` / `always` 三种 mode 比对源/目标，仅复制差异对象，跳过完全一致的对象；internal/service.SyncKeys 复用 MigrateKeys 完成实际复制；`s3wrap.Client.Endpoint()` 访问器供 SameEndpoint 比对。
- **桶策略可视化编辑器**：web/src/bucketPolicy.ts + BucketPolicyVisualEditor.vue：Statement（Effect/Principal/Actions/Resources/Sid）表单式编辑，4 个常用模板（公共读 / 公共读写 / 拒绝 List / 清空），实时 JSON 预览 + validateDoc 校验；不支持的结构（NotPrincipal/嵌套）自动回退原始 JSON 模式，避免覆盖用户已写的高级策略。
- **对象元数据批量编辑**：web/src/batchMetadata.ts 4 路有界并发 worker-pool（BATCH_META_CONCURRENCY=4 与 copy/migrate 同上限）；BatchMetadataDialog：ACL 下拉 / 标签替换或清空 / 存储类型输入，3 个 fieldset 独立勾选；运行中进度 + 失败明细可滚动展示；ObjectToolbar 新增「批量改元数据」按钮。

### 工程化
- **CI 工具链对齐**：所有 workflow 的 pnpm 统一为 11；`desktop-build` 拆为「先装 tauri-cli（CI 缓存）→ 再 tauri build」，去掉 `|| true` 静默失败。
- **OpenAPI 生成器死代码清理**：删除零引用的 `Prop()` 与从未使用的 `Request.Example` / `Response.Headers` / `Response.Content` / `Schema.Additional`；`internal/openapi` 覆盖率仍 100%，`/api/openapi.json` 输出字节级不变（对外契约不动）。
- **文档收敛**：三份已迁移的历史评估快照（`full-assessment.md` / `code-review.md` / `code-review-v1.0.0-rc1.md`）移除；散落的待处理事项统一汇总到 `docs/todolist.md`（单一待办来源），`FEATURES.md` / `README.md` 同步更新链接。
- **存储驱动再收敛**：`EncryptedStore` / `encryptedCodec` 并入统一 `Store` + 单一 `storeCodec`（`strict` 区分 json permissive / encrypted 严格语义），`encrypted.go` 删除，`NewEncrypted` 返回 `*Store`；磁盘格式、错误文案、盐策略与覆盖率 100% 不变，解决历史「双驱动功能重叠」遗留项。
- **账号响应契约收敛**：账号响应从 `secretKey: "******"` 占位改为 `AccountView`（新增 `secretSet: boolean`，**不再回传 `secretKey`**）；请求仍以 `secretKey` 提交（编辑留空 = 保持不变）。后端（`model.AccountView` / handler 视图转换）、前端（`types.ts` `Account.secretSet`）、OpenAPI 与 `docs/API.md` 同步更新；`internal/model` 覆盖率 100%。
- **OpenAPI components 接线 `$ref`**：新增 `openapi.Ref()` 与 `Param.Ref` / `Response.Ref` 渲染分支；共享 `schemas`（Error/Account/Bucket/ObjectItem/ListObjectsResp）、`parameters`（AccountID/Bucket/Prefix/MaxKeys/ContinuationToken）、`responses`（BadRequest/NotFound）经 `refSchema` / `refParam` / `refResp` 全部接线为 `$ref`（109 处引用、12 个唯一目标），消灭「components 0 引用」死代码状态；契约测试改为先解析 `$ref` 再校验路径参数 / 响应描述，新增「components 无死片段」断言（`Unauthorized`/`TooManyRequests`/`InternalError` 为全局错误词汇除外）；`internal/openapi` 覆盖率保持 100%。`/api/openapi.json` 输出形状从内联改为 `$ref`（对外契约结构性更新，运行时 API 行为不变，前端不消费该文档）。
- **待办清理**：`docs/todolist.md` 移除「三、历史评审遗留」整节（7 项均已复核关闭，详见 `FEATURES.md` C / D / E），待办清单仅保留仍未决事项。
- **待办归档**：`docs/todolist.md` 全部 4 项（存储驱动再收敛 / `secretSet` 账号契约 / OpenAPI `$ref` 接线 / `preview-buckets` 预览桶契约）均已完成，从待办清单移除并归档至 `FEATURES.md`「一、产品功能」与「二、已完成修复与优化」（A / B 段补记存储驱动收敛完成态），待办清单当前为空。
- **综合评估 + 文档结构重构**：4 路并行深度审查（后端 Go / 前端 Vue-TS / 安全威胁与依赖 / SRE 可靠性）产出 `docs/ASSESSMENT.md`（五维度评分：代码质量 82 / 漏洞 72 / 死代码 70 / 降级 74 / 自我迭代 90），发现 OpenAPI 契约失真、Go 1.26.6 修复线、幽灵 SHA、异步任务无恢复等 20+ 项并回写 `todolist.md`；文档按开源项目常见结构重构——新增 `SECURITY.md` / `CODE_OF_CONDUCT.md` / `.github/ISSUE_TEMPLATE` / `PULL_REQUEST_TEMPLATE.md` / `docs/architecture.md` / `docs/development.md`（合并原 `agents.md`）/ `docs/deployment.md` / `docs/security.md` / `docs/decisions/`（ADR-001~004），README / FEATURES 链接同步更新。
- **Trivy 镜像扫描**：CI 在 Docker 构建后跑 `aquasecurity/trivy:0.58.1`，CRITICAL/HIGH 漏洞硬失败；新增 `.trivyignore` 与 `--ignorefile` 集中收纳可忽略的 CVE。
- **构建层升级 Node 24**：Dockerfile 与所有 workflow 的 `node-version` 升到 24；pnpm 锁回 9.15.0（与 `package.json` 的 `packageManager` 声明一致，兼容现有 `pnpm-lock.yaml`）。
- **配置校验函数 `Config.Validate()`**：`MinTokenLength=16`、非回环无 token 拒绝启动。
- **`go mod tidy` 显式化**：Makefile 移除每次 `server` 启动的隐式 tidy，新增 `make tidy` 显式入口。
- **SSE 卸载中断**：`useObjectActions` `showDetail` 加 `detailSeq` 序号守卫；`MigratePanel` 组件级 `activeUnsub`，`onBeforeUnmount` 断开进行中的 SSE。
- **列表请求 AbortController**：`useObjectBrowser.load` 每次新请求取消旧的；`onBeforeUnmount` 取消进行中的 fetch。
- **双文件存储驱动去重**：`json` / `encrypted` 驱动共享 unexported `fileStore` + 注入式 `fileCodec`，锁 / CRUD / 持久化回滚 / 快照排序 / JSON 中间表示单点实现（净减约 331 行），文件名、磁盘格式、错误文案与加密语义不变。
- **OpenAPI 契约测试**：`routes.go` ↔ 规范双向一致（漏登记与陈旧条目都红灯）、operation 完整性 / 路径参数 / `$ref` 可解析 / `MarshalJSON` 确定性；`internal/openapi` 包与 handler 内全部 OpenAPI 函数达 100% statement 覆盖。
- **全仓 gofmt 对齐 Go 1.26**：修正结构体字段对齐与文档注释空行规则，CI `gofmt -l .` 门禁恢复干净。

### 前端
- **Token 默认 sessionStorage**：`s3c_token_persistent='1'` 显式开启才写 localStorage；旧版本 localStorage 遗留值首次读取时自动迁移到 sessionStorage 并清空 localStorage 副本。
- **MigratePanel 虚拟滚动**：`listAll` 上限 200×1000 不再 `<tr v-for>` 全量渲染，按 ObjectList 同款窗口化方案。
- **showDetail seq 守卫**：快速切对象时过期响应丢弃。
- **lint 收紧**：`@typescript-eslint/no-explicit-any` 由 off 改 warn；新增 `isTauri` 类型明确化。
- **集中式 `requireAccId()`**：`useObjectActions` 13 处 `ctx.account.value!.id` 收敛为单一异常入口。
- **SSE 卸载中断补齐**：`DestDialog` 迁移进度流与 `useObjectActions` 前缀删除流在 `onBeforeUnmount` 主动 abort，不再后台悬挂推送。
- **VersionsDialog 分页**：按 `keyMarker`/`versionIdMarker` 翻页（≤20 页），仍被截断时显示提示，不再静默丢弃第 1000 条之后的版本。
- **消除非空断言**：剩余 3 处 `ctx.account.value!` 与 `ObjectsPanel` 的 `{} as KeyBindings` 改为守卫 / 可选类型（`KeyBindings` 字段可选 + `?.` 调用）。
- **blob 下载延迟回收**：`revokeObjectURL` 由同步改为延迟 60s，避免中断浏览器尚未开始的下载。
- **lint 再收紧**：`@typescript-eslint/no-explicit-any` 由 warn 改 error（当前 0 违规）。


## [v1.0.0-rc1] - 2026-09-02

相对 `v1.0.0-rc0`：桌面安装包由 CI 在打 tag 时交叉构建并挂到 GitHub Release。

### 工程化
- **桌面发版 CI**：`release-desktop.yml` 在 `v*` tag（或手动指定 tag）时交叉构建 NSIS `.exe` / `.deb` / `.dmg`，并上传到对应 GitHub Release。

## [v1.0.0-rc0] - 2026-09-02

首次候选发布（`develop` 全栈快照）。相对时间戳构建的主要增量：CI（gofmt / golangci-lint v2 / desktop cargo / race）打通；不做服务降级与无历史加密/账号文件格式兼容。

### 安全与可靠性
- **不做服务降级 / 无历史格式兼容**：`/api/health` store 失败返回 `503` + `status:error`；`encrypted` 仅 `S3C2`；JSON 账号文件写入 `0600`（加载时不再改旧权限）；移除 `sameEndpoint` 测试 shim 与前端「旧后端无 version」兼容注释。

### 安全与可靠性（ANALYSIS 整改·续2）
- **copyPrefix 异步**：`POST .../copy-prefix/async` 复用 job+SSE；前端文件夹复制改走异步。
- **copy-objects / delete-prefix 异步**：`POST .../copy-objects/async`、`POST .../delete-prefix/async`；多选移动与文件夹删除走 SSE。
- **多 token**：`S3C_TOKEN` 逗号分隔支持轮换。
- **防腐层**：`s3wrap.UserMessage`/`HTTPStatus`/`IsNotFound`/`ObjectStream`；handler 生产代码不再直接依赖 smithy/types；`service.CopyKeys`/`MigrateKeys`/`WriteObjectsZip`/`JobRegistry`；i18n≈570 键。
- **service 收口**：迁移引擎、ZIP、异步 Job 注册表迁出 handler；短写校验；`Entry`/`SortKey` 提到 `types.ts`。
- **可观测**：`GET /api/metrics`（Prometheus 文本）、`X-Request-ID`、可选 `S3C_LOG_JSON=1`。
- **TLS**：`docker-compose.tls.yml` 叠加 prod；a11y（aria-current/aria-sort/焦点恢复/网格键盘）；对比度与死资源清理；ANALYSIS §七整改对照。
- **长尾**：ZIP 4-worker 并行拉取；`docs/ERRORS.md`；coverage_gaps 拆分；`noUnusedLocals`；ModalDialog 组件测；i18n≈640；ServerPanel 多 token 提示；desktop tauri build 仅手动 workflow_dispatch。

### 安全与可靠性（ANALYSIS 整改·续）
- **>5GB 迁移 multipart**：跨端点流式分段；同端点 CopyObject EntityTooLarge 自动回退 multipart（`internal/service`）。
- **copyPrefix**：4 worker 并发 + `withStreamLimit`；delete-prefix 同样纳入流限。
- **加密 KDF**：`encrypted` 存储改为 magic+盐派生（现为 Argon2id / `S3C2`）。
- **API 限速**：每 IP 令牌桶约 120/min（health/静态除外）。
- **错误映射**：`writeInternalErr` 统一走 `s3HTTPStatus`（NoSuchBucket→404 等）。
- **生产**：TLS nginx 示例、README token 必填说明、`service` 包抽离流式复制。

### 前端（续）
- ObjectList 列表虚拟滚动；i18n≈308 键；ESLint 基线；token 可选 sessionStorage（`s3c_token_ephemeral=1`）。

### 安全与可靠性（ANALYSIS 整改）
- **SSRF**：S3 HTTP 客户端禁重定向 + Dial/创建时拦截链路本地与云元数据地址。
- **鉴权**：非回环监听且无 `S3C_TOKEN` 时**拒绝启动**；compose 强制 `${S3C_TOKEN:?}`。
- **RustFS 联调**：口令改为环境变量必填；CORS 收紧；镜像 pin `1.0.0-rc.3`；健康检查不依赖 curl。
- **流式写超时**：`statusRecorder.Unwrap` 修复静默失效；改为 5 分钟滚动空闲超时。
- **异步迁移**：引擎级超时/取消 API、SSE 心跳与终态可靠投递、关停取消任务；前端 EOF 回读 + 取消按钮。
- **其他**：`sameEndpoint` 保留 scheme；presign 未知 method→400、默认 1h/上限 24h；ZIP 已压缩 Store + 断开感知；SVG 排除 inline；zip-slip NTFS 消毒；5GB 用 `5e9` 字节；标签上限 10。
- **运维**：`docker-compose.prod.yml`、nginx `proxy_buffering off`、health 探测 store、SQLite `user_version`、dependabot、CI gofmt/race/cover/desktop。

### 前端
- Escape 键栈（多层弹窗只关顶层）；KeepAlive max=5；分段上传段级重试；ZIP 优先 File System Access 流式落盘；i18n 高频键扩展；回收站减少空页扫库。

### 工程化与质量
- **s3wrap 拆分**：989 行单文件拆为 `client` / `bucket` / `object` / `presign` / `multipart` / `helpers` 六文件（单文件 <400 行）。
- **错误处理收敛**：`writeInternalErr` / `writeBadJSON` / `s3UserMessage` / `batchItemError` / `s3HTTPStatus`。
- **CI**：Web `pnpm test` + lint；E2E 独立 workflow（PR 路径触发 + 每周定时）；gofmt/race/cover/golangci 门禁。
- **前端**：Hash 深链接；`upload` / `i18n` 单测；主导航中英文切换（i18n 脚手架）。
- **发版**：`scripts/release-version.sh`；`make test-all`。
- **账号存储 SQLite**：`S3C_STORE_DRIVER=sqlite`（纯 Go `modernc.org/sqlite`，无 CGO），WAL 模式 `accounts.db`。
- **账号存储加密**：`S3C_STORE_DRIVER=encrypted` + `S3C_STORE_KEY`，AES-256-GCM 静态加密 `accounts.json.enc`。
- **移除遗留迁移**：不再从明文 `accounts.json` 自动导入到 sqlite/encrypted；前端不再迁移旧版单地址 localStorage 配置。
- **异步迁移 SSE**：`POST /api/migrate/async` + `GET /api/migrate/jobs/{id}/events` 实时进度；前端迁移面板改用 SSE。

## [v1.0.0-20260901182023] - 2026-09-01

> 版本命名改为 `v1.0.0-年月日十分秒`（如 `v1.0.0-20260901182023`）。本版本整合「对象存储类型 / 版本比较 / 一键还原、桶管理、回收站」三块迭代。

### 对象存储类型 + 版本比较 + 一键还原（Batch1）
- 存储类型（StorageClass）展示与切换：`HeadObject` 返回 `storageClass`（支持 `versionId`）；对象列表/网格与「对象详情」展示；新增 `POST /api/accounts/{id}/storage-class`（`CopyObject` 副本到自身 + `x-amz-storage-class`）一键切换（标准/低频/单区/智能分层/归档等）。
- 版本比较/详情：`presign` 支持 `versionId`（`PresignGetVersion` 对历史版本生成签名 GET）；「版本」对话框新增「版本比较」——选两个内容版本并排比元数据（大小/时间/存储类型/ETag）+ 逐行内容差异高亮（二进制或 >2MB 仅比元数据，可分别下载）。
- 一键还原已删除对象：新增 `POST /api/accounts/{id}/delete-marker/restore`（`DeleteObject` 删除标记版本）撤销删除；「版本」对话框删除标记行新增「还原」。

### 桶管理菜单（Batch2）
- 新顶层菜单「桶管理」：列出/新建/删除/管理桶；桶详情含 **概览（版本控制开关）、生命周期、加密（SSE）、CORS、网站托管、桶策略、桶标签** 7 个页签。
- 后端新增 `Get/Put/DeleteBucketEncryption`、`Get/Put/DeleteBucketCors`、`Get/Put/DeleteBucketWebsite`、`Get/Put/DeleteBucketPolicy`、`Get/Put/DeleteBucketTagging`（15 个端点）；`isNoSuchBucketSetting` 把「未配置」错误映射为空响应。

### 回收站菜单（Batch3）
- 新顶层菜单「回收站」：遍历列出桶内全部删除标记（分页游标，空页自动翻页）；每项支持 **一键还原** 与 **彻底清除**（`POST /api/accounts/{id}/trash/purge`，`PurgeObject` 永删该 key 全部版本+标记）。
- 新增端点：`GET /api/accounts/{id}/trash`、`POST /api/accounts/{id}/trash/purge`。

### 测试
- 假 S3 单测：`TestChangeStorageClass` / `TestRestoreDeleteMarker` / `TestPresignGetVersion` / `TestHeadObject` / `TestListObjectVersions` / `TestBucketSettings` / `TestTrash`，覆盖各读/写/校验与未配置分支。
- 真实 MinIO E2E：`TestE2EBatch1`（删除标记还原 + 版本预签名）、`TestE2EBucketSettings`（桶策略/标签）、`TestE2ETrash`（彻底清除 3 个版本+标记、无残留）均通过（MinIO 对扩展存储类型/CORS/网站托管/SSE 的 S3 API 支持受限，已做容错说明）。

### 四轴评审反馈修复（安全 / 正确性 / 可维护性）
- **前端修复**：Vue `key` 为保留属性、不复用为 prop——7 个弹窗（ACL/HTTP头/标签/存储类型/版本/版本比较/复制移动）的对象 key 全部改为 `objectKey`，此前这些功能实际不可用（已用 Vue 3.5 SSR 复现验证）；桶管理 6 个设置页签的 `watch` 增加 `immediate`（此前永不加载、保存会用默认值覆盖真实配置）。
- **安全加固**：CORS 中间件对非白名单 Origin 的普通请求直接 403（此前仅删响应头、请求照常执行），`readJSON` 强制 `application/json`；代理 `mode=inline` 增加 MIME 白名单（恶意 HTML/SVG 强制 attachment），代理支持 `versionId` 并按版本下载、补充 `Accept-Ranges`/416；创建账号忽略客户端提交的 `id`（杜绝覆盖已有账号）；ZIP 条目名同时按 `/` 与 `\` 消毒；compose 默认仅回环发布 + 未鉴权非回环监听启动警告 + Token 过短警告；实现 `.env` 加载（自带 30 行极简解析器）。
- **正确性/健壮性**：`PurgeObject` 改 `DeleteObjects` 批量（1000/批）消除 N+1；`deletePrefix` 上限改为页前检查+跨页切分（不再超删整页）；存储类型切换映射 `InvalidRequest`→400；multipart 段号校验（1..10000 且唯一）；（copy/migrate 改为 4 路有界并发，migrate 限 10k key、failKeys 上限 200；桶属性不再静默吞错（记录日志）。
- **前端健壮性**：对象浏览/桶/回收站/迁移的列表加载增加导航序号防过期响应；上传入队时捕获所属桶（防止切换桶串桶）；迁移面板列表补传 source bucket；shift 范围选择改为从 mousedown 捕获；数据面板用 `KeepAlive` 保持状态；`catch (e: any)` 全面改为 `catch (e)` + `toErrorMessage`；ModalDialog 增加焦点陷阱与初始焦点；版本比较下载改走服务端代理（不再 `window.open` 可执行内容）。
- **测试**：handler 覆盖率 60.3% → **70.8%**，原先 0% 的主端点全部补测（列表对象/批量删除/账号 GET-PUT-DELETE/连通性/预览桶/列出桶/桶级 5 个 DELETE/CORS 拒跨域/JSON Content-Type）；CORS 拒绝行为新增断言；E2E 空标签探针改为真实断言；新增前端 vitest 测试（`versionDiff`/`format`/`storageClass`/`errors`，17 例）并提取公共 `versionDiff` 模块（修复公共后缀被标成 `context` 的语义错误）。
- **清理**：删除死代码（`s3wrap.PresignGet`、`s3api.health`/`getAccount`、`genSign`/`SignUrlDialog`）；`.gitignore` 加入根 `data/`；文档同步（API.md 补 `preview-buckets`、README SDK 接口列表补全纠错、E2E 命令改为 `-run 'TestE2E'` 全量）。
- **第二轮补漏**：AWS SDK 类型外泄清理（对象/桶/ACL 转换下沉 `s3wrap`：`FromS3Object`/`FormatBuckets`/`DescribeACL`/`GranteeLabel`，handler 不再直接依赖 `s3.ListBucketsOutput` 等）；`streamCopy` 增加 5GB 单次上传上限提示；`filename*` 改用 RFC 5987 编码；store 持久化临时文件改 `O_CREATE|O_EXCL`（防抢占/符号链接重定向）且目录 0700、掩码标记语义化（`model.IsMaskedSecret`）；PreviewOverlay 支持 Escape 关闭；对象右键菜单支持方向键导航与初始焦点；抽取 `useBucketSetting` 组合式（桶管理 6 个设置页签去重，剩各自取数/提交逻辑）；api.ts 补充 localStorage 令牌安全权衡说明。

## [20260901.2] - 2026-09-01

### 对象版本：删除指定版本 / 版本回滚
- 后端 `s3wrap` 新增 `DeleteObjectVersion`（`DeleteObject` 带 `versionId`）与 `RestoreObjectVersion`（`CopyObject` 从 `?versionId=` 复制回当前 key；版本控制下写出一条新版本，返回新 VersionId）。
- 新增端点：`DELETE /api/accounts/{id}/version`（删除指定版本）与 `POST /api/accounts/{id}/version/restore`（把某历史版本恢复为当前）。
- 「对象版本」对话框每行新增「恢复」（删除标记禁用）与「删除」操作，均带确认弹窗与结果提示，操作后自动刷新版本列表。
- 假 S3 单元测试覆盖：删除版本（传 `versionId`）、恢复版本（`X-Amz-Version-Id` 回读）、缺 key / versionId 校验 400。
- 真实 MinIO 端到端验证：恢复历史版本（`CopyObject` 带 `?versionId=`）成功并写出一条新版本、删除指定版本成功。

## [20260901] - 2026-09-01

### 桶属性 + 对象版本管理
- 后端 `s3wrap` 新增 `GetBucketLocation` / `GetBucketVersioning` / `PutBucketVersioning` / `ListObjectVersions`。
- 新增端点：`GET /bucket-info`（区域 / 创建时间 / 版本控制状态）、`PUT /bucket-versioning`（`Enabled|Suspended`）、`GET /versions`（对象各版本 + 删除标记 + 分页游标）。
- 对象浏览页位置栏新增「桶属性」：展示区域、创建时间、版本控制状态，并可一键**开启/暂停版本控制**。
- 对象右键新增「版本」：列出该对象的历史版本（最新 / 历史 / 删除标记，含 VersionId/时间/大小）。
- **真实 MinIO 端到端联调**（新增 `s3wrap` E2E 测试，`S3CLINET_E2E=1` 运行）：验证建桶、PutObject/GetObject/HeadObject、`CopyObject`、**预签名 PUT 直传**、**三段式 Multipart（预签名单段 PUT + 组装）**（12MB 组装回读一致）、对象标签、`PutBucketVersioning` + 多版本覆盖写 + `ListObjectVersions`。
- 假 S3 单元测试覆盖：桶属性（区域/创建时间/版本状态）、版本控制开关与非法状态 400、版本列表（版本 + 删除标记）。

### 代码质量（Review 驱动重构）
- **后端 handler 拆分**：`handler.go` 2186 行 → 131 行，按领域拆出 `routes.go`/`middleware.go`/`accounts.go`/`buckets.go`/`objects.go`/`copy.go`/`proxy.go`/`zip.go`/`headers.go`/`multipart.go`/`metadata.go`/`migrate.go`/`health.go`（同包，行为不变）。
- **去除冗余 `/copy` 端点**（`copyObjects`，前端未使用），保留 `copy-object` / `copy-objects` / `copy-prefix` 三件套；相应移除其单测与文档。
- **默认桶可空语义**：`BucketOrDefault()` 空时返回 `""`（不再臆造 `"default"`）；新增 `bucketOr` 助手，请求缺省桶且账号无默认桶时返回明确的 `400 bucket is required`（替代静默打到不存在的桶）。
- **`downloadZip`**：zip 条目名脱敏（路径分隔符、`..` 上跳），单次打包上限 `1000` 个对象。
- **安全强化**：新增 `Content-Security-Policy`（脚本仅同源等）；`http.Server` 增加 `ReadTimeout`（不设 `WriteTimeout` 以免截断流式大文件）；`main.go` 版本默认值同步为 `20260901`。
- **前端 ObjectsPanel 拆分**：2041 行 → **442 行的编排层**。把 11 个弹窗/覆盖层与对象列表、工具栏、上传队列、桶列表、右键菜单抽为独立 `*.vue` 子组件；再把约 900 行编排逻辑抽到 `web/src/composables/`（`useObjectBrowser` / `useObjectActions` / `usePreview`）。行为与视觉不变。
- **`proxyUrl` 去重**：抽到共享 `web/src/proxy.ts`，`ObjectsPanel` 与 `PreviewOverlay` 共用一份实现。
- **handler 测试拆分**：`handler_test.go` 1861 行 → 按领域拆成 `helpers_test.go`/`accounts_test.go`/`buckets_test.go`/`objects_test.go`/`multipart_test.go`/`metadata_test.go`/`migrate_test.go`（33 个测试函数 + 6 个助手，无重复丢失）。
- 新增 `agents.md`：TDD 优先的开发规范与验收清单（与本改动一并落地）。

## [1.0.0] - 2026-08-28

首个稳定版（v1）。整合 v0.16.0 之后的全部迭代，此后按 `v1.0.0-年月日十分秒` 版本发布。

### 复制 / 移动 / 删除（跨桶，全闭环）
- 单文件复制 `POST /copy-object`（不删源）与批量复制/移动 `POST /copy-objects`（保留文件名，`deleteSource` 移动）；文件/文件夹统一「复制到… / 移动到…」对话框（目标桶下拉 + 目标路径/前缀）。
- 移动语义：文件 = 复制成功后删除源；文件夹 = 复制成功后删除源前缀（副本有失败则保留源）。
- 修复 `rename` 跨桶同名移动被误拒（同桶内 `newKey == key` 才拒绝）。
- 未知类型文件行内「预览」按钮常显。

### 大文件分段上传（Multipart Upload 直传）
- 后端 `multipart/init | part | complete | abort` 四端点；前端 `≥100MB` 自动切 10MB 段、4 路并发直传，任一段失败自动 abort；小文件仍单 PUT。
- 要求 Bucket CORS 暴露 `ETag` 响应头（见 README）。

### 对象元数据管理（HTTP 头 / ACL / 标签）
- 对象权限 ACL：`GET/PUT /object-acl`，私有/公共读切换 + 复制公开链接。
- 对象标签 Tagging：`GET/PUT /object-tags`，键值编辑、一键清空；无标签返回空列表（兼容 `NoSuchTagSet`）。

### 导航与交互
- 左侧菜单更名（账号管理 / 文件上传 / 服务器设置等），按依赖分组「数据操作 / 配置」，首次进入按账号存在路由到「对象管理」。

### 后端与测试
- 新增 S3 接口：`CreateMultipartUpload` / `Complete/AbortMultipartUpload` / `PresignUploadPart` / `Get/PutObjectAcl` / `Get/Put/DeleteObjectTagging`。
- 假 S3 单元测试覆盖上述能力与参数校验；`go test ./...`、`pnpm build` 均通过。

## [0.16.0] - 2026-08-28

### 文件迁移优化
- **源/目标 Bucket 下拉选择**：不再手输，加载账号下全部桶（含默认桶选项）。
- **目标账号默认值**：打开面板自动预选（优先另一个账号，其次同账号=复制到其他桶/前缀），避免忘选；下拉标注「（同账号）」。
- **迁移进度**：分批执行（每批 50 个）并实时显示「迁移中 x/y」与进度条。
- **迁移结果弹窗**：成功/失败/总计统计 + **失败对象清单**（后端新增 `failedKeys` 字段，最多展示 200 个，可滚动）+ 首个错误详情 + 「**去目标账号查看**」一键跳转（切换账号并进入对象管理）。
- **已选统计**：显示已选文件数与合计大小。
- 真实 MinIO 端到端验证：同账号迁移（CopyObject 路径）成功、目标桶结构正确、清理干净；`failedKeys` 字段经真实失败场景验证。

## [0.15.0] - 2026-08-28

### 行内「⋯ 更多操作」（互联网表格习惯）
- 对象列表每行右侧新增**常显「⋯」按钮**：点击弹出与右键一致的完整菜单（文件：下载/预览/复制签名链接/复制 Key/重命名/详情/删除；文件夹：打开/复制路径/复制/移动/删除文件夹），锚定按钮下方右对齐。
- 文件行操作列精简为「下载 / 预览 + ⋯」：签名、详情等低频操作统一收入更多菜单，行更干净（其他按钮仍 hover 显示，⋯ 常显）。
- 文件夹行操作列不再空白，同样提供 ⋯ 入口。
- CDP 真实浏览器验证：⋯ 弹出菜单、菜单项完整、菜单内操作（详情）联动正常。

## [0.14.0] - 2026-08-28

### 查看对象（显式化，云控制台习惯）
- **行操作新增「预览」按钮**：hover 行即可看到「下载 / 预览 / 签名 / 详情」，预览入口不再深藏右键菜单。
- **双击 = 查看**：双击文件打开预览弹窗（图片/视频/音频/PDF/文本/代码；未知类型自动转下载），双击文件夹进入——与 OSS/COS 控制台一致。
- **Enter = 查看选中文件**：快捷键语义从「下载」改为「查看」，未知类型自动转下载；快捷键提示条文案同步。
- 预览面板本身不变（v0.9.0 起的安全代理预览，文本转义渲染、PDF 沙箱）。

## [0.13.0] - 2026-08-28

### 对象属性与生命周期（参考 OSS/COS/TOS 控制台共性）
- **编辑对象 HTTP 头**：详情弹窗「编辑 HTTP 头」——修改 Content-Type 与自定义元数据（键值行编辑、可增删）；后端 `CopyObject` 复制到自己并 `MetadataDirective: REPLACE`，保存后详情即时刷新。
- **生命周期规则**：Bucket 列表每行「生命周期」——规则管理弹窗（规则 ID/前缀/过期天数，增删后整体保存）；后端 `Get/PutBucketLifecycleConfiguration`（简化版前缀过期删除）。
  - 未配置规则正确返回空列表（兼容 `NoSuchLifecycleConfiguration` 错误码）；
  - 清空规则走 `DeleteBucketLifecycle`（空 PUT 会被 MinIO 等实现拒绝，真实环境验证后修正）。
- 假 S3 单元测试：REPLACE 指令与 Content-Type/元数据头、生命周期读写 XML、非法规则（天数/重复 ID）校验；真实 MinIO 端到端验证（set-headers 后 head 确认、规则保存/读取/清空）。

## [0.12.0] - 2026-08-28

### 控制台化：Bucket 管理（参考 OSS/COS/TOS 网页控制台共性）
- **Bucket 列表页**：对象管理首层展示 Bucket 表格（名称/创建时间/进入/删除），点击「进入」管理对象；有默认桶的账号自动进入默认桶。
- **创建 Bucket**：弹窗表单（名称 + 读写权限：私有/公共读/公共读写），后端 `CreateBucket`（自动附带地域 LocationConstraint）；创建后自动进入。
- **删除 Bucket**：确认框提示须先清空；桶非空时后端返回 409 语义化提示。
- **统计条**：对象页顶部显示当前目录对象数/总大小、选中项数与合计大小（控制台习惯）。
- **列表 / 网格视图切换**：网格卡片（类型图标/名称/大小），单击选中、双击打开、右键菜单与列表一致。
- **返回桶列表**：对象页操作栏「← 返回桶列表」。
- 后端：`POST /api/accounts/{id}/bucket`、`DELETE /api/accounts/{id}/bucket`（含假 S3 测试：创建路径/ACL 头/命名校验/删除/非空 409）+ 真实 MinIO 端到端验证（创建/列表/删除/非法名 400）。

## [0.11.0] - 2026-08-28

### UI 交互：点击内容统一弹窗化
- 新增通用 **ModalDialog** 组件（标题栏 + 关闭按钮 + Esc/遮罩关闭 + 内容滚动）。
- **对象详情**、**签名 URL 结果**：由列表下方内嵌面板改为弹窗，不再被滚动带走，查看/复制更聚焦。
- **账号新增/编辑表单**、**服务端新增/编辑表单**：由内嵌表单改为弹窗（模态操作更清晰，主流 SaaS 习惯）。
- 保留内嵌的：上传进度（需持续可见）、迁移结果（消息条足够）、Toast、确认/输入/预览弹窗（已是弹窗）。

## [0.10.0] - 2026-08-28

### UI 交互重新设计
- **对象管理工具条重构**：拆分为「位置栏」（Bucket + 面包屑 + 路径编辑 + 过滤）与「操作栏」（刷新/上级 | 上传/新建文件夹 | 全选/批量操作），分组分隔线，主操作更突出。
- **文件管理器式行交互**：单击行=切换选中（文件夹=进入），双击=打开，操作按钮（下载/签名/详情）hover 时显示（窄屏常显），行内新增「详情」入口。
- **键盘快捷键**：`Enter` 下载选中文件、`F2` 重命名、`Delete` 删除所选、`Ctrl/Cmd+A` 全选（输入框/按钮聚焦时不抢占）；首次展示可关闭的快捷键提示条（localStorage 记忆）。
- **账号快速切换**：对象管理面板头部「当前账号」下拉，直接切换账号无需回账号面板。
- **统一 busy 态**：递归复制/移动/删除文件夹等耗时操作加防重复提交，相关按钮联动禁用。
- **空状态引导**：无账号时提供「+ 创建第一个账号」按钮，一键跳转账号面板并自动打开新增表单。
- **错误提示可重试**：对象管理错误条附「重试」按钮。
- **跨面板联动**：前端直传完成 Toast 带「查看对象」动作按钮，一键跳转对象管理；Toast 组件支持动作按钮。
- 上传入口更名为「上传文件」，操作栏按钮文案精简。

## [0.9.0] - 2026-08-28

### 新增
- **常见格式安全预览**：对象管理右键「预览」按类型分发——图片（含 SVG，`<img>` 上下文脚本不执行）、视频 / 音频（原生播放器，代理支持 Range 拖动）、PDF（`sandbox` iframe 禁脚本）、文本与代码（txt/md/json/csv/log/源码等 50+ 扩展名，转义文本展示，超大文件截断提示）；未知格式提示下载。

### 安全加固（展示侧攻击面收敛）
- **预览 / 下载统一走服务端代理**（`GET /api/accounts/{id}/proxy`）：
  - `mode=download` 强制 `Content-Disposition: attachment`，恶意 HTML/SVG/JS 内容**永不进入浏览器渲染管道**；
  - `mode=text` 强制 `text/plain + nosniff` 并服务端截断（默认 1MB），前端以转义文本渲染，根除 HTML 注入 / XSS；
  - 文件名清洗（去路径分隔符 / 引号 / 控制字符），防 `Content-Disposition` 头注入。
- 移除预览面板「在新窗口打开」；签名 URL 面板提示勿在浏览器直接打开（复制分享场景保留）。
- 对象管理所有「下载」入口（行内按钮 / 右键 / 双击）改为代理下载。
- 文本内容全程 `{{ }}` 转义展示，不渲染任何 HTML/Markdown。

### 后端
- 新增 `GET /api/accounts/{id}/proxy`（download/inline/text 三模式，Range 透传，404/400 语义化）。
- 假 S3 单元测试：attachment/inline 头、Content-Type 透传、Range 转发、文本强制纯文本与截断标记、nosniff、404/400、文件名清洗；真实 MinIO 端到端验证。

## [0.8.0] - 2026-08-28

### 新增
- **上传到当前目录**：对象管理工具栏直接选择文件上传（复用 presign PUT 直传，2 路并发），内嵌进度条与逐文件状态，完成后自动刷新列表；无需再切换到「前端直传」面板。
- **面包屑地址栏可编辑**：点击 ✎ 将路径变为输入框，直接输入完整前缀回车跳转（Esc 取消），深目录直达。
- **图片预览**：右键图片文件「预览」，弹层内嵌大图展示（缩放自适应），可一键在新窗口打开原图。
- **Shift 范围多选**：按住 Shift 点击复选框，连续选中区间内所有文件（文件管理器习惯）。

### 修复与改进
- 对象管理面板工具栏布局调整：上传/刷新/上级目录分组，右侧主操作（新建文件夹）保持。
- 预览、上传进度等新组件样式与深色模式自动适配。

## [0.7.0] - 2026-08-28

### 新增
- **国内主流对象存储预设**：登录页服务商增加腾讯云 COS、华为云 OBS、火山引擎 TOS、百度智能云 BOS、京东云 OSS、七牛云 Kodo；选区域自动填充 Endpoint / 公网 Endpoint（S3 兼容域名）。
- **海外常见对象存储预设**：AWS S3、Cloudflare R2、Wasabi、Backblaze B2、DigitalOcean Spaces、Linode/Akamai、Scaleway、Hetzner。
- **服务商三行分组**：兼容 / 国内 / 国外，便于快速选择。
- **批量下载（ZIP 打包）**：对象管理选中多个文件一键打包下载；后端 `POST /download-zip` 流式打包（不落盘、不占内存），获取失败的对象写入包内 `_下载失败清单.txt`。
- **删除文件夹（递归）**：右键文件夹「删除文件夹（含全部内容）」，后端循环 `ListObjectsV2` + 批量删除，上限 10 万对象保护；空前缀拒绝（防误删全桶）。
- **复制 / 移动文件夹（递归）**：右键输入目标前缀，后端逐 key `CopyObject`（同桶目标前缀与源重叠时拒绝，防无限复制）；移动 = 全部复制成功后才删除源。
- 真实 MinIO 端到端验证：复制结构正确、ZIP 条目正确、递归删除干净。

### 后端
- 新增 `POST /api/accounts/{id}/delete-prefix`、`POST /api/accounts/{id}/copy-prefix`、`POST /api/accounts/{id}/download-zip` 三个端点。
- 假 S3 单元测试：ZIP 内容与失败清单、递归删除分页计数、递归复制计数与重叠前缀拒绝、跨桶同前缀放行。

### 修复与改进
- 前端 `requestBlob` 支持二进制响应（错误时仍解析 JSON error）。
- 对象管理工具栏新增「下载所选(ZIP)」（打包中有 loading 态）。

## [0.6.0] - 2026-08-28

### 新增
- **对象列表「加载全部」**：循环分页直到末尾（上限 200 页保护），完成后汇总提示已加载的文件/文件夹数。
- **迁移面板「列出全部文件」**：列出源前缀下**所有文件（含子目录）**，循环分页（单页 1000、上限 20 万对象），支持全选一键迁移；已用真实 MinIO 验证递归与单层列出的差异。
- **上传完成项「复制链接」**：上传完成后一键复制 1 小时签名下载链接，即传即分享。
- 剪贴板能力抽取为共享模块 `web/src/clipboard.ts`（Clipboard API + textarea 降级），对象面板与上传面板统一使用。

### 修复与改进
- 对象列表与迁移面板的批量加载均有页数上限保护，避免超大桶长时间卡死；超出上限时明确提示已加载数量。
- 迁移面板「列出全部」与「列出对象」互斥禁用，防止并发请求交错。

## [0.5.0] - 2026-08-28

### 新增
- **对象管理增强**：
  - **新建文件夹**：工具栏一键创建（服务端 PUT 空对象，key 自动补全 `/` 结尾），真实 S3 验证通过。
  - **重命名 / 移动**：右键菜单操作，输入新 Key（可含路径即移动）；后端先 `CopyObject` 成功后才删除源，复制失败不丢数据；支持跨桶移动（`newBucket`）。
  - **对象详情**：右键「详情」调 `HeadObject`，展示 Key / 大小 / 修改时间 / Content-Type / ETag / 元数据；对象不存在返回 404。
  - **批量复制签名链接**：选中多个文件一键复制 1 小时签名链接（每行一个）。
- 通用输入对话框 `PromptDialog`（自动聚焦全选、Enter 确认、Esc 取消、内置校验），与确认框视觉统一。

### 后端
- 新增 `GET /api/accounts/{id}/head`（HeadObject 详情，404 语义化）、`POST /api/accounts/{id}/mkdir`（空对象建目录）、`POST /api/accounts/{id}/rename`（copy+delete）。
- 假 S3 单元测试：head 详情字段与 404/400、mkdir 路径规范化与空 body、rename 先复制后删除及复制失败不删源、同 key 拒绝。

### 修复与改进
- 前端交互文案与空状态保持一致；详情面板与签名面板同风格（深色模式自动适配）。

## [0.4.0] - 2026-08-28

### 新增
- **深色模式**：支持「跟随系统 / 浅色 / 深色」三态循环切换（顶栏主题按钮），选择持久化；全部颜色 token 化，深色下表格、表单、弹层、Toast、骨架屏等一并适配。
- **自定义确认对话框**：替换原生 `confirm()`（删除账号/对象/服务端统一体验），支持 Esc 取消、Enter 确认、遮罩点击关闭，危险操作红色警示图标。
- **对象管理右键菜单**：文件 → 下载 / 复制签名链接（1 小时）/ 复制 Key / 删除；文件夹 → 打开 / 复制路径。
- **对象管理双击交互**（文件管理器习惯）：双击文件=下载，双击文件夹=进入。
- **对象列表列排序**：名称 / 大小 / 修改时间表头可点击排序（升/降序切换），文件夹恒置顶。
- **对象列表本地即时过滤**：输入关键字过滤当前目录已加载条目，显示命中数，一键清除。
- **记住上次选中的账号**：刷新 / 重开后自动恢复上次登录的账号。
- 上传面板新增「清除已完成」：只移除已完成项，保留等待 / 失败项便于重试。

### 修复与改进
- 顶栏新增主题切换按钮（☀️ / 🌙 图标随实际主题变化，系统主题变化时自动刷新）。
- 表格可排序表头带 hover 反馈与排序指示箭头。
- 全局按钮 / 输入框 / 表格 / 面板等硬编码颜色全部收敛为 CSS 变量，为深色主题与后续定制提供统一入口。
- 对象管理空状态区分「无对象」与「无匹配项」两种提示。

## [0.3.0] - 2026-08-28

### 新增
- **OSS 式登录**（参考阿里云 OSS Browser）：账号表单新增「服务商」选择（阿里云 OSS / AWS S3 / S3 兼容）+「区域」预设下拉（OSS 21 个地域、AWS 20 个区域，选中自动填充 Endpoint 与公网 Endpoint），保存后自动切换为当前账号；新增 `web/src/regions.ts` 预设数据。
- 对象管理：新增「下载」操作（短时效 presign GET 直接打开）；「加载更多」改为追加分页并显示已加载数量。
- 前端直传：3 路并发上传 + 「重试失败」一键重传。
- CI：新增 GitHub Actions 工作流（Go vet/test/build、Web typecheck/build、Docker 镜像构建）。
- Web：`pnpm typecheck`（vue-tsc），`pnpm build` 前置类型检查。
- Go 测试：SPA fallback、鉴权路径边界、JSON 尾部数据、copy 部分失败（假 S3）、账号文件权限。
- **`/api/health` 返回服务端版本号**（`version` 字段，ldflags 注入）；「服务端配置」连通性检测后展示后端版本标签。
- 所有响应（含静态资源）新增基础安全头：`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: no-referrer`。

### UI 改版
- **C 端视觉风格**：浅色网格渐变背景、玻璃拟态顶栏（backdrop-blur）、品牌渐变 Logo 与渐变标题、侧边栏激活项渐变胶囊、浮动卡片（大圆角 + 柔和投影）、渐变主按钮/进度条、标签页切换过渡动画、卡片入场动效、细圆角滚动条。
- 设计系统重写：语义化 CSS token、统一间距/圆角/阴影、按钮尺寸体系（`btn sm`）、表格粘性表头与选中行高亮、`focus-visible` 焦点环。
- 头部重构：服务器地址/Token 收进「⚙ 设置」弹层；新增后端连接状态指示灯（已连接/连接异常）。
- 全局 Toast 反馈（右下角、自动消失）：账号增删改、对象删除、复制、上传完成等操作结果。
- 对象管理：面包屑导航（根目录 / 逐级可点）、骨架屏加载、目录名改为可键盘聚焦的按钮、空状态带图标。
- 前端直传：拖拽上传区（点击/拖拽/键盘均可触发）、整体进度条、单文件移除。
- 账号表单：编辑时 SecretKey 占位提示「留空则保持不变」；连通性状态简化为 未测试/测试中/正常/失败。
- 迁移面板：Bucket 输入框占位显示账号默认桶；大小列格式化。
- 响应式：窄屏（<900px）侧边栏折叠为顶部横向导航；`index.html` 增加 favicon 与 theme-color。
- 主题色：品牌渐变与主色调由科技蓝切换为**薄荷绿**（`--brand` `#2dbd98→#0e8c66`、`--primary` `#10a37c`），并联动阴影、焦点环与各 tint。

### 修复与改进
- **修复跨 endpoint 流式迁移在明文 HTTP 端点失败**：AWS SDK for Go v2 1.107 默认对 `PutObject` 计算请求 CRC32 校验与 payload SHA-256，而迁移的流式 body 不可 seek，非 TLS 端点直接报错（`unseekable stream is not supported...`）。现改为 `RequestChecksumCalculation=WhenRequired`，并通过 finalize 中间件在 payload hash 计算前预置 `UNSIGNED-PAYLOAD`（与 HTTPS 下 SDK 行为一致），流式迁移恢复正常（新增假 S3 集成测试覆盖）。
- **修复 SPA fallback 失效**：静态文件未命中时此前直接 404，现对无扩展名的页面路由正确回退 `index.html`；带扩展名的缺失资源仍 404。
- **安全**：Bearer Token 校验改为常量时间比较（SHA-256 摘要 + `subtle.ConstantTimeCompare`），降低时序侧信道风险；`accounts.json` 写入权限 `0600`（含明文密钥）；`Update` 持久化失败回滚内存状态。
- **健壮性**：S3 HTTP client 增加建连/TLS/响应头超时与连接池上限（不设整体超时，避免截断大文件流式迁移）。
- 前端「下载」由 `window.open` 改为程序化锚点点击，避免异步请求后被浏览器弹窗拦截（新标签页无法打开）。
- `/api/accounts/{id}/copy` 与 migrate 行为对齐：逐 key 复制、失败继续，响应新增 `failed`/`lastError` 字段。
- 鉴权路径判断精确化（`/api` 或 `/api/` 前缀），不再误伤 `/apiary` 之类路径；JSON 请求体拒绝尾部多余数据；健康检查日志降为 debug。
- 日志状态记录器忽略多余的 `WriteHeader` 调用，避免状态与实际响应不一致。
- 前端：对象列表分隔符默认 `/`（与占位提示一致，目录可浏览）；全选 checkbox 状态修正；迁移列表大小格式化；GET 请求不再携带多余 `Content-Type`（避免跨域预检）。
- 重构：新增 `web/src/format.ts` 共享格式化工具（`fmtSize`/`fmtDate`），对象/直传/迁移三个面板去重。
- 构建：Dockerfile 拷贝 `pnpm-lock.yaml` 并改 `--frozen-lockfile`（可复现）；Go 构建通过 ldflags 注入版本号；Makefile 补全 `.PHONY` 并新增 `test`/`vet`/`web-typecheck`/`docker` 目标；移除 pnpm 11 已忽略的 `package.json#pnpm` 字段。
- 测试：新增 config 包测试（默认值/覆盖/列表解析）、健康检查版本与安全头断言、错误 Token（含长度不同）鉴权用例、migrate 跨 endpoint 流式复制假 S3 集成测试。

## [0.2.0] - 2026-08-22

### 新增
- 服务端集中配置（`internal/config`），支持 `S3C_*` 环境变量与 `.env`（见 `server/.env.example`）。
- **安全加固**：默认绑定回环 `127.0.0.1`；CORS 白名单（`S3C_CORS_ORIGINS`）；可选 Bearer 鉴权（`S3C_TOKEN`）。
- **容器化**：多阶段 `server/Dockerfile`（web + Go + `debian:bookworm-slim` 运行时，非 root `app`，自带 `HEALTHCHECK`，`/data` 已授权）；`docker-compose.yml`（server + MinIO）；构建参数 `GOPROXY`/`NPM_REGISTRY` 可覆盖；新增 `/s3clinet-server -healthcheck` 自检子命令。
- Go 单元测试：store / s3wrap / handler（CRUD、持久化、脱敏、原子写、CORS 策略、鉴权、迁移端点判断）。
- 文档：REST API 参考（`docs/API.md`）、配置矩阵、运行/部署说明、容器直传可达性说明。

### 修复与改进
- 修复跨 provider 迁移误用 `CopyObject`：`sameEndpoint` 仅在两端点一致且非空时判定为同端点，否则流式复制。
- 跨 endpoint 迁移保留源对象 `Content-Type` 与元数据。
- 预签名有效期限制在 1s–7 天；对象 `maxKeys` 限制 1–1000。
- 账号更新增加必填校验，避免清空 endpoint/accessKey。
- 账号存储改为原子写（临时文件 + rename）；`Create` 写盘失败回滚内存状态。
- 账号缺省 region 为空时回退 `us-east-1`。
- 前端：上传队列直接持有 `File` 引用（避免同名误匹配）；设置区新增 Token；签名复制支持降级；对象列表加载状态。

## [0.1.0] - 2026-08-22

### 新增
- 首个版本：Go 后端（AWS SDK for Go v2，封装 11 个 S3 接口）、Vue3+Vite+TS 前端（账号/列对象/直传/签名/删除/迁移/加前缀）、Tauri 2 桌面壳（无 IPC，B/S）。
