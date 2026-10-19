package httptransport

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"vucdiagnostic/internal/promptbank"
	"vucdiagnostic/internal/usecase"
)

type countingModel struct {
	calls atomic.Int32
	reply func(user string) string
}

func (m *countingModel) Name() string { return "fake-model" }
func (m *countingModel) Complete(_ context.Context, c usecase.Completion) (usecase.Reply, error) {
	m.calls.Add(1)
	return usecase.Reply{Text: m.reply(c.User), Version: "fake-1", In: 1, Out: 1}, nil
}

// portraitAnswer is an answer that follows the instruction for the given input.
func portraitAnswer(user string) string {
	var input struct {
		Levels map[string]string `json:"уровни"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	potential := []any{}
	for _, area := range []string{"спорт", "музыка и творчество", "военно-патриотическая деятельность", "наука", "волонтёрство", "иностранные языки"} {
		potential = append(potential, map[string]any{"area": area, "level": input.Levels[area], "facts": "Сведений нет", "conclusion": "Опыт не указан"})
	}
	data, _ := json.Marshal(map[string]any{"summary": "Краткая характеристика.", "potential": potential, "attention": map[string]any{"use": []any{}, "consider": []any{"Факт."}}})
	return string(data)
}

func (s *formScenario) submit(t *testing.T) {
	t.Helper()
	s.put(t, "/consent", map[string]any{"value": true})
	for _, id := range sectionOrder {
		s.put(t, "/sections/"+id, map[string]any{"answers": minimalAnswers[id]})
	}
	requireCode(t, callAccount(t, s.a, s.path("/completion"), map[string]any{"revision": s.revision}, s.student), 200)
}

func (s *formScenario) portrait(t *testing.T, method string, cookie bool) (int, map[string]any) {
	t.Helper()
	c := s.teacher
	if !cookie {
		c = nil
	}
	w := callAccount(t, s.a, method+" /api/v1/portraits/autobiography/"+s.submission, nil, c)
	return w.Code, decoded(t, w)
}

func (s *formScenario) waitFor(t *testing.T, status string) map[string]any {
	t.Helper()
	// Three attempts with the real pause between them take over four seconds.
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		if _, view := s.portrait(t, "GET", true); view["status"] == status {
			return view
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, view := s.portrait(t, "GET", true)
	t.Fatalf("status %v, want %s", view, status)
	return nil
}

func TestPortraitRoutes(t *testing.T) {
	s := newFormScenario(t)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.a.service.RegisterPrompts(prompts...)
	m := &countingModel{reply: portraitAnswer}
	s.a.service.UseModel(m)
	defer s.a.service.Close()

	// A draft has no portrait.
	if code, _ := s.portrait(t, "GET", true); code != 404 {
		t.Fatalf("draft: %d", code)
	}
	s.submit(t)
	view := s.waitFor(t, "ready")
	portrait := view["portrait"].(map[string]any)
	output := portrait["output"].(map[string]any)
	if output["summary"] != "Краткая характеристика." || len(output["potential"].([]any)) != 6 || portrait["created"] == "" || m.calls.Load() != 1 {
		t.Fatalf("view %v after %d calls", view, m.calls.Load())
	}

	// Only the teacher of the assignment sees it.
	if code, _ := s.portrait(t, "GET", false); code != 401 {
		t.Fatalf("no login: %d", code)
	}
	requireCode(t, callAccount(t, s.a, "/api/v1/portraits/autobiography/"+s.submission, nil, s.student), 401)
	other := signupCookie(t, s.a, "teacher", "Другой Учитель", "other@example.test", "")
	requireCode(t, callAccount(t, s.a, "/api/v1/portraits/autobiography/"+s.submission, nil, other), 404)
	requireCode(t, callAccount(t, s.a, "POST /api/v1/portraits/autobiography/"+s.submission, nil, other), 404)
	requireCode(t, callAccount(t, s.a, "/api/v1/portraits/nope/"+s.submission, nil, s.teacher), 404)
	requireCode(t, callAccount(t, s.a, "/api/v1/portraits/autobiography/missing", nil, s.teacher), 404)

	// «Повторить» on a ready portrait returns it and does not call the model.
	code, again := s.portrait(t, "POST", true)
	if code != 200 || again["status"] != "ready" || m.calls.Load() != 1 {
		t.Fatalf("retry: %d %v after %d calls", code, again, m.calls.Load())
	}
	if strings.Contains(callAccount(t, s.a, "/api/v1/portraits/autobiography/"+s.submission, nil, s.teacher).Body.String(), "возраст") {
		t.Fatal("the input of the model is shown")
	}
}

func TestPortraitStatusOffWithoutModelAndFailedUntilRetried(t *testing.T) {
	s := newFormScenario(t)
	prompts, _ := promptbank.Load()
	s.a.service.RegisterPrompts(prompts...)
	s.submit(t)
	if code, view := s.portrait(t, "GET", true); code != 200 || view["status"] != "off" {
		t.Fatalf("without a model: %d %v", code, view)
	}
	var broken atomic.Bool
	broken.Store(true)
	m := &countingModel{reply: func(user string) string {
		if broken.Load() {
			return "не JSON"
		}
		return portraitAnswer(user)
	}}
	s.a.service.UseModel(m)
	defer s.a.service.Close()
	failed := s.waitFor(t, "failed")
	if failed["reason"] == "" || failed["portrait"] != nil {
		t.Fatalf("failed view %v", failed)
	}
	broken.Store(false)
	if code, view := s.portrait(t, "POST", true); code != 200 || view["status"] == "failed" {
		t.Fatalf("retry: %d %v", code, view)
	}
	s.waitFor(t, "ready")
}
