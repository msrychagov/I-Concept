package usecase

import (
	"encoding/json"
	"time"
	"vucdiagnostic/internal/domain/model"
)

// sampleAnswers fills a form with placeholder values that satisfy every
// required field. Sections listed in detailed get the "yes" branch of their
// opening question; the others get "no". Used for demonstration data and tests.
func sampleAnswers(form model.Form, detailed map[string]bool, today time.Time) (map[string]json.RawMessage, error) {
	answers := map[string]json.RawMessage{}
	for _, section := range form.Sections {
		input := sampleFields(form, section.Fields, nil, detailed[section.ID], true)
		cleaned, err := cleanSection(form, section, input, today)
		if err != nil {
			return nil, err
		}
		for key, value := range cleaned {
			answers[key] = value
		}
	}
	return answers, nil
}

func sampleFields(form model.Form, fields []model.FormField, outer map[string][]string, yes, top bool) map[string]json.RawMessage {
	input := map[string]json.RawMessage{}
	values := map[string][]string{}
	put := func(field model.FormField, value any) {
		data, _ := json.Marshal(value)
		input[field.ID] = data
		if list, ok := value.([]string); ok {
			values[field.ID] = list
		} else if text, ok := value.(string); ok {
			values[field.ID] = []string{text}
		}
	}
	for _, field := range fields {
		if !conditionHolds(field.ShowIf, values, outer) {
			continue
		}
		switch field.Type {
		case "choice":
			options := form.Options(field)
			pick := options[0].ID
			if top && hasOption(form, field, "no") && hasOption(form, field, "yes") {
				pick = "no"
				if yes {
					pick = "yes"
				}
			} else if pick == "other" && len(options) > 1 {
				pick = options[1].ID
			}
			put(field, pick)
		case "multi":
			put(field, []string{form.Options(field)[0].ID})
		case "text", "textarea":
			put(field, "Образец")
		case "date":
			put(field, "2005-03-15")
		case "month":
			put(field, "2022-09")
		case "year":
			put(field, "2020")
		case "group":
			if field.Min > 0 || yes {
				input[field.ID], _ = json.Marshal([]map[string]json.RawMessage{sampleFields(form, field.Fields, values, yes, false)})
			}
		}
	}
	return input
}
