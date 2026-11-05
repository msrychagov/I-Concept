package usecase

import (
	"cmp"
	"slices"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// The groups of students of a platoon who need separate work (test-platoon/v4,
// Вики/Тестирование/План конкретных рекомендаций и Квена.md, decisions Д1–Д5,
// approved by the user on 27 October 2026, after the remark of the supervisor:
// «нужна конкретика, с каким студентом или группой студентов»). The program
// puts the students together by their results; the model writes what to do
// with each group and never gets the names; the screen shows them.

// PlatoonWorkGroup is a group of students for separate work: its label, the
// students, the bases in plain words, what to look at by the next testing,
// the directions of the catalogue that fit it and whether the model writes
// the action for it (the first workGroupModel groups).
type PlatoonWorkGroup struct {
	Label    string           `json:"label"`
	Students []MethodsStudent `json:"students"`
	Bases    []string         `json:"bases"`
	// Plain are the bases in plain words (test-platoon/v5, Инструкция по
	// взводу v5.md): the input of v5 and the screen.
	Plain []string `json:"plain"`
	Look  []string `json:"look"`
	Codes []string `json:"codes"`
	Model bool     `json:"model"`
	// groups are the groups of the image the bases are about: their kinds of
	// activity and conditions are what the group is linked to in the
	// environment; lowest are the conditions of them its students rated lowest.
	groups []string
	lowest []string
}

// workGroupModel is how many groups get the action of the model (decision Д4).
const workGroupModel = 5

// workGroupLabels are the labels of the groups: letters that are not taken
// for a digit or another letter.
var workGroupLabels = strings.Fields("А Б В Г Д Е Ж И К Л М Н П Р С Т У Ф Х Ц Ч Ш Э Ю Я")

// workCandidate is one basis of a rule over the students it holds.
type workCandidate struct {
	rule     int
	seq      int
	students []MethodsStudent
	basis    string
	plain    humanBasis
	look     string
	groups   []string
	codes    []string
}

// The situations of a group of the image a basis is about, in the order of
// the table of Д1, and the type of direction each calls for.
var workSituations = []struct{ situation, text, kind string }{
	{situationNotAccepted, "образ не принят как свой", "value"},
	{situationBelow, "планка ниже, чем оценка себя сейчас", "reflexive"},
	{situationFar, "образ принят, до него далеко", "activity"},
}

// The rules of Д1 after the situations, whose rules are their places in
// workSituations.
const (
	ruleMethod   = 3
	ruleIdentity = 4
	rulePressure = 5
)

// workIdentity are the leading statuses of the military speciality that call
// for separate work, in the order of the method.
var workIdentity = []string{"NP", "NV", "M"}

// directionsOf are the codes of the catalogue of the second edition for a
// group of the image of the given types.
func (a *Service) directionsOf(groups []string, types ...string) []string {
	codes := []string{}
	for _, d := range a.directions(2) {
		if !slices.Contains(types, d.Type) {
			continue
		}
		for _, g := range groups {
			if slices.Contains(d.Groups, g) && !slices.Contains(codes, d.ID) {
				codes = append(codes, d.ID)
			}
		}
	}
	return codes
}

// workGroups puts the students of a platoon together by the rules of Д1 and
// joins the groups of the same students (Д3).
func (a *Service) workGroups(portraits []studentPortrait) []PlatoonWorkGroup {
	total := len(portraits)
	candidates := []workCandidate{}
	add := func(c workCandidate) {
		if len(c.students) > 0 {
			c.seq = len(candidates)
			candidates = append(candidates, c)
		}
	}
	short := func(name string) string {
		if g, ok := plainGroups[name]; ok {
			return g.Short
		}
		return lowerFirst(name)
	}
	for rule, s := range workSituations {
		for _, def := range a.defender.Groups {
			students := []MethodsStudent{}
			for _, p := range portraits {
				for _, g := range p.data.Defender {
					if g.ID == def.ID && g.Situation == s.situation {
						students = append(students, p.student)
					}
				}
			}
			// «До образа далеко» is separate work only for fewer than half of
			// the platoon; for more it is the work with the platoon.
			if s.situation == situationFar && 2*len(students) >= total {
				continue
			}
			add(workCandidate{rule: rule, students: students, basis: short(def.Name) + ": " + s.text,
				plain: humanBasis{qualities(def.Name), forCount(len(students), humanWork[s.situation])},
				look:  "как оценивает себя в качествах «" + short(def.Name) + "»", groups: []string{def.ID}, codes: a.directionsOf([]string{def.ID}, s.kind)})
		}
	}
	methods, order := map[string]MethodLevel{}, []string{}
	for _, p := range portraits {
		for _, m := range p.data.Methods {
			if _, seen := methods[m.ID]; !seen {
				methods[m.ID], order = m, append(order, m.ID)
			}
		}
	}
	for _, id := range order {
		students := []MethodsStudent{}
		for _, p := range portraits {
			for _, m := range p.data.Methods {
				if m.ID == id && m.Level == "low" {
					students = append(students, p.student)
				}
			}
		}
		groups := []string{}
		for _, def := range a.defender.Groups {
			if slices.Contains(def.Methods, id) {
				groups = append(groups, def.ID)
			}
		}
		name := methods[id].Name
		add(workCandidate{rule: ruleMethod, students: students, basis: lowerFirst(name) + " — на низком уровне",
			plain: humanBasis{"", cmpOr(humanLowScales[name], lowerFirst(name)+" — пока слабо")},
			look:  name, groups: groups, codes: a.directionsOf(groups, "psychological", "activity")})
	}
	if method := a.bank.Identity; method != nil {
		names := map[string]string{}
		for _, scale := range method.Scales {
			names[scale.ID] = scale.Name
		}
		for _, id := range workIdentity {
			students := []MethodsStudent{}
			for _, p := range portraits {
				for _, profile := range p.data.Identity {
					if profile.ID == "military" && slices.Contains(profile.Leading, id) {
						students = append(students, p.student)
					}
				}
			}
			add(workCandidate{rule: ruleIdentity, students: students, basis: "по военной специальности — " + names[id] + ": " + statusMeanings[names[id]],
				plain: humanBasis{"", forCount(len(students), humanWorkIdentity[id])},
				look:  "выбор военной специальности", groups: []string{"G2"}, codes: a.directionsOf([]string{"G2"}, "value")})
		}
	}
	for _, def := range a.defender.Groups {
		students := []MethodsStudent{}
		for _, p := range portraits {
			for _, g := range p.data.Defender {
				if g.ID == def.ID && g.Pressure == "yes" {
					students = append(students, p.student)
				}
			}
		}
		basis := short(def.Name) + ": по мнению самих студентов, окружение ждёт от них заметно большего, чем они видят в себе"
		if len(students) == 1 {
			basis = short(def.Name) + ": по мнению студента, окружение ждёт от него заметно большего, чем он видит в себе"
		}
		add(workCandidate{rule: rulePressure, students: students, basis: basis,
			plain: humanBasis{qualities(def.Name), forCount(len(students), humanWorkPressure)},
			look:  "ждут ли от него большего в качествах «" + short(def.Name) + "»", groups: []string{def.ID}, codes: a.directionsOf([]string{def.ID}, "reflexive")})
	}

	slices.SortStableFunc(candidates, func(x, y workCandidate) int {
		return cmp.Or(cmp.Compare(x.rule, y.rule), cmp.Compare(len(y.students), len(x.students)), cmp.Compare(x.seq, y.seq))
	})
	groups, at, plains := []PlatoonWorkGroup{}, map[string]int{}, [][]humanBasis{}
	for _, c := range candidates {
		ids := []string{}
		for _, s := range c.students {
			ids = append(ids, s.SessionID)
		}
		key := strings.Join(ids, "/")
		n, seen := at[key]
		if !seen {
			n = len(groups)
			at[key] = n
			groups = append(groups, PlatoonWorkGroup{Students: c.students, Bases: []string{}, Plain: []string{}, Look: []string{}, Codes: []string{}})
			plains = append(plains, nil)
		}
		g := &groups[n]
		g.Bases = append(g.Bases, c.basis)
		plains[n] = append(plains[n], c.plain)
		g.Look = addLook(g.Look, c.look)
		for _, id := range c.groups {
			if !slices.Contains(g.groups, id) {
				g.groups = append(g.groups, id)
			}
		}
		for _, code := range c.codes {
			if !slices.Contains(g.Codes, code) {
				g.Codes = append(g.Codes, code)
			}
		}
	}
	defs := map[string]model.DefenderGroupDef{}
	for _, g := range a.defender.Groups {
		defs[g.ID] = g
	}
	for n := range groups {
		groups[n].Plain = joinedBases(plains[n], len(a.defender.Groups))
		conditions := []string{}
		for _, id := range groups[n].groups {
			for _, c := range defs[id].Conditions {
				if !slices.Contains(conditions, c) {
					conditions = append(conditions, c)
				}
			}
		}
		groups[n].lowest = lowestConditions(portraits, groups[n].Students, conditions)
		groups[n].Codes = a.catalogueOrder(groups[n].Codes)
		if n < len(workGroupLabels) {
			groups[n].Label = workGroupLabels[n]
		}
		groups[n].Model = n < workGroupModel
	}
	return groups
}

// catalogueOrder puts codes in the order of the catalogue.
func (a *Service) catalogueOrder(codes []string) []string {
	ordered := []string{}
	for _, d := range a.directions(2) {
		if slices.Contains(codes, d.ID) {
			ordered = append(ordered, d.ID)
		}
	}
	return ordered
}

// groupCount is how many students a group has, in words relative to the
// platoon, as rule 1 of the instruction wants.
func groupCount(count, total int) string {
	if count == 1 {
		return "один студент"
	}
	return relativeCount(count, total)
}

// workGroupList is the input of the groups that get the action of the model:
// no names, the bases, what they are linked to in the environment and the
// directions that fit.
func (a *Service) workGroupList(data PlatoonPortraitData) []any {
	defs := map[string]model.DefenderGroupDef{}
	for _, g := range a.defender.Groups {
		defs[g.ID] = g
	}
	kindNames := map[string]string{}
	for _, k := range a.bank.ActivityKinds {
		kindNames[k.ID] = k.Name
	}
	indicatorNames := map[string]string{}
	for _, ind := range a.bank.Indicators {
		indicatorNames[ind.ID] = ind.Name
	}
	directions := map[string]string{}
	for _, d := range a.directions(2) {
		directions[d.ID] = d.Name
	}
	list := []any{}
	for _, g := range data.WorkGroups {
		if !g.Model {
			continue
		}
		o := newObject()
		o.set("метка", g.Label)
		o.set("студентов", groupCount(len(g.Students), data.Students))
		o.set("основания", anyList(g.Bases))
		linked := newObject()
		kinds, seen := []any{}, map[string]bool{}
		for _, id := range g.groups {
			for _, link := range defs[id].Kinds {
				if key := link.ID + "/" + link.Forms; !seen[key] {
					seen[key] = true
					k := newObject()
					k.set("вид", kindNames[link.ID])
					k.put("формы", link.Forms)
					kinds = append(kinds, k)
				}
			}
		}
		linked.set("виды_деятельности", kinds)
		if len(g.lowest) > 0 {
			names := []any{}
			for _, id := range g.lowest {
				names = append(names, indicatorNames[id])
			}
			linked.set("условия_ниже_всего", names)
		}
		o.set("связано_в_среде", linked)
		codes := []any{}
		for _, code := range g.Codes {
			d := newObject()
			d.set("код", code)
			d.set("направление", directions[code])
			codes = append(codes, d)
		}
		o.set("подходящие_направления", codes)
		list = append(list, o)
	}
	return list
}

// workLowest is how many conditions the input of a group names as rated lowest.
const workLowest = 2

// lowestConditions are the conditions of the group of students with the
// lowest mean of their scores, up to workLowest of them.
func lowestConditions(portraits []studentPortrait, of []MethodsStudent, conditions []string) []string {
	students := map[string]bool{}
	for _, s := range of {
		students[s.SessionID] = true
	}
	means, rated := map[string]float64{}, []string{}
	for _, id := range conditions {
		scores := []float64{}
		for _, p := range portraits {
			if !students[p.student.SessionID] {
				continue
			}
			for _, level := range p.data.Conditions {
				for _, ind := range level.Indicators {
					if ind.ID == id && ind.Score != nil {
						scores = append(scores, *ind.Score)
					}
				}
			}
		}
		if len(scores) > 0 {
			means[id], rated = meanOf(scores), append(rated, id)
		}
	}
	lowest := sortedByLowest(rated, means)
	return lowest[:min(workLowest, len(lowest))]
}

// addLook adds what to look at by the next testing; looks of the same kind
// about different qualities go in one: «ситуация по качествам «A», «B»».
func addLook(looks []string, look string) []string {
	prefix, quoted, found := strings.Cut(look, "«")
	for n, l := range looks {
		if l == look {
			return looks
		}
		if found && strings.HasPrefix(l, prefix+"«") {
			looks[n] = l + ", «" + quoted
			return looks
		}
	}
	return append(looks, look)
}

// humanBasis is a basis of a group for separate work in plain words: the
// qualities of the image it is about, if any, and what is the matter.
type humanBasis struct{ qualities, phrase string }

// forCount is the phrase for one student or for several: «one|many».
func forCount(n int, phrases string) string {
	one, many, _ := strings.Cut(phrases, "|")
	if n == 1 || many == "" {
		return one
	}
	return many
}

// qualities is the first quality of a group of the image, as its short name
// says it: «верность долгу». The run of 5 November: given «профессионализм,
// умение принимать решения», the model wrote the name of the group, which the
// check forbids, «профессионализм и умение принимать решения».
func qualities(name string) string {
	if g, ok := plainGroups[name]; ok {
		first, _, _ := strings.Cut(g.Short, " и ")
		return first
	}
	return lowerFirst(name)
}

// The plain bases of the situations, the leading statuses of the military
// speciality, the pressure and the scales on the low level.
var (
	humanWork = map[string]string{
		situationNotAccepted: "пока не считает это для себя важным|пока не считают это для себя важным",
		situationBelow:       "ставит себе планку ниже, чем оценивает себя сейчас|ставят себе планку ниже, чем оценивают себя сейчас",
		situationFar:         "это для него важно, но до этого пока далеко|это для них важно, но до этого пока далеко",
	}
	humanWorkIdentity = map[string]string{
		"NP": "о выборе военной специальности пока всерьёз не задумывается|о выборе военной специальности пока всерьёз не задумываются",
		"NV": "военную специальность выбрал под влиянием других|военную специальность выбрали под влиянием других",
		"M":  "с военной специальностью ещё выбирает и сомневается|с военной специальностью ещё выбирают и сомневаются",
	}
	humanWorkPressure = "чувствует, что от него ждут большего, чем он может сейчас|чувствуют, что от них ждут большего, чем они могут сейчас"
	humanLowScales    = map[string]string{
		"Военно-профессиональная мотивация": "желание служить пока слабое",
		"Выраженность волевых качеств":      "волевые качества пока развиты слабо",
		"Общая волевая саморегуляция":       "владеть собой пока трудно",
		"Настойчивость":                     "не хватает настойчивости",
		"Самообладание":                     "не хватает самообладания",
		"Эмоциональная устойчивость":        "трудно справляться с волнением",
		"Активность расширения и обогащения профессиональных и общекультурных знаний, умений, навыков": "стремление узнавать и уметь больше пока слабое",
		"Критическая, справедливая оценка накопленного опыта":                                          "трезво оценивать свой опыт пока трудно",
		"Реальное и потенциальное использование приобретаемого опыта, знаний, умений, навыков":         "применять опыт в новых делах пока трудно",
	}
)

// joinedBases puts the bases of a group with the same phrase in one, their
// qualities together; all the groups of the image are «все качества
// защитника Отечества».
func joinedBases(bases []humanBasis, groups int) []string {
	order, by := []string{}, map[string][]string{}
	for _, b := range bases {
		if _, seen := by[b.phrase]; !seen {
			order = append(order, b.phrase)
		}
		if b.qualities != "" {
			by[b.phrase] = append(by[b.phrase], b.qualities)
		} else if by[b.phrase] == nil {
			by[b.phrase] = []string{}
		}
	}
	joined := []string{}
	for _, phrase := range order {
		switch q := by[phrase]; {
		case len(q) == 0:
			joined = append(joined, phrase)
		case len(q) == groups:
			joined = append(joined, "все качества защитника Отечества: "+phrase)
		default:
			joined = append(joined, strings.Join(q, "; ")+": "+phrase)
		}
	}
	return joined
}
