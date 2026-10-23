package formbank

import (
	"embed"
	"vucdiagnostic/internal/domain/model"
)

//go:embed autobiography.json essay.json refs/*.json
var files embed.FS

// Load returns the fixed questionnaires built into the binary.
func Load() ([]model.Form, error) { return Parse(files) }
