package usecase

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

type cohortReview struct {
	Author string `json:"автор"`
	Path   struct {
		Number string `json:"номер"`
		Text   string `json:"текст"`
	} `json:"путь"`
	Conclusion struct {
		Number string `json:"номер"`
	} `json:"итог"`
	Moments []struct {
		Number string `json:"номер"`
	} `json:"моменты"`
	Positive []struct {
		Number string `json:"номер"`
	} `json:"положительное"`
	Negative []struct {
		Number string `json:"номер"`
	} `json:"отрицательное"`
	Attention []struct {
		Number string `json:"номер"`
	} `json:"на_что_обратить_внимание"`
	Change []struct {
		Number string `json:"номер"`
	} `json:"что_изменилось"`
}

type cohortInput struct {
	Essays   int            `json:"эссе_в_сводке"`
	Previous int            `json:"с_прежним_эссе"`
	Reviews  []cohortReview `json:"разборы"`
}

func readCohortInput(user string) cohortInput {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input cohortInput
	if err := json.Unmarshal([]byte(user), &input); err != nil {
		panic(fmt.Sprintf("%v: %.80s", err, user))
	}
	return input
}

// goodCohort answers the way the instruction for the summary asks: every common
// point stands on the first two authors; the third one's point of attention is
// left out of the common ones.
func goodCohort(user string) map[string]any {
	input := readCohortInput(user)
	both := func(kind string) []any { return []any{"1." + kind, "2." + kind} }
	moment := func(key string) map[string]any {
		return map[string]any{"text": "Авторы отмечают это в эссе.", "refs": both(key)}
	}
	change := []any{}
	if input.Previous > 0 {
		for _, key := range momentKeys {
			change = append(change, map[string]any{"moment": momentNames[key], "text": "Со временем это меняется.", "refs": []any{"1.change." + key}})
		}
	}
	return map[string]any{
		"overview":    "В эссе пишут о первых впечатлениях и о привыкании.",
		"moments":     map[string]any{"beliefs": moment("beliefs"), "emotions": moment("emotions"), "values": moment("values"), "abilities": moment("abilities")},
		"positive":    []any{map[string]any{"text": "Авторы ценят помощь товарищей.", "about": "о себе", "refs": both("pos.1")}},
		"negative":    []any{},
		"attention":   []any{map[string]any{"text": "Нужна поддержка в первые недели.", "about": "о ВУЦ", "refs": both("att.1")}},
		"paths":       []any{map[string]any{"text": "Приходят с ожиданиями, привыкают, начинают думать о службе.", "refs": []any{"1.path", "2.path", "1.conclusion", "2.conclusion"}}},
		"environment": []any{map[string]any{"text": "Авторы связывают привыкание с поддержкой преподавателей.", "moment": "эмоции", "refs": both("emotions")}},
		"conclusion":  map[string]any{"text": "По разборам видно, что становление идёт.", "refs": both("conclusion")},
		"change":      change,
	}
}

// goodCohortV2 answers the way the instruction essay-cohort/v2 asks: every
// paragraph stands on the first two authors; the first recommendation on their
// first ones, the second on a paragraph of one and a recommendation of the other.
func goodCohortV2(string) map[string]any {
	both := func(kind string) []any { return []any{"1." + kind, "2." + kind} }
	paragraph := func(key string) map[string]any {
		return map[string]any{"text": "Авторы отмечают это в эссе.", "refs": both(key)}
	}
	return map[string]any{
		"formation": paragraph("formation"), "moments": paragraph("moments"), "environment": paragraph("environment"),
		"recommendations": []any{
			map[string]any{"text": "Поддерживать новичков в первые недели.", "refs": both("rec.1")},
			map[string]any{"text": "Объяснять смысл порядков.", "refs": []any{"1.moments", "2.rec.2"}},
		},
	}
}

