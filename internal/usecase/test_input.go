package usecase

import (
	"fmt"
	"slices"
	"sort"
	"vucdiagnostic/internal/promptbank"
)

// The input of the language model for the portrait by the test: the counted
// portrait without the name, platoon or dates. Keys are in Russian, as in
// the autobiography input; the order of keys follows the first chapter.

func scoreText(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func indicatorList(rows []IndicatorScore) []any {
	list := []any{}
	for _, row := range rows {
		item := newObject()
		item.set("показатель", row.Name)
		item.put("балл", scoreText(row.Score))
		list = append(list, item)
	}
	return list
}

func imageObject(pair ImagePair) *object {
	o := newObject()
	o.set("название", pair.Name)
	o.put("я_реальное", scoreText(pair.Real))
	o.put("я_идеальное", scoreText(pair.Ideal))
	o.put("как_видят_другие", scoreText(pair.Mirror))
	o.put("чего_ждут_другие", scoreText(pair.Norm))
	o.put("расхождение", scoreText(pair.Gap))
	o.put("оценка_расхождения", pair.GapText)
	o.put("самооценка", pair.SelfText)
	return o
}

// testPayload turns the counted portrait into the model input.
func testPayload(data *TestPortraitData) *object {
	payload := newObject()

	methods := []any{}
	for _, m := range data.Methods {
		o := newObject()
		o.set("компонент", portraitComponentNames[m.Component])
		o.set("показатель", m.Name)
		o.set("методика", m.Method)
		if m.Score != nil {
			o.set("баллы", fmt.Sprintf("%d (диапазон %d–%d)", *m.Score, m.Min, m.Max))
		}
		o.set("уровень", m.LevelText)
		methods = append(methods, o)
	}
	payload.set("методики", methods)

	images := newObject()
	components := []any{}
	for _, pair := range data.Components {
		components = append(components, imageObject(pair))
	}
	images.set("по_компонентам", components)
	qualities := append([]ImagePair(nil), data.Images...)
	sort.SliceStable(qualities, func(i, j int) bool {
		gap := func(p ImagePair) float64 {
			if p.Gap == nil {
				return -100
			}
			return *p.Gap
		}
		return gap(qualities[i]) > gap(qualities[j])
	})
	largest, below := []any{}, []any{}
	for _, pair := range qualities {
		if pair.Gap == nil {
			continue
		}
		if *pair.Gap >= 3 && len(largest) < 5 {
			largest = append(largest, imageObject(pair))
		}
		if *pair.Gap < 0 {
			below = append(below, imageObject(pair))
		}
	}
	images.set("наибольшие_расхождения", largest)
	images.set("идеальное_ниже_реального", below)
	payload.set("образы_я", images)

	signs := []any{}
	for _, g := range data.Signs {
		o := newObject()
		o.set("признак", g.Name)
		o.set("показатели", indicatorList(g.Indicators))
		signs = append(signs, o)
	}
	payload.set("признаки", signs)

	conditions := []any{}
	for _, g := range data.Conditions {
		o := newObject()
		o.set("уровень_условий", g.Name)
		o.set("показатели", indicatorList(g.Indicators))
		conditions = append(conditions, o)
	}
	payload.set("условия_среды", conditions)

	kinds := []any{}
	for _, k := range data.Kinds {
		o := newObject()
		o.set("вид", k.Name)
		o.set("уровень", k.LevelText)
		answers := []any{}
		for _, sign := range k.Signs {
			a := newObject()
			a.set("признак", sign.Text)
			a.put("ответ", sign.Answer)
			a.set("уровень_признака", sign.LevelText)
			answers = append(answers, a)
		}
		o.set("ответы_анкеты", answers)
		if len(k.Indicators) > 0 {
			o.set("показатели", indicatorList(k.Indicators))
		}
		actions := []any{}
		for _, f := range k.Frequency {
			a := newObject()
			a.set("действие", f.Text)
			a.set("за_четыре_недели", f.Answer)
			actions = append(actions, a)
		}
		if len(actions) > 0 {
			o.set("частота_действий", actions)
		}
		kinds = append(kinds, o)
	}
	payload.set("виды_деятельности", kinds)

	stories := []any{}
	for _, answer := range data.OpenAnswers {
		o := newObject()
		o.put("раздел", answer.Topic)
		o.set("вопрос", answer.Question)
		o.set("ответ", answer.Answer)
		stories = append(stories, o)
	}
	payload.set("рассказы_о_деятельности", stories)
	return payload
}

// insertAfter puts a key right after another one, or at the end when there is
// no such key: the input of a new version keeps the order of the first chapter.
func (o *object) insertAfter(after, key string, value any) {
	if _, seen := o.values[key]; seen {
		o.values[key] = value
		return
	}
	o.values[key] = value
	at := len(o.keys)
	for n, k := range o.keys {
		if k == after {
			at = n + 1
		}
	}
	o.keys = slices.Insert(o.keys, at, key)
}

// testPayloadFor builds the input of an instruction version. The input of v1
// stays as it was: the stored portraits are found by its fingerprint. From v2
// on it has the image of the defender, the congruence of the images, the
// professional identity when the method was given, and the catalogue of
// recommendations (Вики/Тестирование/План портрета test-v2.md, step 3).
func (a *Service) testPayloadFor(version string, data *TestPortraitData) *object {
	payload := testPayload(data)
	if n, _ := promptbank.Number(version); n < 2 {
		return payload
	}
	numbers := map[string]int{}
	for n, g := range a.defender.Groups {
		numbers[g.ID] = n + 1
	}
	if data.Congruence != nil {
		c := newObject()
		c.put("коэффициент", scoreText(data.Congruence.Value))
		c.set("оценка", data.Congruence.Text)
		payload.insertAfter("образы_я", "согласованность_образов", c)
	}
	payload.insertAfter("признаки", "образ_защитника", defenderList(data.Defender, numbers))
	criteria := map[string][]CriterionData{}
	for _, k := range data.Criteria {
		criteria[k.Kind] = k.Criteria
	}
	for n, kind := range objectList(payload.values["виды_деятельности"]) {
		if n < len(data.Kinds) && len(criteria[data.Kinds[n].ID]) > 0 {
			kind.set("критерии", criteriaList(criteria[data.Kinds[n].ID], func(c CriterionData) any { return pointList(c.Data) }))
		}
	}
	if len(data.EnvironmentSigns) > 0 {
		signs := []any{}
		for _, sign := range data.EnvironmentSigns {
			o := newObject()
			o.set("признак", sign.Name)
			o.set("по_главе_1", sign.Description)
			o.put("пометка", sign.Note)
			o.set("данные", pointList(sign.Data))
			signs = append(signs, o)
		}
		payload.insertAfter("условия_среды", "признаки_среды", signs)
	}
	if len(data.Identity) > 0 {
		payload.insertAfter("образ_защитника", "профессиональная_идентичность", identityList(data.Identity))
	}
	catalogue := []any{}
	types := map[string]string{}
	for _, kind := range a.defender.Types {
		types[kind.ID] = kind.Name
	}
	for _, d := range a.defender.Directions {
		o := newObject()
		o.set("код", d.ID)
		o.set("направление", d.Name)
		o.set("тип", types[d.Type])
		// []any, not []int: the cleaning of personal data walks only these.
		groups := []any{}
		for _, id := range d.Groups {
			groups = append(groups, numbers[id])
		}
		o.set("группы", groups)
		catalogue = append(catalogue, o)
	}
	payload.set("каталог_рекомендаций", catalogue)
	return payload
}

func defenderList(groups []DefenderGroup, numbers map[string]int) []any {
	list := []any{}
	for _, g := range groups {
		o := newObject()
		o.set("номер", numbers[g.ID])
		o.set("группа", g.Name)
		o.put("я_реальное", scoreText(g.Real))
		o.put("я_идеальное", scoreText(g.Ideal))
		o.put("до_образа", scoreText(g.Distance))
		o.put("разрыв", scoreText(g.Gap))
		o.set("ситуация", g.SituationText)
		o.put("тип_рекомендации", g.TypeText)
		methods := []any{}
		for _, m := range g.Methods {
			row := newObject()
			row.set("показатель", m.Name)
			row.set("уровень", m.LevelText)
			methods = append(methods, row)
		}
		if len(methods) > 0 {
			o.set("методики", methods)
		}
		kinds := []any{}
		for _, k := range g.Kinds {
			row := newObject()
			row.set("вид", k.Name)
			row.put("формы", k.Forms)
			row.set("уровень", k.LevelText)
			kinds = append(kinds, row)
		}
		o.set("виды_деятельности", kinds)
		o.set("условия_среды", indicatorList(g.Conditions))
		o.put("пометка", g.Note)
		list = append(list, o)
	}
	return list
}

func identityList(profiles []IdentityProfile) []any {
	list := []any{}
	for _, p := range profiles {
		o := newObject()
		o.set("профессия", p.Name)
		statuses := []any{}
		for _, status := range p.Statuses {
			row := newObject()
			row.set("статус", status.Name)
			if status.Score != nil {
				row.set("баллы", fmt.Sprintf("%d (диапазон 0–%d)", *status.Score, status.Max))
			}
			row.put("выраженность", status.Range)
			statuses = append(statuses, row)
		}
		o.set("статусы", statuses)
		o.set("ведущий_статус", p.LeadingText)
		list = append(list, o)
	}
	return list
}
