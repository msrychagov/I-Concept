package usecase

import "testing"

func TestMethodsAreAssignedWhole(t *testing.T) {
	a := testApp(t)
	student := signUp(t, a, "student", "methods@example.test", "101")
	p := a.emptyProfile()
	p.ID, p.Name, p.Author, p.OwnerID, p.BankVersion = "methods-template", "Методики", "Teacher", student.Account.OwnerID, "pilot-2.3"
	p.Questions = clone(a.previous["pilot-2.3"].Items)
	if err := a.addProfile(p); err != nil {
		t.Fatal(err)
	}
	input := Input{"profileId": raw(t, p.ID), "title": raw(t, "Методики"), "methods": raw(t, true)}
	if _, err := a.AssignmentCommandFor(p.OwnerID, input, "assignment"); err != nil {
		t.Fatal(err)
	}
	db := mustSnapshot(t, a.repo)
	modules := map[string]int{}
	for _, q := range db.Assignments[0].Profile.Questions {
		modules[q.Module]++
	}
	if modules["W"] != 30 || modules["U"] != 10 || modules["X"] != 18 || modules["K"] != 19 || modules["V"] != 12 || !db.Assignments[0].Methods {
		t.Fatalf("assignment: %v", modules)
	}
	if _, err := a.StartAssignment(student.Account, db.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	s := mustSnapshot(t, a.repo).Sessions[0].Session
	asked := map[string]int{}
	for _, item := range a.itemsFor(s) {
		asked[item.Module]++
	}
	if !s.Methods || asked["W"] != 30 || asked["K"] != 19 || asked["V"] != 12 {
		t.Fatalf("session asks: %v", asked)
	}
}
