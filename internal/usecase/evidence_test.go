package usecase

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestBankRevisionPreservesLegacyQuestions(t *testing.T) {
	a := testApp(t)
	current := model.Session{Kind: "student", BankVersion: "pilot-2.1"}
	legacy := model.Session{Kind: "student", BankVersion: "pilot-2.0"}
	text := func(s model.Session, id string) string {
		for _, item := range a.itemsFor(s) {
			if item.ID == id {
				return item.Text
			}
		}
		t.Fatalf("missing %s", id)
		return ""
	}
	if text(current, "I014") == text(legacy, "I014") {
		t.Fatal("legacy wording was replaced")
	}
	if text(legacy, "I014") != "После выполнения своей части задания я обычно перестаю считать себя ответственным за конечный результат." {
		t.Fatal("legacy wording changed")
	}
	p := a.emptyProfile()
	p.BankVersion = "pilot-2.0"
	if err := a.addProfile(p); err != nil {
		t.Fatal(err)
	}
	state, err := a.ResearchQuery("state", "", "")
	if err != nil {
		t.Fatal(err)
	}
	saved := state.(map[string]any)["profiles"].([]model.Profile)[0]
	if len(saved.Questions) == 0 {
		t.Fatal("legacy template returned without frozen questions")
	}
	for _, item := range saved.Questions {
		if item.ID == "I014" && item.Text != text(legacy, "I014") {
			t.Fatal("editor would silently replace old question")
		}
	}
}
func TestEvidenceSeparatesSourcesAndRequiresComparableHistory(t *testing.T) {
	a := testApp(t)
	p := a.emptyProfile()
	p.Questions = clone(a.bank.Items)
	previous := model.Session{ID: "previous", StudentID: "student-one", Kind: "student", Participant: "Одинаковое Имя", Profile: clone(p), BankVersion: a.bank.Version, Completed: true, Updated: "2026-01-01T10:00:00Z", Answers: map[string]int{"I001": 3, "I002": 3, "I003": 3, "Q01_R": 9}}
	current := clone(previous)
	current.ID = "current"
	current.Updated = "2026-02-01T10:00:00Z"
	current.Answers["I001"] = 5
	current.Answers["I002"] = 1
	current.Answers["I003"] = 5
	extract := func(all []model.SessionRecord) EvidenceRow {
		report := a.analyze(current, all)
		for _, row := range report["evidence"].([]EvidenceRow) {
			if row.ID == "Q01" {
				return row
			}
		}
		t.Fatal("Q01 missing")
		return EvidenceRow{}
	}
	row := extract([]model.SessionRecord{{Session: previous}})
	if row.SelfImage == nil || *row.SelfImage != 9 || row.Statements == nil || *row.Statements != 5 || !row.Trend.Available || *row.Trend.Delta != 2 {
		t.Fatalf("sources incorrectly combined: %+v", row)
	}
	other := clone(previous)
	other.StudentID = "student-two"
	if extract([]model.SessionRecord{{Session: other}}).Trend.Available {
		t.Fatal("same name matched another student")
	}
	changed := clone(previous)
	changed.Profile.Questions[0].Text = "Другая формулировка"
	if extract([]model.SessionRecord{{Session: changed}}).Trend.Available {
		t.Fatal("different question wording was compared")
	}
	changed = clone(previous)
	changed.BankVersion = "pilot-2.0"
	if extract([]model.SessionRecord{{Session: changed}}).Trend.Available {
		t.Fatal("different bank versions were compared")
	}
	delete(current.Answers, "I003")
	row = extract([]model.SessionRecord{{Session: previous}})
	if row.Statements != nil || row.Trend.Available || row.Coverage != 2 {
		t.Fatal("missing answer treated as a score")
	}
}
