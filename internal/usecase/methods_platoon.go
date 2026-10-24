package usecase

import (
	"math"
	"slices"
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
	portraits := []studentPortrait{}
	for _, s := range latestMethodSessions(db, teacherID) {
		if group != "" && s.Group != group {
			continue
		}
		if data := a.TestPortrait(s); data != nil {
			portraits = append(portraits, studentPortrait{methodsStudent(s), data})
		}
	}
	return a.platoonPortrait(group, portraits), nil
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
	data := PlatoonPortraitData{Group: group, Students: len(portraits), Defender: []PlatoonDefenderGroup{}, Conditions: []PlatoonCondition{}, Identity: []PlatoonIdentityBlock{}}
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
		c := PlatoonCondition{ID: ind.ID, Name: ind.Name, Low: []MethodsStudent{}}
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
