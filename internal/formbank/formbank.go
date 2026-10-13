// Package formbank loads fixed questionnaire descriptions and the lists they refer to.
package formbank

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"vucdiagnostic/internal/domain/model"
)

// Parse reads every *.json form description in the root of files and the
// reference lists in files/refs. Each form receives the lists it uses in Refs.
func Parse(files fs.FS) ([]model.Form, error) {
	names, err := fs.Glob(files, "*.json")
	if err != nil {
		return nil, err
	}
	forms := []model.Form{}
	for _, name := range names {
		var form model.Form
		if err = readJSON(files, name, &form); err != nil {
			return nil, err
		}
		if form.ID == "" {
			return nil, fmt.Errorf("%s: form has no id", name)
		}
		form.Refs = map[string][]model.FormOption{}
		for _, ref := range usedRefs(form) {
			var list []model.FormOption
			if err = readJSON(files, path.Join("refs", ref+".json"), &list); err != nil {
				return nil, fmt.Errorf("%s: %w", form.ID, err)
			}
			if len(list) == 0 {
				return nil, fmt.Errorf("%s: reference list %s is empty", form.ID, ref)
			}
			form.Refs[ref] = list
		}
		if err = Validate(form); err != nil {
			return nil, err
		}
		forms = append(forms, form)
	}
	return forms, nil
}

func readJSON(files fs.FS, name string, target any) error {
	data, err := fs.ReadFile(files, name)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func usedRefs(form model.Form) []string {
	seen := map[string]bool{}
	list := []string{}
	var walk func([]model.FormField)
	walk = func(fields []model.FormField) {
		for _, field := range fields {
			if field.Ref != "" && !seen[field.Ref] && !strings.ContainsAny(field.Ref, "/.\\") {
				seen[field.Ref] = true
				list = append(list, field.Ref)
			}
			walk(field.Fields)
		}
	}
	for _, section := range form.Sections {
		walk(section.Fields)
	}
	return list
}
