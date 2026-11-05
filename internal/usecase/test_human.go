package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"vucdiagnostic/internal/promptbank"
)

// The input and the checks of test/v7 and test-platoon/v5 (Вики/План правок по
// замечаниям 5 ноября.md, part А, and Вики/Тестирование/Инструкция v7.md,
// Инструкция по взводу v5.md, approved by the user on 5 November 2026). The
// supervisor read the texts of v6 as «статистика»: the model retold the labels
// of the input — «расхождения оптимальны», «ведущий статус», «образ принят»,
// the kinds of activity by name. The input of v7 gives what the labels mean in
// plain phrases; the check rejects the labels and tells what to write instead.

// humanVersion is, by task, the first version whose input is in plain phrases.
var humanVersion = map[string]int{testPortraitTask: 7, testPlatoonTask: 5}

func humanInput(task, version string) bool {
	n, _ := promptbank.Number(version)
	return n >= humanVersion[task]
}

// What the situations of a group of the image mean for the student (table of
// Инструкция v7.md); the input of v6 has them under these names.
var humanSituations = map[string]string{
	situationText[situationAchieved]:    "это для него важно и уже получается",
	situationText[situationFar]:         "это для него важно, но до этого пока далеко",
	situationText[situationNotAccepted]: "пока не считает это для себя важным",
	plainSituation:                      "ставит себе планку ниже, чем оценивает себя сейчас",
}

// The gap between the real and the ideal self, the self-esteem, the rules of
// Danilova and the congruence of Rogers in plain phrases.
var (
	humanGaps = map[string]string{
		"образы совпадают":        "каким хочет быть, таким себя уже и считает",
		"оптимальное расхождение": "видит, куда расти, и цель ему по силам",
		"выраженное расхождение":  "хочет заметно измениться, цель пока далека",
		plainSituation: "ставит себе планку ниже, чем оценивает себя сейчас",
	}
	humanSelf = map[string]string{
		"низкая самооценка":  "в себе пока не уверен",
		"средняя самооценка": "оценивает себя сдержанно",
		"высокая самооценка": "в себе уверен",
	}
	humanPressure  = "чувствует, что от него ждут большего, чем он может сейчас"
	humanAdequacy  = map[string]string{adequacyTexts["same"]: "считает, что его видят таким, какой он есть"}
	humanCongruent = map[string]string{
		congruenceTexts["high"]:    "представления о себе нынешнем и желаемом складываются в цельную картину",
		congruenceTexts["partial"]: "представления о себе нынешнем и желаемом складываются лишь отчасти",
		congruenceTexts["low"]:     "представления о себе нынешнем и желаемом пока не складываются: он ещё ищет себя",
		congruenceTexts["none"]:    "по ответам сказать нельзя",
		plainValues["Я-идеальное одинаково по всем качествам — согласованность не определяется."]: "хочет стать одинаково хорошим во всех качествах сразу",
		plainValues["Я-реальное одинаково по всем качествам — согласованность не определяется."]:  "себя сейчас оценивает одинаково во всех качествах — по ответам сказать нельзя",
	}
)

// selfCase is a way the student chose his military speciality or his civil
// profession: the phrase of the input and the words that tell it in a text.
type selfCase struct {
	phrase string
	words  *regexp.Regexp
}

var (
	selfFormed = selfCase{"выбрал сам, после раздумий, и связывает с выбором свои планы",
		regexp.MustCompile(`выбрал[а-я]*\s+(?:е[её]\s+|его\s+)?сам|сам[а-я]*\s+выбрал|самостоятельн|определил|осознанн[а-я]*\s+выбор`)}
	selfSearching = selfCase{"ещё выбирает: сомневается, сравнивает, решения нет",
		regexp.MustCompile(`выбира|сомнева|в\s+поиске|ищ[еу]т`)}
	selfImposed = selfCase{"выбор сделан под влиянием других, без своего поиска",
		regexp.MustCompile(`под\s+влиянием|не\s+сам|чуж[а-я]*\s+выбор|по\s+чужому|за\s+него`)}
	selfIdle = selfCase{"ещё не определился и пока всерьёз не задумывается о выборе",
		regexp.MustCompile(`не\s+определил|не\s+задумыва|не\s+выбрал|не\s+реши`)}
	selfUndecided = selfCase{"ещё не определился",
		regexp.MustCompile(`не\s+определил|не\s+выбрал|не\s+реши|ясного\s+выбора|выбира|сомнева`)}
)

