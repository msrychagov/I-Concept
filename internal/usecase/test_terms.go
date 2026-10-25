package usecase

import (
	"fmt"
	"regexp"
	"strings"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// The portrait by the test without the terms of the model (test/v5 and
// test-platoon/v3, Вики/Тестирование/Инструкция v5.md and Инструкция по
// взводу v3.md, approved by the user on 25 October 2026 after the remarks of
// the supervisor): four paragraphs with no headings, the terms of the first
// chapter only by what they mean, a direction of the catalogue once.

// termsVersion is, by task, the first version of the instruction whose input
// has the dictionary of the terms and whose answer is four paragraphs.
var termsVersion = map[string]int{testPortraitTask: 5, testPlatoonTask: 3}

func termsInput(task, version string) bool {
	n, _ := promptbank.Number(version)
	return n >= termsVersion[task]
}

// modelTerm is a term of the model the input has and what it means by the
// first chapter: the dictionary of the input (Инструкция v5.md, «Словарь»).
type modelTerm struct{ Term, Meaning string }

var modelTerms = []modelTerm{
	{"ценностно-смысловой (компонент)", "ценности, мотивы, личный смысл службы и профессии; отношение к Отечеству, военной службе и профессиональной ответственности"},
	{"эмоционально-волевой (компонент)", "отношение к себе и умение управлять своими состояниями и поведением: эмоциональная устойчивость, выдержка, настойчивость, самоконтроль, способность мобилизоваться"},
	{"деятельностно-практический (компонент)", "воплощение принятых ценностей в действиях: ответственность, дисциплинированность, самостоятельность, умение действовать вместе с другими, инициатива; использование и осмысление своего опыта"},
	{"любовь", "нравственно ответственное отношение к человеку: уважение, забота, взаимопомощь, готовность отвечать за другого"},
	{"соборность", "ощущение себя частью коллектива с общими ценностями и целями без потери личной ответственности: товарищество, взаимовыручка, согласованность действий, ответственность за общий результат"},
	{"патриотизм", "устойчивое ценностное отношение к Отечеству, его истории, культуре и государственности, осознание ответственности за его защиту, добросовестное исполнение обязанностей"},
	{"я_реальное, Я-реальное", "каким студент видит себя сейчас"},
	{"я_идеальное, Я-идеальное", "каким гражданином, защитником Отечества и военным профессионалом студент хочет стать"},
	{"Я-идеальное ниже Я-реального", "студент ставит себе планку ниже, чем оценивает себя сейчас"},
	{"Государственно-патриотические качества", "верность долгу и присяге, преданность Отечеству, готовность защищать страну, уважение к её истории и традициям, честность, мужество, решительность"},
	{"Профессионально-компетентностные и когнитивные качества", "профессионализм, постоянное развитие и освоение новой техники, аналитическое мышление, умение оценить обстановку и принять обоснованное решение, самообразование"},
	{"Коммуникативные и коллективистские качества", "товарищество, коллективизм, взаимовыручка, уважение к людям и культура отношений"},
	{"Дисциплинарные и волевые качества", "дисциплинированность, исполнительность, собранность, ответственность, организованность, стойкость в трудностях"},
	{"Физические и психологические качества", "физическая подготовка, стрессоустойчивость, самообладание в трудных условиях"},
	{"взаимодействие", "совместная деятельность преподавателя и студента, в которой они влияют друг на друга"},
	{"общение", "обмен знаниями и опытом, взаимопонимание, согласование позиций, уважение и доверие"},
	{"сотрудничество", "согласованная совместная работа: распределение обязанностей, общие усилия, взаимная поддержка, личная ответственность за общий результат"},
}

const termsNote = "термины модели из входных данных: в тексте пиши не термин, а то, что он значит"

// reflectionSign is the sign of the environment «Осмысление опыта»: its text
// of the first chapter has «Я-идеальным», which the model is told to retell,
// so the input of v5 says it plainly (decision Д5).
const (
	reflectionSign  = "S6"
	reflectionPlain = "опыт разбирается, соотносится с тем, каким студент хочет стать, переносится"
)

func termsObject() *object {
	terms := []any{}
	for _, t := range modelTerms {
		o := newObject()
		o.set("термин", t.Term)
		o.set("значит", t.Meaning)
		terms = append(terms, o)
	}
	o := newObject()
	o.set("пометка", termsNote)
	o.set("термины", terms)
	return o
}

// insertFirst puts a key before all the others.
func (o *object) insertFirst(key string, value any) {
	if _, seen := o.values[key]; !seen {
		o.keys = append([]string{key}, o.keys...)
	}
	o.values[key] = value
}

// addTerms gives the input of test/v5 or test-platoon/v3 the dictionary first
// and the plain text of the sign «Осмысление опыта»; signs are the ids of the
// signs of the environment in the order of the input.
func addTerms(payload *object, signs []string) {
	payload.insertFirst("словарь", termsObject())
	for n, sign := range objectList(payload.values["признаки_среды"]) {
		if n < len(signs) && signs[n] == reflectionSign {
			sign.set("по_главе_1", reflectionPlain)
		}
	}
}

// termRule finds a term of the model in a lower-case text with «е» for «ё»;
// the problem names the stem of the term and tells what to write instead. The
// run of 25 October: told «термин «профессионально-компетентностные качества»»,
// the model kept «профессионально-компетентностная часть образа» for three
// attempts; it is told the stem, in any form.
type termRule struct {
	pattern *regexp.Regexp
	stem    string
	meaning string
}

func term(pattern, stem, meaning string) termRule {
	return termRule{regexp.MustCompile(pattern), stem, meaning}
}

// selfTerm is «Я-…» with any dash or none: «Я-идеальное», «я идеальное».
const selfTerm = `(?:^|[^а-яa-z])я\s*[-‑–—]?\s*`

// modelTermRules are the words the texts of v5 must not have (decisions А1 and
// Д3): the terms of the first chapter, the names of the groups of the image
// and of the signs of the environment. «Патриотический», «включённость»,
// «осмысление опыта», «взаимодействие» and «общение» are ordinary words.
var modelTermRules = []termRule{
	term(`ценностно-смыслов`, "ценностно-смыслов…", "ценности, мотивы, личный смысл службы и профессии, отношение к Отечеству и службе"),
	term(`эмоционально-волев`, "эмоционально-волев…", "владение своими эмоциями и поведением: устойчивость, выдержка, настойчивость, самоконтроль"),
	term(`деятельностно-практическ`, "деятельностно-практическ…", "воплощение ценностей в действиях: ответственность, самостоятельность, инициатива, использование и осмысление опыта"),
	term(`компонент`, "компонент…", "не называй компоненты — пиши, о чём речь: ценности и мотивы, владение собой, действия и опыт"),
	term(`признак`, "признак…", "не называй признаки — пиши, о чём речь: отношение к людям, к товарищам, к Отечеству; о среде — по «по_главе_1»"),
	term(`любов`, "любов…", "уважение, забота, взаимопомощь, готовность отвечать за другого"),
	term(`соборн`, "соборн…", "товарищество, взаимовыручка, согласованность действий, ответственность за общий результат"),
	term(`патриотизм`, "патриотизм…", "ценностное отношение к Отечеству, ответственность за его защиту, добросовестное исполнение обязанностей; прилагательное «патриотический» можно"),
	term(selfTerm+`реальн`, "Я-реальн…", "каким студент видит себя сейчас"),
	term(selfTerm+`идеал`, "Я-идеал…", "каким студент хочет стать; если планка ниже того, как он оценивает себя сейчас, так и пиши"),
	term(selfTerm+`зеркальн`, "Я-зеркальн…", "как, по мнению студента, его видят другие"),
	term(selfTerm+`нормативн`, "Я-нормативн…", "чего, по мнению студента, от него ждут другие"),
	term(selfTerm+`концепц`, "Я-концепци…", "представления студента о себе"),
	term(`сотрудничеств`, "сотрудничеств…", "согласованная совместная работа: распределение обязанностей, взаимная поддержка, ответственность за общий результат"),
	term(`государственно-патриотическ`, "государственно-патриотическ…", "верность долгу, преданность Отечеству, готовность защищать страну, честность, мужество"),
	term(`профессионально-компетентностн`, "профессионально-компетентностн…", "профессионализм, аналитическое мышление, умение оценить обстановку и принять решение, самообразование"),
	term(`когнитивн`, "когнитивн…", "профессионализм, аналитическое мышление, умение оценить обстановку и принять решение, самообразование"),
	term(`коммуникативн\S*\s+и\s+коллективистск`, "коммуникативн… и коллективистск…", "товарищество, коллективизм, взаимовыручка, культура отношений"),
	term(`дисциплинарн\S*\s+и\s+волев`, "дисциплинарн… и волев…", "дисциплинированность, исполнительность, собранность, организованность, стойкость"),
	term(`физическ\S*\s+и\s+психологическ\S*\s+качеств`, "физическ… и психологическ… качеств…", "физическая подготовка, стрессоустойчивость, самообладание"),
	term(`личностно[\s-]+ориентированн`, "личностно ориентированн…", "доверительный диалог, личный пример, беседы, наставничество, уважение достоинства"),
	term(`ясност\S*\s+(?:и\s+согласованност\S*\s+)?норм`, "ясность … норм", "требования понятны, согласованы, соблюдаются"),
	term(`значим\S*\s+деятельност\S*\s+и\s+инициатив`, "значим… деятельност… и инициатив…", "возможность значимого дела, разумная инициатива"),
}

// termProblems finds the terms of the model in a text, each once.
func termProblems(field, text string) []string {
	lower := lowerText(text)
	problems := []string{}
	for _, rule := range modelTermRules {
		if rule.pattern.MatchString(lower) {
			problems = append(problems, fmt.Sprintf("%s: термин «%s» — это слово нельзя ни в какой форме, и в прилагательном тоже; пиши его содержание: %s", field, rule.stem, rule.meaning))
		}
	}
	return problems
}

// kindsList is the list of all the kinds of activity the supervisor called
// needless: «во всех четырёх видах деятельности» (decision Д4).
var kindsList = regexp.MustCompile(`(?:^|[^а-я])(?:все|всех|всем|всеми|четыре|четырех|четырем|четырьмя)\s+(?:(?:четыре|четырех|четырем|четырьмя)\s+)?вид[а-я]*\s+деятельност`)

func kindsListProblems(field, text string) []string {
	if kindsList.MatchString(lowerText(text)) {
		return []string{field + ": не перечисляй виды деятельности подряд — напиши, в чём активнее всего и чего не хватает"}
	}
	return nil
}

// The hint to a figure: the group of the image by its qualities, not by the
// name, which v5 forbids.
const (
	groupNumberHint  = " — группу образа называй словами, например «в коммуникативных и коллективистских качествах», а не номером"
	groupQualityHint = " — группу образа называй её качествами, например «в товариществе и взаимовыручке», а не номером"
)

func plainHints(problems []string) []string {
	for n, p := range problems {
		problems[n] = strings.Replace(p, groupNumberHint, groupQualityHint, 1)
	}
	return problems
}

// codeKey is the direction of a recommendation alone: since v5 a direction
// goes once in a portrait (decision А2), whatever the group.
func codeKey(item any) (string, bool) {
	record, ok := item.(map[string]any)
	if !ok {
		return "", false
	}
	code, _ := record["code"].(string)
	code = strings.TrimSpace(code)
	return code, code != ""
}

// dropRepeated drops a recommendation whose key an earlier one has, as long as
// least are left; a repeat that would leave fewer stays for the check.
func dropRepeated(output map[string]any, key func(any) (string, bool), least int) (map[string]any, int) {
	items, ok := output["recommendations"].([]any)
	if !ok {
		return output, 0
	}
	seen := map[string]bool{}
	kept := []any{}
	dropped := 0
	for _, item := range items {
		k, ok := key(item)
		if ok && seen[k] && len(items)-dropped-1 >= least {
			dropped++
			continue
		}
		if ok {
			seen[k] = true
		}
		kept = append(kept, item)
	}
	if dropped > 0 {
		output["recommendations"] = kept
	}
	return output, dropped
}

// cleanRepeatedCodes drops a recommendation with the direction of an earlier one.
func cleanRepeatedCodes(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int) {
	return dropRepeated(output, codeKey, recommendationFloor(payload))
}

// recommendationFloor is the fewest recommendations an answer of v5 may have:
// three, or as many as the directions that fit the groups of the student when
// they are fewer — a student whose groups are all «образ принят и во многом
// достигнут» has R15 and R16 only, and a direction goes once (decided by the
// user on 25 October 2026). The input by a platoon has no fitting directions:
// three.
func recommendationFloor(payload *object) int {
	codes := map[string]bool{}
	for _, g := range objectList(payload.values["образ_защитника"]) {
		for _, code := range textList(g.values["подходящие_направления"]) {
			codes[code] = true
		}
	}
	if len(codes) == 0 {
		return recommendationLeast
	}
	return min(recommendationLeast, len(codes))
}

// repeatedCodeProblems finds a direction given twice in one answer.
func repeatedCodeProblems(output map[string]any) []string {
	items, _ := output["recommendations"].([]any)
	seen := map[string]bool{}
	problems := []string{}
	for n, item := range items {
		code, ok := codeKey(item)
		if !ok {
			continue
		}
		if seen[code] {
			problems = append(problems, fmt.Sprintf("recommendations.%d: направление %s уже есть — возьми другое направление, каждое — один раз", n+1, code))
		}
		seen[code] = true
	}
	return problems
}

// The paragraphs of the answer and how many sentences the check takes in each:
// the instructions ask for 3–4, the model joins thoughts by «;» (as with the
// activity of v3) or adds one.
var paragraphKeys = []string{"values", "will", "image", "environment"}

const (
	paragraphLeast = 2
	paragraphMost  = 5
)

// paragraphTexts reads the four paragraphs of an answer to test/v5 or
// test-platoon/v3 and the conclusion, with the problems of their form.
func paragraphTexts(output map[string]any) (map[string]string, []string) {
	paragraphs, _ := output["paragraphs"].(map[string]any)
	texts := map[string]string{}
	problems := []string{}
	for _, key := range paragraphKeys {
		text, _ := paragraphs[key].(string)
		field := "paragraphs." + key
		texts[field] = text
		problems = append(problems, fieldProblems(field, text)...)
		problems = append(problems, sentenceProblem(field, text, paragraphLeast, paragraphMost)...)
	}
	conclusion := outputText(output, "conclusion")
	texts["conclusion"] = conclusion
	problems = append(problems, fieldProblems("conclusion", conclusion)...)
	problems = append(problems, sentenceProblem("conclusion", conclusion, 2, 2)...)
	return texts, problems
}

// joinedParagraphs is the text of the four paragraphs together.
func joinedParagraphs(texts map[string]string) string {
	parts := []string{}
	for _, key := range paragraphKeys {
		parts = append(parts, texts["paragraphs."+key])
	}
	return strings.Join(parts, " ")
}

// checkTestAnswerV5 returns what is wrong with an answer to test/v5: the
// rules of v4 that still apply, the terms of the model, the list of the kinds
// of activity and a direction once.
func checkTestAnswerV5(output map[string]any, payload *object) []string {
	texts, problems := paragraphTexts(output)
	names := levelNames(payload)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, ratedLevelProblems(key, texts[key])...)
		problems = append(problems, namedLevelProblems(key, texts[key], names)...)
		problems = append(problems, kindsListProblems(key, texts[key])...)
	}
	if given, leading := leadingStatuses(payload); given {
		lower := normalizedText(joinedParagraphs(texts))
		for _, list := range leading {
			for _, name := range list {
				if !statusNamed(lower, name) {
					problems = append(problems, "paragraphs.will: не назван ведущий статус «"+name+"»")
				}
			}
		}
	}
	recommendationTexts(output, texts)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, termProblems(key, texts[key])...)
		problems = append(problems, climateLevelProblems(key, texts[key])...)
	}
	problems = append(problems, unfoundStateProblems(texts, payload)...)
	problems = append(problems, recommendationProblems(output, payload, recommendationFloor(payload), 4, situationAllowed(payload))...)
	problems = append(problems, selfDeterminationProblems(output, payload)...)
	problems = append(problems, repeatedCodeProblems(output)...)
	return plainHints(problems)
}

