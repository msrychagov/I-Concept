package questionbank

import (
	"embed"
	"encoding/json"
	"vucdiagnostic/internal/domain/model"
)

//go:embed bank.json pilot-2.0.json psychological.json activity.json methods.json identity.json defender.json environment.json
var files embed.FS

// Defender returns the tables of the portrait by the test: the image of the
// defender of the Fatherland (defender.json) — the groups of qualities, what
// forms them and the catalogue of recommendations — and the criteria of the
// kinds of activity and the signs of the environment (environment.json). They
// are kept apart from the bank, whose questions and version they do not change.
func Defender() (model.DefenderImage, error) {
	var image model.DefenderImage
	data, err := files.ReadFile("defender.json")
	if err != nil {
		return image, err
	}
	if err = json.Unmarshal(data, &image); err != nil {
		return image, err
	}
	data, err = files.ReadFile("environment.json")
	if err != nil {
		return image, err
	}
	var environment struct {
		Kinds []model.KindCriteria    `json:"kinds"`
		Signs []model.EnvironmentSign `json:"signs"`
	}
	if err = json.Unmarshal(data, &environment); err != nil {
		return image, err
	}
	image.Criteria, image.Signs = environment.Kinds, environment.Signs
	return image, nil
}

// Load returns pilot-2.5: pilot-2.4 and the method of the statuses of
// professional identity by А. А. Азбель in two blocks (identity.json).
func Load() (model.Bank, error) {
	bank, err := load24()
	if err != nil {
		return bank, err
	}
	data, err := files.ReadFile("identity.json")
	if err != nil {
		return bank, err
	}
	var addition model.Bank
	if err := json.Unmarshal(data, &addition); err != nil {
		return bank, err
	}
	bank.Version = addition.Version
	for key, value := range addition.Modules {
		bank.Modules[key] = value
	}
	for key, value := range addition.Instructions {
		bank.Instructions[key] = value
	}
	bank.Items = append(bank.Items, addition.Items...)
	bank.Identity = addition.Identity
	return bank, nil
}

// load24 returns pilot-2.4: pilot-2.3 and the methods of the first chapter
// (methods.json) with their keys and levels.
func load24() (model.Bank, error) {
	bank, err := load23()
	if err != nil {
		return bank, err
	}
	data, err := files.ReadFile("methods.json")
	if err != nil {
		return bank, err
	}
	var addition model.Bank
	if err := json.Unmarshal(data, &addition); err != nil {
		return bank, err
	}
	bank.Version = addition.Version
	for key, value := range addition.Modules {
		bank.Modules[key] = value
	}
	for key, value := range addition.Instructions {
		bank.Instructions[key] = value
	}
	bank.Items = append(bank.Items, addition.Items...)
	bank.Methods = addition.Methods
	bank.ActivityKinds = addition.ActivityKinds
	return bank, nil
}

func load23() (model.Bank, error) {
	bank, err := load22()
	if err != nil {
		return bank, err
	}
	data, err := files.ReadFile("activity.json")
	if err != nil {
		return bank, err
	}
	var addition model.Bank
	if err := json.Unmarshal(data, &addition); err != nil {
		return bank, err
	}
	bank.Version = "pilot-2.3"
	bank.Items = append(bank.Items, addition.Items...)
	return bank, nil
}

func load22() (model.Bank, error) {
	data, err := files.ReadFile("bank.json")
	if err != nil {
		return model.Bank{}, err
	}
	var bank model.Bank
	err = json.Unmarshal(data, &bank)
	if err != nil {
		return bank, err
	}
	data, err = files.ReadFile("psychological.json")
	if err != nil {
		return bank, err
	}
	var addition model.Bank
	if err = json.Unmarshal(data, &addition); err != nil {
		return bank, err
	}
	bank.Version = "pilot-2.2"
	for key, value := range addition.Modules {
		bank.Modules[key] = value
	}
	for key, value := range addition.Instructions {
		bank.Instructions[key] = value
	}
	bank.Indicators = append(bank.Indicators, addition.Indicators...)
	bank.Items = append(bank.Items, addition.Items...)
	return bank, nil
}

func Previous() ([]model.Bank, error) {
	data, err := files.ReadFile("pilot-2.0.json")
	if err != nil {
		return nil, err
	}
	var bank model.Bank
	if err = json.Unmarshal(data, &bank); err != nil {
		return nil, err
	}
	data, err = files.ReadFile("bank.json")
	if err != nil {
		return nil, err
	}
	var recent model.Bank
	if err = json.Unmarshal(data, &recent); err != nil {
		return nil, err
	}
	previous, err := load22()
	if err != nil {
		return nil, err
	}
	activity, err := load23()
	if err != nil {
		return nil, err
	}
	methods, err := load24()
	if err != nil {
		return nil, err
	}
	return []model.Bank{bank, recent, previous, activity, methods}, nil
}
