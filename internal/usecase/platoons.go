package usecase

import (
	"strings"
	"time"
	"vucdiagnostic/internal/domain/model"
)

func sameVUC(a, b string) bool { return a != "" && b != "" && strings.EqualFold(a, b) }

func managementTeacher(db model.Database, id string) (model.Account, error) {
	for _, t := range db.Students {
		if t.ID == id && t.Role == "teacher" && !t.Disabled {
			if _, exists := vucIn(db, t.VUC); !exists {
				return t, problem(Forbidden, "ВУЦ не назначен. Обратитесь к администратору.")
			}
			return t, nil
		}
	}
	return model.Account{}, problem(Unauthenticated, "Войдите в кабинет преподавателя.")
}

func platoonIn(db model.Database, id, vuc string) (model.Platoon, bool) {
	for _, p := range db.Platoons {
		if p.ID == id && sameVUC(p.VUC, vuc) {
			return p, true
		}
	}
	return model.Platoon{}, false
}

func (a *Service) PlatoonManagement(teacherID string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	t, err := managementTeacher(db, teacherID)
	if err != nil {
		return nil, err
	}
	platoons := []model.Platoon{}
	students := []map[string]any{}
	for _, p := range db.Platoons {
		if sameVUC(p.VUC, t.VUC) {
			platoons = append(platoons, p)
		}
	}
	for _, s := range db.Students {
		if s.Role == "student" && sameVUC(s.VUC, t.VUC) {
			students = append(students, PublicAccount(s))
		}
	}
	return map[string]any{"vuc": t.VUC, "platoons": platoons, "students": students}, nil
}

func (a *Service) SavePlatoon(teacherID, id, name string) (any, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || len(name) > 80 {
		return nil, problem(Invalid, "Укажите название взвода (до 80 символов).")
	}
	return a.transaction(func(db *model.Database) (any, error) {
		t, err := managementTeacher(*db, teacherID)
		if err != nil {
			return nil, err
		}
		for _, p := range db.Platoons {
			if sameVUC(p.VUC, t.VUC) && strings.EqualFold(p.Name, name) && p.ID != id {
				return nil, problem(Conflict, "Взвод с таким названием уже существует.")
			}
		}
		if id == "" {
			p := model.Platoon{ID: uuid(), Name: name, VUC: t.VUC}
			db.Platoons = append(db.Platoons, p)
			return p, nil
		}
		for i, p := range db.Platoons {
			if p.ID != id || !sameVUC(p.VUC, t.VUC) {
				continue
			}
			db.Platoons[i].Name = name
			for j := range db.Students {
				if db.Students[j].PlatoonID == id {
					db.Students[j].Platoon = name
				}
			}
			for j := range db.Invitations {
				if db.Invitations[j].PlatoonID == id {
					db.Invitations[j].Platoon = name
				}
			}
			// Assignments still use platoon names; update only owners in this center.
			for _, owner := range db.Students {
				if owner.Role == "teacher" && sameVUC(owner.VUC, t.VUC) {
					for j := range db.Assignments {
						if db.Assignments[j].OwnerID == owner.ID && db.Assignments[j].Platoon == p.Name {
							db.Assignments[j].Platoon = name
						}
					}
				}
			}
			return db.Platoons[i], nil
		}
		return nil, problem(NotFound, "Взвод не найден.")
	})
}

func (a *Service) ManageStudent(teacherID, studentID, action, platoonID string) (any, error) {
	return a.transaction(func(db *model.Database) (any, error) {
		t, err := managementTeacher(*db, teacherID)
		if err != nil {
			return nil, err
		}
		for i := range db.Students {
			s := &db.Students[i]
			if s.ID != studentID || s.Role != "student" || !sameVUC(s.VUC, t.VUC) {
				continue
			}
			switch action {
			case "move":
				p, ok := platoonIn(*db, platoonID, t.VUC)
				if !ok {
					return nil, problem(NotFound, "Взвод не найден в вашем ВУЦ.")
				}
				s.PlatoonID = p.ID
				s.Platoon = p.Name
			case "remove":
				s.PlatoonID = ""
				s.Platoon = ""
			case "disable":
				s.Disabled = true
			case "enable":
				s.Disabled = false
			default:
				return nil, problem(Invalid, "Неизвестное действие.")
			}
			if action == "disable" || action == "remove" {
				kept := []model.Login{}
				for _, login := range db.Logins {
					if login.StudentID != s.ID {
						kept = append(kept, login)
					}
				}
				db.Logins = kept
			}
			return PublicAccount(*s), nil
		}
		return nil, problem(NotFound, "Студент не найден.")
	})
}

// AssignTeacherVUC is restricted to the administrative adapter. Existing
// teachers can be assigned once; moving a populated center requires a migration.
func (a *Service) AssignTeacherVUC(id, vuc string) (any, error) {
	vuc = strings.Join(strings.Fields(vuc), " ")
	if vuc == "" || len(vuc) > 200 {
		return nil, problem(Invalid, "Укажите ВУЦ (до 200 символов).")
	}
	return a.transaction(func(db *model.Database) (any, error) {
		center, exists := vucIn(*db, vuc)
		if !exists {
			return nil, problem(NotFound, "ВУЦ не существует. Сначала администратор должен создать его.")
		}
		vuc = center.Name
		for i := range db.Students {
			t := &db.Students[i]
			if t.ID != id || t.Role != "teacher" {
				continue
			}
			if t.VUC != "" {
				return nil, problem(Conflict, "ВУЦ уже назначен.")
			}
			t.VUC = vuc
			for j := range db.Students {
				s := &db.Students[j]
				if s.Role != "student" || s.OwnerID != id || s.VUC != "" {
					continue
				}
				s.VUC = vuc
				if s.Platoon == "" {
					continue
				}
				for _, p := range db.Platoons {
					if sameVUC(p.VUC, vuc) && strings.EqualFold(p.Name, s.Platoon) {
						s.PlatoonID = p.ID
						s.Platoon = p.Name
						break
					}
				}
				if s.PlatoonID == "" {
					p := model.Platoon{ID: uuid(), Name: s.Platoon, VUC: vuc}
					db.Platoons = append(db.Platoons, p)
					s.PlatoonID = p.ID
				}
			}
			for j := range db.Invitations {
				if db.Invitations[j].Role == "student" && db.Invitations[j].IssuerID == id {
					db.Invitations[j].Expires = time.Now().UTC()
				}
			}
			return PublicAccount(*t), nil
		}
		return nil, problem(NotFound, "Преподаватель не найден.")
	})
}
