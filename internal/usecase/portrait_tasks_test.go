package usecase

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"vucdiagnostic/internal/promptbank"
)

// goodPortrait answers the way the instruction asks, for the input it is given.
func goodPortrait(user string) map[string]any {
	var input struct {
		Levels map[string]string `json:"уровни"`
	}
	_ = json.Unmarshal([]byte(user), &input)
	potential := []any{}
	for _, area := range potentialAreas {
		name := lowerFirst(area.Title)
		potential = append(potential, map[string]any{"area": name, "level": input.Levels[name], "facts": "Студент указал опыт.", "conclusion": "Опыт может пригодиться."})
	}
	return map[string]any{
		"summary":   "Студент сообщил о себе, по сведениям, которые сообщил студент, следующее.",
		"potential": potential,
		"attention": map[string]any{"use": []any{"Опыт организатора."}, "consider": []any{"Нет спортивного опыта."}},
	}
}

// portraitModel answers with goodPortrait changed by the mutate function.
func portraitModel(mutate func(map[string]any)) *fakeModel {
	return &fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		answer := goodPortrait(c.User)
		if mutate != nil {
			mutate(answer)
		}
		data, _ := json.Marshal(answer)
		return string(data), nil
	}}
}

func newAutobiographyFixture(t *testing.T) *formFixture {
	t.Helper()
	f := newFormFixture(t)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	f.service.RegisterPrompts(prompts...)
	f.service.portraits.pause = 0
	t.Cleanup(f.service.Close)
	return f
}

func (f *formFixture) submit() {
	f.t.Helper()
	f.consent()
	f.fillAll(nil)
	if _, err := f.command("completion", Input{}); err != nil {
		f.t.Fatal(err)
	}
}

// another is a second student of the same assignment, with the same answers.
func (f *formFixture) another(email string) *formFixture {
	f.t.Helper()
	login := signUp(f.t, f.service, "student", email, "101")
	return &formFixture{t: f.t, service: f.service, form: f.form, token: login.Token, student: login.Account, assignment: f.assignment, submission: addSubmission(f.t, f.service, login.Account.ID, f.assignment)}
}

func (f *formFixture) idle() {
	f.t.Helper()
	(&portraitFixture{t: f.t, service: f.service}).idle()
}

func (f *formFixture) portrait(submission string) PortraitView {
	f.t.Helper()
	view, err := f.service.PortraitFor(f.student.OwnerID, "autobiography", submission)
	if err != nil {
		f.t.Fatal(err)
	}
	return view
}

func TestSubmittedAutobiographyGetsAPortraitOnceAndTheSameForEqualAnswers(t *testing.T) {
	f := newAutobiographyFixture(t)
	m := portraitModel(nil)
	f.service.UseModel(m)
	f.submit()
	f.idle()
	view := f.portrait(f.submission)
	if view.Status != portraitReady || view.Portrait == nil || m.count() != 1 {
		t.Fatalf("%+v after %d calls", view, m.count())
	}
	call := m.calls[0]
	if !strings.HasPrefix(call.System, "Ты помогаешь преподавателю военного учебного центра") || !strings.Contains(string(call.Schema), `"required": ["summary", "potential", "attention"]`) {
		t.Fatalf("instruction or schema: %.80s", call.System)
	}
	// What was sent is the anonymised input and nothing of the person.
	db := mustSnapshot(t, f.service.repo)
	for _, banned := range []string{f.student.Name, f.student.Email, "Тестов"} {
		if strings.Contains(call.User, banned) {
			t.Fatalf("%q reached the model", banned)
		}
	}
	if db.Portraits[0].Input != call.User || !strings.HasPrefix(call.User, "{\n \"возраст\"") {
		t.Fatalf("input: %.60s", call.User)
	}

	second := f.another("second@example.test")
	second.submit()
	f.idle()
	again := f.portrait(second.submission)
	if m.count() != 1 || again.Status != portraitReady || string(again.Portrait.Output) != string(view.Portrait.Output) || len(mustSnapshot(t, f.service.repo).Portraits) != 1 {
		t.Fatalf("second student: %+v after %d calls", again, m.count())
	}
}

func TestAutobiographyPortraitIsMissingBeforeSubmissionAndHiddenFromOtherTeachers(t *testing.T) {
	f := newAutobiographyFixture(t)
	f.service.UseModel(portraitModel(nil))
	if _, err := f.service.PortraitFor(f.student.OwnerID, "autobiography", f.submission); kind(err) != NotFound {
		t.Fatalf("a draft has a portrait: %v", err)
	}
	f.submit()
	f.idle()
	for name, call := range map[string]func() error{
		"get": func() error {
			_, err := f.service.PortraitFor("another-teacher", "autobiography", f.submission)
			return err
		},
		"retry": func() error {
			_, err := f.service.RetryPortrait("another-teacher", "autobiography", f.submission)
			return err
		},
	} {
		if kind(call()) != NotFound {
			t.Errorf("%s by another teacher is not 404", name)
		}
	}
	if view, err := f.service.PortraitFor("", "autobiography", f.submission); err != nil || view.Status != portraitReady {
		t.Fatalf("without a teacher scope: %+v %v", view, err)
	}
}

