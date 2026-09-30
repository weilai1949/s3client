#!/usr/bin/env bash
# scripts/evals/run-golden-task.sh —— 黄金任务集（docs/AGENT_EVALS.md §二）的机械执行器。
#
# 作用：给定一个 GT id，读取 scripts/evals/golden-tasks.yaml 中该任务的 verification_commands
# 并逐条实跑，输出与其后一行机器可读摘要（EVAL_RESULT pass|fail|partial）+ JSON 报告。
#
# 诚实边界（必须知道）：
#   - 本脚本**只跑该任务的「完成判据门禁」**（verification_commands），它证明「判据命令可执行、
#     当前树上的结果是这样」，**不**证明「有代理在干净 checkout 上重做了这道黄金任务」。
#     真正的一次 GT 评测 = 代理重做任务 + 人工按 §三 评分卡打分；本脚本是其中的机械采证环节。
#   - 未知 task id / 任务集不可解析 → **fail closed**（非零退出 + EVAL_RESULT fail），绝不静默跳过。
#   - 命令带 requires_path 且该路径不存在时，记为 skip（整体 partial）并打印原因——不假装通过。
#
# 用法：
#   scripts/evals/run-golden-task.sh --list                 # 列出 GT id 与标题
#   scripts/evals/run-golden-task.sh GT-1                   # 实跑 GT-1 的判据命令
#   scripts/evals/run-golden-task.sh --dry-run GT-2         # 只打印将执行的命令，不执行
#   scripts/evals/run-golden-task.sh --json /tmp/gt1.json GT-1
#   scripts/evals/run-golden-task.sh --help
#
# 退出码：全部通过 / 有 skip（partial）→ 0；任一判据失败 → 1；未知 id / 用法错误 / 缺 python3 → 2。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TASKS_FILE="$ROOT/scripts/evals/golden-tasks.yaml"
# 字段分隔用 ASCII 31（unit separator）而非 TAB：TAB 属于 IFS 空白，bash `read` 会把连续
# TAB 折叠成一个，导致 requires_path 为空时字段整体左移（已踩过：空前置被误读成 "0"）。
SEP=$'\x1f'

# usage 打印脚本头部注释块（首个非注释行之前），不泄漏实现代码。
usage() { awk 'NR > 1 && /^#/ { sub(/^# ?/, ""); print; next } NR > 1 { exit }' "$0"; }

MODE="run"
TASK_ID=""
JSON_PATH=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --help|-h) usage; exit 0 ;;
    --list) MODE="list"; shift ;;
    --dry-run) MODE="dry"; shift ;;
    --json) JSON_PATH="${2:-}"; [[ -n "$JSON_PATH" ]] || { echo "error: --json 需要路径参数" >&2; exit 2; }; shift 2 ;;
    -*) echo "error: unknown arg: $1（见 --help）" >&2; exit 2 ;;
    *) [[ -z "$TASK_ID" ]] || { echo "error: 只接受一个 task id（多余：$1）" >&2; exit 2; }; TASK_ID="$1"; shift ;;
  esac
done

if ! command -v python3 >/dev/null 2>&1; then
  echo "error: 需要 python3 解析 $TASKS_FILE——缺它无法判断任务，fail closed" >&2
  echo "EVAL_RESULT fail"
  exit 2
fi
if [[ ! -f "$TASKS_FILE" ]]; then
  echo "error: 黄金任务集缺失：$TASKS_FILE（docs/AGENT_EVALS.md §二 的机器可读载体）" >&2
  echo "EVAL_RESULT fail"
  exit 2
fi

# gt_extract 用 python3 解析任务集（JSON 内容，.yaml 后缀；PyYAML 可用则优先 yaml.safe_load）。
# 成功时输出：首行 `TASK<US>id<US>title`，随后每条命令一行
# `CMD<US>cmd<US>desc<US>requires_path<US>optional(0|1)`；未知 id → 退出码 3（fail closed）。
gt_extract() { # <task-id>
  python3 - "$TASKS_FILE" "$1" <<'PY'
import json, sys
US = "\x1f"
path, tid = sys.argv[1], sys.argv[2]
data = None
try:
    import yaml  # 文件是 JSON（YAML 子集），PyYAML 只是可选的更通用读法
    with open(path, encoding="utf-8") as f:
        data = yaml.safe_load(f)
except Exception:
    data = None
if data is None:
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
tasks = {t["id"]: t for t in data["tasks"]}
if tid not in tasks:
    sys.stderr.write("unknown task id %r; known: %s\n" % (tid, ", ".join(sorted(tasks))))
    sys.exit(3)
t = tasks[tid]
sys.stdout.write(US.join(["TASK", t["id"], t["title"]]) + "\n")
for c in t.get("verification_commands", []):
    sys.stdout.write(US.join([
        "CMD", c["cmd"], c.get("desc", ""), c.get("requires_path", ""),
        "1" if c.get("optional") else "0"]) + "\n")
PY
}

# gt_parse 读取任务集全部 id/标题（--list 与解析自检用）。
gt_parse() {
  python3 - "$TASKS_FILE" <<'PY'
import json, sys
US = "\x1f"
data = None
try:
    import yaml
    with open(sys.argv[1], encoding="utf-8") as f:
        data = yaml.safe_load(f)
except Exception:
    data = None
if data is None:
    with open(sys.argv[1], encoding="utf-8") as f:
        data = json.load(f)
for t in data["tasks"]:
    sys.stdout.write(US.join([t["id"], t["title"]]) + "\n")
PY
}

