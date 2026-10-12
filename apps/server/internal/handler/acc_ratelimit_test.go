package handler

// acc_ratelimit_test.go —— ratelimit.go 与 client_cache.go 的补测。
// 限流为纯内存令牌桶，直接白盒操作桶状态，避免真实时钟等待导致的 flaky。

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/model"
	"github.com/weilai1949/s3client/apps/server/internal/s3wrap"
)

// TestAccRateLimiterAllowBranches 白盒覆盖 ipLimiter.allow 的：新桶 / 回满上限 / 饿死拒绝。
func TestAccRateLimiterAllowBranches(t *testing.T) {
	l := newIPLimiter()

	// 新 IP：建桶即放行
	if !l.allow("10.0.0.1") {
		t.Fatal("新 IP 应放行")
	}

	// 回满上限：29.5 + 约1秒×2token/s > 30 → 截断到 30 再扣 1 → 剩 29
	capShard := l.shard("cap")
	capShard.mu.Lock()
	capShard.byIP["cap"] = &tokenBucket{tokens: 29.5, last: time.Now().Add(-time.Second)}
	capShard.mu.Unlock()
	if !l.allow("cap") {
		t.Fatal("cap 桶应放行")
	}
	capShard.mu.Lock()
	got := capShard.byIP["cap"].tokens
	capShard.mu.Unlock()
	if got < 28.9 || got > 29.1 {
		t.Fatalf("cap 桶 tokens=%v, want ≈29（先截断到 30 再扣 1）", got)
	}

	// 饿死：tokens<1 → 拒绝且不扣减
	starveShard := l.shard("starve")
	starveShard.mu.Lock()
	starveShard.byIP["starve"] = &tokenBucket{tokens: 0.5, last: time.Now()}
	starveShard.mu.Unlock()
	if l.allow("starve") {
		t.Fatal("tokens<1 应拒绝")
	}
	starveShard.mu.Lock()
	got = starveShard.byIP["starve"].tokens
	starveShard.mu.Unlock()
	// 拒绝不扣减；但 allow 内部会按 elapsed 回填，tokens 只增不减（容差 0.01）
	if got < 0.5-0.01 || got >= 1 {
		t.Fatalf("拒绝后不应扣减（回填除外）, tokens=%v", got)
	}
}

// TestAccRateLimiterReapLocked 白盒：分片 reapLocked 只清理超过 10 分钟未活跃的桶。
func TestAccRateLimiterReapLocked(t *testing.T) {
	l := newIPLimiter()
	sh := l.shard("old")
	sh.mu.Lock()
	sh.byIP["old"] = &tokenBucket{tokens: 5, last: time.Now().Add(-11 * time.Minute)}
	sh.byIP["fresh"] = &tokenBucket{tokens: 5, last: time.Now()}
	sh.reapLocked(time.Now())
	_, oldLeft := sh.byIP["old"]
	_, freshLeft := sh.byIP["fresh"]
	sh.mu.Unlock()
	if oldLeft {
		t.Fatal("过期桶应被清理")
	}
	if !freshLeft {
		t.Fatal("活跃桶应保留")
	}
}

// TestAccRateLimiterShardCapEvictsStaleFirst 分片超硬上限时先清过期桶：
// 过期桶被移除、且未被「随机淘汰」误伤其它活跃桶（即 reap 先于 eviction）。
func TestAccRateLimiterShardCapEvictsStaleFirst(t *testing.T) {
	l := newIPLimiter()
	sh := l.shard("stale")
	sh.mu.Lock()
	sh.byIP["stale"] = &tokenBucket{tokens: 1, last: time.Now().Add(-11 * time.Minute)}
	sh.mu.Unlock()

	// 只往同一分片灌新 IP，直到填满硬上限。
	filled := 0
	for i := 0; filled < maxBucketsPerShard-1; i++ {
		ip := "ip-" + strconv.Itoa(i)
		if l.shard(ip) != sh {
			continue
		}
		l.allow(ip)
		filled++
	}
	sh.mu.Lock()
	before := len(sh.byIP)
	sh.mu.Unlock()
	if before != maxBucketsPerShard {
		t.Fatalf("填满分片后桶数=%d, want %d", before, maxBucketsPerShard)
	}

	// 再插一个新桶 → 超限：应先清掉唯一的过期桶，活跃桶全部保留。
	sh.mu.Lock()
	sh.addBucketLocked("fresh", time.Now())
	_, staleLeft := sh.byIP["stale"]
	n := len(sh.byIP)
	sh.mu.Unlock()
	if staleLeft {
		t.Fatal("超硬上限时应先清理过期桶")
	}
	if n != maxBucketsPerShard {
		t.Fatalf("清理过期桶后桶数=%d, want %d（不应误伤活跃桶）", n, maxBucketsPerShard)
	}
}

