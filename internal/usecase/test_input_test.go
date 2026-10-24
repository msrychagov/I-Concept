package usecase

import (
	"encoding/json"
	"hash/fnv"
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

// From v2 on the input has the image of the defender, the congruence and the
// catalogue; the keys of v1 keep their values and their order.
func TestTestPayloadV2AddsTheImage(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	old, payload := a.testPayloadFor("v1", data), a.testPayloadFor("v2", data)
	added := []string{"согласованность_образов", "образ_защитника", "профессиональная_идентичность", "каталог_рекомендаций"}
	kept := slices.DeleteFunc(slices.Clone(payload.keys), func(k string) bool { return slices.Contains(added, k) })
	if !slices.Equal(kept, old.keys) {
		t.Fatalf("keys of v1 in v2: %v", payload.keys)
	}
	for _, key := range old.keys {
		if string(canonicalJSON(payload.values[key])) != string(canonicalJSON(old.values[key])) {
			t.Fatalf("%s differs in v2", key)
		}
	}
	want := []string{"методики", "образы_я", "согласованность_образов", "признаки", "образ_защитника", "профессиональная_идентичность", "условия_среды", "виды_деятельности", "рассказы_о_деятельности", "каталог_рекомендаций"}
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
	if input.Groups[4].Note == "" || input.Groups[0].Note != "" {
		t.Errorf("notes: %q, %q", input.Groups[0].Note, input.Groups[4].Note)
	}
	if c := input.Catalogue[1]; c.Code != "R02" || c.Type != "ценностная" || !slices.Equal(c.Groups, []int{1, 2}) || c.Name == "" {
		t.Errorf("R02: %+v", c)
	}
}

// The task builds the input of the instruction it was registered with.
func TestTestTaskSendsTheInputOfItsVersion(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	prompts, err := promptbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	newer := promptbank.Prompt{Task: testPortraitTask, Version: "v2", Instruction: "Инструкция.", Schema: json.RawMessage(`{"type":"object"}`)}
	f.service.portraits.tasks = nil
	f.service.RegisterPrompts(append(prompts, newer)...)
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
	if !strings.Contains(user, "\"образ_защитника\"") || !strings.Contains(user, "\"каталог_рекомендаций\"") {
		t.Fatalf("input of v2: %.300s", user)
	}
}
