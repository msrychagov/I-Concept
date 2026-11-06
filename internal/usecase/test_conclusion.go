package usecase

import (
	"cmp"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"vucdiagnostic/internal/formbank"
)

// The conclusion on a finished test «Я-концепция» that the program writes
// itself, without a language model (decided 6 November 2026, Вики/Тестирование/
// План вывода программой.md). The texts are in the phrase book
// internal/formbank/phrases/test-conclusion.json; this file decides which
// phrase goes where. Rules: Вики/Для руководителя/Тест — словарь фраз.md. The
// conclusion is counted from the input of test/v7 — the data of the test in
// plain phrases (test_human.go) — anew on every request and is not stored.

// TestConclusion is what the teacher reads in the window of a result: the
// summary, the details by four parts and the recommendations.
type TestConclusion struct {
	Summary         string             `json:"summary"`
	Details         []ConclusionPart   `json:"details"`
	Recommendations []ConclusionAdvice `json:"recommendations"`
}

// ConclusionPart is one part of «Подробно».
type ConclusionPart struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// ConclusionAdvice is one recommendation: the code of the catalogue and its phrase.
type ConclusionAdvice struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

// testConclusionInput is the version of the input the conclusion is read from.
const testConclusionInput = "v7"

type testPhraseForms struct {
	One  string `json:"one"`
	Many string `json:"many"`
}

type testGroupWords struct {
	Accusative    string `json:"accusative"`
	Nominative    string `json:"nominative"`
	Prepositional string `json:"prepositional"`
	Attention     string `json:"attention"`
}

type testCase struct {
	Accusative string `json:"accusative"`
	Genitive   string `json:"genitive"`
}

