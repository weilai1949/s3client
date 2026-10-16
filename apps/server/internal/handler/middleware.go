package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/weilai1949/s3client/apps/server/internal/httpx"
)

// withSecurityHeaders 为所有响应设置基础安全头（含静态资源与 API）。
//
// defaultCSPConnectSrc 是 connect-src 的**唯一**字面量来源（New 的默认值共用）：
// 两处各写一份字面量必然漂移（review Nit），新增引用请用常量而非复制字符串。
const defaultCSPConnectSrc = "'self' http://127.0.0.1:* http://localhost:*"

// buildCSP 拼出整条 Content-Security-Policy：脚本仅允许同源（杜绝注入脚本执行）；
// 内联样式只允许在元素 style 属性上（Vue :style），内联 <style> 块保持禁用；
// connect-src 默认仅同源 + 本地 Tauri 后端（127.0.0.1/localhost），
// 自定义后端 / 远程多后端需设置 S3C_CSP_CONNECT_SRC 显式放宽（S-9 收紧）；
// frame-ancestors 'none' 与 X-Frame-Options: DENY 双保险防嵌套（桌面壳同值，P-5b）。
//
// 它是纯函数，且**只在启动期调用**（包初始化 / New / SetCSPConnectSrc）：结果预计算进
// Handler.cspHeader，请求路径上不再做字符串拼接与分配（docs/archive/server-review-2026-10-12.md S4）。
// connect 由调用方保证非空（New 传 defaultCSPConnectSrc，SetCSPConnectSrc 忽略空值）。
func buildCSP(connect string) string {
	return "default-src 'self'; script-src 'self'; style-src 'self'; style-src-attr 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; frame-src 'self' blob:; connect-src " + connect + "; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'"
}

// defaultCSPHeader 是默认策略的预计算整串；零值 Handler（未走 New）也用它，
// 保证安全响应头永不缺失。
var defaultCSPHeader = buildCSP(defaultCSPConnectSrc)

func (h *Handler) withSecurityHeaders(next http.Handler) http.Handler {
	csp := h.cspHeader
	if csp == "" {
		csp = defaultCSPHeader
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
}

// maxRequestIDLen 是回显客户端 X-Request-ID 的最大长度。
// 超长值会污染日志与响应头，超出即视为无效并由服务端生成新 id。
const maxRequestIDLen = 128

// validRequestID 判断客户端提供的 X-Request-ID 是否可安全回显：
// 非空、长度 ≤ maxRequestIDLen、且全部为可见 ASCII（0x21–0x7E，排除空格与控制字符）。
// 拒绝控制字符同时也挡住了换行注入（日志伪造 / 响应头拆分）。
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		if c := id[i]; c < 0x21 || c > 0x7E {
			return false
		}
	}
	return true
}

// withLogging 记录每个请求的方法、路径、耗时与 request id；并累计 metrics。
func (h *Handler) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := r.Header.Get("X-Request-ID")
		if !validRequestID(reqID) {
			// 客户端未提供或提供了不可信（超长/含控制字符/非 ASCII）的值时生成服务端 id。
			reqID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", reqID)
		rec := httpx.NewStatusRecorder(w)
		next.ServeHTTP(rec, r)
		// 耗时与状态类同处一次调用：HTTP 延迟直方图与请求计数必须同源（ROADMAP #18）。
		dur := time.Since(start)
		recordHTTPMetric(rec.Status(), dur)
		attrs := []any{"method", r.Method, "path", r.URL.Path, "status", rec.Status(), "dur", dur.String(), "req", reqID}
		if r.URL.Path == "/api/health" || r.URL.Path == "/api/metrics" {
			h.log.Debug("http", attrs...)
			return
		}
		h.log.Info("http", attrs...)
	})
}

// corsAllowedOrigin 依据配置与安全默认值返回允许的 Origin。
// 返回 "" 表示不允许跨域（同源请求无需 CORS 头）。
func (h *Handler) corsAllowedOrigin(origin string) string {
	if origin == "" {
		return "" // 同源或非浏览器请求
	}
	// 显式白名单（S3C_CORS_ORIGINS）
	for _, o := range h.corsOrigins {
		if o == "*" {
			return "*"
		}
		if o == origin {
			return o
		}
	}
	// 安全默认：仅放行 localhost / 127.0.0.1 / tauri 自定义协议
	if isTrustedDefaultOrigin(origin) {
		return origin
	}
	return ""
}

func isTrustedDefaultOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "tauri" {
		// P-13：只信任桌面壳的生产 origin tauri://localhost。历史上放行任意
		// tauri://* 会让本机任意应用 / 恶意页面用 tauri://evil 拿到 ACAO 回显，
		// 进而跨域读写本服务；显式白名单（S3C_CORS_ORIGINS）仍可放行任意 origin。
		return strings.EqualFold(u.Hostname(), "localhost")
	}
	if scheme != "http" && scheme != "https" {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1", "tauri.localhost":
		return true
	}
	return false
}

func (h *Handler) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := h.corsAllowedOrigin(origin)
		if allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			if allowed != "*" {
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			if allowed == "" {
				w.WriteHeader(http.StatusForbidden)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
			return
		}
		// 安全关键：跨域请求（带 Origin 且不在白名单）必须拒绝。
		// 仅给响应加 CORS 头无法阻止浏览器发送"简单请求"（text/plain 等无预检）
		// 执行状态变更接口，这里直接 403 阻断。
		if origin != "" && allowed == "" {
			w.Header().Set("Vary", "Origin")
			h.writeErr(w, http.StatusForbidden, "origin not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withMetricsGate 在未开启 S3C_EXPOSE_METRICS 时返回 404；开启且配置
// S3C_METRICS_TOKEN 时再要求独立 Bearer。未配置该 token 时保持匿名 scrape 兼容。
func (h *Handler) withMetricsGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.exposeMetrics && r.URL.Path == "/api/metrics" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/api/metrics" && h.metricsToken != "" {
			cred, ok := bearerCredential(r.Header.Get("Authorization"))
			if !ok || !secureCompare(cred, h.metricsToken) {
				h.audit(r, auditAuthDenied, "reason", "metrics_token")
				h.writeErr(w, http.StatusUnauthorized, "unauthorized")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// withOpenAPIGate 在未开启 S3C_EXPOSE_OPENAPI 时，对 /api/openapi.json 返回 404，
// 让外部看不出 API 契约（避免辅助攻击者枚举端点/参数）。
func (h *Handler) withOpenAPIGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.exposeOpenAPI && r.URL.Path == "/api/openapi.json" {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withAuth 在配置了 S3C_TOKEN 时，对 /api/* 强制 Bearer 鉴权。
// 支持逗号分隔多 token（轮换）；跳过 OPTIONS 预检与 /api/health。
func (h *Handler) withAuth(next http.Handler) http.Handler {
	if len(h.tokens) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		p := r.URL.Path
		if p != "/api" && !strings.HasPrefix(p, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if p == "/api/health" || p == "/api/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		cred, ok := bearerCredential(r.Header.Get("Authorization"))
		if !ok {
			h.audit(r, auditAuthDenied, "reason", "malformed")
			h.writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if token, matched := h.matchingToken(cred); matched {
			// 常量时间比较命中后，再按该 token 的 S3C_TOKEN_SCOPES 声明判定最小权限；
			// 判定通过的请求带上作用域，供 handler 侧按最小权限过滤响应（计划列表）。
			r, allowed := h.authorize(w, r, token)
			if !allowed {
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		// 审计：鉴权失败（不记录提交的凭证值）。
		h.audit(r, auditAuthDenied, "reason", "bad_token")
		h.writeErr(w, http.StatusUnauthorized, "unauthorized")
	})
}

// bearerCredential 解析 RFC 7235 Bearer 凭证；auth-scheme 大小写不敏感。
func bearerCredential(auth string) (string, bool) {
	scheme, cred, ok := strings.Cut(auth, " ")
	if !ok || !strings.EqualFold(scheme, "bearer") {
		return "", false
	}
	return cred, true
}

func (h *Handler) matchingToken(cred string) (string, bool) {
	for _, token := range h.tokens {
		if secureCompare(cred, token) {
			return token, true
		}
	}
	return "", false
}

// hasValidAPIToken 供 health 决定是否回显 version：探针始终免鉴权，
// 只有命中 S3C_TOKEN 且未过作用域有效期的请求才能看到版本号。
func (h *Handler) hasValidAPIToken(r *http.Request) bool {
	cred, ok := bearerCredential(r.Header.Get("Authorization"))
	if !ok {
		return false
	}
	token, ok := h.matchingToken(cred)
	return ok && !h.tokenExpired(token)
}

// secureCompare 以常量时间比较两个字符串。
func secureCompare(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
