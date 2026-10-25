package usecase

import (
	"cmp"
	"math"
	"slices"
	"vucdiagnostic/internal/domain/model"
)

// The count of a platoon for the conclusion by the platoon (task test-platoon,
// Вики/Тестирование/План портрета test-v2.md, step 7 and decision Д6): the
// latest finished session with the methods of every student, as in the
// summary of the methods. The program shows the lists of students itself; the
// model will get the numbers only. There is no instruction yet, so nothing
// here calls the model.

// PlatoonCount is one value of a distribution and the students who have it.
type PlatoonCount struct {
	ID       string           `json:"id"`
	Text     string           `json:"text"`
	Students []MethodsStudent `json:"students"`
}

// PlatoonDefenderGroup is a group of the image over the platoon: the means of
// the real and the ideal self over the students who have them, the students
// in every situation, and what forms the group (codes of kinds of activity and
// conditions of the environment).
type PlatoonDefenderGroup struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Real       *float64       `json:"real"`
	Ideal      *float64       `json:"ideal"`
	Situations []PlatoonCount `json:"situations"`
	Kinds      []string       `json:"kinds"`
	Conditions []string       `json:"conditions"`
}

// PlatoonCondition is a condition of the environment D01–D12 over the
// platoon: the mean of the students' scores (1–5), how many have a score, and
// the students who rated it 1–2 (their mean is 2 or below).
type PlatoonCondition struct {
	ID    string           `json:"id"`
	Name  string           `json:"name"`
	Level string           `json:"level"`
	Mean  *float64         `json:"mean"`
	Rated int              `json:"rated"`
	Low   []MethodsStudent `json:"low"`
}

// PlatoonIdentityBlock is a block of the professional identity over the
// platoon: the students by leading status. A student whose statuses share the
// highest range is in the list of each of them; "" holds those whose block is
// not complete.
type PlatoonIdentityBlock struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Leading []PlatoonCount `json:"leading"`
}

// PlatoonPortraitData is what the program counts about a platoon.
type PlatoonPortraitData struct {
	Group      string                 `json:"group"`
	Students   int                    `json:"students"`
	Defender   []PlatoonDefenderGroup `json:"defender"`
	Congruence []PlatoonCount         `json:"congruence"`
	Conditions []PlatoonCondition     `json:"conditions"`
	Methods    []MethodsSummaryRow    `json:"methods"`
	Kinds      []MethodsSummaryRow    `json:"kinds"`
	Identity   []PlatoonIdentityBlock `json:"identity"`
	// Relations are the signs love, sobornost and patriotism: the means of
	// their indicators over the platoon.
	Relations []PlatoonRelation `json:"relations"`
	// Criteria and EnvironmentSigns are the data of the criteria of the first
	// chapter and of the signs of the environment over the platoon.
	Criteria         []PlatoonKindCriteria `json:"criteria"`
	EnvironmentSigns []PlatoonSign         `json:"environmentSigns"`
	// Told is how many students wrote about their activity at all.
	Told int `json:"told"`
	// Climate is the climate of the platoon by the students who answered all
	// the pairs of the method (climate.go); nil with fewer than three of them.
	Climate *PlatoonClimate `json:"climate,omitempty"`
}

// PlatoonRelation is a sign of the core over the platoon.
type PlatoonRelation struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Indicators []IndicatorScore `json:"indicators"`
}

// PlatoonPoint is a datum of a criterion or a sign over the platoon: the
// students on each level (by level code, "" when not determined), the students
// by answer about the frequency, the mean of an indicator with the number of
// those who have it and, for a condition, of those who rated it 1–2, the
// number of those who wrote an open answer and, when the criterion asks, the
// answers themselves.
type PlatoonPoint struct {
	Code    string         `json:"code"`
	Kind    string         `json:"kind"`
	Text    string         `json:"text"`
	Levels  map[string]int `json:"levels,omitempty"`
	Answers []int          `json:"answers,omitempty"`
	Mean    *float64       `json:"mean,omitempty"`
	Rated   int            `json:"rated,omitempty"`
	// Condition marks a condition of the environment: Low counts its 1–2.
	Condition bool     `json:"condition,omitempty"`
	Low       int      `json:"low,omitempty"`
	Told      int      `json:"told,omitempty"`
	Texts     []string `json:"texts,omitempty"`
}

type PlatoonCriterion struct {
	Name     string         `json:"name"`
	Coverage string         `json:"coverage"`
	Note     string         `json:"note,omitempty"`
	Points   []PlatoonPoint `json:"points"`
}

type PlatoonKindCriteria struct {
	Kind     string             `json:"kind"`
	Criteria []PlatoonCriterion `json:"criteria"`
}

