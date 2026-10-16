package config

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MinTokenLength 是 S3C_TOKEN 允许的最小字符数。短口令易被暴力猜测，
// 启动时硬失败（生产推荐 openssl rand -hex 32 / 64）。
const MinTokenLength = 16

// MinMetricsTokenLength 是 S3C_METRICS_TOKEN 允许的最小字符数。
// 它保护运行指标端点，强度口径与 S3C_TOKEN 一致。
const MinMetricsTokenLength = 16

// MinStoreKeyLength 是 S3C_STORE_KEY 允许的最小字符数。该口令是账号密钥落盘加密
// 的唯一凭据（Argon2id 派生），过短同样会被暴力破解；非空即校验（已闭环：FEATURES.md §M）。
const MinStoreKeyLength = 16

// DefaultRateLimitRPM / DefaultRateLimitBurst 是 API 限流默认值（S3C_RATE_LIMIT_* 未设置时）。
// 放在 config 包供 handler.New 共用，避免两处字面量漂移。
const (
	DefaultRateLimitRPM   = 120
	DefaultRateLimitBurst = 30
)

// DefaultProbeRateLimitRPM / DefaultProbeRateLimitBurst 是探针端点
// （/api/health、/api/metrics、/api/openapi.json）独立令牌桶的默认值。
// 这三个端点豁免主限流（K8s 探针 / Prometheus scrape 不能被业务限流打成失败），
// 但匿名且会被高频轮询，故用更宽松的独立桶约束（P-12）。
const (
	DefaultProbeRateLimitRPM   = 600
	DefaultProbeRateLimitBurst = 100
)

// DefaultHealthPingCacheSeconds 是 store.Ping 结果 TTL 缓存的默认秒数（P-12）：
// 探针高频轮询时避免每次都打存储；0 = 关闭缓存（每次实打）。
const DefaultHealthPingCacheSeconds = 2

// allowedCSPSourceChars 是 CSP source-list 允许出现的非字母数字字符：
// RFC 9110 URI 安全字符集 + 单引号（关键字）与空格（source 分隔符）。
// `;`（指令注入）、`,`、双引号、反斜杠等一律不在集内（P-5a）。
const allowedCSPSourceChars = "-._~:/?#[]@!$&'()*+=%^*'"

// ErrShortToken 表示 S3C_TOKEN 长度低于 MinTokenLength。
var ErrShortToken = errors.New("S3C_TOKEN too short")

// ErrShortMetricsToken 表示 S3C_METRICS_TOKEN 长度低于 MinMetricsTokenLength。
var ErrShortMetricsToken = errors.New("S3C_METRICS_TOKEN too short")

// ErrShortStoreKey 表示 S3C_STORE_KEY 长度低于 MinStoreKeyLength。
var ErrShortStoreKey = errors.New("S3C_STORE_KEY too short")

// ErrTokenRequiredNonLoopback 表示非回环监听必须设置 S3C_TOKEN。
var ErrTokenRequiredNonLoopback = errors.New("S3C_TOKEN required for non-loopback listen address")

// ErrPlaintextStoreNotAllowed 表示 json / sqlite 驱动在未设置 S3C_STORE_KEY 时会明文落盘
// secretKey，而运维并未显式选择明文（S3C_ALLOW_PLAINTEXT_STORE=1）。安全默认：拒绝启动。
var ErrPlaintextStoreNotAllowed = errors.New("plaintext account store not allowed")

// ErrUnknownStoreDriver 表示 S3C_STORE_DRIVER 不在白名单 json|sqlite|encrypted 内。
// 未知值曾被 store.Open 静默当 json 处理，绕过明文落盘安全闸，现在直接拒绝启动。
var ErrUnknownStoreDriver = errors.New("unknown store driver")

// ErrInvalidEnvValue 表示数值型环境变量（如 S3C_SHUTDOWN_TIMEOUT）取非法值。
// 静默回退默认值会让运维误以为配置已生效，改为拒绝启动。
var ErrInvalidEnvValue = errors.New("invalid numeric environment variable")

// ErrSecretFileUnreadable 表示 <X>_FILE（S3C_STORE_KEY_FILE / S3C_TOKEN_FILE）指向的口令
// 文件不存在 / 不可读，或读出的内容去掉首尾空白后为空。
// 独立于 ErrInvalidEnvValue：文件类失败不是「数值写错」，混用会把运维引向错误的排查方向
// （实测文案曾显示为「invalid numeric environment variable: S3C_STORE_KEY_FILE=… 读取失败」）。
var ErrSecretFileUnreadable = errors.New("unreadable secret file")

