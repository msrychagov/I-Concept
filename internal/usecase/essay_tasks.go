package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// The portrait task «essay»: the analysis of one submitted essay by the four
// moments of the self-concept. The instruction, the schema and what the program
// checks are in Вики/Эссе выпускника/Инструкция.md.

// essayTask takes a submitted essay. The patronymic is not cut out: the model
// must tell «Игорь Петрович» from the instructors of the list.
func (a *Service) essayTask(prompt promptbank.Prompt) portraitTask {
	// Four moments, three lists, teachers, a comparison and a conclusion, with
	// a quote under each point: more than the autobiography; tries are many
	// because a quote must be word for word. 5000 tokens cut off the JSON of a
	// long answer on the first run on GigaChat. The paragraphs of v2 are half as
	// long (essay_v2.go).
	check, clean, tokens := checkEssayAnswer, cleanEssayAnswer, 8000
	if essayShape(prompt.Version) >= 2 {
		check, clean, tokens = checkEssayAnswerV2, cleanEssayAnswerV2, 6000
	}
	return portraitTask{
		ID: essayForm, Version: prompt.Version, Prompt: prompt,
		MaxTokens: tokens, Attempts: 5,
		Keep: []string{"отчество"},
		Accepts: func(db model.Database, s Source) bool {
			submission, ok := findSubmission(db, s)
			return ok && submission.FormID == essayForm && submission.Completed
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) {
			submission, _ := findSubmission(db, s)
			return a.essayPayload(db, submission)
		},
		Clean: clean,
		Check: check,
		Visible: func(db model.Database, s Source, teacherID string) bool {
			submission, ok := findSubmission(db, s)
			return ok && teacherOwnsSubmission(db, teacherID, submission)
		},
	}
}

const (
	essayPointsMost = 5
	essayQuoteWords = 30
)

var (
	essayAbout = []string{"о себе", "о ВУЦ"}
	essayBasis = []string{"сказано автором", "вывод по тексту"}
	essayKinds = []string{"благодарность", "проблема", "упоминание"}
	// essayMoments are the names of the moments in the field «change», in order.
	essayMoments = []string{"убеждения", "эмоции", "ценности", "восприятие своих возможностей"}
	momentKeys   = []string{"beliefs", "emotions", "values", "abilities"}
)

// essayForbidden finds a clinical word, a level (the essay analysis has none)
// and the phrase the instruction forbids. Quotes are not checked: they are the
// words of the author.
var essayForbidden = regexp.MustCompile(`(?i)диагноз|расстройств|патолог|непригодн|к сожалению|(высок|средн|низк)\p{L}*\s+уров`)

var instructorLabelText = regexp.MustCompile(`^\[преподаватель \d+\]$`)

// quoteSpelling makes quotes of different shapes the same.
var quoteSpelling = strings.NewReplacer("«", `"`, "»", `"`, "„", `"`, "“", `"`, "”", `"`, "‟", `"`, "’", `"`, "‘", `"`, "'", `"`, "ё", "е")

// quoteForm brings a text to the form in which a quote is looked up in it: no
// extra spaces, no case, «ё» as «е», one shape of quotes.
func quoteForm(text string) string {
	return strings.Join(strings.Fields(quoteSpelling.Replace(strings.ToLower(text))), " ")
}

// essayAnswers returns the answers of an essay in the input, by question number,
// in the form quotes are looked up in. key is «эссе» or «прежнее_эссе».
func essayAnswers(payload *object, key string) map[string]string {
	answers := map[string]string{}
	for _, row := range objectList(payload.values[key]) {
		answers[objectText(row, "номер")] = quoteForm(objectText(row, "ответ"))
	}
	return answers
}

func asMap(value any) map[string]any {
	m, _ := value.(map[string]any)
	return m
}

func asList(value any) ([]any, bool) {
	list, ok := value.([]any)
	return list, ok
}

