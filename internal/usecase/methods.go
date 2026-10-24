package usecase

import (
	"math"
	"sort"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// The portrait of the first chapter: levels by the methods of the supervisor
// (Вики/Я-концепция.docx, tables 9 and 15–19) counted by their keys, the
// levels of the kinds of activity, the real and the ideal self, and the
// authored indicators as a description without levels. The rules were
// approved by the user on 22 October 2026 (Вики/Тестирование/План ИИ-отчёта по тесту.md).

// methodModules hold the questions of the methods; they are asked only when
// the assignment includes the methods. AZM and AZC are the two blocks of the
// method of professional identity (identity.go).
var methodModules = map[string]bool{"W": true, "U": true, "X": true, "K": true, "AZM": true, "AZC": true}

func methodModule(module string) bool { return methodModules[module] }

var levelRank = map[string]int{"low": 1, "medium": 2, "high": 3}
var levelByRank = []string{"", "low", "medium", "high"}
var levelText = map[string]string{"low": "низкий", "medium": "средний", "high": "высокий", "": "не определён"}

var portraitComponentNames = map[string]string{"CS": "ценностно-смысловой", "EV": "эмоционально-волевой", "DP": "деятельностно-практический"}
var signNames = map[string]string{"LOVE": "любовь", "SOB": "соборность", "PAT": "патриотизм"}
var conditionNames = map[string]string{"O": "организационный", "G": "общепедагогический", "L": "личностный"}

// MethodLevel is the level of one scale of a method. Score is the sum for a
// "sum" scale; a "level" scale has none. Level is empty when not all its
// questions are answered.
type MethodLevel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Method    string `json:"method"`
	Component string `json:"component"`
	Score     *int   `json:"score,omitempty"`
	Min       int    `json:"min"`
	Max       int    `json:"max"`
	Level     string `json:"level"`
	LevelText string `json:"levelText"`
	Answered  int    `json:"answered"`
	Total     int    `json:"total"`
}

// SignAnswer is one sign of the activity questionnaire: the chosen answer and its level.
type SignAnswer struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Answer    string `json:"answer"`
	Level     string `json:"level"`
	LevelText string `json:"levelText"`
}

type IndicatorScore struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Score *float64 `json:"score"`
}

type FrequencyAnswer struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Answer string `json:"answer"`
}

// KindLevel is a kind of activity: the level comes from the signs of the
// questionnaire, the indicators and frequencies only describe it.
type KindLevel struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Level      string            `json:"level"`
	LevelText  string            `json:"levelText"`
	Signs      []SignAnswer      `json:"signs"`
	Indicators []IndicatorScore  `json:"indicators"`
	Frequency  []FrequencyAnswer `json:"frequency"`
	Methods    []string          `json:"methods,omitempty"`
}

// ImagePair compares the real and the ideal self of one quality or component
// on the scale 0–10, by the norms of Dembo — Rubinstein (Prikhozhan) ×10.
type ImagePair struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Component string   `json:"component"`
	Real      *float64 `json:"real"`
	Ideal     *float64 `json:"ideal"`
	Mirror    *float64 `json:"mirror,omitempty"`
	Norm      *float64 `json:"norm,omitempty"`
	Gap       *float64 `json:"gap"`
	GapText   string   `json:"gapText"`
	SelfText  string   `json:"selfText"`
}

type DescriptiveGroup struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Indicators []IndicatorScore `json:"indicators"`
}