// testPhraseBook mirrors the phrase book.
type testPhraseBook struct {
	Titles           map[string]string `json:"titles"`
	MotivationMethod string            `json:"motivationMethod"`
	Motivation       map[string]string `json:"motivation"`
	WillSide         string            `json:"willSide"`
	WillNames        map[string]string `json:"willNames"`
	EmotionalMethod  string            `json:"emotionalMethod"`
	Will             map[string]string `json:"will"`
	ExperienceSide   string            `json:"experienceSide"`
	Experience       []struct {
		Method string `json:"method"`
		Good   string `json:"good"`
		Poor   string `json:"poor"`
	} `json:"experience"`
	Self struct {
		Military     string            `json:"military"`
		Civil        string            `json:"civil"`
		MilitaryText map[string]string `json:"militaryText"`
		CivilText    map[string]string `json:"civilText"`
		Summary      map[string]string `json:"summary"`
	} `json:"self"`
	Relations struct {
		People       string            `json:"people"`
		Platoon      string            `json:"platoon"`
		Country      string            `json:"country"`
		Low          float64           `json:"low"`
		PeopleTop    string            `json:"peopleTop"`
		PeopleLow    string            `json:"peopleLow"`
		PlatoonTop   string            `json:"platoonTop"`
		PlatoonLow   string            `json:"platoonLow"`
		Together     string            `json:"together"`
		CountryTop   string            `json:"countryTop"`
		CountryLow   string            `json:"countryLow"`
		InDeeds      string            `json:"inDeeds"`
		Names        map[string]string `json:"names"`
		CountryVerbs map[string]string `json:"countryVerbs"`
		CountryNames map[string]string `json:"countryNames"`
	} `json:"relations"`
	Image struct {
		Congruence      map[string]string         `json:"congruence"`
		AllAtOnce       string                    `json:"allAtOnce"`
		Expected        string                    `json:"expected"`
		SeenAsIs        string                    `json:"seenAsIs"`
		SeenAsIsSense   string                    `json:"seenAsIsSense"`
		Change          string                    `json:"change"`
		StrongWeak      string                    `json:"strongWeak"`
		Groups          map[string]testGroupWords `json:"groups"`
		Achieved        string                    `json:"achieved"`
		AllAchieved     string                    `json:"allAchieved"`
		Far             string                    `json:"far"`
		AllFar          string                    `json:"allFar"`
		NotAccepted     string                    `json:"notAccepted"`
		Below           string                    `json:"below"`
		AchievedList    string                    `json:"achievedList"`
		FarList         string                    `json:"farList"`
		RestFar         string                    `json:"restFar"`
		NotAcceptedList string                    `json:"notAcceptedList"`
		BelowList       string                    `json:"belowList"`
	} `json:"image"`
	Environment struct {
		Living       string              `json:"living"`
		Training     string              `json:"training"`
		LivingText   map[string]string   `json:"livingText"`
		TrainingText map[string]string   `json:"trainingText"`
		TrainingLow  string              `json:"trainingLow"`
		Above        string              `json:"above"`
		Lacking      string              `json:"lacking"`
		Below        string              `json:"below"`
		Stories      string              `json:"stories"`
		Story        string              `json:"story"`
		StoriesMax   int                 `json:"storiesMax"`
		Ratings      map[string]string   `json:"ratings"`
		Conditions   map[string]testCase `json:"conditions"`
	} `json:"environment"`
	Summary struct {
		Reliable          string            `json:"reliable"`
		ReliableNoChoice  string            `json:"reliableNoChoice"`
		Motivated         string            `json:"motivated"`
		MotivatedTeam     string            `json:"motivatedTeam"`
		Medium            string            `json:"medium"`
		Low               string            `json:"low"`
		AllAtOnce         string            `json:"allAtOnce"`
		Strong            string            `json:"strong"`
		StrongGrowing     string            `json:"strongGrowing"`
		Tense             string            `json:"tense"`
		WeakSides         string            `json:"weakSides"`
		Weak              string            `json:"weak"`
		Example           string            `json:"example"`
		Group             string            `json:"group"`
		GroupWhy          string            `json:"groupWhy"`
		Why               map[string]string `json:"why"`
		GroupCondition    string            `json:"groupCondition"`
		GroupConditions   map[string]string `json:"groupConditions"`
		Requirements      string            `json:"requirements"`
		RequirementsTrust string            `json:"requirementsTrust"`
	} `json:"summary"`
	Advice struct {
		Support      string   `json:"support"`
		MaxGroups    int      `json:"maxGroups"`
		Fallback     []string `json:"fallback"`
		Requirements string   `json:"requirements"`
		Tense        string   `json:"tense"`
		Speciality   string   `json:"speciality"`
	} `json:"advice"`
	Recommendations map[string]testPhraseForms `json:"recommendations"`
	Platoon         struct {
		Minimum        int               `json:"minimum"`
		Motivation     map[string]string `json:"motivation"`
		Strength       string            `json:"strength"`
		StrengthPart   string            `json:"strengthPart"`
		Attention      string            `json:"attention"`
		Tense          string            `json:"tense"`
		TenseMost      string            `json:"tenseMost"`
		Speciality     string            `json:"speciality"`
		SpecialityMost string            `json:"specialityMost"`
		Lacking        string            `json:"lacking"`
		Groups         map[string]string `json:"groups"`
		MaxAdvice      int               `json:"maxAdvice"`
	} `json:"platoon"`
	Forbidden []string `json:"forbidden"`
}

var loadedTestPhrases = sync.OnceValues(func() (*testPhraseBook, error) {
	var p testPhraseBook
	if err := json.Unmarshal(formbank.TestPhrases(), &p); err != nil {
		return nil, err
	}
	return &p, nil
})

// The parts of the input of v7 the conclusion reads.
type testImageRow struct {
	Name  string `json:"название"`
	Sense string `json:"смысл"`
}

type testGroupView struct {
	Number  int      `json:"номер"`
	Now     *float64 `json:"какой_сейчас"`
	Means   string   `json:"что_значит_для_студента"`
	Type    string   `json:"тип_рекомендации"`
	Fitting []string `json:"подходящие_направления"`
}

type testRatedItem struct {
	Name   string `json:"показатель"`
	Rating string `json:"оценка"`
}

type testScoredItem struct {
	Name  string   `json:"показатель"`
	Score *float64 `json:"балл"`
}

