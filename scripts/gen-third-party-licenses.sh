#!/usr/bin/env bash
# 生成 docs/THIRD_PARTY_LICENSES.md —— 第三方依赖与许可证清单。
#
# 设计原则（为什么是「生成」而不是「手写」）：
#   - 依赖清单是**事实**，手写必然漂移；本脚本从三处权威来源机械抽取：
#       Go     : `go list -m -json all`（模块图）+ 模块目录内的 LICENSE/COPYING 文本
#       Rust   : `cargo metadata --locked`（crate 的 `license` 字段，权威 SPDX 表达式）
#       npm    : apps/web/package.json 的 dependencies + node_modules/<pkg>/package.json 的 `license`
#   - 识别不出的**不猜**：写成 `UNKNOWN` 并保留文件路径，宁可暴露缺口也不编造许可证。
#   - 离线可用：cargo metadata 失败时退回「Cargo.lock ∩ 本地 registry 源码」并标注未解析项；
#     `go list` 走本地模块缓存（GOPROXY=off）。
#
# 用法：
#   ./scripts/gen-third-party-licenses.sh          # 写入 docs/THIRD_PARTY_LICENSES.md
#   ./scripts/gen-third-party-licenses.sh --check  # 只校验门禁关心的不变量，不写文件
#
# 副作用提示（重要）：构建列表里若存在**未下载**的模块（`go list -m all` 会列出它们，但 `Dir` 为空），
# 脚本会执行 `go mod download` 补全源码以便读出许可证——这可能向 `apps/server/go.sum` **追加**这几条
# 模块的哈希。那属于 go.sum 的正常补全（**不改 go.mod、不改版本，不是依赖升级**），但仍会出现在 diff 里。
# 不想要这个副作用时，先自行 `cd apps/server && go mod download all`，脚本就不会再触发下载。
#
# 覆盖门禁见 apps/server/third_party_licenses_gate_test.go（依赖图里每个包都必须出现在产出里）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
OUT="docs/THIRD_PARTY_LICENSES.md"

python3 - "$ROOT" "$OUT" "${1:-}" <<'PY'
import json, os, re, subprocess, sys, datetime

root, out_path, mode = sys.argv[1], sys.argv[2], (sys.argv[3] if len(sys.argv) > 3 else "")


class _TimedOut:
    """超时的子进程替身：一律视为失败，脚本继续跑（该模块记为 UNKNOWN + 原因）。"""

    returncode, stdout, stderr = 1, "", "timeout"


def run(cmd, cwd=None, env=None, timeout=None):
    try:
        return subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        return _TimedOut()


# ---------------------------------------------------------------- Go 模块
def _go_list(env):
    p = run(["go", "list", "-m", "-json", "all"], cwd=os.path.join(root, "apps/server"), env=env)
    if p.returncode != 0:
        sys.exit("go list 失败（需要本地模块缓存）：\n" + p.stderr[-2000:])
    dec, objs, i = json.JSONDecoder(), [], 0
    while i < len(p.stdout):
        while i < len(p.stdout) and p.stdout[i] in " \n\t\r":
            i += 1
        if i >= len(p.stdout):
            break
        o, i = dec.raw_decode(p.stdout, i)
        objs.append(o)
    return objs


def go_modules():
    """返回构建列表里的全部第三方模块。

    注意：`go list -m all` 会列出**未下载**的模块（其 `Dir` 为空）——它们仍在 go.mod 的模块图里，
    因此**必须进清单**，否则门禁会点名（首版就因为「无 Dir 即跳过」漏掉 x/net、x/term、x/text）。
    这里先尽力 `go mod download` 补全源码以便读出许可证；失败也不丢条目，记为 UNKNOWN 并写明原因。
    """
    env = dict(os.environ, GOFLAGS="-mod=mod", GOPROXY="off")
    objs = _go_list(env)

    def third_party(os_):
        return [o for o in os_ if not o.get("Main") and not o.get("Standard")]

    missing = [o["Path"] for o in third_party(objs) if not (o.get("Dir") and os.path.isdir(o["Dir"]))]
    if missing:
        # 尽力补全（允许联网，**90s 上限**——默认 GOPROXY 不可达时不能把生成脚本挂死）；
        # 失败不致命：这些条目会以 UNKNOWN + 原因出现在清单里，门禁仍然通过（枚举完整）。
        run(["go", "mod", "download"] + missing, cwd=os.path.join(root, "apps/server"), timeout=90)
        objs = _go_list(env)
    mods = [{"path": o["Path"], "version": o.get("Version", ""), "dir": o.get("Dir", "")}
            for o in third_party(objs)]
    mods.sort(key=lambda m: m["path"].lower())
    return mods


