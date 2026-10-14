package usecase

import (
	"path/filepath"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/formbank"
	"vucdiagnostic/internal/questionbank"
	"vucdiagnostic/internal/repository/jsonfile"
)

func testApp(t *testing.T) *Service {
	t.Helper()
	bank, err := questionbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	repo, err := jsonfile.Open(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	previous, err := questionbank.Previous()
	if err != nil {
		t.Fatal(err)
	}
	forms, err := formbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	service := New(bank, repo, previous...)
	service.RegisterForms(forms...)
	return service
}

func mustSnapshot(t *testing.T, repo Repository) model.Database {
	t.Helper()
	db, err := repo.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	return db
}
func TestBankAndModuleSelection(t *testing.T) {
	a := testApp(t)
	if got := len(a.bank.Indicators); got != 67 {
		t.Fatalf("indicators=%d, want 67", got)
	}
	base := model.Session{Kind: "student"}
	if got := len(a.itemsFor(base)); got != 386 {
		t.Fatalf("base items=%d, want 386", got)
	}
	base.Activity = true
	base.Motivation = true
	if got := len(a.itemsFor(base)); got != 404 {
		t.Fatalf("extended items=%d, want 404", got)
	}
	if got := len(a.itemsFor(model.Session{Kind: "observer"})); got != 8 {
		t.Fatalf("observer items=%d, want 8", got)
	}
}

func TestTemplateControlsQuestionAndReportComposition(t *testing.T) {
	a := testApp(t)
	p := a.emptyProfile()
	delete(p.Values, "Q01")
	s := model.Session{Kind: "student", Profile: p, Answers: map[string]int{}, Notes: map[string]string{}}
	if got := len(a.itemsFor(s)); got != 377 {
		t.Fatalf("items after excluding Q01=%d, want 377", got)
	}
	for _, item := range a.itemsFor(s) {
		if item.Indicator == "Q01" {
			t.Fatalf("excluded indicator item %s remains", item.ID)
		}
	}
	report := a.analyze(s, nil)
	rows := report["rows"].([]reportRow)
	if len(rows) != 38 {
		t.Fatalf("report rows=%d, want 38", len(rows))
	}
}

func TestScoringAndIncompleteMean(t *testing.T) {
	a := testApp(t)
	direct, _ := a.item("I001")
	reverse, _ := a.item("I002")
	if score(direct, 2) != 2 {
		t.Fatal("direct item scored incorrectly")
	}
	if score(reverse, 2) != 4 {
		t.Fatal("reverse item scored incorrectly")
	}
	s := model.Session{Kind: "student", Answers: map[string]int{"I001": 5, "I002": 1}}
	if a.mean(s, []string{"I001", "I002", "I003"}) != nil {
		t.Fatal("incomplete indicator must not have a mean")
	}
	s.Answers["I003"] = 3
	if got := *a.mean(s, []string{"I001", "I002", "I003"}); got != 13.0/3.0 {
		t.Fatalf("mean=%v", got)
	}
	s.Answers["PS_EAT01"] = 0
	s.Answers["PS_EAT02"] = 5 // NA is distinct from never and from refusal.
	s.Answers["PS_EAT03"] = 6 // SKIP
	if a.mean(s, []string{"PS_EAT01"}) != nil {
		t.Fatal("psychological answer entered pedagogical scoring")
	}
	group := a.psychologicalReport(s)[0]
	if group["answered"] != 1 || group["notApplicable"] != 1 || group["declined"] != 1 || group["unanswered"] != 2 {
		t.Fatalf("psychological answer states collapsed: %+v", group)
	}
}

func TestProfileValidation(t *testing.T) {
	a := testApp(t)
	p := a.emptyProfile()
	p.Name = "Шаблон"
	p.Author = "Эксперт"
	p.Group = "1"
	if err := a.validateProfile(p); err != nil {
		t.Fatal(err)
	}
	v := 4.0
	p.Values[a.bank.Indicators[0].ID] = &v
	if err := a.validateProfile(p); err == nil {
		t.Fatal("template with target other than 5 was accepted")
	}
}