type testView struct {
	Methods []struct {
		Side  string `json:"сторона"`
		Name  string `json:"показатель"`
		Level string `json:"уровень"`
	} `json:"методики"`
	Images struct {
		Sides  []testImageRow `json:"по_сторонам"`
		Change []testImageRow `json:"где_больше_всего_хочет_измениться"`
	} `json:"образы_я"`
	Congruence struct {
		Sense string `json:"смысл"`
	} `json:"согласованность_образов"`
	Relations []struct {
		Name  string           `json:"название"`
		Items []testScoredItem `json:"показатели"`
	} `json:"отношения"`
	Groups []testGroupView `json:"образ_защитника"`
	Self   []struct {
		Profession string `json:"профессия"`
		Means      string `json:"что_значит"`
	} `json:"самоопределение"`
	Conditions []struct {
		Items []testRatedItem `json:"показатели"`
	} `json:"условия_среды"`
	Life []struct {
		Answers []struct {
			Title string `json:"название"`
			Level string `json:"уровень_ответа"`
		} `json:"ответы_анкеты"`
	} `json:"жизнь_в_ВУЦ"`
	Environment []struct {
		Data []struct {
			Question string `json:"вопрос_анкеты"`
			Level    string `json:"уровень"`
		} `json:"данные"`
	} `json:"среда_по_главе_1"`
	Stories []struct {
		Answer string `json:"ответ"`
	} `json:"рассказы_о_деятельности"`
}

// testFacts are what the conclusion found, for the conclusion by a platoon.
type testFacts struct {
	motivation string   // high, medium, low or empty
	tense      bool     // emotional stability low beside strong will
	unsure     bool     // the military speciality is not chosen by himself
	support    []int    // groups of the image he leans on
	weakest    int      // the group he rates lowest, 0 when none
	lacking    []string // conditions rated lacking or below the others
	advice     []string // codes of his recommendations
}

var russianLevels = map[string]string{"высокий": "high", "средний": "medium", "низкий": "low"}

// testConclusionOf writes the conclusion on the data of a finished test.
func (a *Service) testConclusionOf(data *TestPortraitData) (*TestConclusion, testFacts, error) {
	p, err := loadedTestPhrases()
	if err != nil {
		return nil, testFacts{}, err
	}
	raw, err := json.Marshal(a.testPayloadFor(testConclusionInput, data))
	if err != nil {
		return nil, testFacts{}, err
	}
	var v testView
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, testFacts{}, err
	}
	w := testWriter{p: p, v: v}
	return w.conclusion(), w.facts, nil
}

// TestConclusion is the conclusion on a finished session with the methods; nil otherwise.
func (a *Service) testConclusion(data *TestPortraitData) *TestConclusion {
	if data == nil {
		return nil
	}
	c, _, err := a.testConclusionOf(data)
	if err != nil {
		return nil
	}
	return c
}

type testWriter struct {
	p     *testPhraseBook
	v     testView
	facts testFacts
}

func (w *testWriter) level(method string) string {
	for _, m := range w.v.Methods {
		if m.Name == method {
			return russianLevels[m.Level]
		}
	}
	return ""
}

// willKind is strong, medium, weak or mixed by the levels of the scales of
// «владение собой», with the plain names of the strong and the weak ones.
func (w *testWriter) willKind() (kind string, strong, weak []string, tense bool) {
	high, medium, low := 0, 0, 0
	for _, m := range w.v.Methods {
		if m.Side != w.p.WillSide {
			continue
		}
		name := cmp.Or(w.p.WillNames[m.Name], strings.ToLower(m.Name))
		switch russianLevels[m.Level] {
		case "high":
			high++
			strong = append(strong, name)
		case "medium":
			medium++
		case "low":
			low++
			weak = append(weak, name)
			if m.Name == w.p.EmotionalMethod {
				tense = true
			}
		}
	}
	switch {
	case high+medium+low == 0:
		return "", nil, nil, false
	case high > 0 && low > 0:
		return "mixed", strong, weak, tense
	case low > 0:
		return "weak", strong, weak, false
	case medium > 0:
		return "medium", strong, weak, false
	}
	return "strong", strong, weak, false
}

func (w *testWriter) experience() (good, poor []string) {
	for _, e := range w.p.Experience {
		switch w.level(e.Method) {
		case "high", "medium":
			good = append(good, e.Good)
		case "low":
			poor = append(poor, e.Poor)
		}
	}
	return good, poor
}

