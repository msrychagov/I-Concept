package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// RegisterPrompts connects the instructions and turns on the portrait tasks
// whose instruction is among them, each with its newest version. It is called
// once at startup, before UseModel.
func (a *Service) RegisterPrompts(prompts ...promptbank.Prompt) {
	if prompt, ok := promptbank.Latest(prompts, autobiographyForm); ok {
		a.registerPortraitTask(a.autobiographyTask(prompt))
	}
	if prompt, ok := promptbank.Latest(prompts, testPortraitTask); ok {
		a.registerPortraitTask(a.testTask(prompt))
	}
	if prompt, ok := promptbank.Latest(prompts, essayForm); ok {
		a.registerPortraitTask(a.essayTask(prompt))
	}
}

// findSubmission returns the submitted autobiography a source points to.
func findSubmission(db model.Database, s Source) (model.FormSubmission, bool) {
	if s.Kind != "submission" {
		return model.FormSubmission{}, false
	}
	for _, submission := range db.Submissions {
		if submission.ID == s.ID {
			return submission, true
		}
	}
	return model.FormSubmission{}, false
}

// autobiographyTask writes the portrait of a submitted autobiography from the
// anonymised input of model_input.go. The levels of potential are counted by
// the program; the answer is only checked against them.
func (a *Service) autobiographyTask(prompt promptbank.Prompt) portraitTask {
	return portraitTask{
		ID: autobiographyForm, Version: prompt.Version, Prompt: prompt,
		Accepts: func(db model.Database, s Source) bool {
			submission, ok := findSubmission(db, s)
			return ok && submission.FormID == autobiographyForm && submission.Completed
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) {
			submission, _ := findSubmission(db, s)
			payload, _, err := submissionPayload(a.forms[submission.FormID], submission)
			return payload, err
		},
		Check: checkAutobiographyAnswer,
		Visible: func(db model.Database, s Source, teacherID string) bool {
			submission, ok := findSubmission(db, s)
			return ok && teacherOwnsSubmission(db, teacherID, submission)
		},
	}
}

const attentionLimit = 5

// consequence finds a conclusion in a point of «what to consider»: what the
// circumstance affects, how to take it into account, what to do about it. The
// point must hold the fact alone (decided 9 October 2026). The words come from
// the answers that broke this rule in the comparison of the models.
var consequence = regexp.MustCompile(`(?i)влия|учитыв|учесть при|при планировани|при распределени|при назначени|рекоменд|следует|необходимо|потребует|может сказаться|указывает на|свидетельству|расширять|развивать|уделить|обратить внимание|допуск|нагрузк|нарядам|нарядов|наряды|в наряд`)

// criminalDetails is the phrase that must follow the student's own criminal record.
var criminalDetails = regexp.MustCompile(`(?i)подробности[^.;]{0,20}в ответах студента`)

// checkAutobiographyAnswer returns what is wrong with an answer to the
// autobiography instruction that a program can find: the schema, the levels,
// the number of points and the rules of «what to consider».
func checkAutobiographyAnswer(output map[string]any, payload *object) []string {
	problems := []string{}
	if text, ok := output["summary"].(string); !ok || strings.TrimSpace(text) == "" {
		problems = append(problems, "нет характеристики")
	}
	expected := map[string]string{}
	areas := []string{}
	if levels, ok := payload.values["уровни"].(*object); ok {
		for _, key := range levels.keys {
			areas = append(areas, key)
			expected[key], _ = levels.values[key].(string)
		}
	}
	items, listed := output["potential"].([]any)
	records := []map[string]any{}
	for _, item := range items {
		if record, ok := item.(map[string]any); ok {
			records = append(records, record)
		}
	}
	if !listed || len(records) != len(items) {
		problems = append(problems, "потенциал не список записей")
	} else {
		given := []string{}
		for _, record := range records {
			name, _ := record["area"].(string)
			given = append(given, name)
		}
		switch {
		case !slices.Equal(slices.Sorted(slices.Values(given)), slices.Sorted(slices.Values(areas))):
			problems = append(problems, fmt.Sprintf("направления не те: %v", given))
		case !slices.Equal(given, areas):
			problems = append(problems, "направления в другом порядке")
		}
		for _, record := range records {
			name, _ := record["area"].(string)
			for _, key := range []string{"level", "facts", "conclusion"} {
				if text, ok := record[key].(string); !ok || strings.TrimSpace(text) == "" {
					problems = append(problems, name+": пустое поле "+key)
				}
			}
			if want, known := expected[name]; known && record["level"] != want {
				problems = append(problems, fmt.Sprintf("%s: уровень изменён, «%v» вместо «%s»", name, record["level"], want))
			}
		}
	}
	attention, ok := output["attention"].(map[string]any)
	if !ok {
		return append(problems, "нет блока «на что обратить внимание»")
	}
	for _, key := range []string{"use", "consider"} {
		list, isList := attention[key].([]any)
		strs := true
		for _, item := range list {
			if _, isString := item.(string); !isString {
				strs = false
			}
		}
		switch {
		case !isList || !strs:
			problems = append(problems, "attention."+key+" не список строк")
		case len(list) > attentionLimit:
			problems = append(problems, "attention."+key+": пунктов больше пяти")
		}
		if key == "consider" && isList && strs {
			problems = append(problems, considerProblems(list, payload)...)
		}
	}
	return problems
}

