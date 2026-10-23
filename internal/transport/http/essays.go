package httptransport

import "net/http"

func (h *Handler) essays(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.EssayOverview(h.teacherID(r))
	h.respond(w, value, err)
}

func (h *Handler) essaySummary(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.EssaySummary(h.teacherID(r), r.PathValue("id"))
	h.respond(w, value, err)
}

func (h *Handler) composeEssaySummary(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(w, r) {
		return
	}
	value, err := h.service.ComposeEssaySummary(h.teacherID(r), r.PathValue("id"))
	h.respond(w, value, err)
}
