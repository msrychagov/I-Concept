package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// Portraits: texts a language model writes from anonymised data. This layer
// knows nothing about what is written: a task (portrait_tasks.go) says which
// sources it takes, how to build the input and how to check the answer.
//
// A portrait is stored under the hash of task, task version, model and input.
// Equal input gives the stored text and the model is not called again; a new
// instruction version or another model gives a new portrait and the old stays.

// Source is what a portrait is about: a submitted form ("submission") or a
// finished test session ("session").
type Source struct{ Kind, ID string }

type portraitTask struct {
	ID, Version string
	Prompt      promptbank.Prompt
	// Accepts says whether the task takes this source.
	Accepts func(db model.Database, s Source) bool
	// Payload builds the anonymised input.
	Payload func(a *Service, db model.Database, s Source) (*object, error)
	// Check returns the problems of an answer; an empty list means it is good.
	Check func(output map[string]any, payload *object) []string
	// Visible says whether this teacher may see the portrait of the source.
	Visible func(db model.Database, s Source, teacherID string) bool
}

const (
	portraitQueued  = "queued"
	portraitWorking = "working"
	portraitFailed  = "failed"
	portraitReady   = "ready"
	portraitOff     = "off"

	portraitAttempts = 3
)

// portraitState is what is known about a portrait that is not stored yet.
type portraitState struct{ Status, Reason string }

type portraitJob struct {
	task   string
	source Source
	key    string
}

// portraitLayer is shared by the copies of a Service. One worker takes the
// jobs one by one: this keeps the load on a free plan low.
type portraitLayer struct {
	mu     sync.Mutex
	tasks  []portraitTask
	model  TextModel
	queue  []portraitJob
	states map[string]*portraitState
	wake   chan struct{}
	stop   context.CancelFunc
	done   chan struct{}
	// pause is the wait between attempts of one portrait.
	pause time.Duration
}

func newPortraitLayer() *portraitLayer {
	return &portraitLayer{states: map[string]*portraitState{}, wake: make(chan struct{}, 1), pause: 2 * time.Second}
}

// UseModel connects the model, starts the worker and queues the portraits
// that are missing. nil means no model: portraits are neither made nor asked for.
func (a *Service) UseModel(m TextModel) {
	l := a.portraits
	l.shutdown()
	l.mu.Lock()
	l.model = m
	l.queue = nil
	l.states = map[string]*portraitState{}
	if m == nil {
		l.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.stop, l.done = cancel, make(chan struct{})
	l.mu.Unlock()
	go a.runPortraits(ctx, l.done)
	a.queueMissingPortraits()
}

// Close stops the worker; a call to the model that is under way is cancelled.
func (a *Service) Close() { a.portraits.shutdown() }

func (l *portraitLayer) shutdown() {
	l.mu.Lock()
	stop, done := l.stop, l.done
	l.stop, l.done = nil, nil
	l.mu.Unlock()
	if stop != nil {
		stop()
		<-done
	}
}

func (l *portraitLayer) task(id string) (portraitTask, bool) {
	for _, task := range l.tasks {
		if task.ID == id {
			return task, true
		}
	}
	return portraitTask{}, false
}

func (a *Service) registerPortraitTask(task portraitTask) {
	a.portraits.tasks = append(a.portraits.tasks, task)
}

// portraitKey is the hash of task, version, model and the canonical input.
func portraitKey(task portraitTask, modelName string, payload *object) string {
	sum := sha256.Sum256([]byte(task.ID + "\n" + task.Version + "\n" + modelName + "\n" + string(canonicalJSON(payload))))
	return hex.EncodeToString(sum[:])
}

func findPortrait(db model.Database, key string) (model.Portrait, bool) {
	for _, portrait := range db.Portraits {
		if portrait.ID == key {
			return portrait, true
		}
	}
	return model.Portrait{}, false
}

// prepare builds the input of a source and the key of its portrait.
func (a *Service) prepare(db model.Database, task portraitTask, s Source, modelName string) (*object, string, error) {
	payload, err := task.Payload(a, db, s)
	if err != nil {
		return nil, "", err
	}
	return payload, portraitKey(task, modelName, payload), nil
}

// queuePortraits queues the portrait of a source for every task that takes it.
// It is called after the source is saved and never fails the caller.
func (a *Service) queuePortraits(s Source) {
	l := a.portraits
	l.mu.Lock()
	m := l.model
	l.mu.Unlock()
	if m == nil {
		return
	}
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		log.Printf("portrait: snapshot: %v", err)
		return
	}
	for _, task := range l.tasks {
		if task.Accepts(db, s) {
			a.enqueuePortrait(db, task, s, false)
		}
	}
}

