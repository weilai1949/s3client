# syntax=docker/dockerfile:1

# 可通过 --build-arg 覆盖镜像仓库/代理（国内环境常用）。
ARG GOPROXY=https://proxy.golang.org,direct
ARG NPM_REGISTRY=https://registry.npmjs.org
# 注入到二进制的版本号（main.version）。
ARG VERSION=v1.0.0

###### Web 前端构建 ######
# Alpine + Node 26（Current；2026-10 转入 LTS）（固定到 patch：大版本 tag 会随上游重建，同一 commit 产出的层会漂移）。
# 供应链（P-7）：tag 之上再叠 `@sha256:` 内容寻址 digest——tag 可被上游重新推送，digest 不可变。
# digest 解析（2026-10-16）：docker pull node:26.10.0-alpine && docker inspect --format '{{index .RepoDigests 0}}' node:26.10.0-alpine
FROM node:26.10.0-alpine@sha256:0b36e8c136b94cd4fcf02188228e76c31ad5872eef3fec8cbd2eee500cfd9e80 AS web
ARG NPM_REGISTRY
WORKDIR /web
# 先还原依赖再拷贝源码，充分利用层缓存；锁文件一并拷贝保证可复现。
COPY apps/web/package.json apps/web/pnpm-workspace.yaml apps/web/pnpm-lock.yaml ./
RUN npm config set registry "$NPM_REGISTRY" \
 && npm install -g --no-fund pnpm@9.15.0
RUN pnpm install --frozen-lockfile
COPY apps/web/ .
RUN pnpm build

###### Go 后端构建 ######
# 固定 patch 版本（go.mod 要求 1.26.6）；alpine 镜像无 shell/apt 包，
# 仅产出静态二进制供运行时拷贝。
# 供应链（P-7）：同上的 tag + digest 双 pin（构建阶段不进运行时漏洞面，但同样要求可复现）。
FROM golang:1.26.9-alpine@sha256:3082400e369fa24d5fc60bca20edab3f6d604e0c5a690ec66b295eff4dd87ade AS build
ARG GOPROXY
ARG VERSION
RUN apk add --no-cache ca-certificates
WORKDIR /src
ENV GOPROXY=$GOPROXY
# 先还原模块再拷贝源码，充分利用层缓存
COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download
COPY apps/server/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/s3client-server .

###### 运行时 ######
# 关键修复：debian:bookworm-slim 体积大、glibc + openssl 漏洞面广（Trivy 反复告警）。
# 改用 alpine + 仅 ca-certificates + wget（供 HEALTHCHECK）：
#   - musl libc 比 glibc 漏洞面窄
#   - alpine 的 OpenSSL 受 Alpine 安全团队快速维护
#   - 体积约 5MB（vs debian-slim ~80MB）
#   - 仍保留 minimal busybox 工具用于 HEALTHCHECK 与调试
#
# 基础版必须是**仍受安全支持**的分支：3.20 的安全支持已于 2026-04-01 结束，而 Trivy 以
# --ignore-unfixed 运行——在 EOL 分支上「所有未来 CVE 永远无修复版」，漏洞门禁会结构性失明。
# 当前 3.24（支持至 2028-06-01）；3.21 于 2026-11-01 EOL，届时须再抬基线。
# 回归门禁见 apps/server/repo_infra_gate_test.go（TestRuntimeBaseImageIsSupportedAlpine）。
# 供应链（P-7）：tag + digest 双 pin。digest 解析（2026-10-16）：
#   docker pull alpine:3.24 && docker inspect --format '{{index .RepoDigests 0}}' alpine:3.24
#   -> alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
# `apk upgrade` 必须在 `apk add` 之前：digest pin 把基础层冻结在构建当时，而该层当前 ship 的
# zlib-1.3.2-r0 命中 CVE-2026-85091（HIGH，仓库已有修复版 1.3.2-r1）。同一次 upgrade 还会带上
# 3.24 分支内的其它安全补丁（实测 libcrypto3/libssl3 3.5.8-r0 → 3.5.9-r0）。
# 代价：apk 索引来自构建当时的网络，同一 digest 在不同日期重建可能拿到更新的补丁版本——
# 这是「在 digest 冻结基础上拿到安全修复」的显式取舍；复现口径见 docs/DEPLOYMENT.md。
RUN apk upgrade --no-cache \
 && apk add --no-cache ca-certificates wget \
 && addgroup -g 1000 -S app \
 && adduser -u 1000 -S -G app -h /home/app app \
 && mkdir -p /data /web/dist \
 && chown -R app:app /data /web/dist
COPY --from=build /out/s3client-server /s3client-server
COPY --from=web /web/dist /web/dist
ENV S3C_ADDR=0.0.0.0:5000 \
    S3C_STATIC_DIR=/web/dist \
    S3C_DATA_DIR=/data \
    S3C_REGION=us-east-1
VOLUME ["/data"]
WORKDIR /
USER app
EXPOSE 5000
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -q -O- http://127.0.0.1:5000/api/health >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/s3client-server"]
