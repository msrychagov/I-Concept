package usecase

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"
	"vucdiagnostic/internal/domain/model"
)

const maxSuggestions = 10

// FormSuggestions offers values that other students already entered into a
// text field marked for suggestions.
func (a *Service) FormSuggestions(loginToken, id, fieldID, query string) (any, error) {
	db := a.repo.Snapshot()
	own, form, err := a.studentForm(&db, loginToken, id)
	if err != nil {
		return nil, err
	}
	if !hasSuggestions(form, fieldID) {
		return nil, problem(Invalid, "Для этого поля подсказок нет.")
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if utf8.RuneCountInString(query) > 100 {
		return nil, problem(Invalid, "Слишком длинный запрос.")
	}
	counts := map[string]int{}
	spelling := map[string]string{}
	for _, other := range db.Submissions {
		if other.FormID != form.ID || other.ID == own.ID {
			continue
		}
		for _, value := range fieldValues(form, other.Answers, fieldID) {
			key := strings.ToLower(value)
			if strings.Contains(key, query) {
				counts[key]++
				if _, seen := spelling[key]; !seen {
					spelling[key] = value
				}
			}
		}
	}
	keys := sortedKeys(counts)
	slices.SortStableFunc(keys, func(x, y string) int { return counts[y] - counts[x] })
	values := []string{}
	for _, key := range keys[:min(len(keys), maxSuggestions)] {
		values = append(values, spelling[key])
	}
	return map[string]any{"values": values}, nil
}

func hasSuggestions(form model.Form, fieldID string) bool {
	found := false
	eachField(form, func(field model.FormField, _ bool) {
		found = found || (field.ID == fieldID && field.Suggest)
	})
	return found
}

// eachField visits every field of the form; nested is true inside a group.
func eachField(form model.Form, visit func(field model.FormField, nested bool)) {
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			visit(field, false)
			for _, inner := range field.Fields {
				visit(inner, true)
			}
		}
	}
}

// fieldValues collects the stored text values of a field, top-level or inside group records.
func fieldValues(form model.Form, answers map[string]json.RawMessage, fieldID string) []string {
	values := []string{}
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			if field.ID == fieldID {
				values = append(values, decodeList(answers[field.ID])...)
			}
			if field.Type != "group" || !slices.ContainsFunc(field.Fields, func(f model.FormField) bool { return f.ID == fieldID }) {
				continue
			}
			var records []map[string]json.RawMessage
			_ = json.Unmarshal(answers[field.ID], &records)
			for _, record := range records {
				values = append(values, decodeList(record[fieldID])...)
			}
		}
	}
	return values
}
