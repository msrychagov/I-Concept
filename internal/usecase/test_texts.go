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
