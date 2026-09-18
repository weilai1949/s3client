package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/weilai1949/s3clinet/apps/server/internal/model"
	"github.com/weilai1949/s3clinet/apps/server/internal/openapi"
	"github.com/weilai1949/s3clinet/apps/server/internal/s3wrap"
	"github.com/weilai1949/s3clinet/apps/server/internal/service"
	"github.com/weilai1949/s3clinet/apps/server/internal/store"
)

// Handler 承载 HTTP 接口逻辑。
type Handler struct {
	store          store.AccountStore
	log            *slog.Logger
	staticDir      string
	corsOrigins    []string // CORS 白名单；空 = 仅同源 + localhost/tauri
	tokens         []string // Bearer 鉴权；可多个（S3C_TOKEN 逗号分隔，支持轮换）
	version        string   // 服务端版本号（ldflags 注入），用于 /api/health 上报
	exposeMetrics  bool     // 是否暴露 /api/metrics（默认 false：404 假装不存在）
	exposeOpenAPI  bool     // 是否暴露 /api/openapi.json（默认 false：404 假装不存在）
	cspConnectSrc  string   // CSP connect-src 白名单
	trustedProxies []string // 可信反向代理 IP；仅这些对端的 X-Forwarded-For 被采信
	clients        *clientCache
	migrateJobs    *service.JobRegistry
	limiter        *ipLimiter
	openapi        *openapi.Registry // 路由旁登记；/api/openapi.json 直接复用
}

// New 构造 handler。token 支持逗号分隔多值（轮换/吊销：去掉旧 token 即可）。
// exposeMetrics=false 时 /api/metrics 一律 404；exposeOpenAPI=false 时 /api/openapi.json 一律 404，
// 均避免公网暴露运行指标 / API 契约信息。cspConnectSrc 默认仅同源 + 本地 Tauri 后端。
func New(st store.AccountStore, log *slog.Logger, staticDir string, corsOrigins []string, token, version string, exposeMetrics, exposeOpenAPI bool) *Handler {
	reg := openapi.New("s3clinet API", version)
	registerOpenAPI(reg, version)
	return &Handler{
		store: st, log: log, staticDir: staticDir, corsOrigins: corsOrigins,
		tokens: splitTokens(token), version: version, exposeMetrics: exposeMetrics, exposeOpenAPI: exposeOpenAPI,
		cspConnectSrc: "'self' http://127.0.0.1:* http://localhost:*",
		clients:       newClientCache(),
		migrateJobs:   service.NewJobRegistry(),
		limiter:       newIPLimiter(),
		openapi:       reg,
	}
}

func splitTokens(s string) []string {
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

// SetCSPConnectSrc 覆盖 CSP connect-src 白名单（默认仅同源 + 本地 Tauri 后端）。
// 用于支持自定义多后端/远程后端地址；调用方需在 Routes() 前设置。
func (h *Handler) SetCSPConnectSrc(src string) {
	if src != "" {
		h.cspConnectSrc = src
	}
}

// SetTrustedProxies 设置可信反向代理 IP 列表；仅来自这些对端的
// X-Forwarded-For 才会被采信（否则回退 RemoteAddr，防直连伪造绕过限速）。
// 需在 Routes() 前调用；不调用则完全不信任 XFF。
func (h *Handler) SetTrustedProxies(proxies []string) {
	h.trustedProxies = proxies
}

// SetJobPersister 替换任务清单持久化器，并立即按历史清单恢复任务（未完成 → interrupted）。
// 需在 Routes() 前调用；不调用则保持纯内存（默认，与历史行为一致）。
//
// 采用 setter 而非扩展 New 的参数：New 有 50+ 处调用点（含测试），
// 追加参数会造成大范围机械改动，且该能力对绝大多数测试无关。
func (h *Handler) SetJobPersister(p service.JobPersister) {
	// 先停掉旧注册表（取消其未完成任务），再以新 persister 重建。
	h.migrateJobs.Stop()
	h.migrateJobs = service.NewJobRegistryWithPersister(p)
}

// Shutdown 取消进行中的异步迁移并停止 reap 循环。
func (h *Handler) Shutdown() {
	if h.migrateJobs != nil {
		h.migrateJobs.Stop()
	}
}

// ---- DTO ----

type objectItem struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
	ContentType  string    `json:"contentType"`
	StorageClass string    `json:"storageClass"`
	IsDir        bool      `json:"isDir"`
}

