package usecase

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/formbank"
)

// demoTestPortraits are the data of the finished demo sessions with the
// methods, by the surname of the student.
func demoTestPortraits(t *testing.T) (*Service, map[string]*TestPortraitData) {
	t.Helper()
	a := testApp(t)
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*TestPortraitData{}
	for _, record := range db.Sessions {
		s := record.Session
		if !s.Completed || !s.Methods || s.Kind != "student" {
			continue
		}
		if data := a.TestPortrait(s); data != nil {
			out[strings.Fields(s.Participant)[0]] = data
		}
	}
	if len(out) == 0 {
		t.Fatal("no finished demo session with the methods")
	}
	return a, out
}

// The conclusions on the demo students are kept in testdata/test_conclusion;
// with -update they are rewritten.
func TestTestConclusionOfDemoStudents(t *testing.T) {
	a, portraits := demoTestPortraits(t)
	dir := filepath.Join("testdata", "test_conclusion")
	for _, name := range sortedKeys(portraits) {
		t.Run(name, func(t *testing.T) {
			c, _, err := a.testConclusionOf(portraits[name])
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(c, "", " ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			path := filepath.Join(dir, name+".json")
			if *update {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s differs; run go test -run TestTestConclusionOfDemoStudents -update\n%s", path, got)
			}
		})
	}
}

// Every conclusion has a summary, the four parts and two to four
// recommendations; no text has a forbidden word, a number or a broken phrase.
func TestTestConclusionWords(t *testing.T) {
	p, err := loadedTestPhrases()
	if err != nil {
		t.Fatal(err)
	}
	digits := regexp.MustCompile(`\d`)
	a, portraits := demoTestPortraits(t)
	for _, name := range sortedKeys(portraits) {
		c := a.testConclusion(portraits[name])
		if c == nil || c.Summary == "" || len(c.Details) != 4 {
			t.Fatalf("%s: incomplete conclusion %+v", name, c)
		}
		if n := len(c.Recommendations); n < 2 || n > 4 {
			t.Errorf("%s: %d recommendations", name, n)
		}
		texts := []string{c.Summary}
		for _, part := range c.Details {
			texts = append(texts, part.Text)
		}
		for _, r := range c.Recommendations {
			texts = append(texts, r.Text)
		}
		for _, text := range texts {
			lower := strings.ToLower(text)
			for _, word := range p.Forbidden {
				if strings.Contains(lower, word) {
					t.Errorf("%s: %q in %q", name, word, text)
				}
			}
			if strings.Contains(text, "{") || strings.Contains(text, "  ") || strings.Contains(text, " ,") || strings.Contains(text, "..") || strings.Contains(text, ";.") {
				t.Errorf("%s: a broken phrase %q", name, text)
			}
			// The stories of the student may have numbers and any marks; the
			// rest may not, and has no colon, dash or semicolon (decided 13
			// November 2026).
			before, _, _ := strings.Cut(text, "В ответах о деятельности")
			if digits.MatchString(before) {
				t.Errorf("%s: a number in %q", name, text)
			}
			if strings.ContainsAny(before, ":;—–") || strings.Contains(before, " - ") {
				t.Errorf("%s: a colon or a dash in %q", name, text)
			}
		}
		codes := []string{}
		for _, r := range c.Recommendations {
			if slices.Contains(codes, r.Code) {
				t.Errorf("%s: %s twice", name, r.Code)
			}
			codes = append(codes, r.Code)
		}
	}
}

// The phrase book has both forms of every direction of the catalogue and
// none of the forbidden words.
func TestTestPhraseBook(t *testing.T) {
	p, err := loadedTestPhrases()
	if err != nil {
		t.Fatal(err)
	}
	a := testApp(t)
	for _, d := range a.defender.Directions {
		forms := p.Recommendations[d.ID]
		if forms.One == "" || forms.Many == "" {
			t.Errorf("%s: no phrase", d.ID)
		}
	}
	for _, level := range []string{"high", "medium", "low"} {
		if p.Motivation[level] == "" {
			t.Errorf("motivation %s: no phrase", level)
		}
	}
	for _, c := range []selfCase{selfFormed, selfSearching, selfImposed, selfIdle, selfUndecided} {
		if p.Self.MilitaryText[c.phrase] == "" || p.Self.CivilText[c.phrase] == "" || p.Self.Summary[c.phrase] == "" {
			t.Errorf("self %q: no phrase", c.phrase)
		}
	}
	for _, sense := range humanCongruent {
		if p.Image.Congruence[sense] == "" && !strings.Contains(sense, "сказать нельзя") {
			t.Errorf("congruence %q: no phrase", sense)
		}
	}
	for _, name := range humanValues {
		if strings.HasPrefix(name, "Закрепить") || strings.HasPrefix(name, "Студент") || strings.HasPrefix(name, "Критерий") {
			continue
		}
		if _, ok := p.Environment.Conditions[name]; !ok {
			t.Errorf("condition %q: no cases", name)
		}
	}
	var book map[string]any
	if err := json.Unmarshal(formbank.TestPhrases(), &book); err != nil {
		t.Fatal(err)
	}
	delete(book, "about")
	delete(book, "forbidden")
	var marks func(path string, node any)
	marks = func(path string, node any) {
		switch x := node.(type) {
		case map[string]any:
			for k, v := range x {
				marks(path+"/"+k, v)
			}
		case []any:
			for _, v := range x {
				marks(path, v)
			}
		case string:
			if strings.ContainsAny(x, ":;—–") {
				t.Errorf("%s: a colon or a dash in %q", path, x)
			}
		}
	}
	marks("", book)
	lower := strings.ToLower(string(formbank.TestPhrases()))
	_, phrases, _ := strings.Cut(lower, `"titles"`)
	phrases, _, _ = strings.Cut(phrases, `"forbidden"`)
	for _, word := range p.Forbidden {
		if strings.Contains(phrases, word) {
			t.Errorf("the phrase book has %q", word)
		}
	}
}

func TestNoTestConclusionWithoutData(t *testing.T) {
	if testApp(t).testConclusion(nil) != nil {
		t.Fatal("a conclusion without data")
	}
}

// The report of a finished session with the methods has the conclusion; an
// unfinished one has the methods only.
func TestReportHasTheTestConclusion(t *testing.T) {
	a := testApp(t)
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range db.Sessions {
		s := record.Session
		if !s.Completed || !s.Methods || s.Kind != "student" {
			continue
		}
		if _, ok := a.analyze(s, db.Sessions)["conclusion"].(*TestConclusion); !ok {
			t.Fatalf("%s: no conclusion in the report", s.Participant)
		}
		s.Completed = false
		result := a.analyze(s, db.Sessions)
		if _, ok := result["conclusion"]; ok {
			t.Fatal("a conclusion on an unfinished session")
		}
		if result["methods"] == nil {
			t.Fatal("no methods in the report")
		}
		return
	}
	t.Fatal("no finished demo session with the methods")
}

// The conclusions by the demo platoons are kept in testdata/test_conclusion,
// «взвод N.json»; with -update they are rewritten.
func TestPlatoonConclusionOfDemoPlatoons(t *testing.T) {
	a := testApp(t)
	if err := a.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	all, err := a.MethodsSummaryFor("", "")
	if err != nil {
		t.Fatal(err)
	}
	digits := regexp.MustCompile(`\d`)
	seen := 0
	for _, group := range all.Groups {
		summary, err := a.MethodsSummaryFor("", group)
		if err != nil {
			t.Fatal(err)
		}
		c := summary.Platoon.Conclusion
		if summary.Platoon.Students < 2 {
			if c != nil {
				t.Errorf("%s: a conclusion by one student", group)
			}
			continue
		}
		seen++
		if c == nil || c.Summary == "" || len(c.Recommendations) == 0 {
			t.Fatalf("%s: incomplete conclusion %+v", group, c)
		}
		if digits.MatchString(c.Summary) || strings.Contains(c.Summary, "один студент") {
			t.Errorf("%s: a number in %q", group, c.Summary)
		}
		if len(c.Work) != len(summary.Platoon.WorkGroups) {
			t.Errorf("%s: %d actions for %d groups", group, len(c.Work), len(summary.Platoon.WorkGroups))
		}
		got, err := json.MarshalIndent(c, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, '\n')
		path := filepath.Join("testdata", "test_conclusion", "взвод "+group+".json")
		if *update {
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs; run go test -run TestPlatoonConclusionOfDemoPlatoons -update\n%s", path, got)
		}
	}
	if seen == 0 {
		t.Fatal("no demo platoon of two students")
	}
}

// A quality stands out only by the gap from the student's own middle; an
// even profile has neither strong nor weak ones (decided 13 November 2026).
func TestStandOut(t *testing.T) {
	score := func(name string, v float64) testScoredItem { return testScoredItem{Name: name, Score: &v} }
	even := []testScoredItem{score("А", 3.67), score("Б", 3.33), score("В", 3.33), score("Г", 3), score("Д", 3)}
	if strong, weak, _, _ := standOut(even, 0.5, 3); len(strong)+len(weak) != 0 {
		t.Fatalf("an even profile stands out: %v %v", strong, weak)
	}
	items := []testScoredItem{score("А", 4.67), score("Б", 4.33), score("В", 3.67), score("Г", 3), score("Д", 3), score("Е", 2)}
	strong, weak, _, _ := standOut(items, 0.5, 3)
	if len(strong) != 2 || strong[0].Name != "А" || len(weak) != 1 || weak[0].Name != "Е" {
		t.Fatalf("strong %v, weak %v", strong, weak)
	}
}

func TestJoinPhrases(t *testing.T) {
	for _, c := range []struct {
		items []string
		want  string
	}{
		{[]string{"а"}, "а"},
		{[]string{"а", "б", "в"}, "а, б и в"},
		{[]string{"а, отделяя б", "в"}, "а, отделяя б, и в"},
		{[]string{"а", "б", "в, отделяя г"}, "а, б и в, отделяя г"},
	} {
		if got := joinPhrases(c.items); got != c.want {
			t.Errorf("%v: %q, want %q", c.items, got, c.want)
		}
	}
}
