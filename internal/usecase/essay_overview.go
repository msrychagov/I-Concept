package usecase

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// What the teacher sees on the tab «Эссе»: the cohorts and the essays he gave,
// the students of each essay with their state, the summary of an assignment
// and the block of instructors. Everything that is a count or a grouping is
// made here by the program; the model writes only the common text.

type essayStudent struct {
	StudentID    string `json:"studentId"`
	Name         string `json:"name"`
	Platoon      string `json:"platoon"`
	Status       string `json:"status"` // not_started, writing, submitted
	SubmissionID string `json:"submissionId,omitempty"`
	CompletedAt  string `json:"completedAt,omitempty"`
	Answered     int    `json:"answered"`
	Total        int    `json:"total"`
	// Analysis is the state of the analysis of a submitted essay: ready, queued,
	// working, failed, none (nobody asked yet) or off (no model).
	Analysis    string `json:"analysis,omitempty"`
	Reason      string `json:"reason,omitempty"`
	HasPrevious bool   `json:"hasPrevious,omitempty"`
}

type essayCounts struct {
	Students     int `json:"students"`
	Submitted    int `json:"submitted"`
	NotSubmitted int `json:"notSubmitted"`
	Analysed     int `json:"analysed"`
	NoAnalysis   int `json:"withoutAnalysis"`
	WithPrevious int `json:"withPrevious"`
}

type essayAssignmentView struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Created    string         `json:"created"`
	Active     bool           `json:"active"`
	CohortID   string         `json:"cohortId"`
	CohortName string         `json:"cohortName"`
	Year       int            `json:"year"`
	Counts     essayCounts    `json:"counts"`
	Students   []essayStudent `json:"students"`
}

// essayStudents are the students of the cohort of an assignment with their
// state, in the order of platoon and name.
func (a *Service) essayStudents(db model.Database, assignment model.Assignment, cohort model.Cohort) ([]essayStudent, essayCounts) {
	form := a.forms[essayForm]
	rows := []essayStudent{}
	counts := essayCounts{}
	for _, account := range db.Students {
		if !cohortStudent(cohort, account) {
			continue
		}
		var submission *model.FormSubmission
		for i := range db.Submissions {
			if db.Submissions[i].StudentID == account.ID && db.Submissions[i].AssignmentID == assignment.ID {
				submission = &db.Submissions[i]
				break
			}
		}
		if account.Disabled && submission == nil {
			continue
		}
		row := essayStudent{StudentID: account.ID, Name: account.Name, Platoon: account.Platoon, Status: "not_started"}
		counts.Students++
		if submission != nil {
			row.SubmissionID = submission.ID
			progress := formProgress(form, *submission)
			row.Answered, row.Total = len(progress.Complete), progress.Total
			row.Status = "writing"
			if submission.Completed {
				row.Status, row.CompletedAt = "submitted", submission.CompletedAt
				counts.Submitted++
				analysis, view := a.essayAnalysis(db, *submission)
				row.Analysis, row.Reason = view.Status, view.Reason
				if analysis != nil {
					counts.Analysed++
				}
				if _, ok := previousEssay(db, *submission); ok {
					row.HasPrevious = true
					counts.WithPrevious++
				}
			}
		}
		rows = append(rows, row)
	}
	counts.NotSubmitted = counts.Students - counts.Submitted
	counts.NoAnalysis = counts.Submitted - counts.Analysed
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Platoon != rows[j].Platoon {
			return normalWord(rows[i].Platoon) < normalWord(rows[j].Platoon)
		}
		return normalWord(rows[i].Name) < normalWord(rows[j].Name)
	})
	return rows, counts
}

