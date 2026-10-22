package usecase

import (
	"math/rand/v2"
	"strings"
	"time"
	"vucdiagnostic/internal/domain/model"
)

// addDemoMethods assigns the methods of the first chapter to platoon 102: two
// students have finished with different plausible profiles, the third has not
// started, so the test can be taken by hand (student6@example.test).
func (a *Service) addDemoMethods(db *model.Database, full model.Profile, created string, now time.Time, teacherID string) {
	p := clone(full)
	p.Questions = a.assignmentQuestions(p, true, true, true)
	as := model.Assignment{ID: uuid(), Title: "Методики уровней Я-концепции", Platoon: "102", Profile: p, Activity: true, Motivation: true, Methods: true, Active: true, Created: created, OwnerID: teacherID}
	db.Assignments = append(db.Assignments, as)
	profiles := []struct {
		seed     uint64
		strength float64
		stories  map[string]string
	}{
		{11, 0.8, map[string]string{
			"ACT_01": "Выступал с докладом на студенческой конференции ВУЦ весной, тема — связь в операциях Великой Отечественной войны.",
			"ACT_15": "Был на полевом выходе в октябре: ориентирование и действия в составе отделения.",
			"ACT_23": "Участвовал в акции памяти и в уборке воинского мемориала.",
			"ACT_31": "Играл за взвод в мини-футбол на спартакиаде ВУЦ, заняли второе место.",
		}},
		{23, 0.45, map[string]string{
			"ACT_15": "Был на учебных сборах летом, отвечал за связь в отделении.",
			"ACT_27": "Один раз сдавал кровь на дне донора в вузе.",
		}},
	}
	seen := 0
	for _, student := range db.Students {
		if student.Role != "student" || student.Platoon != "102" || seen >= len(profiles) {
			continue
		}
		profile := profiles[seen]
		seen++
		s := model.Session{ID: uuid(), StudentID: student.ID, AssignmentID: as.ID, Kind: "student", Participant: student.Name, Group: student.Platoon, Wave: as.Title, Profile: clone(p), BankVersion: a.bank.Version, Created: created, Updated: now.Add(-time.Duration(seen) * time.Hour).Format(time.RFC3339), Seed: uint32(profile.seed), Consent: true, ConsentAt: created, Activity: true, Motivation: true, Methods: true, Completed: true, Answers: map[string]int{}, Notes: map[string]string{}, Plan: []model.PlanItem{}, TeacherID: teacherID}
		demoProfileAnswers(a.itemsFor(s), &s, profile.seed, profile.strength, profile.stories)
		db.Sessions = append(db.Sessions, model.SessionRecord{TokenHash: hash(token()), Session: s})
	}
}

// demoProfileAnswers answers like a coherent student, not by cycling values:
// every quality has its own level, the self-image follows it, and strength is
// how often the best option of a method is chosen.
func demoProfileAnswers(items []model.Item, s *model.Session, seed uint64, strength float64, stories map[string]string) {
	r := rand.New(rand.NewPCG(seed, seed))
	levels := map[string]int{}
	level := func(indicator string) int {
		if _, ok := levels[indicator]; !ok {
			base := 3
			if r.Float64() < strength {
				base = 4 + r.IntN(2)
			} else if r.Float64() < 0.4 {
				base = 2
			}
			levels[indicator] = base
		}
		return levels[indicator]
	}
	for _, item := range items {
		switch {
		case item.Type == "textarea":
			s.Answers[item.ID] = 0
			if text, ok := stories[item.ID]; ok {
				s.Notes[item.ID] = text
			} else {
				s.Notes[item.ID] = "Не участвовал."
			}
		case item.Module == "A":
			real := 2*level(item.Indicator) - r.IntN(2)
			switch item.ID[strings.LastIndex(item.ID, "_")+1:] {
			case "R":
				s.Answers[item.ID] = real
			case "I":
				s.Answers[item.ID] = min(10, real+1+r.IntN(3))
			case "Z":
				s.Answers[item.ID] = max(0, real-r.IntN(2))
			default:
				s.Answers[item.ID] = 9
			}
		case item.Type == "psychological":
			usable := []int{}
			for k, o := range item.Options {
				if o.ID != "NA" && o.ID != "SKIP" {
					usable = append(usable, k)
				}
			}
			if len(usable) > 0 {
				s.Answers[item.ID] = usable[min(len(usable)-1, r.IntN(2))]
			}
		case len(item.Options) > 0:
			// The best option by score, or by level for the activity questionnaire;
			// in emotional stability fewer points are better.
			best := 0
			for k, o := range item.Options {
				better := o.Score > item.Options[best].Score
				if item.Module == "U" {
					better = o.Score < item.Options[best].Score
				}
				if better || o.Level == "high" {
					best = k
				}
			}
			if r.Float64() < strength {
				s.Answers[item.ID] = best
			} else {
				s.Answers[item.ID] = r.IntN(len(item.Options))
			}
		case item.Type == "frequency":
			s.Answers[item.ID] = 1 + r.IntN(3)
		default:
			v := max(item.Min, min(item.Max, level(item.Indicator)+[]int{0, 0, 0, -1, 1}[r.IntN(5)]))
			if item.Reverse {
				v = item.Min + item.Max - v
			}
			s.Answers[item.ID] = v
		}
	}
}