// selfCases are the cases by the name of a status of the input of v6.
var selfCases = map[string]selfCase{
	"сформированная":              selfFormed,
	"мораторий (кризис выбора)":   selfSearching,
	"навязанная":                  selfImposed,
	"неопределённая идентичность": selfIdle,
}

// selfCaseOf is the case of a phrase of the input of v7.
func selfCaseOf(phrase string) (selfCase, bool) {
	for _, c := range []selfCase{selfFormed, selfSearching, selfImposed, selfIdle, selfUndecided} {
		if c.phrase == phrase {
			return c, true
		}
	}
	return selfCase{}, false
}

// The four kinds of activity as the sides of the student's life in the ВУЦ,
// with what each is by the tables 2–5 of the first chapter.
var humanSides = map[string]struct{ name, what string }{
	"Учебная деятельность":          {"освоение военного дела", "занятия и сборы: знания и навыки, самостоятельность, порядок на занятиях, слаженность в коллективе"},
	"Воспитательная деятельность":   {"жизнь во взводе и воспитательные мероприятия", "отношения в коллективе, инициатива, стремление воспитывать себя, поступки"},
	"Повседневная деятельность":     {"военный быт", "привыкание к распорядку, наряду, порядку и требованиям; дисциплина; как переносит трудности"},
	"Просветительская деятельность": {"разговор о смысле службы", "доверительные беседы, личный пример старших, разбор своего опыта, стремление к саморазвитию"},
}

// humanValues are the texts of the input of v7 replaced as a whole: the
// conditions under plain names, the two directions of the catalogue whose
// names say «наставник», a criterion that names a kind of activity.
var humanValues = map[string]string{
	"Понятность и согласованность требований":                                        "Ясность требований",
	"Распределение нагрузки и ответственности":                                       "Справедливое распределение нагрузки и ответственности",
	"Возможности значимой деятельности":                                              "Дело, где можно проявить себя",
	"Доступность ресурсов":                                                           "Доступ к нужным материалам",
	"Осмысление норм и ценностей":                                                    "Разговор о смысле норм и ценностей",
	"Обратная связь и работа с ошибкой":                                              "Разбор ошибок",
	"Рефлексия и самостоятельный выбор":                                              "Возможность самому осмыслить и выбрать",
	"Доверие и уважение достоинства":                                                 "Доверие и уважение к нему",
	"Субъектность и поддержка инициативы":                                            "Свобода проявить инициативу",
	"Индивидуальное сопровождение":                                                   "Внимание к нему лично",
	"Личный пример и последовательность":                                             "Личный пример старших",
	"Наставник для студента: старший курс, куратор, опытный военный":                 "Закрепить за студентом старшего: старший курс, куратор, опытный военный",
	"Студент — наставник младших, помощь товарищам в подготовке":                     "Студент помогает младшим и товарищам в подготовке",
	"Критерий факта: есть ли организованная воспитательная деятельность как система": "Критерий факта: есть ли в ВУЦ воспитательная работа как система",
}

// humanParts are the pieces of longer texts replaced.
var humanParts = strings.NewReplacer(
	"во всех четырёх видах деятельности", "во всех четырёх сторонах жизни ВУЦ",
	"в других видах деятельностях", "в других делах",
	"в других видах деятельности", "в других делах",
	"повседневной деятельности", "военного быта",
	", наставничество,", ", помощь старших,",
	"два статуса", "два варианта",
)

