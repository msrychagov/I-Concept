package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// RegisterPrompts connects the instructions and turns on the portrait tasks
// whose instruction is among them, each with its newest version. It is called
// once at startup, before UseModel.
func (a *Service) RegisterPrompts(prompts ...promptbank.Prompt) {
	if prompt, ok := promptbank.Latest(prompts, autobiographyForm); ok {
		a.registerPortraitTask(a.autobiographyTask(prompt))
	}
	if prompt, ok := promptbank.Latest(prompts, testPortraitTask); ok {
		a.registerPortraitTask(a.testTask(prompt))
	}
	if prompt, ok := promptbank.Latest(prompts, testPlatoonTask); ok {
		a.registerPortraitTask(a.platoonTask(prompt))
	}
	if prompt, ok := promptbank.Latest(prompts, essayForm); ok {
		a.registerPortraitTask(a.essayTask(prompt))
	}
	if prompt, ok := promptbank.Latest(prompts, essayCohortTask); ok {
		a.registerPortraitTask(a.cohortTask(prompt))
	}
}

// findSubmission returns the submitted autobiography a source points to.
func findSubmission(db model.Database, s Source) (model.FormSubmission, bool) {
	if s.Kind != "submission" {
		return model.FormSubmission{}, false
	}
	for _, submission := range db.Submissions {
		if submission.ID == s.ID {
			return submission, true
		}
	}
	return model.FormSubmission{}, false
}

// autobiographyTask writes the portrait of a submitted autobiography from the
// anonymised input of model_input.go. The levels of potential are counted by
// the program; the answer is only checked against them.
func (a *Service) autobiographyTask(prompt promptbank.Prompt) portraitTask {
	// The answer of v4 has other fields and rules; v1–v3 are checked as they
	// always were. The input is the same, so a portrait of v3 is found by the
	// fingerprint and shown while the one of v4 is made.
	check, attempts := checkAutobiographyAnswer, 0
	if n, _ := promptbank.Number(prompt.Version); n >= 4 {
		check, attempts = checkAutobiographyAnswerV4, 5
	}
	return portraitTask{
		ID: autobiographyForm, Version: prompt.Version, Prompt: prompt, Attempts: attempts,
		Accepts: func(db model.Database, s Source) bool {
			submission, ok := findSubmission(db, s)
			return ok && submission.FormID == autobiographyForm && submission.Completed
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) {
			submission, _ := findSubmission(db, s)
			payload, _, err := submissionPayload(a.forms[submission.FormID], submission)
			return payload, err
		},
		Check: check,
		Visible: func(db model.Database, s Source, teacherID string) bool {
			submission, ok := findSubmission(db, s)
			return ok && teacherOwnsSubmission(db, teacherID, submission)
		},
	}
}

const attentionLimit = 5

// consequence finds a conclusion in a point of «what to consider»: what the
// circumstance affects, how to take it into account, what to do about it. The
// point must hold the fact alone (decided 9 October 2026). The words come from
// the answers that broke this rule in the comparison of the models.
var consequence = regexp.MustCompile(`(?i)влия|учитыв|учесть при|при планировани|при распределени|при назначени|рекоменд|следует|необходимо|потребует|может сказаться|указывает на|свидетельству|расширять|развивать|уделить|обратить внимание|допуск|нагрузк|нарядам|нарядов|наряды|в наряд`)

// criminalDetails is the phrase that must follow the student's own criminal record.
var criminalDetails = regexp.MustCompile(`(?i)подробности[^.;]{0,20}в ответах студента`)

// Since v3 (decided 24 October 2026) the model names the qualities and skills
// an activity shows: only from the six areas and the hobbies, always marked
// «по опыту в …», and nothing from the family, citizenship or criminal records.
const noExperience = "Опыт не указан"

// byExperience is the mark of a quality drawn from an activity.
var byExperience = regexp.MustCompile(`(?i)по опыту`)

