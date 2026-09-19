# ADR-003：SSRF 防护放行私网 / 回环地址

## Status

Accepted

## Date

2026-09-16（回溯记录；决策自项目初版沿用）

## Context

s3clinet 允许用户配置任意 S3 兼容 endpoint（MinIO、RustFS、局域网对象存储、云服务商）。SSRF 防护需要平衡：

1. **安全**：禁止服务器访问云元数据（IMDS）与恶意内部地址。
2. **可用性**：自托管 S3（MinIO/RustFS/内网）是主场景，不能整体封锁私网/回环。

## Decision

SSRF 防护**拦截链路本地与云元数据地址，放行私网 / 回环**：

- 创建时 `ValidateEndpoint` + 拨号期 `dialContextSSRF` 双重校验（消除 DNS 重绑定 TOCTOU）。
- 拦截：链路本地单播/组播、阿里 IMDS（100.100.100.200 / 100.96.0.2）、AWS IMDS IPv6（fd00:ec2::254）、GCP metadata 主机名。
- 禁重定向 + 禁 HTTP(S)_PROXY（封环境变量代理绕过路径）。
- 认证是主防线：所有 `/api/*` 可配置 Bearer 鉴权，非回环监听强制鉴权。

## Alternatives Considered

### 整体封锁私网（只允许公网 S3）
- Pros：SSRF 面最小。
- Cons：MinIO / RustFS / 局域网对象存储全部不可用——**直接摧毁核心使用场景**。
- Rejected：项目定位是「S3 兼容客户端」，自托管是刚需。

## Consequences

- 认证用户可让服务器访问任意私网服务（仅依赖鉴权兜底）——已文档化为设计取舍。
- 攻击面：未配置 `S3C_TOKEN` 时服务默认仅回环监听，外部不可达（`config.go:120`）。
- 若需更严格部署，可结合网络层隔离（防火墙 / nginx 访问控制）。

## Update（2026-09-19）

默认策略不变（仍放行私网 / 回环），但补齐了「更严格部署」的进程内开关：

- 新增 `S3C_SSRF_DENY_PRIVATE`（`config.SSRFDenyPrivate` → `s3wrap.SetDenyPrivateNetworks`）。
  置 `1` 后 `ValidateEndpoint` 与 `dialContextSSRF` 双重校验一并拒绝 RFC1918 / ULA / 回环 / 未指定地址；
  公网端点、IMDS 与链路本地拦截逻辑不变。
- 该开关是进程级策略（S3 HTTP 客户端共享），只在启动时设置一次。
- 覆盖：`TestDenyPrivateNetworksOptIn` / `TestDenyPrivateNetworksDialGuard`（默认放行 + 开启后创建期与拨号期均拒绝）。
