package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"vucdiagnostic/internal/domain/model"
)

// The checks of test/v6 and test-platoon/v4 (Вики/Тестирование/План
// конкретных рекомендаций и Квена.md, decisions Д8, Д10, Д13, Д14, approved
// by the user on 27 October 2026): a recommendation is a concrete action —
// who does it, in which activity, what exactly — and not the name of a
// direction; «Кому нужна отдельная работа» gives an action to every group of
// students the program put together.

// actionWho and actionKinds are the values of who and kind (decision Д6).
var (
	actionWho   = []string{"куратор взвода", "преподаватель", "начальник цикла", "психолог"}
	actionKinds = []string{"учебная", "воспитательная", "повседневная", "просветительская"}
)

// actionLeast is the fewest words of an action: «Провести тренинг
// саморегуляции и стрессоустойчивости для развития самообладания» has 8.
const actionLeast = 12

// actionDate is a date, a term or a frequency, which an action must not have.
var actionDate = regexp.MustCompile(`(?:^|[^а-я])(раз\s+в\s|ежедневн|еженедельн|ежемесячн|в\s+течение|январ|феврал|март|апрел|ма[йяе](?:[^а-я]|$)|июн|июл|август|сентябр|октябр|ноябр|декабр)`)

// actionExamples are the examples «так» of the instructions: an action that
// repeats one of them is copied (Д8). The test checks they are those of the wiki.
var actionExamples = []string{
	"Перед нормативами разобрать со студентом, в каких ситуациях на занятиях и в наряде он теряет выдержку, и отработать с ним паузу перед действием; после нормативов вместе обсудить, помогла ли она",
	"К ближайшей памятной дате поручить взводу подготовить рассказ о выпускниках ВУЦ, которые служили в армии, и после него обсудить, что из их опыта студенты взяли бы себе",
	"После ближайшего наряда разобрать с каждым из этих студентов, что получилось и что нет, и поставить одну личную цель по собранности на следующий наряд; на следующем разборе сверить, достигнута ли она",
}

// exampleRun is how many words in a row of an example make a copy.
const exampleRun = 6

// wordsOf are the words of a text in lower case with «е» for «ё».
func wordsOf(text string) []string {
	return strings.FieldsFunc(lowerText(text), func(r rune) bool { return !unicode.IsLetter(r) && r != '-' })
}

// copiedExample says whether a text has exampleRun words in a row of an example.
func copiedExample(text string) bool {
	words := " " + strings.Join(wordsOf(text), " ") + " "
	for _, example := range actionExamples {
		list := wordsOf(example)
		for at := 0; at+exampleRun <= len(list); at++ {
			if strings.Contains(words, " "+strings.Join(list[at:at+exampleRun], " ")+" ") {
				return true
			}
		}
	}
	return false
}

// actionProblems checks the who, the kind and the text of an action.
func actionProblems(field string, record map[string]any) []string {
	problems := []string{}
	who, _ := record["who"].(string)
	if !slices.Contains(actionWho, strings.TrimSpace(who)) {
		problems = append(problems, field+": who — одно из: «"+strings.Join(actionWho, "», «")+"»")
	}
	kind, _ := record["kind"].(string)
	if !slices.Contains(actionKinds, strings.TrimSpace(kind)) {
		problems = append(problems, field+": kind — одно из: «"+strings.Join(actionKinds, "», «")+"»")
	}
	text, _ := record["text"].(string)
	if m := actionDate.FindStringSubmatch(lowerText(text)); m != nil {
		problems = append(problems, fmt.Sprintf("%s.text: без дат, сроков и частоты («%s…») — назови повод: занятие, сборы, наряд, мероприятие, беседа", field, strings.TrimSpace(m[1])))
	}
	if n := len(wordsOf(text)); n < actionLeast {
		problems = append(problems, fmt.Sprintf("%s.text: слишком коротко — это название направления, а не действие; напиши, что именно сделать, по какому поводу (занятие, сборы, наряд, беседа) и зачем, не меньше %d слов", field, actionLeast))
	}
	if copiedExample(text) {
		problems = append(problems, field+".text: повторяет пример из инструкции — придумай действие по данным этого студента или этой группы")
	}
	return problems
}

// recommendationActionProblems checks the action of every recommendation.
func recommendationActionProblems(output map[string]any) []string {
	items, _ := output["recommendations"].([]any)
	problems := []string{}
	for n, item := range items {
		record, _ := item.(map[string]any)
		problems = append(problems, actionProblems(fmt.Sprintf("recommendations.%d", n+1), record)...)
	}
	return problems
}

// foreignLetterProblems finds a letter of another alphabet (Д14): Qwen wrote
// «в动力» on 27 October.
func foreignLetterProblems(field, text string) []string {
	for _, r := range text {
		if unicode.IsLetter(r) && !unicode.Is(unicode.Cyrillic, r) {
			return []string{field + ": буквы не русского алфавита — пиши только по-русски, коды направлений — только в поле code"}
		}
	}
	return nil
}

