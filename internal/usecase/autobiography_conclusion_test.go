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

func conclusionOf(t *testing.T, values answerSet) AutobiographyConclusion {
	t.Helper()
	form := autobiography(t)
	answers := answersOf(t, form, values)
	return autobiographyConclusion(form, answers, autobiographyPotential(form, answers))
}

// The conclusions on the questionnaires of the model comparison are kept in
// testdata/autobiography_conclusion; with -update they are rewritten.
func TestAutobiographyConclusionOfComparison(t *testing.T) {
	dir := filepath.Join("testdata", "autobiography_conclusion")
	for _, name := range sortedKeys(comparisonAnswers) {
		t.Run(name, func(t *testing.T) {
			got, err := json.MarshalIndent(conclusionOf(t, comparisonAnswers[name]), "", " ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			path := filepath.Join(dir, name)
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
				t.Errorf("%s differs; run go test -run TestAutobiographyConclusionOfComparison -update\n%s", path, got)
			}
		})
	}
}

// Every text of every comparison conclusion is free of the words the
// supervisor struck out and of a negation of experience.
func TestAutobiographyConclusionWords(t *testing.T) {
	p, err := loadedPhrases()
	if err != nil {
		t.Fatal(err)
	}
	negation := regexp.MustCompile(`(?i)\b(не служил|не работал|не имеет|не указал|не выражен|потенциал)\b`)
	for _, name := range sortedKeys(comparisonAnswers) {
		c := conclusionOf(t, comparisonAnswers[name])
		texts := slices.Concat(c.Paragraphs, c.Attention, c.Training, c.Consider)
		for _, area := range c.Areas {
			texts = append(texts, area.Text)
		}
		for _, text := range texts {
			lower := strings.ToLower(text)
			for _, word := range p.Forbidden {
				if strings.Contains(lower, word) {
					t.Errorf("%s: %q in %q", name, word, text)
				}
			}
			if negation.MatchString(text) {
				t.Errorf("%s: a negation in %q", name, text)
			}
			if strings.Contains(text, "{") || strings.Contains(text, "  ") || strings.Contains(text, " ,") || strings.Contains(text, "..") {
				t.Errorf("%s: a broken phrase %q", name, text)
			}
		}
	}
}

// The phrase book: every sport and creative direction of the form is in it,
// no phrase holds a struck-out word, every placeholder is one the code knows.
func TestAutobiographyPhraseBook(t *testing.T) {
	p, err := parseConclusionPhrases(formbank.AutobiographyPhrases())
	if err != nil {
		t.Fatal(err)
	}
	form := autobiography(t)
	scale := newPotentialScale(form)
	for _, option := range form.Options(scale.fields["sport_kind"]) {
		kind, ok := p.Sport.Kinds[option.ID]
		if !ok {
			t.Errorf("sport %s (%s) is not in the phrase book", option.ID, option.Text)
			continue
		}
		if _, ok := p.Sport.Groups[kind.Group]; !ok {
			t.Errorf("sport %s: unknown group %s", option.ID, kind.Group)
		}
	}
	for _, option := range form.Options(scale.fields["direction"]) {
		if _, ok := p.Creative.Directions[option.ID]; !ok {
			t.Errorf("creative direction %s is not in the phrase book", option.ID)
		}
	}
	for _, option := range form.Options(scale.fields["service_branch"]) {
		if _, ok := p.Training.Branches[option.ID]; !ok && option.ID != "other" {
			t.Errorf("branch %s is not in the phrase book", option.ID)
		}
	}
	for _, option := range form.Options(scale.fields["hobbies"]) {
		if _, ok := p.Hobbies.Items[option.ID]; !ok {
			t.Errorf("hobby %s is not in the phrase book", option.ID)
		}
	}
	for _, option := range form.Options(scale.fields["patriot_what"]) {
		if _, ok := p.Patriotic.Forms[option.ID]; !ok && option.ID != "other" {
			t.Errorf("patriotic form %s is not in the phrase book", option.ID)
		}
	}
	for _, option := range form.Options(scale.fields["vol_dirs"]) {
		if _, ok := p.Volunteering.Directions[option.ID]; !ok && option.ID != "other" {
			t.Errorf("volunteering direction %s is not in the phrase book", option.ID)
		}
	}
	data := strings.ToLower(string(formbank.AutobiographyPhrases()))
	var book map[string]any
	_ = json.Unmarshal(formbank.AutobiographyPhrases(), &book)
	delete(book, "forbidden")
	plain, _ := json.Marshal(book)
	for _, word := range p.Forbidden {
		if strings.Contains(strings.ToLower(string(plain)), word) {
			t.Errorf("the phrase book holds the struck-out %q", word)
		}
	}
	known := []string{"skills", "other", "level", "area", "forms", "tasks", "qualities", "traits", "kind", "detail", "direction",
		"instr", "prep", "term", "exam", "result", "country", "lead", "list", "short", "directions", "purpose", "own",
		"Parent", "position", "unknown", "contact", "coach", "a", "year", "age", "branch", "details", "rank", "positions",
		"children", "Who", "was", "what", "whom", "speciality"}
	for _, key := range regexp.MustCompile(`\{([A-Za-z]+)\}`).FindAllStringSubmatch(data, -1) {
		if !slices.Contains(known, key[1]) && !slices.Contains(known, strings.ToUpper(key[1][:1])+key[1][1:]) {
			t.Errorf("unknown placeholder {%s}", key[1])
		}
	}
}