// lifeCircumstance finds the family, the upbringing, the marriage, citizenship
// and criminal records. They have no place where the qualities are: a quality
// next to them is drawn from them. The short words must stand alone, so that
// «обратить», «материал» or «восемь» are not taken for a brother, a mother or
// a family.
var lifeCircumstance = regexp.MustCompile(`(?i)семь[яиюеё]|семей|воспитыва|родител|судим|уголовн|гражданств|(^|[^а-яё])(отец|отц[аеуо]м?|мать|матер(и|ью)|брат(а|у|ом|ья|ьев|ьям)?|сестр(а|ы|е|у|ой)?|жена|жены|женат|супруг[а-яё]*|сын(а|у|ом)?|дочь|дочер[а-яё]*|ребён[а-яё]*|вырос|выросш[а-яё]*)([^а-яё]|$)`)

// skillWord is a skill or a quality. Next to a position the student held
// before the university it is drawn from that work, which since D9 (24 October
// 2026) is only a fact: «Работал официантом».
var skillWord = regexp.MustCompile(`(?i)навык|качеств|умени|умеет|дисциплин|коммуникаб|коммуникац|общительн|ответствен|организованн|стрессоуст|выносл|терпени|аккуратн|пунктуальн|исполнительн`)

// jobModifiers are the words of a position that say nothing of the work itself
// and would catch the ranks of the service: «старший сержант».
var jobModifiers = []string{"старший", "младший", "главный", "ведущий", "помощник", "заместитель", "начальник", "специалист", "сотрудник", "работник", "стажёр", "стажер"}

// jobStems returns the stems of the positions the student held before the
// university: «официант» gives «официа», which finds «официантом».
func jobStems(payload *object) []string {
	stems := []string{}
	jobs, _ := payload.values["работа_до_вуза"].([]any)
	for _, job := range jobs {
		record, ok := job.(*object)
		if !ok {
			continue
		}
		position, _ := record.values["должность"].(string)
		for _, word := range strings.FieldsFunc(strings.ToLower(position), func(r rune) bool { return !unicode.IsLetter(r) }) {
			letters := []rune(word)
			if len(letters) < 4 || slices.Contains(jobModifiers, word) {
				continue
			}
			if len(letters) > 5 {
				letters = letters[:max(5, len(letters)-2)]
			}
			stems = append(stems, string(letters))
		}
	}
	return stems
}

// workProblems finds a skill or a quality in the same sentence as a position
// the student held before the university. It names the skill word, never the
// position: that is the student's own text.
func workProblems(where, text string, jobs []string) []string {
	for _, part := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return strings.ContainsRune(".;!?", r) }) {
		found := skillWord.FindString(part)
		if found == "" {
			continue
		}
		for _, stem := range jobs {
			if strings.Contains(part, stem) {
				return []string{fmt.Sprintf("%s: навыки или качества из работы до вуза («%s»)", where, found)}
			}
		}
	}
	return nil
}

// qualityProblems checks a text where the qualities may be: none drawn from
// the family, citizenship, criminal records or the work before the university.
func qualityProblems(where, text string, jobs []string) []string {
	return append(circumstanceProblems(where, text), workProblems(where, text, jobs)...)
}

// sentence trims a text and the full stop at its end, to compare it with a
// fixed phrase of the instruction.
func sentence(text string) string {
	return strings.TrimSuffix(strings.TrimSpace(text), ".")
}

