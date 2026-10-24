package questionbank

import (
	"embed"
	"encoding/json"
	"vucdiagnostic/internal/domain/model"
)

//go:embed bank.json pilot-2.0.json psychological.json activity.json methods.json defender.json
var files embed.FS

// Defender returns the image of the defender of the Fatherland: the groups of
// qualities, what forms them and the catalogue of recommendations. It is kept
// apart from the bank, whose questions and version it does not change.
func Defender() (model.DefenderImage, error) {
	var image model.DefenderImage
	data, err := files.ReadFile("defender.json")
	if err != nil {
		return image, err
	}
	err = json.Unmarshal(data, &image)
	return image, err
}

// Load returns pilot-2.4: pilot-2.3 and the methods of the first chapter
// (methods.json) with their keys and levels.
func Load() (model.Bank, error) {
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
	return []model.Bank{bank, recent, previous, activity}, nil
}
