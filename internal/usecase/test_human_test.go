package usecase

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/promptbank"
)

// goodHumanAction is the text of an action of v7 the checks take.
const goodHumanAction = "На полевом выходе поручить ему задачу, которую в одиночку не выполнить, и после выхода разобрать с отделением, что помогло действовать слаженно. Так воспитывается ответственность за товарищей."

// goodTestAnswerV7 answers test/v7 the way it asks, for the input it is given.
func goodTestAnswerV7(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input struct {
		Groups []struct {
			Number  int      `json:"номер"`
			Fitting []string `json:"подходящие_направления"`
		} `json:"образ_защитника"`
		Self []struct {
			Profession string `json:"профессия"`
			Meaning    string `json:"что_значит"`
		} `json:"самоопределение"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	will := "Владеет собой хорошо и настойчив в деле. Свой опыт он использует и разбирает."
	if len(input.Self) > 0 {
		parts := []string{}
		for _, s := range input.Self {
			parts = append(parts, lowerText(s.Profession)+": "+s.Meaning)
		}
		will += " О выборе можно сказать так — " + strings.Join(parts, "; ") + "."
	}
	recommendations, used := []any{}, map[string]bool{}
	for _, g := range input.Groups {
		for _, code := range g.Fitting {
			if len(recommendations) < 3 && !used[code] {
				text := goodHumanAction
				if g.Number == 2 {
					text = "После практического занятия обсудить с ним, что лично ему даёт выбор военной специальности и где это пригодится. Так выбор становится осознанным."
				}
				used[code] = true
				recommendations = append(recommendations, map[string]any{"group": g.Number, "code": code, "text": text})
				break
			}
		}
	}
	return map[string]any{
		"paragraphs": map[string]any{
			"values":      "Служба для него — осознанный выбор, он ответственен и добросовестен. С людьми уважителен и готов помочь. Работать сообща ему пока труднее, чем отвечать за себя.",
			"will":        will,
			"image":       "Он трезво видит, куда расти, и цели ставит по силам. Верность долгу и исполнительность считает своими качествами. Больше всего ему предстоит сделать в умении работать в команде.",
			"environment": "К распорядку и требованиям он привык и трудности переносит спокойно. Учится с интересом, но ему не хватает дела, где можно проявить себя. Поддерживают его ясные требования и внимание к нему лично.",
		},
		"recommendations": recommendations,
		"conclusion":      "Надёжный и собранный студент, порученное доводит до конца. Главное в работе с ним — давать ему дела вместе с товарищами.",
	}
}

// goodPlatoonAnswerV5 answers test-platoon/v5 the way it asks, for the input.
func goodPlatoonAnswerV5(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	answer := goodPlatoonAnswerV4(user)
	var input struct {
		Self []struct {
			Counts map[string]int `json:"сколько_студентов"`
		} `json:"самоопределение"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	phrases := []string{}
	for _, block := range input.Self {
		for phrase, n := range block.Counts {
			if n > 0 {
				phrases = append(phrases, phrase)
			}
		}
	}
	slices.Sort(phrases)
	will := "Студенты в основном владеют собой. Опыт используют не все."
	if len(phrases) > 0 {
		will += " Во взводе есть те, кто " + strings.Join(phrases, "; ") + "."
	}
	for _, key := range []string{"recommendations", "individual"} {
		for _, item := range answer[key].([]any) {
			record := item.(map[string]any)
			delete(record, "who")
			delete(record, "kind")
			record["text"] = "На ближайшем занятии разобрать со взводом выполненную задачу, чтобы каждый увидел свою часть общего результата. Так воспитывается чувство локтя."
		}
	}
	answer["paragraphs"] = map[string]any{
		"values":      "Студенты настроены служить и добросовестны. У большинства значимы ценности службы. Действовать заодно с товарищами им пока труднее.",
		"will":        will,
		"image":       "Какими хотят стать, студенты видят по-разному. Верность долгу большинство считает своим качеством, а умение работать в команде — пока нет.",
		"environment": "Порядок и требования службы взвод принял. Доверие старших студенты ценят. Общих дел им не хватает.",
	}
	answer["conclusion"] = "Взвод исполнительный и дружный. Главное в работе с ним — общие задачи."
	return answer
}

// The input of v7 and test-platoon/v5 has none of the labels the check
// forbids, nor the terms of v6; the words of the student stay.
func TestHumanInputHasNoLabels(t *testing.T) {
	a := testApp(t)
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	payloads := map[string]*object{"fixed": a.testPayloadFor("v7", a.TestPortrait(fixedTestSession(a, a.bank.Version)))}
	owner := ""
	for _, r := range db.Sessions {
		if data := a.TestPortrait(r.Session); r.Session.Methods && r.Session.Completed && data != nil {
			payloads[r.Session.Participant] = a.testPayloadFor("v7", data)
			owner = r.Session.TeacherID
		}
	}
	for _, g := range []string{"102", "104"} {
		payloads["взвод "+g] = a.platoonPayloadFor("v5", a.platoonPortrait(g, a.platoonStudents(db, owner, g)))
	}
	payloads["взвод 101"] = a.platoonPayloadFor("v5", a.platoonPortrait("101", fixedPlatoon(a, a.bank.Version)))
	if len(payloads) < 9 {
		t.Fatalf("only %d inputs", len(payloads))
	}
	for name, payload := range payloads {
		for _, key := range []string{"профессиональная_идентичность", "виды_деятельности"} {
			if _, ok := payload.values[key]; ok {
				t.Errorf("%s: %s is in the input", name, key)
			}
		}
		texts := []string{}
		inputTexts(payload, false, &texts)
		for _, text := range texts {
			text = strings.ReplaceAll(text, "_", " ")
			if problems := append(humanWordProblems("input", text), termProblems("input", text)...); len(problems) > 0 {
				t.Errorf("%s: %q: %v", name, text, problems)
			}
		}
	}
}

func TestHumanInputsAreKept(t *testing.T) {
	a := testApp(t)
	keptInput(t, "v7.json", indentedJSON(a.testPayloadFor("v7", a.TestPortrait(fixedTestSession(a, a.bank.Version)))))
	keptInput(t, "platoon-v5.json", indentedJSON(a.platoonPayloadFor("v5", a.platoonPortrait("101", fixedPlatoon(a, a.bank.Version)))))
}

// The phrases of the self-determination: one leading status gives its own,
// several equal ones «ещё не определился».
func TestHumanLeading(t *testing.T) {
	block := func(leading string) *object {
		o := newObject()
		statuses := []any{}
		for name := range selfCases {
			s := newObject()
			s.set("статус", name)
			statuses = append(statuses, s)
		}
		o.set("статусы", statuses)
		o.set("ведущий_статус", leading)
		return o
	}
	for leading, want := range map[string]string{
		"сформированная":                               selfFormed.phrase,
		"мораторий (кризис выбора)":                    selfSearching.phrase,
		"навязанная":                                   selfImposed.phrase,
		"неопределённая идентичность":                  selfIdle.phrase,
		"неопределённая идентичность и сформированная": selfUndecided.phrase,
		"неопределённая идентичность, навязанная и сформированная": selfUndecided.phrase,
	} {
		if got := humanLeading(block(leading)); got != want {
			t.Errorf("%s: %s", leading, got)
		}
	}
}

func TestTestAnswerV7Check(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v7", a.TestPortrait(fixedTestSession(a, a.bank.Version)))
	user := string(canonicalJSON(payload))
	if problems := checkTestAnswerV7(goodTestAnswerV7(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	first := func(a map[string]any) map[string]any { return a["recommendations"].([]any)[0].(map[string]any) }
	paragraph := func(key, text string) func(map[string]any) {
		return func(a map[string]any) { a["paragraphs"].(map[string]any)[key] = text }
	}
	for name, c := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"opening": {paragraph("values", "Студент видит себя ответственным гражданином и будущим военным специалистом: он добросовестен. С людьми уважителен. Работать сообща ему труднее."),
			"paragraphs.values: не начинай со слов «Студент видит себя…»"},
		"status": {paragraph("will", "Владеет собой хорошо. Опыт использует. По военной специальности ведущий статус — сформированная идентичность."),
			"paragraphs.will: слово «статус»"},
		"self": {paragraph("will", "Владеет собой хорошо и настойчив в деле. Свой опыт он использует и разбирает. Учится с интересом."),
			"paragraphs.will: не сказано, как студент выбрал военную специальность"},
		"gap": {paragraph("image", "Образ себя сейчас и желаемый образ согласованы, расхождения оптимальны. Он ставит цели. Он растёт."),
			"paragraphs.image: слово «расхождение»"},
		"accepted": {paragraph("image", "Он трезво видит, куда расти. Образ защитника в целом принят и во многом достигнут. Больше всего ему предстоит сделать в команде."),
			"paragraphs.image: «образ принят»"},
		"group": {paragraph("image", "Он трезво видит, куда расти. Ближе всего он в верности долгу и преданности Отечеству. Меньше — в команде."),
			"paragraphs.image: «верность долгу и преданность Отечеству»"},
		"kind": {paragraph("environment", "Активнее всего студент в воспитательной и повседневной деятельности. Учится с интересом. Поддерживают его ясные требования."),
			"paragraphs.environment: вид деятельности по названию"},
		"mentoring": {func(a map[string]any) {
			first(a)["text"] = "На полевом выходе поручить ему наставничество над младшими товарищами по отделению и разобрать итог вместе с ними. Так воспитывается ответственность."
		}, "recommendations.1.text: слово «наставничество»"},
		"official": {func(a map[string]any) {
			first(a)["text"] = "Куратору взвода на полевом выходе поручить ему задачу, которую в одиночку не выполнить, и разобрать итог с отделением. Так воспитывается ответственность."
		}, "recommendations.1.text: без должностей"},
		"teacher": {func(a map[string]any) {
			first(a)["text"] = "Преподаватель на практическом занятии поручает ему показать приём взводу и потом проверяет, как каждый выполнил его сам. Так воспитывается ответственность."
		}, "recommendations.1.text: без должностей"},
		"conclusion": {func(a map[string]any) {
			a["conclusion"] = "Мотивированный студент. Главный резерв — перевод личной надёжности в командную работу, чтобы сблизить образ себя с образом защитника."
		}, "conclusion: общий вывод — обычными словами"},
	} {
		t.Run(name, func(t *testing.T) {
			answer := goodTestAnswerV7(user)
			c.mutate(answer)
			problems := checkTestAnswerV7(answer, payload)
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.HasPrefix(p, c.problem) }) {
				t.Fatalf("want %q, got %v", c.problem, problems)
			}
		})
	}
}

