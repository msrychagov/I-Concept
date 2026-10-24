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
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
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

// Source is what a portrait is about: a submitted form ("submission"), a
// finished test session ("session"), an assignment of an essay ("assignment")
// or a platoon of a teacher ("platoon", platoon_task.go).
type Source struct{ Kind, ID string }

type portraitTask struct {
	ID, Version string
	Prompt      promptbank.Prompt
	// Accepts says whether the task takes this source.
	Accepts func(db model.Database, s Source) bool
	// Payload builds the anonymised input.
	Payload func(a *Service, db model.Database, s Source) (*object, error)
	// Clean mends an answer before it is checked and stored: it returns the
	// answer without what the program can drop on its own and the number of
	// changes. nil for the tasks that only check.
	Clean func(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int)
	// Check returns the problems of an answer; an empty list means it is good.
	Check func(output map[string]any, payload *object) []string
	// Visible says whether this teacher may see the portrait of the source.
	Visible func(db model.Database, s Source, teacherID string) bool
	// MaxTokens limits the answer; zero keeps the model's default.
	MaxTokens int
	// Attempts is how many times the model may answer; zero means portraitAttempts.
	Attempts int
	// Manual tasks are made only when the teacher asks (retry), never by the
	// queue after a source is saved or by opening the portrait.
	Manual bool
	// Stamp gives the source group and the ordered members of a portrait made
	// for a group of sources; nil for the other tasks.
	Stamp func(a *Service, db model.Database, s Source) (source string, sources []string)
	// Earlier builds the input the earlier versions of the instruction got: a
	// portrait made for one of them is shown while the current one is made.
	// nil when the input did not change between the versions.
	Earlier func(a *Service, db model.Database, s Source) []*object
	// Keep lists the kinds of personal data (labels of personalPatterns) that
	// are not cut out of the input of this task: the essay keeps the patronymic,
	// or the model could not tell «Игорь Петрович» from the list of instructors.
	Keep []string
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

// prepare builds the input of a source with the personal data cut out, and
// the number of cuts. The key, the fingerprint and the text that goes to the
// model all come from the cleaned input, so equal answers that differ only in
// a cut piece give one portrait. It needs no model.
func (a *Service) prepare(db model.Database, task portraitTask, s Source) (*object, int, error) {
	payload, err := task.Payload(a, db, s)
	if err != nil {
		return nil, 0, err
	}
	masked, err := maskPersonal(payload, task.Keep...)
	if err != nil {
		return nil, 0, err
	}
	return payload, masked, nil
}

// storedPortrait is the newest portrait of the task made for this input by any
// model or instruction version.
func storedPortrait(db model.Database, task portraitTask, fingerprint string) (model.Portrait, bool) {
	var newest model.Portrait
	found := false
	for _, portrait := range db.Portraits {
		if portrait.Task == task.ID && portrait.Fingerprint == fingerprint && (!found || portrait.Created >= newest.Created) {
			newest, found = portrait, true
		}
	}
	return newest, found
}

// shownPortrait is the newest stored portrait of the task for this input or,
// when there is none, for the input of an earlier version of the instruction.
func (a *Service) shownPortrait(db model.Database, task portraitTask, s Source, payload *object) (model.Portrait, bool) {
	if old, ok := storedPortrait(db, task, payloadFingerprint(payload)); ok {
		return old, true
	}
	if task.Earlier == nil {
		return model.Portrait{}, false
	}
	for _, earlier := range task.Earlier(a, db, s) {
		if _, err := maskPersonal(earlier, task.Keep...); err != nil {
			continue
		}
		if old, ok := storedPortrait(db, task, payloadFingerprint(earlier)); ok {
			return old, true
		}
	}
	return model.Portrait{}, false
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
	payload, _, err := a.prepare(db, task, s)
	if err != nil {
		return portraitState{Status: portraitFailed, Reason: "Не удалось собрать сведения для модели."}
	}
	key := portraitKey(task, m.Name(), payload)
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
	payload, masked, err := a.prepare(db, task, job.source)
	if err != nil {
		l.setState(job.key, &portraitState{Status: portraitFailed, Reason: "Не удалось собрать сведения для модели."})
		return
	}
	key := portraitKey(task, m.Name(), payload)
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
	if masked > 0 {
		log.Printf("portrait %s %.12s: personal data cut out: %d", task.ID, key, masked)
	}
	l.setState(key, &portraitState{Status: portraitWorking})
	user := modelText(canonicalJSON(payload))
	reason, detail := "Модель не ответила.", "no attempt was made"
	started := time.Now()
	// A retry tells the model what the check found: with a temperature near
	// zero the same message gives the same mistake. The stored input stays
	// the data alone.
	message := user
	attempts := portraitAttempts
	if task.Attempts > 0 {
		attempts = task.Attempts
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(l.pause):
			}
		}
		reply, err := m.Complete(ctx, Completion{System: task.Prompt.Instruction, User: message, Schema: task.Prompt.Schema, MaxTokens: task.MaxTokens})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			reason, detail = "Модель не ответила.", fmt.Sprintf("attempt %d: %v", attempt, err)
			continue
		}
		output, problems := parseModelAnswer(reply.Text)
		if len(problems) == 0 && task.Clean != nil {
			var changes int
			if output, changes = task.Clean(a, db, job.source, output, payload); changes > 0 {
				log.Printf("portrait %s %.12s: answer cleaned, source %s %s, changes: %d", task.ID, key, job.source.Kind, job.source.ID, changes)
			}
		}
		if len(problems) == 0 {
			problems = task.Check(output, payload)
		}
		if len(problems) > 0 {
			reason = "Ответ модели не прошёл проверку: " + strings.Join(problems[:min(len(problems), 3)], "; ") + "."
			detail = fmt.Sprintf("attempt %d: %s", attempt, strings.Join(problems, "; "))
			message = user + "\n\n" + retryNote + strings.Join(problems, "; ") + "."
			continue
		}
		data, _ := json.Marshal(output)
		portrait := model.Portrait{
			ID: key, Task: task.ID, TaskVersion: task.Version, Fingerprint: payloadFingerprint(payload),
			Model: m.Name(), ModelVersion: reply.Version, Created: time.Now().UTC().Format(time.RFC3339),
			Input: user, Output: data, Usage: model.PortraitUsage{In: reply.In, Out: reply.Out}, Seconds: reply.Seconds,
		}
		if task.Stamp != nil {
			portrait.Source, portrait.Sources = task.Stamp(a, db, job.source)
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
		log.Printf("portrait %s %.12s: ready (%.1f s, %d+%d tokens, attempt %d)", task.ID, key, reply.Seconds, reply.In, reply.Out, attempt)
		return
	}
	fail(reason, detail, time.Since(started).Seconds())
}

