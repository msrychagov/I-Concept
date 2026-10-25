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
		if familyWords.MatchString(textOf(item["text"])) {
			problems = append(problems, where+": рекомендация опирается на семью автора — так нельзя (правило 5)")
		}
		problems = append(problems, quotesProblems(where, item["quotes"], current, 1, 2)...)
		words += countWords(textOf(item["text"]))
	}
	problems = append(problems, sameAdvice(advice)...)
	problems = append(problems, tooLong(words)...)
	if len(previous) == 0 {
		problems = append(problems, noEarlierProblems(output)...)
	}
	problems = append(problems, teacherProblems(output, payload, current)...)
	return append(problems, labelWordProblems(output, payload)...)
}

// familyWords find a recommendation that stands on the family of the author,
// which rule 5 of v2 forbids (decided 25 October 2026 after the run): «семья»,
// «родные», «отец», «мать», «бабушка», «дедушка», «предки», «брат», «сестра».
// «семинар», «семестр», «материал», «братство» and «Родина» are none of them.
var familyWords = regexp.MustCompile(`(?i)(^|[^\p{L}])(сем(ь|ей|ейн)\p{L}*|родн(ые|ых|ым|ыми)([^\p{L}]|$)|отц\p{L}*|отец|мат(ь|ери|ерью)([^\p{L}]|$)|бабушк\p{L}*|дедушк\p{L}*|предк\p{L}*|брат(а|у|ом|е|ья|ьев|ьям|ьями)?([^\p{L}]|$)|сестр\p{L}*)`)

// earlierEssay finds words about an earlier essay: written of an author who has
// none, they invent a comparison or say that there is nothing to compare.
var earlierEssay = regexp.MustCompile(`(?i)(прежн|предыдущ)\p{L}*(\s+и\s+(нов|последн|нынешн)\p{L}*)?\s+эссе`)

// noEarlierProblems finds a paragraph or a recommendation that speaks of an
// earlier essay the author did not write (decided 25 October 2026 after the run).
func noEarlierProblems(output map[string]any) []string {
	problems := []string{}
	check := func(where string, raw any) {
		if earlierEssay.MatchString(textOf(raw)) {
			problems = append(problems, where+": прежнего эссе у автора нет — не пиши о нём")
		}
	}
	for _, key := range essayParagraphs {
		check(key, asMap(output[key])["text"])
	}
	advice, _ := asList(output["recommendations"])
	for n, raw := range advice {
		check(fmt.Sprintf("recommendations, пункт %d", n+1), asMap(raw)["text"])
	}
	return problems
}

// instructorOf reads an entry of the list in the input, «Иванов Игорь Петрович,
// полковник», back into the four things a student may call him by.
func instructorOf(who string) model.Instructor {
	names, rank, _ := strings.Cut(who, ",")
	parts := strings.Fields(names)
	instructor := model.Instructor{Rank: strings.TrimSpace(rank)}
	if len(parts) > 0 {
		instructor.Surname = parts[0]
	}
	if len(parts) > 1 {
		instructor.Name = parts[1]
	}
	if len(parts) > 2 {
		instructor.Patronymic = strings.Join(parts[2:], " ")
	}
	return instructor
}

// labelWordProblems finds an entry with a label whose quotes hold no word of
// that instructor: no surname, name, patronymic or rank, in any case. A
// surname with a typo counts. Such a label is a guess of the model: «старшина
// Громов» given to a captain of the list (decided 25 October 2026 after the run).
func labelWordProblems(output map[string]any, payload *object) []string {
	list := map[string]model.Instructor{}
	for _, entry := range objectList(payload.values["преподаватели"]) {
		list[objectText(entry, "метка")] = instructorOf(objectText(entry, "кто"))
	}
	problems := []string{}
	teachers, _ := asList(output["teachers"])
	for n, raw := range teachers {
		record := asMap(raw)
		self, labelled := list[textOf(record["who"])]
		if record == nil || !labelled {
			continue
		}
		texts := []string{}
		for _, quote := range quoteList(record["quotes"]) {
			texts = append(texts, textOf(quote["quote"]))
		}
		if found, _ := foundIn(self, texts); len(found) > 0 || surnameWithTypo(self.Surname, texts) {
			continue
		}
		problems = append(problems, fmt.Sprintf("teachers, запись %d: в цитатах нет ни фамилии, ни имени, ни отчества, ни звания преподавателя с этой меткой — если не уверен, что автор пишет о нём, пиши, как его называет автор", n+1))
	}
	return problems
}

