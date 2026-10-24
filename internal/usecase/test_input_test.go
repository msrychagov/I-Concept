package usecase

import (
	"encoding/json"
	"hash/fnv"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/promptbank"
)

// fixedTestSession answers every question of the bank version by a hash of
// its code, so the session is the same on every run.
func fixedTestSession(a *Service, version string) model.Session {
	s := model.Session{ID: "fixed", Kind: "student", BankVersion: version, Methods: true, Activity: true, Answers: map[string]int{}, Notes: map[string]string{}}
	for _, item := range a.itemsFor(s) {
		h := fnv.New32a()
		h.Write([]byte(item.ID))
		n := h.Sum32()
		switch {
		case item.Type == "textarea":
			if n%5 == 0 {
				s.Answers[item.ID] = 0
				s.Notes[item.ID] = "Участвовал в мероприятии " + item.ID + "."
			}
		case len(item.Options) > 0:
			s.Answers[item.ID] = int(n % uint32(len(item.Options)))
		default:
			s.Answers[item.ID] = item.Min + int(n%uint32(item.Max-item.Min+1))
		}
	}
	return s
}

// The input of the instruction v1 must not change: the stored portraits are
// found by its fingerprint. A session of pilot-2.4 and one of the current bank
// with the same answers give the same input. With -update the file is rewritten.
func TestTestPayloadV1IsKept(t *testing.T) {
	a := testApp(t)
	path := filepath.Join("testdata", "test_input", "v1.json")
	for _, version := range []string{"pilot-2.4", a.bank.Version} {
		data := a.TestPortrait(fixedTestSession(a, version))
		if data == nil {
			t.Fatalf("%s: no portrait", version)
		}
		got := indentedJSON(a.testPayloadFor("v1", data))
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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
		if string(got) != string(want) {
			t.Fatalf("%s: the input of v1 changed", version)
		}
	}
}

// From v2 on the input has the image of the defender, the congruence, the
// identity, the signs of the environment and the catalogue, and the kinds of
// activity get the criteria of the first chapter; the other keys of v1 keep
// their values and their order.
func TestTestPayloadV2AddsTheImage(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	old, payload := a.testPayloadFor("v1", data), a.testPayloadFor("v2", data)
	added := []string{"согласованность_образов", "образ_защитника", "профессиональная_идентичность", "признаки_среды", "каталог_рекомендаций"}
	kept := slices.DeleteFunc(slices.Clone(payload.keys), func(k string) bool { return slices.Contains(added, k) })
	if !slices.Equal(kept, old.keys) {
		t.Fatalf("keys of v1 in v2: %v", payload.keys)
	}
	for _, key := range old.keys {
		if key == "виды_деятельности" {
			continue
		}
		if string(canonicalJSON(payload.values[key])) != string(canonicalJSON(old.values[key])) {
			t.Fatalf("%s differs in v2", key)
		}
	}
	want := []string{"методики", "образы_я", "согласованность_образов", "признаки", "образ_защитника", "профессиональная_идентичность", "условия_среды", "признаки_среды", "виды_деятельности", "рассказы_о_деятельности", "каталог_рекомендаций"}
	if !slices.Equal(payload.keys, want) {
		t.Fatalf("keys of v2: %v", payload.keys)
	}
	// Without the method there is no professional identity in the input.
	if old := a.testPayloadFor("v2", a.TestPortrait(fixedTestSession(a, "pilot-2.4"))); slices.Contains(old.keys, "профессиональная_идентичность") {
		t.Fatal("identity in the input of a session without the method")
	}

	var input struct {
		Groups []struct {
			Number    int     `json:"номер"`
			Name      string  `json:"группа"`
			Real      float64 `json:"я_реальное"`
			Situation string  `json:"ситуация"`
			Type      string  `json:"тип_рекомендации"`
			Kinds     []struct {
				Kind  string `json:"вид"`
				Level string `json:"уровень"`
			} `json:"виды_деятельности"`
			Conditions []struct {
				Name  string   `json:"показатель"`
				Score *float64 `json:"балл"`
			} `json:"условия_среды"`
			Note string `json:"пометка"`
		} `json:"образ_защитника"`
		Congruence struct {
			Text string `json:"оценка"`
		} `json:"согласованность_образов"`
		Identity []struct {
			Profession string `json:"профессия"`
			Statuses   []struct {
				Status string `json:"статус"`
				Score  string `json:"баллы"`
				Range  string `json:"выраженность"`
			} `json:"статусы"`
			Leading string `json:"ведущий_статус"`
		} `json:"профессиональная_идентичность"`
		Kinds []struct {
			Kind     string `json:"вид"`
			Criteria []struct {
				Name     string          `json:"критерий"`
				Coverage string          `json:"покрытие"`
				Data     json.RawMessage `json:"данные"`
			} `json:"критерии"`
		} `json:"виды_деятельности"`
		Signs []struct {
			Name string            `json:"признак"`
			Data []json.RawMessage `json:"данные"`
		} `json:"признаки_среды"`
		Catalogue []struct {
			Code   string `json:"код"`
			Name   string `json:"направление"`
			Type   string `json:"тип"`
			Groups []int  `json:"группы"`
		} `json:"каталог_рекомендаций"`
	}
	if err := json.Unmarshal(canonicalJSON(payload), &input); err != nil {
		t.Fatal(err)
	}
	if len(input.Groups) != 5 || input.Congruence.Text == "" || len(input.Catalogue) != 21 {
		t.Fatalf("groups %d, congruence %q, catalogue %d", len(input.Groups), input.Congruence.Text, len(input.Catalogue))
	}
	for n, g := range input.Groups {
		if g.Number != n+1 || g.Name == "" || g.Situation == "" || g.Type == "" || len(g.Kinds) == 0 || len(g.Conditions) == 0 || g.Conditions[0].Score == nil {
			t.Errorf("group %d: %+v", n+1, g)
		}
	}
	if len(input.Identity) != 2 || input.Identity[0].Profession != "Военная специальность" || len(input.Identity[0].Statuses) != 4 || input.Identity[0].Statuses[0].Range == "" || !strings.Contains(input.Identity[0].Statuses[0].Score, "(диапазон 0–25)") || input.Identity[0].Leading == "" {
		t.Errorf("identity: %+v", input.Identity)
	}
	if len(input.Kinds) != 4 || len(input.Signs) != 6 {
		t.Fatalf("kinds %d, signs %d", len(input.Kinds), len(input.Signs))
	}
	for _, k := range input.Kinds {
		if len(k.Criteria) < 3 {
			t.Errorf("%s: %d criteria", k.Kind, len(k.Criteria))
		}
		for _, c := range k.Criteria {
			// A criterion the test does not measure says so; the others list their
			// data or say that the student left them without answers.
			if (c.Coverage == "нет данных") != (string(c.Data) == `"нет данных"`) || c.Coverage != "нет данных" && string(c.Data) != `"нет ответов"` && c.Data[0] != '[' {
				t.Errorf("%s / %s: %s %s", k.Kind, c.Name, c.Coverage, c.Data)
			}
		}
	}
	for _, sign := range input.Signs {
		if len(sign.Data) == 0 {
			t.Errorf("%s has no data", sign.Name)
		}
	}
	if input.Groups[4].Note == "" || input.Groups[0].Note != "" {
		t.Errorf("notes: %q, %q", input.Groups[0].Note, input.Groups[4].Note)
	}
	if c := input.Catalogue[1]; c.Code != "R02" || c.Type != "ценностная" || !slices.Equal(c.Groups, []int{1, 2}) || c.Name == "" {
		t.Errorf("R02: %+v", c)
	}
}

