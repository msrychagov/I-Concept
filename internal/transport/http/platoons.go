package httptransport

import "net/http"

func (h *Handler) platoonManagement(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.PlatoonManagement(h.teacherID(r))
	h.respond(w, value, err)
}
func (h *Handler) savePlatoon(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.SavePlatoon(h.teacherID(r), r.PathValue("id"), body.Name)
	h.respond(w, value, err)
}
func (h *Handler) manageStudent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action    string `json:"action"`
		PlatoonID string `json:"platoonId"`
	}
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.ManageStudent(h.teacherID(r), r.PathValue("id"), body.Action, body.PlatoonID)
	h.respond(w, value, err)
}
func (h *Handler) assignTeacherVUC(w http.ResponseWriter, r *http.Request) {
	var body struct {
		VUC string `json:"vuc"`
	}
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.AssignTeacherVUC(r.PathValue("id"), body.VUC)
	h.respond(w, value, err)
}
