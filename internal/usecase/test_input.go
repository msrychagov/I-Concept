package usecase

import (
	"fmt"
	"sort"
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
