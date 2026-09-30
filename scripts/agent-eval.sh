#!/usr/bin/env bash
# scripts/agent-eval.sh —— 「AI 效果证据」的机械评测部分。
# 人工部分（黄金任务集实操 + 加权评分卡打表）按 docs/AGENT_EVALS.md §评分卡执行，本脚本只跑
# 可复现的机械门禁子集并输出 JSON 报告 + 一行机器可读摘要（EVAL_RESULT pass|fail|partial）。
#
# 与 CI 的分工：本脚本**刻意不跑** docker / e2e-real / 真 S3 E2E（太重，CI 承担）；它只覆盖
# 「提交前门禁」的本地可复现子集，供评测时对一次 AI 改动做快速体检。
#
# 用法：
#   scripts/agent-eval.sh                          # 后端 + 前端（若 apps/web/node_modules 存在）
#   scripts/agent-eval.sh --web                    # 跳过前端（如评测仅涉及后端改动）
#   scripts/agent-eval.sh --json /tmp/agent-eval.json   # 指定 JSON 报告路径
#   # 默认 JSON 落在临时目录（mktemp -d，**不落仓库**），路径在结束时打印
# 退出码：任一阶段失败 → 1；全部通过（含部分跳过）→ 0。EVAL_RESULT=fail 时非零退出。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

WEB_SKIP=0
JSON_PATH=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --web) WEB_SKIP=1; shift ;;
    --json) JSON_PATH="$2"; shift 2 ;;
    --help|-h) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "error: unknown arg: $1（见 --help）" >&2; exit 2 ;;
  esac
done
if [[ -z "$JSON_PATH" ]]; then
  REPORT_DIR="$(mktemp -d "${TMPDIR:-/tmp}/agent-eval.XXXXXX")"
  JSON_PATH="$REPORT_DIR/report.json"
else
  REPORT_DIR="$(dirname "$JSON_PATH")"
  mkdir -p "$REPORT_DIR"
fi

now_ns() { date +%s%N; }
# dur_ms 计算纳秒差并换算毫秒（date +%s%3N 在本仓库所用环境会按 %N 输出纳秒，直接做纳秒差再除 1e6 最稳）。
dur_ms() { echo $(( ($(now_ns) - $1) / 1000000 )); }

STAGE_NAMES=(); STAGE_STATUS=(); STAGE_DUR=(); STAGE_NOTE=()
add_stage() { # name status duration_ms note
  STAGE_NAMES+=("$1"); STAGE_STATUS+=("$2"); STAGE_DUR+=("$3"); STAGE_NOTE+=("$4")
}

# run_stage 在 set -e 下执行阶段命令：失败不退出脚本，只记录 status=fail 与 exit code；
# 输出落进 REPORT_DIR 下以阶段名命名的日志，失败时回显日志尾部（不吞失败原因）。
run_stage() {
  local name="$1"; shift
  local t0 exit_code log
  t0="$(now_ns)"
  log="$REPORT_DIR/${name//:/_}.log"
  if "$@" >"$log" 2>&1; then
    add_stage "$name" "pass" "$(dur_ms "$t0")" ""
  else
    exit_code=$?
    add_stage "$name" "fail" "$(dur_ms "$t0")" "exit=$exit_code log=$log"
    echo "FAIL: $name (exit=$exit_code)" >&2
    tail -n 8 "$log" | sed 's/^/    | /' >&2
  fi
}

# server_cmd 在 apps/server 子目录内执行命令（go vet / go test / go build 都以该目录为模块根）。
server_cmd() { ( cd "$ROOT/apps/server" && "$@" ); }

# ---- 后端（apps/server）----
run_stage "server:vet"   server_cmd go vet ./...
run_stage "server:gate"  server_cmd go test . -count=1
run_stage "server:test"  server_cmd go test ./...
run_stage "server:build" server_cmd go build ./...

# ---- 前端（apps/web）----
if [[ "$WEB_SKIP" == "1" ]]; then
  add_stage "web:check" "skip" "0" "--web（显式跳过前端）"
elif [[ -d "$ROOT/apps/web/node_modules" ]]; then
  run_stage "web:check"     pnpm --dir "$ROOT/apps/web" lint
  run_stage "web:typecheck" pnpm --dir "$ROOT/apps/web" typecheck
  run_stage "web:test"      pnpm --dir "$ROOT/apps/web" test
else
  add_stage "web:check" "skip" "0" "apps/web/node_modules 不存在——未装依赖，跳过前端（先 cd apps/web && pnpm install）"
fi

# ---- 汇总 ----
RESULT="pass"
for s in "${STAGE_STATUS[@]}"; do
  [[ "$s" == "fail" ]] && { RESULT="fail"; break; }
  [[ "$s" == "skip" ]] && RESULT="partial"
done

echo "=== agent-eval 结果（阶段耗时 ms）==="
for i in "${!STAGE_NAMES[@]}"; do
  printf '  %-16s %-7s %6s ms %s\n' "${STAGE_NAMES[$i]}" "${STAGE_STATUS[$i]}" "${STAGE_DUR[$i]}" "${STAGE_NOTE[$i]}"
done
echo "EVAL_RESULT $RESULT"
echo "JSON_REPORT $JSON_PATH"

{
  echo '{'
  echo '  "tool": "scripts/agent-eval.sh",'
  echo "  \"result\": \"$RESULT\","
  echo '  "stages": ['
  for i in "${!STAGE_NAMES[@]}"; do
    comma=","
    [[ $i -eq $((${#STAGE_NAMES[@]} - 1)) ]] && comma=""
    printf '    {"name": "%s", "status": "%s", "duration_ms": %s, "note": "%s"}%s\n' \
      "${STAGE_NAMES[$i]}" "${STAGE_STATUS[$i]}" "${STAGE_DUR[$i]}" "${STAGE_NOTE[$i]}" "$comma"
  done
  echo '  ]'
  echo '}'
} > "$JSON_PATH"

[[ "$RESULT" == "fail" ]] && exit 1
exit 0
