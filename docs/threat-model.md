# 安全设计

> 本文档描述 s3clinet 的威胁模型、安全边界与默认值。漏洞报告流程见 [SECURITY.md](../.github/SECURITY.md)。
> 本文档基于 2026-09-16 综合安全审计（详见 [assessment.md](assessment.md) §二），其后按修复进展滚动更新。
> 最后更新：2026-09-22。

## 1. 威胁模型（STRIDE × 边界）

> 状态图例：✅ 已缓解 · ⚠️ 部分缓解 · ❌ 未缓解 · ➖ 不适用 / 已决策放行（理由见「缓解」列）。

### 边界 A：HTTP API（Bearer 鉴权）

| 威胁 | 缓解 | 状态 |
|---|---|---|
| **S**poofing 仿冒 | Bearer 常量时间比较（sha256+subtle）、scheme 大小写不敏感（RFC 7235）、多 token 轮换、最短 16 字符 | ✅ 已缓解 |
| **T**ampering 篡改 | Content-Type 非空时必须为 `application/json`（缺省放行，仍受 JSON 解码器约束）+ `DisallowUnknownFields` + 16MB body cap（超限回 413）；配合 CORS 使跨域变更必预检 | ✅ 已缓解 |
| **R**epudiation 抵赖 | 安全审计日志（`handler/audit.go`）：401（malformed/bad_token）/ 账号 CRUD / 桶策略设置与清除 / 对象删除与前缀删除 / 回收站清空 / 限速命中，带固定事件名与操作者 IP；另有通用 access log | ✅ 已缓解（原 todo #17，闭环见 [features.md](features.md) §M） |
| **I**nfo disclosure 泄露 | 错误脱敏、S3 错误稳定映射、AccountView 不含 secretKey | ✅ 已缓解 |
| **D**oS 拒绝服务 | IP 令牌桶 120/min、流式并发 32、批量上限齐备；XFF 仅 `S3C_TRUSTED_PROXIES` 命中才采信（默认空 = 不信任，防伪造绕过限速）；在册任务 ≤256 超限 503 | ✅ 已缓解（原 todo #17 两项缺口均已闭环） |
| **E**levation 提权 | 单 token 模型无角色；token 轮换支持 | ✅ 已缓解 |

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
| **I**nfo disclosure 泄露 | 核心威胁：配 `S3C_STORE_KEY` 时 AES-256-GCM + Argon2id（文件盐建时随机并复用，key 最短 16 字符）；数据目录 0700 + 文件 0600；json / sqlite 无 key 拒绝启动（`S3C_ALLOW_PLAINTEXT_STORE=1` 仅限联调，见下） | ✅ 已缓解 |
| **D**oS 拒绝服务 | ➖ 不适用：本地文件无网络面；可用性按 ADR-002 硬失败不降级 | ➖ |
| **E**levation 提权 | 文件 0600 仅属主可读写 + 容器非 root 运行 | ✅ 已缓解 |

**落盘明细（驱动与格式）**：

| 驱动 | 落盘 | 说明 |
|---|---|---|
| `json` | 明文 JSON 或 S3C3 加密 | 配 `S3C_STORE_KEY` 时 AES-256-GCM + Argon2id（参数随文件版本）；permissive 兼容读旧明文与旧 S3C2 |
| `sqlite` | secret_key 列明文或 S3C3 加密 | 配 `S3C_STORE_KEY` 时该列以 AES-256-GCM 密文落盘；历史明文行仍可读、写回即加密（已闭环，证据见 [features.md](features.md) §M；残留风险见 [roadmap.md](roadmap.md) §5.1「已收敛」索引 R3） |
| `encrypted` | S3C3 加密（严格） | ✅ 生产推荐；文件盐建时随机并复用；可读旧 S3C2 库 |