// humanSections are the sections of the questionnaire about the activity whose
// names name a kind of activity.
var humanSections = map[string]string{"Воспитательная деятельность": "Воспитательная работа"}

// humanKeys are the keys of the input renamed.
var humanKeys = map[string]string{
	"виды_деятельности": "жизнь_в_ВУЦ",
	"связано_в_среде":   "связано_в_жизни_ВУЦ",
}

// scaleSenses are what the scales of «действия и опыт» mean; the names of the
// other scales say it themselves.
var scaleSenses = map[string]string{
	"Активность расширения и обогащения профессиональных и общекультурных знаний, умений, навыков": "стремится узнавать и уметь больше",
	"Критическая, справедливая оценка накопленного опыта":                                          "трезво оценивает свой опыт",
	"Реальное и потенциальное использование приобретаемого опыта, знаний, умений, навыков":         "применяет опыт в новых делах",
}

// humanValue rewrites a value of the input of v7: the keys, the texts, a kind
// of activity as a side of the life in the ВУЦ. The words of the student stay.
func humanValue(value any) any {
	switch v := value.(type) {
	case *object:
		for _, key := range slices.Clone(v.keys) {
			if key == "рассказы_о_деятельности" {
				// The words of the student stay; the section of the
				// questionnaire is the program's.
				for _, story := range objectList(v.values[key]) {
					if section, ok := humanSections[objectText(story, "раздел")]; ok {
						story.values["раздел"] = section
					}
				}
				continue
			}
			if studentKeys[key] {
				continue
			}
			item := v.values[key]
			if key == "вид" {
				if side, ok := humanSides[objectText(v, key)]; ok {
					v.rename(key, "сторона", side.name)
					continue
				}
			}
			item = humanValue(item)
			if name, ok := humanKeys[key]; ok {
				v.rename(key, name, item)
			} else {
				v.values[key] = item
			}
		}
		return v
	case []any:
		for n := range v {
			v[n] = humanValue(v[n])
		}
		return v
	case string:
		if text, ok := humanValues[v]; ok {
			return text
		}
		if side, ok := humanSides[v]; ok {
			return side.name
		}
		return humanParts.Replace(v)
	}
	return value
}

// humanImage puts in place of the labels of a side or a quality of the self
// what they mean, in one phrase.
func humanImage(o *object) {
	phrases := []string{}
	if p := humanGaps[objectText(o, "оценка_расхождения")]; p != "" {
		phrases = append(phrases, p)
	}
	if p := humanSelf[objectText(o, "самооценка")]; p != "" {
		phrases = append(phrases, p)
	}
	if p := objectText(o, "внешнее_давление"); p != "" && p != pressureTexts["no"] {
		phrases = append(phrases, humanPressure)
	}
	if p := objectText(o, "адекватность_самооценки"); p != "" {
		phrases = append(phrases, cmpOr(humanAdequacy[p], p))
	}
	for _, key := range []string{"оценка_расхождения", "самооценка", "внешнее_давление", "адекватность_самооценки"} {
		o.remove(key)
	}
	if gap, ok := o.values["расхождение"]; ok {
		o.rename("расхождение", "разница", gap)
	}
	if len(phrases) > 0 {
		o.insertAfter("название", "смысл", strings.Join(phrases, "; "))
	}
}

func cmpOr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// humanDanilova puts the rules of Danilova of a group of the image in plain
// phrases; no pressure is no key.
func humanDanilova(g *object) {
	if p := objectText(g, "внешнее_давление"); p == pressureTexts["no"] {
		g.remove("внешнее_давление")
	} else if p != "" {
		g.values["внешнее_давление"] = humanPressure
	}
	if p := humanAdequacy[objectText(g, "адекватность_самооценки")]; p != "" {
		g.values["адекватность_самооценки"] = p
	}
}