// queueMissingPortraits goes over all sources: it covers the start of the
// program and a new version of an instruction or another model.
func (a *Service) queueMissingPortraits() {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		log.Printf("portrait: snapshot: %v", err)
		return
	}
	sources := []Source{}
	for _, submission := range db.Submissions {
		sources = append(sources, Source{"submission", submission.ID})
	}
	for _, record := range db.Sessions {
		sources = append(sources, Source{"session", record.Session.ID})
	}
	for _, s := range sources {
		for _, task := range a.portraits.tasks {
			if task.Accepts(db, s) {
				a.enqueuePortrait(db, task, s, false)
			}
		}
	}
}

// enqueuePortrait puts one job into the queue unless the portrait exists or is
// already on its way. A failed one is queued again only when retry is set.
func (a *Service) enqueuePortrait(db model.Database, task portraitTask, s Source, retry bool) portraitState {
	l := a.portraits
	l.mu.Lock()
	m := l.model
	l.mu.Unlock()
	if m == nil {
		return portraitState{Status: portraitOff}
	}
	_, key, err := a.prepare(db, task, s, m.Name())
	if err != nil {
		return portraitState{Status: portraitFailed, Reason: "Не удалось собрать сведения для модели."}
	}
	if _, ok := findPortrait(db, key); ok {
		return portraitState{Status: portraitReady}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if state, ok := l.states[key]; ok && (state.Status != portraitFailed || !retry) {
		return *state
	}
	state := &portraitState{Status: portraitQueued}
	l.states[key] = state
	l.queue = append(l.queue, portraitJob{task: task.ID, source: s, key: key})
	select {
	case l.wake <- struct{}{}:
	default:
	}
	return *state
}

func (a *Service) runPortraits(ctx context.Context, done chan struct{}) {
	defer close(done)
	l := a.portraits
	for {
		l.mu.Lock()
		var job portraitJob
		found := len(l.queue) > 0
		if found {
			job, l.queue = l.queue[0], l.queue[1:]
		}
		l.mu.Unlock()
		if !found {
			select {
			case <-ctx.Done():
				return
			case <-l.wake:
				continue
			}
		}
		if ctx.Err() != nil {
			return
		}
		a.ensurePortrait(ctx, job)
	}
}

func (l *portraitLayer) setState(key string, state *portraitState) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if state == nil {
		delete(l.states, key)
	} else {
		l.states[key] = state
	}
}