type PlatoonSign struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Note        string         `json:"note,omitempty"`
	Points      []PlatoonPoint `json:"points"`
}

// lowCondition is the highest mean of a condition rated 1–2.
const lowCondition = 2.0

// PlatoonPortraitFor counts a platoon of the teacher; an empty group counts
// all the teacher's platoons.
func (a *Service) PlatoonPortraitFor(teacherID, group string) (PlatoonPortraitData, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return PlatoonPortraitData{}, err
	}
	return a.platoonPortrait(group, a.platoonStudents(db, teacherID, group)), nil
}

// platoonStudents are the counted portraits of the latest sessions with the
// methods of the students of a platoon of the teacher.
func (a *Service) platoonStudents(db model.Database, teacherID, group string) []studentPortrait {
	portraits := []studentPortrait{}
	for _, s := range latestMethodSessions(db, teacherID) {
		if group != "" && s.Group != group {
			continue
		}
		if data := a.TestPortrait(s); data != nil {
			portraits = append(portraits, studentPortrait{methodsStudent(s), data})
		}
	}
	return portraits
}

// aggregatePoints puts the data of one criterion or sign of every student
// together by code, in the order they first come.
func (a *Service) aggregatePoints(perStudent [][]DataPoint, texts bool, told int) []PlatoonPoint {
	conditions := map[string]bool{}
	for _, ind := range a.bank.Indicators {
		conditions[ind.ID] = ind.Block == "periphery"
	}
	points, at := []PlatoonPoint{}, map[string]int{}
	scores := map[string][]float64{}
	for _, list := range perStudent {
		for _, p := range list {
			n, seen := at[p.Code]
			if !seen {
				n = len(points)
				at[p.Code] = n
				points = append(points, PlatoonPoint{Code: p.Code, Kind: p.Kind, Text: p.Text, Condition: conditions[p.Code]})
				if p.Code == "stories" {
					points[n].Told = told
				}
			}
			row := &points[n]
			switch p.Kind {
			case "activity", "method", "question":
				if row.Levels == nil {
					row.Levels = map[string]int{}
				}
				row.Levels[p.Level]++
			case "frequency":
				if row.Answers == nil {
					row.Answers = make([]int, len(frequencyAnswers))
				}
				row.Answers[*p.Value]++
			case "indicator":
				scores[p.Code] = append(scores[p.Code], *p.Score)
				if conditions[p.Code] && *p.Score <= lowCondition {
					row.Low++
				}
			case "story":
				row.Told++
				if texts {
					row.Texts = append(row.Texts, p.Answer)
				}
			}
		}
	}
	for n := range points {
		if xs := scores[points[n].Code]; len(xs) > 0 {
			points[n].Rated, points[n].Mean = len(xs), meanPointer(xs, 100)
		}
	}
	return points
}

func counts(ids []string, text map[string]string) []PlatoonCount {
	list := []PlatoonCount{}
	for _, id := range ids {
		list = append(list, PlatoonCount{ID: id, Text: text[id], Students: []MethodsStudent{}})
	}
	return list
}

func addTo(list []PlatoonCount, id string, student MethodsStudent) {
	for n := range list {
		if list[n].ID == id {
			list[n].Students = append(list[n].Students, student)
		}
	}
}

func meanPointer(xs []float64, digits float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	v := math.Round(meanOf(xs)*digits) / digits
	return &v
}

