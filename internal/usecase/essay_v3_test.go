package usecase

import (
	"fmt"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// goodEssayV3For is a good answer of v3 to a small input made by essayPayloadFor:
// the author names colonel Ivanov in answer 2.4.
func goodEssayV3For(payload *object) map[string]any {
	output := goodEssayV2For(payload)
	delete(output, "recommendations")
	quote := func() []any {
		return []any{map[string]any{"question": "1.1", "quote": "Я пошёл «в ВУЦ» ради «службы»"}}
	}
	teacher := []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}
	output["environment"] = map[string]any{"text": "ВУЦ воспринимает как среду с понятным порядком.", "quotes": quote()}
	output["developed"] = []any{map[string]any{"text": "Ответственное отношение к службе.", "quotes": quote()}}
	output["strengthen"] = []any{
		map[string]any{"text": "Уверенность в своих силах — давать самостоятельные задачи.", "quotes": quote()},
		map[string]any{"text": "Выдержку — разбирать трудные случаи после занятий.", "quotes": quote()},
	}
	output["teachersConclusion"] = map[string]any{"text": "Преподавателю удалось поддержать автора в трудный момент; стоит и дальше давать ему самостоятельные задачи.", "quotes": teacher}
	output["teachers"] = []any{map[string]any{"position": "преподаватель", "name": "Иванов", "kind": "благодарность", "text": "За помощь.", "quotes": teacher}}
	return output
}

func TestCheckEssayAnswerV3(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было очень страшно.", "2.4": "Хочу поблагодарить полковника Иванова за помощь."}
	earlier := map[string]string{"1.1": "Раньше я хотел только отсрочку от армии.", "1.2": "Раньше было совсем страшно.", "2.4": "Раньше никого не помню."}
	text := func(key, value string) func(map[string]any) {
		return func(o map[string]any) { o[key].(map[string]any)["text"] = value }
	}
	point := func(key string, n int, value string) func(map[string]any) {
		return func(o map[string]any) { o[key].([]any)[n].(map[string]any)["text"] = value }
	}
	item := func(value string) any {
		return map[string]any{"text": value, "quotes": []any{map[string]any{"question": "1.1", "quote": "Я пошёл «в ВУЦ»"}}}
	}
	cases := map[string]struct {
		previous map[string]string
		mutate   func(map[string]any)
		want     string
	}{
		"good answer":                    {nil, func(map[string]any) {}, ""},
		"good answer with earlier essay": {earlier, func(o map[string]any) { o["formation"].(map[string]any)["before"] = []any{map[string]any{"question": "1.1", "quote": "Раньше я хотел только отсрочку"}} }, ""},
		"nothing the center brought up":  {nil, func(o map[string]any) { o["developed"] = []any{} }, ""},
		"four points of developed": {nil, func(o map[string]any) {
			o["developed"] = []any{item("А."), item("Б."), item("В."), item("Г.")}
		}, "developed: пунктов должно быть от 0 до 3"},
		"one point to strengthen": {nil, func(o map[string]any) { o["strengthen"] = []any{item("Уверенность.")} }, "strengthen: пунктов должно быть от 2 до 3"},
		"a point without quotes": {nil, func(o map[string]any) {
			o["developed"].([]any)[0].(map[string]any)["quotes"] = []any{}
		}, "developed, пункт 1: цитат должно быть от 1 до 2"},
		"a repeated point": {nil, point("strengthen", 1, "уверенность в своих силах — давать самостоятельные задачи."), "strengthen, пункт 2: повторяет пункт 1"},
		"no conclusion about the teachers": {nil, func(o map[string]any) { delete(o, "teachersConclusion") }, "teachersConclusion: нет вывода"},
		"a conclusion without teachers": {nil, func(o map[string]any) { o["teachers"] = []any{} }, "teachersConclusion: в teachers никого нет"},
		"an empty conclusion with teachers": {nil, func(o map[string]any) {
			o["teachersConclusion"] = map[string]any{"text": "", "quotes": []any{}}
		}, "teachersConclusion: пустой текст"},
		"no teachers, no conclusion": {nil, func(o map[string]any) {
			o["teachers"] = []any{}
			o["teachersConclusion"] = map[string]any{"text": "", "quotes": []any{}}
		}, ""},
		"a surname in a paragraph": {nil, text("environment", "Благодарит Иванова за помощь."), "environment: фамилия или имя преподавателя в тексте"},
		"a surname in the conclusion": {nil, text("teachersConclusion", "Иванову удалось поддержать автора."), "teachersConclusion: фамилия или имя преподавателя"},
		"a rank in the conclusion":    {nil, text("teachersConclusion", "Полковнику удалось поддержать автора."), "teachersConclusion: звание в тексте"},
		"the family in strengthen":    {nil, point("strengthen", 0, "Уверенность — опираться на пример отца."), "strengthen, пункт 1: пункт опирается на семью"},
		"a copied example": {nil, point("strengthen", 0, "Уверенность в собственных решениях — давать задачи, где решение нужно принять самому."), "strengthen, пункт 1: повторяет пример из инструкции"},
		"a short phrase of the supervisor": {nil, text("moments", "Опирается на приобретённый опыт, умеет воспринимать другую точку зрения."), ""},
		"an earlier essay that is none":     {nil, point("developed", 0, "По сравнению с прежним эссе стал собраннее."), "developed, пункт 1: прежнего эссе у автора нет"},
		"a question number":                 {nil, point("strengthen", 1, "Выдержку — по ответу 3.2."), "strengthen, пункт 2: номер вопроса"},
		"a term":                            {nil, text("teachersConclusion", "Преподаватель укрепил Я-концепцию автора."), "термин «я-концепцию»"},
		"too many words":                    {nil, text("moments", strings.TrimSpace(strings.Repeat("слово ", essayWordsMost))), "нужно не больше 250"},
	}
	for name, c := range cases {
		payload := essayPayloadFor(answers, c.previous)
		output := goodEssayV3For(payload)
		c.mutate(output)
		back := roundTrip(t, output)
		back, _ = cleanEssayAnswerV3(nil, model.Database{}, Source{}, back, payload)
		joined := strings.Join(checkEssayAnswerV3(back, payload), " | ")
		switch {
		case c.want == "" && joined != "":
			t.Errorf("%s: %s", name, joined)
		case c.want != "" && !strings.Contains(joined, c.want):
			t.Errorf("%s: want %q, got %q", name, c.want, joined)
		}
	}
}

