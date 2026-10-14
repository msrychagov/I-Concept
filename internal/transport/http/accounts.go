package httptransport

import (
	"net/http"
	"vucdiagnostic/internal/usecase"
)

func (h *Handler) authRole(w http.ResponseWriter, r *http.Request) (string, bool) {
	role := r.PathValue("role")
	if role != "student" && role != "teacher" {
		h.reply(w, map[string]any{"error": "Роль не найдена."}, 404)
		return "", false
	}
	return role, true
}
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	role, ok := h.authRole(w, r)
	if !ok {
		return
	}
	var body registerRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	h.authenticate(w, r, role, "register", commandInput(body), http.StatusCreated)
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	role, ok := h.authRole(w, r)
	if !ok {
		return
	}
	var body loginRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	h.authenticate(w, r, role, "login", commandInput(body), http.StatusOK)
}
func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request, role, operation string, body usecase.Input, status int) {
	result, err := h.service.Authenticate(role, operation, body)
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "vuc_" + role, Value: result.Token, Path: "/", Expires: result.Expires, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
	h.reply(w, map[string]any{"account": usecase.PublicAccount(result.Account)}, status)
}
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	role, ok := h.authRole(w, r)
	if !ok || !h.validOrigin(w, r) {
		return
	}
	if err := h.service.Logout(loginToken(r, role)); err != nil {
		h.respond(w, nil, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "vuc_" + role, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
	h.reply(w, map[string]any{"ok": true}, 200)
}
func (h *Handler) teacherProfile(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.AccountState(loginToken(r, "teacher"), "teacher")
	h.respond(w, value, err)
}
func (h *Handler) studentProfile(w http.ResponseWriter, r *http.Request) {
	account, ok := h.service.CurrentAccount(loginToken(r, "student"), "student")
	if !ok {
		h.reply(w, map[string]any{"error": "Войдите в личный кабинет."}, 401)
		return
	}
	h.reply(w, map[string]any{"student": usecase.PublicAccount(account)}, 200)
}
func (h *Handler) studentAssignments(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.AccountState(loginToken(r, "student"), "student")
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, map[string]any{"assignments": value.(map[string]any)["assignments"]}, 200)
}
func (h *Handler) updateTeacherProfile(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.service.CurrentAccount(loginToken(r, "teacher"), "teacher"); !ok {
		h.reply(w, map[string]any{"error": "Войдите в кабинет преподавателя."}, 401)
		return
	}
	var body profileRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.UpdateTeacherAccount(loginToken(r, "teacher"), commandInput(body), "update_profile")
	h.respond(w, value, err)
}
func (h *Handler) changeTeacherPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.service.CurrentAccount(loginToken(r, "teacher"), "teacher"); !ok {
		h.reply(w, map[string]any{"error": "Войдите в кабинет преподавателя."}, 401)
		return
	}
	var body passwordRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.UpdateTeacherAccount(loginToken(r, "teacher"), commandInput(body), "change_password")
	h.respond(w, value, err)
}

func (h *Handler) createTeacherInvitation(w http.ResponseWriter, r *http.Request) {
	var body teacherInvitationRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.CreateTeacherInvitation(commandInput(body))
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, value, http.StatusCreated)
}

func (h *Handler) createStudentRegistrationLink(w http.ResponseWriter, r *http.Request) {
	teacher, ok := h.service.CurrentAccount(loginToken(r, "teacher"), "teacher")
	if !ok {
		h.reply(w, map[string]any{"error": "Войдите в кабинет преподавателя."}, 401)
		return
	}
	var body studentRegistrationLinkRequest
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.CreateStudentRegistrationLink(teacher.ID, commandInput(body))
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, value, http.StatusCreated)
}
