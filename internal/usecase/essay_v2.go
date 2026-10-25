package usecase

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// The second version of the essay tasks (decided 25 October 2026 after the
// remarks of the supervisor): the analysis of one essay and the summary of a
// cohort are three paragraphs of conclusions and two to four recommendations,
// not more than 250 words, with no lists, no retelling and no numbers of
// questions. The quotes and the references stay: the teacher opens them by the
// button «Из чего вывод». The instructions, the schemas and what the program
// checks are in Вики/Эссе выпускника/Инструкция v2.md and Инструкция по
// выпуску v2.md. The input of the analysis did not change.

// essayParagraphs are the keys of the paragraphs of v2, in the order they are shown.
var essayParagraphs = []string{"formation", "moments", "environment"}

const (
	// essayWordsMost is the limit of the paragraphs and the recommendations
	// together, quotes not counted (decision В1).
	essayWordsMost   = 250
	essayAdviceLeast = 2
	essayAdviceMost  = 4
)

// essayShape is the number of the version of an essay instruction whose answer
// a stored output follows: 1 for v1, 2 for v2 and later.
func essayShape(version string) int {
	if n, _ := promptbank.Number(version); n >= 2 {
		return 2
	}
	return 1
}

// countWords counts the pieces between spaces that hold a letter or a digit: a
// dash is not a word.
func countWords(text string) int {
	n := 0
	for _, piece := range strings.Fields(text) {
		if strings.IndexFunc(piece, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
			n++
		}
	}
	return n
}

// v2Forbidden finds in a text of v2 what the teacher must not read there: the
// number of a question (the quotes are behind a button), a label of an
// instructor (the text names him as the author did) and the terms the text must
// tell by their content.
var (
	questionNumber = regexp.MustCompile(`(?i)\d+\.\d+|вопрос\p{L}*\s*№?\s*\d`)
	// essayTerms are the words of the theory the text tells by their content
	// (decision Д5): «Я-концепция», «Я-реальное», «Я-идеальное», «рефлексия».
	// «я» stands alone, or «армия реально» would be one.
	essayTerms = regexp.MustCompile(`(?i)(^|[^\p{L}])(я[-‐‑–—]?(концепц|реальн|идеальн)\p{L}*)|рефлекси\p{L}*`)
)

// termProblems finds a term of the theory in a text of v2.
func termProblems(where, text string) []string {
	found := essayTerms.FindStringSubmatch(text)
	if found == nil {
		return nil
	}
	word := found[2]
	if word == "" {
		word = found[0]
	}
	return []string{fmt.Sprintf("%s: термин «%s» — пиши его содержание обычными словами", where, strings.ToLower(word))}
}

// v2TextProblems checks a text of the analysis of v2: not empty, none of the
// words forbidden in v1, no number of a question (the quotes are behind a
// button), no label of an instructor (the text names him as the author did) and
// no term. A problem names the kind of word and never the text.
func v2TextProblems(where string, raw any) []string {
	problems := textProblems(where, raw)
	text := textOf(raw)
	if questionNumber.MatchString(text) {
		problems = append(problems, where+": номер вопроса в тексте")
	}
	if strings.Contains(text, "[преподаватель") {
		problems = append(problems, where+": метка преподавателя в тексте, называй его, как автор")
	}
	return append(problems, termProblems(where, text)...)
}

// sameAdvice finds a recommendation that repeats an earlier one word for word.
func sameAdvice(list []any) []string {
	problems, seen := []string{}, map[string]int{}
	for n, raw := range list {
		text := quoteForm(textOf(asMap(raw)["text"]))
		if text == "" {
			continue
		}
		if first, ok := seen[text]; ok {
			problems = append(problems, fmt.Sprintf("recommendations, пункт %d: повторяет пункт %d", n+1, first))
			continue
		}
		seen[text] = n + 1
	}
	return problems
}

// tooLong is the problem of an answer over the limit of words.
func tooLong(words int) []string {
	if words > essayWordsMost {
		return []string{fmt.Sprintf("в абзацах и рекомендациях вместе %d слов, нужно не больше %d: сократи", words, essayWordsMost)}
	}
	return nil
}

// checkEssayAnswerV2 returns what is wrong with an answer to the instruction
// essay/v2: three paragraphs with their quotes, the quotes of the earlier essay
// only when there is one, two to four recommendations with one or two quotes,
// not more than 250 words, the words a text must not hold and the teachers as
// in v1.
func checkEssayAnswerV2(output map[string]any, payload *object) []string {
	current, previous := essayAnswers(payload, "эссе"), essayAnswers(payload, "прежнее_эссе")
	problems, words := []string{}, 0
	for _, p := range []struct {
		key       string
		low, high int
	}{{"formation", 1, 3}, {"moments", 2, 4}, {"environment", 1, 3}} {
		paragraph := asMap(output[p.key])
		if paragraph == nil {
			problems = append(problems, p.key+": нет абзаца")
			continue
		}
		problems = append(problems, v2TextProblems(p.key, paragraph["text"])...)
		problems = append(problems, quotesProblems(p.key, paragraph["quotes"], current, p.low, p.high)...)
		words += countWords(textOf(paragraph["text"]))
	}
	if formation := asMap(output["formation"]); formation != nil {
		before, ok := asList(formation["before"])
		switch {
		case !ok:
			problems = append(problems, "formation.before: нет списка")
		case len(previous) == 0 && len(before) > 0:
			problems = append(problems, "formation.before: прежнего эссе нет, список должен быть пустым")
		case len(previous) > 0:
			problems = append(problems, quotesProblems("formation.before", before, previous, 1, 2)...)
		}
	}

	advice, ok := asList(output["recommendations"])
	if !ok || len(advice) < essayAdviceLeast || len(advice) > essayAdviceMost {
		problems = append(problems, fmt.Sprintf("recommendations: рекомендаций должно быть от %d до %d", essayAdviceLeast, essayAdviceMost))
	}
	for n, raw := range advice {
		where, item := fmt.Sprintf("recommendations, пункт %d", n+1), asMap(raw)
		if item == nil {
			problems = append(problems, where+": не запись")
			continue
		}
		problems = append(problems, v2TextProblems(where, item["text"])...)
		problems = append(problems, quotesProblems(where, item["quotes"], current, 1, 2)...)
		words += countWords(textOf(item["text"]))
	}
	problems = append(problems, sameAdvice(advice)...)
	problems = append(problems, tooLong(words)...)
	return append(problems, teacherProblems(output, payload, current)...)
}

