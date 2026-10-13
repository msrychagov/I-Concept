package formbank

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func autobiography(t *testing.T) model.Form {
	t.Helper()
	forms, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range forms {
		if form.ID == "autobiography" {
			return form
		}
	}
	t.Fatal("autobiography is not embedded")
	return model.Form{}
}

func TestAutobiographyDescription(t *testing.T) {
	form := autobiography(t)
	if form.Version != "1.0" || len(form.Sections) != 17 {
		t.Fatalf("version %s, %d sections", form.Version, len(form.Sections))
	}
	// Suggestions are keyed by field id, so these ids must not repeat anywhere.
	suggest := map[string]int{}
	var walk func([]model.FormField)
	walk = func(fields []model.FormField) {
		for _, field := range fields {
			if field.Suggest {
				suggest[field.ID]++
			}
			walk(field.Fields)
		}
	}
	for _, section := range form.Sections {
		walk(section.Fields)
	}
	want := []string{"school_city", "school_name", "vk_district", "vk_prev_district", "hse_program"}
	if len(suggest) != len(want) {
		t.Fatalf("fields with suggestions: %v", suggest)
	}
	for _, id := range want {
		if suggest[id] != 1 {
			t.Fatalf("field %s: %d suggestion fields", id, suggest[id])
		}
	}
	top := map[string]bool{}
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			if top[field.ID] {
				t.Fatalf("top-level field id %s repeats across sections", field.ID)
			}
			top[field.ID] = true
		}
	}
}
