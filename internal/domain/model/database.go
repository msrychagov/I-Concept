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
	Portraits   []Portrait       `json:"portraits,omitempty"`
	Instructors []Instructor     `json:"instructors,omitempty"`
}

func (d Database) Empty() bool {
	return len(d.VUCs)+len(d.Platoons)+len(d.Students)+len(d.Assignments)+len(d.Logins)+len(d.Profiles)+len(d.Sessions)+len(d.Submissions)+len(d.Invitations)+len(d.Portraits)+len(d.Instructors) == 0
}

type VUC struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// LastInstructor is the last label number given in the instructor list.
	LastInstructor int `json:"lastInstructor,omitempty"`
}

type Platoon struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	VUC  string `json:"vuc"`
}
