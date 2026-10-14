package usecase

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestEditableQuestionsSnapshotAndScoring(t *testing.T) {
	a := testApp(t)
	p := a.emptyProfile()
	p.Author = "Автор"
	p.Questions = clone(a.bank.Items)
	for n, i := range p.Questions {
		if i.ID == "I001" {
			p.Questions[n].Text = "Новая формулировка"
			p.Questions[n].Section = "D"
		}
	}
	for n, i := range p.Questions {
		if i.ID == "I003" {
			p.Questions = append(p.Questions[:n], p.Questions[n+1:]...)
			break
		}
	}
	p.Questions = append(p.Questions, model.Item{ID: "CUSTOM_1", Module: "B", Section: "D", Indicator: "Q01", Text: "Новый вопрос", Type: "likert", Min: 1, Max: 5, Scored: true})
	if err := a.validateQuestions(p); err != nil {
		t.Fatal(err)
	}
	s := model.Session{Kind: "student", Profile: clone(p), Answers: map[string]int{"I001": 5, "I002": 1, "CUSTOM_1": 2}}
	rows := a.analyze(s, nil)["rows"].([]reportRow)
	r, _ := rowByIndicator(rows, "Q01")
	if r.Actual == nil || *r.Actual != 4 || r.Total != 3 {
		t.Fatalf("wrong result: %+v", r)
	}
	for _, item := range a.publicItems(s) {
		if item["id"] == "I001" && (item["text"] != "Новая формулировка" || item["module"] != "D") {
			t.Fatal("edited question not delivered")
		}
		if item["id"] == "I003" {
			t.Fatal("deleted question delivered")
		}
	}
	p.Questions[0].Text = "Другое изменение"
	if s.Profile.Questions[0].Text == p.Questions[0].Text {
		t.Fatal("snapshot changed")
	}
	old := model.Session{Kind: "student", BankVersion: "pilot-2.1", Answers: map[string]int{}}
	if len(a.itemsFor(old)) != 237 {
		t.Fatal("legacy questionnaire changed")
	}
}

func TestRejectIndicatorWithoutQuestions(t *testing.T) {
	a := testApp(t)
	p := a.emptyProfile()
	p.Questions = []model.Item{}
	for _, i := range a.bank.Items {
		if !contains([]string{"I001", "I002", "I003"}, i.ID) {
			p.Questions = append(p.Questions, i)
		}
	}
	if a.validateQuestions(p) == nil {
		t.Fatal("accepted indicator without scored questions")
	}
}