func textOf(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

// quoteProblems checks one quote {question, quote}: its length and that it is
// in the answer with that number. A problem names the place and the number of
// the question and never repeats the quote.
func quoteProblems(where string, raw any, source map[string]string) []string {
	quote := asMap(raw)
	number, text := textOf(quote["question"]), textOf(quote["quote"])
	if quote == nil || number == "" || text == "" {
		return []string{where + ": цитата без номера вопроса или текста"}
	}
	if words := len(strings.Fields(text)); words < 3 || words > essayQuoteWords {
		return []string{fmt.Sprintf("%s: в цитате к вопросу %s %d слов, нужно от 3 до %d", where, number, words, essayQuoteWords)}
	}
	answer, ok := source[number]
	if !ok {
		return []string{fmt.Sprintf("%s: нет ответа с номером %s", where, number)}
	}
	if !strings.Contains(answer, quoteForm(text)) {
		return []string{fmt.Sprintf("%s: цитаты нет в ответе %s дословно", where, number)}
	}
	return nil
}

// quotesProblems checks a list of low to high quotes.
func quotesProblems(where string, raw any, source map[string]string, low, high int) []string {
	list, ok := asList(raw)
	if !ok || len(list) < low || len(list) > high {
		return []string{fmt.Sprintf("%s: цитат должно быть от %d до %d", where, low, high)}
	}
	problems := []string{}
	for _, quote := range list {
		problems = append(problems, quoteProblems(where, quote, source)...)
	}
	return problems
}

// wordProblems finds the forbidden words in a text written by the model.
func wordProblems(where, text string) []string {
	if found := essayForbidden.FindString(text); found != "" {
		return []string{fmt.Sprintf("%s: недопустимое слово («%s»)", where, strings.ToLower(found))}
	}
	return nil
}

// textProblems checks a field that must hold a text and has no forbidden word.
func textProblems(where string, raw any) []string {
	text := textOf(raw)
	if text == "" {
		return []string{where + ": пустой текст"}
	}
	return wordProblems(where, text)
}

func oneOf(value any, list []string) bool { return slices.Contains(list, textOf(value)) }

// checkEssayAnswer returns what is wrong with an answer to the essay
// instruction: the schema, the number of points, the lists of values, that
// every quote is word for word in the answer it names, that the label of an
// instructor is in the list and the forbidden words.
func checkEssayAnswer(output map[string]any, payload *object) []string {
	current, previous := essayAnswers(payload, "эссе"), essayAnswers(payload, "прежнее_эссе")
	problems := textProblems("summary", output["summary"])

	moments := asMap(output["moments"])
	for _, key := range momentKeys {
		where := "moments." + key
		moment := asMap(moments[key])
		if moment == nil {
			problems = append(problems, where+": нет момента")
			continue
		}
		problems = append(problems, textProblems(where, moment["text"])...)
		problems = append(problems, quotesProblems(where, moment["quotes"], current, 1, 3)...)
	}

	for _, list := range []struct {
		key      string
		low, top int
	}{{"positive", 1, essayPointsMost}, {"negative", 0, essayPointsMost}, {"attention", 0, essayPointsMost}} {
		points, ok := asList(output[list.key])
		if !ok || len(points) < list.low || len(points) > list.top {
			problems = append(problems, fmt.Sprintf("%s: пунктов должно быть от %d до %d", list.key, list.low, list.top))
			continue
		}
		for n, raw := range points {
			where, point := fmt.Sprintf("%s, пункт %d", list.key, n+1), asMap(raw)
			if point == nil {
				problems = append(problems, where+": не запись")
				continue
			}
			problems = append(problems, textProblems(where, point["text"])...)
			if !oneOf(point["about"], essayAbout) {
				problems = append(problems, where+": about должно быть «о себе» или «о ВУЦ»")
			}
			if !oneOf(point["basis"], essayBasis) {
				problems = append(problems, where+": basis должно быть «сказано автором» или «вывод по тексту»")
			}
			problems = append(problems, quotesProblems(where, point["quotes"], current, 1, 3)...)
		}
	}

	problems = append(problems, teacherProblems(output, payload, current)...)

	changes, ok := asList(output["change"])
	switch {
	case !ok:
		problems = append(problems, "change: нет списка")
	case len(previous) == 0 && len(changes) != 0:
		problems = append(problems, "change: прежнего эссе нет, список должен быть пустым")
	case len(previous) > 0 && len(changes) != len(essayMoments):
		problems = append(problems, fmt.Sprintf("change: нужно %d записи по порядку, получено %d", len(essayMoments), len(changes)))
	}
	if len(previous) > 0 && len(changes) == len(essayMoments) {
		for n, raw := range changes {
			where, record := fmt.Sprintf("change, запись %d", n+1), asMap(raw)
			if record == nil || textOf(record["moment"]) != essayMoments[n] {
				problems = append(problems, where+": момент не тот или не по порядку, нужно «"+essayMoments[n]+"»")
				continue
			}
			problems = append(problems, textProblems(where, record["text"])...)
			problems = append(problems, quoteProblems(where+", before", record["before"], previous)...)
			problems = append(problems, quoteProblems(where+", after", record["after"], current)...)
		}
	}

	conclusion := asMap(output["conclusion"])
	text := strings.ToLower(textOf(conclusion["text"]))
	problems = append(problems, textProblems("conclusion", conclusion["text"])...)
	if text != "" && !strings.HasPrefix(text, "по ответам видно") && !strings.HasPrefix(text, "по ответам можно предположить") {
		problems = append(problems, "conclusion: должен начинаться со слов «по ответам видно» или «по ответам можно предположить»")
	}
	return append(problems, quotesProblems("conclusion", conclusion["quotes"], current, 1, 3)...)
}

// teacherProblems checks the list «teachers», the same in every version: a
// label only from the list and once, the kind from its list, a text and one or
// two quotes of the new essay; an entry the program took the label off has the
// words of the author and at least two candidates of the list.
func teacherProblems(output map[string]any, payload *object, current map[string]string) []string {
	labels := []string{}
	for _, entry := range objectList(payload.values["преподаватели"]) {
		labels = append(labels, objectText(entry, "метка"))
	}
	problems := []string{}
	teachers, ok := asList(output["teachers"])
	if !ok {
		problems = append(problems, "teachers: нет списка")
	}
	seen := []string{}
	for n, raw := range teachers {
		where, record := fmt.Sprintf("teachers, запись %d", n+1), asMap(raw)
		if record == nil {
			problems = append(problems, where+": не запись")
			continue
		}
		who := textOf(record["who"])
		// An entry the program took the label off lists the labels of those who fit.
		if candidates, has := record["candidates"]; has {
			list, _ := asList(candidates)
			for _, item := range list {
				if label := textOf(item); !slices.Contains(labels, label) {
					problems = append(problems, where+": кандидат не из списка преподавателей")
				}
			}
			if len(list) < 2 || who == "" || strings.Contains(who, "[преподаватель") {
				problems = append(problems, where+": у записи с кандидатами нужны слова автора и не меньше двух кандидатов")
			}
		}
		switch {
		case who == "":
			problems = append(problems, where+": не сказано, кто")
		case strings.Contains(who, "[преподаватель"):
			switch {
			case !instructorLabelText.MatchString(who) || !slices.Contains(labels, who):
				problems = append(problems, where+": метка не из списка преподавателей")
			case slices.Contains(seen, who):
				problems = append(problems, where+": метка уже есть в другой записи")
			default:
				seen = append(seen, who)
			}
		}
		if !oneOf(record["kind"], essayKinds) {
			problems = append(problems, where+": kind должно быть «благодарность», «проблема» или «упоминание»")
		}
		problems = append(problems, textProblems(where, record["text"])...)
		problems = append(problems, quotesProblems(where, record["quotes"], current, 1, 2)...)
	}
	return problems
}

// queueEssayPortraits queues the analyses of the submitted essays of a center
// again: the instructor list is a part of their input, and after a change it
// is another input.
func (a *Service) queueEssayPortraits(vuc string) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return
	}
	for _, submission := range db.Submissions {
		if submission.FormID != essayForm || !submission.Completed {
			continue
		}
		if student, ok := studentOf(db, submission); ok && sameVUC(student.VUC, vuc) {
			a.queuePortraits(Source{"submission", submission.ID})
		}
	}
}
