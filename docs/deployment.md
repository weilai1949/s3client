# 部署指南

> 涵盖 Docker Compose（base / prod / tls）、Nginx 反向代理、本机运行与桌面端分发。
> 快速开始见 [README.md](../README.md#快速开始)；Nginx 细节见 [deploy/nginx/README.md](../deploy/nginx/README.md)。

## 1. 部署形态总览

| 形态 | 适用场景 | 入口 |
|---|---|---|
| Docker Compose（base） | 本地联调（server + RustFS + nginx） | `docker compose up -d` |
| Docker Compose（prod） | 生产（server + nginx，无 RustFS） | `docker compose -f docker-compose.prod.yml up -d` |
| Docker Compose（prod + TLS） | 生产 + HTTPS | `docker compose -f docker-compose.prod.yml -f docker-compose.tls.yml up -d` |
| 本机直接运行 | 开发 / 单机 | `make dev` 或 `cd server && go run .` |
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
```

> ⚠️ **安全提醒**：`sqlite` 驱动当前将 `secretKey` **明文**落盘；生产请使用 `encrypted` 驱动
> （AES-256-GCM + Argon2id）或配合磁盘级加密。详见 [security.md](security.md)。

### 2.2 生产 compose

```bash
cp .env.example .env
# 编辑 .env 填入 S3C_TOKEN / S3C_STORE_KEY
docker compose -f docker-compose.prod.yml up -d
```

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
cd server && go run .      # 后端 127.0.0.1:8080
cd web && pnpm dev         # 前端 5173（Vite 代理到后端）
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
# {"status":"ok","version":"v1.0.0-rc1","time":"...","store":{"ok":true}}
# store 探测失败返回 503（不做降级，见 ADR-002）
```

### 6.2 优雅关闭

- Go server：`SIGTERM` → 取消异步任务 → `http.Server.Shutdown`（超时 `S3C_SHUTDOWN_TIMEOUT`）
- nginx：`SIGQUIT`（等待 worker 处理完当前请求）
- 脚本：`make restart-server` / `make stop` / `make status`（`scripts/graceful-restart.sh`）

### 6.3 指标

`/api/metrics`（Prometheus 文本格式）**默认 404**，需显式 `S3C_EXPOSE_METRICS=1` 开启。含 HTTP 计数、uptime、goroutine、内存与 `s3c_build_info`。

### 6.4 升级

跟随 [CHANGELOG.md](../CHANGELOG.md) 的 Unreleased 段与 GitHub Release；dependabot 每周自动提交依赖更新 PR，CI 的 Trivy / govulncheck 门禁拦截已知漏洞。

## 7. 回滚

- Docker：`docker compose down && docker compose -f docker-compose.prod.yml up -d`（镜像 tag 指回旧版本）。
- 数据：账号存储（`accounts.json` / `accounts.db` / `accounts.json.enc`）挂载于 `/data` 卷，回滚前先备份。
