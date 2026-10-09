# 安全设计

> 本文档描述 s3client 的威胁模型、安全边界与默认值。漏洞报告流程见 [SECURITY.md](../.github/SECURITY.md)。
> 本文档基于 2026-09-16 综合安全审计（详见 [archive/assessment.md](archive/assessment.md) §二），其后按修复进展滚动更新。
> 最后更新：2026-09-30。

## 1. 威胁模型（STRIDE × 边界）

> 状态图例：✅ 已缓解 · ⚠️ 部分缓解 · ❌ 未缓解 · ➖ 不适用 / 已决策放行（理由见「缓解」列）。

### 边界 A：HTTP API（Bearer 鉴权）

| 威胁 | 缓解 | 状态 |
|---|---|---|
| **S**poofing 仿冒 | Bearer 常量时间比较（sha256+subtle）、scheme 大小写不敏感（RFC 7235）、多 token 轮换、最短 16 字符 | ✅ 已缓解 |
| **T**ampering 篡改 | Content-Type 非空时必须为 `application/json`（缺省放行，仍受 JSON 解码器约束）+ `DisallowUnknownFields` + 16MB body cap（超限回 413）；配合 CORS 使跨域变更必预检 | ✅ 已缓解 |
| **R**epudiation 抵赖 | 安全审计日志（`handler/audit.go`）：401（malformed/bad_token）/ 账号 CRUD / 桶策略设置与清除 / 对象删除与前缀删除 / **移动与重命名**（同步批量 `copy-objects` + `deleteSource`、异步批量、`rename` 复制后删源）/ **版本永久删除**（`DELETE /version`，带 `versionId`）/ 回收站清空 / 限速命中，带固定事件名与操作者 IP；另有通用 access log | ✅ 已缓解（原 todo #17 闭环见 [FEATURES.md](FEATURES.md) §M；2026-09-28 补齐三处破坏性操作覆盖见 §AE） |
| **I**nfo disclosure 泄露 | 错误脱敏、S3 错误稳定映射、AccountView 不含 secretKey | ✅ 已缓解 |
| **D**oS 拒绝服务 | IP 令牌桶 120/min、流式并发 32、批量上限齐备；XFF 仅 `S3C_TRUSTED_PROXIES` 命中才采信（默认空 = 不信任，防伪造绕过限速）；在册任务 ≤256 超限 503 | ✅ 已缓解（原 todo #17 两项缺口均已闭环） |
| **E**levation 提权 | 默认单 token 全权模型；**可选最小权限作用域**（`S3C_TOKEN_SCOPES`：只读 / 桶前缀 / 账号白名单 / 过期）按 token 收敛爆炸半径；token 轮换支持 | ✅ 已缓解（越权 403 + 审计 `auth.scope_denied`；未登记 token 保持全权，见下方「最小权限」） |

**最小权限（`S3C_TOKEN_SCOPES`，ROADMAP §三 #13）**：`S3C_TOKEN` 支持逗号分隔多 token，但历史上每个 token 都是全权。现可选用独立 env `S3C_TOKEN_SCOPES`（JSON：token → `{readonly, prefixes, accounts, expiresAt}`）为单个 token 声明最小权限：

- `readonly`：仅放行 GET / HEAD；写方法（含能铸造写 URL 的预签名 `POST`）一律 403；
- `prefixes`：`"<bucket>"` 或 `"<bucket>/<key前缀>"`，请求涉及的桶/键（query 或 JSON body）与列表 `prefix` 必须落在许可内；桶级操作需该桶的整桶授权；body 无法解析/超限时 fail-closed；计划任务的 `run`/`DELETE` 无 body，按**已存计划的源/目标桶与前缀**注入引用判定（越界计划不可触发 / 删除）；
- `accounts`：仅允许路径 `{id}` 命中的账号；
- `expiresAt`：过期后 401（审计 `reason=token_expired`）。

未在表中登记的 token **仍是全权**（向后兼容）。非法配置（未知字段 / 未登记 token / 空元素 / 坏时间）**拒绝启动**，防止「以为限权、实际全权」。拒绝写审计事件 `auth.scope_denied`（`reason` = `readonly` / `prefix` / `account` / `unparsable_body`），**审计与响应均不含 token 明文**。配置 SSOT 见 [`CONFIGURATION.md`](CONFIGURATION.md)。

> 残留（有意）：作用域是「token 级粗粒度」而非 S3 IAM——`prefixes` 只约束请求显式给出的桶/键，桶列表（`GET /api/accounts/{id}/buckets`）、账号列表与**计划列表（`GET /api/schedules`，可读到越界计划的桶名，但不可触发 / 删除）**不受其约束；`accounts` 作用域按路径 `{id}` 判定，迁移 / 计划 body 里的账号 id 不受其约束（与 `prefixes` 按 body 桶判定不同）；需要更细粒度授权时应在账号侧用 S3 策略收敛。

