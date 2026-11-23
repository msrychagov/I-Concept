package usecase

import (
	"slices"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func dynamicsSession(a *Service, id, date string) model.Session {
	p := a.emptyProfile()
	p.Questions = clone(a.bank.Items)
	return model.Session{ID: id, StudentID: "one", Participant: "Студент", Kind: "student",
		Profile: p, BankVersion: a.bank.Version, Completed: true, Updated: date,
		Created: date, Answers: map[string]int{"I001": 3, "I002": 3, "I003": 3}, Group: "101", TeacherID: "owner"}
}

func TestDynamicsFiltersHistoryAndKeepsMissingReadings(t *testing.T) {
	a := testApp(t)
	current := dynamicsSession(a, "current", "2026-11-20T10:00:00Z")
	old := dynamicsSession(a, "old", "2026-10-01T10:00:00Z")
	other, future, draft, revoked, changed := clone(old), clone(old), clone(old), clone(old), clone(old)
	other.ID, other.StudentID = "other", "two"
	future.ID, future.Updated = "future", "2026-12-20T10:00:00Z"
	draft.ID, draft.Completed = "draft", false
	revoked.ID, revoked.Revoked = "revoked", true
	changed.ID, changed.Profile.Questions[0].Text = "changed", "Другая формулировка"
	all := []model.SessionRecord{{Session: current}, {Session: changed}, {Session: other}, {Session: future}, {Session: draft}, {Session: revoked}, {Session: old}}
	d := a.testDynamics(current, all)
	if len(d.Slices) != 2 || d.Slices[0].ID != "old" || d.Slices[1].ID != "current" || d.Excluded != 1 {
		t.Fatalf("wrong history: %+v", d)
	}
	for _, slice := range d.Slices {
		at := slices.IndexFunc(slice.Metrics, func(m TestMetric) bool { return m.ID == "statement:Q01" })
		if at < 0 || slice.Metrics[at].Value == nil || *slice.Metrics[at].Value != 3 {
			t.Fatalf("actual score not retained: %+v", slice)
		}
		at = slices.IndexFunc(slice.Metrics, func(m TestMetric) bool { return m.ID == "statement:Q02" })
		if at < 0 || slice.Metrics[at].Value != nil {
			t.Fatal("missing answers turned into a score")
		}
	}
	if result := a.testDynamics(draft, all); len(result.Slices) != 0 || result.Reason == "" {
		t.Fatal("unfinished session compared")
	}
}

func TestDynamicsKeepMethodRangesAndSignedClimate(t *testing.T) {
	a := testApp(t)
	s := methodSession(a, func(string) int { return 0 })
	for _, pair := range a.bank.Climate.Pairs {
		s.Answers[pair.Item] = 6 // The last option is the negative side.
	}
	got := a.testSlice(s)
	p := a.TestPortrait(s)
	for _, method := range p.Methods {
		at := slices.IndexFunc(got.Metrics, func(m TestMetric) bool { return m.ID == "method:"+method.ID })
		if method.Max <= method.Min {
			if at >= 0 {
				t.Fatal("categorical method converted to a number")
			}
			continue
		}
		if at < 0 || got.Metrics[at].Min != float64(method.Min) || got.Metrics[at].Max != float64(method.Max) {
			t.Fatalf("method range changed: %+v", method)
		}
	}
	for _, metric := range got.Metrics {
		if metric.Section == "Климат взвода" && (metric.Min != -3 || metric.Max != 3 || metric.Value == nil || *metric.Value != -3) {
			t.Fatalf("climate reading changed: %+v", metric)
		}
	}
}

func TestPlatoonDynamicsRespectOwnershipPlatoonAndLatestAttempt(t *testing.T) {
	a := testApp(t)
	first := dynamicsSession(a, "first", "2026-10-01T10:00:00Z")
	first.AssignmentID = "first-assignment"
	last := clone(first)
	last.ID, last.AssignmentID, last.Updated = "last", "last-assignment", "2026-11-20T10:00:00Z"
	older := clone(last)
	older.ID, older.Updated = "older", "2026-11-19T10:00:00Z"
	otherOwner, otherGroup, legacy := clone(last), clone(last), clone(last)
	otherOwner.ID, otherOwner.TeacherID = "other-owner", "someone-else"
	otherGroup.ID, otherGroup.Group = "other-group", "102"
	legacy.ID, legacy.StudentID = "legacy", ""
	db := model.Database{Sessions: []model.SessionRecord{{Session: older}, {Session: otherOwner}, {Session: otherGroup}, {Session: legacy}, {Session: last}, {Session: first}}}
	got := a.platoonTestDynamics(db, "owner", "101")
	if len(got) != 2 {
		t.Fatalf("wrong assignments: %+v", got)
	}
	for _, course := range got {
		if len(course.Students) != 1 {
			t.Fatal("wrong student set")
		}
		if course.ID == "last-assignment" && course.Students[0].ID != "last" {
			t.Fatal("older attempt selected")
		}
	}
	if len(a.platoonTestDynamics(db, "missing-owner", "101")) != 0 || len(a.platoonTestDynamics(db, "owner", "")) != 0 {
		t.Fatal("history leaked outside selected scope")
	}
}
