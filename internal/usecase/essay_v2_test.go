package usecase

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// goodEssayV2For is a good answer of v2 to a small input made by essayPayloadFor.
func goodEssayV2For(payload *object) map[string]any {
	quote := func() []any {
		return []any{map[string]any{"question": "1.1", "quote": "Я пошёл «в ВУЦ» ради «службы»"}}
	}
	before := []any{}
	if _, ok := payload.values["прежнее_эссе"]; ok {
		before = []any{map[string]any{"question": "1.1", "quote": "Раньше я хотел только отсрочку"}}
	}
	return map[string]any{
		"formation":   map[string]any{"text": "Автор пришёл ради отсрочки и стал серьёзнее относиться к службе.", "quotes": quote(), "before": before},
		"moments":     map[string]any{"text": "Для автора важна служба, первые дни дались ему тяжело.", "quotes": []any{quote()[0], map[string]any{"question": "1.2", "quote": "Первые дни было очень страшно"}}},
		"environment": map[string]any{"text": "Автор благодарен преподавателю за помощь.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}},
		"recommendations": []any{
			map[string]any{"text": "Поддержать автора в первые дни.", "quotes": quote()},
			map[string]any{"text": "Опереться на его интерес к службе.", "quotes": quote()},
		},
		"teachers": []any{},
	}
}

func TestCountWords(t *testing.T) {
	for text, want := range map[string]int{"": 0, "Автор — пришёл, 2 раза.": 4, "  слово  ": 1, "«а» - б": 2} {
		if got := countWords(text); got != want {
			t.Errorf("%q: %d, want %d", text, got, want)
		}
	}
}

func TestCheckEssayAnswerV2(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было   очень  страшно, ЁЛКИ-палки.", "2.4": "Хочу поблагодарить полковника Иванова за помощь."}
	earlier := map[string]string{"1.1": "Раньше я хотел только отсрочку от армии.", "1.2": "Раньше было совсем страшно.", "2.4": "Раньше никого не помню."}
	text := func(key string, value string) func(map[string]any) {
		return func(o map[string]any) { o[key].(map[string]any)["text"] = value }
	}
	advice := func(o map[string]any) []any { return o["recommendations"].([]any) }
	cases := map[string]struct {
		previous map[string]string
		labels   []string
		mutate   func(map[string]any)
		want     string
	}{
		"good answer":                    {nil, nil, func(map[string]any) {}, ""},
		"good answer with earlier essay": {earlier, nil, func(map[string]any) {}, ""},
		"quotes of the earlier essay without one": {nil, nil, func(o map[string]any) {
			o["formation"].(map[string]any)["before"] = []any{map[string]any{"question": "1.1", "quote": "Я пошёл в ВУЦ ради"}}
		}, "formation.before: прежнего эссе нет, список должен быть пустым"},
		"no quotes of the earlier essay": {earlier, nil, func(o map[string]any) { o["formation"].(map[string]any)["before"] = []any{} }, "formation.before: цитат должно быть от 1 до 2"},
		"before is looked up in the earlier essay": {earlier, nil, func(o map[string]any) {
			o["formation"].(map[string]any)["before"] = []any{map[string]any{"question": "1.1", "quote": "Я пошёл в ВУЦ ради службы"}}
		}, "formation.before: цитаты нет в ответе 1.1 дословно"},
		"the moments stand on two quotes": {nil, nil, func(o map[string]any) {
			o["moments"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": "Я пошёл в ВУЦ"}}
		}, "moments: цитат должно быть от 2 до 4"},
		"invented quote": {nil, nil, func(o map[string]any) {
			o["environment"].(map[string]any)["quotes"] = []any{map[string]any{"question": "2.4", "quote": "Хочу поблагодарить майора Петрова"}}
		}, "environment: цитаты нет в ответе 2.4 дословно"},
		"no paragraph":       {nil, nil, func(o map[string]any) { delete(o, "environment") }, "environment: нет абзаца"},
		"an empty paragraph": {nil, nil, text("moments", " "), "moments: пустой текст"},
		"one recommendation": {nil, nil, func(o map[string]any) { o["recommendations"] = advice(o)[:1] }, "recommendations: рекомендаций должно быть от 2 до 4"},
		"five recommendations": {nil, nil, func(o map[string]any) {
			item := func(text string) any {
				return map[string]any{"text": text, "quotes": []any{map[string]any{"question": "1.1", "quote": "Я пошёл в ВУЦ"}}}
			}
			o["recommendations"] = []any{item("Первая."), item("Вторая."), item("Третья."), item("Четвёртая."), item("Пятая.")}
		}, "recommendations: рекомендаций должно быть от 2 до 4"},
		"a recommendation repeated": {nil, nil, func(o map[string]any) {
			advice(o)[1].(map[string]any)["text"] = "поддержать  автора в первые дни."
		}, "recommendations, пункт 2: повторяет пункт 1"},
		"a recommendation without quotes": {nil, nil, func(o map[string]any) { advice(o)[1].(map[string]any)["quotes"] = []any{} }, "recommendations, пункт 2: цитат должно быть от 1 до 2"},
		"the number of a question":        {nil, nil, text("environment", "В ответе 2.4 автор благодарит преподавателя."), "environment: номер вопроса в тексте"},
		"a question by word":              {nil, nil, text("formation", "В вопросе 3 автор пишет о службе."), "formation: номер вопроса в тексте"},
		"a label of an instructor":        {nil, []string{"[преподаватель 1]"}, text("environment", "Автор благодарит [преподаватель 1]."), "environment: метка преподавателя в тексте"},
		"a term":                          {nil, nil, text("formation", "Я-концепция автора меняется."), "formation: термин «я-концепция»"},
		"an ideal self":                   {nil, nil, text("moments", "Его Я‑идеальное выше."), "moments: термин «я‑идеальное»"},
		"reflection":                      {nil, nil, text("moments", "У автора есть рефлексия опыта."), "moments: термин «рефлексия»"},
		"the army really is no term":      {nil, nil, text("moments", "Армия реально изменила автора, и я-то тут ни при чём."), ""},
		"a level":                         {nil, nil, text("moments", "У автора высокий уровень мотивации."), "moments: недопустимое слово («высокий уров»)"},
		"to my regret": {nil, nil, func(o map[string]any) {
			advice(o)[0].(map[string]any)["text"] = "К сожалению, автор устал."
		}, "recommendations, пункт 1: недопустимое слово («к сожалению»)"},
		"the instructors are checked as in v1": {nil, []string{"[преподаватель 3]"}, func(o map[string]any) {
			o["teachers"] = []any{map[string]any{"who": "[преподаватель 7]", "kind": "благодарность", "text": "За помощь.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}}
		}, "teachers, запись 1: метка не из списка"},
		"a label in who is right": {nil, []string{"[преподаватель 3]"}, func(o map[string]any) {
			o["teachers"] = []any{map[string]any{"who": "[преподаватель 3]", "kind": "благодарность", "text": "За помощь.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}}
		}, ""},
	}
	for name, c := range cases {
		payload := essayPayloadFor(answers, c.previous, c.labels...)
		output := goodEssayV2For(payload)
		c.mutate(output)
		problems := strings.Join(checkEssayAnswerV2(roundTrip(t, output), payload), " | ")
		switch {
		case c.want == "" && problems != "":
			t.Errorf("%s: %s", name, problems)
		case c.want != "" && !strings.Contains(problems, c.want):
			t.Errorf("%s: want %q, got %q", name, c.want, problems)
		}
	}
}

// The limit is 250 words of the paragraphs and the recommendations; a quote is not counted.
func TestEssayAnswerOfV2HasNotMoreThan250Words(t *testing.T) {
	payload := essayPayloadFor(map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было очень страшно.", "2.4": "Хочу поблагодарить полковника Иванова."}, nil)
	output := goodEssayV2For(payload)
	rest := 0
	for _, key := range []string{"moments", "environment"} {
		rest += countWords(output[key].(map[string]any)["text"].(string))
	}
	for _, item := range output["recommendations"].([]any) {
		rest += countWords(item.(map[string]any)["text"].(string))
	}
	words := func(n int) string { return strings.TrimSpace(strings.Repeat("слово — ", n)) }
	output["formation"].(map[string]any)["text"] = words(essayWordsMost - rest)
	if problems := checkEssayAnswerV2(roundTrip(t, output), payload); len(problems) != 0 {
		t.Fatalf("250 words: %v", problems)
	}
	output["formation"].(map[string]any)["text"] = words(essayWordsMost - rest + 1)
	if problems := strings.Join(checkEssayAnswerV2(roundTrip(t, output), payload), " | "); !strings.Contains(problems, "вместе 251 слов, нужно не больше 250") {
		t.Fatalf("251 words: %s", problems)
	}
}

func TestCheckOfV2NeverRepeatsTheAnswer(t *testing.T) {
	payload := essayPayloadFor(map[string]string{"1.1": "Я пошёл в ВУЦ ради службы.", "1.2": "Было страшно.", "2.4": "Спасибо."}, nil)
	output := goodEssayV2For(payload)
	output["formation"].(map[string]any)["text"] = "Секретный текст 1.2 [преподаватель 4] и Я-концепция."
	output["moments"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": "Секретная выдуманная цитата модели"}}
	problems := strings.Join(checkEssayAnswerV2(roundTrip(t, output), payload), "|")
	if strings.Contains(problems, "Секрет") || !strings.Contains(problems, "номер вопроса") || !strings.Contains(problems, "метка") || !strings.Contains(problems, "термин") {
		t.Fatalf("problems: %s", problems)
	}
}

// ---- the summary of v2 ----------------------------------------------------

func cohortPayloadV2For() *object {
	payload := newObject()
	reviews := []any{}
	for n := 1; n <= 3; n++ {
		item := func(ref string) *object {
			o := newObject()
			o.set("номер", fmt.Sprintf("%d.%s", n, ref))
			o.set("текст", "Текст.")
			return o
		}
		review := newObject()
		review.set("автор", fmt.Sprint(n))
		review.set("прежнее_эссе", n == 1)
		review.set("абзацы", []any{item("formation"), item("moments"), item("environment")})
		review.set("рекомендации", []any{item("rec.1"), item("rec.2")})
		reviews = append(reviews, review)
	}
	payload.set("разборы", reviews)
	return payload
}

// cleanedV2 runs an answer of v2 through the cleaning and the check, as the layer does.
func cleanedV2(t *testing.T, payload *object, output map[string]any) (map[string]any, int, []string) {
	t.Helper()
	back := roundTrip(t, output)
	back, changes := cleanCohortAnswerV2(nil, model.Database{}, Source{}, back, payload)
	return back, changes, checkCohortAnswerV2(back, payload)
}

func TestCheckCohortAnswerV2(t *testing.T) {
	refs := func(key string, list ...any) func(map[string]any) {
		return func(o map[string]any) { o[key].(map[string]any)["refs"] = list }
	}
	text := func(key, value string) func(map[string]any) {
		return func(o map[string]any) { o[key].(map[string]any)["text"] = value }
	}
	cases := map[string]struct {
		mutate func(map[string]any)
		want   string
	}{
		"good answer":                     {func(map[string]any) {}, ""},
		"a paragraph of one author":       {refs("moments", "1.moments", "1.moments"), "moments: ссылки нужны не меньше чем на 2 разных авторов"},
		"a paragraph on another kind":     {refs("formation", "1.formation", "2.moments", "3.rec.1"), "formation: ссылки нужны не меньше чем на 2 разных авторов"},
		"a paragraph on a missing author": {refs("environment", "1.environment", "9.environment"), "environment: ссылки нужны"},
		"no paragraph":                    {func(o map[string]any) { delete(o, "formation") }, "formation: нет абзаца"},
		"an empty paragraph":              {text("environment", ""), "environment: пустой текст"},
		"a count in words":                {text("formation", "Большинство авторов пришли ради отсрочки."), "formation: слово о количестве"},
		"a digit":                         {text("moments", "Двое из 3 авторов пишут."), "moments: слово о количестве, цифра или метка"},
		"a label":                         {text("environment", "Благодарят [преподаватель 3]."), "environment: слово о количестве, цифра или метка"},
		"a term":                          {text("formation", "Я-концепция выпуска складывается."), "formation: термин «я-концепция»"},
		"a level":                         {text("moments", "У авторов высокий уровень мотивации."), "moments: недопустимое слово"},
		"too many recommendations": {func(o map[string]any) {
			item := func(text string) any { return map[string]any{"text": text, "refs": []any{"1.rec.1", "2.rec.1"}} }
			o["recommendations"] = []any{item("А."), item("Б."), item("В."), item("Г."), item("Д.")}
		}, "recommendations: рекомендаций должно быть от 2 до 4"},
		"no recommendation stands on two authors": {func(o map[string]any) {
			o["recommendations"] = []any{map[string]any{"text": "Одного.", "refs": []any{"1.rec.1", "1.moments"}}, map[string]any{"text": "Тоже одного.", "refs": []any{"3.rec.2"}}}
		}, "recommendations: рекомендаций должно быть от 2 до 4"},
		"a repeated recommendation": {func(o map[string]any) {
			o["recommendations"].([]any)[1].(map[string]any)["text"] = "поддерживать новичков в первые недели."
		}, "recommendations, пункт 2: повторяет пункт 1"},
		"too many words": {text("formation", strings.TrimSpace(strings.Repeat("слово ", essayWordsMost))), "нужно не больше 250"},
	}
	for name, c := range cases {
		output := goodCohortV2("")
		c.mutate(output)
		_, _, problems := cleanedV2(t, cohortPayloadV2For(), output)
		joined := strings.Join(problems, " | ")
		switch {
		case c.want == "" && joined != "":
			t.Errorf("%s: %s", name, joined)
		case c.want != "" && !strings.Contains(joined, c.want):
			t.Errorf("%s: want %q, got %q", name, c.want, joined)
		}
	}
}

func TestCleaningOfV2DropsWrongReferencesAndRecommendationsOfOneAuthor(t *testing.T) {
	output := goodCohortV2("")
	output["moments"].(map[string]any)["refs"] = []any{"1.moments", "2.moments", "3.formation", "2.moments", "4.moments", 7}
	output["recommendations"] = []any{
		map[string]any{"text": "Общая.", "refs": []any{"1.rec.1", "2.environment", "3.path"}},
		map[string]any{"text": "Одного автора.", "refs": []any{"3.rec.1", "3.rec.2"}},
	}
	got, changes, problems := cleanedV2(t, cohortPayloadV2For(), output)
	advice := got["recommendations"].([]any)
	if len(problems) != 0 || fmt.Sprint(got["moments"].(map[string]any)["refs"]) != "[1.moments 2.moments]" || len(advice) != 1 || fmt.Sprint(advice[0].(map[string]any)["refs"]) != "[1.rec.1 2.environment]" || changes != 6 {
		t.Fatalf("moments %v, recommendations %v, %d changes, problems %v", got["moments"], advice, changes, problems)
	}
}

// ---- from v1 to v2 ----------------------------------------------------------

// goodEssayV1 answers the way the instruction essay/v1 asks.
func goodEssayV1(user string) map[string]any {
	answer := goodEssay(user)
	input := readEssayInput(&testing.T{}, user)
	quote := []any{quoteOf(input.Essay[0])}
	moment := func() map[string]any {
		return map[string]any{"text": "По ответам видно, что автор об этом пишет.", "quotes": quote}
	}
	return map[string]any{
		"summary":    "Автор пришёл в ВУЦ с ожиданиями.",
		"moments":    map[string]any{"beliefs": moment(), "emotions": moment(), "values": moment(), "abilities": moment()},
		"positive":   []any{map[string]any{"text": "Автор ценит помощь товарищей.", "about": "о себе", "basis": "сказано автором", "quotes": quote}},
		"negative":   []any{},
		"attention":  []any{map[string]any{"text": "Автору нужна поддержка.", "about": "о ВУЦ", "basis": "вывод по тексту", "quotes": quote}},
		"teachers":   answer["teachers"],
		"change":     []any{},
		"conclusion": map[string]any{"text": "По ответам видно, что становление идёт.", "quotes": quote},
	}
}

// A center that had the analyses and the summary of v1 when v2 came: the
// analysis of v1 is shown until the new one is made, but it does not go into
// the summary of v2; the summary of v1 is shown as it was, with its quotes.
func TestAnalysesOfV1AreShownWhileV2IsMadeAndStayOutOfTheSummary(t *testing.T) {
	f := newEssayFixture(t)
	t.Cleanup(f.service.Close)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	first := []promptbank.Prompt{}
	for _, task := range []string{essayForm, essayCohortTask} {
		prompt, _ := promptbank.Find(prompts, task, "v1")
		first = append(first, prompt)
	}
	f.service.RegisterPrompts(first...)
	f.service.portraits.pause = 0
	if _, err = f.service.SaveInstructor(f.owner, "", InstructorInput{Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович", Rank: "полковник"}); err != nil {
		t.Fatal(err)
	}
	f.service.UseModel(&fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		var answer map[string]any
		if strings.Contains(c.System, "Тебе передаются только эти") {
			answer = goodCohort(c.User)
		} else {
			answer = goodEssayV1(c.User)
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}})
	f.three()
	f.idle()
	if _, err = f.service.ComposeEssaySummary(f.owner, f.assignment); err != nil {
		t.Fatal(err)
	}
	f.idle()
	if done := f.summary(f.owner); done["summary"].(map[string]any)["version"] != "v1" {
		t.Fatalf("summary of v1: %+v", done)
	}

	// v2 comes; the model is not connected yet.
	f.service.UseModel(nil)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(prompts...)
	value, _ := f.service.EssayOverview(f.owner)
	view := roundTrip(t, value.(map[string]any))["assignments"].([]any)[0].(map[string]any)
	if counts := view["counts"].(map[string]any); counts["submitted"] != 3.0 || counts["analysed"] != 0.0 || counts["withoutAnalysis"] != 3.0 {
		t.Fatalf("counts: %+v", counts)
	}
	var student map[string]any
	for _, raw := range view["students"].([]any) {
		if row := raw.(map[string]any); row["status"] == "submitted" {
			student = row
		}
	}
	if student == nil || student["analysis"] != "earlier" {
		t.Fatalf("student: %+v", student)
	}
	shown, err := f.portrait(f.owner, student["submissionId"].(string))
	if err != nil || shown.Status != portraitReady || shown.Portrait.Version != "v1" {
		t.Fatalf("the analysis of v1 is not shown: %+v %v", shown, err)
	}
	old := f.summary(f.owner)
	summary := old["summary"].(map[string]any)
	sources := summary["common"].(map[string]any)["positive"].([]any)[0].(map[string]any)["sources"].([]any)
	if old["canCompose"] != false || summary["version"] != "v1" || summary["older"] != true || len(sources[0].(map[string]any)["quotes"].([]any)) != 1 {
		t.Fatalf("the summary of v1: %+v", old)
	}
	if listed := old["teachers"].(map[string]any)["listed"].([]any); len(listed) != 1 {
		t.Fatalf("the block of instructors stands on the analyses of v1 too: %+v", old["teachers"])
	}

	// With the model the analyses of v2 are made by themselves and the summary of v2 on the button.
	m := bothTasksModel(nil)
	f.service.UseModel(m)
	f.idle()
	if m.count() != 3 {
		t.Fatalf("%d calls: the analyses of v2 must be made by themselves", m.count())
	}
	ready := f.summary(f.owner)
	if ready["canCompose"] != true || ready["counts"].(map[string]any)["analysed"] != 3.0 || ready["summary"].(map[string]any)["version"] != "v1" {
		t.Fatalf("after the analyses of v2: %+v", ready)
	}
	if _, err = f.service.ComposeEssaySummary(f.owner, f.assignment); err != nil {
		t.Fatal(err)
	}
	f.idle()
	fresh := f.summary(f.owner)["summary"].(map[string]any)
	if m.count() != 4 || fresh["version"] != "v2" || fresh["older"] != false || fresh["essays"] != 3.0 {
		t.Fatalf("%d calls, summary: %+v", m.count(), fresh)
	}
}

// A label stands only on quotes that name the instructor: by surname (a typo
// counts), name, patronymic or rank, in any case.
func TestLabelOfAnInstructorNeedsHisWordInTheQuotes(t *testing.T) {
	answers := map[string]string{
		"1.1": "Я пошёл в ВУЦ ради службы и отсрочки.",
		"1.2": "Старшина Громов показал ещё раз и не ругался.",
		"2.4": "Благодарю полковнику Ивнову и Игоря Петровича за помощь.",
	}
	entry := func(label string, quotes ...string) map[string]any {
		list := []any{}
		for _, quote := range quotes {
			question := "2.4"
			if strings.Contains(answers["1.2"], quote) {
				question = "1.2"
			}
			list = append(list, map[string]any{"question": question, "quote": quote})
		}
		return map[string]any{"who": label, "kind": "благодарность", "text": "За помощь.", "quotes": list}
	}
	for name, c := range map[string]struct {
		record map[string]any
		want   bool
	}{
		"by a rank and a surname with a typo": {entry("[преподаватель 1]", "Благодарю полковнику Ивнову"), false},
		"by a surname with a typo only":       {entry("[преподаватель 1]", "полковнику Ивнову и"), false},
		"by a name and a patronymic":          {entry("[преподаватель 1]", "и Игоря Петровича за помощь"), false},
		"someone else":                        {entry("[преподаватель 1]", "Старшина Громов показал ещё раз"), true},
		"one quote of two is enough":          {entry("[преподаватель 1]", "Старшина Громов показал ещё раз", "Благодарю полковнику Ивнову"), false},
		"as the author wrote it":              {entry("старшина Громов", "Старшина Громов показал ещё раз"), false},
	} {
		payload := essayPayloadFor(answers, nil, "[преподаватель 1]")
		output := map[string]any{"teachers": []any{c.record}}
		problems := strings.Join(labelWordProblems(roundTrip(t, output), payload), " | ")
		if got := strings.Contains(problems, "teachers, запись 1: в цитатах нет ни фамилии"); got != c.want {
			t.Errorf("%s: %q", name, problems)
		}
	}
	if surnameWithTypo("Громов", []string{"Гром пришёл"}) || !surnameWithTypo("Белоцерковский", []string{"капитан Белоцерковкий"}) {
		t.Fatal("typo of a surname")
	}
	if got := instructorOf("Кузнецов Илья Павлович, старший лейтенант"); got.Surname != "Кузнецов" || got.Patronymic != "Павлович" || got.Rank != "старший лейтенант" {
		t.Fatalf("instructor: %+v", got)
	}
}

// An author without an earlier essay: the text says nothing about one.
func TestNoWordsOfAnEarlierEssayWhenThereIsNone(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было очень страшно.", "2.4": "Хочу поблагодарить полковника Иванова."}
	earlier := map[string]string{"1.1": "Раньше я хотел только отсрочку от армии.", "1.2": "Раньше было страшно.", "2.4": "Никого."}
	for _, text := range []string{"Прежнего эссе нет, поэтому динамика видна по контрасту.", "Между прежним и новым эссе он изменился.", "В отличие от предыдущего эссе автор спокоен."} {
		payload := essayPayloadFor(answers, nil)
		output := goodEssayV2For(payload)
		output["formation"].(map[string]any)["text"] = text
		if problems := strings.Join(checkEssayAnswerV2(roundTrip(t, output), payload), " | "); !strings.Contains(problems, "formation: прежнего эссе у автора нет") {
			t.Errorf("%q: %s", text, problems)
		}
		payload = essayPayloadFor(answers, earlier)
		output = goodEssayV2For(payload)
		output["formation"].(map[string]any)["text"] = text
		if problems := checkEssayAnswerV2(roundTrip(t, output), payload); len(problems) != 0 {
			t.Errorf("with an earlier essay %q: %v", text, problems)
		}
	}
}

func TestRecommendationOfV2DoesNotStandOnTheFamily(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было очень страшно.", "2.4": "Хочу поблагодарить полковника Иванова."}
	for text, family := range map[string]bool{
		"Предложить короткие отчёты отцу о целях.":                    true,
		"Делиться с родными итогами недели.":                          true,
		"Подготовить сообщение о службе предков.":                     true,
		"Использовать семейную историю как ресурс.":                   true,
		"Рассказать о дедушке на занятии.":                            true,
		"Позвать брата на присягу.":                                   true,
		"Готовить материалы к семинару в этом семестре.":              false,
		"Развивать боевое братство во взводе и любовь к Родине.":      false,
		"Поручать роль старшего в отделении, опираясь на его память.": false,
	} {
		payload := essayPayloadFor(answers, nil)
		output := goodEssayV2For(payload)
		output["recommendations"].([]any)[1].(map[string]any)["text"] = text
		got := strings.Contains(strings.Join(checkEssayAnswerV2(roundTrip(t, output), payload), " | "), "recommendations, пункт 2: рекомендация опирается на семью")
		if got != family {
			t.Errorf("%q: %v", text, got)
		}
	}
}

func TestSummaryOfV2NamesNoInstructorAndGoesOverNoAuthor(t *testing.T) {
	for text, want := range map[string]string{
		"Авторы связывают изменения с подполковником Ковальчуком.":               "звание в тексте",
		"Помогает терпеливый показ старшины.":                                    "звание в тексте",
		"Один связывает цель с пользой, другой — с профессией.":                  "авторы перечислены по одному",
		"Первый делает ставку на теорию, второй — на память.":                    "авторы перечислены по одному",
		"Одни быстрее принимают порядок, другим нужна рамка.":                    "",
		"В первый год авторы тревожатся, во второй год спокойны.":                "",
		"Один путь ведёт к службе, другой путь — к профессии.":                   "",
		"Старшинство и порядок важны; преподаватели-офицеры объясняют спокойно.": "",
	} {
		output := goodCohortV2("")
		output["moments"].(map[string]any)["text"] = text
		_, _, problems := cleanedV2(t, cohortPayloadV2For(), output)
		joined := strings.Join(problems, " | ")
		switch {
		case want == "" && joined != "":
			t.Errorf("%q: %s", text, joined)
		case want != "" && !strings.Contains(joined, "moments: "+want):
			t.Errorf("%q: want %q, got %q", text, want, joined)
		}
	}
}

// «: Авторы …» — the marks before a text are taken off by the program.
func TestCleaningOfV2TakesTheMarksOffTheStartOfTexts(t *testing.T) {
	output := goodCohortV2("")
	output["formation"].(map[string]any)["text"] = ": Авторы приходят."
	output["recommendations"].([]any)[0].(map[string]any)["text"] = " — Ввести планы."
	got, changes, problems := cleanedV2(t, cohortPayloadV2For(), output)
	if len(problems) != 0 || changes != 2 || got["formation"].(map[string]any)["text"] != "Авторы приходят." || got["recommendations"].([]any)[0].(map[string]any)["text"] != "Ввести планы." {
		t.Fatalf("%d changes, %v, problems %v", changes, got["formation"], problems)
	}
	essay := map[string]any{"formation": map[string]any{"text": ";Автор пришёл."}, "recommendations": []any{map[string]any{"text": "«Сначала» понять задачу."}}}
	if n := trimTexts(essay); n != 1 || essay["formation"].(map[string]any)["text"] != "Автор пришёл." || essay["recommendations"].([]any)[0].(map[string]any)["text"] != "«Сначала» понять задачу." {
		t.Fatalf("essay: %d %v", n, essay)
	}
}