type listObjectsResponse struct {
	Objects        []objectItem `json:"objects"`
	CommonPrefixes []string     `json:"commonPrefixes"`
	IsTruncated    bool         `json:"isTruncated"`
	NextToken      string       `json:"nextToken"`
}

func fromS3Object(o s3wrap.ObjectItem) objectItem {
	it := objectItem{
		Key:          o.Key,
		Size:         o.Size,
		LastModified: o.LastModified,
		ETag:         o.ETag,
		StorageClass: o.StorageClass,
	}
	it.IsDir = strings.HasSuffix(it.Key, "/")
	return it
}

// ---- helpers ----

const maxBody = 8 << 20 // 8MB request body cap（批量删除/复制可含大量长 key）

// maxZipKeys 限制单次打包的对象数，防止一次请求无界流式输出。
const maxZipKeys = 1000

// maxBatchKeys 批量复制/迁移等操作单次 key 上限。
const maxBatchKeys = 10_000

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.Error("encode response", "err", err)
	}
}

func (h *Handler) writeErr(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]any{"error": msg})
}

// writeBadJSON 记录解析失败详情，向客户端返回固定消息。
func (h *Handler) writeBadJSON(w http.ResponseWriter, err error) {
	h.log.Debug("invalid request body", "err", err)
	h.writeErr(w, http.StatusBadRequest, "invalid request body")
}

// writeInternalErr 记录内部错误详情到日志，向客户端返回通用消息，避免泄露 S3/SDK 细节。
// 若 err 可识别为 S3 API 错误，使用 s3HTTPStatus + s3UserMessage（统一 NoSuchBucket→404 等）。
func (h *Handler) writeInternalErr(w http.ResponseWriter, err error, publicMsg string) {
	if err != nil {
		if code := s3HTTPStatus(err); code != 500 {
			// s3UserMessage 对非 nil 错误恒返回非空；此处不二次防御。
			msg := s3UserMessage(err)
			h.log.Debug("s3 mapped error", "err", err, "status", code, "public", msg)
			h.writeErr(w, code, msg)
			return
		}
	}
	if publicMsg == "" {
		publicMsg = "internal server error"
	}
	h.log.Error("handler error", "err", err, "public", publicMsg)
	h.writeErr(w, http.StatusInternalServerError, publicMsg)
}

func (h *Handler) readJSON(r *http.Request, v any) error {
	// 强制 JSON 请求体：浏览器对非 JSON 的 POST（text/plain 等）不预检，会放行跨域写；
	// 要求 application/json 让所有变更请求触发 CORS 预检（配合 withCORS 的 403）。
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		return errors.New("Content-Type must be application/json")
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	// 拒绝 JSON 之后的尾部数据，避免歧义请求体。
	if dec.More() {
		return errors.New("unexpected trailing data after JSON body")
	}
	return nil
}

// accountClient 标准前缀：读取账号（404）并构建 S3 客户端（400）。
// 返回 client、account 与 ok；ok=false 时响应已写出，调用方直接 return。
func (h *Handler) accountClient(w http.ResponseWriter, r *http.Request) (*s3wrap.Client, *model.Account, bool) {
	acc, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		h.writeErr(w, http.StatusNotFound, "account not found")
		return nil, nil, false
	}
	client, err := h.clients.get(acc)
	if err != nil {
		h.log.Debug("s3 client init", "err", err)
		h.writeErr(w, http.StatusBadRequest, "invalid account configuration")
		return nil, nil, false
	}
	return client, acc, true
}

// bucketOr 解析 bucket；为空时写 400 并返回 ok=false。
func (h *Handler) bucketOr(w http.ResponseWriter, acc *model.Account, b string) (string, bool) {
	if b == "" {
		b = acc.BucketOrDefault()
	}
	if b == "" {
		h.writeErr(w, http.StatusBadRequest, "bucket is required")
		return "", false
	}
	return b, true
}
