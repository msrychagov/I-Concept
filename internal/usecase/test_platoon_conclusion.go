package usecase

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
)

// The conclusion by a platoon that the program writes itself (decided
// 6 November 2026, Вики/Тестирование/План вывода программой.md, В2): it puts
// together the conclusions on the students of the platoon — what most of them
// have, what most often needs attention, what is most often advised — and
// gives every group for separate work (platoon_groups.go) an action. Counts are
// said in words, without numbers; one student is never named in the text.

// PlatoonConclusion is the conclusion by a platoon.
type PlatoonConclusion struct {
	Summary         string             `json:"summary"`
	Recommendations []ConclusionAdvice `json:"recommendations"`
	// Work is the action for every group for separate work, by its label.
	Work []ConclusionWork `json:"work"`
}

// ConclusionWork is the action for one group for separate work.
type ConclusionWork struct {
	Label string `json:"label"`
	Code  string `json:"code"`
	Text  string `json:"text"`
}

// platoonConclusion is nil with fewer finished students than the minimum.
func (a *Service) platoonConclusion(portraits []studentPortrait, groups []PlatoonWorkGroup) *PlatoonConclusion {
	p, err := loadedTestPhrases()
	if err != nil {
		return nil
	}
	facts := []testFacts{}
	for _, portrait := range portraits {
		if _, f, err := a.testConclusionOf(portrait.data); err == nil {
			facts = append(facts, f)
		}
	}
	n := len(facts)
	if n < p.Platoon.Minimum {
		return nil
	}
	most := func(k int) bool { return 2*k > n }
	part := func(k int) bool { return k >= 2 }
	count := func(test func(testFacts) bool) int {
		k := 0
		for _, f := range facts {
			if test(f) {
				k++
			}
		}
		return k
	}
	sentences := []string{}

	levels := map[string]int{}
	for _, f := range facts {
		if f.motivation != "" {
			levels[f.motivation]++
		}
	}
	motivation := ""
	for _, level := range []string{"high", "medium", "low"} {
		if most(levels[level]) {
			motivation = p.Platoon.Motivation[level]
		}
	}
	if motivation == "" && len(levels) > 1 {
		motivation = p.Platoon.Motivation["mixed"]
	}
	if motivation != "" {
		sentences = append(sentences, motivation)
	}

	if g, k := mostOften(facts, func(f testFacts) []int { return f.support }); k > 0 {
		words := map[string]string{"group": p.Platoon.Groups[strconv.Itoa(g)]}
		switch {
		case most(k):
			sentences = append(sentences, fill(p.Platoon.Strength, words))
		case part(k):
			sentences = append(sentences, fill(p.Platoon.StrengthPart, words))
		}
	}

	tense := count(func(f testFacts) bool { return f.tense })
	switch {
	case most(tense):
		sentences = append(sentences, p.Platoon.TenseMost)
	case part(tense):
		sentences = append(sentences, p.Platoon.Tense)
	}
	if g, k := mostOften(facts, func(f testFacts) []int {
		if f.weakest == 0 {
			return nil
		}
		return []int{f.weakest}
	}); part(k) && !(g == 5 && part(tense)) {
		sentences = append(sentences, fill(p.Platoon.Attention, map[string]string{"group": p.Platoon.Groups[strconv.Itoa(g)]}))
	}

	unsure := count(func(f testFacts) bool { return f.unsure })
	switch {
	case most(unsure):
		sentences = append(sentences, p.Platoon.SpecialityMost)
	case part(unsure):
		sentences = append(sentences, p.Platoon.Speciality)
	}

	lacking := map[string]int{}
	order := []string{}
	for _, f := range facts {
		for _, name := range uniqueStrings(f.lacking) {
			if lacking[name] == 0 {
				order = append(order, name)
			}
			lacking[name]++
		}
	}
	slices.SortStableFunc(order, func(x, y string) int { return cmp.Compare(lacking[y], lacking[x]) })
	if len(order) > 0 && part(lacking[order[0]]) {
		c := p.Environment.Conditions[order[0]]
		sentences = append(sentences, fill(p.Platoon.Lacking, map[string]string{"condition": cmp.Or(c.Genitive, strings.ToLower(order[0]))}))
	}

	codes := map[string]int{}
	for _, f := range facts {
		for _, code := range f.advice {
			codes[code]++
		}
	}
	ranked := []string{}
	for code := range codes {
		ranked = append(ranked, code)
	}
	slices.SortFunc(ranked, func(x, y string) int { return cmp.Or(cmp.Compare(codes[y], codes[x]), cmp.Compare(x, y)) })
	if len(ranked) > 0 && part(codes[ranked[0]]) {
		ranked = slices.DeleteFunc(ranked, func(code string) bool { return codes[code] < 2 })
	}
	c := &PlatoonConclusion{Summary: strings.Join(sentences, " "), Recommendations: []ConclusionAdvice{}, Work: []ConclusionWork{}}
	for _, code := range ranked {
		if len(c.Recommendations) == p.Platoon.MaxAdvice {
			break
		}
		if text := p.Recommendations[code].Many; text != "" {
			c.Recommendations = append(c.Recommendations, ConclusionAdvice{Code: code, Text: text})
		}
	}
	for _, g := range groups {
		for _, code := range g.Codes {
			// One student in the group gets the phrase about «him».
			text := p.Recommendations[code].Many
			if len(g.Students) == 1 {
				text = p.Recommendations[code].One
			}
			if text != "" {
				c.Work = append(c.Work, ConclusionWork{Label: g.Label, Code: code, Text: text})
				break
			}
		}
	}
	return c
}

// mostOften is the group that the most students have in their list, the
// lowest number on a tie, with how many have it.
func mostOften(facts []testFacts, of func(testFacts) []int) (int, int) {
	counts := map[int]int{}
	for _, f := range facts {
		for _, g := range uniqueInts(of(f)) {
			counts[g]++
		}
	}
	best, most := 0, 0
	for g := 1; g <= 5; g++ {
		if counts[g] > most {
			best, most = g, counts[g]
		}
	}
	return best, most
}

func uniqueInts(xs []int) []int {
	out := []int{}
	for _, x := range xs {
		if !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}
