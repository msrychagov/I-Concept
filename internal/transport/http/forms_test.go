package httptransport

import (
	"net/http"
	"strings"
	"testing"
)

// minimal answers that complete every section: each opening question is answered "no".
var minimalAnswers = map[string]map[string]any{
	"about":        {"name_changed": "no", "birth_date": "2005-03-15", "birth_country": "RU", "birth_region": "77", "birth_place": "Москва", "nationality": "russkie"},
	"education":    {"education_list": []map[string]any{{"edu_type": "school", "edu_profile": "none", "edu_region": "77", "school_city": "Москва", "school_name": "Школа 57", "edu_year": "2023", "edu_medal": "no"}}},
	"hse":          {"hse_campus": "msk", "hse_level": "bakalavriat", "hse_faculty_msk": "fakultet_kompyuternykh_nauk", "hse_program": "Прикладная математика и информатика", "hse_start": "2023-09", "hse_end": "2027-06"},
	"work":         {"work_has": "no"},
	"military":     {"served": "no", "vk_region": "77", "vk_district": "Тверской район", "vk_same": "yes"},
	"languages":    {"lang_has": "no"},
	"family":       {"marital": "single", "children_has": "no"},
	"relatives":    {"upbringing": "both", "rel_has": "no"},
	"special":      {"trips_has": "no", "crim_has": "no"},
	"awards":       {"awards_has": "no"},
	"address":      {"reg_region": "77", "reg_address": "Москва, ул. Тверская, 1", "lives_at_reg": "yes"},
	"sport":        {"sport_has": "no"},
	"creative":     {"creative_has": "no"},
	"patriotic":    {"patriot_has": "no"},
	"science":      {"science_has": "no"},
	"volunteering": {"vol_has": "no"},
	"hobbies":      {},
}
var sectionOrder = []string{"about", "education", "hse", "work", "military", "languages", "family", "relatives", "special", "awards", "address", "sport", "creative", "patriotic", "science", "volunteering", "hobbies"}

func signupCookie(t *testing.T, a *testApplication, role, name, email, platoon string) *http.Cookie {
	t.Helper()
	w := callAccount(t, a, "/api/v1/auth/"+role+"/register", map[string]any{"name": name, "email": email, "invite": testInvite(t, a, role, email, platoon), "password": "TestPassword2026!"}, nil)
	requireCode(t, w, 201)
	return w.Result().Cookies()[0]
}

type formScenario struct {
	a                *testApplication
	teacher, student *http.Cookie
	assignment       string
	submission       string
	revision         int
}

func newFormScenario(t *testing.T) *formScenario {
	t.Helper()
	a := testApp(t)
	s := &formScenario{a: a}
	s.teacher = signupCookie(t, a, "teacher", "Учитель Тестов", "teacher@example.test", "")
	s.student = signupCookie(t, a, "student", "Студент Первый", "first@example.test", "101")
	requireCode(t, callAccount(t, a, "/api/v1/assignments", map[string]any{"title": "Автобиография", "formId": "autobiography", "platoon": "101"}, s.teacher), 201)
	s.assignment = a.repo.Snapshot().Assignments[0].ID
	start := callAccount(t, a, "POST /api/v1/students/me/assignments/"+s.assignment+"/session", nil, s.student)
	requireCode(t, start, 200)
	url := decoded(t, start)["url"].(string)
	if !strings.HasPrefix(url, "/autobiography?id=") {
		t.Fatalf("url %s", url)
	}
	s.submission = strings.Split(url, "id=")[1]
	return s
}

func (s *formScenario) path(rest string) string {
	return "/api/v1/students/me/submissions/" + s.submission + rest
}

func (s *formScenario) put(t *testing.T, path string, body map[string]any) {
	t.Helper()
	body["revision"] = s.revision
	w := callAccount(t, s.a, "PUT "+s.path(path), body, s.student)
	requireCode(t, w, 200)
	s.revision = int(decoded(t, w)["revision"].(float64))
}

func TestFormRoutesNeedTheRightLogin(t *testing.T) {
	s := newFormScenario(t)
	stranger := signupCookie(t, s.a, "student", "Студент Второй", "second@example.test", "101")
	for _, route := range []string{"GET " + s.path(""), "GET " + s.path("/suggestions?field=school_name&q=ш")} {
		requireCode(t, callAccount(t, s.a, route, nil, nil), 401)
		requireCode(t, callAccount(t, s.a, route, nil, s.teacher), 401)
		requireCode(t, callAccount(t, s.a, route, nil, stranger), 403)
	}
	body := map[string]any{"value": true, "revision": 0}
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/consent"), body, nil), 401)
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/consent"), body, s.teacher), 401)
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/consent"), body, stranger), 403)
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/sections/about"), map[string]any{"answers": map[string]any{}, "revision": 0}, s.teacher), 401)
	requireCode(t, callAccount(t, s.a, s.path("/completion"), map[string]any{"revision": 0}, s.teacher), 401)
	for _, route := range []string{"/api/v1/forms/autobiography", "/api/v1/forms/autobiography/overview", "/api/v1/submissions/" + s.submission} {
		requireCode(t, callAccount(t, s.a, route, nil, nil), 401)
		requireCode(t, callAccount(t, s.a, route, nil, s.student), 401)
		requireCode(t, callAccount(t, s.a, route, nil, s.teacher), 200)
	}
	requireCode(t, callAccount(t, s.a, "/api/v1/forms/unknown", nil, s.teacher), 404)
	requireCode(t, callAccount(t, s.a, "/api/v1/submissions/unknown", nil, s.teacher), 404)
	requireCode(t, callAccount(t, s.a, "/api/v1/assignments", map[string]any{"title": "X", "formId": "unknown"}, s.teacher), 400)
}

