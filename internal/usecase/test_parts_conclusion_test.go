package usecase

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// partSets are the sets of submitted parts a teacher can meet before the test is whole.
var partSets = [][]string{{"self"}, {"will"}, {"vuc"}, {"self", "will"}, {"self", "vuc"}, {"will", "vuc"}}

// demoPartSessions are the finished demo sessions of pilot-2.7, by surname.
func demoPartSessions(t *testing.T) (*Service, map[string]model.Session) {
	t.Helper()
	a := testApp(t)
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]model.Session{}
	for _, record := range db.Sessions {
		s := record.Session
		if s.Completed && s.Methods && s.Kind == "student" && len(a.testParts(s)) > 0 {
			out[strings.Fields(s.Participant)[0]] = s
		}
	}
	if len(out) == 0 {
		t.Fatal("no finished demo session of pilot-2.7")
	}
	return a, out
}

// withParts is the session as if only the parts were submitted.
func withParts(s model.Session, parts []string) model.Session {
	out := clone(s)
	out.Completed, out.CompletedPartial, out.Parts = false, false, slices.Clone(parts)
	return out
}

// sentences are the sentences of a text; a sentence ends with a full stop
// outside the quotes of a story.
func sentences(text string) []string {
	out, start, quoted := []string{}, 0, false
	runes := []rune(text)
	for n, r := range runes {
		switch r {
		case '«':
			quoted = true
		case '»':
			quoted = false
		case '.', '!', '?':
			if !quoted && (n+1 == len(runes) || runes[n+1] == ' ') {
				out = append(out, strings.TrimSpace(string(runes[start:n+1])))
				start = n + 1
			}
		}
	}
	if rest := strings.TrimSpace(string(runes[start:])); rest != "" {
		out = append(out, rest)
	}
	return out
}

// Before all parts are submitted the teacher sees a part of the conclusion:
// every sentence is word for word a sentence of the conclusion on the whole
// test; there is no summary and no recommendation (decided on 27 November 2026).
func TestPartialConclusionIsAPartOfTheWhole(t *testing.T) {
	a, sessions := demoPartSessions(t)
	for _, name := range sortedKeys(sessions) {
		s := sessions[name]
		result := a.analyze(s, nil)
		whole, ok := result["conclusion"].(*TestConclusion)
		if !ok || whole.Partial || whole.Summary == "" {
			t.Fatalf("%s: no whole conclusion", name)
		}
		full := map[string]string{}
		for _, d := range whole.Details {
			full[d.ID] = d.Text
		}
		for _, parts := range partSets {
			result := a.analyze(withParts(s, parts), nil)
			c, ok := result["conclusion"].(*TestConclusion)
			if !ok || !c.Partial || c.Summary != "" || len(c.Recommendations) != 0 {
				t.Fatalf("%s %v: conclusion %+v", name, parts, c)
			}
			if len(c.Details) == 0 {
				t.Errorf("%s %v: nothing of the conclusion", name, parts)
			}
			for _, d := range c.Details {
				for _, sentence := range sentences(d.Text) {
					if !slices.Contains(sentences(full[d.ID]), sentence) {
						t.Errorf("%s %v, %s: %q is not in the whole conclusion", name, parts, d.ID, sentence)
					}
				}
			}
		}
	}
}

// Which part of the conclusion comes with which part of the test
// (Вики/Тестирование/План разбивки теста на три части.md): the self-image
// only with the first, the environment only with the third, the attitude to
// the Fatherland only with the first and the second.
func TestPartialConclusionByPart(t *testing.T) {
	a, sessions := demoPartSessions(t)
	p, err := loadedTestPhrases()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range sortedKeys(sessions) {
		s := sessions[name]
		ids := func(parts ...string) []string {
			c := a.analyze(withParts(s, parts), nil)["conclusion"].(*TestConclusion)
			out := []string{}
			for _, d := range c.Details {
				out = append(out, d.ID)
			}
			return out
		}
		if got := ids("self"); slices.Contains(got, "environment") || slices.Contains(got, "will") || !slices.Contains(got, "image") {
			t.Errorf("%s, first part: %v", name, got)
		}
		if got := ids("will"); slices.Contains(got, "image") || slices.Contains(got, "environment") || !slices.Contains(got, "will") {
			t.Errorf("%s, second part: %v", name, got)
		}
		if got := ids("vuc"); slices.Contains(got, "image") || !slices.Contains(got, "environment") {
			t.Errorf("%s, third part: %v", name, got)
		}
		// The kinds of activity read all parts: their sentences never come before.
		for _, parts := range partSets {
			c := a.analyze(withParts(s, parts), nil)["conclusion"].(*TestConclusion)
			for _, d := range c.Details {
				for _, text := range p.Environment.AllKinds {
					if text != "" && strings.Contains(d.Text, text) {
						t.Errorf("%s %v: the kinds of activity before all parts", name, parts)
					}
				}
			}
		}
	}
}

// A session in parts shows the methods only of the submitted parts: the
// answers of a part in progress are not counted.
func TestMethodsOfSubmittedPartsOnly(t *testing.T) {
	a, sessions := demoPartSessions(t)
	for _, name := range sortedKeys(sessions) {
		data, ok := a.analyze(withParts(sessions[name], []string{"vuc"}), nil)["methods"].(*TestPortraitData)
		if !ok {
			t.Fatalf("%s: no methods", name)
		}
		for _, m := range data.Methods {
			if m.ID != "DP_A" && m.ID != "DP_B" && m.ID != "DP_C" && m.Level != "" {
				t.Errorf("%s: %s has a level before its part is submitted", name, m.ID)
			}
		}
		return
	}
}

// The conclusions by parts of one demo student are kept in
// testdata/test_conclusion/части; with -update they are rewritten.
func TestPartialConclusionOfDemoStudent(t *testing.T) {
	a, sessions := demoPartSessions(t)
	const name = "Морозов"
	s, ok := sessions[name]
	if !ok {
		t.Fatalf("no demo student %s", name)
	}
	dir := filepath.Join("testdata", "test_conclusion", "части")
	for _, parts := range partSets {
		file := name + " — " + strings.Join(parts, ", ") + ".json"
		got, err := json.MarshalIndent(a.analyze(withParts(s, parts), nil)["conclusion"], "", " ")
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, '\n')
		path := filepath.Join(dir, file)
		if *update {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
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
			t.Errorf("%s differs; run go test -run TestPartialConclusionOfDemoStudent -update\n%s", path, got)
		}
	}
}
