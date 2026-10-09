package model

type PlanItem struct {
	Indicator string `json:"indicator"`
	Action    string `json:"action"`
	Owner     string `json:"owner"`
	Due       string `json:"due"`
	Status    string `json:"status"`
	Note      string `json:"note"`
}

type Session struct {
	ObserverID         string            `json:"observerId,omitempty"`
	ObservationDate    string            `json:"observationDate,omitempty"`
	ObservationContext string            `json:"observationContext,omitempty"`
	StudentID          string            `json:"studentId,omitempty"`
	AssignmentID       string            `json:"assignmentId,omitempty"`
	ID                 string            `json:"id"`
	Kind               string            `json:"kind"`
	Participant        string            `json:"participant"`
	Group              string            `json:"group"`
	Wave               string            `json:"wave"`
	Observer           string            `json:"observer"`
	Parent             *string           `json:"parent"`
	Profile            Profile           `json:"profile"`
	BankVersion        string            `json:"bankVersion"`
	Created            string            `json:"created"`
	Updated            string            `json:"updated"`
	Seed               uint32            `json:"seed"`
	Activity           bool              `json:"activity"`
	Motivation         bool              `json:"motivation"`
	Consent            bool              `json:"consent"`
	ConsentAt          string            `json:"consentAt,omitempty"`
	Completed          bool              `json:"completed"`
	CompletedPartial   bool              `json:"completedPartial,omitempty"`
	Answers            map[string]int    `json:"answers"`
	Notes              map[string]string `json:"notes"`
	Plan               []PlanItem        `json:"plan"`
	Revision           int               `json:"revision"`
	Revoked            bool              `json:"revoked"`
}

type SessionRecord struct {
	TokenHash string  `json:"tokenHash"`
	Session   Session `json:"session"`
}
