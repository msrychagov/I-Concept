package usecase

import (
	"encoding/json"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// goodTestAnswer answers the test instruction the way it asks, for the input it is given.
func goodTestAnswer(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input struct {
		Kinds []struct {
			Kind  string `json:"вид"`
			Level string `json:"уровень"`
		} `json:"виды_деятельности"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	activity := []any{}
	for _, k := range input.Kinds {
		activity = append(activity, map[string]any{"kind": strings.ToLower(k.Kind), "level": k.Level, "text": "По ответам студента вид деятельности описан."})
	}
	return map[string]any{
		"portrait": "По результатам методик Я-концепция студента описана целостно.",
		"core": map[string]any{
			"cs": "По результатам методики мотивация описана.",
			"ev": "По ответам студента саморегуляция описана.",
			"dp": "По ответам студента опыт описан.",
		},
		"signs":       "Сильнее выражена соборность, слабее — любовь.",
		"images":      "Расхождение Я-реального и Я-идеального оптимальное.",
		"environment": "Студент оценивает выше личностные условия, ниже — организационные.",
		"activity":    activity,
		"link":        "На фоне такой оценки условий сочетаются участие в деятельности и устойчивость.",
		"support":     []any{"Опереться на настойчивость."},
		"growth":      []any{"Работать над включённостью в коллектив."},
	}
}

type testPortraitFixture struct {
	t         *testing.T
	service   *Service
	teacherID string
	session   string
}

// newTestPortraitFixture finishes a session with the methods answered.
func newTestPortraitFixture(t *testing.T, methods bool) *testPortraitFixture {
	t.Helper()
	s := testApp(t)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.RegisterPrompts(prompts...)
	s.portraits.pause = 0
	t.Cleanup(s.Close)
	student := signUp(t, s, "student", "portrait@example.test", "101")
	p := s.emptyProfile()
	p.ID, p.Name, p.Author, p.OwnerID = "template", "Шаблон", "Тест", student.Account.OwnerID
	if err := s.addProfile(p); err != nil {
		t.Fatal(err)
	}
	input := Input{"title": raw(t, "Тест"), "profileId": raw(t, "template"), "methods": raw(t, methods)}
	if _, err := s.AssignmentCommandFor(student.Account.OwnerID, input, "assignment"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartAssignment(student.Account, mustSnapshot(t, s.repo).Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := s.repo.Update(func(db *model.Database) error {
		session := &db.Sessions[0].Session
		id = session.ID
		for _, item := range s.itemsFor(*session) {
			switch {
			case len(item.Options) > 0:
				session.Answers[item.ID] = 1 % len(item.Options)
			case item.Type != "textarea":
				session.Answers[item.ID] = item.Max
			}
		}
		session.Consent, session.Completed = true, true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return &testPortraitFixture{t: t, service: s, teacherID: student.Account.OwnerID, session: id}
}

func (f *testPortraitFixture) wait() PortraitView {
	f.t.Helper()
	(&portraitFixture{t: f.t, service: f.service}).idle()
	view, err := f.service.PortraitFor(f.teacherID, testPortraitTask, f.session)
	if err != nil {
		f.t.Fatal(err)
	}
	return view
}

func TestSessionWithMethodsGetsATestPortrait(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	var user string
	m := &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		user = c.User
		if c.MaxTokens != 4000 {
			t.Errorf("max tokens %d", c.MaxTokens)
		}
		data, _ := json.Marshal(goodTestAnswerV2(c.User))
		return string(data), nil
	}}
	f.service.UseModel(m)
	f.service.queuePortraits(Source{"session", f.session})
	if view := f.wait(); view.Status != portraitReady || view.Portrait == nil {
		t.Fatalf("view: %+v", view)
	}
	if m.count() != 1 || strings.Contains(user, "portrait@example.test") || !strings.Contains(user, "\"методики\"") || !strings.Contains(user, "Самообладание") || !strings.Contains(user, "\"образ_защитника\"") {
		t.Fatalf("%d calls, input: %.300s", m.count(), user)
	}
	for _, name := range mustSnapshot(t, f.service.repo).Students {
		if name.Name != "" && strings.Contains(user, name.Name) {
			t.Fatal("the name of the student reached the model")
		}
	}
}

func TestSessionWithoutMethodsHasNoTestPortrait(t *testing.T) {
	f := newTestPortraitFixture(t, false)
	f.service.UseModel(&fakeModel{name: "fake", reply: func(int, Completion) (string, error) { return "{}", nil }})
	if _, err := f.service.PortraitFor(f.teacherID, testPortraitTask, f.session); kind(err) != NotFound {
		t.Fatalf("portrait of a session without methods: %v", err)
	}
}

func TestTestAnswerCheckKeepsTheRules(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	db := mustSnapshot(t, f.service.repo)
	payload := testPayload(f.service.TestPortrait(db.Sessions[0].Session))
	user := string(canonicalJSON(payload))
	if problems := checkTestAnswer(goodTestAnswer(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	levels := map[string]string{}
	for _, m := range f.service.TestPortrait(db.Sessions[0].Session).Methods {
		levels[m.Name] = m.LevelText
	}
	wrong := map[string]string{"высокий": "низком", "средний": "высоком", "низкий": "высоком"}[levels["Самообладание"]]
	for name, c := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"changed level": {func(a map[string]any) { a["activity"].([]any)[0].(map[string]any)["level"] = "выдающийся" }, "уровень изменён"},
		"wrong order":   {func(a map[string]any) { list := a["activity"].([]any); list[0], list[1] = list[1], list[0] }, "не тот вид"},
		"three kinds":   {func(a map[string]any) { a["activity"] = a["activity"].([]any)[:3] }, "видов деятельности 3"},
		"cause in link": {func(a map[string]any) {
			a["link"] = "Высокая самооценка сложилась благодаря среде."
		}, "причина вместо сочетания"},
		"influence verb": {func(a map[string]any) {
			a["link"] = "Слабая обратная связь ограничивает соборность."
		}, "влияние вместо сочетания"},
		"verb in core": {func(a map[string]any) {
			a["core"].(map[string]any)["ev"] = "Устойчивость усиливает саморегуляцию."
		}, ""},
		"adverb of rating": {func(a map[string]any) {
			a["environment"] = "Студент высоко оценивает доступность ресурсов."
		}, ""},
		"supporting": {func(a map[string]any) {
			a["link"] = "Условия, которые студент оценивает выше, сочетаются с поддерживающим наставничеством."
		}, ""},
		"level in kind": {func(a map[string]any) {
			a["activity"].([]any)[0].(map[string]any)["text"] = "Активность на занятиях высока."
		}, ""},
		"level of score": {func(a map[string]any) {
			a["signs"] = "Коллективизм низок, уважение выражено сильнее."
		}, "уровень у балла 1–5"},
		"diagnosis":  {func(a map[string]any) { a["portrait"] = "Признаков расстройства нет." }, "недопустимое слово"},
		"no growth":  {func(a map[string]any) { a["growth"] = []any{} }, "growth: нужно от 1 до 5 пунктов"},
		"no section": {func(a map[string]any) { delete(a, "environment") }, "нет раздела environment"},
		"method level": {func(a map[string]any) {
			a["core"].(map[string]any)["ev"] = "Самообладание на " + wrong + " уровне."
		}, "уровень показателя «Самообладание» не тот"},
		"growth may push": {func(a map[string]any) {
			a["growth"] = []any{"Повышать включённость; это ограничивает изоляцию."}
		}, ""},
	} {
		answer := goodTestAnswer(user)
		c.mutate(answer)
		problems := strings.Join(checkTestAnswer(answer, payload), "; ")
		if c.problem == "" && problems != "" || c.problem != "" && !strings.Contains(problems, c.problem) {
			t.Errorf("%s: %q", name, problems)
		}
	}
}
