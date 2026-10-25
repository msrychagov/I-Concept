package usecase

import (
	"time"
	"vucdiagnostic/internal/domain/model"
)

// The input of the language model for the analysis of one essay
// (Вики/Эссе выпускника/Инструкция.md): the answers by question with the
// number and the block, the earlier essay of the same author when there is one,
// and the list of the instructors of the center with their labels. The name,
// platoon, cohort and dates of the author do not go. The answers go as the
// author wrote them; only the personal data of portraits.go is cut out, and
// the patronymic is not (essayTask.Keep).

// essayRows are the answers of a submission in the order of the form. A
// question left blank is not sent.
func essayRows(form model.Form, submission model.FormSubmission) []any {
	rows := []any{}
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			text := formAnswers(submission.Answers).text(field.ID)
			if text == "" {
				continue
			}
			row := newObject()
			row.set("номер", field.Number)
			row.set("блок", section.Title)
			row.set("вопрос", field.Label)
			row.set("ответ", text)
			rows = append(rows, row)
		}
	}
	return rows
}

// completedAt reads the time a submission was handed in; zero when unknown.
func completedAt(submission model.FormSubmission) time.Time {
	at, _ := time.Parse(time.RFC3339, submission.CompletedAt)
	return at
}

// previousEssay is the last essay of the same student handed in before this
// one, from any assignment: the author's earlier essay for the comparison.
func previousEssay(db model.Database, submission model.FormSubmission) (model.FormSubmission, bool) {
	now := completedAt(submission)
	var previous model.FormSubmission
	found := false
	for _, other := range db.Submissions {
		if other.FormID != essayForm || other.StudentID != submission.StudentID || !other.Completed || other.ID == submission.ID {
			continue
		}
		at := completedAt(other)
		if at.IsZero() || !at.Before(now) {
			continue
		}
		if !found || at.After(completedAt(previous)) {
			previous, found = other, true
		}
	}
	return previous, found
}

// studentOf finds the account that wrote a submission.
func studentOf(db model.Database, submission model.FormSubmission) (model.Account, bool) {
	for _, account := range db.Students {
		if account.ID == submission.StudentID {
			return account, true
		}
	}
	return model.Account{}, false
}

// essayPayload is the model input of a submitted essay. The input of v1 has
// the list of the instructors of the center with labels; the input of v2 has
// none: instructors are told by their positions (decided 25 October 2026).
func (a *Service) essayPayload(db model.Database, submission model.FormSubmission, list bool) (*object, error) {
	form, known := a.forms[essayForm]
	if submission.FormID != essayForm || !submission.Completed || !known {
		return nil, problem(Invalid, "Сведения для модели собираются только по сданному эссе.")
	}
	payload := newObject()
	payload.set("эссе", essayRows(form, submission))
	if previous, ok := previousEssay(db, submission); ok {
		payload.set("прежнее_эссе", essayRows(form, previous))
	}
	if student, ok := studentOf(db, submission); ok && list {
		if labels := instructorLabels(db, student.VUC); len(labels) > 0 {
			list := []any{}
			for _, label := range labels {
				entry := newObject()
				entry.set("метка", label.Label)
				entry.set("кто", label.Who)
				list = append(list, entry)
			}
			payload.set("преподаватели", list)
		}
	}
	return payload, nil
}
