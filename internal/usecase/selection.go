package usecase

import (
	"math/rand/v2"
	"vucdiagnostic/internal/domain/model"
)

// assignmentQuestions samples once, before the assignment snapshot is saved.
// Limits are a workload policy, not a claim of psychometric validation.
func (a *Service) assignmentQuestions(p model.Profile, activity, motivation bool) []model.Item {
	engine := a.engineFor(model.Session{Profile: p, BankVersion: p.BankVersion})
	eligible := a.itemsFor(model.Session{Kind: "student", Profile: p, BankVersion: p.BankVersion, Activity: activity, Motivation: motivation})
	groups := map[string][]model.Item{}
	keys := []string{}
	selected := map[string]bool{}
	for _, item := range eligible {
		// Images R/I/Z/N and fixed motivation/activity blocks are linked sets.
		if item.Module == "A" || item.Module == "V" || item.Module == "F" {
			selected[item.ID] = true
			continue
		}
		scored := item.Scored && !item.ResponseOnly
		for _, indicator := range engine.bank.Indicators {
			if !indicator.Descriptive && contains(indicator.Items, item.ID) {
				scored = true
			}
		}
		kind := "auxiliary"
		if scored {
			kind = "scored"
		}
		if item.ResponseOnly {
			kind = "descriptive"
		}
		key := kind + ":" + item.Module + ":" + item.Indicator
		if _, exists := groups[key]; !exists {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], item)
	}
	for _, key := range keys {
		items := groups[key]
		rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
		limit := 2
		if len(key) >= 7 && key[:7] == "scored:" {
			limit = 3
		}
		// Preserve both direct and reverse wording when a scale contains both.
		count := 0
		if limit == 3 {
			for _, reverse := range []bool{false, true} {
				for _, item := range items {
					if item.Reverse == reverse {
						selected[item.ID] = true
						count++
						break
					}
				}
			}
		}
		for _, item := range items {
			if count >= limit {
				break
			}
			if !selected[item.ID] {
				selected[item.ID] = true
				count++
			}
		}
	}
	// Keep canonical snapshot order; the student's stable seed controls display.
	result := []model.Item{}
	for _, item := range eligible {
		if selected[item.ID] {
			result = append(result, item)
		}
	}
	return result
}