// A problem names the place, never the surname: it goes to the journal.
func TestCheckOfV3NeverRepeatsTheSurname(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было очень страшно.", "2.4": "Хочу поблагодарить полковника Иванова за помощь."}
	payload := essayPayloadFor(answers, nil)
	output := goodEssayV3For(payload)
	output["moments"].(map[string]any)["text"] = "Автор благодарен Иванову."
	if problems := strings.Join(checkEssayAnswerV3(roundTrip(t, output), payload), "|"); problems == "" || strings.Contains(problems, "Иванов") {
		t.Fatalf("problems: %s", problems)
	}
}

func TestCleaningOfV3TakesTheMarksOffTheStartOfEveryText(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было очень страшно.", "2.4": "Хочу поблагодарить полковника Иванова за помощь."}
	output := roundTrip(t, goodEssayV3For(essayPayloadFor(answers, nil)))
	output["developed"].([]any)[0].(map[string]any)["text"] = ": Ответственность."
	output["teachersConclusion"].(map[string]any)["text"] = "— Преподавателю удалось."
	got, changes := cleanEssayAnswerV3(nil, model.Database{}, Source{}, output, nil)
	if changes != 2 || got["developed"].([]any)[0].(map[string]any)["text"] != "Ответственность." || got["teachersConclusion"].(map[string]any)["text"] != "Преподавателю удалось." {
		t.Fatalf("%d changes: %+v", changes, got)
	}
}

// The examples the copy check looks for are those of the embedded instructions.
func TestExamplesOfV3AreThoseOfTheInstructions(t *testing.T) {
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	for task, examples := range map[string][]string{essayForm: essayExamplesV3, essayCohortTask: cohortExamplesV3} {
		prompt, ok := promptbank.Find(prompts, task, "v3")
		if !ok {
			t.Fatalf("%s v3 is not embedded", task)
		}
		for _, example := range examples {
			if !strings.Contains(prompt.Instruction, "«"+example+"»") {
				t.Errorf("%s: the example %q is not in the instruction", task, example)
			}
		}
	}
}

// ---- the summary of a cohort, v3 ------------------------------------------

func cohortPayloadV3For(teachers int) *object {
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
		review.set("воспитал_вуц", []any{item("dev.1")})
		review.set("усилить", []any{item("str.1"), item("str.2")})
		if n <= teachers {
			review.set("о_преподавателях", item("teachers"))
		}
		reviews = append(reviews, review)
	}
	payload.set("разборы", reviews)
	return payload
}

// cleanedV3 runs an answer of v3 through the cleaning and the check, as the layer does.
func cleanedV3(t *testing.T, payload *object, output map[string]any) (map[string]any, int, []string) {
	t.Helper()
	back := roundTrip(t, output)
	back, changes := cleanCohortAnswerV3(nil, model.Database{}, Source{}, back, payload)
	return back, changes, checkCohortAnswerV3(back, payload)
}

