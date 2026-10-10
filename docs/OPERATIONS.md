# 运维手册（Operations）

> 本文件覆盖**自托管单实例**的日常运维：可观测性（健康端点 / 指标 / 日志）、SLO 与告警基线（建议值）、
> 故障处置 Runbook、备份与恢复、灾难恢复、事故响应、容量与单实例约束。
> **不覆盖**：部署形态与 compose / nginx / TLS 细节（见 [`DEPLOYMENT.md`](DEPLOYMENT.md)）、
> 配置项的取值与约束（唯一事实来源是 [`CONFIGURATION.md`](CONFIGURATION.md)，本文件只引用结论）、
> 接口字段（见 [`api.md`](api.md)）、错误码文案（见 [`errors.md`](errors.md)）、威胁模型（见 [`threat-model.md`](threat-model.md)）。
> 文中每条命令、路径、指标名、环境变量名均取自仓库源码；凡属**建议**而非代码强制的数值，均在原地显式标注。

## 1. 适用范围与读者

| 读者 | 最需要的章节 |
|---|---|
| 自托管实例的运维者（单机 / 单副本） | §5 Runbook、§6 备份与恢复、§10 容量与单实例约束 |
| 值班 / 响应者（接到告警或用户报障） | §3 可观测性、§4 告警基线、§5 Runbook、§9 事故响应 |
| 想核对「部署后行为是否符合预期」的贡献者 | §2 服务形态速查、§3 可观测性、§8 升级与回滚 |

**前置假设**：已按 [`DEPLOYMENT.md`](DEPLOYMENT.md) 完成部署；能访问 server 的 HTTP 端口与宿主机 / 容器 shell；
知道本实例 `S3C_DATA_DIR` 的实际路径（默认 `./data`，容器内 `/data`）。

### 1.1 本文件的证据等级约定

全文对每条结论标注来源强度，**不要把「建议值」当成已生效的配置**：

| 标记 | 含义 | 例子 |
|---|---|---|
| **代码强制** | 由源码 / 进程行为 / 事务强制，不满足即拒绝启动或返回错误 | fail-closed 启动校验、`/api/health` 503 硬失败、`flock` 单写者锁、指标名与并发上限常量 |
| **文档约定** | 仓库文档已声明；实现是「尽力而为」，失败被忽略 | 数据目录 0700 / 文件 0600 的 `chmod`（失败不影响启动） |
| **建议值** | 本文件给出的运维基线，**未在代码或 CI 中强制** | SLO 目标、告警阈值、RTO / RPO、备份频率、值班轮换、容量水位 |

> 本文件与源码冲突时**以源码为准**；发现漂移请按 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 同步修正（运维行为相关的代码改动会触碰本文件）。

## 2. 服务形态速查

一句话：**Go 后端（默认 `127.0.0.1:8080`）+ 可选 nginx 单 worker 反向代理（容器内 `:8080`）+ 文件型账号存储**，
三者都由 [`docker-compose.yml`](../docker-compose.yml) / [`docker-compose.prod.yml`](../docker-compose.prod.yml) /
[`docker-compose.tls.yml`](../docker-compose.tls.yml) 编排；对象数据不在本机，全部在 S3 上游。

下表只列**运维动作会用到**的形态信息；部署形态选择、compose 差异、TLS、桌面端分发见 [`DEPLOYMENT.md`](DEPLOYMENT.md)。

| 组件 | 进程 / 端点 | 信号与关停 | 日志位置 |
|---|---|---|---|
| Go 后端 | 默认 `127.0.0.1:8080`（`S3C_ADDR`）；容器内 `0.0.0.0:8080`，非 root 用户 `app`（uid 1000） | `SIGTERM` → 取消异步任务 → `http.Server.Shutdown`（上限 `S3C_SHUTDOWN_TIMEOUT`） | 容器：`docker compose logs server`（json-file，10m × 3 或 5）；本机：`.run/server.log` |
| nginx | 单 worker 反向代理 / 静态托管；compose 发布 `127.0.0.1:8080:8080` | `SIGQUIT` 优雅停止；`nginx -s reload` 热加载 | `docker compose logs nginx` |
| 账号存储 | 文件型：`accounts.json` / `accounts.db` / `accounts.json.enc`（取决于 `S3C_STORE_DRIVER`）+ `jobs.json` 任务清单 + `schedules.json` 计划任务（0600） + `shutdown.json` 上次关停耗时 | 随进程退出释放 `flock` | 不单独打日志，错误由后端日志承载 |

## 3. 可观测性

### 3.1 健康端点与语义

```
GET /api/health        # 免鉴权（withAuth 显式豁免），响应恒带 X-Request-ID
```

| 状态 | 响应体（逐字字段） | 语义 |
|---|---|---|
| 200 | `{"status":"ok","version":"v1.0.0","time":"...","store":{"ok":true}}` | store 探测通过 |
| 503 | `{"status":"error","version":"v1.0.0","time":"...","store":{"ok":false,"error":"store unavailable"}}` | store 探测失败；**硬失败不降级**（[ADR-002](decisions/0002-store-fail-closed.md)） |

- 503 时**不回传**底层错误原因：原始错误只写进 Debug 日志（`msg="health store ping"`，需 `S3C_LOG_LEVEL=debug`）。
- `version` 由构建时注入（`make server-build` / Dockerfile 的 `-X main.version`），可用于核对**前后端版本是否匹配**。
- **探测粒度按驱动不同（易误判，务必知悉）**：`sqlite` 驱动的 `Ping` 是真实的 `db.Ping()`；`json` / `encrypted`
  驱动的 `Ping` **恒返回 nil**（`apps/server/internal/store/filestore.go`）——因此后两种驱动下，磁盘写满 / 只读挂载
  **不会**让 `/api/health` 变 503，会以写请求 500 的形式暴露（见 §5 R-3）。
- 容器 `HEALTHCHECK` 走 `s3client-server -healthcheck` 子命令（GET `http://<S3C_ADDR host>:<port>/api/health`，3 秒超时，200 → 0），
  间隔 30s / 超时 5s / 重试 3 次 / start-period 5s；compose 的 nginx 依赖 `condition: service_healthy`。
- 恢复后**无需重启**：下一次探测成功即回到 200（[`DEPLOYMENT.md`](DEPLOYMENT.md) §6.1）。
- `/api/health` 是**免鉴权**端点（Docker 探针需要），`version` 暴露属已接受项（[`threat-model.md`](threat-model.md) §6.2）。

```bash
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/api/health   # 200 / 503
```

### 3.2 指标全清单

```
GET /api/metrics       # Prometheus 文本格式；默认 404，仅 S3C_EXPOSE_METRICS=1 时 200
```

> ⚠️ **`/api/metrics` 不受 `S3C_TOKEN` 保护**：开启后**匿名可读**（含版本、存储可达性、S3 上游调用统计、SSRF 生效策略）。
> 只在内网 / 反向代理鉴权之后暴露，切勿把开启 metrics 的实例直接放上公网。

暴露方式：设置 `S3C_EXPOSE_METRICS=1` 后重启进程（配置在启动时读取）。

| 指标（逐字） | 类型 | 含义 | 建议告警用法（**建议值，未强制**） |
|---|---|---|---|
| `s3c_http_requests_total` | counter | 已处理的 HTTP 请求总数（所有响应） | 作为错误率分母 |
| `s3c_http_responses_total{class="2xx"\|"4xx"\|"5xx"}` | counter | 按状态类分桶的响应数 | `5xx` 增速 > 1%/5m 告警（见 §4） |
| `s3c_http_request_duration_seconds` | histogram | **HTTP 请求处理耗时**；输出 `_bucket{le=...}` / `_sum` / `_count`（含 proxy / download / zip 等流式端点） | `histogram_quantile(0.95, ...)` > 5s 持续 10 分钟告警（建议，见 §4.2） |
| `s3c_stream_interrupted_total` | counter | 流式传输**在完成前中断**的次数（上游读失败 / 写超时；客户端主动断开不计入，只记 Debug 日志） | 15 分钟内有增量即查（R-5） |
| `s3c_zip_partial_failures_total` | counter | ZIP 打包「部分成功」次数 | 有增量即查（ZIP 缺文件比整体失败更隐蔽） |
| `s3c_zip_failed_keys_total` | counter | ZIP 打包中失败对象的累计个数 | 与上一项联动定位是单对象还是批量失败 |
| `s3c_zip_failed_total` | counter | ZIP 打包整体失败次数 | 有增量即查 |
| `s3c_uptime_seconds` | gauge | 进程运行秒数 | 突降 = 发生过重启（可用于发现容器重启循环） |
| `s3c_go_goroutines` | gauge | goroutine 数 | 持续单调上升 = 疑似泄漏 |
| `s3c_go_memstats_alloc_bytes` | gauge | 当前堆占用字节 | 逼近容器内存上限（server 512M）即告警 |
| `s3c_build_info{version="..."}` | gauge | 构建版本（恒为 1，版本在标签里） | 升级后核对版本是否与预期 tag 一致 |
| `s3c_store_up` | gauge | 账号存储可达性：1 / 0 | `== 0` 持续 1 分钟即告警（沿用 [`DEPLOYMENT.md`](DEPLOYMENT.md) §6.1 建议） |
| `s3c_store_write_failures_total` | counter | **账号库写入失败次数**（落盘 / SQL 写入出错、写操作已回滚；重复 ID、NotFound 这类业务拒绝**不计数**） | 15 分钟内有增量即告警（critical，见 §4.2）——`json` / `encrypted` 驱动唯一的主动存储故障信号 |
| `s3c_volume_size_bytes` / `s3c_volume_free_bytes` | gauge | **数据卷容量**：`S3C_DATA_DIR` 所在文件系统总字节 / 本进程可用字节 | 剩余占比 < 20% 持续 10 分钟告警（见 §4.2）；**序列缺失 = 取不到**（见下方口径） |
| `s3c_jobs_active` | gauge | **在册（未终结）异步任务数**，与 `JobRegistry` 上限同口径（上限 256） | `>= 230` 持续 10 分钟告警（见 §4.2） |
| `s3c_persist_failures_total` | counter | **计划 / 任务清单落盘（`Save`）失败次数**（内存态保真、失败降级；此前为静默，KNOWN_ISSUES #83） | 有增量即查（磁盘满 / 只读）；与 `s3c_store_write_failures_total` 区分——后者只覆盖账号库 |
| `s3c_last_shutdown_duration_seconds` | gauge | **上一次优雅关停耗时**（由 `data/shutdown.json` 在启动时载入；0 = 尚无记录） | 与 `S3C_SHUTDOWN_TIMEOUT`（默认 30s）对比，逼近即说明关停吃紧 |
| `s3c_ssrf_deny_private` | gauge | SSRF 生效策略：1 = 拒绝私网 / 回环 S3 端点 | 与预期配置比对（`S3C_SSRF_DENY_PRIVATE` 是否真的生效） |
| `s3c_s3_calls_total` | counter | S3 上游 API 调用总数（成功 + 失败） | 作为上游错误率分母 |
| `s3c_s3_call_errors_total{code="..."}` | counter | 上游失败调用按错误类分类 | 按 `code` 分诊（R-6）；`SignatureDoesNotMatch` / `InvalidAccessKeyId` 出现即查 |
| `s3c_s3_call_duration_seconds` | histogram | 上游调用耗时；输出 `_bucket{le=...}` / `_sum` / `_count` | `histogram_quantile(0.95, ...)` > 2s 持续 10 分钟告警（建议） |
| `s3c_s3_stream_bytes_total` | counter | 经本服务从 S3 流式读出的字节数 | 与业务量比对；中断时用于估算已读量 |

