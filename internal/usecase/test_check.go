package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The check of the answers to the instructions test/v2 and test-platoon/v1 by
// their sections «Что программа проверяет» (Вики/Тестирование/Инструкция v2.md,
// Инструкция по взводу.md, approved on 24 October 2026). A problem names the
// field, never quotes the answer: the problems go to the journal.

// componentLevelText names the level of a component by the levels of its
// scales (Вики/Тестирование/Образец портрета.md): one level — that level;
// different ones — «преимущественно» and the prevailing one; a tie — the tied
// levels from the lowest to the highest through a dash. Scales without a level
// do not count; with none the level is not determined.
func componentLevelText(levels []string) string {
	counts := map[string]int{}
	for _, level := range levels {
		if levelRank[level] > 0 {
			counts[level]++
		}
	}
	best := 0
	for _, n := range counts {
		best = max(best, n)
	}
	top := []string{}
	for _, level := range []string{"low", "medium", "high"} {
		if counts[level] == best && best > 0 {
			top = append(top, levelText[level])
		}
	}
	switch {
	case len(top) == 0:
		return levelText[""]
	case len(counts) == 1:
		return top[0]
	case len(top) == 1:
		return "преимущественно " + top[0]
	}
	return strings.Join(top, " — ")
}

var levelCode = map[string]string{"низкий": "low", "средний": "medium", "высокий": "high"}

// levelStems are the stems of the levels a text of the component must use.
func levelStems(text string) []string {
	stems := []string{}
	for _, word := range []string{"низк", "средн", "высок"} {
		if strings.Contains(text, word) {
			stems = append(stems, word)
		}
	}
	return stems
}

// normalizedText compares texts of the model: lower case, «е» for «ё», one
// kind of dash, single spaces, no final full stop.
func normalizedText(text string) string {
	text = strings.ToLower(strings.ReplaceAll(text, "ё", "е"))
	text = strings.NewReplacer("–", "—", "−", "—", " - ", " — ").Replace(text)
	return strings.TrimSuffix(strings.Join(strings.Fields(text), " "), ".")
}

// sentenceEnd is the end of a sentence: a stop, maybe a quote or a bracket,
// spaces and a capital letter or an opening quote.
var sentenceEnd = regexp.MustCompile(`[.!?…]+[»")]*\s+[«"(]?[A-ZА-ЯЁ]`)

// sentenceCount counts sentences. A stop after a single letter is not an end:
// «т. е.», «т. п.» and the initials «А. А. Азбель».
func sentenceCount(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	count := 1
	for _, at := range sentenceEnd.FindAllStringIndex(text, -1) {
		letters := 0
		for before := text[:at[0]]; before != ""; letters++ {
			r, size := utf8.DecodeLastRuneInString(before)
			if !unicode.IsLetter(r) {
				break
			}
			before = before[:len(before)-size]
		}
		if letters == 1 {
			continue
		}
		count++
	}
	return count
}

var digit = regexp.MustCompile(`[0-9]`)

// appointment finds a recommendation of a post, forbidden by both instructions.
var appointment = regexp.MustCompile(`(?i)назначить|назначени[еяю] на`)

// fieldProblems are the rules every text field follows: no figures, no words
// of a diagnosis, of fitness or of an appointment.
func fieldProblems(field, text string) []string {
	problems := []string{}
	if strings.TrimSpace(text) == "" {
		return []string{"нет поля " + field}
	}
	if digit.MatchString(text) {
		problems = append(problems, field+": цифры в тексте")
	}
	if found := diagnosis.FindString(text); found != "" {
		problems = append(problems, fmt.Sprintf("%s: недопустимое слово («%s»)", field, strings.ToLower(found)))
	}
	if found := appointment.FindString(text); found != "" {
		problems = append(problems, fmt.Sprintf("%s: о назначении на должность («%s»)", field, strings.ToLower(found)))
	}
	return problems
}

func sentenceProblem(field, text string, least, most int) []string {
	n := sentenceCount(text)
	if strings.TrimSpace(text) == "" || n >= least && n <= most {
		return nil
	}
	if least == most {
		return []string{fmt.Sprintf("%s: предложений %d вместо %d", field, n, least)}
	}
	return []string{fmt.Sprintf("%s: предложений %d вместо %d–%d", field, n, least, most)}
}

func outputText(output map[string]any, key string) string {
	text, _ := output[key].(string)
	return text
}

// window is the text of up to sixty letters after a position, up to the end
// of the sentence or of a clause.
func window(text string, at int) string {
	tail := []rune(text[at:])
	tail = tail[:min(len(tail), 60)]
	result := string(tail)
	if end := strings.IndexAny(result, ".;!?"); end >= 0 {
		result = result[:end]
	}
	return result
}

