# 贡献指南（Contributing）

感谢你愿意为 s3clinet 贡献！本仓库包含 Go 后端、Vue 前端与 Tauri 桌面壳。参与即表示你同意
[行为准则](CODE_OF_CONDUCT.md)；发现安全问题请走 [安全策略](SECURITY.md) 的私有渠道，不要开 public issue。

## 开发规范速览

- **TDD 优先**、必验门禁、验收清单、Red Flags：见 [docs/development.md](docs/development.md)
- **文档同步**：修复 bug 或新增功能完成后**必须**更新相关文档（README / `docs/API.md` / `CHANGELOG.md` / `docs/todolist.md` 等），文档未同步视为改动未完成；对照表见 [docs/development.md](docs/development.md) §4 与 [agents.md](agents.md)
- **架构**与关键决策：见 [docs/architecture.md](docs/architecture.md) 与 [docs/decisions/](docs/decisions/index.md)
- **API 契约**：见 [docs/API.md](docs/API.md)（与 OpenAPI 注册表一致；改 handler 请求体时同步更新 `openapi_register_*.go`）

## 分支与提交流程

采用 `develop`（主开发）→ `main`（稳定发布）工作流：

```bash
git checkout develop
git pull
git checkout -b feat/<你的功能>
# ... 开发（TDD：先红灯，再绿灯，再重构）...
cd server && go vet ./... && go test ./...
cd web && pnpm test && pnpm build
git commit -m "feat: ..."
```

## 开发环境

- Go 1.26+ / Node 24+ / pnpm 9+ / Rust（桌面端）
- 后端本地启动 + 本机 RustFS 联调：见 [docs/development.md](docs/development.md) §2「真实 RustFS 联调」

## 提交规范

- 使用 [Conventional Commits](https://www.conventionalcommits.org/zh-CN/)：`feat:`、`fix:`、`docs:`、`refactor:`、`test:` 等。
- 每个 commit 只做一件事；重构与功能分开。
- 改动前先跑 `go test ./...`、`go vet ./...`、`pnpm build`（或 `make test-all`）。

## 发布流程

见 [docs/deployment.md](docs/deployment.md) §5 与 `scripts/release-version.sh`（同步 8 处版本号）。
