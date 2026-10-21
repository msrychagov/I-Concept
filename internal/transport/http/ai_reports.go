package httptransport

import "net/http"

func (h *Handler) studentAIReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodPost && !h.validOrigin(w, r) {
		return
	}
	value, err := h.service.AIAnalysisFor(h.teacherID(r), "student", r.PathValue("id"), r.Method == http.MethodPost)
	h.respond(w, value, err)
}

func (h *Handler) platoonAIReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodPost && !h.validOrigin(w, r) {
		return
	}
	value, err := h.service.AIAnalysisFor(h.teacherID(r), "platoon", r.URL.Query().Get("group"), r.Method == http.MethodPost)
	h.respond(w, value, err)
}
