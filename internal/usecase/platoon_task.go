package usecase

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// The conclusion by a platoon (task test-platoon, Вики/Тестирование/Инструкция
// по взводу.md, approved on 24 October 2026; decision Д6 of the plan): made by
// the teacher's button over the latest finished session with the methods of
// every student of the platoon, of three students at least. The model gets the
// numbers of the platoon only, no names; the lists of students the program
// shows itself.

const testPlatoonTask = "test-platoon"

// platoonMinimum is the fewest students a conclusion by a platoon is made of.
const platoonMinimum = 3

// platoonSource is the source of the conclusion: a platoon as one teacher sees
// it, since the platoon is counted over the teacher's own sessions.
func platoonSource(teacherID, group string) Source {
	return Source{"platoon", teacherID + "/" + group}
}

func splitPlatoon(s Source) (teacherID, group string, ok bool) {
	if s.Kind != "platoon" {
		return "", "", false
	}
	teacherID, group, ok = strings.Cut(s.ID, "/")
	return teacherID, group, ok && teacherID != "" && group != ""
}

func (a *Service) platoonTask(prompt promptbank.Prompt) portraitTask {
	// The conclusion by an earlier version is shown while the one by a newer
	// version is made: the input of v2 has the climate and the second edition
	// of the catalogue, that of v3 the dictionary of the terms.
	var earlier func(a *Service, db model.Database, s Source) []*object
	check, clean := checkPlatoonAnswer, cleanRepeatedDirections
	versions := []string{"v1"}
	if termsInput(testPlatoonTask, prompt.Version) {
		check, clean, versions = checkPlatoonAnswerV3, cleanRepeatedCodes, []string{"v2", "v1"}
	}
	if climateInput(testPlatoonTask, prompt.Version) {
		earlier = func(a *Service, db model.Database, s Source) []*object {
			teacherID, group, _ := splitPlatoon(s)
			data := a.platoonPortrait(group, a.platoonStudents(db, teacherID, group))
			list := []*object{}
			for _, version := range versions {
				list = append(list, a.platoonPayloadFor(version, data))
			}
			return list
		}
	}
	return portraitTask{
		ID: testPlatoonTask, Version: prompt.Version, Prompt: prompt,
		MaxTokens: 4000, Attempts: 5, Manual: true,
		Accepts: func(db model.Database, s Source) bool {
			teacherID, group, ok := splitPlatoon(s)
			return ok && len(a.platoonStudents(db, teacherID, group)) >= platoonMinimum
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) {
			teacherID, group, _ := splitPlatoon(s)
			return a.platoonPayloadFor(prompt.Version, a.platoonPortrait(group, a.platoonStudents(db, teacherID, group))), nil
		},
		Earlier: earlier,
		Stamp: func(a *Service, db model.Database, s Source) (string, []string) {
			teacherID, group, _ := splitPlatoon(s)
			ids := []string{}
			for _, p := range a.platoonStudents(db, teacherID, group) {
				ids = append(ids, p.student.SessionID)
			}
			return group, ids
		},
		Clean: clean, Check: check,
		Visible: func(db model.Database, s Source, teacherID string) bool {
			owner, _, ok := splitPlatoon(s)
			return ok && owner == teacherID
		},
	}
}

// levelCounts are the students on each level, the undetermined only when there are some.
func levelCounts(levels map[string]int) *object {
	o := newObject()
	for _, level := range []string{"high", "medium", "low"} {
		o.set(levelText[level], levels[level])
	}
	if levels[""] > 0 {
		o.set(levelText[""], levels[""])
	}
	return o
}

func rowCounts(row MethodsSummaryRow) *object {
	levels := map[string]int{}
	for _, l := range row.Levels {
		levels[l.Level] = len(l.Students)
	}
	return levelCounts(levels)
}

// platoonSee says where the data of a reference are in the input by a platoon.
var platoonSee = map[string]string{
	"images":   "соотношение Я-реального и Я-идеального — в «образ_защитника»",
	"defender": "группы образа защитника — в «образ_защитника»",
	"signs":    "выраженность качеств — в «признаки»",
}