### 边界 B：预签名 URL 直传

| 威胁 | 缓解 | 状态 |
|---|---|---|
| **S**poofing 仿冒 | 签发端点在 Bearer 鉴权内（`routes.go` `POST /api/accounts/{id}/presign`）；URL 内嵌 AWS SigV4 签名（派生自 SecretKey，覆盖方法 + 对象 key + 过期时间），持有者仅得该次操作 | ✅ 已缓解 |
| **T**ampering 篡改 | 方法白名单 `get\|put\|post`（缺省 `put`，非法 400）；`expiresIn` ≤0 取默认 1h、**>24h 钳制 24h、无 1h 下限**（`objects.go` `presign`、`multipart.go` `multipartPart`）；签名被改一字节即失效，S3 端复核 | ✅ 已缓解 |
| **R**epudiation 抵赖 | ➖ 不适用：直传不经本服务（[architecture.md](architecture.md) §5），无服务端操作可记；对象侧留痕以 S3 访问日志为准 | ➖ |
| **I**nfo disclosure 泄露 | SecretKey 永不回传浏览器，只回预签名 URL；URL 随过期失效 | ✅ 已缓解 |
| **D**oS 拒绝服务 | ➖ 不适用（对本服务）：直传流量不经本服务，容量与限速归 S3 上游；签发面受边界 A 的限速 / body cap 保护 | ➖ |
| **E**levation 提权 | ➖ 不适用：URL 仅授予签名时指定的单次操作权限，不引入角色、不超过账号既有策略 | ➖ |

### 边界 C：账号存储（SecretKey 落盘）

| 威胁 | 缓解 | 状态 |
|---|---|---|
| **S**poofing 仿冒 | ➖ 不适用：本地文件系统位于应用信任边界外（OS / 容器隔离；容器 `USER app` 非 root） | ➖ |
| **T**ampering 篡改 | 原子写（临时文件 + rename）+ 写失败回滚内存；GCM 认证标签使密文被改即解密失败 → `store.Open` 失败 → 进程硬失败退出（[ADR-002](decisions/0002-store-fail-closed.md)）；数据目录 flock 单写者锁（`store/lock.go` `AcquireDataDirLock`，第二实例立即失败） | ✅ 已缓解 |
| **R**epudiation 抵赖 | 账号 CRUD 记安全审计日志（`handler/audit.go` `account.*`，见边界 A）；落盘动作本身不单独记审计 | ✅ 已缓解（审计在边界 A） |
| **I**nfo disclosure 泄露 | 核心威胁：配 `S3C_STORE_KEY` 时 AES-256-GCM + Argon2id（文件盐建时随机并复用，key 最短 16 字符）；数据目录 0700 + 文件 0600（目录在 `store.Open` / `store.AcquireDataDirLock` 启动时**主动 chmod 收紧**，Docker volume / systemd 预建的 0755 也会被改成 0700；文件侧 `chmodSQLitePerms` 同口径）；json / sqlite 无 key 拒绝启动（`S3C_ALLOW_PLAINTEXT_STORE=1` 仅限联调，见下） | ✅ 已缓解 |
| **D**oS 拒绝服务 | ➖ 不适用：本地文件无网络面；可用性按 ADR-002 硬失败不降级 | ➖ |
| **E**levation 提权 | 文件 0600 仅属主可读写 + 容器非 root 运行 | ✅ 已缓解 |

**落盘明细（驱动与格式）**：

| 驱动 | 落盘 | 说明 |
|---|---|---|
| `json` | 明文 JSON 或 S3C3 加密 | 配 `S3C_STORE_KEY` 时 AES-256-GCM + Argon2id（参数随文件版本）；permissive 兼容读旧明文与旧 S3C2 |
| `sqlite` | secret_key 列明文或 S3C3 加密 | 配 `S3C_STORE_KEY` 时该列以 AES-256-GCM 密文落盘；历史明文行仍可读、写回即加密（已闭环，证据见 [FEATURES.md](FEATURES.md) §M；残留风险见 [ROADMAP.md](ROADMAP.md) §5.1「已收敛」索引 R3） |
| `encrypted` | S3C3 加密（严格） | ✅ 生产推荐；文件盐建时随机并复用；可读旧 S3C2 库 |