func TestCheckCohortAnswerV3(t *testing.T) {
	text := func(key, value string) func(map[string]any) {
		return func(o map[string]any) { o[key].(map[string]any)["text"] = value }
	}
	payloadOf := func(teachers int) string { return modelText(canonicalJSON(cohortPayloadV3For(teachers))) }
	cases := map[string]struct {
		teachers int
		mutate   func(map[string]any)
		want     string
	}{
		"good answer":                  {3, func(map[string]any) {}, ""},
		"good answer, one teacher":     {1, func(map[string]any) {}, ""},
		"no developed":                 {3, func(o map[string]any) { o["developed"] = []any{} }, ""},
		"«Общее:» opens a paragraph":   {3, text("moments", "Общее: ценность порядка и честности."), "moments: предложение начинается с «Общее:»"},
		"«Расхождения —» in the midst": {3, text("moments", "Авторы ценят порядок. Расхождения — в акцентах."), "moments: предложение начинается с «Общее:» или «Расхождения —»"},
		"«Расхождения в» opens":        {3, text("formation", "Расхождения в мотивах невелики."), "formation: предложение начинается"},
		"other words with общее":       {3, text("formation", "Авторов объединяет общее отношение к службе."), ""},
		"a conclusion of one author": {1, func(o map[string]any) {
			o["teachersConclusion"] = map[string]any{"text": "Преподавателям удалось многое.", "refs": []any{"1.teachers"}}
		}, "teachersConclusion: вывод о преподавателях есть меньше чем у двух авторов"},
		"no conclusion with three authors": {3, func(o map[string]any) {
			o["teachersConclusion"] = map[string]any{"text": "", "refs": []any{}}
		}, "teachersConclusion: пустой текст"},
		"a conclusion on paragraphs": {3, func(o map[string]any) {
			o["teachersConclusion"].(map[string]any)["refs"] = []any{"1.environment", "2.environment"}
		}, "teachersConclusion: ссылки нужны не меньше чем на 2 разных авторов"},
		"a rank in the conclusion": {3, text("teachersConclusion", "Подполковнику удалось многое; стоит продолжать."), "teachersConclusion: звание в тексте"},
		"nothing to strengthen of two authors": {3, func(o map[string]any) {
			o["strengthen"] = []any{map[string]any{"text": "Одного.", "refs": []any{"1.str.1", "1.moments"}}}
		}, "strengthen: пунктов должно быть от 1 до 3"},
		"a copied example": {3, func(o map[string]any) {
			o["strengthen"].([]any)[0].(map[string]any)["text"] = "Уверенность в своих силах — давать каждому опыт самостоятельных решений с разбором после."
		}, "strengthen, пункт 1: повторяет пример из инструкции"},
		"a count in words": {3, text("environment", "Большинство авторов ценят порядок."), "environment: слово о количестве"},
		"too many words":   {3, text("formation", strings.TrimSpace(strings.Repeat("слово ", essayWordsMost))), "нужно не больше 250"},
	}
	for name, c := range cases {
		output := goodCohortV3(payloadOf(c.teachers))
		c.mutate(output)
		_, _, problems := cleanedV3(t, cohortPayloadV3For(c.teachers), output)
		joined := strings.Join(problems, " | ")
		switch {
		case c.want == "" && joined != "":
			t.Errorf("%s: %s", name, joined)
		case c.want != "" && !strings.Contains(joined, c.want):
			t.Errorf("%s: want %q, got %q", name, c.want, joined)
		}
	}
}

func TestCleaningOfV3DropsWrongReferencesAndPointsOfOneAuthor(t *testing.T) {
	output := goodCohortV3(modelText(canonicalJSON(cohortPayloadV3For(3))))
	output["developed"] = []any{
		map[string]any{"text": "Общее.", "refs": []any{"1.dev.1", "2.formation", "3.str.1", "3.rec.1"}},
		map[string]any{"text": "Одного автора.", "refs": []any{"3.dev.1", "3.moments"}},
	}
	output["teachersConclusion"].(map[string]any)["refs"] = []any{"1.teachers", "2.teachers", "3.environment", "2.teachers"}
	got, changes, problems := cleanedV3(t, cohortPayloadV3For(3), output)
	developed := got["developed"].([]any)
	if len(problems) != 0 || len(developed) != 1 || fmt.Sprint(developed[0].(map[string]any)["refs"]) != "[1.dev.1 2.formation]" ||
		fmt.Sprint(got["teachersConclusion"].(map[string]any)["refs"]) != "[1.teachers 2.teachers]" || changes != 5 {
		t.Fatalf("developed %v, conclusion %v, %d changes, problems %v", developed, got["teachersConclusion"], changes, problems)
	}
}

