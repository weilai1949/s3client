# 架构设计

> 本文档描述 s3clinet 的整体架构与关键设计决策。架构决策记录（ADR）见
> [`docs/decisions/`](decisions/index.md)。

## 1. 总体架构

s3clinet 采用 **B/S（Browser/Server）架构 + Tauri 2 桌面壳（无 IPC）**：

```
┌─────────────────────────────┐
│  Web 端 (浏览器)             │
│  Tauri 2 桌面壳 (B/S, 无IPC) │
└──────────────┬──────────────┘
               │ HTTP (REST /api, 可选 Bearer)
┌──────────────▼──────────────┐
│  Go 后端 (B/S Server)        │
│  · 账号配置存储              │
│  · S3 SDK v2 封装            │
│  · v4 预签名 URL 生成        │
└──────┬───────────┬─────────┘
       │           │ 前端直传
       │           ▼ (presigned v4 URL 直接 PUT 到 S3)
       │      ┌─────────────┐
       └────▶ │ S3 兼容服务  │
             │(阿里/腾讯/…)  │
             └─────────────┘
```

### 核心设计决策

1. **前端直传**：浏览器从后端拿到 v4 签名 URL 后**直接**上传/下载 S3，大流量不经过 Go 服务。密钥永不回传前端。
2. **桌面端无 IPC**：Tauri 2 仅作壳，前后端全部走 HTTP，攻击面最小（见 [ADR-001](decisions/0001-desktop-no-ipc.md)）。
3. **REST API 为唯一入口**：Web 与桌面端复用同一 `/api/*`，OpenAPI 3.0.3 契约自动生成（`/api/openapi.json`）。

## 2. 后端分层

```
apps/web/src (Vue 3)
   │  HTTP + JSON
   ▼
apps/server/internal/handler    HTTP 层：路由、参数校验、错误映射、DTO 转换
   │
   ▼
apps/server/internal/service    批量/迁移/异步任务/zip 等业务编排
   │
   ▼
apps/server/internal/s3wrap      AWS SDK v2 封装 + SSRF 防护 + 预签名（防腐层）
   │
   ▼
apps/server/internal/store       账号存储（json / sqlite / encrypted，统一入口）
   ▲
   │
apps/server/internal/model        领域模型（Account / AccountView）
```

- **依赖方向**：`handler → service → s3wrap`、`handler/store → model`，无反向依赖、无循环。
- **防腐层**：`s3wrap` 是 AWS 类型与外界的唯一边界，AWS 类型不外泄到 handler/前端。
- **统一入口**：`store.Open(dataDir, driver, key)` 按驱动打开存储。

### 关键机制

| 机制 | 位置 | 说明 |
|---|---|---|
| 错误映射 | `s3wrap/errors.go` | S3 错误 → 稳定 HTTP 状态 + 短英文用户消息（sentinel + `errors.Is`） |
| SSRF 防护 | `s3wrap/ssrf.go` | 创建时 + 拨号期双重校验（禁 IMDS/链路本地、禁重定向、禁代理）；`S3C_SSRF_DENY_PRIVATE=1` 可连私网/回环一并拒绝 |
| 预签名 | `s3wrap/presign.go` | v4 签名 URL，过期钳制 [1h, 24h] |
| 原子写 | `store/atomic.go` | 临时文件 + rename + 0600；写失败回滚内存 |
| 单写者锁 | `store/lock.go` | `flock` 锁 `DataDir`（`.s3clinet.lock`），第二实例启动即失败；非 unix 为 no-op |
| 任务清单落盘 | `service/job_persist.go` | 同上原子写策略，但自包含于 `service` 包：`service→store` 会形成分层倒置 |
| 流式限并发 | `handler/stream.go` | 全局 32 并发 + 滚动空闲写超时 5min |
| 批量有界并发 | `service/batch.go` | `RunBatch` 无缓冲结果通道，内存 O(workers) |
| 异步任务 | `service/job.go` + `job_persist.go` | JobRegistry + SSE 进度 + TTL reap；任务清单可选落盘（`JobPersister`），启动时把非终态任务标记 `interrupted` 并对账 |