// Cases the comparison questionnaires do not have.
func TestAutobiographyConclusionCases(t *testing.T) {
	base := func() answerSet {
		values := answerSet{}
		for key, value := range comparisonAnswers["01 Минимальная.json"] {
			values[key] = value
		}
		return values
	}
	t.Run("nothing at all", func(t *testing.T) {
		values := base()
		values["hobbies"] = []string{}
		c := conclusionOf(t, values)
		if len(c.Paragraphs) != 1 || c.Paragraphs[0] != "Сведений о занятиях не указал." || len(c.Areas) != 0 {
			t.Fatalf("%+v", c)
		}
	})
	t.Run("cadet class at school is told once", func(t *testing.T) {
		values := base()
		values["education_list"] = []record{{"edu_type": "school", "edu_profile": "cadet", "edu_year": "2025", "edu_medal": "no"}}
		values["patriot_has"], values["patriot_what"], values["patriot_role"], values["patriot_term"], values["patriot_level"] = "yes", []string{"cadet_class", "yunarmia"}, "member", "y13", "city"
		c := conclusionOf(t, values)
		if !strings.Contains(c.Paragraphs[0], "Окончил школу с кадетским классом.") || strings.Contains(c.Paragraphs[0], "кадетском классе") {
			t.Fatalf("%q", c.Paragraphs[0])
		}
	})
	t.Run("a student raised by the mother whose father died", func(t *testing.T) {
		values := base()
		values["upbringing"] = "mother"
		values["rel_list"] = []record{
			{"kind": "father", "known": "yes", "alive": "no", "death_year": "2015"},
			{"kind": "mother", "known": "yes", "alive": "yes", "work": "works", "sphere": "meditsina", "position": "медсестра", "foreign": "no", "crim": "no"}}
		c := conclusionOf(t, values)
		if len(c.Attention) != 2 || !strings.Contains(c.Attention[1], "Отец умер в 2015 году, когда студенту было около 7 лет.") {
			t.Fatalf("%q", c.Attention)
		}
		if strings.Contains(c.Attention[0], "сведений об отце") || strings.Contains(c.Attention[0], "общается ли с отцом") {
			t.Fatalf("%q", c.Attention[0])
		}
	})
	t.Run("an exam written as a level and an unknown one", func(t *testing.T) {
		if cefr("ielts", "7.5") != "C1" || cefr("goethe", "B1") != "B1" || cefr("toefl", "100") != "C1" || cefr("hsk", "4") != "" {
			t.Fatal("cefr")
		}
	})
	t.Run("a sport not in the list", func(t *testing.T) {
		values := base()
		values["sport_has"] = "yes"
		values["sport_list"] = []record{{"sport_kind": "other", "sport_kind_other": "Городки", "sport_now": "now", "sport_years": "y13", "sport_rank": "none", "sport_level": "city"}}
		c := conclusionOf(t, values)
		if !strings.Contains(c.Paragraphs[0], "До трёх лет занимается видом спорта «Городки», выступал на соревнованиях городского уровня.") {
			t.Fatalf("%q", c.Paragraphs[0])
		}
		if len(c.Areas) != 1 || strings.Contains(c.Areas[0].Text, "развиты") {
			t.Fatalf("%+v", c.Areas)
		}
	})
}
