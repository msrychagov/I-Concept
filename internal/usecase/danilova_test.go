package usecase

import (
	"fmt"
	"testing"
)

// The rules of Danilova (Д1, Д2) on both sides of the thresholds, by component:
// the difference is rounded to tenths and then to whole points, as in gapText.
func TestDanilovaRulesByComponent(t *testing.T) {
	for _, c := range []struct {
		name               string
		real, norm, mirror []float64
		pressure, adequacy string
	}{
		{"norm 2.4 above", []float64{5}, []float64{7.4}, []float64{5}, "no", "same"},
		{"norm 2.5 above rounds to 3", []float64{5}, []float64{7.5}, []float64{5}, "yes", "same"},
		{"norm 3 above", []float64{4, 6}, []float64{8, 8}, []float64{5, 5}, "yes", "same"},
		{"norm below the real", []float64{8}, []float64{3}, []float64{8}, "no", "same"},
		{"mirror 2.5 above", []float64{5}, []float64{5}, []float64{7.5}, "no", "better"},
		{"mirror 2.4 above", []float64{5}, []float64{5}, []float64{7.4}, "no", "same"},
		{"mirror 2.5 below", []float64{7.5}, []float64{7.5}, []float64{5}, "no", "worse"},
		{"mirror 2.4 below", []float64{7.4}, []float64{7.4}, []float64{5}, "no", "same"},
		{"no norm", []float64{5}, nil, []float64{9}, "", "better"},
		{"no mirror", []float64{2}, []float64{9}, nil, "yes", ""},
		{"no real", nil, []float64{9}, []float64{9}, "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			values := map[string][]float64{"R": c.real, "I": c.real, "N": c.norm, "Z": c.mirror}
			pair := imagePair("CS", "ценностно-смысловой", "CS", values)
			if pair.Pressure != c.pressure || pair.Adequacy != c.adequacy || pair.PressureText != pressureTexts[c.pressure] || pair.AdequacyText != adequacyTexts[c.adequacy] {
				t.Fatalf("pressure %q (%q), adequacy %q (%q)", pair.Pressure, pair.PressureText, pair.Adequacy, pair.AdequacyText)
			}
		})
	}
	if pressureTexts["yes"] != "студент считает, что окружение ждёт от него заметно большего" || adequacyTexts["worse"] != "считает, что другие видят его хуже, чем он себя" {
		t.Fatal("the texts of Д1 and Д2 changed")
	}
}

// By group of the image the rules take the means over the qualities answered
// in the real self and in the image they need; without such qualities the
// fields stay empty, not «нет».
func TestDanilovaRulesByGroup(t *testing.T) {
	for _, c := range []struct {
		name               string
		pairs              []ImagePair
		pressure, adequacy string
	}{
		{"means of two qualities", []ImagePair{
			{Real: pointer(4), Ideal: pointer(9), Norm: pointer(7), Mirror: pointer(1)},
			{Real: pointer(6), Ideal: pointer(9), Norm: pointer(8), Mirror: pointer(4)},
		}, "yes", "worse"},
		{"below the thresholds", []ImagePair{
			{Real: pointer(5), Ideal: pointer(9), Norm: pointer(7), Mirror: pointer(7)},
			{Real: pointer(5), Ideal: pointer(9), Norm: pointer(7), Mirror: pointer(6)},
		}, "no", "same"},
		{"a quality without the real self is left out", []ImagePair{
			{Real: pointer(5), Ideal: pointer(9), Norm: pointer(6), Mirror: pointer(9)},
			{Norm: pointer(10), Mirror: pointer(0)},
		}, "no", "better"},
		{"no normative and no mirror self", []ImagePair{
			{Real: pointer(5), Ideal: pointer(9)},
		}, "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			data := &TestPortraitData{}
			qualities := []string{}
			for n, pair := range c.pairs {
				pair.ID = fmt.Sprintf("Q%02d", n+1)
				qualities = append(qualities, pair.ID)
				data.Images = append(data.Images, pair)
			}
			g := defenderGroups(oneGroupImage(qualities...), data, noScore)[0]
			if g.Pressure != c.pressure || g.Adequacy != c.adequacy || g.PressureText != pressureTexts[c.pressure] || g.AdequacyText != adequacyTexts[c.adequacy] {
				t.Fatalf("pressure %q, adequacy %q, norm %v, mirror %v", g.Pressure, g.Adequacy, g.Norm, g.Mirror)
			}
			if (c.pressure == "") != (g.Norm == nil) || (c.adequacy == "") != (g.Mirror == nil) {
				t.Fatalf("means: norm %v, mirror %v", g.Norm, g.Mirror)
			}
		})
	}
}

// A counted session has the rules by component and by group.
func TestTestPortraitHasTheRulesOfDanilova(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	if len(data.Components) != 3 {
		t.Fatalf("components: %d", len(data.Components))
	}
	for _, pair := range data.Components {
		if pair.Pressure == "" || pair.Adequacy == "" {
			t.Errorf("%s: %+v", pair.ID, pair)
		}
	}
	for _, g := range data.Defender {
		if g.Pressure == "" || g.Adequacy == "" || g.Norm == nil || g.Mirror == nil {
			t.Errorf("%s: pressure %q, adequacy %q", g.ID, g.Pressure, g.Adequacy)
		}
	}
}