// humanLeading is the phrase of a block of the identity of a student: the
// phrase of its leading status, or «ещё не определился» when there are several.
func humanLeading(block *object) string {
	leading := objectText(block, "ведущий_статус")
	cases := []selfCase{}
	for _, status := range objectList(block.values["статусы"]) {
		name := objectText(status, "статус")
		if c, ok := selfCases[name]; ok && strings.Contains(leading, name) {
			cases = append(cases, c)
		}
	}
	if len(cases) == 1 {
		return cases[0].phrase
	}
	return selfUndecided.phrase
}

// humanScales adds what a scale of «действия и опыт» means.
func humanScales(payload *object) {
	for _, m := range objectList(payload.values["методики"]) {
		if sense := scaleSenses[objectText(m, "показатель")]; sense != "" {
			m.insertAfter("показатель", "смысл", sense)
		}
	}
}

// humanKinds puts after every side of the life in the ВУЦ what it is.
func humanKinds(payload *object) {
	for _, kind := range objectList(payload.values["виды_деятельности"]) {
		if side, ok := humanSides[objectText(kind, "вид")]; ok {
			kind.insertAfter("вид", "что_это", side.what)
		}
	}
}

// humanTestPayload turns the input of test/v6 into that of v7.
func humanTestPayload(payload *object) {
	if images, ok := payload.values["образы_я"].(*object); ok {
		for _, side := range objectList(images.values["по_сторонам"]) {
			humanImage(side)
		}
		// The qualities with the largest gaps and those whose bar is below
		// the self-esteem.
		for _, key := range slices.Clone(images.keys) {
			if list, ok := images.values[key].([]any); ok && key != "по_сторонам" {
				for _, q := range objectList(list) {
					humanImage(q)
				}
			}
		}
		if list, ok := images.values["наибольшие_расхождения"]; ok {
			images.rename("наибольшие_расхождения", "где_больше_всего_хочет_измениться", list)
		}
	}
	if c, ok := payload.values["согласованность_образов"].(*object); ok {
		o := newObject()
		o.set("смысл", cmpOr(humanCongruent[objectText(c, "оценка")], humanCongruent[congruenceTexts["none"]]))
		payload.values["согласованность_образов"] = o
	}
	for _, g := range objectList(payload.values["образ_защитника"]) {
		g.remove("группа")
		if s := objectText(g, "ситуация"); s != "" {
			g.rename("ситуация", "что_значит_для_студента", cmpOr(humanSituations[s], s))
		}
		humanDanilova(g)
	}
	if blocks, ok := payload.values["профессиональная_идентичность"]; ok {
		list := []any{}
		for _, block := range objectList(blocks) {
			o := newObject()
			o.set("профессия", objectText(block, "профессия"))
			o.set("что_значит", humanLeading(block))
			list = append(list, o)
		}
		payload.rename("профессиональная_идентичность", "самоопределение", list)
	}
	humanScales(payload)
	humanKinds(payload)
	humanValue(payload)
}

// humanPlatoonPayload turns the input of test-platoon/v4 into that of v5: the
// counts under plain phrases, the bases of the groups for separate work in
// plain words.
func humanPlatoonPayload(payload *object, data PlatoonPortraitData) {
	phrases := func(counts *object, names map[string]string) *object {
		o := newObject()
		for _, name := range counts.keys {
			key := cmpOr(names[name], name)
			n, _ := counts.values[name].(int)
			had, _ := o.values[key].(int)
			o.set(key, had+n)
		}
		return o
	}
	for _, g := range objectList(payload.values["образ_защитника"]) {
		g.remove("группа")
		if counts, ok := g.values["ситуации"].(*object); ok {
			g.rename("ситуации", "что_значат_для_студентов", phrases(counts, humanSituations))
		}
	}
	if counts, ok := payload.values["согласованность_образов"].(*object); ok {
		payload.values["согласованность_образов"] = phrases(counts, humanCongruent)
	}
	if blocks, ok := payload.values["профессиональная_идентичность"]; ok {
		names := map[string]string{}
		for name, c := range selfCases {
			names[name] = c.phrase
		}
		list := []any{}
		for _, block := range objectList(blocks) {
			o := newObject()
			o.set("профессия", objectText(block, "профессия"))
			if counts, ok := block.values["ведущий_статус"].(*object); ok {
				o.set("сколько_студентов", phrases(counts, names))
			}
			o.put("пометка", objectText(block, "пометка"))
			list = append(list, o)
		}
		payload.rename("профессиональная_идентичность", "самоопределение", list)
	}
	plain := map[string][]string{}
	for _, g := range data.WorkGroups {
		plain[g.Label] = g.Plain
	}
	for _, g := range objectList(payload.values["отдельная_работа"]) {
		if bases, ok := plain[objectText(g, "метка")]; ok {
			g.set("основания", anyList(bases))
		}
	}
	humanScales(payload)
	humanKinds(payload)
	humanValue(payload)
}