// checkPlatoonAnswerV3 returns what is wrong with an answer to
// test-platoon/v3: the rules of v2 that still apply and those of test/v5.
func checkPlatoonAnswerV3(output map[string]any, payload *object) []string {
	texts, problems := paragraphTexts(output)
	for _, key := range sortedTextKeys(texts) {
		problems = append(problems, ratedLevelProblems(key, texts[key])...)
		problems = append(problems, kindsListProblems(key, texts[key])...)
	}
	if identity, given := payload.values["профессиональная_идентичность"]; given {
		problems = append(problems, platoonIdentityProblems("paragraphs.will", joinedParagraphs(texts), identity)...)
	}
	problems = append(problems, recommendationProblems(output, payload, recommendationLeast, 5, nil)...)
	problems = append(problems, repeatedCodeProblems(output)...)
	recommendationTexts(output, texts)
	individual, ok := output["individual"].([]any)
	if !ok || len(individual) > 3 {
		problems = append(problems, "individual: нужен список не больше чем из трёх пунктов")
	}
	for n, item := range individual {
		text, _ := item.(string)
		field := fmt.Sprintf("individual.%d", n+1)
		problems = append(problems, fieldProblems(field, text)...)
		texts[field] = text
	}
	_, measured := payload.values["климат_взвода"]
	for _, key := range sortedTextKeys(texts) {
		if !measured {
			problems = append(problems, climateProblems(key, texts[key])...)
		}
		problems = append(problems, climateLevelProblems(key, texts[key])...)
		problems = append(problems, termProblems(key, texts[key])...)
	}
	problems = append(problems, unfoundStateProblems(texts, payload)...)
	total, _ := payload.values["студентов"].(int)
	problems = append(problems, platoonCountProblems(texts, total)...)
	return plainHints(problems)
}
