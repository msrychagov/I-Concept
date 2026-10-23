package model

// Instructor is a teacher of a military training center as students name them
// in texts. It is not an account: some instructors never register.
type Instructor struct {
	ID         string `json:"id"`
	VUC        string `json:"vuc"`
	Surname    string `json:"surname"`
	Name       string `json:"name"`
	Patronymic string `json:"patronymic,omitempty"`
	Rank       string `json:"rank,omitempty"`
	// UpdatedBy is the account that saved the entry last; Updated is when.
	UpdatedBy string `json:"updatedBy,omitempty"`
	Updated   string `json:"updated,omitempty"`
}
