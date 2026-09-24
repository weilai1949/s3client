# s3clinet 功能大全（Features）

> 本文件是 s3clinet 的**单一事实来源**：产品能力总览 + 全部已完成修复 / 优化记录。
> 已把散落在各评估文档与 [`CHANGELOG.md`](../CHANGELOG.md) 中的「已实现 / 已修复 / 已完善」功能统一汇总于此；CHANGELOG 仍保留逐字发布历史。
>
> - 待处理事项：[`todolist.md`](todolist.md) · 发版历史：[`CHANGELOG.md`](../CHANGELOG.md) · 综合评估：[`assessment.md`](assessment.md)
> - 接口细节：[`api.md`](api.md) · 错误约定：[`errors.md`](errors.md) · 开发规范：[`development.md`](development.md) · 安全设计：[`threat-model.md`](threat-model.md) · Nginx 部署：[`deploy/nginx/README.md`](../deploy/nginx/README.md)
>
> 最后更新：2026-09-22（`v1.0.0` 之后的 Unreleased 区间；含分支状态审查 P0 / §三 / P1 / P2 四轮处置 + §7.4 门禁盲区收尾）

## 目录

- [一、产品功能](#一产品功能)
  - [1. 账号管理](#1-账号管理) · [2. 存储桶与桶属性](#2-存储桶与桶属性) · [3. 对象浏览与检索](#3-对象浏览与检索)
  - [4. 上传](#4-上传) · [5. 下载与预览](#5-下载与预览) · [6. 对象操作](#6-对象操作)
  - [7. 版本与回收站](#7-版本与回收站) · [8. 迁移与增量同步](#8-迁移与增量同步)
  - [9. 存储驱动与数据安全](#9-存储驱动与数据安全) · [10. 服务端安全与鉴权](#10-服务端安全与鉴权)
  - [11. API 与契约](#11-api-与契约) · [12. 前端体验与无障碍](#12-前端体验与无障碍)
  - [13. 桌面端](#13-桌面端) · [14. 部署、CI 与工程化](#14-部署ci-与工程化)
- [二、已完成修复与优化](#二已完成修复与优化) — A 本轮增量 · B 驱动去重明细 · C 全方位评估 58 项 · D v1.0.0-rc1 评估 21 项 · E Optional/Nit 长尾 · F 历史版本全量台账（0.1.0→v1.0.0-rc1） · G Unreleased · H–Y 各轮处置台账
- [三、质量与覆盖率现状](#三质量与覆盖率现状)

---

## 一、产品功能

### 1. 账号管理
| 能力 | 说明 |
|---|---|
| 账号 CRUD | 增删改查；ID 由服务端生成，忽略客户端提交的 id；`Create` 存防御性副本 |
| 连通性测试 | `HeadBucket` 探测，返回 `ok` / `error` |
| 预览桶（不落库） | 用临时凭据 `ListBuckets`，不写入存储 |
| 服务商预设 | 按「兼容 / 国内 / 国外」分组（MinIO、阿里 OSS、腾讯 COS、华为 OBS、火山 TOS、AWS、R2、Wasabi、B2、Spaces 等） |
| 连接参数 | endpoint / publicEndpoint / region / AK / SK / bucket / pathStyle / useSSL |
| 密钥不出口 | 响应为 `AccountView`：不包含 `secretKey`，仅 `secretSet: boolean`；请求侧仍以 `secretKey` 提交 |

### 2. 存储桶与桶属性
| 能力 | 说明 |
|---|---|
| 桶列表 / 创建 / 删除 | 含桶名严格校验（长度、首尾字符、连续 `..`/`.-`/`-.`） |
| 桶属性 | 区域、创建时间、版本控制状态 |
| 版本控制 | 一键开启 / 暂停 |
| 生命周期 | 前缀过期删除规则读写 |
| 服务端加密（SSE） | 读写 / 开关 |
| CORS 规则 | 读写 / 开关 |
| 静态网站托管 | 读写 / 开关 |
| 桶策略 | 可视化编辑器（Statement 表单 + 4 模板 + 实时 JSON 预览）+ 原始 JSON 回退 |
| 桶标签 | 读写 / 开关 |

### 3. 对象浏览与检索
| 能力 | 说明 |
|---|---|
| 列表分页 | `ListObjectsV2`，「加载更多」追加 |
| 目录浏览 | 前缀 + 分隔符（默认 `/`）；20 万行虚拟滚动（窗口化渲染） |
| 对象详情 | 大小 / ETag / Content-Type / 存储类型 / 元数据 |
| 存储类型 | 列表 / 详情 / 版本中展示并一键切换（`CopyObject` 副本到自身） |
| 新建文件夹 / 重命名 / 移动 | 前缀级操作 |

### 4. 上传
| 能力 | 说明 |
|---|---|
| 小文件直传 | 服务端 v4 签名 PUT URL，浏览器直传 S3（2 路并发、进度、失败重试） |
| 大文件分段 | `≥100MB` 自动切 10MB/段、4 路并发直传，任一段失败即 abort 清理 |
| 上传队列 | 面板与对象区共享同一状态机；支持取消 / 重试；`cancelled` 终态不自动重启 |
| 前缀追加 | 上传时可给 key 追加前缀 |

### 5. 下载与预览
| 能力 | 说明 |
|---|---|
| 一键下载 | 短时效签名 GET URL |
| 分享链接 | 1 小时签名 URL 复制；批量复制链接并行 presign（`Promise.allSettled`） |
| 安全代理 | 下载 / 预览走服务端代理（三模式 + 类型拒渲染 + sandbox），key 拒绝控制字符 |
| ZIP 打包 | 批量下载流式打包（`io.Copy` 零整对象缓冲，滚动写超时） |
| blob 回收 | `revokeObjectURL` 延迟 60s，避免中断尚未开始的下载 |

### 6. 对象操作
| 能力 | 说明 |
|---|---|
| 删除 | 单个 / 批量（`≤1000` 一次，超限 400）/ 前缀递归 |
| 复制 / 移动 | 跨桶、文件夹递归；同桶文件走 `CopyObject`；移动 = 复制成功后删源 |
| 设置 HTTP 头 | `set-headers`（Cache-Control / Content-Disposition 等） |
| ACL | 读 / 写，私有 ↔ 公共读，复制公开访问链接 |
| 标签 | 读 / 写 / 清空，键值行编辑 |
| 批量改元数据 | ACL / 标签（替换或清空）/ 存储类型；4 路有界并发，进度 + 失败明细；全空标签禁止提交 |
| 批量 key 上限 | `BATCH_META_MAX_KEYS=10000`，超限抛错 |

### 7. 版本与回收站
| 能力 | 说明 |
|---|---|
| 版本列表 | `ListObjectVersions`，含删除标记；`keyMarker`/`versionIdMarker` 翻页（≤20 页），仍截断时显式提示 |
| 删除指定版本 | `DeleteObject` 带 `versionId` |
| 版本回滚 | 恢复某版本为当前 |
| 版本比较 / 详情 | 选两个内容版本做差异比对 |
| 回收站 | 独立菜单，列出全部删除标记，一键还原（撤销删除）/ 彻底清除（永久删除该 key 全部版本） |

### 8. 迁移与增量同步
| 能力 | 说明 |
|---|---|
| 同 endpoint 复制 | 服务端 `CopyObject` |
| 跨 endpoint 复制 | `GetObject` → `PutObject` 流式转发（保留 Content-Type 与元数据，64MB 分段有界复用） |
| 单 / 批量迁移 | 逐 key 执行，失败继续并汇总；异步任务 + SSE 进度 |
| 增量同步 | `POST /api/migrate/sync`，按 `etag`（默认）/ `size_mtime` / `always` 比对，仅复制差异对象 |
| 前缀追加 | 迁移时可给目标 key 追加前缀 |
| 进度 | `RunBatch` 无缓冲 results 通道，边收边回调（内存 O(workers)） |

### 9. 存储驱动与数据安全
| 能力 | 说明 |
|---|---|
| 三驱动 | `json`（默认）/ `sqlite` / `encrypted`，统一 `store.Open` 入口 |
| 共享实现 | `json` 与 `encrypted` 统一为 `Store` + 单一 `storeCodec`（strict 区分），复用 `fileStore`（锁 / CRUD / 回滚 / 快照单点实现） |
| 落盘加密 | `S3C_STORE_KEY` → Argon2id + AES-256-GCM，当前写入格式 `S3C3`（Argon2id 参数随文件头保存，故可在不破坏既有库的前提下调参）；兼容读旧 `S3C2` |
| 原子写 | 临时文件 + 写后 rename；写前清残骸（`O_EXCL`）；0600 权限 |
| 兼容性 | `json` 驱动同时读明文与历史 S3C2 文件（permissive，写盘换新盐）；`encrypted` 严格模式只接受加密信封（S3C3 当前 / S3C2 兼容），盐建时随机并随文件复用 |
| 失败回滚 | `Create`/`Update`/`Delete` 持久化失败回滚内存，避免内存/磁盘漂移 |

### 10. 服务端安全与鉴权
| 能力 | 说明 |
|---|---|
| 鉴权 | Bearer 多 token（`S3C_TOKEN` 逗号分隔，支持轮换）；凭证常量时间比较，scheme 大小写不敏感（RFC 7235） |
| 启动校验 | 短 token（< 16 字符）拒绝启动；非回环未设 token 拒绝启动 |
| CSRF 双防 | CORS 白名单外 Origin 直接 403 + `readJSON` 要求 **非空** `Content-Type` 必须为 `application/json`（缺省放行，仍受 JSON 解码器约束）触发预检 |
| SSRF 双校验 | 创建时校验 + 拨号时二次校验（防 DNS rebinding）；禁重定向；`Proxy=nil` 防环境变量代理绕过 |
| 响应头 | CSP（默认 `connect-src 'self'`，`S3C_CSP_CONNECT_SRC` 可放宽）+ 安全响应头 |
| 限速 | 按客户端 IP 限速；`X-Forwarded-For` 仅当直连对端命中 `S3C_TRUSTED_PROXIES` 时采信首段（默认空 = 不信任，防直连伪造绕过） |
| 端点门控 | `/api/openapi.json`（`withOpenAPIGate`）与 `/api/metrics`（`withMetricsGate`）默认 404 |
| 输入防护 | 路径遍历 / 控制字符 key 拒绝；user metadata 边界 400（非 500） |
| 请求 ID | 回显 `X-Request-ID` 仅限 ≤128 可见 ASCII，否则服务端生成，防日志/响应头注入 |
| 错误脱敏 | `UserMessage` 防腐层统一映射，sentinel（`ErrObjectTooLarge` / `ErrSourceDeleteFailed`）替代文案匹配 |
| 账号密钥不出口 | 账号响应为 `AccountView`：**不包含 `secretKey`**，仅 `secretSet: boolean` 表示是否已设置；存储层 `Sanitized()` 占位不再进入 HTTP 响应 |

### 11. API 与契约
| 能力 | 说明 |
|---|---|
| REST 端点 | **70** 个 `/api/*` 端点（含 health/metrics/openapi.json） |
| OpenAPI 3.0.3 | `/api/openapi.json` 自动生成，按域登记（accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system） |
| 契约测试 | `routes.go` ↔ 规范双向一致、operation 完整性、路径参数、`$ref` 可解析、`MarshalJSON` 确定性 |
| 错误约定 | 统一 JSON 错误体，见 [`errors.md`](errors.md) |
| 静态托管 | SPA fallback，`/api/*` 未匹配直接 404（不返回 HTML） |

### 12. 前端体验与无障碍
| 能力 | 说明 |
|---|---|
| 技术栈 | Vue 3 + Vite + TS；`dependencies` 仅 `vue` |
| 国际化 | zh / en 按域拆分（`i18n/messages/*`） |
| 主题 | 明暗主题切换 |
| 快捷键 | 全局键盘 + 面板激活守卫（KeepAlive 下不误触） |
| 竞态防护 | `loadSeq` / `detailSeq` 序号守卫 + `AbortController`；SSE 卸载主动 abort |
| 无障碍 | `label` / `aria-label` / `aria-live` / `role="alert"` / 焦点陷阱 |
| 预览管线 | 三模式代理 + 类型拒渲染 + `sandbox` |
| 弹窗 | `ModalDialog` footer 插槽、z-index 分层、右键菜单溢出滚动 |
| 加载态 | 骨架 / 进度条 / 错误横幅与重试 |

### 13. 桌面端
| 能力 | 说明 |
|---|---|
| Tauri 2 | B/S 架构，**无 IPC**，前后端全 HTTP |
| 权限收敛 | `withGlobalTauri` 关闭 + capability 裁剪 |
| 分发 | CI 交叉构建 `.exe`(NSIS) / `.deb` / `.dmg`，挂 GitHub Release，附 SHA256SUMS |

### 14. 部署、CI 与工程化
| 能力 | 说明 |
|---|---|
| 容器 | Alpine 运行镜像；非 root；healthcheck；compose base / prod / tls 三套 |
| 安全默认 | 回环发布、token 强制注入、server/rustfs/nginx 显式内存上限（nginx 128M） |
| CI | `gofmt` / `go vet` / `go test -race` / 覆盖率门禁 / Docker 构建 / Trivy / Playwright E2E |
| CI 双平台 | 同一套门禁同时落 GitHub Actions 与 GitLab CI（`.gitlab-ci.yml`）：server / web / docker / desktop / desktop-build（手动）/ rustfs-e2e / playwright-e2e 逐 job 对应，命令与阈值一致；`release-desktop.yml` 有意不镜像（GitHub Release 专属）。本地用 `make gcl` / `make gcl-docker`（pin `gitlab-ci-local@4.75.1`，`.gitlab-ci-local-env` 默认挂 docker.sock；Trivy DB 可用 `TRIVY_DB_REPOSITORY` / `.gitlab-ci-local-variables.yml.example`） |
| 供应链 | GitHub Actions 全部 pin SHA；Trivy 用官方镜像 `aquasec/trivy`（`aquasecurity/trivy` 仅存在于 ghcr.io，写错会以 exit 125 失败而非漏洞失败）；发布产物附 SHA256SUMS |
| 本地 | Makefile（`test` / `test-cover` / `web-test` / `vet` / `install-hooks` / `gcl` / `gcl-list` / `gcl-docker`）；pre-commit（gofmt + vet + 前端 typecheck） |
| 版本 | `scripts/release-version.sh` 同步 8 处版本号 |

---

## 二、已完成修复与优化

> 状态图例：✅ 已完成 · ➖ 已评估 / 无需改动 · ⏳ 说明见备注。

### A. 2026-09 本轮增量（store 去重 / OpenAPI 契约 / 长尾）

| 项 | 状态 | 内容 |
|---|---|---|
| 双存储驱动重叠 | ✅ | 抽取共享 `fileStore` + `fileCodec`（净减约 331 行）后**再收敛完成**：`EncryptedStore`/`encryptedCodec` 并入统一 `Store` + 单一 `storeCodec`（strict 区分 json permissive / encrypted 严格），`encrypted.go` 删除、`NewEncrypted` 返回 `*Store`；磁盘格式 / 错误文案 / 加密语义不变 |
| `Create` 跨驱动一致性 | ✅ | 统一存防御性副本（此前 json 驱动别名调用方指针，encrypted 已复制） |
| 跨驱动对照测试 | ✅ | `crossdriver_test.go`：三驱动 CRUD 等价、遗留 S3C2 信封可读、落盘字节布局 |
| OpenAPI 覆盖率 | ✅ | `internal/openapi` 与 handler 内全部 OpenAPI 函数 **100.0%** statement |
| OpenAPI 契约强度 | ✅ | `openapi_contract_test.go`：69 条路由↔规范双向一致（漏登记/陈旧都红灯）、operation 完整性 + 唯一 operationId、`{param}` 必填 path 参数、`$ref` 可解析（含解析器自检）、`MarshalJSON` 确定性（含并发）、顶层 3.0.3 形状 |
| OpenAPI `$ref` 接线 | ✅ | 新增 `openapi.Ref()` / `Param.Ref` / `Response.Ref`；共享 schema / parameter / response 经 `refSchema` / `refParam` / `refResp` 全部接线（109 处 `$ref`、12 个唯一目标），契约测试支持 `$ref` 解析并新增「components 无死片段」断言（全局错误词汇除外） |
| 错误映射去字符串匹配 | ✅ | sentinel `ErrObjectTooLarge` / `ErrSourceDeleteFailed` + `errors.Is`；`PutObject`/`CopyObject` 归一 S3 `EntityTooLarge` |
| `X-Request-ID` 加固 | ✅ | ≤128 且仅可见 ASCII，非法值服务端生成 UUID |
| Bearer scheme | ✅ | RFC 7235 大小写不敏感，凭证仍常量时间比较 |
| `.env` 解耦 CWD | ✅ | `S3C_ENV_FILE`（唯一来源）→ CWD → 可执行文件同目录；真实环境变量优先 |
| 流式复制缓冲 | ✅ | 有界复用（常驻上限 1×64MB = 512M 容器预算 1/8），成功与 abort 路径都归还 |
| `metadata.go` 变量遮蔽 | ✅ | 循环变量 `r` → `row`（不再遮蔽 `*http.Request`） |
| SSE 卸载中断 | ✅ | `DestDialog` 迁移进度流、`useObjectActions` 前缀删除流 `onBeforeUnmount` 主动 abort |
| VersionsDialog 分页 | ✅ | `keyMarker`/`versionIdMarker` 翻页（≤20 页），其余部分显式截断提示 |
| 非空断言清理 | ✅ | 剩余 3 处 `ctx.account.value!` + `{} as KeyBindings` 改为守卫 / 可选类型 |
| ESLint 收紧 | ✅ | `@typescript-eslint/no-explicit-any` warn → error（0 违规） |
| gofmt 对齐 Go 1.26 | ✅ | 7 个既有文件按新字段对齐 / 文档注释规则格式化，CI `gofmt -l .` 恢复干净 |

### B. 双存储驱动去重明细

| 文件 | 变化 |
|---|---|
| `apps/server/internal/store/filestore.go` | 新增（213 行）：共享内存状态 + 锁 + CRUD + 回滚 + `snapshotLocked`/`persistLocked` |
| `apps/server/internal/store/crypto.go` | 新增（66 行）：S3C2 常量、`deriveKey`、`envelope`、AES-256-GCM 加解密 |
| `apps/server/internal/store/store.go` | 244 → 93 行：仅选 `storeCodec`（明文 + 兼容 S3C2） |
| `apps/server/internal/store/encrypted.go` | 267 → 80 行（去重后仅选 `encryptedCodec`）；**再收敛后已删除**，逻辑并入 `store.go` 的 `storeCodec`（strict） |
| `apps/server/internal/store/account_store.go` | `ErrNotFound` 移到接口旁 |
| `open.go` / `sqlite.go` / `atomic.go` | 未改动（文件路径、返回类型、故障注入 seams 全部保留） |

> 再收敛已完成（本轮）：`encryptedCodec` / `EncryptedStore` 并入统一 `storeCodec`（`strict` 区分
> permissive/严格语义），`encrypted.go` 删除，`NewEncrypted` 返回 `*Store`；行为、错误文案、磁盘格式不变。

### C. 全方位评估 58 项（2026-04-19 全部落地）

#### 安全性
| 项 | 状态 | 修复 |
|---|---|---|
| S-1 明文 SecretKey | ✅ | JSON Store 落盘 AES-256-GCM（`S3C_STORE_KEY` 派生，S3C2｜salt｜ciphertext，兼容明文旧文件） |
| S-2 XFF 限速绕过 | ✅ | `clientIP` 优先 `X-Forwarded-For`（逗号分割），回退 `RemoteAddr` |
| S-3/S-6 openapi.json 泄露 | ✅ | `S3C_EXPOSE_OPENAPI` 门控（默认 404），开启后需 Bearer 鉴权；移出 `withAuth` 绕过 |
| S-4 parsePolicy 误判 | ✅ | UI 明确提示「不支持结构需用原始 JSON 编辑」，消除静默误解析 |
| S-5 批量 key 无上限 | ✅ | `BATCH_META_MAX_KEYS=10000`（超限抛错） |
| S-7 testAccount 泄露 | ➖ | `UserMessage` 防腐层已映射为通用消息，无内部细节泄露 |
| S-8 proxy key 遍历 | ✅ | 拒绝含控制字符的 key（S3 对 `..` 字面段安全） |
| S-9 CSP connect-src | ✅ | 默认收紧为 `'self'` + 本地 Tauri 后端；`S3C_CSP_CONNECT_SRC` 可显式放宽 |

#### 架构与设计
| 项 | 状态 | 修复 |
|---|---|---|
| A-1/M-1 openapi_register 拆分 | ✅ | 拆为 9 个 `openapi_register_*.go` 按域文件 |
| A-2/M-2 Handler 拆分 | ➖ | 评估后维持：方法已按域分文件，拆子 handler 收益低、风险高 |
| A-3/P-6/C-7 共享 HTTP 客户端 | ✅ | `MaxConnsPerHost=32`，批量操作不无限占 fd |
| A-4 SyncKeys 内存 | ✅ | `indexDst` 增加 100k 硬上限（与 listAll 一致） |
| A-5/P-5/C-6 results 缓冲 | ✅ | `RunBatch` results 通道改无缓冲：内存 O(workers)，进度即时回调 |

#### 性能与并发
| 项 | 状态 | 修复 |
|---|---|---|
| P-1 awsconfig 不可取消 | ✅ | 10s timeout context（`cfgCtx`） |
| P-2 SameEndpoint 回退 | ✅ | region-aware：两端均为空时仅当 region 相同才视为同端 |
| P-3 zip goroutine 泄漏 | ✅ | `ctxCancelReader` + `context.AfterFunc` 中断阻塞读 |
| P-4/C-5 列举间无 ctx 检查 | ✅ | `SyncKeys` 列举后检查 `ctx.Err()`，取消即返回 |

#### 测试质量
| 项 | 状态 | 修复 |
|---|---|---|
| T-1 zip cancel 时序 | ✅ | `ready.WaitGroup` 同步 + 取消路径确定性触发 |
| T-2 batchMetadata mock | ✅ | `vi.mocked` 类型化 + 签名对齐 |
| T-3/T-4 测试全局变量 | ✅ | `syncStore` 加 mutex（`sync_test` 已有 `s3FakeMu`） |
| T-5 同步测试缺口 | ✅ | 新增 CompareSizeTime / prefix 过滤 / 跨端点 StreamCopy 三测试 |
| T-6 e2e 覆盖缺口 | ✅ | `e2e/features.spec.ts` 10 用例；15 passed / 0 skipped |
| T-7 Makefile | ✅ | `test` 加 `-race`；新增 `test-cover` / `web-test-cover` |
| T-8 store 编译回归 | ✅ | 复用 `encrypted.go` 共享加密助手；store 覆盖率提升 |
| T-9 前端覆盖率 | ✅ | 11.28% → 95.27% statements；36 组件 + 全部 composables 覆盖；顺带修 3 个缺陷（dropzone 点击递归、MigratePanel ResizeObserver 时机、deleteServer 当前服务器 applyProfile） |

#### 文档与 DevOps
| 项 | 状态 | 修复 |
|---|---|---|
| D-1 pre-commit | ✅ | `.githooks/pre-commit`（gofmt + go vet + fe typecheck）+ `make install-hooks` |
| D-2 e2e 重试 | ✅ | Playwright CI retries 2 |
| D-3 Makefile 缺失 | ➖ | 根 Makefile 存在且已强化 |
| D-4 errors.md 未更新 | ✅ | 补充 migrate/sync 与 openapi.json 的错误约定 |

#### 前端 UX 与无障碍
| 项 | 状态 | 修复 |
|---|---|---|
| UX-1/3 tag 输入 label | ✅ | `sr-only` label + i18n（tagKey/tagValue） |
| UX-2 策略编辑器 label | ➖ | 已由外层 `<label>` 包裹（核查确认满足） |
| UX-4 watcher 级联 | ✅ | 移除 deep `watch(doc)`，`prevRaw` 守卫防 dirty 重置 |
| UX-5 进度条 | ✅ | `<progress>` + onProgress 实时推进 |
| UX-6/E-1 死代码 | ✅ | catch 改通用 `step:'batch'` |
| UX-7 copySelectedLinks | ✅ | `Promise.allSettled` 部分成功 |
| UX-8 runUpload 队列 | ✅ | 清空移入 try（完成后清理） |
| UX-9/M-4 templateLabels | ✅ | 从 `POLICY_TEMPLATES` 派生 |
| UX-10 全选 label | ➖ | 已由 `<label>` 包裹 |
| UX-11 emoji 按钮 | ✅ | `📊` + `aria-label` |
| UX-12 emoji 提示 | ✅ | `aria-hidden` + i18n |
| UX-13 错误截断 | ✅ | 「显示全部 n 条 / 收起」 |
| UX-14 空标签提交 | ✅ | `hasTagChange`（全空禁止提交） |
| M-3 缩进 | ✅ | 统一 4 空格 |

#### API 设计 / 错误处理
| 项 | 状态 | 修复 |
|---|---|---|
| API-1/4 批量元数据端点 | ➖ | 前端编排用的单对象端点（object-acl / tags / storage-class）已在 OpenAPI 登记 |
| API-2 migrate/sync 登记 | ➖ | 已在 OpenAPI 登记 |
| API-3 maxBody 偏小 | ✅ | 4MB → 8MB |
| E-2 removeSelected toast | ✅ | `objects.toastDeleteFailed` |
| E-3 copySignLink toast | ✅ | 成功带 key、失败新 toast |
| E-4 错误横幅 | ✅ | `role="alert"` + 关闭按钮 |
| E-5 错误泄露 | ✅ | `toErrorMessage` 防腐确认 |

#### 附带修复的真实缺陷
- **ModalDialog 缺少 footer 插槽**：BatchMetadataDialog 确定/取消按钮永不渲染 → 已加 `<slot name="footer" />`。
- **ConfirmDialog / PromptDialog z-index 相同**：弹出在 ModalDialog 背后不可见 → z-index 200 → 300。
- **ObjectContextMenu 溢出视口**：低处菜单项不可点击 → `max-height` + 滚动 + 防负值。
- **BatchMetadataDialog open 受控化**：父级 `:open` 而非 `v-if`，保留进行中状态。

### D. v1.0.0-rc1 评估 21 项（含后续补修）

| 严重度 | 项 | 状态 |
|---|---|---|
| CRITICAL | `gaps_test.go` `release` channel 未关闭 / `done` channel 竞争 / 冗余 `sync.Once` | ✅ |
| HIGH | BatchMetadataDialog 空 tags 在 replace 模式静默清空服务端标签 | ✅ |
| HIGH | BatchMetadataDialog 异常无用户反馈 | ✅ |
| MEDIUM | `zip.go` `CreateHeader` 失败时 `item.body` 泄漏 | ✅ |
| MEDIUM | `zip.go` manifest 文件句柄无效 `defer` | ✅ |
| MEDIUM | `zip.go` `io.Copy` 失败后 ZIP 流状态未定义 | ✅ |
| MEDIUM | `batchMetadata.ts` `acl` 类型过宽 | ✅ |
| MEDIUM | `batchMetadata.test.ts` 未使用参数 | ✅ |
| LOW | BatchMetadataDialog 无障碍（aria-label / aria-live） | ✅ |
| LOW | `batchMetadata.ts` `accId` 别名 / 冗余 `as` / 空 keys 守卫 | ✅ |
| LOW | `zip.go` `ctxReader` 无法中断阻塞式底层 Read | ✅（后续 `ctxCancelReader` + `context.AfterFunc` 专项修复） |
| LOW | `zip.go` producer goroutine 泄漏 | ➖（复核为无泄漏） |
| MEDIUM | `batchMetadata.ts` 计数器并发非原子 | ➖（JS 单线程事件循环保证原子性） |
| INFO | 「GET /api/accounts 返回明文 SecretKey」指控 | ➖（误报：所有出口均 `Sanitized()`；后续契约收敛为 `AccountView.secretSet`，见 §10「账号密钥不出口」） |
| — | 其余 Info 项 | ✅ / ➖（见 `CHANGELOG.md` 对应版本段） |

### E. Optional / Nit 长尾采纳情况（v1.0.0-rc1 评估）

| 项 | 状态 | 说明 |
|---|---|---|
| encrypted Update 失败回滚内存 | ✅ | 已与 json 驱动对齐 |
| 流式复制 64MB/worker 缓冲 | ✅ | 改为有界复用（常驻 ≤64MB） |
| `/api/metrics` 鉴权暴露 | ✅ | `withMetricsGate` 默认 404，`S3C_EXPOSE_METRICS=1` 显式开启 |
| migrate SSE 写超时 | ✅ | 滚动 `SetWriteDeadline`（`streamIdleTimeout`） |
| worker-pool 仓内重复 4 份 | ✅ | 收敛为 `service.RunBatch` |
| `ProxyFromEnvironment` 绕过 SSRF | ✅ | transport 显式 `Proxy=nil`，测试 `ssrf_gap_test.go` |
| deleteObjects key 数上限 | ✅ | `maxDeleteKeys=1000`，超限 400 |
| user metadata 以 500 返回 | ✅ | `ValidateUserMetadata` → 400 |
| `ssrf.go` `To4` 死分支 | ➖ | 代码中已无 `To4`；拨号循环分支互异 |
| showDetail 无 seq 守卫 | ✅ | `useObjectActions` `detailSeq` |
| SSE 订阅未在卸载 abort | ✅ | DestDialog / useObjectActions 已补 |
| VersionsDialog 忽略分页截断 | ✅ | 翻页 + 截断提示 |
| copySelectedLinks 串行 presign N+1 | ✅ | `Promise.allSettled` 并行 |
| 面板挂载隐式切全局账号 | ➖ | 复核：onMounted/onActivated 仅 reload，不 `selectAccount` |
| `no-explicit-any:'off'` | ✅ | off → warn → **error**（0 违规） |
| X-Request-ID 超长回显 | ✅ | 长度 + 可见 ASCII 校验 |
| Bearer scheme 大小写敏感 | ✅ | `strings.Cut` + `EqualFold` |
| `metadata.go` 循环变量遮蔽 | ✅ | 循环变量改名 |
| `.env` 按 CWD 相对加载 | ✅ | `S3C_ENV_FILE` + 可执行文件同目录候选 |
| UserMessage 字符串匹配 `"exceeds 5GB"` | ✅ | sentinel + `errors.Is` |
| `validBucketName` 首/尾字符 | ✅ | 拒绝首尾 `.`/`-` 与连续分隔符 |
| 前端 `api.ts` 死代码 | ✅ | `requestBlob` / `downloadZip` 已删 |
| 同批 files 排序两次 | ✅ | 已移除重复排序 |
| 13 处 `ctx.account.value!.id` | ✅ | 收敛入口，剩余 3 处本轮清理 |
| blob 下载同步 `revokeObjectURL` | ✅ | 延迟 60s |
| `{} as KeyBindings` cast hack | ✅ | 改 `const bindings: KeyBindings = {}` + 可选字段 |
| pnpm 版本口径不一（9 vs 11） | ✅ | 全仓统一 `9.15.0`（`packageManager` + 各 workflow + Dockerfile）；`apps/desktop/package.json` 亦补 `packageManager`，由 `repo_infra_gate_test.go` 断言两个 package.json 与 CI 的 pnpm 版本一致 |
| Node 只 pin 大版本 | ✅ | 统一到 `24.21.0`（`node-version` / `node:24.21.0-alpine` / `node:24.21.0-bookworm` / NodeSource `nodejs=24.21.0-1nodesource1`），由 `TestNodePinnedToPatchVersion` 守住 |
| Rust 工具链未 pin | ✅ | `dtolnay/rust-toolchain` 由 `stable` 浮动分支 tip 改为版本分支 SHA（`ce678459… # 1.98.1`），新增 `apps/desktop/src-tauri/rust-toolchain.toml`（`channel = "1.98.1"`），GitLab 镜像 `rust:1.98.1-bookworm`；由 `TestRustToolchainIsVersionPinned` 守住 |
| Trivy 固定 0.58.1（落后 + `--vuln-type` deprecated） | ✅ | 升到 `0.74.0` 并加 digest 双 pin（`aquasec/trivy:0.74.0@sha256:62b1e65e…`），`--vuln-type` → `--pkg-types os,library`；由 `TestTrivyImageIsVersionAndDigestPinned` 守住 |
| `.dockerignore` 漏覆盖率/缓存目录 | ✅ | 补 `apps/web/coverage`、`.pnpm-store`、`test-results`、`playwright-report`、`.run/`、`.cargo/`、`.trivy-cache/`、`.gitlab-ci-local/`、`coverage.out`/`.txt`、`*.tsbuildinfo`、`*.log` 等；由 `TestDockerignoreExcludesBuildArtifacts` 守住（注释行不计入） |
| README 示例绑回环 | ✅ | `-p 127.0.0.1:8080:8080` + 说明 |
| nginx 容器缺内存限制 | ✅ | 两个 compose 均 `memory: 128M` |
| `\|\| true` 吞安装失败 | ➖ | 已无 install 命令使用；仅保留 read/kill 的良性回退 |
| Makefile 每次启动 `go mod tidy` | ✅ | 移除隐式 tidy，新增 `make tidy` |

### F. 历史版本功能与修复全量台账（0.1.0 → v1.0.0-rc1）

> 从 [`CHANGELOG.md`](../CHANGELOG.md) 的 22 个版本段完整汇总；逐字发布历史仍以 CHANGELOG 为准。

#### v1.0.0-rc1（2026-09-02）
- 桌面发版 CI：`release-desktop.yml` 在 `v*` tag 交叉构建 NSIS `.exe` / `.deb` / `.dmg` 并上传 GitHub Release。

#### v1.0.0-rc0（2026-09-02）
- 不做服务降级 / 无历史格式兼容：`/api/health` store 失败 503 + `status:error`；`encrypted` 仅 `S3C2`；JSON 账号文件 0600；移除兼容 shim 与旧后端注释。
- 异步 SSE：`copy-prefix/async`、`copy-objects/async`、`delete-prefix/async`；多 token 轮换（`S3C_TOKEN` 逗号分隔）。
- 防腐层与 service 收口：`UserMessage` / `HTTPStatus` / `IsNotFound` / `ObjectStream`；迁移引擎、ZIP、`JobRegistry` 迁出 handler；短写校验。
- 可观测：`GET /api/metrics`（Prometheus 文本）、`X-Request-ID`、可选 `S3C_LOG_JSON=1`。
- 安全：SSRF 禁重定向 + 拦截链路本地 / 云元数据；非回环无 token 拒绝启动；IP 令牌桶限速约 120/min；错误映射统一 `s3HTTPStatus`。
- 流式：写超时改 5 分钟滚动空闲；>5GB 迁移 multipart；同端点 `EntityTooLarge` 自动回退 multipart。
- 存储：`S3C_STORE_DRIVER=sqlite`（纯 Go `modernc.org/sqlite`，WAL）；`encrypted` + `S3C_STORE_KEY`（AES-256-GCM，Argon2id / `S3C2`）；移除遗留明文自动导入。
- 工程化：`s3wrap` 拆六文件（各 <400 行）；错误处理收敛；CI gofmt / race / cover / golangci + Web test / lint + E2E workflow；`release-version.sh`；`make test-all`。
- 前端：Escape 键栈、`KeepAlive max=5`、分段上传段级重试、ZIP File System Access 流式落盘、Hash 深链接、i18n 脚手架、token 可选 sessionStorage。
- 运维：`docker-compose.prod.yml`、TLS 叠加、nginx `proxy_buffering off`、health 探测 store、SQLite `user_version`、dependabot。

#### v1.0.0-20260901182023（2026-09-01）
- **Batch1 存储类型 + 版本比较 + 一键还原**：`HeadObject` 返回 `storageClass`；`POST /storage-class` 一键切换（标准 / 低频 / 单区 / 智能分层 / 归档）；版本比较（并排元数据 + 逐行内容差异，二进制或 >2MB 仅比元数据）；`POST /delete-marker/restore` 撤销删除。
- **Batch2 桶管理菜单**：7 页签（概览 / 生命周期 / SSE / CORS / 网站托管 / 桶策略 / 桶标签）+ 15 个 `Get/Put/DeleteBucket*` 端点；`isNoSuchBucketSetting` 把「未配置」映射为空响应。
- **Batch3 回收站菜单**：`GET /trash` + `POST /trash/purge`，列全部删除标记、一键还原、彻底清除（永删该 key 全部版本）。
- **四轴评审修复**：Vue `key` 为保留属性导致 7 个弹窗实际不可用 → 统一 `objectKey`（Vue 3.5 SSR 复现验证）；桶设置页签 `watch` 补 `immediate`（此前永不加载、保存会用默认值覆盖）；CORS 非白名单普通请求直接 403 + `readJSON` 强制 JSON；`inline` 代理 MIME 白名单；代理支持 `versionId` + `Accept-Ranges`/416；创建账号忽略客户端 id；ZIP 条目 `/` 与 `\` 双消毒；`.env` 加载；`PurgeObject` 改 `DeleteObjects` 批量 1000/批；`deletePrefix` 页前检查 + 跨页切分；multipart 段号 1..10000 且唯一校验；copy/migrate 4 路有界并发、migrate 限 10k key、failKeys ≤200<sup>†</sup>；列表导航序号防过期响应；上传入队捕获所属桶；焦点陷阱与初始焦点；版本比较下载走服务端代理。
  > <sup>†</sup> **该条当时并不成立**（2026-09-21 复核，docs/archive/review-2026-09-19.md §7.3 D4）：服务端 `BatchResult.FailKeys` 不做截断，只有异步 `delete-prefix` 在 handler 内裁剪，copy/migrate/sync 原样回传——10k 全失败会回约 10 MB 并落盘 `jobs.json`。**2026-09-22 已真正实现**：上限提为 handler 层共享常量 `maxFailKeys = 200` + `capFailKeys`，所有回传 `failedKeys` 的端点（copy-objects 同步/异步、migrate 同步/异步、migrate/sync、delete-prefix 同步/异步）统一裁剪，异步路径在 `jobResultFromBatch` 中于 `Finish`（落盘）之前裁剪；回归测试 `TestOlCopyManyFailKeysAll` / `TestOlMigrateSyncFailKeysCapped` / `TestOlMigrateAsyncFailKeysCappedBeforePersist`。
- 测试与清理：handler 覆盖率 60.3% → 70.8%；AWS SDK 类型外泄清理（`FromS3Object` / `FormatBuckets` / `DescribeACL` / `GranteeLabel`）；`filename*` RFC 5987；store 临时文件 `O_CREATE|O_EXCL`；删除死代码（`PresignGet`、`genSign`、`SignUrlDialog` 等）。

#### 20260901.2（2026-09-01）
- 对象版本：删除指定版本 `DELETE /version`、版本回滚 `POST /version/restore`（`CopyObject` 带 `?versionId=`，版本控制下写出一条新版本）；版本对话框「恢复 / 删除」+ 确认与结果提示；MinIO E2E 验证。

#### 20260901（2026-09-01）
- 桶属性 `GET /bucket-info`（区域 / 创建时间 / 版本控制）+ `PUT /bucket-versioning`（Enabled|Suspended）+ `GET /versions`（版本 + 删除标记 + 游标）；对象右键「版本」。
- 真实 MinIO E2E：预签名 PUT 直传、三段式 Multipart（12MB 组装回读一致）、`PutBucketVersioning` + 多版本覆盖写 + `ListObjectVersions`。
- 重构：`handler.go` 2186 → 131 行按域拆分；`ObjectsPanel.vue` 2041 → 442 行 + `useObjectBrowser` / `useObjectActions` / `usePreview`；`proxyUrl` 抽共享 `proxy.ts`；handler 测试拆分；新增 `agents.md`。
- 正确性：默认桶可空语义（`bucketOr`，缺省桶缺失返回 400）；`downloadZip` 条目脱敏 + 单次上限 1000；CSP；`ReadTimeout`（不设 `WriteTimeout`）。

#### 1.0.0（2026-08-28）
- 复制 / 移动 / 删除跨桶全闭环（`copy-object` / `copy-objects`；移动 = 复制成功后删源，文件夹副本有失败则保留源）；修复跨桶同名重命名被误拒。
- 大文件分段上传四端点（`init | part | complete | abort`），前端 ≥100MB 自动切 10MB/段、4 路并发，任一段失败自动 abort；小文件仍单 PUT。
- 对象 ACL（私有 ↔ 公共读 + 公开链接）、对象标签（键值编辑 / 一键清空 / 兼容 `NoSuchTagSet`）。
- 左侧菜单按「数据操作 / 配置」分组，首次按账号存在路由到对象管理。

#### 0.16.0（2026-08-28）
- 迁移优化：源 / 目标 Bucket 下拉、目标账号默认预选（含「（同账号）」标注）、每批 50 分批进度条、结果弹窗（成功 / 失败 / 总计 + 失败清单 ≤200 + 首个错误 + 「去目标账号查看」跳转）、已选数量与合计大小。

#### 0.15.0（2026-08-28）
- 行内常显「⋯ 更多操作」菜单（与右键菜单一致）；文件行操作列精简为「下载 / 预览 + ⋯」；文件夹行同样提供入口。

#### 0.14.0（2026-08-28）
- 行内「预览」按钮；双击 = 查看（文件预览 / 文件夹进入）；`Enter` 语义由下载改为查看（未知类型自动转下载）。

#### 0.13.0（2026-08-28）
- 编辑对象 HTTP 头（`CopyObject` 到自己 + `MetadataDirective: REPLACE`，保存后详情即时刷新）；生命周期规则管理（规则 ID / 前缀 / 过期天数整体保存；未配置返回空；清空走 `DeleteBucketLifecycle`）。

#### 0.12.0（2026-08-28）
- Bucket 列表页（名称 / 创建时间 / 进入 / 删除）；创建 Bucket（权限 + LocationConstraint，创建后自动进入）；删除 Bucket（非空 409）；对象页统计条；列表 / 网格切换；返回桶列表。

#### 0.11.0（2026-08-28）
- 通用 `ModalDialog` 组件（标题栏 + 关闭 + Esc / 遮罩关闭）；对象详情、签名 URL 结果、账号表单、服务端表单弹窗化。

#### 0.10.0（2026-08-28）
- 工具条拆「位置栏 + 操作栏」；文件管理器式行交互；键盘快捷键 `Enter` / `F2` / `Delete` / `Ctrl/Cmd+A` + 可关闭提示条；账号快速切换；统一 busy 防重复提交；空状态引导；错误条「重试」；跨面板联动 Toast；上传入口更名。

#### 0.9.0（2026-08-28）
- 安全预览：图片（含 SVG，`<img>` 上下文不执行脚本）/ 视频 / 音频（Range 拖动）/ PDF（sandbox iframe）/ 文本与代码 50+ 扩展名（转义展示、超大截断），未知格式转下载。
- 预览 / 下载统一服务端代理 `GET /proxy`：`download` 强制 attachment、`text` 强制 `text/plain + nosniff` 且服务端截断、文件名清洗防 `Content-Disposition` 注入；移除「在新窗口打开」；文本全程 `{{ }}` 转义。

#### 0.8.0（2026-08-28）
- 上传到当前目录（复用 presign PUT，2 路并发，内嵌进度）；面包屑可编辑直达深目录；图片预览；Shift 范围多选。

#### 0.7.0（2026-08-28）
- 服务商预设扩展（国内 COS / OBS / TOS / BOS / OSS / Kodo；海外 AWS / R2 / Wasabi / B2 / Spaces / Linode / Scaleway / Hetzner）+ 兼容 / 国内 / 国外三行分组。
- 批量 ZIP 打包下载（流式、不落盘，失败对象写入 `_下载失败清单.txt`）；递归删除文件夹（≤10 万对象、空前缀拒绝）；递归复制 / 移动文件夹（重叠前缀拒绝）。
- 新增 `delete-prefix` / `copy-prefix` / `download-zip` 端点。

#### 0.6.0（2026-08-28）
- 对象列表「加载全部」（循环分页 ≤200 页）；迁移面板「列出全部文件」（单页 1000、≤20 万对象、全选迁移）；上传完成一键复制 1 小时签名链接；`clipboard.ts` 共享模块。

#### 0.5.0（2026-08-28）
- 新建文件夹（PUT 空对象 + `/` 结尾）；重命名 / 移动（先复制后删源、支持跨桶）；对象详情（`HeadObject`）；批量复制签名链接；通用 `PromptDialog`。

#### 0.4.0（2026-08-28）
- 深色模式三态（跟随系统 / 浅色 / 深色）；自定义确认对话框替换原生 `confirm()`；对象右键菜单；双击交互；列排序（文件夹恒置顶）；本地即时过滤；记住上次账号；「清除已完成」。

#### 0.3.0（2026-08-28）
- OSS 式登录：服务商 + 区域预设（OSS 21 地域 / AWS 20 区域）自动填充 Endpoint 与公网 Endpoint；下载操作；3 路并发上传 + 失败重试。
- **修复跨 endpoint 明文 HTTP 流式迁移失败**：SDK 默认 CRC32 / payload SHA-256 对不可 seek 流报错 → `RequestChecksumCalculation=WhenRequired` + 预置 `UNSIGNED-PAYLOAD`；修复 SPA fallback 失效。
- 安全：Bearer 常量时间比较、`accounts.json` 0600、Update 持久化失败回滚；S3 client 连接 / TLS / 响应头超时 + 连接池上限。
- UI 改版：C 端视觉、CSS token 设计系统、`⚙ 设置` 弹层、连接状态灯、全局 Toast、面包屑 + 骨架屏、拖拽上传、响应式折叠；主题色科技蓝 → 薄荷绿。
- CI 首次接入；`/api/health` 返回版本号；基础安全响应头。

#### 0.2.0（2026-08-22）
- `internal/config` 集中配置 + `.env`；默认回环绑定 / CORS 白名单 / 可选 Bearer 鉴权。
- 容器化：多阶段 `Dockerfile`（非 root、HEALTHCHECK、`/data`）、`docker-compose.yml`、`-healthcheck` 子命令。
- 修复跨 provider 迁移误用 `CopyObject`；跨 endpoint 保留 Content-Type 与元数据；预签名 1s–7 天、`maxKeys` 1–1000；账号更新必填校验；账号存储原子写 + `Create` 失败回滚；region 缺省回退 `us-east-1`。
- 文档：`docs/api.md`、配置矩阵、运行 / 部署说明。

#### 0.1.0（2026-08-22）
- 首个版本：Go 后端（AWS SDK for Go v2，封装 11 个 S3 接口）、Vue3 + Vite + TS 前端（账号 / 列对象 / 直传 / 签名 / 删除 / 迁移 / 加前缀）、Tauri 2 桌面壳（无 IPC，B/S）。

### G. 已记录的 Unreleased 项（已完成）
运行时基镜像 Alpine 3.20、`/api/metrics` 默认关闭、S3C_TOKEN 短口令硬失败、S3 出站禁代理、桶名校验收紧、user metadata 400、delete-objects ≤1000、migrate SSE 写超时、store 失败回滚、错误 sentinel 化、X-Request-ID 加固、Bearer scheme、`.env` 解耦、nginx 内存上限、OpenAPI 自动生成 + 契约测试、**OpenAPI components 接线 `$ref`（109 处引用，消灭 0 引用死代码）**、双驱动去重、存储驱动再收敛（统一 `storeCodec`）、账号响应契约收敛（`secretSet` 替代 `"******"` 占位）、全仓 gofmt 对齐、Playwright E2E、增量同步、桶策略可视化编辑器、批量元数据编辑、存储类型切换、回收站、桌面端 SHA 校验与 actions pin SHA、**gitlab-ci-local 本地体验收口**（`.gitlab-ci-local-env` 默认挂 docker.sock、`make gcl*` pin `@4.75.1`、Trivy DB / 镜像源变量示例）等。完整逐条见 [`CHANGELOG.md`](../CHANGELOG.md)。

### H. 2026-09-16 评估 P0 发布阻塞修复

> 来源：[`assessment.md`](assessment.md) §六 P0（H1 / H4 / M8+C2 / S7+C3）。对应 todolist 原 #5 / #6 / #7 / #15，均已归档。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P0-1 | OpenAPI 契约与真实 handler 字段级不一致（H1） | ✅ | `/api/migrate`、`/api/migrate/async` 请求体字段改为 handler 实际解析的 `sourceAccountId` / `sourceBucket` / `sourceKeys` / `targetAccountId` / `targetBucket` / `targetPrefix`（删除虚构的 `deleteSource` / `storageClass`）；presign `method` 枚举 `GET/PUT/DELETE/HEAD` → `get/put/post`；delete 移除 `versionId`；multipart/part 补 `expiresIn`。新增 `TestOpenAPI_ContractRequestBodyMatchesHandlers` 做「registry schema ↔ handler DTO」字段级三方一致性兜底（含 `$ref` 解引用与 enum 断言） |
| P0-2 | 前端 Token 明文双份落 `localStorage['s3c.servers']`（H4） | ✅ | `s3c.servers` 只存 `{id,name,base}`；token 单副本按 `s3c.token.<serverId>` 独立存储（默认 sessionStorage，仅显式「跨会话保留」时写 localStorage）。旧版本内嵌 token 首次读取时一次性迁移并回填活动服务器 profile；删除服务器同步清理 per-server token |
| P0-3 | `loadAll()` 在 `nextToken` 为空时不发请求却误报「已加载全部」（M8 / C2） | ✅ | 改为 do-while：`nextToken` 为空时仍加载第一页（reset 语义，替换旧列表避免重复项）；某页失败即停止续页且不再发成功 toast；`MAX_ALL_PAGES` 上限保留 |
| P0-4 | 批量复制/移动/删除 SSE 终态检测导致 Promise 悬挂、`opsBusy` 永不复位（S7 / C3） | ✅ | `subscribeMigrateEvents` 在流 EOF 且未收到终态时轮询 `migrateJobStatus`（500ms 间隔、30s 上限）直到 done/cancelled，并在回读结果上合成终态 `status`；连续 3 次回读失败快速 `onError`。三个调用方（`ctxDeleteFolder` / `DestDialog` / `MigratePanel`）由此保证拿到终态或错误，不再永久禁用按钮 |

### I. 2026-09-16 P1 稳定版门槛修复

> 来源：[`assessment.md`](assessment.md) §六 P1 与 §二 L4 / S1。对应 todolist #13 / #14 / #19 / #24，以及 P0-1 的契约收尾。
> 修复后 `govulncheck ./...` 由 6 个可达 stdlib 漏洞降为 **0**；全仓 action SHA 经 GitHub API 核验均有效。
> **P1 至此清零**，`v1.0.0` 稳定版门槛达成。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P1-1 | Go 1.26.5 的 6 个可达 stdlib 漏洞（H2） | ✅ | `apps/server/go.mod` 与 `apps/server/Dockerfile` 升至 1.26.6（CI 经 `go-version-file` 自动跟随）。CI 在 `go vet` 后新增固定版本 `govulncheck@v1.8.0` 门禁，补上 Trivy 只扫 OS/库、拦不住标准库 CVE 的盲区 |
| P1-2 | workflow 幽灵 action SHA（H3，实际 2 处） | ✅ | `e2e-playwright.yml` 的 `pnpm/action-setup` 与 `actions/upload-artifact` 改为正确 SHA；全仓 10 个 action SHA 经 GitHub API 逐一核验 |
| P1-3 | 3 个 i18n 键被引用但未定义（L4 / R1） | ✅ | 补齐 `objects.toastCopyFailed` / `batchEdit.tagsNeedKey` / `common.working` 的 zh-CN / en-US 文案；新增 `src/i18n/coverage.test.ts` 静态扫描（`import.meta.glob` 读源码，不引入 `node:*` 依赖），已验证删键即失败 |
| P1-4 | `delete-marker/restore` 契约字段漂移（P0-1 收尾） | ✅ | 请求体 `deleteMarkerId` → `versionId`（对齐 `restoreDeleteMarker` handler 与 `docs/api.md`）；契约测试扩展至 `delete-marker/restore` / `version`(DELETE) / `version/restore`，并验证回退修复时测试确实失败 |
| P1-5 | 异步任务丢失无法恢复（S1，v1.0.0 最后一项门槛） | ✅ | ① `service/job_persist.go`：`JobPersister` 抽象 + `FileJobPersister` 原子写（临时文件 → rename → 0600），`Create`/`Finish` 必落盘、中间进度 2s 节流；② 启动恢复：非终态任务标记 `interrupted` 并回写，`NewJobRegistry()` 保持纯内存语义不破坏既有测试；③ `interrupted` 独立 7 天保留期（`JobInterruptedTTL`），修复「恢复后首次 reap 即被 30 分钟 TTL 清除」的缺陷（有回归测试）；④ 新增 `GET /api/migrate/jobs`（路由数 69 → 70，OpenAPI 同步）；⑤ 前端 `MigratePanel` 新增「未完成任务」区块，提示移动任务「已复制但源未删除」并可逐条忽略；⑥ `main.go` 注入 `dataDir/jobs.json`。落盘未复用 `store/atomic.go`：`service→store` 会形成分层倒置（技术策略一致，已在 `architecture.md` 记录） |

### J. 2026-09-16 P2 加固修复（第一批）

> 来源：[`assessment.md`](assessment.md) §二 L1/L2、§二 S2、§二 M4/M6。对应 todolist #17（部分）/ #18（部分）/ #20 / #23（部分）。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P2-1 | 预签名错误被吞（L1） | ✅ | `objects.go` 三处 + `multipart.go` 一处 `u, _ := client.PresignXxx(...)` → 统一 `writePresignResult`，失败 500 而非 `200 {"url":""}`；新增源码级门禁 `TestPresignErrorsNotSwallowed`（逐行剔除注释后匹配，回退即失败） |
| P2-2 | 流式传输错误被静默吞（S2） | ✅ | `copyStream` 返回 `(int64, error)`；`recordStreamOutcome` 区分「真实中断」（Warn + `s3c_stream_interrupted_total`）与「客户端主动断开」（Debug，不计数） |
| P2-3 | JobRegistry 无总上限（M4） | ✅ | 新增 `TryCreate` + `ErrTooManyJobs`；上限 `maxJobs = 256` 只统计未终结任务（避免恢复的 interrupted 任务永久占满）；4 个异步端点超限返回 503 并释放 ctx；`Create` 保持原签名 |
| P2-4 | TLS 前置无 HSTS / Permissions-Policy（M6） | ✅ | TLS 示例配置补 `Strict-Transport-Security`（180 天）+ `Permissions-Policy`（关闭定位/麦克风/摄像头/支付/USB/interest-cohort） |
| P2-5 | 后端死代码（D1-D4） | ✅ | 删 `ctxReader`（与 `ctxCancelReader` 职责重复、零生产引用）、`batchItemError`（生产零引用）、`Client.S3()`（导出零生产调用，E2E 清理改用已有 `DeleteObjectVersion`/`DeleteObject`）；**`isNoSuchBucketSetting` 经核验有 5 处生产调用，非死代码，保留** |
| P2-6 | compose 未启用结构化日志（S4） | ✅ | `docker-compose.yml` / `docker-compose.prod.yml` 注入 `S3C_LOG_JSON: "${S3C_LOG_JSON:-1}"`，可设 0 覆盖；`.env.example` 同步说明；经 `docker compose config` 验证默认 1、可覆盖 0 |

**未纳入本轮**（评估为需更大改动或属行为变更）：
- **Argon2 `t=1` → `t≥2`**：加密文件格式只存 `S3C2` magic + salt，**不存 KDF 参数**。直接改 `argonTime` 会让所有既有 `accounts.json.enc` / `accounts.db` 无法解密。需先引入带参数的新格式版本（如 `S3C3`）并实现双版本读取迁移，属独立工作项。
- **`/api/health` 移除 `version`**：前端不消费该字段，但移除会改变既有响应契约（`accounts_test.go` 已断言其存在）。保留是运维定位版本的实际需要，且该端点通常位于内网或鉴权之后。
- **compose 默认改 encrypted**：会改变默认部署行为并涉及密钥管理（`S3C_STORE_KEY` 分发），需配套部署文档与升级说明。

### K. 2026-09-17 P2 加固修复（第二批）

> 来源：[`assessment.md`](assessment.md) §二 D5/D6。对应 todolist #10。

| # | 项 | 状态 | 修复内容 |
|---|----|------|----------|
| P2-7 | endpoint 归一化三份实现且行为不一致（D5） | ✅ | `s3wrap/client.go` 的 `normalizeEndpoint` 改为导出的 `NormalizeEndpoint` 并修正语义：**去首尾空白 + scheme 大小写不敏感 + host 小写 + 保留路径前缀**。旧实现只做大小写敏感前缀判断，把 `"HTTP://MinIO:9000"` 拼成损坏的 `"http://HTTP://MinIO:9000"`。`service/migrate.go` 删除本地副本改为复用；`s3wrap/presign.go` 的 `PublicURL` 同样复用，修掉 `" http://a.com/ /b/k"` 这类带空格的链接。`TestNormalizeEndpoint` / `TestPublicURL` 表驱动锁定，含路径前缀保留用例 |
| P2-8 | `ValidateEndpoint` 用未 trim 原串解析（D5 连带） | ✅ | 该函数此前 `strings.TrimSpace` 仅用于判空、解析仍用原串，导致用户手填的 `" http://127.0.0.1:9000 "` 被误判 `invalid endpoint URL`。改为先归一化再解析。**同时保持 fail-closed**：非空输入归一化后退化（`"http://"`、`"/"`、`"https:///"`）仍报错，不因归一化静默放行 —— 已加回归用例。E2E 实证：畸形 endpoint 建账号后 `preview-buckets` 由 400 `invalid account configuration` 变为通过校验进入网络调用，日志中 BaseEndpoint 为干净的 `http://127.0.0.1:9000` |
| P2-9 | handler 侧 `derefString`/`derefInt64`/`timeOrZero`/`boolOrFalse` 死代码（D6） | ✅ | 4 个包装在生产代码零引用，仅 `TestAccDerefHelpers` 引用（覆盖率注水样本）。连同该测试一并删除。`s3wrap/helpers.go` 与 `s3wrap_dto.go` 中的同名 helper **保留** —— 分别有 29/1/5/2/4 处生产调用（`derefString`/`derefInt32`/`boolOrFalse`/`derefInt64`/`timeOrZero`），属包内合理复用，非重复 |

### L. 2026-09-17 目录迁移后的可移除项清理

> 来源：[`assessment.md`](assessment.md) §二 D7/D8/D10。对应 todolist #11。

| # | 项 | 状态 | 处理内容 |
|---|----|------|----------|
| L1 | i18n 死键（D7） | ✅ | 全量扫描确认 **26 个**键中英各 26 行从未被引用（`app.name`、`common.back/confirm/danger/default/done/empty/failed/none/search/selected/test/upload/waiting`、`objects.bucket/prefix`、`toolbar.delete/downloadZip/filter/refresh`、`trash.empty`、`upload.queue`、`server.save`、`batchEdit.aclNoChange/storageNoChange`、`bucketTags.errEmptyKey`），字典键数 695 → 669。**根因**：既有门禁只校验「被引用 → 有定义」，无人校验反向，且 `i18n/**` 被排除在覆盖率统计外。已在 `i18n/coverage.test.ts` 增加**反向门禁**——字典中任何既非字面量引用、也不匹配动态拼接模式的键都会让测试变红；`storage.class.*` / `provider.*` 动态键按「字面命名空间前缀 + 插值」形状生成豁免正则。**关键点**：豁免必须限定该形状，否则 `/api/accounts/${id}`、`Bearer ${token}` 这类非 i18n 模板串会生成几乎匹配任意键的正则，让门禁恒真（初版即踩此坑，实测由「26 死键」退化为「0 死键」） |
| L2 | `@vitest/coverage-v8` 冗余依赖（D10） | ✅ | `vite.config.ts` 的 provider 为 `istanbul`，v8 provider 从未启用。已从 `package.json` 移除并重算锁文件（`pnpm install --frozen-lockfile` 通过；锁内残留条目仅为 vitest 的 optional peer 声明，不再安装）。四指标覆盖率复测仍 100% |
| L3 | `ObjectList.vue` 未使用的 `visibleCount` prop（D8） | ✅ | 父组件 `ObjectsPanel.vue` 传入、组件内从不读取（空态判断实际用 `totalCount`）。连同父组件绑定与 `ObjectList.test.ts` fixture 一并删除。`ObjectToolbar` 的同名 prop **保留**——真实用于 `{{ visibleCount }}/{{ totalCount }}` 计数展示。**剩余**：grid 视图 `v-for` 全量渲染未窗口化（D8 另一半） |
| L4 | `.gitignore` 冗余与无关条目 | ✅ | 202 行 → 80 行。删去 28 行与本站技术栈无关的脚手架模板条目（Bower / jspm / Snowpack、Next/Nuxt/Gatsby、SvelteKit、Docusaurus、VitePress、Serverless、FuseBox、DynamoDB、Firebase、Yarn、`.tern-port` 等），并按**实测**（逐条注释该规则后 `git check-ignore` 复查）删除 4 条被通用规则覆盖的冗余项（`apps/server/data/` ← `data/`；`apps/web/node_modules/`、`apps/desktop/node_modules/` ← `node_modules/`；`apps/web/dist/` ← `dist/`）。删除前后 `git status --ignored` 忽略集合逐行比对一致，唯一差异是把过窄的 `.cache/` 收敛为 `.cargo/`（前者会顺带忽略任意层级 `.cache/`，后者才是桌面端构建真实产物；已确认 `.cargo/` 下无可提交的 `config.toml`）。已确认无任何**已跟踪**文件落入新规则 |

### M. 2026-09-17 路线图 v1.0.0 / v1.0.x / v1.1.0 收口（#1–#8、#10、#11）

> 来源：[`roadmap.md`](roadmap.md) 三～五章。除 #12（桌面端签名）外的开放条目全部落地。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | `docs/api.md` 自动化校验 | ✅ | 新增 `handler/api_doc_test.go`：从 `docs/api.md` 解析 `METHOD /api/...` 行，与 `routes.go` 注册表做**双向 diff**（文档多写 / 少写均红灯）。已注入漂移实测两侧变红后还原 |
| 2 | SQLite 密钥加密 + Argon2 加强 | ✅ | 新加密格式 **S3C3**：`magic + time(4,BE) + memory(4,BE) + threads(1) + salt(16) + ciphertext`，把 KDF 参数写进文件头，读取按文件参数派生 → 可在不影响既有库的前提下调参。`argonTimeV3=2`（OWASP 建议 ≥2）。**双版本读取**：S3C2 旧库（t=1）仍可解。`SQLiteStore` 新增 `storeKey`，`secret_key` 列以 S3C3 密文落盘、读时解密、历史明文行原样兼容；无 key 读密文行显式报错。`config.MinStoreKeyLength = 16` 校验 `S3C_STORE_KEY` |
| 3 | 安全审计日志 + XFF 可信代理 | ✅ | 新增 `handler/audit.go`：稳定事件常量 + `h.audit(r, event, ...)`，覆盖 401（含 malformed/bad_token 原因）、账号 CRUD、桶策略设置/清除、对象删除/前缀删除、回收站清空、限速命中。`clientIPWithProxies` 仅在直连对端命中 `S3C_TRUSTED_PROXIES` 时才采信 XFF 首段，否则回退 `RemoteAddr`（防伪造 XFF 绕过限速，已有集成用例）。`S3C_TRUSTED_PROXIES` 默认空 |
| 4 | ZIP 部分失败可见 | ✅ | `zip.go` 不再丢弃 `WriteObjectsZip` 的 `failKeys`：部分失败落 Warn 日志并计入 `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total`，整体失败计入 `s3c_zip_failed_total` |
| 5 | S3 上游指标 | ✅ | 新增 `s3wrap/metrics.go`：smithy `Finalize` 中间件采集调用数、耗时直方图（11 桶）、错误按码分类（API code / `canceled` / `timeout` / `transport`，基数有界）、流字节数。`/api/metrics` 输出 `s3c_s3_*` 系列；`copyStream` 回传字节数计入 `s3c_s3_stream_bytes_total` |
| 6 | 错误文案不回显用户输入 | ✅ | `headers.go` 的 `ValidateUserMetadata` 错误改为固定文案 + `Debug` 日志（原样回传含用户 key 的错误串）；`metadata.go` / `objects.go` / `multipart.go` 的 `unsupported acl: <值>`、`duplicate tag key`、`unsupported storageClass` 等一并改为固定文案。新增 `error_echo_gate_test.go` 源码级门禁防复发 |
| 7 | grid 视图窗口化 | ✅ | `ObjectList.vue` 的 grid 分支改为渲染 `gridItems`（上限 300 条），超出时显示截断提示（`objects.gridTruncated`）；列表视图此前已有窗口化 |
| 8 | 前端健康轮询与自动恢复 | ✅ | 新增 `composables/useHealthPoll.ts`：后端出错后每 5s 探测 `/api/health`，成功即回调 `loadAccounts` 重载并清除错误横幅；未出错时不轮询。`useBucketSetting.reload()` 加 seq 竞态守卫（旧请求的失败/loading 不覆盖新请求） |
| 10 | `entries` / `visibleEntries` 单次排序 | ✅ | `useObjectBrowser.ts` 抽出 `compareEntries`，`entries` 一次排序（文件夹恒在前），`visibleEntries` 只做过滤、保持顺序，消除过滤态下的重复排序 |
| 11 | 覆盖率门禁去「注水」 | ✅ | 前端 `vite.config.ts` 不再整体排除 `src/i18n/**`，只排除纯数据模块 `src/i18n/messages/**`；`index.ts` 纳入统计后补齐 `readLocale` 回退/异常、`setLocale` 写入失败、`cycleLocale`、`locale()`、`i18nKeyCount` 缺省参数等行为测试，四指标仍 100%。后端把确实不可达的防御分支删除（`encryptSecret` 的 AES 错误分支、`os.MkdirAll` 冗余判断），改为行为断言而非 gap 测试；`make test-cover` 的 `count==0` 检查归零 |

---

### N. 2026-09-19 明文落盘启动告警与旧编号引用清理

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 落盘明文启动告警 | ✅ | `config.Config.StorePlaintextWarning()`：`json` / `sqlite` 驱动且 `S3C_STORE_KEY` 为空时返回可执行文案（含驱动名、`DataDir`、`encrypted` 建议与最短长度），其余配置（含未知驱动）返回空串；`runServer` 在 `Validate` 之后以 `logger.Warn` 输出。先写失败测试（`TestStorePlaintextWarning` 六例表驱动）再实现；`TestMainServerWarnsPlaintextStore` 用 fork 子进程跑完整 `main()`，断言 `sqlite` + 空 key 启动日志确实出现「明文落盘」，加密配置不告警。对应风险 [roadmap.md](roadmap.md) §5.1「已收敛」索引 R3（启动告警守卫）。**2026-09-20 强化**：该告警仅在显式 `S3C_ALLOW_PLAINTEXT_STORE=1` 放行时才会出现——默认改为硬失败，见 §T |
| 2 | 旧 `roadmap #N` 引用清零 | ✅ | 2026-09-17 收口后，代码注释里仍留 **27 处**（25 个文件）指向已归档编号的 `roadmap #N`（`docker-compose.yml`、`config.go`、`store/*`、`handler/*`、`s3wrap/*`、`apps/web/src/*`）。按语义改写：带 `ASSESSMENT` 编号的只保留该编号；纯 roadmap 编号的改为「已闭环：features.md §M」；`docker-compose.yml` 的密钥加密说明改指 `features.md` §M + `roadmap.md` §5.1「已收敛」索引 R3。全仓 `grep -rn "roadmap #[0-9]"`（除 `CHANGELOG.md` 的历史记录）为零，编号引用规则见 [roadmap.md](roadmap.md) §六 第 6 条 |

---

### O. 2026-09-19 风险登记集中处置（R1 / R2 / R4 / R6 / R7 / R8）

> 来源：[`roadmap.md`](roadmap.md) §5.1 风险登记（R5 桌面签名经决策暂不处理）。每条都补了
> 可复现的门禁或测试，收敛后移入该节末尾的「已收敛」索引（R8 维持 ➖ + 可选加固）。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | R1 契约字段级门禁 + 三处注册表失真修复 | ✅ | 新增 `TestAPIDocDocumentsRequestBodyFields`（`api_doc_test.go`）：OpenAPI 每个 `requestBody` 的字段名必须出现在该端点 `docs/api.md` 正文，支持「请求体与 `POST /api/x` 相同」别名（`apiDocSections` / `sectionText`）。门禁上线即抓到三处**客户端可见**失真：`mkdir` 注册表写 `prefix`（handler/frontend/文档都是 `key`）、`copy-objects` 写 `items`（真实是 `keys`）、`DELETE /api/accounts/{id}/version` 把 query 参数误声明成 requestBody。已修注册表并加回归断言（`TestOpenAPI_ContractRequestBodyMatchesHandlers` 第 7 项 + `requestQueryParams`），`docs/api.md` 补齐当时判定为漏记的字段。**事后更正**：其中只有 `publicEndpoint` / `metadata` 是真漏记，`provider`/`replaceTags`/`cacheControl`/`contentEnc`/`contentLang`/`disposition` 六个字段在 handler 中**并不存在**（反向漂移：文档与注册表凭空多写），按文档示例体调用会因 `DisallowUnknownFields` 直接 400。2026-09-19 P0-5 已把这六个幻影字段从注册表与 `docs/api.md` 删除，并把该端点族的断言改成**字段集完全相等**（第 8 项），同时补上真漏记的 `useSSL`。另加**通用输入源门禁** `TestOpenAPIRequestDeclarationMatchesHandlerInput`：解析 `routes.go`（含 `h.withStreamLimit(h.x)` 包装）→ handler 方法调用闭包找 `readJSON` / `json.NewDecoder` ⇔ 注册表 `Request:` 双向比对，覆盖全部 70 条路由；已用「给 `GET /api/health` 注入假 requestBody」的变异验证过会红灯后还原 |
| 2 | R2 消音式死代码门禁 | ✅ | 实测 `golangci-lint`（`run.tests: true`）**能**报未引用的测试函数/方法，盲区只有显式消音。新增 `apps/server/deadcode_gate_test.go` 扫全仓 Go 源码禁止 `_ = ident` 与 `var _ = expr`（允许 `_ = f()` 这类显式忽略返回值），并自检扫描文件数避免路径写错静默变绿。清掉 11 处遗留消音（`acc_ratelimit_test.go` 的 `var _ = errors.New`、`bs_gap_test.go` 的只声明不用的 `deletes`、`object.go` 里 `PurgeObject` 的冗余 `_ = keyMarker` 等） |
| 3 | R4 DataDir 单写者锁 | ✅ | 新增 `store.AcquireDataDirLock`：unix `flock(LOCK_EX\|LOCK_NB)` 锁 `<DataDir>/.s3clinet.lock`，第二个实例启动即失败（`runServer` 返回 1）；内核在进程退出时释放，无陈旧锁。`main.go` 在开 store 前加锁。测试：`TestDataDirLockExcludesSecondInstance`（互斥 + 释放后可重加）、`TestDataDirLockErrors`（目录不可建 / 锁文件被占）、`TestRunServerRejectsLockedDataDir`（端到端拒绝启动）；`GOOS=windows go build ./...` 通过（非 unix 为文档化 no-op） |
| 4 | R6 RustSec 审计入 CI | ✅ | `cargo audit`（pin `cargo-audit 0.22.2`）加入 GitHub `ci.yml` 与 GitLab `desktop` job，新增本地 `make rust-audit`。实跑结果：**0 个漏洞**，7 条 unmaintained / unsound 告警（`proc-macro-error`、5 个 `unic-*`、`glib 0.18.5`）逐条 triage——均为上游未发修复版本的传递依赖，默认退出码策略是「漏洞红灯、告警可见」，不用 ignore 清单掩盖 |
| 5 | R8 可选 SSRF 加固 | ✅ | 新增 `S3C_SSRF_DENY_PRIVATE`（默认关闭，保持 [ADR-003](decisions/0003-ssrf-private-allow.md) 自托管放行）：置位后 `isBlockedIP` 连 RFC1918 / ULA / 回环 / 未指定地址一并拒绝，创建期与拨号期双重校验同时生效。测试：`TestDenyPrivateNetworksOptIn`（默认放行 vs 开启后 6 类地址全拒、公网仍放行）、`TestDenyPrivateNetworksDialGuard`（拨号期）、`TestFromEnvSSRFDenyPrivate`。ADR-003 增加 Update 段记录该开关 |
| 6 | R7 分段上传 ETag 失败路径可见 | ✅ | 新增 `upload.test.ts` 用例：分段 PUT 返回 2xx 但缺 `ETag` → 报「未读取到 ETag」并在组装前 `multipartAbort`（不留半成品）。README 补多厂商 CORS 兼容性矩阵（RustFS 有真对端 E2E，MinIO / AWS S3 / 阿里 OSS / 腾讯 COS 标注手动），明确 `ExposeHeader: ETag` 是分段组装硬前提 |
| 7 | R8 存储硬失败可观测 | ✅ | `/api/metrics` 新增 `s3c_store_up` gauge（store 掉线为 0），与既有 `/api/health` 503、`TestHealthStoreUnavailable` 一起构成 ADR-002 的可告警面；`TestMetricsStoreUpAndSSRFPolicy` 覆盖 1/0 两态。[deployment.md](deployment.md) §6.1 补 503 处置顺序与告警建议、§6.3 与 [api.md](api.md) 指标清单同步 |
| 8 | R8 SSRF 生效策略可见 | ✅ | 启动日志新增 `ssrfDenyPrivate` 字段；`/api/metrics` 新增 `s3c_ssrf_deny_private` gauge（读新增的 `s3wrap.DenyPrivateNetworks()`），运维可核对 `S3C_SSRF_DENY_PRIVATE` 是否生效；`TestMetricsStoreUpAndSSRFPolicy` 覆盖 1/0 两态 |

---

### P. 2026-09-19 分支状态审查 A1–A3 处置（前端 API 拆分 / 死代码门禁 / 测试接缝）

> 来源：2026-09-19 分支状态审查 [`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) 的架构项 A1–A3 与
> 浏览器 E2E 空转用例（已在审查文档中销项——该文档按约定只保留**未闭环**的发现）。归档证据即本表。
> 该审查登记的 P0/P1 缺陷（B1 SSE 自旋、B2 SyncKeys 前缀、F1 白名单 crash、R1 CI docker job、
> §7.2 契约幻影字段等）**仍未处置**，不在本节范围内。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | A1 `api.ts` 838 行单文件拆分 | ✅ | 拆为 `apps/web/src/api/` 下 7 个模块：`storage.ts`(339) / `endpoints.ts`(192) / `jobs.ts`(119) / `download.ts`(85) / `index.ts`(82) / `upload.ts`(48) / `http.ts`(38)。单向依赖 `index → {endpoints,jobs,download,upload} → http → storage`，无环；`./api`、`../api` 仍解析到 `api/index.ts`，**import 路径与对外契约不变**（用 `git mv api.ts api/index.ts` 保留历史）。顺带消除 `migrateJobStatus` 在 endpoints/jobs 各写一份的重复定义。**纯搬运**：`fetch` headers 合并顺序等既有语义原样保留（审查 §F10 潜伏缺陷不混入）。验证：`vue-tsc` / `eslint` 通过、986 例（64 文件）单测全绿、覆盖率四指标 100%（statements 3941 / branches 2787 / functions 1082 / lines 3392，7 个新模块全部满覆盖）、Playwright 15 passed / 0 skipped |
| 2 | A3 前端死代码门禁 | ✅ | 新增 `apps/web/src/deadcode_gate.test.ts`：API 公开面（`s3api.*` / `api.*` / 具名导出）每个成员必须在**至少一个非测试源文件**中被引用。eslint/vue-tsc 只报未使用的局部变量，看不见导出符号，而测试引用会让死代码"活着"——这正是本项要堵的洞。用 `import.meta.glob(?raw)` 读源码（不引入 `node:fs`/`@types/node`）；按 **import 别名精确匹配 `别名.成员`**，不做裸词/子串匹配（避免 `'migrate'` 字面量与 `migrateAsync` 前缀碰撞假绿）；三重自检（扫描文件数 ≥50 / 公开面 ≥50 / 引用总数 ≥30）防"空跑变绿"；`INTENTIONAL_UNUSED` 豁免清单附"豁免不得腐烂"检查。**变异验证**：注入 `s3api.zzDeadProbe` → 红灯；复刻历史缺陷（`s3api.copyFiles` 仅被 `api.test.ts` 调用）→ **仍红灯**（测试引用不算使用） |
| 3 | A3 清理零生产调用的公开面 | ✅ | 删除 5 个 `s3api` 方法（`copyFiles` / `deletePrefix` / `copyPrefix` / `migrate` / `migrateSync`，全仓仅测试引用；对应后端端点保留）与 2 个无用 barrel 导出（`requestResponse` 仅 `download.ts` 内部使用、`downloadZipToDisk` 生产只走 `s3api.downloadZipToDisk`） |
| 4 | A2 生产包剔除测试专用接缝 | ✅ | `s3wrap.ResetMetrics`（导出且自称"仅测试使用"）→ 移入 `metrics_test.go` 内的 `resetMetrics()`，不再进入生产二进制；`service.SetMaxJobsForTest` 删除，在册上限由**可变全局** `var maxJobs` 改为**每实例**构造期选项 `NewJobRegistry(WithMaxJobs(n))`（原写法跨用例/跨实例污染，并发下即数据竞争），handler 侧接缝移到 `internal/handler/export_test.go`（仅 `go test` 编译）。以行为用例替换原「钩子返回旧值」用例：`TestWithMaxJobsLimitsRegistry`（上限生效 + 实例间互不影响）、`TestWithMaxJobsIgnoresNonPositive`（非正上限回落默认） |
| 5 | 浏览器 E2E 空转用例 | ✅ | `account-flow.spec.ts` 两个用例由「1 skipped + 1 个断言自身 fixture」改为真实 UI 驱动（`role=button` 入口、空态文案、新增表单 → POST 请求体 → 列表渲染；以及 `/api/accounts` 失败 → 断连徽标 → `/api/health` 恢复后自动重连）。Playwright 15 passed / **0 skipped** |

---

### Q. 2026-09-19 分支状态审查 P0 处置（5 项全部闭环）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md)。该审查登记的 P0 已全部处置并从审查文档移除
> （该文档只保留未闭环发现）；本表即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。
> 剩余的 P1/P2 发现仍开放。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | P0 SSE 对 `interrupted` 任务无限自旋 | ✅ | `case p, open := <-ch: if !open { return }` + `service.IsTerminalJobStatus(p.Status)`；SSE 路由补 `withStreamLimit`；`Job.Subscribe` 加每任务订阅上限 16（饱和回 503）。测试 `TestMigrateJobEventsInterruptedStreamCloses`（真实 `jobs.json` 恢复路径 + 真 HTTP 流，断言只 1 帧且 EOF）/ `TestMigrateJobEventsSubscriberCap`；前者经「还原旧循环」变异验证会红灯 |
| 2 | P0 `s3c.servers` 坏数据整站白屏 | ✅ | `sanitizeServer()` 逐元素形状校验 + 回写修复清单；`getActiveServer()` 改为渲染期安全访问器。测试：`api.test.ts` 5 种坏存储 + 混合条目，`App.corruptStorage.test.ts` 用真实 `./api` 挂载真实 App 断言不白屏 |
| 3 | P0 `SyncKeys` 目标 key 映射错误、永不收敛 | ✅ | 抽出复制内核 `migrateKeys(..., dstKeyFor, ...)`，`MigrateKeys` / `SyncKeys` 共用；`SyncKeys` 只留一个映射表达式；`stripPrefix` 补段边界校验。测试 `TestSync_PrefixMappingConverges` / `TestSync_PrefixSegmentBoundary` / `TestMigrateSync_PrefixFilter`（含二次同步 `copied:0`） |
| 4 | P0 GitHub CI `docker` job 结构性失败 | ✅ | `build-push-action` 加 `load: true` + 新增 `docker image inspect` 前置步骤。本地 `docker-container` builder 复现：不加 `--load` 构建 exit 0 但镜像不在 daemon，加 `--load` 后可见 |
| 5 | P0 OpenAPI 契约幻影字段（6 个） | ✅ | `provider` / `replaceTags` / `cacheControl` / `contentEnc` / `contentLang` / `disposition` 从注册表与 `docs/api.md` 删除（无实现意图，前端亦无对应入参）；补上真漏记的 `useSSL`。`TestOpenAPI_ContractRequestBodyMatchesHandlers` 第 8 项断言四个端点的**字段集完全相等** |

---

### R. 2026-09-19 分支状态审查 §三 正确性处置（后端 B3–B11 / 前端 F2–F10）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) 的「阶段 2 · 正确性」全部发现项。该审查文档按约定
> 只保留**未闭环**发现，本节即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。
> 原则：**补门禁而不只是补缺陷**——每条修复都配了会先失败的回归测试，覆盖「分支/语义正确但断言缺席」这类盲区。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | B3 `DeleteObjects` 吞掉 200 响应体内的逐 key 失败 | ✅ | `s3wrap.DeleteObjects` 改返回 `[]DeleteFailure{Key,Code,Message}`（`err` 仅表示传输/协议失败，两者可同时非空）；`POST …/delete` 回 `{deleted,failed,lastError}`、`delete-prefix`（同步/异步）与回收站 `PurgeObject` 一律按「请求数 − 逐 key 失败数」记账，新增 sentinel `ErrPartialDelete`。连带修 `runDeletePrefix` 的进度口径（只看 `deleted` 会在「全部删除失败」时空转到 2h 超时）。测试：s3wrap 3 例 + handler 4 例（含 100 页全失败仍终止） |
| 2 | B4 ZIP 打包 `break` → 永久 goroutine + 连接泄漏 | ✅ | `service/zip.go` 的 `break` → `continue`：无缓冲 `results` 必须继续消费，否则其余 worker 永久阻塞在发送上、已取回 body 不关闭（用户取消下载即触发）。测试 `TestWriteObjectsZipCopyErrorDoesNotLeak` 断言所有取回 body 最终关闭且全部失败 key 上报（旧实现泄漏 3 个 body）。`features.md` 早先「P-3 zip goroutine 泄漏 ✅」至此才成立 |
| 3 | B5 增量同步吞掉列举错误 | ✅ | `listAll`/`indexDst` 返回 `error`，`SyncKeys` 签名改为 `(SyncResult, error)`；`cancelOrErr` 区分「客户端取消（返回部分结果、不报错）」与「真实列举错误（上抛）」；handler 走 `writeInternalErr`（源端 AccessDenied → 403，不再回 `200 {scanned:0}`）。测试：端点级 1 例 + service 2 例 |
| 4 | B6 `indexDst` 无页数上限、循环内不查 ctx | ✅ | `listAll`/`indexDst` 共用硬上限（100 页 × 1000 key × 10 万总量）+ 每轮 ctx 检查 + 「NextToken 未前进即停」。测试 4 例（含 5s 超时护栏：旧实现会挂死） |
| 5 | B7 `Job.Emit` 锁外投递 vs `Finish` 关闭 channel | ✅ | 改为锁内非阻塞投递（`select`+`default`），落盘 I/O 移到锁外。测试用「节流落盘钩子」把 Emit 卡在「快照之后、投递之前」再 `Finish`：旧实现**确定性 panic（send on closed channel → 进程退出）**，新实现通过；另加 20×50 并发压测 |
| 6 | B9 8 MB body 上限 vs 10 000 key 批量 | ✅ | 上限提到 16 MiB（10 000×1 KB ≈ 10.3 MB 必须通过），并用 `LimitedReader{N: maxBody+1}` 把「超限」与「JSON 无效」分开：超限回 **413** 而非 400。测试 4 例 |
| 7 | B10① 第 10000 段被误判超限 | ✅ | 上限判断移到**上传前**（段号 10000 合法）；常量提为可注入的 `maxMultipartParts`（默认 10000 有断言保护），测试用小上限精确覆盖边界，避免真跑 1 万次 UploadPart |
| 8 | B10② `filename*` 用 `QueryEscape`（空格 → `+`） | ✅ | 新增 `rfc5987Escape`（attr-char 白名单 + UTF-8 逐字节 `%XX`）。测试含中文文件名与「不得出现 `+`」断言 |
| 9 | B10③ download-zip 不校验空 key | ✅ | 空 key 直接 400（否则 S3 把 `GET /bucket/?key=""` 当列举桶，ListBucket XML 被塞进 ZIP）。测试 1 例 |
| 10 | B10④ `mode=text` 忽略读错误仍回 200 | ✅ | 非 `EOF`/`ErrUnexpectedEOF` 的读错误走 `proxyErr`；`ErrUnexpectedEOF` 再用 `Content-Length` 判定短读。测试 2 例（畸形 chunked / 声明长度大于实发） |
| 11 | B10⑤ 分段顺序不校验（`InvalidPartOrder` → 500） | ✅ | handler 提交前拒绝降序/重复段号（400）；`HTTPStatus`/`UserMessageForCode` 补 `InvalidPartOrder → 400 / invalid request`。测试 2 处 |
| 12 | B10⑥ 指标标签取服务端原始 `<Code>`（基数无界，同 §S5） | ✅ | `errorClass` 改走 20 个已识别错误码的白名单，其余归 `other`。测试 `TestErrorClassUnknownCodeFoldsToOther`（断言原始码不出现在 `ErrorsByCode`） |
| 13 | B11 `statusRecorder` 未覆写 `Write` | ✅ | 覆写 `Write` 标记「已写出」，隐式 200 之后的多余 `WriteHeader` 不再透传、日志状态不被改写。测试 2 例 |
| 14 | F2 虚拟列表窗口不随 `entries` 重置 → 0 行空白表 | ✅ | 抽出共用 `src/virtualList.ts`（`virtualWindow` + 单一 `ROW_HEIGHT`），两组件在 `entries` 变化时归零 `scrollTop` 并同步写回 DOM |
| 15 | F3 客户端不分片，撞服务端上限整批失败 | ✅ | 新增 `src/limits.ts`（1000 / 10000 上限来自后端契约）+ `batchKeys`；删除与迁移改为分片串行 + 结果聚合（单片失败不中断后续），新增 i18n 键 `objects.toastDeletePartial` |
| 16 | F4 复制文件夹忽略 `truncated` | ✅ | 按 `start.truncated` 提示「仅复制一部分」（i18n 键 `dest.toastCopiedTruncated`） |
| 17 | F5 `loadAllSourceObjects` 分页内读实时前缀 | ✅ | 循环外快照 bucket/prefix + `listGen` 代次守卫（成功/失败/finally 三处） |
| 18 | F6 迁移 SSE 无空闲超时 | ✅ | 45s 空闲超时（任何数据/心跳重置），超时经既有 `onError` 上报并主动释放连接；EOF 补偿轮询语义不变 |
| 19 | F7 分享链接对全部选中并发 presign | ✅ | 复用 `batchMetadata` 的 4 路有界池 |
| 20 | F8 `AccountsPanel.load` 无 try/catch | ✅ | 捕获并渲染带「重试」按钮的错误条（i18n 键 `accounts.loadFailed`） |
| 21 | F9 健康轮询续跑 / RO 首屏失效 / 行高错位 / localStorage 无保护 | ✅ | ① 代次 + `disposed` 守卫；② `ObjectList` 的 ResizeObserver 改 `watch(scrollEl)`；③ 行高常量与 CSS 统一 42px；④ `theme.ts`/`store.ts`/`useObjectBrowser.ts` 全部 try/catch 降级 |
| 22 | F10 headers 合并顺序陷阱 | ✅ | 先合成默认 headers 再 `Object.assign` 调用方 headers（默认 Authorization/Content-Type 不再被整体覆盖） |

**门禁**：`go vet` / `golangci-lint run ./...` **0 issues** / `go test -race -count=1` 8/8 包 + **每包 100.0%**（
`awk '$NF==0'` 零块）/ `gofmt -l` 干净；前端 `pnpm lint`（0 警告）/ `pnpm typecheck` /
`pnpm test:coverage`（66 文件 / 1039 用例，四指标 100%）/ `pnpm build` 全绿；
**真对端 RustFS E2E 4/4 通过**（`S3CLINET_E2E=1`，覆盖 12 MiB 分段组装、批量删除、桶配置、回收站 purge——
本轮改动直接触及 `DeleteObjects` / `PurgeObject` / 分段上传，故按 `AGENTS.md` 要求追加实跑）。

---

### S. 2026-09-19 分支状态审查 P1 处置（§9.2 全部闭环）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §9.2 的 8 项 P1。该审查文档按约定只保留
> **未闭环**发现，本节即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。
> 原则同 §R：**补门禁而不只是补缺陷**——每项都配了会先失败的回归测试；新增/改造的门禁在文件头
> 写明「断言范围」，避免下轮把"有门禁"误读为"已全量收敛"。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | §7.2 修正 `components.schemas.Account` | ✅ | `internal/openapi/openapi.go` 删幻影字段 `provider`/`forcePathStyle`/`insecureSkipVerify`、补真实 `useSSL`；与 `model.AccountView` 逐字段一致 |
| 2 | §7.2 新增**响应契约门禁** | ✅ | 新增 `openapi_response_contract_test.go`：对 `components.schemas` 每个共享 schema 用 `go/parser` 抽取对应 Go DTO 的 `json` tag，做**双向**比对（多写=幻影字段，漏写=客户端丢字段）；含 `Account` 逐字段断言与 `secretKey` 不得出现断言。上线即红灯，修正后转绿 |
| 3 | §4.3 文档字段门禁改**双向**、去子串匹配 | ✅ | `api_doc_test.go` 的 `TestAPIDocDocumentsRequestBodyFields` 改为机械抽取 `docs/api.md` 请求体 JSON 的**顶层键**（自写容错扫描器，支持注释剥离 / 嵌套对象 / `200 {…}` 响应体区分），与注册表双向比对；彻底去掉 `strings.Contains`（原实现下 `key` 会被 `keys`/`secretKey` 满足）。别名段支持「与 `POST /api/x` 相同」与省略方法名的「与 `copy-prefix` 相同」 |
| 4 | §4.3 请求体字段门禁改**全量遍历** | ✅ | 新增 `openapi_request_fields_test.go`：routes.go → handler 方法调用闭包 → `readJSON` 目标变量 → 结构体字段集，与注册表 requestBody schema 全量双向比对。端点专属 DTO 双向相等；共享模型走 `serverManagedFields` 白名单。**上线即抓到 3 处新漂移**：`storage-class` 漏 `versionId`、`preview-buckets` 漏 `publicEndpoint`/`useSSL`——已修注册表与 `docs/api.md` |
| 5 | R2 tag ↔ 清单一致性 | ✅ | `release-desktop.yml` 新增 `Verify tag matches manifest versions`：tag（去 `v`）必须等于 `tauri.conf.json` 与 `Cargo.toml` 的 `version`，否则 `exit 1`。本地实测 rc1 通过、rc2 正确失败 |
| 6 | R3 `SHA256SUMS` 竞态 | ✅ | 三平台各自上传**平台内唯一**的 `SHA256SUMS-<bundle>.txt`；新增 `aggregate-checksums` job（`needs: publish`）拉取三份合并去重为唯一 `SHA256SUMS.txt`（含行数自检），并清理中间资产 |
| 7 | R4 Trivy DB 缓存 / 重试 | ✅ | 两套 CI 均拆出 `--download-db-only` + 4 次指数退避重试，扫描阶段加 `--skip-db-update` 复用已就绪 DB；GitHub 侧加 `actions/cache`（`.trivy-cache`），GitLab 侧加 `cache: paths`。把「网络抖动」与「真的扫出漏洞」区分开 |
| 8 | S1 运行镜像 alpine EOL | ✅ | `apps/server/Dockerfile` 运行镜像 `alpine:3.20`（2026-04-01 EOL）→ `alpine:3.24`（支持至 2028-06-01） |
| 9 | S2 `.env.example` 占位 token | ✅ | 根 `.env.example` 的 `S3C_TOKEN` 由 `change-me-use-openssl-rand-hex-32`（33 字符，是**有效口令**）改为置空，对齐 `apps/server/.env.example` |
| 10 | 配置类门禁（防复发） | ✅ | 新增 `apps/server/repo_infra_gate_test.go`：`.env.example` token 不得 ≥ `MinTokenLength`、运行镜像不得是 EOL alpine、发布 workflow 必须做 tag↔清单比对与平台内唯一 checksum、两套 CI 的 Trivy 必须缓存+重试。四项均为「YAML/配置正确但断言缺席」的盲区 |
| 11 | §7.3 文档失真 D2 / D3 / D6 / D8 / D9（同批修正） | ✅ | D2：`openapi_handler.go` 鉴权注释改为与实测一致（401/404）；D3：`release-version.sh` 正则放宽到通用 semver（接受纯 `v1.0.0`/`v1.1.0`）、同步文件补到 12 处、`Cargo.lock` 只改本包；D6：`api.md` 的 presign `expiresIn` 改为「默认 1h、上限 24h」；D8：现行能力描述「3 路并发」→ **2 路**；D9：`development.md` 存放约定改为「除根目录约定文件与 `.github/` 外统一放 `docs/`」 |
| 12 | SSOT：`docs/todolist.md` 补登记开放项 | ✅ | 此前写着「无待办」而审查仍有大量开放发现，违反「唯一待办来源」约定；现按 #26–#46 补登记 P2 全部条目（契约残留 / 安全 / 发布链 / 可观测性 / 文档失真），编号稳定不重排 |

**门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** /
`go test -race -count=1 ./...` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）；
新增门禁均已做「先红后绿」验证（Account schema 与 3 处注册表漂移均先失败再修绿）。

---

### T. 2026-09-20 审查 §9.3 P2 处置（契约 #26–#28 + 安全 / 供应链 #29–#34）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §9.3 P2。该审查文档按约定只保留**未闭环**发现，
> 本节即归档证据，逐条改动见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。原则同 §R / §S：
> **补门禁而不只是补缺陷**——每项都配了会先失败的回归测试，新增门禁在文件头写明「断言范围」。
> #25（桌面端签名）经确认是**外部凭证阻塞**（roadmap E6），代码侧无剩余工作，转入 ⛔ 外部阻塞。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | #26 query 参数漏声明 | ✅ | 补 `head.versionId` / `proxy.maxBytes` / `trash.prefix` / `objects.startAfter`（+ `docs/api.md` 补 `startAfter`）。新增 `openapi_query_params_test.go`：routes.go → handler 调用闭包 → 机械抽取 `q.Get(...)`，与注册表 `in:query` **双向**比对（含自检计数）。**上线即抓到第 5 处漂移**：`GET /versions` 的幻影参数 `key`（handler 从不读取）→ 移除 |
| 2 | #27 类型 / required / 枚举语义门禁 | ✅ | 新增 `openapi_semantics_test.go`。**枚举**：每个 `EnumStr` 站点须在 `enumContracts` 登记，真实值从 `switch` case / `map[string]bool` / 跨包 `service` 常量抽取后双向比对 → 抓到 `PUT object-acl` 漏 `authenticated-read`/`aws-exec-read`。**required**：注册表 required 必须是 handler 真实解码且非「空值→默认」的字段 → 修正 `presign.method`、`copy-object.newBucket`、`copy-prefix.targetBucket` 三处错误 required。残留（`bucketOr` 默认桶模式）写入文件头 |
| 3 | #28 响应门禁残留范围 | ✅ | `openapi_response_contract_test.go` 新增**端点级响应门禁**：对声明具体 `properties` 的 2xx 响应，抽取 handler `writeJSON` 表达式的键（字面量 map / 命名 struct tag / 变量 / 同包 helper 递归）双向比对；4 个注册表文件约 30 个 `Obj()` 响应升级为 `BuildObj` 纳入门禁；修正 3 个异步端点 `200`→`202`。残留（其他注册表自由体、运行时动态键 map）写入文件头 |
| 4 | #29 / #31 明文落盘安全默认 | ✅ | `Config.Validate` 对 `json`/`sqlite` + 空 `S3C_STORE_KEY` 返回新 sentinel `ErrPlaintextStoreNotAllowed` **硬失败**，除非 `S3C_ALLOW_PLAINTEXT_STORE=1`（opt-in 时保留 WARN）；报错给出两条出路且不泄露密钥。base compose 改 `${S3C_STORE_KEY:?}` 强制非空；`.env.example`（根 + server）补说明；`e2e.yml` 补 dummy key（`docker compose up` 会插值整个文件）。测试 `TestValidatePlaintextStoreRejected` / `TestMainServerRejectsPlaintextStoreWithoutOptIn` / `TestMainServerAllowsPlaintextStoreWithOptIn` / `TestFromEnvAllowPlaintextStore` |
| 5 | #30 metrics 免鉴权文档声明 | ✅ | 行为不变（有意内网 scrape），在 `README.md` / `docs/api.md` / `docs/deployment.md` §6.3 / `docs/threat-model.md` §2 显式声明「即使配了 `S3C_TOKEN` 也不受保护、匿名可读、勿直接暴露公网」；`TestMetricsEndpointUnauthenticatedEvenWithToken` 钉住行为（metrics 无 token 200 / accounts 无 token 401） |
| 6 | #32 Trivy 升级 + digest pin | ✅ | `aquasec/trivy:0.58.1` → `0.74.0@sha256:62b1e65e…1969`（两套 CI；digest 经 `docker pull` + ghcr.io + public.ecr.aws 三源核对）；`--vuln-type` → `--pkg-types os,library`（0.74.0 `--help` + `pkg/flag/package_flags.go` 核实）。门禁 `TestTrivyImageIsVersionAndDigestPinned`（两套 CI 版本一致 + 必须带 digest + 禁用 `--vuln-type`） |
| 7 | #33 工具链 pin | ✅ | **关键发现**：原 `dtolnay/rust-toolchain` SHA `6bed0761…` 是 `stable` **浮动分支 tip**（commit 即 `toolchain: stable`），并非版本 pin——改为版本分支 SHA `ce678459… # 1.98.1`；新增 `rust-toolchain.toml`（`channel = "1.98.1"`）；GitLab `rust:1.98.1-bookworm`。Node `24` → 精确 `24.21.0`（workflow / alpine / bookworm / NodeSource）。desktop `packageManager: pnpm@9.15.0`（实测不破坏 lockfile）。门禁 `TestRustToolchainIsVersionPinned` / `TestNodePinnedToPatchVersion` / `TestPackageManagerIsPinnedToCIPnpm` |
| 8 | #34 `.dockerignore` 补齐 | ✅ | 补 `apps/web/coverage`、`.pnpm-store`、`test-results`、`playwright-report`、`.run/`、`.cargo/`、`.trivy-cache/`、`.gitlab-ci-local/`、`coverage.out`/`.txt`、`*.tsbuildinfo`、`*.log` 等；`apps/server/*.log` → `**/*.log`。门禁 `TestDockerignoreExcludesBuildArtifacts`（11 条必需模式，注释行不计入） |

**门禁实跑**：`gofmt -l` 干净 / `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** /
`go test -race -count=1 -coverprofile` 8/8 包通过且**每包 100.0% 语句覆盖**（`awk '$NF==0'` 零块）/
`go build ./...` OK；前端 `pnpm lint` 0 告警 / `pnpm test` 66 文件 1039 用例全绿 / `pnpm build` OK；
`docker compose config` 缺 `S3C_STORE_KEY` → exit 1、补齐 → exit 0。供应链 pin 均经上游实测核验
（Trivy 版本 + digest、Node 24.21.0、Rust 1.98.1），新门禁均做「先红后绿」与变异验证。

---

### U. 2026-09-22 审查 §9.3 P2 收尾（D1 / D4 / D5 / D10 + P3 + R7 / R9 / R10 + 门禁质量）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §7.3 与 §9.3 的余项；逐条改动见
> [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。原则同 §R / §S / §T：**补门禁而不只是补缺陷**，
> 每项都配了会先失败的回归测试，关键修复做「还原旧实现」变异验证。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | D1 Prometheus 直方图语义违规 | ✅ | 根因：`s3wrap` 记的是**每桶增量**，`handler/metrics.go` 直接当**累积**值按 `_bucket{le=…}` 输出 → `le` 非单调、`+Inf`≠`_count`（实测 `le="0.01"`=3 / `le="+Inf"`=0 / `_count`=3），`histogram_quantile()` 全错。现 `MetricsSnapshot` 快照时累积为真累积值；并把 `calls` 由 `atomic.Int64` 移入同一把锁（消除「`+Inf`=0 而 `_count`=3」的读写不一致窗口）。测试：`TestS3MetricsHistogramIsCumulativeAcrossBuckets`（3 个耗时逐桶断言累积值）、`TestS3MetricsHistogramConcurrentContract`（8×50 并发）、`TestMetricsHistogramSatisfiesPrometheusContract`（端点级解析 `le` 单调性与 `+Inf == _count`）。**变异验证**：把 `cumulative += c` 还原为 `cumulative = c` → 三条全部红灯，且复现报告中 `+Inf`=0 / `_count`=3 的原始症状 |
| 2 | P3 `/api/openapi.json` 无缓存 | ✅ | `Registry` 增加 `spec` 缓存：`Operation` / `Param` / `Respond` / `SetInfo` / `AddServer` 全部置空（语义与「每次重算」等价）；`MarshalJSON` 双检加锁只算一次并返回**副本**（调用方改写不污染缓存）。测试：`TestRegistry_MarshalJSONCached`（含副本隔离）、`TestRegistry_MarshalJSONCacheInvalidated`（四类注册动作逐一验证）、`TestRegistry_MarshalJSONConcurrent`。既有 `TestOpenAPI_ContractMarshalDeterministic`（8×8 并发字节一致）继续通过 |
| 3 | D4 `failKeys ≤ 200` 承诺不成立 | ✅ | 根因：承诺写在 `features.md` / `api.md`，但服务端只有异步 `delete-prefix` 在 handler 内裁剪；`copy-objects`（同步/异步）、`migrate`（同步/异步）、`migrate/sync` 全部原样回传 → 10k 全失败约 10 MB 并落盘 `jobs.json`。现上限提为 handler 层共享常量 `maxFailKeys = 200` + `capFailKeys`，**所有**回传 `failedKeys` 的端点统一裁剪；异步路径经新增 `jobResultFromBatch` 在 `Job.Finish`（落盘）**之前**裁剪；`failed` 计数不受裁剪影响。测试：`TestOlCopyManyFailKeysAll`（201 失败 → 列表 200、计数 201）、`TestOlMigrateSyncFailKeysCapped`、`TestOlMigrateAsyncFailKeysCappedBeforePersist`（**直接读 `jobs.json`** 断言裁剪发生在持久化前）。同时修正 `service/batch.go` 与 `copy.go` 中两处互相矛盾的注释 |
| 4 | D5 `git tag v1.0.0` 不存在 | ✅ | `scripts/release-version.sh v1.0.0` 同步 15 处版本串（Makefile / Dockerfile / compose×2 / web+desktop `package.json` / `tauri.conf.json` / `Cargo.toml` / `Cargo.lock`（只改本包）/ `main.go` / README / docs），并打 `v1.0.0` tag。历史性提及（CHANGELOG 已发布区、roadmap 里程碑行、本审查报告）按「不追溯篡改」纪律保留 |
| 5 | D10 CHANGELOG 段内过期计数 | ✅ | `[Unreleased]` 内两处「当时实测值」（覆盖率 3883/2769/1059/3339、前端 63 文件 / 983 测试）已失真。按本文件「不追溯篡改已发布区」的纪律**保留原值并加注**当时时点与新值（4074/2844/1095/3503、66 文件 / 1039——后者是 §W 追加用例前的值，§W 后为 1042），不改写历史数字 |
| 6 | R7 本地 `make` ≠ CI | ✅ | 新增 `lint`（golangci-lint）/ `govulncheck` / `check`（聚合静态检查 + 双侧覆盖率）目标；全部 `pnpm install` → `--frozen-lockfile`；`test-all: test-cover web-test-cover`（此前无覆盖率门禁）；`.PHONY` 补全全部已定义目标。门禁 `TestMakefileMirrorsCIGates`（含「所有已定义目标必须在 .PHONY」的机械检查；**变异验证**：还原裸 `pnpm install` → 红灯） |
| 7 | R9 GitHub 不传 `VERSION` build-arg | ✅ | 构建与推送两处都显式注入 `VERSION=ci`（此前回落到 Dockerfile 过期字面量，而 GitLab 传 `ci`、Makefile 传真实版本——一个 Dockerfile 三种行为）。门禁 `TestGitHubWorkflowInjectsVersionBuildArg`（要求 ≥2 处，保证扫描与推送版本一致；**变异验证**：改掉一处 → 红灯） |
| 8 | R10 无 workflow 推送镜像 | ✅ | 新增 `publish` job：`needs: docker`（Trivy 通过才推）、`if: github.event_name != 'pull_request'`、`permissions: packages: write`、`docker/login-action` 登录 GHCR、按 commit SHA + 分支双标签推送。因 buildx `push` 与 `load` 互斥且扫描需 `load`，拆为独立 job（同时避免推出未扫描镜像）。门禁 `TestGitHubWorkflowPushesImage` |
| 9 | 测试质量：21 个用例名硬编码过期行号 | ✅ | 如 `api.test.ts` 写 `line 125 else`（当时 `listServers` 在 `storage.ts:301`）、`bucketPolicy.test.ts` 写 `line 111`（当时 `:108`）——测试在描述一个不存在的版本。已从全部用例名移除行号（改为描述行为，行号留在注释），新增门禁：`deadcode_gate.test.ts` 的「测试名不得硬编码源码行号」（含扫描下限自检防空跑） |
| 10 | 测试质量：i18n 门禁两处盲区 | ✅ | ① `usedKeyTexts` 对整份源码做正则 → **注释里的键名也算「已使用」**；现先按字符扫描剥离注释（正确跳过字符串字面量，避免把 `'https://x'` 的 `//` 当注释）再匹配。② 双语只比**键数量** → `zh-CN` 缺 `a` 而 `en-US` 多 `b` 时数量相等仍绿；现按语言解析键集合做**集合级双向比对**，并新增「每个键两种语言取值都非空」。**变异验证**：改名一个 en-US 键（数量不变）→ 集合门禁红灯；删真实引用只留注释 → 死键门禁红灯 |
| 11 | 文档失真小项 | ✅ | `openapi_contract_test.go` 的 `apiRouteCount` 注释算错（写「69 = 68 + 1」，常量与实际均为 70 → 改「70 = 69 + 1」）；`docs/todolist.md` 补记 #40 / #42 / #43 / #44 为**从未启用的保留空号**（避免被误读为漏登记），#15 补入已完成编号 |

**门禁实跑**：后端 `gofmt -l` 干净 / `go vet ./...` 0 告警 / `go test -race -count=1 -coverprofile`
8/8 包通过且**每包 100.0% 语句覆盖**；前端 `pnpm lint` 0 告警 / `pnpm typecheck` 通过 /
`pnpm test:coverage` **66 文件 1042 用例全绿，四项覆盖率 100%**（statements 4074 / branches 2844 /
functions 1095 / lines 3503）。

> **仍未纳入机械门禁的残留**（须人工跟踪，已写入各门禁文件头）：端点内联 / 运行时动态键的响应体、
> query 参数的 `Has()` / `Values()` 读取口径、md 表格中的叙述性数字。
> （**真实 Go 后端 + 真实前端产物**的浏览器联调冒烟已于 §V 闭环，不再属于本残留清单。）
>
> **2026-09-22 收尾（§W）**：上述三项残留**全部闭环**——端点级响应门禁升级为全量覆盖（15→61 个
> 成功响应）、query 门禁扩到四种读取口径、新增 md 叙述性数字门禁与 `_test.go` 导出符号死代码门禁。
> 现顶层自由体响应仅剩 `/api/openapi.json` 一个（响应体即 OpenAPI 规范本身，非 `writeJSON` 写出）；
> `metadata` / `fields` 等嵌套自由体只比对到顶层键。

---

### V. 2026-09-22 真实后端 + 真实 RustFS 浏览器联调（todolist #37）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §9.3 的**唯一保留项**。此前
> `e2e-playwright.yml` / `playwright-e2e` 的 `/api/**` 全被 `page.route` mock，没有任何门禁
> 验证「真实 Go 后端 + 真实 RustFS + 真实构建产物」拼在一起时的行为——尤其**浏览器直传**
> （预签名 PUT 是浏览器 → S3 的跨源请求），mock 时代在结构上不可能被覆盖。本轮补齐后
> 审查 §9.3 清零。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 真实联调用例（不 mock `/api`） | ✅ | 新增 `apps/web/e2e-real/real-backend.spec.ts` + 专用 `playwright.real.config.ts`（不自拉 webServer，`PLAYWRIGHT_BASE_URL` 指向真实后端）。3 条用例：① 账号 CRUD 真实落库（后端 `GET /api/accounts/{id}` 可见、`secretKey` 不回传、UI「测试连接」走真实 S3 成功）；② 建桶 → 列桶（真实 `CreateBucket`/`ListBuckets`）；③ **浏览器真实直传**：UI 选文件 → 前端取预签名 PUT → 浏览器 XHR PUT 到 RustFS → 列对象 → 预签名 GET 回读逐字节校验。用例先 `resetBackend` 清空账号表，保证每个用例从真实空状态出发（失败重试也不会串状态） |
| 2 | 本地一键运行（docker 自动起 RustFS） | ✅ | 新增 `scripts/e2e-real.sh` + `make e2e-real`：自动 docker 起一份真实 RustFS（**显式配 `RUSTFS_CORS_ALLOWED_ORIGINS`**，否则浏览器直传被 CORS 拦下）→ `pnpm build` 真实产物 → 起真实 Go 后端（`S3C_STATIC_DIR` 托管产物）→ 跑 `pnpm e2e:real` → `trap` 自动清理；`--keep` / `--skip-build` / `--no-rustfs`（复用外部对端）便于排查与 CI 复用；幂等 `playwright install chromium` 修掉新克隆直接失败的 UX 缺口 |
| 3 | 两套 CI 接入（GitHub + GitLab） | ✅ | GitHub 新增 `.github/workflows/e2e-real.yml`（PR 按路径 + 手动 + 每周五定时）；GitLab 新增 `e2e-real` job（RustFS 用 GitLab **service**，与 `rustfs-e2e` 同款；service 变量里配好 `RUSTFS_CORS_ALLOWED_ORIGINS`）。触发规则与另两套 E2E 对齐，不阻塞普通 push |
| 4 | 编排收敛为单一来源 | ✅ | 初版三处各写一份编排（改一处漏两处即漂移）。现两套 CI 都 `bash scripts/e2e-real.sh`（GitHub 自起容器；GitLab `--no-rustfs` + `RUSTFS_ENDPOINT=http://rustfs:9000` 复用 service），各自只留「装工具链 / 装浏览器系统依赖」 |
| 5 | 机械门禁防漂移 | ✅ | `TestRealE2EUsesSharedScript`（两侧 CI 必须**实际调用** `bash scripts/e2e-real.sh`——仅出现在 `paths:`/`changes:` 里不算）、`TestRustFSImageIsConsistentlyPinned`（脚本默认值 / compose / GitLab service 三处镜像版本一致，禁 `latest`）、`TestRealE2EArtifactsExist`（**联调 spec 出现 `page.route(` 即红灯**，防退化成第二个 mock 版）、`TestE2ESourcesAreTypechecked`、`TestLocalRealE2ETargetExists` |
| 6 | E2E 源码静态检查 | ✅ | `e2e/` 与 `e2e-real/` 此前不在主 `tsconfig.json` 的 `include` 内，**零类型检查与 lint**。现新增 `apps/web/tsconfig.e2e.json` + `pnpm typecheck:e2e`，`pnpm lint` 扩到 `eslint src e2e e2e-real`（顺带修掉一个未使用变量）；两套 CI 的 `web` job 与本地 `make check` 都纳入 |

**变异验证**：① 去掉 RustFS 容器/CORS 配置后重跑本地真实联调——第 3 条（浏览器直传）**红灯**
（跨源 PUT 被拦，超时），前两条不经浏览器的用例仍绿，证明该用例确实依赖真实对端 + 真 CORS；
② 门禁侧逐条变异：改 GitLab service 镜像版本 → `TestRustFSImageIsConsistentlyPinned` 红灯；
把 GitLab 的 `bash scripts/e2e-real.sh` 换成 `pnpm e2e:real` → `TestRealE2EUsesSharedScript` 红灯
（该门禁最初用子串 `scripts/e2e-real.sh`，被 `changes:` 路径命中而假绿，已改为要求实际调用）；
删 CI 里的 `typecheck:e2e` → `TestE2ESourcesAreTypechecked` 红灯；往联调 spec 塞类型错误 → 类型检查红灯。

**门禁实跑**：`make e2e-real` 3 passed（默认 docker 模式与 `--no-rustfs` 外部对端模式各跑一遍）；后端新增 5 条门禁纳入 `go test` 100% 覆盖率。

---

### W. 2026-09-22 审查 §7.4 门禁盲区收尾（响应门禁全量 + query 读取口径 + 叙述性数字 + 导出测试符号）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §4.3 / §7.4 / §9.1 列出的**最后四类残留**
> ——即门禁矩阵里标「部分」与「否」的行。原则同 §R / §S / §T / §U：**补门禁而不只是补缺陷**；
> 本轮新增/扩展的门禁全部做了变异验证，并在升级过程中**抓到 4 处真实漂移**（见条目 5–8）。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 端点级响应门禁：**顶层**自由体 `openapi.Obj()` 全量收敛 | ✅ | 把 accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system 十个注册表的**顶层**自由体响应全部升级为 `openapi.BuildObj` 具体 `properties`（嵌套形状抽为共享构造器：`corsRuleSchema` / `tagRowSchema` / `lifecycleRuleSchema` / `versionEntrySchema` / `deleteMarkerSchema` / `jobProgressSchema` / `jobResultSchema` / `jobRecordSchema`）。端点级门禁覆盖面 **15 → 61** 个成功响应，**顶层**自由体从 ~45 个降到 **1 个**（`/api/openapi.json`：响应体即规范本身，由 `Registry.HTTPHandler()` 直接写，非 `writeJSON`，机械抽取无意义）。自检从 `checked ≥ 15` 收紧为 `checked ≥ 60` 且 `untyped ≤ 1`——**回退成顶层自由体会直接红灯**。注意：`metadata` / `fields` 等**嵌套**自由体仍只比对到顶层键 |
| 2 | query 参数门禁：读取口径扩展 | ✅ | 抽取器从只认 `Get()` 扩到四种字面量口径：`Get` / `Has` / `Values` / `Query()["x"]`，并覆盖 `q := r.URL.Query()` 绑定后的同名形式。新增**非字面量键检测**（`q.Get(name)` / `q[k]`）——命中即红灯要求改字面量，杜绝「动态键读取静默逃逸」。新增口径测试 `TestQueryReadExtractorCoversAllForms`（8 种形态逐一断言，含「无读取不得抽出参数」的反向断言） |
| 3 | md 叙述性数字门禁（矩阵唯一标「否」的行） | ✅ | 新增 `apps/server/doc_number_gate_test.go`：以 `routes.go` 的 `mux.HandleFunc` 注册数为**唯一真值**，校验 README / `api.md` / `roadmap.md` / `features.md` 里「N 个 `/api/*` 端点」的 N。要求每条声明**至少命中一次**（文案漂移导致正则失配即红灯，防门禁静默失效）。变异验证：把 `features.md` 改回 69 → 红灯 |
| 4 | `_test.go` 导出符号死代码门禁 | ✅ | `golangci-lint unused` 对**导出**符号因「可能被包外引用」而豁免，但 `_test.go` 不参与库构建、永远无包外引用——实测给 `_test.go` 加无人调用的导出函数/类型，`golangci-lint run` 报 **0 issues**。新增 `TestNoUnusedExportedTestSymbols`：扫描全部 `_test.go` 的导出包级符号，无引用即红灯；`export_test.go`（约定的测试接缝）整体豁免。检测逻辑抽为纯函数 `findUnusedExportedTestSymbols`，由 `TestFindUnusedExportedTestSymbols` 用**合成源码**做口径测试（死符号必报、被引用/非导出/Test 入口/接缝文件不得误伤）——不依赖「仓库里正好有个死符号」来证明门禁有效 |
| 5 | **真实漂移（升级中抓到）**：`s3wrap.CorsRule` 缺 `json` tag | ✅ | 该结构体经 handler **直接序列化**进 `GET /bucket/cors` 响应，却无 tag → Go 输出 PascalCase（`AllowedMethods` / `MaxAgeSeconds`），而前端 `types.ts` 与 `docs/api.md` 都是 camelCase。Go 的 `json.Unmarshal` 大小写不敏感，**服务端既有测试一直全绿**；但 JavaScript 严格区分大小写，浏览器读 `x.allowedMethods` 恒为 `undefined` → CORS 规则的方法/来源被静默归一化成空数组。已补全 6 个 camelCase tag；新增 `TestCorsRuleJSONFieldNames`（双向：必须有 camelCase、不得再有 PascalCase）与 `TestCorsRuleJSONRoundTrip`（前端发来的 camelCase 能解析）。**变异验证**：去掉 tag → 两条红灯并列出实际 PascalCase 键 |
| 6 | **真实漂移**：`POST /api/migrate/async` 误声明 200 | ✅ | handler 写 `StatusAccepted`（202），`docs/api.md` 也写 `202`，只有注册表写 200。已改 202 并由端点级门禁钉住 |
| 7 | **真实漂移**：`docs/api.md` 称 openapi.json「不进鉴权层」 | ✅ | 与实现相反：`withAuth` 只豁免 `health` / `metrics`，openapi.json 配了 token 时无凭证返回 401，且需 `S3C_EXPOSE_OPENAPI=1` 否则 404（既有 `TestOpenAPI_GateAndAuth` 早已钉住该行为，仅文档失真）。已改为「经过鉴权层」并写明两个前置条件 |
| 8 | **真实漂移**：`docs/features.md` 称 69 个端点 | ✅ | 实际为 70（`routes.go` 70 条 `mux.HandleFunc`，README / `api.md` / `roadmap.md` 与 `apiRouteCount` 常量均为 70）。已改为 70，并纳入条目 3 的数字门禁 |
| 9 | 文档请求体缺失：`PUT /api/accounts/{id}` | ✅ | 该端点请求体在 `docs/api.md` 只有散文描述（「字段同创建」），导致 `TestAPIDocDocumentsRequestBodyFields` 无法机械比对字段集而红灯。已补显式 JSON 示例（并注明 `secretKey` 可省略、必填为 `name`/`endpoint`/`accessKey`），与注册表的 required 声明一致 |

**门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues** / `go build ./...` OK /
`go test ./...` 全绿（含新增门禁）；四条新门禁与两处缺陷修复均经**变异验证**（改回旧写法必红灯）。

> **本节后的残留**（已写入各门禁文件头）：**顶层**自由体响应仅 `/api/openapi.json` 一个；
> `metadata` / `fields` 等**嵌套**自由体与嵌套对象/数组的**深层字段**只比对到顶层键（元素形状由
> 注册表共享构造器统一维护）；md 中依赖运行环境或时刻的叙述性数字（用例数 / 覆盖率 / 行数）
> 按「历史记录」保留，不做静态门禁；md 端点数量叙述数字由 `doc_number_gate_test.go` 覆盖。

---

### X. 2026-09-22 审查 §4.3「维持」项复核收尾（门禁自身口径 + path 参数门禁）

> 来源：[`docs/archive/review-2026-09-19.md`](archive/review-2026-09-19.md) §4.3 表格里最后五行标「维持」的项。
> 复核后发现**每一项都藏着真实缺口**（不是「已足够好」，而是「没人验过它能不能被绕过」）——
> 全部经**实测确认缺口存在**，再改为 AST 判定或新增门禁，并逐条做变异验证（改回旧写法必红灯）。

| # | 门禁 | 实测缺口 | 处置与验证 |
|---|------|----------|------------|
| 1 | `openapi_inputsource_test.go` | 用裸字符串匹配 `readJSON(` 判断「handler 是否解码请求体」→ **注释与字符串字面量都能骗过它**。实测 `// 这里提到 readJSON(` 与 `s := "readJSON("` 均被判定为「解码了」 | 改为 **AST 判定**（`parseBodyCalls` 解析语法树，只认真实调用表达式）。新增 `TestDecodesBodyIgnoresCommentsAndStrings` 用合成源码钉住口径（注释/字面量不算、真调用与委托算）。**变异验证**：删掉 `createAccount` 的真实 `readJSON` 调用、只在注释留 `readJSON(` → 旧实现假绿、新实现红灯 |
| 2 | `api_doc_test.go` `TestAPIDocMatchesRoutes` | 解析正则只认行首，缩进行在两个方向上都被漏。实测「文档里有**陈旧**路由、且写成缩进」时双向 diff **静默通过**（fail-open）——陈旧条目会永久留在文档里 | 新增 `TestAPIDocRoutesAreFlushLeft`：路由声明行必须顶格，缩进即红灯（含扫描下限自检）。**变异验证**：插入缩进的 `DELETE /api/nonexistent-stale` → 新门禁红灯，旧双向 diff 无反应 |
| 3 | `deadcode_gate_test.go` | 旧实现是整行正则 `^_ = ident` / `^var _ = expr`。实测 `x := 1; _ = x`、`if true { _ = x }`、`_ = x.Field`、`_ = s[0]` **全部逃逸**（fail-open） | 改为 **AST 判定**（`findSilencingDeadCode`）：按语句而非按行，判据是「左侧是否全为 `_`」。`_, ok := m[k]`（comma-ok 惯用法）与 `_, _ = w.Write(b)`（显式丢弃返回值）放行；`var _ Iface = (*T)(nil)`（编译期接口断言）放行。新增 `TestFindSilencingDeadCodeCoverage` 断言「AST 命中数严格大于旧正则」（实测 9 vs 5），防止退回旧口径。**变异验证**：四种逃逸形状现全部被拦 |
| 4 | `golangci-lint` 配置 | **未启用 `nolintlint`**：在函数上方加一行无理由的 `//nolint:all`，`golangci-lint run` 仍报 **0 issues**——`//nolint` 是 golangci 自身的指令，默认不受任何 linter 审查；而 `AGENTS.md` / `development.md` 明令禁止用它消音 | `.golangci.yml` 启用 `nolintlint`（`require-explanation: true` + `require-specific: true` + `allow-unused: false`）。既有唯一一处 `//nolint:staticcheck // SA1019: ...` 已合规。**变异验证**：植入 `//nolint:all` → 红灯（"should mention specific linter"） |
| 5 | path 参数（新发现的缺口） | `TestOpenAPI_ContractPathParamsDeclared` 只校验注册表**内部**自洽（模板 `{x}` ⇔ 声明 `in:path`），**没有任何一条**比对 handler 是否真的 `r.PathValue("x")` 读取它——注册表与 handler 同时改名才一致，只改一边无人拦 | 新增 `openapi_path_params_test.go`：注册表 `in:path` 参数名集 ⇔ handler 沿调用闭包 `PathValue` 读取集，**双向**比对（70 个端点 / 60 个带 path 参数），含非字面量键检测与口径测试。**变异验证**：把 `getAccount` 的 `PathValue("id")` 改成 `PathValue("accountId")` → 同时报「漏声明」与「幻影参数」 |

**门禁实跑**：后端 `go vet ./...` 0 告警 / `golangci-lint run ./...` **0 issues**（含新启用的 `nolintlint`）/
`go build ./...` OK / `go test ./...` 全绿（8/8 包，**每包 100.0% 覆盖且零未覆盖块**）。

> **本节后的残留**：§4.3 表格已无「维持」项（每行都是 ✅ 且带回归门禁）。仅剩的**结构性**无法
> 机械拦截项同 §W——`/api/openapi.json` 顶层自由体响应、`metadata`/`fields` 嵌套自由体与嵌套结构深层字段、依赖运行环境/时刻的
> 叙述性数字；均已在对应门禁文件头写明断言范围。
>
> **2026-09-22 追加复核（门禁自身再检查）**：发现并关闭三处新缺口——① `openapi_query_params_test.go`
> 的动态键检测只认 `r.URL.Query().Get(name)`，绑定变量后的 `q := r.URL.Query(); q.Get(name)` / `q[k]`
> 会同时逃过抽取与动态检测（fail-open，已补 `hasDynamicQueryRead` 与合成用例）；
> ② `openapi_path_params_test.go` 的动态键检测还是裸正则（注释/字符串会误报），已改 AST
> （`hasDynamicPathValueRead`）；③ `openapi_request_fields_test.go` 的 `readJSON` 定位仍是逐行正则
> （注释/字符串里的调用会被当成解码点），已改 AST（`findReadJSONTarget`）。同时把
> `delete-prefix/async`、`copy-prefix/async` 的请求体从 `openapi.Obj()` 自由体升级为具体
> `properties`，使请求体字段门禁覆盖这两个端点。

---

### Y. 2026-09-22 子审查复核：异步任务契约统一 + 门禁口径补充

> 来源：子审查 724461ee 的复核发现。逐条修复证据见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]`。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 异步任务失败 key 契约分裂（`failKeys` vs `failedKeys`） | ✅ | 公共契约统一为 `failedKeys`（`JobResult` json tag + OpenAPI `jobResultSchema` + `docs/api.md` list 示例）；`job_persist.go` 的 `UnmarshalJSON` 兼容旧落盘 `failKeys`。新增 `TestOlMigrateJobsListUsesFailedKeys`（清单接口 result 必须含 `failedKeys`、不得泄露 `failKeys`）与 `TestJobResultUnmarshalAcceptsLegacyFailKeys`（新旧格式 + 非法 JSON 错误路径） |
| 2 | query 字面量空白盲区 | ✅ | `q.Get( "x" )` / `q[ "x" ]` 既可被抽取、也不会被误判动态；动态下标正则补 `\s` 排除。合成用例覆盖空白字面量与空白动态键 |
| 3 | 请求体解码 receiver 不敏感 | ✅ | `parseBodyCalls` 只认 `h.readJSON` / `json.NewDecoder`；`findReadJSONTarget` 要求 receiver 为 `h`。新增 `TestParseBodyCallsDecodeReceivers` 与 `other.readJSON` 负例 |
| 4 | 死代码符号计数跨包互相抵消 | ✅ | 按「归一化包名 + 符号名」计数（`foo_test` 与 `foo` 同作用域）；新增 `TestFindUnusedExportedTestSymbolsScopesByPackage` 跨包同名合成用例 |

> **仍未机械覆盖**：仍保留 §X 列出的结构性残留；本节的 query 抽取本身仍是正则（动态键已显式红灯），
> 如需彻底 AST 化，应另立任务而不是把「有门禁」读成「已无裸正则」。

---

### Z. 2026-09-24 roadmap §三 #3 死代码纪律收口：前后端两道「零生产引用」导出门禁

> 来源：roadmap §三 #3（原 ➖「维持现状：由覆盖率门禁调整一并治理」）评估后改判为**两道显式门禁**收口；
> 逐条改动与实现细节见 [`CHANGELOG.md`](../CHANGELOG.md) `[Unreleased]` 同日条目。

| # | 条目 | 状态 | 实现与验证 |
|---|------|------|------------|
| 1 | 后端生产导出符号零引用门禁（Gate 1） | ✅ | `apps/server/deadcode_gate_test.go` 扩展：AST 扫描全仓 Go 文件，导出包级符号与导出方法一旦**零生产引用**（仅测试引用同样算死）即红灯并报出文件与处置指引。口径：按「归一化包 clause + 符号名」分组（`foo_test` 与 `foo` 同作用域）、方法裸名归并；豁免 `//nolint` 消音符号、门禁自身反射用例（`reflectiveMethodNames` 7 项，配合成用例兜底）、const 表成员（枚举契约）。**变异验证**：注入 `MutantDeadExportedFunc` → 精确红灯（184 文件扫描、定位 `internal/service/job.go`、给出处置指引）→ 撤回 → 复绿且 grep 残留 0 |
| 2 | 前端非 API 模块导出门禁（Gate 2，关闭文件头盲区 #2） | ✅ | `apps/web/src/deadcode_gate.test.ts` 新增 B 半边：非 `src/api` 模块的每个运行期导出必须被生产代码引用（剥掉 import 与导出声明后裸词计数，仅测试引用 = 死）；每个生产源模块必须被生产代码 import——`.vue` 组件要求**默认导入**（`import type` 不算在用）、`export { … } from` 再导出边计入 import 图（`api/index.ts → ./upload` 实例）、`main.ts` 入口豁免（`index.html` 引用）；`default` 导出与 `as` 重命名导入用**前置断言**拦死（出现即红灯，先改写再进门禁）；自检下限 79 文件 / ≥90 导出 / ≥37 组件防空跑。**变异验证**：注入 `mutantDeadExport` + `MutantOrphan.vue` → 两半同时红灯并指名 → 撤回 → 7/7 复绿 |
| 3 | 门禁红灯清出的死代码清零 | ✅ | 后端 5 个生产死导出删除：`openapi.OpBuilder` 链式配置器（`OpBuilder` / `Param` / `Respond`——端点已全部改为 `r.Operation(...)` 内联 `Op{}` 直注册，链只剩测试引用）与 `service.RegistryOption` / `WithMaxJobs` 选项接缝（原 `SetMaxJobsForTest` 的再包装，零生产调用），`Registry.Operation` 改无返回值、`NewJobRegistry*` 去掉空转选项参数；测试接缝统一为 `FillJobSlotsForTest(t, h)`（`export_test.go`，生产文件零 `*ForTest` 钩子）。前端 2 个真死导出删除：`i18nKeyCount`（`i18n/index.ts`）、`shouldUseMultipart`（`upload.ts`，`uploadObject` 已内联同义判断）；引用方测试同步改写（中英键数一致由 `coverage.test.ts` 集合级比对承担、阈值分支由 `uploadObject` 用例覆盖） |
| 4 | 文档同步（与代码同 commit） | ✅ | `roadmap` §三 #3 行移除（编号不重排，#3 留空号并在 3.2 注中说明）、§四 基线日期与前端数字实测更新；`development.md` §3 门禁落点补两半口径与前端落点；`architecture.md` §3 提法扩为两半；`CHANGELOG` `[Unreleased]` 同日条目；本节即证据台账 |

> **仍未机械覆盖**：门禁文件头保留 3 条盲区——类型导出（`vue-tsc` 承担）、`s3api[name]` 动态成员（全仓无此写法）、
> 裸词计数的同名抵消与注释 / 字符串命中（只漏报、不误报，与后端 Gate 1 同向）。`as` 重命名导入与 `default`
> 导出属**前置断言**而非盲区：出现即红灯。

---

## 三、质量与覆盖率现状

> 2026-09-15 本机实测；2026-09-16 P0 + P1 修复后复测：`go vet ./...` 干净、`go test -race ./...` 8/8 包通过
> （`handler` 覆盖率 100.0%）、`govulncheck ./...` **0 可达漏洞**、前端 62 文件 / **967** 测试全绿且四指标均 100%、
> `vue-tsc` / `vite build` / `eslint` 干净。
>
> 2026-09-19 风险登记集中处置后复测：`make test-cover` 8/8 包 **100.0%**（含新增 `deadcode_gate_test.go`）、
> `golangci-lint run ./...` **0 issues**、前端 64 文件 / **986** 测试全绿、`cargo audit` **0 漏洞**（7 条告警已 triage）。
>
> 2026-09-19 审查 §三（正确性 B3–B11 / F2–F10）处置后复测见 §R 末段：后端 8/8 包 **100.0%**、前端 66 文件 / **1039** 测试全绿。
>
> 2026-09-20 审查 §9.3 P2（契约 #26–#28 + 安全 / 供应链 #29–#34）处置后复测见 §T 末段：
> 后端 8/8 包 **100.0%**（`awk '$NF==0'` 零块）、`golangci-lint` **0 issues**、前端 66 文件 / **1039** 测试全绿、
> `docker compose config` 在缺 `S3C_STORE_KEY` 时拒绝启动。
>
> 2026-09-22 §37 真实联调收口后复测（roadmap §四 门禁基线同步为此轮实跑值）：后端 8/8 包 **100.0%**、
> `golangci-lint` **0 issues**、前端 66 文件 / **1042** 测试全绿（覆盖率 4074 / 2844 / 1095 / 3503 四指标 100%）、
> `pnpm audit` **0 漏洞**、`make e2e-real` **3 passed**。
>
> 2026-09-24 roadmap §三 #3 死代码纪律收口（前后端两道「零生产引用」导出门禁，见 §Z）后复测
> （roadmap §四 门禁基线同步为此轮实跑值）：`gofmt -l` 干净 / `go vet` 0 告警 / 后端 `go test` **8/8 包通过**、
> `golangci-lint` **0 issues**、`go build` 干净；前端 `pnpm lint` 0 告警 / `pnpm typecheck` + `typecheck:e2e` exit 0 /
> **66 文件 1043 例全绿**（覆盖率 **4072 / 2843 / 1093 / 3501 四指标 100%**）/ `pnpm build` OK。

| 门禁 | 结果 |
|---|---|
| `go vet ./...` | 干净 |
| `go test -race -count=1 ./...` | 8/8 包通过 |
| `govulncheck ./...` | **0 可达漏洞**（go1.26.6；修复前 6 个） |
| `golangci-lint run ./...` | **0 issues**（errcheck / staticcheck / govet / ineffassign / unused / gosec / nolintlint 零告警，`run.tests: true` 含测试文件） |
| 后端覆盖率 | **每个包 + 汇总均 100.0% statements**（main / config / model / openapi / store / service / s3wrap / handler） |
| 前端 `pnpm test` | 66 文件 / **1043** 测试全绿（2026-09-17 新增 health poll / grid 窗口化 / reload 竞态 / i18n 分支用例；2026-09-19 补分段缺 ETag 用例与前端公开面死代码门禁，审查 §三 处置再补虚拟窗口重置 / 分片提交 / SSE 空闲超时 / 存储降级等用例；2026-09-22 §37 联调后再 +3；2026-09-24 §Z 删 2 例仅测试引用直测、非 API 导出门禁 +3 例） |
| 前端覆盖率 | **statements / branches / functions / lines 均 100%**（含 `src/i18n/index.ts`） |
| `vue-tsc --noEmit` / `vite build` | 干净 / OK（360.52 KB，gzip 110.85 kB） |
| `eslint` | 0 违规（`no-explicit-any: error`） |
| `gofmt -l .` | 干净 |
| `docker compose config` | base / prod / tls 均通过 |
| E2E（Playwright，`/api` mock） | 15 passed / 0 skipped |
| E2E（真实 RustFS，`S3CLINET_E2E=1`） | 按需运行，默认不阻塞 CI |
| E2E（真实后端 + 真实 RustFS + 真实产物，`make e2e-real`） | **3 passed / 0 skipped**（todolist #37；不 mock `/api`，含浏览器直传） |
| Rust 依赖审计（`cargo audit`） | **0 漏洞**；7 条 unmaintained / unsound 告警已 triage（[threat-model.md](threat-model.md) §5） |

### 已知边界与取舍
- **单实例**：文件型 store（`json` / `sqlite`）+ 内存任务表不支持多副本，启动时对 `S3C_DATA_DIR` 加 `flock` 单写者锁，第二实例启动即失败；非 unix（Windows）无跨进程文件锁，为文档化 no-op。
- **SSRF 默认放行私网 / 回环**（自托管主场景，[ADR-003](decisions/0003-ssrf-private-allow.md)）；需要更严策略时设 `S3C_SSRF_DENY_PRIVATE=1`。
- OpenAPI `components.schemas` / `parameters` / `responses` 已全部接线为 `$ref`（`refSchema` / `refParam` / `refResp`，109 处引用）；`Unauthorized` / `TooManyRequests` / `InternalError` 作为全局错误词汇保留在 `components.responses`（Bearer 鉴权全局生效），不绑定单个端点。
- `POST /api/accounts/preview-buckets` 使用表单临时凭据只读 `ListBuckets` 并立即返回，**不落库、不校验对已有账号**；缺 `endpoint`/`accessKey`/`secretKey` 返回 400，上游 `ListBuckets` 失败返回 500。仍受 `s3wrap.New` 的 endpoint 格式校验与拨号期 SSRF（禁 IMDS/链路本地）防护。
- 桌面端仅做壳与分发，不使用 Tauri IPC。
