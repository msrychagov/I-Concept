package httptransport

import "net/http"

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.TeachingStateFor(h.teacherID(r))
	h.respond(w, value, err)
}
func (h *Handler) assignments(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.TeachingStateFor(h.teacherID(r))
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, map[string]any{"assignments": value.(map[string]any)["assignments"]}, 200)
}
func (h *Handler) createAssignment(w http.ResponseWriter, r *http.Request) {
	var body assignmentRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.AssignmentCommandFor(h.teacherID(r), commandInput(body), "assignment")
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, value, 201)
}
func (h *Handler) updateAssignment(w http.ResponseWriter, r *http.Request) {
	var body assignmentStatusRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	if body.Active == nil {
		h.reply(w, map[string]any{"error": "Укажите active: true или false."}, 400)
		return
	}
	input := commandInput(body)
	setField(input, "id", r.PathValue("id"))
	value, err := h.service.AssignmentCommandFor(h.teacherID(r), input, "assignment_status")
	h.respond(w, value, err)
}
func (h *Handler) startAssignment(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(w, r) {
		return
	}
	student, ok := h.service.CurrentAccount(loginToken(r, "student"), "student")
	if !ok {
		h.reply(w, map[string]any{"error": "Войдите в кабинет."}, 401)
		return
	}
	value, err := h.service.StartAssignment(student, r.PathValue("id"))
	h.respond(w, value, err)
}
