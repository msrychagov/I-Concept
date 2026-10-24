// Package promptbank holds the instructions and answer schemas of the language
// model tasks. Each task has a folder per version; a version never changes
// after it is used: a new wording is a new folder, as with the question banks.
package promptbank

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

//go:embed */*/instruction.txt */*/schema.json
var files embed.FS

// Prompt is what the model gets besides the data: the instruction and the
// JSON schema its answer must follow.
type Prompt struct {
	Task        string
	Version     string
	Instruction string
	Schema      json.RawMessage
}

// Load returns every embedded prompt, checked: the instruction is not empty
// and the schema is a JSON object.
func Load() ([]Prompt, error) { return parse(files) }

func parse(root fs.FS) ([]Prompt, error) {
	paths, err := fs.Glob(root, "*/*/instruction.txt")
	if err != nil {
		return nil, err
	}
	prompts := []Prompt{}
	for _, path := range paths {
		parts := strings.Split(path, "/")
		prompt := Prompt{Task: parts[0], Version: parts[1]}
		name := prompt.Task + "/" + prompt.Version
		if _, ok := versionNumber(prompt.Version); !ok {
			return nil, fmt.Errorf("%s: the folder of a version is named v1, v2, …", name)
		}
		text, err := fs.ReadFile(root, path)
		if err != nil {
			return nil, err
		}
		if prompt.Instruction = strings.TrimSpace(string(text)); prompt.Instruction == "" {
			return nil, fmt.Errorf("%s: instruction is empty", name)
		}
		schema, err := fs.ReadFile(root, name+"/schema.json")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		var object map[string]any
		if err = json.Unmarshal(schema, &object); err != nil || len(object) == 0 {
			return nil, fmt.Errorf("%s: schema is not a JSON object", name)
		}
		prompt.Schema = json.RawMessage(strings.TrimSpace(string(schema)))
		prompts = append(prompts, prompt)
	}
	return prompts, nil
}

// Find returns the prompt of a task version.
func Find(prompts []Prompt, task, version string) (Prompt, bool) {
	for _, prompt := range prompts {
		if prompt.Task == task && prompt.Version == version {
			return prompt, true
		}
	}
	return Prompt{}, false
}

// Number reads the number of a version named v1, v2, …: the tasks build their
// input by it.
func Number(version string) (int, bool) { return versionNumber(version) }

// versionNumber reads the number of a version named v1, v2, …
func versionNumber(version string) (int, bool) {
	digits, found := strings.CutPrefix(version, "v")
	n, err := strconv.Atoi(digits)
	return n, found && err == nil && n > 0 && digits == strconv.Itoa(n)
}

// Latest returns the newest version of a task: the greatest number after "v",
// so v10 is newer than v2.
func Latest(prompts []Prompt, task string) (Prompt, bool) {
	var newest Prompt
	best := 0
	for _, prompt := range prompts {
		if n, ok := versionNumber(prompt.Version); ok && prompt.Task == task && n > best {
			newest, best = prompt, n
		}
	}
	return newest, best > 0
}
