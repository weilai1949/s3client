package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MinTokenLength 是 S3C_TOKEN 允许的最小字符数。短口令易被暴力猜测，
// 启动时硬失败（生产推荐 openssl rand -hex 32 / 64）。
const MinTokenLength = 16

// MinStoreKeyLength 是 S3C_STORE_KEY 允许的最小字符数。该口令是账号密钥落盘加密
// 的唯一凭据（Argon2id 派生），过短同样会被暴力破解；非空即校验（已闭环：features.md §M）。
const MinStoreKeyLength = 16

// ErrShortToken 表示 S3C_TOKEN 长度低于 MinTokenLength。
var ErrShortToken = errors.New("S3C_TOKEN too short")

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

// normalizeStoreDriver 归一化驱动名（去空白 + 小写）。FromEnv / Validate /
// StorePlaintextWarning 三处共用，保证 "JSON" / " json " 与小写行为完全一致——
// 任何一处漏归一化都会重新打开 R4 的安全闸绕过口子。
func normalizeStoreDriver(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// loadDotEnvFile 从指定路径加载 KEY=VALUE 到环境变量（已存在的环境变量优先）。
// 支持注释行、引号与空行；实现为 30 行的极简解析器，不引入第三方依赖。
func loadDotEnvFile(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
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

// loadDotEnv 加载第一个存在的候选 .env 文件（不存在则静默跳过，保持零配置可启动）。
func loadDotEnv() {
	for _, p := range envFileCandidates() {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		loadDotEnvFile(p)
		return
	}
}

// Config 汇总服务端配置。所有项均可通过环境变量覆盖，并内置安全默认值。
type Config struct {
	Addr                string   // 监听地址，默认回环 127.0.0.1:8080（更安全）
	DataDir             string   // 数据目录，存放账号持久化文件
	StaticDir           string   // Web 静态资源目录
	Region              string   // 账号缺省 region
	Token               string   // 可选 API 鉴权 token；非空则要求 Bearer
	CORSOrigins         []string // CORS 白名单；空 = 仅同源 + localhost/tauri
	LogLevel            string   // debug|info|warn|error
	LogJSON             bool     // true = slog JSON（容器/生产更易采集）
	StoreDriver         string   // json|sqlite|encrypted，账号存储后端（FromEnv 归一化为小写）
	StoreKey            string   // encrypted 模式必填；Argon2id+盐派生（仅 S3C2）
	AllowPlaintextStore bool     // true = 显式允许 json/sqlite 无 StoreKey 明文落盘（S3C_ALLOW_PLAINTEXT_STORE=1，仅本地）
	ShutdownTimeoutSec  int      // SIGTERM 后等待活跃连接结束的最长时间（秒）
	ExposeMetrics       bool     // true = 暴露 /api/metrics（Prometheus 文本）；默认 false，避免公网信息泄露
	ExposeOpenAPI       bool     // true = 暴露 /api/openapi.json（API 契约）；默认 false，避免公网泄露端点信息
	CSPConnectSrc       string   // CSP connect-src 白名单；默认仅同源 + 本地 Tauri 后端；多后端/远程需显式放宽
	TrustedProxies      []string // 可信反向代理 IP；仅这些对端的 X-Forwarded-For 被采信（默认空 = 不信任 XFF）
	SSRFDenyPrivate     bool     // true = 连私网/回环端点也拒绝（默认 false：自托管场景放行，见 ADR-003）

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

// FromEnv 从环境变量构建配置；启动时按 envFileCandidates 加载 .env（真实环境变量优先）。
// 环境变量解析失败记入 Config.envErr，由 Validate 首查上抛使启动失败。
func FromEnv() Config {
	loadDotEnv()
	shutdownTimeout, envErr := envOrInt("S3C_SHUTDOWN_TIMEOUT", 30)
	return Config{
		Addr:                envOr("S3C_ADDR", "127.0.0.1:8080"),
		DataDir:             envOr("S3C_DATA_DIR", "./data"),
		StaticDir:           envOr("S3C_STATIC_DIR", "../web/dist"),
		Region:              envOr("S3C_REGION", "us-east-1"),
		Token:               os.Getenv("S3C_TOKEN"),
		CORSOrigins:         splitList(envOr("S3C_CORS_ORIGINS", "")),
		LogLevel:            envOr("S3C_LOG_LEVEL", "info"),
		LogJSON:             envTruthy("S3C_LOG_JSON"),
		StoreDriver:         normalizeStoreDriver(envOr("S3C_STORE_DRIVER", "json")),
		StoreKey:            os.Getenv("S3C_STORE_KEY"),
		AllowPlaintextStore: envTruthy("S3C_ALLOW_PLAINTEXT_STORE"),
		ShutdownTimeoutSec:  shutdownTimeout,
		ExposeMetrics:       envTruthy("S3C_EXPOSE_METRICS"),
		ExposeOpenAPI:       envTruthy("S3C_EXPOSE_OPENAPI"),
		CSPConnectSrc:       envOr("S3C_CSP_CONNECT_SRC", "'self' http://127.0.0.1:* http://localhost:*"),
		TrustedProxies:      splitList(envOr("S3C_TRUSTED_PROXIES", "")),
		SSRFDenyPrivate:     envTruthy("S3C_SSRF_DENY_PRIVATE"),
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
// 空 host（如 ":8080"）不算回环：net.Listen 会把它绑到 [::]——全部网卡，
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
//   - 非回环监听必须设置 S3C_TOKEN；
//   - 非空 S3C_STORE_KEY 必须达到最短长度（落盘加密口令，已闭环：features.md §M）；
//   - json / sqlite 且 S3C_STORE_KEY 为空时必须显式 opt-in 明文落盘，否则拒绝启动（KNOWN_ISSUES #29/#31）。
//     大小写/空白变体先归一化再比较，保证 "JSON" 与 "json" 触发同一道闸。
//
// 多 token 时以单 token 最短者判定长度。
func (c Config) Validate() error {
	if c.envErr != nil {
		return c.envErr
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
// 生产必须用 encrypted 或 sqlite + S3C_STORE_KEY（残留风险见 docs/roadmap.md §5.1 R3）。
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
