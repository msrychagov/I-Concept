package model

type Database struct {
	VUCs        []VUC            `json:"vucs,omitempty"`
	Platoons    []Platoon        `json:"platoons,omitempty"`
	Students    []Account        `json:"students"`
	Assignments []Assignment     `json:"assignments"`
	Logins      []Login          `json:"logins"`
	Profiles    []Profile        `json:"profiles"`
	Sessions    []SessionRecord  `json:"sessions"`
	Submissions []FormSubmission `json:"submissions"`
	Invitations []Invitation     `json:"invitations,omitempty"`
}

type VUC struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Platoon struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	VUC  string `json:"vuc"`
}
