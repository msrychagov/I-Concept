package usecase

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func namesakeDB(extra ...model.Instructor) model.Database {
	list := []model.Instructor{
		{ID: "i1", Number: 1, Surname: "Кузнецов", Name: "Илья", Patronymic: "Павлович", Rank: "старший лейтенант"},
		{ID: "i2", Number: 2, Surname: "Кузнецов", Name: "Никита", Patronymic: "Львович", Rank: "лейтенант"},
		{ID: "i3", Number: 3, Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович", Rank: "полковник"},
		{ID: "i4", Number: 4, Surname: "Сидоров", Name: "Игорь", Patronymic: "Петрович", Rank: "майор"},
		{ID: "i5", Number: 5, Surname: "Петрова", Name: "Анна", Patronymic: "Сергеевна"},
		{ID: "i6", Number: 6, Surname: "Ковальчук", Name: "Олег", Patronymic: "Ильич", Rank: "подполковник"},
	}
	for _, instructor := range extra {
		list = append(list, instructor)
	}
	for i := range list {
		list[i].VUC = "ВУЦ"
	}
	return model.Database{
		Students:    []model.Account{{ID: "s1", Role: "student", VUC: "ВУЦ"}},
		Submissions: []model.FormSubmission{{ID: "sub1", FormID: essayForm, StudentID: "s1", Completed: true}},
		Instructors: list,
	}
}

// teacherRecord is an entry of the answer with a label; its quote is of answer 2.4.
func teacherRecord(who string) map[string]any {
	return map[string]any{"who": who, "kind": "благодарность", "text": "Благодарит.", "quotes": []any{map[string]any{"question": "2.4", "quote": "три слова подряд"}}}
}

func cleanedTeachers(t *testing.T, db model.Database, answer string, who string) map[string]any {
	t.Helper()
	payload := essayPayloadFor(map[string]string{"1.1": "Ответ один.", "1.2": "Ответ два.", "2.4": answer}, nil)
	output := map[string]any{"teachers": []any{teacherRecord(who)}}
	out, changes := cleanEssayAnswer(nil, db, Source{"submission", "sub1"}, output, payload)
	record := out["teachers"].([]any)[0].(map[string]any)
	if _, stripped := record["candidates"]; stripped != (changes == 1) {
		t.Fatalf("%d changes for %v", changes, record)
	}
	return record
}

func labelsOf(record map[string]any) []string {
	labels := []string{}
	for _, item := range record["candidates"].([]any) {
		labels = append(labels, item.(string))
	}
	slices.Sort(labels)
	return labels
}

func TestNamesakeLabelIsTakenOffWhenTheWordsDoNotTellThemApart(t *testing.T) {
	db := namesakeDB()
	// Only the surname two instructors share, in a case form.
	record := cleanedTeachers(t, db, "Спасибо Кузнецову за помощь на зачёте.", "[преподаватель 1]")
	if record["who"] != "Кузнецову" || fmt.Sprint(labelsOf(record)) != "[[преподаватель 1] [преподаватель 2]]" {
		t.Fatalf("surname only: %v", record)
	}
	// Only the name and patronymic two instructors share.
	record = cleanedTeachers(t, db, "Хочу поблагодарить Игоря Петровича за терпение.", "[преподаватель 3]")
	if record["who"] != "Игоря Петровича" || fmt.Sprint(labelsOf(record)) != "[[преподаватель 3] [преподаватель 4]]" {
		t.Fatalf("name and patronymic only: %v", record)
	}
	// The rank is the same for both: still no way to tell.
	same := namesakeDB()
	same.Instructors[1].Rank = "старший лейтенант"
	if record = cleanedTeachers(t, same, "Спасибо старшему лейтенанту Кузнецову.", "[преподаватель 1]"); record["candidates"] == nil {
		t.Fatalf("same rank: %v", record)
	}
	if got := record["who"].(string); !strings.Contains(got, "Кузнецову") || !strings.Contains(got, "старшему") || !strings.Contains(got, "лейтенанту") {
		t.Fatalf("words as written: %q", got)
	}
}

func TestWordsOfAnEntryAreWrittenOnceInTheOrderTheyCome(t *testing.T) {
	db := namesakeDB()
	record := cleanedTeachers(t, db, "Игорь Петрович помог мне, спасибо Игорю Петровичу.", "[преподаватель 3]")
	if record["who"] != "Игорь Петрович" {
		t.Fatalf("%v", record)
	}
	record = cleanedTeachers(t, db, "Кузнецов объяснил тему. Благодарю Кузнецова.", "[преподаватель 1]")
	if record["who"] != "Кузнецов" {
		t.Fatalf("%v", record)
	}
}

func TestNamesakeLabelStaysWhenTheWordsTellThemApart(t *testing.T) {
	petrov := model.Instructor{ID: "i7", Number: 7, Surname: "Петров", Name: "Игорь", Patronymic: "Петрович", Rank: "подполковник"}
	cases := map[string]struct {
		db     model.Database
		answer string
		who    string
	}{
		"surname and rank of one of two Kuznetsovs": {namesakeDB(), "Помог лейтенант Кузнецов.", "[преподаватель 2]"},
		"the other rank in a case form":             {namesakeDB(), "Поблагодарю старшего лейтенанта Кузнецова.", "[преподаватель 1]"},
		"name and surname":                          {namesakeDB(), "Никита Кузнецов объяснил тему.", "[преподаватель 2]"},
		"surname, name and patronymic":              {namesakeDB(petrov), "Петров Игорь Петрович помог.", "[преподаватель 7]"},
		"surname of one of two with a shared name":  {namesakeDB(), "Благодарю Иванова Игоря Петровича.", "[преподаватель 3]"},
		"only the rank, which the other has not":    {namesakeDB(), "Помог полковник, спасибо.", "[преподаватель 3]"},
		"a typo: nothing to check":                  {namesakeDB(), "Спасибо Кузнцову за помощь.", "[преподаватель 1]"},
		"an abbreviation: nothing to check":         {namesakeDB(), "Спасибо Кузн. за помощь.", "[преподаватель 2]"},
		"no namesake":                               {namesakeDB(), "Спасибо Анне Сергеевне.", "[преподаватель 5]"},
		"no namesake, only the surname":             {namesakeDB(), "Ковальчук помог.", "[преподаватель 6]"},
		"a label that is not in the list":           {namesakeDB(), "Спасибо Кузнецову.", "[преподаватель 9]"},
		"a name as the author wrote it":             {namesakeDB(), "Спасибо Кузнецову.", "полковник Ивнов"},
	}
	for name, c := range cases {
		record := cleanedTeachers(t, c.db, c.answer, c.who)
		if record["who"] != c.who || record["candidates"] != nil {
			t.Errorf("%s: %v", name, record)
		}
	}
	// Words of Il'ya are not words of Il'ich, and Petrov is not Petrovich.
	db := namesakeDB()
	if record := cleanedTeachers(t, db, "Олег Ильич, спасибо.", "[преподаватель 6]"); record["who"] != "[преподаватель 6]" {
		t.Fatalf("%v", record)
	}
	if found, _ := foundIn(db.Instructors[0], []string{"Ильича поблагодарю"}); found["name"] {
		t.Fatal("Ильича is not Илья")
	}
	if found, _ := foundIn(model.Instructor{Surname: "Петров"}, []string{"Игорь Петрович"}); found["surname"] {
		t.Fatal("Петрович is not Петров")
	}
}

func TestCheckAllowsEntriesWithCandidatesFromTheList(t *testing.T) {
	answers := map[string]string{"1.1": "Я пошёл «в ВУЦ» ради «службы» и отсрочки.", "1.2": "Первые дни было страшно.", "2.4": "Спасибо Кузнецову за помощь на зачёте."}
	payload := essayPayloadFor(answers, nil, "[преподаватель 1]", "[преподаватель 2]")
	good := func(mutate func(map[string]any)) string {
		output := goodEssayFor(payload)
		record := map[string]any{"who": "Кузнецову", "candidates": []any{"[преподаватель 1]", "[преподаватель 2]"}, "kind": "благодарность", "text": "Благодарит.", "quotes": []any{map[string]any{"question": "2.4", "quote": "Спасибо Кузнецову за помощь"}}}
		mutate(record)
		output["teachers"] = []any{record}
		return strings.Join(checkEssayAnswer(roundTrip(t, output), payload), " | ")
	}
	if problems := good(func(map[string]any) {}); problems != "" {
		t.Fatalf("good entry: %s", problems)
	}
	if problems := good(func(r map[string]any) {
		r["candidates"] = []any{"[преподаватель 1]", "[преподаватель 9]"}
	}); !strings.Contains(problems, "кандидат не из списка") {
		t.Fatalf("a label out of the list: %s", problems)
	}
	if problems := good(func(r map[string]any) { r["candidates"] = []any{"[преподаватель 1]"} }); !strings.Contains(problems, "не меньше двух кандидатов") {
		t.Fatalf("one candidate: %s", problems)
	}
	if problems := good(func(r map[string]any) { r["who"] = "[преподаватель 1]" }); !strings.Contains(problems, "нужны слова автора") {
		t.Fatalf("a label with candidates: %s", problems)
	}
}

// Through the layer of v1 (v2 has no labels): the model labels «Кузнецову», the
// stored analysis has no label.
func TestAnalysisStoresNamesakeEntryWithoutTheLabelAndTheSummaryShowsItApart(t *testing.T) {
	f := newEssayPortraitFixtureAt(t, "v1")
	journalOut := captureJournal(t)
	for _, entry := range []InstructorInput{{Surname: "Кузнецов", Name: "Илья", Patronymic: "Павлович", Rank: "майор"}, {Surname: "Кузнецов", Name: "Никита", Patronymic: "Львович", Rank: "капитан"}} {
		if _, err := f.service.SaveInstructor(f.owner, "", entry); err != nil {
			t.Fatal(err)
		}
	}
	m := essayModelV1(func(answer map[string]any, input essayInput) {
		for _, row := range input.Essay {
			if row.Number == "2.4" {
				answer["teachers"] = []any{map[string]any{"who": "[преподаватель 1]", "kind": "благодарность", "text": "Благодарит за помощь.", "quotes": []any{quoteOf(row)}}}
			}
		}
	})
	f.service.UseModel(m)
	_, id := f.submitted("namesake@example.test", "2401", f.assignment, map[string]string{"2.4": "Спасибо Кузнецову за помощь на зачёте."})
	f.idle()
	view, err := f.portrait(f.owner, id)
	if err != nil || view.Status != portraitReady {
		t.Fatalf("%+v %v", view, err)
	}
	var output struct {
		Teachers []struct {
			Who        string   `json:"who"`
			Candidates []string `json:"candidates"`
		} `json:"teachers"`
	}
	if err = json.Unmarshal(view.Portrait.Output, &output); err != nil || len(output.Teachers) != 1 || output.Teachers[0].Who != "Кузнецову" || len(output.Teachers[0].Candidates) != 2 {
		t.Fatalf("stored teachers: %+v %v", output, err)
	}
	if !strings.Contains(journalOut.String(), "answer cleaned") || strings.Contains(journalOut.String(), "Кузнецову") {
		t.Fatalf("journal: %s", journalOut.String())
	}
	summary := f.summary(f.owner)
	block := summary["teachers"].(map[string]any)
	unlisted := block["unlisted"].([]any)
	if len(block["listed"].([]any)) != 0 || len(unlisted) != 1 {
		t.Fatalf("block: %+v", block)
	}
	entry := unlisted[0].(map[string]any)
	if entry["who"] != "Кузнецову" || fmt.Sprint(entry["candidates"]) != "[Кузнецов И. П., майор Кузнецов Н. Л., капитан]" {
		t.Fatalf("entry: %+v", entry)
	}
}
