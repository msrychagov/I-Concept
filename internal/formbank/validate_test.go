package formbank

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestValidateRejectsBrokenDescriptions(t *testing.T) {
	fork := model.FormField{ID: "a", Label: "A", Type: "choice", Options: []model.FormOption{{ID: "no", Text: "Нет"}, {ID: "yes", Text: "Да"}}}
	build := func(fields ...model.FormField) model.Form {
		return model.Form{ID: "t", Sections: []model.FormSection{{ID: "s", Title: "S", Fields: fields}}}
	}
	dependent := func(field string, values ...string) model.FormField {
		return model.FormField{ID: "b", Label: "B", Type: "text", ShowIf: &model.FormCondition{Field: field, In: values}}
	}
	if err := Validate(build(fork, dependent("a", "yes"))); err != nil {
		t.Fatalf("valid form rejected: %v", err)
	}
	cases := map[string]model.Form{
		"unknown field":        build(fork, dependent("zzz", "yes")),
		"later field":          build(dependent("a", "yes"), fork),
		"unknown value":        build(fork, dependent("a", "maybe")),
		"no options":           build(model.FormField{ID: "a", Label: "A", Type: "choice"}),
		"repeated id":          build(fork, fork),
		"unknown type":         build(model.FormField{ID: "a", Label: "A", Type: "number"}),
		"suggest on choice":    build(model.FormField{ID: "a", Label: "A", Type: "choice", Suggest: true, Options: fork.Options}),
		"group without limits": build(model.FormField{ID: "g", Label: "G", Type: "group", Fields: []model.FormField{fork}}),
		"nested group": build(model.FormField{ID: "g", Label: "G", Type: "group", Max: 2, Fields: []model.FormField{
			{ID: "h", Label: "H", Type: "group", Max: 2, Fields: []model.FormField{fork}}}}),
	}
	for name, form := range cases {
		if Validate(form) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	record := build(model.FormField{ID: "g", Label: "G", Type: "group", Max: 2, Fields: []model.FormField{fork, dependent("a", "yes")}})
	if err := Validate(record); err != nil {
		t.Fatalf("condition inside a record rejected: %v", err)
	}
	outside := build(fork, model.FormField{ID: "g", Label: "G", Type: "group", Max: 2, Fields: []model.FormField{dependent("a", "yes")}})
	if err := Validate(outside); err != nil {
		t.Fatalf("record condition on a section field rejected: %v", err)
	}
}
