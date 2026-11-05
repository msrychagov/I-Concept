package usecase

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
	"vucdiagnostic/internal/valuebank"
)

// The portrait task «values» (decided 5 November 2026): the model marks a text
// by «Онтограф ценностей 1.1» — a submitted essay or the open answers of the
// test «Мои действия и опыт» — with a quote word for word under every mark, and
// writes no text. The program counts by the coding rules of the ontograph:
// only how many times and where, no levels and no common score. The
// instruction, the schema, the counting and the screens are in Вики/Онтограф
// ценностей 1.1/Инструкция разметки.md. The analysis of the essay does not take
// the marks: the two tasks are independent (Г6).

const (
	valuesTask      = "values"
	valuesMarksMost = 40
)

var (
	valueSigns = []string{"принимает", "отвергает"}
	valueWhose = []string{"своё", "одобряет у других", "другое"}
	// valueCounted are the kinds of «whose» the numbers take: the author's own
	// words and what he endorses in others (own and endorse of the rules).
	valueCounted = []string{"своё", "одобряет у других"}
)

// valuesTask marks the essays and the open answers of the tests. The essay keeps
// the patronymic, as its analysis does; the open answers of a test are cut as
// for the portrait by the test, by valuesRows. Since v2 (decided 5 November
// 2026 after the run) the notions go with their definitions and the program
// refuses a marking that puts most marks on one value or takes no notion; the
// marking of v1, made for another input, is shown while the one of v2 is made.
func (a *Service) valuesTask(prompt promptbank.Prompt, o *valuebank.Ontograph) portraitTask {
	second := false
	if n, _ := promptbank.Number(prompt.Version); n >= 2 {
		second = true
	}
	payloadOf := func(a *Service, db model.Database, s Source, definitions bool) (*object, error) {
		rows, _ := a.valuesRows(db, s)
		if len(rows) == 0 {
			return nil, problem(Invalid, "Нет текста для разметки.")
		}
		payload := newObject()
		// The input is cleaned in place: the ontograph is built anew every time.
		payload.set("онтограф", ontographInput(o, definitions))
		payload.set("текст", rows)
		return payload, nil
	}
	check := func(output map[string]any, payload *object) []string { return checkValuesAnswer(o, output, payload) }
	var earlier func(a *Service, db model.Database, s Source) []*object
	if second {
		check = func(output map[string]any, payload *object) []string {
			return append(checkValuesAnswer(o, output, payload), spreadProblems(o, output)...)
		}
		earlier = func(a *Service, db model.Database, s Source) []*object {
			if payload, err := payloadOf(a, db, s, false); err == nil {
				return []*object{payload}
			}
			return nil
		}
	}
	return portraitTask{
		ID: valuesTask, Version: prompt.Version, Prompt: prompt,
		// Up to 40 marks of five short fields with a quote each.
		MaxTokens: 4000, Attempts: 5,
		Keep: []string{"отчество"},
		Accepts: func(db model.Database, s Source) bool {
			rows, _ := a.valuesRows(db, s)
			return len(rows) > 0
		},
		Payload: func(a *Service, db model.Database, s Source) (*object, error) { return payloadOf(a, db, s, second) },
		Clean:   cleanValuesAnswer, Check: check, Earlier: earlier,
		Visible: func(db model.Database, s Source, teacherID string) bool {
			if submission, ok := findSubmission(db, s); ok {
				return teacherOwnsSubmission(db, teacherID, submission)
			}
			session, ok := findSession(db, s)
			return ok && teacherOwnsSession(db, teacherID, session)
		},
	}
}

// ontographInput is the ontograph as the model gets it: every node a line «id —
// название: определение», the notions of a value inside it as «id — название»,
// with their definitions since v2 (without the tail «Ключ: …»); no ideas,
// threats, links or quotes of the report.
func ontographInput(o *valuebank.Ontograph, definitions bool) *object {
	line := func(node valuebank.Node) string { return node.ID + " — " + node.Label + ": " + node.Definition }
	lines := func(class string) []string {
		list := []string{}
		for _, node := range o.Of(class) {
			list = append(list, line(node))
		}
		return list
	}
	values := []any{}
	for _, value := range o.Of(valuebank.Value) {
		notions := []string{}
		for _, notion := range o.Of(valuebank.Aspect) {
			if notion.ParentValue != value.ID {
				continue
			}
			text := notion.ID + " — " + notion.Label
			if definitions {
				definition, _, _ := strings.Cut(notion.Definition, " Ключ:")
				text += ": " + definition
			}
			notions = append(notions, text)
		}
		entry := newObject()
		entry.set("узел", line(value))
		entry.set("понятия", notions)
		values = append(values, entry)
	}
	input := newObject()
	input.set("ценности", values)
	input.set("антиценности", lines(valuebank.Antivalue))
	input.set("ценности_в_повседневной_жизни", lines(valuebank.Everyday))
	input.set("антиценности_в_повседневной_жизни", lines(valuebank.EverydayAnti))
	return input
}

