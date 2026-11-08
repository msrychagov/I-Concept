package usecase

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/formbank"
)

// The conclusion on a submitted autobiography that the program writes itself,
// without a language model (decided 6 November 2026, Вики/Автобиография/План
// вывода программой.md). The texts are in the phrase book
// internal/formbank/phrases/autobiography-conclusion.json; this file decides
// which phrase goes where. Rules: Вики/Для руководителя/Выводы без модели —
// словарь фраз и ключ.md, section «Автобиография». The conclusion is counted
// anew on every request and is not stored: the same answers give the same text.

// AutobiographyConclusion is what the teacher reads in the window «Анкета».
type AutobiographyConclusion struct {
	// Paragraphs: the story of the student (facts and results), then what
	// follows from it (qualities). The second one is absent when no rule holds.
	Paragraphs []string `json:"paragraphs"`
	// Areas are the lines of the areas with experience, in the order of the potential.
	Areas []ConclusionArea `json:"areas"`
	// Attention is about the family and upbringing, for the teacher.
	Attention []string `json:"attention"`
	// Training: service, work before university, family status.
	Training []string `json:"training"`
	// Consider: criminal liability of the student and relatives, foreign citizenship.
	Consider []string `json:"consider"`
}

// ConclusionArea is one line «Спорт: физически развит отлично. …»; Text starts
// with a small letter, it goes after the title and a colon.
type ConclusionArea struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type sportGroup struct {
	Physical  bool   `json:"physical"`
	Qualities string `json:"qualities"`
	Traits    string `json:"traits"`
}

type sportKind struct {
	Name  string `json:"name"`
	Group string `json:"group"`
}

type creativeDirection struct {
	Name      string `json:"name"`
	Qualities string `json:"qualities"`
}

// conclusionPhrases mirrors the phrase book.
type conclusionPhrases struct {
	Nothing   string                       `json:"nothing"`
	Levels    map[string]map[string]string `json:"levels"`
	Terms     map[string]string            `json:"terms"`
	Education struct {
		Types       map[string]string `json:"types"`
		TypesIn     map[string]string `json:"typesIn"`
		Feminine    []string          `json:"feminine"`
		Profiles    map[string]string `json:"profiles"`
		CadetClass  string            `json:"cadetClass"`
		Finished    string            `json:"finished"`
		NotFinished string            `json:"notFinished"`
		Medal       string            `json:"medal"`
		Honours     string            `json:"honours"`
		Speciality  string            `json:"speciality"`
	} `json:"education"`
	Patriotic struct {
		Forms       map[string]string `json:"forms"`
		Levels      map[string]string `json:"levels"`
		Roles       map[string]string `json:"roles"`
		LineRoles   map[string]string `json:"lineRoles"`
		Skills      map[string]string `json:"skills"`
		Training    string            `json:"training"`
		Own         string            `json:"own"`
		OwnWords    []string          `json:"ownWords"`
		Responsible string            `json:"responsible"`
		Heritage    string            `json:"heritage"`
		Other       string            `json:"other"`
		OnlyOther   string            `json:"onlyOther"`
	} `json:"patriotic"`
	Science struct {
		Forms    map[string]string `json:"forms"`
		Areas    map[string]string `json:"areas"`
		Levels   map[string]string `json:"levels"`
		Results  map[string]string `json:"results"`
		InArea   string            `json:"inArea"`
		Analytic string            `json:"analytic"`
		Tasks    map[string]string `json:"tasks"`
	} `json:"science"`
	Sport struct {
		Now               string                `json:"now"`
		Before            string                `json:"before"`
		Earlier           string                `json:"earlier"`
		Ranks             map[string]string     `json:"ranks"`
		ShortRanks        map[string]string     `json:"shortRanks"`
		Contests          string                `json:"contests"`
		Levels            map[string]string     `json:"levels"`
		GTO               map[string]string     `json:"gto"`
		Professional      string                `json:"professional"`
		Amateur           string                `json:"amateur"`
		Developed         string                `json:"developed"`
		Experience        string                `json:"experience"`
		Fighter           string                `json:"fighter"`
		Physical          string                `json:"physical"`
		LookAhead         string                `json:"lookAhead"`
		Groups            map[string]sportGroup `json:"groups"`
		Fighters          []string              `json:"fighters"`
		FighterRanks      []string              `json:"fighterRanks"`
		LookAheadGroups   []string              `json:"lookAheadGroups"`
		ProfessionalRanks []string              `json:"professionalRanks"`
		ExperienceLevels  []string              `json:"experienceLevels"`
		Kinds             map[string]sportKind  `json:"kinds"`
		OtherKind         string                `json:"otherKind"`
	} `json:"sport"`
	Creative struct {
		Directions       map[string]creativeDirection `json:"directions"`
		InstrumentSchool string                       `json:"instrumentSchool"`
		Learning         map[string]string            `json:"learning"`
		Contests         string                       `json:"contests"`
		Levels           map[string]string            `json:"levels"`
		Developed        string                       `json:"developed"`
	} `json:"creative"`
	Languages struct {
		Fluent        string            `json:"fluent"`
		Talk          string            `json:"talk"`
		Dict          string            `json:"dict"`
		LineDegrees   map[string]string `json:"lineDegrees"`
		Practice      map[string]string `json:"practice"`
		Summary       string            `json:"summary"`
		SummaryFluent string            `json:"summaryFluent"`
		SummaryPlain  string            `json:"summaryPlain"`
		SummaryItem   string            `json:"summaryItem"`
		SummaryNext   string            `json:"summaryNext"`
		SummaryBare   string            `json:"summaryBare"`
		LevelExam     string            `json:"levelExam"`
		ExamOnly      string            `json:"examOnly"`
		Native        string            `json:"native"`
		AbroadLine    string            `json:"abroadLine"`
	} `json:"languages"`
	Volunteering struct {
		Directions  map[string]string `json:"directions"`
		Sentence    string            `json:"sentence"`
		Coordinator string            `json:"coordinator"`
		Organizer   string            `json:"organizer"`
	} `json:"volunteering"`
	Trips struct {
		Purposes map[string]string `json:"purposes"`
		Sentence string            `json:"sentence"`
		Lived    string            `json:"lived"`
	} `json:"trips"`
	Hobbies struct {
		Items    map[string]string `json:"items"`
		Sentence string            `json:"sentence"`
		Own      string            `json:"own"`
	} `json:"hobbies"`
	Routine          string `json:"routine"`
	Leader           string `json:"leader"`
	PatriotOrganizer string `json:"patriotOrganizer"`
	Attention        struct {
		Military      string            `json:"military"`
		NoFather      string            `json:"noFather"`
		UnknownFather string            `json:"unknownFather"`
		ContactFather string            `json:"contactFather"`
		Coach         string            `json:"coach"`
		NoMother      string            `json:"noMother"`
		UnknownMother string            `json:"unknownMother"`
		Guardians     string            `json:"guardians"`
		Relatives     string            `json:"relatives"`
		Orphanage     string            `json:"orphanage"`
		Died          string            `json:"died"`
		Age           string            `json:"age"`
		Parents       map[string]string `json:"parents"`
	} `json:"attention"`
	Training struct {
		Served         string            `json:"served"`
		ServedOther    string            `json:"servedOther"`
		Position       string            `json:"position"`
		Rank           string            `json:"rank"`
		Work           string            `json:"work"`
		WorkNoPosition string            `json:"workNoPosition"`
		Married        string            `json:"married"`
		Children       string            `json:"children"`
		Child          string            `json:"child"`
		Kids           string            `json:"kids"`
		WithChildren   string            `json:"withChildren"`
		Branches       map[string]string `json:"branches"`
	} `json:"training"`
	Consider struct {
		Own         string            `json:"own"`
		Relative    string            `json:"relative"`
		Foreign     string            `json:"foreign"`
		Who         map[string]string `json:"who"`
		Whom        map[string]string `json:"whom"`
		Feminine    []string          `json:"feminine"`
		Was         string            `json:"was"`
		WasFeminine string            `json:"wasFeminine"`
	} `json:"consider"`
	Countries struct {
		In       map[string]string `json:"in"`
		To       map[string]string `json:"to"`
		Fallback string            `json:"fallback"`
	} `json:"countries"`
	Forbidden []string `json:"forbidden"`
}

