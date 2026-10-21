package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type testAIModel struct {
	Calls int
	Mode  string
}

func (*testAIModel) Name() string { return "test-local" }
func (m *testAIModel) Complete(_ context.Context, c Completion) (Reply, error) {
	m.Calls++
	if m.Mode == "empty" {
		return Reply{}, nil
	}
	if m.Mode == "error" {
		return Reply{}, errors.New("LLM unavailable")
	}
	if m.Mode == "malformed" {
		return Reply{Text: `{this is not json}`}, nil
	}
	var payload aiReportPayload
	if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimPrefix(c.User, "Итоговый отчёт:\n"), "Показатели:\n")), &payload); err != nil {
		return Reply{}, err
	}
	portrait := strings.Repeat("Показатель самооценки описывает представления студента о собственных действиях, переживаниях и учебном взаимодействии в ВУЦ. ", 3)
	if m.Mode == "weak" {
		portrait = "Студент показывает ограниченную степень самоотчётности в ключевых аспектах личности."
	}
	items := []map[string]string{}
	for _, v := range payload.Indicators {
		analysis := "Данные самоотчёта по показателю «" + v.Name + "» позволяют обсудить субъективное представление студента о себе в учебной деятельности ВУЦ. Объективное поведение пока не проверено."
		if m.Mode == "weak" {
			analysis = "Студент показывает ограниченную степень самоотчётности в ключевых аспектах личности."
		}
		items = append(items, map[string]string{"id": v.ID, "analysis": analysis, "action": "Разобрать со студентом один конкретный учебный эпизод и его собственные основания действий.", "question": "В каких учебных ситуациях проявляется этот показатель?"})
	}
	b, _ := json.Marshal(map[string]any{"indicators": items, "portrait": portrait, "actions": []string{"Обсудить учебный эпизод.", "Запланировать совместное упражнение."}})
	return Reply{Text: string(b)}, nil
}

