# 版本号：日常发版 v1.0.0-YYYYMMDDHHmmss；预发布可用 v1.0.0-rcN。
# 手动覆盖构建：make VERSION=v1.0.0-rc0 server-build
VERSION ?= v1.0.0

.PHONY: server server-build tidy web web-build web-typecheck desktop-dev desktop-build rust-audit test test-cover web-test web-test-cover test-all vet lint govulncheck check install-hooks docker all dev dev-nginx restart restart-server restart-web restart-nginx restart-docker restart-all stop status gcl gcl-list gcl-docker

# Pin gitlab-ci-local，避免 npx latest 漂移。`.gitlab-ci-local-env` 已默认挂 docker.sock。
GCL ?= npx --yes gitlab-ci-local@4.75.1
GCL_JOBS ?=
GCL_EXTRA ?=

# 本地跑 GitLab 流水线（docker executor，无需 GitLab 实例）
gcl-list:
	$(GCL) --list $(GCL_EXTRA)

gcl:
	$(GCL) $(GCL_JOBS) $(GCL_EXTRA)

gcl-docker:
	$(GCL) docker $(GCL_EXTRA)

# Go 后端（构建并运行）
# 不在每次启动时跑 `go mod tidy`，避免依赖被无意识升级导致开发与 CI 漂移；
# 依赖更新请显式执行 `make tidy`。
server:
	cd apps/server && go build -ldflags="-X main.version=$(VERSION)" -o s3clinet-server . && ./s3clinet-server

# 显式同步依赖（开发者升级依赖或 PR 触发 CI 前的统一入口）
tidy:
	cd apps/server && go mod tidy

# 构建 Go 二进制（注入版本号）
server-build:
	cd apps/server && go build -ldflags="-X main.version=$(VERSION)" -o s3clinet-server .

# Web 前端开发
web:
	cd apps/web && pnpm install --frozen-lockfile && pnpm dev

# 构建 web 产物（含类型检查）
web-build:
	cd apps/web && pnpm install --frozen-lockfile && pnpm build

# 前端类型检查
web-typecheck:
	cd apps/web && pnpm install --frozen-lockfile && pnpm typecheck

# 桌面端开发（Tauri）
desktop-dev:
	cd apps/desktop && pnpm install --frozen-lockfile && pnpm tauri dev

# 打包桌面端
desktop-build:
	cd apps/web && pnpm install --frozen-lockfile && pnpm build
	cd apps/desktop && pnpm install --frozen-lockfile && pnpm tauri build

# RustSec 依赖审计（桌面端；与 CI desktop job 同命令，需本机已装 cargo-audit）
rust-audit:
	cd apps/desktop/src-tauri && cargo audit

# 后端测试（含 -race，检测数据竞争；CI 亦复用此目标）
test:
	cd apps/server && go test -race -count=1 -timeout 600s ./...

# 后端测试 + 覆盖率报告（100% 门禁：profile 中任何 count==0 的语句块即失败，与前端同级）
test-cover:
	cd apps/server && go test -race -count=1 -timeout 600s -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
	cd apps/server && awk 'NR > 1 && $$NF == 0 { print "uncovered block: " $$0; bad = 1 } END { if (bad) exit 1 }' coverage.out

# 前端单元测试
web-test:
	cd apps/web && pnpm install --frozen-lockfile && pnpm test

# 前端单元测试 + 覆盖率（vitest v8）
web-test-cover:
	cd apps/web && pnpm install --frozen-lockfile && pnpm test:coverage

# 后端 + 前端单测 + 两侧覆盖率门禁（与 CI 同强度；#38：test-all 此前无覆盖率门禁）
test-all: test-cover web-test-cover

# 后端静态检查
vet:
	cd apps/server && go vet ./...

# 后端 Lint（golangci-lint，与 CI 同版本 v2.13.2；#38）
lint:
	cd apps/server && golangci-lint run ./...

# Go 标准库/依赖可达漏洞门禁（与 CI 同版本；#38）
govulncheck:
	cd apps/server && go install golang.org/x/vuln/cmd/govulncheck@v1.8.0 && "$$(go env GOPATH)/bin/govulncheck" ./...

# 提交前门禁聚合：本地一条命令跑完与 CI 等价的静态检查 + 单测 + 覆盖率。
# 注意：CI 额外有 Trivy 镜像扫描、RustFS E2E、Playwright E2E，本目标不含（见 docs/development.md）。
check: vet lint test-cover web-test-cover

# 安装 git pre-commit hook（静态检查：gofmt / go vet / 前端 typecheck）
install-hooks:
	git config core.hooksPath .githooks
	@echo "pre-commit hook 已安装（gofmt + go vet + frontend typecheck）"

# 构建 Docker 镜像（注入版本号）
docker:
	docker build -f apps/server/Dockerfile -t s3clinet/server:$(VERSION) --build-arg VERSION=$(VERSION) .

# 一键构建全部
all: server-build web-build

# ---- 本地开发 & 优雅重启 ----
dev:
	bash scripts/run-dev.sh

dev-nginx:
	bash scripts/run-dev.sh --nginx

restart-server:
	bash scripts/graceful-restart.sh server

restart-web:
	bash scripts/graceful-restart.sh web

restart-nginx:
	bash scripts/graceful-restart.sh nginx

restart-docker:
	bash scripts/graceful-restart.sh docker

restart-all:
	bash scripts/graceful-restart.sh all

stop:
	bash scripts/graceful-restart.sh stop

status:
	bash scripts/graceful-restart.sh status
