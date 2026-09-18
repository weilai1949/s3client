package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 简易令牌桶限速：每 IP 默认 120 req/min，突发 30。
// 仅保护写密集型 API 被刷；health 与静态资源不计入。
const (
	rateLimitPerMin = 120
	rateLimitBurst  = 30
)

type ipLimiter struct {
	mu   sync.Mutex
	byIP map[string]*tokenBucket
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

func newIPLimiter() *ipLimiter {
	return &ipLimiter{byIP: make(map[string]*tokenBucket)}
}

func (l *ipLimiter) allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.byIP[ip]
	if !ok {
		l.byIP[ip] = &tokenBucket{tokens: rateLimitBurst - 1, last: now}
		// 偶尔清理，避免 map 无限增长
		if len(l.byIP) > 10_000 {
			l.reapLocked(now)
		}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * (float64(rateLimitPerMin) / 60.0)
	if b.tokens > rateLimitBurst {
		b.tokens = rateLimitBurst
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *ipLimiter) reapLocked(now time.Time) {
	for ip, b := range l.byIP {
		if now.Sub(b.last) > 10*time.Minute {
			delete(l.byIP, ip)
		}
	}
}

// clientIP 按 handler 配置的可信代理列表解析客户端 IP。
func (h *Handler) clientIP(r *http.Request) string {
	return clientIPWithProxies(r, h.trustedProxies)
}

// clientIPWithProxies 解析客户端 IP：仅当直连对端（RemoteAddr）在可信代理列表中时
// 才采信 X-Forwarded-For 的首段，否则一律回退 RemoteAddr。
//
// 为什么不能无条件信任 XFF：直连部署（未过代理）时任何客户端都能伪造该头，
// 从而为每个请求换一个「IP」绕过限速（roadmap #3 / ASSESSMENT M4）。
func clientIPWithProxies(r *http.Request, trusted []string) string {
	remote := remoteHost(r.RemoteAddr)
	if len(trusted) == 0 {
		return remote
	}
	for _, p := range trusted {
		if p == remote {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				if i := strings.Index(xff, ","); i >= 0 {
					return strings.TrimSpace(xff[:i])
				}
				return strings.TrimSpace(xff)
			}
			return remote
		}
	}
	return remote
}

// remoteHost 从 RemoteAddr 提取主机部分；无法解析时原样返回。
func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func (h *Handler) withRateLimit(next http.Handler) http.Handler {
	if h.limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/api/health" || p == "/api/metrics" || p == "/api/openapi.json" || !strings.HasPrefix(p, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		ip := h.clientIP(r)
		if !h.limiter.allow(ip) {
			h.audit(r, auditRateLimited, "ip", ip)
			w.Header().Set("Retry-After", "5")
			h.writeErr(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}