type OpenAnswer struct {
	ID       string `json:"id"`
	Topic    string `json:"topic"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// TestPortraitData is everything the program counts for the portrait; the
// model gets it ready and only writes the text.
type TestPortraitData struct {
	Methods     []MethodLevel      `json:"methods"`
	Components  []ImagePair        `json:"components"`
	Images      []ImagePair        `json:"images"`
	Signs       []DescriptiveGroup `json:"signs"`
	Conditions  []DescriptiveGroup `json:"conditions"`
	Kinds       []KindLevel        `json:"kinds"`
	OpenAnswers []OpenAnswer       `json:"openAnswers"`
	// Defender compares the student with the image of the defender by its
	// groups; Congruence is the agreement of the real and the ideal self.
	Defender   []DefenderGroup `json:"defender,omitempty"`
	Congruence *Congruence     `json:"congruence,omitempty"`
	// Identity is the professional identity by block, when the method was given.
	Identity []IdentityProfile `json:"identity,omitempty"`
}

// methodDefinitions are those of the session's bank version; a session of an
// older template that got the methods added at assignment uses the current ones.
func (a *Service) methodDefinitions(engine *Service) ([]model.Method, []model.ActivityKind) {
	if len(engine.bank.Methods) > 0 {
		return engine.bank.Methods, engine.bank.ActivityKinds
	}
	return a.bank.Methods, a.bank.ActivityKinds
}

// answeredOption returns the chosen option of a question with options.
func answeredOption(engine *Service, s model.Session, id string) (model.Item, *model.Option) {
	item, ok := engine.item(id)
	v, answered := s.Answers[id]
	if !ok || !answered || v < 0 || v >= len(item.Options) {
		return item, nil
	}
	return item, &item.Options[v]
}

func levelOf(bands []model.LevelBand, score int) string {
	for _, band := range bands {
		if score >= band.Min && score <= band.Max {
			return band.Level
		}
	}
	return ""
}

// meanRank is the level of the mean rank of answered options, rounded to the
// nearest level (low 1, medium 2, high 3); empty unless all are answered.
func meanRank(engine *Service, s model.Session, ids []string) (string, []SignAnswer) {
	signs := []SignAnswer{}
	total, complete := 0, len(ids) > 0
	for _, id := range ids {
		item, option := answeredOption(engine, s, id)
		if option == nil || levelRank[option.Level] == 0 {
			complete = false
			if item.ID != "" {
				signs = append(signs, SignAnswer{ID: id, Text: item.Text, LevelText: levelText[""]})
			}
			continue
		}
		total += levelRank[option.Level]
		signs = append(signs, SignAnswer{ID: id, Text: item.Text, Answer: option.Text, Level: option.Level, LevelText: levelText[option.Level]})
	}
	if !complete {
		return "", signs
	}
	return levelByRank[int(math.Round(float64(total)/float64(len(ids))))], signs
}

func (a *Service) methodLevels(engine *Service, s model.Session, methods []model.Method) []MethodLevel {
	result := []MethodLevel{}
	for _, m := range methods {
		row := MethodLevel{ID: m.ID, Name: m.Name, Method: m.Method, Component: m.Component, Total: len(m.Items)}
		switch m.Scoring {
		case "sum":
			sum, complete := 0, true
			for _, id := range m.Items {
				_, option := answeredOption(engine, s, id)
				if option == nil {
					complete = false
					continue
				}
				row.Answered++
				sum += int(option.Score)
			}
			if len(m.Levels) > 0 {
				row.Min, row.Max = m.Levels[0].Min, m.Levels[len(m.Levels)-1].Max
				for _, band := range m.Levels {
					row.Min, row.Max = min(row.Min, band.Min), max(row.Max, band.Max)
				}
			}
			if complete {
				row.Score = &sum
				row.Level = levelOf(m.Levels, sum)
			}
		case "level":
			level, signs := meanRank(engine, s, m.Items)
			for _, sign := range signs {
				if sign.Level != "" {
					row.Answered++
				}
			}
			row.Level = level
		}
		row.LevelText = levelText[row.Level]
		result = append(result, row)
	}
	return result
}

// selfText is the level of self-esteem (the real self, 0–10): below 4.5 low,
// below 7.5 medium, otherwise high.
func selfText(real float64) string {
	switch {
	case real < 4.5:
		return "низкая самооценка"
	case real < 7.5:
		return "средняя самооценка"
	}
	return "высокая самооценка"
}

// gapText names the distance from the real to the ideal self, rounded: 0 —
// the images coincide, 1–2 — optimal, 3 and more — marked, below 0 — the
// ideal is below the real.
func gapText(gap float64) string {
	switch g := math.Round(gap); {
	case g < 0:
		return "Я-идеальное ниже Я-реального"
	case g == 0:
		return "образы совпадают"
	case g <= 2:
		return "оптимальное расхождение"
	}
	return "выраженное расхождение"
}

func imagePair(id, name, component string, values map[string][]float64) ImagePair {
	pair := ImagePair{ID: id, Name: name, Component: component}
	mean := func(xs []float64) *float64 {
		if len(xs) == 0 {
			return nil
		}
		total := 0.0
		for _, x := range xs {
			total += x
		}
		v := math.Round(total/float64(len(xs))*10) / 10
		return &v
	}
	pair.Real, pair.Ideal, pair.Mirror, pair.Norm = mean(values["R"]), mean(values["I"]), mean(values["Z"]), mean(values["N"])
	if pair.Real != nil && pair.Ideal != nil {
		gap := math.Round((*pair.Ideal-*pair.Real)*10) / 10
		pair.Gap = &gap
		pair.GapText = gapText(gap)
		pair.SelfText = selfText(*pair.Real)
	}
	return pair
}

// TestPortrait counts the portrait of a session; nil when the session has no methods.
func (a *Service) TestPortrait(s model.Session) *TestPortraitData {
	engine := a.engineFor(s)
	methods, kinds := a.methodDefinitions(engine)
	if !s.Methods || len(methods) == 0 {
		return nil
	}
	data := &TestPortraitData{Methods: a.methodLevels(engine, s, methods), Components: []ImagePair{}, Images: []ImagePair{}, Signs: []DescriptiveGroup{}, Conditions: []DescriptiveGroup{}, Kinds: []KindLevel{}, OpenAnswers: []OpenAnswer{}}

	indicators := map[string]model.Indicator{}
	scores := map[string]*float64{}
	for _, ind := range engine.bank.Indicators {
		indicators[ind.ID] = ind
		if !ind.Descriptive && len(ind.Items) > 0 {
			if v := engine.mean(s, ind.Items); v != nil {
				rounded := math.Round(*v*100) / 100
				scores[ind.ID] = &rounded
			}
		}
	}
	indicatorScore := func(id string) IndicatorScore {
		return IndicatorScore{ID: id, Name: indicators[id].Name, Score: scores[id]}
	}

	// The real and the ideal self (module A) by quality and by component.
	byQuality := map[string]map[string][]float64{}
	byComponent := map[string]map[string][]float64{}
	for _, item := range engine.itemsFor(s) {
		v, ok := s.Answers[item.ID]
		if item.Module != "A" || !ok {
			continue
		}
		kind := item.ID[strings.LastIndex(item.ID, "_")+1:]
		if byQuality[item.Indicator] == nil {
			byQuality[item.Indicator] = map[string][]float64{}
		}
		byQuality[item.Indicator][kind] = append(byQuality[item.Indicator][kind], float64(v))
		component := indicators[item.Indicator].Component
		if byComponent[component] == nil {
			byComponent[component] = map[string][]float64{}
		}
		byComponent[component][kind] = append(byComponent[component][kind], float64(v))
	}
	for _, ind := range engine.bank.Indicators {
		if values, ok := byQuality[ind.ID]; ok {
			data.Images = append(data.Images, imagePair(ind.ID, ind.Name, ind.Component, values))
		}
	}
	for _, component := range []string{"CS", "EV", "DP"} {
		if values, ok := byComponent[component]; ok {
			data.Components = append(data.Components, imagePair(component, portraitComponentNames[component], component, values))
		}
	}

	// Signs and conditions are described by their indicators, strongest first.
	group := func(id, name string, match func(model.Indicator) bool) DescriptiveGroup {
		g := DescriptiveGroup{ID: id, Name: name, Indicators: []IndicatorScore{}}
		for _, ind := range engine.bank.Indicators {
			if match(ind) && scores[ind.ID] != nil {
				g.Indicators = append(g.Indicators, indicatorScore(ind.ID))
			}
		}
		sort.SliceStable(g.Indicators, func(i, j int) bool { return *g.Indicators[i].Score > *g.Indicators[j].Score })
		return g
	}
	for _, tag := range []string{"LOVE", "SOB", "PAT"} {
		data.Signs = append(data.Signs, group(tag, signNames[tag], func(ind model.Indicator) bool { return contains(ind.Tags, tag) }))
	}
	for _, level := range []string{"O", "G", "L"} {
		data.Conditions = append(data.Conditions, group(level, conditionNames[level], func(ind model.Indicator) bool {
			return ind.Block == "periphery" && ind.Component == level
		}))
	}

	// Kinds of activity.
	frequency := []string{"Ни разу", "Один раз", "Два–три раза", "Четыре–пять раз", "Шесть раз и более"}
	for _, k := range kinds {
		level, signs := meanRank(engine, s, k.Items)
		row := KindLevel{ID: k.ID, Name: k.Name, Level: level, LevelText: levelText[level], Signs: signs, Indicators: []IndicatorScore{}, Frequency: []FrequencyAnswer{}, Methods: k.Methods}
		for _, id := range k.Indicators {
			if scores[id] != nil {
				row.Indicators = append(row.Indicators, indicatorScore(id))
			}
		}
		for _, id := range k.Frequency {
			item, ok := engine.item(id)
			v, answered := s.Answers[id]
			if ok && answered && v >= 0 && v < len(frequency) {
				row.Frequency = append(row.Frequency, FrequencyAnswer{ID: id, Text: item.Text, Answer: frequency[v]})
			}
		}
		data.Kinds = append(data.Kinds, row)
	}

	// What the student wrote about the activity: answers that tell something.
	for _, item := range engine.itemsFor(s) {
		note := strings.TrimSpace(s.Notes[item.ID])
		if item.Type != "textarea" || note == "" || strings.HasPrefix(strings.ToLower(note), "не участвовал") {
			continue
		}
		if _, answered := s.Answers[item.ID]; answered {
			data.OpenAnswers = append(data.OpenAnswers, OpenAnswer{ID: item.ID, Topic: item.Topic, Question: item.Text, Answer: note})
		}
	}

	data.Defender = defenderGroups(a.defender, data, indicatorScore)
	data.Congruence = congruence(data.Images)
	data.Identity = a.identityProfiles(engine, s)
	return data
}
