package usecase

import "vucdiagnostic/internal/domain/model"

// NormalizeOwnership adopts legacy unowned data only when there is exactly one
// teacher, so the migration cannot guess and expose records in a multi-teacher
// installation. Ambiguous legacy records remain inaccessible through scoped APIs.
func (a *Service) NormalizeOwnership() error {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return err
	}
	if !normalizeOwnership(&db) {
		return nil
	}
	_, err = a.repo.ReplaceWithBackup(db)
	return err
}

func normalizeOwnership(db *model.Database) bool {
	teachers := []string{}
	for _, account := range db.Students {
		if account.Role == "teacher" {
			teachers = append(teachers, account.ID)
		}
	}
	if len(teachers) == 0 {
		return false
	}
	owner := teachers[0]
	ambiguous := len(teachers) > 1
	changed := false
	for i := range db.Students {
		if db.Students[i].Role == "student" && db.Students[i].OwnerID == "" {
			if ambiguous {
				db.Students[i].OwnerID = "legacy-unassigned-student:" + db.Students[i].ID
			} else {
				db.Students[i].OwnerID = owner
			}
			changed = true
		}
	}
	for i := range db.Assignments {
		if db.Assignments[i].OwnerID == "" {
			if ambiguous {
				db.Assignments[i].OwnerID = "legacy-unassigned"
			} else {
				db.Assignments[i].OwnerID = owner
			}
			changed = true
		}
	}
	for i := range db.Profiles {
		if db.Profiles[i].OwnerID == "" {
			if ambiguous {
				db.Profiles[i].OwnerID = "legacy-unassigned"
			} else {
				db.Profiles[i].OwnerID = owner
			}
			changed = true
		}
	}
	for i := range db.Sessions {
		if db.Sessions[i].Session.TeacherID == "" {
			if ambiguous {
				db.Sessions[i].Session.TeacherID = "legacy-unassigned"
			} else {
				db.Sessions[i].Session.TeacherID = owner
			}
			changed = true
		}
	}
	return changed
}

func inTeacherScope(ownerID, teacherID string) bool {
	return teacherID == "" || ownerID == teacherID
}

func teacherOwnsSession(db model.Database, teacherID string, session model.Session) bool {
	if teacherID == "" {
		return true
	}
	if session.TeacherID != "" {
		return session.TeacherID == teacherID
	}
	for _, assignment := range db.Assignments {
		if assignment.ID == session.AssignmentID {
			return assignment.OwnerID == teacherID
		}
	}
	return false
}