// ensurePortrait makes the portrait of one job: it sends nothing that looks
// like personal data, calls the model up to three times until the answer
// passes the check, and stores the result unless a portrait with the key exists.
func (a *Service) ensurePortrait(ctx context.Context, job portraitJob) {
	l := a.portraits
	l.mu.Lock()
	m := l.model
	l.mu.Unlock()
	task, known := l.task(job.task)
	db, err := a.repo.ReadSnapshot()
	if m == nil || !known || err != nil {
		l.setState(job.key, &portraitState{Status: portraitFailed, Reason: "Не удалось прочитать данные."})
		return
	}
	payload, key, err := a.prepare(db, task, job.source, m.Name())
	if err != nil {
		l.setState(job.key, &portraitState{Status: portraitFailed, Reason: "Не удалось собрать сведения для модели."})
		return
	}
	if key != job.key {
		l.setState(job.key, nil)
	}
	if _, ok := findPortrait(db, key); ok {
		l.setState(key, nil)
		return
	}
	// reason is for the teacher and stays neutral; detail is for the journal. It
	// never holds the input or the text of the answer: a client error is cleaned
	// of the key and the token by the client.
	fail := func(reason, detail string, seconds float64) {
		l.setState(key, &portraitState{Status: portraitFailed, Reason: reason})
		if runes := []rune(detail); len(runes) > 300 {
			detail = string(runes[:300]) + "…"
		}
		log.Printf("portrait %s %.12s: failed (%.1f s): %s", task.ID, key, seconds, detail)
	}
	if label := personalData(payload); label != "" {
		fail("В сведениях найдено похожее на личные данные, отправка отменена.", "personal data in the input: "+label, 0)
		return
	}
	l.setState(key, &portraitState{Status: portraitWorking})
	user := modelText(canonicalJSON(payload))
	reason, detail := "Модель не ответила.", "no attempt was made"
	started := time.Now()
	for attempt := 1; attempt <= portraitAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(l.pause):
			}
		}
		reply, err := m.Complete(ctx, Completion{System: task.Prompt.Instruction, User: user, Schema: task.Prompt.Schema})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			reason, detail = "Модель не ответила.", fmt.Sprintf("attempt %d: %v", attempt, err)
			continue
		}
		output, problems := parseModelAnswer(reply.Text)
		if len(problems) == 0 {
			problems = task.Check(output, payload)
		}
		if len(problems) > 0 {
			reason = "Ответ модели не прошёл проверку: " + strings.Join(problems[:min(len(problems), 3)], "; ") + "."
			detail = fmt.Sprintf("attempt %d: %s", attempt, strings.Join(problems, "; "))
			continue
		}
		data, _ := json.Marshal(output)
		portrait := model.Portrait{
			ID: key, Task: task.ID, TaskVersion: task.Version, Fingerprint: payloadFingerprint(payload),
			Model: m.Name(), ModelVersion: reply.Version, Created: time.Now().UTC().Format(time.RFC3339),
			Input: user, Output: data, Usage: model.PortraitUsage{In: reply.In, Out: reply.Out}, Seconds: reply.Seconds,
		}
		if err = a.repo.Update(func(db *model.Database) error {
			if _, exists := findPortrait(*db, key); !exists {
				db.Portraits = append(db.Portraits, portrait)
			}
			return nil
		}); err != nil {
			fail("Не удалось сохранить вывод.", "store: "+err.Error(), time.Since(started).Seconds())
			return
		}
		l.setState(key, nil)
		log.Printf("portrait %s %.12s: ready (%.1f s, %d+%d tokens)", task.ID, key, reply.Seconds, reply.In, reply.Out)
		return
	}
	fail(reason, detail, time.Since(started).Seconds())
}

// modelText turns the canonical input into the text the model gets: the same
// JSON with an indent of one space, as in the comparison of the models.
func modelText(canonical []byte) string {
	var out bytes.Buffer
	if json.Indent(&out, canonical, "", " ") != nil {
		return string(canonical)
	}
	return out.String()
}

var answerFence = regexp.MustCompile("^```(?:json)?\\s*|\\s*```$")

// parseModelAnswer reads the answer as a JSON object, with or without a
// ```json fence around it.
func parseModelAnswer(text string) (map[string]any, []string) {
	cleaned := answerFence.ReplaceAllString(strings.TrimSpace(text), "")
	var output map[string]any
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil || output == nil {
		return nil, []string{"ответ не объект JSON"}
	}
	return output, nil
}

