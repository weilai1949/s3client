#!/usr/bin/env bash
# 真实联调浏览器冒烟（todolist #37）本地 / CI 统一入口。
#
# 做四件事，跑完自动清理：
#   1) 起真实 RustFS（默认 docker 自拉一份；--no-rustfs 时复用外部对端）；
#   2) 构建真实前端产物（vite build）与 Go 后端二进制；
#   3) 起真实 Go 后端，用 S3C_STATIC_DIR 托管上面的产物（同源提供页面 + /api）；
#   4) 用 Playwright 跑 apps/web/e2e-real/（**不 mock /api**），断言真实
#      账号/建桶/浏览器预签名直传/签名 GET 回读。
#
# **本脚本是这套联调编排的唯一来源**：本地 `make e2e-real`、GitHub Actions 与
# GitLab CI 都调用它，只是各自负责「装依赖 / 装浏览器」等环境准备。这样
# 「本地跑通 ≠ CI 跑通」的漂移面被压到最小（门禁 `TestRealE2EUsesSharedScript`
# 断言两侧 CI 都走本脚本，而不是各抄一份编排）。
#
# 用法：
#   scripts/e2e-real.sh                  # 全自动（docker 自起 RustFS）
#   scripts/e2e-real.sh --no-rustfs      # 复用外部 RustFS（CI service / 已起的实例）
#   scripts/e2e-real.sh --keep           # 跑完不清理，便于排查（打印各端点）
#   scripts/e2e-real.sh --skip-build     # 复用已有 apps/web/dist 与后端二进制
#
# 环境变量：RUSTFS_ENDPOINT（--no-rustfs 时必填）/ RUSTFS_PORT / SERVER_PORT /
#          RUSTFS_IMAGE / S3CLINET_ACCESS_KEY / S3CLINET_SECRET_KEY。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

MANAGE_RUSTFS=1
KEEP=0
SKIP_BUILD=0
for arg in "$@"; do
  case "$arg" in
    --no-rustfs) MANAGE_RUSTFS=0 ;;
    --keep) KEEP=1 ;;
    --skip-build) SKIP_BUILD=1 ;;
    -h|--help)
      sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "未知参数: $arg（-h 查看用法）" >&2; exit 2 ;;
  esac
done

# 与 CI / compose 对齐的默认值：RustFS 镜像 pin 在 docker-compose.yml
# （一致性由 repo_infra_gate_test.go 的 TestRustFSImageIsConsistentlyPinned 守住）。
RUSTFS_IMAGE="${RUSTFS_IMAGE:-rustfs/rustfs:1.0.0-rc.3}"
RUSTFS_PORT="${RUSTFS_PORT:-9000}"
SERVER_PORT="${SERVER_PORT:-8080}"
RUSTFS_ACCESS_KEY="${RUSTFS_ACCESS_KEY:-rustfsadmin}"
RUSTFS_SECRET_KEY="${RUSTFS_SECRET_KEY:-rustfsadmin}"
# 外部对端模式：调用方给出完整 endpoint（如 GitLab service 的 http://rustfs:9000）。
RUSTFS_ENDPOINT="${RUSTFS_ENDPOINT:-}"
SERVER_ORIGIN="http://127.0.0.1:${SERVER_PORT}"

CONTAINER="s3clinet-e2e-rustfs"
WORK_DIR="$ROOT/.run/e2e-real"
SERVER_PID=""

log() { printf '\033[36m[e2e-real]\033[0m %s\n' "$*"; }
die() { printf '\033[31m[e2e-real] %s\033[0m\n' "$*" >&2; exit 1; }

