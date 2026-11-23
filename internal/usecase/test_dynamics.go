package usecase

import (
	"cmp"
	"slices"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// Readings keep their own units; a missing answer remains nil.
type TestMetric struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Section string   `json:"section"`
	Unit    string   `json:"unit"`
	Min     float64  `json:"min"`
	Max     float64  `json:"max"`
	Value   *float64 `json:"value"`
	Count   int      `json:"count,omitempty"`
}

type TestSlice struct {
	ID            string       `json:"id"`
	Date          string       `json:"date"`
	Title         string       `json:"title"`
	StudentID     string       `json:"studentId,omitempty"`
	Student       string       `json:"student,omitempty"`
	Compatibility string       `json:"compatibility,omitempty"`
	Metrics       []TestMetric `json:"metrics"`
}

type TestDynamics struct {
	Slices   []TestSlice `json:"slices"`
	Excluded int         `json:"excluded"`
	Reason   string      `json:"reason"`
}

type TestCohortSlice struct {
	ID       string      `json:"id"`
	Date     string      `json:"date"`
	Title    string      `json:"title"`
	Students []TestSlice `json:"students"`
}

func numberPointer(value int) *float64 {
	v := float64(value)
	return &v
}

func (a *Service) testSlice(s model.Session) TestSlice {
	engine := a.engineFor(s)
	result := TestSlice{ID: s.ID, Date: measurementDate(s), Title: s.Wave,
		StudentID: s.StudentID, Student: s.Participant,
		Compatibility: hash(a.comparisonSignature(s)), Metrics: []TestMetric{}}
	for _, ind := range engine.bank.Indicators {
		if ind.Descriptive || !engine.indicatorEnabled(s.Profile, ind.ID) {
			continue
		}
		result.Metrics = append(result.Metrics, TestMetric{ID: "statement:" + ind.ID,
			Name: ind.Name, Section: "Утверждения · " + componentNames[ind.Component],
			Unit: "баллы 1–5", Min: 1, Max: 5, Value: engine.mean(s, ind.Items)})
	}
	if p := a.TestPortrait(s); p != nil {
		for _, m := range p.Methods {
			if m.Max <= m.Min {
				continue
			}
			var value *float64
			if m.Score != nil {
				value = numberPointer(*m.Score)
			}
			result.Metrics = append(result.Metrics, TestMetric{ID: "method:" + m.ID,
				Name: m.Name, Section: "Методики · " + m.Method, Unit: "баллы",
				Min: float64(m.Min), Max: float64(m.Max), Value: value})
		}
		for _, g := range p.Defender {
			result.Metrics = append(result.Metrics,
				TestMetric{ID: "real:" + g.ID, Name: g.Name, Section: "Я-реальное", Unit: "баллы 0–10", Max: 10, Value: g.Real},
				TestMetric{ID: "ideal:" + g.ID, Name: g.Name, Section: "Я-идеальное", Unit: "баллы 0–10", Max: 10, Value: g.Ideal})
		}
		if p.Climate != nil {
			for _, pair := range p.Climate.Pairs {
				var value *float64
				if pair.Score != nil {
					value = numberPointer(*pair.Score)
				}
				result.Metrics = append(result.Metrics, TestMetric{ID: "climate:" + pair.ID,
					Name: pair.Side, Section: "Климат взвода", Unit: "баллы −3…+3", Min: -3, Max: 3, Value: value})
			}
		}
	}
	return result
}

func (a *Service) testDynamics(s model.Session, all []model.SessionRecord) TestDynamics {
	result := TestDynamics{Slices: []TestSlice{}}
	if !s.Completed || s.Kind != "student" {
		result.Reason = "Сравнение появится после завершения тестирования."
		return result
	}
	seen := map[string]bool{s.ID: true}
	for _, record := range all {
		x := record.Session
		if seen[x.ID] || x.Revoked || !x.Completed || x.Kind != "student" || !sameParticipant(s, x) || measurementDate(x) > measurementDate(s) {
			continue
		}
		seen[x.ID] = true
		if !a.comparable(s, x) {
			result.Excluded++
			continue
		}
		result.Slices = append(result.Slices, a.testSlice(x))
	}
	result.Slices = append(result.Slices, a.testSlice(s))
	slices.SortFunc(result.Slices, func(x, y TestSlice) int {
		return cmp.Or(strings.Compare(x.Date, y.Date), strings.Compare(x.ID, y.ID))
	})
	if len(result.Slices) < 2 {
		result.Reason = "Для динамики нужны хотя бы два завершённых прохождения с одинаковыми вопросами, шкалами и версией банка."
	}
	return result
}

// Per-student readings let the interface keep the same people in each slice.
// Filter by ownership and platoon before exposing history; do not match names.
func (a *Service) platoonTestDynamics(db model.Database, teacherID, group string) []TestCohortSlice {
	result := []TestCohortSlice{}
	if group == "" {
		return result
	}
	assignments := map[string]model.Assignment{}
	for _, assignment := range db.Assignments {
		assignments[assignment.ID] = assignment
	}
	latest := map[string]map[string]model.Session{}
	for _, record := range db.Sessions {
		s := record.Session
		if s.Revoked || !s.Completed || s.Kind != "student" || s.Group != group || s.StudentID == "" || s.AssignmentID == "" || !teacherOwnsSession(db, teacherID, s) {
			continue
		}
		if latest[s.AssignmentID] == nil {
			latest[s.AssignmentID] = map[string]model.Session{}
		}
		old, exists := latest[s.AssignmentID][s.StudentID]
		if !exists || measurementDate(s) > measurementDate(old) || measurementDate(s) == measurementDate(old) && s.ID > old.ID {
			latest[s.AssignmentID][s.StudentID] = s
		}
	}
	for id, students := range latest {
		course := TestCohortSlice{ID: id, Title: assignments[id].Title, Date: assignments[id].Created, Students: []TestSlice{}}
		for _, session := range students {
			course.Students = append(course.Students, a.testSlice(session))
			if course.Title == "" {
				course.Title = session.Wave
			}
			if course.Date == "" || session.Created < course.Date {
				course.Date = session.Created
			}
		}
		slices.SortFunc(course.Students, func(x, y TestSlice) int { return strings.Compare(x.StudentID, y.StudentID) })
		result = append(result, course)
	}
	slices.SortFunc(result, func(x, y TestCohortSlice) int {
		return cmp.Or(strings.Compare(x.Date, y.Date), strings.Compare(x.ID, y.ID))
	})
	return result
}
