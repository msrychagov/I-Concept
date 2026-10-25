package usecase

import (
	"strings"
	"testing"
)

// The fixture fills every section with «нет» and the first hobby: all six
// areas are «не выражен», the hobby gives something to write about.
func TestV4AnswerThatBreaksTheRulesFailsAfterFiveTries(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"one paragraph":   {func(a map[string]any) { a["paragraphs"] = a["paragraphs"].([]any)[:1] }, "абзацев 1"},
		"four paragraphs": {func(a map[string]any) { a["paragraphs"] = append(a["paragraphs"].([]any), "Ещё.", "И ещё.") }, "абзацев 4"},
		"no paragraphs":   {func(a map[string]any) { delete(a, "paragraphs") }, "paragraphs не список строк"},
		"empty paragraph": {func(a map[string]any) { a["paragraphs"].([]any)[1] = " " }, "абзац 2: пустой"},
		"no mark":         {func(a map[string]any) { a["paragraphs"].([]any)[0] = "Студент внимателен." }, "нет «по опыту в …»"},
		"area without exp": {func(a map[string]any) {
			a["paragraphs"].([]any)[1] = "Пригодится в спортивных мероприятиях ВУЦ."
		}, "направление без опыта («спорт»)"},
		"absence": {func(a map[string]any) { a["paragraphs"].([]any)[1] = "Окончил школу без медали." }, "об отсутствии («без медали»)"},
		"family": {func(a map[string]any) {
			a["paragraphs"].([]any)[1] = "По опыту воспитания в семье — ответственность."
		}, "семья, гражданство или судимость"},
		"consider absence": {func(a map[string]any) { a["consider"] = []any{"Детей нет."} }, "consider, пункт 1: написано об отсутствии («нет»)"},
		"consider conclusion": {func(a map[string]any) {
			a["consider"] = []any{"Воспитывался матерью — это влияет на нагрузку."}
		}, "вывод вместо факта"},
		"six points":    {func(a map[string]any) { a["consider"] = []any{"1", "2", "3", "4", "5", "6"} }, "пунктов больше пяти"},
		"consider text": {func(a map[string]any) { a["consider"] = []any{1} }, "consider не список строк"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newAutobiographyFixture(t)
			m := portraitModel(tc.mutate)
			f.service.UseModel(m)
			f.submit()
			f.idle()
			view := f.portrait(f.submission)
			if m.count() != 5 || view.Status != portraitFailed || !strings.Contains(view.Reason, tc.problem) {
				t.Fatalf("%d calls, %+v", m.count(), view)
			}
			m.reply = portraitModel(nil).reply
			if retry, err := f.service.RetryPortrait(f.student.OwnerID, "autobiography", f.submission); err != nil || retry.Status == portraitFailed {
				t.Fatalf("retry: %+v %v", retry, err)
			}
			f.idle()
			if f.portrait(f.submission).Status != portraitReady {
				t.Fatalf("after retry: %+v", f.portrait(f.submission))
			}
		})
	}
}

// levelsPayload is the input of a student with the given levels and hobbies.
func levelsPayload(levels map[string]string, hobbies ...string) *object {
	payload := newObject()
	payload.set("уголовная_ответственность", "не привлекался")
	payload.set("увлечения", hobbies)
	object := newObject()
	for _, area := range potentialAreas {
		name := lowerFirst(area.Title)
		level := levels[name]
		if level == "" {
			level = "не выражен"
		}
		object.set(name, level)
	}
	payload.set("уровни", object)
	return payload
}

