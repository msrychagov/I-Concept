package model

type Assignment struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Platoon    string  `json:"platoon"`
	Profile    Profile `json:"profile"`
	FormID     string  `json:"formId,omitempty"`
	Activity   bool    `json:"activity"`
	Motivation bool    `json:"motivation"`
	Active     bool    `json:"active"`
	Created    string  `json:"created"`
	OwnerID    string  `json:"ownerId,omitempty"`
}
