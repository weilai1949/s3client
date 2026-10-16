package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 简易令牌桶限速：默认每 IP 120 req/min、突发 30（config.DefaultRateLimit*）。
// 仅保护写密集型 API 被刷；health 与静态资源不计入。

// rateLimitShards 是限流桶的分片数。
//
// 旧实现用一把全局 mutex 保护整张 map，而 withRateLimit 挂在鉴权外层——每个 /api/
// 请求都要进同一个临界区。分片后不同 IP 落到不同锁，高并发 + 多核下不再串行
// （docs/archive/server-review-2026-10-12.md S2）。临界区本身是纳秒级，收益只在大并发时可见。
const rateLimitShards = 16

// maxBucketsPerShard 是单分片的桶数**硬上限**；总量硬上限 = rateLimitShards × 该值
// = 10_000，与分片前的「全局 10_000」内存上限等值——分片不得放大总上限。
//
// 旧实现只是「超过 10_000 就尝试清过期桶」，若 IP 都活跃则 map 无硬上限；分片后每个
// 分片都必须自带硬上限，否则总量会放大成 16 × 10_000（docs/archive/server-review-2026-10-12.md S2）。
const maxBucketsPerShard = 10_000 / rateLimitShards

type ipLimiter struct {
	shards [rateLimitShards]limiterShard
	rpm    int
	burst  int
}

// limiterShard 是一组 IP 的令牌桶与它自己的锁。
type limiterShard struct {
	mu   sync.Mutex
	byIP map[string]*tokenBucket
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

func newIPLimiter(rpm, burst int) *ipLimiter {
	l := &ipLimiter{rpm: rpm, burst: burst}
	for i := range l.shards {
		l.shards[i].byIP = make(map[string]*tokenBucket)
	}
	return l
}

// shard 按 IP 选分片（FNV-1a，无分配）。同一 IP 恒在同一分片。
func (l *ipLimiter) shard(ip string) *limiterShard {
	var h uint32 = 2166136261
	for i := 0; i < len(ip); i++ {
		h = (h ^ uint32(ip[i])) * 16777619
	}
	return &l.shards[h%rateLimitShards]
}

func (l *ipLimiter) allow(ip string) bool {
	now := time.Now()
	sh := l.shard(ip)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	b, ok := sh.byIP[ip]
	if !ok {
		sh.addBucketLocked(ip, now, l.burst)
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * (float64(l.rpm) / 60.0)
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// addBucketLocked 插入新桶并把分片桶数收回硬上限内：先清过期桶，仍超限再淘汰
// **任意**一个桶（map 迭代顺序随机，单次 O(1)）。
//
// 淘汰活跃桶不会削弱限速强度：能撑爆分片的本就是持有大量 IP 的攻击者，被淘汰的
// 也只会是他们自己的桶；而对集群而言内存有确定上界，比「随 IP 数无限增长」安全。
func (s *limiterShard) addBucketLocked(ip string, now time.Time, burst int) {
	s.byIP[ip] = &tokenBucket{tokens: float64(burst) - 1, last: now}
	if len(s.byIP) <= maxBucketsPerShard {
		return
	}
	s.reapLocked(now)
	for len(s.byIP) > maxBucketsPerShard {
		for k := range s.byIP {
			delete(s.byIP, k)
			break
		}
	}
}

func (s *limiterShard) reapLocked(now time.Time) {
	for ip, b := range s.byIP {
		if now.Sub(b.last) > 10*time.Minute {
			delete(s.byIP, ip)
		}
	}
}

// clientIP 按 handler 配置的可信代理列表解析客户端 IP。
func (h *Handler) clientIP(r *http.Request) string {
	return clientIPWithProxies(r, h.trustedProxies)
}

// clientIPWithProxies 解析客户端 IP：仅当直连对端（RemoteAddr）在可信代理列表中时
// 才采信 X-Forwarded-For 的**最后一段**，否则一律回退 RemoteAddr。
//
// 为什么不能无条件信任 XFF：直连部署（未过代理）时任何客户端都能伪造该头，
// 从而为每个请求换一个「IP」绕过限速（ASSESSMENT M4）。
// 为什么取最后一段而不是首段：右侧条目由可信代理自己追加（它直连的对端），
// 上游客户端只能在左侧预置伪造值；取首段会让伪造者每换一个值就换一个限速桶（review R1）。
func clientIPWithProxies(r *http.Request, trusted []string) string {
	remote := remoteHost(r.RemoteAddr)
	if len(trusted) == 0 {
		return remote
	}
	for _, p := range trusted {
		if p == remote {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				if i := strings.LastIndex(xff, ","); i >= 0 {
					return strings.TrimSpace(xff[i+1:])
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

// rateLimitKey 返回限流桶键：IPv6 按 /64 前缀聚合——攻击者可用单 /64 内海量地址
// 各自换取独立令牌桶绕过限速；IPv4（含 IPv4 映射 IPv6）与无法解析的地址原样返回。
// 仅用于限流桶键；审计日志仍记完整 IP（clientIP 原值，不经本函数）。
func rateLimitKey(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() != nil {
		return ip
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String()
}

// probePaths 是豁免**主**限流器的探针端点（精确匹配，不含子路径）：
// K8s 探针 / Prometheus scrape / 契约抓取不能被业务限流打成失败，否则会误判服务不可用。
// 但它们匿名且会被高频轮询，故改由独立的宽松令牌桶（S3C_PROBE_RATE_LIMIT_*）约束（P-12）。
var probePaths = map[string]bool{
	"/api/health":       true,
	"/api/metrics":      true,
	"/api/openapi.json": true,
}

func (h *Handler) withRateLimit(next http.Handler) http.Handler {
	if h.limiter == nil && h.probeLimiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !strings.HasPrefix(p, "/api/") || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		// 探针端点走独立宽松桶；其余 /api/ 路径走主桶。两桶按同一 rateLimitKey
		// （IPv6 /64 聚合）计桶，彼此不消耗对方额度。
		lim, retryAfter := h.limiter, "5"
		if probePaths[p] {
			lim, retryAfter = h.probeLimiter, "1"
		}
		ip := h.clientIP(r)
		if lim != nil && !lim.allow(rateLimitKey(ip)) {
			h.audit(r, auditRateLimited, "ip", ip)
			w.Header().Set("Retry-After", retryAfter)
			h.writeErr(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}