// considerProblems checks the points of «what to consider» against the rules
// of the instruction. A problem names the point by its number and never quotes
// its text: the problems go to the journal, the answer must not.
func considerProblems(points []any, payload *object) []string {
	problems := []string{}
	details := false
	for n, point := range points {
		text := point.(string)
		if found := consequence.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("attention.consider, пункт %d: вывод вместо факта («%s»)", n+1, strings.ToLower(found)))
		}
		details = details || criminalDetails.MatchString(text)
	}
	if record, _ := payload.values["уголовная_ответственность"].(string); record == "привлекался" && !details {
		problems = append(problems, "attention.consider: нет «подробности — в ответах студента»")
	}
	return problems
}

// testPortraitTask writes the portrait of the first chapter by a finished test
// session with the methods: the levels are counted by methods.go, the model
// only writes the text (Вики/Тестирование/Инструкция.md).
const testPortraitTask = "test"

func findSession(db model.Database, s Source) (model.Session, bool) {
	if s.Kind != "session" {
		return model.Session{}, false
	}
	for _, record := range db.Sessions {
		if record.Session.ID == s.ID {
			return record.Session, true
		}
	}
	return model.Session{}, false
}

func (a *Service) testTask(prompt promptbank.Prompt) portraitTask {
	return portraitTask{
		ID: testPortraitTask, Version: prompt.Version, Prompt: prompt,
		// The portrait has nine sections: about 2000 tokens of Russian text.
		// More rules than in the autobiography: more tries, each with the notes.
		MaxTokens: 4000, Attempts: 5,
		Accepts: func(db model.Database, s Source) bool {
			session, ok := findSession(db, s)
			return ok && session.Completed && session.Methods && a.TestPortrait(session) != nil
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) {
			session, _ := findSession(db, s)
			data := a.TestPortrait(session)
			if data == nil {
				return nil, problem(Invalid, "В тестировании нет методик.")
			}
			return testPayload(data), nil
		},
		Check: checkTestAnswer,
		Visible: func(db model.Database, s Source, teacherID string) bool {
			session, ok := findSession(db, s)
			return ok && teacherOwnsSession(db, teacherID, session)
		},
	}
}

// cause finds the words of a cause the user forbade on 22 October 2026: the
// portrait shows how things go together, not what influenced what. Checked in
// every section but support and growth.
var cause = regexp.MustCompile(`(?i)повлия|благодаря|привел[оаи]? к|привело к|в результате|под влиянием|обусловл`)

// influence finds softer verbs of influence; they are checked only in link,
// the section about how conditions, activity and the self go together.
var influence = regexp.MustCompile(`(?i)ограничива|сдержива|усилива|способству|снижа[ею]т`)

// diagnosis finds clinical words and conclusions about fitness or beliefs.
var diagnosis = regexp.MustCompile(`(?i)диагноз|расстройств|патолог|непригод|пригоден к|лояльн|к сожалению`)

// shortLevel finds a level put on a 1–5 score by a short adjective
// («коллективизм низок»). Adverbs («высоко оценивает») retell the student's own
// rating and pass; the full forms name the levels of the methods and are
// checked against them.
var shortLevel = regexp.MustCompile(`(?i)(^|[^а-яё])(низок|низка|низки|высок|высока|высоки)([^а-яё]|$)`)

var levelWord = regexp.MustCompile(`(?i)низк|средн|высок`)

func objectList(value any) []*object {
	list, _ := value.([]any)
	result := []*object{}
	for _, item := range list {
		if o, ok := item.(*object); ok {
			result = append(result, o)
		}
	}
	return result
}