// surnameWithTypo finds the surname written with a typo, «Белоцерковкий» for
// «Белоцерковский»: a word whose stem is one letter off a short surname or two
// off a long one, in any case.
func surnameWithTypo(surname string, texts []string) bool {
	stem := wordStem(surname)
	if len(stem) < 5 {
		return false
	}
	most := 1
	if len(stem) >= 8 {
		most = 2
	}
	for _, text := range texts {
		_, normal := tokens(text)
		for _, token := range normal {
			for _, ending := range caseEndings {
				if base, ok := strings.CutSuffix(token, ending); ok && editDistance([]rune(base), stem) <= most {
					return true
				}
			}
		}
	}
	return false
}

// editDistance is the number of letters to add, drop or change to turn one word into the other.
func editDistance(a, b []rune) int {
	row := make([]int, len(b)+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= len(a); i++ {
		previous := row[0]
		row[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			previous, row[j] = row[j], min(row[j]+1, row[j-1]+1, previous+cost)
		}
	}
	return row[len(b)]
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
	changes := trimTexts(output)
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

// leadingMarks are what the model sometimes puts before a text: «: Авторы …».
const leadingMarks = " \t\n:;,.—–-"

// trimTexts takes the marks off the start of the paragraphs and the
// recommendations of an answer of v2 and counts the texts it changed.
func trimTexts(output map[string]any) int {
	changes := 0
	trim := func(item map[string]any) {
		if text, ok := item["text"].(string); ok {
			if trimmed := strings.TrimLeft(text, leadingMarks); trimmed != text {
				item["text"] = trimmed
				changes++
			}
		}
	}
	for _, key := range essayParagraphs {
		if paragraph := asMap(output[key]); paragraph != nil {
			trim(paragraph)
		}
	}
	advice, _ := asList(output["recommendations"])
	for _, raw := range advice {
		if item := asMap(raw); item != nil {
			trim(item)
		}
	}
	return changes
}

// cleanEssayAnswerV2 takes the marks off the start of the texts and the
// labels of namesakes off the entries about instructors (essay_namesakes.go).
func cleanEssayAnswerV2(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int) {
	changes := trimTexts(output)
	output, more := cleanEssayAnswer(a, db, s, output, payload)
	return output, changes + more
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
		if militaryRank.MatchString(textOf(raw)) {
			problems = append(problems, where+": звание в тексте — не называй конкретных преподавателей, о них отдельный блок")
		}
		if singlesOut(textOf(raw)) {
			problems = append(problems, where+": авторы перечислены по одному («один … другой», «первый … второй») — различия пиши группами: «у одних авторов …, у других …», без перечня авторов")
		}
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

// militaryRank finds a rank in a text of the summary: with a surname it names
// an instructor, which the summary must not (decided 25 October 2026 after the
// run: «подполковником Ковальчуком», «старшины Громова»).
var militaryRank = regexp.MustCompile(`(?i)(^|[^\p{L}])(лейтенант|капитан|майор|подполковник|полковник|генерал|старшин(а|ы|е|у|ой)([^\p{L}]|$)|сержант|прапорщик|ефрейтор)`)

// countedNouns follow «первый» or «один» when they count time or things, not
// authors: «в первый год …, во второй».
var countedNouns = []string{"год", "семестр", "курс", "раз", "ден", "дн", "недел", "месяц", "эссе", "выезд", "сбор", "этап", "блок", "вопрос", "занят", "путь", "пут"}

var clauseEnd = regexp.MustCompile(`[.!?;]`)

// singlesOut finds a sentence that goes over the authors one by one: «один …,
// другой …», «первый …, второй …» (decided 25 October 2026 after the run). The
// groups «одни …, другие …» are allowed.
func singlesOut(text string) bool {
	counted := func(normal []string, at int) bool {
		if at+1 >= len(normal) {
			return false
		}
		for _, noun := range countedNouns {
			if strings.HasPrefix(normal[at+1], noun) {
				return true
			}
		}
		return false
	}
	for _, sentence := range clauseEnd.Split(text, -1) {
		_, normal := tokens(sentence)
		first := -1
		for at, word := range normal {
			switch {
			case first < 0 && (word == "один" || word == "первый") && !counted(normal, at):
				first = at
			case first >= 0 && (word == "другой" || word == "второй") && !counted(normal, at):
				return true
			}
		}
	}
	return false
}