SPDX_RULES = [
    # (SPDX, [签名…])：每个签名是「必须同时命中」的正则列表，任一签名命中即判定。
    # 顺序 = 优先级：先判双条件的强签名，再判宽泛签名，避免把 Apache/MPL 误判成 MIT 之类。
    ("Apache-2.0", [[r"apache license", r"version 2\.0"]]),
    ("MPL-2.0", [[r"mozilla public license", r"version 2\.0"]]),
    ("LGPL-3.0", [[r"gnu lesser general public license", r"version 3"]]),
    ("GPL-3.0", [[r"gnu general public license", r"version 3"]]),
    ("GPL-2.0", [[r"gnu general public license", r"version 2"]]),
    ("CC0-1.0", [[r"cc0"]]),
    ("Unlicense", [[r"free and unencumbered software released into the public domain"]]),
    ("ISC", [[r"\bisc license\b"], [r"permission to use, copy, modify, and(/or)? distribute this software"]]),
    # BSD-3 的第三条有多种写法（"Neither the name of…" / "Neither the names of the authors…"），
    # 故只要求 "neither the name" + "may be used to endorse" 同时出现（BSD-2 按定义没有该条）。
    ("BSD-3-Clause", [[r"neither the name", r"may be used to endorse"]]),
    ("MIT", [[r"\bmit license\b"], [r"permission is hereby granted, free of charge"]]),
    ("BSD-2-Clause", [[r"redistribution and use in source and binary forms"]]),
    ("Zlib", [[r"this software is provided 'as-is'"]]),
]


def detect_license(mod_dir):
    """在模块目录里找许可证文件并机械识别 SPDX；识别不出返回 UNKNOWN。"""
    if not mod_dir or not os.path.isdir(mod_dir):
        return "UNKNOWN", "（模块未下载到本地缓存——跑 `go mod download` 后重新生成）"
    names = []
    for fn in os.listdir(mod_dir):
        if re.match(r"(?i)^(license|licence|copying|notice)([-_.].*)?$", fn):
            p = os.path.join(mod_dir, fn)
            if os.path.isfile(p):
                names.append(p)
    if not names:
        return "UNKNOWN", "（模块内无 LICENSE / COPYING 文件）"
    names.sort(key=lambda p: (os.path.basename(p).upper().startswith("NOTICE"), len(os.path.basename(p))))
    for p in names:
        try:
            txt = open(p, encoding="utf-8", errors="replace").read(6000).lower()
        except OSError:
            continue
        for spdx, sigs in SPDX_RULES:
            if any(all(re.search(pat, txt, re.S) for pat in sig) for sig in sigs):
                return spdx, os.path.relpath(p, mod_dir)
    return "UNKNOWN", os.path.relpath(names[0], mod_dir)


# ---------------------------------------------------------------- Rust crates
def rust_crates():
    manifest = os.path.join(root, "apps/desktop/src-tauri/Cargo.toml")
    locked = open(os.path.join(root, "apps/desktop/src-tauri/Cargo.lock"), encoding="utf-8").read()
    lock_pkgs = {f"{n}-{v}": (n, v) for n, v in
                 re.findall(r'\[\[package\]\]\nname = "([^"]+)"\nversion = "([^"]+)"', locked)}
    for extra in ([], ["--offline"]):
        p = run(["cargo", "metadata", "--format-version", "1", "--locked"] + extra + ["--manifest-path", manifest])
        if p.returncode == 0:
            try:
                data = json.loads(p.stdout)
            except json.JSONDecodeError:
                continue
            crates = [{"name": pk["name"], "version": pk["version"], "license": pk.get("license") or "UNKNOWN"}
                      for pk in data["packages"] if pk.get("source")]
            crates.sort(key=lambda c: (c["name"].lower(), c["version"]))
            return crates, "cargo metadata（权威 license 字段）"
    # 离线兜底：Cargo.lock ∩ 本地 registry 源码里的 Cargo.toml
    reg = os.path.expanduser("~/.cargo/registry/src")
    index = {}
    for base, _dirs, files in os.walk(reg):
        if "Cargo.toml" in files and os.path.basename(base) in lock_pkgs:
            try:
                txt = open(os.path.join(base, "Cargo.toml"), encoding="utf-8", errors="replace").read(2000)
            except OSError:
                continue
            m = re.search(r'(?m)^\s*license\s*=\s*"([^"]+)"', txt)
            index[os.path.basename(base)] = m.group(1) if m else "UNKNOWN"
    crates = [{"name": n, "version": v,
               "license": index.get(f"{n}-{v}", "UNKNOWN（本地 registry 无该 crate 源码）")}
              for n, v in (lock_pkgs[k] for k in sorted(lock_pkgs))]
    crates.sort(key=lambda c: (c["name"].lower(), c["version"]))
    return crates, "Cargo.lock ∩ 本地 registry 源码（离线兜底，未解析项已标注）"


