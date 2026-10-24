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

// Direction is one direction of the catalogue of recommendations.
type Direction struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Groups []string `json:"groups"`
	Source string   `json:"source"`
}
