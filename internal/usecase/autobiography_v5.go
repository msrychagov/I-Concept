package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// Since v5 (decided by the user 5 November 2026, Вики/Автобиография/Инструкция
// v5 — черновик.md, after the remarks of the supervisor of 5 November) the
// model writes as a military psychologist: two paragraphs (the experience as
// facts, then the traits of a future reserve specialist), a line for each area
// with experience, and three lists in place of «what to consider»: for the
// psychologist (the family and the upbringing), for the military training (the
// service, the work, the marriage) and the facts (criminal records and foreign
// citizenship). The qualities are named at once: no «по опыту в …», no «это
// показывает», nothing of where they will be of use. The copy of these checks
// is in compare.py: change them together.

// autobiographyAttemptsV5 and autobiographyTokensV5 are D13: five tries, as v4,
// and a longer answer.
const (
	autobiographyAttemptsV5 = 5
	autobiographyTokensV5   = 3000
)

// withoutExperienceV5 is the neutral line the program writes for an area at
// the level «не выражен» (D2): the model does not write about such an area.
// «Не выражен» is given exactly when the student answered «нет» in the
// section, so the line is always true.
var withoutExperienceV5 = map[string]string{
	"sport":        "Опыта в спорте не указал.",
	"creative":     "Опыта в музыке и творчестве не указал.",
	"patriotic":    "Опыта в военно-патриотической деятельности не указал.",
	"science":      "Опыта в науке не указал.",
	"volunteering": "Опыта в волонтёрстве не указал.",
	"languages":    "Иностранных языков не указал.",
}

// autobiographyExamplesV5 are the examples «Так» of the instruction v5, word
// for word; a test compares them with the instruction. Six words in a row from
// one of them in an answer are a copy of the example (D10).
var autobiographyExamplesV5 = []string{
	"Быстро ориентируется в игре, умеет действовать заодно с командой и мгновенно принимать решения",
	"Два года занимался в военно-патриотическом клубе и на полевых сборах, где был заместителем командира взвода; имеет навыки строевой подготовки, радиосвязи и оказания первой помощи, опыт руководства взводом на учебных занятиях клуба",
	"Военное дело ему интересно давно и всерьёз. Привык к порядку и чётким командам, умеет руководить сверстниками и отвечать за них, в учебной обстановке не теряет собранности",
	"Бокс развил быстроту реакции, выносливость и умение держать удар",
	"Испанский — DELE B2: свободно читает и объясняется на учебные и бытовые темы",
	"Воспитывался бабушкой. В беседе уточнить, на кого из взрослых он привык опираться, и обратить внимание, как он входит в новый коллектив",
	"Мать — военнослужащая. В беседе уточнить, как студент сам пришёл к решению учиться военному делу и чего ждёт от службы",
	"Работал электромонтёром: знает электрооборудование и правила безопасности при работе с ним",
	"Женат, растит двоих детей: привык отвечать не только за себя, но и за близких",
	"У дяди есть судимость",
}

// exampleRunV5 is the number of words in a row that make a copy.
const exampleRunV5 = 6

// machinePhraseV5 is the machine way of drawing a quality from an activity
// (B1): in every field.
var machinePhraseV5 = regexp.MustCompile(`(?i)по опыту|(это|что) показыва[а-яё]*|свидетельству[а-яё]*|(^|[^а-яё])говорит о|указывает на|подтвержда[а-яё]*`)

// usefulWhereV5 tells where a quality will be of use (B2): the paragraphs and
// the lines of the areas name the ability itself.
var usefulWhereV5 = regexp.MustCompile(`(?i)пригод[а-яё]*|(^|[^а-яё])полез(ен|на|ны|ным|ной|ными)([^а-яё]|$)|(^|[^а-яёa-z])вуц([^а-яёa-z]|$)|я-концепц[а-яё]*`)

// potentialWordV5 is a level of the potential in the line of an area (D11).
// The line of the languages may say «на среднем уровне» of the language
// itself: there only the potential is looked for.
var potentialWordV5 = regexp.MustCompile(`(?i)потенциал[а-яё]*`)
var levelWordV5 = regexp.MustCompile(`(?i)(высок|средн|начальн)[а-яё]* уров[а-яё]*`)