func parseConclusionPhrases(data []byte) (*conclusionPhrases, error) {
	var p conclusionPhrases
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("autobiography phrases: %w", err)
	}
	return &p, nil
}

var loadedPhrases = sync.OnceValues(func() (*conclusionPhrases, error) {
	return parseConclusionPhrases(formbank.AutobiographyPhrases())
})

var placeholder = regexp.MustCompile(`\{[A-Za-z]+\}`)

// fill puts the values into a phrase: {level} → values["level"].
func fill(phrase string, values map[string]string) string {
	return placeholder.ReplaceAllStringFunc(phrase, func(key string) string {
		return values[strings.Trim(key, "{}")]
	})
}

// joinAnd is «A, B и C»; when an item has its own «и», only commas: «первую
// помощь и тактическую медицину, топографию».
func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	if slices.ContainsFunc(items, func(item string) bool { return strings.Contains(item, " и ") }) {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:len(items)-1], ", ") + " и " + items[len(items)-1]
}

// sentence ends a text with a full stop and starts it with a capital letter.
func fullSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if !strings.HasSuffix(text, ".") {
		text += "."
	}
	return upperFirst(text)
}

// yearsWord is «1 год», «3 года», «11 лет».
func yearsWord(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 14:
		return fmt.Sprintf("%d лет", n)
	case n%10 == 1:
		return fmt.Sprintf("%d год", n)
	case n%10 >= 2 && n%10 <= 4:
		return fmt.Sprintf("%d года", n)
	}
	return fmt.Sprintf("%d лет", n)
}

// conclusionWriter holds what every part of the conclusion reads.
type conclusionWriter struct {
	p         *conclusionPhrases
	scale     potentialScale
	a         formAnswers
	potential Potential
}

func (w conclusionWriter) typed(r formAnswers, id string) string {
	return strings.TrimRight(strings.Join(strings.Fields(r.text(id)), " "), ". ;,")
}

func (w conclusionWriter) level(area string) string {
	for _, item := range w.potential.Areas {
		if item.ID == area {
			return item.Level
		}
	}
	return "none"
}

func (w conclusionWriter) sports() []formAnswers {
	if !w.a.is("sport_has", "yes") {
		return nil
	}
	return w.a.records("sport_list")
}

func (w conclusionWriter) sportGroup(r formAnswers) sportGroup {
	return w.p.Sport.Groups[w.p.Sport.Kinds[r.text("sport_kind")].Group]
}

