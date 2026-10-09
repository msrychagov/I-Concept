package model

type Database struct {
	Students    []Account        `json:"students"`
	Assignments []Assignment     `json:"assignments"`
	Logins      []Login          `json:"logins"`
	Profiles    []Profile        `json:"profiles"`
	Sessions    []SessionRecord  `json:"sessions"`
	Submissions []FormSubmission `json:"submissions"`
}
