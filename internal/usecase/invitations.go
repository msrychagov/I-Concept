package usecase

import (
	"net/mail"
	"strings"
	"time"
	"vucdiagnostic/internal/domain/model"
)

const teacherInvitationLifetime = 24 * time.Hour
const studentLinkLifetime = 48 * time.Hour

// CreateTeacherInvitation is an administrator-only grant for one teacher.
// The HTTP and CLI adapters authenticate the administrator.
func (a *Service) CreateTeacherInvitation(b Input) (any, error) {
	vuc := strings.Join(strings.Fields(str(b, "vuc")), " ")
	if vuc == "" || len(vuc) > 200 {
		return nil, problem(Invalid, "Укажите ВУЦ преподавателя (до 200 символов).")
	}
	email := strings.ToLower(strings.TrimSpace(str(b, "email")))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 {
		return nil, problem(Invalid, "Укажите корректную электронную почту.")
	}
	code := newInviteCode()
	now := time.Now().UTC()
	expires := now.Add(teacherInvitationLifetime)
	err = a.repo.Update(func(db *model.Database) error {
		center, exists := vucIn(*db, vuc)
		if !exists {
			return problem(NotFound, "ВУЦ не существует. Сначала администратор должен создать его.")
		}
		vuc = center.Name
		for _, account := range db.Students {
			if account.Email == email && account.Role == "teacher" {
				return problem(Conflict, "Преподаватель с этой почтой уже существует.")
			}
		}
		for i := range db.Invitations {
			inv := &db.Invitations[i]
			if inv.Role == "teacher" && inv.Email == email && inv.UsedAt.IsZero() {
				inv.Expires = now
			}
		}
		db.Invitations = append(db.Invitations, model.Invitation{VUC: vuc, Hash: hash(normalizeInviteCode(code)), Role: "teacher", Email: email, Created: now, Expires: expires})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"code": code, "role": "teacher", "email": email, "vuc": vuc, "expires": expires}, nil
}

// CreateStudentRegistrationLink creates a reusable 48-hour link for one
// teacher and platoon. Reissuing invalidates the previous link for that platoon.
func (a *Service) CreateStudentRegistrationLink(teacherID string, b Input) (any, error) {
	platoonID := str(b, "platoonId")
	if teacherID == "" || platoonID == "" {
		return nil, problem(Invalid, "Укажите взвод студента.")
	}
	token := newStudentInviteToken()
	now := time.Now().UTC()
	expires := now.Add(studentLinkLifetime)
	var platoon model.Platoon
	err := a.repo.Update(func(db *model.Database) error {
		teacher, err := managementTeacher(*db, teacherID)
		if err != nil {
			return err
		}
		var found bool
		platoon, found = platoonIn(*db, platoonID, teacher.VUC)
		if !found {
			return problem(NotFound, "Взвод не найден в вашем ВУЦ.")
		}
		for i := range db.Invitations {
			inv := &db.Invitations[i]
			if inv.Role == "student" && inv.PlatoonID == platoon.ID && now.Before(inv.Expires) {
				inv.Expires = now
			}
		}
		db.Invitations = append(db.Invitations, model.Invitation{VUC: platoon.VUC, PlatoonID: platoon.ID, Hash: hash(token), Role: "student", Platoon: platoon.Name, IssuerID: teacherID, Created: now, Expires: expires})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"link": "/student?invite=" + token, "role": "student", "platoon": platoon.Name, "platoonId": platoon.ID, "vuc": platoon.VUC, "expires": expires}, nil
}

func validTeacherInvitation(inv model.Invitation, email, code string, now time.Time) bool {
	return inv.Hash == hash(code) && inv.Role == "teacher" && inv.Email == email && inv.UsedAt.IsZero() && now.Before(inv.Expires)
}

func validStudentLink(inv model.Invitation, token string, now time.Time) bool {
	return inv.Hash == hash(token) && inv.Role == "student" && inv.IssuerID != "" && inv.Platoon != "" && now.Before(inv.Expires)
}
