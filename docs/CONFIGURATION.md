# 配置参考

> 服务端环境变量与客户端设置的**单一事实来源（SSOT）**。改配置项时本文件与
> [`.env.example`](../.env.example)、[`apps/server/.env.example`](../apps/server/.env.example)、
> [`DEPLOYMENT.md`](DEPLOYMENT.md) 属同一个 PR（见 [DEVELOPMENT.md](DEVELOPMENT.md) §4「文档同步门禁」）。
> 真值来源：`apps/server/internal/config/config.go` 的 `FromEnv` / `Validate`。
>
> **两份 `.env.example` 的口径分工**（避免「同名两份、内容各半」的漂移）：
> - 根 [`.env.example`](../.env.example) = **compose / 本地联调**变量：compose 实际透传的
>   `${S3C_*}` 与 RustFS 联调凭据，**只列这些**——因此它**不**包含本节全部可选项；
> - [`apps/server/.env.example`](../apps/server/.env.example) = **服务端全量可选项**示例（含
>   `S3C_EXPOSE_METRICS` / `S3C_EXPOSE_OPENAPI` / `S3C_CSP_CONNECT_SRC` 等）。
>
> 「同 PR 同步」指**相应的那一份**：新增 compose 透传项改根文件，新增服务端可选项改
> `apps/server/.env.example`，两者语义不同故不互相复制。

## 1. 配置来源与优先级

服务端全部配置经环境变量注入，也可写进 `.env` 文件。**真实环境变量始终优先于文件**
（`loadDotEnvFile` 只在该变量尚未存在于进程中时才写入）。

`.env` 查找顺序：

| 优先级 | 路径 | 说明 |
|---|---|---|
| 1 | `S3C_ENV_FILE` 指定路径 | 设置后即为**唯一**来源（不再回退其它候选）。绝对路径可与进程 CWD 解耦，适合 systemd / 容器 |
| 2 | 进程工作目录 `.env` | 历史默认，本地开发习惯 |
| 3 | 可执行文件同目录 `.env` | systemd / 双击启动时 CWD 往往不是安装目录 |

`.env` 解析规则：忽略空行与 `#` 注释行；按第一个 `=` 切分；值两侧的成对单/双引号会被剥除。

> **只有显式 `S3C_ENV_FILE` 是 fail-closed 的**：该路径不存在 / 不可读时进程**拒绝启动**
> （`ErrInvalidEnvFile`），不会静默回退默认值。否则写在该文件里的加固项（`S3C_TOKEN`、
> `S3C_SSRF_DENY_PRIVATE`、`S3C_TRUSTED_PROXIES`、`S3C_CSP_CONNECT_SRC`…）会静默失效，
> `S3C_DATA_DIR` 更会静默丢失导致账号列表「凭空清空」。未显式指定时候选缺失直接跳过，
> 保持零配置可启动。

## 2. 服务端环境变量

布尔项一律接受 `1` / `true` / `yes` / `on`（大小写与首尾空白不敏感），其余值视为关闭。

| 变量 | 默认值 | 取值 / 约束 | 说明 |
|---|---|---|---|
| `S3C_ENV_FILE` | 空 | 路径 | 显式指定 `.env` 路径；设置后为唯一来源，且路径不存在 / 不可读时**拒绝启动** |
| `S3C_ADDR` | `127.0.0.1:8080` | `host:port` | 监听地址。回环更安全；需远程访问改 `0.0.0.0:8080`，此时**必须**同时设 `S3C_TOKEN` |
| `S3C_DATA_DIR` | `./data` | 目录路径 | 数据目录：`accounts.json` / `accounts.db` / `accounts.json.enc`，任务清单 `jobs.json`，上次关停耗时 `shutdown.json`，以及单写者锁文件 `.s3clinet.lock` |
| `S3C_STATIC_DIR` | `../web/dist` | 目录路径（相对进程 CWD） | Web 静态资源目录；`make server` / `cd apps/server` 启动时指向 `apps/web/dist` |
| `S3C_REGION` | `us-east-1` | 区域字符串 | 账号缺省 region（账号可单独覆盖） |
| `S3C_TOKEN` | 空 | ≥ 16 字符；逗号分隔可多值 | 非空时所有 `/api/*` 需 `Authorization: Bearer <token>`（`/api/health`、`/api/metrics` 豁免）。非回环监听时**必填**。多 token 轮换时按**最短者**判定长度；删掉旧值即吊销。生成：`openssl rand -hex 32` |
| `S3C_CORS_ORIGINS` | 空 | 逗号分隔 Origin 列表 | CORS 白名单。留空 = 仅同源 + `localhost` / `127.0.0.1` / `[::1]` / `tauri.localhost`（含 `tauri` 自定义协议）。`*` 可行但不推荐 |
| `S3C_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` | 日志级别 |
| `S3C_LOG_JSON` | 关 | 布尔 | 开 = `slog` JSON 输出（容器 / 生产更易采集）。compose 已默认注入 `1`，设 `0` 可退回纯文本 |
| `S3C_SHUTDOWN_TIMEOUT` | `30` | 整数秒，**1–3600** | 收到 `SIGTERM` 后等待活跃连接结束的最长时间。超上界拒绝启动（秒数过大会让 `time.Duration` 溢出为负时长，优雅关停被静默跳过） |
| `S3C_STORE_DRIVER` | `json` | `json` / `sqlite` / `encrypted` | 账号存储驱动。空串 = 未指定（按 `json`）；未知值**拒绝启动**（历史上被静默当 `json`，绕过明文闸）。取值大小写 / 空白先归一化再比较 |
| `S3C_STORE_KEY` | 空 | ≥ 16 字符 | 落盘加密口令。`encrypted` 模式**必填**（整文件 AES-256-GCM，格式 `S3C3`，兼容读旧 `S3C2`）；`json` / `sqlite` 设置后启用加密（`sqlite` 加密 `secret_key` 列）。Argon2id + 盐派生，参数随文件头保存 |
| `S3C_ALLOW_PLAINTEXT_STORE` | 关 | 布尔 | **仅本地联调**：允许 `json` / `sqlite` 在无 `S3C_STORE_KEY` 下运行，`secretKey` 明文落盘并在启动日志打出 WARN。**不要在生产设置** |
| `S3C_EXPOSE_METRICS` | 关 | 布尔 | 开 = 暴露 `GET /api/metrics`（Prometheus 文本）；默认 404。**该端点不受 `S3C_TOKEN` 保护**，开启后匿名可读，应仅在内网 / 反代鉴权后放行 |
| `S3C_EXPOSE_OPENAPI` | 关 | 布尔 | 开 = 暴露 `GET /api/openapi.json`（API 契约）；默认 404，避免公网泄露端点信息 |
| `S3C_CSP_CONNECT_SRC` | `'self' http://127.0.0.1:* http://localhost:*` | CSP `connect-src` 值 | 前端可连接的后端白名单。默认仅同源 + 本地 Tauri 后端；**多后端 / 远程后端需显式放宽**，否则浏览器按 CSP 拦截 |
| `S3C_TRUSTED_PROXIES` | 空 | 逗号分隔 IP | 可信反向代理 IP。**仅**这些对端的 `X-Forwarded-For` 被采信用于限速与审计。默认不信任 XFF，防直连伪造绕过限速；直连部署应保持留空 |
| `S3C_SSRF_DENY_PRIVATE` | 关 | 布尔 | 开 = 连私网 / 回环 S3 端点也拒绝（SSRF 加固）。默认关闭：自托管 MinIO / RustFS / 局域网放行，见 [ADR-003](decisions/0003-ssrf-private-allow.md) |

