# 全仓安全评审（2026-10-15）

> **状态**：**活跃快照**——R1 / R4 / R5 已闭环；R2a / R2b / R3a 与两项低风险加固已落地并验证，
> R3a 按 [`AI_POLICY.md`](AI_POLICY.md) §3 仍须人类专家复核后方可合并。复核完成后按 [archive/index.md](archive/index.md)
>「归档四步」冻结。结论绑定评审时点 HEAD；文中 `file:line` 与工具数字均为**评审时点快照**，不回写。
>
> **同日追加（第二轮并行全仓安全评审）**：另一次按攻击面并行识别（Go 后端 handler / 服务层 /
> Web 前端 / CI 与桌面壳）+ 独立误报过滤的复核，结论同为「无高危 / 中危漏洞」；其识别出的两条
> **已文档化残留**于同日收口（R6 / R7，证据见 [`FEATURES.md`](FEATURES.md) §CT），三项防御纵深
> 随后收口（R8–R10，证据见 [`FEATURES.md`](FEATURES.md) §CU）。
>
> **范围**：全仓——`apps/server` Go 后端（308 个 `.go`）、`apps/web` Vue 3 + TS（185 个生产
> `.ts`/`.vue`）、`apps/desktop`（Tauri 2 无 IPC 壳）、部署（Dockerfile / compose / nginx TLS）、
> CI / 供应链（GitHub Actions + GitLab CI）、`.env` 与 Git 历史泄露面。
>
> **方法**：第一性原理——先枚举**资产**（S3 凭据 / Bearer Token / 账号库 / 对象数据 / 预签名 URL）
> 与**信任边界**（浏览器 ↔ Go 后端、后端 ↔ S3、磁盘账号库、Tauri 壳、部署入口），再按 STRIDE
> 逐边界核验代码中的控制是否**真实兑现**（不采信文档声明，逐条回到源码 `file:line`）。
>
> **结论**：**Approve**。未发现高危 / 中危漏洞；R1–R5 均为低危或观察项。安全不是靠扫描器堆出来的，
> 而是在架构层（分层、无 IPC 桌面壳、fail-closed 配置）就做了正确取舍。

## 1. 机械门禁与工具实测（评审时点，如实披露）

| 检查 | 结果 | 说明 |
|---|---|---|
| `cd apps/server && go test . -count=1`（包根门禁） | **PASS**（6.2s） | 改文档前先跑基线；含 `doc_link` / `doc_index` / `docs_naming` 等门禁 |
| `govulncheck ./...` | **0 可达漏洞** | 另有 21 条存在于 required modules 但调用链不可达，不作门禁 |
| `pnpm audit --prod`（npmjs registry） | **0 漏洞** | `apps/web` 唯一运行时依赖为 `vue` |
| `cargo audit` | **2 条已放行告警** | `proc-macro-error` unmaintained（RUSTSEC-2024-0370）与 `glib 0.18.5` unsound（RUSTSEC-2024-0429），均为 Tauri 传递依赖，与既有台账口径一致 |
| 前端危险模式静态扫描 | **0 处** | 生产代码无 `v-html` / `innerHTML` / `outerHTML` / `eval(` / `new Function` / `document.cookie` / `insertAdjacentHTML`（仅测试文件清理 DOM 用到 `innerHTML`） |
| Go 危险模式静态扫描 | **干净** | 无 `InsecureSkipVerify`；`exec.Command` 仅出现在门禁测试；S3 出站 `CheckRedirect` 拒绝 + `Proxy = nil` |
| 秘密泄露面 | **干净** | `.env` 未入库（`git ls-files` 无）；`git log --all -- .env` 为空；tracked 文件无真实凭据字面量（命中均为测试样例 / 配置占位） |
| SQLite 注入面 | **干净** | 全参数化；唯一拼接是常量列清单 `sqliteListCols`（`internal/store/sqlite.go:240`） |

## 2. 处置状态总览

