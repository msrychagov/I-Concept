package usecase

import "testing"

func TestDemoAutobiographyHasOneSubmittedAndOneStartedForm(t *testing.T) {
	s := testApp(t)
	if _, err := s.ResetDemo(); err != nil {
		t.Fatal(err)
	}
	db := s.repo.Snapshot()
	if len(db.Assignments) != 4 || db.Assignments[3].FormID != "autobiography" || db.Assignments[3].Platoon != "101" {
		t.Fatalf("assignments: %+v", db.Assignments)
	}
	if len(db.Submissions) != 2 || !db.Submissions[0].Completed || db.Submissions[1].Completed {
		t.Fatalf("submissions: %+v", db.Submissions)
	}
	form := s.forms["autobiography"]
	if progress := formProgress(form, db.Submissions[0]); len(progress.Complete) != 17 {
		t.Fatalf("submitted demo form is incomplete: %v", progress.Missing)
	}
	if progress := formProgress(form, db.Submissions[1]); len(progress.Complete) != 4 || len(progress.Missing) != 12 {
		t.Fatalf("started demo form: %+v", progress)
	}
	for _, rec := range db.Sessions {
		if rec.Session.AssignmentID == db.Assignments[3].ID {
			t.Fatal("a test session was created for the questionnaire")
		}
	}
}