func platoonPointList(points []PlatoonPoint) any {
	if len(points) == 0 {
		return noAnswers
	}
	list := []any{}
	for _, p := range points {
		o := newObject()
		switch p.Kind {
		case "see":
			if p.Code == "stories" {
				o.set("рассказали_о_деятельности", p.Told)
			} else {
				o.set("см", platoonSee[p.Code])
			}
		case "activity":
			o.set("вид", p.Text)
			o.set("уровни", levelCounts(p.Levels))
		case "method":
			o.set("методика", p.Text)
			o.set("уровни", levelCounts(p.Levels))
		case "question":
			o.set("вопрос_анкеты", p.Text)
			o.set("уровни", levelCounts(p.Levels))
		case "frequency":
			o.set("действие", p.Text)
			answers := newObject()
			for n, count := range p.Answers {
				if count > 0 {
					answers.set(frequencyAnswers[n], count)
				}
			}
			o.set("за_четыре_недели", answers)
		case "indicator":
			o.set("показатель", p.Text)
			o.put("средний_балл", scoreText(p.Mean))
			o.set("оценили", p.Rated)
			if p.Condition {
				o.set("недостаточно", p.Low)
			}
		case "story":
			o.set("вопрос", p.Text)
			o.set("рассказали", p.Told)
			if len(p.Texts) > 0 {
				texts := []any{}
				for _, text := range p.Texts {
					texts = append(texts, text)
				}
				o.set("ответы", texts)
			}
		}
		list = append(list, o)
	}
	return list
}

// The texts of the climate in the input of test-platoon/v2 (Инструкция по взводу v2.md).
const (
	platoonClimateNote     = "по оценкам самих студентов взвода; уровня климата нет"
	platoonClimateEqual    = "все стороны климата оценены одинаково"
	platoonClimateSee      = "климат взвода — в «климат_взвода»"
	platoonClimateCoverage = "по взводу"
	platoonClimateOwnNote  = "по оценкам самих студентов"
)

// platoonPayloadFor builds the input of an instruction version. The input of
// v1 stays as it was: the conclusions made are found by its fingerprint. From
// v2 on it has the second edition of the catalogue and, when three students
// at least answered all the pairs of the method, the climate of the platoon
// (Д5, Д6); from v3 on, the dictionary of the terms (Инструкция по взводу v3.md).
func (a *Service) platoonPayloadFor(version string, data PlatoonPortraitData) *object {
	payload := a.platoonPayload(data)
	if !climateInput(testPlatoonTask, version) {
		return payload
	}
	a.addPlatoonClimate(payload, data, version)
	if termsInput(testPlatoonTask, version) {
		signs := []string{}
		for _, sign := range data.EnvironmentSigns {
			signs = append(signs, sign.ID)
		}
		addTerms(payload, signs)
	}
	return payload
}

// addPlatoonClimate adds to the input of test-platoon/v2 and newer the second
// edition of the catalogue and the climate of the platoon, when there is one.
func (a *Service) addPlatoonClimate(payload *object, data PlatoonPortraitData, version string) {
	numbers := map[string]int{}
	for n, g := range a.defender.Groups {
		numbers[g.ID] = n + 1
	}
	payload.set("каталог_рекомендаций", a.catalogueList(numbers, catalogueEdition(testPlatoonTask, version)))
	climate := data.Climate
	if climate == nil {
		return
	}
	o := newObject()
	o.set("оценили", len(climate.Rated))
	if climate.Equal {
		o.set("пометка", platoonClimateNote+"; "+platoonClimateEqual)
	} else {
		o.set("сильнее_всего", anyList(climate.Strongest))
		o.set("слабее_всего", anyList(climate.Weakest))
		o.set("пометка", platoonClimateNote)
	}
	payload.insertAfter("признаки_среды", "климат_взвода", o)
	for n, sign := range objectList(payload.values["признаки_среды"]) {
		if n < len(a.defender.Signs) && a.defender.Signs[n].Climate {
			sign.set("пометка", platoonClimateSee)
		}
	}
	a.climateCriterion(payload, platoonClimateCoverage, platoonClimateOwnNote, platoonClimateSee)
}

