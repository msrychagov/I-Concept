package model

// DefenderImage is the image of the defender of the Fatherland of the first
// chapter: five groups of qualities with the data that describe them, the
// kinds of activity and conditions of the environment each is formed through,
// and the directions of work a recommendation is chosen from. It is not a part
// of the question bank: the questions and their version do not change
// (Вики/Тестирование/План портрета test-v2.md, decision Д3).
type DefenderImage struct {
	Groups     []DefenderGroupDef   `json:"groups"`
	Types      []RecommendationType `json:"types"`
	Directions []Direction          `json:"directions"`
	// Criteria and Signs come from environment.json: how the test covers the
	// criteria of the first chapter and the signs of positive activity and
	// environment (Вики/Тестирование/Оценка деятельности и среды.md).
	Criteria []KindCriteria    `json:"criteria,omitempty"`
	Signs    []EnvironmentSign `json:"signs,omitempty"`
}

// KindCriteria are the criteria of the first chapter of one kind of activity.
type KindCriteria struct {
	Kind     string      `json:"kind"`
	Criteria []Criterion `json:"criteria"`
}

// Criterion is one criterion with the data of the test that refer to it. Data
// are codes of methods, indicators and questions, or kinds (the levels of the
// four kinds), images, defender, signs and stories (what is in those parts of
// the input). Coverage is «есть», «частично», «по взводу» or «нет». Texts asks
// the conclusion by a platoon for the answers themselves, not their number.
type Criterion struct {
	Name     string   `json:"name"`
	Coverage string   `json:"coverage"`
	Note     string   `json:"note,omitempty"`
	Data     []string `json:"data"`
	Texts    bool     `json:"texts,omitempty"`
}

// EnvironmentSign is one of the six signs of positive activity and
// environment; the program counts no level of it, only gathers its data.
type EnvironmentSign struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Note        string   `json:"note,omitempty"`
	Data        []string `json:"data"`
}

// DefenderGroupDef is one group of the image. Qualities are the indicators
// Q01–Q24 of modules A and B; Indicators, Signs and Methods are the other data
// of the group; Kinds and Conditions are what forms it.
type DefenderGroupDef struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Qualities   []string   `json:"qualities"`
	Indicators  []string   `json:"indicators,omitempty"`
	Signs       []string   `json:"signs,omitempty"`
	Methods     []string   `json:"methods,omitempty"`
	Kinds       []KindLink `json:"kinds"`
	Conditions  []string   `json:"conditions"`
	// Note is what the test does not measure in the group.
	Note string `json:"note,omitempty"`
}

// KindLink is a kind of activity and the forms of it that form the group.
type KindLink struct {
	ID    string `json:"id"`
	Forms string `json:"forms,omitempty"`
}

type RecommendationType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Direction is one direction of the catalogue of recommendations. Edition is
// the edition of the catalogue it came in, zero for the first: an instruction
// gets the catalogue of its own edition, so the input of the older ones stays.
type Direction struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Groups  []string `json:"groups"`
	Source  string   `json:"source"`
	Edition int      `json:"edition,omitempty"`
}
