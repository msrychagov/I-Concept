package usecase

import (
	"io"
	"vucdiagnostic/internal/domain/model"
)

func (a *Service) ResearchQuery(op, id, group string) (any, error) {
	if op == "teaching" {
		return a.TeachingState()
	}
	db := a.repo.Snapshot()
	switch op {
	case "state":
		for n, p := range db.Profiles {
			if p.Questions == nil {
				db.Profiles[n].Questions = clone(a.engineFor(model.Session{Profile: p, BankVersion: p.BankVersion}).bank.Items)
			}
			db.Profiles[n].Questions = studentQuestions(db.Profiles[n].Questions)
		}
		summaries := []map[string]any{}
		for _, x := range db.Sessions {
			s := x.Session
			if s.Kind == "observer" {
				continue
			}
			summaries = append(summaries, map[string]any{"id": s.ID, "kind": s.Kind, "participant": s.Participant, "group": s.Group, "wave": s.Wave, "observer": s.Observer, "parent": s.Parent, "created": s.Created, "updated": s.Updated, "completed": s.Completed, "revoked": s.Revoked, "revision": s.Revision, "answered": len(s.Answers), "total": len(a.itemsFor(s)), "profileName": s.Profile.Name, "profileId": s.Profile.ID})
		}
		return map[string]any{"profiles": db.Profiles, "sessions": summaries, "bank": a.Bank(), "emptyProfile": a.emptyProfile()}, nil
	case "report":
		s, ok := a.sessionByID(id)
		if !ok || s.Kind == "observer" {
			return nil, failure(NotFound, map[string]any{"error": "Сессия не найдена."})
		}
		return map[string]any{"session": s, "report": a.analyze(s, db.Sessions), "items": a.itemsFor(s)}, nil
	case "export":
		s, ok := a.sessionByID(id)
		if !ok || s.Kind == "observer" {
			return nil, failure(NotFound, map[string]any{"error": "Сессия не найдена."})
		}
		return s, nil
	case "group":
		return a.groupReport(db.Sessions, group)
	default:
		return nil, failure(NotFound, map[string]any{"error": "Неизвестный запрос."})
	}
}
func (a *Service) Survey(loginToken, id string) (any, error) {
	s, ok := a.AssignedSession(loginToken, id)
	if !ok {
		return nil, problem(Forbidden, "Ссылка недействительна или отозвана.")
	}
	pub := clone(s)
	pub.Profile = model.Profile{}
	return map[string]any{"session": pub, "items": a.publicItems(s), "modules": a.Bank().Modules, "version": s.BankVersion, "instructions": a.engineFor(s).bank.Instructions}, nil
}
func (a *Service) ExportCSV(w io.Writer, group string) error {
	list := filterRecords(a.repo.Snapshot().Sessions, func(s model.Session) bool { return s.Kind != "observer" })
	if group != "" {
		list = filterRecords(list, func(s model.Session) bool { return s.Group == group })
	}
	return a.writeCSV(w, list)
}
