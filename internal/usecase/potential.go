package usecase

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
	"vucdiagnostic/internal/domain/model"
)

// The potential of a student by the autobiography: a level in each of six
// areas, counted by the rules of Вики/Автобиография/Шкала потенциала.md.
// Levels describe what the student reported about themselves; free text
// (achievements) does not take part in the rules.

const autobiographyForm = "autobiography"

const (
	levelNone = iota
	levelInitial
	levelMedium
	levelHigh
)

var levelIDs = []string{"none", "initial", "medium", "high"}
var levelTexts = []string{"не выражен", "начальный", "средний", "высокий"}

var potentialAreas = []struct{ ID, Title string }{
	{"sport", "Спорт"},
	{"creative", "Музыка и творчество"},
	{"patriotic", "Военно-патриотическая деятельность"},
	{"science", "Наука"},
	{"volunteering", "Волонтёрство"},
	{"languages", "Иностранные языки"},
}

// PotentialArea is the level in one area and the answers it came from.
type PotentialArea struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Level     string   `json:"level"`
	LevelText string   `json:"levelText"`
	Reasons   []string `json:"reasons"`
}

type PotentialAward struct {
	Name string `json:"name"`
	Year string `json:"year"`
}

// Potential has no overall level: High lists the areas with the high level instead.
// State awards do not affect levels and are shown on their own line.
type Potential struct {
	Areas  []PotentialArea  `json:"areas"`
	High   []string         `json:"high"`
	Awards []PotentialAward `json:"awards"`
}

// formAnswers is one level of stored answers: the submission or a group record.
type formAnswers map[string]json.RawMessage

func (a formAnswers) text(id string) string {
	var value string
	_ = json.Unmarshal(a[id], &value)
	return value
}

func (a formAnswers) list(id string) []string {
	var value []string
	_ = json.Unmarshal(a[id], &value)
	return value
}

func (a formAnswers) records(id string) []formAnswers {
	var value []formAnswers
	_ = json.Unmarshal(a[id], &value)
	return value
}

func (a formAnswers) is(id string, values ...string) bool {
	return slices.Contains(values, a.text(id))
}

// candidate is one rule that holds: the level it gives and why.
type candidate struct {
	level  int
	reason string
}

// best returns the highest level and the reasons of every rule that gives it.
func best(candidates []candidate) (int, []string) {
	level := levelNone
	for _, c := range candidates {
		level = max(level, c.level)
	}
	reasons := []string{}
	for _, c := range candidates {
		if c.level == level && level > levelNone {
			reasons = append(reasons, c.reason)
		}
	}
	return level, reasons
}

// potentialScale reads option texts of the form for the reasons.
type potentialScale struct {
	form   model.Form
	fields map[string]model.FormField
}

func newPotentialScale(form model.Form) potentialScale {
	scale := potentialScale{form: form, fields: map[string]model.FormField{}}
	var walk func([]model.FormField)
	walk = func(fields []model.FormField) {
		for _, field := range fields {
			if _, seen := scale.fields[field.ID]; !seen {
				scale.fields[field.ID] = field
			}
			walk(field.Fields)
		}
	}
	for _, section := range form.Sections {
		walk(section.Fields)
	}
	return scale
}

func (p potentialScale) option(field, value string) string {
	for _, option := range p.form.Options(p.fields[field]) {
		if option.ID == value {
			return option.Text
		}
	}
	return value
}

// pair is "label — value", e.g. "разряд — КМС".
func (p potentialScale) pair(a formAnswers, field string) string {
	return lowerFirst(p.fields[field].Label) + " — " + p.option(field, a.text(field))
}

func lowerFirst(text string) string {
	if text == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(text)
	return string(unicode.ToLower(r)) + text[size:]
}

func upperFirst(text string) string {
	if text == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(text)
	return string(unicode.ToUpper(r)) + text[size:]
}

// named returns the option text of a record's main choice, or the text
// typed for "other".
func (p potentialScale) named(a formAnswers, field, other string) string {
	if a.text(field) == "other" && strings.TrimSpace(a.text(other)) != "" {
		return a.text(other)
	}
	return p.option(field, a.text(field))
}

// autobiographyPotential counts the potential of a submitted autobiography.
func autobiographyPotential(form model.Form, answers map[string]json.RawMessage) Potential {
	scale := newPotentialScale(form)
	a := formAnswers(answers)
	levels := map[string]func(formAnswers) (int, []string){
		"sport":        scale.sport,
		"creative":     scale.creative,
		"patriotic":    scale.patriotic,
		"science":      scale.science,
		"volunteering": scale.volunteering,
		"languages":    scale.languages,
	}
	result := Potential{Areas: []PotentialArea{}, High: []string{}, Awards: []PotentialAward{}}
	for _, area := range potentialAreas {
		level, reasons := levels[area.ID](a)
		result.Areas = append(result.Areas, PotentialArea{ID: area.ID, Title: area.Title, Level: levelIDs[level], LevelText: levelTexts[level], Reasons: reasons})
		if level == levelHigh {
			result.High = append(result.High, area.Title)
		}
	}
	if a.is("awards_has", "yes") {
		for _, award := range a.records("awards_list") {
			result.Awards = append(result.Awards, PotentialAward{Name: award.text("award_name"), Year: award.text("award_year")})
		}
	}
	return result
}

