# syntax=docker/dockerfile:1
#
# 仓库内 nginx 边缘镜像（P-4）。
#
# 为什么自建：上游 `nginx:*-alpine` 官方镜像的 digest 一旦 pin 住就被冻结，而当前**所有**
# nginx alpine tag（1.30.6 / stable / mainline）都还 ship `zlib-1.3.2-r0` —— 命中
# CVE-2026-85091（HIGH），而 Alpine 3.24 分支里修复版 `1.3.2-r1` 早已发布。上游重建时间
# 不可控，于是采取与 `apps/server/Dockerfile` 运行阶段相同的结构性解法：
#   **pin 上游基座 digest（不可变）+ 构建期 `apk upgrade` 重打分支补丁**。
#
# 基座 = nginx stable 分支（与 `nginx:stable-alpine` 同一 digest），Alpine 3.24.2（支持至
# 2028-06-01）。digest 解析（2026-10-16）：
#   docker pull nginx:1.30.6-alpine
#   docker inspect --format '{{index .RepoDigests 0}}' nginx:1.30.6-alpine
#   -> nginx@sha256:970364228b1e4f932abc22d321932816782cca8a20b85a77552a7cfe9acfb3b5
# 升级节奏：基座 digest 由 Dependabot `docker` 规则（`.github/dependabot.yml`，`/deploy/nginx`，月度）
# 跟随上游刷新，PR 触发 CI 重建 + Trivy 扫描；基座 Alpine 分支 EOL（3.24 → 2028-06-01）前抬到下一个
# 受支持分支并重解 digest。`apk upgrade` 的补丁面来自构建当时的 v3.24 索引。
#
# 门禁：apps/server/repo_infra_gate_test.go::TestNginxEdgeDockerfilePinsBaseAndAppliesPatches
#   （digest pin + `apk upgrade` 缺一即红）；compose 侧一致性见
#   TestNginxEdgeImageIsConsistentAcrossCompose。
FROM nginx:1.30.6-alpine@sha256:970364228b1e4f932abc22d321932816782cca8a20b85a77552a7cfe9acfb3b5

# 拿到 zlib-1.3.2-r1（CVE-2026-85091 HIGH）与同批安全更新。
# 验证：docker run --rm s3client/nginx:1.30.6-alpine apk version -v | grep zlib
RUN apk upgrade --no-cache
