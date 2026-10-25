package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// The portrait task «essay-cohort»: the common picture of all the analyses of
// one assignment of an essay to a cohort. The model gets the analyses of the
// authors without quotes and without the teachers, numbered in a stable
// random order, and writes only the common text with references to the points
// it is made of. How many authors stand behind a point is counted by the
// program from the references. The instruction, the schema and what the
// program checks are in Вики/Эссе выпускника/Инструкция по выпуску.md.

// cohortMinimum is the number of analysed essays below which there is nothing common.
const cohortMinimum = 2

// momentNames are the keys of the four moments and their names, in order.
var momentNames = map[string]string{"beliefs": "убеждения", "emotions": "эмоции", "values": "ценности", "abilities": "восприятие своих возможностей"}

// essayAuthor is a submitted essay of the assignment with the analysis of it
// and the version of the instruction the analysis was made by.
type essayAuthor struct {
	Submission model.FormSubmission
	Analysis   map[string]any
	Version    string
}

func findAssignment(db model.Database, id string) (model.Assignment, bool) {
	for _, assignment := range db.Assignments {
		if assignment.ID == id {
			return assignment, true
		}
	}
	return model.Assignment{}, false
}

// essayAnalysis returns the analysis of a submitted essay that the teacher
// sees, as the layer knows it, and its status when there is none yet.
func (a *Service) essayAnalysis(db model.Database, submission model.FormSubmission) (map[string]any, PortraitView) {
	task, ok := a.portraits.task(essayForm)
	if !ok || !task.Accepts(db, Source{"submission", submission.ID}) {
		return nil, PortraitView{Status: portraitOff}
	}
	view, _ := a.portraitNow(db, task, Source{"submission", submission.ID})
	if view.Status != portraitReady {
		return nil, view
	}
	var output map[string]any
	if json.Unmarshal(view.Portrait.Output, &output) != nil {
		return nil, PortraitView{Status: portraitFailed, Reason: "Не удалось прочитать разбор."}
	}
	return output, view
}

// assignmentSubmissions are the submitted essays of the assignment, oldest first.
func assignmentSubmissions(db model.Database, assignmentID string) []model.FormSubmission {
	list := []model.FormSubmission{}
	for _, submission := range db.Submissions {
		if submission.AssignmentID == assignmentID && submission.FormID == essayForm && submission.Completed {
			list = append(list, submission)
		}
	}
	slices.SortStableFunc(list, func(x, y model.FormSubmission) int { return strings.Compare(x.CompletedAt, y.CompletedAt) })
	return list
}

// authorOrder is the stable permutation of the authors: by the hash of the
// assignment and the essay, so equal sets give equal input and the number tells
// nothing about who wrote.
func authorOrder(assignmentID string, submissions []model.FormSubmission) []model.FormSubmission {
	rank := func(s model.FormSubmission) string {
		sum := sha256.Sum256([]byte(assignmentID + "\n" + s.ID))
		return hex.EncodeToString(sum[:])
	}
	ordered := slices.Clone(submissions)
	slices.SortFunc(ordered, func(x, y model.FormSubmission) int { return strings.Compare(rank(x), rank(y)) })
	return ordered
}

// cohortAuthors are the essays of the assignment that have an analysis the
// summary can be made of, in the order of the numbers the model gets: the
// summary of v2 takes only the analyses of v2 (decision Д9), an analysis of v1
// shown while the new one is made counts as none.
func (a *Service) cohortAuthors(db model.Database, assignmentID string) []essayAuthor {
	return a.cohortAuthorsOf(db, assignmentID, a.cohortShape())
}

// cohortShape is the shape of analysis the registered summary is made of.
func (a *Service) cohortShape() int {
	if task, ok := a.portraits.task(essayCohortTask); ok {
		return essayShape(task.Version)
	}
	return 1
}