// ErrSecretFileAmbiguous 表示同一口令的明文变量与 <X>_FILE 同时（非空）设置——配置歧义（ambiguous）。
// 取任一个都可能锁死或放开不该放开的口令，故 fail-closed 拒绝启动，不静默取其一。
var ErrSecretFileAmbiguous = errors.New("ambiguous secret configuration")

// ErrInvalidCSPValue 表示 S3C_CSP_CONNECT_SRC 不是合法的 CSP source-list
// （含控制字符 / 纯空白 / 允许字符集之外的字符）。
// 独立于 ErrInvalidEnvValue：这是**字符串语法**校验，复用「invalid numeric …」文案会把
// 运维引向「是不是数字写错了」的错误排查方向（Lead 端到端实测踩到）。
var ErrInvalidCSPValue = errors.New("invalid CSP source-list")

// ErrInvalidEnvFile 表示显式指定的 S3C_ENV_FILE 不存在或不可读。
// 显式路径是运维的显式选择：静默回退默认值会让写在该文件里的加固项静默失效
// （或让 S3C_DATA_DIR 静默丢失导致账号列表「凭空清空」），与上面同口径 fail-closed。
var ErrInvalidEnvFile = errors.New("unreadable explicit env file")

// maxShutdownTimeoutSec 是 S3C_SHUTDOWN_TIMEOUT 的上界（秒）。
// main.go 以 time.Duration(n) * time.Second 换算，n 超过 MaxInt64/1e9（≈92.2 亿）
// 会溢出成负时长 ⇒ 关停保护窗口被静默清零。3600 秒远超任何合理关停窗口，且远离溢出点。
const maxShutdownTimeoutSec = 3600

// normalizeStoreDriver 归一化驱动名（去空白 + 小写）。FromEnv / Validate /
// StorePlaintextWarning 三处共用，保证 "JSON" / " json " 与小写行为完全一致——
// 任何一处漏归一化都会重新打开 R4 的安全闸绕过口子。
func normalizeStoreDriver(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// loadDotEnvFile 从指定路径加载 KEY=VALUE 到环境变量（已存在的环境变量优先）。
// 支持注释行、引号与空行；实现为 30 行的极简解析器，不引入第三方依赖。
// 返回读取失败的错误，由调用方决定是否 fail-closed（显式 S3C_ENV_FILE 必须拒绝启动）。
func loadDotEnvFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
	return nil
}

// envFileCandidates 返回 .env 的候选路径（按优先级）：
//  1. 显式 S3C_ENV_FILE（部署可用绝对路径，与进程 CWD 解耦；设置后即为唯一来源）；
//  2. 进程工作目录 .env（历史默认，本地开发习惯）；
//  3. 可执行文件同目录 .env（systemd / 双击启动时 CWD 往往不是安装目录）。
func envFileCandidates() []string {
	if p := strings.TrimSpace(os.Getenv("S3C_ENV_FILE")); p != "" {
		return []string{p}
	}
	out := []string{".env"}
	// os.Executable 在极少数平台可能失败；失败时只保留 CWD 候选。
	if exe, err := os.Executable(); err == nil {
		if p := filepath.Join(filepath.Dir(exe), ".env"); p != ".env" {
			out = append(out, p)
		}
	}
	return out
}

// loadDotEnv 加载第一个存在的候选 .env 文件。
//
// 未显式指定 S3C_ENV_FILE 时保持零配置可启动：候选缺失就跳过（本地开发常态，
// 非显式路径不该有 fail-closed）。显式指定时（envFileCandidates 此时只有一项）
// **必须 fail-closed**——文件缺失 / 不可读一律返回 ErrInvalidEnvFile，否则写在
// 该文件里的加固项（S3C_TOKEN、S3C_SSRF_DENY_PRIVATE、S3C_TRUSTED_PROXIES、
// S3C_CSP_CONNECT_SRC…）会静默失效，S3C_DATA_DIR 更会静默丢失，启动后账号列表
// 「凭空清空」（原数据其实没坏，极易引发重复建号）。
func loadDotEnv() error {
	explicit := strings.TrimSpace(os.Getenv("S3C_ENV_FILE")) != ""
	for _, p := range envFileCandidates() {
		if _, err := os.Stat(p); err != nil {
			if explicit {
				return fmt.Errorf("%w: %s 不存在或不可读", ErrInvalidEnvFile, p)
			}
			continue
		}
		if err := loadDotEnvFile(p); err != nil && explicit {
			return fmt.Errorf("%w: %s 读取失败: %v", ErrInvalidEnvFile, p, err)
		}
		return nil
	}
	return nil
}