// ---- the summary of a cohort, v2 ------------------------------------------

// cohortV2Kinds are the kinds of reference a recommendation of the summary may
// stand on: any paragraph or recommendation of an analysis.
var cohortV2Kinds = kindIs("formation", "moments", "environment", "rec")

// cohortPayloadV2 is the input of the summary of v2: the paragraphs and the
// recommendations of the analyses of v2, every one under its number, and
// whether the author wrote an essay before. Quotes and teachers do not go.
func (a *Service) cohortPayloadV2(db model.Database, assignmentID string) (*object, error) {
	authors := a.cohortAuthorsOf(db, assignmentID, 2)
	if len(authors) < cohortMinimum {
		return nil, problem(Invalid, "Для сводки нужно хотя бы два эссе с разбором.")
	}
	payload := newObject()
	reviews := []any{}
	withPrevious := 0
	for n, author := range authors {
		id := strconv.Itoa(n + 1)
		before, _ := asList(asMap(author.Analysis["formation"])["before"])
		if len(before) > 0 {
			withPrevious++
		}
		review := newObject()
		review.set("автор", id)
		review.set("прежнее_эссе", len(before) > 0)
		paragraphs := []any{}
		for _, key := range essayParagraphs {
			entry := newObject()
			entry.set("номер", id+"."+key)
			entry.set("текст", textOf(asMap(author.Analysis[key])["text"]))
			paragraphs = append(paragraphs, entry)
		}
		review.set("абзацы", paragraphs)
		advice := []any{}
		for i, raw := range listOf(author.Analysis, "recommendations") {
			entry := newObject()
			entry.set("номер", fmt.Sprintf("%s.rec.%d", id, i+1))
			entry.set("текст", textOf(asMap(raw)["text"]))
			advice = append(advice, entry)
		}
		review.set("рекомендации", advice)
		reviews = append(reviews, review)
	}
	payload.set("эссе_в_сводке", len(authors))
	payload.set("с_прежним_эссе", withPrevious)
	payload.set("разборы", reviews)
	return payload, nil
}

// cleanCohortAnswerV2 drops what the program can drop without asking again, as
// in v1: references that are not in the input, of the wrong kind or repeated,
// and the recommendations left with fewer than two authors. A paragraph keeps
// its place: left without authors, it is refused by the check.
func cleanCohortAnswerV2(_ *Service, _ model.Database, _ Source, output map[string]any, payload *object) (map[string]any, int) {
	known := payloadRefs(payload)
	changes := 0
	for _, key := range essayParagraphs {
		if paragraph := asMap(output[key]); paragraph != nil {
			refs, dropped := cleanRefs(paragraph["refs"], known, kindIs(key))
			paragraph["refs"], changes = refs, changes+dropped
		}
	}
	if advice, ok := asList(output["recommendations"]); ok {
		kept := []any{}
		for _, raw := range advice {
			item := asMap(raw)
			if item == nil {
				kept = append(kept, raw)
				continue
			}
			refs, dropped := cleanRefs(item["refs"], known, cohortV2Kinds)
			item["refs"], changes = refs, changes+dropped
			if len(refAuthors(refs)) < 2 {
				changes++
				continue
			}
			kept = append(kept, item)
		}
		output["recommendations"] = kept
	}
	return output, changes
}

// checkCohortAnswerV2 returns what is wrong with an answer to the instruction
// essay-cohort/v2 after the cleaning: each paragraph with a text and at least
// two authors, one to four recommendations with two authors each, not more
// than 250 words, no words of quantity, digits, labels or terms.
func checkCohortAnswerV2(output map[string]any, payload *object) []string {
	known := payloadRefs(payload)
	problems, words := []string{}, 0
	text := func(where string, raw any) {
		problems = append(problems, cohortTextProblems(where, raw)...)
		problems = append(problems, termProblems(where, textOf(raw))...)
		words += countWords(textOf(raw))
	}
	for _, key := range essayParagraphs {
		paragraph := asMap(output[key])
		if paragraph == nil {
			problems = append(problems, key+": нет абзаца")
			continue
		}
		text(key, paragraph["text"])
		more, _ := refsOf(key, paragraph["refs"], known, kindIs(key), 2)
		problems = append(problems, more...)
	}
	advice, ok := asList(output["recommendations"])
	if !ok || len(advice) == 0 || len(advice) > essayAdviceMost {
		problems = append(problems, fmt.Sprintf("recommendations: рекомендаций должно быть от %d до %d, у каждой ссылки на двух и более авторов", essayAdviceLeast, essayAdviceMost))
	}
	for n, raw := range advice {
		where, item := fmt.Sprintf("recommendations, пункт %d", n+1), asMap(raw)
		if item == nil {
			problems = append(problems, where+": не запись")
			continue
		}
		text(where, item["text"])
		more, _ := refsOf(where, item["refs"], known, cohortV2Kinds, 2)
		problems = append(problems, more...)
	}
	problems = append(problems, sameAdvice(advice)...)
	return append(problems, tooLong(words)...)
}