// wordAt finds a word, whole, in a lower-case text.
func wordAt(text, word string) int {
	for from := 0; ; {
		at := strings.Index(text[from:], word)
		if at < 0 {
			return -1
		}
		at += from
		before, _ := utf8.DecodeLastRuneInString(text[:at])
		after, _ := utf8.DecodeRuneInString(text[at+len(word):])
		if !unicode.IsLetter(before) && !unicode.IsLetter(after) {
			return at
		}
		from = at + len(word)
	}
}

// namedLevelProblems checks that the level written right after the name of a
// kind of activity or of a scale of a method is the program's.
func namedLevelProblems(field, text string, names map[string]string) []string {
	problems := []string{}
	lower := normalizedText(text)
	keys := make([]string, 0, len(names))
	for name := range names {
		keys = append(keys, name)
	}
	slices.Sort(keys)
	for _, name := range keys {
		want := names[name]
		if levelCode[want] == "" {
			continue
		}
		at := wordAt(lower, normalizedText(name))
		if at < 0 {
			continue
		}
		if found := levelWord.FindString(window(lower, at+len(normalizedText(name)))); found != "" && !strings.HasPrefix(want, found) {
			problems = append(problems, fmt.Sprintf("%s: уровень «%s» не тот", field, name))
		}
	}
	return problems
}

// componentStems find a component in a text in any case.
var componentStems = map[string]string{"CS": "ценностно-смыслов", "EV": "эмоционально-волев", "DP": "деятельностно-практическ"}

// expectedComponentLevels are the levels of the components by the levels of the
// scales of the methods in the input.
func expectedComponentLevels(payload *object) map[string]string {
	byComponent := map[string][]string{}
	for _, m := range objectList(payload.values["методики"]) {
		for code, name := range portraitComponentNames {
			if objectText(m, "компонент") == name {
				byComponent[code] = append(byComponent[code], levelCode[objectText(m, "уровень")])
			}
		}
	}
	expected := map[string]string{}
	for code := range portraitComponentNames {
		expected[code] = componentLevelText(byComponent[code])
	}
	return expected
}

// componentProblems checks components.levels by the rule and that the text
// names each component with the same level.
func componentProblems(components map[string]any, payload *object) []string {
	problems := []string{}
	levels, _ := components["levels"].(map[string]any)
	text, _ := components["text"].(string)
	expected := expectedComponentLevels(payload)
	lower := normalizedText(text)
	for _, code := range []string{"CS", "EV", "DP"} {
		key := strings.ToLower(code)
		given, _ := levels[key].(string)
		want := expected[code]
		problems = append(problems, fieldProblems("components.levels."+key, given)...)
		if strings.TrimSpace(given) != "" && normalizedText(given) != normalizedText(want) {
			problems = append(problems, fmt.Sprintf("components.levels.%s: по правилу «%s»", key, want))
		}
		stems := levelStems(want)
		if len(stems) == 0 || text == "" {
			continue
		}
		at := strings.Index(lower, componentStems[code])
		if at < 0 {
			problems = append(problems, "components.text: не назван "+portraitComponentNames[code]+" компонент")
			continue
		}
		found := levelWord.FindString(window(lower, at+len(componentStems[code])))
		if found == "" {
			before := []rune(lower[:at])
			words := levelWord.FindAllString(string(before[max(0, len(before)-60):]), -1)
			if len(words) > 0 {
				found = words[len(words)-1]
			}
		}
		if found == "" || !slices.Contains(stems, strings.ToLower(found)) {
			problems = append(problems, "components.text: уровень компонента «"+portraitComponentNames[code]+"» не тот, что в levels")
		}
	}
	return problems
}

// statusStem is the stem of the name of a status of identity: its first word
// without the ending, so that «сформированная», «сформирована» and
// «сформированность» all have it.
func statusStem(name string) string {
	word := strings.Fields(normalizedText(name))[0]
	for _, ending := range []string{"ая", "ий", "ый"} {
		if trimmed, ok := strings.CutSuffix(word, ending); ok {
			word = trimmed
			break
		}
	}
	if trimmed, ok := strings.CutSuffix(word, "нн"); ok {
		word = trimmed + "н"
	}
	return word
}

const identityAbsent = "Методика профессиональной идентичности не проводилась."

// identityProblems checks identity: the fixed phrase without the method; with
// it, the leading statuses named. leading gives the names of the statuses the
// text must name for every block.
func identityProblems(text string, given bool, leading [][]string) []string {
	if !given {
		if strings.TrimSpace(text) != identityAbsent {
			return []string{"identity: без методики нужно ровно «" + identityAbsent + "»"}
		}
		return nil
	}
	if strings.TrimSpace(text) == identityAbsent {
		return []string{"identity: методика проводилась"}
	}
	lower := normalizedText(text)
	for _, names := range leading {
		for _, name := range names {
			if !strings.Contains(lower, statusStem(name)) {
				return []string{"identity: не назван ведущий статус «" + name + "»"}
			}
		}
	}
	return nil
}

