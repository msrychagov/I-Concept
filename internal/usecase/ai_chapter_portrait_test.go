package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type chapterSpyModel struct{ Prompts []Completion }

func (*chapterSpyModel) Name() string { return "spy-qwen" }
func (s *chapterSpyModel) Complete(_ context.Context, p Completion) (Reply, error) {
	s.Prompts = append(s.Prompts, p)
	m := &testAIModel{}
	reply, err := m.Complete(context.Background(), p)
	if err != nil {
		return Reply{}, err
	}
	var decoded aiPedagogicalReport
	_ = json.Unmarshal([]byte(reply.Text), &decoded)
	decoded.Portrait = "В собственной оценке ответственности студент подчёркивает значимость выполнения обязательств."
	raw, _ := json.Marshal(decoded)
	return Reply{Text: string(raw)}, nil
}

func TestChapterPromptUsesCalculatedIndicatorsAndSelfImages(t *testing.T) {
	model := &chapterSpyModel{}
	v := 3.67
	input := aiInput{Kind: "student", Metrics: []aiMetric{
		{ID: "Q05", Name: "Ответственность", Component: "DP", Block: "core", Value: &v, Answered: 3, Total: 3,
			SelfImage: map[string]int{"R": 5, "I": 9, "Z": 3, "N": 8}},
		{ID: "P01", Name: "Ценностное отношение к Отечеству", Value: &v, Answered: 3, Total: 3, Component: "CS"},
	}}
	report, err := produceAIReport(context.Background(), model, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Prompts) != 2 {
		t.Fatalf("expected indicator batch and summary: %d", len(model.Prompts))
	}
	indicatorPrompt := model.Prompts[0]
	for _, want := range []string{"МЕТОДОЛОГИЧЕСКАЯ ОСНОВА", "Я-реальное", `"value":3.67`, `"R":5`, `"I":9`} {
		if !strings.Contains(strings.ToLower(indicatorPrompt.System+indicatorPrompt.User), strings.ToLower(want)) {
			t.Errorf("missing input/theory %q", want)
		}
	}
	if strings.Contains(model.Prompts[0].User, `"id":"P01"`) {
		t.Fatal("political position leaked into portrait inference")
	}
	if report.Indicators[0].Interpretation == "" || !strings.Contains(report.Summary, "В собственной оценке ответственности") || report.GenerationMode != "ai-assisted" {
		t.Fatalf("no AI narrative: %+v", report)
	}
	if report.Indicators[1].Interpretation == "" || !strings.Contains(report.Indicators[1].Interpretation, "заявленная") {
		t.Fatal("missing separate P01 treatment")
	}
	if report.Indicators[0].Score == nil || *report.Indicators[0].Score != v {
		t.Fatal("AI altered deterministic score")
	}
}

func TestChapterPortraitNeverInventsNoData(t *testing.T) {
	spy := &chapterSpyModel{}
	in := aiInput{Kind: "student", Metrics: []aiMetric{{ID: "Q01", Name: "Честность", Answered: 1, Total: 3}}}
	out, err := produceAIReport(context.Background(), spy, in)
	if err == nil || out != nil {
		t.Fatal("missing data must not produce reserve output")
	}
	if len(spy.Prompts) != 0 {
		t.Fatal("model called without calculated values")
	}
}

func TestChapterModelContentIsNotFiltered(t *testing.T) {
	v := 3.5
	input := aiInput{Kind: "student", Metrics: []aiMetric{{ID: "Q05", Name: "Ответственность", Value: &v, Answered: 3, Total: 3}}}
	answer := `{"portrait":"Короткий вывод","actions":["Одна рекомендация"],"indicators":[{"id":"Q05","analysis":"Темы для беседы","action":"Коротко","question":"Без вопросительного знака"},{"id":"unknown","analysis":"Дополнительный текст"}]}`
	m := &fakeModel{name: "unfiltered", reply: func(n int, _ Completion) (string, error) {
		return answer, nil
	}}
	r, err := produceAIReport(context.Background(), m, input)
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary != "Короткий вывод" || len(r.Recommendations) != 1 || r.Indicators[0].Interpretation != "Темы для беседы" || r.Indicators[0].Discussion != "Без вопросительного знака" {
		t.Fatalf("model content was filtered: %+v", r)
	}
	if len(r.ModelResponses) != 2 || r.ModelResponses[0] != answer || r.ModelResponses[1] != answer {
		t.Fatal("complete model responses were not preserved")
	}
}