// Personal data must not reach the model whatever the form lets through. Go
// \b knows only ASCII letters, so the Cyrillic patterns use their own edges.
var personalPatterns = []struct {
	label string
	re    *regexp.Regexp
}{
	{"полная дата", regexp.MustCompile(`\b\d{1,2}\.\d{1,2}\.\d{4}\b|\b\d{4}-\d{2}-\d{2}\b`)},
	{"отчество", regexp.MustCompile(`(?:^|[^\p{L}])[А-ЯЁ][а-яё]+(?:ович|евич|ьич|овна|евна|ична)(?:а|у|ем|ом|е|ы|ой|ну|не)?(?:[^\p{L}]|$)`)},
	{"почта", regexp.MustCompile(`[\p{L}\p{N}_.+-]+@[\p{L}\p{N}_-]+\.[\p{L}\p{N}_]+`)},
	{"телефон", regexp.MustCompile(`(?:\+7|\b8)[\s(-]*\d{3}[\s)-]*\d{3}[\s-]*\d{2}[\s-]*\d{2}\b`)},
}

// personalData names the first kind of personal data found in any text value
// of the input, or returns "".
func personalData(payload *object) string {
	var data any
	_ = json.Unmarshal(canonicalJSON(payload), &data)
	found := ""
	var walk func(any)
	walk = func(value any) {
		if found != "" {
			return
		}
		switch v := value.(type) {
		case string:
			for _, pattern := range personalPatterns {
				if pattern.re.MatchString(v) {
					found = pattern.label
					return
				}
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		case map[string]any:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(data)
	return found
}

// PortraitView is what the teacher gets: a status and, when ready, the text.
type PortraitView struct {
	Status   string        `json:"status"`
	Portrait *PortraitText `json:"portrait,omitempty"`
	Reason   string        `json:"reason,omitempty"`
}

type PortraitText struct {
	Output  json.RawMessage `json:"output"`
	Created string          `json:"created"`
}

func missingPortrait() error { return problem(NotFound, "Вывод не найден.") }

// portraitSource finds the task and checks that the source exists, is taken
// by the task and belongs to the teacher.
func (a *Service) portraitSource(db model.Database, teacherID, taskID, sourceID string) (portraitTask, Source, error) {
	task, ok := a.portraits.task(taskID)
	if !ok {
		return task, Source{}, missingPortrait()
	}
	for _, kind := range []string{"submission", "session"} {
		s := Source{kind, sourceID}
		if task.Accepts(db, s) && task.Visible(db, s, teacherID) {
			return task, s, nil
		}
	}
	return task, Source{}, missingPortrait()
}

// PortraitFor returns the status and the text of a portrait. Without a model
// the status is "off". A portrait that is neither stored nor on its way is queued.
func (a *Service) PortraitFor(teacherID, taskID, sourceID string) (PortraitView, error) {
	return a.portraitView(teacherID, taskID, sourceID, false)
}

// RetryPortrait takes a failed portrait off its failure and queues it again.
// A stored portrait is returned as it is and the model is not called.
func (a *Service) RetryPortrait(teacherID, taskID, sourceID string) (PortraitView, error) {
	return a.portraitView(teacherID, taskID, sourceID, true)
}

func (a *Service) portraitView(teacherID, taskID, sourceID string, retry bool) (PortraitView, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return PortraitView{}, err
	}
	task, s, err := a.portraitSource(db, teacherID, taskID, sourceID)
	if err != nil {
		return PortraitView{}, err
	}
	l := a.portraits
	l.mu.Lock()
	m := l.model
	l.mu.Unlock()
	if m == nil {
		return PortraitView{Status: portraitOff}, nil
	}
	_, key, err := a.prepare(db, task, s, m.Name())
	if err != nil {
		return PortraitView{Status: portraitFailed, Reason: "Не удалось собрать сведения для модели."}, nil
	}
	if portrait, ok := findPortrait(db, key); ok {
		return PortraitView{Status: portraitReady, Portrait: &PortraitText{Output: portrait.Output, Created: portrait.Created}}, nil
	}
	state := a.enqueuePortrait(db, task, s, retry)
	return PortraitView{Status: state.Status, Reason: state.Reason}, nil
}