// sentenceEndV5 ends a sentence: a full stop between digits («7.5») does not.
var sentenceEndV5 = regexp.MustCompile(`[.!?…]+(\s+|$)`)

// stillAliveV5 is «нет в живых»: a fact of the family, not an absence.
var stillAliveV5 = regexp.MustCompile(`(?i)нет в живых`)

// askWordV5 is what the psychologist is to clarify or to look at (D6).
var askWordV5 = regexp.MustCompile(`(?i)обрат[а-яё]* внимани|уточн|выясн|в беседе`)

// familyWordV5 is the family or the upbringing in a point for the psychologist.
var familyWordV5 = regexp.MustCompile(`(?i)отц|отец|мать|матер|родител|бабушк|дедушк|опекун|при[её]мн|детск[а-яё]* дом|интернат|воспитыва|воспитан|семь|семей|брат|сестр|родствен|близк`)

// recordWordV5 is a criminal record or a foreign citizenship: only a fact of
// «Что учесть».
var recordWordV5 = regexp.MustCompile(`(?i)судим|уголовн|гражданств|подданств`)

// labelWordV5 is a label, a clinical word, a prognosis, a cause drawn from the
// family or a word of prohibition 5 in a point for the psychologist (B4): the
// circumstance by itself tells nothing of the man.
var labelWordV5 = regexp.MustCompile(`(?i)риск|травм|дефицит|неблагополуч|неполн|сирот|депривац|девиа|отклонени|нарушени|асоциал|маргинал|запущен|безотцов|ущерб|комплекс|агресс|конфликтн|замкнут|тревожн|уязвим|нестабильн|неустойчив|надёжн|надежн|проблем|негатив|недостат|нехватк|лишён|лишен|склон|предрасполож|пригод|лояльн|диагноз|расстройств|патолог|психиатр|прогноз|вероятн|скорее всего|наверняка|трудност|сложност|мо(жет|гут) (испыт|возникн|прояв|столкн|привест|привести)|будет испыт|поэтому|из-за|вследстви|следовательно|в результате|к сожалению|несмотря на|тем не менее|однако`)

// trainingWordV5 is the service, the work or the family in a point for the
// military training (B5); the positions of the work are added from the input.
var trainingWordV5 = regexp.MustCompile(`(?i)служ|арми|войск|звани|сержант|ефрейтор|рядов|механик|работ|жен|супруг|ребён|ребен|дет|сын|доч|семь|семей`)

// upbringingWordV5 is the upbringing: it belongs to the psychologist.
var upbringingWordV5 = regexp.MustCompile(`(?i)воспитывал(ся|и|а)|воспитан|родител|детск[а-яё]* дом|интернат|опекун`)

// planningWordV5 is how to plan the time or what to recommend in a point for
// the military training: the point tells what he knows and is used to. «Нагрузка»
// itself passes: «привык к физическим нагрузкам» (decided by the user after the
// run of 5 November 2026).
var planningWordV5 = regexp.MustCompile(`(?i)при планировани|при распределени|допуск|влия|рекоменд|следует|необходимо|пригоден|непригод|пригодност`)

// Checks added after the run of 6 November 2026 (decided by the user); the
// terms of the activities are in autobiography_v5_terms.go.

// absenceV5 is what the student has not, in the words the list of v4 lacks:
// «До поступления не работал», «школа без профиля».
var absenceV5 = regexp.MustCompile(`(?i)(^|[^а-яё])(не работал[а-яё]*|без профил[а-яё]*|без углубл[её]нн[а-яё]*)([^а-яё]|$)`)

// lackingV5 is what the student has not yet or what will help him later, in
// the paragraphs and the lines of the areas: «военное дело пока знает мало»,
// «новая область», «поможет освоить».
var lackingV5 = regexp.MustCompile(`(?i)(знает|умеет|владеет|разбирается)\s+(пока\s+)?мало|пока мало|(^|[^а-яё])пока не([^а-яё]|$)|нов(ая|ое|ой|ую)\s+(для него\s+)?(област|дело|сфер)[а-яё]*|(^|[^а-яё])помо(жет|гут)([^а-яё]|$)`)

