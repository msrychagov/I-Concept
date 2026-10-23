package formbank

import (
	"os"
	"regexp"
	"strconv"
	"strings"
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

func essay(t *testing.T) model.Form {
	t.Helper()
	forms, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range forms {
		if form.ID == "essay" {
			return form
		}
	}
	t.Fatal("essay is not embedded")
	return model.Form{}
}

func TestEssayDescription(t *testing.T) {
	form := essay(t)
	if form.Version != "1.0" || len(form.Sections) != 3 || !form.StudentView || form.Texts == nil {
		t.Fatalf("version %s, %d sections, student view %v, texts %v", form.Version, len(form.Sections), form.StudentView, form.Texts)
	}
	if autobiography(t).StudentView {
		t.Fatal("the autobiography must not be shown back to a student")
	}
	numbers := map[string]bool{}
	count := 0
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			count++
			if field.Type != "textarea" || !field.Required || field.MaxLength != 2000 || field.Hint == "" || field.Number == "" || numbers[field.Number] {
				t.Errorf("field %s: %+v", field.ID, field)
			}
			numbers[field.Number] = true
			// A dot in an id would be read as a group record path.
			if strings.Contains(field.ID, ".") {
				t.Errorf("id %s has a dot", field.ID)
			}
		}
	}
	if count != 13 {
		t.Fatalf("%d questions", count)
	}
}

// The questions are the text of Вопросы эссе.md word for word; the wiki is in
// the working folder, not in the repository.
func TestEssayMatchesTheWikiWhenItIsThere(t *testing.T) {
	data, err := os.ReadFile("../../../Вики/Эссе выпускника/Вопросы эссе.md")
	if err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	wiki := string(data)
	form := essay(t)
	rows := regexp.MustCompile(`(?m)^\| (\d\.\d) \| (.+?) \| (.+?) \| .+? \|$`).FindAllStringSubmatch(wiki, -1)
	byNumber := map[string]model.FormField{}
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			byNumber[field.Number] = field
		}
	}
	if len(rows) != len(byNumber) {
		t.Fatalf("%d questions in the wiki, %d in the form", len(rows), len(byNumber))
	}
	for _, row := range rows {
		field, ok := byNumber[row[1]]
		if !ok || field.Label != row[2] || field.Hint != row[3] {
			t.Errorf("question %s differs: %+v", row[1], field)
		}
	}
	for n, section := range form.Sections {
		if !strings.Contains(wiki, "## Блок "+strconv.Itoa(n+1)+". "+section.Title+"\n") {
			t.Errorf("block %d is not %q in the wiki", n+1, section.Title)
		}
	}
	for _, paragraph := range strings.Split(form.Intro, "\n\n") {
		if !strings.Contains(strings.ReplaceAll(wiki, "\n> ", " "), paragraph) {
			t.Errorf("intro differs from the wiki: %.60q", paragraph)
		}
	}
	if hint := form.Sections[2].Hint; hint == "" || !strings.Contains(strings.ReplaceAll(wiki, "\n> ", " "), hint) {
		t.Errorf("hint over block 3: %q", hint)
	}
}
