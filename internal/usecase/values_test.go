package usecase

import (
	"encoding/json"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
	"vucdiagnostic/internal/valuebank"
)

// promptsWithout leaves a task out: a fixture that counts the calls of the
// model of other tasks does not want the values of its texts.
func promptsWithout(prompts []promptbank.Prompt, task string) []promptbank.Prompt {
	kept := []promptbank.Prompt{}
	for _, prompt := range prompts {
		if prompt.Task != task {
			kept = append(kept, prompt)
		}
	}
	return kept
}

func mustOntograph(t *testing.T) *valuebank.Ontograph {
	t.Helper()
	o, err := loadOntograph()
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestOntographInputHasTheCoderNodesAndNoPersonalData(t *testing.T) {
	o := mustOntograph(t)
	input := newObject()
	input.set("онтограф", ontographInput(o, false))
	if label := personalData(input); label != "" {
		t.Fatalf("the ontograph looks like personal data: %s", label)
	}
	text := modelText(canonicalJSON(input))
	values := objectList(ontographInput(o, false).values["ценности"])
	notions := 0
	for _, value := range values {
		list, _ := value.values["понятия"].([]string)
		notions += len(list)
	}
	if len(values) != 17 || notions != 178 || !strings.Contains(text, "aid_help — Помощь друг другу") || !strings.Contains(text, "av_col — Эгоизм:") {
		t.Fatalf("%d values, %d notions", len(values), notions)
	}
	for _, banned := range []string{"pravda —", "th_west", "th_ideology", "Ключ:"} {
		if strings.Contains(text, banned) {
			t.Errorf("%q is in the input", banned)
		}
	}
	// About 17 thousand characters (Д-З1): the definitions of the notions do not go.
	if len([]rune(text)) > 20000 {
		t.Errorf("the ontograph is %d characters long", len([]rune(text)))
	}
}

// valuesPayloadFor is the input of the marking of three answers.
func valuesPayloadFor(answers map[string]string) *object {
	rows := []any{}
	for _, number := range []string{"1.1", "1.3", "3.3"} {
		o := newObject()
		o.set("номер", number)
		o.set("вопрос", "Вопрос "+number+"?")
		o.set("ответ", answers[number])
		rows = append(rows, o)
	}
	payload := newObject()
	payload.set("текст", rows)
	return payload
}

var valuesAnswersOfA = map[string]string{
	"1.1": "Отец когда-то служил, дома говорили, что армия учит ответственности.",
	"1.3": "Помогло то, что в первые недели товарищи подсказывали мелочи: как подшить подворотничок, как отвечать офицеру.",
	"3.3": "О родных думаю чаще: понял, как отцу было непросто на службе, и позвонил ему с расспросами. Во взводе появились товарищи, на которых можно положиться.",
}

func valueMark(node, sign, whose, question, quote string) map[string]any {
	return map[string]any{"node": node, "sign": sign, "whose": whose, "question": question, "quote": quote}
}

func TestCheckValuesAnswer(t *testing.T) {
	o := mustOntograph(t)
	good := func() map[string]any {
		return map[string]any{"marks": []any{
			valueMark("dig_resp", "принимает", "другое", "1.1", "дома говорили, что армия учит ответственности"),
			valueMark("aid_help", "принимает", "одобряет у других", "1.3", "товарищи подсказывали мелочи: как подшить подворотничок"),
			valueMark("aid_vyr", "принимает", "своё", "3.3", "Во взводе появились товарищи, на которых можно положиться"),
		}}
	}
	mark := func(field, value string) func(map[string]any) {
		return func(output map[string]any) { output["marks"].([]any)[1].(map[string]any)[field] = value }
	}
	for name, c := range map[string]struct {
		mutate func(map[string]any)
		want   string
	}{
		"good answer":        {func(map[string]any) {}, ""},
		"no marks":           {func(output map[string]any) { output["marks"] = []any{} }, ""},
		"no list":            {func(output map[string]any) { delete(output, "marks") }, "marks: нет списка"},
		"an unknown node":    {mark("node", "v_honest"), "marks, отметка 2: узла «v_honest» нет в онтографе"},
		"an idea":            {mark("node", "rodina"), "marks, отметка 2: «rodina» — идея"},
		"a threat":           {mark("node", "th_west"), "marks, отметка 2: узла «th_west» нет в онтографе"},
		"a sign of its own":  {mark("sign", "одобряет"), "marks, отметка 2: sign"},
		"whose of its own":   {mark("whose", "чужое"), "marks, отметка 2: whose"},
		"no such answer":     {mark("question", "2.4"), "marks, отметка 2: нет ответа с номером 2.4"},
		"a quote not there":  {mark("quote", "товарищи помогали во всём и всегда"), "marks, отметка 2: цитаты нет в ответе 1.3 дословно"},
		"a quote too short":  {mark("quote", "товарищи подсказывали"), "marks, отметка 2: в цитате к вопросу 1.3 2 слов"},
		"forty-one marks": {func(output map[string]any) {
			list := []any{}
			for range valuesMarksMost + 1 {
				list = append(list, valueMark("aid_vyr", "принимает", "своё", "3.3", "Во взводе появились товарищи"))
			}
			output["marks"] = list
		}, "marks: отметок 41, нужно не больше 40"},
	} {
		output := good()
		c.mutate(output)
		joined := strings.Join(checkValuesAnswer(o, roundTrip(t, output), valuesPayloadFor(valuesAnswersOfA)), " | ")
		switch {
		case c.want == "" && joined != "":
			t.Errorf("%s: %s", name, joined)
		case c.want != "" && !strings.Contains(joined, c.want):
			t.Errorf("%s: want %q, got %q", name, c.want, joined)
		}
		if strings.Contains(joined, "во всём и всегда") {
			t.Errorf("%s: a problem repeats the quote", name)
		}
	}
}

func TestCleaningOfValuesDropsRepeatedMarks(t *testing.T) {
	output := roundTrip(t, map[string]any{"marks": []any{
		valueMark("aid_vyr", "принимает", "своё", "3.3", "Во взводе появились товарищи"),
		valueMark("aid_vyr", "принимает", "своё", "3.3", "во взводе  появились Товарищи"),
		valueMark("col_together", "принимает", "своё", "3.3", "Во взводе появились товарищи"),
	}})
	got, changes := cleanValuesAnswer(nil, model.Database{}, Source{}, output, nil)
	if marks := got["marks"].([]any); changes != 1 || len(marks) != 2 {
		t.Fatalf("%d changes, %d marks", changes, len(marks))
	}
}

// The marks of the sample of author A (Вики/Для руководителя/Образцы — эссе.md)
// give the numbers the sample shows.
func TestValuesAreCountedByTheCodingRules(t *testing.T) {
	o := mustOntograph(t)
	output := roundTrip(t, map[string]any{"marks": []any{
		valueMark("dig_resp", "принимает", "другое", "1.1", "дома говорили, что армия учит ответственности"),
		valueMark("aid_help", "принимает", "одобряет у других", "1.3", "товарищи подсказывали мелочи"),
		valueMark("ev_craft", "принимает", "своё", "1.4", "везде нужна точность"),
		valueMark("rig_resp", "принимает", "своё", "2.2", "Рапорт я написал сам"),
		valueMark("dign_other", "принимает", "одобряет у других", "2.2", "говорили как со взрослым"),
		valueMark("honest", "принимает", "своё", "2.3", "о проблемах лучше говорить сразу"),
		valueMark("mem_parents", "принимает", "своё", "3.3", "понял, как отцу было непросто"),
		valueMark("aid_vyr", "принимает", "своё", "3.3", "товарищи, на которых можно положиться"),
		valueMark("col_coop", "принимает", "своё", "3.5", "умение работать в команде"),
		valueMark("srv_post", "принимает", "своё", "3.5", "Хочу получить звание офицера запаса"),
		// The same quote under two notions of one value counts once.
		valueMark("aid_sup", "принимает", "своё", "3.3", "товарищи, на которых можно положиться"),
		// An antivalue the author condemns is counted apart.
		valueMark("av_col", "отвергает", "своё", "3.3", "Во взводе появились товарищи"),
	}})
	count := countValues(o, output, map[string]string{"2.3": "Чем закончилась ситуация?"})
	ideas := map[string]IdeaCount{}
	order := []string{}
	for _, idea := range count.Ideas {
		ideas[idea.ID] = idea
		order = append(order, idea.ID)
	}
	if strings.Join(order, " ") != "pravda rodina vera my volya" || count.Marks != 12 || len(count.Other) != 1 || count.Other[0].Node != "dig_resp" {
		t.Fatalf("ideas %v, %d marks, other %+v", order, count.Marks, count.Other)
	}
	for id, want := range map[string][2]int{"my": {4, 0}, "volya": {2, 1}, "rodina": {2, 0}, "pravda": {0, 0}, "vera": {0, 0}} {
		if got := ideas[id]; got.Accept != want[0] || got.EverydayAccept != want[1] {
			t.Errorf("%s: %d (+%d), want %d (+%d)", id, got.Accept, got.EverydayAccept, want[0], want[1])
		}
	}
	values := map[string]ValueCount{}
	for _, value := range ideas["my"].Values {
		values[value.ID] = value
	}
	for _, value := range ideas["volya"].Values {
		values[value.ID] = value
	}
	// «Честность» counts for «Крепкая семья», its parent in the file for the coder (Т3).
	if values["v_aid"].Accept != 2 || len(values["v_aid"].Quotes) != 3 || values["v_fam"].Accept != 1 || values["v_col"].Accept != 1 || values["v_lab"].Accept != 0 || values["v_lab"].EverydayAccept != 1 {
		t.Fatalf("values: %+v", values)
	}
	if quote := values["v_fam"].Quotes[0]; quote.Label != "Честность" || quote.Asked != "Чем закончилась ситуация?" {
		t.Fatalf("quote: %+v", quote)
	}
	if len(count.Antivalues) != 1 || count.Antivalues[0].ID != "av_col" || count.Antivalues[0].Reject != 1 || count.Antivalues[0].IdeaLabel != "Мы (семья)" {
		t.Fatalf("antivalues: %+v", count.Antivalues)
	}
}

// valuesModel marks the first answer of a text with one value of the author and
// answers the other tasks with an empty object.
func valuesModel() *fakeModel {
	return &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		if !strings.Contains(c.System, "Онтограф ценностей") {
			return "{}", nil
		}
		user, _, _ := strings.Cut(c.User, "\n\n"+retryNote)
		var input struct {
			Text []struct {
				Number string `json:"номер"`
				Answer string `json:"ответ"`
			} `json:"текст"`
		}
		_ = json.Unmarshal([]byte(user), &input)
		words := strings.Fields(input.Text[0].Answer)
		answer := map[string]any{"marks": []any{valueMark("col_act", "принимает", "своё", input.Text[0].Number, strings.Join(words[:min(4, len(words))], " "))}}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
}

func valuesPrompt(t *testing.T) promptbank.Prompt {
	t.Helper()
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	prompt, ok := promptbank.Latest(prompts, valuesTask)
	if !ok {
		t.Fatal("values is not embedded")
	}
	return prompt
}

// A submitted essay is marked by itself; the teacher who gave it sees the
// numbers and the summary of the assignment; the patronymic of the essay stays.
func TestEssayIsMarkedByItselfAndCountedOverTheAssignment(t *testing.T) {
	f := newEssayFixture(t)
	f.service.RegisterPrompts(valuesPrompt(t))
	f.service.portraits.pause = 0
	t.Cleanup(f.service.Close)
	m := valuesModel()
	f.service.UseModel(m)
	_, first := f.submitted("values-a@example.test", "2401", f.assignment, map[string]string{"1.1": "Пришёл служить вместе с другом Иваном Петровичем."})
	f.submitted("values-b@example.test", "2401", f.assignment, map[string]string{"1.1": "Пришёл ради отсрочки и остался."})
	f.idle()
	if m.count() != 2 {
		t.Fatalf("%d calls for two essays", m.count())
	}
	if call := m.calls[0]; !strings.Contains(call.User, `"онтограф"`) || !strings.Contains(call.User, "Иваном Петровичем") || strings.Contains(call.User, `"блок"`) || call.MaxTokens != 4000 {
		t.Fatalf("input: %.200s", call.User)
	}
	view, err := f.service.ValuesFor(f.owner, first, false)
	if err != nil || view.Status != portraitReady || view.Count == nil {
		t.Fatalf("%+v %v", view, err)
	}
	my := view.Count.Ideas[3]
	if my.ID != "my" || my.Accept != 1 || my.Values[0].ID != "v_col" || my.Values[0].Quotes[0].Asked != "Почему вы решили поступать в ВУЦ?" {
		t.Fatalf("count: %+v", view.Count.Ideas)
	}
	if _, err = f.service.ValuesFor(f.other, first, false); kind(err) != NotFound {
		t.Fatalf("another teacher: %v", err)
	}
	summary, err := f.service.EssayValues(f.owner, f.assignment)
	if err != nil || summary.Texts != 2 || summary.Marked != 2 || summary.Ideas[3].Authors != 2 || len(summary.Ideas[3].Values[0].Authors) != 2 {
		t.Fatalf("summary: %+v %v", summary, err)
	}
	if _, err = f.service.EssayValues(f.other, f.assignment); kind(err) != NotFound {
		t.Fatalf("another teacher's summary: %v", err)
	}
}

// The open answers «Мои действия и опыт» of a finished test are marked too, with
// the personal data cut as for the portrait by the test; the platoon counts the
// students.
func TestOpenAnswersOfATestAreMarkedAndCountedOverThePlatoon(t *testing.T) {
	f := newPlatoonFixture(t, 3)
	if err := f.service.repo.Update(func(db *model.Database) error {
		session := &db.Sessions[0].Session
		for _, item := range f.service.itemsFor(*session) {
			if item.Type == "textarea" {
				session.Notes[item.ID] = "Помогал Ивану Петровичу на сборах."
				break
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f.service.RegisterPrompts(valuesPrompt(t))
	m := valuesModel()
	f.service.UseModel(m)
	(&portraitFixture{t: t, service: f.service}).idle()
	marked := 0
	for _, call := range m.calls {
		if strings.Contains(call.System, "Онтограф ценностей") {
			marked++
			if strings.Contains(call.User, "Петровичу") || strings.Contains(call.User, "Иванов Иван") {
				t.Fatalf("personal data reached the model: %.300s", call.User)
			}
		}
	}
	// Two students wrote the same answers: one marking serves both.
	if marked != 2 {
		t.Fatalf("%d markings for three tests, two of them alike", marked)
	}
	view, err := f.service.ValuesFor(f.teacherID, f.sessions[1], false)
	if err != nil || view.Status != portraitReady || view.Count.Ideas[3].Accept != 1 || !strings.HasPrefix(view.Count.Ideas[3].Values[0].Quotes[0].Question, "ACT_") {
		t.Fatalf("%+v %v", view, err)
	}
	summary, err := f.service.PlatoonValues(f.teacherID, "101")
	if err != nil || summary.Texts != 3 || summary.Marked != 3 || summary.Ideas[3].Authors != 3 {
		t.Fatalf("platoon: %+v %v", summary, err)
	}
	if other, _ := f.service.PlatoonValues("someone-else", "101"); other.Texts != 0 {
		t.Fatalf("another teacher sees the platoon: %+v", other)
	}
}

// v2 gives the notions with their definitions, without the tail «Ключ: …».
func TestOntographInputOfV2HasTheDefinitionsOfTheNotions(t *testing.T) {
	o := mustOntograph(t)
	input := newObject()
	input.set("онтограф", ontographInput(o, true))
	if label := personalData(input); label != "" {
		t.Fatalf("the ontograph looks like personal data: %s", label)
	}
	text := modelText(canonicalJSON(input))
	if !strings.Contains(text, "aid_vyr — Взаимовыручка: Взаимовыручка рядом с солидарностью") || strings.Contains(text, "Ключ:") || len([]rune(text)) > 35000 {
		t.Fatalf("%d characters: %.300s", len([]rune(text)), text)
	}
}

func TestSpreadOfTheMarksOfV2(t *testing.T) {
	o := mustOntograph(t)
	mark := func(node, whose string) any { return valueMark(node, "принимает", whose, "1.1", "Пришёл служить вместе с другом") }
	for name, c := range map[string]struct {
		marks []any
		want  string
	}{
		"four of six on one value": {[]any{mark("v_srv", "своё"), mark("srv_post", "своё"), mark("v_srv", "своё"), mark("srv_resp", "своё"), mark("aid_vyr", "своё"), mark("honest", "своё")}, "больше половины отметок (4 из 6) — на «Служение Отечеству»"},
		"three of six is a half":   {[]any{mark("v_srv", "своё"), mark("srv_post", "своё"), mark("srv_resp", "своё"), mark("aid_vyr", "своё"), mark("honest", "своё"), mark("col_act", "своё")}, ""},
		"five marks, one value":    {[]any{mark("srv_post", "своё"), mark("srv_resp", "своё"), mark("srv_act", "своё"), mark("srv_valor", "своё"), mark("srv_free", "своё")}, ""},
		"«другое» is not counted":  {[]any{mark("v_srv", "своё"), mark("v_srv", "другое"), mark("v_srv", "другое"), mark("v_srv", "другое"), mark("aid_vyr", "своё"), mark("honest", "своё"), mark("col_act", "своё")}, ""},
		"four marks, no notion":    {[]any{mark("v_srv", "своё"), mark("v_col", "своё"), mark("v_mem", "своё"), mark("ev_craft", "своё")}, "ни одного понятия"},
		"three marks, no notion":   {[]any{mark("v_srv", "своё"), mark("v_col", "своё"), mark("v_mem", "своё")}, ""},
	} {
		joined := strings.Join(spreadProblems(o, roundTrip(t, map[string]any{"marks": c.marks})), " | ")
		if c.want == "" && joined != "" || c.want != "" && !strings.Contains(joined, c.want) {
			t.Errorf("%s: %s", name, joined)
		}
	}
}

// The marking of v1 is shown while the one of v2 is made: their inputs differ
// by the definitions of the notions (decision Д-З10).
func TestMarkingOfV1IsShownWhileV2IsMade(t *testing.T) {
	f := newEssayFixture(t)
	t.Cleanup(f.service.Close)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	first, _ := promptbank.Find(prompts, valuesTask, "v1")
	f.service.RegisterPrompts(first)
	f.service.portraits.pause = 0
	m := valuesModel()
	f.service.UseModel(m)
	_, id := f.submitted("values-v1@example.test", "2401", f.assignment, map[string]string{"1.1": "Пришёл служить вместе с другом."})
	f.idle()
	if m.count() != 1 || strings.Contains(m.calls[0].User, "Взаимовыручка рядом") {
		t.Fatalf("%d calls of v1", m.count())
	}
	f.service.UseModel(nil)
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(valuesPrompt(t))
	view, err := f.service.ValuesFor(f.owner, id, false)
	if err != nil || view.Status != portraitReady || view.Count == nil || view.Count.Ideas[3].Accept != 1 {
		t.Fatalf("the marking of v1 is not shown: %+v %v", view, err)
	}
	f.service.UseModel(m)
	f.idle()
	if m.count() != 2 || !strings.Contains(m.calls[1].User, "Взаимовыручка рядом") || !strings.Contains(m.calls[1].System, "Сначала ищи понятие") {
		t.Fatalf("%d calls: the marking of v2 must be made by itself", m.count())
	}
}
