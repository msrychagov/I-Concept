package usecase

import (
	"fmt"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func pointer(v float64) *float64 { return &v }

// oneGroupImage is an image of one group over the given qualities.
func oneGroupImage(qualities ...string) model.DefenderImage {
	return model.DefenderImage{
		Groups: []model.DefenderGroupDef{{ID: "G1", Name: "Группа", Description: "качества", Qualities: qualities}},
		Types:  []model.RecommendationType{{ID: "support", Name: "опора"}, {ID: "activity", Name: "деятельностная"}, {ID: "value", Name: "ценностная"}, {ID: "reflexive", Name: "рефлексивная"}},
	}
}

func noScore(id string) IndicatorScore { return IndicatorScore{ID: id} }

// The situations of Д1 on both sides of every threshold: the distances are
// rounded to tenths first and then to whole points, as in gapText.
func TestDefenderSituationsAtTheThresholds(t *testing.T) {
	for _, c := range []struct {
		name        string
		real, ideal []float64
		situation   string
		kind        string
	}{
		{"ideal half a point below", []float64{8, 9}, []float64{8, 8}, situationBelow, "рефлексивная"},
		{"ideal 0.4 below rounds to zero", []float64{8.4}, []float64{8}, situationAchieved, "опора"},
		{"distance 2.5 rounds to 3", []float64{7, 8}, []float64{7, 8}, situationNotAccepted, "ценностная"},
		{"distance 2.4 rounds to 2", []float64{7.6}, []float64{7.6}, situationAchieved, "опора"},
		{"gap 2.5 rounds to 3", []float64{7, 8}, []float64{10, 10}, situationFar, "деятельностная"},
		{"gap 2.4 rounds to 2", []float64{7.6}, []float64{10}, situationAchieved, "опора"},
		{"gap counted on tenths", []float64{5.7}, []float64{8.2}, situationFar, "деятельностная"},
		{"below is checked first", []float64{8, 8}, []float64{6, 6}, situationBelow, "рефлексивная"},
		{"not accepted before far", []float64{2, 2}, []float64{6, 6}, situationNotAccepted, "ценностная"},
	} {
		t.Run(c.name, func(t *testing.T) {
			data := &TestPortraitData{}
			qualities := []string{}
			for n := range c.real {
				id := fmt.Sprintf("Q%02d", n+1)
				qualities = append(qualities, id)
				data.Images = append(data.Images, ImagePair{ID: id, Real: pointer(c.real[n]), Ideal: pointer(c.ideal[n])})
			}
			groups := defenderGroups(oneGroupImage(qualities...), data, noScore)
			if len(groups) != 1 {
				t.Fatalf("%d groups", len(groups))
			}
			g := groups[0]
			if g.Situation != c.situation || g.TypeText != c.kind || g.SituationText != situationText[c.situation] {
				t.Fatalf("real %v, ideal %v, distance %v, gap %v: %s / %s", *g.Real, *g.Ideal, *g.Distance, *g.Gap, g.Situation, g.TypeText)
			}
		})
	}
}

// A group with no quality answered in both images gets no values, not zeros,
// and no situation; a quality answered in one image only is left out.
func TestDefenderGroupWithoutAnswers(t *testing.T) {
	data := &TestPortraitData{Images: []ImagePair{{ID: "Q01", Real: pointer(9)}, {ID: "Q02", Real: pointer(5), Ideal: pointer(9)}}}
	groups := defenderGroups(oneGroupImage("Q01"), data, noScore)
	if g := groups[0]; g.Real != nil || g.Ideal != nil || g.Gap != nil || g.Distance != nil || g.Situation != "" || g.Type != "" || g.SituationText != "не определена" || g.Answered != 0 || g.Total != 1 {
		t.Fatalf("group without answers: %+v", g)
	}
	groups = defenderGroups(oneGroupImage("Q01", "Q02"), data, noScore)
	if g := groups[0]; g.Real == nil || *g.Real != 5 || *g.Ideal != 9 || g.Answered != 1 || g.Situation != situationFar {
		t.Fatalf("group with one quality answered: %+v", g)
	}
	if groups := defenderGroups(model.DefenderImage{}, data, noScore); groups != nil {
		t.Fatalf("groups without the image: %+v", groups)
	}
}

func TestCongruenceByRogers(t *testing.T) {
	profile := func(real, ideal []float64) []ImagePair {
		pairs := []ImagePair{}
		for n := range real {
			pairs = append(pairs, ImagePair{ID: fmt.Sprintf("Q%02d", n+1), Real: pointer(real[n]), Ideal: pointer(ideal[n])})
		}
		return pairs
	}
	for _, c := range []struct {
		name        string
		real, ideal []float64
		value       *float64
		text        string
	}{
		{"strong", []float64{1, 2, 3, 4, 5, 6}, []float64{2, 4, 5, 4, 5, 9}, pointer(0.85), "профили во многом совпадают"},
		{"negative", []float64{5, 6, 7, 8, 9, 10}, []float64{10, 10, 9, 10, 9, 10}, pointer(-0.21), "профили почти не совпадают"},
		{"one line", []float64{1, 2, 3}, []float64{3, 5, 7}, pointer(1), "профили во многом совпадают"},
		{"ideal everywhere 10", []float64{4, 7, 9}, []float64{10, 10, 10}, nil, "Я-идеальное одинаково по всем качествам — согласованность не определяется."},
		{"real everywhere the same", []float64{6, 6, 6}, []float64{7, 9, 10}, nil, "Я-реальное одинаково по всем качествам — согласованность не определяется."},
		{"too few qualities", []float64{1, 2}, []float64{2, 4}, nil, "Качеств с ответами меньше трёх — согласованность не определяется."},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := congruence(profile(c.real, c.ideal))
			if got == nil || got.Text != c.text || got.Qualities != len(c.real) || (got.Value == nil) != (c.value == nil) || got.Value != nil && *got.Value != *c.value {
				t.Fatalf("congruence: %+v", got)
			}
		})
	}
	if got := congruence([]ImagePair{{ID: "Q01", Real: pointer(5)}}); got != nil {
		t.Fatalf("congruence without pairs: %+v", got)
	}
	for r, want := range map[float64]string{0.7: "профили во многом совпадают", 0.69: "профили совпадают частично", 0.3: "профили совпадают частично", 0.29: "профили почти не совпадают", -0.5: "профили почти не совпадают"} {
		if got := congruenceText(r); got != want {
			t.Errorf("%v: %s", r, got)
		}
	}
}

