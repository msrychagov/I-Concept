package usecase

import (
	"encoding/json"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// The phrases of the answers accepted on 24–25 October that broke rule 2, and
// those that keep it.
func TestRatedLevelOfTheConditions(t *testing.T) {
	for text, want := range map[string]string{
		"Студент включён во все четыре вида деятельности на высоком уровне, активно использует ресурсы; условия ВУЦ оцениваются высоко, особенно по ясности норм.": "высоко",
		"Студент включён во все четыре вида деятельности на высоком или среднем уровне, особенно в учебной, и высоко оценивает организационные условия ВУЦ.":       "высоко",
		"Условия среды оцениваются высоко по возможностям значимой деятельности, но ниже по понятности требований.":                                                "высоко",
		"Взвод даёт условиям ВУЦ высокие оценки.":                        "высок…",
		"Организационные условия оцениваются как низкие.":                "низк…",
		"Условия ВУЦ — на среднем уровне.":                               "средн…",
		"Студент невысоко оценивает доверие и уважение достоинства.":     "невысоко",
		"Студент включён в учебную деятельность на высоком уровне.":      "",
		"Условия ВУЦ он оценивает выше в личностном плане.":              "",
		"Самооценка высокая, условия он оценивает выше.":                 "",
		"Студент критично оценивает опыт на высоком уровне.":             "",
		"В условиях высокой нагрузки студент сохраняет самообладание.":   "",
		"Учебная деятельность — на низком уровне, условия недостаточны.": "",
	} {
		problems := strings.Join(ratedLevelProblems("activity", text), "; ")
		if want == "" && problems != "" || want != "" && !strings.Contains(problems, "activity: слово «"+want+"» об оценке условий или признаков среды запрещено") {
			t.Errorf("%q: %q", text, problems)
		}
	}
}

// The checks of both answers take the rule.
func TestRatedLevelInBothAnswers(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v3", a.TestPortrait(fixedTestSession(a, a.bank.Version)))
	answer := goodTestAnswerV2(string(canonicalJSON(payload)))
	answer["activity"] = "Студент включён в деятельность ВУЦ. Условия ВУЦ оцениваются высоко. Это сдерживает соборность."
	if problems := strings.Join(checkTestAnswerV2(answer, payload), "; "); !strings.Contains(problems, "activity: слово «высоко»") {
		t.Errorf("portrait: %s", problems)
	}

	f := newPlatoonFixture(t, 3)
	db := mustSnapshot(t, f.service.repo)
	platoon := f.service.platoonPayload(f.service.platoonPortrait("101", f.service.platoonStudents(db, f.teacherID, "101")))
	conclusion := goodPlatoonAnswer(string(canonicalJSON(platoon)))
	conclusion["activity"] = "Взвод включён в учебную деятельность. Взвод высоко оценивает организационные условия. Это сдерживает соборность."
	if problems := strings.Join(checkPlatoonAnswer(conclusion, platoon), "; "); !strings.Contains(problems, "activity: слово «высоко»") {
		t.Errorf("platoon: %s", problems)
	}
}

// A state is made up when the input has no word of it, as «тревога» in the
// portrait of 25 October; a student who wrote of it himself may be written of.
func TestUnfoundStates(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v3", a.TestPortrait(fixedTestSession(a, a.bank.Version)))
	answer := goodTestAnswerV2(string(canonicalJSON(payload)))
	answer["recommendations"].([]any)[1].(map[string]any)["text"] = "Организовать консультацию психолога для проработки тревоги в стрессовых ситуациях."
	if problems := strings.Join(checkTestAnswerV2(answer, payload), "; "); !strings.Contains(problems, "recommendations.2.text: о состоянии «тревог…», о котором во входных данных ничего нет") {
		t.Errorf("portrait: %s", problems)
	}

	told := newObject()
	told.set("рассказы_о_деятельности", []any{"Перед сборами тревожился, но справился."})
	texts := map[string]string{"activity": "Студент справляется с тревогой.", "conclusion": "Студент замкнут."}
	if problems := strings.Join(unfoundStateProblems(texts, told), "; "); problems != "conclusion: о состоянии «замкнут…», о котором во входных данных ничего нет — пиши только о том, что в них есть" {
		t.Errorf("told: %s", problems)
	}

	f := newPlatoonFixture(t, 3)
	db := mustSnapshot(t, f.service.repo)
	platoon := f.service.platoonPayload(f.service.platoonPortrait("101", f.service.platoonStudents(db, f.teacherID, "101")))
	conclusion := goodPlatoonAnswer(string(canonicalJSON(platoon)))
	conclusion["individual"] = []any{"Студентам с повышенной тревожностью — консультация психолога."}
	if problems := strings.Join(checkPlatoonAnswer(conclusion, platoon), "; "); !strings.Contains(problems, "individual.1: о состоянии «тревог…»") {
		t.Errorf("platoon: %s", problems)
	}
}

// The conclusion of 25 October counted the students: «у одного …, у двух …».
func TestPlatoonCountsInWords(t *testing.T) {
	for text, want := range map[string]string{
		"По военной специальности у одного студента сформированная идентичность, у двух — мораторий.": "двух",
		"Двое из трёх студентов приняли образ.":                                                       "двое",
		"Мораторий у трёх студентов взвода.":                                                          "трех",
		"Два из пяти студентов на высоком уровне.":                                                    "два",
		"У большинства сформированная идентичность, у одного студента — мораторий.":                   "",
		"Взвод включён во все четыре вида деятельности.":                                              "",
		"Около половины студентов ближе к образу в пяти группах.":                                     "",
		"Треть студентов оценивает условия выше.":                                                     "",
	} {
		problems := strings.Join(platoonCountProblems(map[string]string{"identity": text}, 3), "; ")
		if want == "" && problems != "" || want != "" && !strings.Contains(problems, "identity: число студентов числительным («"+want+"»)") {
			t.Errorf("%q: %q", text, problems)
		}
	}

	f := newPlatoonFixture(t, 3)
	db := mustSnapshot(t, f.service.repo)
	platoon := f.service.platoonPayload(f.service.platoonPortrait("101", f.service.platoonStudents(db, f.teacherID, "101")))
	conclusion := goodPlatoonAnswer(string(canonicalJSON(platoon)))
	conclusion["recommendations"].([]any)[0].(map[string]any)["text"] = "Провести беседу с двумя студентами, у которых мораторий."
	if problems := strings.Join(checkPlatoonAnswer(conclusion, platoon), "; "); !strings.Contains(problems, "recommendations.1.text: число студентов числительным («двумя»)") {
		t.Errorf("platoon: %s", problems)
	}
}

// With the moratorium by the military speciality, as at Лебедев and Козлов on
// 25 October, a point to group 2 is about the self-determination.
func TestSelfDeterminationPoint(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	military := -1
	for n, p := range data.Identity {
		if p.ID == "military" {
			military = n
		}
	}
	if military < 0 {
		t.Fatal("no military block")
	}
	for leading, want := range map[string]string{
		"мораторий (кризис выбора)":   "о самоопределении как военного специалиста",
		"неопределённая идентичность": "о самоопределении как военного специалиста",
		"навязанная":                  "о личном смысле выбора",
		"сформированная":              "",
	} {
		data.Identity[military].LeadingText = leading
		payload := a.testPayloadFor("v3", data)
		answer := goodTestAnswerV2(string(canonicalJSON(payload)))
		if problems := checkTestAnswerV2(answer, payload); len(problems) != 0 {
			t.Errorf("%s, a good answer fails: %v", leading, problems)
		}
		var second map[string]any
		for _, item := range answer["recommendations"].([]any) {
			if item.(map[string]any)["group"] == 2 {
				second = item.(map[string]any)
			}
		}
		second["text"] = "Привлечь к наставничеству для конкретизации профессиональных компетенций."
		problems := strings.Join(checkTestAnswerV2(answer, payload), "; ")
		if want == "" && strings.Contains(problems, "к группе 2") || want != "" && !strings.Contains(problems, "ведущий статус «"+leading+"» — нужен пункт к группе 2 "+want) {
			t.Errorf("%s: %q", leading, problems)
		}
	}
}

func recommendationList(pairs ...any) []any {
	list := []any{}
	for n := 0; n < len(pairs); n += 2 {
		list = append(list, map[string]any{"group": pairs[n], "code": pairs[n+1], "text": "Пункт."})
	}
	return list
}

// A direction given to a group twice is dropped while three are left, as at
// Морозов on 25 October; otherwise the model is asked again.
func TestRepeatedDirections(t *testing.T) {
	for name, c := range map[string]struct {
		given   []any
		left    string
		changes int
		problem string
	}{
		"Морозов":         {recommendationList(1.0, "R16", 3.0, "R15", 5.0, "R16", 1.0, "R16"), "1/R16 3/R15 5/R16", 1, ""},
		"two repeats":     {recommendationList(1.0, "R16", 1.0, "R16", 2.0, "R11", 1.0, "R16", 3.0, "R15"), "1/R16 2/R11 3/R15", 2, ""},
		"three only":      {recommendationList(1.0, "R16", 3.0, "R15", 1.0, " R16"), "1/R16 3/R15 1/R16", 0, "recommendations.3: направление R16 к группе 1 уже есть — возьми другое направление или другую группу"},
		"other groups":    {recommendationList(1.0, "R16", 3.0, "R16", 5.0, "R16"), "1/R16 3/R16 5/R16", 0, ""},
		"one repeat of 4": {recommendationList(2.0, "R11", 2.0, "R13", 4.0, "R14", 2.0, "R11"), "2/R11 2/R13 4/R14", 1, ""},
	} {
		output, changes := cleanRepeatedDirections(nil, model.Database{}, Source{}, map[string]any{"recommendations": c.given}, nil)
		left := []string{}
		for _, item := range output["recommendations"].([]any) {
			key, _ := directionKey(item)
			left = append(left, key)
		}
		problems := strings.Join(repeatedDirectionProblems(output), "; ")
		if strings.Join(left, " ") != c.left || changes != c.changes || problems != c.problem {
			t.Errorf("%s: %v, %d changes, %q", name, left, changes, problems)
		}
	}
}

// The portrait is stored without the repeat, and the model is not asked again.
func TestRepeatedDirectionIsCleanedFromThePortrait(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	m := &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		answer := goodTestAnswerV2(c.User)
		list := answer["recommendations"].([]any)
		answer["recommendations"] = append(list, list[0])
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
	f.service.UseModel(m)
	f.service.queuePortraits(Source{"session", f.session})
	view := f.wait()
	if view.Status != portraitReady || view.Portrait == nil || m.count() != 1 {
		t.Fatalf("%d calls, view: %+v", m.count(), view)
	}
	var stored struct {
		Recommendations []any `json:"recommendations"`
	}
	if err := json.Unmarshal(view.Portrait.Output, &stored); err != nil || len(stored.Recommendations) != 3 {
		t.Fatalf("stored: %s", view.Portrait.Output)
	}
}

// A status is named by its first word or by the words in its brackets, as the
// conclusion by platoon 104 wrote «кризис выбора» on 25 October.
func TestStatusNamed(t *testing.T) {
	for text, want := range map[string]bool{
		"ведущая позиция — кризис выбора":       true,
		"большинство в кризисе выбора":          true,
		"у большинства мораторий":               true,
		"большинство переживает кризис":         false,
		"выбор у большинства сделан":            false,
		"у большинства идентичность навязанная": false,
	} {
		if got := statusNamed(normalizedText(text), "мораторий (кризис выбора)"); got != want {
			t.Errorf("%q: %v", text, got)
		}
	}
	if !statusNamed("идентичность сформирована", "сформированная") {
		t.Error("сформированная")
	}
}

// A group named by its number is told to be named in words.
func TestGroupNumberHint(t *testing.T) {
	if got := strings.Join(fieldProblems("activity", "Это отдаляет его от образа в группе 3."), "; "); !strings.Contains(got, "activity: цифры в тексте — группу образа называй словами") {
		t.Errorf("group: %s", got)
	}
	if got := strings.Join(fieldProblems("activity", "Оценка 7 из 10."), "; "); got != "activity: цифры в тексте" {
		t.Errorf("figure: %s", got)
	}
}

// The retry is told the word that fits the count in the platoon.
func TestRelativeCount(t *testing.T) {
	for _, c := range []struct {
		count, total int
		want         string
	}{{2, 3, "большинство"}, {3, 3, "все"}, {1, 3, "треть"}, {2, 4, "половина"}, {2, 5, "несколько студентов"}, {4, 5, "большинство"}, {0, 3, ""}} {
		if got := relativeCount(c.count, c.total); got != c.want {
			t.Errorf("%d of %d: %q", c.count, c.total, got)
		}
	}
	got := strings.Join(platoonCountProblems(map[string]string{"components": "Оценка опыта у двоих — на среднем уровне."}, 3), "; ")
	if !strings.HasSuffix(got, "; здесь — «большинство»") {
		t.Errorf("hint: %s", got)
	}
}

// The point about the self-determination is told the codes that fit group 2.
func TestSelfDeterminationHintNamesTheCodes(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	for n := range data.Identity {
		if data.Identity[n].ID == "military" {
			data.Identity[n].LeadingText = "навязанная"
		}
	}
	payload := a.testPayloadFor("v3", data)
	fitting := ""
	for _, g := range objectList(payload.values["образ_защитника"]) {
		if g.values["номер"] == 2 {
			fitting = strings.Join(textList(g.values["подходящие_направления"]), ", ")
		}
	}
	answer := goodTestAnswerV2(string(canonicalJSON(payload)))
	answer["recommendations"] = answer["recommendations"].([]any)[:1]
	problems := strings.Join(selfDeterminationProblems(answer, payload), "; ")
	if fitting == "" || !strings.HasSuffix(problems, "; код — из подходящих группе 2: "+fitting) {
		t.Errorf("%q, fitting %q", problems, fitting)
	}
}
