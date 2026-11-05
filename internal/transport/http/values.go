package httptransport

import "net/http"

// values is the marking of an essay or a test by «Онтограф ценностей 1.1» and
// what the program counted; a missing one is queued.
func (h *Handler) values(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ValuesFor(h.teacherID(r), r.PathValue("sourceId"), false)
	h.respond(w, value, err)
}

// retryValues takes a failed marking off its failure and queues it again.
func (h *Handler) retryValues(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(w, r) {
		return
	}
	value, err := h.service.ValuesFor(h.teacherID(r), r.PathValue("sourceId"), true)
	h.respond(w, value, err)
}

// essayValues counts the values over the submitted essays of an assignment.
func (h *Handler) essayValues(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.EssayValues(h.teacherID(r), r.PathValue("id"))
	h.respond(w, value, err)
}

// platoonValues counts the values over the open answers of a platoon.
func (h *Handler) platoonValues(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.PlatoonValues(h.teacherID(r), r.PathValue("group"))
	h.respond(w, value, err)
}
