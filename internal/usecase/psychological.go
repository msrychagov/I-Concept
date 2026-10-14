package usecase

import "vucdiagnostic/internal/domain/model"

// Psychological answers are descriptive records, never pedagogical scores,
// diagnostic conclusions, risk classifications or suitability recommendations.
func (a *Service) psychologicalReport(s model.Session) []map[string]any {
	result := []map[string]any{}
	for _, ind := range a.bank.Indicators {
		if !ind.Descriptive || !a.indicatorEnabled(s.Profile, ind.ID) {
			continue
		}
		answers := []map[string]any{}
		answered, declined, notApplicable := 0, 0, 0
		for _, item := range a.itemsFor(s) {
			if item.Indicator != ind.ID || !item.ResponseOnly {
				continue
			}
			selected, code := "Нет ответа", ""
			if raw, ok := s.Answers[item.ID]; ok && raw >= 0 && raw < len(item.Options) {
				option := item.Options[raw]
				selected, code = option.Text, option.ID
				switch code {
				case "SKIP":
					declined++
				case "NA":
					notApplicable++
				default:
					answered++
				}
			}
			answers = append(answers, map[string]any{"id": item.ID, "text": item.Text, "timeWindow": item.TimeWindow, "selected": selected, "code": code, "note": s.Notes[item.ID]})
		}
		if len(answers) == 0 {
			continue
		}
		result = append(result, map[string]any{"id": ind.ID, "name": ind.Name, "limitations": ind.Limitations, "answers": answers, "total": len(answers), "answered": answered, "declined": declined, "notApplicable": notApplicable, "unanswered": len(answers) - answered - declined - notApplicable})
	}
	return result
}