// connectorV5 is a machine link from a fact to a quality: «показывают
// готовность», «что поддерживает работу с информацией».
var connectorV5 = regexp.MustCompile(`(?i)показыва[а-яё]*\s+(готовност|способност|умени|стремлени|интерес|склонност|навык|качеств)[а-яё]*|что\s+(поддерживает|помогает|позволяет)`)

// tripWordV5 is a trip abroad: a fact, not an activity, so no quality comes
// from it.
var tripWordV5 = regexp.MustCompile(`(?i)поездк|поездок|за границ|за рубеж|выезд|выезжал`)

// qualityHintV5 is a quality or a quality drawn from something, next to a trip.
var qualityHintV5 = regexp.MustCompile(`(?i)собранн|готовност|способн|развил|развит|сформир|показыва|(^|[^а-яё])(да[её]т|дают|дал[аио]?)([^а-яё]|$)`)

// serviceKindV5, familyKindV5 find the service and the family in a point for
// the military training; the work is «работал» or a position of the input.
// One point is one circumstance.
var serviceKindV5 = regexp.MustCompile(`(?i)(^|[^а-яё])(служ|арми|армей|войск|сержант|ефрейтор|рядов)`)
var workKindV5 = regexp.MustCompile(`(?i)(^|[^а-яё])работал`)
var familyKindV5 = regexp.MustCompile(`(?i)(^|[^а-яё])(женат|жена|жены|жене|женой|супруг[а-яё]*|ребён[а-яё]*|ребен[а-яё]*|дет(и|ей|ьми|ям|ях)|сын(а|у|ом|овья|овей)?|доч(ь|ери|ерью|ерей)|семь[яиюеё]|семей[а-яё]*)([^а-яё]|$)`)

// activeV5 tells whether the student has an activity or a hobby: without them
// the only paragraph is noActivities.
func activeV5(payload *object) bool {
	levels, _ := payload.values["уровни"].(*object)
	return len(textList(payload.values["увлечения"])) > 0 || levels == nil || len(emptyAreas(payload)) < len(levels.keys)
}

// experienceAreasV5 returns the areas of the input above «не выражен», in
// the order of the input.
func experienceAreasV5(payload *object) []string {
	areas := []string{}
	if levels, ok := payload.values["уровни"].(*object); ok {
		for _, key := range levels.keys {
			if level, _ := levels.values[key].(string); level != "не выражен" {
				areas = append(areas, key)
			}
		}
	}
	return areas
}

// recordsV5 reads a list of records of the input, as it is built or as it is read back.
func recordsV5(value any) []*object {
	records := []*object{}
	list, _ := value.([]any)
	for _, item := range list {
		if record, ok := item.(*object); ok {
			records = append(records, record)
		}
	}
	return records
}

// fieldV5 is a text field of a record of the input.
func fieldV5(record *object, key string) string {
	if record == nil {
		return ""
	}
	text, _ := record.values[key].(string)
	return text
}

// relativesV5 are the relatives of the input.
func relativesV5(payload *object) []*object {
	circle, _ := payload.values["семья"].(*object)
	if circle == nil {
		return nil
	}
	return recordsV5(circle.values["родственники"])
}

// parentKindsV5 are the relatives who bring a child up.
var parentKindsV5 = []string{"Отец", "Мать", "Опекун, приёмный родитель"}

// psychologistCircumstancesV5 tells whether the input has a circumstance for
// the psychologist (D5): not brought up by both parents, nothing known of a
// parent, a parent dead, a relative in the military service.
func psychologistCircumstancesV5(payload *object) bool {
	circle, _ := payload.values["семья"].(*object)
	if upbringing := fieldV5(circle, "кто_воспитывал"); upbringing != "" && upbringing != "Оба родителя" {
		return true
	}
	for _, relative := range relativesV5(payload) {
		parent := slices.Contains(parentKindsV5, fieldV5(relative, "кем_приходится"))
		if parent && (fieldV5(relative, "сведения") == "Сведений не имею" || fieldV5(relative, "жив") == "Нет") || fieldV5(relative, "сфера") == "Военная служба" {
			return true
		}
	}
	return false
}