// checkAutobiographyAnswer returns what is wrong with an answer to the
// autobiography instruction that a program can find: the schema, the levels,
// the number of points, the qualities by experience and the rules of «what to
// consider».
func checkAutobiographyAnswer(output map[string]any, payload *object) []string {
	problems := []string{}
	if text, ok := output["summary"].(string); !ok || strings.TrimSpace(text) == "" {
		problems = append(problems, "нет характеристики")
	} else if byExperience.MatchString(text) {
		problems = append(problems, "summary: качества «по опыту» в характеристике, а не в выводах")
	}
	expected := map[string]string{}
	areas := []string{}
	if levels, ok := payload.values["уровни"].(*object); ok {
		for _, key := range levels.keys {
			areas = append(areas, key)
			expected[key], _ = levels.values[key].(string)
		}
	}
	jobs := jobStems(payload)
	items, listed := output["potential"].([]any)
	records := []map[string]any{}
	for _, item := range items {
		if record, ok := item.(map[string]any); ok {
			records = append(records, record)
		}
	}
	if !listed || len(records) != len(items) {
		problems = append(problems, "потенциал не список записей")
	} else {
		given := []string{}
		for _, record := range records {
			name, _ := record["area"].(string)
			given = append(given, name)
		}
		switch {
		case !slices.Equal(slices.Sorted(slices.Values(given)), slices.Sorted(slices.Values(areas))):
			problems = append(problems, fmt.Sprintf("направления не те: %v", given))
		case !slices.Equal(given, areas):
			problems = append(problems, "направления в другом порядке")
		}
		for _, record := range records {
			name, _ := record["area"].(string)
			for _, key := range []string{"level", "facts", "conclusion"} {
				if text, ok := record[key].(string); !ok || strings.TrimSpace(text) == "" {
					problems = append(problems, name+": пустое поле "+key)
				}
			}
			if want, known := expected[name]; known && record["level"] != want {
				problems = append(problems, fmt.Sprintf("%s: уровень изменён, «%v» вместо «%s»", name, record["level"], want))
			}
			problems = append(problems, conclusionProblems(name, record, expected[name], jobs)...)
		}
	}
	problems = append(problems, hobbiesProblems(output["hobbies"], payload, jobs)...)
	attention, ok := output["attention"].(map[string]any)
	if !ok {
		return append(problems, "нет блока «на что обратить внимание»")
	}
	for _, key := range []string{"use", "consider"} {
		list, isList := attention[key].([]any)
		strs := true
		for _, item := range list {
			if _, isString := item.(string); !isString {
				strs = false
			}
		}
		switch {
		case !isList || !strs:
			problems = append(problems, "attention."+key+" не список строк")
		case len(list) > attentionLimit:
			problems = append(problems, "attention."+key+": пунктов больше пяти")
		}
		if key == "consider" && isList && strs {
			problems = append(problems, considerProblems("attention.consider", list, payload)...)
		}
		if key == "use" && isList && strs {
			for n, point := range list {
				problems = append(problems, qualityProblems(fmt.Sprintf("attention.use, пункт %d", n+1), point.(string), jobs)...)
			}
		}
	}
	return problems
}

// conclusionProblems checks the conclusion of one area: the qualities marked
// «по опыту в …», or exactly «Опыт не указан» when there is nothing to draw
// them from. A level above «не выражен» always rests on some experience.
func conclusionProblems(area string, record map[string]any, level string, jobs []string) []string {
	text, _ := record["conclusion"].(string)
	facts, _ := record["facts"].(string)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	problems := []string{}
	none := sentence(text) == noExperience
	switch {
	case !none && !byExperience.MatchString(text):
		problems = append(problems, area+": в выводе нет «по опыту в …»")
	case none && level != "" && level != "не выражен":
		problems = append(problems, fmt.Sprintf("%s: уровень «%s», а вывод «%s»", area, level, noExperience))
	case !none && sentence(facts) == "Сведений нет":
		problems = append(problems, fmt.Sprintf("%s: сведений нет, а вывод не «%s»", area, noExperience))
	}
	return append(problems, qualityProblems(area+", вывод", text, jobs)...)
}

// hobbiesProblems checks the qualities by the hobbies: at most one point a
// hobby of the questionnaire, each marked «по опыту в …».
func hobbiesProblems(value any, payload *object, jobs []string) []string {
	list, isList := value.([]any)
	for _, item := range list {
		if _, isString := item.(string); !isString {
			isList = false
		}
	}
	if !isList {
		return []string{"hobbies не список строк"}
	}
	problems := []string{}
	if given := len(textList(payload.values["увлечения"])); len(list) > given {
		problems = append(problems, fmt.Sprintf("hobbies: пунктов %d, а увлечений в анкете %d", len(list), given))
	}
	for n, item := range list {
		where := fmt.Sprintf("hobbies, пункт %d", n+1)
		if !byExperience.MatchString(item.(string)) {
			problems = append(problems, where+": нет «по опыту в …»")
		}
		problems = append(problems, qualityProblems(where, item.(string), jobs)...)
	}
	return problems
}

