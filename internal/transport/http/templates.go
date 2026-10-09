package httptransport

import (
	"net/http"
	"vucdiagnostic/internal/domain/model"
)

func (h *Handler) questionBank(w http.ResponseWriter, r *http.Request) {
	h.reply(w, h.service.Bank(), 200)
}
func (h *Handler) templates(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ResearchQuery("state", "", "")
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	state := value.(map[string]any)
	h.reply(w, map[string]any{"profiles": state["profiles"], "emptyProfile": state["emptyProfile"]}, 200)
}
func (h *Handler) template(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ResearchQuery("state", "", "")
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	for _, profile := range value.(map[string]any)["profiles"].([]model.Profile) {
		if profile.ID == r.PathValue("id") {
			h.reply(w, profile, 200)
			return
		}
	}
	h.reply(w, map[string]any{"error": "Шаблон не найден."}, 404)
}
func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) { h.saveTemplate(w, r, false) }
func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request) { h.saveTemplate(w, r, true) }
func (h *Handler) saveTemplate(w http.ResponseWriter, r *http.Request, update bool) {
	var profile model.Profile
	if !h.decodeBody(w, r, 3_000_000, &profile) {
		return
	}
	operation := "profile"
	status := 201
	if update {
		profile.ID = r.PathValue("id")
		operation = "profile_update"
		status = 200
	} else {
		profile.ID = ""
	}
	value, err := h.service.ResearchCommand(commandInput(map[string]any{"profile": profile}), operation)
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	if !update {
		hLocation := "/api/v1/templates/" + value.(model.Profile).ID
		w.Header().Set("Location", hLocation)
	}
	h.reply(w, value, status)
}