// trainingCircumstancesV5 tells whether the input has something for the
// military training (D7): the service, a work before the university, a wife or
// children.
func trainingCircumstancesV5(payload *object) bool {
	service, _ := payload.values["служба"].(*object)
	marital, _ := payload.values["семейное_положение"].(*object)
	children := 0
	if marital != nil {
		switch n := marital.values["детей"].(type) {
		case int:
			children = n
		case float64:
			children = int(n)
		}
	}
	return fieldV5(service, "служил") == "Да" || len(recordsV5(payload.values["работа_до_вуза"])) > 0 || fieldV5(marital, "статус") == "Женат" || children > 0
}

// recordFactsV5 are the facts of «Что учесть» in the input (D8).
func recordFactsV5(payload *object) (own, convicted, foreign bool) {
	record, _ := payload.values["уголовная_ответственность"].(string)
	own = record == "привлекался"
	for _, relative := range relativesV5(payload) {
		convicted = convicted || fieldV5(relative, "судимость") == "Да"
		foreign = foreign || fieldV5(relative, "иностранное_гражданство") == "Да"
	}
	return own, convicted, foreign
}

// workStemsV5 are the stems of the positions before the university, without
// the words of the position in the service: a mechanic-driver of the army
// may be named in the paragraphs.
func workStemsV5(payload *object) []string {
	service, _ := payload.values["служба"].(*object)
	position := strings.ToLower(fieldV5(service, "должность"))
	stems := []string{}
	for _, stem := range jobStems(payload) {
		if position == "" || !strings.Contains(position, stem) {
			stems = append(stems, stem)
		}
	}
	return stems
}

// languageStemV5 is the beginning of the name of a language that every form of
// it holds: «англий» for «английский» and «по-английски».
func languageStemV5(name string) string {
	letters := []rune(strings.ToLower(strings.TrimSpace(name)))
	return string(letters[:min(len(letters), max(4, len(letters)-4))])
}

// latinLookalikesV5 turns the Cyrillic letters that look Latin into Latin ones,
// so that «В1» is «B1» and «7,5» is «7.5».
var latinLookalikesV5 = strings.NewReplacer("а", "a", "в", "b", "с", "c", "е", "e", "о", "o", "р", "p", "х", "x", "к", "k", "м", "m", "н", "h", "т", "t", ",", ".")

func examResultV5(text string) string {
	return latinLookalikesV5.Replace(strings.ToLower(strings.Join(strings.Fields(text), "")))
}

// languagesV5 are the names of the languages and the results of their exams.
func languagesV5(payload *object) (names, results []string) {
	for _, language := range recordsV5(payload.values["языки"]) {
		if name := fieldV5(language, "язык"); name != "" {
			names = append(names, name)
		}
		for _, practice := range recordsV5(language.values["практика"]) {
			if result := fieldV5(practice, "результат"); result != "" {
				results = append(results, result)
			}
		}
	}
	return names, results
}