// cohortAuthorsOf are the essays of the assignment with an analysis of the
// shape (1 for v1, 2 for v2 and later, 0 for any), in the order of the numbers.
func (a *Service) cohortAuthorsOf(db model.Database, assignmentID string, shape int) []essayAuthor {
	authors := []essayAuthor{}
	for _, submission := range authorOrder(assignmentID, assignmentSubmissions(db, assignmentID)) {
		analysis, view := a.essayAnalysis(db, submission)
		if analysis == nil || shape != 0 && essayShape(view.Portrait.Version) != shape {
			continue
		}
		authors = append(authors, essayAuthor{submission, analysis, view.Portrait.Version})
	}
	return authors
}

// essayAnalysisOf finds the stored analysis of a submitted essay made by a
// version of the instruction for its input now: a summary of v1 shows the
// quotes of the analyses of v1 it was made of.
func (a *Service) essayAnalysisOf(db model.Database, submission model.FormSubmission, version string) map[string]any {
	task, ok := a.portraits.task(essayForm)
	if !ok {
		return nil
	}
	payload, err := a.essayPayload(db, submission, essayShape(version) == 1)
	if err != nil {
		return nil
	}
	if _, err = maskPersonal(payload, task.Keep...); err != nil {
		return nil
	}
	fingerprint := payloadFingerprint(payload)
	var newest model.Portrait
	found := false
	for _, portrait := range db.Portraits {
		if portrait.Task == essayForm && portrait.TaskVersion == version && portrait.Fingerprint == fingerprint && (!found || portrait.Created >= newest.Created) {
			newest, found = portrait, true
		}
	}
	var output map[string]any
	if !found || json.Unmarshal(newest.Output, &output) != nil {
		return nil
	}
	return output
}

func listOf(analysis map[string]any, key string) []any {
	list, _ := analysis[key].([]any)
	return list
}

// cohortPayload is the input of the summary: the analyses without quotes and
// without teachers, every point under its number.
func (a *Service) cohortPayload(db model.Database, assignmentID string) (*object, error) {
	authors := a.cohortAuthorsOf(db, assignmentID, 1)
	if len(authors) < cohortMinimum {
		return nil, problem(Invalid, "Для сводки нужно хотя бы два эссе с разбором.")
	}
	payload := newObject()
	reviews := []any{}
	withPrevious := 0
	for n, author := range authors {
		id := strconv.Itoa(n + 1)
		review := newObject()
		review.set("автор", id)
		path := newObject()
		path.set("номер", id+".path")
		path.set("текст", textOf(author.Analysis["summary"]))
		review.set("путь", path)
		conclusion := newObject()
		conclusion.set("номер", id+".conclusion")
		conclusion.set("текст", textOf(asMap(author.Analysis["conclusion"])["text"]))
		review.set("итог", conclusion)
		moments := []any{}
		for _, key := range momentKeys {
			moment := newObject()
			moment.set("номер", id+"."+key)
			moment.set("момент", momentNames[key])
			moment.set("текст", textOf(asMap(asMap(author.Analysis["moments"])[key])["text"]))
			moments = append(moments, moment)
		}
		review.set("моменты", moments)
		for _, list := range []struct{ key, short, name string }{{"positive", "pos", "положительное"}, {"negative", "neg", "отрицательное"}, {"attention", "att", "на_что_обратить_внимание"}} {
			points := []any{}
			for i, raw := range listOf(author.Analysis, list.key) {
				point := asMap(raw)
				entry := newObject()
				entry.set("номер", fmt.Sprintf("%s.%s.%d", id, list.short, i+1))
				entry.set("текст", textOf(point["text"]))
				entry.set("о_чём", textOf(point["about"]))
				entry.set("основа", textOf(point["basis"]))
				points = append(points, entry)
			}
			review.set(list.name, points)
		}
		if changes := listOf(author.Analysis, "change"); len(changes) > 0 {
			withPrevious++
			entries := []any{}
			for i, raw := range changes {
				if i >= len(momentKeys) {
					break
				}
				entry := newObject()
				entry.set("номер", id+".change."+momentKeys[i])
				entry.set("момент", momentNames[momentKeys[i]])
				entry.set("текст", textOf(asMap(raw)["text"]))
				entries = append(entries, entry)
			}
			review.set("что_изменилось", entries)
		}
		reviews = append(reviews, review)
	}
	payload.set("эссе_в_сводке", len(authors))
	payload.set("с_прежним_эссе", withPrevious)
	payload.set("разборы", reviews)
	return payload, nil
}

