package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/formbank"
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

func callBasic(t *testing.T, a *testApplication, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	r.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}

func testInvite(t *testing.T, a *testApplication, role, email, platoon string) string {
	t.Helper()
	issuerID := ""
	if role == "teacher" {
		platoon = ""
	}
	if role == "student" {
		for _, account := range a.repo.Snapshot().Students {
			if account.Role == "teacher" {
				issuerID = account.ID
				break
			}
		}
		if issuerID == "" {
			t.Fatal("student invitation needs a teacher")
		}
	}
	var value any
	var err error
	if role == "teacher" {
		value, err = a.service.CreateTeacherInvitation(commandInput(map[string]any{"email": email, "vuc": "Тестовый ВУЦ"}))
	} else {
		platoonID := ""
		for _, p := range a.repo.Snapshot().Platoons {
			if p.Name == platoon {
				platoonID = p.ID
				break
			}
		}
		if platoonID == "" {
			p, createErr := a.service.SavePlatoon(issuerID, "", platoon)
			if createErr != nil {
				t.Fatal(createErr)
			}
			platoonID = p.(model.Platoon).ID
		}
		value, err = a.service.CreateStudentRegistrationLink(issuerID, commandInput(map[string]any{"platoonId": platoonID}))
	}
	if err != nil {
		t.Fatal(err)
	}
	if role == "teacher" {
		return value.(map[string]any)["code"].(string)
	}
	return value.(map[string]any)["link"].(string)
}

