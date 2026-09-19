package s3wrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

var (
	errRedirectDenied  = errors.New("HTTP redirect denied (SSRF protection)")
	errEndpointBlocked = errors.New("endpoint host is blocked (link-local / cloud metadata)")
)

// denyPrivateNetworks 是 S3C_SSRF_DENY_PRIVATE 的进程级开关：默认 false，保留 ADR-003 的
// 自托管主场景（MinIO / RustFS / 局域网放行）；开启后私网、回环与未指定地址一并拒绝。
// S3 HTTP 客户端是进程级共享的，因此策略按进程生效，只在启动时设置一次。
var denyPrivateNetworks atomic.Bool

// SetDenyPrivateNetworks 设置「拒绝私网 / 回环端点」策略（只应在进程启动时调用一次）。
func SetDenyPrivateNetworks(v bool) { denyPrivateNetworks.Store(v) }

// DenyPrivateNetworks 返回当前生效策略，供启动日志与 /api/metrics 反映实际值（便于运维核对部署配置）。
func DenyPrivateNetworks() bool { return denyPrivateNetworks.Load() }

// ValidateEndpoint 校验账号 Endpoint / PublicEndpoint：禁止指向云元数据与链路本地地址。
// 私网 / 回环（MinIO、RustFS、局域网）默认允许，因自托管是主场景；SSRF 主防线是鉴权 + 禁重定向 + 禁 IMDS。
// 需要更严策略时用 SetDenyPrivateNetworks(true)（S3C_SSRF_DENY_PRIVATE=1）连私网一并拒绝。
func ValidateEndpoint(endpoint string) error {
	// 空值代表「使用 S3 默认端点」，合法豁免。
	if strings.TrimSpace(endpoint) == "" {
		return nil
	}
	// 归一化后再解析：用户手填的 endpoint 可能带首尾空白或大小写 scheme，
	// 直接用原串 url.Parse 会把合法地址误判为非法（features.md §K，原 todolist #10）。
	raw := NormalizeEndpoint(endpoint, false)
	if raw == "" {
		// 非空输入却归一化为空（如 "http://"、"/"）：退化地址必须 fail-closed，
		// 不能因归一化而静默放行。
		return fmt.Errorf("invalid endpoint URL: %q", endpoint)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	host := strings.ToLower(u.Hostname())
	if isBlockedHostname(host) {
		return fmt.Errorf("%w: %s", errEndpointBlocked, host)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		// DNS 失败留给后续连接阶段；此处仅拦已解析到的危险地址。
		return nil
	}
	for _, ipa := range ips {
		if isBlockedIP(ipa.IP) {
			return fmt.Errorf("%w: %s -> %s", errEndpointBlocked, host, ipa.IP)
		}
	}
	return nil
}

func isBlockedHostname(host string) bool {
	switch host {
	case "metadata.google.internal", "metadata", "metadata.goog",
		"instance-data", "kubernetes.default", "kubernetes.default.svc":
		return true
	}
	return false
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// 常见云元数据（非链路本地段）：
	//   100.100.100.200 / 100.96.0.2 —— 阿里云 IMDS 及火山引擎内网元数据
	//   fd00:ec2::254 —— AWS IMDS IPv6 端点（unique-local，非 link-local）
	// 只拦精确地址；fd00::/8 整段放行以支持自托管 IPv6 ULA 主场景。
	blocked := []string{"100.100.100.200", "100.96.0.2", "fd00:ec2::254"}
	for _, s := range blocked {
		if ip.Equal(net.ParseIP(s)) {
			return true
		}
	}
	// S3C_SSRF_DENY_PRIVATE=1：连自托管私网/回环也拒绝（可选加固，默认关闭见 ADR-003）。
	if denyPrivateNetworks.Load() && (ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified()) {
		return true
	}
	return false
}

// dialContextSSRF 在拨号前二次校验目标 IP，防止 DNS 重绑定绕过创建时校验。
func dialContextSSRF(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var last error
	d := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	for _, ipa := range ips {
		if isBlockedIP(ipa.IP) {
			last = fmt.Errorf("%w: %s", errEndpointBlocked, ipa.IP)
			continue
		}
		var target = net.JoinHostPort(ipa.IP.String(), port)
		conn, err := d.DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
		last = err
	}
	// dialContextSSRF 仅在「DNS/连接失败时 err != nil」分支被调用；last 必非 nil。
	return nil, last
}

func checkRedirectDenied(_ *http.Request, _ []*http.Request) error {
	return errRedirectDenied
}
