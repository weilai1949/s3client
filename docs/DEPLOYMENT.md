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

### 2.1 环境变量（生产必填）

```bash
# 鉴权（必填，openssl rand -hex 32 生成）
S3C_TOKEN=<随机 token>

# 存储：生产推荐 encrypted（密钥落盘加密）
S3C_STORE_DRIVER=encrypted
S3C_STORE_KEY=<强密钥>

# 监听：经 nginx 反向代理时绑定 0.0.0.0
S3C_ADDR=0.0.0.0:5000

# 日志：结构化输出便于采集
S3C_LOG_JSON=1

# 可选：SSRF 加固，拒绝私网 / 回环 S3 端点（默认关闭，自托管场景不设）
# S3C_SSRF_DENY_PRIVATE=1
```

> **安全提醒（安全默认：明文存储拒绝启动）**：`json` / `sqlite` 驱动只有在设置 `S3C_STORE_KEY` 时
> 才会把 `secretKey` 加密落盘（AES-256-GCM）；**不设 key 时进程直接拒绝启动**，除非显式设置
> `S3C_ALLOW_PLAINTEXT_STORE=1`（仅限本地联调，启动日志会打出「secretKey 将明文落盘」WARN）。
> 生产推荐 `encrypted` 驱动（整文件加密）或 `sqlite` + `S3C_STORE_KEY`（至少 16 字符，
> `openssl rand -hex 32`）。`docker-compose.yml`（base）已用 `${S3C_STORE_KEY:?…}` 做非空守卫，
> 未填 key 时 `docker compose up` 直接失败；`docker-compose.prod.yml` 用 `encrypted` + 强制 key。
> 加密文件格式为 S3C3（Argon2id 参数随文件头保存），并兼容读取旧的 S3C2 库。
> 详见 [threat-model.md](threat-model.md)。

### 2.2 生产 compose

