package usecase

import (
	"slices"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func cohortApp(t *testing.T) *Service {
	t.Helper()
	a := instructorApp(t)
	if err := a.repo.Update(func(db *model.Database) error {
		db.Platoons = []model.Platoon{{ID: "p1", Name: "2401", VUC: "ВУЦ один"}, {ID: "p2", Name: "2402", VUC: "ВУЦ один"}, {ID: "p3", Name: "2403", VUC: "ВУЦ один"}, {ID: "px", Name: "9999", VUC: "ВУЦ два"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return a
}

func cohortsOf(t *testing.T, a *Service, teacherID string) []CohortView {
	t.Helper()
	value, err := a.CohortList(teacherID)
	if err != nil {
		t.Fatal(err)
	}
	return value.(map[string]any)["cohorts"].([]CohortView)
}

func TestCohortRules(t *testing.T) {
	a := cohortApp(t)
	saved, err := a.SaveCohort("t1", "", CohortInput{Name: "  Выпуск  2027 ", Year: 2027, PlatoonIDs: []string{"p1", "p1", "p2"}})
	if err != nil {
		t.Fatal(err)
	}
	id := saved.(model.Cohort).ID
	if got := saved.(model.Cohort); got.Name != "Выпуск 2027" || got.VUC != "ВУЦ один" || got.CreatedBy != "t1" || !slices.Equal(got.PlatoonIDs, []string{"p1", "p2"}) {
		t.Fatalf("saved: %+v", got)
	}
	for name, input := range map[string]CohortInput{
		"без названия": {Name: " ", Year: 2027},
		"год не число": {Name: "Выпуск", Year: 0},
		"год далеко":   {Name: "Выпуск", Year: 3000},
	} {
		if _, err = a.SaveCohort("t1", "", input); errorKind(err) != Invalid {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// A platoon is in one cohort only, even for another teacher of the center.
	if _, err = a.SaveCohort("t2", "", CohortInput{Name: "Выпуск 2028", Year: 2028, PlatoonIDs: []string{"p2", "p3"}}); errorKind(err) != Conflict {
		t.Fatalf("platoon in two cohorts: %v", err)
	}
	// A platoon of another center is not found.
	if _, err = a.SaveCohort("t2", "", CohortInput{Name: "Выпуск 2028", Year: 2028, PlatoonIDs: []string{"px"}}); errorKind(err) != NotFound {
		t.Fatalf("foreign platoon: %v", err)
	}
	if _, err = a.SaveCohort("t2", "", CohortInput{Name: "выпуск 2027", Year: 2027, PlatoonIDs: []string{"p3"}}); errorKind(err) != Conflict {
		t.Fatalf("same name and year: %v", err)
	}
	if _, err = a.SaveCohort("t2", "", CohortInput{Name: "Выпуск 2028", Year: 2028, PlatoonIDs: []string{"p3"}}); err != nil {
		t.Fatal(err)
	}
	// The cohort is visible to the whole center, not to another center.
	list := cohortsOf(t, a, "t2")
	if len(list) != 2 || list[0].Year != 2028 || list[0].Mine != true || list[1].Mine != false || list[1].CreatedByName != "Первый Преподаватель" {
		t.Fatalf("list of the center: %+v", list)
	}
	if got := cohortsOf(t, a, "t3"); len(got) != 0 {
		t.Fatalf("other center sees: %+v", got)
	}
	// Only the creator changes or deletes it; a platoon can move on edit.
	if _, err = a.SaveCohort("t2", id, CohortInput{Name: "Выпуск 2027", Year: 2027, PlatoonIDs: []string{"p1"}}); errorKind(err) != Forbidden {
		t.Fatalf("foreign edit: %v", err)
	}
	if _, err = a.DeleteCohort("t2", id); errorKind(err) != Forbidden {
		t.Fatalf("foreign delete: %v", err)
	}
	if _, err = a.SaveCohort("t3", id, CohortInput{Name: "Чужой", Year: 2027}); errorKind(err) != NotFound {
		t.Fatalf("other center edit: %v", err)
	}
	if _, err = a.SaveCohort("t1", id, CohortInput{Name: "Выпуск 2027", Year: 2027, PlatoonIDs: []string{"p1", "p2"}}); err != nil {
		t.Fatalf("own edit keeps its platoons: %v", err)
	}
	// A cohort that got an essay cannot be deleted.
	if err = a.repo.Update(func(db *model.Database) error {
		db.Assignments = append(db.Assignments, model.Assignment{ID: "as1", FormID: "essay", CohortID: id, OwnerID: "t1"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = a.DeleteCohort("t1", id); errorKind(err) != Conflict {
		t.Fatalf("delete with assignments: %v", err)
	}
	foreign := cohortsOf(t, a, "t1")[0]
	if _, err = a.DeleteCohort("t1", foreign.ID); foreign.CreatedBy != "t2" || errorKind(err) != Forbidden {
		t.Fatalf("t1 deleted a cohort of t2: %v", err)
	}
	if _, err = a.DeleteCohort("t2", foreign.ID); err != nil || len(cohortsOf(t, a, "t1")) != 1 {
		t.Fatalf("creator did not delete: %v", err)
	}
}