func (w conclusionWriter) hasSportGroup(groups []string) bool {
	return slices.ContainsFunc(w.sports(), func(r formAnswers) bool {
		return slices.Contains(groups, w.p.Sport.Kinds[r.text("sport_kind")].Group)
	})
}

func (w conclusionWriter) cadet() bool {
	for _, r := range w.a.records("education_list") {
		if r.is("edu_type", "cadet", "suvorov") || r.is("edu_profile", "cadet") {
			return true
		}
	}
	return false
}

func (w conclusionWriter) languages() []formAnswers {
	if !w.a.is("lang_has", "yes") {
		return nil
	}
	return w.a.records("lang_list")
}

// autobiographyConclusion writes the conclusion on submitted answers.
func autobiographyConclusion(form model.Form, answers map[string]json.RawMessage, potential Potential) AutobiographyConclusion {
	result := AutobiographyConclusion{Paragraphs: []string{}, Areas: []ConclusionArea{}, Attention: []string{}, Training: []string{}, Consider: []string{}}
	p, err := loadedPhrases()
	if err != nil {
		return result
	}
	w := conclusionWriter{p: p, scale: newPotentialScale(form), a: formAnswers(answers), potential: potential}
	story := strings.Join(nonEmpty(w.education(), w.patriotic(), w.science(), w.sport(), w.creative(), w.languageStory(), w.volunteering(), w.trips(), w.hobbies()), " ")
	if story == "" {
		story = p.Nothing
	}
	result.Paragraphs = append(result.Paragraphs, story)
	if qualities := strings.Join(w.qualities(), " "); qualities != "" {
		result.Paragraphs = append(result.Paragraphs, qualities)
	}
	lines := map[string]func(string) string{
		"sport": w.sportLine, "creative": w.creativeLine, "patriotic": w.patrioticLine,
		"science": w.scienceLine, "volunteering": w.volunteeringLine, "languages": w.languagesLine,
	}
	for _, area := range potential.Areas {
		if area.Level == "none" {
			continue
		}
		result.Areas = append(result.Areas, ConclusionArea{ID: area.ID, Title: area.Title, Text: lines[area.ID](area.Level)})
	}
	result.Attention = w.attention()
	result.Training = w.training()
	result.Consider = w.consider()
	return result
}

func nonEmpty(parts ...string) []string {
	out := []string{}
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return out
}

// levelWord is «физически развит отлично» or «подготовлен хорошо».
func (w conclusionWriter) levelWord(kind, level string) string {
	return w.p.Levels[kind][level]
}

// Section 2. Only a schooling worth telling: cadet, a profile, a medal or a speciality.
func (w conclusionWriter) education() string {
	e := w.p.Education
	out := []string{}
	for _, r := range w.a.records("education_list") {
		kind, profile := r.text("edu_type"), r.text("edu_profile")
		higher := kind == "college" || kind == "university"
		medal := r.is("edu_medal", "yes")
		speciality := w.typed(r, "edu_speciality")
		adjective := e.Profiles[profile]
		if !(kind == "cadet" || kind == "suvorov" || profile == "cadet" || adjective != "" || medal || (higher && speciality != "")) {
			continue
		}
		// The medal goes right after «Окончил»: «Окончил с медалью
		// физико-математический лицей», not «лицей с медалью».
		finished := e.Finished
		if medal && !r.is("edu_finished", "no") {
			if higher {
				finished += " " + e.Honours
			} else {
				finished += " " + e.Medal
			}
		}
		var text string
		switch {
		case profile == "cadet" && kind != "cadet" && kind != "suvorov":
			text = strings.Replace(e.CadetClass, e.Finished, finished, 1)
		case r.is("edu_finished", "no"):
			text = e.NotFinished + " " + w.profiled(kind, adjective, e.TypesIn[kind], true)
		default:
			text = finished + " " + w.profiled(kind, adjective, e.Types[kind], false)
		}
		if higher && speciality != "" {
			text += " " + fill(e.Speciality, map[string]string{"speciality": speciality})
		}
		out = append(out, fullSentence(text))
	}
	return strings.Join(out, " ")
}

// profiled puts the profile before the school: «физико-математический лицей»,
// «гуманитарную гимназию»; «в гуманитарной гимназии» for a school not finished.
func (w conclusionWriter) profiled(kind, adjective, noun string, in bool) string {
	if adjective == "" || kind == "cadet" || kind == "suvorov" {
		return noun
	}
	feminine := slices.Contains(w.p.Education.Feminine, kind)
	switch {
	case in && feminine:
		adjective = replaceEnding(adjective, "ой")
	case in:
		adjective = replaceEnding(adjective, "ом")
	case feminine:
		adjective = replaceEnding(adjective, "ую")
	}
	return adjective + " " + noun
}

// replaceEnding changes «-ий» / «-ый» of an adjective; «художественно-музыкальный» → «-ую».
func replaceEnding(adjective, ending string) string {
	for _, old := range []string{"ий", "ый"} {
		if stem, ok := strings.CutSuffix(adjective, old); ok {
			if ending == "ой" && old == "ий" && strings.HasSuffix(stem, "к") {
				return stem + "ой"
			}
			return stem + ending
		}
	}
	return adjective
}

func (w conclusionWriter) term(id string) string { return w.p.Terms[id] }

// Section 14.
func (w conclusionWriter) patrioticForms() []string {
	forms := []string{}
	for _, id := range w.a.list("patriot_what") {
		if id == "cadet_class" && w.cadet() {
			continue
		}
		if text := w.p.Patriotic.Forms[id]; text != "" {
			forms = append(forms, text)
		}
	}
	return forms
}