# ---- --list ----
if [[ "$MODE" == "list" ]]; then
  if ! LISTED="$(gt_parse)"; then
    echo "error: 任务集解析失败：$TASKS_FILE" >&2
    exit 2
  fi
  while IFS= read -r line; do
    IFS="$SEP" read -r id title <<<"$line"
    printf '%s\t%s\n' "$id" "$title"
  done <<<"$LISTED"
  exit 0
fi

# ---- 读取任务 ----
if [[ -z "$TASK_ID" ]]; then
  echo "error: 缺少 task id（如 GT-1）；用 --list 查看（见 --help）" >&2
  exit 2
fi
if ! EXTRACTED="$(gt_extract "$TASK_ID")"; then
  echo "error: 未知 task id「$TASK_ID」或任务集不可解析——fail closed，不猜测、不跳过" >&2
  echo "EVAL_RESULT fail"
  exit 2
fi

TASK_TITLE=""
COMMANDS=()
while IFS= read -r line; do
  IFS="$SEP" read -r kind a b <<<"$line"
  case "$kind" in
    TASK) TASK_TITLE="$b" ;;
    CMD) COMMANDS+=("$line") ;;
  esac
done <<<"$EXTRACTED"

if [[ ${#COMMANDS[@]} -eq 0 ]]; then
  echo "error: $TASK_ID 的 verification_commands 为空——判据缺失，fail closed（任务集被掏空）" >&2
  echo "EVAL_RESULT fail"
  exit 2
fi

# ---- --dry-run ----
if [[ "$MODE" == "dry" ]]; then
  echo "DRY_RUN $TASK_ID  $TASK_TITLE"
  for entry in "${COMMANDS[@]}"; do
    IFS="$SEP" read -r _ cmd desc req opt <<<"$entry"
    printf '  - %s\n    cmd: %s\n' "$desc" "$cmd"
    [[ -n "$req" ]] && printf '    requires_path: %s\n' "$req"
  done
  echo "（dry-run 未执行任何判据命令，无 EVAL_RESULT）"
  exit 0
fi

# ---- 执行 ----
if [[ -z "$JSON_PATH" ]]; then
  REPORT_DIR="$(mktemp -d "${TMPDIR:-/tmp}/golden-task.XXXXXX")"
  JSON_PATH="$REPORT_DIR/report.json"
else
  REPORT_DIR="$(dirname "$JSON_PATH")"
  mkdir -p "$REPORT_DIR"
fi

now_ns() { date +%s%N; }
dur_ms() { echo $(( ($(now_ns) - $1) / 1000000 )); }

NAMES=(); CMDS=(); STATUS=(); DUR=(); NOTE=()
RESULT="pass"

echo "=== golden-task $TASK_ID  $TASK_TITLE ==="
idx=0
for entry in "${COMMANDS[@]}"; do
  IFS="$SEP" read -r _ cmd desc req opt <<<"$entry"
  idx=$((idx + 1))
  log="$REPORT_DIR/cmd${idx}.log"
  if [[ -n "$req" && ! -e "$ROOT/$req" ]]; then
    NAMES+=("$desc"); CMDS+=("$cmd"); STATUS+=("skip"); DUR+=("0")
    NOTE+=("前置缺失：$req 不存在——命令未运行（非静默跳过，装依赖后重跑）")
    [[ "$RESULT" == "pass" ]] && RESULT="partial"
    printf '  [%d] SKIP    %s\n        %s\n' "$idx" "$desc" "$cmd"
    continue
  fi
  t0="$(now_ns)"
  if ( cd "$ROOT" && bash -c "$cmd" ) >"$log" 2>&1; then
    ms="$(dur_ms "$t0")"
    NAMES+=("$desc"); CMDS+=("$cmd"); STATUS+=("pass"); DUR+=("$ms"); NOTE+=("")
    printf '  [%d] PASS %6s ms  %s\n' "$idx" "$ms" "$desc"
  else
    code=$?
    ms="$(dur_ms "$t0")"
    NAMES+=("$desc"); CMDS+=("$cmd"); STATUS+=("fail"); DUR+=("$ms"); NOTE+=("exit=$code log=$log")
    RESULT="fail"
    printf '  [%d] FAIL %6s ms  %s (exit=%s, log=%s)\n' "$idx" "$ms" "$desc" "$code" "$log"
    tail -n 8 "$log" | sed 's/^/        | /'
  fi
done

echo "EVAL_RESULT $RESULT"
echo "JSON_REPORT $JSON_PATH"

{
  printf '{\n'
  printf '  "tool": "scripts/evals/run-golden-task.sh",\n'
  printf '  "task_id": "%s",\n' "$TASK_ID"
  printf '  "task_title": "%s",\n' "$TASK_TITLE"
  printf '  "result": "%s",\n' "$RESULT"
  printf '  "scope_note": "verification_commands 实跑结果，不等于「代理端到端重做 GT」；五维评分属人工评审",\n'
  printf '  "commands": [\n'
  for i in "${!NAMES[@]}"; do
    comma=","
    [[ $i -eq $((${#NAMES[@]} - 1)) ]] && comma=""
    printf '    {"desc": "%s", "cmd": "%s", "status": "%s", "duration_ms": %s, "note": "%s"}%s\n' \
      "${NAMES[$i]}" "${CMDS[$i]}" "${STATUS[$i]}" "${DUR[$i]}" "${NOTE[$i]}" "$comma"
  done
  printf '  ]\n'
  printf '}\n'
} >"$JSON_PATH"

[[ "$RESULT" == "fail" ]] && exit 1
exit 0
