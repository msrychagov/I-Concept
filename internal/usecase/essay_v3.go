package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// The third version of the essay tasks (decided 5 November 2026 after the
// remarks of the supervisor): the conclusion is written as a military
// psychologist would, in general words; instead of the recommendations it has
// what the center brought up in the author (developed) and what to strengthen,
// and a pedagogical conclusion about the teachers by their positions. The
// instructions, the schemas and what the program checks are in Вики/Эссе
// выпускника/Инструкция v3.md and Инструкция по выпуску v3.md. The input of
// the analysis did not change, so an analysis of v2 is shown while the one of
// v3 is made.

const (
	essayDevelopedMost   = 3
	essayStrengthenLeast = 2
	essayStrengthenMost  = 3
	essayTeachersMost    = 3
)

// exampleWholeLeast is the fewest words of an example «так» of an instruction
// whose copy is refused: a short phrase of the supervisor («опирается на
// приобретённый опыт…», 8 words) may be repeated (decision Д6).
const exampleWholeLeast = 10

// essayExamplesV3 and cohortExamplesV3 are the examples «так» of the
// instructions; the test checks they are those of the embedded texts.
var (
	essayExamplesV3 = []string{
		"Опирается на приобретённый опыт, умеет воспринимать другую точку зрения",
		"За время обучения стал психологически устойчивее",
		"Уверенность в собственных решениях — давать задачи, где решение нужно принять самому",
		"Преподавателю удалось завоевать доверие автора: с трудностями тот стал приходить открыто и решать их сразу",
	}
	cohortExamplesV3 = []string{
		"У авторов складываются ценностные установки на честность и взаимную поддержку, умение работать в команде, адекватная реакция на недостатки в ходе приобретения опыта военной подготовки",
		"Уверенность в своих силах — давать каждому опыт самостоятельных решений с разбором после",
	}
)

// copiesExample says whether a text holds a whole example of exampleWholeLeast
// words or more.
func copiesExample(text string, examples []string) bool {
	words := " " + strings.Join(wordsOf(text), " ") + " "
	for _, example := range examples {
		if list := wordsOf(example); len(list) >= exampleWholeLeast && strings.Contains(words, " "+strings.Join(list, " ")+" ") {
			return true
		}
	}
	return false
}

// essayText is a text of an answer of v3 with the place it is at.
type essayText struct {
	where string
	raw   any
}

// essayTextsV3 are the texts of an analysis or a summary of v3 the limit of
// words and the common checks go over: the paragraphs, developed, strengthen
// and the conclusion about the teachers.
func essayTextsV3(output map[string]any) []essayText {
	texts := []essayText{}
	for _, key := range essayParagraphs {
		if paragraph := asMap(output[key]); paragraph != nil {
			texts = append(texts, essayText{key, paragraph["text"]})
		}
	}
	for _, key := range []string{"developed", "strengthen"} {
		items, _ := asList(output[key])
		for n, raw := range items {
			if item := asMap(raw); item != nil {
				texts = append(texts, essayText{fmt.Sprintf("%s, пункт %d", key, n+1), item["text"]})
			}
		}
	}
	if conclusion := asMap(output["teachersConclusion"]); conclusion != nil && textOf(conclusion["text"]) != "" {
		texts = append(texts, essayText{"teachersConclusion", conclusion["text"]})
	}
	return texts
}

// samePoints finds a point of a list that repeats an earlier one word for word.
func samePoints(key string, list []any) []string {
	problems, seen := []string{}, map[string]int{}
	for n, raw := range list {
		text := quoteForm(textOf(asMap(raw)["text"]))
		if text == "" {
			continue
		}
		if first, ok := seen[text]; ok {
			problems = append(problems, fmt.Sprintf("%s, пункт %d: повторяет пункт %d", key, n+1, first))
			continue
		}
		seen[text] = n + 1
	}
	return problems
}

