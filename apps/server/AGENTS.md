# AGENTS.md（apps/server 子树）

> 仅列 **Go 服务端专属**约束与入口；仓库级规则见根 [`AGENTS.md`](../../AGENTS.md)（已被注入，不复制）。

- **分层**：`config / atomicfile → model → store → s3wrap → service → handler → openapi`；
  AWS SDK 类型不得越过 `s3wrap` 进入 handler。单文件 ≈1000 行上限。
- **提交前**（在 `apps/server/` 内）：`go vet ./...`、`golangci-lint run`（配置 `.golangci.yml`，
  含 `unused`/`staticcheck`/`gosec`/`nolintlint`，**0 issues**）、`go test ./...`、`go build ./...`。
- **覆盖率门禁**是「100% 语句覆盖」而非百分比：CI 直接查 profile 里的 `count==0`；本地用
  `make test-cover`。真值口径见 [`docs/DEVELOPMENT.md`](../../docs/DEVELOPMENT.md) §3。
- **门禁测试在包根**（`package main`，改文档 / 配置 / CI / 环境变量**必须**先跑 `go test . -count=1`；
  下列为常用几道，**全量以包根 `*_gate_test.go` 为准**）：
  - `doc_link_gate_test.go`：全仓 md 相对链接目标 + GitHub slug 锚点必须存在（变异验证过）；
  - `doc_number_gate_test.go`：md 里「N 个 `/api/*` 端点」必须等于 `internal/handler/routes.go` 的注册数；
  - `config_doc_gate_test.go`：`internal/config` 读的每个 `S3C_*` 必须收录进 `docs/CONFIGURATION.md`；
  - `repo_infra_gate_test.go`：`.env.example` / Dockerfile / workflow / Makefile / Prometheus 规则；
  - `deadcode_gate_test.go`：测试与生产代码里的消音式死代码（`_ = x` / `var _ = x` / 未引用导出符号）；
  - `docs_naming_gate_test.go`：`docs/` 命名清单必须在 AGENTS / DEVELOPMENT §4 / llms.txt 三处同步登记。
- **配置项**：只在 `internal/config/config.go` 的 `FromEnv`/`Validate` 读取；新增 `S3C_*` 同 PR
  改 `docs/CONFIGURATION.md` + `apps/server/.env.example`（另有其它包直读环境变量的历史例外，
  见 `config_doc_gate_test.go` 头注释）。
- **真实 S3 对端 E2E**（签名 / 预签名 / 分段 / 复制 / 标签 / 版本控制改动必须跑）：
  `S3CLINET_E2E=1 go test ./internal/s3wrap/ -run TestE2E -v`。
