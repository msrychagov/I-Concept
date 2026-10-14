package usecase

import (
	"encoding/json"
	"slices"
	"time"
	"vucdiagnostic/internal/domain/model"
)

// RegisterForms connects fixed questionnaires. It is called once at startup.
func (a *Service) RegisterForms(forms ...model.Form) {
	for _, form := range forms {
		a.forms[form.ID] = model.Clone(form)
	}
}

func staleForm() error {
	return failure(Conflict, map[string]any{"error": "Данные изменились в другом окне. Обновите страницу.", "conflict": true})
}

// studentForm finds the caller's own unfinished submission of an active
// assignment that applies to their platoon.
func (a *Service) studentForm(db *model.Database, loginToken, id string) (*model.FormSubmission, model.Form, error) {
	denied := failure(Forbidden, map[string]any{"error": "Ссылка недействительна."})
	student, ok := accountIn(*db, loginToken, "student")
	if !ok {
		return nil, model.Form{}, denied
	}
	for n := range db.Submissions {
		submission := &db.Submissions[n]
		form, known := a.forms[submission.FormID]
		if submission.ID != id || submission.StudentID != student.ID || submission.Completed || !known {
			continue
		}
		for _, as := range db.Assignments {
			if as.ID == submission.AssignmentID && as.FormID == submission.FormID && as.Active && assignmentApplies(as, student) {
				return submission, form, nil
			}
		}
	}
	return nil, model.Form{}, denied
}

// FormOpen returns the student's draft with the form description and progress.
func (a *Service) FormOpen(loginToken, id string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	submission, form, err := a.studentForm(&db, loginToken, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"submission": submission, "form": form, "progress": formProgress(form, *submission)}, nil
}

// FormCommand changes a submission: op is consent, section or completion.
// The input carries revision and, for a section, sectionId and answers.
func (a *Service) FormCommand(loginToken, id, op string, b Input) (any, error) {
	return a.transaction(func(db *model.Database) (any, error) {
		submission, form, err := a.studentForm(db, loginToken, id)
		if err != nil {
			return nil, err
		}
		if integer(b, "revision") != submission.Revision {
			return nil, staleForm()
		}
		now := time.Now().UTC()
		result := map[string]any{}
		switch op {
		case "consent":
			if !boolean(b, "value") {
				return nil, problem(Invalid, "Для начала нужно согласие.")
			}
			submission.Consent = true
			submission.ConsentAt = now.Format(time.RFC3339)
		case "section":
			if !submission.Consent {
				return nil, problem(Invalid, "Сначала подтвердите согласие.")
			}
			if err = saveFormSection(form, submission, b, now); err != nil {
				return nil, err
			}
			result["progress"] = formProgress(form, *submission)
		case "completion":
			if !submission.Consent {
				return nil, problem(Invalid, "Сначала подтвердите согласие.")
			}
			progress := formProgress(form, *submission)
			if len(progress.Complete) != progress.Total {
				pending := []string{}
				for _, section := range form.Sections {
					if !slices.Contains(progress.Complete, section.ID) {
						pending = append(pending, section.ID)
					}
				}
				return nil, failure(Incomplete, map[string]any{"error": "Заполнены не все разделы. Ответы сохранены.", "sections": pending, "missing": progress.Missing})
			}
			submission.Completed = true
			submission.CompletedAt = now.Format(time.RFC3339)
		default:
			return nil, problem(Invalid, "Неизвестное действие.")
		}
		submission.Updated = now.Format(time.RFC3339)
		submission.Revision++
		result["revision"] = submission.Revision
		result["updated"] = submission.Updated
		if op == "completion" {
			result["completed"] = true
		}
		return result, nil
	})
}

// saveFormSection replaces the stored answers of one section.
func saveFormSection(form model.Form, submission *model.FormSubmission, b Input, now time.Time) error {
	var target *model.FormSection
	for n := range form.Sections {
		if form.Sections[n].ID == str(b, "sectionId") {
			target = &form.Sections[n]
		}
	}
	if target == nil {
		return problem(Invalid, "Раздел не найден.")
	}
	var input map[string]json.RawMessage
	if data, ok := b["answers"]; ok && string(data) != "null" {
		if json.Unmarshal(data, &input) != nil {
			return problem(Invalid, "Ответы должны быть объектом.")
		}
	}
	cleaned, err := cleanSection(form, *target, input, now)
	if err != nil {
		return err
	}
	if submission.Answers == nil {
		submission.Answers = map[string]json.RawMessage{}
	}
	for _, field := range target.Fields {
		delete(submission.Answers, field.ID)
	}
	for key, value := range cleaned {
		submission.Answers[key] = value
	}
	if !slices.Contains(submission.SavedSections, target.ID) {
		submission.SavedSections = append(submission.SavedSections, target.ID)
	}
	return nil
}
