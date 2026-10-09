package model

type Bank struct {
	Instructions map[string]string `json:"instructions,omitempty"`
	Version      string            `json:"version"`
	Modules      map[string]string `json:"modules"`
	Indicators   []Indicator       `json:"indicators"`
	Items        []Item            `json:"items"`
}

type Indicator struct {
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
	KeyStatus string   `json:"keyStatus,omitempty"`
	Section   string   `json:"section,omitempty"`
	Scored    bool     `json:"scored,omitempty"`
	ID        string   `json:"id"`
	Module    string   `json:"module"`
	Text      string   `json:"text"`
	Min       int      `json:"min"`
	Max       int      `json:"max"`
	Type      string   `json:"type"`
	Indicator string   `json:"indicator"`
	Secondary []string `json:"secondary,omitempty"`
	Reverse   bool     `json:"reverse,omitempty"`
	Source    string   `json:"source,omitempty"`
	Options   []Option `json:"options,omitempty"`
}