所有驱动：原子写（临时文件 + rename）+ 0600 权限 + 写失败回滚内存。**未配置 `S3C_STORE_KEY` 的
`json` / `sqlite` 驱动会拒绝启动**（安全默认；#29/#31 已闭环，证据见 [FEATURES.md](FEATURES.md) §T），除非显式设置
`S3C_ALLOW_PLAINTEXT_STORE=1`——此时允许运行并打出「secretKey 将明文落盘」WARN
（[ROADMAP.md](ROADMAP.md) §5.1「已收敛」索引 R3），仅限本地联调；生产必须用 `encrypted` 或
`sqlite` + key。base `docker-compose.yml` 亦以 `${S3C_STORE_KEY:?…}` 强制非空。
加密文件格式：S3C3 头部内嵌 Argon2id 参数（time/memory/threads），因此可在不破坏既有库的前提下调参；
S3C2 旧格式仍可读（升级路径）。`S3C_STORE_KEY` 非空时要求 ≥ 16 字符。

### 边界 D：S3 上游（SSRF）

| 威胁 | 缓解 | 状态 |
|---|---|---|
| **S**poofing 仿冒 | DNS 重绑定消除 TOCTOU：创建期 `ValidateEndpoint` + 拨号期 `dialContextSSRF` 二次解析逐 IP 校验 | ✅ 已缓解 |
| **T**ampering 篡改 | 无 `InsecureSkipVerify`（证书校验不降级）；禁重定向（防 3xx 跳到元数据地址）+ 禁 HTTP(S)_PROXY（封环境变量代理绕过） | ✅ 已缓解 |
| **R**epudiation 抵赖 | ➖ 不适用：出站目标决策不记审计事件；策略生效值可观测（见下「生效值核对」） | ➖ |
| **I**nfo disclosure 泄露 | 拦截链路本地与云元数据：阿里云 IMDS（100.100.100.200）、火山引擎（100.96.0.2）、AWS IMDS IPv6（fd00:ec2::254）、GCP metadata 主机名；私网 / 回环放行为自托管取舍（[ADR-003](decisions/0003-ssrf-private-allow.md)），`S3C_SSRF_DENY_PRIVATE=1` 可将私网 / 回环 / 未指定地址一并拒绝（默认关） | ✅ 已缓解（私网放行为已决策取舍，见 §6.2） |
| **D**oS 拒绝服务 | ➖ 不适用（按威胁域划分）：洪泛类归边界 A；本边界防的是「访问了不该访问的目标」而非可用性 | ➖ |
| **E**levation 提权 | 认证用户可让服务器访问其可达的私网服务——依赖鉴权兜底 + [ADR-003](decisions/0003-ssrf-private-allow.md) 明文记录取舍；严格部署显式开 `S3C_SSRF_DENY_PRIVATE` | ➖ 已决策放行（见 §6.2） |

**生效值核对**：启动日志 `ssrfDenyPrivate` + `/api/metrics` 的 `s3c_ssrf_deny_private`。

### 边界 E：桌面壳（Tauri 2）

| 威胁 | 缓解 | 状态 |
|---|---|---|
| **S**poofing 仿冒 | 纯 B/S：无 IPC、无自定义 command（无 `invoke_handler`，[ADR-001](decisions/0001-desktop-no-ipc.md)），不存在可被仿冒的本地命令面 | ✅ 已缓解 |
| **T**ampering 篡改 | ➖ 不适用：壳层不持有数据、不改写存储（落盘与校验在 Go 后端，见边界 A / C） | ➖ |
| **R**epudiation 抵赖 | ➖ 不适用：壳层无业务操作 | ➖ |
| **I**nfo disclosure 泄露 | 前端 0 处 `v-html` / `innerHTML` / `eval`（XSS 面为零）+ 后端 CSP `script-src 'self'`（`middleware.go` `withSecurityHeaders`） | ✅ 已缓解 |
| **D**oS 拒绝服务 | ➖ 不适用：壳层无服务监听面 | ➖ |
| **E**levation 提权 | capabilities 空权限集（`capabilities/default.json` `permissions: []`，[ADR-001](decisions/0001-desktop-no-ipc.md)）——无 fs / shell / 进程 API 暴露给 JS，权限完全取决于同源策略 + 服务端鉴权 | ✅ 已缓解 |

## 2. 安全默认值

