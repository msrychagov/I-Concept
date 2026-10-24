package usecase

import (
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// The statuses of professional identity by А. А. Азбель in two blocks — the
// military speciality and the civil profession (Вики/Тестирование/Азбель —
// адаптация для ВУЦ.md, approved on 23 October 2026). The sum of every scale
// falls into a range of the key; the leading status is the scale in the
// highest range, all of them when several share it.

// IdentityStatus is one scale of a block. Score and Range are empty unless all
// the questions of the block are answered; Rank is the number of the range, 1–5.
type IdentityStatus struct {
	ID    string `json:"id"`
	Short string `json:"short"`
	Name  string `json:"name"`
	Score *int   `json:"score"`
	Max   int    `json:"max"`
	Range string `json:"range"`
	Rank  int    `json:"rank"`
}

// IdentityProfile is the profile of one block; Leading are the codes of the
// leading statuses, LeadingText their names.
type IdentityProfile struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Method      string           `json:"method"`
	Statuses    []IdentityStatus `json:"statuses"`
	Leading     []string         `json:"leading"`
	LeadingText string           `json:"leadingText"`
	Answered    int              `json:"answered"`
	Total       int              `json:"total"`
}

// identityMethod is that of the session's bank version; a session of an older
// template that got the methods added at assignment uses the current one.
func (a *Service) identityMethod(engine *Service) *model.IdentityMethod {
	if engine.bank.Identity != nil {
		return engine.bank.Identity
	}
	return a.bank.Identity
}

// listText joins names as in a sentence: «a, b и c».
func listText(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " и " + names[len(names)-1]
}

// identityProfiles counts the blocks the session was asked; nil when the
// method was not given.
func (a *Service) identityProfiles(engine *Service, s model.Session) []IdentityProfile {
	method := a.identityMethod(engine)
	if method == nil {
		return nil
	}
	asked := map[string]bool{}
	for _, item := range engine.itemsFor(s) {
		asked[item.ID] = true
	}
	profiles := []IdentityProfile{}
	for _, block := range method.Blocks {
		given := false
		for _, id := range block.Items {
			given = given || asked[id]
		}
		if !given {
			continue
		}
		p := IdentityProfile{ID: block.ID, Name: block.Name, Method: method.Method, Statuses: []IdentityStatus{}, Leading: []string{}, Total: len(block.Items)}
		sums, highest := map[string]int{}, map[string]int{}
		for _, id := range block.Items {
			item, option := answeredOption(engine, s, id)
			for _, scale := range method.Scales {
				top := 0
				for _, o := range item.Options {
					if o.Scale == scale.ID {
						top = max(top, int(o.Score))
					}
				}
				highest[scale.ID] += top
			}
			if option != nil {
				p.Answered++
				sums[option.Scale] += int(option.Score)
			}
		}
		complete := p.Answered == p.Total
		best := 0
		for _, scale := range method.Scales {
			status := IdentityStatus{ID: scale.ID, Short: scale.Short, Name: scale.Name, Max: highest[scale.ID]}
			if complete {
				sum := sums[scale.ID]
				status.Score = &sum
				for n, bound := range scale.Bounds {
					if sum >= bound && n < len(method.Ranges) {
						status.Rank, status.Range = n+1, method.Ranges[n]
					}
				}
				best = max(best, status.Rank)
			}
			p.Statuses = append(p.Statuses, status)
		}
		names := []string{}
		for _, status := range p.Statuses {
			if best > 0 && status.Rank == best {
				p.Leading = append(p.Leading, status.ID)
				names = append(names, status.Name)
			}
		}
		p.LeadingText = listText(names)
		if p.LeadingText == "" {
			p.LeadingText = "не определён"
		}
		profiles = append(profiles, p)
	}
	if len(profiles) == 0 {
		return nil
	}
	return profiles
}
