package usecase

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// goodAction is an action of v6 the checks take: who, kind and a concrete text.
func goodAction(group int, code, text string) map[string]any {
	return map[string]any{"group": group, "code": code, "who": "куратор взвода", "kind": "повседневная", "text": text}
}

const goodActionText = "На ближайших сборах поручить студенту вести разбор выполненной задачи с отделением, чтобы он связал свой опыт с общим результатом."

// goodTestAnswerV6 answers test/v6 the way it asks, for the input it is given.
func goodTestAnswerV6(user string) map[string]any {
	answer := goodTestAnswerV5(user)
	var input struct {
		Identity []struct {
			Leading string `json:"ведущий_статус"`
		} `json:"профессиональная_идентичность"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	for _, item := range answer["recommendations"].([]any) {
		record := item.(map[string]any)
		record["who"], record["kind"] = "куратор взвода", "повседневная"
		text := goodActionText
		if record["group"] == 2 {
			text = "После полевого занятия обсудить со студентом, что ему дало это занятие для самоопределения как военного специалиста и личного смысла выбора."
		}
		record["text"] = text
	}
	return answer
}

// goodPlatoonAnswerV4 answers test-platoon/v4 the way it asks, for the input.
func goodPlatoonAnswerV4(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input struct {
		Identity []struct {
			Leading map[string]int `json:"ведущий_статус"`
		} `json:"профессиональная_идентичность"`
		Catalogue []struct {
			Code   string `json:"код"`
			Groups []int  `json:"группы"`
		} `json:"каталог_рекомендаций"`
		Work []struct {
			Label      string `json:"метка"`
			Directions []struct {
				Code string `json:"код"`
			} `json:"подходящие_направления"`
		} `json:"отдельная_работа"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	statuses := []string{}
	for _, block := range input.Identity {
		for name := range block.Leading {
			if name != levelText[""] {
				statuses = append(statuses, name)
			}
		}
	}
	slices.Sort(statuses)
	will := "Студенты в основном владеют собой. Опыт используют не все."
	if len(statuses) > 0 {
		will += " Во взводе встречаются статусы: " + strings.Join(statuses, ", ") + "."
	}
	recommendations, used := []any{}, map[string]bool{}
	for group := 1; group <= 3; group++ {
		for _, d := range input.Catalogue {
			if slices.Contains(d.Groups, group) && !used[d.Code] {
				used[d.Code] = true
				recommendations = append(recommendations, goodAction(group, d.Code, "На ближайшем занятии разобрать со взводом выполненную задачу, чтобы каждый увидел свою часть общего результата."))
				break
			}
		}
	}
	individual := []any{}
	for _, g := range input.Work {
		individual = append(individual, map[string]any{"label": g.Label, "code": g.Directions[0].Code, "who": "психолог", "kind": "повседневная",
			"text": "После ближайшего занятия поговорить с этими студентами о том, что им мешает, и вместе выбрать одно дело на следующую неделю службы."})
	}
	return map[string]any{
		"paragraphs": map[string]any{
			"values":      "Представления студентов о себе складываются неравномерно. У большинства значимы ценности службы. Готовность действовать заодно с товарищами выражена слабее.",
			"will":        will,
			"image":       "Каким хотят стать, студенты видят по-разному. Ближе всего взвод к образу в верности долгу, дальше всего — в товариществе и взаимовыручке.",
			"environment": "Взвод активнее в учебной деятельности. Выше других взвод оценивает доверие преподавателей. Недостаток общих задач сдерживает приближение к образу.",
		},
		"recommendations": recommendations,
		"individual":      individual,
		"conclusion":      "Взвод ответственный. Главное в работе с ним — совместные задачи.",
	}
}

func TestTestAnswerV6Check(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v6", a.TestPortrait(fixedTestSession(a, a.bank.Version)))
	user := string(canonicalJSON(payload))
	if problems := checkTestAnswerV6(goodTestAnswerV6(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	first := func(a map[string]any) map[string]any { return a["recommendations"].([]any)[0].(map[string]any) }
	for name, c := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"who": {func(a map[string]any) { first(a)["who"] = "командир" }, "recommendations.1: who — одно из"},
		"kind": {func(a map[string]any) { first(a)["kind"] = "служебная" }, "recommendations.1: kind — одно из"},
		"short": {func(a map[string]any) {
			first(a)["text"] = "Провести тренинг саморегуляции и стрессоустойчивости для развития самообладания."
		}, "recommendations.1.text: слишком коротко"},
		"frequency": {func(a map[string]any) {
			first(a)["text"] = "Раз в неделю разбирать со студентом после занятий, что получилось и что нет, чтобы он видел свой опыт."
		}, "recommendations.1.text: без дат, сроков и частоты («раз в…»)"},
		"month": {func(a map[string]any) {
			first(a)["text"] = "В ноябре на сборах поручить студенту вести разбор выполненной задачи с отделением, чтобы он видел свой опыт."
		}, "recommendations.1.text: без дат, сроков и частоты («ноябр…»)"},
		"copied example": {func(a map[string]any) {
			first(a)["text"] = "Перед нормативами разобрать со студентом, в каких ситуациях на занятиях и в наряде он теряет выдержку."
		}, "recommendations.1.text: повторяет пример из инструкции"},
		"direction name": {func(a map[string]any) {
			first(a)["code"] = "R19"
			first(a)["text"] = "Провести для студента тренинг саморегуляции и стрессоустойчивости, чтобы он научился лучше справляться с нагрузкой на занятиях."
		}, "recommendations.1.text: повторяет название направления «тренинг саморегуляции и стрессоустойчивости»"},
		"latin": {func(a map[string]any) {
			a["conclusion"] = "Студент ответственный и устойчивый. Главное в работе с ним — feedback."
		}, "conclusion: буквы не русского алфавита"},
		"chinese": {func(a map[string]any) {
			a["paragraphs"].(map[string]any)["values"] = "Студент превращает трудности в动力 для роста. Ценности службы для него значимы. Товарищество выражено слабее."
		}, "paragraphs.values: буквы не русского алфавита"},
		"no status": {func(a map[string]any) {
			a["paragraphs"].(map[string]any)["will"] = "Студент владеет собой и настойчив. Опыт он использует и осмысливает."
		}, "paragraphs.will: не назван ведущий статус по военной специальности — «"},
		"term": {func(a map[string]any) {
			a["paragraphs"].(map[string]any)["image"] = "Я-идеальное студента согласовано. Ближе всего к образу он в верности долгу."
		}, "paragraphs.image: термин «Я-идеал…»"},
	} {
		answer := goodTestAnswerV6(user)
		c.mutate(answer)
		if problems := strings.Join(checkTestAnswerV6(answer, payload), "; "); !strings.Contains(problems, c.problem) {
			t.Errorf("%s: %q", name, problems)
		}
	}
}

// A status of identity is named by its name or by its meaning in the input.
func TestStatusNamedByItsMeaning(t *testing.T) {
	for name, text := range map[string]string{
		"неопределённая идентичность": "По военной специальности студент пока всерьёз не задумывается о выборе.",
		"мораторий (кризис выбора)":   "Студент в поиске: сомневается, решения ещё нет.",
		"навязанная":                  "Выбор сделан под влиянием других.",
		"сформированная":              "Выбор сделан самостоятельно, после раздумий.",
	} {
		if !statusNamedPlain(normalizedText(text), name) {
			t.Errorf("%s: %q", name, text)
		}
	}
	if statusNamedPlain(normalizedText("Студент активен на занятиях."), "неопределённая идентичность") {
		t.Fatal("named without the name or the meaning")
	}
	for name, meaning := range statusMeanings {
		if !statusNamedPlain(normalizedText(meaning), name) {
			t.Errorf("the meaning of %s does not name it", name)
		}
	}
}

func TestPlatoonAnswerV4Check(t *testing.T) {
	a := testApp(t)
	for group, portraits := range demoPlatoons(t, a) {
		payload := a.platoonPayloadFor("v4", a.platoonPortrait(group, portraits))
		user := string(canonicalJSON(payload))
		if problems := checkPlatoonAnswerV4(goodPlatoonAnswerV4(user), payload); len(problems) != 0 {
			t.Fatalf("%s: a good answer fails: %v", group, problems)
		}
		if len(objectList(payload.values["отдельная_работа"])) < 2 {
			continue
		}
		individual := func(a map[string]any) []any { return a["individual"].([]any) }
		for name, c := range map[string]struct {
			mutate  func(map[string]any)
			problem string
		}{
			"missing group": {func(a map[string]any) { a["individual"] = individual(a)[1:] }, "individual: нужно ровно"},
			"unknown label": {func(a map[string]any) { individual(a)[0].(map[string]any)["label"] = "Я" }, "individual.1: label — метка группы"},
			"repeated label": {func(a map[string]any) {
				individual(a)[1].(map[string]any)["label"] = individual(a)[0].(map[string]any)["label"]
			}, "individual.2: группа «А» уже есть"},
			"unfit code": {func(a map[string]any) { individual(a)[0].(map[string]any)["code"] = "R99" }, "individual.1: код не из «подходящие_направления»"},
			"short":      {func(a map[string]any) { individual(a)[0].(map[string]any)["text"] = "Тренинг саморегуляции." }, "individual.1.text: слишком коротко"},
			"numeral": {func(a map[string]any) {
				individual(a)[0].(map[string]any)["text"] = "Двум студентам после ближайшего наряда разобрать с куратором, что получилось и что нет, и выбрать одно дело."
			}, "число студентов числительным"},
			"term in individual": {func(a map[string]any) {
				individual(a)[0].(map[string]any)["text"] = "После наряда разобрать с ними, почему Я-идеальное ниже, и выбрать одно дело на следующий наряд службы."
			}, "individual.1.text: термин «Я-идеал…»"},
		} {
			answer := goodPlatoonAnswerV4(user)
			c.mutate(answer)
			if problems := strings.Join(checkPlatoonAnswerV4(answer, payload), "; "); !strings.Contains(problems, c.problem) {
				t.Errorf("%s, %s: %q", group, name, problems)
			}
		}
	}
	// Without groups individual is empty.
	payload := a.platoonPayloadFor("v4", a.platoonPortrait("101", fixedPlatoon(a, a.bank.Version)))
	payload.remove("отдельная_работа")
	answer := goodPlatoonAnswerV4(string(canonicalJSON(payload)))
	answer["individual"] = []any{map[string]any{"label": "А", "code": "R06", "who": "психолог", "kind": "повседневная", "text": goodActionText}}
	if problems := strings.Join(checkPlatoonAnswerV4(answer, payload), "; "); !strings.Contains(problems, "нужен пустой список") {
		t.Fatalf("no groups: %q", problems)
	}
}

// A recommendation with a code that does not fit its group goes when enough
// are left (decision Д13), and stays for the check when it would leave fewer.
func TestUnfitCodesAreCleaned(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v6", a.TestPortrait(fixedTestSession(a, a.bank.Version)))
	answer := goodTestAnswerV6(string(canonicalJSON(payload)))
	items := answer["recommendations"].([]any)
	if len(items) != 3 {
		t.Fatalf("recommendations: %d", len(items))
	}
	answer["recommendations"] = append(items, goodAction(1, "R99", goodActionText))
	cleaned, dropped := cleanUnfitCodes(a, model.Database{}, Source{}, answer, payload)
	if dropped != 1 || len(cleaned["recommendations"].([]any)) != 3 {
		t.Fatalf("dropped %d: %v", dropped, cleaned["recommendations"])
	}
	answer = goodTestAnswerV6(string(canonicalJSON(payload)))
	answer["recommendations"].([]any)[0].(map[string]any)["code"] = "R99"
	if _, dropped := cleanUnfitCodes(a, model.Database{}, Source{}, answer, payload); dropped != 0 {
		t.Fatalf("dropped below the floor: %d", dropped)
	}
}

// The examples of the program are those of the instructions in the wiki.
func TestActionExamplesAreThoseOfTheWiki(t *testing.T) {
	student, err := os.ReadFile("../../../Вики/Тестирование/Инструкция v6.md")
	if err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	platoon, _ := os.ReadFile("../../../Вики/Тестирование/Инструкция по взводу v4.md")
	student7, _ := os.ReadFile("../../../Вики/Тестирование/Инструкция v7.md")
	platoon5, _ := os.ReadFile("../../../Вики/Тестирование/Инструкция по взводу v5.md")
	wiki := string(student) + string(platoon) + string(student7) + string(platoon5)
	for _, example := range actionExamples {
		if !strings.Contains(wiki, "«"+example+"»") {
			t.Errorf("not in the wiki: %s", example)
		}
	}
	for _, who := range actionWho {
		if !strings.Contains(string(student), "«"+who+"»") {
			t.Errorf("who %s is not in the instruction", who)
		}
	}
	for name, meaning := range statusMeanings {
		if !strings.Contains(string(student), name+" — "+meaning) {
			t.Errorf("the meaning of %s differs from the wiki", name)
		}
	}
}

// While the portrait by v6 is made, the one by v5 is shown.
func TestPortraitV5IsShownWhileV6IsMade(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	// Without the versions after v6: this test is about v6.
	prompts = upTo(prompts, testPortraitTask, 6)
	older := upTo(prompts, testPortraitTask, 5)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(older...)
	answer := func(_ int, c Completion) (string, error) {
		data, _ := json.Marshal(goodTestAnswerV2(c.User))
		return string(data), nil
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: answer})
	f.service.queuePortraits(Source{"session", f.session})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v5" {
		t.Fatalf("v5: %+v", view)
	}
	f.service.UseModel(nil)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(prompts...)
	view, err := f.service.PortraitFor(f.teacherID, testPortraitTask, f.session)
	if err != nil || view.Status != portraitReady || view.Portrait.Version != "v5" {
		t.Fatalf("v5 while v6 is not made: %+v %v", view, err)
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: answer})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v6" || !strings.Contains(string(view.Portrait.Output), `"who"`) {
		t.Fatalf("v6: %+v", view)
	}
}

// The head of the name of a direction, word after word in any form, is a
// repeated name; a head of two words is an ordinary phrase.
func TestRepeatsDirection(t *testing.T) {
	for text, want := range map[string]bool{
		"Провести индивидуальные беседы-диалоги о ценностях и смыслах службы с теми, кто не принял образ.": true,
		"Провести тренинг саморегуляции и стрессоустойчивости для тех, у кого эмоциональная устойчивость низкая.": true,
		"Перед нормативами научить студента паре приёмов, которые помогают сохранять выдержку.":                 false,
		"Привлечь студента к волонтёрской деятельности взвода на ближайшей акции памяти.":                       false,
	} {
		code := "R19"
		if strings.Contains(text, "беседы") {
			code = "R06"
		}
		if strings.Contains(text, "волонт") {
			code = "R09"
		}
		names := map[string]string{"R06": "Индивидуальная беседа-диалог о ценностях и смыслах службы", "R09": "Волонтёрская деятельность", "R19": "Тренинг саморегуляции и стрессоустойчивости"}
		if got := repeatsDirection(text, names[code]); got != want {
			t.Errorf("%s: %v", text, got)
		}
	}
}