// teacherNameWords are the words of the names the analysis gives the
// teachers, in the form words are compared in; an initial is not one.
func teacherNameWords(output map[string]any) []string {
	words := []string{}
	teachers, _ := asList(output["teachers"])
	for _, raw := range teachers {
		_, normal := tokens(textOf(asMap(raw)["name"]))
		for _, word := range normal {
			if len([]rune(word)) >= 3 {
				words = append(words, word)
			}
		}
	}
	return words
}

// namesIn says whether a text holds a word of a name in any case.
func namesIn(text string, names []string) bool {
	_, normal := tokens(text)
	for _, token := range normal {
		for _, name := range names {
			if isWordOf(token, name) {
				return true
			}
		}
	}
	return false
}

// conclusionRank finds a rank in the conclusion about the teachers of an
// analysis. «Старшина» is not one here (decided 5 November 2026 after the
// run): the authors call so the one who leads the drill, «старшина Громов»,
// and the model kept it for his position; a surname is caught apart.
var conclusionRank = regexp.MustCompile(`(?i)(^|[^\p{L}])(лейтенант|капитан|майор|подполковник|полковник|генерал|сержант|прапорщик|ефрейтор)`)

// tooLongV3 is the problem of an answer of v3 over the limit of words.
func tooLongV3(words int) []string {
	if words > essayWordsMost {
		return []string{fmt.Sprintf("в абзацах, developed, strengthen и teachersConclusion вместе %d слов, нужно не больше %d: сократи", words, essayWordsMost)}
	}
	return nil
}

// checkEssayAnswerV3 returns what is wrong with an answer to the instruction
// essay/v3: three paragraphs with their quotes, the quotes of the earlier essay
// only when there is one, none to three points of what the center brought up
// and two or three of what to strengthen with one or two quotes each, the
// conclusion about the teachers only when the author names some, not more than
// 250 words, no names or ranks of the teachers in the texts, no copy of an
// example, the words a text must not hold and the teachers as in v2.
func checkEssayAnswerV3(output map[string]any, payload *object) []string {
	current, previous := essayAnswers(payload, "эссе"), essayAnswers(payload, "прежнее_эссе")
	problems := []string{}
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

	for _, list := range []struct {
		key       string
		low, high int
	}{{"developed", 0, essayDevelopedMost}, {"strengthen", essayStrengthenLeast, essayStrengthenMost}} {
		items, ok := asList(output[list.key])
		if !ok || len(items) < list.low || len(items) > list.high {
			problems = append(problems, fmt.Sprintf("%s: пунктов должно быть от %d до %d", list.key, list.low, list.high))
		}
		for n, raw := range items {
			where, item := fmt.Sprintf("%s, пункт %d", list.key, n+1), asMap(raw)
			if item == nil {
				problems = append(problems, where+": не запись")
				continue
			}
			problems = append(problems, v2TextProblems(where, item["text"])...)
			if list.key == "strengthen" && familyWords.MatchString(textOf(item["text"])) {
				problems = append(problems, where+": пункт опирается на семью автора — так нельзя (правило 5)")
			}
			problems = append(problems, quotesProblems(where, item["quotes"], current, 1, 2)...)
		}
		problems = append(problems, samePoints(list.key, items)...)
	}

	teachers, _ := asList(output["teachers"])
	conclusion := asMap(output["teachersConclusion"])
	switch {
	case conclusion == nil:
		problems = append(problems, "teachersConclusion: нет вывода")
	case len(teachers) == 0:
		if quotes, _ := asList(conclusion["quotes"]); textOf(conclusion["text"]) != "" || len(quotes) > 0 {
			problems = append(problems, "teachersConclusion: в teachers никого нет — text должен быть пустой строкой, quotes — пустым списком")
		}
	default:
		text := textOf(conclusion["text"])
		problems = append(problems, v2TextProblems("teachersConclusion", conclusion["text"])...)
		problems = append(problems, quotesProblems("teachersConclusion", conclusion["quotes"], current, 1, essayTeachersMost)...)
		if familyWords.MatchString(text) {
			problems = append(problems, "teachersConclusion: вывод опирается на семью автора — так нельзя (правило 5)")
		}
		if conclusionRank.MatchString(text) {
			problems = append(problems, "teachersConclusion: звание в тексте — называй преподавателя по должности: преподаватель, куратор взвода, начальник цикла")
		}
	}

	words, names := 0, teacherNameWords(output)
	for _, t := range essayTextsV3(output) {
		text := textOf(t.raw)
		words += countWords(text)
		if namesIn(text, names) {
			problems = append(problems, t.where+": фамилия или имя преподавателя в тексте — называй его по должности: преподаватель, куратор взвода, начальник цикла; фамилия — только в teachers")
		}
		if copiesExample(text, essayExamplesV3) {
			problems = append(problems, t.where+": повторяет пример из инструкции — пиши своими словами о своём авторе")
		}
		if len(previous) == 0 && earlierEssay.MatchString(text) {
			problems = append(problems, t.where+": прежнего эссе у автора нет — не пиши о нём")
		}
	}
	problems = append(problems, tooLongV3(words)...)
	return append(problems, teacherProblemsV2(output, current)...)
}