func (w conclusionWriter) patriotic() string {
	if !w.a.is("patriot_has", "yes") {
		return ""
	}
	pp := w.p.Patriotic
	other := ""
	if slices.Contains(w.a.list("patriot_what"), "other") {
		other = w.typed(w.a, "patriot_other")
	}
	forms := w.patrioticForms()
	var text string
	if len(forms) == 0 && other != "" {
		text = fill(pp.OnlyOther, map[string]string{"other": other})
	} else if len(forms) > 0 {
		text = strings.TrimSpace(w.term(w.a.text("patriot_term")) + " участвовал в " + joinAnd(forms))
		if other != "" {
			text += fill(pp.Other, map[string]string{"other": other})
		}
	} else {
		return ""
	}
	if level := pp.Levels[w.a.text("patriot_level")]; level != "" {
		text += " " + level
	}
	if role := pp.Roles[w.a.text("patriot_role")]; role != "" {
		text += " " + role
	}
	skills := []string{}
	for _, id := range w.a.list("patriot_skills") {
		if s := pp.Skills[id]; s != "" {
			skills = append(skills, s)
		}
	}
	if len(skills) > 0 {
		text += "; " + fill(pp.Training, map[string]string{"skills": joinAnd(skills)})
	}
	out := fullSentence(text)
	if ach := w.typed(w.a, "patriot_ach"); ach != "" {
		lower := strings.ToLower(ach)
		if slices.ContainsFunc(pp.OwnWords, func(word string) bool { return strings.HasPrefix(lower, word) }) {
			ach = pp.Own + lowerFirst(ach)
		}
		out += " " + fullSentence(ach)
	}
	return out
}

func (w conclusionWriter) patrioticLine(level string) string {
	pp := w.p.Patriotic
	text := w.levelWord("other", level) + "."
	if forms := w.patrioticForms(); len(forms) > 0 {
		role := pp.LineRoles[w.a.text("patriot_role")]
		if role == "" {
			role = pp.LineRoles["member"]
		}
		text += " " + fullSentence("имеет опыт "+role+" в "+joinAnd(forms))
	}
	traits := []string{}
	if w.a.is("patriot_role", "leader", "organizer") {
		traits = append(traits, pp.Responsible)
	}
	heritage := slices.Contains(w.a.list("hobbies"), "history") || slices.ContainsFunc(w.a.list("patriot_what"), func(id string) bool { return id == "search" || id == "honor" || id == "reenact" }) ||
		(w.a.is("vol_has", "yes") && slices.Contains(w.a.list("vol_dirs"), "patriotic"))
	if heritage {
		traits = append(traits, pp.Heritage)
	}
	if len(traits) > 0 {
		text += " " + fullSentence(strings.Join(traits, ", "))
	}
	return text
}

// Section 15. A result first, then the forms.
func (w conclusionWriter) scienceHead() (string, bool) {
	s := w.p.Science
	if ach := w.typed(w.a, "science_ach"); ach != "" {
		return ach, true
	}
	if phrase := s.Results[w.a.text("science_result")]; phrase != "" {
		return fill(phrase, map[string]string{"level": s.Levels[w.a.text("science_level")]}), true
	}
	return "", false
}

func (w conclusionWriter) scienceForms(forms map[string]string) []string {
	ach := strings.ToLower(w.typed(w.a, "science_ach"))
	out := []string{}
	for _, id := range w.a.list("science_forms") {
		if id == "olympiads" && strings.Contains(ach, "олимпиад") {
			continue
		}
		if text := forms[id]; text != "" {
			out = append(out, text)
		}
	}
	return out
}

func (w conclusionWriter) science() string {
	if !w.a.is("science_has", "yes") {
		return ""
	}
	s := w.p.Science
	head, ok := w.scienceHead()
	forms := strings.Join(w.scienceForms(s.Forms), ", ")
	area := ""
	if name := s.Areas[w.a.text("science_area")]; name != "" {
		area = fill(s.InArea, map[string]string{"area": name})
	}
	switch {
	case ok && forms != "":
		return fullSentence(head + "; " + forms)
	case ok:
		return fullSentence(head)
	case forms != "":
		return fullSentence(strings.TrimSpace(forms + " " + area))
	}
	return ""
}

func (w conclusionWriter) scienceLine(level string) string {
	text := w.levelWord("other", level) + "."
	if story := w.science(); story != "" {
		text += " " + story
	}
	return text
}

// Section 12.
func (w conclusionWriter) sportName(r formAnswers) string {
	if r.is("sport_kind", "other") || w.p.Sport.Kinds[r.text("sport_kind")].Name == "" {
		kind := w.typed(r, "sport_kind_other")
		if kind == "" {
			kind = w.scale.option("sport_kind", r.text("sport_kind"))
		}
		return fill(w.p.Sport.OtherKind, map[string]string{"kind": kind})
	}
	return w.p.Sport.Kinds[r.text("sport_kind")].Name
}

