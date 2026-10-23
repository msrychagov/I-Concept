package usecase

import (
	"encoding/json"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestEssayOfACohortReachesItsStudentsWhoeverInvitedThem(t *testing.T) {
	f := newEssayFixture(t)
	insider := f.student("essay-insider@example.test", "2401")
	outsider := f.student("essay-outsider@example.test", "2402")
	if insider.Account.OwnerID == f.owner {
		t.Fatal("the fixture must invite students with another account")
	}
	state, err := f.service.AccountState(insider.Token, "student")
	if err != nil {
		t.Fatal(err)
	}
	entries := state.(map[string]any)["assignments"].([]map[string]any)
	if len(entries) != 1 || entries[0]["id"] != f.assignment || entries[0]["formId"] != essayForm || entries[0]["studentView"] != true {
		t.Fatalf("cabinet of a student of the cohort: %+v", entries)
	}
	state, _ = f.service.AccountState(outsider.Token, "student")
	if entries = state.(map[string]any)["assignments"].([]map[string]any); len(entries) != 0 {
		t.Fatalf("a student of another platoon sees the essay: %+v", entries)
	}
	if _, err = f.service.StartAssignment(outsider.Account, f.assignment); kind(err) != Forbidden {
		t.Fatalf("a student of another platoon started the essay: %v", err)
	}
	// Starting twice gives the same page.
	first := f.start(insider, f.assignment)
	if second := f.start(insider, f.assignment); second != first {
		t.Fatalf("second start: %s and %s", first, second)
	}
	// The draft is the student's own.
	if _, err = f.service.FormOpen(outsider.Token, first); kind(err) != Forbidden {
		t.Fatalf("the draft was opened by another student: %v", err)
	}
	// A student who left the cohort's platoon loses the essay; a closed essay cannot be written.
	if err = f.service.repo.Update(func(db *model.Database) error {
		for i := range db.Assignments {
			db.Assignments[i].Active = false
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.FormOpen(insider.Token, first); kind(err) != Forbidden {
		t.Fatalf("a closed essay was opened: %v", err)
	}
}

func TestEssayAssignmentRules(t *testing.T) {
	f := newEssayFixture(t)
	input := func(fields Input) Input {
		fields["title"] = raw(t, "Эссе")
		fields["formId"] = raw(t, essayForm)
		return fields
	}
	for name, c := range map[string]struct {
		teacher string
		input   Input
		want    ErrorKind
	}{
		"another teacher of the center":  {f.other, input(Input{"cohortId": raw(t, f.cohort)}), Forbidden},
		"no cohort":                      {f.owner, input(Input{}), NotFound},
		"unknown cohort":                 {f.owner, input(Input{"cohortId": raw(t, "nope")}), NotFound},
		"a platoon, not a cohort":        {f.owner, input(Input{"cohortId": raw(t, f.cohort), "platoon": raw(t, "2401")}), Invalid},
		"a cohort for the autobiography": {f.owner, Input{"title": raw(t, "А"), "formId": raw(t, "autobiography"), "cohortId": raw(t, f.cohort)}, Invalid},
	} {
		if _, err := f.service.AssignmentCommandFor(c.teacher, c.input, "assignment"); kind(err) != c.want {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A cohort without platoons gets no essay.
	empty, err := f.service.SaveCohort(f.owner, "", CohortInput{Name: "Пустой", Year: 2030})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.AssignmentCommandFor(f.owner, input(Input{"cohortId": raw(t, empty.(model.Cohort).ID)}), "assignment"); kind(err) != Invalid || !strings.Contains(err.Error(), "нет взводов") {
		t.Fatalf("empty cohort: %v", err)
	}
	if got := len(mustSnapshot(t, f.service.repo).Assignments); got != 1 {
		t.Fatalf("%d assignments after refusals", got)
	}
}

func TestStudentSeesOnlyOwnSubmittedEssay(t *testing.T) {
	f := newEssayFixture(t)
	mine, mineID := f.submitted("essay-mine@example.test", "2401", f.assignment, map[string]string{"1.1": "Хотел служить."})
	other, otherID := f.submitted("essay-theirs@example.test", "2401", f.assignment, nil)
	draft := f.student("essay-draft@example.test", "2401")
	draftID := f.start(draft, f.assignment)

	value, err := f.service.FormOwnAnswers(mine.Token, mineID)
	if err != nil {
		t.Fatal(err)
	}
	shown := value.(map[string]any)
	if form := shown["form"].(model.Form); form.ID != essayForm || len(form.Sections) != 3 {
		t.Fatalf("form: %+v", form.ID)
	}
	if answers := shown["submission"].(map[string]any)["answers"].(map[string]json.RawMessage); !strings.Contains(string(answers["q1_1"]), "Хотел служить.") {
		t.Fatalf("answers: %v", answers)
	}
	for name, c := range map[string]struct{ token, id string }{
		"someone else's essay": {mine.Token, otherID},
		"own draft":            {draft.Token, draftID},
		"unknown":              {mine.Token, "nope"},
		"no login":             {"", mineID},
	} {
		if _, err = f.service.FormOwnAnswers(c.token, c.id); err == nil {
			t.Errorf("%s was shown", name)
		}
	}
	_ = other
	// The autobiography is not shown back.
	auto := signUp(t, f.service, "student", "essay-auto@example.test", "2401")
	assignment := addFormAssignment(t, f.service, "", auto.Account.OwnerID)
	id := addSubmission(t, f.service, auto.Account.ID, assignment)
	if err = f.service.repo.Update(func(db *model.Database) error {
		for i := range db.Submissions {
			if db.Submissions[i].ID == id {
				db.Submissions[i].Completed = true
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.FormOwnAnswers(auto.Token, id); kind(err) != NotFound {
		t.Fatalf("the autobiography was shown back: %v", err)
	}
}
