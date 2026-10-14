package postgres

import "vucdiagnostic/internal/domain/model"

var accounts = recordSet[model.Account]{"accounts", func(v model.Account) string { return v.ID }}
var logins = recordSet[model.Login]{"logins", func(v model.Login) string { return v.Hash }}
var invitations = recordSet[model.Invitation]{"invitations", func(v model.Invitation) string { return v.Hash }}
