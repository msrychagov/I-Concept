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
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return model.Account{}, false
	}
	return accountIn(db, loginToken, role)
}
func accountIn(db model.Database, loginToken, role string) (model.Account, bool) {
	if loginToken == "" {
		return model.Account{}, false
	}
	for _, login := range db.Logins {
		if login.Hash == hash(loginToken) && time.Now().Before(login.Expires) {
			for _, account := range db.Students {
				if account.ID == login.StudentID && account.Role == role && !account.Disabled {
					return account, true
				}
			}
		}
	}
	return model.Account{}, false
}
func PublicAccount(account model.Account) map[string]any {
	return map[string]any{"id": account.ID, "name": account.Name, "email": account.Email, "platoon": account.Platoon, "platoonId": account.PlatoonID, "vuc": account.VUC, "disabled": account.Disabled}
}
func studentPublic(account model.Account) map[string]any { return PublicAccount(account) }
func assignmentOwnedByStudent(as model.Assignment, account model.Account) bool {
	return !account.Disabled && as.OwnerID == account.OwnerID
}
func assignmentApplies(as model.Assignment, account model.Account) bool {
	if !assignmentOwnedByStudent(as, account) {
		return false
	}
	if account.VUC != "" && account.PlatoonID == "" {
		return false
	}
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
	snapshot, err := a.repo.ReadSnapshot()
	if err != nil {
		return LoginResult{}, err
	}
	if op == "register" {
		name := strings.TrimSpace(str(b, "name"))
		credential := normalizeInviteCode(str(b, "invite"))
		if role == "student" {
			credential = normalizeStudentInvite(str(b, "invite"))
		}
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || len(strings.Fields(name)) < 2 || len(name) > 200 || credential == "" || len(password) < 10 {
			return LoginResult{}, problem(Invalid, "Проверьте ФИО, почту, приглашение и пароль длиной не менее 10 символов.")
		}
		var invitation model.Invitation
		now := time.Now().UTC()
		for _, candidate := range snapshot.Invitations {
			valid := role == "teacher" && validTeacherInvitation(candidate, email, credential, now) || role == "student" && validStudentLink(candidate, credential, now)
			if valid {
				invitation = candidate
				break
			}
		}
		if invitation.Hash == "" {
			return LoginResult{}, problem(Forbidden, "Приглашение недействительно или срок его действия истёк.")
		}
		salt := token()
		account = model.Account{ID: uuid(), Role: role, Name: name, Platoon: invitation.Platoon, Email: email, Salt: salt, PasswordHash: passwordHash(password, salt), Created: now.Format(time.RFC3339), OwnerID: invitation.IssuerID}
	} else if op == "login" {
		for _, other := range snapshot.Students {
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
	err = a.repo.Update(func(db *model.Database) error {
		if op == "register" {
			for _, other := range db.Students {
				if other.Email == email && other.Role == role {
					return problem(Conflict, "Эта почта уже зарегистрирована. Войдите в кабинет.")
				}
			}
			credential := normalizeInviteCode(str(b, "invite"))
			if role == "student" {
				credential = normalizeStudentInvite(str(b, "invite"))
			}
			now := time.Now().UTC()
			consumed := false
			for i := range db.Invitations {
				valid := role == "teacher" && validTeacherInvitation(db.Invitations[i], email, credential, now) || role == "student" && validStudentLink(db.Invitations[i], credential, now)
				if valid {
					inv := db.Invitations[i]
					if _, exists := vucIn(*db, inv.VUC); !exists {
						return problem(Forbidden, "ВУЦ приглашения не существует. Обратитесь к администратору.")
					}
					if role == "teacher" && inv.VUC == "" {
						return problem(Forbidden, "Запросите у администратора код с указанием ВУЦ.")
					}
					if role == "student" {
						p, ok := platoonIn(*db, inv.PlatoonID, inv.VUC)
						if !ok {
							return problem(Forbidden, "Приглашение недействительно. Запросите новую ссылку.")
						}
						account.Platoon = p.Name
					}
					account.VUC, account.PlatoonID = inv.VUC, inv.PlatoonID
					result.Account = account
					if role == "teacher" {
						db.Invitations[i].UsedAt = now
						db.Invitations[i].AccountID = account.ID
					} else {
						db.Invitations[i].UseCount++
					}
					consumed = true
					break
				}
			}
			if !consumed {
				return problem(Forbidden, "Приглашение недействительно или срок его действия истёк.")
			}
			db.Students = append(db.Students, account)
		} else {
			valid := false
			for _, current := range db.Students {
				if current.ID == account.ID && current.PasswordHash == account.PasswordHash && current.Email == email && !current.Disabled {
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
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return model.Session{}, false
	}
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
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	list := []map[string]any{}
	for _, as := range db.Assignments {
		if !assignmentOwnedByStudent(as, s) {
			continue
		}
		if as.FormID != "" {
			if entry, shown := a.formEntry(db, as, s); shown {
				list = append(list, entry)
			}
			continue
		}
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

// formEntry describes a questionnaire assignment in the student's cabinet.
func (a *Service) formEntry(db model.Database, as model.Assignment, student model.Account) (map[string]any, bool) {
	if !assignmentOwnedByStudent(as, student) {
		return nil, false
	}
	var submission *model.FormSubmission
	for n := range db.Submissions {
		if db.Submissions[n].StudentID == student.ID && db.Submissions[n].AssignmentID == as.ID {
			submission = &db.Submissions[n]
			break
		}
	}
	if !assignmentApplies(as, student) && submission == nil {
		return nil, false
	}
	entry := map[string]any{"id": as.ID, "title": as.Title, "created": as.Created, "active": as.Active, "completed": false, "status": "Не начато", "formId": as.FormID}
	if submission == nil {
		return entry, true
	}
	entry["completed"] = submission.Completed
	entry["sessionId"] = submission.ID
	if submission.Completed {
		entry["status"] = "Завершено"
		entry["finishedAt"] = submission.CompletedAt
		return entry, true
	}
	entry["status"] = "В процессе"
	if form, known := a.forms[submission.FormID]; known {
		progress := formProgress(form, *submission)
		entry["answered"] = len(progress.Complete)
		entry["total"] = progress.Total
	}
	return entry, true
}