// valuesRows are the answers of the source that are marked, each with its
// number, question and answer: all the answers of a submitted essay, or the
// open answers of a finished test «Мои действия и опыт» that tell something.
func (a *Service) valuesRows(db model.Database, s Source) ([]any, bool) {
	row := func(number, question, answer string) any {
		o := newObject()
		o.set("номер", number)
		o.set("вопрос", question)
		o.set("ответ", answer)
		return o
	}
	rows := []any{}
	if submission, ok := findSubmission(db, s); ok {
		form, known := a.forms[essayForm]
		if !known || submission.FormID != essayForm || !submission.Completed {
			return nil, false
		}
		for _, raw := range essayRows(form, submission) {
			essay := raw.(*object)
			rows = append(rows, row(objectText(essay, "номер"), objectText(essay, "вопрос"), objectText(essay, "ответ")))
		}
		return rows, true
	}
	session, ok := findSession(db, s)
	if !ok || !session.Completed || !session.Activity {
		return nil, false
	}
	for _, item := range a.itemsFor(session) {
		note := session.Notes[item.ID]
		if _, answered := session.Answers[item.ID]; item.Module != "F" || item.Type != "textarea" || !answered || !storyTold(note) {
			continue
		}
		// The task keeps the patronymic for the essay; a test cuts it as its portrait does.
		answer, _ := maskText(strings.TrimSpace(note))
		rows = append(rows, row(item.ID, item.Text, answer))
	}
	return rows, true
}

// valuesAnswers are the answers of the input by number, in the form quotes are
// looked up in.
func valuesAnswers(payload *object) map[string]string {
	answers := map[string]string{}
	for _, row := range objectList(payload.values["текст"]) {
		answers[objectText(row, "номер")] = quoteForm(objectText(row, "ответ"))
	}
	return answers
}

// checkValuesAnswer returns what is wrong with an answer to the instruction
// values/v1: a list of not more than 40 marks, each naming a node of a coder
// class of the ontograph, a sign and a «whose» from their lists and a quote word
// for word in the answer it names. A problem names the mark and the node, never
// the quote.
func checkValuesAnswer(o *valuebank.Ontograph, output map[string]any, payload *object) []string {
	marks, ok := asList(output["marks"])
	if !ok {
		return []string{"marks: нет списка"}
	}
	answers := valuesAnswers(payload)
	problems := []string{}
	if len(marks) > valuesMarksMost {
		problems = append(problems, fmt.Sprintf("marks: отметок %d, нужно не больше %d", len(marks), valuesMarksMost))
	}
	for n, raw := range marks {
		where, mark := fmt.Sprintf("marks, отметка %d", n+1), asMap(raw)
		if mark == nil {
			problems = append(problems, where+": не запись")
			continue
		}
		id := textOf(mark["node"])
		if _, known := o.Node(id); !known {
			problems = append(problems, fmt.Sprintf("%s: узла «%s» нет в онтографе — бери id только из онтографа во входе", where, truncate(id, 30)))
		} else if !o.Coded(id) {
			problems = append(problems, fmt.Sprintf("%s: «%s» — идея, её не отмечай: отмечай ценности, понятия, антиценности и их проявления в повседневной жизни", where, id))
		}
		if !oneOf(mark["sign"], valueSigns) {
			problems = append(problems, where+": sign — «принимает» или «отвергает»")
		}
		if !oneOf(mark["whose"], valueWhose) {
			problems = append(problems, where+": whose — «своё», «одобряет у других» или «другое»")
		}
		problems = append(problems, quoteProblems(where, mark, answers)...)
	}
	return problems
}

