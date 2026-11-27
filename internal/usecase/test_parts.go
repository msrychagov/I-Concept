package usecase

import (
	"slices"
	"vucdiagnostic/internal/domain/model"
)

// The test of pilot-2.7 is split into three parts (Вики/Тестирование/План
// разбивки теста на три части.md, approved on 27 November 2026). A student
// submits each part when he likes, in any order; the session is finished when
// all parts are submitted. A session of an older bank has no parts and is
// taken whole, as before.

// PartState is a part of the test of a session as the student and the teacher see it.
type PartState struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Modules   []string `json:"modules"`
	Answered  int      `json:"answered"`
	Total     int      `json:"total"`
	Submitted bool     `json:"submitted"`
}

// testParts are the parts of the bank of the session; nil when it has none,
// and when a part has no question of the session: a short test of an old
// template is taken whole, as before the parts.
func (a *Service) testParts(s model.Session) []model.BankPart {
	if s.Kind != "student" {
		return nil
	}
	parts := a.engineFor(s).bank.Parts
	if len(parts) == 0 {
		return nil
	}
	asked := map[string]bool{}
	for _, item := range a.itemsFor(s) {
		asked[partOf(parts, item.Module)] = true
	}
	for _, part := range parts {
		if !asked[part.ID] {
			return nil
		}
	}
	return parts
}

// partOf is the id of the part a module belongs to; empty when none.
func partOf(parts []model.BankPart, module string) string {
	for _, part := range parts {
		if slices.Contains(part.Modules, module) {
			return part.ID
		}
	}
	return ""
}

// partItems are the questions of the session in the part.
func (a *Service) partItems(s model.Session, part model.BankPart) []model.Item {
	out := []model.Item{}
	for _, item := range a.itemsFor(s) {
		if slices.Contains(part.Modules, item.Module) {
			out = append(out, item)
		}
	}
	return out
}

// partStates are the parts of the session with their progress; nil without parts.
func (a *Service) partStates(s model.Session) []PartState {
	parts := a.testParts(s)
	if len(parts) == 0 {
		return nil
	}
	out := []PartState{}
	for _, part := range parts {
		state := PartState{ID: part.ID, Name: part.Name, Modules: slices.Clone(part.Modules), Submitted: slices.Contains(s.Parts, part.ID)}
		for _, item := range a.partItems(s, part) {
			state.Total++
			if _, ok := s.Answers[item.ID]; ok {
				state.Answered++
			}
		}
		out = append(out, state)
	}
	return out
}

// submittedModules are the modules whose answers count: with parts, those of
// the submitted parts; nil without parts — then everything counts.
func (a *Service) submittedModules(s model.Session) map[string]bool {
	parts := a.testParts(s)
	if len(parts) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, part := range parts {
		if slices.Contains(s.Parts, part.ID) {
			for _, module := range part.Modules {
				out[module] = true
			}
		}
	}
	return out
}

// partsStarted is true when the session has parts and some, not all, are submitted.
func (a *Service) partsStarted(s model.Session) bool {
	return !s.Completed && len(s.Parts) > 0 && len(a.testParts(s)) > 0
}

// submitPart marks the part submitted; the session is completed with the last
// one. The questions of the part must be answered unless allowPartial. With
// no part named, all the parts not yet submitted are submitted at once: the
// test is finished whole, as before the parts.
func (a *Service) submitPart(s *model.Session, id string, allowPartial bool) (finished bool, err error) {
	states := a.partStates(*s)
	if id == "" {
		for _, state := range states {
			if !state.Submitted && state.Answered != state.Total && !allowPartial {
				return false, failure(Incomplete, map[string]any{"error": "Есть пропуски. Ответы сохранены.", "partial": true})
			}
		}
		for _, state := range states {
			if !state.Submitted {
				s.Parts = append(s.Parts, state.ID)
			}
		}
		s.Completed = true
		s.CompletedPartial = len(s.Answers) != len(a.itemsFor(*s))
		return true, nil
	}
	index := slices.IndexFunc(states, func(p PartState) bool { return p.ID == id })
	if index < 0 {
		return false, failure(Invalid, map[string]any{"error": "Выберите часть тестирования."})
	}
	part := states[index]
	if part.Submitted {
		return false, failure(Conflict, map[string]any{"error": "Эта часть уже сдана."})
	}
	if part.Answered != part.Total && !allowPartial {
		return false, failure(Incomplete, map[string]any{"error": "Есть пропуски. Ответы сохранены.", "partial": true})
	}
	s.Parts = append(s.Parts, id)
	for _, state := range states {
		if !slices.Contains(s.Parts, state.ID) {
			return false, nil
		}
	}
	s.Completed = true
	s.CompletedPartial = len(s.Answers) != len(a.itemsFor(*s))
	return true, nil
}

// partLocked is true when the question is in a part already submitted.
func (a *Service) partLocked(s model.Session, item string) bool {
	parts := a.testParts(s)
	if len(parts) == 0 {
		return false
	}
	for _, i := range a.itemsFor(s) {
		if i.ID == item {
			return slices.Contains(s.Parts, partOf(parts, i.Module))
		}
	}
	return false
}

// onlyModules is the session with the answers of the modules only.
func onlyModules(items []model.Item, s model.Session, modules map[string]bool) model.Session {
	out := clone(s)
	out.Answers, out.Notes = map[string]int{}, map[string]string{}
	for _, item := range items {
		if !modules[item.Module] {
			continue
		}
		if v, ok := s.Answers[item.ID]; ok {
			out.Answers[item.ID] = v
		}
		if note, ok := s.Notes[item.ID]; ok {
			out.Notes[item.ID] = note
		}
	}
	return out
}
