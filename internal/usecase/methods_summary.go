package usecase

import (
	"cmp"
	"slices"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// The platoon summary of the methods: for every scale of a method and every
// kind of activity, the students on each level. Only the program's levels,
// no text of a model (decided 22 October 2026). The latest finished session
// with the methods of every student counts.

type MethodsStudent struct {
	SessionID string `json:"sessionId"`
	StudentID string `json:"studentId"`
	Name      string `json:"name"`
	Group     string `json:"group"`
}

type MethodsLevelGroup struct {
	Level     string           `json:"level"`
	LevelText string           `json:"levelText"`
	Students  []MethodsStudent `json:"students"`
}

// MethodsSummaryRow is one scale ("method") or one kind of activity ("activity").
type MethodsSummaryRow struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Kind      string              `json:"kind"`
	Component string              `json:"component,omitempty"`
	Levels    []MethodsLevelGroup `json:"levels"`
}

type MethodsSummary struct {
	Group    string              `json:"group"`
	Groups   []string            `json:"groups"`
	Sessions int                 `json:"sessions"`
	Rows     []MethodsSummaryRow `json:"rows"`
	// Platoon is what the screen of the conclusion by a platoon needs; only
	// for one platoon.
	Platoon *PlatoonView `json:"platoon,omitempty"`
}

// PlatoonView is the groups of the image over a platoon with the students in
// every situation, how many students there are and how many the conclusion
// needs, and the catalogue the recommendations of the conclusion refer to.
type PlatoonView struct {
	Students  int                    `json:"students"`
	Minimum   int                    `json:"minimum"`
	Defender  []PlatoonDefenderGroup `json:"defender"`
	Catalogue []model.Direction      `json:"catalogue"`
	// Climate is the climate of the platoon by the method of Lutoshkin; nil
	// when fewer than three students answered all its pairs.
	Climate *PlatoonClimate `json:"climate,omitempty"`
}

// latestMethodSessions are the latest finished sessions with the methods of
// every student of the teacher, ordered by the name of the student.
func latestMethodSessions(db model.Database, teacherID string) []model.Session {
	latest := map[string]model.Session{}
	for _, record := range db.Sessions {
		s := record.Session
		if !s.Completed || !s.Methods || s.Kind != "student" || !teacherOwnsSession(db, teacherID, s) {
			continue
		}
		key := s.StudentID
		if key == "" {
			key = s.ID
		}
		if previous, seen := latest[key]; !seen || s.Updated > previous.Updated {
			latest[key] = s
		}
	}
	sessions := make([]model.Session, 0, len(latest))
	for _, s := range latest {
		sessions = append(sessions, s)
	}
	// The session id breaks a tie of names: the input of a platoon must not
	// depend on the order of a map.
	slices.SortStableFunc(sessions, func(x, y model.Session) int {
		return cmp.Or(strings.Compare(x.Participant, y.Participant), strings.Compare(x.ID, y.ID))
	})
	return sessions
}

// studentPortrait is the counted portrait of one student of a summary.
type studentPortrait struct {
	student MethodsStudent
	data    *TestPortraitData
}

func methodsStudent(s model.Session) MethodsStudent {
	return MethodsStudent{SessionID: s.ID, StudentID: s.StudentID, Name: s.Participant, Group: s.Group}
}

// levelGroups are the levels of a row, highest first, the undetermined last.
func levelGroups() []MethodsLevelGroup {
	levels := []MethodsLevelGroup{}
	for _, level := range []string{"high", "medium", "low", ""} {
		levels = append(levels, MethodsLevelGroup{Level: level, LevelText: levelText[level], Students: []MethodsStudent{}})
	}
	return levels
}

// levelRows puts the students on the levels of every scale of a method and
// every kind of activity. The rows are made in a fixed order from the current
// bank, so that the table has the same shape whatever sessions there are.
func (a *Service) levelRows(portraits []studentPortrait) []MethodsSummaryRow {
	result := []MethodsSummaryRow{}
	rows := map[string]int{}
	row := func(id, name, kind, component string) int {
		if at, ok := rows[kind+":"+id]; ok {
			return at
		}
		result = append(result, MethodsSummaryRow{ID: id, Name: name, Kind: kind, Component: component, Levels: levelGroups()})
		rows[kind+":"+id] = len(result) - 1
		return len(result) - 1
	}
	add := func(at int, level string, student MethodsStudent) {
		for i := range result[at].Levels {
			if result[at].Levels[i].Level == level {
				result[at].Levels[i].Students = append(result[at].Levels[i].Students, student)
			}
		}
	}
	for _, m := range a.bank.Methods {
		row(m.ID, m.Name, "method", m.Component)
	}
	for _, k := range a.bank.ActivityKinds {
		row(k.ID, k.Name, "activity", "")
	}
	for _, p := range portraits {
		for _, m := range p.data.Methods {
			add(row(m.ID, m.Name, "method", m.Component), m.Level, p.student)
		}
		for _, k := range p.data.Kinds {
			add(row(k.ID, k.Name, "activity", ""), k.Level, p.student)
		}
	}
	return result
}

// MethodsSummaryFor builds the summary over the teacher's sessions, for one
// platoon or, when group is empty, for all of them.
func (a *Service) MethodsSummaryFor(teacherID, group string) (MethodsSummary, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return MethodsSummary{}, err
	}
	summary := MethodsSummary{Group: group, Groups: []string{}}
	portraits := []studentPortrait{}
	for _, s := range latestMethodSessions(db, teacherID) {
		if !slices.Contains(summary.Groups, s.Group) {
			summary.Groups = append(summary.Groups, s.Group)
		}
		if group != "" && s.Group != group {
			continue
		}
		if data := a.TestPortrait(s); data != nil {
			portraits = append(portraits, studentPortrait{methodsStudent(s), data})
		}
	}
	summary.Sessions = len(portraits)
	summary.Rows = a.levelRows(portraits)
	if group != "" {
		platoon := a.platoonPortrait(group, portraits)
		summary.Platoon = &PlatoonView{Students: platoon.Students, Minimum: platoonMinimum, Defender: platoon.Defender, Catalogue: a.defender.Directions, Climate: platoon.Climate}
	}
	slices.Sort(summary.Groups)
	return summary, nil
}
