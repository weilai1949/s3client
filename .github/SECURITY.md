# 安全策略（Security Policy）

## 支持的版本

当前版本以 [`Makefile`](../Makefile) 的 `VERSION` 与 `GET /api/health` 的 `version` 字段为准（由 [`scripts/release-version.sh`](../scripts/release-version.sh) 同步，**本文件刻意不 pin 具体版本号**以免发版后漂移）；稳定里程碑之后日常发版用时间戳版 `v1.0.0-YYYYMMDDHHmmss`（命名约定见 [README.md](../README.md)）。仅维护最新版本，安全修复随下一个版本发布。

| 版本 | 支持状态 |
|---|---|
| 最新发布（纯 semver 里程碑，或时间戳版如 `v1.0.0-YYYYMMDDHHmmss`） | ✅ 支持 |
| 预发布（`-rc*` / `-alpha*` / `-beta*`） | ❌ 已被对应正式版取代，请升级 |
| `0.x`（历史） | ❌ 不再支持，请升级 |

## 报告漏洞

**请不要公开披露漏洞**（不要开 public issue）。请通过以下任一渠道私下报告：

- **GitHub 私有漏洞报告**（**推荐，唯一保证可达的渠道**）：仓库页面 → `Security` → `Report a vulnerability`

> 本仓库**未公开安全邮箱**：此前这里指向「[CONTRIBUTING.md](CONTRIBUTING.md) 中列出的维护者邮箱」，但该文件从未列出过邮箱（死链，已于 2026-09-29 移除）。维护者联系方式与其它支持渠道见 [CONTRIBUTING.md §联系与支持](CONTRIBUTING.md#联系与支持)。

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
4. 定期升级到最新版本，跟随 [dependabot](https://github.com/weilai1949/s3client/security/dependabot) 与 CI 的 Trivy / govulncheck 门禁