// Config 汇总服务端配置。所有项均可通过环境变量覆盖，并内置安全默认值。
type Config struct {
	Addr                string                // 监听地址，默认回环 127.0.0.1:5000（更安全）
	DataDir             string                // 数据目录，存放账号持久化文件
	StaticDir           string                // Web 静态资源目录
	Region              string                // 账号缺省 region
	Token               string                // 可选 API 鉴权 token；非空则要求 Bearer
	TokenScopes         map[string]TokenScope // S3C_TOKEN_SCOPES：按 token 声明最小权限（未登记 = 全权）
	MetricsToken        string                // S3C_METRICS_TOKEN：设置后 /api/metrics 仅接受该独立 Bearer
	CORSOrigins         []string              // CORS 白名单；空 = 仅同源 + localhost/tauri
	LogLevel            string                // debug|info|warn|error
	LogJSON             bool                  // true = slog JSON（容器/生产更易采集）
	StoreDriver         string                // json|sqlite|encrypted，账号存储后端（FromEnv 归一化为小写）
	StoreKey            string                // encrypted 模式必填；Argon2id+盐派生（仅 S3C2）
	AllowPlaintextStore bool                  // true = 显式允许 json/sqlite 无 StoreKey 明文落盘（S3C_ALLOW_PLAINTEXT_STORE=1，仅本地）
	ShutdownTimeoutSec  int                   // SIGTERM 后等待活跃连接结束的最长时间（秒）
	ExposeMetrics       bool                  // true = 暴露 /api/metrics（Prometheus 文本）；默认 false，避免公网信息泄露
	ExposeOpenAPI       bool                  // true = 暴露 /api/openapi.json（API 契约）；默认 false，避免公网泄露端点信息
	CSPConnectSrc       string                // CSP connect-src 白名单；默认仅同源 + 本地 Tauri 后端；多后端/远程需显式放宽
	TrustedProxies      []string              // 可信反向代理 IP；仅这些对端的 X-Forwarded-For 被采信（默认空 = 不信任 XFF）
	SSRFDenyPrivate     bool                  // true = 连私网/回环端点也拒绝（默认 false：自托管场景放行，见 ADR-003）
	RateLimitRPM        int                   // 每 IP（IPv6 按 /64 聚合）每分钟放行请求数；默认 DefaultRateLimitRPM
	RateLimitBurst      int                   // 令牌桶突发容量；默认 DefaultRateLimitBurst
	ProbeRateLimitRPM   int                   // 探针端点（health/metrics/openapi）独立桶每分钟放行数；默认 DefaultProbeRateLimitRPM
	ProbeRateLimitBurst int                   // 探针端点独立桶突发容量；默认 DefaultProbeRateLimitBurst
	HealthPingCacheSec  int                   // store.Ping 结果 TTL 缓存秒数；默认 DefaultHealthPingCacheSeconds，0 = 每次实打
	OTelEndpoint        string                // OTLP/HTTP 采集端基址；空 = 关闭 tracing（默认）
	OTelSampleRatio     float64               // 新建 trace 采样比例，取值 [0,1]（默认 1）
	OTelServiceName     string                // OTLP resource attribute service.name（默认 s3client）

	// envErr 记录 FromEnv 阶段的环境变量解析失败（见 envOrInt）。FromEnv 无错误返回值、
	// main 只接「FromEnv → Validate → 失败即退出码 1」这条既有链路，故错误在此暂存、
	// 由 Validate 首查上抛，实现「非法值 → 启动失败」而不改调用方。
	envErr error
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envOrInt 解析正整数环境变量：未设置（空串）取默认值；非数字或 <1 返回错误，
// 不再静默回退（否则运维会以为超时等配置已实际生效）。
func envOrInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%w: %s=%q 不是合法整数", ErrInvalidEnvValue, key, v)
	}
	if n < 1 {
		return 0, fmt.Errorf("%w: %s=%q 必须是 >=1 的整数", ErrInvalidEnvValue, key, v)
	}
	return n, nil
}

