#!/usr/bin/env bash
# 本地开发：启动 server / web /（可选）nginx，PID 写入 .run/
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/lib/process.sh
source "$ROOT/scripts/lib/process.sh"

USE_NGINX=0
for arg in "$@"; do
  case "$arg" in
    --nginx) USE_NGINX=1 ;;
    -h|--help)
      echo "用法: $0 [--nginx]"
      echo "  --nginx  Go 后端监听 5001，nginx(worker_processes=1) 对外 5000"
      exit 0
      ;;
  esac
done

start_server() {
  graceful_stop server "$(shutdown_timeout)"
  cd "$ROOT/apps/server"
  # 不跑 `go mod tidy`：它会在每次启动时改写 go.mod/go.sum（实测 `make dev` 每次删 go.sum 6 行），
  # 与 Makefile `server` 目标注释「不在每次启动时跑 go mod tidy，避免依赖被无意识升级导致开发与
  # CI 漂移」的口径相悖。`go build` 默认 -mod=readonly：依赖不一致会**显式报错**而非静默改写；
  # 要同步依赖请显式执行 `make tidy`（由 dev_scripts_gate_test.go 守住）。
  go build -o s3client-server .
  if (( USE_NGINX )); then
    export S3C_ADDR=127.0.0.1:5001
  else
    export S3C_ADDR="${S3C_ADDR:-127.0.0.1:5000}"
  fi
  # shellcheck disable=SC1091
  [[ -f .env ]] && set -a && source .env && set +a
  ./s3client-server >>"$RUN_DIR/server.log" 2>&1 &
  write_pid server $!
  wait_http "http://127.0.0.1:${S3C_ADDR##*:}/api/health" 30
  echo "[server] 已启动 pid=$(read_pid server) addr=$S3C_ADDR"
}

start_web() {
  graceful_stop web 15
  cd "$ROOT/apps/web"
  pnpm install --silent
  # 直接 exec vite：`.bin/vite` 是 pnpm 生成的 sh shim，末行 `exec node …/vite/bin/vite.js`
  # 不换 PID，故 write_pid 记录的就是 vite 本身。**不要改回 `pnpm dev`**——那会多出
  # pnpm.mjs → pnpm.cjs 两层包装，记录的 PID 命令行不含 "vite"，与
  # process.sh 的 expected_process_pattern web 失配 → 旧 vite 不被停止、继续占用 1949。
  # `--strictPort`：端口被占时显式失败，而不是静默换到 1950（见 dev_scripts_gate_test.go）。
  ./node_modules/.bin/vite --strictPort >>"$RUN_DIR/web.log" 2>&1 &
  write_pid web $!
  wait_http "http://127.0.0.1:1949/" 60
  echo "[web] 已启动 pid=$(read_pid web) http://127.0.0.1:1949"
}

start_server
start_web
if (( USE_NGINX )); then
  bash "$ROOT/scripts/nginx-local.sh"
fi

echo ""
echo "开发环境已就绪。停止: $ROOT/scripts/graceful-restart.sh stop"
echo "优雅重启: $ROOT/scripts/graceful-restart.sh all"