func (w conclusionWriter) sport() string {
	s := w.p.Sport
	clauses := []string{}
	for _, r := range w.sports() {
		verb := s.Now
		if r.is("sport_now", "before") {
			verb = s.Before
		}
		clause := strings.TrimSpace(w.term(r.text("sport_years")) + " " + verb + " " + w.sportName(r))
		if r.is("sport_now", "before") {
			clause = s.Earlier + " " + clause
		}
		if rank := s.Ranks[r.text("sport_rank")]; rank != "" {
			clause += ", " + rank
		}
		if ach := w.typed(r, "sport_ach"); ach != "" {
			clause += ", " + lowerFirst(ach)
		} else if level := s.Levels[r.text("sport_level")]; level != "" {
			clause += ", " + fill(s.Contests, map[string]string{"level": level})
		}
		clauses = append(clauses, clause)
	}
	return strings.Join(nonEmpty(fullSentence(strings.Join(clauses, "; ")), fullSentence(s.GTO[w.a.text("gto")])), " ")
}

func (w conclusionWriter) sportLine(level string) string {
	s := w.p.Sport
	kind := "other"
	if slices.ContainsFunc(w.sports(), func(r formAnswers) bool { return w.sportGroup(r).Physical }) {
		kind = "physical"
	}
	text := w.levelWord(kind, level) + "."
	for _, r := range w.sports() {
		name := w.scale.named(r, "sport_kind", "sport_kind_other")
		grade := s.Amateur
		if slices.Contains(s.ProfessionalRanks, r.text("sport_rank")) {
			grade = s.Professional
		}
		details := nonEmpty(s.ShortRanks[r.text("sport_rank")], w.typed(r, "sport_ach"))
		line := name + " — " + grade
		if len(details) > 0 {
			line += " (" + strings.Join(details, ", ") + ")"
		}
		extra := []string{}
		if q := w.sportGroup(r).Qualities; q != "" {
			extra = append(extra, fill(s.Developed, map[string]string{"qualities": q}))
		}
		if slices.Contains(s.ExperienceLevels, r.text("sport_level")) {
			extra = append(extra, fill(s.Experience, map[string]string{"level": s.Levels[r.text("sport_level")]}))
		}
		if len(extra) > 0 {
			line += ": " + strings.Join(extra, ", ")
		}
		text += " " + fullSentence(line)
	}
	return text
}

// Section 13.
func (w conclusionWriter) creativeClause(r formAnswers) string {
	c := w.p.Creative
	direction := r.text("direction")
	detail := w.typed(r, "detail")
	var clause string
	if direction == "instrument" && r.is("learning", "art_school_done") && detail != "" {
		clause = fill(c.InstrumentSchool, map[string]string{"detail": detail})
	} else {
		name := c.Directions[direction].Name
		if name == "" {
			name = c.Directions["other"].Name
		}
		if detail != "" {
			name += " (" + detail + ")"
		}
		learning := c.Learning[r.text("learning")]
		if learning == "" {
			learning = c.Learning["self"]
		}
		clause = fill(learning, map[string]string{"direction": name})
	}
	if level := c.Levels[r.text("level")]; level != "" {
		contest := fill(c.Contests, map[string]string{"level": level})
		if strings.HasPrefix(level, "вс") {
			contest = strings.Replace(contest, "в ", "во ", 1)
		}
		clause += ", " + contest
	}
	if ach := w.typed(r, "ach"); ach != "" {
		clause += "; " + lowerFirst(ach)
	}
	return clause
}

func (w conclusionWriter) creatives() []formAnswers {
	if !w.a.is("creative_has", "yes") {
		return nil
	}
	return w.a.records("creative_list")
}

func (w conclusionWriter) creative() string {
	out := []string{}
	for _, r := range w.creatives() {
		out = append(out, fullSentence(w.creativeClause(r)))
	}
	return strings.Join(out, " ")
}

func (w conclusionWriter) creativeLine(level string) string {
	c := w.p.Creative
	text := w.levelWord("other", level) + "."
	for _, r := range w.creatives() {
		clause := w.creativeClause(r)
		if q := c.Directions[r.text("direction")].Qualities; q != "" {
			clause += ": " + fill(c.Developed, map[string]string{"qualities": q})
		}
		text += " " + fullSentence(clause)
	}
	return text
}

// Section 6. «английским языком», «на английском языке»; a name that is not an
// adjective stays as it is: «языком хинди».
func languageForms(name string) (instr, prep, short string) {
	lower := strings.ToLower(name)
	if stem, ok := strings.CutSuffix(lower, "ий"); ok && !strings.Contains(lower, " ") {
		return stem + "им языком", stem + "ом языке", stem + "им"
	}
	return "языком «" + name + "»", "языке «" + name + "»", "языком «" + name + "»"
}

func (w conclusionWriter) languageName(r formAnswers) string {
	return w.scale.named(r, "language", "language_other")
}

func (w conclusionWriter) examName(r formAnswers) string {
	if r.is("exam", "other") {
		return w.typed(r, "exam_other")
	}
	return w.scale.option("exam", r.text("exam"))
}

// examResult writes a number with a comma: «7,5».
func examResult(result string) string {
	if _, err := strconv.ParseFloat(result, 64); err == nil {
		return strings.Replace(result, ".", ",", 1)
	}
	return result
}

var cefrLevel = regexp.MustCompile(`(?i)\b([ABC][12])\b`)

