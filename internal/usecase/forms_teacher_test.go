package usecase

import (
	"encoding/json"
	"slices"
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

func TestPotentialSummaryCountsTheLatestSubmittedAutobiographies(t *testing.T) {
	s := testApp(t)
	first := signUp(t, s, "student", "first@example.test", "101").Account
	second := signUp(t, s, "student", "second@example.test", "102").Account
	drafting := signUp(t, s, "student", "third@example.test", "101").Account
	assignment := addFormAssignment(t, s, "", first.OwnerID)
	answers := func(values answerSet) map[string]json.RawMessage {
		result := map[string]json.RawMessage{}
		for key, value := range values {
			result[key] = raw(t, value)
		}
		return result
	}
	strong := answers(answerSet{"sport_has": "yes", "sport_list": []record{{"sport_kind": "sambo", "sport_now": "now", "sport_years": "y5", "sport_rank": "kms", "sport_level": "russia"}}})
	ids := map[string]string{}
	err := s.repo.Update(func(db *model.Database) error {
		add := func(name string, student model.Account, completedAt string, values map[string]json.RawMessage) {
			ids[name] = uuid()
			db.Submissions = append(db.Submissions, model.FormSubmission{ID: ids[name], FormID: "autobiography", FormVersion: "1.0", StudentID: student.ID, AssignmentID: assignment, Completed: completedAt != "", CompletedAt: completedAt, SavedSections: []string{}, Answers: values})
		}
		add("earlier", first, "2026-10-01T10:00:00Z", answers(answerSet{"sport_has": "no"}))
		add("latest", first, "2026-10-10T10:00:00Z", strong)
		add("second", second, "2026-10-05T10:00:00Z", answers(answerSet{"sport_has": "no"}))
		add("draft", drafting, "", strong)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	summary, err := s.PotentialSummaryFor(first.OwnerID, "autobiography", "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Submitted != 2 || !slices.Equal(summary.Platoons, []string{"101", "102"}) || len(summary.Areas) != 6 {
		t.Fatalf("summary: %+v", summary)
	}
	sport := summary.Areas[0]
	if sport.ID != "sport" || len(sport.Levels) != 4 || sport.Levels[0].Level != "high" || sport.Levels[3].Level != "none" {
		t.Fatalf("sport levels: %+v", sport)
	}
	if high := sport.Levels[0].Students; len(high) != 1 || high[0].SubmissionID != ids["latest"] || high[0].Platoon != "101" {
		t.Fatalf("high sport: %+v", high)
	}
	if none := sport.Levels[3].Students; len(none) != 1 || none[0].SubmissionID != ids["second"] {
		t.Fatalf("no sport: %+v", none)
	}
	if science := summary.Areas[3]; science.ID != "science" || len(science.Levels[3].Students) != 2 {
		t.Fatalf("science: %+v", science)
	}

	only, err := s.PotentialSummaryFor(first.OwnerID, "autobiography", "102")
	if err != nil || only.Submitted != 1 || len(only.Platoons) != 2 || len(only.Areas[0].Levels[3].Students) != 1 || len(only.Areas[0].Levels[0].Students) != 0 {
		t.Fatalf("platoon 102: %+v %v", only, err)
	}
	if other, err := s.PotentialSummaryFor("another-teacher", "autobiography", ""); err != nil || other.Submitted != 0 || len(other.Platoons) != 0 {
		t.Fatalf("another teacher: %+v %v", other, err)
	}
	if _, err = s.PotentialSummaryFor("", "missing", ""); kind(err) != NotFound {
		t.Fatalf("unknown form: %v", err)
	}

	detail, err := s.FormSubmissionDetail(ids["latest"])
	if err != nil {
		t.Fatal(err)
	}
	if potential, ok := detail.(map[string]any)["potential"].(Potential); !ok || potential.Areas[0].Level != "high" {
		t.Fatalf("detail potential: %v", detail.(map[string]any)["potential"])
	}
	draft, err := s.FormSubmissionDetail(ids["draft"])
	if err != nil {
		t.Fatal(err)
	}
	if _, leaked := draft.(map[string]any)["potential"]; leaked {
		t.Fatal("potential of a draft")
	}
}
