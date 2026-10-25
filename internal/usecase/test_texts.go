package usecase

import (
	"fmt"
	"regexp"
	"strings"
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
			found = m[1] + m[2]
		} else if m := conditionLevel.FindStringSubmatch(clause); m != nil {
			found = m[2]
		}
		if found != "" {
			return []string{fmt.Sprintf("%s: уровень («%s») у оценки условий или признаков среды — пиши «оценивает выше», «выражено сильнее», «недостаточно»", field, found)}
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

// platoonCountProblems finds a numeral of students in the texts of the
// conclusion by a platoon.
func platoonCountProblems(texts map[string]string) []string {
	problems := []string{}
	for _, key := range sortedTextKeys(texts) {
		lower := lowerText(texts[key])
		for _, pattern := range platoonCount {
			if m := pattern.FindStringSubmatch(lower); m != nil {
				problems = append(problems, fmt.Sprintf("%s: число студентов числительным («%s») — пиши относительно взвода: «большинство», «около половины», «треть», «несколько студентов»", key, m[1]))
				break
			}
		}
	}
	return problems
}
