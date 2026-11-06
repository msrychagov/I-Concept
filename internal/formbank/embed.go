package formbank

import (
	"embed"
	"vucdiagnostic/internal/domain/model"
)

//go:embed autobiography.json essay.json refs/*.json
var files embed.FS

// Load returns the fixed questionnaires built into the binary.
func Load() ([]model.Form, error) { return Parse(files) }

// The phrase book of the autobiography conclusion the program writes without
// a language model. It is not a questionnaire, so it lives in its own folder
// that Parse does not read.
//
//go:embed phrases/autobiography-conclusion.json
var autobiographyPhrases []byte

// AutobiographyPhrases returns the phrase book as it is built into the binary.
func AutobiographyPhrases() []byte { return autobiographyPhrases }