// statusMeaningWords are the words that name a status of identity by its
// meaning in the input of v6 (decision Д10), by the normalized name.
var statusMeaningWords = map[string]*regexp.Regexp{
	"сформированная":              regexp.MustCompile(`(?:^|[^а-я])самостоятельн[а-я]*\s+(?:сделал|сделан|выбрал|выбор)|выбор\s+сделан\s+самостоятельн|осознанн[а-я]*\s+выбор`),
	"мораторий (кризис выбора)":   regexp.MustCompile(`в\s+поиске|сомневает|сомнева[а-я]*\s+в\s+выборе|решени[а-я]*\s+(?:ещ[её]\s+)?нет`),
	"навязанная":                  regexp.MustCompile(`под\s+влиянием\s+(?:других|родител|близких|окружени)|без\s+собственного\s+поиска`),
	"неопределенная идентичность": regexp.MustCompile(`не\s+задумыва|не\s+определил|не\s+ищет|не\s+выбрал`),
}

// statusNamedPlain says whether a lower-case text names a status by its name
// or by its meaning.
func statusNamedPlain(lower, name string) bool {
	if statusNamed(lower, name) {
		return true
	}
	words, ok := statusMeaningWords[normalizedText(name)]
	return ok && words.MatchString(lower)
}

// leadingStatusProblems finds, by profession, the leading statuses a
// lower-case text does not name by their name or meaning. The run of 27
// October on Qwen 3.5 9B: told the status alone, the model wrote of one
// profession an attempt and left out the other; it is told the profession
// and, when two statuses are equal, both.
func leadingStatusProblems(payload *object, lower string) []string {
	problems := []string{}
	_, leading := leadingStatuses(payload)
	for n, block := range objectList(payload.values["профессиональная_идентичность"]) {
		if n >= len(leading) {
			break
		}
		missing := []string{}
		for _, name := range leading[n] {
			if !statusNamedPlain(lower, name) {
				missing = append(missing, name)
			}
		}
		if len(missing) == 0 {
			continue
		}
		profession := "по гражданской профессии"
		if strings.HasPrefix(lowerText(objectText(block, "профессия")), "военн") {
			profession = "по военной специальности"
		}
		statuses := "«" + strings.Join(leading[n], "», «") + "»"
		switch len(leading[n]) {
		case 1:
		case 2:
			statuses += " (выражены одинаково — назови оба)"
		default:
			statuses += " (выражены одинаково — назови все)"
		}
		problems = append(problems, fmt.Sprintf("paragraphs.will: не назван ведущий статус %s — %s; во втором абзаце напиши о самоопределении и по военной специальности, и по гражданской профессии, назови статус этим словом или его смыслом из «что_значит»", profession, statuses))
	}
	return problems
}

// checkTestAnswerV6 returns what is wrong with an answer to test/v6: the
// rules of v5, a status by its name or meaning, the actions and no letters of
// another alphabet.
func checkTestAnswerV6(output map[string]any, payload *object) []string {
	texts, problems := paragraphTexts(output)
	names := levelNames(payload)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, ratedLevelProblems(key, texts[key])...)
		problems = append(problems, namedLevelProblems(key, texts[key], names)...)
		problems = append(problems, kindsListProblems(key, texts[key])...)
	}
	problems = append(problems, leadingStatusProblems(payload, normalizedText(joinedParagraphs(texts)))...)
	recommendationTexts(output, texts)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, termProblems(key, texts[key])...)
		problems = append(problems, climateLevelProblems(key, texts[key])...)
		problems = append(problems, foreignLetterProblems(key, texts[key])...)
	}
	problems = append(problems, unfoundStateProblems(texts, payload)...)
	recommendations := recommendationProblems(output, payload, recommendationFloor(payload), 4, situationAllowed(payload))
	problems = append(problems, recommendations...)
	problems = append(problems, fewDirectionsProblems(recommendations, payload)...)
	problems = append(problems, selfDeterminationProblems(output, payload)...)
	problems = append(problems, repeatedCodeProblems(output)...)
	problems = append(problems, recommendationActionProblems(output)...)
	return plainHints(problems)
}

