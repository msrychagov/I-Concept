package usecase

import (
	"cmp"
	"encoding/json"
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
	Can        string `json:"can"`
}

// testAnswerPhrases are the phrases by the level of the answer to one
// question of the questionnaire of the activity component (module K).
type testAnswerPhrases struct {
	Question string `json:"question"`
	High     string `json:"high"`
	Medium   string `json:"medium"`
	Low      string `json:"low"`
}

func (a testAnswerPhrases) of(level string) string {
	switch level {
	case "high":
		return a.High
	case "medium":
		return a.Medium
	case "low":
		return a.Low
	}
	return ""
}

// testPhraseBook mirrors the phrase book.
type testPhraseBook struct {
	Titles           map[string]string `json:"titles"`
	MotivationMethod string            `json:"motivationMethod"`
	Motivation       map[string]string `json:"motivation"`
	WillSide         string            `json:"willSide"`
	WillNames        map[string]string `json:"willNames"`
	EmotionalMethod  string            `json:"emotionalMethod"`
	Will             struct {
		Emotional         map[string]string `json:"emotional"`
		RegulationMethods []string          `json:"regulationMethods"`
		Regulation        map[string]string `json:"regulation"`
		QualitiesMethod   string            `json:"qualitiesMethod"`
		Qualities         map[string]string `json:"qualities"`
	} `json:"will"`
	ExperienceSide string `json:"experienceSide"`
	Experience     []struct {
		Method string `json:"method"`
	} `json:"experience"`
	Practice struct {
		StudyText string              `json:"studyText"`
		Study     []testAnswerPhrases `json:"study"`
		Ready     testAnswerPhrases   `json:"ready"`
		Expect    testAnswerPhrases   `json:"expect"`
		Value     testAnswerPhrases   `json:"value"`
	} `json:"practice"`
	Self struct {
		Military        string            `json:"military"`
		Civil           string            `json:"civil"`
		MilitaryText    map[string]string `json:"militaryText"`
		CivilText       map[string]string `json:"civilText"`
		Graduate        testAnswerPhrases `json:"graduate"`
		Need            testAnswerPhrases `json:"need"`
		Use             testAnswerPhrases `json:"use"`
		UseAfterLow     string            `json:"useAfterLow"`
		Contradiction   string            `json:"contradiction"`
		Summary         map[string]string `json:"summary"`
		SummaryGraduate map[string]string `json:"summaryGraduate"`
	} `json:"self"`
	Relations struct {
		People           string            `json:"people"`
		Platoon          string            `json:"platoon"`
		Country          string            `json:"country"`
		Gap              float64           `json:"gap"`
		High             float64           `json:"high"`
		Low              float64           `json:"low"`
		Max              int               `json:"max"`
		MaxWeak          int               `json:"maxWeak"`
		PeopleStrong     map[string]string `json:"peopleStrong"`
		PeopleWeak       map[string]string `json:"peopleWeak"`
		PeopleNames      map[string]string `json:"peopleNames"`
		PeopleStrongText string            `json:"peopleStrongText"`
		PeopleWeakText   string            `json:"peopleWeakText"`
		PeopleEvenText   string            `json:"peopleEvenText"`
		PeopleAllEven    string            `json:"peopleAllEven"`
		PeopleAllHigh    string            `json:"peopleAllHigh"`
		PeopleAllLow     string            `json:"peopleAllLow"`
		Teachers         struct {
			Activity      testAnswerPhrases `json:"activity"`
			Opening       string            `json:"opening"`
			Tones         map[string]string `json:"tones"`
			Trust         string            `json:"trust"`
			Attention     string            `json:"attention"`
			Example       string            `json:"example"`
			TrustText     map[string]string `json:"trustText"`
			AttentionText map[string]string `json:"attentionText"`
			ExampleText   map[string]string `json:"exampleText"`
			Lacking       string            `json:"lacking"`
		} `json:"teachers"`
		Overall           map[string]string `json:"overall"`
		Work              testAnswerPhrases `json:"work"`
		PlatoonNames      map[string]string `json:"platoonNames"`
		PlatoonStrong     string            `json:"platoonStrong"`
		PlatoonWeak       string            `json:"platoonWeak"`
		PlatoonWeakOnly   string            `json:"platoonWeakOnly"`
		Together          string            `json:"together"`
		PlatoonEven       map[string]string `json:"platoonEven"`
		CountryStrong     map[string]string `json:"countryStrong"`
		CountryWeak       map[string]string `json:"countryWeak"`
		CountryNames      map[string]string `json:"countryNames"`
		CountryOrder      []string          `json:"countryOrder"`
		CountryMiddle     string            `json:"countryMiddle"`
		CountryMiddleOne  string            `json:"countryMiddleOne"`
		CountryMiddleHigh string            `json:"countryMiddleHigh"`
		CountryEven       map[string]string `json:"countryEven"`
	} `json:"relations"`
	Image struct {
		Congruence      map[string]string         `json:"congruence"`
		Whole           string                    `json:"whole"`
		WholeEven       string                    `json:"wholeEven"`
		How             map[string]string         `json:"how"`
		Confident       float64                   `json:"confident"`
		Gap             float64                   `json:"gap"`
		AllAtOnce       string                    `json:"allAtOnce"`
		Pressure        string                    `json:"pressure"`
		Seen            map[string]string         `json:"seen"`
		SeenSense       map[string]string         `json:"seenSense"`
		Strong          string                    `json:"strong"`
		Weak            string                    `json:"weak"`
		WeakOnly        string                    `json:"weakOnly"`
		Change          string                    `json:"change"`
		ChangeOnly      string                    `json:"changeOnly"`
		Even            string                    `json:"even"`
		Groups          map[string]testGroupWords `json:"groups"`
		Achieved        string                    `json:"achieved"`
		Far             string                    `json:"far"`
		NotAccepted     string                    `json:"notAccepted"`
		Below           string                    `json:"below"`
		NotAcceptedList string                    `json:"notAcceptedList"`
		NotAcceptedAll  string                    `json:"notAcceptedAll"`
		BelowList       string                    `json:"belowList"`
	} `json:"image"`
	Environment struct {
		Kinds     map[string]string `json:"kinds"`
		AllKinds  map[string]string `json:"allKinds"`
		KindsText map[string]string `json:"kindsText"`
		Satisfied []struct {
			Question string `json:"question"`
			Name     string `json:"name"`
		} `json:"satisfied"`
		SatisfiedText map[string]string   `json:"satisfiedText"`
		Verbs         map[string]string   `json:"verbs"`
		Patriotic     testAnswerPhrases   `json:"patriotic"`
		Lacking       string              `json:"lacking"`
		Above         string              `json:"above"`
		AboveMax      int                 `json:"aboveMax"`
		LackingMax    int                 `json:"lackingMax"`
		Stories       string              `json:"stories"`
		Story         string              `json:"story"`
		StoriesMax    int                 `json:"storiesMax"`
		Ratings       map[string]string   `json:"ratings"`
		Conditions    map[string]testCase `json:"conditions"`
	} `json:"environment"`
	Summary struct {
		Reliable          string            `json:"reliable"`
		ReliableNoChoice  string            `json:"reliableNoChoice"`
		Motivated         string            `json:"motivated"`
		MotivatedTeam     string            `json:"motivatedTeam"`
		Even              string            `json:"even"`
		Medium            string            `json:"medium"`
		Low               string            `json:"low"`
		WholeEven         string            `json:"wholeEven"`
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
		Needs             string            `json:"needs"`
		Goals             map[string]string `json:"goals"`
		NeedOf            map[string]string `json:"needOf"`
		NeedsMax          int               `json:"needsMax"`
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
		Side  string `json:"сторона"`
		Level string `json:"уровень"`
	} `json:"жизнь_в_ВУЦ"`
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
	weakest    int      // the group he rates clearly lowest, 0 when none
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
	var all any
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, testFacts{}, err
	}
	w := testWriter{p: p, v: v, answers: map[string]string{}}
	w.collectAnswers(all)
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
	p       *testPhraseBook
	v       testView
	answers map[string]string // the level of the answer by the question of the questionnaire
	facts   testFacts
}

