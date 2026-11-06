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
			// The stories of the student may have numbers; the rest may not.
			if before, _, _ := strings.Cut(text, "Из ответов о деятельности"); digits.MatchString(before) {
				t.Errorf("%s: a number in %q", name, text)
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
