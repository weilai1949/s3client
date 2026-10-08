# ADR-0013：零第三方依赖自研 OTLP tracing（W3C traceparent + OTLP/HTTP JSON，默认关闭）

## Status

Accepted（已采纳）

## Date

2026-10-08

## Context

ROADMAP §三 3.2 #11 要求把 `X-Request-ID` 升级为可跨 `presign` / `proxy` / `migrate` 关联的
trace。现状是：

- 已有 Prometheus 指标（`/api/metrics`，见 [`../architecture.md`](../architecture.md) §2）与
  `X-Request-ID` 请求关联，以及可选的 `S3C_LOG_JSON` 结构化日志；缺的是**跨子操作的 span 树**
  与 W3C Trace Context 互操作（上游网关 / 下游采集端按标准头传递采样决定）。
- 本仓库的可执行体是**单个 Go 二进制**（`apps/server`），依赖清单受
  [`../DEVELOPMENT.md`](../DEVELOPMENT.md) §4「依赖增删改后重新生成 THIRD_PARTY_LICENSES」与
  供应链门禁约束；前端已有“零运行时依赖”先例（ADR-004 / ADR-008）。
- 本期需要的 trace 面很窄：每请求一个 server span + 三个业务子 span（`presign` / `proxy` /
  `migrate`），导出到标准 OTLP 采集端。不需要 metrics / logs 信号，不需要 tail sampling、
  不需要 OTLP/gRPC、不需要自定义 resource 探测。

因此必须决定：接入官方 OpenTelemetry Go SDK，还是用标准库自研最小实现；以及默认开关与
采样口径。

## Decision

**用 Go 标准库自研最小 tracer，零新增第三方依赖；OTLP/HTTP JSON 导出；默认关闭。**

落点：

- 新包 [`../../apps/server/internal/tracing/`](../../apps/server/internal/tracing/tracing.go)：
  - `Config{Endpoint, ServiceName string; SampleRatio float64}`；`New(Config) (*Tracer, error)`；
    `(*Tracer).Enabled()`；`(*Tracer).Middleware(http.Handler) http.Handler`；
    包级 `Start(ctx, name) (context.Context, func())`；`(*Tracer).Close(context.Context) error`。
  - W3C `traceparent` 解析 / 生成（`00-<32hex>-<16hex>-<2hex>`；版本非 `00`、字段数 / 长度不符、
    trace id / span id 全 0 一律按“不可继承、新建 trace”处理）。
  - 导出 `POST {Endpoint}/v1/traces`，`Content-Type: application/json`，结构
    `resourceSpans[].scopeSpans[].spans[]`（含 `traceId` / `spanId` / `parentSpanId` /
    `name` / `kind` / `startTimeUnixNano` / `endTimeUnixNano` / `attributes`；resource attribute
    至少 `service.name`）。有界队列（512）+ 后台批量（64/批，1s ticker）+ `Close` 刷出；
    导出失败只记日志，绝不影响请求。
- 接线：[`../../apps/server/main.go`](../../apps/server/main.go) 用 `tracer.Middleware(h.Routes())`
  包住 `http.Server.Handler`，退出路径 `Close` 刷出；`presign` / `proxy` / `migrate`
  （含 `/api/migrate/sync`、`/api/migrate/async`）用 `tracing.Start` 打子 span；server span
  带 `http.request.method` / `url.path` / `http.response.status_code` / `request.id`
  （即既有 `X-Request-ID`，不新增日志字段）。
- 配置（[`../CONFIGURATION.md`](../CONFIGURATION.md) SSOT）：`S3C_OTEL_ENDPOINT`（默认空 =
  关闭）、`S3C_OTEL_SAMPLE_RATIO`（默认 `1`，仅接受 `[0,1]`，非法即拒绝启动）、
  `S3C_OTEL_SERVICE_NAME`（默认 `s3client`）。
- 采样口径：`SampleRatio` 只决定**新建 trace**；入站 `traceparent` 采样位为 `1` 时即使
  ratio=0 也继承采样，为 `0` 时不重新采样（不做尾部采样）。
- 运维口径见 [`../OPERATIONS.md`](../OPERATIONS.md) §3.4。

## Alternatives Considered

1. **接入官方 `go.opentelemetry.io/otel` + `otlptracehttp`**
   - Pros：规范完整（span status / events / links / sampler 生态）、长期由社区维护、
     未来接 metrics / logs 同一套 API。
   - Cons：引入十数个传递依赖（`otel` API/SDK/exporters、`grpc`/`protobuf` 视 exporter 而定），
     二进制与依赖清单显著变大；需同步重生成 `THIRD_PARTY_LICENSES.md` 并接受新的供应链面；
     对当前「4 类 span、单一 OTLP/HTTP 接收端」的窄需求属于过度配置。若将来需要完整语义，
     可另写 ADR 推翻本篇（ADR 不归档、只叠加）。
2. **OTLP/gRPC（`otlptracegrpc` / 自研 gRPC）**
   - Pros：采集端默认同时开 4317，传输更省。
   - Cons：必须依赖 `google.golang.org/grpc` + `protobuf`，与「零新增依赖」直接冲突；
     自研 gRPC 帧与 protobuf 编解码的成本远高于 OTLP/HTTP JSON。
3. **什么都不做，继续只用 Prometheus 指标 + `X-Request-ID`**
   - Pros：零成本、零新失败模式。
   - Cons：无法表达「一次请求内的子操作耗时 / 父子关系」，也没有 W3C 头互操作；
     #11 的验收目标（trace 贯穿签名 / 代理 / 迁移）无法达成。
4. **只把 trace id 写进结构化日志（结构化日志追踪）**
   - Pros：实现最省，复用现有日志管道。
   - Cons：没有 span 树与采集端生态，跨服务传播仍要靠自定义字段；不满足 OTLP 互操作诉求。
5. **改用 Jaeger / Zipkin 私有 JSON 协议**
   - Pros：实现同样简单，且这些后端成熟。
   - Cons：与 OTLP 这一事实标准分叉，采集端换型要重写导出；违背“对齐标准”的目标。

## Consequences

- **正面**：零新增依赖（`go.mod` 不变、`THIRD_PARTY_LICENSES.md` 无需重生成）；默认关闭时
  请求路径零开销；标准 W3C `traceparent` 可与上游网关 / 下游采集端互操作；`presign` / `proxy` /
  `migrate` 具备可导出的子 span 树，且与既有 `X-Request-ID` 日志口径通过 span 属性 `request.id`
  对齐；采集端故障、队列溢出都不会影响业务请求。
- **代价 / 风险**：只实现 OTLP 字段子集（无 events / links / span status、resource 只有
  `service.name`、无 OTLP/gRPC / protobuf）；采样为 head sampling，无尾部采样；队列容量与批量
  参数是代码内常量（无环境变量）；队列满时静默（WARN + 计数）丢弃 span，观测数据可能不完整；
  `Close` 的 5s 刷出窗口内采集端不可用会丢最后一批。
- **后续**：配置与运维口径已同步 [`../CONFIGURATION.md`](../CONFIGURATION.md)、
  [`../OPERATIONS.md`](../OPERATIONS.md) §3.4、[`../architecture.md`](../architecture.md) §2/§7 与本索引；
  行为由 `apps/server/internal/tracing/*_test.go`（导出报文 / traceparent / 采样 / 队列）与
  `apps/server/internal/handler/tracing_wiring_test.go`（子 span 接线）守住。若将来需要完整
  OTel 语义或 gRPC，按上述替代方案 1/2 另写 ADR 推翻本篇。
