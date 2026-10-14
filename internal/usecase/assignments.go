package usecase

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"vucdiagnostic/internal/domain/model"
)

func (a *Service) StartAssignment(student model.Account, id string) (any, error) {
	return a.transaction(func(db *model.Database) (any, error) {
		var assignment model.Assignment
		for _, x := range db.Assignments {
			if x.ID == id {
				assignment = x
				break
			}
		}
		if assignment.ID == "" || !assignment.Active || !assignmentApplies(assignment, student) {
			if assignment.FormID != "" {
				return nil, failure(Forbidden, map[string]any{"error": "Задание недоступно."})
			}
			return nil, failure(Forbidden, map[string]any{"error": "Тестирование недоступно."})
		}
		if assignment.FormID != "" {
			return a.startForm(db, student, assignment)
		}
		for _, rec := range db.Sessions {
			if rec.Session.StudentID == student.ID && rec.Session.AssignmentID == id {
				if rec.Session.Completed {
					return nil, failure(Conflict, map[string]any{"error": "Тестирование уже завершено."})
				}
				return map[string]any{"url": "/survey?id=" + rec.Session.ID}, nil
			}
		}
		now := time.Now().UTC().Format(time.RFC3339)
		s := model.Session{ID: uuid(), StudentID: student.ID, AssignmentID: id, Kind: "student", Participant: student.Name, Group: student.Platoon, Wave: assignment.Title, Profile: clone(assignment.Profile), BankVersion: assignment.Profile.BankVersion, Created: now, Updated: now, Seed: uint32(time.Now().UnixNano()), Activity: assignment.Activity, Motivation: assignment.Motivation, Answers: map[string]int{}, Notes: map[string]string{}, Plan: []model.PlanItem{}, TeacherID: assignment.OwnerID}
		if s.BankVersion == "" {
			s.BankVersion = a.bank.Version
		}
		if len(a.itemsFor(s)) == 0 {
			return nil, failure(Invalid, map[string]any{"error": "В тестировании нет вопросов."})
		}
		db.Sessions = append(db.Sessions, model.SessionRecord{TokenHash: hash(token()), Session: s})

		return map[string]any{"url": "/survey?id=" + s.ID}, nil
	})
}
func (a *Service) AssignmentCommand(b Input, op string) (any, error) {
	return a.AssignmentCommandFor("", b, op)
}

func (a *Service) AssignmentCommandFor(teacherID string, b Input, op string) (any, error) {
	return a.transaction(func(db *model.Database) (any, error) {
		if op == "assignment" && str(b, "formId") != "" {
			return a.assignForm(db, teacherID, b)
		}
		if op == "assignment" {
			var p model.Profile
			for _, x := range db.Profiles {
				if x.ID == str(b, "profileId") && inTeacherScope(x.OwnerID, teacherID) {
					p = x
					break
				}
			}
			title := strings.TrimSpace(str(b, "title"))
			platoon := strings.TrimSpace(str(b, "platoon"))
			if p.ID == "" || title == "" || len(title) > 200 || len(platoon) > 80 {
				return nil, failure(Invalid, map[string]any{"error": "Выберите шаблон и укажите название тестирования."})
			}
			if p.BankVersion == "" {
				p.BankVersion = a.bank.Version
			}
			if p.Questions == nil {
				p.Questions = clone(a.engineFor(model.Session{Profile: p, BankVersion: p.BankVersion}).bank.Items)
			}
			p.Questions = a.assignmentQuestions(p, boolean(b, "activity"), boolean(b, "motivation"))
			if len(p.Questions) == 0 {
				return nil, failure(Invalid, map[string]any{"error": "В выбранном шаблоне нет доступных вопросов."})
			}
			as := model.Assignment{ID: uuid(), Title: title, Platoon: platoon, Profile: clone(p), Activity: boolean(b, "activity"), Motivation: boolean(b, "motivation"), Active: true, Created: time.Now().UTC().Format(time.RFC3339), OwnerID: teacherID}
			db.Assignments = append(db.Assignments, as)

			return map[string]any{"ok": true}, nil
		}
		if op == "assignment_status" {
			for n := range db.Assignments {
				if db.Assignments[n].ID == str(b, "id") && inTeacherScope(db.Assignments[n].OwnerID, teacherID) {
					db.Assignments[n].Active = boolean(b, "active")

					return map[string]any{"ok": true}, nil
				}
			}
			return nil, failure(NotFound, map[string]any{"error": "Тестирование не найдено."})
		}
		return nil, failure(Invalid, map[string]any{"error": fmt.Sprintf("Неизвестное действие %s", op)})
	})
}

// assignForm assigns a fixed questionnaire; it needs no template.
func (a *Service) assignForm(db *model.Database, teacherID string, b Input) (any, error) {
	form, known := a.forms[str(b, "formId")]
	title := strings.TrimSpace(str(b, "title"))
	platoon := strings.TrimSpace(str(b, "platoon"))
	if !known || str(b, "profileId") != "" || title == "" || len(title) > 200 || len(platoon) > 80 {
		return nil, failure(Invalid, map[string]any{"error": "Выберите анкету и укажите название задания."})
	}
	db.Assignments = append(db.Assignments, model.Assignment{ID: uuid(), Title: title, Platoon: platoon, FormID: form.ID, Active: true, Created: time.Now().UTC().Format(time.RFC3339), OwnerID: teacherID})
	return map[string]any{"ok": true}, nil
}

// startForm returns the student's submission for the assignment, creating it on first start.
func (a *Service) startForm(db *model.Database, student model.Account, assignment model.Assignment) (any, error) {
	form, known := a.forms[assignment.FormID]
	if !known {
		return nil, failure(Invalid, map[string]any{"error": "Анкета не найдена."})
	}
	for _, submission := range db.Submissions {
		if submission.StudentID == student.ID && submission.AssignmentID == assignment.ID {
			if submission.Completed {
				return nil, failure(Conflict, map[string]any{"error": "Анкета уже сдана."})
			}
			return map[string]any{"url": "/autobiography?id=" + submission.ID}, nil
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	submission := model.FormSubmission{ID: uuid(), FormID: form.ID, FormVersion: form.Version, StudentID: student.ID, AssignmentID: assignment.ID, Created: now, Updated: now, SavedSections: []string{}, Answers: map[string]json.RawMessage{}}
	db.Submissions = append(db.Submissions, submission)
	return map[string]any{"url": "/autobiography?id=" + submission.ID}, nil
}
