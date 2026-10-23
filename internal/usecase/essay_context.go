package usecase

import "vucdiagnostic/internal/domain/model"

// essayContext is what a teacher sees next to an essay besides its answers: the
// title of the assignment and the earlier essay of the same author, even when
// another teacher gave it (decided 23 October 2026).
func essayContext(db model.Database, submission model.FormSubmission) map[string]any {
	context := map[string]any{}
	for _, as := range db.Assignments {
		if as.ID == submission.AssignmentID {
			context["assignmentTitle"] = as.Title
		}
	}
	if !submission.Completed {
		return context
	}
	if previous, ok := previousEssay(db, submission); ok {
		title := ""
		for _, as := range db.Assignments {
			if as.ID == previous.AssignmentID {
				title = as.Title
			}
		}
		context["previous"] = map[string]any{"id": previous.ID, "assignmentTitle": title, "completedAt": previous.CompletedAt, "answers": previous.Answers, "savedSections": previous.SavedSections}
	}
	return context
}