// collectAnswers finds the answers of the questionnaire of the activity
// component wherever the input has them: {вопрос_анкеты, уровень} or
// {название, ответ, уровень_ответа}.
func (w *testWriter) collectAnswers(node any) {
	switch x := node.(type) {
	case map[string]any:
		question, _ := x["вопрос_анкеты"].(string)
		level, _ := x["уровень"].(string)
		if question == "" {
			if _, ok := x["ответ"]; ok {
				question, _ = x["название"].(string)
				level, _ = x["уровень_ответа"].(string)
			}
		}
		if question != "" && russianLevels[level] != "" && w.answers[question] == "" {
			w.answers[question] = russianLevels[level]
		}
		for _, value := range x {
			w.collectAnswers(value)
		}
	case []any:
		for _, value := range x {
			w.collectAnswers(value)
		}
	}
}

// answer is the phrase of the level of the answer to a question; empty when
// there is no answer.
func (w *testWriter) answer(a testAnswerPhrases) string {
	return a.of(w.answers[a.Question])
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

// experienceHigh is true when all three indicators of the experience are high.
func (w *testWriter) experienceHigh() bool {
	for _, e := range w.p.Experience {
		if w.level(e.Method) != "high" {
			return false
		}
	}
	return len(w.p.Experience) > 0
}

func (w *testWriter) self(profession string) string {
	for _, s := range w.v.Self {
		if s.Profession == profession {
			return s.Means
		}
	}
	return ""
}

// joinPhrases is «A, B и C»; when the phrase before the last has its own
// comma, the last «и» gets one too: «уважает достоинство каждого, отделяя поступок от личности,
// и учитывает…».
func joinPhrases(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	last := " и "
	if strings.Contains(items[len(items)-2], ",") {
		last = ", и "
	}
	return strings.Join(items[:len(items)-1], ", ") + last + items[len(items)-1]
}

// joinNouns is «A, B и C» always.
func joinNouns(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " и " + items[len(items)-1]
}

// standOut splits the indicators by the student's own middle: strong are at
// least gap above it, weak at least gap below, the rest are even; strong
// from the highest, weak from the lowest, at most max of each. mean is the
// middle of all scored ones.
func standOut(items []testScoredItem, gap float64, max int) (strong, weak, even []testScoredItem, mean float64) {
	scored := slices.DeleteFunc(slices.Clone(items), func(i testScoredItem) bool { return i.Score == nil })
	if len(scored) == 0 {
		return nil, nil, nil, 0
	}
	for _, item := range scored {
		mean += *item.Score
	}
	mean /= float64(len(scored))
	for _, item := range scored {
		switch {
		case *item.Score >= mean+gap-0.005:
			strong = append(strong, item)
		case *item.Score <= mean-gap+0.005:
			weak = append(weak, item)
		default:
			even = append(even, item)
		}
	}
	slices.SortStableFunc(strong, func(x, y testScoredItem) int { return cmp.Compare(*y.Score, *x.Score) })
	slices.SortStableFunc(weak, func(x, y testScoredItem) int { return cmp.Compare(*x.Score, *y.Score) })
	if len(strong) > max {
		strong = strong[:max]
	}
	if len(weak) > max {
		weak = weak[:max]
	}
	return strong, weak, even, mean
}

func hasItem(items []testScoredItem, name string) bool {
	return slices.ContainsFunc(items, func(i testScoredItem) bool { return i.Name == name })
}

func itemPhrases(items []testScoredItem, phrases map[string]string) []string {
	out := []string{}
	for _, item := range items {
		if phrase := phrases[item.Name]; phrase != "" {
			out = append(out, phrase)
		}
	}
	return out
}

// height is high, medium or low by the middle score of a list of 1–5.
func (w *testWriter) height(mean float64) string {
	switch r := w.p.Relations; {
	case mean >= r.High-0.005:
		return "high"
	case mean <= r.Low+0.005:
		return "low"
	}
	return "medium"
}

func (w *testWriter) values() string {
	sentences := []string{}
	if text := w.p.Motivation[w.level(w.p.MotivationMethod)]; text != "" {
		sentences = append(sentences, text)
	}
	r := w.p.Relations
	for _, relation := range w.v.Relations {
		switch relation.Name {
		case r.People:
			sentences = append(sentences, w.people(relation.Items)...)
		case r.Platoon:
			sentences = append(sentences, w.platoon(relation.Items)...)
		case r.Country:
			sentences = append(sentences, w.country(relation.Items)...)
		}
	}
	return strings.Join(sentences, " ")
}

// people are three or four sentences on the relations in the ВУЦ: with the
// comrades of the platoon, with the teachers, and what it says of him.
func (w *testWriter) people(items []testScoredItem) []string {
	r := w.p.Relations
	strong, weak, even, mean := standOut(items, r.Gap, r.Max)
	if mean == 0 {
		return nil
	}
	if len(weak) > r.MaxWeak {
		weak = weak[:r.MaxWeak]
	}
	sentences := []string{}
	evenNames := []string{}
	for _, name := range []string{"Отзывчивость", "Взаимопомощь", "Ответственность", "Доброта", "Порядочность", "Уважение", "Честность", "Человечность", "Эмпатия"} {
		if hasItem(even, name) && len(evenNames) < 3 {
			evenNames = append(evenNames, r.PeopleNames[name])
		}
	}
	switch {
	case len(strong) == 0 && len(weak) == 0:
		switch w.height(mean) {
		case "high":
			sentences = append(sentences, r.PeopleAllHigh)
		case "low":
			sentences = append(sentences, r.PeopleAllLow)
		}
		if len(evenNames) > 0 {
			sentences = append(sentences, fill(r.PeopleAllEven, map[string]string{"list": strings.Join(evenNames, ", ")}))
		}
	default:
		if phrases := itemPhrases(strong, r.PeopleStrong); len(phrases) > 0 {
			sentences = append(sentences, fill(r.PeopleStrongText, map[string]string{"list": joinPhrases(phrases)}))
		}
		if phrases := itemPhrases(weak, r.PeopleWeak); len(phrases) > 0 {
			sentences = append(sentences, upperFirst(fill(r.PeopleWeakText, map[string]string{"list": joinPhrases(phrases)})))
		} else if len(evenNames) > 0 {
			sentences = append(sentences, fill(r.PeopleEvenText, map[string]string{"list": strings.Join(evenNames, ", ")}))
		}
	}
	if text := w.teachers(); text != "" {
		sentences = append(sentences, text)
	}
	switch {
	case len(strong) > len(weak):
		sentences = append(sentences, r.Overall["warm"])
	case len(weak) > len(strong):
		sentences = append(sentences, r.Overall["cold"])
	case len(strong) == 0:
		sentences = append(sentences, r.Overall["even"])
	default:
		sentences = append(sentences, r.Overall["mixed"])
	}
	return sentences
}

// rating is above, equal or lacking by the rating of a condition; empty
// when the condition is not rated.
func (w *testWriter) rating(name string) string {
	e := w.p.Environment
	switch {
	case w.ratedAs(name, e.Ratings["above"]):
		return "above"
	case w.ratedAs(name, e.Ratings["equal"]):
		return "equal"
	case w.ratedAs(name, e.Ratings["lacking"], e.Ratings["below"]):
		return "lacking"
	}
	return ""
}

// teachers is the sentence on the relations with the teachers: how they
// notice him in class and how he feels the trust, the attention and the
// example of the elders.
func (w *testWriter) teachers() string {
	t := w.p.Relations.Teachers
	parts, lacking, above := []string{}, []string{}, 0
	if text := w.answer(t.Activity); text != "" {
		parts = append(parts, text)
	}
	for _, c := range []struct {
		name  string
		texts map[string]string
	}{{t.Trust, t.TrustText}, {t.Attention, t.AttentionText}, {t.Example, t.ExampleText}} {
		rating := w.rating(c.name)
		if rating == "above" {
			above++
		}
		if rating == "lacking" {
			lacking = append(lacking, w.conditionWords([]string{c.name}, true))
		} else if text := c.texts[rating]; text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 && len(lacking) == 0 {
		return ""
	}
	tone := t.Tones["even"]
	switch {
	case len(lacking) >= 2:
		tone = t.Tones["reserved"]
	case len(lacking) == 0 && above > 0:
		tone = t.Tones["trust"]
	}
	text := strings.Join(append([]string{fill(t.Opening, map[string]string{"tone": tone})}, parts...), ", ")
	if len(lacking) > 0 {
		text += ", " + fill(t.Lacking, map[string]string{"list": joinNouns(lacking)})
	}
	return fullSentence(text)
}

// platoon are two sentences: how he works in the platoon and which values of
// the collective are closest to him.
func (w *testWriter) platoon(items []testScoredItem) []string {
	r := w.p.Relations
	sentences := []string{}
	if text := w.answer(r.Work); text != "" {
		sentences = append(sentences, text)
	}
	strong, weak, _, mean := standOut(items, r.Gap, r.Max)
	if mean == 0 {
		return sentences
	}
	names := func(items []testScoredItem) string { return joinNouns(itemPhrases(items, r.PlatoonNames)) }
	var text string
	switch {
	case len(strong) > 0:
		text = fill(r.PlatoonStrong, map[string]string{"list": names(strong)})
		if len(weak) > 0 {
			text += ", " + fill(r.PlatoonWeak, map[string]string{"list": names(weak)})
		}
		if hasItem(strong, "Работа в коллективе") && hasItem(weak, "Готовность брать ответственность") {
			text += ", " + r.Together
		}
	case len(weak) > 0:
		text = upperFirst(fill(r.PlatoonWeakOnly, map[string]string{"list": names(weak)}))
	default:
		return append(sentences, r.PlatoonEven[w.height(mean)])
	}
	return append(sentences, fullSentence(text))
}

// country are the sentences on the attitude to the Fatherland by the three
// sides of patriotism of chapter 1: the value of the Fatherland, the
// responsibility for its defence and the conscientious fulfilment of duties.
func (w *testWriter) country(items []testScoredItem) []string {
	r := w.p.Relations
	strong, weak, even, mean := standOut(items, r.Gap, 2)
	if mean == 0 {
		return nil
	}
	if hasItem(strong, "Личная ответственность за защиту") {
		strong = slices.DeleteFunc(strong, func(i testScoredItem) bool { return i.Name == "Ответственность" })
		even = slices.DeleteFunc(even, func(i testScoredItem) bool { return i.Name == "Ответственность" })
	}
	if len(strong) == 0 && len(weak) == 0 {
		return []string{r.CountryEven[w.height(mean)]}
	}
	// «Патриотизм в делах» and the conscientiousness say the same from two
	// sides: when one is strong, the other is not called weak beside it.
	for _, pair := range [][2]string{{"Добросовестность", "Деятельностная направленность позиции"}, {"Деятельностная направленность позиции", "Добросовестность"}} {
		if hasItem(strong, pair[0]) {
			weak = slices.DeleteFunc(weak, func(i testScoredItem) bool { return i.Name == pair[1] })
		}
	}
	sentences := itemPhrases(strong, r.CountryStrong)
	middle, sum := []string{}, 0.0
	for _, name := range r.CountryOrder {
		for _, item := range even {
			if item.Name == name && len(middle) < 3 {
				middle = append(middle, r.CountryNames[name])
				sum += *item.Score
			}
		}
	}
	if len(middle) > 0 {
		phrase := r.CountryMiddle
		if len(middle) == 1 {
			phrase = r.CountryMiddleOne
		}
		if sum/float64(len(middle)) >= r.High-0.005 {
			phrase = r.CountryMiddleHigh
		}
		sentences = append(sentences, upperFirst(fill(phrase, map[string]string{"list": joinNouns(middle)})))
	}
	return append(sentences, itemPhrases(weak, r.CountryWeak)...)
}

func (w *testWriter) will() string {
	sentences := []string{}
	wl := w.p.Will
	if text := wl.Emotional[w.level(w.p.EmotionalMethod)]; text != "" {
		sentences = append(sentences, text)
	}
	byLevel := map[string][]string{}
	for _, method := range wl.RegulationMethods {
		if level := w.level(method); level != "" {
			byLevel[level] = append(byLevel[level], cmp.Or(w.p.WillNames[method], strings.ToLower(method)))
		}
	}
	verb := func(names []string) string { return expressed(len(names)) }
	switch {
	case len(byLevel) == 1:
		for level := range byLevel {
			sentences = append(sentences, wl.Regulation[level])
		}
	case len(byLevel["high"]) > 0:
		rest := append(slices.Clone(byLevel["medium"]), byLevel["low"]...)
		sentences = append(sentences, fullSentence(fill(wl.Regulation["mixedHigh"], map[string]string{"strong": upperFirst(joinNouns(byLevel["high"])), "verb": verb(byLevel["high"]), "weak": joinNouns(rest)})))
	case len(byLevel) > 1:
		sentences = append(sentences, fullSentence(fill(wl.Regulation["mixedLow"], map[string]string{"weak": upperFirst(joinNouns(byLevel["low"])), "verb": verb(byLevel["low"]), "strong": joinNouns(byLevel["medium"])})))
	}
	if text := wl.Qualities[w.level(wl.QualitiesMethod)]; text != "" {
		sentences = append(sentences, text)
	}
	sentences = append(sentences, w.practice()...)
	return strings.Join(append(sentences, w.choice()...), " ")
}

func expressed(n int) string {
	if n == 1 {
		return "выражена"
	}
	return "выражены"
}

var answerRank = map[string]int{"low": 1, "medium": 2, "high": 3}

// practice are the sentences on his experience in the ВУЦ by the answers of
// the questionnaire: how he studies and what the experience means for him.
func (w *testWriter) practice() []string {
	pr := w.p.Practice
	sentences := []string{}
	study := []string{}
	for _, a := range pr.Study {
		if text := w.answer(a); text != "" {
			study = append(study, text)
		}
	}
	if len(study) > 0 {
		sentences = append(sentences, fill(pr.StudyText, map[string]string{"list": joinPhrases(study)}))
	}
	clauses := []string{}
	for _, a := range []testAnswerPhrases{pr.Ready, pr.Expect} {
		if text := w.answer(a); text != "" {
			clauses = append(clauses, text)
		}
	}
	text := strings.Join(clauses, ", и ")
	if value := w.answer(pr.Value); value != "" {
		best := max(answerRank[w.answers[pr.Ready.Question]], answerRank[w.answers[pr.Expect.Question]])
		switch {
		case text == "":
			text = value
		case answerRank[w.answers[pr.Value.Question]] < best:
			text += ", но " + value
		default:
			text += ", а " + value
		}
	}
	if text != "" {
		sentences = append(sentences, fullSentence(text))
	}
	return sentences
}

// choice are the sentences on the military speciality — chosen consciously
// or not — and on how he links the military training with his civil work.
func (w *testWriter) choice() []string {
	s := w.p.Self
	sentences := []string{}
	if text := s.MilitaryText[w.self(s.Military)]; text != "" {
		sentences = append(sentences, text)
	}
	civil := s.CivilText[w.self(s.Civil)]
	if graduate := w.answer(s.Graduate); civil != "" && graduate != "" {
		sentences = append(sentences, fullSentence(civil+", "+graduate))
	} else if civil != "" {
		sentences = append(sentences, fullSentence(civil))
	}
	need, use := w.answers[s.Need.Question], w.answers[s.Use.Question]
	switch {
	case need == "high" && use == "low":
		sentences = append(sentences, s.Contradiction)
	case need != "" && use != "":
		phrase := s.Use.of(use)
		if need == "low" && use == "high" {
			phrase = s.UseAfterLow
		}
		sentences = append(sentences, fullSentence(s.Need.of(need)+", "+phrase))
	case need != "":
		sentences = append(sentences, fullSentence(s.Need.of(need)))
	}
	return sentences
}

// groups are the groups of the image with a rating, by number.
func (w *testWriter) groups() []testGroupView {
	return slices.DeleteFunc(slices.Clone(w.v.Groups), func(g testGroupView) bool { return g.Now == nil })
}

// extremes are the group rated clearly highest and the one rated clearly
// lowest: at least the gap of the phrase book above or below the middle of
// his ratings; the first by number on a tie.
func (w *testWriter) extremes() (strongest testGroupView, hasStrong bool, weakest testGroupView, hasWeak bool, mean float64) {
	groups := w.groups()
	if len(groups) < 2 {
		return testGroupView{}, false, testGroupView{}, false, 0
	}
	strongest, weakest = groups[0], groups[0]
	for _, g := range groups {
		mean += *g.Now
		if *g.Now > *strongest.Now+0.005 {
			strongest = g
		}
		if *g.Now < *weakest.Now-0.005 {
			weakest = g
		}
	}
	mean /= float64(len(groups))
	gap := w.p.Image.Gap
	return strongest, *strongest.Now >= mean+gap-0.005, weakest, *weakest.Now <= mean-gap+0.005, mean
}

func (w *testWriter) groupWords(g testGroupView) testGroupWords {
	return w.p.Image.Groups[strconv.Itoa(g.Number)]
}

// even is true when no group of the image stands out.
func (w *testWriter) even() bool {
	_, hasStrong, _, hasWeak, mean := w.extremes()
	return mean > 0 && !hasStrong && !hasWeak
}

func (w *testWriter) image() string {
	im := w.p.Image
	sentences := []string{}
	_, _, _, _, mean := w.extremes()
	if w.v.Congruence.Sense == im.Whole && w.even() {
		how := im.How["low"]
		if mean >= im.Confident-0.005 {
			how = im.How["high"]
		}
		sentences = append(sentences, fill(im.WholeEven, map[string]string{"how": how}))
	} else if text := im.Congruence[w.v.Congruence.Sense]; text != "" {
		sentences = append(sentences, text)
	}
	pressure, seen := false, ""
	for _, side := range w.v.Images.Sides {
		pressure = pressure || strings.Contains(side.Sense, humanPressure)
		if seen == "" {
			for kind, sense := range im.SeenSense {
				if strings.Contains(side.Sense, sense) {
					seen = im.Seen[kind]
				}
			}
		}
	}
	about := []string{}
	if pressure {
		about = append(about, im.Pressure)
	}
	if seen != "" {
		about = append(about, seen)
	}
	if len(about) > 0 {
		sentences = append(sentences, fullSentence("Он "+joinPhrases(about)))
	}
	names := []string{}
	for _, row := range w.v.Images.Change {
		if len(names) < 4 {
			names = append(names, accusativeQuality(strings.ToLower(row.Name)))
		}
	}
	strongest, hasStrong, weakest, hasWeak, _ := w.extremes()
	parts := []string{}
	if hasStrong {
		parts = append(parts, fill(im.Strong, map[string]string{"group": w.groupWords(strongest).Prepositional}))
	}
	if hasWeak {
		phrase := im.Weak
		if !hasStrong {
			phrase = im.WeakOnly
		}
		parts = append(parts, fill(phrase, map[string]string{"group": w.groupWords(weakest).Accusative}))
	}
	switch {
	case len(parts) > 0:
		if len(names) > 0 {
			parts = append(parts, fill(im.Change, map[string]string{"names": joinNouns(names)}))
		}
		sentences = append(sentences, fullSentence(strings.Join(parts, ", ")))
	case mean > 0:
		sentences = append(sentences, im.Even)
		if len(names) > 0 {
			sentences = append(sentences, fill(im.ChangeOnly, map[string]string{"names": joinNouns(names)}))
		}
	}
	means := map[string][]testGroupView{}
	for _, g := range w.groups() {
		means[g.Means] = append(means[g.Means], g)
	}
	list := func(groups []testGroupView, form func(testGroupWords) string) string {
		out := []string{}
		for _, g := range groups {
			out = append(out, form(w.groupWords(g)))
		}
		return joinNouns(out)
	}
	if groups := means[im.NotAccepted]; len(groups) > 0 && len(groups) == len(w.groups()) && len(groups) > 1 {
		sentences = append(sentences, im.NotAcceptedAll)
	} else if len(groups) > 0 {
		sentences = append(sentences, upperFirst(fill(im.NotAcceptedList, map[string]string{"names": list(groups, func(g testGroupWords) string { return g.Accusative })})))
	}
	if groups := means[im.Below]; len(groups) > 0 {
		sentences = append(sentences, fill(im.BelowList, map[string]string{"names": list(groups, func(g testGroupWords) string { return g.Prepositional })}))
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
	return joinNouns(words)
}

// environment is one paragraph on how he feels in the ВУЦ: in the four kinds
// of activity of chapter 1, what of it satisfies him, how far he is from the
// ideal image of the student — the conditions he lacks — and what he values.
func (w *testWriter) environment() string {
	e := w.p.Environment
	sentences := []string{}
	byLevel, kinds := map[string][]string{}, 0
	for _, side := range w.v.Life {
		if kind, level := e.Kinds[side.Side], russianLevels[side.Level]; kind != "" && level != "" {
			byLevel[level] = append(byLevel[level], kind)
			kinds++
		}
	}
	if len(byLevel) == 1 && kinds == len(e.Kinds) {
		for level := range byLevel {
			sentences = append(sentences, e.AllKinds[level])
		}
	} else {
		for _, level := range []string{"high", "medium", "low"} {
			if names := byLevel[level]; len(names) > 0 {
				sentences = append(sentences, fill(e.KindsText[level], map[string]string{"list": joinNouns(names)}))
			}
		}
	}
	clauses := []string{}
	for _, level := range []string{"high", "medium", "low"} {
		names := []string{}
		for _, s := range e.Satisfied {
			if w.answers[s.Question] == level {
				names = append(names, s.Name)
			}
		}
		if len(names) == 0 {
			continue
		}
		verb := e.Verbs["one"]
		if len(names) > 1 {
			verb = e.Verbs["many"]
		}
		clauses = append(clauses, fill(e.SatisfiedText[level], map[string]string{"list": joinNouns(names), "verb": verb}))
	}
	text := strings.Join(clauses, ", ")
	if patriotic := w.answer(e.Patriotic); patriotic != "" {
		if text == "" {
			text = patriotic
		} else {
			text += ", а " + patriotic
		}
	}
	if text != "" {
		sentences = append(sentences, fullSentence(text))
	}
	lacking := append(w.rated(e.Ratings["lacking"]), w.rated(e.Ratings["below"])...)
	if len(lacking) > e.LackingMax {
		lacking = lacking[:e.LackingMax]
	}
	if len(lacking) > 0 {
		sentences = append(sentences, fill(e.Lacking, map[string]string{"list": w.conditionWords(lacking, true)}))
	}
	can := []string{}
	for _, name := range w.rated(e.Ratings["above"]) {
		if phrase := e.Conditions[name].Can; phrase != "" && len(can) < e.AboveMax {
			can = append(can, phrase)
		}
	}
	if len(can) > 0 {
		sentences = append(sentences, fill(e.Above, map[string]string{"list": joinPhrases(can)}))
	}
	stories := []string{}
	for _, story := range w.v.Stories {
		text := strings.TrimRight(strings.TrimSpace(story.Answer), ".")
		if text != "" && len(stories) < e.StoriesMax {
			stories = append(stories, fill(e.Story, map[string]string{"text": text}))
		}
	}
	if len(stories) > 0 {
		sentences = append(sentences, fill(e.Stories, map[string]string{"list": joinNouns(stories)}))
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
	_, _, lowest, hasWeak, _ := w.extremes()
	teamSupport := slices.ContainsFunc(groups, func(g testGroupView) bool { return g.Number == 3 && g.Type == w.p.Advice.Support }) && !(hasWeak && lowest.Number == 3)
	military := w.self(w.p.Self.Military)
	motivation := w.level(w.p.MotivationMethod)
	if motivation != "high" && w.even() && w.level(w.p.EmotionalMethod) == "high" {
		sentences = append(sentences, s.Even)
	}
	switch motivation {
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
	switch {
	case w.v.Congruence.Sense == w.p.Image.AllAtOnce:
		sentences = append(sentences, s.AllAtOnce)
	case w.v.Congruence.Sense == w.p.Image.Whole && w.even():
		sentences = append(sentences, s.WholeEven)
	}
	kind, _, weak, tense := w.willKind()
	switch kind {
	case "strong":
		if w.experienceHigh() {
			sentences = append(sentences, s.StrongGrowing)
		} else {
			sentences = append(sentences, s.Strong)
		}
	case "mixed":
		if tense {
			sentences = append(sentences, s.Tense)
		} else {
			sentences = append(sentences, fill(s.WeakSides, map[string]string{"list": joinNouns(weak)}))
		}
	case "weak":
		sentences = append(sentences, s.Weak)
	}
	if text := w.p.Self.Summary[military]; text != "" {
		if military == selfFormed.phrase {
			if graduate := w.p.Self.SummaryGraduate[w.answers[w.p.Self.Graduate.Question]]; graduate != "" {
				text += ", " + graduate
			}
		}
		sentences = append(sentences, fullSentence(text))
	}
	if motivation == "high" && slices.ContainsFunc(advice, func(a ConclusionAdvice) bool { return a.Code == "R16" }) {
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
	needs := []string{}
	for _, name := range append(w.rated(below), w.rated(lacking)...) {
		if requirements && name == "Доверие и уважение к нему" {
			continue
		}
		if need := s.NeedOf[name]; need != "" && !slices.Contains(needs, need) && len(needs) < s.NeedsMax {
			needs = append(needs, need)
		}
	}
	if goal := s.Goals[motivation]; goal != "" && len(needs) > 0 {
		sentences = append(sentences, fill(s.Needs, map[string]string{"goal": goal, "list": joinPhrases(needs)}))
	}
	return strings.Join(sentences, " ")
}

// attentionGroup is the group the summary asks to give more attention to: the
// lowest rated of those to develop, or the lowest of all when he leans on all.
// Why is empty when it is the lowest of all — «здесь он оценивает себя
// скромнее всего»; otherwise it is said by what the group means for him.
func (w *testWriter) attentionGroup() (testGroupView, string, bool) {
	_, _, weakest, ok, _ := w.extremes()
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
	_, _, weakest, hasWeak, _ := w.extremes()
	if !hasWeak {
		weakest = testGroupView{}
	}
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
