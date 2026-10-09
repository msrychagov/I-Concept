package model

type Assignment struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Platoon    string  `json:"platoon"`
	Profile    Profile `json:"profile"`
	Activity   bool    `json:"activity"`
	Motivation bool    `json:"motivation"`
	Active     bool    `json:"active"`
	Created    string  `json:"created"`
}
