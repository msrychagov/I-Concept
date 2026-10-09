package usecase

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"vucdiagnostic/internal/domain/model"
)

const (
	defaultTextLength     = 300
	defaultTextareaLength = 2000
	earliestYear          = 1900
	futureYears           = 10
)

// FormProgress tells which sections are done and what is still missing.
// A section is done when it was saved at least once and nothing required is blank.
type FormProgress struct {
	Complete []string            `json:"complete"`
	Missing  map[string][]string `json:"missing"`
	Total    int                 `json:"total"`
}

func formProblem(path, label, text string) error {
	return failure(Invalid, map[string]any{"error": fmt.Sprintf("«%s»: %s", label, text), "field": path})
}

// cleanSection validates answers to the fields of one section and returns them
// in canonical form. Hidden fields are dropped; blank values are not stored.
// The draft may be incomplete: required fields are checked only on completion.
func cleanSection(form model.Form, section model.FormSection, input map[string]json.RawMessage, today time.Time) (map[string]json.RawMessage, error) {
	own := map[string]bool{}
	for _, field := range section.Fields {
		own[field.ID] = true
	}
	for _, key := range sortedKeys(input) {
		if own[key] {
			continue
		}
		return nil, failure(Invalid, map[string]any{"error": fmt.Sprintf("Поле «%s» не относится к этому разделу.", key), "field": key})
	}
	cleaned, _, err := cleanFields(form, section.Fields, input, nil, "", today)
	return cleaned, err
}

// cleanFields processes one level of fields in order. outer holds the visible
// values of the enclosing section when the level is a group record.
func cleanFields(form model.Form, fields []model.FormField, input map[string]json.RawMessage, outer map[string][]string, prefix string, today time.Time) (map[string]json.RawMessage, map[string][]string, error) {
	cleaned := map[string]json.RawMessage{}
	values := map[string][]string{}
	for _, field := range fields {
		raw, present := input[field.ID]
		if !present || string(raw) == "null" || !conditionHolds(field.ShowIf, values, outer) {
			continue
		}
		path := prefix + field.ID
		if field.Type == "group" {
			records, err := cleanRecords(form, field, raw, values, path, today)
			if err != nil {
				return nil, nil, err
			}
			if len(records) > 0 {
				data, _ := json.Marshal(records)
				cleaned[field.ID] = data
			}
			continue
		}
		data, list, err := cleanValue(form, field, raw, path, today)
		if err != nil {
			return nil, nil, err
		}
		if data != nil {
			cleaned[field.ID] = data
			values[field.ID] = list
		}
	}
	return cleaned, values, nil
}

func cleanRecords(form model.Form, field model.FormField, raw json.RawMessage, outer map[string][]string, path string, today time.Time) ([]map[string]json.RawMessage, error) {
	var records []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, formProblem(path, field.Label, "ожидается список записей.")
	}
	if len(records) > field.Max {
		return nil, formProblem(path, field.Label, fmt.Sprintf("записей не больше %d.", field.Max))
	}
	own := map[string]bool{}
	for _, nested := range field.Fields {
		own[nested.ID] = true
	}
	result := []map[string]json.RawMessage{}
	for n, record := range records {
		recordPath := fmt.Sprintf("%s.%d.", path, n)
		for _, key := range sortedKeys(record) {
			if !own[key] {
				return nil, formProblem(recordPath+key, field.Label, fmt.Sprintf("неизвестное поле «%s».", key))
			}
		}
		cleaned, _, err := cleanFields(form, field.Fields, record, outer, recordPath, today)
		if err != nil {
			return nil, err
		}
		result = append(result, cleaned)
	}
	return result, nil
}

// cleanValue checks a single non-group value. It returns nil data for a blank value.
func cleanValue(form model.Form, field model.FormField, raw json.RawMessage, path string, today time.Time) (json.RawMessage, []string, error) {
	bad := func(text string) error { return formProblem(path, field.Label, text) }
	switch field.Type {
	case "choice":
		var value string
		if json.Unmarshal(raw, &value) != nil {
			return nil, nil, bad("ожидается один вариант.")
		}
		if value == "" {
			return nil, nil, nil
		}
		if !hasOption(form, field, value) {
			return nil, nil, bad("такого варианта нет в списке.")
		}
		data, _ := json.Marshal(value)
		return data, []string{value}, nil
	case "multi":
		var picked []string
		if json.Unmarshal(raw, &picked) != nil {
			return nil, nil, bad("ожидается список вариантов.")
		}
		result := []string{}
		for _, value := range picked {
			if !hasOption(form, field, value) {
				return nil, nil, bad("такого варианта нет в списке.")
			}
			if !slices.Contains(result, value) {
				result = append(result, value)
			}
		}
		if len(result) == 0 {
			return nil, nil, nil
		}
		data, _ := json.Marshal(result)
		return data, result, nil
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return nil, nil, bad("ожидается строка.")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil, nil
	}
	if err := checkScalar(field, value, today); err != nil {
		return nil, nil, bad(err.Error())
	}
	data, _ := json.Marshal(value)
	return data, []string{value}, nil
}