直方图桶上界（逐字）。上游调用 `s3c_s3_call_duration_seconds`：`"0.01"`、`"0.05"`、`"0.1"`、`"0.25"`、`"0.5"`、`"1"`、`"2.5"`、`"5"`、`"10"`、`"30"`、`"+Inf"`；
HTTP 请求 `s3c_http_request_duration_seconds`：`"0.005"`、`"0.01"`、`"0.025"`、`"0.05"`、`"0.1"`、`"0.25"`、`"0.5"`、`"1"`、`"2.5"`、`"5"`、`"10"`、`"30"`、`"+Inf"`。
两者的 `_bucket{le="+Inf"}` 都恒等于 `_count`，可用于自检指标完整性
（HTTP 侧还恒等于 `s3c_http_requests_total`——同一次请求在同一处记录，不可能漂移）。

**`s3c_s3_call_errors_total` 的 `code` 标签取值是有限白名单**（防止不可信上游用任意 `<Code>` 撑爆标签基数）：
`AccessDenied`、`BucketNotEmpty`、`EntityTooLarge`、`InvalidAccessKeyId`、`InvalidArgument`、`InvalidPartOrder`、
`InvalidRange`、`InvalidRequest`、`InvalidStorageClass`、`MalformedPolicy`、`MalformedXML`、`NoSuchBucket`、
`NoSuchKey`、`NoSuchUpload`、`NoSuchVersion`、`NotFound`、`RequestTimeout`、`ServiceUnavailable`、
`SignatureDoesNotMatch`、`SlowDown`；白名单之外的 API 错误码**一律收敛为 `other`**；非 API 错误归
`transport`（连接 / DNS 等）、`timeout`（上下文超时）、`canceled`（上下文取消）。

> **指标基线特性（排查前必读）**：除 `s3c_last_shutdown_duration_seconds`（唯一落盘的指标，
> 见 `data/shutdown.json`）外，全部指标是**进程级内存计数**，进程重启即清零；不落盘、无历史。
> 因此「重启后再看指标」无法判断故障前状态——**先取证，再重启**（§5 开头）。

**2026-09-30 观测缺口已全部补齐**（原「账号库写入失败次数、在册任务数、HTTP 请求延迟直方图、
卷 / 磁盘容量、优雅关停耗时」五项，现均为上表内置指标；补齐记录见 [`ROADMAP.md`](ROADMAP.md)
空号 **#18** 与 [`FEATURES.md`](FEATURES.md) §BL）。仍**只能**外部采集的观测面只剩两类：
**inode 用量**与**跨服务 trace**（见 §4.3）。

> **卷容量序列缺失 ≠ 容量为 0**：`s3c_volume_*` 只在 statfs / `GetDiskFreeSpaceExW` 取到结果时输出。
> 平台不支持（Linux / macOS / FreeBSD / Windows 之外）、`S3C_DATA_DIR` 未传给 `/api/metrics`
> 或 statfs 失败（路径不存在）时**不发序列**，此时按 §4.3 用宿主 `node_exporter` / `df` 采集。

```bash
# 只取关键几行
curl -sS http://127.0.0.1:8080/api/metrics | grep -E '^s3c_(store_up|ssrf_deny_private|stream_interrupted_total|s3_call_errors_total)'
```

### 3.3 日志

| 项 | 取值 / 行为 | 依据 |
|---|---|---|
| 级别 | `S3C_LOG_LEVEL` = `debug` / `info` / `warn` / `error`，默认 `info` | [`CONFIGURATION.md`](CONFIGURATION.md) |
| 格式 | `S3C_LOG_JSON=1` → `slog` JSON（容器 / 生产推荐，compose 默认注入 `1`）；否则纯文本 | `main.go` |
| 启动日志 | `msg="s3client server"` + `version` / `addr` / `dataDir` / `store` / `staticDir` / `auth` / `cors` / `region` / `ssrfDenyPrivate` / `tracing` | `main.go` |
| 关停日志 | `msg="shutting down..."` → `msg="shutdown complete"`；两者间隔即实际关停耗时（同一耗时也落盘为 `s3c_last_shutdown_duration_seconds`，见 §3.2） | `main.go` |
| 访问日志 | `msg="http"` + `method` / `path` / `status` / `dur` / `req` | `middleware.go` |
| 审计日志 | `msg="audit"` + `audit`（事件名）/ `ip` / `method` / `path`（+ 事件附加字段，如账号 id、bucket） | `handler/audit.go` |
| 关键错误日志 | `msg="配置校验失败"`、`msg="data dir lock"`、`msg="init store"`、`msg="handler error"`、`msg="stream interrupted"` | `main.go` / `store` / `handler` |

**`req` 与 `X-Request-ID` 的对应关系**：服务端对**每个**响应回写 `X-Request-ID`；客户端传入的值只有
「≤128 且全为可见 ASCII」才被透传，否则服务端生成 UUID。访问日志字段 `req` 就是该值——**用响应头里的
`X-Request-ID` 去日志里 grep 同值**即可定位单次请求：

```bash
curl -sS -D- -o /dev/null http://127.0.0.1:8080/api/accounts | grep -i x-request-id
# 容器（S3C_LOG_JSON=1）
docker compose logs --since 15m server | grep '"req":"<上面拿到的 id>"'
```

> 注意：`/api/health` 与 `/api/metrics` 的访问日志**只在 Debug 级别输出**（避免探针刷屏）；默认 `info` 下
> 看不到这两条。
>
> **nginx 访问日志已带请求 ID**（2026-09-29 修复 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #68）：
> `deploy/nginx/nginx.conf` 与 `nginx.docker.conf` 的 `log_format main` 现含两个字段——
> `rid=$http_x_request_id`（客户端**请求**里带的值，用户从报错提示抄下来的通常是它）与
> `req=$upstream_http_x_request_id`（**后端在响应上回显**的 ID，即后端访问日志里的 `req`，
> 两者同源，**这才是用来对齐的值**）。只记后者会在客户端传了 ID 时丢掉原始值；只记前者会在
> 客户端没传时记成空——而那正是最常见的情形（浏览器不会自己加这个头）。由
> `repo_infra_gate_test.go` 的 `TestNginxAccessLogCarriesRequestID` 守住。
> 于是跨层定位链路完整：`docker compose logs nginx | grep 'req=<id>'` → `docker compose logs server | grep '"req":"<id>"'`。

**审计日志覆盖的安全敏感动作**（事件名是稳定契约，可直接做检索与告警）：
`auth.denied`、`account.create`、`account.update`、`account.delete`、`bucket.policy.update`、`bucket.policy.delete`、
`objects.delete`、`objects.delete_prefix`、`objects.move`、`trash.purge`、`rate_limit.exceeded`。
审计日志**不记录任何密钥 / 凭据值**；`ip` 只在 `S3C_TRUSTED_PROXIES` 命中时采信 `X-Forwarded-For`（默认不信任，防伪造）。

```bash
# 本机开发进程日志（scripts/run-dev.sh、scripts/graceful-restart.sh 写入；S3C_RUN_DIR 可改目录）
tail -f .run/server.log
```

### 3.4 OTel tracing（可选，默认关闭）

| 项 | 行为 |
|---|---|
| 开启 | 设 `S3C_OTEL_ENDPOINT=http://<collector>:4318`（标准 OTLP/HTTP 端口，导出 `POST {Endpoint}/v1/traces`） |
| 关闭（默认） | `S3C_OTEL_ENDPOINT` 留空：中间件原样透传、不注入响应头、不导出，零开销 |
| 采样 | `S3C_OTEL_SAMPLE_RATIO`（默认 `1`）只决定**新建 trace**；入站 `traceparent` 的采样位被继承（`01` 即使 ratio=0 也采样，`00` 不导出，不做尾部采样） |
| 服务标识 | `S3C_OTEL_SERVICE_NAME`（默认 `s3client`）写入 resource attribute `service.name` |
| 导出内容 | W3C `traceparent`（解析请求头 / 回写响应头）+ 每请求一个 server span；`presign` / `proxy` / `migrate`（含 `/api/migrate/sync` 与 `/api/migrate/async`）子 span。server span 带 `http.request.method` / `url.path` / `http.response.status_code` / `request.id`（即 §3.3 的 `X-Request-ID`，同一请求两处同值） |

**采集端期望**：标准 OTLP/HTTP JSON 接收端（如 otel-collector 的 `otlp` receiver——默认同时监听
gRPC 4317 与 HTTP 4318；本服务**只发 HTTP/JSON**，不发 protobuf / gRPC）。`traceId` / `spanId` 为小写
十六进制；时间戳与 int64 属性按 OTLP/JSON 约定编码为十进制字符串；`kind` 按 OTLP/JSON 约定用整数。
导出报文仅覆盖本服务实际产出的字段子集（无 events / links / status / 自定义 resource 属性）。

**失败模式**（一律「只记日志、不影响请求」）：

- 网络失败 / 采集端非 2xx：WARN `tracing: OTLP 导出失败` 或 `tracing: OTLP 采集端返回非 2xx`，请求照常返回。
- 有界队列满（默认 512 个待导出 span）：丢弃新 span 并 WARN `tracing: span 队列已满，丢弃 span`（`droppedTotal` 累计）；**不阻塞请求路径**。
- 序列化 / 构造请求失败：WARN 后跳过该批（正常配置下不可达，属防御分支）。
- 关停：SIGTERM 后 `Close` 刷出剩余 span 并等待后台导出结束（最多 5s），随后进程退出；采集端恰在此窗口不可用时，最后一批 span 可能丢失。