| 默认值 | 位置 |
|---|---|
| 回环绑定 `127.0.0.1:8080` | `config.go` `FromEnv`（`S3C_ADDR` 默认值） |
| 非回环监听强制 `S3C_TOKEN`（否则拒绝启动） | `config.go` `Validate`（`ErrTokenRequiredNonLoopback`） |
| token 最短 16 字符 | `config.go` `MinTokenLength` |
| 可选 token 作用域（`S3C_TOKEN_SCOPES` 非法即拒绝启动；未登记 token 保持全权） | `config.go` `parseTokenScopes` / `validateTokenScopes`、`handler/scope.go` |
| `json`/`sqlite` 无 `S3C_STORE_KEY` 时拒绝启动（需显式 `S3C_ALLOW_PLAINTEXT_STORE=1` 放行） | `config.go` `Validate` |
| `/api/metrics` 与 `/api/openapi.json` 默认 404 | `middleware.go` `withMetricsGate` / `withOpenAPIGate` |
| `/api/metrics` 开启后**不受 token 保护**（有意为内网 scrape；匿名可读，勿直接暴露公网） | `middleware.go` `withAuth` 豁免 |
| `/api/health` 同样免鉴权（探针用途；`version` 暴露为已接受项，见 §6.2） | `middleware.go` `withAuth` 豁免 |
| XFF 默认不信任（`S3C_TRUSTED_PROXIES` 默认空；仅直连对端命中白名单才采信首段） | `config.go` `FromEnv`、`ratelimit.go` `clientIPWithProxies` |
| SSRF 私网 / 回环默认放行（`S3C_SSRF_DENY_PRIVATE` 默认关闭，可显式收紧，见 ADR-003） | `config.go` `FromEnv`、`s3wrap/ssrf.go` |
| CORS 白名单仅 localhost/127.0.0.1/tauri + 跨域 403 硬阻断 | `middleware.go` `corsAllowedOrigin` / `withCORS` |
| 安全头：nosniff / X-Frame-Options DENY / Referrer-Policy / CSP | `middleware.go` `withSecurityHeaders` |
| TLS 站点额外下发 HSTS（180 天）+ Permissions-Policy | `deploy/nginx/conf.d/s3client-tls.example.conf` |
| 容器非 root（`USER app`）、HEALTHCHECK、内存上限 | `apps/server/Dockerfile`、compose |
| ReadHeaderTimeout 15s（防慢速请求头攻击） | `main.go` `runServer`（`http.Server` 字面量） |

## 3. 前端安全

- **Token 存储**：默认 `sessionStorage`（关标签即清）；「跨会话保留」显式开启才写 `localStorage`。
  多服务器 `s3c.servers` 只存 `{id,name,base}` **不落 token**；每服务器 token 按 `s3c.token.<serverId>`
  独立存储、策略同上（原 todo #15「token 明文进 localStorage」已闭环，见 [FEATURES.md](FEATURES.md) §H）。
- **XSS**：全库 0 处 `v-html`；对象名/错误消息全部经 Vue 插值转义。
- **预览管线**：服务端三模式代理（`download` 强制附件 / `inline` 类型白名单 / `text` 强制纯文本）+ `sandbox`。

## 4. 输入校验

| 输入 | 校验 | 位置 |
|---|---|---|
| 桶名 | 3–63 字符 / 仅小写字母数字与 `.-` / 首尾必须字母数字 / 相邻字符不得同为 `.` 或 `-`（禁 `..` `.-` `-.` `--`） | `accounts.go` `validBucketName` |
| 对象 key | 禁控制字符（代理层） | `proxy.go` `proxyObject` |
| ZIP 条目名 | 防 zip-slip（trim + `..`→`_` + path.Clean） | `service/zip.go` `SanitizeZipName` |
| 批量 key | delete ≤1000 / copy ≤10000 / zip ≤1000 / migrate ≤10000 | `objects.go` `maxDeleteKeys`、`handler.go` `maxBatchKeys` / `maxZipKeys` |
| 请求体 | 16MB cap（超限回 413 而非 400）+ `DisallowUnknownFields` + 尾部数据拒绝 | `handler.go` `readJSON` / `maxBody` |
| X-Request-ID | ≤128 可见 ASCII（防日志/响应头注入） | `middleware.go` `validRequestID` / `maxRequestIDLen` |
| 元数据（tags） | key ≤128 / value ≤256 **字节**（`len()` 口径，非按字符计；错误消息文案写作 chars 属既有措辞）、标签数 ≤10、键必填与去重（400 非 500） | `metadata.go` `putObjectTags` |

## 5. 依赖与供应链

