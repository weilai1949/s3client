# 部署指南

> 涵盖 Docker Compose（base / prod / tls）、Nginx 反向代理、本机运行与桌面端分发。
> 快速开始见 [README.md](../README.md#快速开始)；Nginx 细节见 [deploy/nginx/README.md](../deploy/nginx/README.md)。

## 1. 部署形态总览

| 形态 | 适用场景 | 入口 |
|---|---|---|
| Docker Compose（base） | 本地联调（server + RustFS + nginx） | `docker compose up -d` |
| Docker Compose（prod） | 生产（server + nginx，无 RustFS） | `docker compose -f docker-compose.prod.yml up -d` |
| Docker Compose（prod + TLS） | 生产 + HTTPS | `docker compose -f docker-compose.prod.yml -f docker-compose.tls.yml up -d` |
| 本机直接运行 | 开发 / 单机 | `make dev` 或 `cd apps/server && go run .` |
| 桌面端 | 桌面 GUI | GitHub Release 安装包（.exe / .deb / .dmg） |

## 2. 生产部署（推荐）

### 2.1 密钥注入与环境变量（生产必填）

compose 部署（推荐）经 **docker secrets** 注入两枚密钥，服务端用 `*_FILE` 读取（见 §2.1.1）；
非 compose 部署（`docker run -e` / systemd `LoadCredential`）也可直接注入同名环境变量：

```bash
# 鉴权（必填，openssl rand -hex 32 生成）
# compose：S3C_TOKEN_FILE=/run/secrets/api_token（见 §2.1.1）；直接注入则用 S3C_TOKEN=<随机 token>
S3C_TOKEN_FILE=/run/secrets/api_token

# 存储：生产推荐 encrypted（密钥落盘加密）
S3C_STORE_DRIVER=encrypted
# compose：S3C_STORE_KEY_FILE=/run/secrets/store_key；直接注入则用 S3C_STORE_KEY=<强密钥>
S3C_STORE_KEY_FILE=/run/secrets/store_key

# 监听：经 nginx 反向代理时绑定 0.0.0.0
S3C_ADDR=0.0.0.0:5000

# 日志：结构化输出便于采集
S3C_LOG_JSON=1

# 可选：SSRF 加固，拒绝私网 / 回环 S3 端点（默认关闭，自托管场景不设）
# S3C_SSRF_DENY_PRIVATE=1

# 可选：开启 metrics，并用独立 token 保护 Prometheus scrape（两者都需要）
# S3C_EXPOSE_METRICS=1
# S3C_METRICS_TOKEN=<openssl rand -hex 32>
```

> **安全提醒（安全默认：明文存储拒绝启动）**：`json` / `sqlite` 驱动只有在设置 `S3C_STORE_KEY`
> （或 `S3C_STORE_KEY_FILE`）时才会把 `secretKey` 加密落盘（AES-256-GCM）；**不设 key 时进程直接
> 拒绝启动**，除非显式设置 `S3C_ALLOW_PLAINTEXT_STORE=1`（仅限本地联调，启动日志会打出「secretKey
> 将明文落盘」WARN）。生产推荐 `encrypted` 驱动（整文件加密）或 `sqlite` + key（至少 16 字符，
> `openssl rand -hex 32`）。
> 加密文件格式为 S3C3（Argon2id 参数随文件头保存），并兼容读取旧的 S3C2 库。
> 详见 [threat-model.md](threat-model.md)。

### 2.1.1 密钥经文件注入（docker secrets，compose 默认形态）

`S3C_STORE_KEY_FILE` / `S3C_TOKEN_FILE` 非空时从该文件读取，值 = 文件内容去掉**首尾空白与换行**；
多 token 仍按逗号分隔写在文件内容里。三条 **fail-closed** 规则（任一命中即拒绝启动，退出码 1）：
文件不可读 / 不存在、读出内容为空、与同名明文变量（`S3C_STORE_KEY` / `S3C_TOKEN`）**同时非空设置**
（配置歧义，不静默取其一；明文变量为空串不算「设置」）。

两个 compose 文件已内置 secret 定义（base 与 prod 一致）：`store_key` → `./secrets/store_key`、
`api_token` → `./secrets/api_token`；容器内路径为 `/run/secrets/<name>`，server 服务用
`S3C_STORE_KEY_FILE` / `S3C_TOKEN_FILE` 指向它们，**明文 `environment:` 注入已删除**——
`docker inspect` 与 `/proc/<pid>/environ` 不再能看到两枚密钥（渗透测试 2026-10-15 P-8）。

首次部署 / 从明文 `.env` 迁移：

```bash
mkdir -p secrets
openssl rand -hex 32 > secrets/store_key
openssl rand -hex 32 > secrets/api_token
chmod 600 secrets/store_key secrets/api_token
# compose 的**文件型** secret 按宿主机属主 / 权限原样 bind-mount；server 容器以 uid 1000（USER app）运行。
# 只 chmod 600、属主仍是 root 的话容器读不到 → 进程 fail-closed 起不来：
sudo chown 1000:1000 secrets/store_key secrets/api_token   # 生成用户本身就是 uid 1000 时可跳过
docker compose -f docker-compose.yml up -d --build          # 生产：-f docker-compose.prod.yml
```

> ⚠️ `uid` / `gid` / `mode` **长语法**在 file 型 secret 下会被 compose 接受但**忽略**（实测无效）——
> 不要指望用它改属主 / 权限，必须在宿主机上按上面的 `chmod` + `chown` 处理。
>
> **属主不对会怎样（实测负向证据）**：把 `secrets/store_key` 改回 `root:root` + `chmod 600` 后重启，
> 容器 `exit=1` 反复重启，日志为
> `配置校验失败 err="unreadable secret file: S3C_STORE_KEY_FILE=/run/secrets/store_key`
> `不存在或不可读: open /run/secrets/store_key: permission denied"`——所以 `chown 1000:1000` **不是可选项**；
> 恢复属主后重新 `healthy`。

验证（三行都应给出预期输出）：

```bash
docker compose exec -T server sh -c 'test -r /run/secrets/store_key && test -r /run/secrets/api_token && echo READABLE'
docker compose exec -T server env | grep -E '^S3C_(STORE_KEY|TOKEN)_FILE='
docker inspect "$(docker compose ps -q server)" --format '{{json .Config.Env}}' | tr ',' '\n' \
  | grep -E '^"?S3C_(STORE_KEY|TOKEN)=' || echo '无明文密钥注入'
```

- `secrets/` 目录建议 `0700`，且已在 `.gitignore` 中忽略（`/secrets/*`，仅 `*.example` 例外）；
- 迁移完成后可删掉根 `.env` 里的 `S3C_STORE_KEY` / `S3C_TOKEN`（compose 已不再插值这两个键）；
- **零配置冒烟不受影响**：`docker compose config -q` 在 secret 文件缺失时仍 exit 0，`e2e.yml` 只起
  RustFS、不启 server；但全栈 `docker compose up -d` 前必须先按上面生成两个文件。

### 2.2 生产 compose

```bash
cp .env.example .env
# 编辑 .env 填入非密钥项（S3C_IMAGE_TAG / S3C_CORS_ORIGINS / 限流等）；
# 两枚密钥改由 secrets 注入，见 §2.1.1
docker compose -f docker-compose.prod.yml up -d
```

> **根 `.env` 只承载 compose 会用到的键**（`docker-compose*.yml` 里 `${VAR}` 插值的变量，见根
> [`.env.example`](../.env.example)）。compose 的 `environment:` 是**显式白名单**且未用 `env_file:`——
> `S3C_SSRF_DENY_PRIVATE` / `S3C_TRUSTED_PROXIES` 这类服务端专属项写进 `.env` **不会进入容器**；
> 需要时请用 `docker run -e` / systemd 注入，或自行把该键加入 compose 白名单（口径见
> [`CONFIGURATION.md`](CONFIGURATION.md) §1）。

> **单实例约束**：账号存储是文件型的（`json` / `sqlite`），异步任务表在内存中，因此服务启动时对
> `S3C_DATA_DIR` 加 `flock` 单写者锁（`.s3client.lock`）。同一数据卷起第二个实例会立即失败并报
> `data dir … is already in use`——不要为同一 `/data` 卷编排多副本；水平扩容需先替换外部存储（未立项）。

### 2.3 生产 + TLS（nginx 终止 TLS）

```bash
mkdir -p certs
cp deploy/nginx/conf.d/s3client-tls.example.conf deploy/nginx/conf.d/s3client-tls.conf
# 编辑证书路径 / 域名 / 补 HSTS 头（见 deploy/nginx/README.md）
docker compose -f docker-compose.prod.yml -f docker-compose.tls.yml up -d
```

## 3. Nginx 反向代理要点

- **单 worker**（`worker_processes 1`）：与 Go 后端一对一，`nginx -s reload` 零停机热加载。
- **流式兼容**：`proxy_buffering off` + `proxy_read_timeout 3600s`（大文件下载/迁移 SSE 不被截断）。
- **内存上限**：`deploy.resources.limits.memory: 128M`。
- **优雅 reload**：`docker compose exec nginx nginx -s reload`；优雅停止用 `SIGQUIT`。
- **边缘镜像自建（渗透测试 2026-10-15 P-4，方案：仓库内构建）**：`deploy/nginx/Dockerfile`
  以 `FROM nginx:1.30.6-alpine@sha256:970364228b1e4f932abc22d321932816782cca8a20b85a77552a7cfe9acfb3b5`
  为基座（**digest pin 保证可复现**）+ `RUN apk upgrade --no-cache`（**构建期保证拿到 alpine 分支内的
  安全补丁**）；两个 compose 的 nginx 服务改为 `build:` 该 Dockerfile（镜像名 `s3client/nginx:<tag>`）。
  这样「digest 冻结吃不到补丁」的问题**从结构上消除**——实测上游所有 nginx alpine tag
  （1.30.x / 1.31 / mainline / stable）当前都 ship `zlib-1.3.2-r0` 而 `1.3.2-r1` 已发布，
  直接引用上游镜像无法在运行时升级；构建产物实测为 `zlib-1.3.2-r1`（`libssl3` / `libcrypto3` 亦
  全部 `=`、无待升级项）。**仍未引入任何 `.trivyignore` 忽略条目**：edge 镜像构建产物由 CI
  用同一套阻塞式 Trivy（`--exit-code 1 --severity CRITICAL,HIGH --ignore-unfixed`）扫描，实测 0 条。
  **基座升级节奏**：抬 `deploy/nginx/Dockerfile` 的基座 digest（当前 Alpine 3.24 分支支持至 2028-06-01）。
- **升级 / 运维命令（注意 nginx 不再是拉取的上游镜像）**：改完 Dockerfile 或需要重建边缘镜像时
  `docker compose build nginx` → `docker compose up -d --build`；**`docker compose pull` 不会覆盖 nginx**
  （它是本地构建产物），配置变更仍需 `build` + `up -d --build`。
- **容器加固（渗透测试 2026-10-15 P-14）**：两个 compose 的 server 与 nginx 均启用
  `read_only: true` + `security_opt: [no-new-privileges:true]` + `cap_drop: [ALL]`。server 额外挂
  `tmpfs: /tmp`（可写临时目录），`/data` 仍是 named volume（`read_only` 下仍可写）；
  nginx 的 `read_only` 需要两条可写路径：`tmpfs: /var/cache/nginx`（临时目录与 `client_body_temp` 等）
  与 `tmpfs: /run`（或 `/var/run`，pid 文件 `/var/run/nginx.pid` 落在这里）；`cap_drop: [ALL]` 之外还需
  `cap_add: [CHOWN, SETGID, SETUID]`——官方 entrypoint 要 `chown` `/var/cache/nginx` 并切换到 `nginx`
  用户，只 drop 不给这三个 cap 时会以 `chown("/var/cache/nginx/client_temp", 101) failed
  (1: Operation not permitted)` 启动失败。**不要**给 `/var/log/nginx` 挂 tmpfs：官方镜像的
  `access.log` / `error.log` 是指向 `/dev/stdout` / `/dev/stderr` 的**符号链接**，覆盖挂载会把软链挡住，
  nginx 启动失败、日志也无法被 json-file 采集。

## 4. 本机运行

```bash
# 一键（server + web + 可选 nginx）
make dev            # server(5000) + web(1949)
make dev-nginx      # Go(5001) + nginx(5000)

# 分开跑
cd apps/server && go run .      # 后端 127.0.0.1:5000
cd apps/web && pnpm dev         # 前端 127.0.0.1:1949（Vite 代理到后端）
```

## 5. 桌面端分发

打 tag（`v*`）触发 [release-desktop.yml](../.github/workflows/release-desktop.yml)，交叉构建：

- Windows：NSIS `.exe`（未签名，首次运行会被 SmartScreen 拦截，需手动放行）
- Linux：`.deb`
- macOS：`.dmg`（未公证，用户需手动允许打开）

产物挂 GitHub Release，附 `SHA256SUMS`。桌面端为纯 B/S 壳（无 IPC），运行后访问本地 Go 后端或远程后端（见「服务器」设置）。

## 6. 运维

### 6.1 健康检查

```bash
curl http://127.0.0.1:5000/api/health
# {"status":"ok","time":"...","store":{"ok":true}}
# 带有效 S3C_TOKEN 时响应另含 "version":"v1.0.0"
# store 探测失败返回 503（不做降级，见 ADR-002）
```

`503` 时的处置顺序（硬失败设计下服务不会「假装可用」）：

1. 确认现象：`/api/metrics` 的 `s3c_store_up 0`（需 `S3C_EXPOSE_METRICS=1`），服务端日志 Debug 级有 `health store ping`。
2. 检查 `S3C_DATA_DIR` 挂载与写权限（目录 0700 / 文件 0600，运行用户 `app`），以及卷是否写满。
3. 恢复后无需重启：下一次探测成功即回到 200，前端健康轮询会自动恢复界面。

> 告警建议：`s3c_store_up == 0` 持续 1 分钟即告警；硬失败意味着此时所有写操作都在拒绝，不能等业务 5xx 才发现。
>
> **探针限流与 Ping 缓存（P-12）**：`/api/health` / `/api/metrics` / `/api/openapi.json` 不被主限流桶
> （`S3C_RATE_LIMIT_RPM`）拖累，但受**独立探针令牌桶**约束（`S3C_PROBE_RATE_LIMIT_RPM` 默认 600、
> `S3C_PROBE_RATE_LIMIT_BURST` 默认 100，超限 429 + `Retry-After: 1`）——监控 probe 频率过高时先看 429
> 再排查存储。`store.Ping` 结果按 `S3C_HEALTH_PING_CACHE_SECONDS`（默认 2 秒，`0` = 不缓存）短 TTL
> 缓存，匿名与已认证同一口径，**失败结果同样缓存**：TTL 窗口内 503 会稳定复现、恢复最多延迟一个 TTL
> 才转 200，告警 `for` 窗口要按此留余量。

### 6.2 优雅关闭

- Go server：`SIGTERM` → 取消异步任务 → `http.Server.Shutdown`（超时 `S3C_SHUTDOWN_TIMEOUT`，取值 **1–3600 秒**，超上界拒绝启动——秒数过大时 `time.Duration` 会溢出为负时长，把优雅关停静默清零）
- nginx：`SIGQUIT`（等待 worker 处理完当前请求）
- 脚本：`make restart-server` / `make stop` / `make status`（`scripts/graceful-restart.sh`）

### 6.3 指标

`/api/metrics`（Prometheus 文本格式）**默认 404**，需显式 `S3C_EXPOSE_METRICS=1` 开启。含 HTTP 计数与延迟直方图、uptime、goroutine、内存、`s3c_build_info`，以及 `s3c_store_up`、`s3c_store_write_failures_total`、容量 / inode、`s3c_jobs_active`、`s3c_last_shutdown_duration_seconds`、`s3c_ssrf_deny_private`、`s3c_rate_limit_rpm` / `s3c_rate_limit_burst` 与 `s3c_stream_interrupted_total`。完整清单见 [`OPERATIONS.md`](OPERATIONS.md) §3.2。

> **metrics 鉴权**：设置 `S3C_METRICS_TOKEN` 后必须携带该独立 Bearer；不设置则保持匿名 scrape 兼容。生产建议设置独立 token，并在 Prometheus 用 `authorization.credentials_file` 挂载 secret（示例见 [`OPERATIONS.md`](OPERATIONS.md) §3.2）。请只在内网 / 反向代理鉴权之后暴露。

### 6.4 升级

依赖面：跟随 [CHANGELOG.md](../CHANGELOG.md) 的 Unreleased 段与 GitHub Release；dependabot 每周自动提交依赖更新 PR，CI 的 Trivy / govulncheck 门禁拦截已知漏洞。

可执行步骤（Docker 部署，prod compose；本地二进制同理替换第 1/4 步）：

1. **升级前备份**：按 [`OPERATIONS.md`](OPERATIONS.md) §6 做一次 `/data` 卷备份，并确认 `S3C_STORE_KEY` 可用（丢了 key 备份也解不开）；读 CHANGELOG 确认目标版本无破坏性变更。
2. **换 tag**：改根 `.env` 的 `S3C_IMAGE_TAG=<新版本>`（prod compose 的镜像为 `s3client/server:${S3C_IMAGE_TAG:-v1.0.0}`；基础 `docker-compose.yml` 则**写死** tag，升级需直接改该文件）。
3. **拉新版并重启**：`docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml build nginx && docker compose -f docker-compose.prod.yml stop && docker compose -f docker-compose.prod.yml up -d`（`pull` 只对 `s3client/server` 生效；nginx 是 §3 的**本地构建**边缘镜像，`pull` 不会覆盖它，需 `build nginx` 或直接 `up -d --build`；宽限期见 [`OPERATIONS.md`](OPERATIONS.md) R-9）。
4. **验证六项**：① `GET /api/health` 返回 200；② `/api/metrics` 的 `s3c_build_info{version=...}` 与目标 tag 一致（需 `S3C_EXPOSE_METRICS=1`）；③ 服务端日志无启动期 fail-closed 报错；④ 抽查一次流式下载；⑤ 抽查一次写操作；⑥ 前端页面版本与后端一致（前端「关于 / 版本」处核对）。
5. **异常即回滚**：见 §7（tag 指回旧版本，回滚前先备份 `/data`）。

> **格式兼容提醒**：加密格式当前写入 `S3C3`（可读旧 `S3C2`）；**降级到不认识新格式的旧版本可能读不了账号库**——降级前的备份是硬要求，详见 [`OPERATIONS.md`](OPERATIONS.md) §8 与 [`compatibility.md`](compatibility.md) §4。

## 7. 回滚

- **Docker（prod compose）**：根 `.env` 把 `S3C_IMAGE_TAG` 改回上一个已知良好版本 → `docker compose -f docker-compose.prod.yml up -d`（基础 `docker-compose.yml` 直接改文件里写死的镜像 tag）。镜像 tag 之外的回滚面（`S3C_STORE_DRIVER` / `S3C_STORE_KEY` 等配置）保持升级前取值，不在回滚时顺手改动。
- **回滚后验证**：同 §6.4 第 4 步六项（`/api/health` 200、`s3c_build_info` 与旧 tag 一致、日志干净、一次下载 + 一次写操作抽查）；检查表口径见 [`OPERATIONS.md`](OPERATIONS.md) §8「升级与回滚」。
- **数据**：账号存储（`accounts.json` / `accounts.db` / `accounts.json.enc`）挂载于 `/data` 卷，**回滚前先备份**——旧版本可能读不了 `S3C3` 新格式（降级格式风险见 §6.4 提醒）；
  同目录的 `.s3client.lock` 只是 flock 锁文件，不需要备份（进程退出即释放）。
