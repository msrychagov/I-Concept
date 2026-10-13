package httptransport

import (
	"net/http"
	"strings"
)

// Each operation has a method-specific route and a dedicated handler.
func (h *Handler) apiRoutes() {
	h.mux.HandleFunc("POST /api/v1/auth/{role}/register", h.register)
	h.mux.HandleFunc("POST /api/v1/auth/{role}/login", h.login)
	h.mux.HandleFunc("POST /api/v1/auth/{role}/logout", h.logout)
	h.mux.HandleFunc("GET /api/v1/teachers/me", h.teacherProfile)
	h.mux.HandleFunc("PATCH /api/v1/teachers/me", h.updateTeacherProfile)
	h.mux.HandleFunc("PUT /api/v1/teachers/me/password", h.changeTeacherPassword)
	h.mux.HandleFunc("GET /api/v1/students/me", h.studentProfile)
	h.mux.HandleFunc("GET /api/v1/students/me/assignments", h.studentAssignments)
	h.mux.HandleFunc("POST /api/v1/students/me/assignments/{id}/session", h.startAssignment)
	h.mux.HandleFunc("GET /api/v1/students/me/sessions/{id}", h.survey)
	h.mux.HandleFunc("PUT /api/v1/students/me/sessions/{id}/consent", h.consent)
	h.mux.HandleFunc("PUT /api/v1/students/me/sessions/{id}/answers/{itemId}", h.answer)
	h.mux.HandleFunc("POST /api/v1/students/me/sessions/{id}/completion", h.completeSurvey)
	h.mux.HandleFunc("GET /api/v1/students/me/submissions/{id}", h.studentOnly(h.formSubmission))
	h.mux.HandleFunc("PUT /api/v1/students/me/submissions/{id}/consent", h.studentOnly(h.formConsent))
	h.mux.HandleFunc("PUT /api/v1/students/me/submissions/{id}/sections/{sectionId}", h.studentOnly(h.formSection))
	h.mux.HandleFunc("POST /api/v1/students/me/submissions/{id}/completion", h.studentOnly(h.formCompletion))
	h.mux.HandleFunc("GET /api/v1/students/me/submissions/{id}/suggestions", h.studentOnly(h.formSuggestions))
	h.mux.HandleFunc("GET /api/v1/forms/{formId}", h.teacherOnly(h.formDescription))
	h.mux.HandleFunc("GET /api/v1/forms/{formId}/overview", h.teacherOnly(h.formOverview))
	h.mux.HandleFunc("GET /api/v1/submissions/{id}", h.teacherOnly(h.formSubmissionDetail))
	h.mux.HandleFunc("GET /api/v1/question-bank", h.teacherOnly(h.questionBank))
	h.mux.HandleFunc("GET /api/v1/dashboard", h.teacherOnly(h.dashboard))
	h.mux.HandleFunc("GET /api/v1/templates", h.teacherOnly(h.templates))
	h.mux.HandleFunc("GET /api/v1/templates/{id}", h.teacherOnly(h.template))
	h.mux.HandleFunc("POST /api/v1/templates", h.teacherOnly(h.createTemplate))
	h.mux.HandleFunc("PUT /api/v1/templates/{id}", h.teacherOnly(h.updateTemplate))
	h.mux.HandleFunc("GET /api/v1/assignments", h.teacherOnly(h.assignments))
	h.mux.HandleFunc("POST /api/v1/assignments", h.teacherOnly(h.createAssignment))
	h.mux.HandleFunc("PATCH /api/v1/assignments/{id}", h.teacherOnly(h.updateAssignment))
	h.mux.HandleFunc("GET /api/v1/sessions", h.teacherOnly(h.sessions))
	h.mux.HandleFunc("GET /api/v1/sessions/{id}/report", h.teacherOnly(h.sessionReport))
	h.mux.HandleFunc("GET /api/v1/sessions/{id}/export", h.teacherOnly(h.exportSession))
	h.mux.HandleFunc("POST /api/v1/sessions/import", h.teacherOnly(h.importSession))
	h.mux.HandleFunc("POST /api/v1/sessions/{id}/reopen", h.teacherOnly(h.reopenSession))
	h.mux.HandleFunc("POST /api/v1/sessions/{id}/revoke", h.teacherOnly(h.revokeSession))
	h.mux.HandleFunc("PUT /api/v1/sessions/{id}/plan", h.teacherOnly(h.updatePlan))
	h.mux.HandleFunc("GET /api/v1/reports/groups", h.teacherOnly(h.groupReport))
	h.mux.HandleFunc("GET /api/v1/exports/answers.csv", h.teacherOnly(h.exportCSV))
	h.mux.HandleFunc("GET /api/v1/openapi.json", h.openapi)
	h.mux.HandleFunc("GET /swagger/{$}", func(w http.ResponseWriter, r *http.Request) { h.serveAsset(w, "pages/swagger.html") })
	// Keep unknown API paths inside the API (never fall through to an HTML page).
	h.mux.HandleFunc("/api/", h.apiNotFound)

}
func (h *Handler) teacherOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.authorized(w, r) {
			next(w, r)
		}
	}
}

func (h *Handler) apiNotFound(w http.ResponseWriter, r *http.Request) {
	allowed := []string{}
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
		probe := r.Clone(r.Context())
		probe.Method = method
		_, pattern := h.mux.Handler(probe)
		if pattern != "/api/" && pattern != "" && pattern != "/" {
			allowed = append(allowed, method)
		}
	}
	if len(allowed) > 0 {
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		h.reply(w, map[string]any{"error": "Метод не поддерживается."}, 405)
		return
	}
	h.reply(w, map[string]any{"error": "Маршрут API не найден."}, 404)
}