// cefr is the level of a language by an exam: IELTS 7,0–8,0 — C1 and so on;
// a result written as a level («B1») is taken as it is.
func cefr(exam, result string) string {
	result = strings.TrimSpace(strings.ReplaceAll(result, ",", "."))
	if m := cefrLevel.FindStringSubmatch(result); m != nil {
		return strings.ToUpper(m[1])
	}
	score, err := strconv.ParseFloat(result, 64)
	switch {
	case exam == "ielts" && err == nil:
		switch {
		case score >= 8.5:
			return "C2"
		case score >= 7:
			return "C1"
		case score >= 5.5:
			return "B2"
		case score >= 4:
			return "B1"
		}
	case exam == "toefl" && err == nil:
		switch {
		case score >= 114:
			return "C2"
		case score >= 95:
			return "C1"
		case score >= 72:
			return "B2"
		case score >= 42:
			return "B1"
		}
	case exam == "cambridge":
		upper := strings.ToUpper(result)
		switch {
		case strings.Contains(upper, "CPE"):
			return "C2"
		case strings.Contains(upper, "CAE"):
			return "C1"
		case strings.Contains(upper, "FCE"):
			return "B2"
		}
	}
	return ""
}

func (w conclusionWriter) country(code string, maps map[string]string) string {
	if text := maps[code]; text != "" {
		return text
	}
	if code == "" {
		return ""
	}
	return fill(w.p.Countries.Fallback, map[string]string{"country": w.scale.option("abroad_country", code)})
}

func (w conclusionWriter) languageStory() string {
	l := w.p.Languages
	out := []string{}
	for _, r := range w.languages() {
		instr, prep, _ := languageForms(w.languageName(r))
		var head string
		switch r.text("degree") {
		case "fluent":
			head = fill(l.Fluent, map[string]string{"instr": instr})
		case "talk":
			head = fill(l.Talk, map[string]string{"prep": prep})
		default:
			head = fill(l.Dict, map[string]string{"prep": prep})
		}
		practice := []string{}
		for _, id := range r.list("practice") {
			switch id {
			case "school":
				practice = append(practice, l.Practice["school_"+r.text("school_kind")])
			case "courses":
				practice = append(practice, strings.TrimSpace(fill(l.Practice["courses"], map[string]string{"term": w.term(r.text("courses_time"))})))
			case "exam":
				if result := w.typed(r, "exam_result"); result != "" {
					practice = append(practice, fill(l.Practice["exam"], map[string]string{"exam": w.examName(r), "result": examResult(result)}))
				}
			case "abroad":
				practice = append(practice, strings.TrimSpace(fill(l.Practice["abroad"], map[string]string{"term": w.term(r.text("abroad_time")), "country": w.country(r.text("abroad_country"), w.p.Countries.In)})))
			case "env":
				practice = append(practice, strings.TrimSpace(fill(l.Practice["env"], map[string]string{"country": w.country(r.text("env_country"), w.p.Countries.In)})))
			default:
				practice = append(practice, l.Practice[id])
			}
		}
		out = append(out, fullSentence(strings.Join(nonEmpty(append([]string{head}, practice...)...), ", ")))
	}
	return strings.Join(out, " ")
}

func (w conclusionWriter) languageLevel(r formAnswers) string {
	if !slices.Contains(r.list("practice"), "exam") {
		return ""
	}
	return cefr(r.text("exam"), w.typed(r, "exam_result"))
}

// languageSummary is «Свободно владеет английским языком на уровне C1, немецким — на уровне B1».
func (w conclusionWriter) languageSummary() string {
	l := w.p.Languages
	items := []string{}
	lead := ""
	for _, r := range w.languages() {
		level := w.languageLevel(r)
		fluent := r.is("degree", "fluent")
		if !fluent && (level == "" || level == "A1" || level == "A2") {
			continue
		}
		instr, _, short := languageForms(w.languageName(r))
		if lead == "" {
			lead = l.SummaryPlain
			if fluent {
				lead = l.SummaryFluent
			}
		}
		switch {
		case level == "":
			items = append(items, fill(l.SummaryBare, map[string]string{"instr": instr}))
		case len(items) == 0:
			items = append(items, fill(l.SummaryItem, map[string]string{"instr": instr, "level": level}))
		default:
			items = append(items, fill(l.SummaryNext, map[string]string{"short": short, "level": level}))
		}
	}
	if len(items) == 0 {
		return ""
	}
	return fill(l.Summary, map[string]string{"lead": lead, "list": strings.Join(items, ", ")})
}

func (w conclusionWriter) languagesLine(level string) string {
	l := w.p.Languages
	parts := []string{}
	for _, r := range w.languages() {
		part := w.languageName(r) + " — " + l.LineDegrees[r.text("degree")]
		if len(parts) > 0 {
			part = lowerFirst(part)
		}
		result := w.typed(r, "exam_result")
		if lvl := w.languageLevel(r); lvl != "" {
			exam := w.examName(r)
			if !cefrLevel.MatchString(result) {
				exam += " " + examResult(result)
			}
			part += ", " + fill(l.LevelExam, map[string]string{"level": lvl, "exam": exam})
		} else if slices.Contains(r.list("practice"), "exam") && result != "" {
			part += ", " + fill(l.ExamOnly, map[string]string{"exam": w.examName(r) + " " + examResult(result)})
		}
		if slices.Contains(r.list("practice"), "native") {
			part += ", " + l.Native
		}
		if slices.Contains(r.list("practice"), "abroad") {
			part += ", " + fill(l.AbroadLine, map[string]string{"country": w.country(r.text("abroad_country"), w.p.Countries.In)})
		}
		parts = append(parts, part)
	}
	text := w.levelWord("other", level) + "."
	if len(parts) > 0 {
		text += " " + fullSentence(strings.Join(parts, "; "))
	}
	return text
}