// envOrFloat 解析 [0,1] 区间的浮点环境变量（OTel 采样比例）：未设置（空串）取默认值；
// 非数字 / NaN / 越界返回错误，不静默回退（否则运维会以为采样比例已生效）。
func envOrFloat(key string, def float64) (float64, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s=%q 不是合法数字", ErrInvalidEnvValue, key, v)
	}
	if math.IsNaN(f) || f < 0 || f > 1 {
		return 0, fmt.Errorf("%w: %s=%q 必须在 [0,1] 区间", ErrInvalidEnvValue, key, v)
	}
	return f, nil
}

// envOrNonNegInt 解析**非负**整数环境变量：未设置（空串）取默认值；非数字或 <0 返回错误。
// 与 envOrInt 的唯一区别是 0 合法（S3C_HEALTH_PING_CACHE_SECONDS=0 = 关闭 Ping 缓存），
// 故不复用后者的 >=1 约束，也不静默回退。
func envOrNonNegInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%w: %s=%q 不是合法整数", ErrInvalidEnvValue, key, v)
	}
	if n < 0 {
		return 0, fmt.Errorf("%w: %s=%q 必须是 >=0 的整数", ErrInvalidEnvValue, key, v)
	}
	return n, nil
}

// envOrFile 解析「明文变量 或 <fileKey> 文件」二选一形式的密钥配置
// （S3C_STORE_KEY / S3C_STORE_KEY_FILE、S3C_TOKEN / S3C_TOKEN_FILE，P-8）：
// 容器编排（docker secrets）把密钥挂成文件，避免明文出现在 docker inspect /
// /proc/<pid>/environ。
//
//   - 未设置 fileKey 时原样返回明文变量（历史语义不变，含空串）；
//   - 设置了 fileKey 时读文件，内容去掉首尾空白与换行后作为值；
//   - 明文变量（去空白后**非空**）与 fileKey 同时设置 = 配置歧义，fail-closed
//     拒绝启动（不得静默取其一；「存在但为空串」不算设置，避免显式传空值的编排被误杀）；
//   - 文件不可读、或内容去空白后为空 = fail-closed（静默空值会让加固项失效、
//     甚至变成无鉴权启动）。
//
// 三类失败都 fail-closed 拒绝启动，且各自返回语义正确的 sentinel：文件不可读 / 内容为空 →
// ErrSecretFileUnreadable；明文与文件同时设置 → ErrSecretFileAmbiguous。错误串只含**变量名与
// 路径**，绝不回显文件内容（口令不得进日志）。
// 两个变量名都以字面量传入（而非拼 `_FILE` 后缀）：config_doc_gate_test.go 靠
// 源码里的 `"S3C_*"` 字面量核对文档收录，拼后缀会让新变量对门禁不可见。
func envOrFile(key, fileKey string) (string, error) {
	direct := os.Getenv(key)
	path := strings.TrimSpace(os.Getenv(fileKey))
	if path == "" {
		return direct, nil
	}
	if strings.TrimSpace(direct) != "" {
		return "", fmt.Errorf("%w: %s 与 %s 不能同时设置（明文赋值与文件挂载二选一，取错一个会锁死或放开口令）",
			ErrSecretFileAmbiguous, key, fileKey)
	}
	// 路径来自运维显式配置（S3C_STORE_KEY_FILE / S3C_TOKEN_FILE，docker secrets /
	// systemd LoadCredential 的挂载点），与 S3C_DATA_DIR 同信任级、不是终端用户输入，
	// 且这里没有更窄的路径约束可用。G304（同族规则）已按同一理由在 .golangci.yml
	// 排除，G703 是它在 gosec v2 的 taint 变体。
	b, err := os.ReadFile(path) // #nosec G703 -- 路径来自运维显式配置的 *_FILE 挂载点，非用户输入
	if err != nil {
		return "", fmt.Errorf("%w: %s=%s 不存在或不可读: %v", ErrSecretFileUnreadable, fileKey, path, err)
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "", fmt.Errorf("%w: %s=%s 内容为空（去掉首尾空白后无有效值）", ErrSecretFileUnreadable, fileKey, path)
	}
	return v, nil
}

