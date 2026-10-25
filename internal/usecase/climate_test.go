package usecase

import (
	"fmt"
	"slices"
	"testing"
)

// climateSession is a fixed session with the pairs of the climate rated as
// given, from +3 to −3; a missing rating leaves the pair without an answer.
func climateSession(a *Service, name string, scores ...int) studentPortrait {
	s := fixedTestSession(a, a.bank.Version)
	s.ID, s.Participant, s.Group = "session-"+name, name, "101"
	for n := 1; n <= 14; n++ {
		id := fmt.Sprintf("CL%02d", n)
		delete(s.Answers, id)
		if n <= len(scores) {
			s.Answers[id] = 3 - scores[n-1]
		}
	}
	return studentPortrait{methodsStudent(s), a.TestPortrait(s)}
}

func sides(a *Service, numbers ...int) []string {
	list := []string{}
	for _, n := range numbers {
		list = append(list, a.bank.Climate.Pairs[n-1].Side)
	}
	return list
}

// The student's sum and his sides: those of his highest and of his lowest
// rating, all of them; none when every rating is the same or a pair is left.
func TestStudentClimate(t *testing.T) {
	a := testApp(t)
	c := climateSession(a, "А", 3, 3, 2, 2, 2, 1, 1, 1, 0, 0, -1, -1, -2, -2).data.Climate
	if !c.Complete() || *c.Sum != 9 || c.Min != -42 || c.Max != 42 || c.Total != 14 || c.Answered != 14 || c.Equal {
		t.Fatalf("climate: %+v", c)
	}
	if !slices.Equal(c.Strongest, sides(a, 1, 2)) || !slices.Equal(c.Weakest, sides(a, 13, 14)) {
		t.Fatalf("strongest %v, weakest %v", c.Strongest, c.Weakest)
	}
	if c.Pairs[0].Answer != "Всегда так, как в первом" || c.Pairs[13].Opposite == "" || *c.Pairs[13].Score != -2 {
		t.Fatalf("pairs: %+v", c.Pairs)
	}
	equal := climateSession(a, "Б", 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3).data.Climate
	if !equal.Equal || *equal.Sum != 42 || len(equal.Strongest) != 0 || len(equal.Weakest) != 0 {
		t.Fatalf("equal: %+v", equal)
	}
	part := climateSession(a, "В", 3, 2, 1).data.Climate
	if part.Complete() || part.Answered != 3 || len(part.Strongest) != 0 {
		t.Fatalf("part: %+v", part)
	}
	old := a.TestPortrait(fixedTestSession(a, "pilot-2.5"))
	if old.Climate != nil {
		t.Fatalf("a session of pilot-2.5 has a climate: %+v", old.Climate)
	}
}

// threeRated is a platoon of three students who rated the pairs the same.
func threeRated(a *Service, scores ...int) []studentPortrait {
	return []studentPortrait{climateSession(a, "А", scores...), climateSession(a, "Б", scores...), climateSession(a, "В", scores...)}
}

