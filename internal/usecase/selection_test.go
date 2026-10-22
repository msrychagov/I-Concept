package usecase

import (
	"reflect"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestAssignmentSelectionPreservesScalesAndSnapshot(t *testing.T) {
	a := testApp(t)
	p := a.emptyProfile()
	p.ID = "selection-template"
	p.Questions = clone(a.bank.Items)
	original := clone(p)
	selected := a.assignmentQuestions(p, false, false, false)
	if len(selected) >= len(a.itemsFor(model.Session{Kind: "student", Profile: p})) {
		t.Fatal("full bank was not reduced")
	}
	if !reflect.DeepEqual(original, p) {
		t.Fatal("selection changed template")
	}
	counts := map[string]int{}
	ids := map[string]bool{}
	for _, item := range selected {
		if ids[item.ID] {
			t.Fatal("duplicate selected item")
		}
		ids[item.ID] = true
		if item.ResponseOnly {
			counts[item.Indicator]++
		}
		if item.Module == "E" || item.Module == "V" || item.Module == "F" {
			t.Fatal("disabled module selected")
		}
	}
	for _, indicator := range a.bank.Indicators {
		if indicator.Descriptive {
			if counts[indicator.ID] == 0 || counts[indicator.ID] > 2 {
				t.Fatalf("descriptive coverage %s: %d", indicator.ID, counts[indicator.ID])
			}
			continue
		}
		for _, id := range indicator.Items {
			if !ids[id] {
				t.Fatalf("existing three-item scale lost %s", id)
			}
		}
	}
	p.Questions = selected
	s := model.Session{Kind: "student", Profile: p, Answers: map[string]int{}}
	for _, item := range selected {
		s.Answers[item.ID] = item.Min
	}
	for _, indicator := range a.engineFor(s).bank.Indicators {
		if !indicator.Descriptive && a.mean(s, indicator.Items) == nil {
			t.Fatalf("cannot calculate %s", indicator.ID)
		}
	}
	if err := a.repo.Update(func(db *model.Database) error { db.Profiles = append(db.Profiles, original); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AssignmentCommand(Input{"profileId": []byte(`"selection-template"`), "title": []byte(`"Selection"`)}, "assignment"); err != nil {
		t.Fatal(err)
	}
	db := mustSnapshot(t, a.repo)
	snapshot := clone(db.Assignments[0].Profile.Questions)
	if !reflect.DeepEqual(db.Profiles[0].Questions, original.Questions) {
		t.Fatal("assignment changed source template")
	}
	student := model.Account{ID: "selection-student", Role: "student"}
	for range 2 {
		if _, err := a.StartAssignment(student, db.Assignments[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	db = mustSnapshot(t, a.repo)
	if len(db.Sessions) != 1 || !reflect.DeepEqual(db.Sessions[0].Session.Profile.Questions, snapshot) {
		t.Fatal("session resampled assignment")
	}
	t.Logf("Full student bank: %d; selected: %d", len(a.itemsFor(model.Session{Kind: "student", Profile: original})), len(selected))
}
