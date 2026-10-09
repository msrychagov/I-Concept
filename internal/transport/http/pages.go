package httptransport

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

func (h *Handler) page(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if _, ok := h.service.CurrentAccount(loginToken(r, "teacher"), "teacher"); ok {
		http.Redirect(w, r, "/research", http.StatusSeeOther)
		return
	}
	if _, ok := h.service.CurrentAccount(loginToken(r, "student"), "student"); ok {
		http.Redirect(w, r, "/student", http.StatusSeeOther)
		return
	}
	h.serveAsset(w, "pages/index.html")
}
func (h *Handler) research(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.service.CurrentAccount(loginToken(r, "teacher"), "teacher"); !ok {
		http.Redirect(w, r, "/teacher", http.StatusSeeOther)
		return
	}
	h.serveAsset(w, "pages/research.html")
}
func (h *Handler) asset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/assets/")
	if name == "bank.json" {
		if h.authorized(w, r) {
			h.reply(w, h.service.Bank(), 200)
		}
		return
	}
	if !fs.ValidPath(name) {
		http.NotFound(w, r)
		return
	}
	h.serveAsset(w, "static/"+name)
}
func (h *Handler) serveAsset(w http.ResponseWriter, name string) {
	data, err := fs.ReadFile(h.files, name)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}