func (w *testWriter) self(profession string) string {
	for _, s := range w.v.Self {
		if s.Profession == profession {
			return s.Means
		}
	}
	return ""
}

func sameScore(x, y float64) bool { return math.Abs(x-y) < 0.005 }

// topItems are the indicators with the highest score; when only one has it,
// those with the next score too; at most three.
func topItems(items []testScoredItem) []testScoredItem {
	scored := slices.DeleteFunc(slices.Clone(items), func(i testScoredItem) bool { return i.Score == nil })
	if len(scored) == 0 {
		return nil
	}
	slices.SortStableFunc(scored, func(x, y testScoredItem) int { return cmp.Compare(*y.Score, *x.Score) })
	top := []testScoredItem{}
	levels := 0
	for n, item := range scored {
		if n > 0 && !sameScore(*item.Score, *scored[n-1].Score) {
			levels++
			if levels == 2 || len(top) > 1 {
				break
			}
		}
		top = append(top, item)
		if len(top) == 3 {
			break
		}
	}
	return top
}

// lowItems are the two lowest indicators at or below the threshold, lowest first.
func lowItems(items []testScoredItem, top []testScoredItem, threshold float64) []testScoredItem {
	low := []testScoredItem{}
	for _, item := range items {
		if item.Score != nil && *item.Score <= threshold+0.005 && !slices.ContainsFunc(top, func(t testScoredItem) bool { return t.Name == item.Name }) {
			low = append(low, item)
		}
	}
	slices.SortStableFunc(low, func(x, y testScoredItem) int { return cmp.Compare(*x.Score, *y.Score) })
	if len(low) > 2 {
		low = low[:2]
	}
	return low
}

func hasItem(items []testScoredItem, name string) bool {
	return slices.ContainsFunc(items, func(i testScoredItem) bool { return i.Name == name })
}

func (w *testWriter) itemNames(items []testScoredItem, names map[string]string) []string {
	out := []string{}
	for _, item := range items {
		out = append(out, cmp.Or(names[item.Name], strings.ToLower(item.Name)))
	}
	return out
}

func expressed(n int) string {
	if n == 1 {
		return "выражена"
	}
	return "выражены"
}

func (w *testWriter) values() string {
	r := w.p.Relations
	sentences := []string{}
	if text := w.p.Motivation[w.level(w.p.MotivationMethod)]; text != "" {
		sentences = append(sentences, text)
	}
	for _, relation := range w.v.Relations {
		top := topItems(relation.Items)
		if len(top) == 0 {
			continue
		}
		low := lowItems(relation.Items, top, r.Low)
		var text string
		switch relation.Name {
		case r.People:
			text = fill(r.PeopleTop, map[string]string{"top": joinAnd(w.itemNames(top, r.Names))})
			if len(low) > 0 {
				text += "; " + fill(r.PeopleLow, map[string]string{"low": joinAnd(w.itemNames(low, r.Names))})
			}
		case r.Platoon:
			text = fill(r.PlatoonTop, map[string]string{"top": joinAnd(w.itemNames(top, r.Names))})
			if len(low) > 0 {
				text += "; " + fill(r.PlatoonLow, map[string]string{"low": joinAnd(w.itemNames(low, r.Names)), "verb": expressed(len(low))})
				if hasItem(top, "Работа в коллективе") && hasItem(low, "Готовность брать ответственность") {
					text += " — " + r.Together
				}
			}
		case r.Country:
			if hasItem(top, "Личная ответственность за защиту") {
				top = slices.DeleteFunc(top, func(i testScoredItem) bool { return i.Name == "Ответственность" })
				low = slices.DeleteFunc(low, func(i testScoredItem) bool { return i.Name == "Ответственность" })
			}
			text = fill(r.CountryTop, map[string]string{"top": joinAnd(w.itemNames(top, r.CountryVerbs))})
			if len(low) > 0 {
				text += "; " + fill(r.CountryLow, map[string]string{"low": joinAnd(w.itemNames(low, r.CountryNames)), "verb": expressed(len(low))})
				if hasItem(low, "Деятельностная направленность позиции") {
					text += " — " + r.InDeeds
				}
			}
		default:
			continue
		}
		sentences = append(sentences, fullSentence(text))
	}
	return strings.Join(sentences, " ")
}

