package httptransport

import (
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"vucdiagnostic/web"
)

func TestOpenAPIPathsAndReferences(t *testing.T) {
	a := testApp(t)
	response := callAccount(t, a, "/api/v1/openapi.json", nil, nil)
	requireCode(t, response, 200)
	var spec map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Fatal("unexpected OpenAPI version")
	}
	ids := map[string]bool{}
	for path, operations := range spec["paths"].(map[string]any) {
		for method, raw := range operations.(map[string]any) {
			operation := raw.(map[string]any)
			id := operation["operationId"].(string)
			if ids[id] {
				t.Fatalf("duplicate operationId %s", id)
			}
			ids[id] = true
			concrete := strings.ReplaceAll(path, "{role}", "teacher")
			concrete = regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(concrete, "example")
			request := httptest.NewRequest(strings.ToUpper(method), "/api/v1"+concrete, nil)
			_, pattern := a.mux.Handler(request)
			if pattern != strings.ToUpper(method)+" /api/v1"+path {
				t.Fatalf("documented operation has no matching route: %s %s => %s", method, path, pattern)
			}
		}
	}
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			if raw, ok := v["$ref"]; ok {
				path := raw.(string)
				var target any = spec
				for _, key := range strings.Split(strings.TrimPrefix(path, "#/"), "/") {
					object, ok := target.(map[string]any)
					if !ok {
						t.Fatalf("invalid reference %s", path)
					}
					target, ok = object[key]
					if !ok {
						t.Fatalf("unresolved reference %s", path)
					}
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(spec)
	requireCode(t, callAccount(t, a, "/swagger/", nil, nil), 200)
	for _, name := range []string{"swagger-ui-bundle.js", "swagger-ui.css"} {
		if _, err := web.Files.ReadFile("static/vendor/swagger-ui/" + name); err != nil {
			t.Fatal(err)
		}
	}
}
func TestRESTValidationAndMethodHandling(t *testing.T) {
	a := testApp(t)
	bad := callAccount(t, a, "/api/v1/auth/teacher/login", map[string]any{"email": "x@y.test", "password": "secret", "op": "login"}, nil)
	requireCode(t, bad, 400)
	requireCode(t, callAccount(t, a, "/api/v1/auth/teacher/login", nil, nil), 405)
	requireCode(t, callAccount(t, a, "/api/vuc?op=state", nil, nil), 404)
	requireCode(t, callAccount(t, a, "/api/v1/unknown", nil, nil), 404)
	requireCode(t, callAccount(t, a, "/api/v1/auth/admin/login", map[string]any{"email": "x@y.test", "password": "secret"}, nil), 404)
}
func TestTemplateRESTLifecycle(t *testing.T) {
	a := testApp(t)
	registration := callAccount(t, a, "/api/v1/auth/teacher/register", map[string]any{"name": "Автор Шаблона", "email": "author@example.test", "invite": testInvite(t, a, "teacher", "author@example.test", ""), "password": "TestPassword2026!"}, nil)
	requireCode(t, registration, 201)
	cookie := registration.Result().Cookies()[0]
	profile := a.emptyProfile()
	profile.Author = "Автор Шаблона"
	created := callAccount(t, a, "/api/v1/templates", profile, cookie)
	requireCode(t, created, 201)
	id := decoded(t, created)["id"].(string)
	if created.Header().Get("Location") != "/api/v1/templates/"+id {
		t.Fatal("missing Location header")
	}
	requireCode(t, callAccount(t, a, "/api/v1/templates/"+id, nil, cookie), 200)
	profile.ID = "ignored-body-id"
	profile.Name = "Обновлённый шаблон"
	updated := callAccount(t, a, "PUT /api/v1/templates/"+id, profile, cookie)
	requireCode(t, updated, 200)
	if decoded(t, updated)["id"] != id {
		t.Fatal("body id overrides route id")
	}
	requireCode(t, callAccount(t, a, "/api/v1/templates/missing", nil, cookie), 404)
	requireCode(t, callAccount(t, a, "PATCH /api/v1/assignments/missing", map[string]any{}, cookie), 400)
}
