package httptransport

import (
	"net/http"
	"vucdiagnostic/internal/usecase"
)

func (h *Handler) instructors(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.InstructorList(h.teacherID(r))
	h.respond(w, value, err)
}

func (h *Handler) saveInstructor(w http.ResponseWriter, r *http.Request) {
	var body usecase.InstructorInput
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.SaveInstructor(h.teacherID(r), r.PathValue("id"), body)
	if err == nil && r.PathValue("id") == "" {
		h.reply(w, value, http.StatusCreated)
		return
	}
	h.respond(w, value, err)
}

func (h *Handler) deleteInstructor(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(w, r) {
		return
	}
	value, err := h.service.DeleteInstructor(h.teacherID(r), r.PathValue("id"))
	h.respond(w, value, err)
}
