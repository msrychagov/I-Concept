package usecase

import (
	"vucdiagnostic/internal/domain/model"
)

// Service coordinates domain rules through a persistence port; it has no HTTP
// or filesystem dependencies. The question bank is immutable after construction.
type Service struct {
	bank     model.Bank
	previous map[string]model.Bank
	repo     Repository
	forms    map[string]model.Form
	// portraits is shared by the copies of the service made for old bank versions.
	portraits *portraitLayer
	ai        *aiLayer
}

func New(bank model.Bank, repo Repository, previous ...model.Bank) *Service {
	history := map[string]model.Bank{}
	for _, old := range previous {
		history[old.Version] = model.Clone(old)
	}
	return &Service{bank: model.Clone(bank), repo: repo, previous: history, forms: map[string]model.Form{}, portraits: newPortraitLayer(), ai: newAILayer()}
}

// The question wording and key of earlier sessions remain tied to their bank version.
func (a *Service) engineFor(s model.Session) *Service {
	copyService := *a
	version := s.BankVersion
	if version == "" {
		version = s.Profile.BankVersion
	}
	if old, ok := a.previous[version]; ok {
		copyService.bank = old
	}
	return copyService.forTemplate(s.Profile)
}
func (a *Service) Version() string { return a.bank.Version }
func (a *Service) Bank() model.Bank {
	bank := model.Clone(a.bank)
	bank.Items = studentQuestions(bank.Items)
	delete(bank.Modules, "E")
	delete(bank.Instructions, "E")
	return bank
}

// Retain archived bank snapshots while excluding the retired observation module.
func studentQuestions(items []model.Item) []model.Item {
	if items == nil {
		return nil
	}
	result := make([]model.Item, 0, len(items))
	for _, item := range items {
		if item.Module != "E" {
			result = append(result, item)
		}
	}
	return result
}
func clone[T any](value T) T { return model.Clone(value) }

func (a *Service) supportsVersion(version string) bool {
	_, old := a.previous[version]
	return version == a.bank.Version || old
}
