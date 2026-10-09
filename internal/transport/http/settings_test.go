package httptransport

import (
	"net/http"
	"strings"
	"testing"
)

func TestTeacherSettingsAuthorizationAndPasswordChange(t *testing.T) {
	a := testApp(t)
	register := func(endpoint, email string) *http.Cookie {
		response := callAccount(t, a, endpoint, map[string]any{"name": "Тестовый Пользователь", "platoon": "101", "email": email, "password": "TestPassword2026!"}, nil)
		requireCode(t, response, 201)
		return response.Result().Cookies()[0]
	}
	teacher := register("/api/v1/auth/teacher/register", "teacher@example.test")
	student := register("/api/v1/auth/student/register", "student@example.test")
	other := register("/api/v1/auth/teacher/register", "other@example.test")
	_ = other
	body := map[string]any{"name": "Новое Имя", "email": "new@example.test", "currentPassword": "TestPassword2026!"}
	requireCode(t, callAccount(t, a, "PATCH /api/v1/teachers/me", body, nil), 401)
	requireCode(t, callAccount(t, a, "PATCH /api/v1/teachers/me", body, student), 401)
	requireCode(t, callAccount(t, a, "PATCH /api/v1/students/me", body, student), 405)
	body["currentPassword"] = "wrong"
	requireCode(t, callAccount(t, a, "PATCH /api/v1/teachers/me", body, teacher), 400)
	body["currentPassword"] = "TestPassword2026!"
	body["email"] = "other@example.test"
	requireCode(t, callAccount(t, a, "PATCH /api/v1/teachers/me", body, teacher), 409)
	body["email"] = "new@example.test"
	requireCode(t, callAccount(t, a, "PATCH /api/v1/teachers/me", body, teacher), 200)
	second := callAccount(t, a, "/api/v1/auth/teacher/login", map[string]any{"email": "new@example.test", "password": "TestPassword2026!"}, nil)
	requireCode(t, second, 200)
	change := map[string]any{"currentPassword": "TestPassword2026!", "newPassword": "ReplacementPassword2026!", "confirmPassword": "ReplacementPassword2026!"}
	requireCode(t, callAccount(t, a, "PUT /api/v1/teachers/me/password", change, teacher), 200)
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me", nil, second.Result().Cookies()[0]), 401)
	public := callAccount(t, a, "/api/v1/teachers/me", nil, teacher)
	requireCode(t, public, 200)
	if strings.Contains(public.Body.String(), "passwordHash") || strings.Contains(public.Body.String(), "salt") {
		t.Fatal("private credentials leaked")
	}
	requireCode(t, callAccount(t, a, "/api/v1/auth/teacher/login", map[string]any{"email": "new@example.test", "password": "TestPassword2026!"}, nil), 401)
	requireCode(t, callAccount(t, a, "/api/v1/auth/teacher/login", map[string]any{"email": "new@example.test", "password": "ReplacementPassword2026!"}, nil), 200)
}