func TestSubmissionWorksWithoutAModel(t *testing.T) {
	f := newAutobiographyFixture(t)
	f.submit()
	view := f.portrait(f.submission)
	if view.Status != portraitOff || view.Portrait != nil {
		t.Fatalf("%+v", view)
	}
	m := portraitModel(nil)
	f.service.UseModel(m)
	f.idle()
	if m.count() != 1 || f.portrait(f.submission).Status != portraitReady {
		t.Fatalf("not made at start: %d calls", m.count())
	}
}

func TestAnswerThatBreaksTheRulesFailsAfterThreeTries(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate  func(map[string]any)
		problem string
	}{
		"changed level": {func(a map[string]any) {
			a["potential"].([]any)[0].(map[string]any)["level"] = "высокий-превосходный"
		}, "уровень изменён"},
		"six points": {func(a map[string]any) {
			a["attention"].(map[string]any)["consider"] = []any{"1", "2", "3", "4", "5", "6"}
		}, "пунктов больше пяти"},
		"five areas": {func(a map[string]any) { a["potential"] = a["potential"].([]any)[:5] }, "направления не те"},
		"other order": {func(a map[string]any) {
			p := a["potential"].([]any)
			p[0], p[1] = p[1], p[0]
		}, "в другом порядке"},
		"no summary":   {func(a map[string]any) { a["summary"] = " " }, "нет характеристики"},
		"empty facts":  {func(a map[string]any) { a["potential"].([]any)[2].(map[string]any)["facts"] = "" }, "пустое поле facts"},
		"no attention": {func(a map[string]any) { delete(a, "attention") }, "нет блока"},
		"use not text": {func(a map[string]any) { a["attention"].(map[string]any)["use"] = []any{1} }, "не список строк"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newAutobiographyFixture(t)
			m := portraitModel(tc.mutate)
			f.service.UseModel(m)
			f.submit()
			f.idle()
			view := f.portrait(f.submission)
			if m.count() != 3 || view.Status != portraitFailed || !strings.Contains(view.Reason, tc.problem) {
				t.Fatalf("%d calls, %+v", m.count(), view)
			}
			// «Повторить» with a good model makes the portrait.
			m.reply = portraitModel(nil).reply
			if retry, err := f.service.RetryPortrait(f.student.OwnerID, "autobiography", f.submission); err != nil || retry.Status == portraitFailed {
				t.Fatalf("retry: %+v %v", retry, err)
			}
			f.idle()
			if f.portrait(f.submission).Status != portraitReady || m.count() != 4 {
				t.Fatalf("after retry: %+v, %d calls", f.portrait(f.submission), m.count())
			}
		})
	}
}

func TestNewInstructionVersionMakesANewPortraitAndKeepsTheOld(t *testing.T) {
	f := newAutobiographyFixture(t)
	m := portraitModel(nil)
	f.service.UseModel(m)
	f.submit()
	f.idle()
	task, _ := f.service.portraits.task("autobiography")
	next := task
	next.Version = "v2"
	f.service.portraits.tasks = []portraitTask{next}
	f.service.UseModel(m)
	f.idle()
	db := mustSnapshot(t, f.service.repo)
	if m.count() != 2 || len(db.Portraits) != 2 || db.Portraits[0].TaskVersion != "v1" || db.Portraits[1].TaskVersion != "v2" || db.Portraits[0].ID == db.Portraits[1].ID || db.Portraits[0].Fingerprint != db.Portraits[1].Fingerprint {
		t.Fatalf("%d calls: %+v", m.count(), db.Portraits)
	}
}

func TestPortraitUsesTheLevelsOfTheProgramAndFindsThemInTheInput(t *testing.T) {
	f := newAutobiographyFixture(t)
	f.submit()
	db := mustSnapshot(t, f.service.repo)
	payload, _, err := submissionPayload(f.form, db.Submissions[0])
	if err != nil {
		t.Fatal(err)
	}
	levels := autobiographyPotential(f.form, db.Submissions[0].Answers)
	want := []string{}
	for _, area := range levels.Areas {
		want = append(want, lowerFirst(area.Title)+"="+area.LevelText)
	}
	got := []string{}
	for _, key := range payload.values["уровни"].(*object).keys {
		got = append(got, fmt.Sprintf("%s=%s", key, payload.values["уровни"].(*object).values[key]))
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("%v and %v", got, want)
	}
	if problems := checkAutobiographyAnswer(goodPortrait(string(canonicalJSON(payload))), payload); len(problems) != 0 {
		t.Fatalf("a good answer fails: %v", problems)
	}
}

func TestWaitingForANewPortraitDoesNotBlockTheSubmission(t *testing.T) {
	f := newAutobiographyFixture(t)
	release := make(chan struct{})
	slow := &fakeModel{name: "slow", reply: func(call int, c Completion) (string, error) {
		<-release
		data, _ := json.Marshal(goodPortrait(c.User))
		return string(data), nil
	}}
	f.service.UseModel(slow)
	started := time.Now()
	f.submit()
	if time.Since(started) > 2*time.Second {
		t.Fatal("the submission waited for the model")
	}
	if view := f.portrait(f.submission); view.Status != portraitQueued && view.Status != portraitWorking {
		t.Fatalf("%+v", view)
	}
	close(release)
	f.idle()
	if f.portrait(f.submission).Status != portraitReady {
		t.Fatal("not ready")
	}
}
