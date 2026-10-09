package questionbank

import (
	"embed"
	"encoding/json"
	"vucdiagnostic/internal/domain/model"
)

//go:embed bank.json pilot-2.0.json
var files embed.FS

func Load() (model.Bank, error) {
	data, err := files.ReadFile("bank.json")
	if err != nil {
		return model.Bank{}, err
	}
	var bank model.Bank
	err = json.Unmarshal(data, &bank)
	return bank, err
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
	return []model.Bank{bank}, nil
}
