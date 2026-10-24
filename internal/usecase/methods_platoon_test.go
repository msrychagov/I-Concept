package usecase

import (
	"fmt"
	"strings"
	"testing"
)

// platoonStudent is a fixed session with the self-images, the conditions of
// the environment and the method of identity answered as given.
func platoonStudent(a *Service, name string, real, ideal, condition, military int) studentPortrait {
	s := fixedTestSession(a, a.bank.Version)
	s.ID, s.Participant, s.Group = "session-"+name, name, "101"
	for _, item := range a.itemsFor(s) {
		switch {
		case item.Module == "A" && strings.HasSuffix(item.ID, "_R"):
			s.Answers[item.ID] = real
		case item.Module == "A" && strings.HasSuffix(item.ID, "_I"):
			s.Answers[item.ID] = ideal
		case item.Module == "D":
			s.Answers[item.ID] = condition
			if item.Reverse {
				s.Answers[item.ID] = item.Min + item.Max - condition
			}
		case item.Module == "AZM" && military >= 0:
			s.Answers[item.ID] = military
		}
	}
	if military < 0 {
		delete(s.Answers, "AZM01")
	}
	return studentPortrait{methodsStudent(s), a.TestPortrait(s)}
}

func countsText(list []PlatoonCount) string {
	parts := []string{}
	for _, c := range list {
		names := []string{}
		for _, s := range c.Students {
			names = append(names, s.Name)
		}
		parts = append(parts, fmt.Sprintf("%s: %s", c.ID, strings.Join(names, ",")))
	}
	return strings.Join(parts, "; ")
}

func TestPlatoonCountsTheSituationsConditionsAndStatuses(t *testing.T) {
	a := testApp(t)
	students := []studentPortrait{
		platoonStudent(a, "A", 9, 10, 5, 1), // reached; conditions 5; military all «б»: НВ and СФ share the range
		platoonStudent(a, "B", 5, 10, 1, 3), // far; conditions 1; all «г»: СФ
		platoonStudent(a, "C", 2, 5, 2, -1), // not accepted; conditions 2; military not complete
		platoonStudent(a, "D", 8, 6, 3, 3),  // ideal below the real; conditions 3
	}
	data := a.platoonPortrait("101", students)
	if data.Students != 4 || len(data.Defender) != 5 || len(data.Conditions) != 12 || len(data.Identity) != 2 {
		t.Fatalf("students %d, groups %d, conditions %d, identity %d", data.Students, len(data.Defender), len(data.Conditions), len(data.Identity))
	}
	for _, g := range data.Defender {
		if got := countsText(g.Situations); got != "achieved: A; far: B; not_accepted: C; below: D; : " {
			t.Errorf("%s: %s", g.ID, got)
		}
		if g.Real == nil || *g.Real != 6 || *g.Ideal != 7.8 || len(g.Kinds) == 0 || len(g.Conditions) == 0 {
			t.Errorf("%s: real %v, ideal %v", g.ID, g.Real, g.Ideal)
		}
	}
	for _, c := range data.Conditions {
		names := []string{}
		for _, s := range c.Low {
			names = append(names, s.Name)
		}
		if c.Rated != 4 || c.Mean == nil || *c.Mean != 2.75 || strings.Join(names, ",") != "B,C" || !strings.HasPrefix(c.ID, "D") {
			t.Errorf("%s: rated %d, mean %v, low %v", c.ID, c.Rated, c.Mean, names)
		}
	}
	if got := countsText(data.Identity[0].Leading); got != "NP: ; NV: A; M: ; SF: A,B,D; : C" {
		t.Errorf("military: %s", got)
	}
	if data.Identity[0].Leading[4].Text != "не определён" || data.Identity[0].Name != "Военная специальность" {
		t.Errorf("military: %+v", data.Identity[0])
	}
	// Every self-image the same: the congruence is not counted for anyone.
	if got := countsText(data.Congruence); got != "high: ; partial: ; low: ; none: A,B,C,D" {
		t.Errorf("congruence: %s", got)
	}
	for _, rows := range [][]MethodsSummaryRow{data.Methods, data.Kinds} {
		for _, row := range rows {
			total := 0
			for _, level := range row.Levels {
				total += len(level.Students)
			}
			if total != 4 {
				t.Errorf("%s: %d students on the levels", row.ID, total)
			}
		}
	}
	if len(data.Methods) != len(a.bank.Methods) || len(data.Kinds) != len(a.bank.ActivityKinds) {
		t.Fatalf("methods %d, kinds %d", len(data.Methods), len(data.Kinds))
	}
}

// The platoon is counted over the latest finished session of every student of
// the teacher; another teacher or another platoon gets nobody.
func TestPlatoonPortraitIsTheTeachersOwn(t *testing.T) {
	f := newTestPortraitFixture(t, true)
	session := mustSnapshot(t, f.service.repo).Sessions[0].Session
	data, err := f.service.PlatoonPortraitFor(f.teacherID, "101")
	if err != nil || data.Students != 1 || data.Group != "101" {
		t.Fatalf("platoon: %+v %v", data.Students, err)
	}
	found := false
	for _, c := range data.Defender[0].Situations {
		for _, s := range c.Students {
			found = found || s.SessionID == session.ID
		}
	}
	if !found {
		t.Fatal("the student is in no situation")
	}
	for _, other := range []struct{ teacher, group string }{{"someone-else", "101"}, {f.teacherID, "999"}} {
		if data, _ := f.service.PlatoonPortraitFor(other.teacher, other.group); data.Students != 0 {
			t.Fatalf("%+v: %d students", other, data.Students)
		}
	}
}
