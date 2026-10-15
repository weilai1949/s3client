package handler

import (
	"net/http"
	"time"
)

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{
		"time":  time.Now().UTC(),
		"store": map[string]any{"ok": true},
	}
	if h.hasValidAPIToken(r) {
		body["version"] = h.version
	}
	if err := h.store.Ping(); err != nil {
		h.log.Debug("health store ping", "err", err)
		body["status"] = "error"
		body["store"] = map[string]any{"ok": false, "error": "store unavailable"}
		h.writeJSON(w, http.StatusServiceUnavailable, body)
		return
	}
	body["status"] = "ok"
	h.writeJSON(w, http.StatusOK, body)
}