// recommendationProblems checks the list of recommendations: the number, the
// code from the catalogue that fits the group, and, when allowed is given, the
// type of the direction allowed for the group.
func recommendationProblems(output map[string]any, payload *object, least, most int, allowed func(group int, kind string) bool) []string {
	problems := []string{}
	items, ok := output["recommendations"].([]any)
	if !ok || len(items) < least || len(items) > most {
		return []string{fmt.Sprintf("recommendations: нужно от %d до %d пунктов", least, most)}
	}
	catalogue := map[string]*object{}
	for _, d := range objectList(payload.values["каталог_рекомендаций"]) {
		catalogue[objectText(d, "код")] = d
	}
	for n, item := range items {
		record, _ := item.(map[string]any)
		field := fmt.Sprintf("recommendations.%d", n+1)
		number, isNumber := numberOf(record["group"])
		code, _ := record["code"].(string)
		text, _ := record["text"].(string)
		problems = append(problems, fieldProblems(field+".text", text)...)
		direction, known := catalogue[strings.TrimSpace(code)]
		switch {
		case !isNumber || number < 1 || number > 5 || number != float64(int(number)):
			problems = append(problems, field+": group — номер группы от 1 до 5")
		case !known:
			problems = append(problems, field+": кода нет в каталоге")
		case !slices.Contains(direction.values["группы"].([]any), any(int(number))):
			problems = append(problems, field+": направление не подходит к группе")
		case allowed != nil && !allowed(int(number), objectText(direction, "тип")):
			problems = append(problems, field+": тип направления не подходит к ситуации группы")
		}
	}
	return problems
}

// numberOf reads a number of an answer: float64 from JSON, int when built in Go.
func numberOf(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	}
	return 0, false
}

// checkTestAnswerV2 returns what is wrong with an answer to test/v2.
func checkTestAnswerV2(output map[string]any, payload *object) []string {
	problems := []string{}
	for _, key := range []string{"portrait", "identity", "ideal", "activity", "conclusion"} {
		problems = append(problems, fieldProblems(key, outputText(output, key))...)
	}
	components, ok := output["components"].(map[string]any)
	if !ok {
		problems = append(problems, "нет поля components")
		components = map[string]any{}
	}
	text, _ := components["text"].(string)
	problems = append(problems, fieldProblems("components.text", text)...)
	problems = append(problems, sentenceProblem("portrait", outputText(output, "portrait"), 1, 1)...)
	problems = append(problems, sentenceProblem("components.text", text, 4, 6)...)
	problems = append(problems, sentenceProblem("identity", outputText(output, "identity"), 1, 2)...)
	problems = append(problems, sentenceProblem("ideal", outputText(output, "ideal"), 2, 2)...)
	problems = append(problems, sentenceProblem("activity", outputText(output, "activity"), 3, 4)...)
	problems = append(problems, sentenceProblem("conclusion", outputText(output, "conclusion"), 2, 2)...)
	problems = append(problems, componentProblems(components, payload)...)

	names := map[string]string{}
	for _, k := range objectList(payload.values["виды_деятельности"]) {
		if words := strings.Fields(objectText(k, "вид")); len(words) > 0 {
			names[words[0]] = objectText(k, "уровень")
		}
	}
	lowEV := false
	for _, m := range objectList(payload.values["методики"]) {
		names[objectText(m, "показатель")] = objectText(m, "уровень")
		lowEV = lowEV || objectText(m, "компонент") == portraitComponentNames["EV"] && objectText(m, "уровень") == levelText["low"]
	}
	for _, key := range []string{"portrait", "activity", "conclusion"} {
		problems = append(problems, namedLevelProblems(key, outputText(output, key), names)...)
	}
	problems = append(problems, namedLevelProblems("components.text", text, names)...)

	identity, given := payload.values["профессиональная_идентичность"]
	leading := [][]string{}
	for _, block := range objectList(identity) {
		names := []string{}
		for _, status := range objectList(block.values["статусы"]) {
			if strings.Contains(objectText(block, "ведущий_статус"), objectText(status, "статус")) {
				names = append(names, objectText(status, "статус"))
			}
		}
		leading = append(leading, names)
	}
	problems = append(problems, identityProblems(outputText(output, "identity"), given, leading)...)

	types := map[int]string{}
	for _, g := range objectList(payload.values["образ_защитника"]) {
		number, _ := g.values["номер"].(int)
		types[number] = objectText(g, "тип_рекомендации")
	}
	problems = append(problems, recommendationProblems(output, payload, 3, 4, func(group int, kind string) bool {
		return types[group] == "" || kind == types[group] || kind == "психологическая" && lowEV
	})...)
	return problems
}
