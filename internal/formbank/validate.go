package formbank

import (
	"fmt"
	"slices"
	"vucdiagnostic/internal/domain/model"
)

var fieldTypes = []string{"choice", "multi", "text", "textarea", "date", "month", "year", "group"}

// Validate checks that a form description is internally consistent: unique
// identifiers, known types, lists present, and branching that points at an
// earlier field of the same section (or of the same group record) and at a
// value the field can actually take.
func Validate(form model.Form) error {
	if len(form.Sections) == 0 {
		return fmt.Errorf("form %s has no sections", form.ID)
	}
	sections := map[string]bool{}
	for _, section := range form.Sections {
		if section.ID == "" || sections[section.ID] {
			return fmt.Errorf("form %s: empty or repeated section id %q", form.ID, section.ID)
		}
		sections[section.ID] = true
		if err := validateFields(form, section.Fields, nil, section.ID); err != nil {
			return err
		}
	}
	return nil
}

// validateFields checks one level of fields. outer holds the fields of the
// section when the level is a group record; a condition may refer to either.
func validateFields(form model.Form, fields, outer []model.FormField, where string) error {
	earlier := map[string]model.FormField{}
	for n, field := range fields {
		at := where + "/" + field.ID
		if field.ID == "" || field.Label == "" {
			return fmt.Errorf("form %s: %s has no id or label", form.ID, at)
		}
		if _, repeated := earlier[field.ID]; repeated {
			return fmt.Errorf("form %s: repeated field id %s", form.ID, at)
		}
		if !slices.Contains(fieldTypes, field.Type) {
			return fmt.Errorf("form %s: %s has unknown type %q", form.ID, at, field.Type)
		}
		if err := validateOptions(form, field, at); err != nil {
			return err
		}
		if field.Suggest && field.Type != "text" {
			return fmt.Errorf("form %s: %s has suggestions but is not a text field", form.ID, at)
		}
		if field.ShowIf != nil {
			source, ok := earlier[field.ShowIf.Field]
			if !ok {
				source, ok = lookup(outer, field.ShowIf.Field)
			}
			if !ok || len(field.ShowIf.In) == 0 {
				return fmt.Errorf("form %s: %s depends on missing or later field %q", form.ID, at, field.ShowIf.Field)
			}
			if source.Type != "choice" && source.Type != "multi" {
				return fmt.Errorf("form %s: %s depends on %s, which is not a choice", form.ID, at, source.ID)
			}
			for _, value := range field.ShowIf.In {
				if !slices.ContainsFunc(form.Options(source), func(o model.FormOption) bool { return o.ID == value }) {
					return fmt.Errorf("form %s: %s depends on value %q that %s does not have", form.ID, at, value, source.ID)
				}
			}
		}
		if field.Type == "group" {
			if len(field.Fields) == 0 || field.Max <= 0 || field.Min < 0 || field.Min > field.Max {
				return fmt.Errorf("form %s: group %s needs fields and a valid min/max", form.ID, at)
			}
			if outer != nil {
				return fmt.Errorf("form %s: group %s is nested in a group", form.ID, at)
			}
			// A record sees only the section fields declared before the group.
			if err := validateFields(form, field.Fields, fields[:n], at); err != nil {
				return err
			}
		} else if len(field.Fields) > 0 || field.Min != 0 || field.Max != 0 {
			return fmt.Errorf("form %s: %s is not a group but has group settings", form.ID, at)
		}
		earlier[field.ID] = field
	}
	return nil
}

func validateOptions(form model.Form, field model.FormField, at string) error {
	choice := field.Type == "choice" || field.Type == "multi"
	if !choice {
		if len(field.Options) > 0 || field.Ref != "" {
			return fmt.Errorf("form %s: %s has options but is not a choice", form.ID, at)
		}
		return nil
	}
	if (len(field.Options) == 0) == (field.Ref == "") {
		return fmt.Errorf("form %s: %s needs either options or a reference list", form.ID, at)
	}
	seen := map[string]bool{}
	for _, option := range form.Options(field) {
		if option.ID == "" || option.Text == "" || seen[option.ID] {
			return fmt.Errorf("form %s: %s has an empty or repeated option %q", form.ID, at, option.ID)
		}
		seen[option.ID] = true
	}
	if field.Ref != "" && len(form.Refs[field.Ref]) == 0 {
		return fmt.Errorf("form %s: %s refers to missing list %s", form.ID, at, field.Ref)
	}
	return nil
}

func lookup(fields []model.FormField, id string) (model.FormField, bool) {
	for _, field := range fields {
		if field.ID == id {
			return field, true
		}
	}
	return model.FormField{}, false
}