// The platoon: the means over the students who answered all the pairs, three
// of them at least; three sides each way with the ties at the border, a side
// in one list only.
func TestPlatoonClimate(t *testing.T) {
	a := testApp(t)
	students := []studentPortrait{
		climateSession(a, "А", 3, 3, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "Б", 3, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "В", 3, 1, 2, 2, 1, 1, 1, 0, 0, 0, -1, -1, -2, -3),
		climateSession(a, "Г", 3, 2, 1), // not all pairs: left out
	}
	c := platoonClimate(students)
	if c == nil || len(c.Rated) != 3 || *c.Sum != 4.7 || c.Min != -42 || c.Equal {
		t.Fatalf("climate: %+v", c)
	}
	if *c.Pairs[1].Mean != 1.67 || *c.Pairs[13].Mean != -3 {
		t.Fatalf("means: %v %v", *c.Pairs[1].Mean, *c.Pairs[13].Mean)
	}
	// Means 3, 1.67, 2, 2, 1, …: the third is 2, so pairs 3 and 4 share the border.
	if !slices.Equal(c.Strongest, sides(a, 1, 3, 4)) || !slices.Equal(c.Weakest, sides(a, 14, 13, 11, 12)) {
		t.Fatalf("strongest %v, weakest %v", c.Strongest, c.Weakest)
	}
	two := platoonClimate(threeRated(a, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0))
	if !slices.Equal(two.Weakest, sides(a, 14)) || len(two.Strongest) != 13 {
		t.Fatalf("two values: strongest %d, weakest %v", len(two.Strongest), two.Weakest)
	}
	same := platoonClimate(threeRated(a, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1))
	if !same.Equal || len(same.Strongest)+len(same.Weakest) != 0 {
		t.Fatalf("same: %+v", same)
	}
	// Two students with all the pairs are not enough (decided on 25 October 2026).
	if platoonClimate(students[1:]) != nil || platoonClimate(students[3:]) != nil {
		t.Fatal("a climate of fewer than three students")
	}
	data := a.platoonPortrait("101", students)
	if data.Climate == nil || len(data.Climate.Rated) != 3 || data.Climate.Rated[0].Name != "А" {
		t.Fatalf("platoon portrait: %+v", data.Climate)
	}
}

// The demo platoons 102 and 104 rated their climate differently, every
// student all the pairs; in 104 the others see one student better and one
// worse than he sees himself, and some are under outer pressure.
func TestDemoPlatoonsRateTheirClimate(t *testing.T) {
	a := testApp(t)
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	byPlatoon := map[string][]studentPortrait{}
	adequacy, pressure := map[string]int{}, 0
	for _, record := range db.Sessions {
		s := record.Session
		if !s.Methods || (s.Group != "102" && s.Group != "104") {
			continue
		}
		data := a.TestPortrait(s)
		if !data.Climate.Complete() || data.Climate.Equal {
			t.Fatalf("%s: climate %+v", s.Participant, data.Climate)
		}
		byPlatoon[s.Group] = append(byPlatoon[s.Group], studentPortrait{methodsStudent(s), data})
		if s.Group == "104" {
			for _, c := range data.Components {
				adequacy[c.Adequacy]++
				if c.Pressure == "yes" {
					pressure++
				}
			}
		}
	}
	// Platoon 102 has two students who finished: too few for its climate,
	// though they rate it higher than platoon 104.
	mean := func(list []studentPortrait) float64 {
		total := 0
		for _, p := range list {
			total += *p.data.Climate.Sum
		}
		return float64(total) / float64(len(list))
	}
	low := platoonClimate(byPlatoon["104"])
	if len(byPlatoon["102"]) != 2 || platoonClimate(byPlatoon["102"]) != nil || low == nil || len(low.Rated) != len(demoPlatoon104) || mean(byPlatoon["102"]) <= mean(byPlatoon["104"]) {
		t.Fatalf("climates: 102 %d students, 104 %+v", len(byPlatoon["102"]), low)
	}
	if adequacy["better"] == 0 || adequacy["worse"] == 0 || adequacy["same"] == 0 || pressure == 0 {
		t.Fatalf("adequacy %v, pressure %d", adequacy, pressure)
	}
}

// The student gets both statements of a pair and its seven options.
func TestStudentGetsBothStatementsOfAPair(t *testing.T) {
	a := testApp(t)
	s := fixedTestSession(a, a.bank.Version)
	found := 0
	for _, item := range a.publicItems(s) {
		if item["module"] != "CL" {
			continue
		}
		found++
		if item["type"] != "pair" || item["text"] == "" || item["opposite"] == nil || item["opposite"] == "" || len(item["options"].([]map[string]string)) != 7 {
			t.Fatalf("pair: %v", item)
		}
	}
	if found != 14 {
		t.Fatalf("%d pairs", found)
	}
}
