# 安全策略（Security Policy）

## 支持的版本

当前版本 `v1.0.0`；稳定里程碑之后日常发版用时间戳版 `v1.0.0-YYYYMMDDHHmmss`（命名约定见 [README.md](../README.md)）。仅维护最新版本，安全修复随下一个版本发布。

| 版本 | 支持状态 |
|---|---|
| `v1.0.0` 及时间戳版 `v1.0.0-*`（最新） | ✅ 支持 |
| `v1.0.0-rc*`（预发布） | ❌ 已被 `v1.0.0` 取代，请升级 |
| `0.x`（历史） | ❌ 不再支持，请升级 |

## 报告漏洞

**请不要公开披露漏洞**（不要开 public issue）。请通过以下任一渠道私下报告：

- **GitHub 私有漏洞报告**：仓库页面 → `Security` → `Report a vulnerability`
- **邮件**：维护者邮箱（见 [CONTRIBUTING.md](CONTRIBUTING.md)）

请在你的报告中包含：

1. 漏洞类型与严重程度评估
2. 受影响的版本与配置
3. 可复现步骤（PoC 优先）
4. 影响范围与可能的缓解

我们承诺：

- 48 小时内确认收到报告
- 评估后告知修复计划与时间表
- 修复发布后，若你同意，会在 [CHANGELOG.md](../CHANGELOG.md) 中致谢

## 安全设计要点

本项目的安全模型，详见 [threat-model.md](../docs/threat-model.md)。关键默认值：

- **默认回环绑定** `127.0.0.1:8080`；非回环监听必须配置 `S3C_TOKEN`（否则拒绝启动）
- **Bearer 鉴权**：常量时间比较、多 token 轮换、最短 16 字符
- **SSRF 防护**：创建时 + 拨号期双重校验（禁云元数据/链路本地）、禁重定向、禁代理
- **CSRF 防护**：CORS 白名单外 Origin 直接 403 + 请求体 `Content-Type` **非空**时必须为 `application/json`（缺省放行，仍受 JSON 解码器约束，见 [threat-model.md](../docs/threat-model.md) 边界 A）
- **密钥存储**：`encrypted` 驱动 AES-256-GCM + Argon2id；响应不回传 `secretKey`
- **指标/契约默认隐藏**：`/api/metrics` 与 `/api/openapi.json` 默认 404

## 部署安全建议

1. **生产必须启用鉴权**：`S3C_TOKEN`（`openssl rand -hex 32`）
2. **生产推荐 `encrypted` 存储驱动**：`S3C_STORE_DRIVER=encrypted` + `S3C_STORE_KEY`（整库加密）。`json` / `sqlite` 配同一个 key 时也会加密落盘（`sqlite` 加密 `secret_key` 列，其余列仍为明文），而**无 key 时进程拒绝启动**——`S3C_ALLOW_PLAINTEXT_STORE=1` 仅限本地联调（详见 [threat-model.md](../docs/threat-model.md) 边界 C）
3. **经反向代理 + TLS 对外暴露**，并配置 HSTS（见 [deploy/nginx/README.md](../deploy/nginx/README.md)）
4. 定期升级到最新版本，跟随 [dependabot](https://github.com/weilai1949/s3clinet/security/dependabot) 与 CI 的 Trivy / govulncheck 门禁