// bothTasksModel answers the analysis of an essay and the summary by what it is asked.
func bothTasksModel(mutate func(map[string]any)) *fakeModel {
	return &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		var answer map[string]any
		if strings.Contains(c.System, "Тебе передаются только эти") {
			answer = goodCohortV2(c.User)
			if mutate != nil {
				mutate(answer)
			}
		} else {
			answer = goodEssay(c.User)
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
}

func (f *essayFixture) summary(teacher string) map[string]any {
	f.t.Helper()
	value, err := f.service.EssaySummary(teacher, f.assignment)
	if err != nil {
		f.t.Fatal(err)
	}
	return roundTrip(f.t, value.(map[string]any))
}

// three writes three different essays of the assignment.
func (f *essayFixture) three() {
	f.t.Helper()
	f.submitted("cohort-a@example.test", "2401", f.assignment, map[string]string{"1.1": "Пошёл служить, как отец.", "2.4": "Спасибо полковнику Иванову за помощь."})
	f.submitted("cohort-b@example.test", "2401", f.assignment, map[string]string{"1.1": "Хотел отсрочку от армии.", "2.4": "Благодарю Иванова за то, что объяснил."})
	f.submitted("cohort-c@example.test", "2401", f.assignment, map[string]string{"1.1": "Позвал друг из университета.", "2.4": "Помог полковник Сидоров, а Иванов нет."})
}

func TestSummaryIsComposedOnTheButtonAndCountedByTheProgram(t *testing.T) {
	f := newEssayPortraitFixture(t)
	if _, err := f.service.SaveInstructor(f.owner, "", InstructorInput{Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович", Rank: "полковник"}); err != nil {
		t.Fatal(err)
	}
	m := bothTasksModel(nil)
	f.service.UseModel(m)
	f.three()
	f.idle()
	if m.count() != 3 {
		t.Fatalf("%d calls for three essays: the summary must not be made by itself", m.count())
	}
	before := f.summary(f.owner)
	counts := before["counts"].(map[string]any)
	if before["canCompose"] != true || before["status"] != portraitNone || before["summary"] != nil || counts["submitted"] != 3.0 || counts["analysed"] != 3.0 || counts["withPrevious"] != 0.0 {
		t.Fatalf("before the button: %+v", before)
	}
	if m.count() != 3 {
		t.Fatal("reading the summary called the model")
	}

	if _, err := f.service.ComposeEssaySummary(f.owner, f.assignment); err != nil {
		t.Fatal(err)
	}
	f.idle()
	if m.count() != 4 {
		t.Fatalf("%d calls after the button", m.count())
	}
	call := m.calls[3]
	if !strings.Contains(call.System, "Тебе передаются только эти выводы") || strings.Contains(call.User, `"quote`) || strings.Contains(call.User, "преподаватели") && strings.Contains(call.User, `"метка"`) {
		t.Fatalf("input of the summary: %.200s", call.User)
	}
	done := f.summary(f.owner)
	summary := done["summary"].(map[string]any)
	common := summary["common"].(map[string]any)
	formation := common["formation"].(map[string]any)
	if done["status"] != portraitReady || summary["stale"] != false || summary["essays"] != 3.0 || summary["version"] != "v2" || formation["authors"] != 2.0 || formation["of"] != 3.0 || len(formation["sources"].([]any)) != 2 {
		t.Fatalf("summary: %+v", done)
	}
	// A source is the paragraph of the analysis, without quotes (decision Д12).
	source := formation["sources"].([]any)[0].(map[string]any)
	if source["student"] != "Тестов Тест" || source["platoon"] != "2401" || source["text"] != "Автор пришёл в ВУЦ с ожиданиями и привык к порядкам." || len(source["quotes"].([]any)) != 0 || source["submissionId"] == "" {
		t.Fatalf("source: %+v", source)
	}
	advice := common["recommendations"].([]any)
	if len(advice) != 2 || advice[1].(map[string]any)["authors"] != 2.0 || len(advice[1].(map[string]any)["sources"].([]any)) != 2 {
		t.Fatalf("recommendations: %+v", advice)
	}
	// The recommendations of the analyses no recommendation of the summary stands on stay on the screen.
	left := common["notIncluded"].([]any)
	if len(left) != 3 || left[0].(map[string]any)["ref"] != "1.rec.2" || left[0].(map[string]any)["text"] != "Опираться на его интерес к службе." {
		t.Fatalf("not included: %+v", left)
	}
	// Equal analyses give the same summary and no call of the model.
	if _, err := f.service.ComposeEssaySummary(f.owner, f.assignment); err != nil {
		t.Fatal(err)
	}
	f.idle()
	if m.count() != 4 {
		t.Fatalf("%d calls after a second button", m.count())
	}

	// The block of instructors needs no summary: the program groups the positions and the surnames.
	positions := done["teachers"].(map[string]any)["positions"].([]any)
	if len(positions) != 1 || positions[0].(map[string]any)["position"] != "преподаватель" {
		t.Fatalf("positions: %+v", positions)
	}
	people := positions[0].(map[string]any)["people"].([]any)
	first, second := people[0].(map[string]any), people[1].(map[string]any)
	if len(people) != 2 || first["name"] != "Иванов" || first["thanks"] != 2.0 || len(first["lines"].([]any)) != 2 || second["name"] != "Сидоров" || second["thanks"] != 1.0 {
		t.Fatalf("people: %+v", people)
	}

	// A new essay makes the summary stale until the button is pressed again.
	f.submitted("cohort-d@example.test", "2401", f.assignment, map[string]string{"1.1": "Зачем-то поступил в ВУЦ."})
	f.idle()
	stale := f.summary(f.owner)
	if stale["summary"].(map[string]any)["stale"] != true || stale["summary"].(map[string]any)["essays"] != 3.0 || stale["counts"].(map[string]any)["analysed"] != 4.0 {
		t.Fatalf("after a new essay: %+v", stale["summary"])
	}
	if _, err := f.service.ComposeEssaySummary(f.owner, f.assignment); err != nil {
		t.Fatal(err)
	}
	f.idle()
	fresh := f.summary(f.owner)
	if m.count() != 6 || fresh["summary"].(map[string]any)["stale"] != false || fresh["summary"].(map[string]any)["essays"] != 4.0 {
		t.Fatalf("%d calls, summary: %+v", m.count(), fresh["summary"])
	}
}

func TestSummaryNeedsTwoAnalysedEssaysAndItsOwnerOnly(t *testing.T) {
	f := newEssayPortraitFixture(t)
	m := bothTasksModel(nil)
	f.service.UseModel(m)
	f.submitted("cohort-one@example.test", "2401", f.assignment, nil)
	f.idle()
	one := f.summary(f.owner)
	if one["canCompose"] != false || one["status"] != portraitNone {
		t.Fatalf("one essay: %+v", one)
	}
	if _, err := f.service.ComposeEssaySummary(f.owner, f.assignment); kind(err) != Invalid {
		t.Fatalf("one essay: %v", err)
	}
	f.submitted("cohort-two@example.test", "2401", f.assignment, map[string]string{"1.1": "Совсем другой ответ тут."})
	f.idle()
	if _, err := f.service.EssaySummary(f.other, f.assignment); kind(err) != NotFound {
		t.Fatalf("another teacher reads: %v", err)
	}
	if _, err := f.service.ComposeEssaySummary(f.other, f.assignment); kind(err) != NotFound {
		t.Fatalf("another teacher composes: %v", err)
	}
	if _, err := f.service.PortraitFor(f.other, essayCohortTask, f.assignment); kind(err) != NotFound {
		t.Fatalf("another teacher reads the portrait: %v", err)
	}
	// Opening the portrait of the summary does not make it either.
	calls := m.count()
	if view, err := f.service.PortraitFor(f.owner, essayCohortTask, f.assignment); err != nil || view.Status != portraitNone || m.count() != calls {
		t.Fatalf("%+v %v, %d calls", view, err, m.count())
	}
}

func TestSummaryWithoutAModelStillHasTheBlockOfInstructors(t *testing.T) {
	f := newEssayPortraitFixture(t)
	f.three()
	view := f.summary(f.owner)
	if view["status"] != portraitOff || view["summary"] != nil || view["teachers"] == nil {
		t.Fatalf("%+v", view)
	}
	if _, err := f.service.ComposeEssaySummary(f.owner, f.assignment); kind(err) != Invalid {
		t.Fatalf("no analyses without a model: %v", err)
	}
}

func TestCohortPayloadIsStableAndHoldsNoQuotesAndNoTeachers(t *testing.T) {
	f := newEssayPortraitFixture(t)
	f.service.UseModel(bothTasksModel(nil))
	f.three()
	f.idle()
	db := mustSnapshot(t, f.service.repo)
	first, err := f.service.cohortPayloadV2(db, f.assignment)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := f.service.cohortPayloadV2(db, f.assignment)
	if string(canonicalJSON(first)) != string(canonicalJSON(second)) {
		t.Fatal("the same set gives another input")
	}
	text := string(canonicalJSON(first))
	for _, banned := range []string{"quote", "teachers", "преподаватели", "Тестов", "2401", "before"} {
		if strings.Contains(text, banned) {
			t.Fatalf("%q is in the summary input", banned)
		}
	}
	var input struct {
		Essays   int `json:"эссе_в_сводке"`
		Previous int `json:"с_прежним_эссе"`
		Reviews  []struct {
			Author     string `json:"автор"`
			Earlier    bool   `json:"прежнее_эссе"`
			Paragraphs []struct {
				Number string `json:"номер"`
				Text   string `json:"текст"`
			} `json:"абзацы"`
			Advice []struct {
				Number string `json:"номер"`
			} `json:"рекомендации"`
		} `json:"разборы"`
	}
	if err = json.Unmarshal([]byte(modelText(canonicalJSON(first))), &input); err != nil || input.Essays != 3 || input.Previous != 0 || len(input.Reviews) != 3 {
		t.Fatalf("input: %+v %v", input, err)
	}
	for n, review := range input.Reviews {
		id := fmt.Sprint(n + 1)
		if review.Author != id || review.Earlier || len(review.Paragraphs) != 3 || review.Paragraphs[0].Number != id+".formation" || review.Paragraphs[2].Number != id+".environment" ||
			review.Paragraphs[1].Text != "Автор ценит помощь товарищей." || len(review.Advice) != 2 || review.Advice[1].Number != id+".rec.2" {
			t.Fatalf("review %s: %+v", id, review)
		}
	}
	// The order of the authors follows the hash, not the time of writing.
	ordered := authorOrder(f.assignment, assignmentSubmissions(db, f.assignment))
	authors := f.service.cohortAuthors(db, f.assignment)
	for n := range ordered {
		if authors[n].Submission.ID != ordered[n].ID {
			t.Fatalf("author %d", n+1)
		}
	}
}

func TestParseRef(t *testing.T) {
	for ref, want := range map[string]string{
		"3.pos.2": "pos", "10.neg.1": "neg", "1.att.12": "att", "2.beliefs": "moment.beliefs", "2.abilities": "moment.abilities",
		"4.path": "path", "4.conclusion": "conclusion", "5.change.values": "change.values",
		"3.formation": "formation", "2.moments": "moments", "1.environment": "environment", "4.rec.2": "rec",
	} {
		if _, kind, ok := parseRef(ref); !ok || kind != want {
			t.Errorf("%s: %q %v", ref, kind, ok)
		}
	}
	for _, ref := range []string{"", "pos.1", "0.path", "03.path", "1.pos", "1.pos.0", "1.pos.x", "1.change", "1.change.other", "1.other", "1.path.1", "1.2.3.4", "-1.path", "1.rec", "1.rec.0", "1.formation.1", "1.change.formation"} {
		if _, _, ok := parseRef(ref); ok {
			t.Errorf("%q accepted", ref)
		}
	}
}

func cohortPayloadFor(withChange bool) *object {
	payload := newObject()
	reviews := []any{}
	for n := 1; n <= 3; n++ {
		review := newObject()
		item := func(ref string) *object {
			o := newObject()
			o.set("номер", fmt.Sprintf("%d.%s", n, ref))
			o.set("текст", "Текст.")
			return o
		}
		review.set("автор", fmt.Sprint(n))
		review.set("путь", item("path"))
		review.set("итог", item("conclusion"))
		review.set("моменты", []any{item("beliefs"), item("emotions"), item("values"), item("abilities")})
		review.set("положительное", []any{item("pos.1"), item("pos.2")})
		review.set("отрицательное", []any{item("neg.1")})
		review.set("на_что_обратить_внимание", []any{item("att.1")})
		if withChange {
			review.set("что_изменилось", []any{item("change.beliefs"), item("change.emotions"), item("change.values"), item("change.abilities")})
		}
		reviews = append(reviews, review)
	}
	payload.set("разборы", reviews)
	return payload
}

func TestCheckCohortAnswer(t *testing.T) {
	cases := map[string]struct {
		change bool
		mutate func(map[string]any)
		want   string
	}{
		"good answer":             {false, func(map[string]any) {}, ""},
		"good answer with change": {true, func(map[string]any) {}, ""},
		"reference that is not in the input": {false, func(o map[string]any) {
			o["positive"].([]any)[0].(map[string]any)["refs"] = []any{"1.pos.1", "2.pos.9"}
		}, "positive, пункт 1: ссылки «2.pos.9» нет во входных данных"},
		"reference of another kind": {false, func(o map[string]any) {
			o["positive"].([]any)[0].(map[string]any)["refs"] = []any{"1.pos.1", "2.neg.1"}
		}, "positive, пункт 1: ссылка «2.neg.1» не того вида"},
		"a moment stands on another moment": {false, func(o map[string]any) {
			o["moments"].(map[string]any)["values"].(map[string]any)["refs"] = []any{"1.values", "2.emotions"}
		}, "moments.values: ссылка «2.emotions» не того вида"},
		"one author only": {false, func(o map[string]any) {
			o["positive"].([]any)[0].(map[string]any)["refs"] = []any{"1.pos.1", "1.pos.2"}
		}, "positive, пункт 1: ссылки нужны не меньше чем на 2 разных авторов"},
		"no references": {false, func(o map[string]any) { o["conclusion"].(map[string]any)["refs"] = []any{} }, "conclusion: ссылки нужны не меньше чем на 2 разных авторов"},
		"a point in two common points": {false, func(o map[string]any) {
			point := func(refs ...any) map[string]any {
				return map[string]any{"text": "Пункт.", "about": "о себе", "refs": refs}
			}
			o["positive"] = []any{point("1.pos.1", "2.pos.1"), point("1.pos.1", "3.pos.1")}
		}, "positive, пункт 2: «1.pos.1» уже в пункте 1"},
		"an author in two paths": {false, func(o map[string]any) {
			o["paths"] = []any{map[string]any{"text": "Путь.", "refs": []any{"1.path", "2.path"}}, map[string]any{"text": "Путь.", "refs": []any{"1.conclusion", "3.conclusion"}}}
		}, "paths, путь 2: автор из пути 1 записан и сюда"},
		"a path on a point": {false, func(o map[string]any) {
			o["paths"].([]any)[0].(map[string]any)["refs"] = []any{"1.pos.1", "2.path"}
		}, "paths, путь 1: ссылка «1.pos.1» не того вида"},
		"environment may stand on anything": {false, func(o map[string]any) {
			o["environment"].([]any)[0].(map[string]any)["refs"] = []any{"1.pos.1", "2.att.1"}
		}, ""},
		"too many paths": {false, func(o map[string]any) {
			path := map[string]any{"text": "Путь.", "refs": []any{"1.path", "2.path"}}
			o["paths"] = []any{path, path, path, path, path}
		}, "paths: путей должно быть не больше 4"},
		"too many points": {false, func(o map[string]any) {
			point := o["positive"].([]any)[0]
			o["positive"] = []any{point, point, point, point, point, point, point, point}
		}, "positive: пунктов должно быть не больше 7"},
		"no environment is fine after cleaning": {false, func(o map[string]any) { o["environment"] = []any{}; o["paths"] = []any{}; o["positive"] = []any{} }, ""},
		"a count in words": {false, func(o map[string]any) {
			o["overview"] = "Большинство авторов пишут о товарищах."
		}, "overview: слово о количестве"},
		"everybody": {false, func(o map[string]any) { o["overview"] = "Все авторы пишут о товарищах." }, "overview: слово о количестве"},
		"nobody": {false, func(o map[string]any) {
			o["positive"].([]any)[0].(map[string]any)["text"] = "Никто не жалуется."
		}, "positive, пункт 1: слово о количестве"},
		"a digit": {false, func(o map[string]any) { o["overview"] = "Двое из 3 авторов пишут." }, "overview: слово о количестве, цифра или метка"},
		"a label": {false, func(o map[string]any) { o["overview"] = "Благодарят [преподаватель 3]." }, "overview: слово о количестве, цифра или метка"},
		"all as a part of a word is fine": {false, func(o map[string]any) {
			o["overview"] = "Авторы пишут о всестороннем развитии и о всём, что видят."
		}, ""},
		"a level": {false, func(o map[string]any) {
			o["conclusion"].(map[string]any)["text"] = "По разборам видно высокий уровень."
		}, "conclusion: недопустимое слово"},
		"conclusion starts otherwise": {false, func(o map[string]any) { o["conclusion"].(map[string]any)["text"] = "Становление идёт." }, "conclusion: должен начинаться"},
		"conclusion starts with the other phrase": {false, func(o map[string]any) {
			o["conclusion"].(map[string]any)["text"] = "По разборам можно предположить, что идёт."
		}, ""},
		"change although nobody has an earlier essay": {false, func(o map[string]any) {
			o["change"] = []any{map[string]any{"moment": "убеждения", "text": "Т.", "refs": []any{"1.change.beliefs"}}}
		}, "change: ни у кого нет прежнего эссе"},
		"no change with earlier essays": {true, func(o map[string]any) { o["change"] = []any{} }, "change: нужно 4 записи по порядку, получено 0"},
		"change in another order": {true, func(o map[string]any) {
			change := o["change"].([]any)
			change[0], change[1] = change[1], change[0]
		}, "change, запись 1: момент не тот или не по порядку"},
		"change of another moment": {true, func(o map[string]any) {
			o["change"].([]any)[1].(map[string]any)["refs"] = []any{"1.change.beliefs"}
		}, "change, запись 2: ссылка «1.change.beliefs» не того вида"},
		"change without a reference": {true, func(o map[string]any) { o["change"].([]any)[2].(map[string]any)["refs"] = []any{} }, "change, запись 3: ссылки нужны не меньше чем на 1"},
		"no moment":                  {false, func(o map[string]any) { delete(o["moments"].(map[string]any), "values") }, "moments.values: нет момента"},
		"unknown about":              {false, func(o map[string]any) { o["attention"].([]any)[0].(map[string]any)["about"] = "о людях" }, "attention, пункт 1: about"},
		"unknown moment of environment": {false, func(o map[string]any) {
			o["environment"].([]any)[0].(map[string]any)["moment"] = "настроение"
		}, "environment, пункт 1: moment"},
	}
	for name, c := range cases {
		payload := cohortPayloadFor(c.change)
		output := goodCohortFor(c.change)
		c.mutate(output)
		problems := strings.Join(checkCohortAnswer(roundTrip(t, output), payload), " | ")
		switch {
		case c.want == "" && problems != "":
			t.Errorf("%s: %s", name, problems)
		case c.want != "" && !strings.Contains(problems, c.want):
			t.Errorf("%s: want %q, got %q", name, c.want, problems)
		}
	}
}

func goodCohortFor(withChange bool) map[string]any {
	user := "{\"эссе_в_сводке\":3,\"с_прежним_эссе\":0,\"разборы\":[]}"
	output := goodCohort(user)
	output["positive"].([]any)[0].(map[string]any)["refs"] = []any{"1.pos.1", "2.pos.1"}
	if withChange {
		output["change"] = []any{}
		for _, key := range momentKeys {
			output["change"] = append(output["change"].([]any), map[string]any{"moment": momentNames[key], "text": "Со временем это меняется.", "refs": []any{"1.change." + key}})
		}
	}
	return output
}

func TestOverviewShowsStudentsOfTheCohortWithTheirState(t *testing.T) {
	f := newEssayPortraitFixture(t)
	f.service.UseModel(bothTasksModel(nil))
	_, done := f.submitted("over-done@example.test", "2401", f.assignment, nil)
	writing := f.student("over-writing@example.test", "2401")
	f.start(writing, f.assignment)
	f.student("over-idle@example.test", "2401")
	f.student("over-outside@example.test", "2402")
	f.idle()
	value, err := f.service.EssayOverview(f.owner)
	if err != nil {
		t.Fatal(err)
	}
	overview := roundTrip(t, value.(map[string]any))
	assignments := overview["assignments"].([]any)
	if len(overview["cohorts"].([]any)) != 1 || len(assignments) != 1 || overview["model"] != true {
		t.Fatalf("overview: %+v", overview)
	}
	view := assignments[0].(map[string]any)
	counts := view["counts"].(map[string]any)
	// The student of the fixture, who did not start, is in the cohort too.
	if view["cohortName"] != "Выпуск 2027" || counts["students"] != 4.0 || counts["submitted"] != 1.0 || counts["notSubmitted"] != 3.0 || counts["analysed"] != 1.0 || counts["withoutAnalysis"] != 0.0 {
		t.Fatalf("counts: %+v", counts)
	}
	byStatus := map[string]int{}
	for _, raw := range view["students"].([]any) {
		student := raw.(map[string]any)
		byStatus[student["status"].(string)]++
		if student["status"] == "submitted" && (student["analysis"] != portraitReady || student["submissionId"] != done) {
			t.Fatalf("student: %+v", student)
		}
	}
	if byStatus["submitted"] != 1 || byStatus["writing"] != 1 || byStatus["not_started"] != 2 || len(view["students"].([]any)) != 4 {
		t.Fatalf("states: %v of %+v", byStatus, view["students"])
	}
	// Another teacher has no cohorts and no essays of his own.
	other, _ := f.service.EssayOverview(f.other)
	if got := roundTrip(t, other.(map[string]any)); len(got["cohorts"].([]any)) != 0 || len(got["assignments"].([]any)) != 0 {
		t.Fatalf("another teacher: %+v", got)
	}
}

func TestTeacherSeesTheEarlierEssayEvenFromAnotherTeacher(t *testing.T) {
	f := newEssayPortraitFixture(t)
	login, first := f.submitted("early@example.test", "2401", f.assignment, map[string]string{"1.1": "Раньше писал так."})
	f.backdate(first, 48*3600*1e9)
	// A cohort of another teacher gave the first essay: make the second one from the owner.
	if err := f.service.repo.Update(func(db *model.Database) error {
		for i := range db.Assignments {
			db.Assignments[i].OwnerID = f.other
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	second := f.assign("Выпускное")
	id := f.start(login, second)
	f.write(login, id, nil)
	value, err := f.service.FormSubmissionDetailFor(f.owner, id)
	if err != nil {
		t.Fatal(err)
	}
	context := value.(map[string]any)["essay"].(map[string]any)
	previous, ok := context["previous"].(map[string]any)
	if !ok || previous["id"] != first || context["assignmentTitle"] != "Выпускное" {
		t.Fatalf("context: %+v", context)
	}
}

// cleaned runs an answer of v1 through the cleaning and the check, as the layer does.
func cleaned(t *testing.T, payload *object, output map[string]any) (map[string]any, int, []string) {
	t.Helper()
	back := roundTrip(t, output)
	back, changes := cleanCohortAnswer(nil, model.Database{}, Source{}, back, payload)
	return back, changes, checkCohortAnswer(back, payload)
}

func TestCleaningKeepsASourcePointInTheFirstCommonPointOnly(t *testing.T) {
	payload := cohortPayloadFor(false)
	output := goodCohortFor(false)
	point := func(text string, refs ...any) map[string]any {
		return map[string]any{"text": text, "about": "о себе", "refs": refs}
	}
	output["positive"] = []any{point("Первый.", "1.pos.1", "2.pos.1"), point("Второй.", "1.pos.1", "3.pos.1", "3.pos.2"), point("Третий.", "2.pos.2", "3.pos.2")}
	got, changes, problems := cleaned(t, payload, output)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	positive := got["positive"].([]any)
	refs := func(i int) []any { return positive[i].(map[string]any)["refs"].([]any) }
	// The second point lost the repeated 1.pos.1 and is left with one author (3.pos.1 and 3.pos.2), so it goes;
	// its sources are free again and the third point keeps 3.pos.2.
	if len(positive) != 2 || fmt.Sprint(refs(0)) != "[1.pos.1 2.pos.1]" || fmt.Sprint(refs(1)) != "[2.pos.2 3.pos.2]" || changes != 2 {
		t.Fatalf("positive: %v, %d changes", positive, changes)
	}
}

func TestCleaningDropsWrongAndMissingReferencesAndThinPoints(t *testing.T) {
	payload := cohortPayloadFor(false)
	output := goodCohortFor(false)
	output["attention"] = []any{
		map[string]any{"text": "Вид не тот.", "about": "о ВУЦ", "refs": []any{"1.pos.1", "2.att.1", "3.att.1"}},
		map[string]any{"text": "Нет такой.", "about": "о ВУЦ", "refs": []any{"1.att.9", "2.att.1"}},
	}
	output["environment"] = []any{map[string]any{"text": "Связь.", "moment": "эмоции", "refs": []any{"1.emotions", "1.values"}}}
	got, changes, problems := cleaned(t, payload, output)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	attention := got["attention"].([]any)
	if len(attention) != 1 || fmt.Sprint(attention[0].(map[string]any)["refs"]) != "[2.att.1 3.att.1]" || len(got["environment"].([]any)) != 0 || changes < 4 {
		t.Fatalf("attention %v, environment %v, %d changes", attention, got["environment"], changes)
	}
}

func TestCleaningLeavesNoPathsForThreeAuthorsOfOneEach(t *testing.T) {
	payload := cohortPayloadFor(false)
	output := goodCohortFor(false)
	output["paths"] = []any{
		map[string]any{"text": "Путь.", "refs": []any{"1.path"}},
		map[string]any{"text": "Путь.", "refs": []any{"2.path"}},
		map[string]any{"text": "Путь.", "refs": []any{"3.path"}},
	}
	got, _, problems := cleaned(t, payload, output)
	if len(problems) != 0 || len(got["paths"].([]any)) != 0 {
		t.Fatalf("paths %v, problems %v", got["paths"], problems)
	}
	// An author who is in an earlier path is taken out of a later one.
	output["paths"] = []any{
		map[string]any{"text": "Первый.", "refs": []any{"1.path", "2.path"}},
		map[string]any{"text": "Второй.", "refs": []any{"2.conclusion", "3.conclusion"}},
	}
	got, _, problems = cleaned(t, payload, output)
	if len(problems) != 0 || len(got["paths"].([]any)) != 1 {
		t.Fatalf("paths %v, problems %v", got["paths"], problems)
	}
}

func TestAnswerIsRefusedWhenAMomentTheConclusionOrChangeIsLeftWithoutAuthors(t *testing.T) {
	payload := cohortPayloadFor(true)
	for name, c := range map[string]struct {
		mutate func(map[string]any)
		want   string
	}{
		"a moment of one author": {func(o map[string]any) {
			o["moments"].(map[string]any)["values"].(map[string]any)["refs"] = []any{"1.values", "2.emotions", "9.values"}
		}, "moments.values: ссылки нужны не меньше чем на 2 разных авторов"},
		"a conclusion of one author": {func(o map[string]any) { o["conclusion"].(map[string]any)["refs"] = []any{"1.conclusion", "1.path"} }, "conclusion: ссылки нужны не меньше чем на 2 разных авторов"},
		"a change record without references": {func(o map[string]any) {
			o["change"].([]any)[1].(map[string]any)["refs"] = []any{"1.change.beliefs"}
		}, "change, запись 2: ссылки нужны не меньше чем на 1"},
		"no overview": {func(o map[string]any) { o["overview"] = "" }, "overview: пустой текст"},
		"a quantity word in a point that stays": {func(o map[string]any) {
			o["positive"].([]any)[0].(map[string]any)["text"] = "Все авторы пишут."
		}, "positive, пункт 1: слово о количестве"},
	} {
		output := goodCohortFor(true)
		c.mutate(output)
		_, _, problems := cleaned(t, payload, output)
		if joined := strings.Join(problems, " | "); !strings.Contains(joined, c.want) {
			t.Errorf("%s: want %q, got %q", name, c.want, joined)
		}
	}
	// A record of change keeps only the references of its own moment and is never dropped.
	output := goodCohortFor(true)
	output["change"].([]any)[0].(map[string]any)["refs"] = []any{"1.change.beliefs", "2.change.emotions", "x"}
	got, changes, problems := cleaned(t, payload, output)
	record := got["change"].([]any)[0].(map[string]any)
	if len(problems) != 0 || fmt.Sprint(record["refs"]) != "[1.change.beliefs]" || changes != 2 {
		t.Fatalf("record %v, %d changes, problems %v", record, changes, problems)
	}
}

// A model that repeats refs is not refused any more: the summary is stored cleaned.
func TestSummaryIsStoredCleanedWhenTheModelRepeatsReferences(t *testing.T) {
	f := newEssayPortraitFixture(t)
	journalOut := captureJournal(t)
	m := bothTasksModel(func(answer map[string]any) {
		answer["recommendations"] = []any{
			map[string]any{"text": "Первая.", "refs": []any{"1.rec.1", "1.rec.1", "2.rec.1"}},
			map[string]any{"text": "Одного автора.", "refs": []any{"3.rec.1", "3.moments"}},
		}
		answer["environment"].(map[string]any)["refs"] = []any{"1.environment", "2.environment", "1.rec.1"}
	})
	f.service.UseModel(m)
	f.three()
	f.idle()
	if _, err := f.service.ComposeEssaySummary(f.owner, f.assignment); err != nil {
		t.Fatal(err)
	}
	f.idle()
	calls := m.count()
	view := f.summary(f.owner)
	summary, ok := view["summary"].(map[string]any)
	if !ok || view["status"] != portraitReady || calls != 4 {
		t.Fatalf("%d calls, %+v", calls, view)
	}
	output := summary["output"].(map[string]any)
	advice := output["recommendations"].([]any)
	if len(advice) != 1 || fmt.Sprint(advice[0].(map[string]any)["refs"]) != "[1.rec.1 2.rec.1]" || fmt.Sprint(output["environment"].(map[string]any)["refs"]) != "[1.environment 2.environment]" {
		t.Fatalf("stored answer: %+v", output)
	}
	if log := journalOut.String(); !strings.Contains(log, "answer cleaned") || !strings.Contains(log, "essay-cohort") {
		t.Fatalf("journal: %s", log)
	}
}