func TestFormSectionValidationAndLimits(t *testing.T) {
	s := newFormScenario(t)
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/sections/about"), map[string]any{"answers": map[string]any{"name_changed": "no"}, "revision": 0}, s.student), 400)
	s.put(t, "/consent", map[string]any{"value": true})
	bad := callAccount(t, s.a, "PUT "+s.path("/sections/about"), map[string]any{"answers": map[string]any{"birth_date": "2005-02-30"}, "revision": s.revision}, s.student)
	requireCode(t, bad, 400)
	if decoded(t, bad)["field"] != "birth_date" {
		t.Fatalf("no field in the error: %s", bad.Body.String())
	}
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/sections/about"), map[string]any{"revision": s.revision}, s.student), 400)
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/sections/about"), map[string]any{"answers": map[string]any{}, "revision": s.revision + 5}, s.student), 409)
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/sections/about"), map[string]any{"answers": map[string]any{}, "revision": s.revision, "extra": 1}, s.student), 400)
	huge := map[string]any{"answers": map[string]any{"prev_names": strings.Repeat("я", 40000)}, "revision": s.revision}
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/sections/about"), huge, s.student), 413)
}

func TestAutobiographyEndToEnd(t *testing.T) {
	s := newFormScenario(t)
	open := callAccount(t, s.a, s.path(""), nil, s.student)
	requireCode(t, open, 200)
	if !strings.Contains(open.Body.String(), `"refs"`) || decoded(t, open)["progress"] == nil {
		t.Fatal("form description or progress missing")
	}
	s.put(t, "/consent", map[string]any{"value": true})
	incomplete := callAccount(t, s.a, s.path("/completion"), map[string]any{"revision": s.revision}, s.student)
	requireCode(t, incomplete, 422)
	for n, id := range sectionOrder {
		s.put(t, "/sections/"+id, map[string]any{"answers": minimalAnswers[id]})
		state := decoded(t, callAccount(t, s.a, s.path(""), nil, s.student))
		complete := state["progress"].(map[string]any)["complete"].([]any)
		if len(complete) != n+1 {
			t.Fatalf("after %s: %d sections complete (%v)", id, len(complete), state["progress"])
		}
	}
	suggestions := decoded(t, callAccount(t, s.a, s.path("/suggestions?field=school_name&q=ш"), nil, s.student))
	if len(suggestions["values"].([]any)) != 0 {
		t.Fatalf("own answers suggested back: %v", suggestions)
	}
	requireCode(t, callAccount(t, s.a, s.path("/suggestions?field=birth_place&q=ш"), nil, s.student), 400)
	done := callAccount(t, s.a, s.path("/completion"), map[string]any{"revision": s.revision}, s.student)
	requireCode(t, done, 200)
	if decoded(t, done)["completed"] != true {
		t.Fatal("not completed")
	}

	requireCode(t, callAccount(t, s.a, s.path(""), nil, s.student), 403)
	requireCode(t, callAccount(t, s.a, "PUT "+s.path("/sections/about"), map[string]any{"answers": map[string]any{}, "revision": s.revision + 1}, s.student), 403)
	requireCode(t, callAccount(t, s.a, "POST /api/v1/students/me/assignments/"+s.assignment+"/session", nil, s.student), 409)

	list := callAccount(t, s.a, "/api/v1/students/me/assignments", nil, s.student)
	requireCode(t, list, 200)
	for _, word := range []string{"Москва", "Тверской", "answers", "birth_date", "Школа 57"} {
		if strings.Contains(list.Body.String(), word) {
			t.Fatalf("answers leaked into the cabinet: %s", word)
		}
	}
	entry := decoded(t, list)["assignments"].([]any)[0].(map[string]any)
	if entry["status"] != "Завершено" || entry["formId"] != "autobiography" || entry["completed"] != true {
		t.Fatalf("cabinet entry: %v", entry)
	}

	overview := decoded(t, callAccount(t, s.a, "/api/v1/forms/autobiography/overview", nil, s.teacher))
	submissions := overview["submissions"].([]any)
	if len(submissions) != 1 || submissions[0].(map[string]any)["completed"] != true || submissions[0].(map[string]any)["answered"].(float64) != 17 {
		t.Fatalf("overview: %v", submissions)
	}
	detail := callAccount(t, s.a, "/api/v1/submissions/"+s.submission, nil, s.teacher)
	requireCode(t, detail, 200)
	for _, word := range []string{"Школа 57", "Тверской район", "Студент Первый"} {
		if !strings.Contains(detail.Body.String(), word) {
			t.Fatalf("teacher does not see %q", word)
		}
	}
	tests := decoded(t, callAccount(t, s.a, "/api/v1/dashboard", nil, s.teacher))
	if len(tests["assignments"].([]any)) != 0 {
		t.Fatalf("form assignment on the tests dashboard: %v", tests["assignments"])
	}
}