// EssayOverview lists the cohorts the teacher created and the essays he gave.
func (a *Service) EssayOverview(teacherID string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	t, err := managementTeacher(db, teacherID)
	if err != nil {
		return nil, err
	}
	_, known := a.forms[essayForm]
	cohorts := []map[string]any{}
	for _, cohort := range db.Cohorts {
		if cohort.CreatedBy != t.ID || !sameVUC(cohort.VUC, t.VUC) {
			continue
		}
		names := []string{}
		for _, id := range cohort.PlatoonIDs {
			if p, ok := platoonIn(db, id, t.VUC); ok {
				names = append(names, p.Name)
			}
		}
		cohorts = append(cohorts, map[string]any{"id": cohort.ID, "name": cohort.Name, "year": cohort.Year, "platoons": names})
	}
	assignments := []essayAssignmentView{}
	for _, as := range db.Assignments {
		cohort, ok := cohortIn(db, as.CohortID, t.VUC)
		if !known || as.FormID != essayForm || as.OwnerID != t.ID || !ok {
			continue
		}
		students, counts := a.essayStudents(db, as, cohort)
		assignments = append(assignments, essayAssignmentView{ID: as.ID, Title: as.Title, Created: as.Created, Active: as.Active, CohortID: cohort.ID, CohortName: cohort.Name, Year: cohort.Year, Counts: counts, Students: students})
	}
	sort.SliceStable(assignments, func(i, j int) bool { return assignments[i].Created > assignments[j].Created })
	return map[string]any{"cohorts": cohorts, "assignments": assignments, "model": a.portraitsOn()}, nil
}

// portraitsOn says whether a language model is connected.
func (a *Service) portraitsOn() bool {
	a.portraits.mu.Lock()
	defer a.portraits.mu.Unlock()
	return a.portraits.model != nil
}

// ---- the summary of an assignment ----------------------------------------

type quoteView struct {
	Label    string `json:"label,omitempty"`
	Question string `json:"question"`
	Quote    string `json:"quote"`
}

// sourceView is a point of an analysis a common point stands on.
type sourceView struct {
	Ref          string      `json:"ref"`
	SubmissionID string      `json:"submissionId"`
	Student      string      `json:"student"`
	Platoon      string      `json:"platoon"`
	Text         string      `json:"text"`
	About        string      `json:"about,omitempty"`
	Basis        string      `json:"basis,omitempty"`
	Quotes       []quoteView `json:"quotes"`
}

// sourceGroup is what stands behind one common point: the number of different
// authors counted from the references, out of the authors of the summary.
type sourceGroup struct {
	Authors int          `json:"authors"`
	Of      int          `json:"of"`
	Sources []sourceView `json:"sources"`
}

type refInfo struct{ Text, About, Basis string }

// summaryResolver turns the references of a stored summary into the points of
// the analyses. The text of a point is read from the input the model got, so it
// is the text the summary was written from; the quotes come from the analysis of
// the essay now, and only when the point there is still the same.
type summaryResolver struct {
	a        *Service
	db       model.Database
	authors  []string // submission ids, by author number - 1
	info     map[string]refInfo
	analysis map[string]map[string]any
}

func newSummaryResolver(a *Service, db model.Database, portrait model.Portrait) *summaryResolver {
	r := &summaryResolver{a: a, db: db, authors: portrait.Sources, info: map[string]refInfo{}, analysis: map[string]map[string]any{}}
	var input struct {
		Reviews []map[string]any `json:"разборы"`
	}
	_ = json.Unmarshal([]byte(portrait.Input), &input)
	add := func(raw any) {
		if entry := asMap(raw); entry != nil {
			r.info[textOf(entry["номер"])] = refInfo{textOf(entry["текст"]), textOf(entry["о_чём"]), textOf(entry["основа"])}
		}
	}
	for _, review := range input.Reviews {
		add(review["путь"])
		add(review["итог"])
		for _, key := range []string{"моменты", "положительное", "отрицательное", "на_что_обратить_внимание", "что_изменилось"} {
			list, _ := review[key].([]any)
			for _, item := range list {
				add(item)
			}
		}
	}
	return r
}