const (
	// spreadLeast is the fewest counted marks from which one value must not take
	// more than half of them (decision Д-З7); notionLeast those from which a
	// marking must have a notion (Д-З8).
	spreadLeast = 6
	notionLeast = 4
)

// spreadProblems finds what the run of 5 November showed in the marking of v1:
// one value taking more than half of the marks («служение Отечеству» on any
// moment of the training) and no notion at all. Only the marks the numbers take
// are counted. A problem names the value, never the quote.
func spreadProblems(o *valuebank.Ontograph, output map[string]any) []string {
	marks, _ := asList(output["marks"])
	counted, notions := 0, 0
	byTarget, order := map[string]int{}, []string{}
	for _, raw := range marks {
		mark := asMap(raw)
		id := textOf(mark["node"])
		node, known := o.Node(id)
		if !known || !o.Coded(id) || !slices.Contains(valueCounted, textOf(mark["whose"])) {
			continue
		}
		counted++
		if node.Class == valuebank.Aspect {
			notions++
		}
		target, _ := o.Counted(id)
		if byTarget[target] == 0 {
			order = append(order, target)
		}
		byTarget[target]++
	}
	problems := []string{}
	for _, target := range order {
		if counted >= spreadLeast && 2*byTarget[target] > counted {
			value, _ := o.Node(target)
			problems = append(problems, fmt.Sprintf("marks: больше половины отметок (%d из %d) — на «%s»: отмечай только места, где ценность прямо видна, обычные обстоятельства учёбы не отмечай, бери понятия", byTarget[target], counted, value.Label))
		}
	}
	if counted >= notionLeast && notions == 0 {
		problems = append(problems, "marks: ни одного понятия — сначала ищи понятие внутри ценности, например aid_vyr — Взаимовыручка, honest — Честность")
	}
	return problems
}

// valueKey is what makes two marks one: the node, the answer and the quote.
func valueKey(mark map[string]any) string {
	return textOf(mark["node"]) + "\n" + textOf(mark["question"]) + "\n" + quoteForm(textOf(mark["quote"]))
}

// cleanValuesAnswer drops a mark that repeats another and counts them.
func cleanValuesAnswer(_ *Service, _ model.Database, _ Source, output map[string]any, _ *object) (map[string]any, int) {
	marks, ok := asList(output["marks"])
	if !ok {
		return output, 0
	}
	kept, seen, changes := []any{}, map[string]bool{}, 0
	for _, raw := range marks {
		if mark := asMap(raw); mark != nil {
			if seen[valueKey(mark)] {
				changes++
				continue
			}
			seen[valueKey(mark)] = true
		}
		kept = append(kept, raw)
	}
	output["marks"] = kept
	return output, changes
}

// ---- the counting -----------------------------------------------------------

// ValueQuote is a mark as the teacher reads it: the node it names, what the
// author does with it, whose it is and the quote with its question.
type ValueQuote struct {
	Node     string `json:"node"`
	Label    string `json:"label"`
	Class    string `json:"class"`
	Sign     string `json:"sign"`
	Whose    string `json:"whose"`
	Question string `json:"question"`
	Asked    string `json:"asked"`
	Quote    string `json:"quote"`
}

// ValueCount is the count of a value or an antivalue in one text: «принимает»
// and «отвергает» apart, the everyday forms apart from the main count.
type ValueCount struct {
	ID             string       `json:"id"`
	Label          string       `json:"label"`
	Idea           string       `json:"idea"`
	IdeaLabel      string       `json:"ideaLabel"`
	Accept         int          `json:"accept"`
	Reject         int          `json:"reject"`
	EverydayAccept int          `json:"everydayAccept"`
	EverydayReject int          `json:"everydayReject"`
	Quotes         []ValueQuote `json:"quotes"`
}

// IdeaCount is an idea of the formula: the sum of the counts of its values.
type IdeaCount struct {
	ID             string       `json:"id"`
	Label          string       `json:"label"`
	Accept         int          `json:"accept"`
	Reject         int          `json:"reject"`
	EverydayAccept int          `json:"everydayAccept"`
	EverydayReject int          `json:"everydayReject"`
	Values         []ValueCount `json:"values"`
}

