package httptransport

import (
	"net/http"
	"vucdiagnostic/internal/usecase"
)

func (h *Handler) cohorts(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.CohortList(h.teacherID(r))
	h.respond(w, value, err)
}

func (h *Handler) saveCohort(w http.ResponseWriter, r *http.Request) {
	var body usecase.CohortInput
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.SaveCohort(h.teacherID(r), r.PathValue("id"), body)
	if err == nil && r.PathValue("id") == "" {
		h.reply(w, value, http.StatusCreated)
		return
	}
	h.respond(w, value, err)
}

func (h *Handler) deleteCohort(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(w, r) {
		return
	}
	value, err := h.service.DeleteCohort(h.teacherID(r), r.PathValue("id"))
	h.respond(w, value, err)
}