func (r *summaryResolver) submission(author int) (model.FormSubmission, bool) {
	if author < 1 || author > len(r.authors) {
		return model.FormSubmission{}, false
	}
	for _, submission := range r.db.Submissions {
		if submission.ID == r.authors[author-1] {
			return submission, true
		}
	}
	return model.FormSubmission{}, false
}

func (r *summaryResolver) current(submission model.FormSubmission) map[string]any {
	if analysis, ok := r.analysis[submission.ID]; ok {
		return analysis
	}
	analysis, _ := r.a.essayAnalysis(r.db, submission)
	r.analysis[submission.ID] = analysis
	return analysis
}

func quotesOf(label string, raw any) []quoteView {
	list, _ := raw.([]any)
	views := []quoteView{}
	for _, item := range list {
		if quote := asMap(item); quote != nil {
			views = append(views, quoteView{Label: label, Question: textOf(quote["question"]), Quote: textOf(quote["quote"])})
		}
	}
	return views
}

// source resolves one reference; false when the essay or the point is gone.
func (r *summaryResolver) source(ref string) (sourceView, bool) {
	author, kind, ok := parseRef(ref)
	info, known := r.info[ref]
	submission, found := r.submission(author)
	if !ok || !known || !found {
		return sourceView{}, false
	}
	student, _ := studentOf(r.db, submission)
	view := sourceView{Ref: ref, SubmissionID: submission.ID, Student: student.Name, Platoon: student.Platoon, Text: info.Text, About: info.About, Basis: info.Basis, Quotes: []quoteView{}}
	analysis := r.current(submission)
	var point map[string]any
	switch {
	case kind == "pos" || kind == "neg" || kind == "att":
		list := map[string]string{"pos": "positive", "neg": "negative", "att": "attention"}[kind]
		index, _ := strconv.Atoi(strings.Split(ref, ".")[2])
		if items := listOf(analysis, list); index >= 1 && index <= len(items) {
			point = asMap(items[index-1])
		}
	case strings.HasPrefix(kind, "moment."):
		point = asMap(asMap(analysis["moments"])[strings.TrimPrefix(kind, "moment.")])
	case kind == "conclusion":
		point = asMap(analysis["conclusion"])
	case strings.HasPrefix(kind, "change."):
		key := strings.TrimPrefix(kind, "change.")
		if items := listOf(analysis, "change"); slices.Index(momentKeys, key) >= 0 && slices.Index(momentKeys, key) < len(items) {
			if record := asMap(items[slices.Index(momentKeys, key)]); textOf(record["text"]) == info.Text {
				view.Quotes = append(quotesOf("было", []any{record["before"]}), quotesOf("стало", []any{record["after"]})...)
			}
		}
	}
	if point != nil && textOf(point["text"]) == info.Text {
		view.Quotes = quotesOf("", point["quotes"])
	}
	return view, true
}

// group resolves the references of a common point and counts the authors.
func (r *summaryResolver) group(refs any) sourceGroup {
	list, _ := refs.([]any)
	group := sourceGroup{Of: len(r.authors), Sources: []sourceView{}}
	authors := map[int]bool{}
	for _, item := range list {
		ref, _ := item.(string)
		if view, ok := r.source(ref); ok {
			author, _, _ := parseRef(ref)
			authors[author] = true
			group.Sources = append(group.Sources, view)
		}
	}
	group.Authors = len(authors)
	return group
}