func TestAIReportKeepsDeterministicValues(t *testing.T) {
	model := &testAIModel{}
	x := 4.25
	metrics := make([]aiMetric, 0, 12)
	for i := 0; i < 12; i++ {
		metrics = append(metrics, aiMetric{ID: fmt.Sprintf("Q%02d", i+1), Name: "Показатель", Value: &x, Answered: 3, Total: 3})
	}
	report, err := produceAIReport(context.Background(), model, aiInput{Kind: "student", Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	if model.Calls != 4 {
		t.Fatalf("expected 3 indicator batches and a summary, got %d", model.Calls)
	}
	if report.FormatVersion != "2" || report.GenerationMode != "ai-assisted" || len(report.Indicators) != 12 {
		t.Fatalf("invalid fixed report structure: %+v", report)
	}
	if len(report.Findings) < 3 || len(report.Recommendations) == 0 || report.DataQuality == "" || report.Summary == "" || report.Limitations == "" {
		t.Fatalf("incomplete sections: %+v", report)
	}
	for _, ind := range report.Indicators {
		if ind.Score == nil || *ind.Score != 4.25 || ind.Status != "Рассчитан" || ind.Evidence == "" || ind.Interpretation == "" || ind.NextStep == "" || !strings.HasSuffix(ind.Discussion, "?") {
			t.Fatalf("invalid indicator: %+v", ind)
		}
	}
}

func TestAIMissingDataStillProducesSameSections(t *testing.T) {
	model := &testAIModel{}
	metrics := []aiMetric{
		{ID: "A", Name: "Первый", Answered: 0, Total: 5},
		{ID: "B", Name: "Второй", Answered: 2, Total: 4},
	}
	report, err := produceAIReport(context.Background(), model, aiInput{Kind: "student", Metrics: metrics})
	if err == nil || report != nil {
		t.Fatal("expected failure without a model report")
	}
}

func TestAIIncompleteStudentValueNotPublished(t *testing.T) {
	value := 4.6
	input := aiInput{Kind: "student", Metrics: []aiMetric{{ID: "P01", Name: "Условия", Value: &value, Answered: 2, Total: 4}}}
	report, err := produceAIReport(context.Background(), &testAIModel{}, input)
	if err == nil || report != nil {
		t.Fatal("expected failure without a model report")
	}
}

func TestAIResponseContentDoesNotFailReport(t *testing.T) {
	v := 3.5
	for _, kind := range []string{"student", "platoon"} {
		input := aiInput{Kind: kind, Metrics: []aiMetric{{ID: "Q01", Name: "Самооценка", Value: &v, Answered: 3, Total: 3, N: 3}}}
		for _, mode := range []string{"error", "malformed", "weak", "empty"} {
			t.Run(kind+mode, func(t *testing.T) {
				report, err := produceAIReport(context.Background(), &testAIModel{Mode: mode}, input)
				if mode == "error" {
					if err == nil || report != nil {
						t.Fatal("transport failure without an answer must fail")
					}
				} else if err != nil || report == nil || len(report.ModelResponses) != 2 {
					t.Fatal("model content was rejected", err)
				}
			})
		}
	}
}

func TestAIPlatoonReportWithSparseCoverage(t *testing.T) {
	v := 3.25
	input := aiInput{Kind: "platoon", Participants: 7, Included: 4, Excluded: 3, Metrics: []aiMetric{
		{ID: "D01", Name: "Условия", Value: &v, N: 3, Answered: 9, Total: 12},
		{ID: "D02", Name: "Среда", N: 2, Answered: 6, Total: 12},
	}}
	report := aiCore(input, "test-local")
	if report.Kind != "platoon" || report.Included != 4 || report.Excluded != 3 || report.FormatVersion != "2" {
		t.Fatalf("missing platoon structure: %+v", report)
	}
	if report.Indicators[0].Score == nil || report.Indicators[1].Score != nil {
		t.Fatalf("suppression failed: %+v", report.Indicators)
	}
	if !strings.Contains(report.DataQuality, "4 из 7") || len(report.Recommendations) == 0 {
		t.Fatalf("cohort coverage missing: %+v", report)
	}
}

func TestAIEmptyMetricListStillReturnsReport(t *testing.T) {
	report, err := produceAIReport(context.Background(), &testAIModel{}, aiInput{Kind: "student"})
	if err == nil || report != nil {
		t.Fatal("expected failure without a model report")
	}
}

func TestAIThirtyNineIndicatorsNeverLoseRows(t *testing.T) {
	model := &testAIModel{}
	metrics := make([]aiMetric, 0, 39)
	for i := 1; i <= 39; i++ {
		val := 2.0 + float64(i%3)
		metrics = append(metrics, aiMetric{ID: fmt.Sprintf("M%02d", i), Name: fmt.Sprintf("Показатель %d", i), Value: &val, Answered: 4, Total: 4})
	}
	report, err := produceAIReport(context.Background(), model, aiInput{Kind: "student", Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	if model.Calls != 9 || len(report.Indicators) != 39 {
		t.Fatalf("expected 8 indicator batches and a summary for 39 rows: calls=%d rows=%d", model.Calls, len(report.Indicators))
	}
	for i, row := range report.Indicators {
		if row.ID != metrics[i].ID || row.Status != "Рассчитан" || row.Evidence == "" || row.Interpretation == "" || row.Discussion == "" {
			t.Fatalf("missing row %d: %+v", i, row)
		}
	}
}

func TestAITinyPlatoonIsCoverageOnly(t *testing.T) {
	v := 4.9
	report, err := produceAIReport(context.Background(), &testAIModel{}, aiInput{
		Kind: "platoon", Participants: 2, Included: 2, Metrics: []aiMetric{{ID: "Q01", Name: "Показатель", Value: &v, Answered: 6, Total: 6, N: 2}},
	})
	if err == nil || report != nil {
		t.Fatal("expected failure without a model report")
	}
}

func TestAIStudentSummaryIsConcreteAcrossBlocks(t *testing.T) {
	high, low, medium := 4.75, 2.25, 3.5
	r, err := produceAIReport(context.Background(), &testAIModel{Mode: "error"}, aiInput{Kind: "student", Metrics: []aiMetric{
		{ID: "Q01", Name: "Ответственность", Block: "core", Value: &high, Answered: 3, Total: 3},
		{ID: "Q02", Name: "Самообладание", Block: "core", Value: &low, Answered: 3, Total: 3},
		{ID: "D01", Name: "Доступность ресурсов", Block: "periphery", Value: &medium, Answered: 3, Total: 3},
		{ID: "D02", Name: "Обратная связь", Block: "periphery", Value: &low, Answered: 3, Total: 3},
		{ID: "P01", Name: "Заявленная гражданская позиция", Block: "core", Value: &high, Answered: 3, Total: 3},
	}})
	if err == nil || r != nil {
		t.Fatal("expected failure without a model report")
	}
}

func TestAIStudentSummaryOneOrNoCompletedIndicators(t *testing.T) {
	x := 3.6
	cases := []struct {
		name    string
		metrics []aiMetric
		wants   []string
	}{
		{"one", []aiMetric{{ID: "Q03", Name: "Самостоятельность", Value: &x, Answered: 2, Total: 2}, {ID: "Q04", Name: "Настойчивость", Answered: 1, Total: 3}}, []string{"Самостоятельность", "Предварительный", "Я-концепцией"}},
		{"partial", []aiMetric{{ID: "Q01", Name: "Ответственность", Answered: 2, Total: 5}}, []string{"частичные ответы", "Ответственность", "2 из 5", "портрет пока нельзя"}},
		{"empty", []aiMetric{{ID: "Q01", Name: "Ответственность", Total: 4}}, []string{"пока не представлены", "описывать особенности"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := produceAIReport(context.Background(), &testAIModel{Mode: "error"}, aiInput{Kind: "student", Metrics: tc.metrics})
			if err == nil || r != nil {
				t.Fatal("expected error, no reserve summary")
			}
		})
	}
}

func TestAIStudentSummaryEqualScoresAndPositionsAlone(t *testing.T) {
	x := 4.0
	r, err := produceAIReport(context.Background(), nil, aiInput{Kind: "student", Metrics: []aiMetric{
		{ID: "Q01", Name: "Честность", Value: &x, Answered: 2, Total: 2},
		{ID: "Q02", Name: "Доброта", Value: &x, Answered: 2, Total: 2},
	}})
	if err == nil || r != nil {
		t.Fatal("expected failure without a model report")
	}
}
