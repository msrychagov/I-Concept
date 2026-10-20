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