**跨层定位**：响应头 `traceparent`（`00-<traceId>-<spanId>-<flags>`）里的 `traceId` 与子 span 一致，
配合 §3.3 的 `req=<X-Request-ID>`（即 span 属性 `request.id`），可把一次请求在采集端与日志之间对齐。
依赖预算与取舍见 [ADR-0013](decisions/0013-zero-dep-otlp-tracing.md)。

## 4. SLO / 告警基线

> ⚠️ **本节为建议基线，尚未在代码或 CI 中强制**——按实际部署调整后再落地。
> **可执行形态**：[`deploy/prometheus/s3client.rules.yml`](../deploy/prometheus/s3client.rules.yml)
> （2026-09-29 新增）把下面的 SLI 记录规则与全部告警规则落成 Prometheus 规则文件，挂进
> `rule_files` 即可用。它与本节**必须同改**：该文件由
> `repo_infra_gate_test.go` 的 `TestPrometheusRulesReferenceRealMetrics` 机械校验
> ——引用的每个 `s3c_*` 指标必须真实存在于发射点、每个 `code` 取值必须在 `s3wrap` 白名单内
> （白名单外的码会被折叠成 `other`，写进表达式即**永不命中**且 Prometheus 不报错）。
> **SLO 仪表盘已随仓库分发**：[`deploy/grafana/s3client.dashboard.json`](../deploy/grafana/s3client.dashboard.json)
> （2026-09-30 新增）。Grafana → **Dashboards → New → Import** → 上传该 JSON → 数据源选你的 Prometheus 即可；
> 盘上的 `s3client:*` 记录规则与本节表格同源（**本节与 rules.yml 必须同改**），面板在规则未加载时也各有等价的原始表达式。
> 该 JSON 由 `grafana_dashboard_gate_test.go` 机械校验：引用的每个指标 / `code` 取值 / 记录规则名都必须真实存在。
> **OTel trace 已于 2026-10-08 落地**（§3.4；`S3C_OTEL_ENDPOINT` 默认空 = 关闭，见 [ADR-0013](decisions/0013-zero-dep-otlp-tracing.md)）；告警规则本身已随仓库分发。

### 4.1 建议 SLI / SLO

单实例 + 文件型 store + 无 HA（§7.2），因此 SLO 只能按「单机可用性」定义；任一 SLO 都**不含**上游 S3 的可用性
（那部分只统计为错误，不计入本服务违约）。

| SLI | 计算方式（用现有指标） | 建议目标（**未强制**） |
|---|---|---|
| 服务可用性 | 外部探测 `/api/health` 返回 200 的成功率 | ≥ 99.5% / 月 |
| 存储可用性 | `avg_over_time(s3c_store_up[30d])` | ≥ 99.9% / 30 天（注意仅 `sqlite` 驱动能实时反映，见 §3.1） |
| 服务端错误率 | `sum(rate(s3c_http_responses_total{class="5xx"}[5m])) / sum(rate(s3c_http_requests_total[5m]))` | ≤ 0.5% |
| HTTP 延迟 | `histogram_quantile(0.95, sum(rate(s3c_http_request_duration_seconds_bucket[5m])) by (le))` | ≤ 5s（含 proxy / download / zip 流式端点，长尾天然存在；普通 API 看 p50 分布） |
| 账号库写入 | `increase(s3c_store_write_failures_total[15m])` | 0（出现即查；业务性拒绝不计数） |
| 数据卷剩余空间 | `s3c_volume_free_bytes / s3c_volume_size_bytes` | ≥ 20%（序列缺失的平台按 §4.3 外部采集） |
| 上游错误率 | `sum(rate(s3c_s3_call_errors_total[5m])) / sum(rate(s3c_s3_calls_total[5m]))` | ≤ 1%（`canceled` 属客户端取消，评估时可剔除） |
| 上游延迟 | `histogram_quantile(0.95, sum(rate(s3c_s3_call_duration_seconds_bucket[5m])) by (le))` | ≤ 2s |
| 流式中断 | `increase(s3c_stream_interrupted_total[1h])` | ≤ 1 次 / 小时 |
| ZIP 打包质量 | `increase(s3c_zip_partial_failures_total[1h])` + `increase(s3c_zip_failed_total[1h])` | 0（出现即查） |

### 4.2 建议告警规则

| 告警 | 表达式（Prometheus 语法） | 建议阈值（**未强制**） | 为什么这样定 |
|---|---|---|---|
| 账号存储掉线 | `s3c_store_up == 0` | `for: 1m` | 硬失败设计下此时写操作全在拒绝；沿用 [`DEPLOYMENT.md`](DEPLOYMENT.md) §6.1 建议 |
| 健康探测失败 | 外部黑盒探测 `/api/health` | 连续 3 次失败（30s 间隔） | 覆盖 503 与进程 / 端口级故障 |
| 5xx 比例升高 | 见 §4.1 服务端错误率 | `> 1%` 持续 10 分钟 | 与 SLO（0.5%）留一倍缓冲，避免抖动误报 |
| 凭据类上游错误 | `increase(s3c_s3_call_errors_total{code=~"SignatureDoesNotMatch\|InvalidAccessKeyId\|AccessDenied"}[15m]) > 0` | 立即 | 账号密钥 / 桶策略漂移，不修则全账号不可用 |
| 上游限流 / 过载 | `increase(s3c_s3_call_errors_total{code=~"SlowDown\|ServiceUnavailable\|RequestTimeout"}[15m])` | `> 5` | 上游按量限流，继续加压会放大失败 |
| 上游不可达 | `increase(s3c_s3_call_errors_total{code="transport"}[15m])` | `> 0` | DNS / 连接层问题（区别于 S3 业务错误码） |
| 流式中断 | `increase(s3c_stream_interrupted_total[15m]) > 0` | 立即 | 客户端主动断开不计入，故有增量即真实中断 |
| 进程重启 | `resets(s3c_uptime_seconds[15m]) > 0` | 立即 | 指标为进程级、重启清零；非计划重启需查因 |
| 内存逼近上限 | `s3c_go_memstats_alloc_bytes` | > 容器上限的 80%（server 512M → 约 410M） | 容器内存上限由 compose 强制 |
| 账号库写入失败 | `increase(s3c_store_write_failures_total[15m]) > 0` | 立即（critical） | `json` / `encrypted` 驱动的 `Ping` 恒 nil、`s3c_store_up` 对它们永远是 1——落盘失败计数是这两类驱动**唯一主动**的存储故障信号；重复 ID / NotFound 等业务拒绝不计数，故有增量即真实故障 |
| 在册任务逼近上限 | `s3c_jobs_active >= 230` | `for: 10m` | 上限 256，满则异步迁移 / 复制全部 503；230 ≈ 90%，给 reap 回收终态任务留观察窗 |
| 数据卷剩余空间 | `s3c_volume_free_bytes / s3c_volume_size_bytes < 0.2` | `for: 10m` | 卷写满会直接导致账号库落盘失败（与上一行联动）；序列缺失的平台按 §4.3 外部采集 |
| HTTP 延迟升高 | `s3client:http_latency_p95:rate5m > 5` | `for: 10m` | 先排除流式端点，再对照上游耗时直方图区分「上游慢」与「本服务排队」 |

### 4.3 必须靠外部采集的观测面（本服务不提供指标）

| 关注点 | 现状 | 建议做法（**建议值**） |
|---|---|---|
| 数据卷 **inode**（容量本身已内置 `s3c_volume_size_bytes` / `s3c_volume_free_bytes`，2026-09-30） | 仍无内置指标 | 宿主 `node_exporter` / `df -i` 采集，inode 使用率 > 80% 告警；卷字节使用率直接用 §4.2 的 S3ClientVolumeSpaceLow |
| 数据卷容量（**仅在序列缺失的平台**：Linux / macOS / FreeBSD / Windows 之外） | 该平台 `volumeUsage` 无免依赖实现 → 不发序列 | 宿主 `node_exporter` / `df -h` 采集，卷使用率 > 80% 告警 |
| 账号库可写性（`json` / `encrypted`） | **主动探针仍无**（`Ping` 恒 nil）；但**写入失败已内置计数** `s3c_store_write_failures_total`（事后信号，2026-09-30） | 用 §4.2 的 S3ClientStoreWriteFailures（critical）+ 5xx 比例 + `msg="handler error"` 日志三者交叉确认（R-3） |
| 跨服务 trace | 未接入（[`ROADMAP.md`](ROADMAP.md) §三 3.2 #11 ⬜） | 现阶段用 `X-Request-ID` + nginx 日志关联 |

> 2026-09-30 起，本表原有的「数据卷**容量** / 在册异步任务数 / 优雅关停耗时」三项已由内置指标补齐
> （`s3c_volume_*`、`s3c_jobs_active`、`s3c_last_shutdown_duration_seconds`，见 §3.2 与 §4.2），
> 补齐记录见 [`ROADMAP.md`](ROADMAP.md) 空号 **#18** 与 [`FEATURES.md`](FEATURES.md) §BL。

## 5. Runbook（故障处置）

> **铁律：先取证，再重启。** 指标是进程级内存计数（重启清零），重启还会把进行中的异步任务标记为
> `interrupted` 并要求人工对账。重启前至少保存：`/api/health` 响应、`/api/metrics` 全文、最近 15 分钟日志、
> `S3C_DATA_DIR` 下的文件清单（`ls -l`）与 `jobs.json` 副本。

通用取证顺序（后续每条 Runbook 的「判据」都基于这套命令）：

```bash
curl -sS -o /dev/null -w 'health=%{http_code}\n' http://127.0.0.1:8080/api/health
curl -sS http://127.0.0.1:8080/api/metrics | grep -E '^s3c_'          # 需 S3C_EXPOSE_METRICS=1
make status                                                          # 本机 PID + health(8080) / web(1949)
docker compose -f docker-compose.prod.yml logs --since 15m server     # 容器部署
docker compose -f docker-compose.prod.yml ps
ls -l "${S3C_DATA_DIR:-./data}"                                       # 本机默认 ./data；容器内为 /data；目录 0700 / 文件 0600
```

### R-1 账号存储不可达（`/api/health` 503、`s3c_store_up == 0`）