## 3. 启动期硬失败（fail-closed）清单

以下情形进程**直接拒绝启动**（退出码 1），不做静默降级：

| 触发条件 | 拒绝原因 |
|---|---|
| 显式 `S3C_ENV_FILE` 不存在 / 不可读 | 防止加固项与 `S3C_DATA_DIR` 静默失效 |
| `S3C_SHUTDOWN_TIMEOUT` 非整数 / `< 1` / `> 3600` | 防止优雅关停窗口被静默清零 |
| `S3C_STORE_DRIVER` 不在 `json` / `sqlite` / `encrypted` 内 | 防止未知值被当 `json` 绕过明文闸 |
| `S3C_TOKEN` 非空但最短 token < 16 字符 | 短口令易被暴力猜测 |
| `S3C_ADDR` 非回环且 `S3C_TOKEN` 为空 | 防止账号管理 API 无鉴权暴露 |
| `S3C_STORE_KEY` 非空但 < 16 字符 | 落盘加密口令过短会被暴力破解 |
| `S3C_STORE_DRIVER` 为 `json` / `sqlite` 且 `S3C_STORE_KEY` 为空且未开 `S3C_ALLOW_PLAINTEXT_STORE` | 防止 `secretKey` 明文落盘 |
| `S3C_DATA_DIR` 已被另一实例加锁（`.s3clinet.lock`） | 文件型存储 + 内存任务表只支持单实例 |

## 4. 安全默认值摘要

回环绑定 + CORS 白名单 + 短 token 拒绝启动 + **明文存储拒绝启动** + 指标与契约端点默认隐藏 +
CSP `connect-src` 收窄 + 不信任 `X-Forwarded-For`。

生产推荐 `docker compose -f docker-compose.prod.yml`（强制 token + `encrypted`，无内置 RustFS）。
威胁模型与边界见 [threat-model.md](threat-model.md)。

## 5. 客户端设置（Web / 桌面）

前端无构建期配置，全部设置经界面「设置 / Server」维护，存于浏览器存储：

| 设置项 | 存储键 | 说明 |
|---|---|---|
| API 基址 | `s3c.apiBase` | 后端地址。桌面端打开后自动设为 `http://127.0.0.1:8080`，可改 |
| Bearer Token | `s3c.token` | **默认存 `sessionStorage`**（关标签即清）；勾选「跨会话保留」（`s3c_token_persistent`）才写 `localStorage`。`SecretKey` 任何情况下都不落 localStorage |
| 多服务器 | `s3c.servers` / `s3c.activeServerId` | 服务器列表与当前选中项 |
| 当前账号 | `s3c.currentAccountId` | 上次选中的账号 |
| 语言 / 主题 / 提示 | `s3c.locale` / `s3c.theme` / `s3c.hintsHidden` | 纯界面偏好 |

> 后端若设置了 `S3C_CSP_CONNECT_SRC`，切换 API 基址到白名单外的地址会被浏览器 CSP 拦截——
> 服务端需同步放宽该白名单。

## 6. 相关文档

- 部署形态、生产必填项、运维与回滚：[DEPLOYMENT.md](DEPLOYMENT.md)
- 威胁模型与安全边界：[threat-model.md](threat-model.md)
- 漏洞报告渠道：[SECURITY.md](../.github/SECURITY.md)
- 快速开始与功能总览：[README.md](../README.md)
