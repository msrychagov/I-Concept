package httptransport

import "net/http"

func (h *Handler) openapi(w http.ResponseWriter, r *http.Request) {
	h.serveAsset(w, "static/openapi.json")
}
