package httptransport

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"vucdiagnostic/internal/usecase"
)

func (h *Handler) reply(w http.ResponseWriter, value any, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (h *Handler) respond(w http.ResponseWriter, value any, err error) {
	if err == nil {
		h.reply(w, value, 200)
		return
	}
	var applicationError *usecase.Error
	if errors.As(err, &applicationError) {
		codes := map[usecase.ErrorKind]int{usecase.Invalid: 400, usecase.Unauthenticated: 401, usecase.Forbidden: 403, usecase.NotFound: 404, usecase.Conflict: 409, usecase.Gone: 410, usecase.Incomplete: 422}
		code, ok := codes[applicationError.Kind]
		if !ok {
			code = 500
		}
		h.reply(w, applicationError.Details, code)
		return
	}
	log.Print(err)
	h.reply(w, map[string]any{"error": "Не удалось сохранить данные."}, 500)
}
func loginToken(r *http.Request, role string) string {
	cookie, err := r.Cookie("vuc_" + role)
	if err != nil {
		return ""
	}
	return cookie.Value
}
func (h *Handler) authorized(w http.ResponseWriter, r *http.Request) bool {
	if _, ok := h.service.CurrentAccount(loginToken(r, "teacher"), "teacher"); ok {
		return true
	}
	user, password, ok := r.BasicAuth()
	if !ok || h.config.AdminPassword == "" || subtle.ConstantTimeCompare([]byte(user), []byte(h.config.AdminUser)) != 1 || subtle.ConstantTimeCompare([]byte(password), []byte(h.config.AdminPassword)) != 1 {
		h.reply(w, map[string]any{"error": "Требуется вход исследователя."}, 401)
		return false
	}
	return true
}
func (h *Handler) validOrigin(w http.ResponseWriter, r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host && origin != "https://"+r.Host {
		h.reply(w, map[string]any{"error": "Недопустимый источник."}, 403)
		return false
	}
	return true
}
func (h *Handler) decodeBody(w http.ResponseWriter, r *http.Request, limit int64, target any) bool {
	if !h.validOrigin(w, r) {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		h.reply(w, map[string]any{"error": "Используйте Content-Type: application/json."}, 415)
		return false
	}
	if r.ContentLength > limit {
		h.reply(w, map[string]any{"error": "Слишком большой запрос."}, 413)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(target); err != nil {
		h.decodeError(w, err)
		return false
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		h.reply(w, map[string]any{"error": "Ожидается один JSON-объект."}, 400)
		return false
	}
	return true
}
func (h *Handler) decodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		h.reply(w, map[string]any{"error": "Слишком большой запрос."}, 413)
		return
	}
	h.reply(w, map[string]any{"error": "Некорректный JSON или неизвестные поля запроса."}, 400)
}
func (h *Handler) requiredRevision(w http.ResponseWriter, revision *int) bool {
	if revision == nil || *revision < 0 {
		h.reply(w, map[string]any{"error": "Укажите неотрицательную ревизию сессии (revision)."}, 400)
		return false
	}
	return true
}
