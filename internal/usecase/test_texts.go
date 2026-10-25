package usecase

import (
	"fmt"
	"regexp"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// The checks of the texts of the portrait by the test and of the conclusion by
// a platoon added after the run of 24–25 October 2026 on GigaChat-3-Ultra
// (Вики/Тестирование/прогоны/2026-10-24/Сводка.md, «Что заметно в принятых
// текстах»; approved by the user on 25 October 2026). The rules are those of
// the instructions test/v3 and test-platoon/v1, which the program did not
// check. A problem names the field and the word found, never the text around it.

// lowerText is a text in lower case with «е» for «ё».
func lowerText(text string) string {
	return strings.ReplaceAll(strings.ToLower(text), "ё", "е")
}

// clauseBreak splits a text into clauses: a level and a rating are taken
// together only within one.
var clauseBreak = regexp.MustCompile(`[,;:.!?()]`)

// levelAdverb is a level said by an adverb: «высоко оценивает».
var levelAdverb = regexp.MustCompile(`(^|[^а-я])((?:не)?(?:высоко|низко|средне))([^а-я]|$)`)

// ratingLevel is a level next to a rating: «высокие оценки», «оцениваются как высокие».
var ratingLevel = regexp.MustCompile(`(высок|низк|средн)[а-я]*\s+(?:оценк|оценив|оценен|балл)|(?:оценк|оценив|оценен)[а-я]*\s+(?:как\s+)?(высок|низк|средн)`)

// conditionLevel is a level put on the conditions: «условия ВУЦ — на высоком уровне».
var conditionLevel = regexp.MustCompile(`(^|[^а-я])услови[яй]\s+(?:[а-я-]+\s+){0,2}(?:—\s+)?(?:на\s+)?(высок|низк|средн)`)

// ratedLevelProblems finds a level put on how the conditions or the signs of
// the environment are rated: rule 2 of both instructions allows «оценивает
// выше», «выражено сильнее», «недостаточно», not «высокий», «средний»,
// «низкий». The levels of the kinds of activity («включён … на высоком
// уровне») are the program's and pass; so does «самооценка», whose level the
// input names.
func ratedLevelProblems(field, text string) []string {
	lower := strings.ReplaceAll(lowerText(text), "самооцен", "само-цен")
	for _, clause := range clauseBreak.Split(lower, -1) {
		found := ""
		if m := levelAdverb.FindStringSubmatch(clause); m != nil && strings.Contains(clause, "оцен") {
			found = m[2]
		} else if m := ratingLevel.FindStringSubmatch(clause); m != nil {
			found = m[1] + m[2] + "…"
		} else if m := conditionLevel.FindStringSubmatch(clause); m != nil {
			found = m[2] + "…"
		}
		// The run of 25 October: told «пиши «оценивает выше»», the model kept
		// «высоко оценивает» for five attempts; it is told the word itself.
		if found != "" {
			return []string{fmt.Sprintf("%s: слово «%s» об оценке условий или признаков среды запрещено — пиши «выше всего оценивает …», «ниже оценивает …», «недостаточно»", field, found)}
		}
	}
	return nil
}

// unfoundStates are the states a psychologist might write of that the test
// does not measure, each by the stems of its words. In an answer such a word
// must come from the input — from what the student wrote of himself — or it is
// made up: rule 5 of test/v3 and rule 4 of test-platoon/v1, «опирайся только
// на входные данные».
var unfoundStates = [][]string{{"тревог", "тревож"}, {"страх"}, {"агресси"}, {"апати"}, {"депресси"}, {"выгоран"}, {"переутомл"}, {"конфликтн"}, {"замкнут"}, {"импульсивн"}, {"неуверенн"}, {"застенчив"}, {"раздражит"}, {"вспыльчив"}}

// containsAny says whether a text has one of the stems.
func containsAny(text string, stems []string) bool {
	for _, stem := range stems {
		if strings.Contains(text, stem) {
			return true
		}
	}
	return false
}

// unfoundStateProblems finds in the texts of an answer a state the input has
// no word of.
func unfoundStateProblems(texts map[string]string, payload *object) []string {
	input := lowerText(string(canonicalJSON(payload)))
	problems := []string{}
	for _, key := range sortedTextKeys(texts) {
		lower := lowerText(texts[key])
		for _, stems := range unfoundStates {
			if containsAny(lower, stems) && !containsAny(input, stems) {
				problems = append(problems, fmt.Sprintf("%s: о состоянии «%s…», о котором во входных данных ничего нет — пиши только о том, что в них есть", key, stems[0]))
			}
		}
	}
	return problems
}

// recommendationTexts adds the texts of the recommendations to the texts of an
// answer, under recommendations.N.text.
func recommendationTexts(output map[string]any, texts map[string]string) {
	items, _ := output["recommendations"].([]any)
	for n, item := range items {
		if record, ok := item.(map[string]any); ok {
			text, _ := record["text"].(string)
			texts[fmt.Sprintf("recommendations.%d.text", n+1)] = text
		}
	}
}

// numeral is a number written in words, from two up.
const numeral = `два|две|двух|двум|двумя|три|трех|трем|тремя|четыре|четырех|четырем|четырьмя|пять|пяти|пятью|шесть|шести|семь|семи|восемь|восьми|девять|девяти|десять|десяти|[а-я]+надцат[а-я]*|двадцат[а-я]*|тридцат[а-я]*`

// platoonCount finds a number of students given by a numeral: «двое», «у
// двух», «трёх студентов», «два из …». Rule 1 of test-platoon/v1 wants it
// relative to the platoon; «один студент» the rule allows itself, and «четыре
// вида деятельности» or «пять групп» are not students.
var platoonCount = []*regexp.Regexp{
	regexp.MustCompile(`(?:^|[^а-я])(двое|двоих|трое|троих|четверо|четверых|пятеро|пятерых|шестеро|шестерых|семеро|семерых)(?:[^а-я]|$)`),
	regexp.MustCompile(`(?:^|[^а-я])у\s+(` + numeral + `)(?:[^а-я]|$)`),
	regexp.MustCompile(`(?:^|[^а-я])(` + numeral + `)\s+(?:[а-я]+\s+)?(?:студент|курсант|человек|из\s)`),
}

// numeralValues are the numbers of the numerals a count of students is
// written with, to tell the retry the word relative to the platoon.
var numeralValues = map[string]int{}

func init() {
	for n, words := range [][]string{
		2:  {"два", "две", "двух", "двум", "двумя", "двое", "двоих"},
		3:  {"три", "трех", "трем", "тремя", "трое", "троих"},
		4:  {"четыре", "четырех", "четырем", "четырьмя", "четверо", "четверых"},
		5:  {"пять", "пяти", "пятью", "пятеро", "пятерых"},
		6:  {"шесть", "шести", "шестеро", "шестерых"},
		7:  {"семь", "семи", "семеро", "семерых"},
		8:  {"восемь", "восьми"},
		9:  {"девять", "девяти"},
		10: {"десять", "десяти"},
	} {
		for _, word := range words {
			numeralValues[word] = n
		}
	}
}

// relativeCount is the word of rule 1 of test-platoon/v1 for so many students
// of the platoon.
func relativeCount(count, total int) string {
	switch {
	case count == 0 || total == 0:
		return ""
	case count >= total:
		return "все"
	case 2*count > total:
		return "большинство"
	case 2*count == total:
		return "половина"
	case 3*count == total:
		return "треть"
	}
	return "несколько студентов"
}

// platoonCountProblems finds a numeral of students in the texts of the
// conclusion by a platoon of total students.
func platoonCountProblems(texts map[string]string, total int) []string {
	problems := []string{}
	for _, key := range sortedTextKeys(texts) {
		lower := lowerText(texts[key])
		for _, pattern := range platoonCount {
			if m := pattern.FindStringSubmatch(lower); m != nil {
				// The run of 25 October: told the list of words, the model kept
				// «у двоих» for four attempts; it is told the word that fits.
				fits := ""
				if word := relativeCount(numeralValues[m[1]], total); word != "" {
					fits = fmt.Sprintf("; здесь — «%s»", word)
				}
				problems = append(problems, fmt.Sprintf("%s: число студентов числительным («%s») — пиши относительно взвода: «большинство», «около половины», «треть», «несколько студентов»%s", key, m[1], fits))
				break
			}
		}
	}
	return problems
}

// selfDetermination is what a point about the self-determination as a
// military specialist speaks of.
var selfDetermination = regexp.MustCompile(`самоопредел|выбор|военн[а-я]*\s+специал`)

// selfDeterminationProblems: when the leading status of the military
// speciality is the undetermined identity or the moratorium, one point of the
// recommendations goes to group 2 about the self-determination as a military
// specialist; when it is the imposed one, about the personal sense of the
// choice (field recommendations of test/v3). The group of the imposed one is
// taken to be 2 as well (approved by the user on 25 October 2026).
func selfDeterminationProblems(output map[string]any, payload *object) []string {
	leading := ""
	for _, block := range objectList(payload.values["профессиональная_идентичность"]) {
		if strings.HasPrefix(lowerText(objectText(block, "профессия")), "военн") {
			leading = objectText(block, "ведущий_статус")
		}
	}
	status := lowerText(leading)
	undecided := strings.Contains(status, "мораторий") || strings.Contains(status, "неопределенн")
	if !undecided && !strings.Contains(status, "навязанн") {
		return nil
	}
	items, _ := output["recommendations"].([]any)
	for _, item := range items {
		record, _ := item.(map[string]any)
		number, _ := numberOf(record["group"])
		text, _ := record["text"].(string)
		text = lowerText(text)
		if number == 2 && (undecided && selfDetermination.MatchString(text) || !undecided && strings.Contains(text, "смысл") && strings.Contains(text, "выбор")) {
			return nil
		}
	}
	// The run of 25 October: the model put the point to group 2 with a code
	// that does not fit it for five attempts; it is told the codes that do.
	codes := ""
	for _, g := range objectList(payload.values["образ_защитника"]) {
		if number, _ := numberOf(g.values["номер"]); number == 2 {
			codes = strings.Join(textList(g.values["подходящие_направления"]), ", ")
		}
	}
	if codes != "" {
		codes = "; код — из подходящих группе 2: " + codes
	}
	if undecided {
		return []string{fmt.Sprintf("recommendations: по военной специальности ведущий статус «%s» — нужен пункт к группе 2 о самоопределении как военного специалиста, со словом «самоопределение»%s", leading, codes)}
	}
	return []string{fmt.Sprintf("recommendations: по военной специальности ведущий статус «%s» — нужен пункт к группе 2 о личном смысле выбора, со словами «смысл» и «выбор»%s", leading, codes)}
}

// recommendationLeast is the fewest recommendations both instructions ask for.
const recommendationLeast = 3

// directionKey is the group and the direction of a recommendation.
func directionKey(item any) (string, bool) {
	record, ok := item.(map[string]any)
	if !ok {
		return "", false
	}
	number, isNumber := numberOf(record["group"])
	code, _ := record["code"].(string)
	if !isNumber || strings.TrimSpace(code) == "" {
		return "", false
	}
	return fmt.Sprintf("%g/%s", number, strings.TrimSpace(code)), true
}

// cleanRepeatedDirections drops a recommendation that gives a group the
// direction an earlier one already gave it (Морозов on 25 October: R16 twice to
// group 1), as long as three are left; a repeat that would leave fewer stays
// for the check, and the model is asked again.
func cleanRepeatedDirections(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int) {
	items, ok := output["recommendations"].([]any)
	if !ok {
		return output, 0
	}
	seen := map[string]bool{}
	kept := []any{}
	dropped := 0
	for _, item := range items {
		key, ok := directionKey(item)
		if ok && seen[key] && len(items)-dropped-1 >= recommendationLeast {
			dropped++
			continue
		}
		if ok {
			seen[key] = true
		}
		kept = append(kept, item)
	}
	if dropped > 0 {
		output["recommendations"] = kept
	}
	return output, dropped
}

// repeatedDirectionProblems finds a direction given to the same group twice.
func repeatedDirectionProblems(output map[string]any) []string {
	items, _ := output["recommendations"].([]any)
	seen := map[string]bool{}
	problems := []string{}
	for n, item := range items {
		key, ok := directionKey(item)
		if !ok {
			continue
		}
		if seen[key] {
			group, code, _ := strings.Cut(key, "/")
			problems = append(problems, fmt.Sprintf("recommendations.%d: направление %s к группе %s уже есть — возьми другое направление или другую группу", n+1, code, group))
		}
		seen[key] = true
	}
	return problems
}
