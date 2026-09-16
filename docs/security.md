# 安全设计

> 本文档描述 s3clinet 的威胁模型、安全边界与默认值。漏洞报告流程见 [SECURITY.md](../SECURITY.md)。
> 本文档基于 2026-09-16 综合安全审计（详见 [ASSESSMENT.md](ASSESSMENT.md) §二）。

## 1. 威胁模型（STRIDE × 边界）

### 边界 A：HTTP API（Bearer 鉴权）

| 威胁 | 缓解 | 状态 |
|---|---|---|
| **S**poofing 仿冒 | Bearer 常量时间比较（sha256+subtle）、scheme 大小写不敏感（RFC 7235）、多 token 轮换、最短 16 字符 | ✅ 已缓解 |
| **T**ampering 篡改 | 强制 `application/json` + `DisallowUnknownFields` + 8MB body cap；配合 CORS 使跨域变更必预检 | ✅ 已缓解 |
| **R**epudiation 抵赖 | ⚠️ 仅通用 access log，**无安全审计日志**（todo #17） | ❌ 未缓解 |
| **I**nfo disclosure 泄露 | 错误脱敏、S3 错误稳定映射、AccountView 不含 secretKey | ✅ 已缓解 |
| **D**oS 拒绝服务 | IP 令牌桶 120/min、流式并发 32、批量上限齐备 | ⚠️ 部分缓解（XFF 伪造、job 无上限，todo #17） |
| **E**levation 提权 | 单 token 模型无角色；token 轮换支持 | ✅ 已缓解 |

### 边界 B：预签名 URL 直传

- v4 签名由 AWS SDK 生成，过期钳制 [1h, 24h]（`objects.go:433`）。
- SecretKey 永不回传浏览器，只回预签名 URL。
- 直传流量不经本服务（见 [architecture.md](architecture.md) §5）。

### 边界 C：账号存储（SecretKey 落盘）

| 驱动 | 落盘 | 说明 |
|---|---|---|
| `json` | 明文 JSON 或 S3C2 加密 | 配 `S3C_STORE_KEY` 时 AES-256-GCM + Argon2id；permissive 兼容读旧明文 |
| `sqlite` | **secret_key 明文** | ⚠️ 生产勿用或配合磁盘级加密（todo #16） |
| `encrypted` | S3C2 加密（严格） | ✅ 生产推荐；文件盐建时随机并复用 |

所有驱动：原子写（临时文件 + rename）+ 0600 权限 + 写失败回滚内存。

### 边界 D：S3 上游（SSRF）

- **双重校验**：创建时 `ValidateEndpoint` + 拨号期 `dialContextSSRF` 二次解析逐 IP 校验（消除 DNS 重绑定 TOCTOU）。
- 拦截：链路本地、阿里 IMDS（100.100.100.200 / 100.96.0.2）、AWS IMDS IPv6（fd00:ec2::254）、GCP metadata 主机名。
- 禁重定向 + 禁 HTTP(S)_PROXY（封环境变量代理绕过）。
- 无 `InsecureSkipVerify`。
- **设计取舍**：私网/回环放行（自托管主场景，[ADR-003](decisions/0003-ssrf-private-allow.md)）。

### 边界 E：桌面壳（Tauri 2）

- 纯 B/S 无 IPC、无 command；capabilities 空权限集（[ADR-001](decisions/0001-desktop-no-ipc.md)）。
- 前端 0 处 `v-html`/`innerHTML`/`eval`（XSS 面为零）。

## 2. 安全默认值

| 默认值 | 位置 |
|---|---|
| 回环绑定 `127.0.0.1:8080` | `config.go:120` |
| 非回环监听强制 `S3C_TOKEN`（否则拒绝启动） | `config.go:184-186` |
| token 最短 16 字符 | `config.go:180` |
| `/api/metrics` 与 `/api/openapi.json` 默认 404 | `middleware.go:177-199` |
| CORS 白名单仅 localhost/127.0.0.1/tauri + 跨域 403 硬阻断 | `middleware.go:105-172` |
| 安全头：nosniff / X-Frame-Options DENY / Referrer-Policy / CSP | `middleware.go:17-28` |
| 容器非 root（USER app）、HEALTHCHECK、内存上限 | `Dockerfile`、compose |
| ReadHeaderTimeout 15s（防慢速请求头攻击） | `main.go:74` |

## 3. 前端安全

- **Token 存储**：默认 `sessionStorage`（关标签即清）；「跨会话保留」显式开启才写 `localStorage`。
  ⚠️ 已知问题：多服务器 `s3c.servers` 把 token 明文写入 `localStorage`（todo #15，待修复）。
- **XSS**：全库 0 处 `v-html`；对象名/错误消息全部经 Vue 插值转义。
- **预览管线**：服务端三模式代理（`download` 强制附件 / `inline` 类型白名单 / `text` 强制纯文本）+ `sandbox`。

## 4. 输入校验

| 输入 | 校验 | 位置 |
|---|---|---|
| 桶名 | 长度/首尾字符/连续 `..`/`.-`/`-.` | `accounts.go:171-190` |
| 对象 key | 禁控制字符（代理层） | `proxy.go:36-41` |
| ZIP 条目名 | 防 zip-slip（trim + `..`→`_` + path.Clean） | `zip.go:149-168` |
| 批量 key | delete ≤1000 / copy ≤10000 / zip ≤1000 / migrate ≤10000 | `handler.go:118-122` 等 |
| 请求体 | 8MB cap + `DisallowUnknownFields` + 尾部数据拒绝 | `handler.go:161-177` |
| X-Request-ID | ≤128 可见 ASCII（防日志/响应头注入） | `middleware.go:35-50` |
| 元数据 | 键值长度/字节总长边界（400 非 500） | `metadata.go:151-163` |

## 5. 依赖与供应链

- **CI 门禁**：Trivy（容器 OS/库，CRITICAL/HIGH 失败）+ `govulncheck@v1.8.0`（Go 可达漏洞，
  go1.26.6 下 0 告警）+ actions 全部 pin SHA（10 个 SHA 经 GitHub API 核验有效）。
- `.trivyignore`：空清单（无掩盖性忽略）。
- dependabot：gomod（周）/ npm（周）/ cargo（月）/ actions（月）/ docker（月）。

## 6. 已知风险与路线

> 完整清单见 [todolist.md](todolist.md)「四、安全 / 供应链待办」与 [ASSESSMENT.md](ASSESSMENT.md) §二。

- ~~Go 1.26.5 → 1.26.6（6 个可达 stdlib CVE）~~ ✅ 已升级 1.26.6 + `govulncheck` CI 门禁
- SQLite 明文密钥（生产改 encrypted）
- 安全审计日志缺失
- XFF 伪造绕过限速 / JobRegistry 无上限
- TLS 前置无 HSTS
