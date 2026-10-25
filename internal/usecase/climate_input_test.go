package usecase

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/promptbank"
)

// signByName is a sign of the environment in an input by its name.
func signByName(payload *object, name string) *object {
	for _, sign := range objectList(payload.values["признаки_среды"]) {
		if objectText(sign, "признак") == name {
			return sign
		}
	}
	return nil
}

// climateCriterionOf is the criterion of the climate of the first chapter in an input.
func climateCriterionOf(payload *object) *object {
	for _, kind := range objectList(payload.values["виды_деятельности"]) {
		for _, c := range objectList(kind.values["критерии"]) {
			if objectText(c, "критерий") == "Психологический климат, сплочённость взвода" {
				return c
			}
		}
	}
	return nil
}

// From v4 on the input has the rules of Danilova by component and by group
// and, when the student rated all the pairs, his climate of the platoon in the
// sign «сотрудничество и сплочённость» with a reference in the criterion; no
// new keys at the top, the second edition of the catalogue.
func TestTestPayloadV4AddsDanilovaAndClimate(t *testing.T) {
	a := testApp(t)
	data := climateSession(a, "А", 3, 3, 2, 2, 2, 1, 1, 1, 0, 0, -1, -1, -2, -2).data
	old, payload := a.testPayloadFor("v3", data), a.testPayloadFor("v4", data)
	if !slices.Equal(old.keys, payload.keys) || string(canonicalJSON(old)) == string(canonicalJSON(payload)) {
		t.Fatalf("keys of v4: %v", payload.keys)
	}
	for _, c := range objectList(payload.values["образы_я"].(*object).values["по_компонентам"]) {
		if objectText(c, "внешнее_давление") == "" || objectText(c, "адекватность_самооценки") == "" {
			t.Errorf("component: %s", canonicalJSON(c))
		}
	}
	for _, g := range objectList(payload.values["образ_защитника"]) {
		at := slices.Index(g.keys, "разрыв")
		if at < 0 || g.keys[at+1] != "внешнее_давление" || g.keys[at+2] != "адекватность_самооценки" {
			t.Errorf("group: %v", g.keys)
		}
	}
	sign := signByName(payload, "Сотрудничество и сплочённость")
	list := sign.values["данные"].([]any)
	rating, _ := json.Marshal(list[len(list)-1])
	want, _ := json.Marshal(map[string]any{"климат_взвода": map[string]any{"сильнее_всего": sides(a, 1, 2), "слабее_всего": sides(a, 13, 14)}})
	if objectText(sign, "пометка") != climateNote || string(rating) != string(want) {
		t.Fatalf("sign: %s", canonicalJSON(sign))
	}
	criterion := climateCriterionOf(payload)
	if got := string(canonicalJSON(criterion)); got != `{"критерий":"Психологический климат, сплочённость взвода","покрытие":"частично","пометка":"по оценке самого студента","данные":[{"см":"оценка студентом климата взвода — в «признаки_среды», «сотрудничество и сплочённость»"}]}` {
		t.Fatalf("criterion: %s", got)
	}
	if len(objectList(payload.values["каталог_рекомендаций"])) != 24 {
		t.Fatal("catalogue of v4")
	}

	equal := a.testPayloadFor("v4", climateSession(a, "Б", 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1).data)
	list = signByName(equal, "Сотрудничество и сплочённость").values["данные"].([]any)
	if got := string(canonicalJSON(list[len(list)-1])); got != `{"климат_взвода":"все стороны климата студент оценил одинаково"}` {
		t.Fatalf("equal: %s", got)
	}

	// Without the method, or with a pair left, the climate stays as in v3.
	for _, other := range []*TestPortraitData{a.TestPortrait(fixedTestSession(a, "pilot-2.5")), climateSession(a, "В", 3, 2, 1).data} {
		input := a.testPayloadFor("v4", other)
		sign := signByName(input, "Сотрудничество и сплочённость")
		if objectText(sign, "пометка") != "о климате взвода данных нет" || strings.Contains(string(canonicalJSON(sign)), "климат_взвода") {
			t.Errorf("sign without the climate: %s", canonicalJSON(sign))
		}
		if objectText(climateCriterionOf(input), "данные") != noData {
			t.Errorf("criterion without the climate: %s", canonicalJSON(climateCriterionOf(input)))
		}
		if objectText(objectList(input.values["образ_защитника"])[0], "внешнее_давление") == "" {
			t.Error("no pressure without the climate")
		}
	}
}

