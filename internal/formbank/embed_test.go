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

func TestSchoolRegionsAreRegionsPlusAbroad(t *testing.T) {
	form := autobiography(t)
	regions, school := form.Refs["regions"], form.Refs["school_regions"]
	if len(school) != len(regions)+1 {
		t.Fatalf("%d school regions for %d regions", len(school), len(regions))
	}
	for i, region := range regions {
		if school[i] != region {
			t.Fatalf("lists differ at %d: %v and %v", i, region, school[i])
		}
	}
	if last := school[len(school)-1]; last.ID != "abroad" || last.Text != "За пределами России" {
		t.Fatalf("last school region: %v", last)
	}
	// Only the school may be abroad: birth place, military office and registration use the plain list.
	users := map[string]string{}
	var walk func([]model.FormField)
	walk = func(fields []model.FormField) {
		for _, f := range fields {
			if f.Ref == "regions" || f.Ref == "school_regions" {
				users[f.ID] = f.Ref
			}
			walk(f.Fields)
		}
	}
	for _, s := range form.Sections {
		walk(s.Fields)
	}
	for id, ref := range users {
		if want := map[bool]string{true: "school_regions", false: "regions"}[id == "edu_region"]; ref != want {
			t.Errorf("%s uses %s, want %s", id, ref, want)
		}
	}
	if len(users) != 5 {
		t.Errorf("fields with a region list: %v", users)
	}
}