// retryNote opens the list of problems sent with a retry.
const retryNote = "Предыдущий ответ не прошёл проверку программы. Исправь эти места и ответь заново по той же схеме, соблюдая все правила инструкции: "

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

// Personal data must not reach the model whatever the form lets through: every
// piece found in a text value is replaced by personalMask and the rest is sent.
// Go \b knows only ASCII letters, so a patronymic is found by its own pattern
// and its edges are checked by hand.
const personalMask = "[скрыто]"

var personalPatterns = []struct {
	label string
	re    *regexp.Regexp
}{
	{"полная дата", regexp.MustCompile(`\b\d{1,2}\.\d{1,2}\.\d{4}\b|\b\d{4}-\d{2}-\d{2}\b`)},
	{"отчество", regexp.MustCompile(`[А-ЯЁ][а-яё]+(?:ович|евич|ьич|овна|евна|ична)(?:а|у|ем|ом|е|ы|ой|ну|не)?`)},
	{"почта", regexp.MustCompile(`[\p{L}\p{N}_.+-]+@[\p{L}\p{N}_-]+\.[\p{L}\p{N}_]+`)},
	{"телефон", regexp.MustCompile(`(?:\+7|\b8)[\s(-]*\d{3}[\s)-]*\d{3}[\s-]*\d{2}[\s-]*\d{2}\b`)},
}

type personalSpan struct {
	label      string
	start, end int
}

// personalSpans lists the pieces of a text that look like personal data, by
// position, without overlaps.
func personalSpans(text string, keep ...string) []personalSpan {
	spans := []personalSpan{}
	for _, pattern := range personalPatterns {
		if slices.Contains(keep, pattern.label) {
			continue
		}
		for _, at := range pattern.re.FindAllStringIndex(text, -1) {
			if pattern.label == "отчество" {
				before, _ := utf8.DecodeLastRuneInString(text[:at[0]])
				after, _ := utf8.DecodeRuneInString(text[at[1]:])
				if unicode.IsLetter(before) || unicode.IsLetter(after) {
					continue
				}
			}
			spans = append(spans, personalSpan{pattern.label, at[0], at[1]})
		}
	}
	slices.SortFunc(spans, func(x, y personalSpan) int { return x.start - y.start })
	merged := []personalSpan{}
	for _, span := range spans {
		if last := len(merged) - 1; last >= 0 && span.start < merged[last].end {
			merged[last].end = max(merged[last].end, span.end)
			continue
		}
		merged = append(merged, span)
	}
	return merged
}

// maskText replaces every piece of personal data with personalMask and counts
// them; the kinds in keep stay as they are.
func maskText(text string, keep ...string) (string, int) {
	spans := personalSpans(text, keep...)
	if len(spans) == 0 {
		return text, 0
	}
	var out strings.Builder
	at := 0
	for _, span := range spans {
		out.WriteString(text[at:span.start])
		out.WriteString(personalMask)
		at = span.end
	}
	out.WriteString(text[at:])
	return out.String(), len(spans)
}

