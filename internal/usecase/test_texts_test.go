package usecase

import (
	"strings"
	"testing"
)

// The phrases of the answers accepted on 24–25 October that broke rule 2, and
// those that keep it.
func TestRatedLevelOfTheConditions(t *testing.T) {
	for text, want := range map[string]string{
		"Студент включён во все четыре вида деятельности на высоком уровне, активно использует ресурсы; условия ВУЦ оцениваются высоко, особенно по ясности норм.": "высоко",
		"Студент включён во все четыре вида деятельности на высоком или среднем уровне, особенно в учебной, и высоко оценивает организационные условия ВУЦ.":       "высоко",
		"Условия среды оцениваются высоко по возможностям значимой деятельности, но ниже по понятности требований.":                                                "высоко",
		"Взвод даёт условиям ВУЦ высокие оценки.":                        "высок",
		"Организационные условия оцениваются как низкие.":                "низк",
		"Условия ВУЦ — на среднем уровне.":                               "средн",
		"Студент невысоко оценивает доверие и уважение достоинства.":     "невысоко",
		"Студент включён в учебную деятельность на высоком уровне.":      "",
		"Условия ВУЦ он оценивает выше в личностном плане.":              "",
		"Самооценка высокая, условия он оценивает выше.":                 "",
		"Студент критично оценивает опыт на высоком уровне.":             "",
		"В условиях высокой нагрузки студент сохраняет самообладание.":   "",
		"Учебная деятельность — на низком уровне, условия недостаточны.": "",
	} {
		problems := strings.Join(ratedLevelProblems("activity", text), "; ")
		if want == "" && problems != "" || want != "" && !strings.Contains(problems, "activity: уровень («"+want+"») у оценки условий") {
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
	if problems := strings.Join(checkTestAnswerV2(answer, payload), "; "); !strings.Contains(problems, "activity: уровень («высоко»)") {
		t.Errorf("portrait: %s", problems)
	}

	f := newPlatoonFixture(t, 3)
	db := mustSnapshot(t, f.service.repo)
	platoon := f.service.platoonPayload(f.service.platoonPortrait("101", f.service.platoonStudents(db, f.teacherID, "101")))
	conclusion := goodPlatoonAnswer(string(canonicalJSON(platoon)))
	conclusion["activity"] = "Взвод включён в учебную деятельность. Взвод высоко оценивает организационные условия. Это сдерживает соборность."
	if problems := strings.Join(checkPlatoonAnswer(conclusion, platoon), "; "); !strings.Contains(problems, "activity: уровень («высоко»)") {
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
		problems := strings.Join(platoonCountProblems(map[string]string{"identity": text}), "; ")
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