// TestAccRateLimiterShardHardCap 只灌同一分片：桶数必须停在硬上限，不得随 IP 数增长。
func TestAccRateLimiterShardHardCap(t *testing.T) {
	l := newIPLimiter()
	sh := l.shard("seed")
	filled := 0
	for i := 0; filled < maxBucketsPerShard*4; i++ {
		ip := "ip-" + strconv.Itoa(i)
		if l.shard(ip) != sh {
			continue
		}
		l.allow(ip)
		filled++
		sh.mu.Lock()
		n := len(sh.byIP)
		sh.mu.Unlock()
		if n > maxBucketsPerShard {
			t.Fatalf("第 %d 个桶后分片桶数=%d, 超过硬上限 %d", filled, n, maxBucketsPerShard)
		}
	}
}

// TestAccRateLimiterTotalBucketsBounded 分片后**总量**上限仍为 10_000 量级：
// 灌入远超上限的 IP 后，驻留桶数不得超过 分片数 × (maxBucketsPerShard+1)。
func TestAccRateLimiterTotalBucketsBounded(t *testing.T) {
	l := newIPLimiter()
	const ips = 30_000
	for i := 0; i < ips; i++ {
		l.allow("ip-" + strconv.Itoa(i))
	}
	total := 0
	for i := range l.shards {
		sh := &l.shards[i]
		sh.mu.Lock()
		n := len(sh.byIP)
		sh.mu.Unlock()
		if n > maxBucketsPerShard {
			t.Fatalf("分片 %d 桶数=%d, 应不超过 %d", i, n, maxBucketsPerShard)
		}
		total += n
	}
	if limit := rateLimitShards * maxBucketsPerShard; total > limit {
		t.Fatalf("驻留桶总数=%d, 应不超过 %d（分片不得放大内存上限）", total, limit)
	}
}

// TestAccRateLimiterConcurrent 并发 allow（go test -race 下验证分片锁无数据竞争）。
func TestAccRateLimiterConcurrent(t *testing.T) {
	l := newIPLimiter()
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				l.allow("10.0." + strconv.Itoa(g) + "." + strconv.Itoa(i%8))
			}
		}(g)
	}
	wg.Wait()
}

// TestAccClientIPNoPort 补测 clientIP：RemoteAddr 无端口时原样返回；无可信代理时不信任 XFF。
func TestAccClientIPNoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:5555"
	if got := clientIPWithProxies(req, nil); got != "203.0.113.9" {
		t.Fatalf("clientIP=%q, want 203.0.113.9", got)
	}
	req.RemoteAddr = "opaque-addr"
	if got := clientIPWithProxies(req, nil); got != "opaque-addr" {
		t.Fatalf("clientIP=%q, want 原样返回", got)
	}
}

// TestAccWithRateLimitNilLimiter 白盒：limiter 为 nil 时直接放行（如手工构造的 Handler）。
func TestAccWithRateLimitNilLimiter(t *testing.T) {
	h := &Handler{log: accDiscardLogger()} // limiter == nil
	called := false
	wrapped := h.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	if !called {
		t.Fatal("nil limiter 应直接放行")
	}
}

// TestAccWithRateLimitOptionsBypass 白盒：OPTIONS 预检不计入限流。
func TestAccWithRateLimitOptionsBypass(t *testing.T) {
	h := &Handler{log: accDiscardLogger(), limiter: newIPLimiter()}
	called := false
	wrapped := h.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodOptions, "/api/accounts", nil))
	if !called {
		t.Fatal("OPTIONS 应绕过限流")
	}
	// 非 /api 路径（静态资源）不计入限流
	called = false
	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/assets/x.js", nil))
	if !called {
		t.Fatal("非 /api 路径应绕过限流")
	}
}

// TestAccRateLimitExcludedAndTriggered 黑盒全链路：health/metrics 不限流；同 IP 打满触发 429。
func TestAccRateLimitExcludedAndTriggered(t *testing.T) {
	env := accNewEnv(t, "http://127.0.0.1:1", "b")

	// health / metrics 不限流：无论多少次都不应 429
	for i := 0; i < 5; i++ {
		if rr := env.accDoRec("GET", "/api/health", ""); rr.Code == http.StatusTooManyRequests {
			t.Fatal("health 不应被限流")
		}
	}

	// 连续打同一写路径直到令牌耗尽（阈值 30，留足余量），应出现 429 + Retry-After
	got429 := false
	for i := 0; i < 60; i++ {
		rr := env.accDoRec("GET", "/api/accounts", "")
		if rr.Code == http.StatusTooManyRequests {
			got429 = true
			if got := rr.Header().Get("Retry-After"); got != "5" {
				t.Fatalf("Retry-After=%q, want 5", got)
			}
			if !strings.Contains(rr.Body.String(), "rate limit exceeded") {
				t.Fatalf("429 body=%s", rr.Body.String())
			}
			break
		}
	}
	if !got429 {
		t.Fatal("连续请求应触发 429 限流")
	}
}