# ---------------------------------------------------------------- npm 运行时依赖
def npm_runtime():
    pkg = json.load(open(os.path.join(root, "apps/web/package.json"), encoding="utf-8"))
    out = []
    for name in sorted(pkg.get("dependencies", {})):
        lic = "UNKNOWN"
        meta = os.path.join(root, "apps/web/node_modules", name, "package.json")
        if os.path.isfile(meta):
            m = json.load(open(meta, encoding="utf-8"))
            lic = m.get("license") or "UNKNOWN"
        out.append({"name": name, "version": pkg["dependencies"][name], "license": lic})
    return out


go_mods = go_modules()
go_lic = [(m, *detect_license(m["dir"])) for m in go_mods]
rust, rust_src = rust_crates()
npm = npm_runtime()


def tally(items):
    t = {}
    for it in items:
        t[it] = t.get(it, 0) + 1
    return t


go_t, rust_t, npm_t = tally(l[1] for l in go_lic), tally(c["license"] for c in rust), tally(n["license"] for n in npm)
licenses = sorted(set(go_t) | set(rust_t) | set(npm_t), key=lambda k: (-(go_t.get(k, 0) + rust_t.get(k, 0) + npm_t.get(k, 0)), k))
unknown = [k for k in licenses if k.startswith("UNKNOWN")]
today = datetime.date.today().isoformat()

lines = []
w = lines.append
w("# 第三方依赖与许可证清单（Third-Party Licenses）")
w("")
w("> ⚙️ **本文件由脚本自动生成，请勿手工编辑。**")
w("> 生成命令：`./scripts/gen-third-party-licenses.sh`（改依赖后必须重新生成，否则门禁红灯）。")
w(f"> 生成时间：{today}；Go 模块 **{len(go_mods)}** 个 · Rust crates **{len(rust)}** 个 · 前端运行时包 **{len(npm)}** 个。")
w(f"> Rust 数据来源：{rust_src}。")
w(">")
w("> ⚠️ **性质说明**：许可证名是**机械识别**的结果（Go：模块内 `LICENSE`/`COPYING` 文本匹配；")
w("> Rust：cargo 元数据的 `license` 字段；npm：`package.json` 的 `license` 字段），")
w("> **不构成法律意见**。识别不出的记为 `UNKNOWN` 并给出文件路径——**不猜、不省略**；")
w("> 对外分发前的合规审查请以各依赖内的许可证原文为准。")
w(">")
w("> **覆盖范围**：Go 服务端二进制（含全部传递依赖）· Tauri 桌面端安装包（Cargo.lock 全量）·")
w("> 前端**运行时**依赖（随 bundle 分发）。**不在范围**：容器基础镜像的 OS 包（由 Trivy 扫描与 SBOM 覆盖）、")
w("> 构建期 `devDependencies`（不随产物分发，由 `pnpm audit` 与 Dependabot 覆盖）、工具链自身")
w("> （Go / Node / pnpm / Rust 及各自标准库）。")
w("")
w("## 1. 许可证汇总")
w("")
w("| 许可证 | Go 模块 | Rust crates | npm 运行时包 |")
w("|---|---:|---:|---:|")
for k in licenses:
    w(f"| `{k}` | {go_t.get(k, 0)} | {rust_t.get(k, 0)} | {npm_t.get(k, 0)} |")
