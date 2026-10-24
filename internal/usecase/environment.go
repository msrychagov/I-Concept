package usecase

import (
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// The criteria of the first chapter by kind of activity and the six signs of
// positive activity and environment (Вики/Тестирование/Оценка деятельности и
// среды.md, approved on 24 October 2026). The program gathers the data of the
// test that refer to them and counts no level of its own: the levels of the
// kinds stay those of the activity questionnaire.

// frequencyAnswers are the answers of the questions about the frequency of
// actions (module F, 0–4).
var frequencyAnswers = []string{"Ни разу", "Один раз", "Два–три раза", "Четыре–пять раз", "Шесть раз и более"}

// DataPoint is one piece of data of a criterion or a sign. Kind is method,
// activity (the level of a kind), indicator (a mean 1–5), question (an answer
// of the activity questionnaire), frequency, story (an open answer) or see
// (the data are in another part of the input).
type DataPoint struct {
	Code      string   `json:"code"`
	Kind      string   `json:"kind"`
	Text      string   `json:"text"`
	Answer    string   `json:"answer,omitempty"`
	Value     *int     `json:"value,omitempty"`
	Level     string   `json:"level,omitempty"`
	LevelText string   `json:"levelText,omitempty"`
	Score     *float64 `json:"score,omitempty"`
}

// CriterionData is a criterion with the data of the session.
type CriterionData struct {
	Name     string      `json:"name"`
	Coverage string      `json:"coverage"`
	Note     string      `json:"note,omitempty"`
	Texts    bool        `json:"texts,omitempty"`
	Data     []DataPoint `json:"data"`
}

type KindCriteriaData struct {
	Kind     string          `json:"kind"`
	Criteria []CriterionData `json:"criteria"`
}

// EnvironmentSignData is a sign of the environment with the data of the session.
type EnvironmentSignData struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Note        string      `json:"note,omitempty"`
	Data        []DataPoint `json:"data"`
}

// seeText says where the data of a reference are in the input.
var seeText = map[string]string{
	"images":   "Я-реальное и Я-идеальное — в «образы_я»",
	"defender": "группы образа защитника — в «образ_защитника»",
	"signs":    "выраженность качеств по утверждениям — в «признаки»",
	"stories":  "что студент написал о себе — в «рассказы_о_деятельности»",
}

// storyTold is an open answer that tells something.
func storyTold(note string) bool {
	note = strings.TrimSpace(note)
	return note != "" && !strings.HasPrefix(strings.ToLower(note), "не участвовал")
}

// dataPoints resolves the codes of a criterion or a sign for a session; what
// the session was not asked or did not answer is left out.
func dataPoints(engine *Service, s model.Session, data *TestPortraitData, score func(string) IndicatorScore, asked map[string]bool, codes []string) []DataPoint {
	methods := map[string]MethodLevel{}
	for _, m := range data.Methods {
		methods[m.ID] = m
	}
	indicators := map[string]bool{}
	for _, ind := range engine.bank.Indicators {
		indicators[ind.ID] = true
	}
	points := []DataPoint{}
	for _, code := range codes {
		if text, ok := seeText[code]; ok {
			points = append(points, DataPoint{Code: code, Kind: "see", Text: text})
			continue
		}
		if code == "kinds" {
			for _, k := range data.Kinds {
				points = append(points, DataPoint{Code: "kind:" + k.ID, Kind: "activity", Text: k.Name, Level: k.Level, LevelText: k.LevelText})
			}
			continue
		}
		if m, ok := methods[code]; ok {
			points = append(points, DataPoint{Code: code, Kind: "method", Text: m.Name, Level: m.Level, LevelText: m.LevelText})
			continue
		}
		if indicators[code] {
			if row := score(code); row.Score != nil {
				points = append(points, DataPoint{Code: code, Kind: "indicator", Text: row.Name, Score: row.Score})
			}
			continue
		}
		item, ok := engine.item(code)
		v, answered := s.Answers[code]
		if !ok || !asked[code] || !answered {
			continue
		}
		switch {
		case item.Type == "textarea":
			if storyTold(s.Notes[code]) {
				points = append(points, DataPoint{Code: code, Kind: "story", Text: item.Text, Answer: strings.TrimSpace(s.Notes[code])})
			}
		case item.Type == "frequency":
			if v >= 0 && v < len(frequencyAnswers) {
				value := v
				points = append(points, DataPoint{Code: code, Kind: "frequency", Text: item.Text, Answer: frequencyAnswers[v], Value: &value})
			}
		case v >= 0 && v < len(item.Options):
			option := item.Options[v]
			points = append(points, DataPoint{Code: code, Kind: "question", Text: item.Text, Answer: option.Text, Level: option.Level, LevelText: levelText[option.Level]})
		}
	}
	return points
}

