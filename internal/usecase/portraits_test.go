package usecase

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// fakeModel answers from a function of the call number and records what it got.
type fakeModel struct {
	name  string
	reply func(call int, c Completion) (string, error)
	mu    sync.Mutex
	calls []Completion
}

func (m *fakeModel) Name() string { return m.name }

func (m *fakeModel) Complete(ctx context.Context, c Completion) (Reply, error) {
	m.mu.Lock()
	m.calls = append(m.calls, c)
	n := len(m.calls)
	m.mu.Unlock()
	text, err := m.reply(n, c)
	return Reply{Text: text, Version: m.name + "-1", In: 10, Out: 5, Seconds: 0.5}, err
}

func (m *fakeModel) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func answering(text string) *fakeModel {
	return &fakeModel{name: "fake", reply: func(int, Completion) (string, error) { return text, nil }}
}

const goodAnswer = `{"summary":"Текст вывода."}`

// sessionTask is a task over the finished sessions of "Я-концепции": it stands
// for the tasks to come and proves that the layer knows nothing about forms.
func sessionTask(extra func(*object)) portraitTask {
	find := func(db model.Database, s Source) (model.Session, bool) {
		if s.Kind != "session" {
			return model.Session{}, false
		}
		for _, record := range db.Sessions {
			if record.Session.ID == s.ID {
				return record.Session, true
			}
		}
		return model.Session{}, false
	}
	return portraitTask{
		ID: "test-session", Version: "v1",
		Prompt: promptbank.Prompt{Task: "test-session", Version: "v1", Instruction: "Опиши сессию.", Schema: []byte(`{"type":"object"}`)},
		Accepts: func(db model.Database, s Source) bool {
			session, ok := find(db, s)
			return ok && session.Completed
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) {
			session, _ := find(db, s)
			payload := newObject()
			payload.set("ответов", len(session.Answers))
			if extra != nil {
				extra(payload)
			}
			return payload, nil
		},
		Check: func(output map[string]any, payload *object) []string {
			if text, _ := output["summary"].(string); text == "" {
				return []string{"нет текста"}
			}
			return nil
		},
		Visible: func(db model.Database, s Source, teacherID string) bool {
			session, ok := find(db, s)
			return ok && teacherOwnsSession(db, teacherID, session)
		},
	}
}

type portraitFixture struct {
	t         *testing.T
	service   *Service
	teacherID string
	assigned  string
}