// circumstanceProblems finds the family, citizenship or a criminal record
// where the qualities are. It names the word found, never the text around it.
func circumstanceProblems(where, text string) []string {
	found := lifeCircumstance.FindStringSubmatch(text)
	if found == nil {
		return nil
	}
	word := found[0]
	if found[2] != "" {
		word = found[2]
	}
	return []string{fmt.Sprintf("%s: семья, гражданство или судимость рядом с качествами («%s»)", where, strings.ToLower(word))}
}

// textList reads a list of texts of the input, as it is built or as it is read back.
func textList(value any) []string {
	switch list := value.(type) {
	case []string:
		return list
	case []any:
		texts := []string{}
		for _, item := range list {
			if text, ok := item.(string); ok {
				texts = append(texts, text)
			}
		}
		return texts
	}
	return nil
}

// considerProblems checks the points of «what to consider» against the rules
// of the instruction. A problem names the point by its number and never quotes
// its text: the problems go to the journal, the answer must not.
func considerProblems(where string, points []any, payload *object) []string {
	problems := []string{}
	details := false
	for n, point := range points {
		text := point.(string)
		if found := consequence.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s, пункт %d: вывод вместо факта («%s»)", where, n+1, strings.ToLower(found)))
		}
		if byExperience.MatchString(text) {
			problems = append(problems, fmt.Sprintf("%s, пункт %d: качества «по опыту» вместо факта", where, n+1))
		}
		details = details || criminalDetails.MatchString(text)
	}
	if record, _ := payload.values["уголовная_ответственность"].(string); record == "привлекался" && !details {
		problems = append(problems, where+": нет «подробности — в ответах студента»")
	}
	return problems
}

// Since v4 (decided 25 October 2026) the model writes two or three paragraphs
// and «what to consider»: of what the student has, never of what he has not.
// The summary and the table are gone: the facts are in the printed
// autobiography and the levels in the program's own block.

// noActivities is the only paragraph of a student with no activity and no hobby.
const noActivities = "Занятий, по которым можно назвать качества и навыки, студент не указал"

// areaWords find an area of the potential in the text. An area at the level
// «не выражен» must not be mentioned: the student has no experience in it.
// «Спорт» stands alone, so that «киберспорт» and «военно-спортивные сборы» pass;
// «научно-популярный» passes too.
var areaWords = map[string]*regexp.Regexp{
	"спорт":               regexp.MustCompile(`(?i)(^|[^а-яё-])спорт|разряд|(^|[^а-яё])(гто|кмс)([^а-яё]|$)`),
	"музыка и творчество": regexp.MustCompile(`(?i)музык|творчеств|вокал|танц|театр|хореограф|(^|[^а-яё])квн([^а-яё]|$)`),
	"военно-патриотическая деятельность": regexp.MustCompile(`(?i)военно-патриот|юнарм|зарниц|кадет`),
	"наука":             regexp.MustCompile(`(?i)научн([^о]|о[^-]|о?$)|(^|[^а-яё])наук(а|е|у|ой)([^а-яё]|$)|олимпиад|конференц|публикац`),
	"волонтёрство":      regexp.MustCompile(`(?i)волонт[её]р|донор`),
	"иностранные языки": regexp.MustCompile(`(?i)иностранн[а-яё]* язык|английск|немецк|французск|испанск|китайск|ielts|toefl|delf|dalf|hsk|goethe|dele|cambridge`),
}

// absence finds what the student has not: «без медали», «наград не имеет»,
// «нет опыта», «не служил». The words stand alone.
var absence = regexp.MustCompile(`(?i)(^|[^а-яё])(нет|без медал[а-яё]*|без разряд[а-яё]*|без наград[а-яё]*|без опыта|не имеет|не имел[а-яё]*|не имею|не выезжал[а-яё]*|не был[а-яё]*|не бывал[а-яё]*|не служил[а-яё]*|не проходил[а-яё]*|не привлекал[а-яё]*|не участвовал[а-яё]*|не занимал[а-яё]*|не указ[а-яё]*|не выражен[а-яё]*|не владеет|не женат|отсутству[а-яё]*)([^а-яё]|$)`)