// wordsOfV5 are the words of a text in lower case.
func wordsOfV5(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// exampleRunsV5 are all the runs of six words of the examples.
var exampleRunsV5 = func() map[string]bool {
	runs := map[string]bool{}
	for _, example := range autobiographyExamplesV5 {
		words := wordsOfV5(example)
		for n := 0; n+exampleRunV5 <= len(words); n++ {
			runs[strings.Join(words[n:n+exampleRunV5], " ")] = true
		}
	}
	return runs
}()

// copiesExampleV5 tells whether a text repeats six words in a row of an example.
func copiesExampleV5(text string) bool {
	words := wordsOfV5(text)
	for n := 0; n+exampleRunV5 <= len(words); n++ {
		if exampleRunsV5[strings.Join(words[n:n+exampleRunV5], " ")] {
			return true
		}
	}
	return false
}

// sentencesV5 counts the sentences of a text.
func sentencesV5(text string) int {
	count := 0
	for _, part := range sentenceEndV5.Split(strings.TrimSpace(text), -1) {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

// foundV5 is a word the rule found, without the signs around it.
func foundV5(text string) string {
	return strings.ToLower(strings.Trim(text, " ,.;:!?«»()—"))
}

// commonProblemsV5 are the rules of every text of the answer: no machine
// phrase, no appointment (prohibition 4), no copy of an example where examples
// is set. The points for the psychologist and the military training may repeat
// their examples (decided by the user after the run of 5 November 2026), and
// so may the line of the languages (6 November: «читает и объясняется на
// учебные и бытовые темы» is how a level is told). A problem names the word of
// the rule and how to write instead, never the text.
func commonProblemsV5(where, text string, examples bool) []string {
	problems := []string{}
	if found := machinePhraseV5.FindString(text); found != "" {
		problems = append(problems, fmt.Sprintf("%s: машинный оборот («%s») — назови качество сразу: «вынослив, быстро реагирует», без «по опыту в …» и «это показывает»", where, foundV5(found)))
	}
	if found := postAppointment.FindString(troopsPurpose.ReplaceAllString(strings.ToLower(text), "")); found != "" {
		problems = append(problems, fmt.Sprintf("%s: назначение, зачисление или отчисление («%s») — запрет 4, о должностях и допуске не пиши", where, found))
	}
	if examples && copiesExampleV5(text) {
		problems = append(problems, where+": повторяет пример из инструкции — напиши по сведениям этого студента")
	}
	return problems
}

// workSkillV5 is a skill or a quality next to a position before the
// university: «освоил работу в потоке» as well as «навыки общения».
var workSkillV5 = regexp.MustCompile(`(?i)освоил|научил|привык|приобр[её]л|развил|знает|владеет`)

// qualityProblemsV5 are the rules of the paragraphs and the lines of the
// areas, where the qualities are: from the activities only, nothing of where
// they will be of use. The work before the university is in a paragraph only
// as a fact, «До вуза работал …», with no skill or quality in the sentence
// (decided by the user after the run of 5 November 2026); a line of an area
// has no work at all (work is false).
func qualityProblemsV5(where, text string, jobs []string, work bool) []string {
	problems := []string{}
	lower := strings.ToLower(text)
	if found := lifeCircumstance.FindStringSubmatch(text); found != nil {
		word := found[0]
		if found[2] != "" {
			word = found[2]
		}
		problems = append(problems, fmt.Sprintf("%s: семья, брак, гражданство или судимость («%s») — здесь только занятия; семья — в psychologist, брак — в training, судимости и гражданство — в consider", where, strings.ToLower(strings.TrimSpace(word))))
	}
	if found := usefulWhereV5.FindString(text); found != "" {
		problems = append(problems, fmt.Sprintf("%s: где пригодится («%s») — не пиши, где это пригодится, — назови саму способность", where, foundV5(found)))
	}
	if found := connectorV5.FindString(text); found != "" {
		problems = append(problems, fmt.Sprintf("%s: машинный оборот («%s») — назови качество сразу: «вынослив, быстро реагирует», без «показывает готовность» и «что позволяет»", where, foundV5(found)))
	}
	if found := lackingV5.FindString(text); found != "" {
		problems = append(problems, fmt.Sprintf("%s: чего ещё нет или что поможет потом («%s») — пиши только о том, что у студента есть сейчас", where, foundV5(found)))
	}
	for _, part := range strings.FieldsFunc(lower, func(r rune) bool { return strings.ContainsRune(".;!?", r) }) {
		if found := tripWordV5.FindString(part); found != "" && (skillWord.MatchString(part) || qualityHintV5.MatchString(part)) {
			problems = append(problems, fmt.Sprintf("%s: качества из поездок («%s») — поездку можно назвать фактом, качества называй только по занятиям", where, found))
			break
		}
	}
	for _, part := range strings.FieldsFunc(lower, func(r rune) bool { return strings.ContainsRune(".;!?", r) }) {
		if found := schooling.FindString(part); found != "" && !schoolActivity.MatchString(part) && (skillWord.MatchString(part) || inference.MatchString(part)) {
			problems = append(problems, fmt.Sprintf("%s: качества из учёбы («%s») — качества называй только по занятиям и увлечениям", where, strings.Fields(found)[0]))
			break
		}
	}
	for _, part := range strings.FieldsFunc(lower, func(r rune) bool { return strings.ContainsRune(".;!?", r) }) {
		if !slices.ContainsFunc(jobs, func(stem string) bool { return strings.Contains(part, stem) }) {
			continue
		}
		switch {
		case !work:
			problems = append(problems, where+": работа до вуза — о ней пиши в training: чем она полезна в военной подготовке")
		case !workFact.MatchString(part) || skillWord.MatchString(part) || workSkillV5.MatchString(part):
			problems = append(problems, where+": работа до вуза не фактом — в абзаце только «До вуза работал …», без навыков и качеств; что из неё полезно — в training")
		default:
			continue
		}
		break
	}
	return problems
}

// absenceProblemsV5 finds the absence of something in a text; the clauses
// that tell nothing is known of a relative and «нет в живых» are facts.
func absenceProblemsV5(where, text string, empty []string) []string {
	text = stillAliveV5.ReplaceAllString(withoutUnknownRelatives(text), "")
	problems := absenceProblems(where, text, empty)
	if found := absenceV5.FindStringSubmatch(text); found != nil {
		problems = append(problems, fmt.Sprintf("%s: написано об отсутствии («%s») — убери это, пиши только о том, что у студента есть", where, strings.ToLower(found[2])))
	}
	return problems
}

// areasProblemsV5 checks the lines of the areas (B3): exactly the areas with
// experience, in their order, one or two sentences each, no level of the
// potential, every language and every result of an exam named.
func areasProblemsV5(value any, payload *object, empty, jobs []string) []string {
	list, ok := value.([]any)
	if !ok {
		return []string{"areas не список записей {area, text}"}
	}
	want := experienceAreasV5(payload)
	got := []string{}
	problems := []string{}
	for _, item := range list {
		record, isRecord := item.(map[string]any)
		area, areaText := record["area"].(string)
		text, textText := record["text"].(string)
		if !isRecord || !areaText || !textText {
			return []string{"areas не список записей {area, text}"}
		}
		area = strings.ToLower(strings.TrimSpace(area))
		got = append(got, area)
		where := fmt.Sprintf("areas, «%s»", area)
		if strings.TrimSpace(text) == "" {
			problems = append(problems, where+": пустой текст")
			continue
		}
		if n := sentencesV5(text); n > 2 {
			problems = append(problems, fmt.Sprintf("%s: предложений %d — нужно одно, не больше двух", where, n))
		}
		if found := potentialWordV5.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: уровень потенциала («%s») — уровень не называй, его показывает программа", where, strings.ToLower(found)))
		} else if found := levelWordV5.FindString(text); found != "" && area != "иностранные языки" {
			problems = append(problems, fmt.Sprintf("%s: уровень потенциала («%s») — уровень не называй, его показывает программа", where, strings.ToLower(found)))
		}
		problems = append(problems, commonProblemsV5(where, text, area != "иностранные языки")...)
		problems = append(problems, qualityProblemsV5(where, text, jobs, false)...)
		problems = append(problems, absenceProblemsV5(where, text, empty)...)
		if area == "иностранные языки" {
			names, results := languagesV5(payload)
			lower := strings.ToLower(text)
			for _, name := range names {
				if !strings.Contains(lower, languageStemV5(name)) {
					problems = append(problems, fmt.Sprintf("%s: не назван язык «%s» — назови каждый язык и результат экзамена", where, strings.ToLower(name)))
				}
			}
			for _, result := range results {
				if !strings.Contains(examResultV5(text), examResultV5(result)) {
					problems = append(problems, fmt.Sprintf("%s: не назван результат экзамена «%s» — назови каждый язык и результат экзамена", where, result))
				}
			}
		}
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		names := "ни одного: оставь список пустым"
		if len(want) > 0 {
			names = strings.Join(want, ", ")
		}
		problems = append([]string{fmt.Sprintf("areas: направления не те или в другом порядке — пиши строки только для направлений: %s", names)}, problems...)
	}
	return problems
}

// psychologistProblemsV5 checks the points for the psychologist (B4, D5, D6).
func psychologistProblemsV5(points []any, payload *object) []string {
	problems := []string{}
	circumstances := psychologistCircumstancesV5(payload)
	switch {
	case len(points) > 3:
		problems = append(problems, "psychologist: пунктов больше трёх")
	case !circumstances && len(points) > 0:
		problems = append(problems, "psychologist: обстоятельств для психолога нет: «Воспитывался обоими родителями» не пиши, оставь список пустым")
	case circumstances && len(points) == 0:
		problems = append(problems, "psychologist: пусто — напиши, кто воспитывал студента или о ком из родителей нет сведений, и что уточнить в беседе")
	}
	for n, point := range points {
		text, where := point.(string), fmt.Sprintf("psychologist, пункт %d", n+1)
		if !askWordV5.MatchString(text) {
			problems = append(problems, where+": нет того, что уточнить — напиши, что уточнить в беседе и на что обратить внимание, а не свойство студента")
		}
		if !familyWordV5.MatchString(text) {
			problems = append(problems, where+": это не о семье и воспитании — служба, работа, брак — в training")
		}
		if found := recordWordV5.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: судимость или гражданство («%s») — это фактом в consider", where, strings.ToLower(found)))
		}
		if found := labelWordV5.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: ярлык или прогноз («%s») — без ярлыков и прогнозов: напиши, на что обратить внимание и что уточнить", where, strings.ToLower(found)))
		}
		problems = append(problems, commonProblemsV5(where, text, false)...)
		problems = append(problems, absenceProblemsV5(where, text, nil)...)
	}
	return problems
}

