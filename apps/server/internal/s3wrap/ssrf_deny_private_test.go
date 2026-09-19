package s3wrap

import (
	"context"
	"errors"
	"testing"
)

// TestDenyPrivateNetworksOptIn S3C_SSRF_DENY_PRIVATE 的语义（roadmap §5.1 R8）：
// 默认放行私网/回环（ADR-003 自托管主场景），显式开启后创建期校验必须拒绝。
func TestDenyPrivateNetworksOptIn(t *testing.T) {
	t.Cleanup(func() { SetDenyPrivateNetworks(false) })

	SetDenyPrivateNetworks(false)
	if DenyPrivateNetworks() {
		t.Fatal("默认策略应为放行私网（ADR-003）")
	}
	for _, ep := range []string{
		"http://127.0.0.1:9000", "http://192.168.1.10:9000", "http://10.0.0.5:9000", "http://[::1]:9000",
	} {
		if err := ValidateEndpoint(ep); err != nil {
			t.Fatalf("默认放行下 ValidateEndpoint(%q) = %v, want nil（ADR-003）", ep, err)
		}
	}

	SetDenyPrivateNetworks(true)
	if !DenyPrivateNetworks() {
		t.Fatal("SetDenyPrivateNetworks(true) 后 getter 应返回 true")
	}
	for _, ep := range []string{
		"http://127.0.0.1:9000", "http://192.168.1.10:9000", "http://10.0.0.5:9000",
		"http://172.16.0.1:9000", "http://[::1]:9000", "http://0.0.0.0:9000",
	} {
		err := ValidateEndpoint(ep)
		if err == nil {
			t.Fatalf("开启后 ValidateEndpoint(%q) = nil, want blocked", ep)
		}
		if !errors.Is(err, errEndpointBlocked) {
			t.Fatalf("ValidateEndpoint(%q) err = %v, want errEndpointBlocked", ep, err)
		}
	}
	if err := ValidateEndpoint("http://8.8.8.8:9000"); err != nil {
		t.Fatalf("公网地址不应被拒: %v", err)
	}
}

// TestDenyPrivateNetworksDialGuard 拨号期二次校验同样遵守开关（防 DNS 重绑定绕过创建期校验）。
func TestDenyPrivateNetworksDialGuard(t *testing.T) {
	t.Cleanup(func() { SetDenyPrivateNetworks(false) })

	SetDenyPrivateNetworks(true)
	if _, err := dialContextSSRF(context.Background(), "tcp", "127.0.0.1:1"); !errors.Is(err, errEndpointBlocked) {
		t.Fatalf("dialContextSSRF(127.0.0.1) err = %v, want errEndpointBlocked", err)
	}

	SetDenyPrivateNetworks(false)
	if _, err := dialContextSSRF(context.Background(), "tcp", "127.0.0.1:1"); err == nil || errors.Is(err, errEndpointBlocked) {
		t.Fatalf("默认放行下 dialContextSSRF 应返回连接错误而非 SSRF 拦截，got %v", err)
	}
}