# 端口是否被占用（bash /dev/tcp，无需 lsof）。
port_busy() { (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null; }

# 轮询 URL 直到 2xx，最多 tries 次（每次间隔 1s）。用「置标志位」而非 `&& exit 0`，
# 避免在 GitLab 那种「整段 script 拼成一个 shell」的环境里提前结束整个 job。
wait_http() {
  local url=$1 tries=${2:-30} i
  for ((i = 0; i < tries; i++)); do
    if curl -sf --max-time 2 "$url" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  return 1
}

cleanup() {
  local code=$?
  if (( KEEP )); then
    log "--keep：保留资源（RustFS 容器=$CONTAINER 后端 pid=$SERVER_PID endpoint=$RUSTFS_ENDPOINT）"
    log "  手动清理：kill $SERVER_PID; docker rm -f $CONTAINER"
    return 0
  fi
  if [[ -n "$SERVER_PID" ]]; then kill "$SERVER_PID" 2>/dev/null || true; fi
  # 只清自己起的容器：--no-rustfs 时对端由调用方（CI service）管理。
  if (( MANAGE_RUSTFS )); then docker rm -f "$CONTAINER" >/dev/null 2>&1 || true; fi
  exit "$code"
}
trap cleanup EXIT

command -v go >/dev/null || die "未找到 go"
command -v pnpm >/dev/null || die "未找到 pnpm"
if (( MANAGE_RUSTFS )); then
  command -v docker >/dev/null || die "未找到 docker：自起 RustFS 需要它（或改用 --no-rustfs 复用外部对端）"
  command -v curl >/dev/null || die "未找到 curl"
fi

mkdir -p "$WORK_DIR"

# ---- 1) 真实 RustFS ----------------------------------------------------------
if (( MANAGE_RUSTFS )); then
  RUSTFS_ENDPOINT="http://127.0.0.1:${RUSTFS_PORT}"
  if port_busy "$RUSTFS_PORT"; then
    die "端口 $RUSTFS_PORT 已被占用：请先停掉占用进程，或用 RUSTFS_PORT=其他端口 重跑"
  fi
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  log "启动 RustFS（$RUSTFS_IMAGE，端口 $RUSTFS_PORT）"
  # RUSTFS_CORS_ALLOWED_ORIGINS 是**必需**的：真实浏览器直传（预签名 PUT）是跨源请求
  # （页面在 :$SERVER_PORT，S3 在 :$RUSTFS_PORT），无 CORS 会被浏览器拦下；而 curl /
  # Playwright 的 APIRequestContext 不经 CORS 因而会「假绿」。必须显式放行页面 Origin，
  # 才真正验证到浏览器直传路径（todolist #37 的核心）。
  docker run -d --name "$CONTAINER" \
    -p "127.0.0.1:${RUSTFS_PORT}:9000" \
    -e RUSTFS_VOLUMES=/data \
    -e RUSTFS_ADDRESS=0.0.0.0:9000 \
    -e RUSTFS_CONSOLE_ADDRESS=0.0.0.0:9001 \
    -e RUSTFS_CONSOLE_ENABLE=true \
    -e RUSTFS_ACCESS_KEY="$RUSTFS_ACCESS_KEY" \
    -e RUSTFS_SECRET_KEY="$RUSTFS_SECRET_KEY" \
    -e RUSTFS_OBS_LOGGER_LEVEL=error \
    -e "RUSTFS_CORS_ALLOWED_ORIGINS=${SERVER_ORIGIN},http://localhost:${SERVER_PORT}" \
    -e "RUSTFS_CONSOLE_CORS_ALLOWED_ORIGINS=${SERVER_ORIGIN},http://localhost:${SERVER_PORT}" \
    "$RUSTFS_IMAGE" >/dev/null
else
  [[ -n "$RUSTFS_ENDPOINT" ]] || die "--no-rustfs 需要 RUSTFS_ENDPOINT（如 http://rustfs:9000）"
  log "复用外部 RustFS：$RUSTFS_ENDPOINT"
fi

log "等待 RustFS 就绪：$RUSTFS_ENDPOINT"
if ! wait_http "${RUSTFS_ENDPOINT}/health" 60; then
  (( MANAGE_RUSTFS )) && docker logs "$CONTAINER" >&2 2>/dev/null || true
  die "RustFS 未在 60s 内就绪（$RUSTFS_ENDPOINT/health）"
fi

# ---- 2) 真实构建产物 ---------------------------------------------------------
if (( ! SKIP_BUILD )); then
  log "构建前端真实产物（pnpm build）"
  (cd "$ROOT/apps/web" && pnpm install --frozen-lockfile >/dev/null && pnpm build >/dev/null)
  log "构建 Go 后端"
  (cd "$ROOT/apps/server" && go build -o "$WORK_DIR/s3clinet-server" .)
fi
[[ -f "$ROOT/apps/web/dist/index.html" ]] || die "缺少 apps/web/dist/index.html（去掉 --skip-build 重新构建）"
[[ -x "$WORK_DIR/s3clinet-server" ]] || die "缺少后端二进制 $WORK_DIR/s3clinet-server（去掉 --skip-build 重新构建）"

# 浏览器：幂等（已缓存则秒过）。修掉「新克隆下 make e2e-real 直接失败」的 UX 缺口；
# 系统依赖（--with-deps 需要 root/apt）由 CI 的 before_script 负责。
log "确保 Playwright chromium 已安装（幂等）"
(cd "$ROOT/apps/web" && pnpm exec playwright install chromium >/dev/null)

# ---- 3) 真实 Go 后端（托管真实产物 + /api） ----------------------------------
if port_busy "$SERVER_PORT"; then
  die "端口 $SERVER_PORT 已被占用：请先停掉占用进程，或用 SERVER_PORT=其他端口 重跑"
fi
DATA_DIR="$WORK_DIR/data"
rm -rf "$DATA_DIR"; mkdir -p "$DATA_DIR"
log "启动真实 Go 后端（$SERVER_ORIGIN，静态目录 apps/web/dist）"
(
  cd "$ROOT"
  S3C_ADDR="127.0.0.1:${SERVER_PORT}" \
  S3C_STATIC_DIR="$ROOT/apps/web/dist" \
  S3C_DATA_DIR="$DATA_DIR" \
  S3C_STORE_DRIVER=json \
  S3C_ALLOW_PLAINTEXT_STORE=1 \
  S3C_LOG_LEVEL=warn \
  "$WORK_DIR/s3clinet-server" >"$WORK_DIR/server.log" 2>&1 &
  echo $! >"$WORK_DIR/server.pid"
)
SERVER_PID="$(cat "$WORK_DIR/server.pid")"

log "等待后端 /api/health 就绪"
if ! wait_http "${SERVER_ORIGIN}/api/health" 30; then
  cat "$WORK_DIR/server.log" >&2 2>/dev/null || true
  die "后端未在 30s 内就绪"
fi

# ---- 4) 真实联调 Playwright --------------------------------------------------
log "运行 Playwright 真实联调（不 mock /api）"
(
  cd "$ROOT/apps/web"
  PLAYWRIGHT_BASE_URL="$SERVER_ORIGIN" \
  S3CLINET_ENDPOINT="$RUSTFS_ENDPOINT" \
  S3CLINET_ACCESS_KEY="$RUSTFS_ACCESS_KEY" \
  S3CLINET_SECRET_KEY="$RUSTFS_SECRET_KEY" \
  pnpm e2e:real
)
log "真实联调冒烟通过"