// maskPersonal cuts personal data out of every text value of the input, in
// place, and returns the number of cuts. A value of a kind it does not know
// to hold no text is an error: the input is then not sent at all.
func maskPersonal(payload *object, keep ...string) (int, error) {
	count := 0
	var walk func(any) (any, error)
	walk = func(value any) (any, error) {
		switch v := value.(type) {
		case nil, bool, int, int64, float64:
			return v, nil
		case string:
			text, n := maskText(v, keep...)
			count += n
			return text, nil
		case []string:
			list := make([]string, len(v))
			for i, item := range v {
				text, n := maskText(item, keep...)
				list[i], count = text, count+n
			}
			return list, nil
		case []any:
			list := make([]any, len(v))
			for i, item := range v {
				masked, err := walk(item)
				if err != nil {
					return nil, err
				}
				list[i] = masked
			}
			return list, nil
		case map[string][]any:
			result := make(map[string][]any, len(v))
			for key, item := range v {
				masked, err := walk(item)
				if err != nil {
					return nil, err
				}
				result[key] = masked.([]any)
			}
			return result, nil
		case *object:
			for _, key := range v.keys {
				masked, err := walk(v.values[key])
				if err != nil {
					return nil, err
				}
				v.values[key] = masked
			}
			return v, nil
		}
		return nil, fmt.Errorf("portrait: input value of type %T is not checked for personal data", value)
	}
	if _, err := walk(payload); err != nil {
		return 0, err
	}
	// The cut must leave nothing behind.
	if label := personalData(payload, keep...); label != "" {
		return 0, fmt.Errorf("portrait: %s is left in the input after the cut", label)
	}
	return count, nil
}

// personalData names the first kind of personal data found in any text value
// of the input, or returns "".
func personalData(payload *object, keep ...string) string {
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
			if spans := personalSpans(v, keep...); len(spans) > 0 {
				found = spans[0].label
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

// PortraitText is a stored portrait; Version is the version of the
// instruction it was made by, so that a screen can show an earlier format.
type PortraitText struct {
	Output  json.RawMessage `json:"output"`
	Created string          `json:"created"`
	Version string          `json:"version,omitempty"`
}

func missingPortrait() error { return problem(NotFound, "Вывод не найден.") }

// portraitSource finds the task and checks that the source exists, is taken
// by the task and belongs to the teacher.
func (a *Service) portraitSource(db model.Database, teacherID, taskID, sourceID string) (portraitTask, Source, error) {
	task, ok := a.portraits.task(taskID)
	if !ok {
		return task, Source{}, missingPortrait()
	}
	for _, kind := range []string{"submission", "session", "assignment", "platoon"} {
		s := Source{kind, sourceID}
		if kind == "platoon" {
			s = platoonSource(teacherID, sourceID)
		}
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
	payload, _, err := a.prepare(db, task, s)
	if err != nil {
		return PortraitView{Status: portraitFailed, Reason: "Не удалось собрать сведения для модели."}, nil
	}
	ready := func(p model.Portrait) PortraitView {
		return PortraitView{Status: portraitReady, Portrait: &PortraitText{Output: p.Output, Created: p.Created, Version: p.TaskVersion}}
	}
	if m != nil {
		if portrait, ok := findPortrait(db, portraitKey(task, m.Name(), payload)); ok {
			return ready(portrait), nil
		}
	}
	// A portrait that was made for the same input earlier, by another model or
	// another version of the instruction, is shown while the current model makes
	// its own in the background; the next opening shows the new one.
	if old, ok := a.shownPortrait(db, task, s, payload); ok {
		if m != nil && (!task.Manual || retry) {
			a.enqueuePortrait(db, task, s, retry)
		}
		return ready(old), nil
	}
	if m == nil {
		return PortraitView{Status: portraitOff}, nil
	}
	if task.Manual && !retry {
		now, _ := a.portraitNow(db, task, s)
		return now, nil
	}
	state := a.enqueuePortrait(db, task, s, retry)
	return PortraitView{Status: state.Status, Reason: state.Reason}, nil
}

// portraitNone is the status of a portrait nobody has asked for yet.
const portraitNone = "none"

// portraitNow tells what is known about a portrait without queueing anything:
// the stored text when there is one (of the current model or of any earlier
// one made for the same input) and otherwise the state of the job. The second
// value is the stored record, nil when there is none.
func (a *Service) portraitNow(db model.Database, task portraitTask, s Source) (PortraitView, *model.Portrait) {
	l := a.portraits
	l.mu.Lock()
	m := l.model
	l.mu.Unlock()
	payload, _, err := a.prepare(db, task, s)
	if err != nil {
		return PortraitView{Status: portraitFailed, Reason: "Не удалось собрать сведения для модели."}, nil
	}
	ready := func(p model.Portrait) (PortraitView, *model.Portrait) {
		return PortraitView{Status: portraitReady, Portrait: &PortraitText{Output: p.Output, Created: p.Created, Version: p.TaskVersion}}, &p
	}
	if m != nil {
		if portrait, ok := findPortrait(db, portraitKey(task, m.Name(), payload)); ok {
			return ready(portrait)
		}
	}
	if old, ok := a.shownPortrait(db, task, s, payload); ok {
		return ready(old)
	}
	if m == nil {
		return PortraitView{Status: portraitOff}, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if state, ok := l.states[portraitKey(task, m.Name(), payload)]; ok {
		return PortraitView{Status: state.Status, Reason: state.Reason}, nil
	}
	return PortraitView{Status: portraitNone}, nil
}