// platoonPayload is the input of the model by a platoon of test-platoon/v1:
// numbers only, no names.
func (a *Service) platoonPayload(data PlatoonPortraitData) *object {
	payload := newObject()
	payload.set("студентов", data.Students)

	methods := []any{}
	for _, row := range data.Methods {
		o := newObject()
		o.set("компонент", portraitComponentNames[row.Component])
		o.set("показатель", row.Name)
		o.set("уровни", rowCounts(row))
		methods = append(methods, o)
	}
	payload.set("методики", methods)

	defs := map[string]model.DefenderGroupDef{}
	numbers := map[string]int{}
	for n, g := range a.defender.Groups {
		defs[g.ID], numbers[g.ID] = g, n+1
	}
	kindNames := map[string]string{}
	for _, k := range a.bank.ActivityKinds {
		kindNames[k.ID] = k.Name
	}
	indicatorNames := map[string]string{}
	for _, ind := range a.bank.Indicators {
		indicatorNames[ind.ID] = ind.Name
	}
	groups := []any{}
	for _, g := range data.Defender {
		o := newObject()
		o.set("номер", numbers[g.ID])
		o.set("группа", g.Name)
		o.put("я_реальное_среднее", scoreText(g.Real))
		o.put("я_идеальное_среднее", scoreText(g.Ideal))
		situations := newObject()
		for _, c := range g.Situations {
			if c.ID != "" || len(c.Students) > 0 {
				situations.set(c.Text, len(c.Students))
			}
		}
		o.set("ситуации", situations)
		kinds := []any{}
		for _, link := range defs[g.ID].Kinds {
			k := newObject()
			k.set("вид", kindNames[link.ID])
			k.put("формы", link.Forms)
			kinds = append(kinds, k)
		}
		o.set("виды_деятельности", kinds)
		conditions := []any{}
		for _, id := range g.Conditions {
			conditions = append(conditions, indicatorNames[id])
		}
		o.set("условия_среды", conditions)
		groups = append(groups, o)
	}
	payload.set("образ_защитника", groups)

	congruence := newObject()
	for _, c := range data.Congruence {
		if c.ID != "none" || len(c.Students) > 0 {
			congruence.set(c.Text, len(c.Students))
		}
	}
	payload.set("согласованность_образов", congruence)

	relations := []any{}
	for _, r := range data.Relations {
		o := newObject()
		o.set("признак", r.Name)
		list := []any{}
		for _, ind := range r.Indicators {
			row := newObject()
			row.set("показатель", ind.Name)
			row.put("средний_балл", scoreText(ind.Score))
			list = append(list, row)
		}
		o.set("показатели", list)
		relations = append(relations, o)
	}
	payload.set("признаки", relations)

	identity := []any{}
	for _, block := range data.Identity {
		total := 0
		leading := newObject()
		for _, c := range block.Leading {
			total += len(c.Students)
			if c.ID != "" || len(c.Students) > 0 {
				leading.set(c.Text, len(c.Students))
			}
		}
		if total == 0 {
			continue
		}
		o := newObject()
		o.set("профессия", block.Name)
		o.set("ведущий_статус", leading)
		o.set("пометка", "студент, у которого два статуса выражены одинаково, учтён у каждого")
		identity = append(identity, o)
	}
	if len(identity) > 0 {
		payload.set("профессиональная_идентичность", identity)
	}

	criteria := map[string][]PlatoonCriterion{}
	for _, k := range data.Criteria {
		criteria[k.Kind] = k.Criteria
	}
	kinds := []any{}
	for _, row := range data.Kinds {
		o := newObject()
		o.set("вид", row.Name)
		o.set("уровни", rowCounts(row))
		list := []any{}
		for _, c := range criteria[row.ID] {
			item := newObject()
			item.set("критерий", c.Name)
			item.set("покрытие", coverageText(c.Coverage))
			item.put("пометка", c.Note)
			if c.Coverage == "нет" {
				item.set("данные", noData)
			} else {
				item.set("данные", platoonPointList(c.Points))
			}
			list = append(list, item)
		}
		if len(list) > 0 {
			o.set("критерии", list)
		}
		kinds = append(kinds, o)
	}
	payload.set("виды_деятельности", kinds)

	conditions := []any{}
	for _, c := range data.Conditions {
		o := newObject()
		o.set("показатель", c.Name)
		o.set("уровень_условий", c.Level)
		o.put("средний_балл", scoreText(c.Mean))
		o.set("оценили", c.Rated)
		o.set("недостаточно", len(c.Low))
		conditions = append(conditions, o)
	}
	payload.set("условия_среды", conditions)

	signs := []any{}
	for _, sign := range data.EnvironmentSigns {
		o := newObject()
		o.set("признак", sign.Name)
		o.set("по_главе_1", sign.Description)
		o.put("пометка", sign.Note)
		o.set("данные", platoonPointList(sign.Points))
		signs = append(signs, o)
	}
	payload.set("признаки_среды", signs)
	payload.set("каталог_рекомендаций", a.catalogueList(numbers, 1))
	return payload
}

// climate finds the climate and the cohesion of the platoon, which the test
// does not measure; «сотрудничество и сплочённость» is the name of a sign of
// the environment in the input and passes.
var climate = regexp.MustCompile(`(?i)климат|сплоч[её]нн?ост`)
var cohesionSign = regexp.MustCompile(`(?i)сотрудничеств[а-я]*\s+и\s+$`)

func climateProblems(field, text string) []string {
	for _, at := range climate.FindAllStringIndex(text, -1) {
		if strings.HasPrefix(strings.ToLower(text[at[0]:]), "климат") || !cohesionSign.MatchString(text[:at[0]]) {
			return []string{field + ": о климате или сплочённости взвода, которых тест не меряет"}
		}
	}
	return nil
}