// The portrait of a session has the five groups of the image with what forms
// them; the group of physical qualities says the test has no data on fitness.
func TestTestPortraitComparesWithTheImage(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	if len(data.Defender) != 5 || data.Congruence == nil {
		t.Fatalf("defender %d, congruence %v", len(data.Defender), data.Congruence)
	}
	total := 0
	for _, g := range data.Defender {
		total += g.Total
		if g.Answered != g.Total || g.Situation == "" || g.Type == "" || g.Statements == nil || len(g.Kinds) == 0 || len(g.Conditions) == 0 {
			t.Errorf("%s: %+v", g.ID, g)
		}
		for _, c := range g.Conditions {
			if c.Name == "" || c.Score == nil {
				t.Errorf("%s: condition %+v", g.ID, c)
			}
		}
	}
	if total != 24 {
		t.Fatalf("qualities in the groups: %d", total)
	}
	g1, g4, g5 := data.Defender[0], data.Defender[3], data.Defender[4]
	if len(g1.Methods) != 1 || g1.Methods[0].ID != "VPM" || len(g1.Indicators) != 3 || len(g1.Signs) != 1 || g1.Signs[0] != "патриотизм" {
		t.Fatalf("G1: %+v", g1)
	}
	if len(g4.Methods) != 4 || g4.Kinds[0].Forms == "" {
		t.Fatalf("G4: %+v", g4)
	}
	if g5.Note == "" || len(g5.Methods) != 1 || g5.Methods[0].ID != "EMO" {
		t.Fatalf("G5: %+v", g5)
	}
}