| 编号 | 级别 | 一句话 | 状态 |
|---|---|---|---|
| R1 | 低（文档漂移） | [threat-model.md](threat-model.md) §2 称 XFF「采信首段」，代码实际取**最后一段**（`apps/server/internal/handler/ratelimit.go:120`）——代码正确（取最后一段防客户端在左侧预置伪造值换桶绕过限速），文档过期 | ✅ 同日修正（订正为「采信其直连对端追加的最后一段」） |
| R2a | 低（信息暴露） | `/api/health` 免鉴权暴露 `version` | ✅ 已修（仅有效 `S3C_TOKEN` 可见 `version`；探针仍 200） |
| R2b | 低（信息暴露） | `S3C_EXPOSE_METRICS=1` 时 `/api/metrics` 可匿名读 | ✅ 已修（新增可选 `S3C_METRICS_TOKEN`；未设保持兼容） |
| R3a | 低（CSP 取舍） | 服务端 CSP `style-src 'unsafe-inline'` 允许内联 `<style>` | ⚠️ 代码已收紧（`style-src 'self'; style-src-attr 'unsafe-inline'`）；待人类专家复核 |
| R3b | 低（CSP 取舍） | 桌面端 `connect-src http: https:` 全放开（须连任意用户配置的 S3 端点） | ➖ 维持（2026-10-14 C1 已复核并记入 `apps/desktop/AGENTS.md`） |
| 配置边界 | 信息（加固） | `S3C_CSP_CONNECT_SRC` 无启动期内容校验 | ✅ 已修（控制字符 / 换行 / 纯空白拒绝启动） |
| 限流可观测 | 信息（加固） | 限流参数无运行时指标 | ✅ 已修（新增 rpm / burst gauge） |
| R4 | 低（限流残余面） | 限流固定 120 req/min · 突发 30 且按**单 IP** 计数（`apps/server/internal/handler/ratelimit.go:13`）——IPv6 攻击者可用 /64 内海量地址各自获桶 | ✅ 已修（同日：新增 `S3C_RATE_LIMIT_RPM` / `S3C_RATE_LIMIT_BURST` 可配置 + IPv6 按 /64 前缀聚合） |
| R5 | 信息（行为惊讶） | `presign` 的 `expiresIn <= 0` 静默取 1h 而非 400（`apps/server/internal/handler/objects.go:428`）——非漏洞，仅行为惊讶 | ✅ 已修（同日：负数返回 400；`0` / 缺省仍默认 1h，`multipart/part` 同口径） |
| R6 | 低（越权面） | `accounts` 作用域只按路径 `{id}` 判定，迁移 / 计划 body 与已存计划里的账号 id 落空——`accounts:[A]` 的 token 可让服务端用 B 的存储凭据执行迁移（越权读写） | ✅ 已修（同日第二轮：`requestAccountRefs` 纳入判定，越界 `403`） |
| R7 | 低（信息暴露） | `GET /api/schedules` 对受限 token 返回全部计划，越界计划的桶名 / 账号 id 可见（不可触发 / 删除） | ✅ 已修（同日第二轮：`schedulesInScope` 按作用域裁剪） |
| R8 | 低（防御纵深） | nginx 层未设 `X-Content-Type-Options: nosniff`；直连静态文件或未来新增 vhost 会绕过后端中间件兜底 | ✅ 已修（base / TLS nginx 配置补 `always` 头 + 门禁） |
| R9 | 低（防御纵深） | GitLab CI 用 `curl ... | bash` 执行 NodeSource 安装脚本，远端脚本内容未被校验 | ✅ 已修（固定 Node 版本压缩包 + 官方 SHA256 校验 + 门禁） |
| R10 | 低（防御纵深） | 下载文件名取自 S3 key 最后一段并直接交给 `showSaveFilePicker` / anchor `download` | ✅ 已修（保存前清洗控制字符、路径分隔符、非法文件名字符并限长） |

## 3. 第一性原理核验（资产 → 边界 → 控制）

### 3.1 身份认证与授权

- Bearer Token 用 SHA-256 + `subtle.ConstantTimeCompare` 常量时间比较，防时序侧信道（`apps/server/internal/handler/middleware.go:236`）；多 Token 轮换遍历比较，命中后再做作用域判定。
- 非回环监听强制 Token、最短 16 字符、非法值 fail-closed 拒绝启动（`apps/server/internal/config/config.go:14`、`Validate`）。
- Token 作用域最小权限 + 过期时间：受限 Token 无法铸造新账号（`apps/server/internal/handler/scope.go:62`）、端点级作用域闸收口 `preview-buckets` 跳板面。

### 3.2 网络边界（CORS / 限流 / SSRF）

- CORS 白名单外**硬 403 阻断**（而非仅省略响应头），配合「变更请求强制 `application/json` 触发预检」，防简单请求跨域写（`apps/server/internal/handler/middleware.go:165`）。
- 限流挂在鉴权**外层**（鉴权失败也消耗额度）；XFF 仅当直连对端在 `S3C_TRUSTED_PROXIES` 白名单时采信其追加的**最后一段**，否则回退 `RemoteAddr`（`apps/server/internal/handler/ratelimit.go:120`）；分片桶 + 总量硬上限 10 000，内存有界。
- SSRF 双重防线：创建期 `ValidateEndpoint` + 拨号期 `dialContextSSRF` 逐 IP 复查（防 DNS 重绑定 TOCTOU）；禁重定向、禁 `HTTP(S)_PROXY` 环境变量代理、拦截链路本地与云元数据地址（`apps/server/internal/s3wrap/ssrf.go`、`client.go:192`）；私网默认放行为 [decisions/0003-ssrf-private-allow.md](decisions/0003-ssrf-private-allow.md) 的显式取舍，可 `S3C_SSRF_DENY_PRIVATE=1` 收紧。
- 安全头：CSP `script-src 'self'` / `style-src 'self'` / `style-src-attr 'unsafe-inline'` / nosniff / `X-Frame-Options: DENY` / `Referrer-Policy: no-referrer`（`apps/server/internal/handler/middleware.go`）；TLS 层补 HSTS 180 天 + `Permissions-Policy`（`deploy/nginx/conf.d/s3client-tls.example.conf`）。