// The task builds the input and checks the answer of the instruction it was
// registered with: an older program that knows only v1 sends the input of v1.
func TestTestTaskSendsTheInputOfItsVersion(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	first := slices.DeleteFunc(prompts, func(p promptbank.Prompt) bool { return p.Task == testPortraitTask && p.Version != "v1" })
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(first...)
	var user string
	f.service.UseModel(&fakeModel{name: "fake", reply: func(_ int, c Completion) (string, error) {
		user = c.User
		data, _ := json.Marshal(goodTestAnswer(c.User))
		return string(data), nil
	}})
	f.service.queuePortraits(Source{"session", f.session})
	if view := f.wait(); view.Status != portraitReady {
		t.Fatalf("view: %+v", view)
	}
	if strings.Contains(user, "\"образ_защитника\"") || !strings.Contains(user, "\"методики\"") {
		t.Fatalf("input of v1: %.300s", user)
	}
}

// The data of a criterion are the program's own: the level of the kinds, the
// levels of the methods, the means of the indicators, the answers of the
// questionnaire and the open answers that tell something.
func TestCriteriaDataComeFromTheSession(t *testing.T) {
	a := testApp(t)
	s := fixedTestSession(a, a.bank.Version)
	s.Notes["ACT_18"], s.Answers["ACT_18"] = "Разбираю темы по учебнику и конспекту.", 0
	s.Notes["ACT_20"], s.Answers["ACT_20"] = "Не участвовал.", 0
	data := a.TestPortrait(s)
	study := data.Criteria[0]
	if study.Kind != "study" || len(data.EnvironmentSigns) != 6 {
		t.Fatalf("criteria: %+v", data.Criteria)
	}
	kinds := map[string]string{}
	for _, point := range study.Criteria[0].Data {
		kinds[point.Code] = point.Kind
		if point.Code == "ACT_18" && point.Answer != "Разбираю темы по учебнику и конспекту." {
			t.Errorf("story: %+v", point)
		}
	}
	if !maps.Equal(kinds, map[string]string{"K02": "question", "K03": "question", "ACT2": "frequency", "Q12": "indicator", "Q22": "indicator", "ACT_18": "story"}) {
		t.Fatalf("data of the first criterion: %v", kinds)
	}
	for _, point := range study.Criteria[1].Data {
		if point.Code == "ACT_20" {
			t.Fatal("an answer that tells nothing is a datum")
		}
	}
	first := data.EnvironmentSigns[0].Data
	if len(first) < 4 || first[0].Kind != "activity" || first[0].Level != data.Kinds[0].Level || first[len(first)-1].Kind != "see" {
		t.Fatalf("data of the first sign: %+v", first)
	}
	for _, sign := range data.EnvironmentSigns {
		for _, point := range sign.Data {
			if point.Kind == "method" || point.Text == "" {
				t.Errorf("%s: %+v", sign.ID, point)
			}
		}
	}
}
