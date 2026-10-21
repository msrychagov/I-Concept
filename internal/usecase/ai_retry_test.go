package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"vucdiagnostic/internal/domain/model"
)

func TestAIRetryBypassesReadyAndFailedCache(t *testing.T) {
	f := newPortraitFixture(t, nil)
	id := f.finish("retry@example.test")
	err := f.service.repo.Update(func(db *model.Database) error {
		for i := range db.Sessions {
			s := &db.Sessions[i].Session
			if s.ID == id {
				for _, item := range f.service.itemsFor(*s) {
					s.Answers[item.ID] = item.Min
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	m := &fakeModel{name: "retry-model", reply: func(int, Completion) (string, error) { return "", errors.New("offline") }}
	f.service.UseAnalysisModel(m)
	request := func(start bool) AIReportState {
		t.Helper()
		state, err := f.service.AIAnalysisFor(f.teacherID, "student", id, start)
		if err != nil {
			t.Fatal(err)
		}
		return state
	}
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			state := request(false)
			if state.Status == "failed" {
				if state.Report != nil || state.Reason != errAIReport.Error() {
					t.Fatal(state)
				}
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("generation did not finish")
	}
	if request(true).Status != "working" {
		t.Fatal("not started")
	}
	wait()
	if m.count() != 1 {
		t.Fatal(m.count())
	}
	request(false)
	if m.count() != 1 {
		t.Fatal("GET regenerated")
	}
	request(true)
	wait()
	if m.count() != 2 {
		t.Fatal("failed cache reused")
	}
	f.service.ai.mu.Lock()
	for key := range f.service.ai.jobs {
		f.service.ai.jobs[key] = AIReportState{Status: "ready", Report: &AIReport{Summary: "old cached text"}}
	}
	f.service.ai.mu.Unlock()
	if request(true).Status != "working" {
		t.Fatal("ready cache reused")
	}
	wait()
	if m.count() != 3 {
		t.Fatal("no new model call")
	}
}

func TestAIUnstructuredSummaryIsPublished(t *testing.T) {
	v := 3.5
	input := aiInput{Kind: "student", Metrics: []aiMetric{{ID: "Q05", Name: "Ответственность", Value: &v, Answered: 3, Total: 3}}}
	for _, reply := range []string{"", `{}`, `{"portrait":"invalid","actions":[]}`} {
		m := &fakeModel{name: "partial", reply: func(n int, c Completion) (string, error) {
			return reply, nil
		}}
		report, err := produceAIReport(context.Background(), m, input)
		if err != nil || report == nil || m.count() != 2 || len(report.ModelResponses) != 2 || report.ModelResponses[1] != reply {
			t.Fatal("model response lost", report, err, m.count())
		}
	}
}

func TestAIPlatoonUsesOnlyModelNarrative(t *testing.T) {
	v := 3.5
	input := aiInput{Kind: "platoon", Included: 3, Participants: 3, Metrics: []aiMetric{{ID: "Q05", Name: "Ответственность", Value: &v, Answered: 9, Total: 9, N: 3}}}
	spy := &chapterSpyModel{}
	report, err := produceAIReport(context.Background(), spy, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(spy.Prompts) != 2 || report.GenerationMode != "ai-assisted" {
		t.Fatal("missing generated group report")
	}
	if !strings.Contains(spy.Prompts[0].User, `"kind":"platoon"`) || !strings.Contains(spy.Prompts[0].System, "агрегированные показатели взвода") {
		t.Fatal("individual prompt used for group")
	}
	if !strings.Contains(report.Limitations, "Агрегированные") {
		t.Fatal("group limitations lost")
	}
}

func TestAIPreservesAnswerWhenNextRequestFails(t *testing.T) {
	v := 3.5
	input := aiInput{Kind: "student", Metrics: []aiMetric{{ID: "Q05", Name: "Ответственность", Value: &v, Answered: 3, Total: 3}}}
	answer := "Свободный текст вместо JSON <script>alert(1)</script>"
	m := &fakeModel{name: "partial", reply: func(n int, _ Completion) (string, error) {
		if n == 1 {
			return answer, nil
		}
		return "", errors.New("offline")
	}}
	r, err := produceAIReport(context.Background(), m, input)
	if err != nil || r == nil {
		t.Fatalf("partial answer lost: %v", err)
	}
	if len(r.ModelResponses) != 1 || r.ModelResponses[0] != answer || r.GenerationError == "" {
		t.Fatalf("partial answer not preserved: %+v", r)
	}
}
