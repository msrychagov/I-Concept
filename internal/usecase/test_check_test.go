package usecase

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// goodTestAnswerV2 answers test/v2 the way it asks, for the input it is given.
func goodTestAnswerV2(user string) map[string]any {
	user, _, _ = strings.Cut(user, "\n\n"+retryNote)
	var input struct {
		Methods []struct {
			Component string `json:"компонент"`
			Level     string `json:"уровень"`
		} `json:"методики"`
		Groups []struct {
			Number int    `json:"номер"`
			Type   string `json:"тип_рекомендации"`
		} `json:"образ_защитника"`
		Identity []struct {
			Leading string `json:"ведущий_статус"`
		} `json:"профессиональная_идентичность"`
		Catalogue []struct {
			Code   string `json:"код"`
			Type   string `json:"тип"`
			Groups []int  `json:"группы"`
		} `json:"каталог_рекомендаций"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	levels := map[string][]string{}
	for _, m := range input.Methods {
		levels[m.Component] = append(levels[m.Component], levelCode[m.Level])
	}
	component := func(code string) string { return componentLevelText(levels[portraitComponentNames[code]]) }
	text := []string{}
	for _, code := range []string{"CS", "EV", "DP"} {
		name := portraitComponentNames[code]
		text = append(text, strings.ToUpper(name[:2])+name[2:]+" компонент — "+component(code)+".")
	}
	text = append(text, "Самооценка согласуется с Я-идеальным.", "Сильнее выражен патриотизм, слабее — соборность.")
	identity := identityAbsent
	if len(input.Identity) > 0 {
		leading := []string{}
		for _, block := range input.Identity {
			leading = append(leading, block.Leading)
		}
		identity = "Ведущие статусы — " + strings.Join(leading, "; ") + "."
	}
	recommendations := []any{}
	for _, g := range input.Groups {
		for _, d := range input.Catalogue {
			if len(recommendations) < 3 && slices.Contains(d.Groups, g.Number) && (g.Type == "" || d.Type == g.Type) {
				recommendations = append(recommendations, map[string]any{"group": g.Number, "code": d.Code, "text": "Включать студента в эту работу, чтобы образ стал ближе."})
				break
			}
		}
	}
	return map[string]any{
		"portrait":        "Я-концепция студента складывается вокруг отношения к Отечеству.",
		"components":      map[string]any{"levels": map[string]any{"cs": component("CS"), "ev": component("EV"), "dp": component("DP")}, "text": strings.Join(text, " ")},
		"identity":        identity,
		"ideal":           "Ближе всего студент к образу в патриотических качествах. Дальше всего — в коллективистских, образ принят как свой.",
		"activity":        "Студент включён в деятельность ВУЦ неравномерно. Условия ВУЦ он оценивает выше в личностном плане. Слабее выражено сотрудничество, это сдерживает соборность.",
		"recommendations": recommendations,
		"conclusion":      "Студент патриотически направлен и устойчив. Главное в работе с ним — совместные задачи.",
	}
}

func TestComponentLevelByTheScales(t *testing.T) {
	for _, c := range []struct {
		levels []string
		want   string
	}{
		{[]string{"high"}, "высокий"},
		{[]string{"high", "high", ""}, "высокий"},
		{[]string{"high", "medium", "high"}, "преимущественно высокий"},
		{[]string{"low", "low", "medium", "high", "high"}, "низкий — высокий"},
		{[]string{"high", "medium"}, "средний — высокий"},
		{[]string{"medium", "high", "low"}, "низкий — средний — высокий"},
		{[]string{"", ""}, "не определён"},
		{nil, "не определён"},
	} {
		if got := componentLevelText(c.levels); got != c.want {
			t.Errorf("%v: %q, want %q", c.levels, got, c.want)
		}
	}
}

func TestSentenceCount(t *testing.T) {
	for text, want := range map[string]int{
		"Одно предложение.":                          1,
		"Первое. Второе! Третье?":                    3,
		"Методика А. А. Азбель показала мораторий.":  1,
		"Студент учится, т. е. Готовится к службе.":  1,
		"Он сказал «да». Потом ушёл.":                2,
		"Сформирован (на среднем уровне). Устойчив.": 2,
		"Высокий уровень… Средний.":                  2,
		"": 0,
		"Первое предложение без точки в конце":          1,
		"Конец с кавычкой.» Новое предложение здесь.":   2,
		"Указаны ценности и т. п. в описании студента.": 1,
	} {
		if got := sentenceCount(text); got != want {
			t.Errorf("%q: %d, want %d", text, got, want)
		}
	}
}

func TestTestAnswerV2Check(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	payload := a.testPayloadFor("v2", data)
	user := string(canonicalJSON(payload))
	if problems := checkTestAnswerV2(goodTestAnswerV2(user), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	expected := expectedComponentLevels(payload)
	kinds := map[string]string{}
	for _, k := range data.Kinds {
		kinds[k.ID] = k.LevelText
	}
	wrongKind := map[string]string{"высокий": "низком", "средний": "высоком", "низкий": "высоком"}[kinds["study"]]
	// A code of the catalogue that fits group 1 but is of another type than its situation.
	types := map[string]string{}
	for _, d := range objectList(payload.values["каталог_рекомендаций"]) {
		types[objectText(d, "код")] = objectText(d, "тип")
	}
	groupType := objectText(objectList(payload.values["образ_защитника"])[0], "тип_рекомендации")
	otherType := "R17"
	if types[otherType] == groupType {
		otherType = "R06"
	}
	for name, c := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"figure":        {func(a map[string]any) { a["portrait"] = "Студент оценивает себя на 7 из 10." }, "portrait: цифры в тексте"},
		"two sentences": {func(a map[string]any) { a["portrait"] = "Первое. Второе." }, "portrait: предложений 2 вместо 1"},
		"one in ideal":  {func(a map[string]any) { a["ideal"] = "Образ принят." }, "ideal: предложений 1 вместо 2"},
		"long activity": {func(a map[string]any) { a["activity"] = "Раз. Два. Три. Четыре. Пять." }, "activity: предложений 5 вместо 2–4"},
		"two sentences of activity": {func(a map[string]any) {
			a["activity"] = "Студент включён в деятельность ВУЦ; условия он оценивает выше. Слабее выражено сотрудничество, это сдерживает соборность."
		}, ""},
		"one sentence of activity": {func(a map[string]any) {
			a["activity"] = "Студент включён в деятельность ВУЦ."
		}, "activity: предложений 1 вместо 2–4"},
		"appointment hint": {func(a map[string]any) {
			a["recommendations"].([]any)[0].(map[string]any)["text"] = "Назначить студента наставником для младших."
		}, "пиши «поручить» или «привлечь к»"},
		"no conclusion": {func(a map[string]any) { delete(a, "conclusion") }, "нет поля conclusion"},
		"level by rule": {func(a map[string]any) {
			a["components"].(map[string]any)["levels"].(map[string]any)["ev"] = "выдающийся"
		}, "components.levels.ev: по правилу «" + expected["EV"] + "»"},
		"level in text": {func(a map[string]any) {
			c := a["components"].(map[string]any)
			wrong := map[bool]string{true: "низкий", false: "высокий"}[!strings.Contains(expected["CS"], "низк")]
			c["text"] = strings.Replace(c["text"].(string), "Ценностно-смысловой компонент — "+expected["CS"], "Ценностно-смысловой компонент — "+wrong, 1)
		}, "components.text: уровень компонента «ценностно-смысловой» не тот"},
		"component missing": {func(a map[string]any) {
			c := a["components"].(map[string]any)
			c["text"] = strings.Replace(c["text"].(string), "Деятельностно-практический компонент — "+expected["DP"]+".", "Опыт осмысливается.", 1)
		}, "components.text: не назван деятельностно-практический компонент"},
		"kind level": {func(a map[string]any) {
			a["activity"] = "Учебная деятельность — на " + wrongKind + " уровне. Условия он оценивает выше. Это сдерживает соборность."
		}, "activity: уровень «Учебная» не тот"},
		"identity absent phrase": {func(a map[string]any) { a["identity"] = identityAbsent }, "identity: методика проводилась"},
		"identity status":        {func(a map[string]any) { a["identity"] = "Студент в поиске своего пути." }, "identity: не назван ведущий статус"},
		"two recommendations": {func(a map[string]any) {
			a["recommendations"] = a["recommendations"].([]any)[:2]
		}, "recommendations: нужно от 3 до 4 пунктов"},
		"unknown code": {func(a map[string]any) {
			a["recommendations"].([]any)[0].(map[string]any)["code"] = "R99"
		}, "recommendations.1: кода нет в каталоге"},
		"code of another group": {func(a map[string]any) {
			a["recommendations"].([]any)[0].(map[string]any)["group"] = 5
			a["recommendations"].([]any)[0].(map[string]any)["code"] = "R01"
		}, "recommendations.1: направление не подходит к группе"},
		"type of another situation": {func(a map[string]any) {
			a["recommendations"].([]any)[0].(map[string]any)["group"] = 1
			a["recommendations"].([]any)[0].(map[string]any)["code"] = otherType
		}, "recommendations.1: тип направления не подходит"},
		"appointment": {func(a map[string]any) {
			a["conclusion"] = "Студента стоит назначить командиром. Это поможет."
		}, "conclusion: о назначении на должность"},
		"diagnosis": {func(a map[string]any) {
			a["ideal"] = "Признаков расстройства нет. Образ принят."
		}, "ideal: недопустимое слово"},
		"abbreviation is not a stop": {func(a map[string]any) {
			a["ideal"] = "Ближе всего — в патриотических качествах, т. е. образ принят. Дальше всего — в коллективистских."
		}, ""},
	} {
		answer := goodTestAnswerV2(user)
		c.mutate(answer)
		problems := strings.Join(checkTestAnswerV2(answer, payload), "; ")
		if c.problem == "" && problems != "" || c.problem != "" && !strings.Contains(problems, c.problem) {
			t.Errorf("%s: %q", name, problems)
		}
	}
}

// Without the method of identity the answer must say so in the fixed words.
func TestIdentityAbsentPhrase(t *testing.T) {
	a := testApp(t)
	payload := a.testPayloadFor("v2", a.TestPortrait(fixedTestSession(a, "pilot-2.4")))
	answer := goodTestAnswerV2(string(canonicalJSON(payload)))
	if problems := checkTestAnswerV2(answer, payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
	answer["identity"] = "Ведущий статус — мораторий."
	if problems := strings.Join(checkTestAnswerV2(answer, payload), "; "); !strings.Contains(problems, "без методики нужно ровно") {
		t.Fatalf("problems: %s", problems)
	}
}

// A psychological direction fits any group when a scale of the
// emotional-volitional component is low.
func TestPsychologicalDirectionNeedsALowVolitionalScale(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	for n := range data.Methods {
		if data.Methods[n].Component == "EV" {
			data.Methods[n].Level, data.Methods[n].LevelText = "high", levelText["high"]
		}
	}
	for n := range data.Defender {
		data.Defender[n].Type, data.Defender[n].TypeText = "support", "опора"
	}
	payload := a.testPayloadFor("v2", data)
	answer := goodTestAnswerV2(string(canonicalJSON(payload)))
	answer["recommendations"].([]any)[0].(map[string]any)["group"] = 5
	answer["recommendations"].([]any)[0].(map[string]any)["code"] = "R21"
	if problems := strings.Join(checkTestAnswerV2(answer, payload), "; "); !strings.Contains(problems, "тип направления не подходит") {
		t.Fatalf("psychological without a low scale: %s", problems)
	}
	data.Methods[slices.IndexFunc(data.Methods, func(m MethodLevel) bool { return m.ID == "EMO" })].Level = "low"
	for n := range data.Methods {
		if data.Methods[n].ID == "EMO" {
			data.Methods[n].LevelText = levelText["low"]
		}
	}
	payload = a.testPayloadFor("v2", data)
	if problems := strings.Join(checkTestAnswerV2(answer, payload), "; "); strings.Contains(problems, "recommendations.1") {
		t.Fatalf("psychological with a low scale: %s", problems)
	}
}

// A retry is told what is allowed: the directions that fit the group and its
// situation, and that a sentence ends with a stop.
func TestProblemsTellTheRetryWhatFits(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	for n := range data.Defender {
		data.Defender[n].Type, data.Defender[n].TypeText = "support", "опора"
	}
	for n := range data.Methods {
		data.Methods[n].Level, data.Methods[n].LevelText = "high", levelText["high"]
	}
	payload := a.testPayloadFor("v2", data)
	answer := goodTestAnswerV2(string(canonicalJSON(payload)))
	answer["recommendations"].([]any)[0].(map[string]any)["group"] = 3
	answer["recommendations"].([]any)[0].(map[string]any)["code"] = "R20"
	answer["activity"] = "Студент включён в деятельность; условия он оценивает выше."
	problems := strings.Join(checkTestAnswerV2(answer, payload), "; ")
	for _, want := range []string{"recommendations.1: тип направления не подходит к ситуации группы — для группы 3 подходят: R15, R16", "activity: предложений 1 вместо 2–4 — раздели текст на отдельные предложения"} {
		if !strings.Contains(problems, want) {
			t.Errorf("no «%s» in %s", want, problems)
		}
	}
}

// The directions the input lists for a group are exactly those the check
// takes for it.
func TestFittingDirectionsAreThoseTheCheckTakes(t *testing.T) {
	a := testApp(t)
	for _, low := range []bool{false, true} {
		data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
		for n := range data.Methods {
			level := "high"
			if low && data.Methods[n].ID == "EMO" {
				level = "low"
			}
			data.Methods[n].Level, data.Methods[n].LevelText = level, levelText[level]
		}
		payload := a.testPayloadFor("v2", data)
		for _, g := range objectList(payload.values["образ_защитника"]) {
			number := g.values["номер"].(int)
			listed := map[string]bool{}
			for _, code := range g.values["подходящие_направления"].([]any) {
				listed[code.(string)] = true
			}
			if len(listed) == 0 {
				t.Errorf("group %d: nothing fits", number)
			}
			for _, d := range objectList(payload.values["каталог_рекомендаций"]) {
				answer := goodTestAnswerV2(string(canonicalJSON(payload)))
				answer["recommendations"].([]any)[0].(map[string]any)["group"] = number
				answer["recommendations"].([]any)[0].(map[string]any)["code"] = objectText(d, "код")
				passes := !strings.Contains(strings.Join(checkTestAnswerV2(answer, payload), "; "), "recommendations.1:")
				if passes != listed[objectText(d, "код")] {
					t.Errorf("low %v, group %d, %s: listed %v, passes %v", low, number, objectText(d, "код"), listed[objectText(d, "код")], passes)
				}
			}
		}
	}
}