- **CI 门禁**：Trivy（容器 OS/库，CRITICAL/HIGH 失败）+ `govulncheck@v1.8.0`（Go 可达漏洞，
  go1.26.6 下 0 告警）+ `cargo audit 0.22.2`（RustSec，桌面依赖；0 漏洞）+
  **OpenSSF Scorecard**（每周六 + 手动，仓库健康度评分发布到 OpenSSF API 与 code scanning，见 §5.5）+
  **Dependency Review**（每个 PR 对依赖 diff 做漏洞 / 许可证审查，见 §5.5）+
  actions 全部 pin 完整 commit SHA（门禁 `TestWorkflowActionsAreShaPinned` 扫描
  `.github/workflows/` 全部 `uses:`；本轮新增两枚 SHA 经 GitHub API 三重核验，2026-09-30：
  `ossf/scorecard-action` v2.4.4 → `2d1146689b8cda280b9bc96326124645441f03bc`、
  `actions/dependency-review-action` v5.0.0 → `a1d282b36b6f3519aa1f3fc636f609c47dddb294`）。
- **Rust 告警 triage**：`cargo audit` 当前 7 条 unmaintained / unsound 告警（`proc-macro-error`、
  5 个 `unic-*`、`glib 0.18.5`），均为上游尚未发布修复版本的传递依赖（tauri/wry 链路），
  不用 `.cargo/audit.toml` ignore 清单掩盖；新增可达漏洞会让 CI 红灯。
- `.trivyignore`：仅策略注释、0 忽略条目（无掩盖性忽略）。
- dependabot：gomod（周）/ npm（周）/ cargo（月）/ actions（月）/ docker（月）。

### 5.1 锁定策略（构建输入的不可变性）

| 输入 | 锁定方式 | 回归门禁 |
|---|---|---|
| Go 依赖 | `apps/server/go.mod` + `go.sum`（含哈希） | `go mod verify` / `govulncheck` |
| 前端依赖 | `pnpm-lock.yaml` + `packageManager: pnpm@9.15.0` + Node 精确 patch | `TestPackageManagerIsPinnedToCIPnpm` / `TestNodePinnedToPatchVersion` |
| 桌面依赖 | `Cargo.lock` + `rust-toolchain.toml`（`channel = "1.98.1"`） | `TestRustToolchainIsVersionPinned` / `cargo audit` |
| GitHub Actions | **全部 pin 到 commit SHA**（豁免 `./` 本地 action 与 `docker://`） | `TestWorkflowActionsAreShaPinned` |
| 容器镜像（Trivy / RustFS） | 按 **digest** pin，不按可变 tag | `TestTrivyImageIsVersionAndDigestPinned` / `TestRustFSImageIsConsistentlyPinned` |
| 运行镜像基础版 | alpine 受支持分支（EOL 即红灯） | `TestRuntimeBaseImageIsSupportedAlpine` |

### 5.2 第三方依赖与许可证清单

全量清单见 [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md)——**由脚本自动生成**
（`./scripts/gen-third-party-licenses.sh`），不从记忆里手写。实测规模（生成时）：

- **Go 43 个模块**：BSD-3-Clause 21 / Apache-2.0 19 / MIT 3——**全部宽松许可**；
- **Rust 428 个 crate**：`MIT OR Apache-2.0` 201、`MIT` 99、`Apache-2.0 OR MIT` 32、…，
  其中 **`MPL-2.0` 5 个**与 **含 `LGPL-2.1-or-later` 的表达式 2 个**属 copyleft 关键词，
  清单顶部已列为「需人工阅读原文」；
- **npm 运行时 1 个**（`vue`，MIT）——前端刻意维持生产依赖仅 `vue`（[ADR-004](decisions/0004-minimal-frontend-deps.md)）。

许可证名是对 `LICENSE` 文本 / `license` 字段的**机械识别**，**不构成法律意见**；识别不出的记为
`UNKNOWN` 并给出文件路径（当前 **0** 项）。覆盖门禁 `TestThirdPartyLicensesAreComplete` 断言
「依赖图里的每个包都出现在清单里」——新增 / 升级依赖而忘记重新生成会**红灯点名**。

### 5.3 消费者如何验证产物（命令取自 CI/CD 工作流本身）

镜像 `ghcr.io/<owner>/<repo>/server`：

| 验证什么 | 命令 |
|---|---|
| 来源证明（绑定 commit / workflow） | `gh attestation verify oci://ghcr.io/<owner>/<repo>/server@<digest> --repo <owner>/<repo>` |
| BuildKit provenance 与 SPDX SBOM（OCI attestation） | `docker buildx imagetools inspect <ref> --format '{{json .Provenance}}'` |
| cosign 无密钥签名 | `cosign verify <ref>@<digest>`——keyless 必须同时指定 `--certificate-oidc-issuer https://token.actions.githubusercontent.com` 与 `--certificate-identity-regexp`（形式同下方桌面端示例，workflow 文件名换成 `ci.yml`） |
| 可下载的 CycloneDX SBOM | CI 产物 `sbom-cyclonedx`（保留 90 天） |