func (w *testWriter) will() string {
	sentences := []string{}
	kind, strong, weak, tense := w.willKind()
	switch kind {
	case "strong", "medium", "weak":
		sentences = append(sentences, w.p.Will[kind])
	case "mixed":
		phrase := w.p.Will["mixed"]
		if tense {
			phrase = w.p.Will["mixedTense"]
		}
		sentences = append(sentences, fullSentence(fill(phrase, map[string]string{"strong": upperFirst(joinAnd(strong)), "weak": joinAnd(weak)})))
	}
	if good, poor := w.experience(); len(good)+len(poor) > 0 {
		text := joinAnd(good)
		if len(poor) > 0 {
			text = strings.TrimPrefix(text+"; "+joinAnd(poor), "; ")
		}
		sentences = append(sentences, fullSentence(text))
	}
	military, civil := w.p.Self.MilitaryText[w.self(w.p.Self.Military)], w.p.Self.CivilText[w.self(w.p.Self.Civil)]
	switch {
	case military != "" && civil != "":
		sentences = append(sentences, fullSentence(military+"; "+civil))
	case military != "" || civil != "":
		sentences = append(sentences, fullSentence(military+civil))
	}
	return strings.Join(sentences, " ")
}

// groups are the groups of the image with a rating, by number.
func (w *testWriter) groups() []testGroupView {
	return slices.DeleteFunc(slices.Clone(w.v.Groups), func(g testGroupView) bool { return g.Now == nil })
}

// extremes are the groups rated highest and lowest; the first by number on a tie.
func (w *testWriter) extremes() (strongest, weakest testGroupView, ok bool) {
	groups := w.groups()
	if len(groups) < 2 {
		return testGroupView{}, testGroupView{}, false
	}
	strongest, weakest = groups[0], groups[0]
	for _, g := range groups[1:] {
		if *g.Now > *strongest.Now+0.005 {
			strongest = g
		}
		if *g.Now < *weakest.Now-0.005 {
			weakest = g
		}
	}
	return strongest, weakest, strongest.Number != weakest.Number
}

func (w *testWriter) groupWords(g testGroupView) testGroupWords {
	return w.p.Image.Groups[strconv.Itoa(g.Number)]
}

func (w *testWriter) image() string {
	im := w.p.Image
	sentences := []string{}
	if text := im.Congruence[w.v.Congruence.Sense]; text != "" {
		sentences = append(sentences, text)
	}
	pressure, seen := false, ""
	for _, side := range w.v.Images.Sides {
		pressure = pressure || strings.Contains(side.Sense, humanPressure)
		if seen == "" {
			for _, s := range []string{im.SeenAsIsSense, adequacyTexts["better"], adequacyTexts["worse"]} {
				if strings.Contains(side.Sense, s) {
					seen = s
				}
			}
		}
	}
	if seen == im.SeenAsIsSense {
		seen = im.SeenAsIs
	}
	switch {
	case pressure && seen != "":
		sentences = append(sentences, fullSentence(im.Expected+", и "+seen))
	case pressure:
		sentences = append(sentences, fullSentence(im.Expected))
	case seen != "":
		sentences = append(sentences, fullSentence(seen))
	}
	names := []string{}
	for _, row := range w.v.Images.Change {
		if len(names) < 4 {
			names = append(names, accusativeQuality(strings.ToLower(row.Name)))
		}
	}
	if len(names) > 0 {
		sentences = append(sentences, fill(im.Change, map[string]string{"names": joinAnd(names)}))
	}
	if strongest, weakest, ok := w.extremes(); ok {
		text := fill(im.StrongWeak, map[string]string{"strong": w.groupWords(strongest).Accusative, "weak": w.groupWords(weakest).Accusative})
		means := map[string][]testGroupView{}
		for _, g := range w.groups() {
			means[g.Means] = append(means[g.Means], g)
		}
		switch {
		case len(means) == 1 && means[im.Achieved] != nil:
			text += "; " + im.AllAchieved
		case len(means) == 1 && means[im.Far] != nil:
			text += "; " + im.AllFar
		}
		sentences = append(sentences, fullSentence(text))
		if len(means) > 1 {
			names := func(groups []testGroupView, form func(testGroupWords) string) string {
				out := []string{}
				for _, g := range groups {
					out = append(out, form(w.groupWords(g)))
				}
				return joinAnd(out)
			}
			nominative := func(g testGroupWords) string { return g.Nominative }
			if groups := means[im.Achieved]; len(groups) > 0 {
				sentences = append(sentences, upperFirst(fill(im.AchievedList, map[string]string{"names": names(groups, nominative)})))
			}
			if groups := means[im.Far]; len(groups) > 0 {
				if len(groups) >= 3 && len(means) == 2 {
					sentences = append(sentences, im.RestFar)
				} else {
					sentences = append(sentences, upperFirst(fill(im.FarList, map[string]string{"names": names(groups, nominative)})))
				}
			}
			if groups := means[im.NotAccepted]; len(groups) > 0 {
				sentences = append(sentences, upperFirst(fill(im.NotAcceptedList, map[string]string{"names": names(groups, func(g testGroupWords) string { return g.Accusative })})))
			}
			if groups := means[im.Below]; len(groups) > 0 {
				sentences = append(sentences, fill(im.BelowList, map[string]string{"names": names(groups, func(g testGroupWords) string { return g.Prepositional })}))
			}
		}
	}
	return strings.Join(sentences, " ")
}