// humanRule finds a label of the input in a lower-case text and tells what to
// write instead; a rule of the paragraphs is not checked in the
// recommendations.
type humanRule struct {
	pattern    *regexp.Regexp
	hint       string
	paragraphs bool
}

func human(pattern, hint string) humanRule {
	return humanRule{regexp.MustCompile(pattern), hint, false}
}

// humanParagraphs is a rule of the paragraphs and the conclusion only: the run
// of 5 November — «Так воспитываются профессионализм и умение принимать
// решения» in a recommendation is not the list of the groups the supervisor
// called «статистика».
func humanParagraphs(pattern, hint string) humanRule {
	return humanRule{regexp.MustCompile(pattern), hint, true}
}

const (
	humanSelfHint  = "передай смысл самоопределения обычными словами: «военную специальность выбрал сам и связывает с ней планы», «ещё выбирает», «с гражданской профессией ещё не определился»"
	humanImageHint = "напиши, что студент считает важным для себя, что у него уже получается и где больше всего предстоит сделать"
	humanGroupHint = "не называй группу качеств целиком — назови одно-два её качества обычными словами («верность долгу», «исполнительность», «выносливость»)"
	humanSideHint  = "не называй вид деятельности — пиши, как студент живёт в ВУЦ: как привык к военному быту и требованиям и как переносит трудности, как осваивает военное дело, чем живёт во взводе"
	humanTeachHint = "не пиши «наставничество» — опиши, что именно делает студент: кому помогает, в чём, как это проверяется"
)

// humanRules are the labels of the input the texts of v7 must not have.
var humanRules = []humanRule{
	human(`статус`, "слово «статус» — "+humanSelfHint),
	human(`идентичност`, "слово «идентичность» — "+humanSelfHint),
	human(`мораторий`, "слово «мораторий» — напиши «ещё выбирает, сомневается»"),
	human(`навязанн`, "слово «навязанный» — напиши «выбрал под влиянием других»"),
	human(`(?:^|[^а-я])ведущ[а-я]*\s+(?:статус|позици|идентичност)`, "«ведущая позиция» — "+humanSelfHint),
	human(`расхожден`, "слово «расхождение» — напиши, видит ли студент, куда расти, и по силам ли ему цель"),
	human(`оптимальн`, "слово «оптимальный» — напиши, видит ли студент, куда расти, и по силам ли ему цель"),
	human(`принят[а-я]*\s+как\s+сво`, "«принят как свой» — "+humanImageHint),
	human(`образ[а-я]*\s+(?:защитника\s+(?:отечества\s+)?)?(?:в\s+целом\s+)?(?:не\s+)?принят`, "«образ принят» — "+humanImageHint),
	human(`во\s+многом\s+достигнут`, "«во многом достигнут» — "+humanImageHint),
	human(`ближе\s+всего\s+(?:он\s+|взвод\s+)?к\s+образу|дальше\s+всего\s+(?:он\s+|взвод\s+)?от\s+образа`, "«ближе / дальше всего к образу» — "+humanImageHint),
	humanParagraphs(`верност[а-я]*\s+долгу\s+и\s+преданност`, "«верность долгу и преданность Отечеству» — "+humanGroupHint),
	humanParagraphs(`профессионализм[а-я]*\s+и\s+умени[а-я]*\s+принимать`, "«профессионализм и умение принимать решения» — "+humanGroupHint),
	humanParagraphs(`товариществ[а-я]*\s+и\s+культур[а-я]*\s+отношени`, "«товарищество и культура отношений» — "+humanGroupHint),
	humanParagraphs(`дисциплин[а-я]*\s+и\s+собранност`, "«дисциплина и собранность» — "+humanGroupHint),
	humanParagraphs(`физическ[а-я]*\s+подготовк[а-я]*\s+и\s+самообладани`, "«физическая подготовка и самообладание» — "+humanGroupHint),
	human(`(?:учебн|воспитательн|повседневн|просветительск)[а-я]*\s+деятельност|(?:^|[^а-я])вид[а-я]*\s+деятельност`, "вид деятельности по названию — "+humanSideHint),
	human(`субъектност`, "слово «субъектность» — напиши «свобода проявить инициативу»"),
	human(`наставни`, "слово «наставничество» — "+humanTeachHint),
}

