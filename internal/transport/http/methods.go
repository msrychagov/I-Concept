package httptransport

import "net/http"

// methodsSummary gives the levels of the methods by platoon.
func (h *Handler) methodsSummary(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.MethodsSummaryFor(h.teacherID(r), r.URL.Query().Get("group"))
	h.respond(w, value, err)
}
