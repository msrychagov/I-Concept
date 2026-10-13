package usecase

func missingForm() error { return problem(NotFound, "Анкета не найдена.") }

// FormDescription returns the questionnaire as the teacher may preview it.
func (a *Service) FormDescription(id string) (any, error) {
	form, ok := a.forms[id]
	if !ok {
		return nil, missingForm()
	}
	return clone(form), nil
}

// FormOverview lists students, the assignments of this form and brief
// submission records without answers.
func (a *Service) FormOverview(id string) (any, error) {
	form, ok := a.forms[id]
	if !ok {
		return nil, missingForm()
	}
	db := a.repo.Snapshot()
	students := []map[string]any{}
	for _, account := range db.Students {
		if account.Role == "student" {
			students = append(students, studentPublic(account))
		}
	}
	assignments := []map[string]any{}
	for _, as := range db.Assignments {
		if as.FormID == id {
			assignments = append(assignments, map[string]any{"id": as.ID, "title": as.Title, "platoon": as.Platoon, "active": as.Active, "created": as.Created})
		}
	}
	submissions := []map[string]any{}
	for _, submission := range db.Submissions {
		if submission.FormID != id {
			continue
		}
		progress := formProgress(form, submission)
		submissions = append(submissions, map[string]any{"id": submission.ID, "studentId": submission.StudentID, "assignmentId": submission.AssignmentID, "completed": submission.Completed, "completedAt": submission.CompletedAt, "updated": submission.Updated, "answered": len(progress.Complete), "total": progress.Total})
	}
	return map[string]any{"students": students, "assignments": assignments, "submissions": submissions}, nil
}

// FormSubmissionDetail returns one submission with answers, submitted or not.
func (a *Service) FormSubmissionDetail(id string) (any, error) {
	db := a.repo.Snapshot()
	for _, submission := range db.Submissions {
		form, known := a.forms[submission.FormID]
		if submission.ID != id || !known {
			continue
		}
		var student map[string]any
		for _, account := range db.Students {
			if account.ID == submission.StudentID {
				student = studentPublic(account)
			}
		}
		return map[string]any{"submission": submission, "form": form, "student": student, "progress": formProgress(form, submission)}, nil
	}
	return nil, problem(NotFound, "Анкета студента не найдена.")
}
