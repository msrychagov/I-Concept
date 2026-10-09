package model

import "time"

type Account struct {
	ID           string `json:"id"`
	Role         string `json:"role"`
	Name         string `json:"name"`
	Platoon      string `json:"platoon"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
	Salt         string `json:"salt"`
	Created      string `json:"created"`
}
type Login struct {
	Hash      string    `json:"hash"`
	StudentID string    `json:"studentId"`
	Expires   time.Time `json:"expires"`
}
