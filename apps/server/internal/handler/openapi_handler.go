package handler

import "net/http"

// openapiSpec 暴露当前路由表的 OpenAPI 3.0 JSON。
// 该端点**经过**鉴权层：配置了 S3C_TOKEN 时无 token 访问返回 401（`withAuth` 只豁免
// health/metrics，见 middleware.go）；未开启 `S3C_EXPOSE_OPENAPI` 时由 `withOpenAPIGate`
// 直接 404。此处注释曾误称「不经鉴权层」，与实际相反（实际行为更安全，见
// docs/review-2026-09-19.md §7.3 D2）。
func (h *Handler) openapiSpec(w http.ResponseWriter, r *http.Request) {
	h.openapi.HTTPHandler().ServeHTTP(w, r)
}