// Section 16.
func (w conclusionWriter) volunteeringSentence() string {
	v := w.p.Volunteering
	if !w.a.is("vol_has", "yes") {
		return ""
	}
	dirs := []string{}
	for _, id := range w.a.list("vol_dirs") {
		if id == "other" {
			if other := w.typed(w.a, "vol_other"); other != "" {
				dirs = append(dirs, other)
			}
			continue
		}
		if text := v.Directions[id]; text != "" {
			dirs = append(dirs, text)
		}
	}
	if len(dirs) == 0 {
		return ""
	}
	text := strings.TrimSpace(fill(v.Sentence, map[string]string{"term": w.term(w.a.text("vol_term")), "directions": joinAnd(dirs)}))
	if w.a.is("vol_role", "coordinator") {
		text += v.Coordinator
	}
	return fullSentence(text)
}

func (w conclusionWriter) volunteering() string { return w.volunteeringSentence() }

func (w conclusionWriter) volunteeringLine(level string) string {
	text := w.levelWord("other", level) + "."
	if s := w.volunteeringSentence(); s != "" {
		text += " " + s
	}
	return text
}

// Section 9, trips that the languages did not tell already.
func (w conclusionWriter) trips() string {
	t := w.p.Trips
	if !w.a.is("trips_has", "yes") {
		return ""
	}
	abroad, env := map[string]bool{}, map[string]bool{}
	for _, r := range w.languages() {
		if slices.Contains(r.list("practice"), "abroad") {
			abroad[r.text("abroad_country")] = true
		}
		if slices.Contains(r.list("practice"), "env") {
			env[r.text("env_country")] = true
		}
	}
	out := []string{}
	for _, r := range w.a.records("trips_list") {
		code, purpose := r.text("country"), r.text("purpose")
		if (purpose == "study" && abroad[code]) || (purpose == "lived" && env[code]) {
			continue
		}
		if purpose == "lived" {
			out = append(out, fullSentence(fill(t.Lived, map[string]string{"country": w.tripCountry(code, w.p.Countries.In)})))
			continue
		}
		out = append(out, fullSentence(strings.Join(strings.Fields(fill(t.Sentence, map[string]string{"purpose": t.Purposes[purpose], "country": w.tripCountry(code, w.p.Countries.To)})), " ")))
	}
	return strings.Join(out, " ")
}

func (w conclusionWriter) tripCountry(code string, maps map[string]string) string {
	if text := maps[code]; text != "" {
		return text
	}
	return fill(w.p.Countries.Fallback, map[string]string{"country": w.scale.option("country", code)})
}

// Section 17, without what is told above (chess in the sport).
func (w conclusionWriter) hobbies() string {
	h := w.p.Hobbies
	items := []string{}
	for _, id := range w.a.list("hobbies") {
		if id == "chess" && w.hasSportGroup(w.p.Sport.LookAheadGroups) {
			continue
		}
		if text := h.Items[id]; text != "" {
			items = append(items, text)
		}
	}
	out := ""
	if len(items) > 0 {
		out = fill(h.Sentence, map[string]string{"list": joinAnd(items)})
	}
	if own := w.typed(w.a, "hobby_own"); own != "" {
		out = strings.TrimSpace(out + " " + fill(h.Own, map[string]string{"own": own}))
	}
	return out
}

// qualities is the second paragraph: a phrase for every rule that holds.
func (w conclusionWriter) qualities() []string {
	s := w.p.Sport
	out := []string{}
	camps := slices.ContainsFunc(w.a.list("patriot_what"), func(id string) bool { return id == "camps" || id == "school_camps" }) && w.a.is("patriot_has", "yes")
	if w.cadet() || camps || w.a.is("served", "yes") {
		out = append(out, w.p.Routine)
	}
	switch {
	case w.a.is("patriot_has", "yes") && w.a.is("patriot_role", "leader"):
		out = append(out, w.p.Leader)
	case w.a.is("patriot_has", "yes") && w.a.is("patriot_role", "organizer"):
		out = append(out, w.p.PatriotOrganizer)
	case w.a.is("vol_has", "yes") && w.a.is("vol_role", "coordinator"):
		out = append(out, w.p.Volunteering.Organizer)
	}
	if slices.ContainsFunc(w.sports(), func(r formAnswers) bool {
		return slices.Contains(s.Fighters, s.Kinds[r.text("sport_kind")].Group) && slices.Contains(s.FighterRanks, r.text("sport_rank"))
	}) {
		out = append(out, s.Fighter)
	}
	if w.level("sport") == "high" {
		for _, r := range w.sports() {
			if g := w.sportGroup(r); g.Physical && g.Traits != "" {
				out = append(out, fill(s.Physical, map[string]string{"traits": g.Traits}))
				break
			}
		}
	}
	lookAhead := w.hasSportGroup(s.LookAheadGroups)
	tasks := w.p.Science.Tasks[w.a.text("science_area")]
	switch {
	case w.a.is("science_has", "yes") && w.level("science") == "high" && tasks != "":
		text := fill(w.p.Science.Analytic, map[string]string{"tasks": tasks})
		if lookAhead {
			text += ", " + s.LookAhead
		}
		out = append(out, fullSentence(text))
	case lookAhead:
		out = append(out, fullSentence(s.LookAhead))
	}
	if summary := w.languageSummary(); summary != "" {
		out = append(out, summary)
	}
	return out
}

func (w conclusionWriter) relatives() []formAnswers {
	if !w.a.is("rel_has", "yes") {
		return nil
	}
	return w.a.records("rel_list")
}

