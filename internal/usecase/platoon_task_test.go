package usecase

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// goodPlatoonAnswer answers test-platoon/v1 the way it asks, for the input.
func goodPlatoonAnswer(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input struct {
		Identity []struct {
			Leading map[string]int `json:"ведущий_статус"`
		} `json:"профессиональная_идентичность"`
		Catalogue []struct {
			Code   string `json:"код"`
			Groups []int  `json:"группы"`
		} `json:"каталог_рекомендаций"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	identity := identityAbsent
	if len(input.Identity) > 0 {
		names := []string{}
		for _, block := range input.Identity {
			for name := range block.Leading {
				if name != levelText[""] {
					names = append(names, name)
				}
			}
		}
		slices.Sort(names)
		identity = "Во взводе встречаются статусы: " + strings.Join(names, ", ") + "."
	}
	recommendations := []any{}
	for group := 1; group <= 3; group++ {
		for _, d := range input.Catalogue {
			if slices.Contains(d.Groups, group) {
				recommendations = append(recommendations, map[string]any{"group": group, "code": d.Code, "text": "Усилить эту работу во взводе, чтобы образ стал ближе."})
				break
			}
		}
	}
	return map[string]any{
		"overview":        "Я-концепция во взводе складывается неравномерно.",
		"components":      "У большинства ценностно-смысловой компонент на среднем уровне. Эмоционально-волевой выражен по-разному. Сильнее выражен патриотизм.",
		"identity":        identity,
		"ideal":           "Ближе всего взвод к образу в патриотических качествах. Дальше всего — в коллективистских.",
		"activity":        "Взвод включён в учебную деятельность. Условия ВУЦ взвод оценивает выше в личностном плане. Сотрудничество и сплочённость выражены слабее, это сдерживает соборность.",
		"recommendations": recommendations,
		"individual":      []any{"Студентам, у которых Я-идеальное ниже Я-реального, нужна отдельная беседа."},
		"conclusion":      "Взвод патриотически направлен. Главное в работе с ним — совместные задачи.",
	}
}

type platoonFixture struct {
	service   *Service
	teacherID string
	sessions  []string
	names     []string
}

// newPlatoonFixture finishes the test with the methods for n students of
// platoon 101, every one with answers of his own.
func newPlatoonFixture(t *testing.T, n int) *platoonFixture {
	t.Helper()
	s := testApp(t)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.RegisterPrompts(prompts...)
	s.portraits.pause = 0
	t.Cleanup(s.Close)
	f := &platoonFixture{service: s}
	accounts := []model.Account{}
	for i := 0; i < n; i++ {
		student := signUp(t, s, "student", fmt.Sprintf("platoon%d@example.test", i), "101")
		accounts = append(accounts, student.Account)
		f.teacherID = student.Account.OwnerID
	}
	p := s.emptyProfile()
	p.ID, p.Name, p.Author, p.OwnerID = "template", "Шаблон", "Тест", f.teacherID
	if err := s.addProfile(p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AssignmentCommandFor(f.teacherID, Input{"title": raw(t, "Тест"), "profileId": raw(t, "template"), "methods": raw(t, true), "activity": raw(t, true)}, "assignment"); err != nil {
		t.Fatal(err)
	}
	for _, account := range accounts {
		if _, err := s.StartAssignment(account, mustSnapshot(t, s.repo).Assignments[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.repo.Update(func(db *model.Database) error {
		for n := range db.Sessions {
			session := &db.Sessions[n].Session
			f.sessions = append(f.sessions, session.ID)
			f.names = append(f.names, session.Participant)
			for _, item := range s.itemsFor(*session) {
				h := fnv.New32a()
				h.Write([]byte(fmt.Sprint(item.ID, n)))
				v := h.Sum32()
				switch {
				case item.Type == "textarea":
					session.Answers[item.ID], session.Notes[item.ID] = 0, "Участвовал в мероприятии взвода."
				case len(item.Options) > 0:
					session.Answers[item.ID] = int(v % uint32(len(item.Options)))
				default:
					session.Answers[item.ID] = item.Min + int(v%uint32(item.Max-item.Min+1))
				}
			}
			session.Consent, session.Completed = true, true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestPlatoonConclusionIsMadeByTheButtonOfThreeStudents(t *testing.T) {
	few := newPlatoonFixture(t, 2)
	few.service.UseModel(&fakeModel{name: "fake", reply: func(int, Completion) (string, error) { return "{}", nil }})
	if _, err := few.service.PortraitFor(few.teacherID, testPlatoonTask, "101"); kind(err) != NotFound {
		t.Fatalf("two students: %v", err)
	}

	f := newPlatoonFixture(t, 3)
	// The portraits of the students are made on their own; the conclusion by
	// the platoon is made only by the button.
	var user string
	platoonCalls := 0
	m := &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		answer := goodTestAnswerV2(c.User)
		if strings.Contains(c.System, "одного учебного взвода") {
			user, answer = c.User, goodPlatoonAnswer(c.User)
			platoonCalls++
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
	f.service.UseModel(m)
	(&portraitFixture{t: t, service: f.service}).idle()
	view, err := f.service.PortraitFor(f.teacherID, testPlatoonTask, "101")
	if err != nil || view.Status != portraitNone {
		t.Fatalf("before the button: %+v %v", view, err)
	}
	if view, err = f.service.RetryPortrait(f.teacherID, testPlatoonTask, "101"); err != nil || view.Status != portraitQueued {
		t.Fatalf("the button: %+v %v", view, err)
	}
	(&portraitFixture{t: t, service: f.service}).idle()
	if view, _ = f.service.PortraitFor(f.teacherID, testPlatoonTask, "101"); view.Status != portraitReady {
		t.Fatalf("after: %+v", view)
	}
	if platoonCalls != 1 || m.count() != 4 || !strings.Contains(user, `"студентов": 3`) || !strings.Contains(user, `"образ_защитника"`) || !strings.Contains(user, `"признаки_среды"`) {
		t.Fatalf("%d calls, input: %.400s", m.count(), user)
	}
	for _, name := range f.names {
		if strings.Contains(user, name) {
			t.Fatal("a name reached the model")
		}
	}
	// The stored conclusion keeps the platoon and the sessions it was made of.
	stored := slices.DeleteFunc(mustSnapshot(t, f.service.repo).Portraits, func(p model.Portrait) bool { return p.Task != testPlatoonTask })
	if len(stored) != 1 || stored[0].Source != "101" || !slices.Equal(slices.Sorted(slices.Values(stored[0].Sources)), slices.Sorted(slices.Values(f.sessions))) {
		t.Fatalf("stored: %+v", stored)
	}
	if _, err := f.service.PortraitFor("someone-else", testPlatoonTask, "101"); kind(err) != NotFound {
		t.Fatalf("another teacher: %v", err)
	}
}

func TestPlatoonAnswerCheck(t *testing.T) {
	f := newPlatoonFixture(t, 3)
	db := mustSnapshot(t, f.service.repo)
	payload := f.service.platoonPayload(f.service.platoonPortrait("101", f.service.platoonStudents(db, f.teacherID, "101")))
	user := string(canonicalJSON(payload))
	if problems := checkPlatoonAnswer(goodPlatoonAnswer(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	for name, c := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"figure": {func(a map[string]any) { a["overview"] = "Во взводе 3 студента." }, "overview: цифры в тексте"},
		"long": {func(a map[string]any) {
			a["components"] = "Раз. Два. Три. Четыре. Пять. Шесть."
		}, "components: предложений 6 вместо 3–5"},
		"climate": {func(a map[string]any) {
			a["activity"] = "Климат во взводе тёплый. Условия он оценивает выше. Это сдерживает соборность."
		}, "activity: о климате или сплочённости"},
		"cohesion": {func(a map[string]any) {
			a["conclusion"] = "Взвод сплочён и сплочённость высока. Главное — задачи."
		}, "conclusion: о климате или сплочённости"},
		"six points": {func(a map[string]any) {
			r := a["recommendations"].([]any)
			a["recommendations"] = append(r, r[0], r[0], r[0])
		}, "recommendations: нужно от 3 до 5 пунктов"},
		"individual":  {func(a map[string]any) { a["individual"] = []any{"а", "б", "в", "г"} }, "individual: нужен список не больше чем из трёх пунктов"},
		"no identity": {func(a map[string]any) { a["identity"] = "Студенты в поиске своего пути." }, "identity: не назван самый частый ведущий статус"},
		"wrong group": {func(a map[string]any) {
			a["recommendations"].([]any)[0].(map[string]any)["group"] = 5
			a["recommendations"].([]any)[0].(map[string]any)["code"] = "R01"
		}, "recommendations.1: направление не подходит к группе"},
		"any type": {func(a map[string]any) {
			a["recommendations"].([]any)[0].(map[string]any)["group"] = 5
			a["recommendations"].([]any)[0].(map[string]any)["code"] = "R21"
		}, ""},
		"appointment": {func(a map[string]any) {
			a["individual"] = []any{"Назначить командиром отделения лучших."}
		}, "individual.1: о назначении на должность"},
		"no individual": {func(a map[string]any) { a["individual"] = []any{} }, ""},
	} {
		answer := goodPlatoonAnswer(user)
		c.mutate(answer)
		problems := strings.Join(checkPlatoonAnswer(answer, payload), "; ")
		if c.problem == "" && problems != "" || c.problem != "" && !strings.Contains(problems, c.problem) {
			t.Errorf("%s: %q", name, problems)
		}
	}
}

// The platoon input holds the numbers of the platoon: the situations of the
// groups, the levels, the conditions rated 1–2 and the answers about events.
func TestPlatoonPayloadHoldsTheNumbers(t *testing.T) {
	f := newPlatoonFixture(t, 3)
	db := mustSnapshot(t, f.service.repo)
	payload := f.service.platoonPayload(f.service.platoonPortrait("101", f.service.platoonStudents(db, f.teacherID, "101")))
	want := []string{"студентов", "методики", "образ_защитника", "согласованность_образов", "признаки", "профессиональная_идентичность", "виды_деятельности", "условия_среды", "признаки_среды", "каталог_рекомендаций"}
	if !slices.Equal(payload.keys, want) {
		t.Fatalf("keys: %v", payload.keys)
	}
	var input struct {
		Groups []struct {
			Situations map[string]int `json:"ситуации"`
		} `json:"образ_защитника"`
		Kinds []struct {
			Levels   map[string]int `json:"уровни"`
			Criteria []struct {
				Name string          `json:"критерий"`
				Data json.RawMessage `json:"данные"`
			} `json:"критерии"`
		} `json:"виды_деятельности"`
		Conditions []struct {
			Rated int `json:"оценили"`
		} `json:"условия_среды"`
	}
	if err := json.Unmarshal(canonicalJSON(payload), &input); err != nil {
		t.Fatal(err)
	}
	for _, g := range input.Groups {
		total := 0
		for _, n := range g.Situations {
			total += n
		}
		if total != 3 || len(g.Situations) < 4 {
			t.Errorf("situations: %v", g.Situations)
		}
	}
	for _, c := range input.Conditions {
		if c.Rated != 3 {
			t.Errorf("condition rated by %d", c.Rated)
		}
	}
	// The criterion of the fact of upbringing gets what the students named.
	fact := string(input.Kinds[1].Criteria[0].Data)
	if !strings.Contains(fact, "Участвовал в мероприятии взвода.") || !strings.Contains(fact, `"рассказали":3`) {
		t.Fatalf("fact: %.300s", fact)
	}
	if other := string(input.Kinds[0].Criteria[0].Data); strings.Contains(other, "Участвовал в мероприятии") {
		t.Fatalf("answers of another criterion: %.300s", other)
	}
}