**症状**：`/api/health` 返回 503（`"status":"error"`、`store.ok=false`）；`s3c_store_up 0`；前端健康轮询提示不可用；
容器 `HEALTHCHECK` 失败，`restart: unless-stopped` 触发反复重启。

**判据**：
1. `curl` 状态码 503 且 `store.error == "store unavailable"`；
2. `s3c_store_up 0`（需 `S3C_EXPOSE_METRICS=1`）；
3. `S3C_LOG_LEVEL=debug` 时日志出现 `msg="health store ping"` + `err`（**根因只有这里能看到**）；
4. 容器 `docker inspect --format '{{.State.Health.Status}}' s3client-server` = `unhealthy`。

**处置步骤**：
1. **先确认驱动**：只有 `sqlite` 驱动的探测是真实 IO（`db.Ping`）。若本实例是 `json` / `encrypted` 却出现
   503，说明问题不在磁盘，而在进程 / 端口 / 前置代理层 —— 按 R-3、R-10 排查。
2. 查数据目录可用性：卷是否写满、是否只读挂载、`S3C_DATA_DIR` 是否指向了错误路径（容器内为 `/data`）；
   目录应为 0700、文件 0600、属主 `app`（容器 uid 1000）。
3. `sqlite` 驱动额外检查：`accounts.db` 与侧车 `accounts.db-wal` / `accounts.db-shm` 是否同一卷、是否被
   外部进程（如宿主 `sqlite3`）长时间占用。
4. **不要靠删库 / 换空目录「恢复服务」**——那等于清空全部账号配置。恢复优先级：修卷 → 修权限 → §6 从备份恢复。
5. 恢复后无需重启：下一次探测成功即自动回到 200。

**何时升级**：503 持续 > 10 分钟且伴随卷满 / 库文件损坏；或重启后 `msg="init store"` 仍失败 → 进入 §6 恢复流程，
并按 §9 记录事件（涉及数据丢失时按 P0/P1 处理）。

### R-2 数据目录锁冲突（`.s3client.lock`，第二个实例启动失败）

**症状**：第二个实例启动即失败，日志 `msg="data dir lock"` + `err="data dir <路径> is already in use by another s3client instance: ..."`，
进程退出码 1；容器部署下表现为新副本反复重启。

**判据**：
1. 上述日志（unix 平台由 `flock(LOCK_EX|LOCK_NB)` 触发，**立刻失败**而非等待）；
2. `ls -l "$S3C_DATA_DIR/.s3client.lock"` 存在（unix；非 unix 平台**不创建也不检查**该文件）；
3. 同一 `S3C_DATA_DIR` 上确有另一个进程在运行（`docker compose ps`、宿主进程列表）。

**处置步骤**：
1. 找出并停掉多余实例。注意 compose 固定了 `container_name: s3client-server`，`--scale server=2` 本身也会因
   容器名冲突失败。
2. **不要通过删除锁文件来「解锁」**：锁由内核持有，与文件是否存在无关；删除文件不会释放锁。
3. **残留锁文件无害**：进程退出（含 panic / `SIGKILL`）时内核自动释放，新实例重新 `flock` 即成功，无需清理。
4. 非 unix 平台是**文档化 no-op**（`apps/server/internal/store/lock_other.go`）：Windows 上两个实例共享同一
   数据目录**不会**被拦住，会静默互相覆盖写入、重复执行迁移任务——单副本约束只能靠部署方式保证
   （[`ROADMAP.md`](ROADMAP.md) §五 5.1 R4）。

**何时升级**：发现「同一目录被两个实例写入」的迹象（账号配置回退 / 丢失、异步任务重复）→ 立即只保留一个实例，
按 §6 从备份恢复，并按 §9 记录（数据一致性事件）。

### R-3 数据卷写满 / 权限异常（目录 0700 / 文件 0600，运行用户 `app`）

**症状**：
- **启动期**：`msg="init store"` + `err`，退出码 1（读不到 / 建不了库 / 写不了锁文件）；
- **运行期**：写操作返回 **500**（如 `{"error":"failed to create account"}`），日志 `msg="handler error"` + `err`
  含 `no space left on device` / `permission denied` / `database or disk is full` 之类；
- **关键差异**：`json` / `encrypted` 驱动下 `/api/health` 与 `s3c_store_up` **不会**变化（`Ping` 恒 nil），
  故障只能从 5xx 与 `handler error` 日志发现。

**判据**：
1. `s3c_http_responses_total{class="5xx"}` 增速上升，日志出现 `msg="handler error"`；
2. 宿主 `df -h` / 容器 `docker system df` 显示卷满；
3. 目录 / 文件权限不是 0700 / 0600，或属主不是运行用户（容器为 `app`，uid 1000）。

> 写失败具备**内存回滚**语义：`fileStore` 写盘失败会回滚内存状态，因此不会出现「界面提示成功、磁盘却没有」；
> 反过来也意味着**失败的写不会留下半份状态**（判断影响面时以「最后一次成功写盘」为准）。

**处置步骤**：
1. 先判断影响面：读操作（列桶 / 列对象 / 下载）通常仍可用，写操作（建 / 改 / 删账号、改桶设置）已失败。
2. 释放空间：清理无用镜像与构建缓存；日志已由 compose 的 json-file 轮转限制（`max-size: 10m`，`max-file: 3` 或 `5`）。
3. 修正权限：`chmod 700 "$S3C_DATA_DIR"`、`chmod 600` 账号库文件与锁文件、属主改为运行用户
   （容器 `chown 1000:1000`）。注意 0700 / 0600 是**文档约定的加固**，`chmod` 失败会被忽略、不影响启动。
4. 恢复写入后，逐账号跑一次连通性测试（§6.3 第 2 步）确认配置未损坏。
5. 若已经出现账号配置丢失 / 回退（写失败前的最后一次成功写盘才是磁盘状态）→ 走 §6 恢复。

**何时升级**：卷在短时间内再次写满（说明有持续增长源：日志、`jobs.json`、外部写入）；或出现配置丢失 →
按 P1/P0 处理。

### R-4 存储密钥缺失或错误导致的启动失败（fail-closed 清单）

**症状**：进程**拒绝启动**（退出码 1），容器反复重启；日志首行即错误。完整 fail-closed 清单见
[`CONFIGURATION.md`](CONFIGURATION.md) §3，下表是运维最常撞到的几类与处置：

| 启动日志特征 | 原因 | 处置 |
|---|---|---|
| `msg="配置校验失败"` + `S3C_STORE_KEY too short` | `S3C_STORE_KEY` 非空但 < 16 字符 | 用 `openssl rand -hex 32` 重新生成；**换 key 后旧库读不出**，见下方警告 |
| `msg="配置校验失败"` + `plaintext account store not allowed` | `json` / `sqlite` 且 `S3C_STORE_KEY` 为空且未开 `S3C_ALLOW_PLAINTEXT_STORE` | 设置 `S3C_STORE_KEY`；仅本地联调才显式开明文（生产绝不） |
| `msg="init store"` + `S3C_STORE_KEY is required for encrypted store` | `S3C_STORE_DRIVER=encrypted` 但未提供 key | 补上与建库时**一致**的 key（密钥库 / `.env`） |
| `msg="init store"` + `decrypt accounts` / `encrypted account file` | key 与库不匹配，或库文件被截断 / 损坏 | **不要反复重启试探**（不会自愈）；确认挂载的是正确卷 / 文件，用正确 key 启动；否则走 §6 恢复 |
| `msg="配置校验失败"` + `unknown store driver` | `S3C_STORE_DRIVER` 不在 `json` / `sqlite` / `encrypted` 内 | 改正取值（大小写与空白会先归一化再比较） |
| `msg="配置校验失败"` + `unreadable explicit env file` | 显式 `S3C_ENV_FILE` 不存在 / 不可读 | 修路径与权限；该路径**不会**静默回退默认值（防止 `S3C_DATA_DIR` 静默丢失导致账号列表「凭空清空」） |
| `msg="配置校验失败"` + `S3C_TOKEN required for non-loopback listen address` | 非回环监听未设 `S3C_TOKEN` | 设置 token（≥ 16 字符） |
| `level=WARN` + `S3C_STORE_KEY 为空：… secretKey 将明文落盘` | 显式开了 `S3C_ALLOW_PLAINTEXT_STORE=1` | **生产不应出现**：改为 `encrypted` 或设置 `S3C_STORE_KEY` |
| 运行期 `encrypted secret_key found but S3C_STORE_KEY is not set`（`sqlite`） | 库中该列是密文，但本次进程未配 key（且显式放行了明文） | 补回 key；否则列表可见但该账号的密钥取不出、不可用 |