// trimTextsV3 takes the marks off the start of every text of an answer of v3
// and counts the texts it changed.
func trimTextsV3(output map[string]any) int {
	changes := 0
	trim := func(item map[string]any) {
		if text, ok := item["text"].(string); ok {
			if trimmed := strings.TrimLeft(text, leadingMarks); trimmed != text {
				item["text"] = trimmed
				changes++
			}
		}
	}
	for _, key := range slices.Concat(essayParagraphs, []string{"teachersConclusion"}) {
		if item := asMap(output[key]); item != nil {
			trim(item)
		}
	}
	for _, key := range []string{"developed", "strengthen"} {
		items, _ := asList(output[key])
		for _, raw := range items {
			if item := asMap(raw); item != nil {
				trim(item)
			}
		}
	}
	return changes
}

// cleanEssayAnswerV3 takes the marks off the start of the texts and cuts the
// quotes that are too long.
func cleanEssayAnswerV3(_ *Service, _ model.Database, _ Source, output map[string]any, payload *object) (map[string]any, int) {
	changes := trimTextsV3(output)
	if payload != nil {
		changes += trimLongQuotes(output, essayAnswers(payload, "эссе"), essayAnswers(payload, "прежнее_эссе"))
	}
	return output, changes
}

// trimLongQuotes cuts a quote of more than essayQuoteWords words that stands
// word for word in its answer down to its first essayQuoteWords words: it
// stays word for word (decided 5 November 2026 after the run, where the model
// quoted an answer of 33 words whole in 50 tries). A quote not found in the
// answer is left to the check. It returns the number of quotes cut.
func trimLongQuotes(output map[string]any, current, previous map[string]string) int {
	changes := 0
	cut := func(raw any, answers map[string]string) {
		for _, quote := range quoteList(raw) {
			text := textOf(quote["quote"])
			words := strings.Fields(text)
			if len(words) <= essayQuoteWords || !strings.Contains(answers[textOf(quote["question"])], quoteForm(text)) {
				continue
			}
			quote["quote"] = strings.Join(words[:essayQuoteWords], " ")
			changes++
		}
	}
	for _, key := range slices.Concat(essayParagraphs, []string{"teachersConclusion"}) {
		if item := asMap(output[key]); item != nil {
			cut(item["quotes"], current)
		}
	}
	if formation := asMap(output["formation"]); formation != nil {
		cut(formation["before"], previous)
	}
	for _, key := range []string{"developed", "strengthen", "teachers"} {
		items, _ := asList(output[key])
		for _, raw := range items {
			if item := asMap(raw); item != nil {
				cut(item["quotes"], current)
			}
		}
	}
	return changes
}

// ---- the summary of a cohort, v3 ------------------------------------------

var (
	// cohortDevelopedKinds, cohortStrengthenKinds are the kinds of reference a
	// point of the summary may stand on: its own kind of point and any paragraph.
	cohortDevelopedKinds  = kindIs("dev", "formation", "moments", "environment")
	cohortStrengthenKinds = kindIs("str", "formation", "moments", "environment")
)