// trainingProblemsV5 checks the points for the military training (B5, D7).
func trainingProblemsV5(points []any, payload *object) []string {
	problems := []string{}
	circumstances := trainingCircumstancesV5(payload)
	switch {
	case len(points) > 3:
		problems = append(problems, "training: пунктов больше трёх")
	case !circumstances && len(points) > 0:
		problems = append(problems, "training: студент не служил, не работал, не женат и детей нет — оставь список пустым")
	case circumstances && len(points) == 0:
		problems = append(problems, "training: пусто — напиши, чем для военной подготовки полезны его служба, работа до вуза или семейное положение")
	}
	jobs := jobStems(payload)
	if kinds := trainingKindsInputV5(payload); len(points) <= 3 && len(points) > 0 && len(points) < kinds {
		problems = append(problems, fmt.Sprintf("training: пунктов %d, а обстоятельств %d — по пункту на службу, работу до вуза и семью", len(points), kinds))
	}
	work := workStemsV5(payload)
	for n, point := range points {
		text, where := point.(string), fmt.Sprintf("training, пункт %d", n+1)
		lower := strings.ToLower(text)
		if kinds := trainingKindsV5(text, work); len(kinds) > 1 {
			problems = append(problems, fmt.Sprintf("%s: в одном пункте сразу %s и %s — по пункту на каждое обстоятельство", where, strings.Join(kinds[:len(kinds)-1], ", "), kinds[len(kinds)-1]))
		}
		about := trainingWordV5.MatchString(text)
		for _, stem := range jobs {
			about = about || strings.Contains(lower, stem)
		}
		if !about {
			problems = append(problems, where+": в training — только служба, работа до вуза и семейное положение")
		}
		if found := upbringingWordV5.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: воспитание («%s») — это в psychologist", where, strings.ToLower(found)))
		}
		if found := recordWordV5.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: судимость или гражданство («%s») — это фактом в consider", where, strings.ToLower(found)))
		}
		if found := planningWordV5.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: планирование, допуск или рекомендация («%s») — напиши, что он знает и умеет и к чему привык, без планирования времени, допусков и должностей", where, strings.ToLower(found)))
		}
		problems = append(problems, commonProblemsV5(where, text, false)...)
		problems = append(problems, absenceProblemsV5(where, text, nil)...)
	}
	return problems
}

