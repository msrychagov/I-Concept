package usecase

import (
	"fmt"
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
		a.demoClimateAnswers(&s, profile.seed)
		db.Sessions = append(db.Sessions, model.SessionRecord{TokenHash: hash(token()), Session: s})
	}
}

// demoClimate are the ratings of the climate of a demo platoon (+3…−3) by
// pair of the method of Lutoshkin, around which its students answer: platoon
// 102 rates its climate higher; in platoon 104 the criticism, the common work,
// the groups within and the unity in hard times are rated against.
var demoClimate = map[string][]int{
	"102": {3, 2, 2, 2, 1, 2, 2, 2, 1, 0, 0, 1, 2, 3},
	"104": {1, 1, -1, 0, 0, -2, 1, -1, 0, 1, 0, -2, 0, 2},
}

// demoClimateAnswers rates the pairs of the climate around the ratings of the
// student's platoon, a point up or down now and then.
func (a *Service) demoClimateAnswers(s *model.Session, seed uint64) {
	base, method := demoClimate[s.Group], a.bank.Climate
	if method == nil || base == nil {
		return
	}
	r := rand.New(rand.NewPCG(seed, seed+7))
	for n, pair := range method.Pairs {
		item, ok := a.item(pair.Item)
		if !ok || n >= len(base) {
			continue
		}
		score := max(-3, min(3, base[n]+[]int{-1, 0, 0, 1}[r.IntN(4)]))
		for k, o := range item.Options {
			if int(o.Score) == score {
				s.Answers[pair.Item] = k
			}
		}
	}
}

// demoPlatoonStudent is an invented student of platoon 104: the real and the
// ideal self by group of the image (G1–G5), how the others see him in his
// opinion against the real self (mirror), how often the best option of a
// method is chosen, whether the emotional stability is low, the leading status
// of identity in each block (a scale of the key) and what he wrote. What the
// others expect is his ideal self.
type demoPlatoonStudent struct {
	name        string
	real, ideal [5]int
	mirror      int
	varied      bool
	strength    float64
	lowEmotion  bool
	military    string
	civil       string
	stories     map[string]string
}

// demoPlatoon104 covers the four situations of the groups, an ideal self of 10
// in every quality and low emotional stability (Вики/Тестирование/План портрета
// test-v2.md, step 8). The students and their answers are invented.
var demoPlatoon104 = []demoPlatoonStudent{
	{name: "Морозов Кирилл Андреевич", real: [5]int{9, 9, 9, 9, 9}, ideal: [5]int{10, 10, 10, 10, 10}, mirror: -1, strength: 0.85, military: "SF", civil: "SF", stories: map[string]string{
		"ACT_15": "Был на летних сборах, командовал отделением на полевом выходе.",
		"ACT_18": "Составляю план подготовки к каждому занятию и разбираю ошибки после контрольных.",
		"ACT_23": "Участвовал в подготовке военно-патриотической игры для школьников.",
		"ACT_31": "Выступал за ВУЦ на соревнованиях по военно-прикладному многоборью.",
		"ACT_38": "Помогал товарищам по взводу готовиться к зачёту по тактической подготовке.",
	}},
	{name: "Лебедев Глеб Сергеевич", real: [5]int{6, 5, 6, 6, 8}, ideal: [5]int{10, 10, 10, 10, 9}, mirror: 3, strength: 0.5, military: "M", civil: "SF", stories: map[string]string{
		"ACT_14": "Сдал нормативы по физической подготовке на «хорошо».",
		"ACT_24": "Участвовал в церемонии возложения цветов в День защитника Отечества.",
	}},
	{name: "Егоров Тимур Олегович", real: [5]int{5, 5, 6, 5, 5}, ideal: [5]int{6, 6, 7, 6, 6}, mirror: -3, strength: 0.4, military: "NP", civil: "M", stories: map[string]string{
		"ACT_30": "Был на экскурсии в музее военной истории вместе со взводом.",
	}},
	{name: "Павлов Роман Игоревич", real: [5]int{8, 7, 7, 8, 7}, ideal: [5]int{7, 9, 9, 7, 9}, mirror: -1, strength: 0.6, military: "NV", civil: "SF", stories: map[string]string{
		"ACT_20": "Распорядок соблюдаю, опозданий на занятия не было.",
		"ACT_34": "Выступал на концерте ВУЦ ко Дню Победы.",
	}},
	{name: "Козлов Даниил Витальевич", real: [5]int{7, 6, 8, 6, 5}, ideal: [5]int{10, 10, 10, 10, 10}, mirror: -1, varied: true, strength: 0.55, lowEmotion: true, military: "M", civil: "NP", stories: map[string]string{
		"ACT_29": "Был на встрече с ветеранами боевых действий.",
		"ACT_37": "Получил благодарность за помощь в организации дня открытых дверей ВУЦ.",
	}},
}