// validateCSPSourceList 校验 CSP source-list 语法（P-5a）：只允许空格分隔的
// source 字符集。`;` 会被浏览器当作**下一条指令**的分隔符——S3C_CSP_CONNECT_SRC
// 注入 `; frame-ancestors *` 可架空 X-Frame-Options: DENY，因此启动期必须拒绝。
func validateCSPSourceList(src string) error {
	for _, r := range src {
		switch {
		case r == ' ':
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune(allowedCSPSourceChars, r):
		default:
			return fmt.Errorf("%w: S3C_CSP_CONNECT_SRC=%q 含非法字符 %q"+
				"（source-list 只允许空格分隔的 [A-Za-z0-9%s]）",
				ErrInvalidCSPValue, src, string(r), allowedCSPSourceChars)
		}
	}
	return nil
}

// FromEnv 从环境变量构建配置；启动时按 envFileCandidates 加载 .env（真实环境变量优先）。
// 显式 S3C_ENV_FILE 不可读、或数值型环境变量解析失败，都记入 Config.envErr
// 由 Validate 首查上抛使启动失败——**不静默回退默认值**。
func FromEnv() Config {
	loadErr := loadDotEnv()
	shutdownTimeout, secErr := envOrInt("S3C_SHUTDOWN_TIMEOUT", 30)
	rateLimitRPM, rpmErr := envOrInt("S3C_RATE_LIMIT_RPM", DefaultRateLimitRPM)
	rateLimitBurst, burstErr := envOrInt("S3C_RATE_LIMIT_BURST", DefaultRateLimitBurst)
	probeRPM, probeRPMErr := envOrInt("S3C_PROBE_RATE_LIMIT_RPM", DefaultProbeRateLimitRPM)
	probeBurst, probeBurstErr := envOrInt("S3C_PROBE_RATE_LIMIT_BURST", DefaultProbeRateLimitBurst)
	pingCacheSec, pingCacheErr := envOrNonNegInt("S3C_HEALTH_PING_CACHE_SECONDS", DefaultHealthPingCacheSeconds)
	storeKey, storeKeyErr := envOrFile("S3C_STORE_KEY", "S3C_STORE_KEY_FILE")
	token, tokenErr := envOrFile("S3C_TOKEN", "S3C_TOKEN_FILE")
	tokenScopes, scopeErr := parseTokenScopes(os.Getenv("S3C_TOKEN_SCOPES"))
	otelSampleRatio, ratioErr := envOrFloat("S3C_OTEL_SAMPLE_RATIO", 1)
	envErr := loadErr
	// 逐个补齐第一个非 nil 错误（保持既有链式风格：先发生的配置错误优先上抛）。
	for _, err := range []error{
		secErr, scopeErr, ratioErr, rpmErr, burstErr,
		probeRPMErr, probeBurstErr, pingCacheErr, storeKeyErr, tokenErr,
	} {
		if envErr == nil {
			envErr = err
		}
	}
	return Config{
		Addr:                envOr("S3C_ADDR", "127.0.0.1:5000"),
		DataDir:             envOr("S3C_DATA_DIR", "./data"),
		StaticDir:           envOr("S3C_STATIC_DIR", "../web/dist"),
		Region:              envOr("S3C_REGION", "us-east-1"),
		Token:               token,
		TokenScopes:         tokenScopes,
		MetricsToken:        os.Getenv("S3C_METRICS_TOKEN"),
		CORSOrigins:         splitList(envOr("S3C_CORS_ORIGINS", "")),
		LogLevel:            envOr("S3C_LOG_LEVEL", "info"),
		LogJSON:             envTruthy("S3C_LOG_JSON"),
		StoreDriver:         normalizeStoreDriver(envOr("S3C_STORE_DRIVER", "json")),
		StoreKey:            storeKey,
		AllowPlaintextStore: envTruthy("S3C_ALLOW_PLAINTEXT_STORE"),
		ShutdownTimeoutSec:  shutdownTimeout,
		ExposeMetrics:       envTruthy("S3C_EXPOSE_METRICS"),
		ExposeOpenAPI:       envTruthy("S3C_EXPOSE_OPENAPI"),
		CSPConnectSrc:       envOr("S3C_CSP_CONNECT_SRC", "'self' http://127.0.0.1:* http://localhost:*"),
		TrustedProxies:      splitList(envOr("S3C_TRUSTED_PROXIES", "")),
		SSRFDenyPrivate:     envTruthy("S3C_SSRF_DENY_PRIVATE"),
		RateLimitRPM:        rateLimitRPM,
		RateLimitBurst:      rateLimitBurst,
		ProbeRateLimitRPM:   probeRPM,
		ProbeRateLimitBurst: probeBurst,
		HealthPingCacheSec:  pingCacheSec,
		OTelEndpoint:        envOr("S3C_OTEL_ENDPOINT", ""),
		OTelSampleRatio:     otelSampleRatio,
		OTelServiceName:     envOr("S3C_OTEL_SERVICE_NAME", "s3client"),
		envErr:              envErr,
	}
}

