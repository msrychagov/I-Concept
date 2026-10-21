package questionbank

import (
	"embed"
	"encoding/json"
	"vucdiagnostic/internal/domain/model"
)

//go:embed bank.json pilot-2.0.json psychological.json activity.json
var files embed.FS

func Load() (model.Bank, error) {
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
	return []model.Bank{bank, recent, previous}, nil
}
