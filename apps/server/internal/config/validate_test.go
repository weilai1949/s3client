package config

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{"loopback no token ok", Config{Addr: "127.0.0.1:8080", Token: ""}, nil},
		{"empty host no token ok", Config{Addr: ":8080", Token: ""}, nil},
		{"non-loopback no token rejected", Config{Addr: "0.0.0.0:8080", Token: ""}, ErrTokenRequiredNonLoopback},
		{"non-loopback with token ok", Config{Addr: "0.0.0.0:8080", Token: strings.Repeat("a", MinTokenLength)}, nil},
		{"short token rejected (loopback)", Config{Addr: "127.0.0.1:8080", Token: "short"}, ErrShortToken},
		{"short token rejected (non-loopback)", Config{Addr: "0.0.0.0:8080", Token: "short"}, ErrShortToken},
		{"multi token shortest applies", Config{Addr: "127.0.0.1:8080", Token: strings.Repeat("a", MinTokenLength) + ",short"}, ErrShortToken},
		{"multi token all long ok", Config{Addr: "127.0.0.1:8080", Token: strings.Repeat("a", MinTokenLength) + "," + strings.Repeat("b", MinTokenLength+5)}, nil},
		{"empty token piece ignored in shortest", Config{Addr: "127.0.0.1:8080", Token: "," + strings.Repeat("a", MinTokenLength)}, nil},
		// S3C_STORE_KEY 最短长度校验（已闭环：features.md §M）：短口令会被 Argon2 暴力破解。
		{"short store key rejected", Config{Addr: "127.0.0.1:8080", StoreKey: "short"}, ErrShortStoreKey},
		{"short store key rejected (encrypted)", Config{Addr: "127.0.0.1:8080", StoreDriver: "encrypted", StoreKey: "short"}, ErrShortStoreKey},
		// 安全默认：未显式选择驱动（空串）不触发明文落盘校验，由 json/sqlite 用例单独覆盖。
		{"empty store key ok (no driver)", Config{Addr: "127.0.0.1:8080", StoreKey: ""}, nil},
		{"long store key ok", Config{Addr: "127.0.0.1:8080", StoreKey: strings.Repeat("k", MinStoreKeyLength)}, nil},
		{"json empty key with opt-in ok", Config{Addr: "127.0.0.1:8080", StoreDriver: "json", AllowPlaintextStore: true}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.Validate()
			if c.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() err=%v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want %v", c.wantErr)
			}
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("Validate() err=%v, want wraps %v", err, c.wantErr)
			}
		})
	}
}

// TestValidatePlaintextStoreRejected 安全默认（todolist #29/#31）：json / sqlite 且
// S3C_STORE_KEY 为空时，除非显式 AllowPlaintextStore，否则 Validate 必须硬失败，
// 并给出两条出路（设 S3C_STORE_KEY / 显式 opt-in）。encrypted 与非空 key 不受影响。
func TestValidatePlaintextStoreRejected(t *testing.T) {
	key := strings.Repeat("k", MinStoreKeyLength)
	cases := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{"json empty key rejected", Config{Addr: "127.0.0.1:8080", StoreDriver: "json"}, ErrPlaintextStoreNotAllowed},
		{"sqlite empty key rejected", Config{Addr: "127.0.0.1:8080", StoreDriver: "sqlite"}, ErrPlaintextStoreNotAllowed},
		{"json opt-in allowed", Config{Addr: "127.0.0.1:8080", StoreDriver: "json", AllowPlaintextStore: true}, nil},
		{"sqlite opt-in allowed", Config{Addr: "127.0.0.1:8080", StoreDriver: "sqlite", AllowPlaintextStore: true}, nil},
		{"json with key allowed", Config{Addr: "127.0.0.1:8080", StoreDriver: "json", StoreKey: key}, nil},
		{"sqlite with key allowed", Config{Addr: "127.0.0.1:8080", StoreDriver: "sqlite", StoreKey: key}, nil},
		{"encrypted empty key not this sentinel", Config{Addr: "127.0.0.1:8080", StoreDriver: "encrypted"}, nil},
		{"unknown driver allowed", Config{Addr: "127.0.0.1:8080", StoreDriver: "pgsql"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.Validate()
			if c.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() err=%v, want nil", err)
				}
				return
			}
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("Validate() err=%v, want wraps %v", err, c.wantErr)
			}
			// 报错必须告诉运维两条出路，且不得泄露任何密钥值。
			for _, want := range []string{"S3C_STORE_KEY", "S3C_ALLOW_PLAINTEXT_STORE", "openssl rand -hex 32"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Validate() err=%q missing %q", err.Error(), want)
				}
			}
		})
	}
}

// TestStorePlaintextWarning 落盘明文告警：json / sqlite 无 key 时给出可执行提示，
// 加密配置（encrypted，或任意驱动 + 非空 key）不告警（roadmap §5.1 R3）。
func TestStorePlaintextWarning(t *testing.T) {
	key := strings.Repeat("k", MinStoreKeyLength)
	cases := []struct {
		name      string
		cfg       Config
		wantWarn  bool
		wantParts []string
	}{
		{"json without key warns", Config{StoreDriver: "json", DataDir: "./data"}, true,
			[]string{"json", "./data", "encrypted", "S3C_STORE_KEY"}},
		{"sqlite without key warns", Config{StoreDriver: "sqlite", DataDir: "/tmp/d"}, true,
			[]string{"sqlite", "/tmp/d"}},
		{"json with key silent", Config{StoreDriver: "json", StoreKey: key}, false, nil},
		{"sqlite with key silent", Config{StoreDriver: "sqlite", StoreKey: key}, false, nil},
		{"encrypted silent", Config{StoreDriver: "encrypted"}, false, nil},
		{"unknown driver silent", Config{StoreDriver: "pgsql"}, false, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.cfg.StorePlaintextWarning()
			if !c.wantWarn {
				if got != "" {
					t.Fatalf("StorePlaintextWarning() = %q, want empty", got)
				}
				return
			}
			if got == "" {
				t.Fatal("StorePlaintextWarning() = empty, want warning")
			}
			for _, p := range c.wantParts {
				if !strings.Contains(got, p) {
					t.Fatalf("warning %q missing %q", got, p)
				}
			}
		})
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"::1:8080", true},
		{"[::1]:8080", true},
		{":8080", true},
		{"0.0.0.0:8080", false},
		{"192.168.1.1:8080", false},
		{"localhost:8080", false}, // 非 IP 视为非回环
	}
	for _, c := range cases {
		if got := IsLoopbackAddr(c.addr); got != c.want {
			t.Errorf("IsLoopbackAddr(%q) = %v, want %v", c.addr, got, c.want)
		}
	}
}
