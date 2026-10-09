package httptransport

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) survey(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Survey(loginToken(r, "student"), r.PathValue("id"))
	h.respond(w, value, err)
}
func (h *Handler) consent(w http.ResponseWriter, r *http.Request) {
	var body consentRequest
	if !h.decodeBody(w, r, 16384, &body) || !h.requiredRevision(w, body.Revision) {
		return
	}
	if body.Value == nil {
		h.reply(w, map[string]any{"error": "Укажите согласие (value)."}, 400)
		return
	}
	value, err := h.service.SurveyCommand(loginToken(r, "student"), r.PathValue("id"), commandInput(body), "consent")
	h.respond(w, value, err)
}
func (h *Handler) answer(w http.ResponseWriter, r *http.Request) {
	var body answerRequest
	if !h.decodeBody(w, r, 16384, &body) || !h.requiredRevision(w, body.Revision) {
		return
	}
	if len(body.Value) == 0 {
		h.reply(w, map[string]any{"error": "Укажите value: целое число или null для удаления ответа."}, 400)
		return
	}
	var answer *int
	if err := json.Unmarshal(body.Value, &answer); err != nil {
		h.reply(w, map[string]any{"error": "Ответ должен быть целым числом или null."}, 400)
		return
	}
	input := commandInput(body)
	setField(input, "item", r.PathValue("itemId"))
	value, err := h.service.SurveyCommand(loginToken(r, "student"), r.PathValue("id"), input, "answer")
	h.respond(w, value, err)
}
func (h *Handler) completeSurvey(w http.ResponseWriter, r *http.Request) {
	var body completionRequest
	if !h.decodeBody(w, r, 16384, &body) || !h.requiredRevision(w, body.Revision) {
		return
	}
	value, err := h.service.SurveyCommand(loginToken(r, "student"), r.PathValue("id"), commandInput(body), "finish")
	h.respond(w, value, err)
}
