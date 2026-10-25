package questionbank

import (
	"fmt"
	"slices"
	"testing"
)

// The tables of the image of the defender (Вики/Тестирование/Портрет по тесту —
// образ, влияние, рекомендации.md, approved 23 October 2026) refer only to what
// the bank has; every quality Q01–Q24 is in exactly one group; every group has
// a direction of each type a situation can call for.
func TestDefenderImageRefersToTheBank(t *testing.T) {
	image, err := Defender()
	if err != nil {
		t.Fatal(err)
	}
	bank, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	indicators, tags, items := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, ind := range bank.Indicators {
		indicators[ind.ID] = true
		for _, tag := range ind.Tags {
			tags[tag] = true
		}
	}
	for _, item := range bank.Items {
		items[item.ID] = true
	}
	methods, kinds := map[string]bool{}, map[string]bool{}
	for _, m := range bank.Methods {
		methods[m.ID] = true
	}
	for _, k := range bank.ActivityKinds {
		kinds[k.ID] = true
	}
	types := map[string]bool{}
	for _, kind := range image.Types {
		types[kind.ID] = true
	}

	if len(image.Groups) != 5 {
		t.Fatalf("%d groups, want 5", len(image.Groups))
	}
	groups := map[string]bool{}
	seen := map[string]string{}
	for n, g := range image.Groups {
		if g.ID != fmt.Sprintf("G%d", n+1) || g.Name == "" || g.Description == "" {
			t.Fatalf("group %d: %+v", n+1, g)
		}
		groups[g.ID] = true
		for _, q := range g.Qualities {
			if !indicators[q] || !items[q+"_R"] || !items[q+"_I"] {
				t.Errorf("%s: quality %s has no indicator or self-image questions", g.ID, q)
			}
			if other, twice := seen[q]; twice {
				t.Errorf("%s is in %s and %s", q, other, g.ID)
			}
			seen[q] = g.ID
		}
		for _, id := range append(slices.Clone(g.Indicators), g.Conditions...) {
			if !indicators[id] {
				t.Errorf("%s: unknown indicator %s", g.ID, id)
			}
		}
		for _, tag := range g.Signs {
			if !tags[tag] {
				t.Errorf("%s: unknown sign %s", g.ID, tag)
			}
		}
		for _, id := range g.Methods {
			if !methods[id] {
				t.Errorf("%s: unknown method %s", g.ID, id)
			}
		}
		if len(g.Kinds) == 0 || len(g.Conditions) == 0 {
			t.Errorf("%s: no kinds of activity or conditions", g.ID)
		}
		for _, k := range g.Kinds {
			if !kinds[k.ID] {
				t.Errorf("%s: unknown kind of activity %s", g.ID, k.ID)
			}
		}
	}
	for n := 1; n <= 24; n++ {
		if q := fmt.Sprintf("Q%02d", n); seen[q] == "" {
			t.Errorf("%s is in no group", q)
		}
	}

	// The first edition is R01–R21; the second adds R22–R24 from the article
	// of the supervisor of 2023 (Вики/Тестирование/План П2, П5 и климат по
	// Лутошкину.md, decision Д3).
	if len(image.Directions) != 24 {
		t.Fatalf("%d directions, want 24", len(image.Directions))
	}
	have := map[string]map[string]bool{}
	for n, d := range image.Directions {
		if d.ID != fmt.Sprintf("R%02d", n+1) || d.Name == "" || d.Source == "" || len(d.Groups) == 0 {
			t.Fatalf("direction %d: %+v", n+1, d)
		}
		if edition := map[bool]int{false: 0, true: 2}[n >= 21]; d.Edition != edition || n >= 21 && d.Source != "статья Митюкова 2023" {
			t.Errorf("%s: edition %d, source %q", d.ID, d.Edition, d.Source)
		}
		if !types[d.Type] {
			t.Errorf("%s: unknown type %s", d.ID, d.Type)
		}
		for _, g := range d.Groups {
			if !groups[g] {
				t.Errorf("%s: unknown group %s", d.ID, g)
			}
			if have[g] == nil {
				have[g] = map[string]bool{}
			}
			have[g][d.Type] = true
		}
	}
	// The psychological directions do not answer a situation: they fit any
	// group when a method of the emotional-volitional component is low (Д4).
	for _, g := range image.Groups {
		for _, kind := range []string{"support", "activity", "value", "reflexive"} {
			if !have[g.ID][kind] {
				t.Errorf("%s has no direction of type %s", g.ID, kind)
			}
		}
	}
}

// The criteria of the first chapter and the six signs of the environment
// (Вики/Тестирование/Оценка деятельности и среды.md, approved 24 October 2026)
// refer to methods, indicators and questions of the bank.
func TestCriteriaAndSignsReferToTheBank(t *testing.T) {
	image, err := Defender()
	if err != nil {
		t.Fatal(err)
	}
	bank, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{"kinds": true, "images": true, "defender": true, "signs": true, "stories": true}
	for _, m := range bank.Methods {
		known[m.ID] = true
	}
	for _, ind := range bank.Indicators {
		known[ind.ID] = true
	}
	for _, item := range bank.Items {
		if item.Module == "K" || item.Module == "F" {
			known[item.ID] = true
		}
	}
	check := func(where string, data []string) {
		for _, code := range data {
			if !known[code] {
				t.Errorf("%s: unknown code %s", where, code)
			}
		}
	}
	kinds := []string{}
	for _, k := range image.Criteria {
		kinds = append(kinds, k.Kind)
		for _, c := range k.Criteria {
			if c.Name == "" || !slices.Contains([]string{"есть", "частично", "по взводу", "нет"}, c.Coverage) || (c.Coverage == "нет") != (len(c.Data) == 0) {
				t.Errorf("%s: %+v", k.Kind, c)
			}
			check(k.Kind+": "+c.Name, c.Data)
		}
	}
	want := []string{}
	for _, k := range bank.ActivityKinds {
		want = append(want, k.ID)
	}
	if !slices.Equal(kinds, want) {
		t.Fatalf("kinds of the criteria: %v", kinds)
	}
	if len(image.Signs) != 6 {
		t.Fatalf("%d signs", len(image.Signs))
	}
	for n, sign := range image.Signs {
		if sign.ID != fmt.Sprintf("S%d", n+1) || sign.Name == "" || sign.Description == "" || len(sign.Data) == 0 {
			t.Errorf("sign %d: %+v", n+1, sign)
		}
		check(sign.ID, sign.Data)
	}
}