// summaryAnnotations is what the program adds to the text of the model.
func (r *summaryResolver) annotations(output map[string]any) map[string]any {
	common := map[string]any{}
	moments := map[string]any{}
	for _, key := range momentKeys {
		moments[key] = r.group(asMap(asMap(output["moments"])[key])["refs"])
	}
	common["moments"] = moments
	each := func(key string) []sourceGroup {
		groups := []sourceGroup{}
		for _, item := range listOf(output, key) {
			groups = append(groups, r.group(asMap(item)["refs"]))
		}
		return groups
	}
	for _, key := range []string{"positive", "negative", "attention", "paths", "environment", "change"} {
		common[key] = each(key)
	}
	common["conclusion"] = r.group(asMap(output["conclusion"])["refs"])
	// The points of attention no common point stands on stay on the screen:
	// one author may write something important that nobody else did.
	used := map[string]bool{}
	for _, item := range listOf(output, "attention") {
		refs, _ := asMap(item)["refs"].([]any)
		for _, ref := range refs {
			text, _ := ref.(string)
			used[text] = true
		}
	}
	rest := []sourceView{}
	for number := range r.authors {
		submission, _ := r.submission(number + 1)
		analysis := r.current(submission)
		for i := range listOf(analysis, "attention") {
			ref := fmt.Sprintf("%d.att.%d", number+1, i+1)
			if view, ok := r.source(ref); ok && !used[ref] {
				rest = append(rest, view)
			}
		}
	}
	common["notIncluded"] = rest
	return common
}

// latestSummary is the newest stored summary of an assignment.
func latestSummary(db model.Database, assignmentID string) (model.Portrait, bool) {
	var newest model.Portrait
	found := false
	for _, portrait := range db.Portraits {
		if portrait.Task == essayCohortTask && portrait.Source == assignmentID && (!found || portrait.Created >= newest.Created) {
			newest, found = portrait, true
		}
	}
	return newest, found
}

func sameSet(x, y []string) bool {
	return len(x) == len(y) && slices.Equal(slices.Sorted(slices.Values(x)), slices.Sorted(slices.Values(y)))
}

// teacherBlock groups what the analyses say about instructors: the entries the
// model matched with the list are counted per instructor, the rest stay as
// lines as the authors wrote them.
func (a *Service) teacherBlock(db model.Database, vuc string, authors []essayAuthor) map[string]any {
	type line struct {
		Kind         string `json:"kind"`
		Student      string `json:"student"`
		Platoon      string `json:"platoon"`
		SubmissionID string `json:"submissionId"`
		Text         string `json:"text"`
		Who          string `json:"who,omitempty"`
		// Candidates are the instructors that fit a name two of them share,
		// «Кузнецов И. П., майор»; such an entry counts for none of them.
		Candidates []string    `json:"candidates,omitempty"`
		Quotes     []quoteView `json:"quotes"`
	}
	type entry struct {
		Who      string `json:"who"`
		Thanks   int    `json:"thanks"`
		Problems int    `json:"problems"`
		Mentions int    `json:"mentions"`
		Lines    []line `json:"lines"`
	}
	listed := map[string]*entry{}
	order := []string{}
	unlisted := []line{}
	for _, author := range authors {
		student, _ := studentOf(db, author.Submission)
		for _, raw := range listOf(author.Analysis, "teachers") {
			record := asMap(raw)
			who := textOf(record["who"])
			item := line{Kind: textOf(record["kind"]), Student: student.Name, Platoon: student.Platoon, SubmissionID: author.Submission.ID, Text: textOf(record["text"]), Quotes: quotesOf("", record["quotes"])}
			if candidates, has := asList(record["candidates"]); has {
				for _, label := range candidates {
					if other, found := instructorByLabel(db, vuc, textOf(label)); found {
						item.Candidates = append(item.Candidates, instructorShort(other))
					}
				}
			}
			instructor, ok := instructorByLabel(db, vuc, who)
			if !ok {
				if instructorLabelText.MatchString(who) {
					who += " (удалён из списка)"
				}
				item.Who = who
				unlisted = append(unlisted, item)
				continue
			}
			if listed[instructor.ID] == nil {
				listed[instructor.ID] = &entry{Who: instructorWho(instructor), Lines: []line{}}
				order = append(order, instructor.ID)
			}
			e := listed[instructor.ID]
			switch item.Kind {
			case "благодарность":
				e.Thanks++
			case "проблема":
				e.Problems++
			default:
				e.Mentions++
			}
			e.Lines = append(e.Lines, item)
		}
	}
	entries := []*entry{}
	for _, id := range order {
		entries = append(entries, listed[id])
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Thanks+entries[i].Problems+entries[i].Mentions > entries[j].Thanks+entries[j].Problems+entries[j].Mentions
	})
	return map[string]any{"listed": entries, "unlisted": unlisted}
}

