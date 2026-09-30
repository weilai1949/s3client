#!/usr/bin/env bash
# 一键同步版本号到仓库内所有约定文件。
# 用法：
#   ./scripts/release-version.sh                         # 自动生成 v1.0.0-YYYYMMDDHHmmss
#   ./scripts/release-version.sh v1.0.0-20260902120000    # 指定时间戳版本（带 v）
#   ./scripts/release-version.sh v1.0.0-rc0               # 指定预发布版本（带 v）
#   ./scripts/release-version.sh v1.0.0                   # 首个稳定里程碑（纯 semver）
#   ./scripts/release-version.sh v1.1.0                   # 后续次版本
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

DISPLAY="${1:-}"
if [[ -z "$DISPLAY" ]]; then
  DISPLAY="v1.0.0-$(date +%Y%m%d%H%M%S)"
fi
# 接受：纯 semver（v1.0.0 / v1.0.1 / v1.1.0）+ 预发布（rcN / alphaN / betaN）+ 时间戳版本。
# 此前正则只接受 `v1.0.0-<时间戳|rcN>`，会**拒绝纯 v1.0.0**（roadmap 的首个稳定里程碑）与
# v1.0.1 / v1.1.0（docs/archive/review-2026-09-19.md §7.3 D3）。
if [[ ! "$DISPLAY" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-([0-9]{14}|(rc|alpha|beta)[0-9]*))?$ ]]; then
  echo "error: version must be vMAJOR.MINOR.PATCH with optional -YYYYMMDDHHmmss / -rcN / -alphaN / -betaN, got: $DISPLAY" >&2
  exit 1
fi
MACHINE="${DISPLAY#v}"

echo "Syncing version: display=$DISPLAY machine=$MACHINE"

# 仓库内所有「旧版本串」的通用匹配（display 带 v 前缀，machine 不带）。
# 此前多处 sed 硬编码 `1\.0\.0-`，导致同步到 v1.1.0 时匹配不到、静默不更新。
OLD_DISPLAY_RE='v[0-9][0-9a-zA-Z.+-]*'
OLD_MACHINE_RE='[0-9][0-9a-zA-Z.+-]*'

# Makefile VERSION
sed -i "s/^VERSION ?= .*/VERSION ?= $DISPLAY/" Makefile

# Go main + Dockerfile ARG
sed -i "s/var version = \"$OLD_DISPLAY_RE\"/var version = \"$DISPLAY\"/" apps/server/main.go
sed -i "s/^ARG VERSION=.*/ARG VERSION=$DISPLAY/" apps/server/Dockerfile

# openapi.go 包注释里的示例版本串
sed -i "s|openapi.New(\"s3clinet API\", \"$OLD_MACHINE_RE\")|openapi.New(\"s3clinet API\", \"$MACHINE\")|" apps/server/internal/openapi/openapi.go

# docker-compose image tag
sed -i "s|image: s3clinet/server:$OLD_DISPLAY_RE|image: s3clinet/server:$DISPLAY|" docker-compose.yml
sed -i "s|S3C_IMAGE_TAG:-$OLD_DISPLAY_RE}|S3C_IMAGE_TAG:-$DISPLAY}|" docker-compose.prod.yml

# npm / cargo machine semver
for f in apps/web/package.json apps/desktop/package.json; do
  sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"$MACHINE\"/" "$f"
done
sed -i "s/^version = \"[^\"]*\"/version = \"$MACHINE\"/" apps/desktop/src-tauri/Cargo.toml
sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"$MACHINE\"/" apps/desktop/src-tauri/tauri.conf.json

# Cargo.lock：只改本包（name = "s3clinet"）的 version，绝不能全局替换——锁文件里有 400+ 个
# 依赖包的 version 行，全局替换会把整个依赖树版本改坏。
if [[ -f apps/desktop/src-tauri/Cargo.lock ]]; then
  awk -v ver="$MACHINE" '
    /^name = "s3clinet"$/ { in_self = 1; print; next }
    in_self && /^version = "/ { sub(/^version = "[^"]*"/, "version = \"" ver "\""); in_self = 0 }
    { print }
  ' apps/desktop/src-tauri/Cargo.lock > apps/desktop/src-tauri/Cargo.lock.tmp \
    && mv apps/desktop/src-tauri/Cargo.lock.tmp apps/desktop/src-tauri/Cargo.lock