// A student who has not chosen the military speciality himself needs a point
// of group 2 about it.
func TestHumanSpecialityPoint(t *testing.T) {
	payload := newObject()
	block := newObject()
	block.set("профессия", "Военная специальность")
	block.set("что_значит", selfSearching.phrase)
	payload.set("самоопределение", []any{block})
	answer := map[string]any{"recommendations": []any{map[string]any{"group": 1, "code": "R01", "text": goodHumanAction}}}
	if problems := humanSpecialityProblems(answer, payload); len(problems) != 1 || strings.Contains(problems[0], "статус") {
		t.Fatalf("no point of group 2: %v", problems)
	}
	answer["recommendations"] = append(answer["recommendations"].([]any), map[string]any{"group": 2, "code": "R05", "text": "Обсудить с ним, что лично ему даёт выбор военной специальности."})
	if problems := humanSpecialityProblems(answer, payload); len(problems) != 0 {
		t.Fatal(problems)
	}
	block.set("что_значит", selfFormed.phrase)
	answer["recommendations"] = answer["recommendations"].([]any)[:1]
	if problems := humanSpecialityProblems(answer, payload); len(problems) != 0 {
		t.Fatal(problems)
	}
}

func TestPlatoonAnswerV5Check(t *testing.T) {
	a := testApp(t)
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	owner := ""
	for _, r := range db.Sessions {
		if r.Session.Methods && r.Session.Completed && r.Session.Group == "104" {
			owner = r.Session.TeacherID
		}
	}
	payload := a.platoonPayloadFor("v5", a.platoonPortrait("104", a.platoonStudents(db, owner, "104")))
	user := string(canonicalJSON(payload))
	if problems := checkPlatoonAnswerV5(goodPlatoonAnswerV5(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	answer := goodPlatoonAnswerV5(user)
	individual := answer["individual"].([]any)
	if len(individual) == 0 {
		t.Fatal("platoon 104 has no groups for separate work")
	}
	individual[0].(map[string]any)["text"] = "Психологу провести с этими студентами беседу о службе и после занятия обсудить, что им мешает собраться. Так воспитывается собранность."
	answer["paragraphs"].(map[string]any)["will"] = "Студенты владеют собой. Опыт используют. По военной специальности чаще всего мораторий."
	problems := checkPlatoonAnswerV5(answer, payload)
	for _, want := range []string{"individual.1.text: без должностей", "paragraphs.will: слово «мораторий»", "paragraphs.will: не сказано, как студенты выбрали"} {
		if !slices.ContainsFunc(problems, func(p string) bool { return strings.HasPrefix(p, want) }) {
			t.Errorf("want %q, got %v", want, problems)
		}
	}
}

// The bases of the groups for separate work in plain words: the bases of the
// same phrase go together, all five groups are all the qualities.
func TestPlainWorkBases(t *testing.T) {
	a := testApp(t)
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	owner := ""
	for _, r := range db.Sessions {
		if r.Session.Methods && r.Session.Completed && r.Session.Group == "104" {
			owner = r.Session.TeacherID
		}
	}
	data := a.platoonPortrait("104", a.platoonStudents(db, owner, "104"))
	plain := []string{}
	for _, g := range data.WorkGroups {
		if len(g.Plain) == 0 {
			t.Errorf("%s: no plain bases", g.Label)
		}
		for _, basis := range g.Plain {
			if problems := humanWordProblems("basis", basis); len(problems) > 0 {
				t.Errorf("%s: %q: %v", g.Label, basis, problems)
			}
		}
		plain = append(plain, g.Plain...)
	}
	for _, want := range []string{
		"все качества защитника Отечества: пока не считает это для себя важным",
		"о выборе военной специальности пока всерьёз не задумывается",
		"с военной специальностью ещё выбирают и сомневаются",
		"трудно справляться с волнением",
	} {
		if !slices.Contains(plain, want) {
			t.Errorf("no %q in %v", want, plain)
		}
	}
	if got := forCount(2, humanWork[situationFar]); got != "это для них важно, но до этого пока далеко" {
		t.Error(got)
	}
}

// While the portrait by v7 is made, the one by v6 is shown; the answer of v7
// has no who.
func TestPortraitV6IsShownWhileV7IsMade(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	prompts = upTo(prompts, testPortraitTask, 7)
	older := upTo(prompts, testPortraitTask, 6)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(older...)
	answer := func(_ int, c Completion) (string, error) {
		data, _ := json.Marshal(goodTestAnswerV2(c.User))
		return string(data), nil
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: answer})
	f.service.queuePortraits(Source{"session", f.session})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v6" {
		t.Fatalf("v6: %+v", view)
	}
	f.service.UseModel(nil)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(prompts...)
	view, err := f.service.PortraitFor(f.teacherID, testPortraitTask, f.session)
	if err != nil || view.Status != portraitReady || view.Portrait.Version != "v6" {
		t.Fatalf("v6 while v7 is not made: %+v %v", view, err)
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: answer})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v7" || strings.Contains(string(view.Portrait.Output), `"who"`) {
		t.Fatalf("v7: %+v", view)
	}
}