// ownEssayAssignment finds an essay assignment of the teacher and its cohort.
func ownEssayAssignment(db model.Database, teacherID, id string) (model.Assignment, model.Cohort, model.Account, error) {
	t, err := managementTeacher(db, teacherID)
	if err != nil {
		return model.Assignment{}, model.Cohort{}, t, err
	}
	for _, as := range db.Assignments {
		if as.ID != id || as.FormID != essayForm || as.OwnerID != t.ID {
			continue
		}
		if cohort, ok := cohortIn(db, as.CohortID, t.VUC); ok {
			return as, cohort, t, nil
		}
	}
	return model.Assignment{}, model.Cohort{}, t, problem(NotFound, "Эссе не найдено.")
}

// EssaySummary is the state of the summary of an assignment: counts, the text of
// the model with the counts of authors behind each common point, the block of
// instructors. The block of instructors does not wait for the model.
func (a *Service) EssaySummary(teacherID, assignmentID string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	return a.essaySummary(db, teacherID, assignmentID)
}

func (a *Service) essaySummary(db model.Database, teacherID, assignmentID string) (any, error) {
	as, cohort, t, err := ownEssayAssignment(db, teacherID, assignmentID)
	if err != nil {
		return nil, err
	}
	_, counts := a.essayStudents(db, as, cohort)
	authors := a.cohortAuthors(db, as.ID)
	result := map[string]any{
		"assignment": map[string]any{"id": as.ID, "title": as.Title, "cohortName": cohort.Name, "year": cohort.Year},
		"counts":     counts,
		"minimum":    cohortMinimum,
		"canCompose": len(authors) >= cohortMinimum,
		"status":     portraitOff,
		"summary":    nil,
		"teachers":   a.teacherBlock(db, t.VUC, authors),
	}
	task, known := a.portraits.task(essayCohortTask)
	if !known {
		return result, nil
	}
	var stored *model.Portrait
	if len(authors) >= cohortMinimum {
		view, record := a.portraitNow(db, task, Source{"assignment", as.ID})
		result["status"], result["reason"], stored = view.Status, view.Reason, record
	} else if a.portraitsOn() {
		result["status"] = portraitNone
	}
	if latest, ok := latestSummary(db, as.ID); stored == nil && ok {
		stored = &latest
	}
	ids := []string{}
	for _, author := range authors {
		ids = append(ids, author.Submission.ID)
	}
	if stored != nil && stored.Source == as.ID {
		var output map[string]any
		if json.Unmarshal(stored.Output, &output) == nil {
			resolver := newSummaryResolver(a, db, *stored)
			result["summary"] = map[string]any{
				"created": stored.Created, "essays": len(stored.Sources), "stale": !sameSet(stored.Sources, ids),
				"output": output, "common": resolver.annotations(output),
			}
			if result["status"] == portraitReady || result["status"] == portraitNone {
				result["status"] = portraitReady
			}
		}
	}
	return result, nil
}

// ComposeEssaySummary queues the summary of an assignment. It is the only way
// to make one: the model is called on the button, not after every submission.
func (a *Service) ComposeEssaySummary(teacherID, assignmentID string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	if _, _, _, err = ownEssayAssignment(db, teacherID, assignmentID); err != nil {
		return nil, err
	}
	if len(a.cohortAuthors(db, assignmentID)) < cohortMinimum {
		return nil, problem(Invalid, "Для сводки нужно хотя бы два эссе с разбором.")
	}
	if _, err = a.RetryPortrait(teacherID, essayCohortTask, assignmentID); err != nil {
		return nil, err
	}
	return a.EssaySummary(teacherID, assignmentID)
}
