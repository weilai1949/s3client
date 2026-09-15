package config

import (
	"os"
	"path/filepath"
	"testing"
)

// unsetEnv 清空某环境变量并在测试结束后恢复（t.Setenv 只能设值，无法 unset）。
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

// TestEnvFileCandidates：显式 S3C_ENV_FILE 优先且唯一；未设置时以 CWD .env 为首选，
// 并附带可执行文件同目录 .env 作为回退（解决「按进程 CWD 相对加载」在服务/双击启动下失效）。
func TestEnvFileCandidates(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit.env")
	t.Setenv("S3C_ENV_FILE", explicit)
	if got := envFileCandidates(); len(got) != 1 || got[0] != explicit {
		t.Fatalf("explicit candidates = %v, want [%s]", got, explicit)
	}

	t.Setenv("S3C_ENV_FILE", "  ")
	got := envFileCandidates()
	if len(got) == 0 || got[0] != ".env" {
		t.Fatalf("default candidates = %v, want first entry .env", got)
	}
	if len(got) < 2 || filepath.Base(got[1]) != ".env" || filepath.Dir(got[1]) == "." {
		t.Fatalf("default candidates = %v, want executable-dir .env fallback", got)
	}
}

// TestFromEnvHonorsExplicitEnvFile 显式 S3C_ENV_FILE 被读取（与进程 CWD 解耦）。
func TestFromEnvHonorsExplicitEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.env")
	if err := os.WriteFile(path, []byte("S3C_LOG_LEVEL=debug\nS3C_SHUTDOWN_TIMEOUT=99\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	unsetEnv(t, "S3C_LOG_LEVEL")
	unsetEnv(t, "S3C_SHUTDOWN_TIMEOUT")
	t.Setenv("S3C_ENV_FILE", path)

	cfg := FromEnv()
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug (from explicit env file)", cfg.LogLevel)
	}
	if cfg.ShutdownTimeoutSec != 99 {
		t.Errorf("ShutdownTimeoutSec = %d, want 99 (from explicit env file)", cfg.ShutdownTimeoutSec)
	}
}

// TestFromEnvExplicitEnvFileMissingUsesDefaults 显式路径不存在时不静默回退到 CWD .env，
// 而是退回内置默认值（显式配置就是唯一来源，避免加载到非预期的文件）。
func TestFromEnvExplicitEnvFileMissingUsesDefaults(t *testing.T) {
	unsetEnv(t, "S3C_LOG_LEVEL")
	t.Setenv("S3C_ENV_FILE", filepath.Join(t.TempDir(), "definitely-missing.env"))
	if got := FromEnv().LogLevel; got != "info" {
		t.Fatalf("LogLevel = %q, want info", got)
	}
}

// TestFromEnvRealEnvWinsOverExplicitEnvFile 真实环境变量仍优先于 .env 文件。
func TestFromEnvRealEnvWinsOverExplicitEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.env")
	if err := os.WriteFile(path, []byte("S3C_LOG_LEVEL=debug\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv("S3C_LOG_LEVEL", "warn")
	t.Setenv("S3C_ENV_FILE", path)
	if got := FromEnv().LogLevel; got != "warn" {
		t.Fatalf("LogLevel = %q, want warn (real env wins)", got)
	}
}