// humanWordProblems finds the labels of the input in a text, each once.
func humanWordProblems(field, text string) []string {
	lower := lowerText(text)
	problems := []string{}
	paragraph := field == "conclusion" || strings.HasPrefix(field, "paragraphs.") || field == "input" || field == "basis"
	for _, rule := range humanRules {
		if (paragraph || !rule.paragraphs) && rule.pattern.MatchString(lower) {
			problems = append(problems, field+": "+rule.hint)
		}
	}
	return problems
}

// opening is the beginning of the first sentence the supervisor called
// needless: «Студент видит себя ответственным гражданином…» (А1).
var opening = regexp.MustCompile(`^\s*(?:студент[а-я]*|взвод)\s+(?:в\s+целом\s+)?вид[а-я]*\s+себя`)

func openingProblems(texts map[string]string) []string {
	problems := []string{}
	for _, key := range paragraphKeys {
		if opening.MatchString(lowerText(texts["paragraphs."+key])) {
			problems = append(problems, "paragraphs."+key+": не начинай со слов «Студент видит себя…» — начни с вывода: что для студента главное в службе")
		}
	}
	return problems
}

// conclusionWords are the words of the conclusion the supervisor did not
// understand: «Главный резерв — перевод личной надёжности…, чтобы…» (А6).
var conclusionWords = regexp.MustCompile(`(?:^|[^а-я])(?:чтобы|резерв[а-я]*|точк[а-я]*\s+рост[а-я]*|ключ\s*[—–-]|ключев[а-я]*)(?:[^а-я]|$)`)

func humanConclusionProblems(text string) []string {
	if m := conclusionWords.FindStringSubmatch(lowerText(text)); m != nil {
		return []string{"conclusion: общий вывод — обычными словами, как сказал бы командир: можно ли на него положиться, как реагирует на замечания, проявляет ли разумную инициативу, и что главное в работе с ним; без «чтобы», «резерв», «точка роста», «ключ» — второе предложение закончи на самом деле, без «чтобы…» в конце"}
	}
	return nil
}

// official finds who does an action, which the recommendations of v7 do not
// say (А5): the curator and the head of a cycle anywhere, the psychologist as
// a person, the teacher at the beginning of a sentence.
var (
	official      = regexp.MustCompile(`куратор|начальник[а-я]*\s+цикл|(?:^|[^а-я])психолог(?:а|у|ом|и|ов|ам)?(?:[^а-я]|$)`)
	officialStart = regexp.MustCompile(`(?:^|[.!?]\s+)(?:преподавател|командир|начальник)`)
)

// humanActionProblems checks the text of an action of v7: no dates, long
// enough, no copy of an example, not the name of the direction, no official.
func humanActionProblems(field string, record map[string]any, directions map[string]string) []string {
	problems := actionTextProblems(field, record, directions)
	text, _ := record["text"].(string)
	if lower := lowerText(text); official.MatchString(lower) || officialStart.MatchString(lower) {
		problems = append(problems, field+".text: без должностей — рекомендация общая для всех, кто работает со студентом; начни с самого дела: «На полевом выходе поручить ему…»")
	}
	return problems
}

