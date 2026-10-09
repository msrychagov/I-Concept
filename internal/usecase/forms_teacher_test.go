package usecase

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestTeacherSeesDraftsAndSubmittedFormsButNoAnswersInTheOverview(t *testing.T) {
	f := newFormFixture(t)
	f.consent()
	f.fillAll(nil)
	if _, err := f.command("completion", Input{}); err != nil {
		t.Fatal(err)
	}
	second := signUp(t, f.service, "student", "second@example.test", "101")
	draft := addSubmission(t, f.service, second.Account.ID, f.assignment)

	value, err := f.service.FormOverview("autobiography")
	if err != nil {
		t.Fatal(err)
	}
	overview := value.(map[string]any)
	submissions := overview["submissions"].([]map[string]any)
	if len(overview["students"].([]map[string]any)) != 2 || len(overview["assignments"].([]map[string]any)) != 1 || len(submissions) != 2 {
		t.Fatalf("overview: %v", overview)
	}
	if submissions[0]["completed"] != true || submissions[0]["answered"] != 17 || submissions[1]["completed"] != false || submissions[1]["answered"] != 0 {
		t.Fatalf("submissions: %v", submissions)
	}
	if _, leaked := submissions[0]["answers"]; leaked {
		t.Fatal("answers in the overview")
	}

	value, err = f.service.FormSubmissionDetail(f.submission)
	if err != nil {
		t.Fatal(err)
	}
	detail := value.(map[string]any)
	if detail["student"].(map[string]any)["name"] != f.student.Name || !detail["submission"].(model.FormSubmission).Completed || len(detail["submission"].(model.FormSubmission).Answers) == 0 {
		t.Fatalf("detail: %v", detail)
	}
	if _, err = f.service.FormSubmissionDetail(draft); err != nil {
		t.Fatalf("draft is hidden from the teacher: %v", err)
	}
	if _, err = f.service.FormSubmissionDetail("missing"); kind(err) != NotFound {
		t.Fatalf("unknown submission: %v", err)
	}
	if _, err = f.service.FormOverview("missing"); kind(err) != NotFound {
		t.Fatalf("unknown form: %v", err)
	}
	described, err := f.service.FormDescription("autobiography")
	if err != nil || len(described.(model.Form).Sections) != 17 || len(described.(model.Form).Refs["regions"]) != 89 {
		t.Fatalf("description: %v", err)
	}
	if _, err = f.service.FormDescription("missing"); kind(err) != NotFound {
		t.Fatalf("unknown form description: %v", err)
	}
}
