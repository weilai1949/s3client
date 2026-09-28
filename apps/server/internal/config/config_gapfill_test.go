package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestEnvOrInt 表驱动覆盖 envOrInt 的全部分支:未设置/空串取默认;非数字、<1
// 一律返回错误(R9:不再静默回退,让运维误以为配置已生效);合法值正常解析。
func TestEnvOrInt(t *testing.T) {
	cases := []struct {
		name    string
		val     string
		set     bool
		def     int
		want    int
		wantErr bool
	}{
		{"unset uses default", "", false, 30, 30, false},
		{"empty uses default", "", true, 30, 30, false},
		{"non numeric rejected", "abc", true, 30, 0, true},
		{"mixed rejected", "5x", true, 30, 0, true},
		{"zero rejected", "0", true, 30, 0, true},
		{"negative rejected", "-3", true, 30, 0, true},
		{"one is valid", "1", true, 30, 1, false},
		{"positive parses", "42", true, 30, 42, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.set {
				t.Setenv("S3C_TEST_TIMEOUT", c.val)
			} else {
				t.Setenv("S3C_TEST_TIMEOUT", "") // 显式置空，隔离外部环境
			}
			got, err := envOrInt("S3C_TEST_TIMEOUT", c.def)
			if (err != nil) != c.wantErr {
				t.Fatalf("envOrInt(%q, %d) err = %v, wantErr = %v", c.val, c.def, err, c.wantErr)
			}
			if err != nil && !errors.Is(err, ErrInvalidEnvValue) {
				t.Fatalf("envOrInt(%q, %d) err = %v, want wraps ErrInvalidEnvValue", c.val, c.def, err)
			}
			if got != c.want {
				t.Errorf("envOrInt(%q, %d) = %d, want %d", c.val, c.def, got, c.want)
			}
		})
	}
}

// TestFromEnvShutdownTimeout 覆盖 S3C_SHUTDOWN_TIMEOUT 的默认/合法/非法三种取值:
// 非法值不再静默回退为默认——FromEnv 记录解析失败,Validate 原样上抛,走 main 已有的
// 「配置校验失败 → 退出码 1」启动失败路径(R9)。
func TestFromEnvShutdownTimeout(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("S3C_SHUTDOWN_TIMEOUT", "")
		if got := FromEnv().ShutdownTimeoutSec; got != 30 {
			t.Errorf("ShutdownTimeoutSec = %d, want 30", got)
		}
	})
	t.Run("valid", func(t *testing.T) {
		t.Setenv("S3C_SHUTDOWN_TIMEOUT", "5")
		if got := FromEnv().ShutdownTimeoutSec; got != 5 {
			t.Errorf("ShutdownTimeoutSec = %d, want 5", got)
		}
	})
	for _, val := range []string{"abc", "0", "-3"} {
		t.Run("invalid "+val+" fails validate", func(t *testing.T) {
			t.Setenv("S3C_SHUTDOWN_TIMEOUT", val)
			cfg := FromEnv()
			if err := cfg.Validate(); !errors.Is(err, ErrInvalidEnvValue) {
				t.Fatalf("Validate() = %v, want wraps ErrInvalidEnvValue", err)
			}
		})
	}
}

// TestLoadDotEnvFileEmptyKey 键为空的行（"=value"）应被跳过；正常键照常写入环境变量。
func TestLoadDotEnvFileEmptyKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# 注释\n=v1\n   =v2\nS3C_STORE_DRIVER=sqlite\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	loadDotEnvFile(path)
	if got := os.Getenv("S3C_STORE_DRIVER"); got != "sqlite" {
		t.Errorf("S3C_STORE_DRIVER = %q, want sqlite", got)
	}
}

// TestLoadDotEnvFileMissingFile 文件不存在时应静默返回（不 panic、不改环境）。
func TestLoadDotEnvFileMissingFile(t *testing.T) {
	t.Setenv("S3C_TOKEN", "")
	loadDotEnvFile(filepath.Join(t.TempDir(), "definitely-missing.env"))
	if got := os.Getenv("S3C_TOKEN"); got != "" {
		t.Errorf("S3C_TOKEN = %q, want unchanged empty", got)
	}
}

// TestFromEnvLogJSONVariants 表驱动验证 S3C_LOG_JSON 的真值判定（大小写与取值变体）。
func TestFromEnvLogJSONVariants(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"", false},
		{"0", false},
		{"no", false},
		{"false", false},
		{"1", true},
		{"TRUE", true},
		{"Yes", true},
		{"on", true},
		{"true ", true}, // 带空格也应视为真
	}
	for _, c := range cases {
		t.Run("S3C_LOG_JSON="+c.val, func(t *testing.T) {
			t.Setenv("S3C_LOG_JSON", c.val)
			if got := FromEnv().LogJSON; got != c.want {
				t.Errorf("LogJSON(%q) = %v, want %v", c.val, got, c.want)
			}
		})
	}
}

// TestFromEnvAllFields 单条用例覆盖全部环境变量与解析结果（含 StoreKey 与 CORS 分隔）。
func TestFromEnvAllFields(t *testing.T) {
	t.Setenv("S3C_ADDR", "0.0.0.0:8081")
	t.Setenv("S3C_DATA_DIR", "/var/lib/s3c")
	t.Setenv("S3C_STATIC_DIR", "/opt/web")
	t.Setenv("S3C_REGION", "cn-hangzhou")
	t.Setenv("S3C_TOKEN", "tk")
	t.Setenv("S3C_CORS_ORIGINS", "http://a,http://b")
	t.Setenv("S3C_LOG_LEVEL", "warn")
	t.Setenv("S3C_LOG_JSON", "on")
	t.Setenv("S3C_STORE_DRIVER", "sqlite")
	t.Setenv("S3C_STORE_KEY", "k3y")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "on")
	t.Setenv("S3C_SHUTDOWN_TIMEOUT", "12")
	cfg := FromEnv()
	if cfg.Addr != "0.0.0.0:8081" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.DataDir != "/var/lib/s3c" || cfg.StaticDir != "/opt/web" || cfg.Region != "cn-hangzhou" {
		t.Errorf("basic fields = %+v", cfg)
	}
	if cfg.Token != "tk" || cfg.StoreKey != "k3y" || cfg.StoreDriver != "sqlite" {
		t.Errorf("auth/store fields = %+v", cfg)
	}
	if !cfg.AllowPlaintextStore {
		t.Errorf("AllowPlaintextStore = false, want true (S3C_ALLOW_PLAINTEXT_STORE=on)")
	}
	if cfg.LogLevel != "warn" || !cfg.LogJSON {
		t.Errorf("log fields = %q/%v", cfg.LogLevel, cfg.LogJSON)
	}
	if cfg.ShutdownTimeoutSec != 12 {
		t.Errorf("ShutdownTimeoutSec = %d", cfg.ShutdownTimeoutSec)
	}
	if !reflect.DeepEqual(cfg.CORSOrigins, []string{"http://a", "http://b"}) {
		t.Errorf("CORSOrigins = %v", cfg.CORSOrigins)
	}
	// S3C_EXPOSE_METRICS 缺省 false；S3C_EXPOSE_METRICS=1 显式开启。
	t.Setenv("S3C_EXPOSE_METRICS", "")
	if FromEnv().ExposeMetrics {
		t.Errorf("ExposeMetrics default = true, want false")
	}
	t.Setenv("S3C_EXPOSE_METRICS", "1")
	if !FromEnv().ExposeMetrics {
		t.Errorf("ExposeMetrics with 1 = false, want true")
	}
	t.Setenv("S3C_EXPOSE_METRICS", "")
}