func (a *Service) cohortTask(prompt promptbank.Prompt) portraitTask {
	owns := func(db model.Database, s Source, teacherID string) bool {
		assignment, ok := findAssignment(db, s.ID)
		return s.Kind == "assignment" && ok && assignment.FormID == essayForm && inTeacherScope(assignment.OwnerID, teacherID)
	}
	// The summary of v2 is made of the analyses of v2 (essay_v2.go).
	shape := essayShape(prompt.Version)
	payload, clean, check := (*Service).cohortPayload, cleanCohortAnswer, checkCohortAnswer
	if shape == 2 {
		payload, clean, check = (*Service).cohortPayloadV2, cleanCohortAnswerV2, checkCohortAnswerV2
	}
	return portraitTask{
		ID: essayCohortTask, Version: prompt.Version, Prompt: prompt,
		MaxTokens: 6000, Attempts: 5, Manual: true,
		Accepts: func(db model.Database, s Source) bool { return owns(db, s, "") },
		Payload: func(a *Service, db model.Database, s Source) (*object, error) { return payload(a, db, s.ID) },
		Stamp: func(a *Service, db model.Database, s Source) (string, []string) {
			ids := []string{}
			for _, author := range a.cohortAuthorsOf(db, s.ID, shape) {
				ids = append(ids, author.Submission.ID)
			}
			return s.ID, ids
		},
		Clean:   clean,
		Check:   check,
		Visible: owns,
	}
}

// parseRef reads «3.pos.2», «3.beliefs», «3.change.beliefs», «3.path» or
// «3.conclusion» into the number of the author and the kind of the point:
// pos, neg, att, path, conclusion, moment.beliefs, change.beliefs; and the
// numbers of v2: «3.formation», «3.moments», «3.environment», «3.rec.2» into
// formation, moments, environment, rec.
func parseRef(ref string) (author int, kind string, ok bool) {
	parts := strings.Split(ref, ".")
	number, err := strconv.Atoi(parts[0])
	if err != nil || number < 1 || strconv.Itoa(number) != parts[0] || len(parts) < 2 || len(parts) > 3 {
		return 0, "", false
	}
	_, moment := momentNames[parts[1]]
	switch {
	case len(parts) == 2 && moment:
		return number, "moment." + parts[1], true
	case len(parts) == 2 && (parts[1] == "path" || parts[1] == "conclusion" || slices.Contains(essayParagraphs, parts[1])):
		return number, parts[1], true
	case len(parts) == 3 && (parts[1] == "pos" || parts[1] == "neg" || parts[1] == "att" || parts[1] == "rec"):
		index, err := strconv.Atoi(parts[2])
		return number, parts[1], err == nil && index >= 1 && strconv.Itoa(index) == parts[2]
	case len(parts) == 3 && parts[1] == "change":
		_, known := momentNames[parts[2]]
		return number, "change." + parts[2], known
	}
	return 0, "", false
}

