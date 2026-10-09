package usecase

import (
	"crypto/subtle"
	"net/mail"
	"strings"
	"time"
	"vucdiagnostic/internal/domain/model"
)

type LoginResult struct {
	Account model.Account
	Token   string
	Expires time.Time
}

func (a *Service) CurrentAccount(loginToken, role string) (model.Account, bool) {
	return accountIn(a.repo.Snapshot(), loginToken, role)
}
func accountIn(db model.Database, loginToken, role string) (model.Account, bool) {
	if loginToken == "" {
		return model.Account{}, false
	}
	for _, login := range db.Logins {
		if login.Hash == hash(loginToken) && time.Now().Before(login.Expires) {
			for _, account := range db.Students {
				if account.ID == login.StudentID && account.Role == role {
					return account, true
				}
			}
		}
	}
	return model.Account{}, false
}
func PublicAccount(account model.Account) map[string]any {
	return map[string]any{"id": account.ID, "name": account.Name, "email": account.Email, "platoon": account.Platoon}
}
func studentPublic(account model.Account) map[string]any { return PublicAccount(account) }
func assignmentApplies(as model.Assignment, account model.Account) bool {
	return as.Platoon == "" || as.Platoon == account.Platoon
}
func (a *Service) Logout(loginToken string) error {
	return a.repo.Update(func(db *model.Database) error {
		kept := []model.Login{}
		for _, login := range db.Logins {
			if login.Hash != hash(loginToken) {
				kept = append(kept, login)
			}
		}
		db.Logins = kept
		return nil
	})
}
func (a *Service) Authenticate(role, op string, b Input) (LoginResult, error) {
	if role != "student" && role != "teacher" {
		return LoginResult{}, problem(Invalid, "Неизвестная роль.")
	}
	email := strings.ToLower(strings.TrimSpace(str(b, "email")))
	password := str(b, "password")
	if len(email) > 254 || len(password) > 256 {
		return LoginResult{}, problem(Invalid, "Проверьте почту и пароль.")
	}
	var account model.Account
	if op == "register" {
		name := strings.TrimSpace(str(b, "name"))
		platoon := strings.TrimSpace(str(b, "platoon"))
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || len(strings.Fields(name)) < 2 || len(name) > 200 || (role == "student" && platoon == "") || len(platoon) > 80 || len(password) < 10 {
			return LoginResult{}, problem(Invalid, "Заполните обязательные поля, корректную почту и пароль длиной не менее 10 символов.")
		}
		salt := token()
		account = model.Account{ID: uuid(), Role: role, Name: name, Platoon: platoon, Email: email, Salt: salt, PasswordHash: passwordHash(password, salt), Created: time.Now().UTC().Format(time.RFC3339)}
	} else if op == "login" {
		for _, other := range a.repo.Snapshot().Students {
			if other.Email == email && other.Role == role {
				account = other
				break
			}
		}
		salt := account.Salt
		if salt == "" {
			salt = "invalid-account"
		}
		candidate := passwordHash(password, salt)
		if account.ID == "" || subtle.ConstantTimeCompare([]byte(candidate), []byte(account.PasswordHash)) != 1 {
			return LoginResult{}, problem(Unauthenticated, "Неверная почта или пароль.")
		}
	} else {
		return LoginResult{}, problem(Invalid, "Неизвестное действие.")
	}
	result := LoginResult{Account: account, Token: token(), Expires: time.Now().Add(24 * time.Hour)}
	err := a.repo.Update(func(db *model.Database) error {
		if op == "register" {
			for _, other := range db.Students {
				if other.Email == email && other.Role == role {
					return problem(Conflict, "Эта почта уже зарегистрирована. Войдите в кабинет.")
				}
			}
			db.Students = append(db.Students, account)
		} else {
			valid := false
			for _, current := range db.Students {
				if current.ID == account.ID && current.PasswordHash == account.PasswordHash && current.Email == email {
					valid = true
					result.Account = current
					break
				}
			}
			if !valid {
				return problem(Unauthenticated, "Данные входа изменились. Войдите заново.")
			}
		}
		logins := []model.Login{}
		for _, login := range db.Logins {
			if time.Now().Before(login.Expires) {
				logins = append(logins, login)
			}
		}
		db.Logins = append(logins, model.Login{Hash: hash(result.Token), StudentID: account.ID, Expires: result.Expires})
		return nil
	})
	return result, err
}
func (a *Service) AssignedSession(loginToken, id string) (model.Session, bool) {
	db := a.repo.Snapshot()
	student, ok := accountIn(db, loginToken, "student")
	if !ok {
		return model.Session{}, false
	}
	for _, record := range db.Sessions {
		s := record.Session
		if s.ID != id || s.StudentID != student.ID || s.Completed {
			continue
		}
		for _, as := range db.Assignments {
			if as.ID == s.AssignmentID && as.Active && assignmentApplies(as, student) {
				return s, true
			}
		}
	}
	return model.Session{}, false
}
func (a *Service) AccountState(loginToken, role string) (any, error) {
	s, ok := a.CurrentAccount(loginToken, role)
	if !ok {
		return nil, problem(Unauthenticated, "Войдите в личный кабинет.")
	}
	if role == "teacher" {
		return map[string]any{"teacher": PublicAccount(s)}, nil
	}
	db := a.repo.Snapshot()
	list := []map[string]any{}
	for _, as := range db.Assignments {
		var session *model.Session
		for _, rec := range db.Sessions {
			if rec.Session.StudentID == s.ID && rec.Session.AssignmentID == as.ID {
				x := rec.Session
				session = &x
				break
			}
		}
		if !assignmentApplies(as, s) && session == nil {
			continue
		}
		entry := map[string]any{"id": as.ID, "title": as.Title, "created": as.Created, "active": as.Active, "completed": false, "status": "Не начато"}
		if session != nil {
			entry["completed"] = session.Completed
			entry["sessionId"] = session.ID
			if session.Completed {
				entry["status"] = "Завершено"
				entry["finishedAt"] = session.Updated
			} else {
				entry["status"] = "В процессе"
				entry["answered"] = len(session.Answers)
				entry["total"] = len(a.itemsFor(*session))
			}
		}
		list = append(list, entry)
	}
	return map[string]any{"student": PublicAccount(s), "assignments": list}, nil
}