func TestRegistrationAssignmentsPrivacyAndRoles(t *testing.T) {
	a := testApp(t)
	signup := func(endpoint, name, email, platoon string) *http.Cookie {
		role := "student"
		if strings.Contains(endpoint, "/teacher/") {
			role = "teacher"
		}
		w := callAccount(t, a, endpoint, map[string]any{"name": name, "email": email, "invite": testInvite(t, a, role, email, platoon), "password": "TestPassword2026!"}, nil)
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

func TestClosedInvitationsAndTeacherIsolation(t *testing.T) {
	a := testApp(t)
	requireCode(t, callAccount(t, a, "/api/v1/admin/vucs", map[string]any{"name": "Новый ВУЦ"}, nil), 401)
	requireCode(t, callBasic(t, a, http.MethodPost, "/api/v1/admin/invitations/teachers", map[string]any{"email": "missing-center@example.test", "vuc": "Новый ВУЦ"}), 404)
	requireCode(t, callBasic(t, a, http.MethodPost, "/api/v1/admin/vucs", map[string]any{"name": "Новый ВУЦ"}), 201)
	requireCode(t, callBasic(t, a, http.MethodPost, "/api/v1/admin/vucs", map[string]any{"name": "  новый   вуц  "}), 409)
	requireCode(t, callBasic(t, a, http.MethodGet, "/api/v1/admin/vucs", nil), 200)
	requireCode(t, callBasic(t, a, http.MethodPost, "/api/v1/admin/invitations/teachers", map[string]any{"email": "missing-center@example.test", "vuc": "новый вуц"}), 201)
	open := callAccount(t, a, "/api/v1/auth/teacher/register", map[string]any{"name": "Чужой Пользователь", "email": "outsider@example.test", "password": "TestPassword2026!", "invite": "ABCDE-FGHJK"}, nil)
	requireCode(t, open, 403)
	requireCode(t, callAccount(t, a, "/api/v1/admin/invitations/teachers", map[string]any{"email": "first-teacher@example.test"}, nil), 401)

	issued := callBasic(t, a, http.MethodPost, "/api/v1/admin/invitations/teachers", map[string]any{"email": "first-teacher@example.test", "vuc": "Тестовый ВУЦ"})
	requireCode(t, issued, 201)
	code := decoded(t, issued)["code"].(string)
	if len(code) != 11 || code[5] != '-' || strings.Contains(string(mustJSON(t, a.repo.Snapshot())), code) {
		t.Fatal("invitation code is not short or was persisted in plaintext")
	}
	firstTeacher := callAccount(t, a, "/api/v1/auth/teacher/register", map[string]any{"name": "Первый Преподаватель", "email": "first-teacher@example.test", "password": "TestPassword2026!", "invite": strings.ToLower(strings.ReplaceAll(code, "-", ""))}, nil)
	requireCode(t, firstTeacher, 201)
	firstTeacherCookie := firstTeacher.Result().Cookies()[0]
	requireCode(t, callAccount(t, a, "/api/v1/admin/vucs", map[string]any{"name": "Чужой ВУЦ"}, firstTeacherCookie), 401)
	requireCode(t, callBasic(t, a, http.MethodPatch, "/api/v1/admin/teachers/"+decoded(t, firstTeacher)["account"].(map[string]any)["id"].(string)+"/vuc", map[string]any{"vuc": "Не существует"}), 404)
	// A teacher cookie cannot authorize creation of another teacher's code.
	requireCode(t, callAccount(t, a, "/api/v1/admin/invitations/teachers", map[string]any{"email": "forbidden@example.test"}, firstTeacherCookie), 401)
	requireCode(t, callAccount(t, a, "/api/v1/auth/teacher/register", map[string]any{"name": "Повторный Пользователь", "email": "first-teacher@example.test", "password": "TestPassword2026!", "invite": code}, nil), 403)
	// Basic Auth is administrative only and can no longer enter teacher APIs.
	requireCode(t, callBasic(t, a, http.MethodGet, "/api/v1/templates", nil), 401)

	firstPlatoon := callAccount(t, a, "/api/v1/teachers/me/platoons", map[string]any{"name": "101"}, firstTeacherCookie)
	requireCode(t, firstPlatoon, 200)
	firstPlatoonID := decoded(t, firstPlatoon)["id"]
	inviteStudent := callAccount(t, a, "/api/v1/teachers/me/student-registration-links", map[string]any{"platoonId": firstPlatoonID}, firstTeacherCookie)
	requireCode(t, inviteStudent, 201)
	studentLink := decoded(t, inviteStudent)["link"].(string)
	dbAfterLink := a.repo.Snapshot()
	studentGrant := dbAfterLink.Invitations[len(dbAfterLink.Invitations)-1]
	studentToken := strings.Split(studentLink, "invite=")[1]
	if studentGrant.Expires.Sub(studentGrant.Created) != 48*time.Hour || strings.Contains(string(mustJSON(t, dbAfterLink)), studentToken) {
		t.Fatal("student link lifetime or storage is unsafe")
	}
	firstStudent := callAccount(t, a, "/api/v1/auth/student/register", map[string]any{"name": "Первый Студент", "email": "first-student@example.test", "password": "TestPassword2026!", "invite": studentLink}, nil)
	requireCode(t, firstStudent, 201)
	firstStudentCookie := firstStudent.Result().Cookies()[0]
	studentAccount := decoded(t, firstStudent)["account"].(map[string]any)
	if studentAccount["platoon"] != "101" {
		t.Fatal("student platoon was not taken from invitation")
	}

	secondCode := testInvite(t, a, "teacher", "second-teacher@example.test", "")
	secondTeacher := callAccount(t, a, "/api/v1/auth/teacher/register", map[string]any{"name": "Второй Преподаватель", "email": "second-teacher@example.test", "password": "TestPassword2026!", "invite": secondCode}, nil)
	requireCode(t, secondTeacher, 201)
	secondCookie := secondTeacher.Result().Cookies()[0]
	// The platoon link is reusable during its 48-hour lifetime.
	requireCode(t, callAccount(t, a, "/api/v1/auth/student/register", map[string]any{"name": "Ещё Один Студент", "email": "another-first@example.test", "password": "TestPassword2026!", "invite": studentLink}, nil), 201)
	replacement := callAccount(t, a, "/api/v1/teachers/me/student-registration-links", map[string]any{"platoonId": firstPlatoonID}, firstTeacherCookie)
	requireCode(t, replacement, 201)
	requireCode(t, callAccount(t, a, "/api/v1/auth/student/register", map[string]any{"name": "Просроченный Студент", "email": "old-link@example.test", "password": "TestPassword2026!", "invite": studentLink}, nil), 403)

	secondPlatoon := callAccount(t, a, "/api/v1/teachers/me/platoons", map[string]any{"name": "202"}, secondCookie)
	requireCode(t, secondPlatoon, 200)
	secondInvite := callAccount(t, a, "/api/v1/teachers/me/student-registration-links", map[string]any{"platoonId": decoded(t, secondPlatoon)["id"]}, secondCookie)
	requireCode(t, secondInvite, 201)
	requireCode(t, callAccount(t, a, "/api/v1/auth/student/register", map[string]any{"name": "Второй Студент", "email": "second-student@example.test", "password": "TestPassword2026!", "invite": decoded(t, secondInvite)["link"]}, nil), 201)

	firstDashboard := decoded(t, callAccount(t, a, "/api/v1/dashboard", nil, firstTeacherCookie))
	secondDashboard := decoded(t, callAccount(t, a, "/api/v1/dashboard", nil, secondCookie))
	if len(firstDashboard["students"].([]any)) != 2 || len(secondDashboard["students"].([]any)) != 1 {
		t.Fatal("teachers can see students outside their invitation scope")
	}

	profile := a.emptyProfile()
	profile.Author = "Первый Преподаватель"
	createdProfile := callAccount(t, a, "/api/v1/templates", profile, firstTeacherCookie)
	requireCode(t, createdProfile, 201)
	profileID := decoded(t, createdProfile)["id"].(string)
	secondTemplates := decoded(t, callAccount(t, a, "/api/v1/templates", nil, secondCookie))["profiles"].([]any)
	if len(secondTemplates) != 0 {
		t.Fatal("another teacher can see a foreign template")
	}
	requireCode(t, callAccount(t, a, "/api/v1/templates/"+profileID, nil, secondCookie), 404)
	createdAssignment := callAccount(t, a, "/api/v1/assignments", map[string]any{"title": "Только первый взвод", "profileId": profileID, "platoon": "101"}, firstTeacherCookie)
	requireCode(t, createdAssignment, 201)
	assignmentID := a.repo.Snapshot().Assignments[0].ID
	requireCode(t, callAccount(t, a, "PATCH /api/v1/assignments/"+assignmentID, map[string]any{"active": false}, secondCookie), 404)
	if len(decoded(t, callAccount(t, a, "/api/v1/students/me/assignments", nil, firstStudentCookie))["assignments"].([]any)) != 1 {
		t.Fatal("student cannot see their teacher's assignment")
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
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
	if len(db.Students) != 10 || len(db.Assignments) != 4 || len(db.Profiles) != 2 || len(db.Submissions) != 2 {
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
	forms, err := formbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	service := usecase.New(bank, repo, previous...)
	if _, err := service.CreateVUC("Тестовый ВУЦ"); err != nil {
		t.Fatal(err)
	}
	service.RegisterForms(forms...)
	return &testApplication{Handler: New(service, web.Files, Config{AdminUser: "admin", AdminPassword: "secret"}), repo: repo}
}
func (a *testApplication) emptyProfile() model.Profile {
	data, _ := a.service.ResearchQuery("state", "", "")
	return data.(map[string]any)["emptyProfile"].(model.Profile)
}
func (a *testApplication) addProfile(p model.Profile) error {
	return a.repo.Update(func(db *model.Database) error {
		for _, account := range db.Students {
			if account.Role == "teacher" {
				p.OwnerID = account.ID
				break
			}
		}
		db.Profiles = append(db.Profiles, p)
		return nil
	})
}
func (a *testApplication) ResetDemo() (string, error) { return a.service.ResetDemo() }
