package usecase

import (
	"slices"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestFormSuggestionsComeFromOtherStudentsAndOnlyForMarkedFields(t *testing.T) {
	f := newFormFixture(t)
	for n, school := range []string{"Лицей № 1580", "Гимназия 1514", "лицей № 1580", "Школа 57"} {
		login := signUp(t, f.service, "student", string(rune('a'+n))+"@example.test", "101")
		id := addSubmission(t, f.service, login.Account.ID, f.assignment)
		err := f.service.repo.Update(func(db *model.Database) error {
			for i := range db.Submissions {
				if db.Submissions[i].ID == id {
					db.Submissions[i].Answers["education_list"] = raw(t, []obj{{"school_name": school}})
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	own := raw(t, []obj{{"school_name": "Лицей собственный"}})
	_ = f.service.repo.Update(func(db *model.Database) error { db.Submissions[0].Answers["education_list"] = own; return nil })
	got, err := f.service.FormSuggestions(f.token, f.submission, "school_name", "ЛИЦЕЙ")
	if err != nil {
		t.Fatal(err)
	}
	values := got.(map[string]any)["values"].([]string)
	if !slices.Equal(values, []string{"Лицей № 1580"}) {
		t.Fatalf("suggestions: %v", values)
	}
	all, _ := f.service.FormSuggestions(f.token, f.submission, "school_name", "")
	if len(all.(map[string]any)["values"].([]string)) != 3 {
		t.Fatalf("all suggestions: %v", all)
	}
	if _, err = f.service.FormSuggestions(f.token, f.submission, "birth_place", "т"); kind(err) != Invalid {
		t.Fatalf("suggestions for an unmarked field: %v", err)
	}
	if _, err = f.service.FormSuggestions("", f.submission, "school_name", "т"); kind(err) != Forbidden {
		t.Fatalf("suggestions for a stranger: %v", err)
	}
}
