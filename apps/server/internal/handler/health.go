package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/weilai1949/s3client/apps/server/internal/store"
)

// pingCache 是 store.Ping 结果的短 TTL 缓存（P-12）：探针端点（/api/health、/api/metrics）
// 匿名且被高频轮询，每次都打存储会把 sqlite 连接与锁打满；TTL 内复用最近一次结果即可。
//
// 成功与失败**同一口径**都缓存：store 掉线时匿名探针不会每次都穿透到存储（正是要堵的放大面），
// 代价是恢复也可能最多延迟一个 TTL 才反映；ttl <= 0 表示关闭缓存、每次实打。
// 匿名与已认证请求共用同一份缓存（不做 per-token 维度）。
type pingCache struct {
	mu  sync.Mutex
	ttl time.Duration
	at  time.Time
	err error
}

// newPingCache 按秒构造缓存；0（或负数，仅在测试里可能出现）表示关闭缓存。
func newPingCache(seconds int) *pingCache {
	return &pingCache{ttl: time.Duration(seconds) * time.Second}
}

// result 返回缓存或实打的结果；ttl <= 0 时直接实打（不进锁，零共享状态）。
func (c *pingCache) result(st store.AccountStore) error {
	if c.ttl <= 0 {
		return st.Ping()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.at.IsZero() && time.Since(c.at) < c.ttl {
		return c.err
	}
	err := st.Ping()
	c.err = err
	c.at = time.Now()
	return err
}

// pingStore 返回 store 可达性：配了缓存则走缓存，未配（零值 Handler）则每次实打。
func (h *Handler) pingStore() error {
	if h.pingCache == nil {
		return h.store.Ping()
	}
	return h.pingCache.result(h.store)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{
		"time":  time.Now().UTC(),
		"store": map[string]any{"ok": true},
	}
	if h.hasValidAPIToken(r) {
		body["version"] = h.version
	}
	if err := h.pingStore(); err != nil {
		h.log.Debug("health store ping", "err", err)
		body["status"] = "error"
		body["store"] = map[string]any{"ok": false, "error": "store unavailable"}
		h.writeJSON(w, http.StatusServiceUnavailable, body)
		return
	}
	body["status"] = "ok"
	h.writeJSON(w, http.StatusOK, body)
}
