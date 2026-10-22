package usecase

import (
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
}

// MethodsSummaryFor builds the summary over the teacher's sessions, for one
// platoon or, when group is empty, for all of them.
func (a *Service) MethodsSummaryFor(teacherID, group string) (MethodsSummary, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return MethodsSummary{}, err
	}
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
	summary := MethodsSummary{Group: group, Groups: []string{}, Rows: []MethodsSummaryRow{}}
	rows := map[string]int{}
	row := func(id, name, kind, component string) int {
		if at, ok := rows[kind+":"+id]; ok {
			return at
		}
		levels := []MethodsLevelGroup{}
		for _, level := range []string{"high", "medium", "low", ""} {
			levels = append(levels, MethodsLevelGroup{Level: level, LevelText: levelText[level], Students: []MethodsStudent{}})
		}
		summary.Rows = append(summary.Rows, MethodsSummaryRow{ID: id, Name: name, Kind: kind, Component: component, Levels: levels})
		rows[kind+":"+id] = len(summary.Rows) - 1
		return len(summary.Rows) - 1
	}
	add := func(at int, level string, student MethodsStudent) {
		for i := range summary.Rows[at].Levels {
			if summary.Rows[at].Levels[i].Level == level {
				summary.Rows[at].Levels[i].Students = append(summary.Rows[at].Levels[i].Students, student)
			}
		}
	}
	sessions := make([]model.Session, 0, len(latest))
	for _, s := range latest {
		sessions = append(sessions, s)
	}
	slices.SortStableFunc(sessions, func(x, y model.Session) int { return strings.Compare(x.Participant, y.Participant) })
	// The rows are made in a fixed order from the current bank, so that the
	// table has the same shape whatever sessions there are.
	for _, m := range a.bank.Methods {
		row(m.ID, m.Name, "method", m.Component)
	}
	for _, k := range a.bank.ActivityKinds {
		row(k.ID, k.Name, "activity", "")
	}
	for _, s := range sessions {
		if !slices.Contains(summary.Groups, s.Group) {
			summary.Groups = append(summary.Groups, s.Group)
		}
		if group != "" && s.Group != group {
			continue
		}
		portrait := a.TestPortrait(s)
		if portrait == nil {
			continue
		}
		summary.Sessions++
		student := MethodsStudent{SessionID: s.ID, StudentID: s.StudentID, Name: s.Participant, Group: s.Group}
		for _, m := range portrait.Methods {
			add(row(m.ID, m.Name, "method", m.Component), m.Level, student)
		}
		for _, k := range portrait.Kinds {
			add(row(k.ID, k.Name, "activity", ""), k.Level, student)
		}
	}
	slices.Sort(summary.Groups)
	return summary, nil
}