// climateLevel is a word of a level next to the climate: the climate has no
// level (decision Д5); «благоприятный» and «неблагоприятный» name the levels
// of the key of Lutoshkin (approved by the user on 25 October 2026).
var climateLevel = regexp.MustCompile(`высок|средн|низк|уровн|благоприят`)

// sentenceBreak splits a text into sentences for the check of the climate.
var sentenceBreak = regexp.MustCompile(`[.!?;]`)

// climateReach is how many letters around the word «климат» the check looks.
const climateReach = 60

func climateLevelProblems(field, text string) []string {
	for _, sentence := range sentenceBreak.Split(lowerText(text), -1) {
		letters := []rune(sentence)
		word := []rune("климат")
		for at := 0; at+len(word) <= len(letters); at++ {
			if string(letters[at:at+len(word)]) != string(word) {
				continue
			}
			around := string(letters[max(0, at-climateReach):min(len(letters), at+len(word)+climateReach)])
			if found := climateLevel.FindString(around); found != "" {
				return []string{fmt.Sprintf("%s: у климата нет уровня («%s…») — описывай климат сторонами, которые выражены сильнее и слабее, без «высокий», «низкий» и «благоприятный»", field, found)}
			}
		}
	}
	return nil
}

// checkPlatoonAnswer returns what is wrong with an answer to test-platoon/v1
// and v2: v2 differs only by the input.
func checkPlatoonAnswer(output map[string]any, payload *object) []string {
	problems := []string{}
	texts := map[string]string{}
	for _, key := range []string{"overview", "components", "identity", "ideal", "activity", "conclusion"} {
		texts[key] = outputText(output, key)
		problems = append(problems, fieldProblems(key, texts[key])...)
	}
	problems = append(problems, sentenceProblem("overview", texts["overview"], 1, 1)...)
	problems = append(problems, sentenceProblem("components", texts["components"], 3, 5)...)
	problems = append(problems, sentenceProblem("identity", texts["identity"], 1, 2)...)
	problems = append(problems, sentenceProblem("ideal", texts["ideal"], 2, 2)...)
	problems = append(problems, sentenceProblem("activity", texts["activity"], activityLeast, 4)...)
	problems = append(problems, sentenceProblem("conclusion", texts["conclusion"], 2, 2)...)
	for _, key := range []string{"overview", "ideal", "activity", "conclusion"} {
		problems = append(problems, ratedLevelProblems(key, texts[key])...)
	}

	identity, given := payload.values["профессиональная_идентичность"]
	problems = append(problems, identityProblems(texts["identity"], given, nil)...)
	if given && strings.TrimSpace(texts["identity"]) != identityAbsent {
		problems = append(problems, platoonIdentityProblems("identity", texts["identity"], identity)...)
	}

	problems = append(problems, recommendationProblems(output, payload, recommendationLeast, 5, nil)...)
	problems = append(problems, repeatedDirectionProblems(output)...)
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
	// The climate is measured when the input has it (test-platoon/v2 and a
	// platoon that took the method); it has no level either way.
	_, measured := payload.values["климат_взвода"]
	for _, key := range sortedTextKeys(texts) {
		if !measured {
			problems = append(problems, climateProblems(key, texts[key])...)
		}
		problems = append(problems, climateLevelProblems(key, texts[key])...)
	}
	problems = append(problems, unfoundStateProblems(texts, payload)...)
	total, _ := payload.values["студентов"].(int)
	problems = append(problems, platoonCountProblems(texts, total)...)
	return problems
}

// platoonIdentityProblems checks that a text names, by block, the leading
// status most students of the platoon have.
func platoonIdentityProblems(field, text string, identity any) []string {
	problems := []string{}
	lower := normalizedText(text)
	for _, block := range objectList(identity) {
		counts, _ := block.values["ведущий_статус"].(*object)
		best, top := 0, []string{}
		for _, name := range counts.keys {
			n, _ := counts.values[name].(int)
			switch {
			case name == levelText[""] || n == 0 || n < best:
			case n > best:
				best, top = n, []string{name}
			default:
				top = append(top, name)
			}
		}
		named := len(top) == 0
		for _, name := range top {
			named = named || statusNamed(lower, name)
		}
		// The run of 25 October: told only the block, the model kept
		// another name for five attempts; it is told the status itself.
		if !named {
			problems = append(problems, fmt.Sprintf("%s: не назван самый частый ведущий статус (%s) — «%s»", field, lowerFirst(objectText(block, "профессия")), strings.Join(top, "» или «")))
		}
	}
	return problems
}

func sortedTextKeys(texts map[string]string) []string {
	keys := make([]string, 0, len(texts))
	for key := range texts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
