package model

type Bank struct {
	Instructions map[string]string `json:"instructions,omitempty"`
	Version      string            `json:"version"`
	Modules      map[string]string `json:"modules"`
	Indicators   []Indicator       `json:"indicators"`
	Items        []Item            `json:"items"`
	// Methods are validated methods with a key: their levels are counted by
	// rules, not by the mean of an indicator. ActivityKinds link the kinds of
	// activity of the first chapter to the questions that describe them.
	Methods       []Method       `json:"methods,omitempty"`
	ActivityKinds []ActivityKind `json:"activityKinds,omitempty"`
}

// Method is one scale of a method: "sum" adds the option scores of its items
// and finds the level in Levels; "level" takes the mean rank of the levels of
// the chosen options (low 1, medium 2, high 3).
type Method struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Method    string      `json:"method"`
	Component string      `json:"component"`
	Scoring   string      `json:"scoring"`
	Items     []string    `json:"items"`
	Levels    []LevelBand `json:"levels,omitempty"`
}

type LevelBand struct {
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Level string `json:"level"`
}

// ActivityKind gets its level from Items (questions whose options carry a
// level); Indicators, Frequency and Methods only describe it.
type ActivityKind struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Items      []string `json:"items"`
	Indicators []string `json:"indicators,omitempty"`
	Frequency  []string `json:"frequency,omitempty"`
	Methods    []string `json:"methods,omitempty"`
}

type Indicator struct {
	Descriptive bool     `json:"descriptive,omitempty"`
	Definition  string   `json:"definition,omitempty"`
	Limitations string   `json:"limitations,omitempty"`
	TargetText  string   `json:"targetText,omitempty"`
	Source      string   `json:"source,omitempty"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Block       string   `json:"block"`
	Component   string   `json:"component"`
	Items       []string `json:"items"`
	Tags        []string `json:"tags"`
}

type Option struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	Score     float64 `json:"score"`
	Level     string  `json:"level,omitempty"`
	Rationale string  `json:"rationale,omitempty"`
}

type Item struct {
	Hint         string   `json:"hint,omitempty"`
	Topic        string   `json:"topic,omitempty"`
	ResponseOnly bool     `json:"responseOnly,omitempty"`
	TimeWindow   string   `json:"timeWindow,omitempty"`
	KeyStatus    string   `json:"keyStatus,omitempty"`
	Section      string   `json:"section,omitempty"`
	Scored       bool     `json:"scored,omitempty"`
	ID           string   `json:"id"`
	Module       string   `json:"module"`
	Text         string   `json:"text"`
	Min          int      `json:"min"`
	Max          int      `json:"max"`
	Type         string   `json:"type"`
	Indicator    string   `json:"indicator"`
	Secondary    []string `json:"secondary,omitempty"`
	Reverse      bool     `json:"reverse,omitempty"`
	Source       string   `json:"source,omitempty"`
	Options      []Option `json:"options,omitempty"`
}