// payloadRefs are the numbers of all the points of a summary input.
func payloadRefs(payload *object) map[string]bool {
	refs := map[string]bool{}
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case *object:
			if number, ok := v.values["номер"].(string); ok {
				refs[number] = true
			}
			for _, key := range v.keys {
				walk(v.values[key])
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(payload)
	return refs
}

// cohortWords finds what the summary must not say: how many authors stand
// behind a point (the program counts them), digits, labels of instructors.
var cohortWords = regexp.MustCompile(`(?i)(^|[^\p{L}])(большинств\p{L}*|половин\p{L}*|все|каждый|каждая|каждое|каждые|никто|единиц\p{L}*)([^\p{L}]|$)|\p{N}|\[преподаватель`)

// cohortProblems finds the forbidden words of a text of the summary.
func cohortTextProblems(where string, raw any) []string {
	text := textOf(raw)
	if text == "" {
		return []string{where + ": пустой текст"}
	}
	problems := wordProblems(where, text)
	if found := cohortWords.FindStringSubmatch(text); found != nil {
		word := strings.ToLower(strings.TrimSpace(found[0]))
		if found[2] == "" {
			word = "цифра или метка преподавателя"
		}
		problems = append(problems, fmt.Sprintf("%s: слово о количестве, цифра или метка («%s»)", where, word))
	}
	return problems
}

// refsOf reads the list of references of a point and checks it: they are all
// in the input and of an allowed kind, and they come from at least minAuthors
// authors. It returns the problems and the valid references.
func refsOf(where string, raw any, known map[string]bool, allow func(kind string) bool, minAuthors int) ([]string, []string) {
	list, ok := raw.([]any)
	if !ok {
		return []string{where + ": refs не список"}, nil
	}
	problems, refs, authors := []string{}, []string{}, map[int]bool{}
	for _, item := range list {
		ref, isText := item.(string)
		ref = strings.TrimSpace(ref)
		author, kind, valid := parseRef(ref)
		switch {
		case !isText || !valid || !known[ref]:
			problems = append(problems, fmt.Sprintf("%s: ссылки «%s» нет во входных данных", where, truncate(ref, 20)))
		case !allow(kind):
			problems = append(problems, fmt.Sprintf("%s: ссылка «%s» не того вида", where, ref))
		default:
			refs = append(refs, ref)
			authors[author] = true
		}
	}
	if len(authors) < minAuthors {
		problems = append(problems, fmt.Sprintf("%s: ссылки нужны не меньше чем на %d разных авторов", where, minAuthors))
	}
	return problems, refs
}

func truncate(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return text
}

func kindIs(want ...string) func(string) bool {
	return func(kind string) bool { return slices.Contains(want, kind) }
}

// cleanRefs keeps the references that are in the input, of an allowed kind and
// not yet seen in the list; it returns them and the number of those dropped.
func cleanRefs(raw any, known map[string]bool, allow func(string) bool) ([]any, int) {
	list, _ := raw.([]any)
	kept, dropped, seen := []any{}, 0, map[string]bool{}
	for _, item := range list {
		ref, isText := item.(string)
		ref = strings.TrimSpace(ref)
		_, kind, valid := parseRef(ref)
		if !isText || !valid || !known[ref] || !allow(kind) || seen[ref] {
			dropped++
			continue
		}
		seen[ref] = true
		kept = append(kept, ref)
	}
	if _, isList := raw.([]any); !isList {
		dropped++
	}
	return kept, dropped
}

func refAuthors(refs []any) map[int]bool {
	authors := map[int]bool{}
	for _, item := range refs {
		ref, _ := item.(string)
		if author, _, ok := parseRef(ref); ok {
			authors[author] = true
		}
	}
	return authors
}

// cleanCohortAnswer drops from the answer of the model what the program can
// drop without asking again (decided 23 October 2026): references that are not
// in the input, of the wrong kind or repeated; a source point already taken by
// an earlier common point of the same list and an author already put in an
// earlier path; and the common points left with fewer than two authors. The
// four moments, the conclusion and «change» keep their records: if they are
// left without authors, the check refuses the answer. It returns the answer
// and the number of changes.
func cleanCohortAnswer(a *Service, db model.Database, s Source, output map[string]any, payload *object) (map[string]any, int) {
	known := payloadRefs(payload)
	changes := 0
	anything := func(string) bool { return true }
	clean := func(point map[string]any, allow func(string) bool) []any {
		refs, dropped := cleanRefs(point["refs"], known, allow)
		changes += dropped
		point["refs"] = refs
		return refs
	}

	for _, key := range momentKeys {
		if moment := asMap(asMap(output["moments"])[key]); moment != nil {
			clean(moment, kindIs("moment."+key))
		}
	}

	// Lists of points: a source point stays in the first common point that has it.
	for _, list := range []struct{ key, kind string }{{"positive", "pos"}, {"negative", "neg"}, {"attention", "att"}} {
		points, ok := asList(output[list.key])
		if !ok {
			continue
		}
		used := map[string]bool{}
		kept := []any{}
		for _, raw := range points {
			point := asMap(raw)
			if point == nil {
				kept = append(kept, raw)
				continue
			}
			refs := clean(point, kindIs(list.kind))
			fresh := []any{}
			for _, ref := range refs {
				if text, _ := ref.(string); used[text] {
					changes++
				} else {
					fresh = append(fresh, ref)
				}
			}
			point["refs"] = fresh
			if len(refAuthors(fresh)) < 2 {
				changes++
				continue
			}
			for _, ref := range fresh {
				used[ref.(string)] = true
			}
			kept = append(kept, point)
		}
		output[list.key] = kept
	}

	// Paths: an author goes into the first path that has them.
	if paths, ok := asList(output["paths"]); ok {
		taken := map[int]bool{}
		kept := []any{}
		for _, raw := range paths {
			path := asMap(raw)
			if path == nil {
				kept = append(kept, raw)
				continue
			}
			refs := clean(path, kindIs("path", "conclusion"))
			fresh := []any{}
			for _, ref := range refs {
				if author, _, _ := parseRef(ref.(string)); taken[author] {
					changes++
				} else {
					fresh = append(fresh, ref)
				}
			}
			path["refs"] = fresh
			authors := refAuthors(fresh)
			if len(authors) < 2 {
				changes++
				continue
			}
			for author := range authors {
				taken[author] = true
			}
			kept = append(kept, path)
		}
		output["paths"] = kept
	}

	if environment, ok := asList(output["environment"]); ok {
		kept := []any{}
		for _, raw := range environment {
			point := asMap(raw)
			if point == nil {
				kept = append(kept, raw)
				continue
			}
			if len(refAuthors(clean(point, anything))) < 2 {
				changes++
				continue
			}
			kept = append(kept, point)
		}
		output["environment"] = kept
	}

	if conclusion := asMap(output["conclusion"]); conclusion != nil {
		clean(conclusion, anything)
	}

	// «change» keeps all its records, each with the references of its own moment.
	if records, ok := asList(output["change"]); ok {
		for n, raw := range records {
			if record := asMap(raw); record != nil && n < len(momentKeys) {
				clean(record, kindIs("change."+momentKeys[n]))
			}
		}
	}
	return output, changes
}

// checkCohortAnswer returns what is wrong with an answer to the instruction
// for the summary after the program has cleaned it: the overview, the four
// moments, the conclusion and «change» (each with its authors), the words of
// quantity and the beginning of the conclusion. The common points the cleaning
// left may be none. Without cleaning it also finds wrong references.
func checkCohortAnswer(output map[string]any, payload *object) []string {
	known := payloadRefs(payload)
	hasChange := false
	for ref := range known {
		if _, kind, ok := parseRef(ref); ok && strings.HasPrefix(kind, "change.") {
			hasChange = true
		}
	}
	problems := cohortTextProblems("overview", output["overview"])

	moments := asMap(output["moments"])
	for _, key := range momentKeys {
		where := "moments." + key
		moment := asMap(moments[key])
		if moment == nil {
			problems = append(problems, where+": нет момента")
			continue
		}
		problems = append(problems, cohortTextProblems(where, moment["text"])...)
		more, _ := refsOf(where, moment["refs"], known, kindIs("moment."+key), 2)
		problems = append(problems, more...)
	}

	for _, list := range []struct{ key, kind string }{{"positive", "pos"}, {"negative", "neg"}, {"attention", "att"}} {
		points, ok := asList(output[list.key])
		if !ok || len(points) > 7 {
			problems = append(problems, fmt.Sprintf("%s: пунктов должно быть не больше 7", list.key))
			continue
		}
		used := map[string]int{}
		for n, raw := range points {
			where, point := fmt.Sprintf("%s, пункт %d", list.key, n+1), asMap(raw)
			if point == nil {
				problems = append(problems, where+": не запись")
				continue
			}
			problems = append(problems, cohortTextProblems(where, point["text"])...)
			if !oneOf(point["about"], essayAbout) {
				problems = append(problems, where+": about должно быть «о себе» или «о ВУЦ»")
			}
			more, refs := refsOf(where, point["refs"], known, kindIs(list.kind), 2)
			problems = append(problems, more...)
			for _, ref := range refs {
				if first, taken := used[ref]; taken && first != n+1 {
					problems = append(problems, fmt.Sprintf("%s: «%s» уже в пункте %d", where, ref, first))
				}
				used[ref] = n + 1
			}
		}
	}

	paths, ok := asList(output["paths"])
	if !ok || len(paths) > 4 {
		problems = append(problems, "paths: путей должно быть не больше 4")
	}
	pathOf := map[int]int{}
	for n, raw := range paths {
		where, path := fmt.Sprintf("paths, путь %d", n+1), asMap(raw)
		if path == nil {
			problems = append(problems, where+": не запись")
			continue
		}
		problems = append(problems, cohortTextProblems(where, path["text"])...)
		more, refs := refsOf(where, path["refs"], known, kindIs("path", "conclusion"), 2)
		problems = append(problems, more...)
		for _, ref := range refs {
			author, _, _ := parseRef(ref)
			if first, taken := pathOf[author]; taken && first != n+1 {
				problems = append(problems, fmt.Sprintf("%s: автор из пути %d записан и сюда", where, first))
			}
			pathOf[author] = n + 1
		}
	}

	environment, ok := asList(output["environment"])
	if !ok || len(environment) > 5 {
		problems = append(problems, "environment: пунктов должно быть не больше 5")
	}
	anything := func(string) bool { return true }
	for n, raw := range environment {
		where, point := fmt.Sprintf("environment, пункт %d", n+1), asMap(raw)
		if point == nil {
			problems = append(problems, where+": не запись")
			continue
		}
		problems = append(problems, cohortTextProblems(where, point["text"])...)
		if !oneOf(point["moment"], essayMoments) {
			problems = append(problems, where+": moment не из списка моментов")
		}
		more, _ := refsOf(where, point["refs"], known, anything, 2)
		problems = append(problems, more...)
	}

	conclusion := asMap(output["conclusion"])
	text := strings.ToLower(textOf(conclusion["text"]))
	problems = append(problems, cohortTextProblems("conclusion", conclusion["text"])...)
	if text != "" && !strings.HasPrefix(text, "по разборам видно") && !strings.HasPrefix(text, "по разборам можно предположить") {
		problems = append(problems, "conclusion: должен начинаться со слов «по разборам видно» или «по разборам можно предположить»")
	}
	more, _ := refsOf("conclusion", conclusion["refs"], known, anything, 2)
	problems = append(problems, more...)

	changes, ok := asList(output["change"])
	switch {
	case !ok:
		problems = append(problems, "change: нет списка")
	case !hasChange && len(changes) != 0:
		problems = append(problems, "change: ни у кого нет прежнего эссе, список должен быть пустым")
	case hasChange && len(changes) != len(momentKeys):
		problems = append(problems, fmt.Sprintf("change: нужно %d записи по порядку, получено %d", len(momentKeys), len(changes)))
	}
	if hasChange && len(changes) == len(momentKeys) {
		for n, raw := range changes {
			where, record := fmt.Sprintf("change, запись %d", n+1), asMap(raw)
			if record == nil || textOf(record["moment"]) != momentNames[momentKeys[n]] {
				problems = append(problems, where+": момент не тот или не по порядку, нужно «"+momentNames[momentKeys[n]]+"»")
				continue
			}
			problems = append(problems, cohortTextProblems(where, record["text"])...)
			more, _ := refsOf(where, record["refs"], known, kindIs("change."+momentKeys[n]), 1)
			problems = append(problems, more...)
		}
	}
	return problems
}
