package usecase

func (a *Service) TeachingState() (any, error) {
	return a.TeachingStateFor("")
}

func (a *Service) TeachingStateFor(teacherID string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	students := []map[string]any{}
	for _, s := range db.Students {
		if s.Role != "student" || !inTeacherScope(s.OwnerID, teacherID) {
			continue
		}
		students = append(students, studentPublic(s))
	}
	assignments := []map[string]any{}
	for _, as := range db.Assignments {
		if as.FormID != "" || !inTeacherScope(as.OwnerID, teacherID) {
			continue
		}
		assignments = append(assignments, map[string]any{"id": as.ID, "title": as.Title, "platoon": as.Platoon, "active": as.Active, "created": as.Created, "profileName": as.Profile.Name})
	}
	sessions := []map[string]any{}
	for _, r := range db.Sessions {
		s := r.Session
		if s.StudentID == "" || !teacherOwnsSession(db, teacherID, s) {
			continue
		}
		sessions = append(sessions, map[string]any{"id": s.ID, "studentId": s.StudentID, "assignmentId": s.AssignmentID, "name": s.Participant, "platoon": s.Group, "title": s.Wave, "completed": s.Completed, "answered": len(s.Answers), "total": len(a.itemsFor(s)), "updated": s.Updated})
	}
	return map[string]any{"students": students, "assignments": assignments, "sessions": sessions}, nil
}