// emptyAreas returns the areas of the input at the level «не выражен».
func emptyAreas(payload *object) []string {
	empty := []string{}
	if levels, ok := payload.values["уровни"].(*object); ok {
		for _, key := range levels.keys {
			if level, _ := levels.values[key].(string); level == "не выражен" {
				empty = append(empty, key)
			}
		}
	}
	return empty
}

// absenceProblems finds an area without experience or the absence of
// something in a text of v4. It names the area or the word of the rule, never
// the text around it.
func absenceProblems(where, text string, empty []string) []string {
	problems := []string{}
	for _, area := range empty {
		if words, ok := areaWords[area]; ok && words.MatchString(text) {
			problems = append(problems, fmt.Sprintf("%s: направление без опыта («%s») — не упоминай его", where, area))
		}
	}
	if found := absence.FindStringSubmatch(text); found != nil {
		problems = append(problems, fmt.Sprintf("%s: написано об отсутствии («%s») — убери это, пиши только о том, что у студента есть", where, strings.ToLower(found[2])))
	}
	return problems
}

// stringList reads a list of strings of the answer.
func stringList(value any) ([]any, bool) {
	list, ok := value.([]any)
	for _, item := range list {
		if _, isString := item.(string); !isString {
			return list, false
		}
	}
	return list, ok
}

// checkAutobiographyAnswerV4 returns what is wrong with an answer to the
// instruction v4: the number of paragraphs, the qualities by experience, no
// family or work skills, nothing of the areas without experience, no absence,
// and «what to consider» as facts.
func checkAutobiographyAnswerV4(output map[string]any, payload *object) []string {
	problems := []string{}
	empty := emptyAreas(payload)
	hobbies := len(textList(payload.values["увлечения"]))
	levels, _ := payload.values["уровни"].(*object)
	active := hobbies > 0 || levels == nil || len(empty) < len(levels.keys)
	jobs := jobStems(payload)
	paragraphs, ok := stringList(output["paragraphs"])
	switch {
	case !ok:
		problems = append(problems, "paragraphs не список строк")
	case !active && (len(paragraphs) != 1 || sentence(paragraphs[0].(string)) != noActivities):
		problems = append(problems, fmt.Sprintf("paragraphs: занятий нет, а вывод не «%s»", noActivities))
	case active && (len(paragraphs) < 2 || len(paragraphs) > 3):
		problems = append(problems, fmt.Sprintf("paragraphs: абзацев %d, а нужно два-три", len(paragraphs)))
	}
	if ok && active {
		marked := false
		for n, item := range paragraphs {
			text, where := item.(string), fmt.Sprintf("paragraphs, абзац %d", n+1)
			if strings.TrimSpace(text) == "" {
				problems = append(problems, where+": пустой")
				continue
			}
			marked = marked || byExperience.MatchString(text)
			problems = append(problems, qualityProblems(where, text, jobs)...)
			problems = append(problems, absenceProblems(where, text, empty)...)
		}
		if !marked {
			problems = append(problems, "paragraphs: нет «по опыту в …» — каждую фразу о качествах начни словами «По опыту в …», а не «По увлечению …» или «Студент увлекается …»")
		}
	}
	consider, ok := stringList(output["consider"])
	switch {
	case !ok:
		problems = append(problems, "consider не список строк")
	case len(consider) > attentionLimit:
		problems = append(problems, "consider: пунктов больше пяти")
	}
	if ok {
		problems = append(problems, considerProblems("consider", consider, payload)...)
		for n, point := range consider {
			problems = append(problems, absenceProblems(fmt.Sprintf("consider, пункт %d", n+1), point.(string), empty)...)
		}
	}
	return problems
}

// testPortraitTask writes the portrait of the first chapter by a finished test
// session with the methods: the levels are counted by methods.go, the model
// only writes the text (Вики/Тестирование/Инструкция.md).
const testPortraitTask = "test"

