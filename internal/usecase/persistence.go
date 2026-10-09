package usecase

import (
	"errors"
	"vucdiagnostic/internal/domain/model"
)

func (a *Service) transaction(fn func(*model.Database) (any, error)) (any, error) {
	var result any
	err := a.repo.Update(func(db *model.Database) error { var err error; result, err = fn(db); return err })
	return result, err
}
func (a *Service) sessionByID(id string) (model.Session, bool) {
	for _, r := range a.repo.Snapshot().Sessions {
		if r.Session.ID == id {
			return r.Session, true
		}
	}
	return model.Session{}, false
}
func (a *Service) updateSession(v model.Session, revision int) (model.Session, error) {
	err := a.repo.Update(func(db *model.Database) error {
		for i := range db.Sessions {
			if db.Sessions[i].Session.ID == v.ID {
				if db.Sessions[i].Session.Revision != revision {
					return problem(Conflict, "CONFLICT")
				}
				v.Revision = revision + 1
				db.Sessions[i].Session = clone(v)
				return nil
			}
		}
		return problem(NotFound, "session not found")
	})
	return v, err
}
func (a *Service) addProfile(p model.Profile) error {
	return a.repo.Update(func(db *model.Database) error { db.Profiles = append(db.Profiles, p); return nil })
}
func (a *Service) updateProfile(p model.Profile) error {
	return a.repo.Update(func(db *model.Database) error {
		for i := range db.Profiles {
			if db.Profiles[i].ID == p.ID {
				db.Profiles[i] = p
				return nil
			}
		}
		return problem(NotFound, "template not found")
	})
}
func (a *Service) addSession(r model.SessionRecord) error {
	return a.repo.Update(func(db *model.Database) error {
		for _, old := range db.Sessions {
			if old.Session.ID == r.Session.ID {
				return problem(Conflict, "Сессия с таким ID уже существует.")
			}
		}
		db.Sessions = append(db.Sessions, r)
		return nil
	})
}
func (a *Service) mutateSession(id string, fn func(*model.SessionRecord) error) error {
	return a.repo.Update(func(db *model.Database) error {
		for i := range db.Sessions {
			if db.Sessions[i].Session.ID == id {
				return fn(&db.Sessions[i])
			}
		}
		return errors.New("session not found")
	})
}
