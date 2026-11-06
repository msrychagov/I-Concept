package usecase

import (
	"slices"

	"vucdiagnostic/internal/promptbank"
)

// upTo leaves the versions of a task up to the given number, so that a test
// about an earlier version is not changed by a newer one.
func upTo(prompts []promptbank.Prompt, task string, last int) []promptbank.Prompt {
	return slices.DeleteFunc(slices.Clone(prompts), func(p promptbank.Prompt) bool {
		n, _ := promptbank.Number(p.Version)
		return p.Task == task && n > last
	})
}
