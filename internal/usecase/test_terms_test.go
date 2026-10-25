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

// goodTestAnswerV5 answers test/v5 the way it asks, for the input it is given:
// four paragraphs without the terms of the model, a direction once.
func goodTestAnswerV5(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input struct {
		Groups []struct {
			Number  int      `json:"номер"`
			Fitting []string `json:"подходящие_направления"`
		} `json:"образ_защитника"`
		Identity []struct {
			Leading string `json:"ведущий_статус"`
		} `json:"профессиональная_идентичность"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	will := "Студент владеет собой и настойчив. Опыт он использует и осмысливает."
	if len(input.Identity) > 0 {
		leading := []string{}
		for _, block := range input.Identity {
			leading = append(leading, block.Leading)
		}
		will += " Ведущие статусы — " + strings.Join(leading, "; ") + "."
	}
	recommendations := []any{}
	used := map[string]bool{}
	for _, g := range input.Groups {
		for _, code := range g.Fitting {
			if len(recommendations) < 3 && !used[code] {
				text := "Включать студента в эту работу, чтобы образ стал ближе."
				if g.Number == 2 {
					text = "Включать студента в эту работу ради самоопределения как военного специалиста и личного смысла выбора."
				}
				used[code] = true
				recommendations = append(recommendations, map[string]any{"group": g.Number, "code": code, "text": text})
				break
			}
		}
	}
	return map[string]any{
		"paragraphs": map[string]any{
			"values":      "Представления студента о себе складываются вокруг ответственности за страну. Ценности службы для него значимы. Готовность действовать заодно с товарищами выражена слабее.",
			"will":        will,
			"image":       "Каким студент хочет стать, он видит ясно, и это согласуется с тем, каким он видит себя сейчас. Ближе всего к образу защитника он в верности долгу, дальше всего — в товариществе и взаимовыручке.",
			"environment": "Активнее всего студент в учебной деятельности, реже включается в общие дела взвода. Выше всего он оценивает доверие преподавателей. Недостаток совместных задач сдерживает его приближение к образу.",
		},
		"recommendations": recommendations,
		"conclusion":      "Студент ответственный и устойчивый. Главное в работе с ним — совместные задачи.",
	}
}

// goodPlatoonAnswerV3 answers test-platoon/v3 the way it asks, for the input.
func goodPlatoonAnswerV3(user string) map[string]any {
	answer := goodPlatoonAnswer(strings.Replace(user, `"словарь"`, `"-"`, 1))
	return map[string]any{
		"paragraphs": map[string]any{
			"values":      "Представления студентов о себе складываются неравномерно. У большинства значимы ценности службы. Готовность действовать заодно с товарищами выражена слабее.",
			"will":        "Студенты в основном владеют собой. Опыт используют не все. " + answer["identity"].(string),
			"image":       "Каким хотят стать, студенты видят по-разному. Ближе всего взвод к образу в верности долгу, дальше всего — в товариществе и взаимовыручке.",
			"environment": "Взвод активнее в учебной деятельности. Выше всего взвод оценивает доверие преподавателей. Недостаток общих задач сдерживает приближение к образу.",
		},
		"recommendations": answer["recommendations"],
		"individual":      []any{"Студентам, которые ставят себе планку ниже, чем оценивают себя сейчас, нужна отдельная беседа."},
		"conclusion":      "Взвод ответственный. Главное в работе с ним — совместные задачи.",
	}
}

// From v5 on the input has the dictionary of the terms first and the sign
// «Осмысление опыта» without the term; the rest is the input of v4.
func TestTestPayloadV5AddsTheTerms(t *testing.T) {
	a := testApp(t)
	data := climateSession(a, "А", 3, 3, 2, 2, 2, 1, 1, 1, 0, 0, -1, -1, -2, -2).data
	old, payload := a.testPayloadFor("v4", data), a.testPayloadFor("v5", data)
	if !slices.Equal(payload.keys, append([]string{"словарь"}, old.keys...)) {
		t.Fatalf("keys of v5: %v", payload.keys)
	}
	for _, key := range old.keys {
		if key != "признаки_среды" && string(canonicalJSON(payload.values[key])) != string(canonicalJSON(old.values[key])) {
			t.Errorf("%s differs in v5", key)
		}
	}
	terms := objectList(payload.values["словарь"].(*object).values["термины"])
	if len(terms) != len(modelTerms) || objectText(terms[4], "термин") != "соборность" {
		t.Fatalf("dictionary: %s", canonicalJSON(payload.values["словарь"]))
	}
	if got := objectText(signByName(payload, "Осмысление опыта"), "по_главе_1"); got != reflectionPlain {
		t.Fatalf("reflection: %q", got)
	}
	if objectText(signByName(old, "Осмысление опыта"), "по_главе_1") == reflectionPlain {
		t.Fatal("the input of v4 changed")
	}
	// The personal data cleaning keeps the dictionary as it is.
	if _, err := maskPersonal(payload); err != nil || len(objectList(payload.values["словарь"].(*object).values["термины"])) != len(modelTerms) {
		t.Fatalf("masked: %v", err)
	}
}

func TestPlatoonPayloadV3AddsTheTerms(t *testing.T) {
	a := testApp(t)
	data := a.platoonPortrait("101", []studentPortrait{
		climateSession(a, "А", 3, 3, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "Б", 3, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "В", 3, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
	})
	old, payload := a.platoonPayloadFor("v2", data), a.platoonPayloadFor("v3", data)
	if !slices.Equal(payload.keys, append([]string{"словарь"}, old.keys...)) || !slices.Contains(payload.keys, "климат_взвода") {
		t.Fatalf("keys of v3: %v", payload.keys)
	}
	if got := objectText(signByName(payload, "Осмысление опыта"), "по_главе_1"); got != reflectionPlain {
		t.Fatalf("reflection: %q", got)
	}
	// A platoon without the climate gets the dictionary too.
	if other := a.platoonPayloadFor("v3", a.platoonPortrait("101", fixedPlatoon(a, "pilot-2.5"))); other.keys[0] != "словарь" || slices.Contains(other.keys, "климат_взвода") {
		t.Fatalf("keys without the climate: %v", other.keys)
	}
}

// The dictionary of the program is the one of the instruction in the wiki.
func TestTermsAreThoseOfTheWiki(t *testing.T) {
	wiki, err := os.ReadFile("../../../Вики/Тестирование/Инструкция v5.md")
	if err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	for _, term := range modelTerms {
		line, _ := json.Marshal(map[string]string{"значит": term.Meaning})
		if !strings.Contains(string(wiki), `{"термин": "`+term.Term+`", "значит": `+strings.TrimSuffix(strings.TrimPrefix(string(line), `{"значит":`), "}")+"}") {
			t.Errorf("%s differs from the wiki", term.Term)
		}
	}
	if !strings.Contains(strings.Join(strings.Fields(string(wiki)), " "), reflectionPlain) {
		t.Error("the sign «Осмысление опыта» differs from the wiki")
	}
}

func TestTermProblems(t *testing.T) {
	for text, want := range map[string]string{
		"Отчётливо выражены ответственность за страну и готовность действовать заодно с товарищами.": "",
		"Участвует в военно-патриотических мероприятиях.":                                            "",
		"Физическая и психологическая устойчивость выражена сильнее.":                                "",
		"Включённость в деятельность и осмысление опыта выражены слабее.":                            "",
		"Взаимодействие и общение с преподавателями он ценит.":                                       "",
		"Сильнее выражен патриотизм.":                                                                "патриотизм",
		"Расхождение Я-реального и Я-идеального оптимально.":                                         "Я-реальное, Я-идеальное",
		"Я идеальное ниже реального.":                                                                "Я-идеальное",
		"Реалистичный Я-идеал.":                                                                      "Я-идеальное",
		"Я-концепция устойчива.":                                                                     "Я-концепция",
		"Ценностно-смысловой компонент сформирован.":                                                 "ценностно-смысловой, компонент",
		"Слабее выражены любовь и соборность.":                                                       "любовь, соборность",
		"Признаки выражены неравномерно.":                                                            "признак",
		"Слабее сотрудничество.":                                                                     "сотрудничество",
		"Дальше всего — в физических и психологических качествах.":                                   "физические и психологические качества",
		"Ближе к образу в коммуникативных и коллективистских.":                                       "коммуникативные и коллективистские качества",
		"Дисциплинарные и волевые качества сильнее.":                                                 "дисциплинарные и волевые качества",
		"В государственно-патриотических качествах образ принят.":                                    "государственно-патриотические качества",
		"Профессионально-компетентностные и когнитивные качества слабее.":                            "профессионально-компетентностные качества, когнитивные",
		"Слабее ясность и согласованность норм.":                                                     "ясность норм",
		"Ниже оценивает ясность норм.":                                                               "ясность норм",
		"Ценит личностно ориентированное взаимодействие.":                                            "личностно ориентированное взаимодействие",
		"Значимая деятельность и инициатива выражены слабее.":                                        "значимая деятельность и инициатива",
	} {
		found := []string{}
		for _, p := range termProblems("paragraphs.values", text) {
			_, rest, _ := strings.Cut(p, "термин «")
			name, _, _ := strings.Cut(rest, "»")
			found = append(found, name)
		}
		if got := strings.Join(found, ", "); got != want {
			t.Errorf("%q: %q, want %q", text, got, want)
		}
	}
}

func TestKindsListProblem(t *testing.T) {
	for text, bad := range map[string]bool{
		"Студент включён во все четыре вида деятельности.":           true,
		"Во всех четырёх видах деятельности он активен.":             true,
		"В четырёх видах деятельности уровень разный.":               true,
		"Во всех видах деятельности студент на высоком уровне.":      true,
		"Активнее всего он в учебной и воспитательной деятельности.": false,
		"Виды деятельности ВУЦ дают ему опыт.":                       false,
	} {
		if got := len(kindsListProblems("paragraphs.environment", text)) > 0; got != bad {
			t.Errorf("%q: problem %v", text, got)
		}
	}
}

// The answer to test/v5: the rules of v4 that still apply and the new ones.
func TestTestAnswerV5Check(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v5", climateSession(a, "А", 3, 3, 2, 2, 2, 1, 1, 1, 0, 0, -1, -1, -2, -2).data)
	user := string(canonicalJSON(payload))
	if problems := checkTestAnswerV5(goodTestAnswerV5(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	paragraph := func(key, text string) func(map[string]any) {
		return func(a map[string]any) { a["paragraphs"].(map[string]any)[key] = text }
	}
	for name, c := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"term": {paragraph("values", "Сильнее выражены патриотизм и ответственность. Ценности службы значимы."), "paragraphs.values: термин «патриотизм» — пиши не термин, а его содержание"},
		"term in a recommendation": {func(a map[string]any) {
			a["recommendations"].([]any)[0].(map[string]any)["text"] = "Развивать соборность через общие задачи."
		}, "recommendations.1.text: термин «соборность»"},
		"term in the conclusion": {func(a map[string]any) {
			a["conclusion"] = "Я-концепция студента устойчива. Главное — общие задачи."
		}, "conclusion: термин «Я-концепция»"},
		"list of kinds":   {paragraph("environment", "Студент включён во все четыре вида деятельности. Выше оценивает доверие."), "paragraphs.environment: не перечисляй виды деятельности"},
		"one sentence":    {paragraph("image", "Образ принят."), "paragraphs.image: предложений 1 вместо 2–5"},
		"no paragraph":    {func(a map[string]any) { delete(a["paragraphs"].(map[string]any), "will") }, "нет поля paragraphs.will"},
		"group number":    {paragraph("image", "Дальше всего он в группе 3. Образ принят."), "группу образа называй её качествами, например «в товариществе и взаимовыручке»"},
		"climate level":   {paragraph("environment", "Климат взвода студент видит благоприятным. Выше оценивает доверие."), "у климата нет уровня"},
		"rated level":     {paragraph("environment", "Студент высоко оценивает доверие преподавателей. Это поддерживает его."), "слово «высоко»"},
		"no status named": {paragraph("will", "Студент владеет собой. Опыт осмысливает."), "paragraphs.will: не назван ведущий статус"},
		"repeated code": {func(a map[string]any) {
			list := a["recommendations"].([]any)
			a["recommendations"] = append(list, list[0])
		}, "recommendations.4: направление"},
	} {
		answer := goodTestAnswerV5(user)
		c.mutate(answer)
		problems := strings.Join(checkTestAnswerV5(answer, payload), "; ")
		if !strings.Contains(problems, c.problem) {
			t.Errorf("%s: %q", name, problems)
		}
		if strings.Contains(problems, "коммуникативных и коллективистских") {
			t.Errorf("%s: the hint names a group by its label: %q", name, problems)
		}
	}
}

// A direction goes once a portrait since v5, whatever the group: Волков on
// 25 October got R15 for groups 3 and 4.
func TestRepeatedCodes(t *testing.T) {
	for name, c := range map[string]struct {
		given   []any
		left    string
		changes int
		problem string
	}{
		"Волков":     {recommendationList(5.0, "R06", 3.0, "R15", 3.0, "R16", 4.0, "R15"), "R06 R15 R16", 1, ""},
		"three only": {recommendationList(3.0, "R15", 4.0, "R15", 1.0, "R16"), "R15 R15 R16", 0, "recommendations.2: направление R15 уже есть — возьми другое направление, каждое — один раз"},
		"no repeat":  {recommendationList(3.0, "R15", 4.0, "R14", 1.0, "R16"), "R15 R14 R16", 0, ""},
	} {
		output, changes := cleanRepeatedCodes(nil, model.Database{}, Source{}, map[string]any{"recommendations": c.given}, newObject())
		left := []string{}
		for _, item := range output["recommendations"].([]any) {
			code, _ := codeKey(item)
			left = append(left, code)
		}
		problems := strings.Join(repeatedCodeProblems(output), "; ")
		if strings.Join(left, " ") != c.left || changes != c.changes || problems != c.problem {
			t.Errorf("%s: %v, %d changes, %q", name, left, changes, problems)
		}
	}
}

// A student whose groups are all «образ принят и во многом достигнут» has two
// directions only: two recommendations are enough, a repeat is dropped down to
// them.
func TestFewerDirectionsGiveFewerRecommendations(t *testing.T) {
	payload := newObject()
	groups := []any{}
	for n := 1; n <= 5; n++ {
		g := newObject()
		g.set("номер", n)
		fitting := []any{"R16"}
		if n >= 2 && n <= 4 {
			fitting = []any{"R15", "R16"}
		}
		g.set("подходящие_направления", fitting)
		groups = append(groups, g)
	}
	payload.set("образ_защитника", groups)
	if floor := recommendationFloor(payload); floor != 2 {
		t.Fatalf("floor: %d", floor)
	}
	output, changes := cleanRepeatedCodes(nil, model.Database{}, Source{}, map[string]any{"recommendations": recommendationList(1.0, "R16", 3.0, "R15", 5.0, "R16")}, payload)
	if left := output["recommendations"].([]any); changes != 1 || len(left) != 2 || len(repeatedCodeProblems(output)) != 0 {
		t.Fatalf("%d changes, left %v", changes, left)
	}
	if floor := recommendationFloor(newObject()); floor != recommendationLeast {
		t.Fatalf("floor of a platoon: %d", floor)
	}
}

func TestPlatoonAnswerV3Check(t *testing.T) {
	a := testApp(t)
	payload := a.platoonPayloadFor("v3", a.platoonPortrait("101", fixedPlatoon(a, "pilot-2.5")))
	user := string(canonicalJSON(payload))
	if problems := checkPlatoonAnswerV3(goodPlatoonAnswerV3(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	for name, c := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"term in individual": {func(a map[string]any) {
			a["individual"] = []any{"Студентам, у которых Я-идеальное ниже Я-реального, — беседа."}
		}, "individual.1: термин «Я-реальное»"},
		"numeral": {func(a map[string]any) {
			a["paragraphs"].(map[string]any)["image"] = "У двух студентов образ не принят. Остальные ближе к образу."
		}, "число студентов числительным («двух»)"},
		"climate not measured": {func(a map[string]any) {
			a["paragraphs"].(map[string]any)["environment"] = "Климат взвода студенты видят по-разному. Выше оценивают доверие."
		}, "о климате или сплочённости взвода, которых тест не меряет"},
		"repeated code": {func(a map[string]any) {
			list := a["recommendations"].([]any)
			a["recommendations"] = append(list, list[0])
		}, "recommendations.4: направление"},
		"no status named": {func(a map[string]any) {
			a["paragraphs"].(map[string]any)["will"] = "Студенты владеют собой. Опыт используют не все."
		}, "paragraphs.will: не назван самый частый ведущий статус"},
	} {
		answer := goodPlatoonAnswerV3(user)
		c.mutate(answer)
		if problems := strings.Join(checkPlatoonAnswerV3(answer, payload), "; "); !strings.Contains(problems, c.problem) {
			t.Errorf("%s: %q", name, problems)
		}
	}
}

// While the portrait by v5 is made, the one by v4 is shown.
func TestPortraitV4IsShownWhileV5IsMade(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	older := slices.DeleteFunc(slices.Clone(prompts), func(p promptbank.Prompt) bool { return p.Task == testPortraitTask && p.Version == "v5" })
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(older...)
	answer := func(_ int, c Completion) (string, error) {
		data, _ := json.Marshal(goodTestAnswerV2(c.User))
		return string(data), nil
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: answer})
	f.service.queuePortraits(Source{"session", f.session})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v4" {
		t.Fatalf("v4: %+v", view)
	}
	f.service.UseModel(nil)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(prompts...)
	view, err := f.service.PortraitFor(f.teacherID, testPortraitTask, f.session)
	if err != nil || view.Status != portraitReady || view.Portrait.Version != "v4" {
		t.Fatalf("v4 while v5 is not made: %+v %v", view, err)
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: answer})
	if view := f.wait(); view.Status != portraitReady || view.Portrait.Version != "v5" || !strings.Contains(string(view.Portrait.Output), `"paragraphs"`) {
		t.Fatalf("v5: %+v", view)
	}
}

// The inputs of test/v4 and test-platoon/v2 must not change either: the
// portraits made by them are shown while those by v5 and v3 are made.
func TestPayloadsV4AndPlatoonV2AreKept(t *testing.T) {
	a := testApp(t)
	keptInput(t, "v4.json", indentedJSON(a.testPayloadFor("v4", a.TestPortrait(fixedTestSession(a, a.bank.Version)))))
	keptInput(t, "platoon-v2.json", indentedJSON(a.platoonPayloadFor("v2", a.platoonPortrait("101", fixedPlatoon(a, a.bank.Version)))))
}