func TestParseRefOfV3(t *testing.T) {
	for ref, want := range map[string]string{"3.dev.1": "dev", "2.str.2": "str", "1.teachers": "teachers"} {
		if _, kind, ok := parseRef(ref); !ok || kind != want {
			t.Errorf("%s: %q %v", ref, kind, ok)
		}
	}
	for _, ref := range []string{"3.dev", "3.str.0", "3.teachers.1"} {
		if _, _, ok := parseRef(ref); ok {
			t.Errorf("%s is read", ref)
		}
	}
}

// The summary of v3 is made of the analyses of v3 on the button: the program
// counts the authors of every paragraph, point and of the conclusion about the
// teachers and shows the points «усилить» no point of the summary stands on.
func TestSummaryOfV3IsComposedOnTheButtonOfTheAnalysesOfV3(t *testing.T) {
	f := newEssayPortraitFixture(t)
	m := bothTasksModel(nil)
	f.service.UseModel(m)
	f.three()
	f.idle()
	if m.count() != 3 {
		t.Fatalf("%d calls for three essays", m.count())
	}
	before := f.summary(f.owner)
	if before["canCompose"] != true || before["counts"].(map[string]any)["analysed"] != 3.0 {
		t.Fatalf("before the button: %+v", before)
	}
	if _, err := f.service.ComposeEssaySummary(f.owner, f.assignment); err != nil {
		t.Fatal(err)
	}
	f.idle()
	call := m.calls[3]
	if !strings.Contains(call.System, "teachersConclusion") || !strings.Contains(call.User, `"воспитал_вуц"`) || !strings.Contains(call.User, `"1.str.2"`) || strings.Contains(call.User, `"quote`) || strings.Contains(call.User, "рекомендации") {
		t.Fatalf("input of the summary: %.300s", call.User)
	}
	done := f.summary(f.owner)
	summary := done["summary"].(map[string]any)
	if done["status"] != portraitReady || summary["version"] != "v3" || summary["older"] != false {
		t.Fatalf("summary: %+v", done)
	}
	common := summary["common"].(map[string]any)
	strengthen := common["strengthen"].([]any)
	if len(common["developed"].([]any)) != 1 || len(strengthen) != 2 || strengthen[1].(map[string]any)["authors"] != 2.0 || common["formation"].(map[string]any)["of"] != 3.0 {
		t.Fatalf("common: %+v", common)
	}
	// The points «усилить» of the analyses no point of the summary stands on stay on the screen.
	left := common["notIncluded"].([]any)
	if len(left) != 3 || left[0].(map[string]any)["ref"] != "1.str.2" {
		t.Fatalf("not included: %+v", left)
	}
	if _, ok := common["teachersConclusion"].(map[string]any); !ok {
		t.Fatalf("no group of the conclusion about the teachers: %+v", common)
	}
	// The block of instructors stands on the analyses of v3.
	if positions := done["teachers"].(map[string]any)["positions"].([]any); len(positions) != 1 {
		t.Fatalf("positions: %+v", positions)
	}
}

// A quote longer than 30 words that is in its answer word for word is cut to its
// first 30 words by the cleaning (decided 5 November 2026 after the run); one
// that is not in the answer is left to the check.
func TestCleaningOfV3CutsALongQuoteThatIsWordForWord(t *testing.T) {
	long := "Тогда мне казалось, что ВУЦ — способ не потерять год и узнать, что такое служба. Сейчас понимаю, что главной причиной было желание проверить себя: смогу ли я выдерживать порядок и отвечать за других."
	answers := map[string]string{"1.1": long, "1.2": "Первые дни было очень страшно.", "2.4": "Хочу поблагодарить полковника Иванова за помощь."}
	payload := essayPayloadFor(answers, nil)
	output := goodEssayV3For(payload)
	output["formation"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": long}}
	output["moments"].(map[string]any)["quotes"] = []any{
		map[string]any{"question": "1.2", "quote": "Первые дни было очень страшно"},
		map[string]any{"question": "1.1", "quote": "Тогда мне казалось, что армия — это " + strings.Repeat("слово ", 30)},
	}
	back := roundTrip(t, output)
	back, changes := cleanEssayAnswerV3(nil, model.Database{}, Source{}, back, payload)
	quote := back["formation"].(map[string]any)["quotes"].([]any)[0].(map[string]any)["quote"].(string)
	if changes != 1 || len(strings.Fields(quote)) != essayQuoteWords || !strings.HasPrefix(long, quote) {
		t.Fatalf("%d changes, quote %q", changes, quote)
	}
	problems := strings.Join(checkEssayAnswerV3(back, payload), " | ")
	if strings.Contains(problems, "formation") || !strings.Contains(problems, "moments: в цитате к вопросу 1.1") {
		t.Fatalf("problems: %s", problems)
	}
}
