package httptransport

import (
	"net/http"
)

// studentOnly answers 401 unless the request carries a student login.
func (h *Handler) studentOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.service.CurrentAccount(loginToken(r, "student"), "student"); !ok {
			h.reply(w, map[string]any{"error": "Войдите в кабинет."}, 401)
			return
		}
		next(w, r)
	}
}

func (h *Handler) formSubmission(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.FormOpen(loginToken(r, "student"), r.PathValue("id"))
	h.respond(w, value, err)
}
func (h *Handler) formConsent(w http.ResponseWriter, r *http.Request) {
	var body consentRequest
	if !h.decodeBody(w, r, 16384, &body) || !h.requiredRevision(w, body.Revision) {
		return
	}
	if body.Value == nil {
		h.reply(w, map[string]any{"error": "Укажите согласие (value)."}, 400)
		return
	}
	value, err := h.service.FormCommand(loginToken(r, "student"), r.PathValue("id"), "consent", commandInput(body))
	h.respond(w, value, err)
}
func (h *Handler) formSection(w http.ResponseWriter, r *http.Request) {
	var body sectionRequest
	if !h.decodeBody(w, r, 65536, &body) || !h.requiredRevision(w, body.Revision) {
		return
	}
	if body.Answers == nil {
		h.reply(w, map[string]any{"error": "Укажите ответы раздела (answers)."}, 400)
		return
	}
	input := commandInput(body)
	setField(input, "sectionId", r.PathValue("sectionId"))
	value, err := h.service.FormCommand(loginToken(r, "student"), r.PathValue("id"), "section", input)
	h.respond(w, value, err)
}
func (h *Handler) formCompletion(w http.ResponseWriter, r *http.Request) {
	var body formCompletionRequest
	if !h.decodeBody(w, r, 16384, &body) || !h.requiredRevision(w, body.Revision) {
		return
	}
	value, err := h.service.FormCommand(loginToken(r, "student"), r.PathValue("id"), "completion", commandInput(body))
	h.respond(w, value, err)
}
func (h *Handler) formSuggestions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	value, err := h.service.FormSuggestions(loginToken(r, "student"), r.PathValue("id"), query.Get("field"), query.Get("q"))
	h.respond(w, value, err)
}
func (h *Handler) formOwnAnswers(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.FormOwnAnswers(loginToken(r, "student"), r.PathValue("id"))
	h.respond(w, value, err)
}
func (h *Handler) formDescription(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.FormDescription(r.PathValue("formId"))
	h.respond(w, value, err)
}
func (h *Handler) formOverview(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.FormOverviewFor(h.teacherID(r), r.PathValue("formId"))
	h.respond(w, value, err)
}
func (h *Handler) formSubmissionDetail(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.FormSubmissionDetailFor(h.teacherID(r), r.PathValue("id"))
	h.respond(w, value, err)
}
func (h *Handler) formPotential(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.PotentialSummaryFor(h.teacherID(r), r.PathValue("formId"), r.URL.Query().Get("platoon"))
	h.respond(w, value, err)
}

// formAutobiography is the printed autobiography of a submitted autobiography.
func (h *Handler) formAutobiography(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.AutobiographyTextFor(h.teacherID(r), r.PathValue("id"))
	h.respond(w, value, err)
}
func (h *Handler) formOwnAutobiography(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.OwnAutobiographyText(loginToken(r, "student"), r.PathValue("id"))
	h.respond(w, value, err)
}
