package usecase

import (
	"strings"
	"testing"
)

func TestDemoAutobiographyHasOneSubmittedAndOneStartedForm(t *testing.T) {
	s := testApp(t)
	if _, err := s.ResetDemo(); err != nil {
		t.Fatal(err)
	}
	db := mustSnapshot(t, s.repo)
	if len(db.Assignments) != 8 || db.Assignments[3].FormID != "autobiography" || db.Assignments[3].Platoon != "101" {
		t.Fatalf("assignments: %+v", db.Assignments)
	}
	// The first two submissions are the autobiography; the essays follow.
	if len(db.Submissions) != 11 || !db.Submissions[0].Completed || db.Submissions[1].Completed {
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

func TestDemoEssaysAreTwoAssignmentsOfOneCohortWithTwoPairs(t *testing.T) {
	s := testApp(t)
	if _, err := s.ResetDemo(); err != nil {
		t.Fatal(err)
	}
	db := mustSnapshot(t, s.repo)
	if len(db.Cohorts) != 1 || db.Cohorts[0].Year != 2027 || len(db.Cohorts[0].PlatoonIDs) != 3 {
		t.Fatalf("cohorts: %+v", db.Cohorts)
	}
	early, final := db.Assignments[5], db.Assignments[6]
	if early.FormID != essayForm || early.Active || final.Active == false || early.CohortID != db.Cohorts[0].ID || final.CohortID != early.CohortID || early.OwnerID != final.OwnerID {
		t.Fatalf("essay assignments: %+v %+v", early, final)
	}
	perStudent := map[string]int{}
	count := map[string]int{}
	form := s.forms[essayForm]
	for _, submission := range db.Submissions {
		if submission.FormID != essayForm {
			continue
		}
		count[submission.AssignmentID]++
		perStudent[submission.StudentID]++
		if progress := formProgress(form, submission); !submission.Completed || len(progress.Complete) != 3 {
			t.Fatalf("an essay is not complete: %v", progress.Missing)
		}
	}
	pairs := 0
	for _, n := range perStudent {
		if n == 2 {
			pairs++
		}
	}
	if count[early.ID] != 3 || count[final.ID] != 6 || len(perStudent) != 7 || pairs != 2 {
		t.Fatalf("essays per assignment %v, per student %v", count, perStudent)
	}
	// The students of the cohort see the open essay, whoever invited them; the earlier essay of a pair is found.
	for _, submission := range db.Submissions {
		if submission.AssignmentID == final.ID {
			if previous, ok := previousEssay(db, submission); ok != (perStudent[submission.StudentID] == 2) || ok && previous.AssignmentID != early.ID {
				t.Fatalf("earlier essay of %s: %v", submission.ID, ok)
			}
		}
	}
}

// The demo essays hold no personal data the cut would change, and every answer fits the form.
func TestDemoEssaysFitTheForm(t *testing.T) {
	s := testApp(t)
	form := s.forms[essayForm]
	numbers := map[string]bool{}
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			numbers[field.Number] = true
		}
	}
	for _, essay := range demoEssays {
		if len(essay.Answers) != len(numbers) {
			t.Errorf("%s: %d answers", essay.Key, len(essay.Answers))
		}
		for number, text := range essay.Answers {
			if !numbers[number] || len([]rune(text)) > 2000 || len(strings.Fields(text)) < 3 {
				t.Errorf("%s %s", essay.Key, number)
			}
			if got, n := maskText(text, "отчество"); n != 0 {
				t.Errorf("%s %s: personal data %q", essay.Key, number, got)
			}
		}
	}
}
