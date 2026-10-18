package postgres

import "vucdiagnostic/internal/domain/model"

var profiles = recordSet[model.Profile]{"profiles", func(v model.Profile) string { return v.ID }}
var assignments = recordSet[model.Assignment]{"assignments", func(v model.Assignment) string { return v.ID }}
var sessions = recordSet[model.SessionRecord]{"sessions", func(v model.SessionRecord) string { return v.Session.ID }}
var submissions = recordSet[model.FormSubmission]{"form_submissions", func(v model.FormSubmission) string { return v.ID }}
var portraits = recordSet[model.Portrait]{"portraits", func(v model.Portrait) string { return v.ID }}