桌面端产物（GitHub Release）：

| 验证什么 | 命令 |
|---|---|
| 文件未被篡改 | `sha256sum -c SHA256SUMS.txt` |
| 来源证明 | `gh attestation verify <安装包路径> --repo <owner>/<repo>` |
| 校验和清单的 cosign 签名 | `cosign verify-blob --certificate SHA256SUMS.txt.pem --signature SHA256SUMS.txt.sig --certificate-oidc-issuer https://token.actions.githubusercontent.com --certificate-identity-regexp '^https://github.com/<owner>/<repo>/.github/workflows/release-desktop.yml@' SHA256SUMS.txt`（`<owner>/<repo>` 换成实际仓库） |

本地复跑审计：`cd apps/server && govulncheck ./...` · `make rust-audit`（cargo audit）· `pnpm audit`（在 `apps/web`）·
Trivy 镜像扫描命令与 CI 同一镜像 digest（见 [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)）。

### 5.4 本层已知缺口（不做假承诺）

- **桌面产物未签名 / 未公证**（Windows SmartScreen、macOS Gatekeeper）——外部凭证阻塞，见
  [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #25；cosign 与 provenance **只能**证明「来源与完整性」，
  **不解决**发布者信誉。
- **Rust 侧 7 条 unmaintained / unsound 告警**维持 triage（见上），不用 ignore 清单掩盖。
- **仓库内不存放 SBOM / 证明文件**：它们是 CI 产物与 registry attestation——入库会让「证据」与
  构建时刻脱钩，核验方式见 §5.3。**本条口径保持不变**：新增的两个 workflow（§5.5）也不产出
  入库 SBOM，Scorecard 的 SARIF 仅以 CI 产物形式保留 5 天。
- **许可证清单不含容器基础镜像的 OS 包**：该部分由 Trivy 扫描与镜像 SBOM 覆盖。
- **GitLab 侧未镜像 Scorecard 与 Dependency Review**：两者是 GitHub 原生产品（依赖
  code scanning API / 依赖图 diff API 与 OIDC 发布通道），GitLab 无等价物——按「未镜像必须
  显式登记」纪律记在 §5.5 与 `DEVELOPMENT.md` §3 对照表；GitLab 侧的 `semgrep-sast`（§7）与
  Dependency Scanning 模板分别是「仓库内代码 SAST」与「发布时点全量扫描」，**不等于**这里的
  仓库健康度评分与 PR 依赖 diff 审查。
- **Scorecard 的 `workflow_dispatch` 触发属上游标记的实验性支持**；其 `Branch-Protection` 检查项
  依赖仓库自身的规则集 / 分支保护配置，未配置时表现为该项低分而不是流水线失败。
- **Dependency Review 只覆盖「PR 内变更的依赖」**：发布时点的全量漏洞门禁仍是 Trivy /
  govulncheck / cargo audit（见 §5 开头），两者互补不互替。

### 5.5 GitHub 原生供应链 workflow（Scorecard + Dependency Review）

2026-09-30 起新增两个 GitHub 原生供应链 workflow，补齐此前「仓库内部门禁齐全、但仓库外部
健康度评分与 PR 期依赖审查缺失」的两块空白。两者与既有 CodeQL 的分工：CodeQL 扫**仓库内代码**，
Scorecard 评**仓库整体健康度**，Dependency Review 审**PR 的依赖 diff**。

**① [`../.github/workflows/scorecard.yml`](../.github/workflows/scorecard.yml)（OpenSSF Scorecard）**

- **触发**：`schedule` 每周六 02:00 UTC（与 codeql 周日、三个 E2E 周一/三/五错开）+ `workflow_dispatch`
  （官方模板另支持 `push` 与 `branch_protection_rule`，本仓库未加——避免每次 push 都跑一条慢任务；
  `workflow_dispatch` 在上游 README 标记为实验性支持，见 §5.4）。
- **步骤**（官方模板口径）：checkout（`persist-credentials: false`）→ `ossf/scorecard-action`
  （`results_format: sarif` + `publish_results: true`）→ `actions/upload-artifact`（SARIF 产物，
  保留 5 天）→ `github/codeql-action/upload-sarif`（进 Security 页 code scanning 面板）。
- **权限**：顶层 `permissions: read-all`；仅 `analysis` job 持 `security-events: write`（上传 SARIF）
  与 `id-token: write`（`publish_results: true` 必需的 OIDC 令牌，结果发布到 api.scorecard.dev，
  公开仓库可挂 Scorecard badge）。这组口径同时满足上游「发布结果」的工作流限制：顶层无 write
  权限、无顶层 env/defaults、仅 scorecard job 可持 `id-token`。
- **只评分、不设阈值门禁**：与 CodeQL 同属报告型——分数与告警看 Security 面板；阈值型门禁仍是
  两套 CI 各自的 Trivy（口径见 `DEVELOPMENT.md` §3）。

**② [`../.github/workflows/dependency-review.yml`](../.github/workflows/dependency-review.yml)（PR 依赖审查）**

- **触发**：每个 `pull_request`（含 dependabot 自动 PR）——依赖审查的价值恰在「变更进入 main
  之前」，故不设 `paths` 过滤。
- **步骤**：checkout → `actions/dependency-review-action`（v5；默认 `fail-on-severity: low`、
  `fail-on-scopes: runtime`、`license-check: true`——与 Trivy 的 CRITICAL/HIGH 阈值不同是**有意的**：
  它只对 PR **引入的**依赖 diff 生效，更严的默认口径成本更低；许可证识别不出只会告知、不失败）。
- **权限**：`permissions: contents: read`（仅此一项；默认不往 PR 发评论，故无需 `pull-requests: write`）。
- **运行环境**：v5 运行时为 node24，要求 runner ≥ 2.327.1（`ubuntu-latest` 已满足）；依赖 GitHub
  依赖图 API，公开仓库直接可用。

**与 GitLab 侧差异（未镜像，显式登记）**：两个 workflow 都是 GitHub 原生能力，GitLab 没有等价
原生产品——GitLab 的 SAST（`semgrep-sast`，见 §7）是仓库内代码扫描，Dependency Scanning 模板是
发布时点全量扫描，都不等于「仓库外部健康度评分」或「PR 依赖 diff 审查」。按 `release-desktop.yml` /
`codeql.yml` 的既有先例，「未镜像必须显式登记」：登记行在 `DEVELOPMENT.md` §3「CI 双平台一致性」
对照表（GitLab 列写「不镜像」及理由）；两侧共同保持的阈值门禁一致（Trivy / govulncheck / cargo audit）。

## 6. 已知风险

### 6.1 已闭环（证据归档）

> 逐项证据见 [FEATURES.md](FEATURES.md) 与 [CHANGELOG.md](../CHANGELOG.md)；原始发现见
> [archive/assessment.md](archive/assessment.md) §二。当前待办见 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)（缺陷 / 技术债）与
> [`ROADMAP.md`](ROADMAP.md) §三 3.2（安全方向候选 #12 / #13）——
> 安全 / 供应链类现无 ⬜/⏳ 未闭环项；#18（health 暴露 version）已于 2026-09-23 复审维持现状并移出待办台账（见 §6.2）。

