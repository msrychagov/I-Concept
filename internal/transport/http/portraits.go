package httptransport

import "net/http"

// portrait gives the status and the text of the portrait of a source.
func (h *Handler) portrait(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.PortraitFor(h.teacherID(r), r.PathValue("task"), r.PathValue("sourceId"))
	h.respond(w, value, err)
}

// retryPortrait queues a failed portrait again; a stored one is returned as it is.
func (h *Handler) retryPortrait(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.RetryPortrait(h.teacherID(r), r.PathValue("task"), r.PathValue("sourceId"))
	h.respond(w, value, err)
}