// Section 12. The level is counted by the best record; a sport given up
// earlier counts the same as a current one.
func (p potentialScale) sport(a formAnswers) (int, []string) {
	candidates := []candidate{}
	if a.is("sport_has", "yes") {
		for _, r := range a.records("sport_list") {
			name := p.named(r, "sport_kind", "sport_kind_other")
			var high, medium []string
			if r.is("sport_rank", "r1", "kms", "ms", "msmk") {
				high = append(high, p.pair(r, "sport_rank"))
			}
			if r.is("sport_level", "region", "russia", "world") {
				high = append(high, p.pair(r, "sport_level"))
			}
			if r.is("sport_rank", "junior", "r3", "r2") {
				medium = append(medium, p.pair(r, "sport_rank"))
			}
			if r.is("sport_level", "school", "city") {
				medium = append(medium, p.pair(r, "sport_level"))
			}
			if r.is("sport_years", "y35", "y5") {
				medium = append(medium, p.pair(r, "sport_years"))
			}
			switch {
			case len(high) > 0:
				candidates = append(candidates, candidate{levelHigh, name + ": " + strings.Join(high, ", ")})
			case len(medium) > 0:
				candidates = append(candidates, candidate{levelMedium, name + ": " + strings.Join(medium, ", ")})
			default:
				candidates = append(candidates, candidate{levelInitial, name + ": " + p.pair(r, "sport_years") + ", без разряда и соревнований"})
			}
		}
		if len(candidates) == 0 {
			candidates = append(candidates, candidate{levelInitial, "Занимается или занимался(лась) спортом"})
		}
	}
	switch a.text("gto") {
	case "gold":
		candidates = append(candidates, candidate{levelMedium, "Знак ГТО — " + p.option("gto", "gold")})
	case "bronze", "silver":
		candidates = append(candidates, candidate{levelInitial, "Знак ГТО — " + p.option("gto", a.text("gto"))})
	}
	return best(candidates)
}

// Section 13, by the best record.
func (p potentialScale) creative(a formAnswers) (int, []string) {
	if !a.is("creative_has", "yes") {
		return levelNone, []string{}
	}
	candidates := []candidate{}
	for _, r := range a.records("creative_list") {
		name := p.option("direction", r.text("direction"))
		if detail := strings.TrimSpace(r.text("detail")); detail != "" {
			name += " (" + detail + ")"
		}
		var high, medium []string
		if r.is("level", "region", "russia", "world") {
			high = append(high, p.pair(r, "level"))
		}
		if r.is("learning", "college") {
			high = append(high, p.pair(r, "learning"))
		}
		if r.is("learning", "art_school_done") {
			medium = append(medium, p.pair(r, "learning"))
		}
		if r.is("level", "school", "city") {
			medium = append(medium, p.pair(r, "level"))
		}
		switch {
		case len(high) > 0:
			candidates = append(candidates, candidate{levelHigh, name + ": " + strings.Join(high, ", ")})
		case len(medium) > 0:
			candidates = append(candidates, candidate{levelMedium, name + ": " + strings.Join(medium, ", ")})
		case r.text("learning") != "":
			candidates = append(candidates, candidate{levelInitial, name + ": " + p.pair(r, "learning") + ", без конкурсов"})
		default:
			candidates = append(candidates, candidate{levelInitial, name + ": без конкурсов"})
		}
	}
	if len(candidates) == 0 {
		candidates = append(candidates, candidate{levelInitial, "Занимается или занимался(лась) музыкой и творчеством"})
	}
	return best(candidates)
}

