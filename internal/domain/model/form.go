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
	// StudentView lets a student reread their own submitted answers (the essay);
	// the autobiography is not shown back.
	StudentView bool `json:"studentView,omitempty"`
	// Texts replace the default words of the student screens, which speak of
	// the autobiography.
	Texts *FormTexts `json:"texts,omitempty"`
}

// FormTexts are the words of the student screens that depend on the form.
type FormTexts struct {
	Consent   string `json:"consent"`
	DoneTitle string `json:"doneTitle"`
	DoneText  string `json:"doneText"`
	Review    string `json:"review"`
}

// Options returns the values a choice field can take: its own or its list's.
func (f Form) Options(field FormField) []FormOption {
	if field.Ref != "" {
		return f.Refs[field.Ref]
	}
	return field.Options
}

type FormSection struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Hint stands over the questions of the section.
	Hint   string      `json:"hint,omitempty"`
	Fields []FormField `json:"fields"`
}

// FormField is one question. Type is choice, multi, text, textarea, date,
// month, year or group (repeatable records of nested fields).
type FormField struct {
	ID string `json:"id"`
	// Number is the number of the question as the author and the teacher see it
	// («1.1»); the id cannot hold it, a dot separates group records in a path.
	Number    string         `json:"number,omitempty"`
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