fi

# README docker 镜像 tag + 当前版本
sed -i "s|s3clinet/server:$OLD_DISPLAY_RE|s3clinet/server:$DISPLAY|g" README.md
sed -i "s/当前版本 \`$OLD_DISPLAY_RE\`/当前版本 \`$DISPLAY\`/" README.md

# docs：健康检查示例（api.md）、部署文档、路线图当前版本、FEATURES 页脚
# 注意：目标文件名一律用大写元文档名（CONFIGURATION/DEPLOYMENT/DEVELOPMENT/AI_POLICY/
# KNOWN_ISSUES/FEATURES/ROADMAP）；改名务必同步本段，否则 sed 找不到文件会让 `set -e` 中断发版。
# 刻意**不**同步 .github/SECURITY.md：它只描述「支持窗口与版本方案」、不 pin 具体版本号
#（原硬编码 `当前版本 v1.0.0`，发版后会漂移；2026-09-29 改为指向 Makefile VERSION / /api/health 的 SSOT）。
sed -i "s/\"version\":\"$OLD_DISPLAY_RE\"/\"version\":\"$DISPLAY\"/" docs/api.md
sed -i "s/\"version\":\"$OLD_DISPLAY_RE\"/\"version\":\"$DISPLAY\"/" docs/DEPLOYMENT.md
sed -i "s/当前版本 \*\*\`$OLD_DISPLAY_RE\`\*\*/当前版本 **\`$DISPLAY\`**/" docs/ROADMAP.md
sed -i "s/最后更新：[0-9-]*（\`$OLD_DISPLAY_RE\` 之后的 Unreleased 区间/最后更新：$(date +%Y-%m-%d)（\`$DISPLAY\` 之后的 Unreleased 区间/" docs/FEATURES.md

# 提交版 OpenAPI 规范：info.version 必须与 apps/server/main.go 的 `var version` 一致，
# 否则 internal/handler 的 TestCommittedOpenAPISpecIsDiscoverable 会红灯。
# 只匹配 `"version": "v…"`（带值），不碰 schema 里作为**属性名**出现的 `"version": {`。
sed -i "s/\"version\": \"$OLD_DISPLAY_RE\"/\"version\": \"$DISPLAY\"/" docs/api/openapi.json

# 运维手册里 /api/health 的响应示例（与 api.md / DEPLOYMENT.md 同一 JSON 形态）。
sed -i "s/\"version\":\"$OLD_DISPLAY_RE\"/\"version\":\"$DISPLAY\"/" docs/OPERATIONS.md

# 兼容性政策里**声明性**的「当前版本」标注。
# 注意：本文件其余 `v1.0.0-YYYYMMDDHHmmss` / `v1.0.0-rcN` 是**版本方案示例**，
# 不随发版变化，故只匹配带「（当前版本）」后缀的那一处，避免 $OLD_DISPLAY_RE
# 的贪婪匹配把格式占位符一起改写（`v[0-9][0-9a-zA-Z.+-]*` 会吃掉 `-YYYYMMDDHHmmss`）。
sed -i "s/\`$OLD_DISPLAY_RE\`（当前版本）/\`$DISPLAY\`（当前版本）/" docs/compatibility.md

# GitHub issue 模板的版本占位
sed -i "s/- 版本：\`$OLD_DISPLAY_RE\`/- 版本：\`$DISPLAY\`/" .github/ISSUE_TEMPLATE/bug_report.md

# 硬检查：CHANGELOG 必须有 `## [<version>]` 版本段（tag 与段的一一对应由
# apps/server/changelog_tag_gate_test.go 校验；历史快照 tag / 段的口径见 CHANGELOG 顶部
# 「tag ↔ 版本段对应关系（唯一台账）」——KNOWN_ISSUES #69 收口）。
if ! grep -q "^## \[$DISPLAY\]" CHANGELOG.md; then
  echo "error: CHANGELOG.md 缺少 '## [$DISPLAY]' 版本段——请先按 CHANGELOG 顶部" \
       "「tag ↔ 版本段对应关系」口径补段（快照 tag 需登记映射表）再重跑" >&2
  exit 1
fi

echo "Done. Files updated under $ROOT"