### 3.3 输入与输出

- 请求体强制 `application/json`、16MB 上限、`DisallowUnknownFields`、拒绝尾随数据（`apps/server/internal/handler/handler.go:250`）；`X-Request-ID` 回显前校验长度与可见 ASCII，防日志伪造 / 响应头拆分。
- API 响应用 `AccountView` DTO，`secretKey` 不出服务端（`apps/server/internal/model/account.go:41`）；内部错误只回通用文案，S3 错误经白名单映射（`apps/server/internal/handler/handler.go:215`）。
- 前端生产代码 **0 处** `v-html` / `innerHTML` / `eval`；高亮组件逐段 `template v-for` 渲染（`apps/web/src/components/HighlightText.vue`）；Token 默认 `sessionStorage`，持久化需显式开启且多服务器 profile 不落 Token（`apps/web/src/api/storage.ts:16`）。

### 3.4 数据与凭据（静态存储）

- 加密驱动：Argon2id（t=2, m=64MiB, p=4）派生 + AES-256-GCM；S3C3 文件头 KDF 参数有上界校验，防篡改参数（如 4GiB memory）造成的读放大 DoS（`apps/server/internal/store/crypto.go:55`）。
- 原子写：唯一临时名 + `O_EXCL`（防符号链接抢占 / 并发互踩）、临时文件先 `chmod 0600` 再 rename（中间窗口无 0644 泄露）、fsync 文件与父目录（`apps/server/internal/atomicfile/atomicfile.go:59`）；SQLite 主库与 WAL 侧车统一 0600、数据目录尽力 0700。
- `json` / `sqlite` 驱动无 `S3C_STORE_KEY` 时拒绝启动（需显式 `S3C_ALLOW_PLAINTEXT_STORE=1`），安全闸 fail-closed（`apps/server/internal/config/config.go:26`）。

### 3.5 桌面壳与部署

- Tauri 零 IPC：`main.rs` 无 `invoke_handler` / 自定义 command、capabilities 空权限、`withGlobalTauri: false`（`apps/desktop/src-tauri/src/main.rs`）——本地命令面不存在，权限完全取决于同源策略 + 服务端鉴权。
- 容器非 root（`USER app`）、镜像按 digest pin、端口默认仅回环发布、`S3C_TOKEN` / `S3C_STORE_KEY` 缺失直接启动失败（`docker-compose.yml:43`）；`ReadHeaderTimeout 15s` 防慢速请求头。
- CI 供应链：GitHub Actions 全部 `uses:` pin 完整 commit SHA；GitLab 需要 Node 的组合工具链 job 下载固定版本官方压缩包并校验 SHA256；CodeQL（Go + TS）、govulncheck、gosec、dependency-review、Scorecard 均在流水线内。

## 4. 残余风险与建议

1. ~~修正 threat-model XFF 表述~~（R1，已同日闭环）。
2. ~~health `version` 信息暴露 / metrics 匿名面~~（R2a / R2b，已同日闭环）。
3. ~~IPv6 限流按 /64 聚合（R4）~~（已同日闭环，见 §2 状态表与 [`../CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`）。
4. R3a 代码已收紧，须按 [`AI_POLICY.md`](AI_POLICY.md) §3 完成人类专家复核；复核通过后本快照方可归档。
5. ~~`accounts` 作用域对迁移 / 计划账号引用的越权面 + 计划列表信息暴露~~（R6 / R7，同日第二轮闭环，证据见 [`FEATURES.md`](FEATURES.md) §CT）。
6. ~~nginx `nosniff`、GitLab NodeSource 安装脚本、下载文件名清洗~~（R8–R10，防御纵深闭环）。

## 5. 验证与未复跑项披露

- **后端**：`go vet ./...` / `go test ./...` / `go build ./...` 全绿；`make test-cover` 11/11 包 **100.0%**、零 `count==0` 块；`golangci-lint run` **0 issues**；包根配置 / 文档 / OpenAPI 门禁全绿。
- **前端与浏览器**：`pnpm lint` / `typecheck` / `typecheck:e2e` / `test`（88 文件 1364 例）/ `test:coverage`（四指标 100%）/ `build` / `gen:api --check` 全绿；`pnpm e2e` **26 passed + 1 skipped**；`make e2e-real`（真实 Go 后端 + RustFS + 真实产物，Playwright 不 mock `/api`）**5/5 PASS**，CSP 收紧后页面渲染、真实落库、建桶与浏览器直传均通过。
- **部署配置**：`docker compose config` 的 base / prod / tls 三套均通过。
- **未复跑**：`S3CLIENT_E2E=1` RustFS 协议 E2E（本批未改签名 / 分段 / 复制等 s3wrap 协议面；最近一次见 [`../CHANGELOG.md`](../CHANGELOG.md) R4/R5 批次）；Trivy / CodeQL / dependency-review 远端流水线。**R3a 尚待 [`AI_POLICY.md`](AI_POLICY.md) §3 要求的人类专家复核**。