// cohortPayloadV3 is the input of the summary of v3: the paragraphs, what the
// center brought up, what to strengthen and the conclusion about the teachers
// of every analysis of v3 under its number, and whether the author wrote an
// essay before. Quotes and the list of teachers do not go.
func (a *Service) cohortPayloadV3(db model.Database, assignmentID string) (*object, error) {
	authors := a.cohortAuthorsOf(db, assignmentID, 3)
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
		for _, list := range []struct{ key, short, name string }{{"developed", "dev", "воспитал_вуц"}, {"strengthen", "str", "усилить"}} {
			points := []any{}
			for i, raw := range listOf(author.Analysis, list.key) {
				entry := newObject()
				entry.set("номер", fmt.Sprintf("%s.%s.%d", id, list.short, i+1))
				entry.set("текст", textOf(asMap(raw)["text"]))
				points = append(points, entry)
			}
			review.set(list.name, points)
		}
		if text := textOf(asMap(author.Analysis["teachersConclusion"])["text"]); text != "" {
			entry := newObject()
			entry.set("номер", id+".teachers")
			entry.set("текст", text)
			review.set("о_преподавателях", entry)
		}
		reviews = append(reviews, review)
	}
	payload.set("эссе_в_сводке", len(authors))
	payload.set("с_прежним_эссе", withPrevious)
	payload.set("разборы", reviews)
	return payload, nil
}

// cleanCohortAnswerV3 drops what the program can drop without asking again, as
// in v2: references that are not in the input, of the wrong kind or repeated,
// and the points of developed and strengthen left with fewer than two authors.
// A paragraph and the conclusion about the teachers keep their place: left
// without authors, they are refused by the check.
func cleanCohortAnswerV3(_ *Service, _ model.Database, _ Source, output map[string]any, payload *object) (map[string]any, int) {
	known := payloadRefs(payload)
	changes := trimTextsV3(output)
	for _, key := range essayParagraphs {
		if paragraph := asMap(output[key]); paragraph != nil {
			refs, dropped := cleanRefs(paragraph["refs"], known, kindIs(key))
			paragraph["refs"], changes = refs, changes+dropped
		}
	}
	if conclusion := asMap(output["teachersConclusion"]); conclusion != nil {
		refs, dropped := cleanRefs(conclusion["refs"], known, kindIs("teachers"))
		conclusion["refs"], changes = refs, changes+dropped
	}
	for _, list := range []struct {
		key   string
		allow func(string) bool
	}{{"developed", cohortDevelopedKinds}, {"strengthen", cohortStrengthenKinds}} {
		items, ok := asList(output[list.key])
		if !ok {
			continue
		}
		kept := []any{}
		for _, raw := range items {
			item := asMap(raw)
			if item == nil {
				kept = append(kept, raw)
				continue
			}
			refs, dropped := cleanRefs(item["refs"], known, list.allow)
			item["refs"], changes = refs, changes+dropped
			if len(refAuthors(refs)) < 2 {
				changes++
				continue
			}
			kept = append(kept, item)
		}
		output[list.key] = kept
	}
	return output, changes
}

// openingWords find a sentence of the summary that opens with «Общее:» or
// «Расхождения —», the words of the text of 25 October the supervisor called
// bad (decision Д15): the summary is a connected text.
var openingWords = regexp.MustCompile(`(?i)(^|[.!?;]\s*)(общее\s*[:—–-]|расхождени\p{L}*\s*([:—–-]|в\s))`)

// cohortAuthorsWith counts the authors of the input that have a point of the kind.
func cohortAuthorsWith(known map[string]bool, kind string) int {
	authors := map[int]bool{}
	for ref := range known {
		if author, k, ok := parseRef(ref); ok && k == kind {
			authors[author] = true
		}
	}
	return len(authors)
}

