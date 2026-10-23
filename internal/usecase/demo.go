package usecase

import (
	"encoding/json"
	"fmt"
	"time"
	"vucdiagnostic/internal/domain/model"
)

// ResetDemo is an explicit maintenance command; never runs on normal startup.
func (a *Service) ResetDemo() (string, error) {
	db, err := a.demoDatabase()
	if err != nil {
		return "", err
	}
	return a.repo.ReplaceWithBackup(db)
}

// SeedDemo only fills an empty repository, even with concurrent writers.
func (a *Service) SeedDemo() error {
	db, err := a.demoDatabase()
	if err != nil {
		return err
	}
	return a.ImportData(db)
}

func (a *Service) ImportData(db model.Database) error {
	if db.Empty() {
		return fmt.Errorf("source data is empty")
	}
	normalizeOwnership(&db)
	return a.repo.Update(func(current *model.Database) error {
		if !current.Empty() {
			return fmt.Errorf("target database is not empty; import/seed cancelled")
		}
		*current = model.Clone(db)
		return nil
	})
}

func (a *Service) demoDatabase() (model.Database, error) {
	now := time.Now().UTC()
	created := now.Add(-72 * time.Hour).Format(time.RFC3339)
	db := model.Database{Students: []model.Account{}, Assignments: []model.Assignment{}, Logins: []model.Login{}, Profiles: []model.Profile{}, Sessions: []model.SessionRecord{}, Submissions: []model.FormSubmission{}}
	db.VUCs = []model.VUC{{ID: uuid(), Name: "Демонстрационный ВУЦ"}}
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
	addAccount := func(role, name, platoon, email, password, ownerID string) string {
		salt := token()
		id := uuid()
		platoonID := ""
		if platoon != "" {
			for _, p := range db.Platoons {
				if p.Name == platoon {
					platoonID = p.ID
				}
			}
			if platoonID == "" {
				platoonID = uuid()
				db.Platoons = append(db.Platoons, model.Platoon{ID: platoonID, Name: platoon, VUC: "Демонстрационный ВУЦ"})
			}
		}
		db.Students = append(db.Students, model.Account{VUC: "Демонстрационный ВУЦ", PlatoonID: platoonID, ID: id, Role: role, Name: name, Platoon: platoon, Email: email, PasswordHash: passwordHash(password, salt), Salt: salt, Created: created, OwnerID: ownerID})
		return id
	}
	teacherID := addAccount("teacher", "Преподаватель Демо", "", "teacher@example.test", "TeacherDemo2026!", "")
	names := []string{"Иванов Иван Иванович", "Петров Пётр Петрович", "Сидоров Алексей Андреевич", "Смирнов Артём Олегович", "Волков Максим Игоревич", "Орлов Дмитрий Сергеевич", "Кузнецов Павел Алексеевич", "Соколов Никита Романович", "Новиков Михаил Денисович"}
	for n, name := range names {
		addAccount("student", name, fmt.Sprint(101+n/3), fmt.Sprintf("student%d@example.test", n+1), "StudentDemo2026!", teacherID)
	}
	// Invented instructors: namesakes on purpose, so that essays have cases the
	// model must not merge (the same name and patronymic, the same surname).
	for n, entry := range [][4]string{
		{"Иванов", "Игорь", "Петрович", "полковник"},
		{"Сидоров", "Игорь", "Петрович", "майор"},
		{"Кузнецов", "Илья", "Павлович", "старший лейтенант"},
		{"Кузнецов", "Никита", "Львович", "лейтенант"},
		{"Петрова", "Анна", "Сергеевна", ""},
		{"Ковальчук", "Олег", "Ильич", "подполковник"},
		{"Белоцерковский", "Юрий", "Андреевич", "капитан"},
	} {
		db.Instructors = append(db.Instructors, model.Instructor{ID: uuid(), VUC: "Демонстрационный ВУЦ", Number: n + 1,
			Surname: entry[0], Name: entry[1], Patronymic: entry[2], Rank: entry[3], UpdatedBy: teacherID, Updated: created})
	}
	db.VUCs[0].LastInstructor = len(db.Instructors)
	full.OwnerID, short.OwnerID = teacherID, teacherID
	db.Profiles = []model.Profile{full, short}
	db.Assignments = []model.Assignment{{ID: uuid(), Title: "Входное тестирование", Profile: short, Active: true, Created: created, OwnerID: teacherID}, {ID: uuid(), Title: "Диагностика взвода 101", Platoon: "101", Profile: full, Active: true, Created: created, OwnerID: teacherID}, {ID: uuid(), Title: "Завершённый этап", Profile: short, Active: false, Created: created, OwnerID: teacherID}}
	for n, student := range db.Students {
		if student.Role != "student" {
			continue
		}
		for j, as := range db.Assignments {
			if !assignmentApplies(db, as, student) {
				continue
			}
			if j == 0 && n%3 == 1 {
				continue
			}
			s := model.Session{ID: uuid(), StudentID: student.ID, AssignmentID: as.ID, Kind: "student", Participant: student.Name, Group: student.Platoon, Wave: as.Title, Profile: clone(as.Profile), BankVersion: a.bank.Version, Created: created, Updated: now.Add(-time.Duration(n) * time.Hour).Format(time.RFC3339), Seed: uint32(n*97 + j), Consent: true, ConsentAt: created, Answers: map[string]int{}, Notes: map[string]string{}, Plan: []model.PlanItem{}, TeacherID: teacherID}
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
	if err := a.addDemoAutobiography(&db, created, now, teacherID); err != nil {
		return model.Database{}, err
	}
	a.addDemoMethods(&db, full, created, now, teacherID)
	return db, nil
}

// addDemoAutobiography assigns the autobiography to platoon 101: the first
// student of the platoon has submitted it, the second has filled a few sections.
func (a *Service) addDemoAutobiography(db *model.Database, created string, now time.Time, teacherID string) error {
	form, ok := a.forms["autobiography"]
	if !ok {
		return nil
	}
	as := model.Assignment{ID: uuid(), Title: "Автобиография", Platoon: "101", FormID: form.ID, Active: true, Created: created, OwnerID: teacherID}
	db.Assignments = append(db.Assignments, as)
	answers, err := sampleAnswers(form, map[string]bool{"work": true, "languages": true, "relatives": true, "sport": true, "science": true}, now)
	if err != nil {
		return err
	}
	students := []model.Account{}
	for _, account := range db.Students {
		if account.Role == "student" && assignmentApplies(*db, as, account) {
			students = append(students, account)
		}
	}
	for n, student := range students[:min(2, len(students))] {
		submission := model.FormSubmission{ID: uuid(), FormID: form.ID, FormVersion: form.Version, StudentID: student.ID, AssignmentID: as.ID, Created: created, Updated: now.Add(-time.Duration(n+1) * time.Hour).Format(time.RFC3339), Consent: true, ConsentAt: created, SavedSections: []string{}, Answers: map[string]json.RawMessage{}}
		for i, section := range form.Sections {
			if n == 1 && i >= 4 {
				break
			}
			submission.SavedSections = append(submission.SavedSections, section.ID)
			for _, field := range section.Fields {
				if value, ok := answers[field.ID]; ok {
					submission.Answers[field.ID] = value
				}
			}
		}
		if n == 0 {
			submission.Completed = true
			submission.CompletedAt = submission.Updated
		}
		db.Submissions = append(db.Submissions, submission)
	}
	return nil
}