所有驱动：原子写（临时文件 + rename）+ 0600 权限 + 写失败回滚内存。**未配置 `S3C_STORE_KEY` 的
`json` / `sqlite` 驱动会拒绝启动**（安全默认；#29/#31 已闭环，证据见 [features.md](features.md) §T），除非显式设置
`S3C_ALLOW_PLAINTEXT_STORE=1`——此时允许运行并打出「secretKey 将明文落盘」WARN
（[roadmap.md](roadmap.md) §5.1「已收敛」索引 R3），仅限本地联调；生产必须用 `encrypted` 或
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
| `json`/`sqlite` 无 `S3C_STORE_KEY` 时拒绝启动（需显式 `S3C_ALLOW_PLAINTEXT_STORE=1` 放行） | `config.go` `Validate` |
| `/api/metrics` 与 `/api/openapi.json` 默认 404 | `middleware.go` `withMetricsGate` / `withOpenAPIGate` |
| `/api/metrics` 开启后**不受 token 保护**（有意为内网 scrape；匿名可读，勿直接暴露公网） | `middleware.go` `withAuth` 豁免 |
| `/api/health` 同样免鉴权（探针用途；`version` 暴露为已接受项，见 §6.2） | `middleware.go` `withAuth` 豁免 |
| XFF 默认不信任（`S3C_TRUSTED_PROXIES` 默认空；仅直连对端命中白名单才采信首段） | `config.go` `FromEnv`、`ratelimit.go` `clientIPWithProxies` |
| SSRF 私网 / 回环默认放行（`S3C_SSRF_DENY_PRIVATE` 默认关闭，可显式收紧，见 ADR-003） | `config.go` `FromEnv`、`s3wrap/ssrf.go` |
| CORS 白名单仅 localhost/127.0.0.1/tauri + 跨域 403 硬阻断 | `middleware.go` `corsAllowedOrigin` / `withCORS` |
| 安全头：nosniff / X-Frame-Options DENY / Referrer-Policy / CSP | `middleware.go` `withSecurityHeaders` |
| TLS 站点额外下发 HSTS（180 天）+ Permissions-Policy | `deploy/nginx/conf.d/s3clinet-tls.example.conf` |
| 容器非 root（`USER app`）、HEALTHCHECK、内存上限 | `apps/server/Dockerfile`、compose |
| ReadHeaderTimeout 15s（防慢速请求头攻击） | `main.go` `runServer`（`http.Server` 字面量） |

## 3. 前端安全

- **Token 存储**：默认 `sessionStorage`（关标签即清）；「跨会话保留」显式开启才写 `localStorage`。
  多服务器 `s3c.servers` 只存 `{id,name,base}` **不落 token**；每服务器 token 按 `s3c.token.<serverId>`
  独立存储、策略同上（原 todo #15「token 明文进 localStorage」已闭环，见 [features.md](features.md) §H）。
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
  go1.26.6 下 0 告警）+ `cargo audit 0.22.2`（RustSec，桌面依赖；0 漏洞）+ actions 全部 pin SHA
  （12 个 SHA 经 GitHub API 核验有效，2026-09-22 复验均 200）。
- **Rust 告警 triage**：`cargo audit` 当前 7 条 unmaintained / unsound 告警（`proc-macro-error`、
  5 个 `unic-*`、`glib 0.18.5`），均为上游尚未发布修复版本的传递依赖（tauri/wry 链路），
  不用 `.cargo/audit.toml` ignore 清单掩盖；新增可达漏洞会让 CI 红灯。
- `.trivyignore`：仅策略注释、0 忽略条目（无掩盖性忽略）。
- dependabot：gomod（周）/ npm（周）/ cargo（月）/ actions（月）/ docker（月）。

## 6. 已知风险

### 6.1 已闭环（证据归档）

> 逐项证据见 [features.md](features.md) 与 [CHANGELOG.md](../CHANGELOG.md)；原始发现见
> [assessment.md](assessment.md) §二。当前待办见 [todolist.md](todolist.md)「四、安全 / 供应链待办」——
> 该节现无 ⬜/⏳ 未闭环项；#18（health 暴露 version）已于 2026-09-23 复审维持现状并移出 todolist（见 §6.2）。

- ~~Go 1.26.5 → 1.26.6（6 个可达 stdlib CVE）~~ ✅ 已升级 1.26.6 + `govulncheck` CI 门禁
- ~~SQLite 明文密钥~~ ✅ 已修：设 `S3C_STORE_KEY` 时 secret_key 列加密；`S3C_STORE_KEY` 最短 16 字符
- ~~安全审计日志缺失~~ ✅ 已闭环：`handler/audit.go`（原 todo #17，见 [features.md](features.md) §M）
- ~~XFF 伪造绕过限速~~ ✅ 已闭环：`S3C_TRUSTED_PROXIES` 可信代理白名单（默认不信任 XFF）；JobRegistry 上限 256 个未终结任务，超限 503
- ~~TLS 前置无 HSTS~~ ✅ 已在 TLS 示例配置加 HSTS + Permissions-Policy

### 6.2 仍接受的风险（有意维持，非缺陷）

- `/api/health` 暴露 `version`（原 todolist #18，2026-09-23 复审维持；清单条目已按「只收录尚未完成」移出）：开源项目版本与依赖本就公开、指纹价值≈0，而该响应是安全补丁验证与运维定位的廉价通道；`withAuth` **显式跳过**该端点（Docker HEALTHCHECK 需无 token 探测），故其为免鉴权端点、不依赖鉴权兜底。移除属行为变更。
- `/api/metrics` 开启后免鉴权（内网 scrape 用途，见 §2；勿直接暴露公网）。
- SSRF 默认放行私网 / 回环（自托管刚需，[ADR-003](decisions/0003-ssrf-private-allow.md)；严格部署用 `S3C_SSRF_DENY_PRIVATE=1` 收紧）。
- `S3C_ALLOW_PLAINTEXT_STORE=1` 可放行明文 store（仅限本地联调；生产必须 `encrypted` 或 `sqlite` + key，见「边界 C」）。