// ValuesCount is what the program counted in one text: the five ideas in the
// order of the formula with the values met, the antivalues met, and the marks
// of «другое», which are not counted.
type ValuesCount struct {
	Marks      int          `json:"marks"`
	Ideas      []IdeaCount  `json:"ideas"`
	Antivalues []ValueCount `json:"antivalues"`
	Other      []ValueQuote `json:"other"`
}

func (c ValueCount) met() bool {
	return c.Accept+c.Reject+c.EverydayAccept+c.EverydayReject > 0
}

// countValues counts the marks of a stored answer by the coding rules: a value
// counts its own marks and those of its notions, an everyday form is added
// apart, an idea sums its values, an antivalue counts the same way apart; one
// quote counts once for a value, whatever notions of it the model put on it.
// asked are the questions by the number of the answer.
func countValues(o *valuebank.Ontograph, output map[string]any, asked map[string]string) ValuesCount {
	counts := map[string]*ValueCount{}
	seen := map[string]bool{}
	result := ValuesCount{Ideas: []IdeaCount{}, Antivalues: []ValueCount{}, Other: []ValueQuote{}}
	marks, _ := asList(output["marks"])
	for _, raw := range marks {
		mark := asMap(raw)
		node, known := o.Node(textOf(mark["node"]))
		if mark == nil || !known || !o.Coded(node.ID) {
			continue
		}
		result.Marks++
		quote := ValueQuote{Node: node.ID, Label: node.Label, Class: node.Class, Sign: textOf(mark["sign"]), Whose: textOf(mark["whose"]),
			Question: textOf(mark["question"]), Asked: asked[textOf(mark["question"])], Quote: textOf(mark["quote"])}
		if !slices.Contains(valueCounted, quote.Whose) {
			result.Other = append(result.Other, quote)
			continue
		}
		target, everyday := o.Counted(node.ID)
		count := counts[target]
		if count == nil {
			value, _ := o.Node(target)
			idea, _ := o.Node(o.IdeaOf(target))
			count = &ValueCount{ID: target, Label: value.Label, Idea: idea.ID, IdeaLabel: idea.Label, Quotes: []ValueQuote{}}
			counts[target] = count
		}
		count.Quotes = append(count.Quotes, quote)
		key := fmt.Sprintf("%s\n%v\n%s\n%s\n%s", target, everyday, quote.Sign, quote.Question, quoteForm(quote.Quote))
		if seen[key] {
			continue
		}
		seen[key] = true
		accept := quote.Sign == "принимает"
		switch {
		case everyday && accept:
			count.EverydayAccept++
		case everyday:
			count.EverydayReject++
		case accept:
			count.Accept++
		default:
			count.Reject++
		}
	}
	for _, idea := range o.Of(valuebank.Idea) {
		entry := IdeaCount{ID: idea.ID, Label: idea.Label, Values: []ValueCount{}}
		for _, value := range o.Of(valuebank.Value) {
			if count := counts[value.ID]; count != nil && count.met() && o.IdeaOf(value.ID) == idea.ID {
				entry.Accept += count.Accept
				entry.Reject += count.Reject
				entry.EverydayAccept += count.EverydayAccept
				entry.EverydayReject += count.EverydayReject
				entry.Values = append(entry.Values, *count)
			}
		}
		result.Ideas = append(result.Ideas, entry)
	}
	for _, antivalue := range o.Of(valuebank.Antivalue) {
		if count := counts[antivalue.ID]; count != nil && count.met() {
			result.Antivalues = append(result.Antivalues, *count)
		}
	}
	return result
}

// loadOntograph reads the embedded ontograph once.
var loadOntograph = sync.OnceValues(valuebank.Load)

// ontograph is the graph the task counts by; nil when it cannot be read, and
// then the task is not on.
func (a *Service) ontograph() *valuebank.Ontograph {
	o, _ := loadOntograph()
	return o
}

// valuesOf counts the marking of a source shown to the teacher now; nil when it
// is not ready. It queues nothing.
func (a *Service) valuesOf(db model.Database, s Source) (*ValuesCount, PortraitView) {
	task, ok := a.portraits.task(valuesTask)
	o := a.ontograph()
	if !ok || o == nil || !task.Accepts(db, s) {
		return nil, PortraitView{Status: portraitOff}
	}
	view, _ := a.portraitNow(db, task, s)
	if view.Status != portraitReady {
		return nil, view
	}
	count, ok := a.countOutput(db, s, view.Portrait.Output)
	if !ok {
		return nil, PortraitView{Status: portraitFailed, Reason: "Не удалось прочитать разметку."}
	}
	return &count, view
}

