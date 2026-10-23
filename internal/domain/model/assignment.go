package model

type Assignment struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Platoon    string  `json:"platoon"`
	Profile    Profile `json:"profile"`
	FormID     string  `json:"formId,omitempty"`
	Activity   bool    `json:"activity"`
	Motivation bool    `json:"motivation"`
	Methods    bool    `json:"methods,omitempty"`
	Active     bool    `json:"active"`
	Created    string  `json:"created"`
	OwnerID    string  `json:"ownerId,omitempty"`
	// CohortID is the cohort an essay is assigned to: its students are all
	// students of the platoons of the cohort, whoever invited them.
	CohortID string `json:"cohortId,omitempty"`
}