// considerProblemsV5 checks «Что учесть» (B6, D8): only the facts of criminal
// records and foreign citizenship, each fact of the input named.
func considerProblemsV5(points []any, payload *object, empty []string) []string {
	problems := []string{}
	own, convicted, foreign := recordFactsV5(payload)
	switch {
	case len(points) > attentionLimit:
		problems = append(problems, "consider: пунктов больше пяти")
	case !own && !convicted && !foreign && len(points) > 0:
		problems = append(problems, "consider: судимостей и иностранного гражданства нет — оставь список пустым")
	}
	problems = append(problems, considerProblems("consider", points, payload)...)
	hasConviction, hasCitizenship := false, false
	for n, point := range points {
		text, where := point.(string), fmt.Sprintf("consider, пункт %d", n+1)
		lower := strings.ToLower(text)
		if !recordWordV5.MatchString(text) {
			problems = append(problems, where+": в consider — только судимости и иностранное гражданство: семья — в psychologist, служба, работа и брак — в training")
		}
		hasConviction = hasConviction || strings.Contains(lower, "судим")
		hasCitizenship = hasCitizenship || strings.Contains(lower, "гражданств") || strings.Contains(lower, "подданств")
		problems = append(problems, commonProblemsV5(where, text, true)...)
		problems = append(problems, absenceProblemsV5(where, text, empty)...)
	}
	if convicted && !hasConviction {
		problems = append(problems, "consider: не названа судимость родственника — напиши, у кого из близких есть судимость")
	}
	if foreign && !hasCitizenship {
		problems = append(problems, "consider: не названо иностранное гражданство родственника — напиши, у кого из близких оно есть")
	}
	return problems
}

