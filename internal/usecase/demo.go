package usecase

import (
	"fmt"
	"time"
	"vucdiagnostic/internal/domain/model"
)

// ResetDemo is an explicit maintenance command; never runs on normal startup.
func (a *Service) ResetDemo() (string, error) {
	now := time.Now().UTC()
	created := now.Add(-72 * time.Hour).Format(time.RFC3339)
	db := model.Database{Students: []model.Account{}, Assignments: []model.Assignment{}, Logins: []model.Login{}, Profiles: []model.Profile{}, Sessions: []model.SessionRecord{}}
	full := a.emptyProfile()
	full.ID = uuid()
	full.Name = "Полная диагностика"
	full.Author = "Демонстрационный преподаватель"
	full.Created = created
	full.BankVersion = a.bank.Version
	full.Questions = clone(a.bank.Items)
	short := clone(full)
	short.ID = uuid()
	short.Name = "Короткое тестирование — 3 показателя"
	short.Values = map[string]*float64{}
	for _, id := range []string{"Q01", "Q02", "D01"} {
		v := 5.0
		short.Values[id] = &v
	}
	short.Questions = []model.Item{}
	for _, i := range a.bank.Items {
		if _, ok := short.Values[i.Indicator]; ok && (i.Module == "B" || i.Module == "D") {
			short.Questions = append(short.Questions, i)
		}
	}
	db.Profiles = []model.Profile{full, short}
	addAccount := func(role, name, platoon, email, password string) {
		salt := token()
		db.Students = append(db.Students, model.Account{ID: uuid(), Role: role, Name: name, Platoon: platoon, Email: email, PasswordHash: passwordHash(password, salt), Salt: salt, Created: created})
	}
	addAccount("teacher", "Преподаватель Демо", "", "teacher@example.test", "TeacherDemo2026!")
	names := []string{"Иванов Иван Иванович", "Петров Пётр Петрович", "Сидоров Алексей Андреевич", "Смирнов Артём Олегович", "Волков Максим Игоревич", "Орлов Дмитрий Сергеевич", "Кузнецов Павел Алексеевич", "Соколов Никита Романович", "Новиков Михаил Денисович"}
	for n, name := range names {
		addAccount("student", name, fmt.Sprint(101+n/3), fmt.Sprintf("student%d@example.test", n+1), "StudentDemo2026!")
	}
	db.Assignments = []model.Assignment{{ID: uuid(), Title: "Входное тестирование", Profile: short, Active: true, Created: created}, {ID: uuid(), Title: "Диагностика взвода 101", Platoon: "101", Profile: full, Active: true, Created: created}, {ID: uuid(), Title: "Завершённый этап", Profile: short, Active: false, Created: created}}
	for n, student := range db.Students {
		if student.Role != "student" {
			continue
		}
		for j, as := range db.Assignments {
			if !assignmentApplies(as, student) {
				continue
			}
			if j == 0 && n%3 == 1 {
				continue
			}
			s := model.Session{ID: uuid(), StudentID: student.ID, AssignmentID: as.ID, Kind: "student", Participant: student.Name, Group: student.Platoon, Wave: as.Title, Profile: clone(as.Profile), BankVersion: a.bank.Version, Created: created, Updated: now.Add(-time.Duration(n) * time.Hour).Format(time.RFC3339), Seed: uint32(n*97 + j), Consent: true, ConsentAt: created, Answers: map[string]int{}, Notes: map[string]string{}, Plan: []model.PlanItem{}}
			s.Completed = j == 2 || n%3 == 0
			items := a.itemsFor(s)
			for k, item := range items {
				if !s.Completed && k >= 3 {
					break
				}
				value := item.Min + (n+k)%(item.Max-item.Min+1)
				s.Answers[item.ID] = value
			}
			db.Sessions = append(db.Sessions, model.SessionRecord{TokenHash: hash(token()), Session: s})
		}
	}
	return a.repo.ReplaceWithBackup(db)
}