func (a *Service) platoonPortrait(group string, portraits []studentPortrait) PlatoonPortraitData {
	data := PlatoonPortraitData{Group: group, Students: len(portraits), Defender: []PlatoonDefenderGroup{}, Conditions: []PlatoonCondition{}, Identity: []PlatoonIdentityBlock{},
		Relations: []PlatoonRelation{}, Criteria: []PlatoonKindCriteria{}, EnvironmentSigns: []PlatoonSign{}}
	rows := a.levelRows(portraits)
	data.Methods = slices.DeleteFunc(slices.Clone(rows), func(r MethodsSummaryRow) bool { return r.Kind != "method" })
	data.Kinds = slices.DeleteFunc(rows, func(r MethodsSummaryRow) bool { return r.Kind != "activity" })

	situations := []string{situationAchieved, situationFar, situationNotAccepted, situationBelow, ""}
	for _, def := range a.defender.Groups {
		g := PlatoonDefenderGroup{ID: def.ID, Name: def.Name, Situations: counts(situations, situationText), Kinds: []string{}, Conditions: slices.Clone(def.Conditions)}
		for _, link := range def.Kinds {
			g.Kinds = append(g.Kinds, link.ID)
		}
		real, ideal := []float64{}, []float64{}
		for _, p := range portraits {
			for _, student := range p.data.Defender {
				if student.ID != def.ID {
					continue
				}
				if student.Real != nil {
					real, ideal = append(real, *student.Real), append(ideal, *student.Ideal)
				}
				addTo(g.Situations, student.Situation, p.student)
			}
		}
		g.Real, g.Ideal = meanPointer(real, 10), meanPointer(ideal, 10)
		data.Defender = append(data.Defender, g)
	}

	data.Congruence = counts([]string{"high", "partial", "low", "none"}, congruenceTexts)
	for _, p := range portraits {
		if p.data.Congruence != nil {
			addTo(data.Congruence, p.data.Congruence.Code, p.student)
		}
	}

	for _, ind := range a.bank.Indicators {
		if ind.Block != "periphery" || ind.Descriptive {
			continue
		}
		c := PlatoonCondition{ID: ind.ID, Name: ind.Name, Level: conditionNames[ind.Component], Low: []MethodsStudent{}}
		scores := []float64{}
		for _, p := range portraits {
			for _, level := range p.data.Conditions {
				for _, score := range level.Indicators {
					if score.ID == ind.ID && score.Score != nil {
						scores = append(scores, *score.Score)
						if *score.Score <= lowCondition {
							c.Low = append(c.Low, p.student)
						}
					}
				}
			}
		}
		c.Rated, c.Mean = len(scores), meanPointer(scores, 100)
		data.Conditions = append(data.Conditions, c)
	}

	for _, p := range portraits {
		if len(p.data.OpenAnswers) > 0 {
			data.Told++
		}
	}
	for _, k := range a.defender.Criteria {
		row := PlatoonKindCriteria{Kind: k.Kind, Criteria: []PlatoonCriterion{}}
		for n, c := range k.Criteria {
			perStudent := [][]DataPoint{}
			for _, p := range portraits {
				for _, own := range p.data.Criteria {
					if own.Kind == k.Kind && n < len(own.Criteria) {
						perStudent = append(perStudent, own.Criteria[n].Data)
					}
				}
			}
			row.Criteria = append(row.Criteria, PlatoonCriterion{Name: c.Name, Coverage: c.Coverage, Note: c.Note, Points: a.aggregatePoints(perStudent, c.Texts, data.Told)})
		}
		data.Criteria = append(data.Criteria, row)
	}
	for n, sign := range a.defender.Signs {
		perStudent := [][]DataPoint{}
		for _, p := range portraits {
			if n < len(p.data.EnvironmentSigns) {
				perStudent = append(perStudent, p.data.EnvironmentSigns[n].Data)
			}
		}
		data.EnvironmentSigns = append(data.EnvironmentSigns, PlatoonSign{ID: sign.ID, Name: sign.Name, Description: sign.Description, Note: sign.Note, Points: a.aggregatePoints(perStudent, false, data.Told)})
	}
	for _, tag := range []string{"LOVE", "SOB", "PAT"} {
		relation := PlatoonRelation{ID: tag, Name: signNames[tag], Indicators: []IndicatorScore{}}
		names, scores, order := map[string]string{}, map[string][]float64{}, []string{}
		for _, p := range portraits {
			for _, g := range p.data.Signs {
				if g.ID != tag {
					continue
				}
				for _, ind := range g.Indicators {
					if _, seen := names[ind.ID]; !seen {
						order = append(order, ind.ID)
						names[ind.ID] = ind.Name
					}
					scores[ind.ID] = append(scores[ind.ID], *ind.Score)
				}
			}
		}
		for _, id := range order {
			relation.Indicators = append(relation.Indicators, IndicatorScore{ID: id, Name: names[id], Score: meanPointer(scores[id], 100)})
		}
		slices.SortStableFunc(relation.Indicators, func(x, y IndicatorScore) int {
			return cmp.Compare(*y.Score, *x.Score)
		})
		data.Relations = append(data.Relations, relation)
	}

	data.Climate = platoonClimate(portraits)
	if method := a.bank.Identity; method != nil {
		names, ids := map[string]string{"": "не определён"}, []string{}
		for _, scale := range method.Scales {
			names[scale.ID] = scale.Name
			ids = append(ids, scale.ID)
		}
		ids = append(ids, "")
		for _, block := range method.Blocks {
			b := PlatoonIdentityBlock{ID: block.ID, Name: block.Name, Leading: counts(ids, names)}
			for _, p := range portraits {
				for _, profile := range p.data.Identity {
					if profile.ID != block.ID {
						continue
					}
					if len(profile.Leading) == 0 {
						addTo(b.Leading, "", p.student)
					}
					for _, id := range profile.Leading {
						addTo(b.Leading, id, p.student)
					}
				}
			}
			data.Identity = append(data.Identity, b)
		}
	}
	return data
}