func (a *Service) countOutput(db model.Database, s Source, data json.RawMessage) (ValuesCount, bool) {
	var output map[string]any
	if json.Unmarshal(data, &output) != nil {
		return ValuesCount{}, false
	}
	asked := map[string]string{}
	rows, _ := a.valuesRows(db, s)
	for _, row := range objectList(rows) {
		asked[objectText(row, "номер")] = objectText(row, "вопрос")
	}
	return countValues(a.ontograph(), output, asked), true
}

// ValuesView is the marking of one text as the teacher sees it: the status of
// the task and, when ready, what the program counted.
type ValuesView struct {
	Status  string       `json:"status"`
	Reason  string       `json:"reason,omitempty"`
	Created string       `json:"created,omitempty"`
	Count   *ValuesCount `json:"count,omitempty"`
}

// ValuesFor returns the marking of an essay or a test of the teacher, queueing
// it when it is missing; retry takes a failed one off its failure.
func (a *Service) ValuesFor(teacherID, sourceID string, retry bool) (ValuesView, error) {
	view, err := a.portraitView(teacherID, valuesTask, sourceID, retry)
	if err != nil {
		return ValuesView{}, err
	}
	result := ValuesView{Status: view.Status, Reason: view.Reason}
	if view.Status != portraitReady || view.Portrait == nil {
		return result, nil
	}
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return ValuesView{}, err
	}
	_, s, err := a.portraitSource(db, teacherID, valuesTask, sourceID)
	if err != nil {
		return ValuesView{}, err
	}
	count, ok := a.countOutput(db, s, view.Portrait.Output)
	if !ok {
		return ValuesView{Status: portraitFailed, Reason: "Не удалось прочитать разметку."}, nil
	}
	result.Created, result.Count = view.Portrait.Created, &count
	return result, nil
}

// ---- over many texts --------------------------------------------------------