// addDemoPlatoon adds platoon 104 of invented students who have all finished
// the test with the methods, enough for the conclusion by the platoon.
func (a *Service) addDemoPlatoon(db *model.Database, full model.Profile, created string, now time.Time, teacherID string) {
	platoonID := uuid()
	db.Platoons = append(db.Platoons, model.Platoon{ID: platoonID, Name: "104", VUC: "Демонстрационный ВУЦ"})
	p := clone(full)
	p.Questions = a.assignmentQuestions(p, true, true, true)
	as := model.Assignment{ID: uuid(), Title: "Методики уровней Я-концепции", Platoon: "104", Profile: p, Activity: true, Motivation: true, Methods: true, Active: true, Created: created, OwnerID: teacherID}
	db.Assignments = append(db.Assignments, as)
	groups := map[string]int{}
	for n, g := range a.defender.Groups {
		for _, q := range g.Qualities {
			groups[q] = n
		}
	}
	for n, student := range demoPlatoon104 {
		salt := token()
		account := model.Account{VUC: "Демонстрационный ВУЦ", PlatoonID: platoonID, ID: uuid(), Role: "student", Name: student.name, Platoon: "104", Email: fmt.Sprintf("student%d@example.test", 10+n), PasswordHash: passwordHash("StudentDemo2026!", salt), Salt: salt, Created: created, OwnerID: teacherID}
		db.Students = append(db.Students, account)
		seed := uint64(101 + n)
		s := model.Session{ID: uuid(), StudentID: account.ID, AssignmentID: as.ID, Kind: "student", Participant: account.Name, Group: "104", Wave: as.Title, Profile: clone(p), BankVersion: a.bank.Version, Created: created, Updated: now.Add(-time.Duration(n+1) * time.Hour).Format(time.RFC3339), Seed: uint32(seed), Consent: true, ConsentAt: created, Activity: true, Motivation: true, Methods: true, Completed: true, Answers: map[string]int{}, Notes: map[string]string{}, Plan: []model.PlanItem{}, TeacherID: teacherID}
		items := a.itemsFor(s)
		demoProfileAnswers(items, &s, seed, student.strength, student.stories)
		for k, item := range items {
			kind := item.ID[strings.LastIndex(item.ID, "_")+1:]
			switch {
			case item.Module == "A":
				g := groups[item.Indicator]
				real := student.real[g]
				if student.varied {
					real = max(0, min(10, real+k%3-1))
				}
				s.Answers[item.ID] = map[string]int{"R": real, "I": student.ideal[g], "Z": max(0, min(10, real+student.mirror)), "N": student.ideal[g]}[kind]
			case item.Module == "U" && student.lowEmotion:
				worst := 0
				for k, o := range item.Options {
					if o.Score > item.Options[worst].Score {
						worst = k
					}
				}
				s.Answers[item.ID] = worst
			case item.Module == "AZM" || item.Module == "AZC":
				target := map[string]string{"AZM": student.military, "AZC": student.civil}[item.Module]
				// Seven answers of ten go to the leading status, the others to the next one.
				if k%10 >= 7 {
					target = map[string]string{"SF": "M", "M": "NP", "NP": "NV", "NV": "SF"}[target]
				}
				for v, o := range item.Options {
					if o.Scale == target {
						s.Answers[item.ID] = v
					}
				}
			}
		}
		a.demoClimateAnswers(&s, seed)
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