// checkCohortAnswerV3 returns what is wrong with an answer to the instruction
// essay-cohort/v3 after the cleaning: each paragraph with a text and at least
// two authors, up to three points of developed, one to three of strengthen,
// the conclusion about the teachers when two authors or more have one, not
// more than 250 words, no words of quantity, digits, labels, terms, ranks,
// authors one by one, «Общее:» or a copy of an example.
func checkCohortAnswerV3(output map[string]any, payload *object) []string {
	known := payloadRefs(payload)
	problems := []string{}
	text := func(where string, raw any) {
		value := textOf(raw)
		problems = append(problems, cohortTextProblems(where, raw)...)
		problems = append(problems, essayTermProblems(where, value)...)
		if militaryRank.MatchString(value) {
			problems = append(problems, where+": звание в тексте — не называй конкретных преподавателей, о них отдельный блок")
		}
		if singlesOut(value) {
			problems = append(problems, where+": авторы перечислены по одному («один … другой», «первый … второй») — различия пиши группами: «у одних авторов …, у других …», без перечня авторов")
		}
		if openingWords.MatchString(value) {
			problems = append(problems, where+": предложение начинается с «Общее:» или «Расхождения —» — пиши связным текстом: «у авторов складываются …», «одни …, другие …»")
		}
		if copiesExample(value, cohortExamplesV3) {
			problems = append(problems, where+": повторяет пример из инструкции — пиши своими словами о своём выпуске")
		}
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
	for _, list := range []struct {
		key       string
		low, high int
		allow     func(string) bool
	}{{"developed", 0, essayDevelopedMost, cohortDevelopedKinds}, {"strengthen", 1, essayStrengthenMost, cohortStrengthenKinds}} {
		items, ok := asList(output[list.key])
		if !ok || len(items) < list.low || len(items) > list.high {
			problems = append(problems, fmt.Sprintf("%s: пунктов должно быть от %d до %d, у каждого ссылки на двух и более авторов", list.key, list.low, list.high))
		}
		for n, raw := range items {
			where, item := fmt.Sprintf("%s, пункт %d", list.key, n+1), asMap(raw)
			if item == nil {
				problems = append(problems, where+": не запись")
				continue
			}
			text(where, item["text"])
			more, _ := refsOf(where, item["refs"], known, list.allow, 2)
			problems = append(problems, more...)
		}
		problems = append(problems, samePoints(list.key, items)...)
	}
	conclusion := asMap(output["teachersConclusion"])
	switch {
	case conclusion == nil:
		problems = append(problems, "teachersConclusion: нет вывода")
	case cohortAuthorsWith(known, "teachers") < 2:
		if textOf(conclusion["text"]) != "" {
			problems = append(problems, "teachersConclusion: вывод о преподавателях есть меньше чем у двух авторов — text должен быть пустой строкой, refs — пустым списком")
		}
	default:
		text("teachersConclusion", conclusion["text"])
		more, _ := refsOf("teachersConclusion", conclusion["refs"], known, kindIs("teachers"), 2)
		problems = append(problems, more...)
	}
	words, earlier := 0, cohortHasEarlier(payload)
	for _, t := range essayTextsV3(output) {
		words += countWords(textOf(t.raw))
		if !earlier && cohortEarlierWords.MatchString(textOf(t.raw)) {
			problems = append(problems, t.where+": прежнего эссе нет ни у кого из авторов — не пиши о прежних эссе и о том, что меняется между эссе")
		}
	}
	return append(problems, tooLongV3(words)...)
}

// cohortEarlierWords find words about the earlier essays in a summary: written
// of a cohort where nobody wrote one, they invent a comparison (decided
// 5 November 2026 after the run: «У тех, кто писал эссе ранее…»).
var cohortEarlierWords = regexp.MustCompile(`(?i)(прежн|предыдущ)\p{L}*(\s+и\s+(нов|последн|нынешн)\p{L}*)?\s+эссе|писал\p{L}*\s+(эссе\s+)?(ранее|раньше|прежде|до\s+этого)|(ранее|раньше|прежде)\s+писал\p{L}*|между\s+(двумя\s+)?эссе`)

// cohortHasEarlier says whether an author of the input of a summary wrote an essay before.
func cohortHasEarlier(payload *object) bool {
	for _, review := range objectList(payload.values["разборы"]) {
		if earlier, _ := review.values["прежнее_эссе"].(bool); earlier {
			return true
		}
	}
	return false
}
