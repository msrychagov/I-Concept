package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/questionbank"
	"vucdiagnostic/internal/repository/jsonfile"
	"vucdiagnostic/internal/usecase"
	"vucdiagnostic/web"
)

func callAccount(t *testing.T, a *testApplication, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	method := "GET"
	var b []byte
	if body != nil {
		method = "POST"
		b, _ = json.Marshal(body)
	}
	if explicit, rest, ok := strings.Cut(path, " "); ok {
		method, path = explicit, rest
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func requireCode(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("status %d want %d: %s", w.Code, code, w.Body.String())
	}
}
func decoded(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestRegistrationAssignmentsPrivacyAndRoles(t *testing.T) {
	a := testApp(t)
	signup := func(endpoint, name, email, platoon string) *http.Cookie {
		w := callAccount(t, a, endpoint, map[string]any{"name": name, "email": email, "platoon": platoon, "password": "TestPassword2026!"}, nil)
		requireCode(t, w, 201)
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly {
			t.Fatal("missing protected cookie")
		}
		return cookies[0]
	}
	teacher := signup("/api/v1/auth/teacher/register", "Учитель Тестов", "teacher@example.test", "")
	first := signup("/api/v1/auth/student/register", "Студент Первый", "first@example.test", "101")
	other := signup("/api/v1/auth/student/register", "Студент Второй", "second@example.test", "102")
	requireCode(t, callAccount(t, a, "/api/v1/templates", nil, first), 401)
	requireCode(t, callAccount(t, a, "/assets/bank.json", nil, first), 401)
	requireCode(t, callAccount(t, a, "/api/v1/teachers/me", nil, first), 401)
	p := a.emptyProfile()
	p.ID = "template"
	p.Author = "Тест"
	if err := a.addProfile(p); err != nil {
		t.Fatal(err)
	}
	requireCode(t, callAccount(t, a, "/api/v1/assignments", map[string]any{"title": "Проверка", "profileId": p.ID, "platoon": "101"}, first), 401)
	requireCode(t, callAccount(t, a, "/api/v1/assignments", map[string]any{"title": "Проверка", "profileId": p.ID, "platoon": "101"}, teacher), 201)
	id := a.repo.Snapshot().Assignments[0].ID
	list := decoded(t, callAccount(t, a, "/api/v1/students/me/assignments", nil, other))
	if len(list["assignments"].([]any)) != 0 {
		t.Fatal("wrong platoon sees assignment")
	}
	requireCode(t, callAccount(t, a, "POST /api/v1/students/me/assignments/"+id+"/session", nil, other), 403)
	start := callAccount(t, a, "POST /api/v1/students/me/assignments/"+id+"/session", nil, first)
	requireCode(t, start, 200)
	url := decoded(t, start)["url"].(string)
	sessionID := strings.Split(url, "id=")[1]
	requireCode(t, callAccount(t, a, "POST /api/v1/students/me/assignments/"+id+"/session", nil, first), 200)
	if len(a.repo.Snapshot().Sessions) != 1 {
		t.Fatal("duplicate attempt")
	}
	requireCode(t, callAccount(t, a, "/api/v1/students/me/sessions/"+sessionID, nil, other), 403)
	survey := callAccount(t, a, "/api/v1/students/me/sessions/"+sessionID, nil, first)
	requireCode(t, survey, 200)
	if strings.Contains(survey.Body.String(), "rationale") || strings.Contains(survey.Body.String(), "\"score\"") {
		t.Fatal("answer keys exposed")
	}
	requireCode(t, callAccount(t, a, "PUT /api/v1/students/me/sessions/"+sessionID+"/consent", map[string]any{"value": true, "revision": 0}, first), 200)
	requireCode(t, callAccount(t, a, "PUT /api/v1/students/me/sessions/"+sessionID+"/answers/I001", map[string]any{"value": 5, "note": "", "revision": 1}, first), 200)
	requireCode(t, callAccount(t, a, "/api/v1/students/me/sessions/"+sessionID+"/completion", map[string]any{"allowPartial": true, "revision": 2}, first), 200)
	completed := callAccount(t, a, "/api/v1/students/me/assignments", nil, first)
	requireCode(t, completed, 200)
	for _, word := range []string{"\"answers\"", "\"rows\"", "\"report\"", "passwordHash", "\"profile\""} {
		if strings.Contains(completed.Body.String(), word) {
			t.Fatalf("private data leaked: %s", word)
		}
	}
	requireCode(t, callAccount(t, a, "/api/v1/sessions/"+sessionID+"/report", nil, first), 401)
	requireCode(t, callAccount(t, a, "/api/v1/sessions/"+sessionID+"/report", nil, teacher), 200)
	requireCode(t, callAccount(t, a, "/api/v1/students/me/sessions/"+sessionID, nil, first), 403)
	requireCode(t, callAccount(t, a, "POST /api/v1/auth/student/logout", nil, first), 200)
	requireCode(t, callAccount(t, a, "/api/v1/students/me/assignments", nil, first), 401)
	bad := callAccount(t, a, "/api/v1/auth/teacher/login", map[string]any{"email": "teacher@example.test", "password": "wrong"}, nil)
	requireCode(t, bad, 401)
	good := callAccount(t, a, "/api/v1/auth/teacher/login", map[string]any{"email": "teacher@example.test", "password": "TestPassword2026!"}, nil)
	requireCode(t, good, 200)
	// A student registering later receives existing assignments for their platoon.
	later := signup("/api/v1/auth/student/register", "Студент Поздний", "later@example.test", "101")
	if len(decoded(t, callAccount(t, a, "/api/v1/students/me/assignments", nil, later))["assignments"].([]any)) != 1 {
		t.Fatal("late registration missed assignment")
	}
	requireCode(t, callAccount(t, a, "PATCH /api/v1/assignments/"+id, map[string]any{"active": false}, teacher), 200)
	requireCode(t, callAccount(t, a, "POST /api/v1/students/me/assignments/"+id+"/session", nil, later), 403)
	requireCode(t, callAccount(t, a, "/api/v1/assignments", map[string]any{"title": "Для всех", "profileId": p.ID, "platoon": ""}, teacher), 201)
	if len(decoded(t, callAccount(t, a, "/api/v1/students/me/assignments", nil, other))["assignments"].([]any)) != 1 {
		t.Fatal("all-platoon assignment missing")
	}
}

func TestDemoBackup(t *testing.T) {
	a := testApp(t)
	p := a.emptyProfile()
	p.ID = "old"
	a.addProfile(p)
	backup, err := a.ResetDemo()
	if err != nil || backup == "" {
		t.Fatalf("backup failed %s %v", backup, err)
	}
	db := a.repo.Snapshot()
	if len(db.Students) != 10 || len(db.Assignments) != 3 || len(db.Profiles) != 2 {
		t.Fatal("demo incomplete")
	}
	old, err := jsonfile.Open(backup)
	if err != nil || old.Snapshot().Profiles[0].ID != "old" {
		t.Fatal("old data not preserved")
	}
}

type testApplication struct {
	*Handler
	repo *jsonfile.Repository
}

func testApp(t *testing.T) *testApplication {
	t.Helper()
	bank, err := questionbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	repo, err := jsonfile.Open(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	previous, err := questionbank.Previous()
	if err != nil {
		t.Fatal(err)
	}
	service := usecase.New(bank, repo, previous...)
	return &testApplication{Handler: New(service, web.Files, Config{AdminUser: "admin", AdminPassword: "secret"}), repo: repo}
}
func (a *testApplication) emptyProfile() model.Profile {
	data, _ := a.service.ResearchQuery("state", "", "")
	return data.(map[string]any)["emptyProfile"].(model.Profile)
}
func (a *testApplication) addProfile(p model.Profile) error {
	return a.repo.Update(func(db *model.Database) error { db.Profiles = append(db.Profiles, p); return nil })
}
func (a *testApplication) ResetDemo() (string, error) { return a.service.ResetDemo() }