// accusativeQuality puts the first word of a quality of module A into the
// accusative: «выдержка» → «выдержку»; the others do not change.
func accusativeQuality(name string) string {
	word, rest, _ := strings.Cut(name, " ")
	switch {
	case strings.HasSuffix(word, "а"):
		word = strings.TrimSuffix(word, "а") + "у"
	case strings.HasSuffix(word, "я"):
		word = strings.TrimSuffix(word, "я") + "ю"
	}
	if rest != "" {
		return word + " " + rest
	}
	return word
}

// rated are the conditions with the rating, in the order of the input.
func (w *testWriter) rated(rating string) []string {
	out := []string{}
	for _, level := range w.v.Conditions {
		for _, item := range level.Items {
			if item.Rating == rating {
				out = append(out, item.Name)
			}
		}
	}
	return out
}

func (w *testWriter) ratedAs(name string, ratings ...string) bool {
	for _, rating := range ratings {
		if slices.Contains(w.rated(rating), name) {
			return true
		}
	}
	return false
}

func (w *testWriter) conditionWords(names []string, genitive bool) string {
	words := []string{}
	for _, name := range names {
		c := w.p.Environment.Conditions[name]
		word := cmp.Or(c.Accusative, strings.ToLower(name))
		if genitive {
			word = cmp.Or(c.Genitive, strings.ToLower(name))
		}
		words = append(words, word)
	}
	return joinAnd(words)
}

func (w *testWriter) environment() string {
	e := w.p.Environment
	sentences := []string{}
	living, training := "", ""
	for _, sign := range w.v.Environment {
		for _, d := range sign.Data {
			if d.Question == e.Living && living == "" {
				living = russianLevels[d.Level]
			}
		}
	}
	for _, side := range w.v.Life {
		for _, answer := range side.Answers {
			if answer.Title == e.Training && training == "" {
				training = russianLevels[answer.Level]
			}
		}
	}
	first := e.LivingText[living]
	if t := e.TrainingText[training]; t != "" {
		if first != "" {
			first += ", " + t
		} else {
			first = upperFirst(t)
		}
	}
	if first != "" {
		sentences = append(sentences, fullSentence(first))
	}
	if names := w.rated(e.Ratings["above"]); len(names) > 0 {
		sentences = append(sentences, fill(e.Above, map[string]string{"list": w.conditionWords(names, false)}))
	}
	if names := w.rated(e.Ratings["lacking"]); len(names) > 0 {
		sentences = append(sentences, fill(e.Lacking, map[string]string{"list": w.conditionWords(names, true)}))
	}
	if names := w.rated(e.Ratings["below"]); len(names) > 0 {
		sentences = append(sentences, fullSentence(upperFirst(fill(e.Below, map[string]string{"list": w.conditionWords(names, false)}))))
	}
	if training == "low" {
		sentences = append(sentences, e.TrainingLow)
	}
	stories := []string{}
	for _, story := range w.v.Stories {
		text := strings.TrimRight(strings.TrimSpace(story.Answer), ".")
		if text != "" && len(stories) < e.StoriesMax {
			stories = append(stories, fill(e.Story, map[string]string{"text": text}))
		}
	}
	if len(stories) > 0 {
		sentences = append(sentences, fill(e.Stories, map[string]string{"list": strings.Join(stories, "; ")}))
	}
	return strings.Join(sentences, " ")
}