// From v2 on the input by a platoon has the climate after the signs of the
// environment, a reference in the criterion and the sign, and the second
// edition of the catalogue; a platoon without the method has no climate.
func TestPlatoonPayloadV2HasTheClimate(t *testing.T) {
	a := testApp(t)
	students := []studentPortrait{
		climateSession(a, "А", 3, 3, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "Б", 3, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "В", 3, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "Г", 3, 2, 1),
	}
	data := a.platoonPortrait("101", students)
	old, payload := a.platoonPayloadFor("v1", data), a.platoonPayloadFor("v2", data)
	want := slices.Clone(old.keys)
	want = slices.Insert(want, slices.Index(want, "признаки_среды")+1, "климат_взвода")
	if !slices.Equal(payload.keys, want) {
		t.Fatalf("keys of v2: %v", payload.keys)
	}
	climate, _ := json.Marshal(payload.values["климат_взвода"])
	expected, _ := json.Marshal(map[string]any{"оценили": 3, "сильнее_всего": sides(a, 1, 3, 4), "слабее_всего": sides(a, 14, 13, 11, 12), "пометка": platoonClimateNote})
	if !strings.HasPrefix(string(climate), `{"оценили":3,"сильнее_всего"`) || len(climate) != len(expected) {
		t.Fatalf("climate: %s", climate)
	}
	if got := string(canonicalJSON(climateCriterionOf(payload))); got != `{"критерий":"Психологический климат, сплочённость взвода","покрытие":"по взводу","пометка":"по оценкам самих студентов","данные":[{"см":"климат взвода — в «климат_взвода»"}]}` {
		t.Fatalf("criterion: %s", got)
	}
	if objectText(signByName(payload, "Сотрудничество и сплочённость"), "пометка") != platoonClimateSee {
		t.Fatal("note of the sign")
	}
	if len(objectList(payload.values["каталог_рекомендаций"])) != 24 || len(objectList(old.values["каталог_рекомендаций"])) != 21 {
		t.Fatal("catalogues")
	}
	// Without the method, or with two students who answered all the pairs,
	// the platoon has no climate in the input, and the criterion no data.
	for _, other := range []PlatoonPortraitData{a.platoonPortrait("101", fixedPlatoon(a, "pilot-2.5")), a.platoonPortrait("101", students[1:])} {
		input := a.platoonPayloadFor("v2", other)
		if slices.Contains(input.keys, "климат_взвода") || objectText(climateCriterionOf(input), "данные") != noData || objectText(signByName(input, "Сотрудничество и сплочённость"), "пометка") != "о климате взвода данных нет" {
			t.Fatalf("a platoon without the climate: %v", input.keys)
		}
	}
}

// The climate has no level: a word of a level near «климат» in one sentence.
func TestClimateLevelCheck(t *testing.T) {
	for text, bad := range map[string]bool{
		"Студенты оценивают климат взвода как благоприятный.":                                                     true,
		"Неблагоприятный климат сдерживает соборность.":                                                           true,
		"Климат во взводе высокого уровня.":                                                                       true,
		"Психологический климат на среднем уровне.":                                                               true,
		"Сильнее всего студенты видят во взводе доброжелательность, слабее — готовность к общему делу.":           false,
		"В климате взвода сильнее выражена доброжелательность; эмоционально-волевой компонент на среднем уровне.": false,
		"Климат взвода студент описывает через доброжелательность и поддержку.":                                   false,
	} {
		if got := len(climateLevelProblems("activity", text)) > 0; got != bad {
			t.Errorf("%q: problem %v", text, got)
		}
	}
}

// With the climate in the input the platoon may write of it, without a level;
// without it the old rule stays.
func TestPlatoonCheckOfTheMeasuredClimate(t *testing.T) {
	a := testApp(t)
	data := a.platoonPortrait("101", []studentPortrait{
		climateSession(a, "А", 3, 3, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "Б", 3, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "В", 2, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
	})
	for _, c := range []struct {
		version, activity, problem string
	}{
		{"v2", "Взвод включён в учебную деятельность. В климате взвода сильнее всего выражены настроение и доброжелательность, слабее — гордость за взвод. Это сдерживает соборность.", ""},
		{"v2", "Взвод включён в учебную деятельность. Климат взвода благоприятный. Это сдерживает соборность.", "activity: у климата нет уровня («благоприят…»)"},
		{"v1", "Взвод включён в учебную деятельность. В климате взвода сильнее всего выражены настроение и доброжелательность, слабее — гордость за взвод. Это сдерживает соборность.", "activity: о климате или сплочённости взвода, которых тест не меряет"},
	} {
		payload := a.platoonPayloadFor(c.version, data)
		answer := goodPlatoonAnswer(string(canonicalJSON(payload)))
		answer["activity"] = c.activity
		problems := strings.Join(checkPlatoonAnswer(answer, payload), "; ")
		if c.problem == "" && problems != "" || c.problem != "" && !strings.Contains(problems, c.problem) {
			t.Errorf("%s %q: %s", c.version, c.activity, problems)
		}
	}
}

// The answer to test/v4 keeps the rules of v2 and v3 and gives the climate no level.
func TestTestAnswerV4Check(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v4", climateSession(a, "А", 3, 3, 2, 2, 2, 1, 1, 1, 0, 0, -1, -1, -2, -2).data)
	answer := goodTestAnswerV2(string(canonicalJSON(payload)))
	if problems := checkTestAnswerV4(answer, payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	answer["conclusion"] = "Студент видит климат взвода благоприятным. Главное — совместные задачи."
	if problems := strings.Join(checkTestAnswerV4(answer, payload), "; "); !strings.Contains(problems, "conclusion: у климата нет уровня") {
		t.Fatalf("problems: %s", problems)
	}
}

// While the portrait by v4 is made, the one by v3 is shown: the input of v4
// differs from that of v3.
func TestPortraitV3IsShownWhileV4IsMade(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	older := slices.DeleteFunc(slices.Clone(prompts), func(p promptbank.Prompt) bool { return p.Task == testPortraitTask && p.Version == "v4" })
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(older...)
	f.service.UseModel(&fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		data, _ := json.Marshal(goodTestAnswerV2(c.User))
		return string(data), nil
	}})
	f.service.queuePortraits(Source{"session", f.session})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v3" {
		t.Fatalf("v3: %+v", view)
	}
	f.service.UseModel(nil)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(prompts...)
	view, err := f.service.PortraitFor(f.teacherID, testPortraitTask, f.session)
	if err != nil || view.Status != portraitReady || view.Portrait.Version != "v3" {
		t.Fatalf("v3 while v4 is not made: %+v %v", view, err)
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		data, _ := json.Marshal(goodTestAnswerV2(c.User))
		return string(data), nil
	}})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v4" {
		t.Fatalf("v4: %+v", view)
	}
}
