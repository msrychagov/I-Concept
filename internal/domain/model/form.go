package model

import "encoding/json"

// Form describes a fixed questionnaire: sections, fields, branching and lists.
type Form struct {
	ID       string                  `json:"id"`
	Version  string                  `json:"version"`
	Title    string                  `json:"title"`
	Intro    string                  `json:"intro"`
	Sections []FormSection           `json:"sections"`
	Refs     map[string][]FormOption `json:"refs,omitempty"`
}

type FormSection struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Fields []FormField `json:"fields"`
}

// FormField is one question. Type is choice, multi, text, textarea, date,
// month, year or group (repeatable records of nested fields).
type FormField struct {
	ID        string         `json:"id"`
	Label     string         `json:"label"`
	Type      string         `json:"type"`
	Hint      string         `json:"hint,omitempty"`
	Required  bool           `json:"required,omitempty"`
	LLM       bool           `json:"llm,omitempty"`
	Suggest   bool           `json:"suggest,omitempty"`
	MaxLength int            `json:"maxLength,omitempty"`
	Options   []FormOption   `json:"options,omitempty"`
	Ref       string         `json:"ref,omitempty"`
	ShowIf    *FormCondition `json:"showIf,omitempty"`
	Min       int            `json:"min,omitempty"`
	Max       int            `json:"max,omitempty"`
	AddLabel  string         `json:"addLabel,omitempty"`
	Fields    []FormField    `json:"fields,omitempty"`
}

type FormOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// FormCondition shows a field only when the value of Field is one of In
// (for multi fields: when it shares at least one value with In).
type FormCondition struct {
	Field string   `json:"field"`
	In    []string `json:"in"`
}

// FormSubmission holds one student's answers to a form for one assignment.
// Answers are keyed by top-level field id; see usecase/forms.go for the formats.
type FormSubmission struct {
	ID            string                     `json:"id"`
	FormID        string                     `json:"formId"`
	FormVersion   string                     `json:"formVersion"`
	StudentID     string                     `json:"studentId"`
	AssignmentID  string                     `json:"assignmentId"`
	Created       string                     `json:"created"`
	Updated       string                     `json:"updated"`
	Consent       bool                       `json:"consent"`
	ConsentAt     string                     `json:"consentAt,omitempty"`
	Completed     bool                       `json:"completed"`
	CompletedAt   string                     `json:"completedAt,omitempty"`
	Revision      int                        `json:"revision"`
	SavedSections []string                   `json:"savedSections"`
	Answers       map[string]json.RawMessage `json:"answers"`
}
