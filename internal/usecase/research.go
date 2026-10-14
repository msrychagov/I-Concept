package usecase

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"vucdiagnostic/internal/domain/model"
)

func (a *Service) ResearchCommand(b Input, op string) (any, error) {
	return a.ResearchCommandFor("", b, op)
}

func (a *Service) ResearchCommandFor(teacherID string, b Input, op string) (any, error) {
	if op == "invite" || op == "rotate" {
		return nil, failure(Gone, map[string]any{"error": "Индивидуальные ссылки заменены назначениями взводам."})
	}
	switch op {
	case "profile", "profile_update":
		var p model.Profile
		if err := json.Unmarshal(b["profile"], &p); err != nil {
			return nil, failure(Invalid, map[string]any{"error": "Некорректный шаблон."})
		}
		p.Questions = studentQuestions(p.Questions)
		// В текущей методике ориентир каждого включённого показателя фиксирован: 5.
		selected := p.Values
		p.Values = map[string]*float64{}
		for _, indicator := range a.bank.Indicators {
			if _, enabled := selected[indicator.ID]; enabled {
				v := 5.0
				p.Values[indicator.ID] = &v
			}
		}
		p.Basis = ""
		p.Approved = true
		p.Group = ""
		if err := a.validateQuestions(p); err != nil {
			return nil, failure(Invalid, map[string]any{"error": err.Error()})
		}
		if err := a.validateProfile(p); err != nil {
			return nil, failure(Invalid, map[string]any{"error": err.Error()})
		}
		if op == "profile_update" {
			oldFound := false
			for _, old := range a.repo.Snapshot().Profiles {
				if old.ID == p.ID && inTeacherScope(old.OwnerID, teacherID) {
					p.Created = old.Created
					p.BankVersion = old.BankVersion
					p.OwnerID = old.OwnerID
					oldFound = true
					break
				}
			}
			if !oldFound {
				return nil, failure(NotFound, map[string]any{"error": "Шаблон не найден."})
			}
			if err := a.updateProfile(p); err != nil {
				return nil, err
			}
		} else {
			p.ID = uuid()
			p.Created = time.Now().UTC().Format(time.RFC3339)
			p.BankVersion = a.bank.Version
			p.OwnerID = teacherID
			if err := a.addProfile(p); err != nil {
				return nil, err
			}
		}
		return p, nil
	case "rotate", "revoke", "reopen", "plan":
		id := str(b, "id")
		db := a.repo.Snapshot()
		session, ok := a.sessionByID(id)
		if !ok || !teacherOwnsSession(db, teacherID, session) {
			return nil, problem(NotFound, "Сессия не найдена.")
		}
		var link string
		err := a.mutateSession(id, func(rec *model.SessionRecord) error {
			switch op {
			case "revoke":
				rec.Session.Revoked = true
			case "rotate":
				t := token()
				rec.TokenHash = hash(t)
				rec.Session.Revoked = false
				link = "/survey#key=" + t
			case "reopen":
				rec.Session.Completed = false
				rec.Session.Revision++
			case "plan":
				var p []model.PlanItem
				if err := json.Unmarshal(b["plan"], &p); err != nil || len(p) > 100 {
					return errors.New("некорректный план")
				}
				for _, x := range p {
					if strings.TrimSpace(x.Action) == "" || len(x.Action) > 2000 || len(x.Owner) > 150 || len(x.Note) > 2000 || !contains([]string{"planned", "doing", "done"}, x.Status) {
						return errors.New("проверьте поля плана")
					}
				}
				rec.Session.Plan = p
				rec.Session.Revision++
			}
			rec.Session.Updated = time.Now().UTC().Format(time.RFC3339)
			return nil
		})
		if err != nil {
			return nil, failure(Invalid, map[string]any{"error": err.Error()})
		}
		if link != "" {
			return map[string]any{"link": link}, nil
		} else {
			return map[string]any{"ok": true}, nil
		}
	case "import":
		var s model.Session
		if err := json.Unmarshal(b["session"], &s); err != nil || s.ID == "" || s.Kind == "observer" || !a.supportsVersion(s.BankVersion) {
			return nil, failure(Invalid, map[string]any{"error": "Файл сессии несовместим."})
		}
		if _, ok := a.sessionByID(s.ID); ok {
			return nil, failure(Conflict, map[string]any{"error": "Сессия с таким ID уже существует."})
		}
		s.Revoked = true
		s.Revision = 0
		s.TeacherID = teacherID
		if s.Answers == nil {
			s.Answers = map[string]int{}
		}
		if s.Notes == nil {
			s.Notes = map[string]string{}
		}
		if err := a.addSession(model.SessionRecord{TokenHash: hash(token()), Session: s}); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	default:
		return nil, failure(NotFound, map[string]any{"error": "Неизвестное действие."})
	}
}