func newPortraitFixture(t *testing.T, extra func(*object)) *portraitFixture {
	t.Helper()
	s := testApp(t)
	s.portraits.pause = 0
	s.registerPortraitTask(sessionTask(extra))
	t.Cleanup(s.Close)
	first := signUp(t, s, "student", "first@example.test", "101")
	p := s.emptyProfile()
	p.ID, p.Name, p.Author, p.OwnerID = "template", "Шаблон", "Тест", first.Account.OwnerID
	if err := s.addProfile(p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AssignmentCommandFor(first.Account.OwnerID, Input{"title": raw(t, "Тест"), "profileId": raw(t, "template")}, "assignment"); err != nil {
		t.Fatal(err)
	}
	return &portraitFixture{t: t, service: s, teacherID: first.Account.OwnerID, assigned: mustSnapshot(t, s.repo).Assignments[0].ID}
}

// finish starts the assignment as a new student and finishes the session.
func (f *portraitFixture) finish(email string) string {
	f.t.Helper()
	login := signUp(f.t, f.service, "student", email, "101")
	if _, err := f.service.StartAssignment(login.Account, f.assigned); err != nil {
		f.t.Fatal(err)
	}
	var id string
	for _, record := range mustSnapshot(f.t, f.service.repo).Sessions {
		if record.Session.StudentID == login.Account.ID {
			id = record.Session.ID
		}
	}
	if _, err := f.service.SurveyCommand(login.Token, id, Input{"revision": raw(f.t, 0), "value": raw(f.t, true)}, "consent"); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.service.SurveyCommand(login.Token, id, Input{"revision": raw(f.t, 1), "allowPartial": raw(f.t, true)}, "finish"); err != nil {
		f.t.Fatal(err)
	}
	return id
}

// idle waits until the queue is empty and nothing is being made.
func (f *portraitFixture) idle() {
	f.t.Helper()
	l := f.service.portraits
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		l.mu.Lock()
		busy := len(l.queue) > 0
		for _, state := range l.states {
			busy = busy || state.Status == portraitQueued || state.Status == portraitWorking
		}
		l.mu.Unlock()
		if !busy {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.t.Fatal("the queue did not become empty")
}

func (f *portraitFixture) view(sessionID string) PortraitView {
	f.t.Helper()
	view, err := f.service.PortraitFor(f.teacherID, "test-session", sessionID)
	if err != nil {
		f.t.Fatal(err)
	}
	return view
}

func TestFinishedSessionGetsAPortraitOnceForEqualInput(t *testing.T) {
	f := newPortraitFixture(t, nil)
	m := answering(goodAnswer)
	f.service.UseModel(m)
	first := f.finish("a@example.test")
	f.idle()
	view := f.view(first)
	if view.Status != portraitReady || view.Portrait == nil || string(view.Portrait.Output) != goodAnswer {
		t.Fatalf("view: %+v", view)
	}
	if m.count() != 1 || !strings.HasPrefix(m.calls[0].System, "Опиши сессию") || !strings.Contains(m.calls[0].User, `"ответов": 0`) {
		t.Fatalf("calls: %+v", m.calls)
	}
	// Another student with the same answers: the stored portrait, no new call.
	second := f.finish("b@example.test")
	f.idle()
	if again := f.view(second); again.Status != portraitReady || string(again.Portrait.Output) != goodAnswer || again.Portrait.Created != view.Portrait.Created {
		t.Fatalf("second view: %+v", again)
	}
	f.view(first)
	if m.count() != 1 {
		t.Fatalf("the model was called %d times", m.count())
	}
	db := mustSnapshot(t, f.service.repo)
	if len(db.Portraits) != 1 {
		t.Fatalf("%d portraits", len(db.Portraits))
	}
	stored := db.Portraits[0]
	if stored.Task != "test-session" || stored.TaskVersion != "v1" || stored.Model != "fake" || stored.ModelVersion != "fake-1" || stored.Usage.In != 10 || stored.Usage.Out != 5 || stored.Input != m.calls[0].User || stored.Fingerprint == "" || stored.Fingerprint == stored.ID {
		t.Fatalf("stored: %+v", stored)
	}
}

func TestPortraitIsRestoredAtStartAndKeptAcrossRestarts(t *testing.T) {
	f := newPortraitFixture(t, nil)
	id := f.finish("a@example.test")
	if got := f.view(id); got.Status != portraitOff {
		t.Fatalf("no model: %+v", got)
	}
	m := answering(goodAnswer)
	f.service.UseModel(m)
	f.idle()
	if m.count() != 1 || f.view(id).Status != portraitReady {
		t.Fatalf("at start: %d calls, %+v", m.count(), f.view(id))
	}
	// A restart with the same model: nothing to make.
	f.service.UseModel(m)
	f.idle()
	if m.count() != 1 {
		t.Fatalf("restart called the model: %d", m.count())
	}
	// Another model is another portrait, the first one stays.
	other := answering(goodAnswer)
	other.name = "other"
	f.service.UseModel(other)
	f.idle()
	if other.count() != 1 || len(mustSnapshot(t, f.service.repo).Portraits) != 2 {
		t.Fatalf("another model: %d calls, %d portraits", other.count(), len(mustSnapshot(t, f.service.repo).Portraits))
	}
	f.service.UseModel(nil)
	if f.view(id).Status != portraitOff {
		t.Fatal("model removed but status is not off")
	}
}

func TestBadAnswersTryThreeTimesThenFailAndRetryQueuesAgain(t *testing.T) {
	for name, answer := range map[string]string{
		"broken JSON":   `{"summary": `,
		"not an object": `[1]`,
		"fails check":   `{"summary":""}`,
	} {
		t.Run(name, func(t *testing.T) {
			f := newPortraitFixture(t, nil)
			m := answering(answer)
			f.service.UseModel(m)
			id := f.finish("a@example.test")
			f.idle()
			view := f.view(id)
			if m.count() != 3 || view.Status != portraitFailed || view.Reason == "" || view.Portrait != nil {
				t.Fatalf("%d calls, %+v", m.count(), view)
			}
			if len(mustSnapshot(t, f.service.repo).Portraits) != 0 {
				t.Fatal("a bad answer was stored")
			}
			// Polling does not restart a failed portrait.
			f.view(id)
			f.idle()
			if m.count() != 3 {
				t.Fatalf("a poll called the model: %d", m.count())
			}
			m.reply = func(int, Completion) (string, error) { return "```json\n" + goodAnswer + "\n```", nil }
			if retry, err := f.service.RetryPortrait(f.teacherID, "test-session", id); err != nil || retry.Status != portraitQueued && retry.Status != portraitWorking {
				t.Fatalf("retry: %+v %v", retry, err)
			}
			f.idle()
			if view = f.view(id); view.Status != portraitReady || string(view.Portrait.Output) != goodAnswer {
				t.Fatalf("after retry: %+v", view)
			}
			// Retry of a ready portrait does not call the model.
			calls := m.count()
			if again, _ := f.service.RetryPortrait(f.teacherID, "test-session", id); again.Status != portraitReady || m.count() != calls {
				t.Fatalf("retry of a ready one: %+v, %d calls", again, m.count())
			}
		})
	}
}

func TestAnErrorOfTheModelIsRetriedAndOneGoodAttemptIsEnough(t *testing.T) {
	f := newPortraitFixture(t, nil)
	m := &fakeModel{name: "fake", reply: func(call int, _ Completion) (string, error) {
		if call < 3 {
			return "", errors.New("timeout")
		}
		return goodAnswer, nil
	}}
	f.service.UseModel(m)
	id := f.finish("a@example.test")
	f.idle()
	if m.count() != 3 || f.view(id).Status != portraitReady {
		t.Fatalf("%d calls, %+v", m.count(), f.view(id))
	}
}

func TestInputWithPersonalDataIsNotSent(t *testing.T) {
	for name, text := range map[string]string{
		"patronymic": "Работал у Петрова Ивана Сергеевича",
		"full date":  "родился 01.02.2005",
		"iso date":   "с 2005-02-01",
		"email":      "ivan@example.test",
		"phone":      "+7 912 345-67-89",
	} {
		t.Run(name, func(t *testing.T) {
			f := newPortraitFixture(t, func(o *object) { o.set("заметка", text) })
			m := answering(goodAnswer)
			f.service.UseModel(m)
			id := f.finish("a@example.test")
			f.idle()
			view := f.view(id)
			if m.count() != 0 || view.Status != portraitFailed || !strings.Contains(view.Reason, "личные данные") {
				t.Fatalf("%d calls, %+v", m.count(), view)
			}
		})
	}
}

func TestPersonalDataPatterns(t *testing.T) {
	object := func(text string) *object {
		o := newObject()
		o.set("поле", []string{"безобидное", text})
		return o
	}
	for text, want := range map[string]string{
		"Иванович":                         "отчество",
		"сын Сергеевича":                   "отчество",
		"Ильинична":                        "отчество",
		"позвонить 8 (912) 345-67-89":      "телефон",
		"31.12.2005":                       "полная дата",
		"a.b-c@mail.ru":                    "почта",
		"с 2023-06 по 2024-08":             "",
		"Нижний Новгород, школа 12":        "",
		"бригада 2005 года":                "",
		"Победитель олимпиады, 3 место":    "",
		"Командир отделения в 2023 году":   "",
		"Работал в Сбербанке":              "",
		"иванович в начале строчной буквы": "",
	} {
		if got := personalData(object(text)); got != want {
			t.Errorf("%q: %q, want %q", text, got, want)
		}
	}
	// Keys are not checked, only values.
	o := newObject()
	o.set("2005-02-01", "значение")
	if personalData(o) != "" {
		t.Error("a key was checked")
	}
}

func TestAForeignTeacherAndAnUnknownTaskGetNotFound(t *testing.T) {
	f := newPortraitFixture(t, nil)
	f.service.UseModel(answering(goodAnswer))
	id := f.finish("a@example.test")
	f.idle()
	for name, call := range map[string]func() error{
		"foreign teacher": func() error { _, err := f.service.PortraitFor("another-teacher", "test-session", id); return err },
		"foreign retry":   func() error { _, err := f.service.RetryPortrait("another-teacher", "test-session", id); return err },
		"unknown task":    func() error { _, err := f.service.PortraitFor(f.teacherID, "nope", id); return err },
		"unknown source":  func() error { _, err := f.service.PortraitFor(f.teacherID, "test-session", "missing"); return err },
	} {
		if kind(call()) != NotFound {
			t.Errorf("%s is not 404", name)
		}
	}
}

func TestACopyOfTheServiceSharesTheLayerAndCloseStopsTheWorker(t *testing.T) {
	f := newPortraitFixture(t, nil)
	f.service.UseModel(answering(goodAnswer))
	copied := *f.service
	if copied.portraits != f.service.portraits {
		t.Fatal("the copy has its own layer")
	}
	f.service.Close()
	f.service.Close()
}

// journal collects what the program writes to the log while a test runs.
type journal struct {
	mu   sync.Mutex
	text strings.Builder
}

func (j *journal) Write(p []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.text.Write(p)
}

func (j *journal) String() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.text.String()
}

func captureJournal(t *testing.T) *journal {
	t.Helper()
	j := &journal{}
	log.SetOutput(j)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return j
}

func TestJournalSaysWhyAPortraitFailedButNotWhatWasSent(t *testing.T) {
	for name, tc := range map[string]struct {
		reply func(int, Completion) (string, error)
		want  string
	}{
		"client error": {func(int, Completion) (string, error) { return "", errors.New("gigachat: HTTP 401 Unauthorized") }, "HTTP 401"},
		"bad answer":   {func(int, Completion) (string, error) { return `{"summary":""}`, nil }, "нет текста"},
		"unreadable":   {func(int, Completion) (string, error) { return "not json", nil }, "ответ не объект JSON"},
	} {
		t.Run(name, func(t *testing.T) {
			out := captureJournal(t)
			f := newPortraitFixture(t, func(o *object) { o.set("возраст", 19) })
			m := &fakeModel{name: "fake", reply: tc.reply}
			f.service.UseModel(m)
			id := f.finish("a@example.test")
			f.idle()
			view := f.view(id)
			if view.Status != portraitFailed || strings.Contains(view.Reason, "HTTP") {
				t.Fatalf("the teacher sees %+v", view)
			}
			text := out.String()
			if !strings.Contains(text, "failed") || !strings.Contains(text, tc.want) || !strings.Contains(text, "attempt 3") {
				t.Fatalf("journal: %s", text)
			}
			if strings.Contains(text, "возраст") || strings.Contains(text, `"ответов"`) {
				t.Fatalf("the input is in the journal: %s", text)
			}
		})
	}
}