func (w conclusionWriter) parent(kind string) (formAnswers, bool) {
	for _, r := range w.relatives() {
		if r.is("kind", kind) {
			return r, true
		}
	}
	return nil, false
}

// attention: the family and upbringing (Вики/Автобиография/Психологу — основа.md).
func (w conclusionWriter) attention() []string {
	at := w.p.Attention
	out := []string{}
	father, hasFather := w.parent("father")
	mother, hasMother := w.parent("mother")
	switch w.a.text("upbringing") {
	case "mother":
		values := map[string]string{}
		switch {
		case !hasFather || father.is("known", "no"):
			values["unknown"] = at.UnknownFather
		case father.is("alive", "yes"):
			values["contact"] = at.ContactFather
		}
		if slices.ContainsFunc(w.sports(), func(r formAnswers) bool { return r.is("sport_years", "y35", "y5") }) {
			values["coach"] = at.Coach
		}
		out = append(out, fill(at.NoFather, values))
	case "father":
		values := map[string]string{}
		if !hasMother || mother.is("known", "no") {
			values["unknown"] = at.UnknownMother
		}
		out = append(out, fill(at.NoMother, values))
	case "guardians":
		out = append(out, at.Guardians)
	case "relatives":
		out = append(out, at.Relatives)
	case "orphanage":
		out = append(out, at.Orphanage)
	}
	birth := 0
	if date := w.a.text("birth_date"); len(date) >= 4 {
		birth, _ = strconv.Atoi(date[:4])
	}
	for _, kind := range []string{"father", "mother"} {
		r, ok := w.parent(kind)
		if !ok || !r.is("known", "yes") || !r.is("alive", "no") {
			continue
		}
		year, _ := strconv.Atoi(r.text("death_year"))
		if year == 0 {
			continue
		}
		values := map[string]string{"Parent": at.Parents[kind], "year": strconv.Itoa(year)}
		if kind == "mother" {
			values["a"] = "ла"
		}
		if birth > 0 && year-birth >= 1 {
			values["age"] = fill(at.Age, map[string]string{"age": yearsWord(year - birth)})
		}
		out = append(out, fill(at.Died, values))
	}
	for _, kind := range []string{"father", "mother"} {
		r, ok := w.parent(kind)
		if !ok || !r.is("known", "yes") || r.is("alive", "no") || !r.is("sphere", "voennaya_sluzhba") {
			continue
		}
		values := map[string]string{"Parent": at.Parents[kind]}
		if position := w.typed(r, "position"); position != "" {
			values["position"] = ", " + position
		}
		out = append(out, fill(at.Military, values))
	}
	return out
}

func (w conclusionWriter) training() []string {
	t := w.p.Training
	out := []string{}
	if w.a.is("served", "yes") {
		branch := t.Branches[w.a.text("service_branch")]
		if branch == "" {
			branch = t.ServedOther
		}
		details := []string{}
		if position := w.typed(w.a, "service_position"); position != "" {
			details = append(details, fill(t.Position, map[string]string{"position": position}))
		}
		if rank := w.a.text("service_rank"); rank != "" {
			details = append(details, fill(t.Rank, map[string]string{"rank": lowerFirst(w.scale.option("service_rank", rank))}))
		}
		joined := ""
		if len(details) > 0 {
			joined = ", " + strings.Join(details, ", ")
		}
		out = append(out, fill(t.Served, map[string]string{"branch": branch, "details": joined}))
	}
	if w.a.is("work_has", "yes") {
		positions := []string{}
		for _, r := range w.a.records("work_list") {
			if position := w.typed(r, "position"); position != "" {
				positions = append(positions, position)
			}
		}
		if len(positions) > 0 {
			out = append(out, fill(t.Work, map[string]string{"positions": strings.Join(positions, ", ")}))
		} else {
			out = append(out, t.WorkNoPosition)
		}
	}
	children := ""
	if w.a.is("children_has", "yes") {
		children = t.Child
		if len(w.a.records("children_list")) > 1 {
			children = t.Kids
		}
	}
	switch {
	case w.a.is("marital", "married"):
		with := ""
		if children != "" {
			with = fill(t.WithChildren, map[string]string{"children": children})
		}
		out = append(out, fill(t.Married, map[string]string{"children": with}))
	case children != "":
		out = append(out, fill(t.Children, map[string]string{"children": children}))
	}
	return out
}

func (w conclusionWriter) consider() []string {
	c := w.p.Consider
	out := []string{}
	if w.a.is("crim_has", "yes") {
		out = append(out, c.Own)
	}
	for _, r := range w.relatives() {
		if !r.is("known", "yes") || !r.is("crim", "yes") {
			continue
		}
		kind := r.text("kind")
		was := c.Was
		if slices.Contains(c.Feminine, kind) {
			was = c.WasFeminine
		}
		what := ""
		if text := w.typed(r, "crim_what"); text != "" {
			what = " (" + text + ")"
		}
		out = append(out, fill(c.Relative, map[string]string{"Who": c.Who[kind], "was": was, "what": what}))
	}
	for _, r := range w.relatives() {
		if !r.is("known", "yes") || !r.is("foreign", "yes") {
			continue
		}
		out = append(out, fill(c.Foreign, map[string]string{"whom": c.Whom[r.text("kind")], "country": w.scale.option("foreign_country", r.text("foreign_country"))}))
	}
	return out
}
