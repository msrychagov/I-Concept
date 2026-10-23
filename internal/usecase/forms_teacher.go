package usecase

import "vucdiagnostic/internal/domain/model"

func missingForm() error { return problem(NotFound, "Анкета не найдена.") }

// FormDescription returns the questionnaire as the teacher may preview it.
func (a *Service) FormDescription(id string) (any, error) {
	form, ok := a.forms[id]
	if !ok {
		return nil, missingForm()
	}
	return clone(form), nil
}

// teacherOwnsSubmission says whether the submission belongs to an assignment
// of the teacher; an empty teacherID sees everything.
func teacherOwnsSubmission(db model.Database, teacherID string, submission model.FormSubmission) bool {
	for _, assignment := range db.Assignments {
		if assignment.ID == submission.AssignmentID && inTeacherScope(assignment.OwnerID, teacherID) {
			return true
		}
	}
	return false
}

// FormOverview lists students, the assignments of this form and brief
// submission records without answers.
func (a *Service) FormOverview(id string) (any, error) {
	return a.FormOverviewFor("", id)
}

func (a *Service) FormOverviewFor(teacherID, id string) (any, error) {
	form, ok := a.forms[id]
	if !ok {
		return nil, missingForm()
	}
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	students := []map[string]any{}
	for _, account := range db.Students {
		if account.Role == "student" && inTeacherScope(account.OwnerID, teacherID) {
			students = append(students, studentPublic(account))
		}
	}
	assignments := []map[string]any{}
	for _, as := range db.Assignments {
		if as.FormID == id && inTeacherScope(as.OwnerID, teacherID) {
			assignments = append(assignments, map[string]any{"id": as.ID, "title": as.Title, "platoon": as.Platoon, "active": as.Active, "created": as.Created})
		}
	}
	submissions := []map[string]any{}
	for _, submission := range db.Submissions {
		if submission.FormID != id {
			continue
		}
		if !teacherOwnsSubmission(db, teacherID, submission) {
			continue
		}
		progress := formProgress(form, submission)
		submissions = append(submissions, map[string]any{"id": submission.ID, "studentId": submission.StudentID, "assignmentId": submission.AssignmentID, "completed": submission.Completed, "completedAt": submission.CompletedAt, "updated": submission.Updated, "answered": len(progress.Complete), "total": progress.Total})
	}
	return map[string]any{"students": students, "assignments": assignments, "submissions": submissions}, nil
}

// FormSubmissionDetail returns one submission with answers, submitted or not.
// A submitted autobiography also carries the potential levels.
func (a *Service) FormSubmissionDetail(id string) (any, error) {
	return a.FormSubmissionDetailFor("", id)
}

func (a *Service) FormSubmissionDetailFor(teacherID, id string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	for _, submission := range db.Submissions {
		form, known := a.forms[submission.FormID]
		if submission.ID != id || !known {
			continue
		}
		if !teacherOwnsSubmission(db, teacherID, submission) {
			continue
		}
		var student map[string]any
		for _, account := range db.Students {
			if account.ID == submission.StudentID {
				student = studentPublic(account)
			}
		}
		result := map[string]any{"submission": submission, "form": form, "student": student, "progress": formProgress(form, submission)}
		if submission.FormID == autobiographyForm && submission.Completed {
			result["potential"] = autobiographyPotential(form, submission.Answers)
		}
		if submission.FormID == essayForm {
			result["essay"] = essayContext(db, submission)
		}
		return result, nil
	}
	return nil, problem(NotFound, "Анкета студента не найдена.")
}
