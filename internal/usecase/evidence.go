package usecase

import (
	"encoding/json"
	"sort"
	"vucdiagnostic/internal/domain/model"
)

type EvidenceChoice struct {
	ID             string `json:"id"`
	Text           string `json:"text"`
	Selected       string `json:"selected"`
	Interpretation string `json:"interpretation"`
}
type EvidenceTrend struct {
	Available         bool     `json:"available"`
	Reason            string   `json:"reason"`
	Previous          *float64 `json:"previous"`
	Delta             *float64 `json:"delta"`
	PreviousDate      string   `json:"previousDate"`
	PreviousSessionID string   `json:"previousSessionId"`
}
type EvidenceRow struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Block       string           `json:"block"`
	Definition  string           `json:"definition"`
	Limitations string           `json:"limitations"`
	SelfImage   *int             `json:"selfImage"`
	Statements  *float64         `json:"statements"`
	Coverage    int              `json:"coverage"`
	Total       int              `json:"total"`
	Choices     []EvidenceChoice `json:"choices"`
	Trend       EvidenceTrend    `json:"trend"`
}

func sameParticipant(a, b model.Session) bool {
	if a.StudentID != "" || b.StudentID != "" {
		return a.StudentID != "" && a.StudentID == b.StudentID
	}
	return a.Participant != "" && a.Participant == b.Participant && a.Group == b.Group
}
func measurementDate(s model.Session) string {
	if s.Completed && s.Updated != "" {
		return s.Updated
	}
	return s.Created
}
func (a *Service) comparable(left, right model.Session) bool {
	if left.BankVersion != right.BankVersion {
		return false
	}
	signature := func(s model.Session) string {
		ids := []string{}
		for id := range s.Profile.Values {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		data, _ := json.Marshal(struct {
			Items      []model.Item
			Indicators []string
		}{a.itemsFor(s), ids})
		return string(data)
	}
	return signature(left) == signature(right)
}
func (a *Service) evidence(s model.Session, all []model.SessionRecord, rows []reportRow) []EvidenceRow {
	result := []EvidenceRow{}
	var previous *model.Session
	for _, record := range all {
		x := record.Session
		if x.ID == s.ID || x.Kind != "student" || !x.Completed || !sameParticipant(s, x) || !a.comparable(s, x) || measurementDate(x) >= measurementDate(s) {
			continue
		}
		if previous == nil || measurementDate(x) > measurementDate(*previous) {
			copy := x
			previous = &copy
		}
	}
	for _, row := range rows {
		entry := EvidenceRow{ID: row.ID, Name: row.Name, Block: row.Block, Statements: row.Actual, Coverage: row.Coverage, Total: row.Total, Choices: []EvidenceChoice{}, Trend: EvidenceTrend{Reason: "Нет предыдущего завершённого прохождения с теми же вопросами, шкалами и версией банка."}}
		var ids []string
		for _, indicator := range a.bank.Indicators {
			if indicator.ID == row.ID {
				entry.Definition = indicator.Definition
				entry.Limitations = indicator.Limitations
				ids = indicator.Items
				break
			}
		}
		if value, ok := s.Answers[row.ID+"_R"]; ok {
			v := value
			entry.SelfImage = &v
		}
		for _, item := range a.itemsFor(s) {
			if item.Module != "S" || item.Indicator != row.ID {
				continue
			}
			if value, ok := s.Answers[item.ID]; ok && value >= 0 && value < len(item.Options) {
				option := item.Options[value]
				entry.Choices = append(entry.Choices, EvidenceChoice{item.ID, item.Text, option.Text, option.Rationale})
			}
		}
		if !s.Completed {
			entry.Trend.Reason = "Динамика рассчитывается после завершения текущего прохождения."
		} else if previous != nil {
			prev := a.mean(*previous, ids)
			if prev != nil && row.Actual != nil {
				delta := *row.Actual - *prev
				entry.Trend = EvidenceTrend{true, "Числовое изменение самоотчёта; значимость и причины программой не устанавливаются.", prev, &delta, measurementDate(*previous), previous.ID}
			} else {
				entry.Trend.Reason = "Для этого показателя недостаточно ответов в одном из сравниваемых прохождений."
			}
		}
		result = append(result, entry)
	}
	return result
}