// advice picks the recommendations: one for each group to develop, weakest
// first, at most maxGroups; then one for a group he leans on. With no group to
// develop — the fallback codes.
func (w *testWriter) advice() []ConclusionAdvice {
	a := w.p.Advice
	_, _, _, tense := w.willKind()
	requirements := w.ratedAs("Ясность требований", w.p.Environment.Ratings["below"], w.p.Environment.Ratings["lacking"])
	unsure := w.self(w.p.Self.Military) != "" && w.self(w.p.Self.Military) != selfFormed.phrase
	used := []string{}
	pick := func(g testGroupView, preferred ...string) {
		for _, code := range preferred {
			if code != "" && slices.Contains(g.Fitting, code) && !slices.Contains(used, code) {
				used = append(used, code)
				return
			}
		}
		for _, code := range g.Fitting {
			if !slices.Contains(used, code) {
				used = append(used, code)
				return
			}
		}
	}
	develop, support := []testGroupView{}, []testGroupView{}
	for _, g := range w.groups() {
		if g.Type == a.Support {
			support = append(support, g)
		} else {
			develop = append(develop, g)
		}
	}
	slices.SortStableFunc(develop, func(x, y testGroupView) int { return cmp.Compare(*x.Now, *y.Now) })
	slices.SortStableFunc(support, func(x, y testGroupView) int { return cmp.Compare(*y.Now, *x.Now) })
	for n, g := range develop {
		if n == a.MaxGroups {
			break
		}
		preferred := []string{}
		if tense {
			preferred = append(preferred, a.Tense)
		}
		if requirements {
			preferred = append(preferred, a.Requirements)
		}
		if unsure {
			preferred = append(preferred, a.Speciality)
		}
		pick(g, preferred...)
	}
	if len(develop) == 0 {
		for _, code := range a.Fallback {
			if slices.ContainsFunc(support, func(g testGroupView) bool { return slices.Contains(g.Fitting, code) }) && !slices.Contains(used, code) {
				used = append(used, code)
			}
		}
	} else if len(support) > 0 {
		pick(support[0], a.Fallback[len(a.Fallback)-1])
	}
	out := []ConclusionAdvice{}
	for _, code := range used {
		if text := w.p.Recommendations[code].One; text != "" {
			out = append(out, ConclusionAdvice{Code: code, Text: text})
		}
	}
	return out
}

