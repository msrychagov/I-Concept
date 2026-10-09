package usecase

import (
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
			return nil, failure(Forbidden, map[string]any{"error": "Тестирование недоступно."})
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
		s := model.Session{ID: uuid(), StudentID: student.ID, AssignmentID: id, Kind: "student", Participant: student.Name, Group: student.Platoon, Wave: assignment.Title, Profile: clone(assignment.Profile), BankVersion: assignment.Profile.BankVersion, Created: now, Updated: now, Seed: uint32(time.Now().UnixNano()), Activity: assignment.Activity, Motivation: assignment.Motivation, Answers: map[string]int{}, Notes: map[string]string{}, Plan: []model.PlanItem{}}
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
	return a.transaction(func(db *model.Database) (any, error) {
		if op == "assignment" {
			var p model.Profile
			for _, x := range db.Profiles {
				if x.ID == str(b, "profileId") {
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
			as := model.Assignment{ID: uuid(), Title: title, Platoon: platoon, Profile: clone(p), Activity: boolean(b, "activity"), Motivation: boolean(b, "motivation"), Active: true, Created: time.Now().UTC().Format(time.RFC3339)}
			db.Assignments = append(db.Assignments, as)

			return map[string]any{"ok": true}, nil
		}
		if op == "assignment_status" {
			for n := range db.Assignments {
				if db.Assignments[n].ID == str(b, "id") {
					db.Assignments[n].Active = boolean(b, "active")

					return map[string]any{"ok": true}, nil
				}
			}
			return nil, failure(NotFound, map[string]any{"error": "Тестирование не найдено."})
		}
		return nil, failure(Invalid, map[string]any{"error": fmt.Sprintf("Неизвестное действие %s", op)})
	})
}
