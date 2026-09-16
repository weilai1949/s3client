# s3clinet 功能大全（Features）

> 本文件是 s3clinet 的**单一事实来源**：产品能力总览 + 全部已完成修复 / 优化记录。
> 已把散落在各评估文档与 [`CHANGELOG.md`](../CHANGELOG.md) 中的「已实现 / 已修复 / 已完善」功能统一汇总于此；CHANGELOG 仍保留逐字发布历史。
>
> - 待处理事项：[`todolist.md`](todolist.md) · 发版历史：[`CHANGELOG.md`](../CHANGELOG.md) · 综合评估：[`ASSESSMENT.md`](ASSESSMENT.md)
> - 接口细节：[`API.md`](API.md) · 错误约定：[`ERRORS.md`](ERRORS.md) · 开发规范：[`development.md`](development.md) · 安全设计：[`security.md`](security.md) · Nginx 部署：[`deploy/nginx/README.md`](../deploy/nginx/README.md)
>
> 最后更新：2026-09-16（`v1.0.0-rc1` 之后的 Unreleased 区间）

## 目录

- [一、产品功能](#一产品功能)
  - [1. 账号管理](#1-账号管理) · [2. 存储桶与桶属性](#2-存储桶与桶属性) · [3. 对象浏览与检索](#3-对象浏览与检索)
  - [4. 上传](#4-上传) · [5. 下载与预览](#5-下载与预览) · [6. 对象操作](#6-对象操作)
  - [7. 版本与回收站](#7-版本与回收站) · [8. 迁移与增量同步](#8-迁移与增量同步)
  - [9. 存储驱动与数据安全](#9-存储驱动与数据安全) · [10. 服务端安全与鉴权](#10-服务端安全与鉴权)
  - [11. API 与契约](#11-api-与契约) · [12. 前端体验与无障碍](#12-前端体验与无障碍)
  - [13. 桌面端](#13-桌面端) · [14. 部署、CI 与工程化](#14-部署ci-与工程化)
- [二、已完成修复与优化](#二已完成修复与优化) — A 本轮增量 · B 驱动去重明细 · C 全方位评估 58 项 · D v1.0.0-rc1 评估 21 项 · E Optional/Nit 长尾 · F 历史版本全量台账（0.1.0→v1.0.0-rc1） · G Unreleased
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
| 小文件直传 | 服务端 v4 签名 PUT URL，浏览器直传 S3（3 路并发、进度、失败重试） |
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
| 落盘加密 | `S3C_STORE_KEY` → Argon2id + AES-256-GCM，格式 `S3C2｜salt｜ciphertext` |
| 原子写 | 临时文件 + 写后 rename；写前清残骸（`O_EXCL`）；0600 权限 |
| 兼容性 | `json` 驱动同时读明文与历史 S3C2 文件（permissive，写盘换新盐）；`encrypted` 严格 S3C2、复用文件盐 |
| 失败回滚 | `Create`/`Update`/`Delete` 持久化失败回滚内存，避免内存/磁盘漂移 |

### 10. 服务端安全与鉴权
| 能力 | 说明 |
|---|---|
| 鉴权 | Bearer 多 token（`S3C_TOKEN` 逗号分隔，支持轮换）；凭证常量时间比较，scheme 大小写不敏感（RFC 7235） |
| 启动校验 | 短 token（< 16 字符）拒绝启动；非回环未设 token 拒绝启动 |
| CSRF 双防 | CORS 白名单外 Origin 直接 403 + `readJSON` 强制 `application/json` 触发预检 |
| SSRF 双校验 | 创建时校验 + 拨号时二次校验（防 DNS rebinding）；禁重定向；`Proxy=nil` 防环境变量代理绕过 |
| 响应头 | CSP（默认 `connect-src 'self'`，`S3C_CSP_CONNECT_SRC` 可放宽）+ 安全响应头 |
| 限速 | 按客户端 IP（优先 `X-Forwarded-For`）限速 |
| 端点门控 | `/api/openapi.json`（`withOpenAPIGate`）与 `/api/metrics`（`withMetricsGate`）默认 404 |
| 输入防护 | 路径遍历 / 控制字符 key 拒绝；user metadata 边界 400（非 500） |
| 请求 ID | 回显 `X-Request-ID` 仅限 ≤128 可见 ASCII，否则服务端生成，防日志/响应头注入 |
| 错误脱敏 | `UserMessage` 防腐层统一映射，sentinel（`ErrObjectTooLarge` / `ErrSourceDeleteFailed`）替代文案匹配 |
| 账号密钥不出口 | 账号响应为 `AccountView`：**不包含 `secretKey`**，仅 `secretSet: boolean` 表示是否已设置；存储层 `Sanitized()` 占位不再进入 HTTP 响应 |

### 11. API 与契约
| 能力 | 说明 |
|---|---|
| REST 端点 | **69** 个 `/api/*` 端点（含 health/metrics/openapi.json） |
| OpenAPI 3.0.3 | `/api/openapi.json` 自动生成，按域登记（accounts / buckets / bucket-settings / objects / object-meta / multipart / versions / trash / migrate / system） |
| 契约测试 | `routes.go` ↔ 规范双向一致、operation 完整性、路径参数、`$ref` 可解析、`MarshalJSON` 确定性 |
| 错误约定 | 统一 JSON 错误体，见 [`ERRORS.md`](ERRORS.md) |
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
| 供应链 | GitHub Actions 全部 pin SHA；发布产物附 SHA256SUMS |
| 本地 | Makefile（`test` / `test-cover` / `web-test` / `vet` / `install-hooks`）；pre-commit（gofmt + vet + 前端 typecheck） |
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
| `server/internal/store/filestore.go` | 新增（213 行）：共享内存状态 + 锁 + CRUD + 回滚 + `snapshotLocked`/`persistLocked` |
| `server/internal/store/crypto.go` | 新增（66 行）：S3C2 常量、`deriveKey`、`envelope`、AES-256-GCM 加解密 |
| `server/internal/store/store.go` | 244 → 93 行：仅选 `storeCodec`（明文 + 兼容 S3C2） |
| `server/internal/store/encrypted.go` | 267 → 80 行（去重后仅选 `encryptedCodec`）；**再收敛后已删除**，逻辑并入 `store.go` 的 `storeCodec`（strict） |
| `server/internal/store/account_store.go` | `ErrNotFound` 移到接口旁 |
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
| T-6 e2e 覆盖缺口 | ✅ | `e2e/features.spec.ts` 10 用例；14 passed / 1 skipped（Tauri-only 入口） |
| T-7 Makefile | ✅ | `test` 加 `-race`；新增 `test-cover` / `web-test-cover` |
| T-8 store 编译回归 | ✅ | 复用 `encrypted.go` 共享加密助手；store 覆盖率提升 |
| T-9 前端覆盖率 | ✅ | 11.28% → 95.27% statements；36 组件 + 全部 composables 覆盖；顺带修 3 个缺陷（dropzone 点击递归、MigratePanel ResizeObserver 时机、deleteServer 当前服务器 applyProfile） |

#### 文档与 DevOps
| 项 | 状态 | 修复 |
|---|---|---|
| D-1 pre-commit | ✅ | `.githooks/pre-commit`（gofmt + go vet + fe typecheck）+ `make install-hooks` |
| D-2 e2e 重试 | ✅ | Playwright CI retries 2 |
| D-3 Makefile 缺失 | ➖ | 根 Makefile 存在且已强化 |
| D-4 ERRORS.md 未更新 | ✅ | 补充 migrate/sync 与 openapi.json 的错误约定 |

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
| `metadata.go:40` 变量遮蔽 | ✅ | 循环变量改名 |
| `.env` 按 CWD 相对加载 | ✅ | `S3C_ENV_FILE` + 可执行文件同目录候选 |
| UserMessage 字符串匹配 `"exceeds 5GB"` | ✅ | sentinel + `errors.Is` |
| `validBucketName` 首/尾字符 | ✅ | 拒绝首尾 `.`/`-` 与连续分隔符 |
| 前端 `api.ts` 死代码 | ✅ | `requestBlob` / `downloadZip` 已删 |
| 同批 files 排序两次 | ✅ | 已移除重复排序 |
| 13 处 `ctx.account.value!.id` | ✅ | 收敛入口，剩余 3 处本轮清理 |
| blob 下载同步 `revokeObjectURL` | ✅ | 延迟 60s |
| `{} as KeyBindings` cast hack | ✅ | 改 `const bindings: KeyBindings = {}` + 可选字段 |
| pnpm 版本口径不一（9 vs 11） | ✅ | 全仓统一 `9.15.0`（`packageManager` + 各 workflow + Dockerfile） |
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
- **四轴评审修复**：Vue `key` 为保留属性导致 7 个弹窗实际不可用 → 统一 `objectKey`（Vue 3.5 SSR 复现验证）；桶设置页签 `watch` 补 `immediate`（此前永不加载、保存会用默认值覆盖）；CORS 非白名单普通请求直接 403 + `readJSON` 强制 JSON；`inline` 代理 MIME 白名单；代理支持 `versionId` + `Accept-Ranges`/416；创建账号忽略客户端 id；ZIP 条目 `/` 与 `\` 双消毒；`.env` 加载；`PurgeObject` 改 `DeleteObjects` 批量 1000/批；`deletePrefix` 页前检查 + 跨页切分；multipart 段号 1..10000 且唯一校验；copy/migrate 4 路有界并发、migrate 限 10k key、failKeys ≤200；列表导航序号防过期响应；上传入队捕获所属桶；焦点陷阱与初始焦点；版本比较下载走服务端代理。
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
- 文档：`docs/API.md`、配置矩阵、运行 / 部署说明。

#### 0.1.0（2026-08-22）
- 首个版本：Go 后端（AWS SDK for Go v2，封装 11 个 S3 接口）、Vue3 + Vite + TS 前端（账号 / 列对象 / 直传 / 签名 / 删除 / 迁移 / 加前缀）、Tauri 2 桌面壳（无 IPC，B/S）。

### G. Unreleased（进行中）已记录项
运行时基镜像 Alpine 3.20、`/api/metrics` 默认关闭、S3C_TOKEN 短口令硬失败、S3 出站禁代理、桶名校验收紧、user metadata 400、delete-objects ≤1000、migrate SSE 写超时、store 失败回滚、错误 sentinel 化、X-Request-ID 加固、Bearer scheme、`.env` 解耦、nginx 内存上限、OpenAPI 自动生成 + 契约测试、**OpenAPI components 接线 `$ref`（109 处引用，消灭 0 引用死代码）**、双驱动去重、存储驱动再收敛（统一 `storeCodec`）、账号响应契约收敛（`secretSet` 替代 `"******"` 占位）、全仓 gofmt 对齐、Playwright E2E、增量同步、桶策略可视化编辑器、批量元数据编辑、存储类型切换、回收站、桌面端 SHA 校验与 actions pin SHA 等。完整逐条见 [`CHANGELOG.md`](../CHANGELOG.md)。

---

## 三、质量与覆盖率现状

> 2026-09-15 本机实测。

| 门禁 | 结果 |
|---|---|
| `go vet ./...` | 干净 |
| `go test -race -count=1 ./...` | 8/8 包通过 |
| 后端覆盖率 | **每个包 + 汇总均 100.0% statements**（main / config / model / openapi / store / service / s3wrap / handler） |
| 前端 `pnpm test` | 61 文件 / 935 测试全绿 |
| 前端覆盖率 | **statements / branches / functions / lines 均 100%** |
| `vue-tsc --noEmit` / `vite build` | 干净 / OK（~355KB，gzip ~109KB） |
| `eslint` | 0 违规（`no-explicit-any: error`） |
| `gofmt -l .` | 干净 |
| `docker compose config` | base / prod / tls 均通过 |
| E2E（Playwright） | 14 passed / 1 skipped（Tauri-only 入口） |
| E2E（真实 RustFS，`S3CLINET_E2E=1`） | 按需运行，默认不阻塞 CI |

### 已知边界与取舍
- OpenAPI `components.schemas` / `parameters` / `responses` 已全部接线为 `$ref`（`refSchema` / `refParam` / `refResp`，109 处引用）；`Unauthorized` / `TooManyRequests` / `InternalError` 作为全局错误词汇保留在 `components.responses`（Bearer 鉴权全局生效），不绑定单个端点。
- `POST /api/accounts/preview-buckets` 使用表单临时凭据只读 `ListBuckets` 并立即返回，**不落库、不校验对已有账号**；缺 `endpoint`/`accessKey`/`secretKey` 返回 400，上游 `ListBuckets` 失败返回 500。仍受 `s3wrap.New` 的 endpoint 格式校验与拨号期 SSRF（禁 IMDS/链路本地）防护。
- 桌面端仅做壳与分发，不使用 Tauri IPC。