// humanSelfProblems finds, by profession, a phrase of the input of v7 the
// paragraph about the will does not tell.
func humanSelfProblems(payload *object, will string) []string {
	problems := []string{}
	lower := normalizedText(will)
	for _, block := range objectList(payload.values["самоопределение"]) {
		phrase := objectText(block, "что_значит")
		if c, ok := selfCaseOf(phrase); ok && !c.words.MatchString(lower) {
			problems = append(problems, fmt.Sprintf("paragraphs.will: не сказано, как студент выбрал %s, — во входных данных: «%s»; напиши это своими словами, без слов «статус» и «идентичность»", professionAccusative(objectText(block, "профессия")), phrase))
		}
	}
	return problems
}

// humanPlatoonSelfProblems finds, by profession, the most frequent phrase of
// the input of test-platoon/v5 the paragraph about the will does not tell.
func humanPlatoonSelfProblems(payload *object, will string) []string {
	problems := []string{}
	lower := normalizedText(will)
	for _, block := range objectList(payload.values["самоопределение"]) {
		counts, _ := block.values["сколько_студентов"].(*object)
		if counts == nil {
			continue
		}
		best, top := 0, []selfCase{}
		for _, phrase := range counts.keys {
			n, _ := counts.values[phrase].(int)
			c, ok := selfCaseOf(phrase)
			switch {
			case !ok || n == 0 || n < best:
			case n > best:
				best, top = n, []selfCase{c}
			default:
				top = append(top, c)
			}
		}
		told := len(top) == 0
		phrases := []string{}
		for _, c := range top {
			told = told || c.words.MatchString(lower)
			phrases = append(phrases, c.phrase)
		}
		if !told {
			problems = append(problems, fmt.Sprintf("paragraphs.will: не сказано, как студенты выбрали %s, — чаще всего во входных данных: «%s»; напиши это своими словами, без слов «статус» и «идентичность»", professionAccusative(objectText(block, "профессия")), strings.Join(phrases, "» или «")))
		}
	}
	return problems
}

// professionAccusative is a profession of the input as the hint says it:
// «выбрал военную специальность».
func professionAccusative(profession string) string {
	switch lower := lowerText(profession); {
	case strings.HasPrefix(lower, "военн"):
		return "военную специальность"
	case strings.HasPrefix(lower, "гражданск"):
		return "гражданскую профессию"
	default:
		return lower
	}
}

// militarySpecialityPoint is what the point about the military speciality of
// v7 speaks of.
var militarySpecialityPoint = regexp.MustCompile(`выбор|специальност|самоопредел`)

// humanSpecialityProblems: when the student has not chosen the military
// speciality himself, one point goes to group 2 about what it gives him.
func humanSpecialityProblems(output map[string]any, payload *object) []string {
	phrase := ""
	for _, block := range objectList(payload.values["самоопределение"]) {
		if strings.HasPrefix(lowerText(objectText(block, "профессия")), "военн") {
			phrase = objectText(block, "что_значит")
		}
	}
	if phrase == "" || phrase == selfFormed.phrase {
		return nil
	}
	items, _ := output["recommendations"].([]any)
	for _, item := range items {
		record, _ := item.(map[string]any)
		number, _ := numberOf(record["group"])
		text, _ := record["text"].(string)
		if number == 2 && militarySpecialityPoint.MatchString(lowerText(text)) {
			return nil
		}
	}
	codes := ""
	for _, g := range objectList(payload.values["образ_защитника"]) {
		if number, _ := numberOf(g.values["номер"]); number == 2 {
			codes = strings.Join(textList(g.values["подходящие_направления"]), ", ")
		}
	}
	if codes != "" {
		codes = "; код — из подходящих группе 2: " + codes
	}
	return []string{fmt.Sprintf("recommendations: по военной специальности студент «%s» — нужен пункт к группе 2 со словами «военная специальность»: что лично ему даёт выбор военной специальности, например после практического занятия обсудить с ним, что из этого дела ему пригодится как будущему военному специалисту%s", phrase, codes)}
}

