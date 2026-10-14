package model

import "time"

type Account struct {
	VUC          string `json:"vuc,omitempty"`
	PlatoonID    string `json:"platoonId,omitempty"`
	Disabled     bool   `json:"disabled,omitempty"`
	ID           string `json:"id"`
	Role         string `json:"role"`
	Name         string `json:"name"`
	Platoon      string `json:"platoon"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
	Salt         string `json:"salt"`
	Created      string `json:"created"`
	// OwnerID is the teacher who invited this student. It is empty for teachers.
	OwnerID string `json:"ownerId,omitempty"`
}
type Login struct {
	Hash      string    `json:"hash"`
	StudentID string    `json:"studentId"`
	Expires   time.Time `json:"expires"`
}

// Invitation grants teacher registration once, or student registration until expiry.
// Only a digest of the code or link token is persisted.
type Invitation struct {
	VUC       string    `json:"vuc,omitempty"`
	PlatoonID string    `json:"platoonId,omitempty"`
	Hash      string    `json:"hash"`
	Role      string    `json:"role"`
	Email     string    `json:"email"`
	Platoon   string    `json:"platoon,omitempty"`
	IssuerID  string    `json:"issuerId,omitempty"`
	Created   time.Time `json:"created"`
	Expires   time.Time `json:"expires"`
	UsedAt    time.Time `json:"usedAt,omitempty"`
	AccountID string    `json:"accountId,omitempty"`
	UseCount  int       `json:"useCount,omitempty"`
}
