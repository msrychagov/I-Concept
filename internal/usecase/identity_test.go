package usecase

import (
	"fmt"
	"strings"
	"testing"
)

// The profile of each block by the key, the ranges and the leading statuses:
// a tie names both; an unanswered question leaves the block without a profile.
func TestIdentityProfilesByTheKey(t *testing.T) {
	a := testApp(t)
	s := fixedTestSession(a, a.bank.Version)
	for n := 1; n <= 20; n++ {
		s.Answers[fmt.Sprintf("AZM%02d", n)] = 1 // all «б»
		s.Answers[fmt.Sprintf("AZC%02d", n)] = 3 // all «г»
	}
	profile := func(p IdentityProfile) string {
		parts := []string{}
		for _, status := range p.Statuses {
			score := "—"
			if status.Score != nil {
				score = fmt.Sprint(*status.Score)
			}
			parts = append(parts, fmt.Sprintf("%s %s/%d %s", status.Short, score, status.Max, status.Range))
		}
		return strings.Join(parts, "; ") + " → " + p.LeadingText
	}
	data := a.TestPortrait(s)
	if len(data.Identity) != 2 {
		t.Fatalf("%d blocks", len(data.Identity))
	}
	for n, want := range []string{
		"НП 5/25 ниже среднего; НВ 10/25 средняя; М 3/25 не выражен; СФ 7/25 средняя → навязанная и сформированная",
		"НП 4/25 ниже среднего; НВ 0/25 не выражен; М 9/25 ниже среднего; СФ 12/25 ярко выражен → сформированная",
	} {
		if got := profile(data.Identity[n]); got != want || data.Identity[n].Answered != 20 {
			t.Errorf("%s: %s", data.Identity[n].Name, got)
		}
	}
	if data.Identity[0].Name != "Военная специальность" || data.Identity[1].Name != "Гражданская профессия" {
		t.Fatalf("blocks: %s, %s", data.Identity[0].Name, data.Identity[1].Name)
	}

	delete(s.Answers, "AZC05")
	civil := a.TestPortrait(s).Identity[1]
	if civil.Answered != 19 || civil.LeadingText != "не определён" || len(civil.Leading) != 0 || civil.Statuses[0].Score != nil || civil.Statuses[0].Range != "" {
		t.Fatalf("incomplete block: %s", profile(civil))
	}

	if old := a.TestPortrait(fixedTestSession(a, "pilot-2.4")); old.Identity != nil {
		t.Fatalf("a session without the method has identity: %+v", old.Identity)
	}
}

// The method is asked with the flag «Методики» only, both blocks in full.
func TestIdentityIsAskedWithTheMethods(t *testing.T) {
	for _, methods := range []bool{true, false} {
		f := newTestPortraitFixture(t, methods)
		session := mustSnapshot(t, f.service.repo).Sessions[0].Session
		count := 0
		for _, item := range f.service.itemsFor(session) {
			if item.Module == "AZM" || item.Module == "AZC" {
				count++
			}
		}
		if want := map[bool]int{true: 40, false: 0}[methods]; count != want {
			t.Fatalf("methods %v: %d questions of the method", methods, count)
		}
	}
}
