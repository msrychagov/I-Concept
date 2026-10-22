package usecase

import (
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// methodSession answers every question of the methods with the option the
// choose function returns for its id.
func methodSession(a *Service, choose func(id string) int) model.Session {
	p := a.emptyProfile()
	p.Questions = clone(a.bank.Items)
	s := model.Session{Kind: "student", Profile: p, BankVersion: a.bank.Version, Methods: true, Answers: map[string]int{}, Notes: map[string]string{}}
	for _, item := range a.itemsFor(s) {
		if methodModule(item.Module) || item.Module == "V" {
			s.Answers[item.ID] = choose(item.ID)
		}
	}
	return s
}

func levelsOf(t *testing.T, a *Service, s model.Session) map[string]MethodLevel {
	t.Helper()
	portrait := a.TestPortrait(s)
	if portrait == nil {
		t.Fatal("no portrait for a session with methods")
	}
	result := map[string]MethodLevel{}
	for _, m := range portrait.Methods {
		result[m.ID] = m
	}
	return result
}

func TestMethodLevelsFollowTheKeys(t *testing.T) {
	a := testApp(t)
	// All first options: «а» in the will test (question 5 is scored the other
	// way round), «Да» in emotional stability, «Верно» in Zverkov — Eidman.
	first := levelsOf(t, a, methodSession(a, func(string) int { return 0 }))
	second := levelsOf(t, a, methodSession(a, func(string) int { return 1 }))
	third := levelsOf(t, a, methodSession(a, func(id string) int {
		if strings.HasPrefix(id, "VPM") || strings.HasPrefix(id, "X") || strings.HasPrefix(id, "K") {
			return 2
		}
		return 0
	}))
	for _, c := range []struct {
		row   MethodLevel
		score int
		level string
	}{
		{first["WILL"], 34, "high"}, {second["WILL"], 18, "low"}, {third["WILL"], 2, "low"},
		{first["EMO"], 20, "medium"}, {second["EMO"], 8, "high"},
		{first["ZE_V"], 12, "medium"}, {first["ZE_N"], 9, "high"}, {first["ZE_S"], 6, "medium"},
		{second["ZE_V"], 12, "medium"}, {second["ZE_N"], 7, "medium"}, {second["ZE_S"], 7, "high"},
		{first["VPM"], 32, "low"}, {third["VPM"], 36, "medium"},
	} {
		if c.row.Score == nil || *c.row.Score != c.score || c.row.Level != c.level {
			t.Errorf("%s: %v %s, want %d %s", c.row.ID, c.row.Score, c.row.Level, c.score, c.level)
		}
	}
	for _, id := range []string{"DP_A", "DP_B", "DP_C"} {
		if first[id].Level != "low" || second[id].Level != "medium" || third[id].Level != "high" {
			t.Errorf("%s: %s %s %s", id, first[id].Level, second[id].Level, third[id].Level)
		}
	}
}

func TestQuestionnaireLevelIsTheMeanRank(t *testing.T) {
	a := testApp(t)
	answers := map[string]int{"K01": 0, "K02": 0, "K03": 0, "K04": 2, "K05": 2, "K06": 2, "K07": 1, "K08": 1, "K09": 1}
	s := methodSession(a, func(id string) int {
		if v, ok := answers[id]; ok {
			return v
		}
		return 2
	})
	if level := levelsOf(t, a, s)["DP_A"].Level; level != "medium" {
		t.Fatalf("mean rank 2: %s", level)
	}
	kinds := map[string]KindLevel{}
	for _, k := range a.TestPortrait(s).Kinds {
		kinds[k.ID] = k
	}
	// Study: K02 low, K03 low, K04–K06 high → (1+1+3+3+3)/5 = 2.2 → medium.
	if kinds["study"].Level != "medium" || kinds["outreach"].Level != "high" || len(kinds) != 4 {
		t.Fatalf("kinds: %+v", kinds)
	}
}

func TestMethodWithoutAllAnswersHasNoLevel(t *testing.T) {
	a := testApp(t)
	s := methodSession(a, func(string) int { return 0 })
	delete(s.Answers, "W02")
	delete(s.Answers, "K01")
	levels := levelsOf(t, a, s)
	for _, id := range []string{"ZE_V", "ZE_N", "DP_A"} {
		if levels[id].Level != "" || levels[id].Score != nil || levels[id].LevelText != "не определён" {
			t.Errorf("%s: %+v", id, levels[id])
		}
	}
	if levels["ZE_S"].Level == "" || levels["WILL"].Level == "" {
		t.Error("a complete scale lost its level")
	}
}

func TestRealAndIdealSelfByTheNorms(t *testing.T) {
	a := testApp(t)
	s := methodSession(a, func(string) int { return 0 })
	for kind, v := range map[string]int{"R": 5, "I": 8, "Z": 6, "N": 9} {
		s.Answers["Q01_"+kind] = v
	}
	for kind, v := range map[string]int{"R": 8, "I": 7} {
		s.Answers["Q02_"+kind] = v
	}
	images := map[string]ImagePair{}
	for _, pair := range a.TestPortrait(s).Images {
		images[pair.ID] = pair
	}
	if p := images["Q01"]; p.GapText != "выраженное расхождение" || p.SelfText != "средняя самооценка" || *p.Gap != 3 {
		t.Fatalf("Q01: %+v", p)
	}
	if p := images["Q02"]; p.GapText != "Я-идеальное ниже Я-реального" || p.SelfText != "высокая самооценка" {
		t.Fatalf("Q02: %+v", p)
	}
}

func TestSessionsWithoutMethodsHaveNoPortrait(t *testing.T) {
	a := testApp(t)
	s := methodSession(a, func(string) int { return 0 })
	s.Methods = false
	if a.TestPortrait(s) != nil {
		t.Fatal("portrait without methods")
	}
	for _, item := range a.itemsFor(s) {
		if methodModule(item.Module) {
			t.Fatalf("%s asked without the methods", item.ID)
		}
	}
}

func TestMethodsSummaryCountsTheLatestSessionOfEveryStudent(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	db := mustSnapshot(t, f.service.repo)
	session := db.Sessions[0].Session
	summary, err := f.service.MethodsSummaryFor(f.teacherID, "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Sessions != 1 || len(summary.Rows) != len(f.service.bank.Methods)+len(f.service.bank.ActivityKinds) || !slices.Equal(summary.Groups, []string{"101"}) {
		t.Fatalf("summary: %+v", summary)
	}
	levels := map[string]string{}
	for _, m := range f.service.TestPortrait(session).Methods {
		levels[m.ID] = m.Level
	}
	for _, row := range summary.Rows {
		if row.Kind != "method" {
			continue
		}
		for _, g := range row.Levels {
			if len(g.Students) == 1 && (g.Level != levels[row.ID] || g.Students[0].SessionID != session.ID) {
				t.Fatalf("%s: %+v", row.ID, g)
			}
		}
	}
	if other, err := f.service.MethodsSummaryFor("someone-else", ""); err != nil || other.Sessions != 0 {
		t.Fatalf("another teacher: %+v %v", other, err)
	}
	if empty, _ := f.service.MethodsSummaryFor(f.teacherID, "999"); empty.Sessions != 0 {
		t.Fatalf("other platoon: %+v", empty)
	}
}
