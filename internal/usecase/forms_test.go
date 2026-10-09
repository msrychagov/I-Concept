package usecase

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

type formFixture struct {
	t          *testing.T
	service    *Service
	form       model.Form
	token      string
	student    model.Account
	assignment string
	submission string
	revision   int
}

func signUp(t *testing.T, s *Service, role, email, platoon string) LoginResult {
	t.Helper()
	body := Input{"name": raw(t, "Тестов Тест"), "email": raw(t, email), "password": raw(t, "TestPassword2026!"), "platoon": raw(t, platoon)}
	result, err := s.Authenticate(role, "register", body)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// addFormAssignment stores a form assignment directly; the assignment scenarios have their own tests.
func addFormAssignment(t *testing.T, s *Service, platoon string) string {
	t.Helper()
	id := uuid()
	err := s.repo.Update(func(db *model.Database) error {
		db.Assignments = append(db.Assignments, model.Assignment{ID: id, Title: "Автобиография", Platoon: platoon, FormID: "autobiography", Active: true})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func addSubmission(t *testing.T, s *Service, studentID, assignmentID string) string {
	t.Helper()
	id := uuid()
	err := s.repo.Update(func(db *model.Database) error {
		db.Submissions = append(db.Submissions, model.FormSubmission{ID: id, FormID: "autobiography", FormVersion: "1.0", StudentID: studentID, AssignmentID: assignmentID, SavedSections: []string{}, Answers: map[string]json.RawMessage{}})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func newFormFixture(t *testing.T) *formFixture {
	t.Helper()
	s := testApp(t)
	login := signUp(t, s, "student", "first@example.test", "101")
	assignment := addFormAssignment(t, s, "101")
	return &formFixture{t: t, service: s, form: autobiography(t), token: login.Token, student: login.Account, assignment: assignment, submission: addSubmission(t, s, login.Account.ID, assignment)}
}

func (f *formFixture) command(op string, fields Input) (any, error) {
	f.t.Helper()
	fields["revision"] = raw(f.t, f.revision)
	result, err := f.service.FormCommand(f.token, f.submission, op, fields)
	if err == nil {
		f.revision = result.(map[string]any)["revision"].(int)
	}
	return result, err
}

func (f *formFixture) consent() {
	f.t.Helper()
	if _, err := f.command("consent", Input{"value": raw(f.t, true)}); err != nil {
		f.t.Fatal(err)
	}
}

func (f *formFixture) save(sectionID string, answers map[string]json.RawMessage) (any, error) {
	f.t.Helper()
	return f.command("section", Input{"sectionId": raw(f.t, sectionID), "answers": raw(f.t, answers)})
}

// fillAll saves every section with sample answers.
func (f *formFixture) fillAll(detailed map[string]bool) {
	f.t.Helper()
	answers, err := sampleAnswers(f.form, detailed, today)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, s := range f.form.Sections {
		part := map[string]json.RawMessage{}
		for _, field := range s.Fields {
			if value, ok := answers[field.ID]; ok {
				part[field.ID] = value
			}
		}
		if _, err = f.save(s.ID, part); err != nil {
			f.t.Fatalf("%s: %v", s.ID, err)
		}
	}
}

func kind(err error) ErrorKind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ""
}

func TestFormSectionsNeedConsentAndReplaceOnlyTheirOwnAnswers(t *testing.T) {
	f := newFormFixture(t)
	if _, err := f.save("about", map[string]json.RawMessage{"name_changed": raw(t, "no")}); kind(err) != Invalid {
		t.Fatalf("saved without consent: %v", err)
	}
	f.consent()
	if _, err := f.save("about", map[string]json.RawMessage{"name_changed": raw(t, "yes"), "prev_names": raw(t, "Иванова Анна")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.save("sport", map[string]json.RawMessage{"sport_has": raw(t, "no")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.save("about", map[string]json.RawMessage{"name_changed": raw(t, "no")}); err != nil {
		t.Fatal(err)
	}
	stored := f.service.repo.Snapshot().Submissions[0]
	if stored.Answers["prev_names"] != nil || stored.Answers["sport_has"] == nil {
		t.Fatalf("answers: %v", stored.Answers)
	}
	if !slices.Equal(stored.SavedSections, []string{"about", "sport"}) {
		t.Fatalf("saved sections: %v", stored.SavedSections)
	}
	if _, err := f.save("nope", nil); kind(err) != Invalid {
		t.Fatal("unknown section accepted")
	}
	if _, err := f.save("sport", map[string]json.RawMessage{"name_changed": raw(t, "no")}); kind(err) != Invalid {
		t.Fatal("field of another section accepted")
	}
}

func TestFormStaleRevisionIsAConflict(t *testing.T) {
	f := newFormFixture(t)
	f.consent()
	f.revision--
	if _, err := f.save("about", map[string]json.RawMessage{"name_changed": raw(t, "no")}); kind(err) != Conflict {
		t.Fatalf("stale revision: %v", err)
	}
}

func TestFormCompletionNeedsEverySection(t *testing.T) {
	f := newFormFixture(t)
	f.consent()
	if _, err := f.save("about", map[string]json.RawMessage{"name_changed": raw(t, "no")}); err != nil {
		t.Fatal(err)
	}
	_, err := f.command("completion", Input{})
	var e *Error
	if !errors.As(err, &e) || e.Kind != Incomplete {
		t.Fatalf("incomplete form completed: %v", err)
	}
	sections := e.Details["sections"].([]string)
	missing := e.Details["missing"].(map[string][]string)
	if !slices.Contains(sections, "sport") || !slices.Contains(sections, "about") || len(sections) != 17 || len(missing["about"]) == 0 {
		t.Fatalf("details: %v", e.Details)
	}
	if f.service.repo.Snapshot().Submissions[0].Completed {
		t.Fatal("completed with gaps")
	}
}

func TestFormCompletionClosesTheSubmission(t *testing.T) {
	f := newFormFixture(t)
	f.consent()
	f.fillAll(nil)
	result, err := f.command("completion", Input{})
	if err != nil || result.(map[string]any)["completed"] != true {
		t.Fatalf("completion: %v %v", result, err)
	}
	stored := f.service.repo.Snapshot().Submissions[0]
	if !stored.Completed || stored.CompletedAt == "" {
		t.Fatal("completion not stored")
	}
	if _, err = f.save("about", map[string]json.RawMessage{"name_changed": raw(t, "no")}); kind(err) != Forbidden {
		t.Fatalf("edited after completion: %v", err)
	}
	if _, err = f.command("completion", Input{}); kind(err) != Forbidden {
		t.Fatalf("completed twice: %v", err)
	}
	if _, err = f.service.FormOpen(f.token, f.submission); kind(err) != Forbidden {
		t.Fatalf("opened after completion: %v", err)
	}
}

func TestFormAccessIsLimitedToTheOwnerOfAnActiveAssignment(t *testing.T) {
	f := newFormFixture(t)
	other := signUp(t, f.service, "student", "second@example.test", "101")
	if _, err := f.service.FormOpen(other.Token, f.submission); kind(err) != Forbidden {
		t.Fatalf("other student opened it: %v", err)
	}
	if _, err := f.service.FormOpen("", f.submission); kind(err) != Forbidden {
		t.Fatalf("anonymous opened it: %v", err)
	}
	teacher := signUp(t, f.service, "teacher", "teacher@example.test", "")
	if _, err := f.service.FormOpen(teacher.Token, f.submission); kind(err) != Forbidden {
		t.Fatalf("teacher token opened a student draft: %v", err)
	}
	if _, err := f.service.FormOpen(f.token, f.submission); err != nil {
		t.Fatal(err)
	}
	err := f.service.repo.Update(func(db *model.Database) error { db.Assignments[0].Active = false; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.FormOpen(f.token, f.submission); kind(err) != Forbidden {
		t.Fatalf("opened through a closed assignment: %v", err)
	}
	err = f.service.repo.Update(func(db *model.Database) error {
		db.Assignments[0].Active = true
		db.Assignments[0].Platoon = "102"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.FormOpen(f.token, f.submission); kind(err) != Forbidden {
		t.Fatalf("opened through another platoon's assignment: %v", err)
	}
}