// humanTextProblems are the checks of every text of v7 and of test-platoon/v5.
func humanTextProblems(texts map[string]string) []string {
	problems := openingProblems(texts)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, humanWordProblems(key, texts[key])...)
	}
	return append(problems, humanConclusionProblems(texts["conclusion"])...)
}

// humanRecommendationProblems checks the action of every recommendation of v7.
func humanRecommendationProblems(output map[string]any, payload *object) []string {
	items, _ := output["recommendations"].([]any)
	directions := catalogueNames(payload)
	problems := []string{}
	for n, item := range items {
		record, _ := item.(map[string]any)
		problems = append(problems, humanActionProblems(fmt.Sprintf("recommendations.%d", n+1), record, directions)...)
	}
	return problems
}

// checkTestAnswerV7 returns what is wrong with an answer to test/v7: the
// rules of v6 without who and kind, the self-determination by its phrase and
// no label of the input.
func checkTestAnswerV7(output map[string]any, payload *object) []string {
	texts, problems := paragraphTexts(output)
	names := levelNames(payload)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, ratedLevelProblems(key, texts[key])...)
		problems = append(problems, namedLevelProblems(key, texts[key], names)...)
		problems = append(problems, kindsListProblems(key, texts[key])...)
	}
	problems = append(problems, humanSelfProblems(payload, texts["paragraphs.will"])...)
	recommendationTexts(output, texts)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, termProblems(key, texts[key])...)
		problems = append(problems, climateLevelProblems(key, texts[key])...)
		problems = append(problems, foreignLetterProblems(key, texts[key])...)
	}
	problems = append(problems, humanTextProblems(texts)...)
	problems = append(problems, unfoundStateProblems(texts, payload)...)
	recommendations := recommendationProblems(output, payload, recommendationFloor(payload), 4, situationAllowed(payload))
	problems = append(problems, recommendations...)
	problems = append(problems, fewDirectionsProblems(recommendations, payload)...)
	problems = append(problems, humanSpecialityProblems(output, payload)...)
	problems = append(problems, repeatedCodeProblems(output)...)
	problems = append(problems, humanRecommendationProblems(output, payload)...)
	return plainHints(problems)
}

// checkPlatoonAnswerV5 returns what is wrong with an answer to
// test-platoon/v5: the rules of v4 without who and kind and the rules of v7.
func checkPlatoonAnswerV5(output map[string]any, payload *object) []string {
	texts, problems := paragraphTexts(output)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, ratedLevelProblems(key, texts[key])...)
		problems = append(problems, kindsListProblems(key, texts[key])...)
	}
	problems = append(problems, humanPlatoonSelfProblems(payload, texts["paragraphs.will"])...)
	problems = append(problems, recommendationProblems(output, payload, recommendationLeast, 5, nil)...)
	problems = append(problems, repeatedCodeProblems(output)...)
	problems = append(problems, humanRecommendationProblems(output, payload)...)
	recommendationTexts(output, texts)
	problems = append(problems, individualProblems(output, payload, texts, humanActionProblems)...)
	_, measured := payload.values["климат_взвода"]
	for _, key := range sortedTextKeys(texts) {
		if !measured {
			problems = append(problems, climateProblems(key, texts[key])...)
		}
		problems = append(problems, climateLevelProblems(key, texts[key])...)
		problems = append(problems, termProblems(key, texts[key])...)
		problems = append(problems, foreignLetterProblems(key, texts[key])...)
	}
	problems = append(problems, humanTextProblems(texts)...)
	problems = append(problems, unfoundStateProblems(texts, payload)...)
	total, _ := payload.values["студентов"].(int)
	problems = append(problems, platoonCountProblems(texts, total)...)
	return plainHints(problems)
}