func findSession(db model.Database, s Source) (model.Session, bool) {
	if s.Kind != "session" {
		return model.Session{}, false
	}
	for _, record := range db.Sessions {
		if record.Session.ID == s.ID {
			return record.Session, true
		}
	}
	return model.Session{}, false
}

func (a *Service) testTask(prompt promptbank.Prompt) portraitTask {
	// The answer of v2 has other fields and rules (test_check.go); v1 is
	// checked as it always was.
	check := checkTestAnswer
	var clean func(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int)
	var earlier func(a *Service, db model.Database, s Source) []*object
	if n, _ := promptbank.Number(prompt.Version); n >= 2 {
		check, clean = checkTestAnswerV2, cleanRepeatedDirections
		// The portrait by an earlier version is shown while the one by a newer
		// version is made (decision Д8): the input of v2 and v3 is the same, so
		// their portraits are found by the fingerprint itself; v4 has its own.
		versions := []string{"v1"}
		if climateInput(testPortraitTask, prompt.Version) {
			check, versions = checkTestAnswerV4, []string{"v3", "v1"}
		}
		earlier = func(a *Service, db model.Database, s Source) []*object {
			session, _ := findSession(db, s)
			data := a.TestPortrait(session)
			if data == nil {
				return nil
			}
			list := []*object{}
			for _, version := range versions {
				list = append(list, a.testPayloadFor(version, data))
			}
			return list
		}
	}
	return portraitTask{
		ID: testPortraitTask, Version: prompt.Version, Prompt: prompt,
		// The portrait has nine sections: about 2000 tokens of Russian text.
		// More rules than in the autobiography: more tries, each with the notes.
		MaxTokens: 4000, Attempts: 5,
		Accepts: func(db model.Database, s Source) bool {
			session, ok := findSession(db, s)
			return ok && session.Completed && session.Methods && a.TestPortrait(session) != nil
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) {
			session, _ := findSession(db, s)
			data := a.TestPortrait(session)
			if data == nil {
				return nil, problem(Invalid, "В тестировании нет методик.")
			}
			return a.testPayloadFor(prompt.Version, data), nil
		},
		Clean: clean, Check: check, Earlier: earlier,
		Visible: func(db model.Database, s Source, teacherID string) bool {
			session, ok := findSession(db, s)
			return ok && teacherOwnsSession(db, teacherID, session)
		},
	}
}

// cause finds the words of a cause the user forbade on 22 October 2026: the
// portrait shows how things go together, not what influenced what. Checked in
// every section but support and growth.
var cause = regexp.MustCompile(`(?i)повлия|благодаря|привел[оаи]? к|привело к|в результате|под влиянием|обусловл`)

// influence finds softer verbs of influence; they are checked only in link,
// the section about how conditions, activity and the self go together.
var influence = regexp.MustCompile(`(?i)ограничива|сдержива|усилива|способству|снижа[ею]т`)

// diagnosis finds clinical words and conclusions about fitness or beliefs.
var diagnosis = regexp.MustCompile(`(?i)диагноз|расстройств|патолог|непригод|пригоден к|лояльн|к сожалению`)

// shortLevel finds a level put on a 1–5 score by a short adjective
// («коллективизм низок»). Adverbs («высоко оценивает») retell the student's own
// rating and pass; the full forms name the levels of the methods and are
// checked against them.
var shortLevel = regexp.MustCompile(`(?i)(^|[^а-яё])(низок|низка|низки|высок|высока|высоки)([^а-яё]|$)`)

var levelWord = regexp.MustCompile(`(?i)низк|средн|высок`)

func objectList(value any) []*object {
	list, _ := value.([]any)
	result := []*object{}
	for _, item := range list {
		if o, ok := item.(*object); ok {
			result = append(result, o)
		}
	}
	return result
}

func objectText(o *object, key string) string {
	text, _ := o.values[key].(string)
	return text
}