// ---- client_cache.go ----

// TestAccClientCacheNilAccount 白盒：acc 为 nil / ID 为空时不走缓存，直接构建（或校验失败）。
func TestAccClientCacheNilAccount(t *testing.T) {
	c := newClientCache()
	if _, err := c.get(nil); err == nil {
		t.Fatal("nil 账号应返回错误")
	}
	// ID 为空但凭证齐全 → 每次新建客户端（不缓存）
	acc := &model.Account{Endpoint: "http://127.0.0.1:9", Region: "us-east-1", AccessKey: "ak", SecretKey: "sk"}
	c1, err := c.get(acc)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	c2, err := c.get(acc)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if c1 == c2 {
		t.Fatal("无 ID 账号不应命中缓存")
	}
	if len(c.m) != 0 {
		t.Fatalf("无 ID 账号不应写缓存, len=%d", len(c.m))
	}
}

// TestAccClientCacheRebuildError 白盒：UpdatedAt 变化触发重建，重建失败返回错误且缓存保留。
func TestAccClientCacheRebuildError(t *testing.T) {
	c := newClientCache()
	acc := &model.Account{ID: "acc-1", Endpoint: "http://127.0.0.1:9", Region: "us-east-1", AccessKey: "ak", SecretKey: "sk", UpdatedAt: time.Unix(1000, 0)}
	if _, err := c.get(acc); err != nil {
		t.Fatalf("首次构建: %v", err)
	}
	// 账号更新（UpdatedAt 变化）且配置变坏（缺密钥）→ 重建失败
	acc.UpdatedAt = time.Unix(2000, 0)
	acc.SecretKey = ""
	cl, err := c.get(acc)
	if err == nil {
		t.Fatal("重建失败应返回错误")
	}
	if cl != nil {
		t.Fatal("失败时不应返回客户端")
	}
	// 旧缓存未被破坏：回退 UpdatedAt 仍能命中旧客户端
	acc.UpdatedAt = time.Unix(1000, 0)
	acc.SecretKey = "sk"
	if _, err := c.get(acc); err != nil {
		t.Fatalf("回退后应命中旧缓存: %v", err)
	}
}

// TestAccClientCacheConcurrentReuse 并发冷启动：多协程同时取同一账号，
// 除首个写入者外其余应命中“双重检查”复用路径（返回同一 client 实例）。
func TestAccClientCacheConcurrentReuse(t *testing.T) {
	const rounds = 8
	const workers = 24
	for round := 0; round < rounds; round++ {
		c := newClientCache()
		acc := &model.Account{
			ID: "acc-shared-" + strconv.Itoa(round), Endpoint: "http://127.0.0.1:9",
			Region: "us-east-1", AccessKey: "ak", SecretKey: "sk", UpdatedAt: time.Unix(int64(round), 0),
		}
		start := make(chan struct{})
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			seen = map[*s3wrap.Client]bool{}
		)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				cl, err := c.get(acc)
				if err != nil {
					t.Errorf("get: %v", err)
					return
				}
				mu.Lock()
				seen[cl] = true
				mu.Unlock()
			}()
		}
		close(start)
		wg.Wait()
		if len(seen) != 1 {
			t.Fatalf("round %d: %d 个不同 client 实例, 应全部复用同一实例", round, len(seen))
		}
	}
}

// TestAccClientCacheEvict 白盒：evict 删除指定条目；空 ID 直接忽略。
func TestAccClientCacheEvict(t *testing.T) {
	c := newClientCache()
	acc := &model.Account{ID: "acc-e", Endpoint: "http://127.0.0.1:9", Region: "us-east-1", AccessKey: "ak", SecretKey: "sk"}
	first, err := c.get(acc)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	c.evict("") // 空 ID：无操作不 panic
	c.evict("acc-e")
	c.mu.RLock()
	n := len(c.m)
	c.mu.RUnlock()
	if n != 0 {
		t.Fatalf("evict 后缓存应清空, len=%d", n)
	}
	second, err := c.get(acc)
	if err != nil {
		t.Fatalf("重建: %v", err)
	}
	if first == second {
		t.Fatal("evict 后应重建新实例")
	}
}
