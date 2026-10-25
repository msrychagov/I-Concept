package usecase

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

type essayRow struct {
	Number string `json:"номер"`
	Block  string `json:"блок"`
	Asked  string `json:"вопрос"`
	Answer string `json:"ответ"`
}

type essayInput struct {
	Essay      []essayRow `json:"эссе"`
	Previous   []essayRow `json:"прежнее_эссе"`
	Instructor []struct {
		Label string `json:"метка"`
		Who   string `json:"кто"`
	} `json:"преподаватели"`
}

func readEssayInput(t *testing.T, user string) essayInput {
	t.Helper()
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input essayInput
	if err := json.Unmarshal([]byte(user), &input); err != nil {
		t.Fatalf("%v: %.80s", err, user)
	}
	return input
}

// quoteOf cuts the first four words of an answer: a quote word for word.
func quoteOf(row essayRow) map[string]any {
	words := strings.Fields(row.Answer)
	return map[string]any{"question": row.Number, "quote": strings.Join(words[:min(4, len(words))], " ")}
}

// goodEssay answers the way the instruction essay/v2 asks, for the input it is given.
func goodEssay(user string) map[string]any {
	input := readEssayInput(&testing.T{}, user)
	quote := []any{quoteOf(input.Essay[0])}
	before := []any{}
	if len(input.Previous) > 0 {
		before = []any{quoteOf(input.Previous[0])}
	}
	// A teacher named in answer 2.4 is thanked, by the surname the author wrote.
	teachers := []any{}
	for _, row := range input.Essay {
		name := ""
		switch {
		case row.Number != "2.4":
		case strings.Contains(row.Answer, "Сидоров"):
			name = "Сидоров"
		case strings.Contains(row.Answer, "Иванов"):
			name = "Иванов"
		}
		if name != "" {
			teachers = append(teachers, map[string]any{"position": "преподаватель", "name": name, "kind": "благодарность", "text": "Благодарит за помощь.", "quotes": []any{quoteOf(row)}})
		}
	}
	return map[string]any{
		"formation":   map[string]any{"text": "Автор пришёл в ВУЦ с ожиданиями и привык к порядкам.", "quotes": quote, "before": before},
		"moments":     map[string]any{"text": "Автор ценит помощь товарищей.", "quotes": []any{quoteOf(input.Essay[0]), quoteOf(input.Essay[1])}},
		"environment": map[string]any{"text": "Порядки ВУЦ автор принимает, когда понимает их смысл.", "quotes": quote},
		"recommendations": []any{
			map[string]any{"text": "Поддержать автора в первые недели.", "quotes": quote},
			map[string]any{"text": "Опираться на его интерес к службе.", "quotes": quote},
		},
		"teachers": teachers,
	}
}

