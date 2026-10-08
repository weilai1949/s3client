package config

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestSplitList(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b ,c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
		{",a,", []string{"a"}},
	}
	for _, c := range cases {
		if got := splitList(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitList(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestFromEnvSSRFDenyPrivate 可选 SSRF 加固开关（roadmap §5.1 R8）：默认关闭，
// 仅显式 truthy 时开启；开启后连私网/回环端点也拒绝（s3wrap 侧生效）。
func TestFromEnvSSRFDenyPrivate(t *testing.T) {
	t.Setenv("S3C_SSRF_DENY_PRIVATE", "")
	if FromEnv().SSRFDenyPrivate {
		t.Error("默认应为关闭（ADR-003 自托管放行私网）")
	}
	t.Setenv("S3C_SSRF_DENY_PRIVATE", "1")
	if !FromEnv().SSRFDenyPrivate {
		t.Error("S3C_SSRF_DENY_PRIVATE=1 应开启")
	}
}

// TestFromEnvAllowPlaintextStore 明文落盘显式 opt-in（KNOWN_ISSUES #29/#31）：默认关闭，
// 仅显式 truthy 时开启，与其他 envTruthy 开关语义一致。
func TestFromEnvAllowPlaintextStore(t *testing.T) {
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "")
	if FromEnv().AllowPlaintextStore {
		t.Error("默认应为关闭（安全默认：明文落盘必须显式 opt-in）")
	}
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	if !FromEnv().AllowPlaintextStore {
		t.Error("S3C_ALLOW_PLAINTEXT_STORE=1 应开启")
	}
}

// TestFromEnvDefaults 验证安全默认值（回环绑定、无鉴权、无 CORS 白名单）。
func TestFromEnvDefaults(t *testing.T) {
	for _, k := range []string{"S3C_ADDR", "S3C_DATA_DIR", "S3C_STATIC_DIR", "S3C_REGION", "S3C_TOKEN", "S3C_CORS_ORIGINS", "S3C_LOG_LEVEL", "S3C_LOG_JSON", "S3C_STORE_DRIVER", "S3C_STORE_KEY", "S3C_ALLOW_PLAINTEXT_STORE", "S3C_SHUTDOWN_TIMEOUT", "S3C_EXPOSE_METRICS"} {
		t.Setenv(k, "")
	}
	cfg := FromEnv()
	if cfg.Addr != "127.0.0.1:8080" {
		t.Errorf("Addr = %q, want 127.0.0.1:8080", cfg.Addr)
	}
	if cfg.DataDir != "./data" {
		t.Errorf("DataDir = %q, want ./data", cfg.DataDir)
	}
	if cfg.StaticDir != "../web/dist" {
		t.Errorf("StaticDir = %q, want ../web/dist", cfg.StaticDir)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("Region = %q, want us-east-1", cfg.Region)
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
	if len(cfg.CORSOrigins) != 0 {
		t.Errorf("CORSOrigins = %v, want empty", cfg.CORSOrigins)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.LogJSON {
		t.Errorf("LogJSON = true, want false")
	}
	if cfg.StoreDriver != "json" {
		t.Errorf("StoreDriver = %q, want json", cfg.StoreDriver)
	}
	if cfg.ShutdownTimeoutSec != 30 {
		t.Errorf("ShutdownTimeoutSec = %d, want 30", cfg.ShutdownTimeoutSec)
	}
}

func TestFromEnvOverrides(t *testing.T) {
	t.Setenv("S3C_ADDR", "0.0.0.0:9000")
	t.Setenv("S3C_DATA_DIR", "/tmp/s3c")
	t.Setenv("S3C_STATIC_DIR", "/srv/web")
	t.Setenv("S3C_REGION", "cn-north-1")
	t.Setenv("S3C_TOKEN", "topsecret")
	t.Setenv("S3C_CORS_ORIGINS", " https://a.example, http://b.example , ")
	t.Setenv("S3C_LOG_LEVEL", "debug")
	cfg := FromEnv()
	if cfg.Addr != "0.0.0.0:9000" {
		t.Errorf("Addr = %q, want 0.0.0.0:9000", cfg.Addr)
	}
	if cfg.Token != "topsecret" {
		t.Errorf("Token = %q, want topsecret", cfg.Token)
	}
	want := []string{"https://a.example", "http://b.example"}
	if !reflect.DeepEqual(cfg.CORSOrigins, want) {
		t.Errorf("CORSOrigins = %v, want %v", cfg.CORSOrigins, want)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
}

// TestFromEnvStoreDriverNormalization R4:FromEnv 阶段归一化(TrimSpace+ToLower),
// 使 "JSON" / "  Json  " 等写法与小写完全等价——明文落盘安全闸不再被大小写/空白绕过。
func TestFromEnvStoreDriverNormalization(t *testing.T) {
	cases := []struct{ in, want string }{
		{"json", "json"},
		{"JSON", "json"},
		{"  Json  ", "json"},
		{"SQLITE", "sqlite"},
		{" Encrypted", "encrypted"},
	}
	for _, c := range cases {
		t.Setenv("S3C_STORE_DRIVER", c.in)
		if got := FromEnv().StoreDriver; got != c.want {
			t.Errorf("FromEnv(StoreDriver=%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// 关键回归:大写 json + 无 STORE_KEY + 未 opt-in 必须被明文闸拦住(R4 原缺陷)。
	t.Setenv("S3C_STORE_DRIVER", "JSON")
	t.Setenv("S3C_STORE_KEY", "")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "")
	t.Setenv("S3C_TOKEN", "")
	t.Setenv("S3C_ADDR", "")
	cfg := FromEnv()
	if err := cfg.Validate(); !errors.Is(err, ErrPlaintextStoreNotAllowed) {
		t.Fatalf("JSON driver bypassed plaintext gate: Validate() = %v, want ErrPlaintextStoreNotAllowed", err)
	}
	// 未知值:FromEnv 归一化原样保留值,Validate 拒绝启动(而非 Open 静默当 json)。
	t.Setenv("S3C_STORE_DRIVER", " pgsql ")
	cfg = FromEnv()
	if cfg.StoreDriver != "pgsql" {
		t.Fatalf("StoreDriver = %q, want normalized pgsql", cfg.StoreDriver)
	}
	if err := cfg.Validate(); !errors.Is(err, ErrUnknownStoreDriver) {
		t.Fatalf("unknown driver: Validate() = %v, want ErrUnknownStoreDriver", err)
	}
}

func TestLoadDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	content := "# comment\nS3C_TOKEN=from-dotenv\nS3C_REGION=\"cn-beijing\"\n\nBADLINE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	loadDotEnvFile(path)
	if got := os.Getenv("S3C_TOKEN"); got != "from-dotenv" {
		t.Errorf("S3C_TOKEN = %q, want from-dotenv", got)
	}
	if got := os.Getenv("S3C_REGION"); got != "cn-beijing" {
		t.Errorf("S3C_REGION = %q, want cn-beijing", got)
	}
	// 已存在的环境变量优先于 .env
	t.Setenv("S3C_TOKEN", "real-env")
	loadDotEnvFile(path)
	if got := os.Getenv("S3C_TOKEN"); got != "real-env" {
		t.Errorf("S3C_TOKEN = %q, want real-env (env wins)", got)
	}
}

// TestFromEnvOTelDefaults OTel tracing 默认关闭：Endpoint 空、采样比例 1、服务名 s3client。
func TestFromEnvOTelDefaults(t *testing.T) {
	t.Setenv("S3C_OTEL_ENDPOINT", "")
	t.Setenv("S3C_OTEL_SAMPLE_RATIO", "")
	t.Setenv("S3C_OTEL_SERVICE_NAME", "")
	// 隔离宿主环境：Validate 只看 OTel 之外的既有闸门是否通过。
	t.Setenv("S3C_TOKEN", "")
	t.Setenv("S3C_ADDR", "127.0.0.1:8080")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	cfg := FromEnv()
	if cfg.OTelEndpoint != "" {
		t.Errorf("OTelEndpoint = %q, want empty（默认关闭）", cfg.OTelEndpoint)
	}
	if cfg.OTelSampleRatio != 1 {
		t.Errorf("OTelSampleRatio = %v, want 1", cfg.OTelSampleRatio)
	}
	if cfg.OTelServiceName != "s3client" {
		t.Errorf("OTelServiceName = %q, want s3client", cfg.OTelServiceName)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

// TestFromEnvOTelOverrides 三个 OTel 变量可覆盖；采样比例接受 [0,1] 闭区间的合法值。
func TestFromEnvOTelOverrides(t *testing.T) {
	t.Setenv("S3C_OTEL_ENDPOINT", "http://collector:4318")
	t.Setenv("S3C_OTEL_SERVICE_NAME", "s3client-prod")
	t.Setenv("S3C_TOKEN", "")
	t.Setenv("S3C_ADDR", "127.0.0.1:8080")
	t.Setenv("S3C_ALLOW_PLAINTEXT_STORE", "1")
	for _, c := range []struct {
		in   string
		want float64
	}{{"0", 0}, {"0.25", 0.25}, {"1", 1}} {
		t.Setenv("S3C_OTEL_SAMPLE_RATIO", c.in)
		cfg := FromEnv()
		if cfg.OTelEndpoint != "http://collector:4318" {
			t.Errorf("OTelEndpoint = %q, want http://collector:4318", cfg.OTelEndpoint)
		}
		if cfg.OTelServiceName != "s3client-prod" {
			t.Errorf("OTelServiceName = %q, want s3client-prod", cfg.OTelServiceName)
		}
		if cfg.OTelSampleRatio != c.want {
			t.Errorf("S3C_OTEL_SAMPLE_RATIO=%q: OTelSampleRatio = %v, want %v", c.in, cfg.OTelSampleRatio, c.want)
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("S3C_OTEL_SAMPLE_RATIO=%q: Validate() = %v, want nil", c.in, err)
		}
	}
}

// TestFromEnvOTelSampleRatioInvalid 非法采样比例走既有 envErr → Validate 拒绝启动，
// 不静默回退默认值（否则运维会以为配置已生效）。
func TestFromEnvOTelSampleRatioInvalid(t *testing.T) {
	for _, v := range []string{"abc", "1.5", "-0.1", "NaN", "Inf"} {
		t.Setenv("S3C_OTEL_SAMPLE_RATIO", v)
		cfg := FromEnv()
		if err := cfg.Validate(); !errors.Is(err, ErrInvalidEnvValue) {
			t.Errorf("S3C_OTEL_SAMPLE_RATIO=%q: Validate() = %v, want ErrInvalidEnvValue", v, err)
		}
	}
}