## 3. 前端架构

```
apps/web/src/
  api/               API 客户端（按职责拆分的模块目录，见下）
    storage.ts         浏览器凭据 / 多服务器 profile 存储（依赖图最底层）
    http.ts            传输层：base + Bearer + JSON + 错误归一
    endpoints.ts       领域端点封装（s3api，~70 个方法）
    jobs.ts            异步任务 SSE 订阅 + EOF 后状态回读兜底
    download.ts        ZIP 流式落盘（File System Access API + blob 兜底）
    upload.ts          预签名直传（XHR，提供上传进度）
    index.ts           公开面 barrel：组装 `api` 对象并重导出
  store.ts           全局状态（账号 / tab / toast）
  types.ts           与后端契约对齐的类型定义
  components/        面板与对话框组件
  composables/       可复用逻辑（对象浏览/操作/上传/预览/快捷键）
  i18n/messages/     按域拆分的 zh/en 消息字典
  router.ts          hash 深链接（无需 vue-router 依赖）
```

> `api/` 原为单文件 `api.ts`（838 行，把凭据存储、传输、领域端点、SSE、上传混在一处）。
> 拆分为目录后 `./api` / `../api` 仍解析到 `api/index.ts`，**对外契约与 import 路径不变**；
> 模块间为单向依赖 `index → {endpoints, jobs, download, upload} → http → storage`，无环。
> 公开面由 `src/deadcode_gate.test.ts` 守住：生产代码零引用的导出会让门禁变红。

- **技术栈**：Vue 3 + Vite + TS，生产依赖**仅 `vue`**（刻意最小化供应链）。
- **竞态防护**：`loadSeq` + `AbortController` 双保险，SSE 订阅在卸载路径全部断开。
- **预览安全**：所有预览走服务端代理（三模式 + 类型白名单 + sandbox），前端 0 处 `v-html`。

## 4. 桌面端

- Tauri 2，纯 B/S 壳，无 IPC、无 command（见 [ADR-001](decisions/0001-desktop-no-ipc.md)）。
- 权限收敛：`withGlobalTauri` 关闭 + capability 裁剪（空权限集）。
- 分发：CI 交叉构建 `.exe`(NSIS) / `.deb` / `.dmg`，挂 GitHub Release，附 SHA256SUMS。

## 5. 数据流示例：对象上传

1. 前端 `POST /api/accounts/{id}/presign`（method=put）→ 后端 `PresignPut` 生成 v4 签名 PUT URL。
2. 前端用 `fetch(URL, { method: 'PUT', body })` 直传 S3（3 路并发 + 进度 + 重试）。
3. ≥100MB 走 multipart：`/multipart/init` → `/multipart/part`（分段预签名）→ `/multipart/complete`。
4. 后端全程不接触对象字节，仅生成签名。

## 6. 配置体系

所有配置通过环境变量注入（`S3C_*`），支持 `.env`（查找顺序：`S3C_ENV_FILE` → CWD → 可执行文件同目录；真实环境变量优先）。完整矩阵见 [README.md](../README.md#配置服务端)。

## 7. 关键取舍（详见 ADR）

| 决策 | 理由 |
|---|---|
| 存储不可用时硬失败而非降级 | 降级只读会导致写丢失（[ADR-002](decisions/0002-store-fail-closed.md)） |
| SSRF 放行私网/回环 | 自托管（MinIO/RustFS/局域网）是主场景（[ADR-003](decisions/0003-ssrf-private-allow.md)）；可用 `S3C_SSRF_DENY_PRIVATE=1` 切到拒绝私网 |
| 桌面端无 IPC | 攻击面最小化（[ADR-001](decisions/0001-desktop-no-ipc.md)） |
| 前端生产依赖仅 vue | 供应链最小化（[ADR-004](decisions/0004-minimal-frontend-deps.md)） |
