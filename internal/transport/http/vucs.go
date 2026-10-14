package httptransport

import "net/http"

func (h *Handler) createVUC(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !h.decodeBody(w, r, 16384, &body) {
		return
	}
	value, err := h.service.CreateVUC(body.Name)
	if err != nil {
		h.respond(w, nil, err)
		return
	}
	h.reply(w, value, http.StatusCreated)
}

func (h *Handler) listVUCs(w http.ResponseWriter, r *http.Request) {
	h.reply(w, h.service.ListVUCs(), http.StatusOK)
}
