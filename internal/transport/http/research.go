package httptransport

import (
	"bytes"
	"net/http"
	"vucdiagnostic/internal/domain/model"
)

func (h *Handler) sessions(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ResearchQueryFor(h.teacherID(r), "state", "", "")
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, map[string]any{"sessions": value.(map[string]any)["sessions"]}, 200)
}
func (h *Handler) sessionReport(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ResearchQueryFor(h.teacherID(r), "report", r.PathValue("id"), "")
	h.respond(w, value, err)
}
func (h *Handler) exportSession(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ResearchQueryFor(h.teacherID(r), "export", r.PathValue("id"), "")
	h.respond(w, value, err)
}
func (h *Handler) groupReport(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.ResearchQueryFor(h.teacherID(r), "group", "", r.URL.Query().Get("group"))
	h.respond(w, value, err)
}
func (h *Handler) importSession(w http.ResponseWriter, r *http.Request) {
	var session model.Session
	if !h.decodeBody(w, r, 3_000_000, &session) {
		return
	}
	value, err := h.service.ResearchCommandFor(h.teacherID(r), commandInput(map[string]any{"session": session}), "import")
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, value, 201)
}
func (h *Handler) reopenSession(w http.ResponseWriter, r *http.Request) {
	h.sessionAction(w, r, "reopen")
}
func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	h.sessionAction(w, r, "revoke")
}
func (h *Handler) sessionAction(w http.ResponseWriter, r *http.Request, action string) {
	if !h.validOrigin(w, r) {
		return
	}
	if _, err := h.service.ResearchQueryFor(h.teacherID(r), "export", r.PathValue("id"), ""); err != nil {
		h.respond(w, nil, err)
		return
	}
	value, err := h.service.ResearchCommandFor(h.teacherID(r), commandInput(map[string]any{"id": r.PathValue("id")}), action)
	h.respond(w, value, err)
}
func (h *Handler) updatePlan(w http.ResponseWriter, r *http.Request) {
	var body planRequest
	if !h.decodeBody(w, r, 300000, &body) {
		return
	}
	if body.Plan == nil {
		h.reply(w, map[string]any{"error": "Укажите массив plan."}, 400)
		return
	}
	if _, err := h.service.ResearchQueryFor(h.teacherID(r), "export", r.PathValue("id"), ""); err != nil {
		h.respond(w, nil, err)
		return
	}
	input := commandInput(body)
	setField(input, "id", r.PathValue("id"))
	value, err := h.service.ResearchCommandFor(h.teacherID(r), input, "plan")
	h.respond(w, value, err)
}
func (h *Handler) exportCSV(w http.ResponseWriter, r *http.Request) {
	var output bytes.Buffer
	if err := h.service.ExportCSVFor(h.teacherID(r), &output, r.URL.Query().Get("group")); err != nil {
		h.respond(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="vuc-answers.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(output.Bytes())
}
