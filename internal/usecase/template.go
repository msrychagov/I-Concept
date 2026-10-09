package usecase

import (
	"fmt"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// Each session owns a snapshot; legacy sessions continue using the original bank.
func (a *Service) questionBank(p model.Profile) []model.Item {
	if p.Questions != nil {
		return p.Questions
	}
	return a.bank.Items
}

func (a *Service) forTemplate(p model.Profile) *Service {
	if p.Questions == nil {
		return a
	}
	copyApp := *a
	copyApp.bank = a.bank
	copyApp.bank.Items = p.Questions
	copyApp.bank.Indicators = append([]model.Indicator(nil), a.bank.Indicators...)
	for j, ind := range copyApp.bank.Indicators {
		ids := []string{}
		for _, item := range p.Questions {
			if item.Indicator == ind.ID && (contains(ind.Items, item.ID) || item.Scored) {
				ids = append(ids, item.ID)
			}
		}
		copyApp.bank.Indicators[j].Items = ids
	}
	return &copyApp
}

func (a *Service) validateQuestions(p model.Profile) error {
	if p.Questions == nil {
		return nil
	}
	if len(p.Questions) == 0 || len(p.Questions) > 1000 {
		return fmt.Errorf("В шаблоне должно быть от 1 до 1000 вопросов.")
	}
	seen := map[string]bool{}
	for _, i := range p.Questions {
		if i.ID == "" || len(i.ID) > 100 || seen[i.ID] {
			return fmt.Errorf("Повторяющийся или пустой код вопроса.")
		}
		seen[i.ID] = true
		if strings.TrimSpace(i.Text) == "" || len(i.Text) > 12000 {
			return fmt.Errorf("Заполните формулировку вопроса %s.", i.ID)
		}
		if _, ok := a.bank.Modules[i.Module]; !ok {
			return fmt.Errorf("Неизвестный модуль вопроса %s.", i.ID)
		}
		if i.Section != "" {
			if _, ok := a.bank.Modules[i.Section]; !ok {
				return fmt.Errorf("Неизвестный раздел вопроса %s.", i.ID)
			}
		}
		if i.Min > i.Max || i.Min < 0 || i.Max > 10 {
			return fmt.Errorf("Некорректная шкала вопроса %s.", i.ID)
		}
		if len(i.Options) > 0 {
			if i.Min != 0 || i.Max != len(i.Options)-1 {
				return fmt.Errorf("Некорректные варианты вопроса %s.", i.ID)
			}
		}
		if i.Scored && (i.Min != 1 || i.Max != 5 || len(i.Options) > 0) {
			return fmt.Errorf("Расчётный вопрос %s должен иметь шкалу 1–5.", i.ID)
		}
	}
	effective := a.forTemplate(p)
	for _, ind := range effective.bank.Indicators {
		if _, ok := p.Values[ind.ID]; ok && len(ind.Items) == 0 {
			return fmt.Errorf("У показателя «%s» нет расчётных вопросов. Добавьте вопрос или исключите показатель.", ind.Name)
		}
	}
	return nil
}