func essayModel(mutate func(map[string]any, essayInput)) *fakeModel {
	return &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		answer := goodEssay(c.User)
		if mutate != nil {
			mutate(answer, readEssayInput(&testing.T{}, c.User))
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
}

func newEssayPortraitFixture(t *testing.T) *essayFixture {
	return newEssayPortraitFixtureAt(t, "")
}

// newEssayPortraitFixtureAt registers the essay tasks of a version, "" for the newest.
func newEssayPortraitFixtureAt(t *testing.T, version string) *essayFixture {
	t.Helper()
	f := newEssayFixture(t)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	if version != "" {
		chosen := []promptbank.Prompt{}
		for _, task := range []string{essayForm, essayCohortTask} {
			prompt, _ := promptbank.Find(prompts, task, version)
			chosen = append(chosen, prompt)
		}
		prompts = chosen
	}
	f.service.RegisterPrompts(prompts...)
	f.service.portraits.pause = 0
	t.Cleanup(f.service.Close)
	return f
}

func (f *essayFixture) idle() { (&portraitFixture{t: f.t, service: f.service}).idle() }

func (f *essayFixture) portrait(teacher, submission string) (PortraitView, error) {
	return f.service.PortraitFor(teacher, essayForm, submission)
}

// backdate makes a submission older, as the clock does not tick between two writes of a test.
func (f *essayFixture) backdate(submission string, ago time.Duration) {
	f.t.Helper()
	if err := f.service.repo.Update(func(db *model.Database) error {
		for i := range db.Submissions {
			if db.Submissions[i].ID == submission {
				db.Submissions[i].CompletedAt = time.Now().UTC().Add(-ago).Format(time.RFC3339)
			}
		}
		return nil
	}); err != nil {
		f.t.Fatal(err)
	}
}

func TestSubmittedEssayGetsAnAnalysisFromTheInputOfItsOwn(t *testing.T) {
	f := newEssayPortraitFixture(t)
	if _, err := f.service.SaveInstructor(f.owner, "", InstructorInput{Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович", Rank: "полковник"}); err != nil {
		t.Fatal(err)
	}
	m := essayModel(nil)
	f.service.UseModel(m)
	login, id := f.submitted("essay-author@example.test", "2401", f.assignment, map[string]string{
		"1.1": "Хотел служить. Родился 12.03.2005 в городе.",
		"2.4": "Благодарю Игоря Петровича за помощь, пишите мне на author@example.test.",
	})
	f.idle()
	view, err := f.portrait(f.owner, id)
	if err != nil || view.Status != portraitReady || m.count() != 1 {
		t.Fatalf("%+v %v after %d calls", view, err, m.count())
	}
	call := m.calls[0]
	if !strings.HasPrefix(call.System, "Ты помогаешь преподавателю военного учебного центра") || !strings.Contains(string(call.Schema), `"formation"`) || call.MaxTokens != 6000 {
		t.Fatalf("instruction or schema: %.80s", call.System)
	}
	// The order of the keys, the cut data, the patronymic kept and nothing of the person.
	if !strings.HasPrefix(call.User, "{\n \"эссе\": [") {
		t.Fatalf("keys: %.60s", call.User)
	}
	input := readEssayInput(t, call.User)
	if len(input.Essay) != 13 || input.Essay[0].Number != "1.1" || input.Essay[0].Block != "Знакомство с ВУЦ и первые впечатления" || input.Essay[0].Asked != "Почему вы решили поступать в ВУЦ?" || len(input.Previous) != 0 {
		t.Fatalf("essay rows: %+v", input.Essay[:1])
	}
	if got := input.Essay[0].Answer; got != "Хотел служить. Родился [скрыто] в городе." {
		t.Fatalf("answer 1.1: %q", got)
	}
	if got := input.Essay[7].Answer; input.Essay[7].Number != "2.4" || got != "Благодарю Игоря Петровича за помощь, пишите мне на [скрыто]." {
		t.Fatalf("answer 2.4: %q", got)
	}
	// The list of the instructors does not go: v2 tells them by positions (decided 25 October 2026).
	if len(input.Instructor) != 0 || strings.Contains(call.User, "преподаватели") {
		t.Fatalf("instructors reached the model: %+v", input.Instructor)
	}
	for _, banned := range []string{login.Account.Name, login.Account.Email, "Выпуск 2027", "2401", "Эссе в конце", "Иванов Игорь Петрович"} {
		if strings.Contains(call.User, banned) {
			t.Fatalf("%q reached the model", banned)
		}
	}
	// The teacher who gave the essay sees it; another teacher does not.
	if _, err = f.portrait(f.other, id); kind(err) != NotFound {
		t.Fatalf("another teacher: %v", err)
	}
	// An essay that is not handed in has no analysis.
	draft := f.student("essay-draft@example.test", "2401")
	if _, err = f.portrait(f.owner, f.start(draft, f.assignment)); kind(err) != NotFound {
		t.Fatalf("a draft: %v", err)
	}
}

func TestEarlierEssayOfTheSameAuthorGoesWithTheNewOne(t *testing.T) {
	f := newEssayPortraitFixture(t)
	m := essayModel(nil)
	f.service.UseModel(m)
	login, first := f.submitted("essay-twice@example.test", "2401", f.assignment, map[string]string{"1.1": "Раньше хотел просто отслужить."})
	f.backdate(first, 48*time.Hour)
	f.idle()
	if m.count() != 1 {
		t.Fatalf("%d calls for the first essay", m.count())
	}
	second := f.assign("Выпускное эссе")
	id := f.start(login, second)
	f.write(login, id, map[string]string{"1.1": "Теперь понимаю, зачем нужна служба."})
	f.idle()
	view, err := f.portrait(f.owner, id)
	if err != nil || view.Status != portraitReady {
		t.Fatalf("%+v %v", view, err)
	}
	input := readEssayInput(t, m.calls[len(m.calls)-1].User)
	if len(input.Previous) != 13 || input.Previous[0].Answer != "Раньше хотел просто отслужить." || input.Essay[0].Answer != "Теперь понимаю, зачем нужна служба." {
		t.Fatalf("earlier essay: %+v", input.Previous[:1])
	}
	var output struct {
		Formation struct {
			Before []struct{ Question string } `json:"before"`
		} `json:"formation"`
	}
	_ = json.Unmarshal(view.Portrait.Output, &output)
	if len(output.Formation.Before) != 1 || output.Formation.Before[0].Question != "1.1" || view.Portrait.Version != "v2" {
		t.Fatalf("quotes of the earlier essay: %+v", output.Formation)
	}
}

func TestEarlierEssayIsTheLastOneHandedInBeforeFromAnyAssignment(t *testing.T) {
	f := newEssayPortraitFixture(t)
	login, oldest := f.submitted("essay-order@example.test", "2401", f.assignment, nil)
	f.backdate(oldest, 72*time.Hour)
	middle := f.start(login, f.assign("Второе"))
	f.write(login, middle, nil)
	f.backdate(middle, 24*time.Hour)
	last := f.start(login, f.assign("Третье"))
	f.write(login, last, nil)
	other, _ := f.submitted("essay-other-author@example.test", "2401", f.assignment, nil)
	_ = other
	db := mustSnapshot(t, f.service.repo)
	got, ok := previousEssay(db, submissionOf(t, f.service, last))
	if !ok || got.ID != middle {
		t.Fatalf("earlier essay of the last: %+v", got.ID)
	}
	if got, ok = previousEssay(db, submissionOf(t, f.service, oldest)); ok {
		t.Fatalf("the oldest has an earlier essay: %s", got.ID)
	}
}

// The list of the instructors is no part of the input of v2: changing it makes
// no analysis again (an instructor who left changes nothing).
func TestChangingTheInstructorListDoesNotMakeTheAnalysesOfV2Again(t *testing.T) {
	f := newEssayPortraitFixture(t)
	m := essayModel(nil)
	f.service.UseModel(m)
	_, id := f.submitted("essay-list@example.test", "2401", f.assignment, nil)
	f.idle()
	if m.count() != 1 {
		t.Fatalf("%d calls", m.count())
	}
	if _, err := f.service.SaveInstructor(f.owner, "", InstructorInput{Surname: "Иванов", Name: "Игорь"}); err != nil {
		t.Fatal(err)
	}
	f.idle()
	if m.count() != 1 {
		t.Fatalf("%d calls after the list changed", m.count())
	}
	view, _ := f.portrait(f.owner, id)
	if view.Status != portraitReady {
		t.Fatalf("%+v", view)
	}
}

func TestPatronymicIsKeptOnlyForTheEssay(t *testing.T) {
	text := "Игорь Петрович и 12.03.2005 и a@b.test"
	if got, n := maskText(text); got != "Игорь [скрыто] и [скрыто] и [скрыто]" || n != 3 {
		t.Fatalf("everything cut: %q %d", got, n)
	}
	if got, n := maskText(text, "отчество"); got != "Игорь Петрович и [скрыто] и [скрыто]" || n != 2 {
		t.Fatalf("patronymic kept: %q %d", got, n)
	}
	o := newObject()
	o.set("а", text)
	if label := personalData(o, "отчество"); label != "полная дата" {
		t.Fatalf("check: %q", label)
	}
	if _, err := maskPersonal(o, "отчество"); err != nil {
		t.Fatalf("a kept patronymic must pass the final check: %v", err)
	}
}

func essayPayloadFor(answers map[string]string, previous map[string]string, labels ...string) *object {
	row := func(number, text string) any {
		o := newObject()
		o.set("номер", number)
		o.set("блок", "Блок")
		o.set("вопрос", "Вопрос?")
		o.set("ответ", text)
		return o
	}
	payload := newObject()
	rows := []any{}
	for _, number := range []string{"1.1", "1.2", "2.4"} {
		rows = append(rows, row(number, answers[number]))
	}
	payload.set("эссе", rows)
	if previous != nil {
		rows = []any{}
		for _, number := range []string{"1.1", "1.2", "2.4"} {
			rows = append(rows, row(number, previous[number]))
		}
		payload.set("прежнее_эссе", rows)
	}
	if len(labels) > 0 {
		list := []any{}
		for _, label := range labels {
			entry := newObject()
			entry.set("метка", label)
			entry.set("кто", "Иванов Игорь Петрович, полковник")
			list = append(list, entry)
		}
		payload.set("преподаватели", list)
	}
	return payload
}

// The answer of v1 is checked as it always was.
func TestCheckEssayAnswer(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было   очень  страшно, ЁЛКИ-палки.", "2.4": "Хочу поблагодарить полковника Иванова за помощь."}
	earlier := map[string]string{"1.1": "Раньше я хотел только отсрочку от армии.", "1.2": "Раньше было совсем страшно.", "2.4": "Раньше никого не помню."}
	mutations := map[string]struct {
		previous map[string]string
		labels   []string
		mutate   func(map[string]any)
		want     string
	}{
		"good answer": {nil, nil, func(map[string]any) {}, ""},
		"quotes of another shape, case, yo and spaces": {nil, nil, func(o map[string]any) {
			o["conclusion"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": `я пошел "в вуц" ради "службы" и отсрочки`}, map[string]any{"question": "1.2", "quote": "Первые  дни было очень страшно, елки-палки"}}
		}, ""},
		"invented quote": {nil, nil, func(o map[string]any) {
			o["positive"].([]any)[0].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": "Я шёл в ВУЦ ради денег"}}
		}, "positive, пункт 1: цитаты нет в ответе 1.1 дословно"},
		"quote from another answer": {nil, nil, func(o map[string]any) {
			o["summary"] = "Текст."
			o["conclusion"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.2", "quote": "Я пошёл в ВУЦ ради службы"}}
		}, "conclusion: цитаты нет в ответе 1.2 дословно"},
		"quote of an unknown question": {nil, nil, func(o map[string]any) {
			o["conclusion"].(map[string]any)["quotes"] = []any{map[string]any{"question": "9.9", "quote": "Я пошёл в ВУЦ"}}
		}, "conclusion: нет ответа с номером 9.9"},
		"quote too short": {nil, nil, func(o map[string]any) {
			o["conclusion"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": "Я пошёл"}}
		}, "conclusion: в цитате к вопросу 1.1 2 слов, нужно от 3 до 30"},
		"quote too long": {nil, nil, func(o map[string]any) {
			o["conclusion"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": strings.TrimSpace(strings.Repeat("слово ", 31))}}
		}, "conclusion: в цитате к вопросу 1.1 31 слов, нужно от 3 до 30"},
		"no quotes": {nil, nil, func(o map[string]any) { o["moments"].(map[string]any)["values"].(map[string]any)["quotes"] = []any{} }, "moments.values: цитат должно быть от 1 до 3"},
		"too many points": {nil, nil, func(o map[string]any) {
			point := o["positive"].([]any)[0]
			o["positive"] = []any{point, point, point, point, point, point}
		}, "positive: пунктов должно быть от 1 до 5"},
		"no positive": {nil, nil, func(o map[string]any) { o["positive"] = []any{} }, "positive: пунктов должно быть от 1 до 5"},
		"wrong about": {nil, nil, func(o map[string]any) { o["positive"].([]any)[0].(map[string]any)["about"] = "об армии" }, "positive, пункт 1: about"},
		"wrong basis": {nil, nil, func(o map[string]any) { o["positive"].([]any)[0].(map[string]any)["basis"] = "догадка" }, "positive, пункт 1: basis"},
		"change without an earlier essay": {nil, nil, func(o map[string]any) {
			o["change"] = []any{map[string]any{"moment": "убеждения", "text": "Т.", "before": map[string]any{}, "after": map[string]any{}}}
		}, "change: прежнего эссе нет, список должен быть пустым"},
		"no change with an earlier essay": {earlier, nil, func(o map[string]any) { o["change"] = []any{} }, "change: нужно 4 записи по порядку, получено 0"},
		"moments of change in another order": {earlier, nil, func(o map[string]any) {
			change := o["change"].([]any)
			change[0], change[1] = change[1], change[0]
		}, "change, запись 1: момент не тот или не по порядку"},
		"before is looked up in the earlier essay": {earlier, nil, func(o map[string]any) {
			o["change"].([]any)[2].(map[string]any)["before"] = map[string]any{"question": "1.1", "quote": "Я пошёл в ВУЦ ради службы"}
		}, "change, запись 3, before: цитаты нет в ответе 1.1 дословно"},
		"after is looked up in the new essay": {earlier, nil, func(o map[string]any) {
			o["change"].([]any)[2].(map[string]any)["after"] = map[string]any{"question": "1.1", "quote": "Раньше я хотел только отсрочку"}
		}, "change, запись 3, after: цитаты нет в ответе 1.1 дословно"},
		"conclusion starts otherwise": {nil, nil, func(o map[string]any) { o["conclusion"].(map[string]any)["text"] = "Становление идёт." }, "conclusion: должен начинаться"},
		"conclusion starts with the other phrase": {nil, nil, func(o map[string]any) {
			o["conclusion"].(map[string]any)["text"] = "По ответам можно предположить, что идёт."
		}, ""},
		"a label of the list": {nil, []string{"[преподаватель 3]"}, func(o map[string]any) {
			o["teachers"] = []any{map[string]any{"who": "[преподаватель 3]", "kind": "благодарность", "text": "За помощь.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}}
		}, ""},
		"a label that is not in the list": {nil, []string{"[преподаватель 3]"}, func(o map[string]any) {
			o["teachers"] = []any{map[string]any{"who": "[преподаватель 7]", "kind": "благодарность", "text": "За помощь.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}}
		}, "teachers, запись 1: метка не из списка"},
		"a label with no list": {nil, nil, func(o map[string]any) {
			o["teachers"] = []any{map[string]any{"who": "[преподаватель 3]", "kind": "проблема", "text": "Т.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}}
		}, "teachers, запись 1: метка не из списка"},
		"one label twice": {nil, []string{"[преподаватель 3]"}, func(o map[string]any) {
			entry := map[string]any{"who": "[преподаватель 3]", "kind": "упоминание", "text": "Т.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}
			o["teachers"] = []any{entry, entry}
		}, "teachers, запись 2: метка уже есть в другой записи"},
		"a name as the author wrote it": {nil, []string{"[преподаватель 3]"}, func(o map[string]any) {
			o["teachers"] = []any{map[string]any{"who": "полковник Ивнов", "kind": "упоминание", "text": "Т.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}}
		}, ""},
		"unknown kind": {nil, nil, func(o map[string]any) {
			o["teachers"] = []any{map[string]any{"who": "Иванов", "kind": "жалоба", "text": "Т.", "quotes": []any{map[string]any{"question": "2.4", "quote": "поблагодарить полковника Иванова"}}}}
		}, "teachers, запись 1: kind"},
		"a level": {nil, nil, func(o map[string]any) {
			o["summary"] = "У автора высокий уровень мотивации."
		}, "summary: недопустимое слово («высокий уров»)"},
		"a diagnosis": {nil, nil, func(o map[string]any) {
			o["positive"].([]any)[0].(map[string]any)["text"] = "Есть расстройство сна."
		}, "positive, пункт 1: недопустимое слово («расстройств»)"},
		"to my regret": {nil, nil, func(o map[string]any) { o["summary"] = "К сожалению, автор устал." }, "summary: недопустимое слово («к сожалению»)"},
		"the author's own words may hold the phrase": {nil, nil, func(o map[string]any) {
			o["conclusion"].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.2", "quote": "Первые дни было очень страшно"}}
		}, ""},
		"empty summary": {nil, nil, func(o map[string]any) { o["summary"] = " " }, "summary: пустой текст"},
		"no moment":     {nil, nil, func(o map[string]any) { delete(o["moments"].(map[string]any), "emotions") }, "moments.emotions: нет момента"},
	}
	for name, c := range mutations {
		payload := essayPayloadFor(answers, c.previous, c.labels...)
		output := goodEssayFor(payload)
		c.mutate(output)
		problems := strings.Join(checkEssayAnswer(roundTrip(t, output), payload), " | ")
		switch {
		case c.want == "" && problems != "":
			t.Errorf("%s: %s", name, problems)
		case c.want != "" && !strings.Contains(problems, c.want):
			t.Errorf("%s: want %q, got %q", name, c.want, problems)
		}
	}
	// A problem never repeats the text of the answer.
	payload := essayPayloadFor(answers, nil)
	output := goodEssayFor(payload)
	output["positive"].([]any)[0].(map[string]any)["quotes"] = []any{map[string]any{"question": "1.1", "quote": "Секретная выдуманная цитата модели"}}
	if problems := strings.Join(checkEssayAnswer(roundTrip(t, output), payload), "|"); strings.Contains(problems, "Секретная") {
		t.Fatalf("the problem quotes the answer: %s", problems)
	}
}

// goodEssayFor is a good answer of v1 to a small input made by essayPayloadFor.
func goodEssayFor(payload *object) map[string]any {
	quote := []any{map[string]any{"question": "1.1", "quote": "Я пошёл «в ВУЦ» ради «службы»"}}
	moment := func() map[string]any {
		return map[string]any{"text": "По ответам видно, что автор об этом пишет.", "quotes": quote}
	}
	change := []any{}
	if _, ok := payload.values["прежнее_эссе"]; ok {
		for _, name := range essayMoments {
			change = append(change, map[string]any{"moment": name, "text": "Раньше было иначе.",
				"before": map[string]any{"question": "1.1", "quote": "Раньше я хотел только отсрочку"}, "after": map[string]any{"question": "1.1", "quote": "Я пошёл в ВУЦ ради службы"}})
		}
	}
	return map[string]any{
		"summary":  "Автор пришёл в ВУЦ с ожиданиями.",
		"moments":  map[string]any{"beliefs": moment(), "emotions": moment(), "values": moment(), "abilities": moment()},
		"positive": []any{map[string]any{"text": "Автор ценит помощь.", "about": "о себе", "basis": "сказано автором", "quotes": quote}},
		"negative": []any{}, "attention": []any{}, "teachers": []any{}, "change": change,
		"conclusion": map[string]any{"text": "По ответам видно, что становление идёт.", "quotes": quote},
	}
}

// roundTrip makes the answer what the layer checks: generic JSON.
func roundTrip(t *testing.T, output map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err = json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	return back
}