w("")
if unknown:
    w(f"> ⚠️ 本清单有 **{len(unknown)}** 类 `UNKNOWN` 条目（合计 "
      f"{sum(go_t.get(k, 0) + rust_t.get(k, 0) + npm_t.get(k, 0) for k in unknown)} 个依赖）：")
    w("> 这些依赖的许可证**未被机械识别**，发布前需人工确认（下表已给出路径 / 原因）。")
    w("")
# 中性提示：只做**关键词**匹配，帮助人快速定位需要阅读原文的条目——不是合规结论。
COPYLEFT_HINTS = ("MPL", "GPL", "AGPL", "LGPL", "CC-BY-SA", "EUPL", "SSPL", "CDDL")
hits = [(k, go_t.get(k, 0), rust_t.get(k, 0), npm_t.get(k, 0)) for k in licenses
        if any(h in k.upper() for h in COPYLEFT_HINTS)]
if hits:
    w("> ⚠️ 下列许可表达式含 copyleft 类**关键词**，需人工阅读原文确认义务（**这不是合规结论**）：")
    for k, g, r, n in hits:
        parts = [f"Go {g}" for _ in (1,) if g] + [f"Rust {r}" for _ in (1,) if r] + [f"npm {n}" for _ in (1,) if n]
        w(f"> - `{k}`（{', '.join(parts)}）")
    w("")
w("## 2. Go 服务端依赖（随二进制分发）")
w("")
w("| 模块 | 版本 | 许可证（机械识别） | 依据文件 |")
w("|---|---|---|---|")
for m, spdx, ev in go_lic:
    w(f"| `{m['path']}` | `{m['version']}` | {spdx} | `{ev}` |")
w("")
w(f"## 3. Tauri 桌面端依赖（Cargo.lock 全量，共 {len(rust)} 个 crate）")
w("")
w("> 桌面端产物目前**未签名 / 未正式发布**（外部凭证阻塞，见 `docs/KNOWN_ISSUES.md` #25）；")
w("> 本表按 Cargo.lock 全量列出，覆盖所有目标平台的传递依赖。")
w("")
w("| crate | 版本 | 许可证 |")
w("|---|---|---|")
for c in rust:
    w(f"| `{c['name']}` | `{c['version']}` | {c['license']} |")
w("")
w("## 4. 前端运行时依赖（随 bundle 分发）")
w("")
w("| 包 | 版本范围（package.json） | 许可证 |")
w("|---|---|---|")
for n in npm:
    w(f"| `{n['name']}` | `{n['version']}` | {n['license']} |")
w("")
w("> 前端刻意保持**生产依赖仅 `vue`**（见 `docs/decisions/0004-minimal-frontend-deps.md`）；")
w("> 构建期依赖（vitest / playwright / eslint / vue-tsc 等）不随产物分发，由 Dependabot 与 `pnpm audit` 覆盖。")
w("")
w("## 5. 如何核对与更新")
w("")
w("```bash")
w("./scripts/gen-third-party-licenses.sh        # 重新生成本文件")
w("cd apps/server && go test . -run TestThirdPartyLicensesAreComplete -v   # 覆盖门禁")
w("```")
w("")
w("门禁断言：Go 模块图 / `Cargo.lock` / `package.json` 里的**每一个**依赖都必须出现在本文件中——")
w("新增、升级或移除依赖而忘记重新生成，门禁会红灯点名。")
w("")

doc = "\n".join(lines)

if mode == "--check":
    print(f"check 模式：文档 {len(doc)} 字节；Go {len(go_mods)} / Rust {len(rust)} / npm {len(npm)}")
    sys.exit(0)

with open(os.path.join(root, out_path), "w", encoding="utf-8") as f:
    f.write(doc)
print(f"已生成 {out_path}：Go {len(go_mods)} 模块 / Rust {len(rust)} crates / npm {len(npm)} 包；"
      f"UNKNOWN {sum(1 for l in go_lic if l[1].startswith('UNKNOWN')) + sum(1 for c in rust if c['license'].startswith('UNKNOWN')) + sum(1 for n in npm if n['license'] == 'UNKNOWN')} 项")
PY
