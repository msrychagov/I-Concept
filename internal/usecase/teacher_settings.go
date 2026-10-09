package usecase

import (
	"crypto/subtle"
	"net/mail"
	"strings"
	"time"
	"vucdiagnostic/internal/domain/model"
)

func (a *Service) UpdateTeacherAccount(loginToken string, b Input, op string) (any, error) {
	account, ok := a.CurrentAccount(loginToken, "teacher")
	if !ok {
		return nil, failure(Unauthenticated, map[string]any{"error": "Войдите в кабинет преподавателя."})
	}
	currentPassword := str(b, "currentPassword")
	if len(currentPassword) > 256 {
		return nil, failure(Invalid, map[string]any{"error": "Проверьте текущий пароль."})
	}
	return a.transaction(func(db *model.Database) (any, error) {
		// Recheck the session under the same lock as the account update.
		valid := false
		for _, login := range db.Logins {
			if login.StudentID == account.ID && login.Hash == hash(loginToken) && time.Now().Before(login.Expires) {
				valid = true
				break
			}
		}
		if !valid {
			return nil, failure(Unauthenticated, map[string]any{"error": "Войдите в кабинет преподавателя заново."})
		}
		for i, existing := range db.Students {
			if existing.ID != account.ID || existing.Role != "teacher" {
				continue
			}
			updated := existing
			if op == "update_profile" {
				name := strings.TrimSpace(str(b, "name"))
				email := strings.ToLower(strings.TrimSpace(str(b, "email")))
				address, err := mail.ParseAddress(email)
				if len(strings.Fields(name)) < 2 || len(name) > 200 || len(email) > 254 || err != nil || address.Address != email {
					return nil, failure(Invalid, map[string]any{"error": "Укажите имя и фамилию (до 200 символов) и корректную электронную почту."})
				}
				for _, other := range db.Students {
					if other.Role == "teacher" && other.ID != existing.ID && other.Email == email {
						return nil, failure(Conflict, map[string]any{"error": "Эта почта уже используется другим преподавателем."})
					}
				}
				updated.Name = name
				updated.Email = email
			}
			if op == "change_password" || updated.Email != existing.Email {
				candidate := passwordHash(currentPassword, existing.Salt)
				if currentPassword == "" || subtle.ConstantTimeCompare([]byte(candidate), []byte(existing.PasswordHash)) != 1 {
					return nil, failure(Invalid, map[string]any{"error": "Неверный текущий пароль."})
				}
			}
			oldLogins := db.Logins
			if op == "change_password" {
				password := str(b, "newPassword")
				if len(password) < 10 || len(password) > 256 || password != str(b, "confirmPassword") {
					return nil, failure(Invalid, map[string]any{"error": "Новый пароль должен содержать от 10 до 256 символов и совпадать с подтверждением."})
				}
				updated.Salt = token()
				updated.PasswordHash = passwordHash(password, updated.Salt)
				// Keep this browser signed in; revoke the teacher's other sessions.
				kept := make([]model.Login, 0, len(oldLogins))
				for _, login := range oldLogins {
					if login.StudentID != existing.ID || login.Hash == hash(loginToken) {
						kept = append(kept, login)
					}
				}
				db.Logins = kept
			}
			db.Students[i] = updated

			return map[string]any{"teacher": studentPublic(updated)}, nil
		}
		return nil, failure(NotFound, map[string]any{"error": "Учётная запись не найдена."})
	})
}
