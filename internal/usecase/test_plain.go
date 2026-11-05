package usecase

import (
	"cmp"
	"slices"
	"strings"
	"vucdiagnostic/internal/promptbank"
)

// The input of test/v6 and test-platoon/v4 without the terms of the
// dissertation (Вики/Тестирование/План конкретных рекомендаций и Квена.md,
// decisions Д9–Д12, approved by the user on 27 October 2026). The run of
// Qwen 3.5 9B on 27 October wrote the terms it saw in the input: the names of
// the groups of the image, the keys «я_реальное», the dictionary itself. The
// input of v6 gives what the terms mean under plain names; the words the check
// forbids are nowhere in it. The words of the student himself stay as he wrote
// them.

// plainVersion is, by task, the first version whose input is plain.
var plainVersion = map[string]int{testPortraitTask: 6, testPlatoonTask: 4}

func plainInput(task, version string) bool {
	n, _ := promptbank.Number(version)
	return n >= plainVersion[task]
}

// plainKeys are the keys of the input renamed.
var plainKeys = map[string]string{
	"я_реальное":               "какой_сейчас",
	"я_идеальное":              "каким_хочет_стать",
	"я_реальное_среднее":       "какой_сейчас_среднее",
	"я_идеальное_среднее":      "каким_хочет_стать_среднее",
	"по_компонентам":           "по_сторонам",
	"компонент":                "сторона",
	"признаки":                 "отношения",
	"признак":                  "название",
	"уровень_признака":         "уровень_ответа",
	"признаки_среды":           "среда_по_главе_1",
	"идеальное_ниже_реального": "планка_ниже_самооценки",
}

// plainSituation is the situation «Я-идеальное ниже Я-реального» in plain words.
const plainSituation = "Планка ниже, чем оценка себя сейчас"

// plainGroup is a group of the image under a short plain name, with all its
// qualities (the meanings of the dictionary of v5).
type plainGroup struct{ Short, Qualities string }

var plainGroups = map[string]plainGroup{
	"Государственно-патриотические качества":                  {"верность долгу и преданность Отечеству", "верность долгу и присяге, преданность Отечеству, готовность защищать страну, уважение к её истории и традициям, честность, мужество, решительность"},
	"Профессионально-компетентностные и когнитивные качества": {"профессионализм и умение принимать решения", "профессионализм, постоянное развитие и освоение новой техники, аналитическое мышление, умение оценить обстановку и принять обоснованное решение, самообразование"},
	"Коммуникативные и коллективистские качества":             {"товарищество и культура отношений", "товарищество, коллективизм, взаимовыручка, уважение к людям и культура отношений"},
	"Дисциплинарные и волевые качества":                       {"дисциплина и собранность", "дисциплинированность, исполнительность, собранность, ответственность, организованность, стойкость в трудностях"},
	"Физические и психологические качества":                   {"физическая подготовка и самообладание", "физическая подготовка, стрессоустойчивость, самообладание в трудных условиях"},
}

// plainComponentNames are the components of the core as the sides of v6.
var plainComponentNames = map[string]string{"CS": "ценности и мотивы", "EV": "владение собой", "DP": "действия и опыт"}

// plainValues are the texts of the input that are replaced as a whole.
var plainValues = map[string]string{
	"ценностно-смысловой":          plainComponentNames["CS"],
	"эмоционально-волевой":         plainComponentNames["EV"],
	"деятельностно-практический":   plainComponentNames["DP"],
	"любовь":                       "отношение к людям",
	"соборность":                   "отношение к взводу и товарищам",
	"патриотизм":                   "отношение к Отечеству",
	"Я-идеальное ниже Я-реального": plainSituation,
	"Организация сотрудничества":   "Организация совместной работы",
	"Ясность и согласованность норм":                  "Понятные и согласованные требования",
	"Личностно ориентированное взаимодействие":        "Доверие и личный пример",
	"Сотрудничество и сплочённость":                   "Совместная работа и сплочённость",
	"Значимая деятельность и инициатива":              "Значимое дело и инициатива",
	"Анкета деятельностного компонента (табл. 18–19)": "Анкета о деятельности и опыте",
	"Я-идеальное одинаково по всем качествам — согласованность не определяется.": "Каким хочет стать — одинаково по всем качествам: согласованность не определяется.",
	"Я-реальное одинаково по всем качествам — согласованность не определяется.":  "Какой сейчас — одинаково по всем качествам: согласованность не определяется.",
}