> ⚠️ **`S3C_STORE_KEY` 不是可随意重置的口令**：`encrypted` 驱动是整文件加密（`accounts.json.enc`），
> `json` / `sqlite` 配 key 时也加密敏感列。**用错 key 或丢 key = 账号库永久不可读**，且 API 不回传 `secretKey`
> （`AccountView` 只有 `secretSet`），无法从运行中的服务反推明文 → 等价于全部账号配置丢失。
> 仓库**未提供**在线换 key / 导出工具：轮换 `S3C_STORE_KEY` 等同于重新录入全部账号凭据（**建议**：把 key
> 当作长期密钥管理，轮换前先做恢复演练）。**确需轮换时按 [§6.5 的 Runbook](#65-轮换-s3c_store_keyrunbook先备份再逐账号重录) 执行**
> （先备份 → 逐账号重录 → 验证 → 可整体回滚）。

**处置步骤**：
1. 不要把「能启动」当作目标：先确定**哪个 key 与当前库匹配**（
   `encrypted` 用建库时的 key；`json` 允许读历史明文库，但也可能已是密文）。
2. 检查环境来源是否被改错：`.env` 查找顺序为 `S3C_ENV_FILE` → 进程 CWD `.env` → 可执行文件同目录 `.env`，
   **真实环境变量优先于文件**（[`CONFIGURATION.md`](CONFIGURATION.md) §1）。容器部署确认 compose / `.env` 注入值。
3. 确认挂载：容器 `/data` 是否指向了含目标库文件的那个卷（空目录会让 `encrypted` 生成**新盐新库**，
   误判为「数据丢失」）。
4. 全部不匹配 → 按 §6 用备份 + 对应 key 恢复。

**何时升级**：无法确定正确 key、或怀疑卷被替换 / 数据被误删 → 立即按 §9 记为数据事件，暂停后续写操作。

### R-5 流式传输中断（`s3c_stream_interrupted_total`）

**症状**：大文件下载、ZIP 打包下载、跨账号迁移在传输中途中断；客户端报网络错误或得到截断文件；
日志出现 `level=WARN msg="stream interrupted"`（含 `bucket` / `key` / `bytes` / `err`）。

**判据**：
1. `increase(s3c_stream_interrupted_total[15m]) > 0`；
2. `s3c_s3_stream_bytes_total` 在该时段有增长，但对象未完整送达（可估算已读字节）；
3. 日志 `msg="stream interrupted"` 的 `err` 区分三类根因：上游读失败 / 写超时 / 其它 IO 错误。
   **客户端主动断开不计入该指标**（只写 Debug 日志 `msg="stream cancelled by client"`）。

**处置步骤**：
1. 按 `err` 分诊：上游读失败 → 查 `s3c_s3_call_errors_total`（转 R-6）；写超时 → 客户端长时间无读取，
   超过 **5 分钟滚动空闲写超时**（每次成功写出后刷新，慢网大文件不会被绝对截止误杀）即中断；
2. 检查是否撞到并发上限：全局并发流式请求 **32**，饱和时新请求直接 503 `too many concurrent streaming requests`；
3. 经 nginx 时确认流式配置未被改小：`proxy_buffering off` + `proxy_read_timeout 3600s`（缺一即可能截断）；
4. 客户端取消下载属正常路径，不必处理；若指标不动而用户仍报失败，转前端 / 网络排查；
5. ZIP 场景额外看 `s3c_zip_partial_failures_total` / `s3c_zip_failed_keys_total`（部分失败）与
   `s3c_zip_failed_total`（整体失败）。

**何时升级**：同一对象可稳定复现中断 → 上游 S3 或网络问题，升级到存储侧；伴随大面积 5xx 时按 P1 处理。

### R-6 S3 上游错误率上升（`s3c_s3_call_errors_total{code=...}`）

**症状**：列桶 / 列对象 / 下载 / 复制 / 迁移失败，返回 4xx / 5xx（文案由 [`errors.md`](errors.md) 映射）；
`s3c_s3_call_errors_total{code=...}` 增长，`s3c_s3_calls_total` 同步增长（分母）。

**判据与按码分诊**：

| `code` | 含义 | 处置 |
|---|---|---|
| `SignatureDoesNotMatch` / `InvalidAccessKeyId` / `AccessDenied` | 凭据或权限漂移 | 用 `POST /api/accounts/{id}/test` 复测该账号；更新 accessKey / secretKey 或桶策略 |
| `NoSuchBucket` / `NoSuchKey` / `NoSuchVersion` / `NotFound` | 目标不存在 | 多为操作面问题：核对桶名 / 对象 key / 版本 id（`NoSuchBucket` 与 `NoSuchKey` / `NotFound` / `NoSuchVersion` 映射 404） |
| `NoSuchUpload` | 分段上传会话不存在（已被 abort / 过期） | 消息单独映射为 `multipart upload not found`，HTTP 走 fallback 500；让前端重开一次分段上传 |
| `SlowDown` / `ServiceUnavailable` / `RequestTimeout` | 上游限流 / 过载 | 降并发、错峰重试；核对上游配额与健康（映射为 503 `storage temporarily unavailable`） |
| `transport` | 非 API 错误：连接 / DNS / 传输失败 | 查网络与端点可达性；注意 SSRF 策略（默认放行私网 / 回环，`S3C_SSRF_DENY_PRIVATE=1` 时拒绝），用 `s3c_ssrf_deny_private` 核对生效值 |
| `timeout` / `canceled` | 上下文超时 / 取消 | 客户端取消属正常；`timeout` 集中出现说明上游变慢或请求量超预期 |
| `other` | **不在白名单内的任意 `<Code>` 都收敛到这里**（防不可信端点撑爆标签基数） | 从业务响应文案（[`errors.md`](errors.md)）定位真实错误码；确属新型上游错误时再考虑扩白名单（代码改动） |

**处置步骤**：
1. 先看分母：`s3c_s3_calls_total` 是否同步暴涨（业务量变化 ≠ 故障）。
2. 单个账号失败 → 走账号配置 / 桶策略；多个账号同时失败 → 网络或上游故障。
3. 用 `curl -sS .../api/metrics | grep s3c_s3_call_errors_total` 取当前分类计数；自检
   `s3c_s3_call_duration_seconds_count == s3c_s3_calls_total` 且 `_bucket{le="+Inf"}` 与之一致。
4. 记录前后快照再重试，避免「重启后指标清零」丢失证据。

**何时升级**：`transport` / 限流类持续 > 15 分钟，或全账号同时失败 → 网络 / 上游事故（§9 P1）。

### R-7 分段上传缺 `ExposeHeader: ETag`（浏览器直传失败）

**症状**：浏览器直传**大文件**（≥ 100MB 走分段，段大小 10MB）在最后一步失败，界面提示
「分段上传完成但未读取到 ETag，请确认 Bucket CORS 暴露 ETag 响应头」；**服务端无相关错误日志**
（直传不经本服务，是浏览器 ↔ S3 上游的跨源请求）。

**判据**：
1. 同一个桶里**小文件**（< 100MB，单次 PUT）成功、大文件失败；
2. 浏览器 Network 面板中每段 PUT 都返回 2xx，但响应头里**没有** `ETag`（被 CORS 规则过滤掉）；
3. 已上传分段被 `multipartAbort` 清理：不留半成品、不产出损坏对象（失败是显式的，不会静默组装）。

**处置步骤**：
1. 给该 Bucket 的 CORS 规则加上 `ExposeHeaders: ["ETag"]`——界面走「桶 CORS」，
   API 走 `PUT /api/accounts/{id}/bucket/cors`（body 的 `rules[].exposeHeaders`）。`rules` 传空数组会**删除全部规则**，改完先 `GET` 确认。
2. 复核 `allowedOrigins` 含实际页面 Origin（如 `http://127.0.0.1:8080`），`allowedMethods` 含 `PUT`。
3. 重试上传（前一次的残留分段已被 abort）。
4. 参考各厂商矩阵（[`../README.md`](../README.md)）：RustFS / MinIO / AWS S3 / 阿里 OSS / 腾讯 COS
   **都要求在 CORS 暴露 `ETag`**，其中只有 RustFS 有自动化真对端 E2E。

**何时升级**：确认已暴露 `ETag` 仍读不到 → 上游实现不返回该头（改用单次 PUT 或更换实现），
按 [`ROADMAP.md`](ROADMAP.md) §五 5.2 依赖清单 E8（「因厂商而异」）登记。

### R-8 异步任务 / SSE 挂起

**症状**：前端进度条长时间不动；SSE 长时间没有 `ping` 帧；`GET /api/migrate/jobs/{id}` 状态长期 `running`；
新的批量操作报 503。

**判据**（按可能性排序）：
1. **SSE 断流 ≠ 任务停止**：浏览器断开后任务仍在服务端跑；此时 `GET /api/migrate/jobs` 仍能看到它；
2. 单任务订阅打满：每任务最多 **16** 个订阅，超出返回 503 `too many subscribers for this job`；
3. 在册任务打满：未终结任务上限 **256**，超出返回 503 `too many running jobs; retry later`；
4. 全局流式槽位打满：SSE 与代理下载 / ZIP 共用 **32** 个并发流槽位，饱和时 503
   `too many concurrent streaming requests`；
5. 心跳缺失：正常时 SSE 每 **15 秒**推一次 `event: ping`（`data: {"ok":true}`），长时间没有即连接已死。

**处置步骤**：
1. 列出在册任务与状态：`GET /api/migrate/jobs`（含跨重启恢复的 `interrupted` 项）。
2. 前端刷新 / 重新订阅即可；SSE 会自动补发当前进度（订阅时会先推一帧）。
3. 确需终止：`POST /api/migrate/jobs/{id}/cancel`。
4. 任务上下文带 **2 小时**超时（`service.JobTimeout`）；到期后批量循环会在下一个检查点停止。
5. 生命周期（用于判断「任务怎么不见了」）：已终结任务保留 **30 分钟**；`interrupted` 任务保留 **7 天**
   （它是「复制成功但源未删除」这类**待人工对账**的证据）；回收循环每 5 分钟跑一次。
6. 进程重启会把所有 `running` 任务标记为 `interrupted`（`jobs.json` 落盘）——**重启不是取消**，
   重启后请到任务列表确认有无需要人工对账的半成品移动。
7. 关停（`SIGTERM`）会先取消全部运行中任务，再等待在途连接（R-9）。

**何时升级**：任务长期 `running` 且 `cancel` 无效；或出现待对账项 → 人工核对源 / 目标后按 §9 记录。

### R-9 优雅关停超时（`S3C_SHUTDOWN_TIMEOUT`）

**症状**：`SIGTERM` 后超过预期时间进程仍未退出；容器在宽限期后被 `SIGKILL`（事件里是 exit 137）；
在途大文件下载被硬切断。

**判据**：
1. 日志 `msg="shutting down..."` 与 `msg="shutdown complete"` 的**时间差** ≈ 关停耗时（超时也会打
   `shutdown complete`——退出码仍为 0，**只能靠日志时间差判断超时**）；
   同一次耗时也落盘为 `s3c_last_shutdown_duration_seconds`（§3.2）：**本次进程的值要等下次启动才可见**
   （进程退出前监听已关闭，scrape 赶不上），可用来做「历次关停趋势」而非实时判断；
2. `S3C_SHUTDOWN_TIMEOUT` 取值必须在 **1–3600** 秒（超上界拒绝启动，防止溢出把关停窗口静默清零）；
3. compose 的 `stop_grace_period: 30s` 与 `S3C_SHUTDOWN_TIMEOUT` 默认都是 30——**若把超时调到
   `stop_grace_period` 之上，docker 会在宽限期结束后直接 `SIGKILL`**，优雅关停形同虚设。

**处置步骤**：
1. 让两者对齐：`stop_grace_period` ≥ `S3C_SHUTDOWN_TIMEOUT`（**建议**，compose 默认两侧都是 30s）。
2. 找出迟迟不退出的请求：关停期间先是「取消异步任务」，再等活跃连接；长连接主要是大文件下载 / ZIP /
   迁移 SSE（空闲写超时 5 分钟滚动）。
3. 退出码语义：正常优雅关停为 0，运行期服务错误（如端口占用）为 1。
4. nginx 用 `SIGQUIT` 优雅停止（compose `stop_signal: SIGQUIT`）。
5. 关停是**有损的**：进行中的迁移 / 复制被打断，重启后标记 `interrupted`（R-8 第 6 步）。

**何时升级**：每次关停都超时（说明有请求长期不结束）→ 先按 R-5 / R-8 定位长连接来源；关停期间出现数据
不一致迹象 → 按 §6 校验账号库，并核对 `jobs.json` 的对账项。

### R-10 nginx 配置 reload 与零停机

**症状**：改了 nginx 配置需要生效；或 reload 后出现 502 / 连接被拒 / 流式下载被截断。

**判据**：
1. 仓库自带 nginx 配置为**单 worker**（`worker_processes 1`，`worker_connections 1024`），与单后端一对一，
   reload 时新 worker 接管、旧 worker 处理完在途请求后退出；
2. compose 中 nginx 发布 `127.0.0.1:8080:8080`，`stop_signal: SIGQUIT`，`stop_grace_period: 30s`；
3. 若出现 502：先看 nginx 错误日志与 `nginx -t` 输出，再看后端 `/api/health`（后端才是根因时按 R-1 / R-3）。

**处置步骤**：
1. **改完先校验**：`docker compose exec nginx nginx -t`（**建议**）。配置非法时 reload 不会替换 worker，
   旧 worker 继续按旧配置服务，不会中断连接。
2. 执行热加载：
   ```bash
   # 容器（生产推荐直接 exec；`make restart-nginx` 的容器分支检测的是 base compose 的 nginx）
   docker compose exec nginx nginx -s reload
   # 本机开发（含 PID 复用防护：PID 不是预期 nginx 进程时只告警、不 kill）
   make restart-nginx
   ```
3. 变更后验证：
   ```bash
   curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/api/health   # 期望 200
   ```
   并抽查一条流式路径（大文件下载 / ZIP）未被截断——这依赖 `proxy_buffering off` 与
   `proxy_read_timeout 3600s` 没被改小。
4. TLS 叠加部署（[`DEPLOYMENT.md`](DEPLOYMENT.md) §2.3）改了证书 / 域名路径时，先确认挂载的配置文件与证书
   可读，再 reload。
5. **零停机的边界**：nginx reload 本身不断连接；但**后端重启不是零停机**——单实例、无多副本可滚动，
   在途请求最多等 `S3C_SHUTDOWN_TIMEOUT`（R-9）。

**何时升级**：reload 后 502 持续 → 用上一版可用配置回滚（保留一份已知可用的配置备份），再查容器日志与
`nginx -t` 输出；若是后端不可用导致的 502，转 R-1 / R-3。

## 6. 备份与恢复

备份对象是**账号配置**（对象数据在 S3 上游，不在本机）。备份频率、保留份数、异地存放策略属**建议值**：
账号库每次变更都会原子落盘，但**仓库未提供服务端导出 / 定时备份能力**，需要外部 cron / 卷快照 / 宿主机备份实现。

### 6.1 需要备份的文件

| 文件（位于 `S3C_DATA_DIR`，容器内 `/data`） | 何时存在 | 是否备份 | 说明 |
|---|---|---|---|
| `accounts.json` | `S3C_STORE_DRIVER=json`（默认） | ✅ 必须 | 账号库（明文或 S3C3 密文，取决于是否配 key） |
| `accounts.json.enc` | `S3C_STORE_DRIVER=encrypted` | ✅ 必须 | 整文件加密账号库；**必须与建库时的 `S3C_STORE_KEY` 配对使用** |
| `accounts.db` | `S3C_STORE_DRIVER=sqlite` | ✅ 必须 | SQLite 主库 |
| `accounts.db-wal` / `accounts.db-shm` | `sqlite`（WAL 模式） | ✅ 必须（与主库**同批**） | 侧车含尚未 checkpoint 的页；只拷 `.db` 可能丢掉最近写入 |
| `jobs.json` | 有过异步任务时 | ⚠️ 可选 | 异步任务清单（含 `interrupted` 对账证据，保留 7 天）；丢失只影响重启后的对账视图，不影响账号数据 |
| `schedules.json` | 建过计划任务时 | ⚠️ 可选 | 计划任务清单（cron / 桶引用 / 排期，0600）；丢失只丢计划本身（可在界面重建），不影响已完成的备份数据 |
| `shutdown.json` | 至少完成过一次优雅关停 | ⚠️ 可选 | 上次关停耗时（`{"durationUs":...}`），供 `s3c_last_shutdown_duration_seconds` 在下次启动读入；丢失只让该指标回落 0，不影响服务 |
| `.s3client.lock` | unix 且进程运行过 | ❌ **不需要** | `flock` 锁文件，内容无意义；锁由内核在进程退出时释放，残留文件不影响下次启动 |
| `accounts.json.tmp` 等 `*.tmp` | 写盘中途崩溃时可能残留 | ❌ 不需要 | 原子写临时残骸；下次写盘会先清理，不影响读取 |

**另需单独保管**：`.env`（或部署平台的环境变量）里的 `S3C_TOKEN` 与 `S3C_STORE_KEY`——它们**不在数据目录内**，
不会被文件级备份覆盖。

> **一致性（建议）**：`sqlite` 驱动请在**停止进程后**整目录打包；热备请用 SQLite 自身的备份机制
> （`.backup` / `VACUUM INTO`，需宿主具备 `sqlite3` 工具）或卷快照，不要只 `cp accounts.db`。`json` / `encrypted`
> 是整文件原子写（临时文件 + fsync + rename + 目录 fsync），单文件拷贝不易读到半成品，但仍建议停进程或快照。

### 6.2 `S3C_STORE_KEY` 必须与备份分开保存（且两者缺一不可）

> ⚠️ **这是本手册最重要的一条数据安全约束**：`S3C_STORE_KEY` 是账号密钥落盘加密的**唯一凭据**（AES-256-GCM +
> Argon2id，参数随文件头保存）。它既不能从密文反推，也不能从运行中的 API 取回——`AccountView` **不回传
> `secretKey`**（只有 `secretSet` 布尔）。

| 情形 | 后果 |
|---|---|
| 只有密文备份，**丢了 key** | 账号库**永久不可读**（`encrypted` 启动即 `msg="init store"` + 解密失败；`json` / `sqlite` 配 key 时同理）。等价于**全部账号配置丢失**，且无法从服务反推 |
| 只有 key，**丢了备份** | 账号配置丢失，只能重新录入全部账号的 accessKey / secretKey |
| key 与备份**放在同一份介质**（同一目录 / 同一个备份包 / 同一个仓库） | 加密失去意义：拿到备份即等于拿到明文等价权限——**违背了「密钥与数据分离」的初衷** |

**落地建议（未在代码中强制）**：

1. 把 `S3C_STORE_KEY` 存进密码管理器 / KMS / Secret Manager，与备份介质**物理或权限分离**；
2. 备份介质上只放密文文件（及可选的 `jobs.json`），不放 `.env`；
3. 至少两人 / 两处可取得 key（避免单点丢失），但每次取用留痕；
4. **恢复演练必须验证「备份 + key」配对可用**——只验证备份文件存在不算验证；
5. 注意 `json` / `sqlite` 驱动在 `S3C_ALLOW_PLAINTEXT_STORE=1` 下可能是**明文库**：此时备份文件本身
   就是敏感数据，保护要求等同 key。

### 6.3 恢复步骤

1. **停服并确认进程退出**（锁才会释放）：
   ```bash
   make stop                                                  # 本机
   docker compose -f docker-compose.prod.yml stop server       # 容器（stop_grace_period 内等它退干净）
   ```
2. **保留现场**：把现有数据目录文件改名留档（如 `accounts.json.bak-<时间戳>`），不要直接覆盖——
   若恢复目标是错的，现场是唯一的证据。
3. **放入备份**：把备份的账号库文件（及 `sqlite` 的三个文件、可选的 `jobs.json`）放回 `S3C_DATA_DIR`。
4. **修权限与属主**：目录 `0700`、文件 `0600`、属主为运行用户（容器 `chown -R 1000:1000`，用户 `app`）。
5. **用与备份时一致的配置启动**：
   - `S3C_STORE_DRIVER` 必须与备份的驱动一致（`json` → `accounts.json`；`encrypted` → `accounts.json.enc`；
     `sqlite` → `accounts.db`）；驱动不匹配会打开**另一个文件**（甚至新建空库），表现为「账号全没了」；
   - `S3C_STORE_KEY` 必须是**建库时的那一个**。
6. **启动并观察启动日志**：应看到 `msg="s3client server"`（无 `配置校验失败` / `init store` 错误）。

### 6.4 恢复后验证

1. 健康与版本：
   ```bash
   curl -sS http://127.0.0.1:8080/api/health          # 200 且 store.ok=true
   ```
2. 账号清单（数量与 `secretSet` 是否为 `true`）：
   ```bash
   curl -sS -H "Authorization: Bearer $S3C_TOKEN" http://127.0.0.1:8080/api/accounts
   # 需要计数时装了 jq 可用：| jq '.accounts | length'
   ```
3. **逐个账号做真实连通性测试**（会真的用 `secretKey` 连 S3，是「密钥解密正确」的最强证据）：
   ```bash
   curl -sS -X POST -H "Authorization: Bearer $S3C_TOKEN" http://127.0.0.1:8080/api/accounts/<id>/test
   # 期望 {"ok":true,...}；ok:false 时响应体内带 error 文案（HTTP 仍 200）
   ```
4. 核对 `s3c_build_info{version="..."}` 与实际部署版本一致（前后端版本匹配）。
5. 若恢复了 `jobs.json`：查 `GET /api/migrate/jobs`，确认有无 `interrupted` 任务需要人工对账
   （典型：复制成功但源未删除）。
6. 观察 15 分钟：`s3c_http_responses_total{class="5xx"}` 无新增、`s3c_store_up 1`、无 `handler error` 日志。

### 6.5 轮换 `S3C_STORE_KEY`（Runbook：先备份，再逐账号重录）

> **结论先行**：本仓库**没有**在线换 key / 重加密工具。`S3C_STORE_KEY` 更换后，用旧 key 加密的账号库
> **无法解密**；运行中的服务也**不回传 `secretKey`**（`AccountView` 只有 `secretSet`），因此**无法**把明文
> 导出来再写回。唯一的换 key 路径是：**先从你自有的密钥来源取回每个账号的 accessKey / secretKey，
> 再用新 key 重建账号库并逐个重录**。轮换 = 重新录入全部账号，请把它当成一次**计划内的数据迁移**，
> 而不是改一个环境变量后重启。

**适用 / 不适用**：

| 情形 | 是否走本 Runbook |
|---|---|
| key 疑似泄露、合规要求定期轮换 | ✅ 按本节执行 |
| 从明文库（`S3C_ALLOW_PLAINTEXT_STORE=1`）首次切到加密库 | ✅ 属于「首次配 key」，旧库是明文、无需解密，直接重录 |
| key 配错 / 丢失，目标是**找回现有数据** | ❌ 不是轮换，是故障：走 R-4 + §6.3 用**正确**的 key 恢复 |

**前置条件（缺一不可）**：

1. 新的 `S3C_STORE_KEY`（≥ 16 字符，建议 `openssl rand -hex 32`）已存入密码管理器 / KMS，且与备份介质分离（§6.2）；
2. 全部待迁移账号的 accessKey / secretKey 可从独立来源取回——这是本流程的真正瓶颈，**先确认再动手**；
3. 已按 §6.1 完成当前账号库备份，并按 §6.4 验证过「备份 + 旧 key」配对可用；
4. 已确认停机窗口：轮换期间全部账号不可用（单实例、无 HA，§7.2）。

**步骤**：

1. **冻结写入并留档**（在旧 key 仍生效时完成）：
   ```bash
   make stop                                                          # 容器：docker compose -f docker-compose.prod.yml stop server
   ls -l "${S3C_DATA_DIR:-./data}"                                    # 记录文件清单与时间戳
   cp -a "${S3C_DATA_DIR:-./data}" "${S3C_DATA_DIR:-./data}.pre-rotate-$(date +%Y%m%d%H%M%S)"
   ```
2. **用旧 key 启动一次，抄下非敏感字段**（`endpoint` / `region` / `accessKey` / `bucket` / `pathStyle` /
   `useSSL` / `publicEndpoint`）作为重录对照——key 换掉后旧库就再也读不出来了：
   ```bash
   curl -sS -H "Authorization: Bearer $S3C_TOKEN" http://127.0.0.1:8080/api/accounts
   ```
   > `secretKey` **不会**出现在响应里；它必须来自你的密钥来源（步骤 2 解决的是「别漏账号、别抄错字段」）。
3. **换 key 并重录**（按账号量选一种）：
   - **就地重建**：把旧账号库改名留档（如 `accounts.json.enc.old-key`），写入新的 `S3C_STORE_KEY`，启动
     （`encrypted` 在空目录会新建库），再用前端 / `POST /api/accounts` 逐个重录；
   - **新目录对拷**：把数据目录切到一个新目录，写入新 key 后重录，确认无误再切换挂载。
4. **逐账号验证**（同 §6.4 第 3 步的最强证据——密钥真能解密并连上 S3）：
   ```bash
   curl -sS -X POST -H "Authorization: Bearer $S3C_TOKEN" http://127.0.0.1:8080/api/accounts/<id>/test
   ```
5. **确认完成**：`/api/health` 200 且 `store.ok=true`；`s3c_store_up 1`（`sqlite` 才有实时意义，§3.1）；
   `GET /api/accounts` 账号数与轮换前一致且每条 `secretSet=true`；每个账号 `test` 返回 `ok:true`；
   观察 15 分钟无 `msg="handler error"`、`s3c_http_responses_total{class="5xx"}` 无新增。

**回滚**（任一步失败，或发现密钥来源不全）：

1. `make stop`；
2. 把步骤 1 的 `.pre-rotate-<时间戳>` 目录恢复回 `S3C_DATA_DIR`（目录 0700 / 文件 0600 / 属主运行用户）；
3. 把 `S3C_STORE_KEY` 改回**旧 key** 后启动；
4. 按 §6.4 复核账号数与连通性。

> ⚠️ 旧 key 在新库上**不可用**、旧库在新 key 下也**不可读**——两者只能整体配对使用。一旦新库已经写入
> （哪怕只重录了一个账号），回滚就会丢掉这些改动；因此备份与回滚窗口必须在**停机状态**下完成。

**何时升级**：某个账号的 secretKey 已无法取回（等于该账号不可恢复，需要在对象存储侧重建凭证）→
按 §9 记为数据事件；轮换期间出现 5xx 或账号缺失 → **立即回滚**，不要边修边写。

## 7. 灾难恢复

> ⚠️ **本节为建议基线，尚未在代码或 CI 中强制**——按实际部署调整后再落地。
> RTO / RPO 的具体数值取决于**你的**备份频率与人员响应速度，下面的数字只是规划起点。

### 7.1 场景与建议 RTO / RPO

| 场景 | 恢复动作 | 建议 RTO（**未强制**） | 建议 RPO（**未强制**） |
|---|---|---|---|
| 进程 / 容器崩溃，数据卷完好 | 重新启动（compose `restart: unless-stopped` 会自动重启） | < 5 分钟 | **0**（每次变更原子落盘：临时文件 + fsync + rename + 目录 fsync） |
| 数据卷损坏 / 误删，有备份 | §6.3 恢复流程 | 30–60 分钟 | = 上次备份时间点（无 binlog / 无复制，RPO 直接等于备份间隔） |
| 宿主机 / 节点整体丢失 | 重建环境（[`DEPLOYMENT.md`](DEPLOYMENT.md)）+ §6.3 恢复 + 从密钥库取回 `S3C_STORE_KEY` / `S3C_TOKEN` | 数小时（取决于环境重建） | = 上次异地备份时间点 |
| `S3C_STORE_KEY` 丢失且无独立保管 | **不可恢复**：密文无法解密，账号库等价全丢 | 不适用 | 数据等价全丢 |
| 上游 S3 不可用 | 本服务无法代为恢复；读 / 写操作大面积失败（R-6） | 取决于上游 SLA | 不适用（数据在上游） |

### 7.2 为什么没有 HA 路径（单实例约束）

| 约束 | 依据 |
|---|---|
| 账号存储是**本地文件**（`json` / `sqlite` / `encrypted`），启动时对 `S3C_DATA_DIR` 加 `flock` 单写者锁，第二个实例**立即失败** | [`DEPLOYMENT.md`](DEPLOYMENT.md) §2.2、`store/lock.go` |
| 异步任务表在**内存**中（`JobRegistry`，落盘仅用于跨重启对账） | `service/job.go` |
| **单 token 模型**：没有租约 / 选主机制 | `config.go`、[`threat-model.md`](threat-model.md) 边界 A |
| 存储不可用即**硬失败不降级**（503 / 退出），没有「部分可用」过渡态 | [ADR-002](decisions/0002-store-fail-closed.md) |
| 非 unix 平台锁是 **no-op**，连「误起第二个实例」都拦不住 | `store/lock_other.go`、[`ROADMAP.md`](ROADMAP.md) §五 5.1 R4 |
| 多副本 / HA（store 外置）仍是**未排期的候选评估项** | [`ROADMAP.md`](ROADMAP.md) §三 3.2 #15（⬜） |

**结论**：单点故障，升级与恢复都必须停机；RTO 主要由「发现时间 + 人工介入速度」决定，而不是技术冗余。

### 7.3 演练建议（**建议值**）

每季度至少演练一次，并记录**实测** RTO / RPO 回填 §7.1：

1. 从备份 + 独立保管的 key 在**干净环境**里完成一次 §6.3 全流程；
2. 跑完 §6.4 的 6 项验证（尤其第 3 步的真实连通性测试）；
3. 故意不提供 `S3C_STORE_KEY`（且不开 `S3C_ALLOW_PLAINTEXT_STORE`），确认失败行为符合预期（应拒绝启动；
   注意区分 `encrypted` 驱动在**空目录**下会新建空库，见 R-4 第 3 步）；
4. 复查 `.env` / 密钥库中的 `S3C_TOKEN` 是否与实际使用的一致。

## 8. 升级与回滚

操作步骤不在此重复，见 [`DEPLOYMENT.md`](DEPLOYMENT.md) §6.4（升级）与 §7（回滚）。运维侧只需要守住下面这张检查表：

| 阶段 | 动作 | 依据 |
|---|---|---|
| 升级前 | 按 §6 完成一次备份，并确认 key 可用；读 [`CHANGELOG.md`](../CHANGELOG.md) 的 `[Unreleased]` 段 | 本手册 §6；[`DEVELOPMENT.md`](DEVELOPMENT.md) §4 |
| 升级前 | 确认数据格式兼容性：加密格式 `S3C3`，**兼容读取旧 `S3C2` 库**（升级路径存在，但降级回旧版本不一定能读新格式） | [`threat-model.md`](threat-model.md) 边界 C |
| 升级中 | 停机窗口内替换镜像 / 二进制（prod compose 镜像 tag 由 `S3C_IMAGE_TAG` 控制）；容器宽限期见 R-9 | [`docker-compose.prod.yml`](../docker-compose.prod.yml) |
| 升级后 | `/api/health` 200；`s3c_build_info{version=...}` 与预期 tag 一致；抽查一次流式下载与一次写操作 | §3.1、§3.2 |
| 回滚 | `docker compose down` → 用旧 tag `up -d`；**回滚前先备份 `/data`**（旧版本可能读不了新格式）；`.s3client.lock` 不需要备份 | [`DEPLOYMENT.md`](DEPLOYMENT.md) §7 |
| 桌面端 | 产物未签名（Windows SmartScreen / macOS 未公证）、**无自动更新通道**，用户需手动升级 | [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #25、[`ROADMAP.md`](ROADMAP.md) §五 5.1 R5 |

## 9. 事故响应

### 9.1 分级（**建议基线，未强制**）

| 级别 | 判据 | 例子 | 建议响应 |
|---|---|---|---|
| **P0** | 数据不可恢复 / 疑似未授权访问 | 账号库无法解密；确认数据目录被未授权写入；token 泄露且已产生越权操作 | 立即止损（断外网 / 停写 / 吊销 token），保全证据，人工决策恢复方案 |
| **P1** | 服务整体不可用 | `/api/health` 持续 503；全部写操作 500；容器反复重启 | 30 分钟内响应，按 R-1 / R-3 / R-4 处置 |
| **P2** | 单账号 / 单桶 / 单功能受损 | 某账号凭据失效；某桶 CORS 导致大文件直传失败；ZIP 部分失败 | 当天响应，按 R-6 / R-7 处置 |
| **P3** | 性能下降 / 咨询 / 观测疑问 | 上游延迟升高但未失败；指标口径疑问 | 记录并按需安排 |

### 9.2 谁决策

- 本仓库是**自托管软件**，不定义值班轮换与升级路径——**值班表与升级链由部署方自定**（**建议**：至少指定一名
  可执行破坏性操作的人类负责人）。
- 破坏性 / 不可逆动作**必须人工确认**，包括：删除或覆盖数据目录与账号库、覆盖备份、轮换 `S3C_TOKEN` /
  `S3C_STORE_KEY`、回滚数据格式、清空桶 / 对象。仓库对 AI 代理的权限矩阵明确要求
  「删除文件或目录」需确认、「自动删除分支或数据」禁止、必须由人类执行（[`AI_POLICY.md`](AI_POLICY.md)）。
- `S3C_TOKEN` 泄露时的止损动作（代码强制语义）：`S3C_TOKEN` 支持**逗号分隔多值**，追加新值 → 逐个客户端切换 →
  删掉旧值即完成吊销；长度按**最短的那个** token 判定（多值轮换期间不要塞入短占位符）。

### 9.3 如何记录

1. **先取证再处置**：`/api/health` 响应体、`/api/metrics` 全文、最近 15 分钟日志、`ls -l` 数据目录、`jobs.json` 副本。
2. **时间线**：记录每个动作的本地时间与响应头里的 `X-Request-ID`（用于从日志精确定位请求）。
3. **审计事件**：检索 `msg="audit"` 的事件名（§3.3 清单）确认是否有越权 / 批量删除 / 限速命中。
4. **影响面**：哪些账号 / 桶 / 对象、是否有半成品（`GET /api/migrate/jobs` 的 `interrupted` 项）。
5. **处置与验证**：写下实际执行的动作与 §6.4 验证结果。
6. **复盘与归档**：把上面五步整理成一份完整复盘——格式与字段见
   [`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md)（元信息 / 影响面 / 时间线 / 取证 / 根因 / 行动项 /
   文档同步表），**首份已填写的实例**（可照抄结构）见
   [`archive/incident-20260916-presign-empty-url.md`](archive/incident-20260916-presign-empty-url.md)；
   填写完成后按 [`archive/index.md`](archive/index.md)「归档操作」`git mv` 冻结进
   `docs/archive/`（命名 `incident-YYYYMMDD-<短名>.md`）并在该索引登记一行。
   根因若是代码缺陷 → 仓库 issue，并按 [`DEVELOPMENT.md`](DEVELOPMENT.md) §4 与代码同 PR 同步
   [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) / [`CHANGELOG.md`](../CHANGELOG.md)。

### 9.4 与漏洞报告渠道的区分（不要混用）

| 事项 | 渠道 | 说明 |
|---|---|---|
| **安全漏洞**（越权、注入、加密 / 密钥处理缺陷、依赖 CVE、供应链） | [`.github/SECURITY.md`](../.github/SECURITY.md) 的**私有漏洞报告** | **不要开 public issue**；仓库未公开安全邮箱，GitHub 私有报告是唯一保证可达的渠道；承诺 48 小时内确认 |
| **运行故障 / 性能 / 兼容性 / 文档错误** | 仓库 issue（公开）或内部事故流程 | 按 §9.3 记录；涉及数据的事件先在内部闭环再决定是否公开 |
| **运维请求**（要权限、要资源、要变更窗口） | 部署方内部流程 | 不在本仓库范围 |

## 10. 容量与单实例约束

### 10.1 单实例约束

| 约束 | 表现 | 依据 |
|---|---|---|
| `flock` 单写者锁（`.s3client.lock`） | 同一 `S3C_DATA_DIR` 的第二个实例**启动即失败** | `store/lock.go`、R-2 |
| 非 unix 平台锁为 no-op | Windows 上不创建 / 不检查锁文件，两个实例可同时写同一目录 | `store/lock_other.go`、[`ROADMAP.md`](ROADMAP.md) §五 5.1 R4 |
| 异步任务表在内存 | 多副本无法共享任务状态；重启后 `running` → `interrupted` | `service/job.go` |
| 单 token，无租约 / 选主 | 无法安全地让两个实例同时对外服务 | `config.go` |
| 无滚动升级 / 无水平扩容 | 升级与恢复都需要停机窗口 | §7.2 |

**容量含义**：本服务**不存对象数据**（对象在 S3 上游），数据卷只承载账号配置（`accounts.json` /
`accounts.db` / `accounts.json.enc`）与 `jobs.json`，因此卷容量需求与「管理多少账号」相关，而与「存了多少对象」无关。

### 10.2 代码 / compose 强制的资源上限

| 约束 | 值 | 依据（**仓库内强制**） |
|---|---|---|
| 未终结异步任务 | **256**，超限 503 `too many running jobs; retry later` | `service/job.go` `defaultMaxJobs` |
| 单任务 SSE 订阅 | **16**，超限 503 `too many subscribers for this job` | `service/job.go` `maxSubscribersPerJob` |
| 并发流式请求（代理 / ZIP / 迁移 SSE） | **32**，饱和时 503 `too many concurrent streaming requests` | `handler/stream.go` `maxConcurrentStreams` |
| 流式写空闲超时 | **5 分钟**滚动（每次成功写出后刷新） | `handler/stream.go` `streamIdleTimeout` |
| 任务上下文超时 | **2 小时** | `service/job.go` `JobTimeout` |
| 任务保留 | 已终结 30 分钟；`interrupted` 7 天；回收每 5 分钟 | `service/job.go` `JobTTL` / `JobInterruptedTTL` |
| 请求体上限 | 16MB，超限 413 | `handler` `maxBody` |
| 单次 PUT / 复制对象 | 5GB（超出明确报错并提示用分段） | [`errors.md`](errors.md) `ErrObjectTooLarge` |
| 流式复制单对象 | 640GB（64MB × 10000 段），超限明确拒绝并 abort（不静默截断） | [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #63 |
| 批量 key 上限 | 删除 ≤1000 / 复制 ≤10000 / ZIP ≤1000 / 迁移 ≤10000 | [`threat-model.md`](threat-model.md) §4 |
| 限速 | IP 令牌桶 120/min | [`threat-model.md`](threat-model.md) 边界 A |
| 请求头读超时 / 读超时 / 空闲超时 | 15s / 60s / 120s（**不设 WriteTimeout**，避免截断大文件流） | `main.go` |
| 容器内存 | server **512M**，nginx **128M** | [`docker-compose.yml`](../docker-compose.yml)、[`docker-compose.prod.yml`](../docker-compose.prod.yml) |
| 容器日志 | json-file，`max-size 10m` × `max-file 3`（base）/ `5`（prod） | 同上 |
| nginx | 单 worker、`worker_connections 1024`、`client_max_body_size 5g` | `deploy/nginx/nginx.conf`、`deploy/nginx/conf.d/` |

### 10.3 容量规划与水位（**建议值，未在代码中强制**）

| 关注点 | 说明 | 建议做法 |
|---|---|---|
| 数据卷使用率 | 字节使用率已内置（`s3c_volume_size_bytes` / `s3c_volume_free_bytes`，2026-09-30）；**inode 仍需宿主采集** | 字节剩余 < 20% 告警（§4.2 S3ClientVolumeSpaceLow）；inode > 80% 告警（§4.3） |
| 账号库写放大 | `json` / `encrypted` **每次写盘重写整个文件**（全量快照 + 原子替换）；`sqlite` 为行级更新 | 账号数达到数百时优先 `sqlite`（**建议**，代码未设阈值） |
| `jobs.json` 增长 | 受 TTL 控制（30 分钟 / 7 天），但大批量任务期间会持续写入 | 无需清理；异常增长按 R-8 排查 |
| 日志占用 | 由 compose 的 json-file 轮转限制硬上限；本机 `.run/*.log` **不轮转** | 本机部署建议外部 `logrotate`（**建议**） |
| 内存 | 单块分段缓冲即 64MB（流式复制），容器上限 512M | 不要同时跑大量超大对象迁移；见 [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) #63 |
| 卷类型 | `sqlite` 依赖文件锁与 fsync 语义 | 使用本地块存储；NFS 等网络文件系统上的 `flock` / fsync 语义无法保证（**建议**） |

## 11. 相关文档

| 需要什么 | 看哪里 |
|---|---|
| 部署形态、compose、nginx、TLS、升级与回滚步骤 | [`DEPLOYMENT.md`](DEPLOYMENT.md) |
| 事故复盘的格式（取证 / 时间线 / 根因 / 行动项）与存档规则 | [`POSTMORTEM_TEMPLATE.md`](POSTMORTEM_TEMPLATE.md) · [`archive/index.md`](archive/index.md) |
| 配置项 SSOT（全部 `S3C_*` 环境变量、fail-closed 清单） | [`CONFIGURATION.md`](CONFIGURATION.md) |
| 接口与请求 / 响应字段（含健康检查、指标、桶 CORS） | [`api.md`](api.md) |
| S3 / API 错误码与用户文案对照 | [`errors.md`](errors.md) |
| 本地热路径性能基线、`make bench` 与性能回归门禁 | [`PERFORMANCE.md`](PERFORMANCE.md) §4 |
| 安全边界、默认值、已接受的风险 | [`threat-model.md`](threat-model.md) |
| 架构与关键决策（含存储硬失败、SSRF 取舍） | [`architecture.md`](architecture.md) · [`decisions/index.md`](decisions/index.md) · [ADR-002](decisions/0002-store-fail-closed.md) |
| 未闭环缺陷 / 外部阻塞 / 技术债 | [`KNOWN_ISSUES.md`](KNOWN_ISSUES.md) |
| 版本规划与长期方向（多副本 / HA、OTel、桌面签名） | [`ROADMAP.md`](ROADMAP.md) |
| 已实现能力台账与验证证据 | [`FEATURES.md`](FEATURES.md) |
| 开发规范、测试分层、文档同步门禁、文档命名约定 | [`DEVELOPMENT.md`](DEVELOPMENT.md) |
| 漏洞披露流程 | [`../.github/SECURITY.md`](../.github/SECURITY.md) |
| 贡献流程与环境搭建 | [`../.github/CONTRIBUTING.md`](../.github/CONTRIBUTING.md) |
| 代理权限边界 / 破坏性操作的人工确认要求 | [`AI_POLICY.md`](AI_POLICY.md) |
| nginx 反向代理运维（reload / 优雅停止） | [`../deploy/nginx/README.md`](../deploy/nginx/README.md) |
| 本地开发进程管理与健康检查脚本 | [`../Makefile`](../Makefile) · [`../scripts/graceful-restart.sh`](../scripts/graceful-restart.sh) |
| 发版历史 | [`../CHANGELOG.md`](../CHANGELOG.md) |