// Section 14 in three steps: participation, breadth of experience, then the
// floor set by military service (section 5) and cadet education (section 2).
// "Up to a year" includes one-off participation.
func (p potentialScale) patriotic(a formAnswers) (int, []string) {
	level, reasons := levelNone, []string{}
	if a.is("patriot_has", "yes") {
		lead := a.is("patriot_role", "leader", "organizer")
		long := a.is("patriot_term", "y13", "y3")
		wide := a.is("patriot_level", "region", "russia")
		switch {
		case lead && long, a.is("patriot_term", "y3"), wide && long:
			level = levelHigh
		case long, lead, wide:
			level = levelMedium
		default:
			level = levelInitial
		}
		reasons = append(reasons, "Участие: "+strings.Join([]string{p.pair(a, "patriot_role"), p.pair(a, "patriot_term"), p.pair(a, "patriot_level")}, ", "))
		if !a.is("patriot_term", "once") && level < levelHigh {
			forms, skills := len(a.list("patriot_what")), len(a.list("patriot_skills"))
			switch {
			case forms >= 3:
				level++
				reasons = append(reasons, fmt.Sprintf("Уровень поднят на ступень: отмечено форм участия — %d", forms))
			case skills >= 4:
				level++
				reasons = append(reasons, fmt.Sprintf("Уровень поднят на ступень: отмечено навыков — %d", skills))
			}
		}
	}
	floors := []candidate{}
	if a.is("served", "yes") {
		floors = append(floors, candidate{levelHigh, "Служил(а) в Вооружённых силах"})
	}
	for _, r := range a.records("education_list") {
		switch {
		case r.is("edu_type", "suvorov"):
			floors = append(floors, candidate{levelHigh, p.option("edu_type", "suvorov")})
		case r.is("edu_type", "cadet"):
			floors = append(floors, candidate{levelMedium, p.option("edu_type", "cadet")})
		case r.is("edu_profile", "cadet"):
			floors = append(floors, candidate{levelMedium, "Профиль «" + p.option("edu_profile", "cadet") + "»"})
		}
	}
	floor, why := best(floors)
	if floor >= level && floor > levelNone {
		for _, text := range uniqueStrings(why) {
			reasons = append(reasons, text+": уровень не ниже "+map[int]string{levelHigh: "высокого", levelMedium: "среднего"}[floor])
		}
		level = floor
	}
	return level, reasons
}

// Section 15. The number of forms does not affect the level.
func (p potentialScale) science(a formAnswers) (int, []string) {
	if !a.is("science_has", "yes") {
		return levelNone, []string{}
	}
	prize := a.is("science_result", "prize", "winner")
	wide := a.is("science_level", "region", "russia", "world")
	outcome := upperFirst(strings.Join([]string{p.pair(a, "science_result"), p.pair(a, "science_level")}, ", "))
	candidates := []candidate{}
	switch {
	case prize && wide:
		candidates = append(candidates, candidate{levelHigh, outcome})
	case prize || wide:
		candidates = append(candidates, candidate{levelMedium, outcome})
	default:
		candidates = append(candidates, candidate{levelInitial, outcome})
	}
	forms := a.list("science_forms")
	if slices.Contains(forms, "publications") {
		candidates = append(candidates, candidate{levelHigh, "Есть публикации"})
	}
	if slices.Contains(forms, "lab") {
		candidates = append(candidates, candidate{levelMedium, p.option("science_forms", "lab")})
	}
	return best(candidates)
}

// Section 16. The number of directions does not affect the level.
// "Up to a year" includes one-off participation.
func (p potentialScale) volunteering(a formAnswers) (int, []string) {
	if !a.is("vol_has", "yes") {
		return levelNone, []string{}
	}
	lead := a.is("vol_role", "coordinator")
	level := levelInitial
	switch {
	case lead && a.is("vol_term", "y13", "y3"), a.is("vol_term", "y3"):
		level = levelHigh
	case lead, a.is("vol_term", "y13"):
		level = levelMedium
	}
	return level, []string{upperFirst(strings.Join([]string{p.pair(a, "vol_role"), p.pair(a, "vol_term")}, ", "))}
}

// Section 6. Only the fact of a certificate counts: its result is typed as text.
func (p potentialScale) languages(a formAnswers) (int, []string) {
	if !a.is("lang_has", "yes") {
		return levelNone, []string{}
	}
	candidates := []candidate{}
	talk := []string{}
	for _, r := range a.records("lang_list") {
		name := p.named(r, "language", "language_other")
		practice := r.list("practice")
		switch r.text("degree") {
		case "fluent":
			candidates = append(candidates, candidate{levelHigh, name + ": " + lowerFirst(p.option("degree", "fluent"))})
		case "talk":
			talk = append(talk, name)
			candidates = append(candidates, candidate{levelMedium, name + ": " + lowerFirst(p.option("degree", "talk"))})
		default:
			candidates = append(candidates, candidate{levelInitial, name + ": " + lowerFirst(p.option("degree", r.text("degree")))})
		}
		if slices.Contains(practice, "second") {
			candidates = append(candidates, candidate{levelHigh, name + ": " + lowerFirst(p.option("practice", "second"))})
		}
		if slices.Contains(practice, "exam") {
			exam := p.named(r, "exam", "exam_other")
			if exam == "" {
				exam = lowerFirst(p.option("practice", "exam"))
			}
			candidates = append(candidates, candidate{levelMedium, name + ": сертификат " + exam})
		}
		for _, kind := range []string{"abroad", "env"} {
			if slices.Contains(practice, kind) {
				candidates = append(candidates, candidate{levelMedium, name + ": " + lowerFirst(p.option("practice", kind))})
			}
		}
	}
	if len(talk) >= 2 {
		candidates = append(candidates, candidate{levelHigh, fmt.Sprintf("Языков на уровне «%s» — %d: %s", p.option("degree", "talk"), len(talk), strings.Join(talk, ", "))})
	}
	if len(candidates) == 0 {
		candidates = append(candidates, candidate{levelInitial, "Владеет иностранным языком"})
	}
	return best(candidates)
}

func uniqueStrings(values []string) []string {
	result := []string{}
	for _, value := range values {
		if !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}