// trainingKindsInputV5 is the number of the circumstances of the military
// training in the input: the service, the work, the wife or children.
func trainingKindsInputV5(payload *object) int {
	service, _ := payload.values["служба"].(*object)
	marital, _ := payload.values["семейное_положение"].(*object)
	kinds := 0
	if fieldV5(service, "служил") == "Да" {
		kinds++
	}
	if len(recordsV5(payload.values["работа_до_вуза"])) > 0 {
		kinds++
	}
	children := 0
	if marital != nil {
		switch n := marital.values["детей"].(type) {
		case int:
			children = n
		case float64:
			children = int(n)
		}
	}
	if fieldV5(marital, "статус") == "Женат" || children > 0 {
		kinds++
	}
	return kinds
}

// trainingKindsV5 are the circumstances one point for the military training
// tells of: «служба», «работа», «семья».
func trainingKindsV5(text string, work []string) []string {
	kinds := []string{}
	if serviceKindV5.MatchString(text) {
		kinds = append(kinds, "служба")
	}
	lower := strings.ToLower(text)
	if workKindV5.MatchString(text) || slices.ContainsFunc(work, func(stem string) bool { return strings.Contains(lower, stem) }) {
		kinds = append(kinds, "работа")
	}
	if familyKindV5.MatchString(text) {
		kinds = append(kinds, "семья")
	}
	return kinds
}

// checkAutobiographyAnswerV5 returns what is wrong with an answer to the
// instruction v5 that a program can find.
func checkAutobiographyAnswerV5(output map[string]any, payload *object) []string {
	problems := []string{}
	empty := emptyAreas(payload)
	jobs := workStemsV5(payload)
	active := activeV5(payload)
	paragraphs, ok := stringList(output["paragraphs"])
	switch {
	case !ok:
		problems = append(problems, "paragraphs не список строк")
	case !active && (len(paragraphs) != 1 || sentence(paragraphs[0].(string)) != noActivities):
		problems = append(problems, fmt.Sprintf("paragraphs: занятий нет, а вывод не «%s»", noActivities))
	case active && len(paragraphs) != 2:
		problems = append(problems, fmt.Sprintf("paragraphs: абзацев %d, а нужно ровно два: опыт фактами и черты будущего военного специалиста", len(paragraphs)))
	}
	if ok && active {
		for n, item := range paragraphs {
			text, where := item.(string), fmt.Sprintf("paragraphs, абзац %d", n+1)
			if strings.TrimSpace(text) == "" {
				problems = append(problems, where+": пустой")
				continue
			}
			problems = append(problems, commonProblemsV5(where, text, true)...)
			problems = append(problems, qualityProblemsV5(where, text, jobs, true)...)
			if n == 0 {
				problems = append(problems, termProblemsV5(where, text, activityTermsV5(payload))...)
			}
			problems = append(problems, absenceProblemsV5(where, text, empty)...)
		}
	}
	problems = append(problems, areasProblemsV5(output["areas"], payload, empty, jobs)...)
	if points, ok := stringList(output["psychologist"]); ok {
		problems = append(problems, psychologistProblemsV5(points, payload)...)
	} else {
		problems = append(problems, "psychologist не список строк")
	}
	if points, ok := stringList(output["training"]); ok {
		problems = append(problems, trainingProblemsV5(points, payload)...)
	} else {
		problems = append(problems, "training не список строк")
	}
	if points, ok := stringList(output["consider"]); ok {
		problems = append(problems, considerProblemsV5(points, payload, empty)...)
	} else {
		problems = append(problems, "consider не список строк")
	}
	return problems
}