// checkTestAnswer returns what is wrong with an answer to the test
// instruction. A problem names the section, never quotes the answer.
func checkTestAnswer(output map[string]any, payload *object) []string {
	problems := []string{}
	sections := map[string]string{}
	for _, key := range []string{"portrait", "signs", "images", "environment", "link"} {
		text, _ := output[key].(string)
		if strings.TrimSpace(text) == "" {
			problems = append(problems, "нет раздела "+key)
		}
		sections[key] = text
	}
	core, _ := output["core"].(map[string]any)
	for _, key := range []string{"cs", "ev", "dp"} {
		text, _ := core[key].(string)
		if strings.TrimSpace(text) == "" {
			problems = append(problems, "нет раздела core."+key)
		}
		sections["core."+key] = text
	}
	kinds := objectList(payload.values["виды_деятельности"])
	items, _ := output["activity"].([]any)
	if len(items) != len(kinds) {
		problems = append(problems, fmt.Sprintf("видов деятельности %d вместо %d", len(items), len(kinds)))
	}
	for n, item := range items {
		record, _ := item.(map[string]any)
		text, _ := record["text"].(string)
		sections[fmt.Sprintf("activity.%d", n+1)] = text
		if n >= len(kinds) {
			continue
		}
		kind, _ := record["kind"].(string)
		level, _ := record["level"].(string)
		want := kinds[n]
		switch {
		case !strings.EqualFold(strings.TrimSpace(kind), objectText(want, "вид")):
			problems = append(problems, fmt.Sprintf("activity.%d: не тот вид или порядок", n+1))
		case strings.TrimSpace(level) != objectText(want, "уровень"):
			problems = append(problems, fmt.Sprintf("activity.%d: уровень изменён", n+1))
		case strings.TrimSpace(text) == "":
			problems = append(problems, fmt.Sprintf("activity.%d: нет текста", n+1))
		}
	}
	for _, key := range []string{"support", "growth"} {
		list, ok := output[key].([]any)
		strs := ok && len(list) > 0 && len(list) <= attentionLimit
		for _, item := range list {
			if text, isText := item.(string); !isText || strings.TrimSpace(text) == "" {
				strs = false
			} else if found := diagnosis.FindString(text); found != "" {
				problems = append(problems, fmt.Sprintf("%s: недопустимое слово («%s»)", key, strings.ToLower(found)))
			}
		}
		if !strs {
			problems = append(problems, key+": нужно от 1 до 5 пунктов")
		}
	}
	keys := make([]string, 0, len(sections))
	for key := range sections {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		text := sections[key]
		if found := cause.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: причина вместо сочетания («%s»)", key, strings.ToLower(found)))
		}
		if found := influence.FindString(text); key == "link" && found != "" {
			problems = append(problems, fmt.Sprintf("%s: влияние вместо сочетания («%s»)", key, strings.ToLower(found)))
		}
		if found := diagnosis.FindString(text); found != "" {
			problems = append(problems, fmt.Sprintf("%s: недопустимое слово («%s»)", key, strings.ToLower(found)))
		}
		// Only these sections speak of 1–5 scores alone; elsewhere the levels of
		// the methods and the questionnaire are named rightly.
		if key == "signs" || key == "environment" {
			if found := shortLevel.FindStringSubmatch(text); found != nil {
				problems = append(problems, fmt.Sprintf("%s: уровень у балла 1–5 («%s»)", key, strings.ToLower(found[2])))
			}
		}
	}
	// A level written right after the name of a method must be its own.
	for _, method := range objectList(payload.values["методики"]) {
		name, want := strings.ToLower(objectText(method, "показатель")), objectText(method, "уровень")
		if name == "" || want == "не определён" {
			continue
		}
		for _, key := range []string{"portrait", "core.cs", "core.ev", "core.dp"} {
			text := strings.ToLower(sections[key])
			for at := strings.Index(text, name); at >= 0; {
				tail := []rune(text[at+len(name):])
				window := string(tail[:min(len(tail), 40)])
				if found := levelWord.FindString(window); found != "" && !strings.HasPrefix(want, found) {
					problems = append(problems, fmt.Sprintf("%s: уровень показателя «%s» не тот", key, objectText(method, "показатель")))
					break
				}
				next := strings.Index(text[at+len(name):], name)
				if next < 0 {
					break
				}
				at += len(name) + next
			}
		}
	}
	return problems
}