func (w *testWriter) summary(advice []ConclusionAdvice) string {
	s := w.p.Summary
	sentences := []string{}
	groups := w.groups()
	allSupport := len(groups) > 0 && !slices.ContainsFunc(groups, func(g testGroupView) bool { return g.Type != w.p.Advice.Support })
	_, lowest, spread := w.extremes()
	teamSupport := slices.ContainsFunc(groups, func(g testGroupView) bool { return g.Number == 3 && g.Type == w.p.Advice.Support }) && !(spread && lowest.Number == 3)
	military := w.self(w.p.Self.Military)
	switch w.level(w.p.MotivationMethod) {
	case "high":
		switch {
		case allSupport && military == selfFormed.phrase:
			sentences = append(sentences, s.Reliable)
		case allSupport:
			sentences = append(sentences, s.ReliableNoChoice)
		case teamSupport:
			sentences = append(sentences, s.MotivatedTeam)
		default:
			sentences = append(sentences, s.Motivated)
		}
	case "medium":
		sentences = append(sentences, s.Medium)
	case "low":
		sentences = append(sentences, s.Low)
	}
	if w.v.Congruence.Sense == w.p.Image.AllAtOnce {
		sentences = append(sentences, s.AllAtOnce)
	}
	kind, _, weak, tense := w.willKind()
	switch kind {
	case "strong":
		if good, poor := w.experience(); len(good) == len(w.p.Experience) && len(poor) == 0 {
			sentences = append(sentences, s.StrongGrowing)
		} else {
			sentences = append(sentences, s.Strong)
		}
	case "mixed":
		if tense {
			sentences = append(sentences, s.Tense)
		} else {
			sentences = append(sentences, fill(s.WeakSides, map[string]string{"list": joinAnd(weak)}))
		}
	case "weak":
		sentences = append(sentences, s.Weak)
	}
	if text := w.p.Self.Summary[military]; text != "" {
		sentences = append(sentences, text)
	}
	example := allSupport || w.level(w.p.MotivationMethod) == "high"
	if example && slices.ContainsFunc(advice, func(a ConclusionAdvice) bool { return a.Code == "R16" }) {
		sentences = append(sentences, s.Example)
	}
	below, lacking := w.p.Environment.Ratings["below"], w.p.Environment.Ratings["lacking"]
	requirements := w.ratedAs("Ясность требований", below, lacking)
	if attention, why, ok := w.attentionGroup(); ok && !(attention.Number == 5 && tense) && !(attention.Number == 4 && requirements) {
		words := w.groupWords(attention)
		condition := s.GroupConditions[strconv.Itoa(attention.Number)]
		switch {
		case why != "":
			sentences = append(sentences, fill(s.GroupWhy, map[string]string{"group": words.Attention, "why": why}))
		case condition != "" && w.ratedAs(condition, below):
			sentences = append(sentences, fill(s.GroupCondition, map[string]string{"group": words.Attention, "condition": w.conditionWords([]string{condition}, false)}))
		default:
			sentences = append(sentences, fill(s.Group, map[string]string{"group": words.Attention}))
		}
	}
	if requirements {
		if w.ratedAs("Доверие и уважение к нему", below, lacking) {
			sentences = append(sentences, s.RequirementsTrust)
		} else {
			sentences = append(sentences, s.Requirements)
		}
	}
	return strings.Join(sentences, " ")
}

// attentionGroup is the group the summary asks to give more attention to: the
// lowest rated of those to develop, or the lowest of all when he leans on all.
// Why is empty when it is the lowest of all — «здесь он оценивает себя
// скромнее всего»; otherwise it is said by what the group means for him.
func (w *testWriter) attentionGroup() (testGroupView, string, bool) {
	_, weakest, ok := w.extremes()
	if !ok {
		return testGroupView{}, "", false
	}
	develop := slices.DeleteFunc(w.groups(), func(g testGroupView) bool { return g.Type == w.p.Advice.Support })
	if len(develop) == 0 {
		return weakest, "", true
	}
	chosen := develop[0]
	for _, g := range develop[1:] {
		if *g.Now < *chosen.Now-0.005 {
			chosen = g
		}
	}
	if chosen.Number == weakest.Number {
		return chosen, "", true
	}
	return chosen, w.p.Summary.Why[chosen.Means], true
}

func (w *testWriter) conclusion() *TestConclusion {
	advice := w.advice()
	c := &TestConclusion{Summary: w.summary(advice), Details: []ConclusionPart{}, Recommendations: advice}
	for _, part := range []struct{ id, text string }{
		{"values", w.values()}, {"will", w.will()}, {"image", w.image()}, {"environment", w.environment()},
	} {
		if part.text != "" {
			c.Details = append(c.Details, ConclusionPart{ID: part.id, Title: w.p.Titles[part.id], Text: part.text})
		}
	}
	_, weakest, _ := w.extremes()
	_, _, _, tense := w.willKind()
	w.facts = testFacts{
		motivation: w.level(w.p.MotivationMethod),
		tense:      tense,
		unsure:     w.self(w.p.Self.Military) != "" && w.self(w.p.Self.Military) != selfFormed.phrase,
		weakest:    weakest.Number,
		lacking:    append(w.rated(w.p.Environment.Ratings["lacking"]), w.rated(w.p.Environment.Ratings["below"])...),
	}
	for _, g := range w.groups() {
		if g.Type == w.p.Advice.Support {
			w.facts.support = append(w.facts.support, g.Number)
		}
	}
	for _, a := range advice {
		w.facts.advice = append(w.facts.advice, a.Code)
	}
	return c
}
