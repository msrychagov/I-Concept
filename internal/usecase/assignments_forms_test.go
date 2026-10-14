package usecase

import (
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func formAssignmentInput(t *testing.T, platoon string) Input {
	return Input{"title": raw(t, "Автобиография"), "formId": raw(t, "autobiography"), "platoon": raw(t, platoon)}
}

func TestFormAssignmentNeedsNoTemplateButTestsStillDo(t *testing.T) {
	s := testApp(t)
	if _, err := s.AssignmentCommand(formAssignmentInput(t, "101"), "assignment"); err != nil {
		t.Fatal(err)
	}
	stored := mustSnapshot(t, s.repo).Assignments
	if len(stored) != 1 || stored[0].FormID != "autobiography" || !stored[0].Active || stored[0].Platoon != "101" || stored[0].Profile.ID != "" {
		t.Fatalf("assignment: %+v", stored)
	}
	for name, input := range map[string]Input{
		"unknown form":      {"title": raw(t, "X"), "formId": raw(t, "nope")},
		"no title":          {"title": raw(t, " "), "formId": raw(t, "autobiography")},
		"form with profile": {"title": raw(t, "X"), "formId": raw(t, "autobiography"), "profileId": raw(t, "p")},
	} {
		if _, err := s.AssignmentCommand(input, "assignment"); kind(err) != Invalid {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := s.AssignmentCommand(Input{"title": raw(t, "Тест")}, "assignment"); kind(err) != Invalid {
		t.Fatalf("test assignment without a template: %v", err)
	}
	p := s.emptyProfile()
	p.ID, p.Name, p.Author = "template", "Шаблон", "Тест"
	if err := s.addProfile(p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AssignmentCommand(Input{"title": raw(t, "Тест"), "profileId": raw(t, "template")}, "assignment"); err != nil {
		t.Fatal(err)
	}
	if got := mustSnapshot(t, s.repo).Assignments[1]; got.FormID != "" || got.Profile.ID != "template" {
		t.Fatalf("test assignment: %+v", got)
	}
}

func TestStartingAFormTwiceReturnsTheSameSubmission(t *testing.T) {
	s := testApp(t)
	login := signUp(t, s, "student", "first@example.test", "101")
	if _, err := s.AssignmentCommandFor(login.Account.OwnerID, formAssignmentInput(t, "101"), "assignment"); err != nil {
		t.Fatal(err)
	}
	id := mustSnapshot(t, s.repo).Assignments[0].ID
	first, err := s.StartAssignment(login.Account, id)
	if err != nil {
		t.Fatal(err)
	}
	url := first.(map[string]any)["url"].(string)
	if !strings.HasPrefix(url, "/autobiography?id=") {
		t.Fatalf("url %s", url)
	}
	second, _ := s.StartAssignment(login.Account, id)
	if second.(map[string]any)["url"] != url || len(mustSnapshot(t, s.repo).Submissions) != 1 {
		t.Fatal("second start created another submission")
	}
	stored := mustSnapshot(t, s.repo).Submissions[0]
	if stored.StudentID != login.Account.ID || stored.AssignmentID != id || stored.FormVersion != "1.0" || stored.Consent || len(mustSnapshot(t, s.repo).Sessions) != 0 {
		t.Fatalf("submission: %+v", stored)
	}
}

func TestStartingAFormRespectsPlatoonAndAccess(t *testing.T) {
	s := testApp(t)
	mine := signUp(t, s, "student", "first@example.test", "101")
	other := signUp(t, s, "student", "second@example.test", "102")
	if _, err := s.AssignmentCommandFor(mine.Account.OwnerID, formAssignmentInput(t, "101"), "assignment"); err != nil {
		t.Fatal(err)
	}
	id := mustSnapshot(t, s.repo).Assignments[0].ID
	if _, err := s.StartAssignment(other.Account, id); kind(err) != Forbidden {
		t.Fatalf("another platoon started it: %v", err)
	}
	if _, err := s.StartAssignment(mine.Account, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AssignmentCommandFor(mine.Account.OwnerID, Input{"id": raw(t, id), "active": raw(t, false)}, "assignment_status"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartAssignment(mine.Account, id); kind(err) != Forbidden {
		t.Fatalf("continued through a closed assignment: %v", err)
	}
	_ = s.repo.Update(func(db *model.Database) error {
		db.Assignments[0].Active = true
		db.Submissions[0].Completed = true
		return nil
	})
	if _, err := s.StartAssignment(mine.Account, id); kind(err) != Conflict {
		t.Fatalf("restarted a submitted form: %v", err)
	}
}