// plainParts are the pieces of longer texts replaced, the longest first.
var plainParts = strings.NewReplacer(
	"соотнесение Я-реального и Я-идеального", "сопоставление того, какой я сейчас, с тем, каким хочу стать",
	"Я-реальное и Я-идеальное — в «образы_я»", "какой сейчас и каким хочет стать — в «образы_я»",
	"соотношение Я-реального и Я-идеального", "какой сейчас и каким хочет стать",
	"в «признаки_среды», «сотрудничество и сплочённость»", "в «среда_по_главе_1», «совместная работа и сплочённость»",
	"в «признаки»", "в «отношения»",
	"к собственному Я-идеальному", "к тому, каким хочу стать",
	"стремиться к Я-идеальному", "стремиться к тому, каким хочу стать",
	"соотносится с Я-идеальным", "соотносится с тем, каким студент хочет стать",
	"эмоционально-волевая устойчивость", "умение владеть собой",
	"Личностно ориентированное взаимодействие:", "Доверие и личный пример:",
)

// plainText is a text of the input in plain words.
func plainText(text string) string {
	if plain, ok := plainValues[text]; ok {
		return plain
	}
	if g, ok := plainGroups[text]; ok {
		return g.Short
	}
	return plainParts.Replace(text)
}

// studentKeys hold the words of the student himself, which stay as they are.
var studentKeys = map[string]bool{"рассказ": true, "ответы": true}

// plainValue rewrites a value of the input in plain words: the keys, the
// texts, and a group of the image gets its qualities after its name.
func plainValue(value any, stories bool) any {
	switch v := value.(type) {
	case *object:
		plainObject(v, stories)
		return v
	case []any:
		for n := range v {
			v[n] = plainValue(v[n], stories)
		}
		return v
	case string:
		return plainText(v)
	}
	return value
}

func plainObject(o *object, stories bool) {
	keys := make([]string, 0, len(o.keys))
	values := map[string]any{}
	for _, key := range o.keys {
		value := o.values[key]
		inStories := stories || key == "рассказы_о_деятельности"
		if !studentKeys[key] && !(stories && key == "ответ") {
			value = plainValue(value, inStories)
		}
		name := key
		if renamed, ok := plainKeys[key]; ok {
			name = renamed
		} else {
			name = plainText(key)
		}
		keys = append(keys, name)
		values[name] = value
		if key == "группа" {
			original, _ := o.values[key].(string)
			if g, ok := plainGroups[original]; ok {
				keys = append(keys, "качества")
				values["качества"] = g.Qualities
			}
		}
	}
	o.keys, o.values = keys, values
}

// remove drops a key.
func (o *object) remove(key string) {
	if _, seen := o.values[key]; !seen {
		return
	}
	delete(o.values, key)
	o.keys = slices.DeleteFunc(o.keys, func(k string) bool { return k == key })
}

// rename gives a key another name in its place.
func (o *object) rename(key, name string, value any) {
	for n, k := range o.keys {
		if k == key {
			o.keys[n] = name
			delete(o.values, key)
			o.values[name] = value
			return
		}
	}
}

// insertBefore puts a key right before another one, or at the end.
func (o *object) insertBefore(before, key string, value any) {
	if _, seen := o.values[key]; seen {
		o.values[key] = value
		return
	}
	o.values[key] = value
	at := len(o.keys)
	for n, k := range o.keys {
		if k == before {
			at = n
			break
		}
	}
	o.keys = slices.Insert(o.keys, at, key)
}

// The ratings of the conditions in words (decision Д11), like the sides of
// the climate: the three highest and the three lowest, all of a rating equal
// at the border, a rating of 2 and below insufficient.
const (
	ratedHigher       = "выше других"
	ratedLower        = "ниже других"
	ratedInsufficient = "недостаточно"
	ratedEven         = "наравне с другими"
	ratedSides        = 3
)

// conditionWords gives every condition with a score its rating in words.
func conditionWords(scores map[string]float64) map[string]string {
	values := []float64{}
	for _, v := range scores {
		values = append(values, v)
	}
	slices.Sort(values)
	words := map[string]string{}
	if len(values) == 0 {
		return words
	}
	// k conditions a side, fewer when there are few: a side takes no more
	// than half of them.
	k := min(ratedSides, len(values)/2)
	lowest, highest := values[0], values[len(values)-1]
	low, high := lowest, highest
	if k > 0 {
		low, high = values[k-1], values[len(values)-k]
	}
	if high <= low {
		low, high = lowest, highest
	}
	for name, v := range scores {
		switch {
		case v <= lowCondition:
			words[name] = ratedInsufficient
		case lowest == highest:
			words[name] = ratedEven
		case v >= high:
			words[name] = ratedHigher
		case v <= low:
			words[name] = ratedLower
		default:
			words[name] = ratedEven
		}
	}
	return words
}

