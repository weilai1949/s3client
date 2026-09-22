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
S3C_ADDR=0.0.0.0:8080

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

> **单实例约束**：账号存储是文件型的（`json` / `sqlite`），异步任务表在内存中，因此服务启动时对
> `S3C_DATA_DIR` 加 `flock` 单写者锁（`.s3clinet.lock`）。同一数据卷起第二个实例会立即失败并报
> `data dir … is already in use`——不要为同一 `/data` 卷编排多副本；水平扩容需先替换外部存储（未立项）。

### 2.3 生产 + TLS（nginx 终止 TLS）

```bash
mkdir -p certs
cp deploy/nginx/conf.d/s3clinet-tls.example.conf deploy/nginx/conf.d/s3clinet-tls.conf
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
make dev            # server(8081) + web(5173)
make dev-nginx      # 加 nginx(8080)

# 分开跑
cd apps/server && go run .      # 后端 127.0.0.1:8080
cd apps/web && pnpm dev         # 前端 5173（Vite 代理到后端）
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
curl http://127.0.0.1:8080/api/health
# {"status":"ok","version":"v1.0.0","time":"...","store":{"ok":true}}
# store 探测失败返回 503（不做降级，见 ADR-002）
```

`503` 时的处置顺序（硬失败设计下服务不会「假装可用」）：

1. 确认现象：`/api/metrics` 的 `s3c_store_up 0`（需 `S3C_EXPOSE_METRICS=1`），服务端日志 Debug 级有 `health store ping`。
2. 检查 `S3C_DATA_DIR` 挂载与写权限（目录 0700 / 文件 0600，运行用户 `app`），以及卷是否写满。
3. 恢复后无需重启：下一次探测成功即回到 200，前端健康轮询会自动恢复界面。

> 告警建议：`s3c_store_up == 0` 持续 1 分钟即告警；硬失败意味着此时所有写操作都在拒绝，不能等业务 5xx 才发现。

### 6.2 优雅关闭

- Go server：`SIGTERM` → 取消异步任务 → `http.Server.Shutdown`（超时 `S3C_SHUTDOWN_TIMEOUT`）
- nginx：`SIGQUIT`（等待 worker 处理完当前请求）
- 脚本：`make restart-server` / `make stop` / `make status`（`scripts/graceful-restart.sh`）

### 6.3 指标

`/api/metrics`（Prometheus 文本格式）**默认 404**，需显式 `S3C_EXPOSE_METRICS=1` 开启。含 HTTP 计数、uptime、goroutine、内存、`s3c_build_info`，以及 `s3c_store_up`（存储可达性，掉线为 0）、`s3c_ssrf_deny_private`（SSRF 生效策略 0/1）与 `s3c_stream_interrupted_total`（流式传输中断计数）。后者用于发现大文件下载被上游读失败/写超时打断的情况——此前这类失败被 `io.Copy` 的返回值吞掉，日志与指标里都没有痕迹。

> **`/api/metrics` 不受 `S3C_TOKEN` 保护**：即使配置了 token，只要 `S3C_EXPOSE_METRICS=1`，该端点无需 `Authorization` 头即返回 200（有意为内网 Prometheus 免 token scrape）。代价是**匿名可读**（版本、存储可达性、S3 上游调用统计等运行信息）。请只在**内网 / 反向代理鉴权之后**暴露，切勿把开启 metrics 的实例直接放上公网。

### 6.4 升级

跟随 [CHANGELOG.md](../CHANGELOG.md) 的 Unreleased 段与 GitHub Release；dependabot 每周自动提交依赖更新 PR，CI 的 Trivy / govulncheck 门禁拦截已知漏洞。

## 7. 回滚

- Docker：`docker compose down && docker compose -f docker-compose.prod.yml up -d`（镜像 tag 指回旧版本）。
- 数据：账号存储（`accounts.json` / `accounts.db` / `accounts.json.enc`）挂载于 `/data` 卷，回滚前先备份；
  同目录的 `.s3clinet.lock` 只是 flock 锁文件，不需要备份（进程退出即释放）。
