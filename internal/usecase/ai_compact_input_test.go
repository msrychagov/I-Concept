package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestAIInputKeepsScoresAndOnlyOpenQuestionText(t *testing.T) {
	bank := model.Bank{Indicators: []model.Indicator{{ID: "Q01", Name: "Показатель", Items: []string{"closed"}}}, Items: []model.Item{
		{ID: "closed", Indicator: "Q01", Text: "Текст закрытого вопроса", Min: 1, Max: 5, Reverse: true},
		{ID: "open", Module: "F", Type: "textarea", Text: "Расскажите об опыте", ResponseOnly: true},
		{ID: "unanswered", Module: "F", Type: "textarea", Text: "Не отвечен", ResponseOnly: true},
	}}
	svc := New(bank, nil)
	session := model.Session{Kind: "student", Activity: true, Answers: map[string]int{"closed": 2, "open": 0}, Notes: map[string]string{"open": "Мой опыт", "unanswered": "Старая заметка", "closed": "Лишний комментарий", "other": "Не из теста"}}
	input := svc.studentAIInput(session)
	if len(input.Metrics) != 1 || input.Metrics[0].Value == nil || *input.Metrics[0].Value != 4 {
		t.Fatalf("calculated score lost: %+v", input)
	}
	if len(input.OpenAnswers) != 1 || input.OpenAnswers[0].Answer != "Мой опыт" || input.OpenAnswers[0].Question != "Расскажите об опыте" {
		t.Fatalf("incorrect open answers: %+v", input.OpenAnswers)
	}
	raw, _ := json.Marshal(input)
	for _, excluded := range []string{"Текст закрытого вопроса", "Лишний комментарий", "Старая заметка", "Не из теста", "agreement", "oriented", "responses"} {
		if strings.Contains(string(raw), excluded) {
			t.Fatalf("unexpected raw input: %s", excluded)
		}
	}
	session.Activity = false
	if len(svc.studentAIInput(session).OpenAnswers) != 0 {
		t.Fatal("disabled activity sent to model")
	}
}

func TestAIOpenOnlyReportIncludesAllAnswersInBoundedCalls(t *testing.T) {
	input := aiInput{Kind: "student", OpenAnswers: []aiOpenAnswer{
		{ID: "one", Question: "Первый вопрос", Answer: strings.Repeat("а", 4000)},
		{ID: "two", Question: "Второй вопрос", Answer: strings.Repeat("б", 4000)},
	}}
	var received []aiOpenAnswer
	m := &fakeModel{name: "open", reply: func(n int, prompt Completion) (string, error) {
		if strings.HasPrefix(prompt.User, "Открытые ответы:\n") {
			var batch []aiOpenAnswer
			if err := json.Unmarshal([]byte(strings.TrimPrefix(prompt.User, "Открытые ответы:\n")), &batch); err != nil {
				t.Fatal(err)
			}
			received = append(received, batch...)
			return `{"portrait":"Обобщение опыта","actions":[]}`, nil
		}
		if !strings.Contains(prompt.User, "Обобщение опыта") {
			t.Fatal("open-answer insights missing from summary")
		}
		return `{"portrait":"Общий вывод","actions":[]}`, nil
	}}
	r, err := produceAIReport(context.Background(), m, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(received) != 2 || received[0] != input.OpenAnswers[0] || received[1] != input.OpenAnswers[1] {
		t.Fatal("open answers were truncated or lost")
	}
	if r.Summary != "Общий вывод" || len(r.ModelResponses) != 3 {
		t.Fatalf("missing report responses: %+v", r)
	}
}

func TestAIIndicatorsAreBatchedBeforeSeparateSummary(t *testing.T) {
	v := 3.75
	input := aiInput{Kind: "student", OpenAnswers: []aiOpenAnswer{{ID: "open", Question: "Что вы сделали?", Answer: "Подготовил совместный проект."}}}
	for i := 1; i <= 12; i++ {
		input.Metrics = append(input.Metrics, aiMetric{ID: fmt.Sprintf("Q%02d", i), Name: "Показатель", Value: &v, Answered: 3, Total: 3})
	}
	seen := map[string]bool{}
	m := &fakeModel{name: "batched", reply: func(n int, prompt Completion) (string, error) {
		if n <= 3 {
			var payload aiReportPayload
			if err := json.Unmarshal([]byte(strings.TrimPrefix(prompt.User, "Показатели:\n")), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Indicators) != []int{5, 5, 2}[n-1] {
				t.Fatalf("wrong batch size: %d", len(payload.Indicators))
			}
			for _, metric := range payload.Indicators {
				if seen[metric.ID] || metric.Value != v {
					t.Fatalf("duplicate or changed metric: %+v", metric)
				}
				seen[metric.ID] = true
			}
			stub := &testAIModel{}
			r, err := stub.Complete(context.Background(), prompt)
			return r.Text, err
		}
		if n == 4 {
			var answers []aiOpenAnswer
			if err := json.Unmarshal([]byte(strings.TrimPrefix(prompt.User, "Открытые ответы:\n")), &answers); err != nil {
				t.Fatal(err)
			}
			if len(answers) != 1 || answers[0] != input.OpenAnswers[0] {
				t.Fatal("open answer lost")
			}
			return `{"portrait":"Обобщение опыта","actions":[]}`, nil
		}
		if n != 5 {
			t.Fatal("unexpected extra request")
		}
		var payload aiReportPayload
		if err := json.Unmarshal([]byte(strings.TrimPrefix(prompt.User, "Итоговый отчёт:\n")), &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Indicators) != 12 || payload.Indicators[0].Insight == "" || len(payload.OpenInsights) != 1 {
			t.Fatal("summary missing calculated results")
		}
		var schema struct {
			Required []string `json:"required"`
		}
		_ = json.Unmarshal(prompt.Schema, &schema)
		if !contains(schema.Required, "portrait") || !contains(schema.Required, "actions") || contains(schema.Required, "indicators") {
			t.Fatal("summary should not regenerate indicator analyses")
		}
		return `{"portrait":"Итог","actions":["Действие"]}`, nil
	}}
	r, err := produceAIReport(context.Background(), m, input)
	if err != nil {
		t.Fatal(err)
	}
	if m.count() != 5 || len(seen) != 12 || r.Summary != "Итог" || r.Indicators[11].Interpretation == "" || len(r.ModelResponses) != 5 {
		t.Fatalf("batched output lost: %+v", r)
	}
}