func objectText(o *object, key string) string {
	text, _ := o.values[key].(string)
	return text
}

// checkTestAnswer returns what is wrong with an answer to the test
// instruction. A problem names the section, never quotes the answer.
func checkTestAnswer(output map[string]any, payload *object) []string {
	problems := []string{}
	sections := map[string]string{}
	for _, key := range []string{"portrait", "signs", "images", "environment", "link"} {
		text, _ := output[key].(string)
		if strings.TrimSpace(text) == "" {
			problems = append(problems, "нет раздела "+key)
		}
		sections[key] = text
	}
	core, _ := output["core"].(map[string]any)
	for _, key := range []string{"cs", "ev", "dp"} {
		text, _ := core[key].(string)
		if strings.TrimSpace(text) == "" {
			problems = append(problems, "нет раздела core."+key)
		}
		sections["core."+key] = text
	}
	kinds := objectList(payload.values["виды_деятельности"])
	items, _ := output["activity"].([]any)
	if len(items) != len(kinds) {
		problems = append(problems, fmt.Sprintf("видов деятельности %d вместо %d", len(items), len(kinds)))
	}
	for n, item := range items {
		record, _ := item.(map[string]any)
		text, _ := record["text"].(string)
		sections[fmt.Sprintf("activity.%d", n+1)] = text
		if n >= len(kinds) {
			continue
		}
		kind, _ := record["kind"].(string)
		level, _ := record["level"].(string)
		want := kinds[n]
		switch {
		case !strings.EqualFold(strings.TrimSpace(kind), objectText(want, "вид")):
			problems = append(problems, fmt.Sprintf("activity.%d: не тот вид или порядок", n+1))
		case strings.TrimSpace(level) != objectText(want, "уровень"):
			problems = append(problems, fmt.Sprintf("activity.%d: уровень изменён", n+1))
		case strings.TrimSpace(text) == "":
			problems = append(problems, fmt.Sprintf("activity.%d: нет текста", n+1))
		}
	}
	for _, key := range []string{"support", "growth"} {
		list, ok := output[key].([]any)
		strs := ok && len(list) > 0 && len(list) <= attentionLimit
		for _, item := range list {
			if text, isText := item.(string); !isText || strings.TrimSpace(text) == "" {
				strs = false
			} else if found := diagnosis.FindString(text); found != "" {
				problems = append(problems, fmt.Sprintf("%s: недопустимое слово («%s»)", key, strings.ToLower(found)))
			}
		}
		if !strs {
			problems = append(problems, key+": нужно от 1 до 5 пунктов")
		}
	}
	keys := make([]string, 0, len(sections))
	for key := range sections {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		text := sections[key]
		if found := cause.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: причина вместо сочетания («%s»)", key, strings.ToLower(found)))
		}
		if found := influence.FindString(text); key == "link" && found != "" {
			problems = append(problems, fmt.Sprintf("%s: влияние вместо сочетания («%s»)", key, strings.ToLower(found)))
		}
		if found := diagnosis.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: недопустимое слово («%s»)", key, strings.ToLower(found)))
		}
		// Only these sections speak of 1–5 scores alone; elsewhere the levels of
		// the methods and the questionnaire are named rightly.
		if key == "signs" || key == "environment" {
			if found := shortLevel.FindStringSubmatch(text); found != nil {
				problems = append(problems, fmt.Sprintf("%s: уровень у балла 1–5 («%s»)", key, strings.ToLower(found[2])))
			}
		}
	}
	// A level written right after the name of a method must be its own.
	for _, method := range objectList(payload.values["методики"]) {
		name, want := strings.ToLower(objectText(method, "показатель")), objectText(method, "уровень")
		if name == "" || want == "не определён" {
			continue
		}
		for _, key := range []string{"portrait", "core.cs", "core.ev", "core.dp"} {
			text := strings.ToLower(sections[key])
			for at := strings.Index(text, name); at >= 0; {
				tail := []rune(text[at+len(name):])
				window := string(tail[:min(len(tail), 40)])
				if found := levelWord.FindString(window); found != "" && !strings.HasPrefix(want, found) {
					problems = append(problems, fmt.Sprintf("%s: уровень показателя «%s» не тот", key, objectText(method, "показатель")))
					break
				}
				next := strings.Index(text[at+len(name):], name)
				if next < 0 {
					break
				}
				at += len(name) + next
			}
		}
	}
	return problems
}
