#!/usr/bin/env bash
# 一键同步版本号到仓库内所有约定文件。
# 用法：
#   ./scripts/release-version.sh                         # 自动生成 v1.0.0-YYYYMMDDHHmmss
#   ./scripts/release-version.sh v1.0.0-20260902120000    # 指定时间戳版本（带 v）
#   ./scripts/release-version.sh v1.0.0-rc0               # 指定预发布版本（带 v）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

DISPLAY="${1:-}"
if [[ -z "$DISPLAY" ]]; then
  DISPLAY="v1.0.0-$(date +%Y%m%d%H%M%S)"
fi
if [[ ! "$DISPLAY" =~ ^v1\.0\.0-([0-9]{14}|rc[0-9]+)$ ]]; then
  echo "error: version must match v1.0.0-YYYYMMDDHHmmss or v1.0.0-rcN, got: $DISPLAY" >&2
  exit 1
fi
MACHINE="${DISPLAY#v}"

echo "Syncing version: display=$DISPLAY machine=$MACHINE"

# Makefile VERSION
sed -i "s/^VERSION ?= .*/VERSION ?= $DISPLAY/" Makefile

# Go main + Dockerfile ARG
sed -i "s/var version = \"v[^\"]*\"/var version = \"$DISPLAY\"/" apps/server/main.go
sed -i "s/^ARG VERSION=.*/ARG VERSION=$DISPLAY/" apps/server/Dockerfile

# docker-compose image tag
sed -i "s|image: s3clinet/server:v[^\"]*|image: s3clinet/server:$DISPLAY|" docker-compose.yml
sed -i "s|S3C_IMAGE_TAG:-v[^}]*}|S3C_IMAGE_TAG:-$DISPLAY}|" docker-compose.prod.yml

# npm / cargo machine semver
for f in apps/web/package.json apps/desktop/package.json; do
  sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"$MACHINE\"/" "$f"
done
sed -i "s/^version = \"[^\"]*\"/version = \"$MACHINE\"/" apps/desktop/src-tauri/Cargo.toml
sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"$MACHINE\"/" apps/desktop/src-tauri/tauri.conf.json

# Cargo.lock：本包版本（与 Cargo.toml 对齐）
if [[ -f apps/desktop/src-tauri/Cargo.lock ]]; then
  # 仅替换 name = "s3clinet-desktop" 或项目包名附近的 version；用宽松匹配本仓库旧版本串
  sed -i "s/version = \"1\\.0\\.0-[^\"]*\"/version = \"$MACHINE\"/" apps/desktop/src-tauri/Cargo.lock
fi

# README docker 镜像 tag
sed -i "s|s3clinet/server:v[0-9a-zA-Z.-]*|s3clinet/server:$DISPLAY|g" README.md
sed -i "s/当前版本 \`v1\\.0\\.0-[^\`]*\`/当前版本 \`$DISPLAY\`/" README.md

# docs/api.md health example
sed -i "s/\"version\":\"v[^\"]*\"/\"version\":\"$DISPLAY\"/" docs/api.md

echo "Done. Files updated under $ROOT"
echo "Remember to add a CHANGELOG.md entry for [$DISPLAY]"
