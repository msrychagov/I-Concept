package usecase

import (
	"encoding/json"
	"fmt"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/formbank"
)

// essayFixture is a center with two teachers and a cohort of one platoon, to
// which the first teacher has given an essay. Students are invited by a third
// account, as in life: the cohort, not the inviter, decides who gets the essay.
type essayFixture struct {
	t          *testing.T
	service    *Service
	form       model.Form
	owner      string
	other      string
	cohort     string
	assignment string
}

func newEssayFixture(t *testing.T) *essayFixture {
	t.Helper()
	s := testApp(t)
	// signUp creates the center and the inviting account.
	first := signUp(t, s, "student", "essay-first@example.test", "2401")
	if err := s.repo.Update(func(db *model.Database) error {
		db.Students = append(db.Students,
			model.Account{ID: "t-owner", Role: "teacher", Name: "Первый Преподаватель", VUC: "Тестовый ВУЦ"},
			model.Account{ID: "t-other", Role: "teacher", Name: "Второй Преподаватель", VUC: "Тестовый ВУЦ"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	saved, err := s.SaveCohort("t-owner", "", CohortInput{Name: "Выпуск 2027", Year: 2027, PlatoonIDs: []string{first.Account.PlatoonID}})
	if err != nil {
		t.Fatal(err)
	}
	f := &essayFixture{t: t, service: s, form: essayFormOf(t), owner: "t-owner", other: "t-other", cohort: saved.(model.Cohort).ID}
	f.assignment = f.assign("Эссе в конце первого семестра")
	return f
}

func essayFormOf(t *testing.T) model.Form {
	t.Helper()
	forms, err := formbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range forms {
		if form.ID == essayForm {
			return form
		}
	}
	t.Fatal("essay is not embedded")
	return model.Form{}
}

// assign gives the essay to the cohort and returns the assignment id.
func (f *essayFixture) assign(title string) string {
	f.t.Helper()
	before := len(mustSnapshot(f.t, f.service.repo).Assignments)
	if _, err := f.service.AssignmentCommandFor(f.owner, Input{"title": raw(f.t, title), "formId": raw(f.t, essayForm), "cohortId": raw(f.t, f.cohort)}, "assignment"); err != nil {
		f.t.Fatal(err)
	}
	list := mustSnapshot(f.t, f.service.repo).Assignments
	if len(list) != before+1 {
		f.t.Fatalf("assignments: %+v", list)
	}
	return list[len(list)-1].ID
}

// student registers a student of a platoon, invited by the fixture account.
func (f *essayFixture) student(email, platoon string) LoginResult {
	f.t.Helper()
	return signUp(f.t, f.service, "student", email, platoon)
}

// start opens the essay of the assignment for the student and returns the submission id.
func (f *essayFixture) start(login LoginResult, assignment string) string {
	f.t.Helper()
	value, err := f.service.StartAssignment(login.Account, assignment)
	if err != nil {
		f.t.Fatal(err)
	}
	url := value.(map[string]any)["url"].(string)
	const prefix = "/essay?id="
	if len(url) <= len(prefix) || url[:len(prefix)] != prefix {
		f.t.Fatalf("url %q", url)
	}
	return url[len(prefix):]
}

// write fills the essay with the answers by question number and submits it.
// Questions without an answer get a plain line.
func (f *essayFixture) write(login LoginResult, submission string, answers map[string]string) {
	f.t.Helper()
	revision := 0
	command := func(op string, fields Input) {
		fields["revision"] = raw(f.t, revision)
		result, err := f.service.FormCommand(login.Token, submission, op, fields)
		if err != nil {
			f.t.Fatalf("%s: %v", op, err)
		}
		revision = result.(map[string]any)["revision"].(int)
	}
	command("consent", Input{"value": raw(f.t, true)})
	for _, section := range f.form.Sections {
		part := map[string]string{}
		for _, field := range section.Fields {
			text, ok := answers[field.Number]
			if !ok {
				text = fmt.Sprintf("Ответ на вопрос %s.", field.Number)
			}
			part[field.ID] = text
		}
		command("section", Input{"sectionId": raw(f.t, section.ID), "answers": raw(f.t, part)})
	}
	command("completion", Input{})
}

// submitted creates a student, who writes an essay for the assignment.
func (f *essayFixture) submitted(email, platoon, assignment string, answers map[string]string) (LoginResult, string) {
	f.t.Helper()
	login := f.student(email, platoon)
	id := f.start(login, assignment)
	f.write(login, id, answers)
	return login, id
}

func submissionOf(t *testing.T, s *Service, id string) model.FormSubmission {
	t.Helper()
	for _, submission := range mustSnapshot(t, s.repo).Submissions {
		if submission.ID == id {
			return submission
		}
	}
	t.Fatalf("no submission %s", id)
	return model.FormSubmission{}
}

func answerOf(t *testing.T, form model.Form, submission model.FormSubmission, number string) string {
	t.Helper()
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			if field.Number == number {
				var text string
				_ = json.Unmarshal(submission.Answers[field.ID], &text)
				return text
			}
		}
	}
	t.Fatalf("no question %s", number)
	return ""
}
