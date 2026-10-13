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
func TestBankAndModuleSelection(t *testing.T) {
	a := testApp(t)
	if got := len(a.bank.Indicators); got != 39 {
		t.Fatalf("indicators=%d, want 39", got)
	}
	base := model.Session{Kind: "student"}
	if got := len(a.itemsFor(base)); got != 237 {
		t.Fatalf("base items=%d, want 237", got)
	}
	base.Activity = true
	base.Motivation = true
	if got := len(a.itemsFor(base)); got != 255 {
		t.Fatalf("extended items=%d, want 255", got)
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
	if got := len(a.itemsFor(s)); got != 228 {
		t.Fatalf("items after excluding Q01=%d, want 228", got)
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
