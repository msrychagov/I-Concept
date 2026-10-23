package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInstructorRoutes(t *testing.T) {
	a := testApp(t)
	email := "instructors@example.test"
	w := callAccount(t, a, "/api/v1/auth/teacher/register", map[string]any{"name": "Учитель Списков", "email": email, "invite": testInvite(t, a, "teacher", email, ""), "password": "TestPassword2026!"}, nil)
	requireCode(t, w, 201)
	teacher := w.Result().Cookies()[0]

	created := callAccount(t, a, "/api/v1/teachers/me/instructors", map[string]any{"surname": "Иванов", "name": "Игорь", "patronymic": "Петрович", "rank": "полковник"}, teacher)
	requireCode(t, created, 201)
	id := decoded(t, created)["id"].(string)
	requireCode(t, callAccount(t, a, "PUT /api/v1/teachers/me/instructors/"+id, map[string]any{"surname": "Иванов", "name": "Игорь", "rank": "генерал-майор"}, teacher), 200)
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me/instructors", map[string]any{"surname": "", "name": "Игорь"}, teacher), 400)
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me/instructors", map[string]any{"surname": "Иванов", "name": "Игорь", "unknown": 1}, teacher), 400)
	list := decoded(t, callAccount(t, a, "/api/v1/teachers/me/instructors", nil, teacher))
	entries := list["instructors"].([]any)
	if list["vuc"] != "Тестовый ВУЦ" || len(entries) != 1 || entries[0].(map[string]any)["rank"] != "генерал-майор" || entries[0].(map[string]any)["updatedByName"] != "Учитель Списков" {
		t.Fatalf("list: %+v", list)
	}
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me/instructors", nil, nil), 401)

	foreign := httptest.NewRequest(http.MethodDelete, "/api/v1/teachers/me/instructors/"+id, nil)
	foreign.Header.Set("Origin", "https://evil.example")
	foreign.AddCookie(teacher)
	response := httptest.NewRecorder()
	a.ServeHTTP(response, foreign)
	requireCode(t, response, 403)
	requireCode(t, callAccount(t, a, "DELETE /api/v1/teachers/me/instructors/"+id, nil, teacher), 200)
	requireCode(t, callAccount(t, a, "DELETE /api/v1/teachers/me/instructors/"+id, nil, teacher), 404)
}

func TestCohortRoutes(t *testing.T) {
	a := testApp(t)
	email := "cohorts@example.test"
	w := callAccount(t, a, "/api/v1/auth/teacher/register", map[string]any{"name": "Учитель Выпусков", "email": email, "invite": testInvite(t, a, "teacher", email, ""), "password": "TestPassword2026!"}, nil)
	requireCode(t, w, 201)
	teacher := w.Result().Cookies()[0]
	platoon := decoded(t, callAccount(t, a, "/api/v1/teachers/me/platoons", map[string]any{"name": "2401"}, teacher))["id"].(string)

	created := callAccount(t, a, "/api/v1/teachers/me/cohorts", map[string]any{"name": "Выпуск 2027", "year": 2027, "platoonIds": []string{platoon}}, teacher)
	requireCode(t, created, 201)
	id := decoded(t, created)["id"].(string)
	requireCode(t, callAccount(t, a, "PUT /api/v1/teachers/me/cohorts/"+id, map[string]any{"name": "Выпуск 2027-А", "year": 2027, "platoonIds": []string{platoon}}, teacher), 200)
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me/cohorts", map[string]any{"name": "", "year": 2027, "platoonIds": []string{}}, teacher), 400)
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me/cohorts", map[string]any{"name": "Другой", "year": 2028, "platoonIds": []string{platoon}}, teacher), 409)
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me/cohorts", map[string]any{"name": "Другой", "year": 2028, "platoonIds": []string{}, "unknown": 1}, teacher), 400)
	list := decoded(t, callAccount(t, a, "/api/v1/teachers/me/cohorts", nil, teacher))
	entries := list["cohorts"].([]any)
	if len(entries) != 1 || entries[0].(map[string]any)["name"] != "Выпуск 2027-А" || entries[0].(map[string]any)["mine"] != true {
		t.Fatalf("list: %+v", list)
	}
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me/cohorts", nil, nil), 401)
	requireCode(t, callAccount(t, a, "DELETE /api/v1/teachers/me/cohorts/"+id, nil, teacher), 200)
	requireCode(t, callAccount(t, a, "DELETE /api/v1/teachers/me/cohorts/"+id, nil, teacher), 404)
}