// checkPlatoonAnswerV4 returns what is wrong with an answer to
// test-platoon/v4: the rules of v3, the actions and an action for every
// group of students for separate work.
func checkPlatoonAnswerV4(output map[string]any, payload *object) []string {
	texts, problems := paragraphTexts(output)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, ratedLevelProblems(key, texts[key])...)
		problems = append(problems, kindsListProblems(key, texts[key])...)
	}
	if identity, given := payload.values["профессиональная_идентичность"]; given {
		problems = append(problems, platoonStatusProblems("paragraphs.will", joinedParagraphs(texts), identity, statusNamedPlain)...)
	}
	problems = append(problems, recommendationProblems(output, payload, recommendationLeast, 5, nil)...)
	problems = append(problems, repeatedCodeProblems(output)...)
	problems = append(problems, recommendationActionProblems(output)...)
	recommendationTexts(output, texts)
	problems = append(problems, individualProblems(output, payload, texts)...)
	_, measured := payload.values["климат_взвода"]
	for _, key := range sortedTextKeys(texts) {
		if !measured {
			problems = append(problems, climateProblems(key, texts[key])...)
		}
		problems = append(problems, climateLevelProblems(key, texts[key])...)
		problems = append(problems, termProblems(key, texts[key])...)
		problems = append(problems, foreignLetterProblems(key, texts[key])...)
	}
	problems = append(problems, unfoundStateProblems(texts, payload)...)
	total, _ := payload.values["студентов"].(int)
	problems = append(problems, platoonCountProblems(texts, total)...)
	return plainHints(problems)
}

// individualProblems checks «Кому нужна отдельная работа» of v4: an action for
// every group of the input, each label once, a code that fits the group; the
// texts are added under individual.N.text.
func individualProblems(output map[string]any, payload *object, texts map[string]string) []string {
	groups := map[string][]string{}
	labels := []string{}
	for _, g := range objectList(payload.values["отдельная_работа"]) {
		label := objectText(g, "метка")
		labels = append(labels, label)
		for _, d := range objectList(g.values["подходящие_направления"]) {
			groups[label] = append(groups[label], objectText(d, "код"))
		}
	}
	items, ok := output["individual"].([]any)
	if !ok {
		return []string{"individual: нужен список"}
	}
	problems := []string{}
	if len(items) != len(labels) {
		if len(labels) == 0 {
			problems = append(problems, "individual: во входных данных нет «отдельная_работа» — нужен пустой список")
		} else {
			problems = append(problems, fmt.Sprintf("individual: нужно ровно %d — по одному пункту на каждую группу из «отдельная_работа»: «%s»", len(labels), strings.Join(labels, "», «")))
		}
	}
	seen := map[string]bool{}
	for n, item := range items {
		record, _ := item.(map[string]any)
		field := fmt.Sprintf("individual.%d", n+1)
		label, _ := record["label"].(string)
		label = strings.TrimSpace(label)
		code, _ := record["code"].(string)
		switch {
		case !slices.Contains(labels, label):
			problems = append(problems, fmt.Sprintf("%s: label — метка группы из «отдельная_работа»: «%s»", field, strings.Join(labels, "», «")))
		case seen[label]:
			problems = append(problems, fmt.Sprintf("%s: группа «%s» уже есть — по одному пункту на группу", field, label))
		case !slices.Contains(groups[label], strings.TrimSpace(code)):
			problems = append(problems, fmt.Sprintf("%s: код не из «подходящие_направления» группы «%s» — подходят: %s", field, label, strings.Join(groups[label], ", ")))
		}
		seen[label] = true
		text, _ := record["text"].(string)
		texts[field+".text"] = text
		problems = append(problems, fieldProblems(field+".text", text)...)
		problems = append(problems, actionProblems(field, record)...)
	}
	return problems
}

// fitsGroup says whether a code fits a group of the image: one of the
// directions the input gives the group, or, when it gives none (a platoon),
// a direction of the catalogue for the group.
func fitsGroup(payload *object, group float64, code string) bool {
	for _, g := range objectList(payload.values["образ_защитника"]) {
		if number, _ := numberOf(g.values["номер"]); number == group {
			if fitting, ok := g.values["подходящие_направления"]; ok {
				return slices.Contains(textList(fitting), code)
			}
		}
	}
	for _, d := range objectList(payload.values["каталог_рекомендаций"]) {
		if objectText(d, "код") == code {
			for _, n := range d.values["группы"].([]any) {
				if number, _ := numberOf(n); number == group {
					return true
				}
			}
		}
	}
	return false
}

// cleanUnfitCodes drops a recommendation whose code does not fit its group
// (decision Д13) and then a repeated direction, as long as the fewest the
// instruction asks for are left; the rest stays for the check.
func cleanUnfitCodes(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int) {
	least := recommendationFloor(payload)
	dropped := 0
	if items, ok := output["recommendations"].([]any); ok {
		kept := []any{}
		for _, item := range items {
			record, _ := item.(map[string]any)
			group, isNumber := numberOf(record["group"])
			code, _ := record["code"].(string)
			if isNumber && !fitsGroup(payload, group, strings.TrimSpace(code)) && len(items)-dropped-1 >= least {
				dropped++
				continue
			}
			kept = append(kept, item)
		}
		if dropped > 0 {
			output["recommendations"] = kept
		}
	}
	output, repeated := dropRepeated(output, codeKey, least)
	return output, dropped + repeated
}