- ~~Go 1.26.5 → 1.26.6（6 个可达 stdlib CVE）~~ ✅ 已升级 1.26.6 + `govulncheck` CI 门禁
- ~~SQLite 明文密钥~~ ✅ 已修：设 `S3C_STORE_KEY` 时 secret_key 列加密；`S3C_STORE_KEY` 最短 16 字符
- ~~安全审计日志缺失~~ ✅ 已闭环：`handler/audit.go`（原 KNOWN_ISSUES #17，见 [FEATURES.md](FEATURES.md) §M）
- ~~XFF 伪造绕过限速~~ ✅ 已闭环：`S3C_TRUSTED_PROXIES` 可信代理白名单（默认不信任 XFF）；JobRegistry 上限 256 个未终结任务，超限 503
- ~~TLS 前置无 HSTS~~ ✅ 已在 TLS 示例配置加 HSTS + Permissions-Policy

### 6.2 仍接受的风险（有意维持，非缺陷）

- `/api/health` 暴露 `version`（KNOWN_ISSUES #18，2026-09-23 复审维持；清单条目已按「只收录尚未完成」移出）：开源项目版本与依赖本就公开、指纹价值≈0，而该响应是安全补丁验证与运维定位的廉价通道；`withAuth` **显式跳过**该端点（Docker HEALTHCHECK 需无 token 探测），故其为免鉴权端点、不依赖鉴权兜底。移除属行为变更。
- `/api/metrics` 开启后免鉴权（内网 scrape 用途，见 §2；勿直接暴露公网）。
- SSRF 默认放行私网 / 回环（自托管刚需，[ADR-003](decisions/0003-ssrf-private-allow.md)；严格部署用 `S3C_SSRF_DENY_PRIVATE=1` 收紧）。
- `S3C_ALLOW_PLAINTEXT_STORE=1` 可放行明文 store（仅限本地联调；生产必须 `encrypted` 或 `sqlite` + key，见「边界 C」）。

---

## 7. SAST 发现项 triage（GitLab `semgrep-sast`）