// ValueAuthor is an author of a text the value is met in.
type ValueAuthor struct {
	Name         string `json:"name"`
	Platoon      string `json:"platoon"`
	SubmissionID string `json:"submissionId,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
}

// ValueAuthors is a value or an antivalue over the texts: who accepts it in the
// main count, who only in everyday life, who rejects it. For an antivalue
// «rejecting» are those who condemn it and «authors» those who take it for a norm.
type ValueAuthors struct {
	ID        string        `json:"id"`
	Label     string        `json:"label"`
	IdeaLabel string        `json:"ideaLabel"`
	Authors   []ValueAuthor `json:"authors"`
	Everyday  []ValueAuthor `json:"everyday"`
	Rejecting []ValueAuthor `json:"rejecting"`
}

// IdeaAuthors is an idea over the texts: the authors who accept any of its
// values and the values met.
type IdeaAuthors struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Authors int            `json:"authors"`
	Values  []ValueAuthors `json:"values"`
}

// ValuesSummary is the marking over the texts of an assignment or a platoon:
// how many texts there are, how many are marked, and by whom each value is met.
type ValuesSummary struct {
	Texts      int            `json:"texts"`
	Marked     int            `json:"marked"`
	Ideas      []IdeaAuthors  `json:"ideas"`
	Antivalues []ValueAuthors `json:"antivalues"`
}

type valuesText struct {
	author ValueAuthor
	source Source
}

// summarizeValues counts over the texts: a value is met by an author whose text
// accepts it in the main count (decision Д-З5); only in everyday life apart.
func (a *Service) summarizeValues(db model.Database, texts []valuesText) ValuesSummary {
	o := a.ontograph()
	summary := ValuesSummary{Texts: len(texts), Ideas: []IdeaAuthors{}, Antivalues: []ValueAuthors{}}
	if o == nil {
		return summary
	}
	byValue := map[string]*ValueAuthors{}
	ideaAuthors := map[string]map[string]bool{}
	entry := func(count ValueCount) *ValueAuthors {
		if byValue[count.ID] == nil {
			byValue[count.ID] = &ValueAuthors{ID: count.ID, Label: count.Label, IdeaLabel: count.IdeaLabel, Authors: []ValueAuthor{}, Everyday: []ValueAuthor{}, Rejecting: []ValueAuthor{}}
		}
		return byValue[count.ID]
	}
	for _, text := range texts {
		count, _ := a.valuesOf(db, text.source)
		if count == nil {
			continue
		}
		summary.Marked++
		key := text.source.Kind + ":" + text.source.ID
		add := func(c ValueCount) {
			e := entry(c)
			switch {
			case c.Accept > 0:
				e.Authors = append(e.Authors, text.author)
				if ideaAuthors[c.Idea] == nil {
					ideaAuthors[c.Idea] = map[string]bool{}
				}
				ideaAuthors[c.Idea][key] = true
			case c.EverydayAccept > 0:
				e.Everyday = append(e.Everyday, text.author)
			}
			if c.Reject > 0 || c.EverydayReject > 0 {
				e.Rejecting = append(e.Rejecting, text.author)
			}
		}
		for _, idea := range count.Ideas {
			for _, value := range idea.Values {
				add(value)
			}
		}
		for _, antivalue := range count.Antivalues {
			e := entry(antivalue)
			if antivalue.Accept+antivalue.EverydayAccept > 0 {
				e.Authors = append(e.Authors, text.author)
			}
			if antivalue.Reject+antivalue.EverydayReject > 0 {
				e.Rejecting = append(e.Rejecting, text.author)
			}
		}
	}
	for _, idea := range o.Of(valuebank.Idea) {
		item := IdeaAuthors{ID: idea.ID, Label: idea.Label, Authors: len(ideaAuthors[idea.ID]), Values: []ValueAuthors{}}
		for _, value := range o.Of(valuebank.Value) {
			if e := byValue[value.ID]; e != nil && o.IdeaOf(value.ID) == idea.ID {
				item.Values = append(item.Values, *e)
			}
		}
		// The most met first, as the program counted them.
		slices.SortStableFunc(item.Values, func(x, y ValueAuthors) int { return len(y.Authors) - len(x.Authors) })
		summary.Ideas = append(summary.Ideas, item)
	}
	for _, antivalue := range o.Of(valuebank.Antivalue) {
		if e := byValue[antivalue.ID]; e != nil {
			summary.Antivalues = append(summary.Antivalues, *e)
		}
	}
	return summary
}

// EssayValues is the marking over the submitted essays of an assignment of the teacher.
func (a *Service) EssayValues(teacherID, assignmentID string) (ValuesSummary, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return ValuesSummary{}, err
	}
	as, _, _, err := ownEssayAssignment(db, teacherID, assignmentID)
	if err != nil {
		return ValuesSummary{}, err
	}
	texts := []valuesText{}
	for _, submission := range assignmentSubmissions(db, as.ID) {
		student, _ := studentOf(db, submission)
		texts = append(texts, valuesText{ValueAuthor{Name: student.Name, Platoon: student.Platoon, SubmissionID: submission.ID}, Source{"submission", submission.ID}})
	}
	return a.summarizeValues(db, texts), nil
}

// PlatoonValues is the marking over the open answers of a platoon of the
// teacher: the last finished test of every student that has told answers.
func (a *Service) PlatoonValues(teacherID, group string) (ValuesSummary, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return ValuesSummary{}, err
	}
	latest := map[string]model.Session{}
	for _, record := range db.Sessions {
		s := record.Session
		if s.Group != group || s.Kind != "student" || !teacherOwnsSession(db, teacherID, s) {
			continue
		}
		if rows, _ := a.valuesRows(db, Source{"session", s.ID}); len(rows) == 0 {
			continue
		}
		key := s.StudentID
		if key == "" {
			key = s.ID
		}
		if previous, seen := latest[key]; !seen || s.Updated > previous.Updated {
			latest[key] = s
		}
	}
	sessions := []model.Session{}
	for _, s := range latest {
		sessions = append(sessions, s)
	}
	slices.SortFunc(sessions, func(x, y model.Session) int { return strings.Compare(x.Participant+x.ID, y.Participant+y.ID) })
	texts := []valuesText{}
	for _, s := range sessions {
		texts = append(texts, valuesText{ValueAuthor{Name: s.Participant, Platoon: s.Group, SessionID: s.ID}, Source{"session", s.ID}})
	}
	return a.summarizeValues(db, texts), nil
}