// environmentData gathers the criteria and the signs of a session.
func environmentData(image model.DefenderImage, engine *Service, s model.Session, data *TestPortraitData, score func(string) IndicatorScore) ([]KindCriteriaData, []EnvironmentSignData) {
	if len(image.Criteria) == 0 && len(image.Signs) == 0 {
		return nil, nil
	}
	asked := map[string]bool{}
	for _, item := range engine.itemsFor(s) {
		asked[item.ID] = true
	}
	criteria := []KindCriteriaData{}
	for _, k := range image.Criteria {
		row := KindCriteriaData{Kind: k.Kind, Criteria: []CriterionData{}}
		for _, c := range k.Criteria {
			row.Criteria = append(row.Criteria, CriterionData{Name: c.Name, Coverage: c.Coverage, Note: c.Note, Texts: c.Texts, Data: dataPoints(engine, s, data, score, asked, c.Data)})
		}
		criteria = append(criteria, row)
	}
	signs := []EnvironmentSignData{}
	for _, sign := range image.Signs {
		signs = append(signs, EnvironmentSignData{ID: sign.ID, Name: sign.Name, Description: sign.Description, Note: sign.Note, Data: dataPoints(engine, s, data, score, asked, sign.Data)})
	}
	return criteria, signs
}

// noData is what the model gets for a criterion the test does not measure;
// noAnswers for one it measures that the student left without answers.
const (
	noData    = "нет данных"
	noAnswers = "нет ответов"
)

// pointList turns the data of a session into the model input.
func pointList(points []DataPoint) any {
	if len(points) == 0 {
		return noAnswers
	}
	list := []any{}
	for _, p := range points {
		o := newObject()
		switch p.Kind {
		case "see":
			o.set("см", p.Text)
		case "activity":
			o.set("вид", p.Text)
			o.set("уровень", p.LevelText)
		case "method":
			o.set("методика", p.Text)
			o.set("уровень", p.LevelText)
		case "indicator":
			o.set("показатель", p.Text)
			o.put("балл", scoreText(p.Score))
		case "question":
			o.set("вопрос_анкеты", p.Text)
			o.put("ответ", p.Answer)
			o.set("уровень", p.LevelText)
		case "frequency":
			o.set("действие", p.Text)
			o.set("за_четыре_недели", p.Answer)
		case "story":
			o.set("вопрос", p.Text)
			o.set("рассказ", p.Answer)
		}
		list = append(list, o)
	}
	return list
}

func coverageText(coverage string) string {
	if coverage == "нет" {
		return noData
	}
	return coverage
}

// criteriaList is the input of the criteria of one kind of activity.
func criteriaList(criteria []CriterionData, points func(CriterionData) any) []any {
	list := []any{}
	for _, c := range criteria {
		o := newObject()
		o.set("критерий", c.Name)
		o.set("покрытие", coverageText(c.Coverage))
		o.put("пометка", c.Note)
		if c.Coverage == "нет" {
			o.set("данные", noData)
		} else {
			o.set("данные", points(c))
		}
		list = append(list, o)
	}
	return list
}
