package handler

import "net/http"

// openapiSpec 暴露当前路由表的 OpenAPI 3.0 JSON。
// 该端点不经鉴权层（CORS / withAuth），与 /api/health 保持一致：它是契约而非业务。
// 注意：路由注册时把它放在 auth 中间件之前（见 routes.go），所以不会被拦。
func (h *Handler) openapiSpec(w http.ResponseWriter, r *http.Request) {
	h.openapi.HTTPHandler().ServeHTTP(w, r)
}
