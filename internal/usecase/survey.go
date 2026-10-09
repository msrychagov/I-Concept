package usecase

import (
	"strings"
	"time"
)

func (a *Service) SurveyCommand(loginToken, sessionID string, b Input, op string) (any, error) {
	s, ok := a.AssignedSession(loginToken, sessionID)
	if !ok {
		return nil, failure(Forbidden, map[string]any{"error": "Ссылка недействительна."})
	}
	rev := integer(b, "revision")
	if rev != s.Revision {
		return nil, failure(Conflict, map[string]any{"error": "Данные изменились в другом окне. Обновите страницу.", "conflict": true})
	}
	if op == "consent" {
		if !boolean(b, "value") {
			return nil, failure(Invalid, map[string]any{"error": "Для начала нужно согласие."})
		}
		s.Consent = true
		s.ConsentAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		if !s.Consent {
			return nil, failure(Invalid, map[string]any{"error": "Сначала подтвердите согласие."})
		}
		if s.Completed {
			return nil, failure(Conflict, map[string]any{"error": "Сессия завершена."})
		}
		if op == "answer" {
			id := str(b, "item")
			var value *int
			if raw, ok := b["value"]; ok && string(raw) != "null" {
				v := integer(b, "value")
				value = &v
			}
			if err := a.validateAnswer(s, id, value); err != nil {
				return nil, failure(Invalid, map[string]any{"error": err.Error()})
			}
			note := str(b, "note")
			if len(note) > 4000 {
				return nil, failure(Invalid, map[string]any{"error": "Заметка слишком длинная."})
			}
			if value == nil {
				delete(s.Answers, id)
			} else {
				s.Answers[id] = *value
			}
			if strings.TrimSpace(note) == "" {
				delete(s.Notes, id)
			} else {
				s.Notes[id] = note
			}
		} else {
			if len(s.Answers) != len(a.itemsFor(s)) && !boolean(b, "allowPartial") {
				return nil, failure(Incomplete, map[string]any{"error": "Есть пропуски. Ответы сохранены.", "partial": true})
			}
			s.Completed = true
			s.CompletedPartial = len(s.Answers) != len(a.itemsFor(s))
		}
	}
	s.Updated = time.Now().UTC().Format(time.RFC3339)
	saved, err := a.updateSession(s, rev)
	if err != nil {
		kind := Invalid
		if err.Error() == "CONFLICT" {
			kind = Conflict
		}
		return nil, failure(kind, map[string]any{"error": err.Error()})
	}
	return map[string]any{"revision": saved.Revision, "updated": saved.Updated, "completed": saved.Completed}, nil
}