func checkScalar(field model.FormField, value string, today time.Time) error {
	switch field.Type {
	case "text", "textarea":
		limit := field.MaxLength
		if limit == 0 {
			limit = defaultTextLength
			if field.Type == "textarea" {
				limit = defaultTextareaLength
			}
		}
		if utf8.RuneCountInString(value) > limit {
			return fmt.Errorf("слишком длинный текст, не больше %d знаков.", limit)
		}
		for _, r := range value {
			if unicode.IsControl(r) && !(field.Type == "textarea" && (r == '\n' || r == '\r' || r == '\t')) {
				return fmt.Errorf("недопустимые символы.")
			}
		}
	case "date":
		date, err := time.Parse("2006-01-02", value)
		if err != nil || date.Year() < earliestYear || date.After(today.AddDate(0, 0, 1)) {
			return fmt.Errorf("укажите существующую дату не позже сегодняшней (ГГГГ-ММ-ДД).")
		}
	case "month":
		month, err := time.Parse("2006-01", value)
		if err != nil || month.Year() < earliestYear || month.Year() > today.Year()+futureYears {
			return fmt.Errorf("укажите месяц и год (ГГГГ-ММ).")
		}
	case "year":
		year, err := time.Parse("2006", value)
		if err != nil || year.Year() < earliestYear || year.Year() > today.Year()+futureYears {
			return fmt.Errorf("укажите год (ГГГГ).")
		}
	}
	return nil
}

func hasOption(form model.Form, field model.FormField, id string) bool {
	return slices.ContainsFunc(form.Options(field), func(o model.FormOption) bool { return o.ID == id })
}

// conditionHolds reports whether a field with this condition is visible. The
// controlling field is looked up in the same record first, then in the section.
func conditionHolds(condition *model.FormCondition, record, section map[string][]string) bool {
	if condition == nil {
		return true
	}
	current, ok := record[condition.Field]
	if !ok {
		current, ok = section[condition.Field]
	}
	if !ok {
		return false
	}
	return slices.ContainsFunc(current, func(v string) bool { return slices.Contains(condition.In, v) })
}

// formMissing lists, per section, the required visible fields that are blank.
// Fields inside group records are named "group.N.field".
func formMissing(form model.Form, answers map[string]json.RawMessage) map[string][]string {
	result := map[string][]string{}
	for _, section := range form.Sections {
		values := map[string][]string{}
		missing := []string{}
		for _, field := range section.Fields {
			if !conditionHolds(field.ShowIf, values, nil) {
				continue
			}
			raw, present := answers[field.ID]
			if field.Type == "group" {
				var records []map[string]json.RawMessage
				if present {
					_ = json.Unmarshal(raw, &records)
				}
				need := field.Min
				if field.Required && need == 0 {
					need = 1
				}
				if len(records) < need {
					missing = append(missing, field.ID)
				}
				for n, record := range records {
					recordValues := map[string][]string{}
					for _, nested := range field.Fields {
						if !conditionHolds(nested.ShowIf, recordValues, values) {
							continue
						}
						value, ok := record[nested.ID]
						if !ok {
							if nested.Required {
								missing = append(missing, fmt.Sprintf("%s.%d.%s", field.ID, n, nested.ID))
							}
							continue
						}
						recordValues[nested.ID] = decodeList(value)
					}
				}
				continue
			}
			if !present {
				if field.Required {
					missing = append(missing, field.ID)
				}
				continue
			}
			values[field.ID] = decodeList(raw)
		}
		if len(missing) > 0 {
			result[section.ID] = missing
		}
	}
	return result
}

// decodeList reads a stored string or list of strings as a list.
func decodeList(raw json.RawMessage) []string {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}
	}
	var many []string
	_ = json.Unmarshal(raw, &many)
	return many
}

func formProgress(form model.Form, submission model.FormSubmission) FormProgress {
	missing := formMissing(form, submission.Answers)
	progress := FormProgress{Complete: []string{}, Missing: missing, Total: len(form.Sections)}
	for _, section := range form.Sections {
		if _, gaps := missing[section.ID]; !gaps && slices.Contains(submission.SavedSections, section.ID) {
			progress.Complete = append(progress.Complete, section.ID)
		}
	}
	return progress
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
