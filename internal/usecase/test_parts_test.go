package usecase

import (
	"slices"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// The test is assigned by one button, without a template or switches, and
// split into three parts of at most 150 questions: 139, 144 and 139
// (Вики/Тестирование/План разбивки теста на три части.md).
func TestAssignedTestHasThreeParts(t *testing.T) {
	a := testApp(t)
	student := signUp(t, a, "student", "parts@example.test", "101")
	if _, err := a.AssignmentCommandFor(student.Account.OwnerID, Input{"title": raw(t, "Диагностическое тестирование")}, "assignment"); err != nil {
		t.Fatal(err)
	}
	as := mustSnapshot(t, a.repo).Assignments[0]
	if !as.Methods || !as.Activity || !as.Motivation || as.Profile.BankVersion != "pilot-2.7" {
		t.Fatalf("assignment: methods %v, activity %v, motivation %v, bank %s", as.Methods, as.Activity, as.Motivation, as.Profile.BankVersion)
	}
	s := model.Session{Kind: "student", Profile: as.Profile, BankVersion: as.Profile.BankVersion, Activity: true, Motivation: true, Methods: true, Answers: map[string]int{}}
	got := map[string]int{}
	for _, part := range a.partStates(s) {
		got[part.ID] = part.Total
	}
	if want := map[string]int{"self": 139, "will": 144, "vuc": 139}; len(got) != 3 || got["self"] != want["self"] || got["will"] != want["will"] || got["vuc"] != want["vuc"] {
		t.Fatalf("questions by part %v, want 139, 144 and 139", got)
	}
	if total := len(a.itemsFor(s)); total != 422 {
		t.Fatalf("%d questions, want 422", total)
	}
}

// firstAnswer is the answer to the item with its first value.
func firstAnswer(t *testing.T, item model.Item, revision int) Input {
	t.Helper()
	in := Input{"item": raw(t, item.ID), "value": raw(t, item.Min), "revision": raw(t, revision)}
	if len(item.Options) > 0 {
		in["value"] = raw(t, 0)
	}
	if item.Type == "textarea" {
		in["value"], in["note"] = raw(t, 0), raw(t, "Не участвовал.")
	}
	return in
}

// answerPart answers every question of the part with its first value.
func answerPart(t *testing.T, a *Service, token, id string, part model.BankPart) {
	t.Helper()
	s, _ := a.sessionByID(id)
	for _, item := range a.partItems(s, part) {
		s, _ = a.sessionByID(id)
		in := firstAnswer(t, item, s.Revision)
		if _, err := a.SurveyCommand(token, id, in, "answer"); err != nil {
			t.Fatalf("%s: %v", item.ID, err)
		}
	}
}

// Parts are submitted in any order; a submitted part is locked; the session
// is finished with the last part, and only then.
func TestPartsAreSubmittedInAnyOrder(t *testing.T) {
	a := testApp(t)
	student := signUp(t, a, "student", "order@example.test", "101")
	if _, err := a.AssignmentCommandFor(student.Account.OwnerID, Input{"title": raw(t, "Тест")}, "assignment"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartAssignment(student.Account, mustSnapshot(t, a.repo).Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	id := mustSnapshot(t, a.repo).Sessions[0].Session.ID
	if _, err := a.SurveyCommand(student.Token, id, Input{"value": raw(t, true), "revision": raw(t, 0)}, "consent"); err != nil {
		t.Fatal(err)
	}
	s, _ := a.sessionByID(id)
	parts := map[string]model.BankPart{}
	for _, part := range a.testParts(s) {
		parts[part.ID] = part
	}
	submit := func(part string, partial bool) error {
		s, _ := a.sessionByID(id)
		_, err := a.SurveyCommand(student.Token, id, Input{"part": raw(t, part), "allowPartial": raw(t, partial), "revision": raw(t, s.Revision)}, "complete")
		return err
	}
	if err := submit("vuc", false); kind(err) != Incomplete {
		t.Fatalf("an unanswered part was submitted: %v", err)
	}
	answerPart(t, a, student.Token, id, parts["vuc"])
	if err := submit("vuc", false); err != nil {
		t.Fatal(err)
	}
	if err := submit("vuc", false); kind(err) != Conflict {
		t.Fatalf("a part was submitted twice: %v", err)
	}
	if err := submit("other", false); kind(err) != Invalid {
		t.Fatalf("an unknown part was submitted: %v", err)
	}
	s, _ = a.sessionByID(id)
	locked := a.partItems(s, parts["vuc"])[0]
	if _, err := a.SurveyCommand(student.Token, id, firstAnswer(t, locked, s.Revision), "answer"); kind(err) != Conflict {
		t.Fatalf("an answer of a submitted part was changed: %v", err)
	}
	answerPart(t, a, student.Token, id, parts["self"])
	if err := submit("self", false); err != nil {
		t.Fatal(err)
	}
	s, _ = a.sessionByID(id)
	if s.Completed || !slices.Equal(s.Parts, []string{"vuc", "self"}) || !a.partsStarted(s) {
		t.Fatalf("after two parts: completed %v, parts %v", s.Completed, s.Parts)
	}
	if err := submit("will", true); err != nil {
		t.Fatal(err)
	}
	s, _ = a.sessionByID(id)
	if !s.Completed || !s.CompletedPartial || len(s.Parts) != 3 || a.partsStarted(s) {
		t.Fatalf("after three parts: completed %v, partial %v, parts %v", s.Completed, s.CompletedPartial, s.Parts)
	}
}

// A session of an older bank has no parts and is finished whole, as before.
func TestOldSessionHasNoParts(t *testing.T) {
	a := testApp(t)
	s := model.Session{Kind: "student", BankVersion: "pilot-2.6", Answers: map[string]int{}}
	if a.partStates(s) != nil || a.partLocked(s, "I001") || a.submittedModules(s) != nil {
		t.Fatal("an old session has parts")
	}
}

// With no part named the test is finished whole, as before the parts.
func TestFinishWithoutPartSubmitsAllParts(t *testing.T) {
	a := testApp(t)
	student := signUp(t, a, "student", "whole@example.test", "101")
	if _, err := a.AssignmentCommandFor(student.Account.OwnerID, Input{"title": raw(t, "Тест")}, "assignment"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartAssignment(student.Account, mustSnapshot(t, a.repo).Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	id := mustSnapshot(t, a.repo).Sessions[0].Session.ID
	if _, err := a.SurveyCommand(student.Token, id, Input{"value": raw(t, true), "revision": raw(t, 0)}, "consent"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SurveyCommand(student.Token, id, Input{"revision": raw(t, 1)}, "finish"); kind(err) != Incomplete {
		t.Fatalf("an unanswered test was finished: %v", err)
	}
	if _, err := a.SurveyCommand(student.Token, id, Input{"allowPartial": raw(t, true), "revision": raw(t, 1)}, "finish"); err != nil {
		t.Fatal(err)
	}
	s, _ := a.sessionByID(id)
	if !s.Completed || len(s.Parts) != 3 {
		t.Fatalf("completed %v, parts %v", s.Completed, s.Parts)
	}
}
