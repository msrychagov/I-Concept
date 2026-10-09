package usecase

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestCabinetShowsFormAssignmentsWithSectionProgress(t *testing.T) {
	f := newFormFixture(t)
	list := func() []map[string]any {
		state, err := f.service.AccountState(f.token, "student")
		if err != nil {
			t.Fatal(err)
		}
		return state.(map[string]any)["assignments"].([]map[string]any)
	}
	if err := f.service.repo.Update(func(db *model.Database) error { db.Submissions = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	entry := list()[0]
	if entry["formId"] != "autobiography" || entry["status"] != "Не начато" || entry["sessionId"] != nil {
		t.Fatalf("not started: %v", entry)
	}
	f.submission = addSubmission(t, f.service, f.student.ID, f.assignment)
	f.consent()
	f.fillAll(nil)
	entry = list()[0]
	if entry["status"] != "В процессе" || entry["answered"] != 17 || entry["total"] != 17 || entry["sessionId"] != f.submission {
		t.Fatalf("filled: %v", entry)
	}
	if _, err := f.command("completion", Input{}); err != nil {
		t.Fatal(err)
	}
	entry = list()[0]
	if entry["status"] != "Завершено" || entry["completed"] != true || entry["finishedAt"] == "" {
		t.Fatalf("completed: %v", entry)
	}
	for _, private := range []string{"answers", "profile", "submission"} {
		if _, leaked := entry[private]; leaked {
			t.Fatalf("%s leaked into the cabinet", private)
		}
	}
}

func TestTeachingStateSkipsFormAssignments(t *testing.T) {
	f := newFormFixture(t)
	state, _ := f.service.TeachingState()
	if got := state.(map[string]any)["assignments"].([]map[string]any); len(got) != 0 {
		t.Fatalf("form assignments listed among tests: %v", got)
	}
	if got := state.(map[string]any)["sessions"].([]map[string]any); len(got) != 0 {
		t.Fatalf("sessions: %v", got)
	}
}