```bash
cp .env.example .env
# 编辑 .env 填入 S3C_TOKEN / S3C_STORE_KEY
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

- Windows：NSIS `.exe`
- Linux：`.deb`
- macOS：`.dmg`（未公证，用户需手动允许打开）

产物挂 GitHub Release，附 `SHA256SUMS`。桌面端为纯 B/S 壳（无 IPC），运行后访问本地 Go 后端或远程后端（见「服务器」设置）。

## 6. 运维

### 6.1 健康检查

```bash
curl http://127.0.0.1:5000/api/health
# {"status":"ok","version":"v1.0.0","time":"...","store":{"ok":true}}
# store 探测失败返回 503（不做降级，见 ADR-002）
```

`503` 时的处置顺序（硬失败设计下服务不会「假装可用」）：

1. 确认现象：`/api/metrics` 的 `s3c_store_up 0`（需 `S3C_EXPOSE_METRICS=1`），服务端日志 Debug 级有 `health store ping`。
2. 检查 `S3C_DATA_DIR` 挂载与写权限（目录 0700 / 文件 0600，运行用户 `app`），以及卷是否写满。
3. 恢复后无需重启：下一次探测成功即回到 200，前端健康轮询会自动恢复界面。

> 告警建议：`s3c_store_up == 0` 持续 1 分钟即告警；硬失败意味着此时所有写操作都在拒绝，不能等业务 5xx 才发现。

### 6.2 优雅关闭

- Go server：`SIGTERM` → 取消异步任务 → `http.Server.Shutdown`（超时 `S3C_SHUTDOWN_TIMEOUT`，取值 **1–3600 秒**，超上界拒绝启动——秒数过大时 `time.Duration` 会溢出为负时长，把优雅关停静默清零）
- nginx：`SIGQUIT`（等待 worker 处理完当前请求）
- 脚本：`make restart-server` / `make stop` / `make status`（`scripts/graceful-restart.sh`）

### 6.3 指标

`/api/metrics`（Prometheus 文本格式）**默认 404**，需显式 `S3C_EXPOSE_METRICS=1` 开启。含 HTTP 计数与延迟直方图、uptime、goroutine、内存、`s3c_build_info`，以及 `s3c_store_up`（存储可达性，掉线为 0）、`s3c_store_write_failures_total`（账号库写入失败，`json` / `encrypted` 驱动唯一的主动故障信号）、`s3c_volume_size_bytes` / `s3c_volume_free_bytes` / `s3c_volume_inode_total` / `s3c_volume_inode_free`（数据卷容量与 inode，取不到时不发序列）、`s3c_jobs_active`（在册异步任务数）、`s3c_last_shutdown_duration_seconds`（上次关停耗时）、`s3c_ssrf_deny_private`（SSRF 生效策略 0/1）与 `s3c_stream_interrupted_total`（流式传输中断计数）。完整清单见 [`OPERATIONS.md`](OPERATIONS.md) §3.2；`s3c_stream_interrupted_total` 用于发现大文件下载被上游读失败/写超时打断的情况——此前这类失败被 `io.Copy` 的返回值吞掉，日志与指标里都没有痕迹。

> **`/api/metrics` 不受 `S3C_TOKEN` 保护**：即使配置了 token，只要 `S3C_EXPOSE_METRICS=1`，该端点无需 `Authorization` 头即返回 200（有意为内网 Prometheus 免 token scrape）。代价是**匿名可读**（版本、存储可达性、S3 上游调用统计等运行信息）。请只在**内网 / 反向代理鉴权之后**暴露，切勿把开启 metrics 的实例直接放上公网。

### 6.4 升级

依赖面：跟随 [CHANGELOG.md](../CHANGELOG.md) 的 Unreleased 段与 GitHub Release；dependabot 每周自动提交依赖更新 PR，CI 的 Trivy / govulncheck 门禁拦截已知漏洞。

可执行步骤（Docker 部署，prod compose；本地二进制同理替换第 1/4 步）：

1. **升级前备份**：按 [`OPERATIONS.md`](OPERATIONS.md) §6 做一次 `/data` 卷备份，并确认 `S3C_STORE_KEY` 可用（丢了 key 备份也解不开）；读 CHANGELOG 确认目标版本无破坏性变更。
2. **换 tag**：改根 `.env` 的 `S3C_IMAGE_TAG=<新版本>`（prod compose 的镜像为 `s3client/server:${S3C_IMAGE_TAG:-v1.0.0}`；基础 `docker-compose.yml` 则**写死** tag，升级需直接改该文件）。
3. **拉新版并重启**：`docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml stop && docker compose -f docker-compose.prod.yml up -d`（宽限期见 [`OPERATIONS.md`](OPERATIONS.md) R-9）。
4. **验证六项**：① `GET /api/health` 返回 200；② `/api/metrics` 的 `s3c_build_info{version=...}` 与目标 tag 一致（需 `S3C_EXPOSE_METRICS=1`）；③ 服务端日志无启动期 fail-closed 报错；④ 抽查一次流式下载；⑤ 抽查一次写操作；⑥ 前端页面版本与后端一致（前端「关于 / 版本」处核对）。
5. **异常即回滚**：见 §7（tag 指回旧版本，回滚前先备份 `/data`）。

> **格式兼容提醒**：加密格式当前写入 `S3C3`（可读旧 `S3C2`）；**降级到不认识新格式的旧版本可能读不了账号库**——降级前的备份是硬要求，详见 [`OPERATIONS.md`](OPERATIONS.md) §8 与 [`compatibility.md`](compatibility.md) §4。

## 7. 回滚

- **Docker（prod compose）**：根 `.env` 把 `S3C_IMAGE_TAG` 改回上一个已知良好版本 → `docker compose -f docker-compose.prod.yml up -d`（基础 `docker-compose.yml` 直接改文件里写死的镜像 tag）。镜像 tag 之外的回滚面（`S3C_STORE_DRIVER` / `S3C_STORE_KEY` 等配置）保持升级前取值，不在回滚时顺手改动。
- **回滚后验证**：同 §6.4 第 4 步六项（`/api/health` 200、`s3c_build_info` 与旧 tag 一致、日志干净、一次下载 + 一次写操作抽查）；检查表口径见 [`OPERATIONS.md`](OPERATIONS.md) §8「升级与回滚」。
- **数据**：账号存储（`accounts.json` / `accounts.db` / `accounts.json.enc`）挂载于 `/data` 卷，**回滚前先备份**——旧版本可能读不了 `S3C3` 新格式（降级格式风险见 §6.4 提醒）；
  同目录的 `.s3client.lock` 只是 flock 锁文件，不需要备份（进程退出即释放）。