> GitLab 侧 SAST 自 2026-09-29 起由 `include: template: Jobs/SAST.gitlab-ci.yml` 引入
> （对应 GitHub 侧 `codeql.yml`）。**它是报告型而非阈值门禁**——`semgrep-sast` 继承模板
> `.sast-analyzer` 的 `allow_failure: true`，发现项本身不会让流水线变红。
> 因此**本节就是它的「处置账」**：不写下来，那份 `gl-sast-report.json` 就只是跑一次没人看的产物。

实跑命令（本地，无需 GitLab 实例）：

```bash
make gcl GCL_JOBS=semgrep-sast     # 产物 gl-sast-report.json（已在 .gitignore）
```

**扫描面**：默认的 `$DEFAULT_SAST_EXCLUDED_PATHS`（`spec, test, tests, tmp`）是**目录名**式的，
与本仓库命名约定不匹配，故 `.gitlab-ci.yml` 的 `semgrep-sast` job 显式追加
`**/*_test.go` / `**/*.test.ts` / `apps/web/e2e{, -real}` / `apps/web/coverage`。
效果：**66 项 → 10 项**（4 High + 6 Medium）——被排除的 56 项里 50 项在测试代码、
6 项在**生成的** Istanbul HTML 报告资产（第三方 `prettify.js` / `sorter.js`）中，
二者都不是本仓库的生产代码（与覆盖率 instrumentation 排除 `*.test.ts` 同一口径）。

**当前 10 项全部判定为「误报 / 设计使然」**，逐条如下（改动相关代码时请连带复核）：

| # | 严重度 | 位置 | 规则 | 判定与依据 |
|---|---|---|---|---|
| 1 | High | `internal/store/sqlite.go`（`PRAGMA user_version`） | SQL Injection | **误报**：`fmt.Sprintf` 的插值是**编译期常量** `sqliteUserVersion`（int 常量），无任何用户输入进入该语句；`PRAGMA` 也不接受参数绑定，故不是「该用占位符却拼接」 |
| 2–4 | High | `api/http.ts` ×2、`api/jobs.ts` ×1 | SSRF | **设计使然**：`fetch(getBase() + path)` 的 `getBase()` 是**用户自己配置的后端地址**——「多后端」就是这个产品的功能。请求由**用户自己的浏览器**发往**用户自己指定的服务器**，不构成服务端 SSRF；服务端侧的出站校验另见「边界 D」与 `s3wrap/ssrf.go` |
| 5–7 | Medium | `handler/trash.go`、`handler/metadata.go`、`handler/objects.go` | Integer overflow | **误报**：三处都是 `int32(n)`，而 **紧邻上一行**就是范围守卫 `n > 0 && n <= 1000`，不可能溢出。该规则不做区间追踪，看不到前置 guard |
| 8 | Medium | `internal/store/open.go`（`ensureDataDirPerm`） | Incorrect permission assignment | **误报（且方向相反）**：`os.Chmod(dir, 0o700)` 是**主动加固**——`os.MkdirAll` 的 mode 只在新建时生效，预建目录（Docker volume / systemd StateDirectory）常为 0755，故此处收紧到属主独占 |
| 9 | Medium | `internal/store/lock_unix.go` | Path Traversal | **误报**：拼接的是**常量**文件名 `.s3client.lock`，目录来自 `S3C_DATA_DIR`（运维配置，非用户输入） |
| 10 | Medium | `api/storage.ts`（`newId`） | Weak PRNG | **可接受**：主路径是 `crypto.randomUUID()`（CSPRNG），`Math.random()` 仅在**前者不可用**时兜底；且该 id 是本地服务器条目的标识，**不是密钥**（token 另存 `s3c.token.<id>`，且匿名可写的 localStorage 早已是失陷前提） |

**为什么不做规则级抑制**：`.gitlab/sast-ruleset.toml` 与 `SAST_RULESET_GIT_REFERENCE` 属
**Ultimate 专属**，本项目按 Free 档设计，故没有「集中的 ignore 文件」可用（Trivy 的
`.trivyignore` 那种做法在这里不可移植）；在代码里逐行加 `// nosemgrep: <rule-id>` 虽全档可用，
但会把工具专有注释写进生产代码，收益不抵。**折中即本节**：把判定记在案，让下一个人不必重做，
同时保持「新增发现项 = 需要重新 triage」的可见性。

> ⚠️ **本节随扫描面变化而失效**：一旦新代码引入新的 High/Medium，或上游规则集更新，
> 上表就不再完整。判断方法：重跑上面的命令，比对 `gl-sast-report.json` 与本表的条数。
