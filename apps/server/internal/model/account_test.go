package model

import "testing"

func TestSanitizedMasksSecret(t *testing.T) {
	a := &Account{ID: "1", Name: "x", SecretKey: "super-secret"}
	s := a.Sanitized()
	if s.SecretKey != MaskedSecret {
		t.Fatalf("secret = %q", s.SecretKey)
	}
	if a.SecretKey != "super-secret" {
		t.Fatal("original mutated")
	}
}

func TestIsMaskedSecret(t *testing.T) {
	if !IsMaskedSecret(MaskedSecret) {
		t.Fatal("expected masked")
	}
	if IsMaskedSecret("real-key") {
		t.Fatal("expected not masked")
	}
}

func TestSanitizedNil(t *testing.T) {
	if (*Account)(nil).Sanitized() != nil {
		t.Fatal("nil account should return nil")
	}
}

func TestViewSecretSet(t *testing.T) {
	cases := []struct {
		name string
		acc  *Account
		want bool
	}{
		{"real secret", &Account{ID: "1", Name: "x", SecretKey: "sk"}, true},
		{"masked secret", &Account{ID: "2", SecretKey: MaskedSecret}, true},
		{"empty secret", &Account{ID: "3"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := c.acc.View()
			if v == nil {
				t.Fatal("View returned nil")
			}
			if v.SecretSet != c.want {
				t.Fatalf("secretSet = %v, want %v", v.SecretSet, c.want)
			}
			if v.ID != c.acc.ID || v.Name != c.acc.Name {
				t.Fatalf("view fields not copied: %+v", v)
			}
		})
	}
}

func TestViewNil(t *testing.T) {
	if (*Account)(nil).View() != nil {
		t.Fatal("nil account View should return nil")
	}
}

// TestViewRedactsEndpointUserinfo —— P-3（2026-10-15 渗透测试）：`AccountView` 的
// endpoint / publicEndpoint 必须回传**脱敏值**，防御契约上线前落盘的历史数据
// （`http://key:secret@host` 会被任意已认证 token 读到）。
//
// 断言的是外部可见返回值（DTO 字段）而非私有实现：带 scheme / 裸端点、有无 userinfo、
// 解析失败 6 种形态；解析失败用 `http://%zz:9000`（invalid URL escape）与 `http://[::1`
// （missing ']' in host）两条**确定性**触发 url.Parse 错误的输入钉住「原样返回」。
func TestViewRedactsEndpointUserinfo(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		want     string
	}{
		{"带 scheme + userinfo", "http://AKIAUSER:URLPASS@minio.local:9000", "http://minio.local:9000"},
		{"仅用户名", "http://AKIAUSER@minio.local:9000", "http://minio.local:9000"},
		{"带 scheme 无 userinfo", "http://minio.local:9000", "http://minio.local:9000"},
		{"裸端点 + userinfo", "AKIAUSER:URLPASS@minio.local:9000", "minio.local:9000"},
		{"裸端点无 userinfo", "minio.local:9000", "minio.local:9000"},
		{"空端点", "", ""},
		{"解析失败（非法转义）", "http://%zz:9000", "http://%zz:9000"},
		{"解析失败（未闭合 IPv6）", "http://[::1", "http://[::1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := (&Account{ID: "1", Endpoint: c.endpoint, PublicEndpoint: c.endpoint}).View()
			if v.Endpoint != c.want {
				t.Errorf("View().Endpoint = %q, want %q", v.Endpoint, c.want)
			}
			if v.PublicEndpoint != c.want {
				t.Errorf("View().PublicEndpoint = %q, want %q", v.PublicEndpoint, c.want)
			}
		})
	}
}