// rateConditions puts the rating in words in place of the score of every
// condition in the input; a list of names of conditions becomes a list of
// conditions with their rating.
func rateConditions(value any, words map[string]string) {
	switch v := value.(type) {
	case *object:
		name := objectText(v, "показатель")
		for _, key := range []string{"балл", "средний_балл"} {
			if word, ok := words[name]; ok {
				if _, has := v.values[key]; has {
					v.rename(key, "оценка", word)
				}
			}
		}
		for _, key := range v.keys {
			if list, ok := v.values[key].([]any); ok && key == "условия_среды" {
				for n, item := range list {
					if text, ok := item.(string); ok {
						o := newObject()
						o.set("показатель", text)
						o.put("оценка", words[text])
						list[n] = o
					}
				}
			}
			rateConditions(v.values[key], words)
		}
	case []any:
		for _, item := range v {
			rateConditions(item, words)
		}
	}
}

// statusMeanings are what the statuses of identity mean, by J. Marcia, on
// whom the method of Azbel is built (decision Д10).
var statusMeanings = map[string]string{
	"сформированная":              "выбор сделан самостоятельно, после раздумий, и студент связывает с ним свои планы",
	"мораторий (кризис выбора)":   "студент в поиске: сомневается, сравнивает, решения ещё нет",
	"навязанная":                  "выбор сделан под влиянием других, без собственного поиска",
	"неопределённая идентичность": "студент пока всерьёз не задумывается о выборе и не ищет его",
}

// shorterKinds drops from a kind of activity of the input of a student the
// answers, the indicators and the actions its criteria already have (decision Д12).
func shorterKinds(payload *object) {
	for _, kind := range objectList(payload.values["виды_деятельности"]) {
		covered := map[string]bool{}
		for _, c := range objectList(kind.values["критерии"]) {
			for _, point := range objectList(c.values["данные"]) {
				for _, key := range []string{"вопрос_анкеты", "показатель", "действие"} {
					if text := objectText(point, key); text != "" {
						covered[text] = true
					}
				}
			}
		}
		if len(covered) == 0 {
			continue
		}
		for list, key := range map[string]string{"ответы_анкеты": "признак", "показатели": "показатель", "частота_действий": "действие"} {
			items, ok := kind.values[list].([]any)
			if !ok {
				continue
			}
			kept := []any{}
			for _, item := range items {
				if o, ok := item.(*object); !ok || !covered[objectText(o, key)] {
					kept = append(kept, item)
				}
			}
			if len(kept) == 0 {
				kind.remove(list)
			} else {
				kind.set(list, kept)
			}
		}
	}
}

// plainTestPayload turns the input of test/v5 without the dictionary into
// that of v6.
func (a *Service) plainTestPayload(payload *object, data *TestPortraitData) {
	scores := map[string]float64{}
	for _, level := range data.Conditions {
		for _, ind := range level.Indicators {
			if ind.Score != nil {
				scores[ind.Name] = *ind.Score
			}
		}
	}
	shorterKinds(payload)
	rateConditions(payload, conditionWords(scores))
	for _, block := range objectList(payload.values["профессиональная_идентичность"]) {
		leading := objectText(block, "ведущий_статус")
		meanings := []string{}
		for _, status := range objectList(block.values["статусы"]) {
			name := objectText(status, "статус")
			if strings.Contains(leading, name) && statusMeanings[name] != "" {
				meanings = append(meanings, name+" — "+statusMeanings[name])
			}
		}
		if len(meanings) == 1 {
			_, meaning, _ := strings.Cut(meanings[0], " — ")
			meanings[0] = meaning
		}
		if len(meanings) > 0 {
			block.insertAfter("ведущий_статус", "что_значит", strings.Join(meanings, "; "))
		}
	}
	plainObject(payload, false)
}

// plainPlatoonPayload turns the input of test-platoon/v3 without the
// dictionary into that of v4: the ratings by the means of the platoon, the
// meanings of the statuses and the groups of students for separate work.
func (a *Service) plainPlatoonPayload(payload *object, data PlatoonPortraitData) {
	scores := map[string]float64{}
	for _, c := range data.Conditions {
		if c.Mean != nil {
			scores[c.Name] = *c.Mean
		}
	}
	rateConditions(payload, conditionWords(scores))
	for _, block := range objectList(payload.values["профессиональная_идентичность"]) {
		counts, _ := block.values["ведущий_статус"].(*object)
		meanings := newObject()
		for _, name := range counts.keys {
			if n, _ := counts.values[name].(int); n > 0 && statusMeanings[name] != "" {
				meanings.set(name, statusMeanings[name])
			}
		}
		if len(meanings.keys) > 0 {
			block.insertAfter("ведущий_статус", "что_значат", meanings)
		}
	}
	if groups := a.workGroupList(data); len(groups) > 0 {
		payload.insertBefore("каталог_рекомендаций", "отдельная_работа", groups)
	}
	plainObject(payload, false)
}

// sortedByLowest are the names of a map by its values, the lowest first, ties
// in the order given.
func sortedByLowest(order []string, values map[string]float64) []string {
	names := slices.Clone(order)
	slices.SortStableFunc(names, func(x, y string) int { return cmp.Compare(values[x], values[y]) })
	return names
}
