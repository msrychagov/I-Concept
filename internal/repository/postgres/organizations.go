package postgres

import "vucdiagnostic/internal/domain/model"

var centers = recordSet[model.VUC]{"vucs", func(v model.VUC) string { return v.ID }}
var platoons = recordSet[model.Platoon]{"platoons", func(v model.Platoon) string { return v.ID }}
