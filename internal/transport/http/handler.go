// Package httptransport adapts HTTP requests and responses to application use cases.
package httptransport

import (
	"io/fs"
	"net/http"
	"vucdiagnostic/internal/usecase"
)

type Config struct{ AdminUser, AdminPassword string }
type Handler struct {
	service *usecase.Service
	files   fs.FS
	config  Config
	mux     *http.ServeMux
}

func New(service *usecase.Service, files fs.FS, config Config) *Handler {
	h := &Handler{service: service, files: files, config: config, mux: http.NewServeMux()}
	h.mux.HandleFunc("/", h.page)
	h.mux.HandleFunc("/student", func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, "pages/student.html") })
	h.mux.HandleFunc("/teacher", func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, "pages/teacher.html") })
	h.mux.HandleFunc("/research", h.research)
	h.mux.HandleFunc("/survey", func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, "pages/survey.html") })
	h.mux.HandleFunc("/autobiography", func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, "pages/form.html") })
	h.mux.HandleFunc("/essay", func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, "pages/form.html") })
	h.mux.HandleFunc("/autobiography/print", func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, "pages/autobiography-print.html") })
	h.mux.HandleFunc("/assets/", h.asset)
	h.apiRoutes()
	h.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		h.reply(w, map[string]any{"ok": true, "version": service.Version()}, 200)
	})
	return h
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }
