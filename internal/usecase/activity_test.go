package usecase

import (
	"bytes"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestActivityTextAnswers(t *testing.T) {
	a := testApp(t)
	student := signUp(t, a, "student", "activity@example.test", "101")
	p := a.emptyProfile()
	p.ID, p.Name, p.Author, p.OwnerID, p.BankVersion = "activity-template", "Activity", "Teacher", student.Account.OwnerID, "pilot-2.2"
	p.Questions = clone(a.previous["pilot-2.2"].Items)
	if err := a.addProfile(p); err != nil {
		t.Fatal(err)
	}
	input := Input{"profileId": raw(t, p.ID), "title": raw(t, "Опыт"), "activity": raw(t, true)}
	if _, err := a.AssignmentCommandFor(p.OwnerID, input, "assignment"); err != nil {
		t.Fatal(err)
	}
	db := mustSnapshot(t, a.repo)
	topics := map[string]int{}
	for _, q := range db.Assignments[0].Profile.Questions {
		if q.Type == "textarea" {
			topics[q.Topic]++
			if q.Hint == "" || !q.ResponseOnly {
				t.Fatal("missing text question metadata")
			}
		}
	}
	if topics["Научная деятельность"] != 10 || topics["Учебно-методическая деятельность"] != 12 || topics["Воспитательная деятельность"] != 16 {
		t.Fatalf("coverage: %v", topics)
	}
	if len(db.Profiles[0].Questions) != len(p.Questions) {
		t.Fatal("legacy template changed")
	}
	if _, err := a.StartAssignment(student.Account, db.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	s := mustSnapshot(t, a.repo).Sessions[0].Session
	if _, err := a.SurveyCommand(student.Token, s.ID, Input{"value": raw(t, true), "revision": raw(t, 0)}, "consent"); err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("я", 4000)
	answer := Input{"item": raw(t, "ACT_01"), "value": raw(t, 0), "note": raw(t, text), "revision": raw(t, 1)}
	if _, err := a.SurveyCommand(student.Token, s.ID, answer, "answer"); err != nil {
		t.Fatal(err)
	}
	s = mustSnapshot(t, a.repo).Sessions[0].Session
	if s.Notes["ACT_01"] != text || s.Revision != 2 {
		t.Fatal("text was not persisted")
	}
	if value, ok := s.Answers["ACT_01"]; !ok || value != 0 {
		t.Fatal("text completion marker missing")
	}
	answer["revision"], answer["note"] = raw(t, 2), raw(t, "   ")
	if _, err := a.SurveyCommand(student.Token, s.ID, answer, "answer"); kind(err) != Invalid {
		t.Fatalf("blank answer accepted: %v", err)
	}
	answer["note"] = raw(t, text+"я")
	if _, err := a.SurveyCommand(student.Token, s.ID, answer, "answer"); kind(err) != Invalid {
		t.Fatalf("oversize answer accepted: %v", err)
	}
	report := a.analyze(s, nil)
	found := false
	for _, row := range report["activity"].([]map[string]any) {
		if row["id"] == "ACT_01" {
			found = row["value"] == text
		}
	}
	if !found {
		t.Fatal("report lost text")
	}
	var csv bytes.Buffer
	if err := a.ExportCSVFor(p.OwnerID, &csv, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv.String(), text) {
		t.Fatal("CSV lost text")
	}
	answer["value"] = raw(t, nil)
	answer["note"] = raw(t, "old text")
	if _, err := a.SurveyCommand(student.Token, s.ID, answer, "answer"); err != nil {
		t.Fatal(err)
	}
	s = mustSnapshot(t, a.repo).Sessions[0].Session
	if _, ok := s.Answers["ACT_01"]; ok || s.Notes["ACT_01"] != "" {
		t.Fatal("delete left text or marker")
	}
	for _, q := range a.itemsFor(model.Session{Kind: "student", Activity: true, BankVersion: "pilot-2.2"}) {
		if q.Type == "textarea" {
			t.Fatal("old session changed")
		}
	}
	input["activity"] = raw(t, false)
	if _, err := a.AssignmentCommandFor(p.OwnerID, input, "assignment"); err != nil {
		t.Fatal(err)
	}
	for _, q := range mustSnapshot(t, a.repo).Assignments[1].Profile.Questions {
		if q.Module == "F" {
			t.Fatal("disabled activity included")
		}
	}
}
