package model

// Cohort is a graduating class: a named set of platoons of one center that
// receives an essay together. Not to be confused with usecase.Cohort, the
// group of sessions of the test report.
type Cohort struct {
	ID   string `json:"id"`
	VUC  string `json:"vuc"`
	Name string `json:"name"`
	Year int    `json:"year"`
	// PlatoonIDs are the platoons of the cohort; a platoon is in one cohort only.
	PlatoonIDs []string `json:"platoonIds"`
	// CreatedBy alone changes, deletes and assigns an essay to the cohort.
	CreatedBy string `json:"createdBy"`
	Created   string `json:"created"`
}