func TestV4FindsAreasWithoutExperience(t *testing.T) {
	empty := []string{"спорт", "музыка и творчество", "военно-патриотическая деятельность", "наука", "волонтёрство", "иностранные языки"}
	for text, area := range map[string]string{
		"Пригодится на спортивных соревнованиях.": "спорт",
		"Спорт ему пригодится.":                   "спорт",
		"Знак ГТО.": "спорт",
		"Пригодится на смотрах художественной самодеятельности и в танцах.": "музыка и творчество",
		"Участвовал в «Юнармии».":           "военно-патриотическая деятельность",
		"Пригодится в научной работе.":      "наука",
		"Призёр олимпиады.":                 "наука",
		"Пригодится в волонтёрских акциях.": "волонтёрство",
		"Знает английский.":                 "иностранные языки",
	} {
		problems := absenceProblems("абзац", text, empty)
		if len(problems) != 1 || !strings.Contains(problems[0], "«"+area+"»") {
			t.Errorf("%q: %v", text, problems)
		}
	}
	for _, text := range []string{
		"По опыту в киберспорте — быстрая реакция.",
		"Пригодится на военно-спортивных сборах и физической подготовке.",
		"По опыту в чтении — интерес к научно-популярной литературе.",
		"Учится на факультете компьютерных наук.",
		"По опыту в программировании — знание языков программирования.",
		"Пригодится на строевой подготовке.",
	} {
		if problems := absenceProblems("абзац", text, empty); len(problems) != 0 {
			t.Errorf("%q: %v", text, problems)
		}
	}
	// An area with experience may be named.
	if problems := absenceProblems("абзац", "По опыту в самбо — выносливость; пригодится в спорте.", empty[1:]); len(problems) != 0 {
		t.Errorf("sport with experience: %v", problems)
	}
}

func TestV4FindsAbsence(t *testing.T) {
	for text, word := range map[string]string{
		"Окончил школу без медали.":                   "без медали",
		"Наград не имеет.":                            "не имеет",
		"Нет спортивного опыта.":                      "нет",
		"Детей нет.":                                  "нет",
		"К уголовной ответственности не привлекался.": "не привлекался",
		"За границу не выезжал.":                      "не выезжал",
		"Родственники с судимостью отсутствуют.":      "отсутствуют",
		"Опыт в науке не указан.":                     "не указан",
	} {
		problems := absenceProblems("пункт", text, nil)
		if len(problems) != 1 || !strings.Contains(problems[0], "«"+word+"»") {
			t.Errorf("%q: %v", text, problems)
		}
	}
	for _, text := range []string{
		"Привлекался к уголовной ответственности; подробности — в ответах студента.",
		"Воспитывался обоими родителями.",
		"Женат, есть один ребёнок.",
		"По опыту в футболе — не только выносливость, но и умение играть в команде.",
		"У брата есть иностранное гражданство (Казахстан).",
	} {
		if problems := absenceProblems("пункт", text, nil); len(problems) != 0 {
			t.Errorf("%q: %v", text, problems)
		}
	}
}

func TestV4StudentWithoutActivitiesGetsTheFixedParagraph(t *testing.T) {
	payload := levelsPayload(nil)
	good := map[string]any{"paragraphs": []any{noActivities + "."}, "consider": []any{}}
	if problems := checkAutobiographyAnswerV4(good, payload); len(problems) != 0 {
		t.Fatalf("the fixed paragraph fails: %v", problems)
	}
	other := map[string]any{"paragraphs": []any{"По опыту в занятиях — внимательность.", "Пригодится в ВУЦ."}, "consider": []any{}}
	if problems := checkAutobiographyAnswerV4(other, payload); len(problems) != 1 || !strings.Contains(problems[0], "занятий нет") {
		t.Fatalf("two paragraphs without activities: %v", problems)
	}
	// With a hobby the fixed paragraph is not enough.
	if problems := checkAutobiographyAnswerV4(good, levelsPayload(nil, "Чтение")); len(problems) == 0 {
		t.Fatal("the fixed paragraph passes with a hobby")
	}
}

func TestV4QualitiesStillComeOnlyFromActivities(t *testing.T) {
	payload := levelsPayload(map[string]string{"спорт": "средний"}, "Чтение")
	work := newObject()
	work.set("должность", "официант")
	payload.set("работа_до_вуза", []any{work})
	answer := func(second string) map[string]any {
		return map[string]any{"paragraphs": []any{"По опыту в футболе — выносливость и командная игра.", second}, "consider": []any{}}
	}
	if problems := checkAutobiographyAnswerV4(answer("До вуза работал официантом. Пригодится на физической подготовке."), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	for second, want := range map[string]string{
		"Работал официантом — навыки общения.":          "из работы до вуза",
		"По опыту жизни с матерью — самостоятельность.": "семья, гражданство или судимость",
	} {
		if problems := checkAutobiographyAnswerV4(answer(second), payload); len(problems) != 1 || !strings.Contains(problems[0], want) || strings.Contains(problems[0], "официант") {
			t.Errorf("%q: %v", second, problems)
		}
	}
}