func envTruthy(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// IsLoopbackAddr 判断监听地址是否**仅**绑定本机回环（127.0.0.1 / ::1）。
//
// 空 host（如 ":5000"）不算回环：net.Listen 会把它绑到 [::]——全部网卡，
// 与 0.0.0.0 等价；历史上把它判为回环会让最常见的通配写法绕过「非回环必须
// 设 S3C_TOKEN」的安全闸，账号管理 API 无鉴权暴露到所有接口（C2）。
// SplitHostPort 解析失败（缺端口）时该地址本就无法监听，按回环处理避免误报。
func IsLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return true // 无法解析时按回环判断，避免误报
	}
	return host == "127.0.0.1" || host == "::1" || host == "[::1]"
}

// Validate 对配置做安全校验：
//   - FromEnv 阶段的环境变量解析失败（envErr）原样上抛，拒绝启动；
//   - S3C_STORE_DRIVER 必须在白名单 json|sqlite|encrypted（空串 = 未显式指定，默认 json），
//     未知值拒绝启动——此前未知值被 store.Open 静默当 json，绕过下面的明文落盘闸（R4）；
//   - 短 S3C_TOKEN 拒绝启动（强制使用足够长度的随机值）；
//   - 非空 S3C_METRICS_TOKEN 必须达到最短长度；S3C_CSP_CONNECT_SRC 不得含控制字符 / 纯空白，
//     且必须是合法 source-list（拒 `;` 指令注入 / `,` / 允许字符集外的字符，P-5a）；
//   - 非回环监听必须设置 S3C_TOKEN；
//   - 非空 S3C_STORE_KEY 必须达到最短长度（落盘加密口令，已闭环：FEATURES.md §M）；
//   - json / sqlite 且 S3C_STORE_KEY 为空时必须显式 opt-in 明文落盘，否则拒绝启动（KNOWN_ISSUES #29/#31）。
//     大小写/空白变体先归一化再比较，保证 "JSON" 与 "json" 触发同一道闸。
//
// 多 token 时以单 token 最短者判定长度。
func (c Config) Validate() error {
	if c.envErr != nil {
		return c.envErr
	}
	// S3C_TOKEN_SCOPES 与 S3C_TOKEN 的交叉约束：登记的 token 必须是 S3C_TOKEN 成员，
	// prefixes / accounts 元素非空，否则 fail-closed（防「以为限制了、其实全权」）。
	if err := validateTokenScopes(c.Token, c.TokenScopes); err != nil {
		return err
	}
	if c.MetricsToken != "" && len(c.MetricsToken) < MinMetricsTokenLength {
		return fmt.Errorf("%w: got %d chars, need >= %d (建议 openssl rand -hex 32)",
			ErrShortMetricsToken, len(c.MetricsToken), MinMetricsTokenLength)
	}
	// S3C_SHUTDOWN_TIMEOUT 必须有上界，否则 time.Duration(n) * time.Second 溢出为负时长
	// → 优雅关停被静默跳过（错误被丢弃、退出码仍 0），在途流式下载被硬切断。
	if c.ShutdownTimeoutSec > maxShutdownTimeoutSec {
		return fmt.Errorf("%w: S3C_SHUTDOWN_TIMEOUT=%d 超过上界 %d 秒",
			ErrInvalidEnvValue, c.ShutdownTimeoutSec, maxShutdownTimeoutSec)
	}
	// S3C_CSP_CONNECT_SRC：空串 = 未指定（SetCSPConnectSrc 忽略空值、沿用默认 CSP，
	// 与 StoreDriver 空串口径一致）；一旦指定则必须可用——控制字符（含换行）会让
	// CSP 响应头写出失败或被客户端丢弃（加固项静默失效），纯空白会把 connect-src
	// 变成空源列表（阻断所有连接），而 source-list 非法字符（尤其 `;`）可注入基策略
	// 中不存在的指令（如 `; frame-ancestors *` 架空 X-Frame-Options: DENY，P-5a）。
	// 启动期 fail-closed，不静默降级。
	if src := c.CSPConnectSrc; src != "" {
		if strings.TrimSpace(src) == "" || strings.ContainsFunc(src, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
			return fmt.Errorf("%w: S3C_CSP_CONNECT_SRC=%q 含控制字符或为空白", ErrInvalidCSPValue, src)
		}
		if err := validateCSPSourceList(src); err != nil {
			return err
		}
	}
	driver := normalizeStoreDriver(c.StoreDriver)
	switch driver {
	case "", "json", "sqlite", "encrypted":
	default:
		// 报错一次性给出全部出路：合法取值，以及选定 json/sqlite 后的明文闸要求
		// （设 key 或本地 opt-in），运维改一处就能启动，不必再撞第二道闸才知道怎么配。
		return fmt.Errorf("%w: S3C_STORE_DRIVER=%q（仅接受 json|sqlite|encrypted）；"+
			"改用 json/sqlite 还需设置 S3C_STORE_KEY（>= %d 字符，openssl rand -hex 32），"+
			"或仅在本地联调时显式设置 S3C_ALLOW_PLAINTEXT_STORE=1",
			ErrUnknownStoreDriver, c.StoreDriver, MinStoreKeyLength)
	}
	if c.Token != "" {
		shortest := len(c.Token)
		for _, t := range strings.Split(c.Token, ",") {
			t = strings.TrimSpace(t)
			if t != "" && len(t) < shortest {
				shortest = len(t)
			}
		}
		if shortest < MinTokenLength {
			return fmt.Errorf("%w: got %d chars, need >= %d (建议 openssl rand -hex 32)", ErrShortToken, shortest, MinTokenLength)
		}
	}
	if c.Token == "" && !IsLoopbackAddr(c.Addr) {
		return fmt.Errorf("%w: 监听 %s 必须设置 S3C_TOKEN", ErrTokenRequiredNonLoopback, c.Addr)
	}
	if c.StoreKey != "" && len(c.StoreKey) < MinStoreKeyLength {
		return fmt.Errorf("%w: got %d chars, need >= %d (建议 openssl rand -hex 32)", ErrShortStoreKey, len(c.StoreKey), MinStoreKeyLength)
	}
	// 明文闸读归一化后的 driver：用户写 "JSON" / " json " 一样要被拦下，
	// 否则大小写变体就能绕过这道安全闸（与上面白名单同理）。
	if (driver == "json" || driver == "sqlite") && c.StoreKey == "" && !c.AllowPlaintextStore {
		return fmt.Errorf("%w: %s 驱动在 S3C_STORE_KEY 为空时会把 secretKey 明文落盘；"+
			"请设置 S3C_STORE_KEY（>= %d 字符，openssl rand -hex 32），"+
			"或仅在本地联调时显式设置 S3C_ALLOW_PLAINTEXT_STORE=1",
			ErrPlaintextStoreNotAllowed, driver, MinStoreKeyLength)
	}
	return nil
}

// StorePlaintextWarning 返回「账号 secretKey 将明文落盘」的启动告警文案，配置已加密时返回空串。
// json / sqlite 驱动在 S3C_STORE_KEY 为空时把 secretKey 明文写入 DataDir，只应出现在本地联调；
// 生产必须用 encrypted 或 sqlite + S3C_STORE_KEY（残留风险见 docs/ROADMAP.md §5.1 R3）。
func (c Config) StorePlaintextWarning() string {
	// 归一化后比较，保证 "JSON" / " json " 与 "json" 给出同一份告警文案。
	driver := normalizeStoreDriver(c.StoreDriver)
	switch driver {
	case "json", "sqlite":
	default:
		return ""
	}
	if c.StoreKey != "" {
		return ""
	}
	return fmt.Sprintf(
		"S3C_STORE_KEY 为空：%s 驱动的 secretKey 将明文落盘于 %s，仅限本地联调；生产请用 S3C_STORE_DRIVER=encrypted 或设置 S3C_STORE_KEY（>= %d 字符）",
		driver, c.DataDir, MinStoreKeyLength,
	)
}
