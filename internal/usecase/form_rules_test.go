package usecase

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/formbank"
)

var today = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func autobiography(t *testing.T) model.Form {
	t.Helper()
	forms, err := formbank.Load()
	if err != nil {
		t.Fatal(err)
	}
	return forms[0]
}

func section(t *testing.T, form model.Form, id string) model.FormSection {
	t.Helper()
	for _, s := range form.Sections {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no section %s", id)
	return model.FormSection{}
}

func raw(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type obj = map[string]any

func clean(t *testing.T, id string, input obj) (map[string]json.RawMessage, error) {
	t.Helper()
	form := autobiography(t)
	in := map[string]json.RawMessage{}
	for key, value := range input {
		in[key] = raw(t, value)
	}
	return cleanSection(form, section(t, form, id), in, today)
}

func TestCleanSectionRejectsBadAnswers(t *testing.T) {
	long := strings.Repeat("я", 301)
	cases := []struct {
		name, section string
		input         obj
	}{
		{"unknown field", "about", obj{"nope": "x"}},
		{"field of another section", "about", obj{"sport_has": "no"}},
		{"option outside the list", "about", obj{"name_changed": "maybe"}},
		{"option of a list", "about", obj{"birth_country": "XX"}},
		{"wrong value type", "about", obj{"name_changed": 5}},
		{"text too long", "about", obj{"name_changed": "yes", "prev_names": long}},
		{"impossible date", "about", obj{"birth_date": "2005-02-30"}},
		{"date in the future", "about", obj{"birth_date": "2030-01-01"}},
		{"date too early", "about", obj{"birth_date": "1850-01-01"}},
		{"date wrong format", "about", obj{"birth_date": "15.03.2005"}},
		{"control characters", "about", obj{"birth_place": "а\x00б"}},
		{"group is not a list", "work", obj{"work_has": "yes", "work_list": "x"}},
		{"unknown field in a record", "work", obj{"work_has": "yes", "work_list": []obj{{"nope": "x"}}}},
	}
	for _, c := range cases {
		if _, err := clean(t, c.section, c.input); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
	records := make([]obj, 11)
	if _, err := clean(t, "work", obj{"work_has": "yes", "work_list": records}); err == nil {
		t.Error("more records than max accepted")
	}
	if _, err := clean(t, "education", obj{"education_list": []obj{{"edu_year": "1850"}}}); err == nil {
		t.Error("year before 1900 accepted")
	}
	if _, err := clean(t, "education", obj{"education_list": []obj{{"edu_year": "2090"}}}); err == nil {
		t.Error("year far in the future accepted")
	}
}

func TestCleanSectionStoresCanonicalValues(t *testing.T) {
	got, err := clean(t, "about", obj{"name_changed": "no", "birth_date": "2005-03-15", "birth_place": "  Тула  ", "birth_country": "RU", "birth_region": "71", "nationality": ""})
	if err != nil {
		t.Fatal(err)
	}
	if string(got["birth_place"]) != `"Тула"` || got["nationality"] != nil {
		t.Fatalf("not normalized: %v", got)
	}
	multi, err := clean(t, "hobbies", obj{"hobbies": []string{"chess", "chess", "reading"}})
	if err != nil || string(multi["hobbies"]) != `["chess","reading"]` {
		t.Fatalf("multi: %v %v", multi, err)
	}
}

func TestForkNoDropsDetailsAndYesRequiresThem(t *testing.T) {
	form := autobiography(t)
	got, err := clean(t, "work", obj{"work_has": "no", "work_list": []obj{{"org": "Завод"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got["work_list"] != nil {
		t.Fatal("details kept after answering no")
	}
	answers := map[string]json.RawMessage{"work_has": raw(t, "no")}
	if gaps := formMissing(form, answers)["work"]; gaps != nil {
		t.Fatalf("section with answer no is incomplete: %v", gaps)
	}
	answers["work_has"] = raw(t, "yes")
	if gaps := formMissing(form, answers)["work"]; !slices.Equal(gaps, []string{"work_list"}) {
		t.Fatalf("yes without records: %v", gaps)
	}
	answers["work_list"] = raw(t, []obj{{"org": "Завод", "still": "yes"}})
	gaps := formMissing(form, answers)["work"]
	for _, need := range []string{"work_list.0.sphere", "work_list.0.position", "work_list.0.from"} {
		if !slices.Contains(gaps, need) {
			t.Errorf("%s not reported in %v", need, gaps)
		}
	}
	if slices.Contains(gaps, "work_list.0.org") || slices.Contains(gaps, "work_list.0.to") {
		t.Errorf("filled or hidden field reported: %v", gaps)
	}
}

func TestConditionsInsideRecordsAndOverMultiAnswers(t *testing.T) {
	record := obj{"language": "angliyskiy", "degree": "fluent", "practice": []string{"exam", "courses"}, "exam": "other", "exam_other": "Свой экзамен", "exam_result": "B2", "exam_year": "2021", "school_kind": "regular", "abroad_time": "m1"}
	got, err := clean(t, "languages", obj{"lang_has": "yes", "lang_list": []obj{record}})
	if err != nil {
		t.Fatal(err)
	}
	var saved []map[string]json.RawMessage
	_ = json.Unmarshal(got["lang_list"], &saved)
	if len(saved) != 1 || saved[0]["exam_other"] == nil || saved[0]["courses_what"] != nil {
		t.Fatalf("record: %v", saved)
	}
	if saved[0]["school_kind"] != nil || saved[0]["abroad_time"] != nil {
		t.Fatalf("fields of unselected practice kept: %v", saved)
	}
	record["exam"] = "ielts"
	got, _ = clean(t, "languages", obj{"lang_has": "yes", "lang_list": []obj{record}})
	saved = nil
	_ = json.Unmarshal(got["lang_list"], &saved)
	if saved[0]["exam_other"] != nil {
		t.Fatal("exam_other kept after another exam was chosen")
	}
}

func TestSampleAnswersSatisfyEverySection(t *testing.T) {
	form := autobiography(t)
	every := map[string]bool{}
	for _, s := range form.Sections {
		every[s.ID] = true
	}
	for name, detailed := range map[string]map[string]bool{"all no": nil, "all yes": every} {
		answers, err := sampleAnswers(form, detailed, today)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if gaps := formMissing(form, answers); len(gaps) != 0 {
			t.Fatalf("%s: missing %v", name, gaps)
		}
		saved := []string{}
		for _, s := range form.Sections {
			saved = append(saved, s.ID)
		}
		progress := formProgress(form, model.FormSubmission{Answers: answers, SavedSections: saved})
		if len(progress.Complete) != 17 || progress.Total != 17 {
			t.Fatalf("%s: progress %+v", name, progress)
		}
	}
}

func TestSectionIsCompleteOnlyAfterItWasSaved(t *testing.T) {
	form := autobiography(t)
	answers, _ := sampleAnswers(form, nil, today)
	progress := formProgress(form, model.FormSubmission{Answers: answers, SavedSections: []string{"about"}})
	if !slices.Equal(progress.Complete, []string{"about"}) {
		t.Fatalf("complete: %v", progress.Complete)
	}
	empty := formProgress(form, model.FormSubmission{Answers: map[string]json.RawMessage{}, SavedSections: []string{"about"}})
	if len(empty.Complete) != 0 || len(empty.Missing["about"]) == 0 {
		t.Fatalf("saved but empty section reported complete: %+v", empty)
	}
}

func TestDeceasedRelativeNeedsOnlyNameBirthAndYearOfDeath(t *testing.T) {
	form := autobiography(t)
	relative := obj{"kind": "father", "known": "yes", "name": "Иванов Иван", "birth": "1960-05-01", "alive": "no"}
	answers := map[string]json.RawMessage{"upbringing": raw(t, "both"), "rel_has": raw(t, "yes"), "rel_list": raw(t, []obj{relative})}
	if gaps := formMissing(form, answers)["relatives"]; !slices.Equal(gaps, []string{"rel_list.0.death_year"}) {
		t.Fatalf("gaps: %v", gaps)
	}
	relative["death_year"] = "2015"
	answers["rel_list"] = raw(t, []obj{relative})
	if gaps := formMissing(form, answers)["relatives"]; gaps != nil {
		t.Fatalf("a deceased relative with a year of death is incomplete: %v", gaps)
	}
	// Everything else a student may have typed before switching to "no" is dropped.
	relative["work"], relative["crim"], relative["foreign"], relative["with_me"] = "works", "yes", "yes", "no"
	got, err := clean(t, "relatives", obj{"upbringing": "both", "rel_has": "yes", "rel_list": []obj{relative}})
	if err != nil {
		t.Fatal(err)
	}
	var saved []map[string]json.RawMessage
	_ = json.Unmarshal(got["rel_list"], &saved)
	if len(saved[0]) != 6 {
		t.Fatalf("kept fields: %v", saved[0])
	}
	relative["alive"] = "yes"
	delete(relative, "death_year")
	answers["rel_list"] = raw(t, []obj{relative})
	if gaps := formMissing(form, answers)["relatives"]; !slices.Contains(gaps, "rel_list.0.crim_year") {
		t.Fatalf("a living relative with a conviction needs details: %v", gaps)
	}
}

func TestYesInSections12To16NeedsARecordOrAnItem(t *testing.T) {
	form := autobiography(t)
	cases := []struct {
		section, fork, details string
		filled                 json.RawMessage
	}{
		{"sport", "sport_has", "sport_list", raw(t, []obj{{"sport_kind": "boks"}})},
		{"creative", "creative_has", "creative_list", raw(t, []obj{{"direction": "vocal"}})},
		{"patriotic", "patriot_has", "patriot_what", raw(t, []string{"yunarmia"})},
		{"science", "science_has", "science_forms", raw(t, []string{"olympiads"})},
		{"volunteering", "vol_has", "vol_dirs", raw(t, []string{"eco"})},
	}
	for _, c := range cases {
		answers := map[string]json.RawMessage{c.fork: raw(t, "no")}
		if gaps := formMissing(form, answers)[c.section]; gaps != nil {
			t.Errorf("%s: answer no leaves gaps %v", c.section, gaps)
		}
		answers[c.fork] = raw(t, "yes")
		if gaps := formMissing(form, answers)[c.section]; !slices.Equal(gaps, []string{c.details}) {
			t.Errorf("%s: yes without details reports %v", c.section, gaps)
		}
		answers[c.details] = c.filled
		if gaps := formMissing(form, answers)[c.section]; gaps != nil {
			t.Errorf("%s: yes with details reports %v", c.section, gaps)
		}
	}
	empty := map[string]json.RawMessage{"sport_has": raw(t, "yes"), "sport_list": raw(t, []obj{{"sport_now": "now"}})}
	if gaps := formMissing(form, empty)["sport"]; !slices.Equal(gaps, []string{"sport_list.0.sport_kind"}) {
		t.Errorf("a record without a sport reports %v", gaps)
	}
	if gaps := formMissing(form, map[string]json.RawMessage{})["hobbies"]; gaps != nil {
		t.Errorf("hobbies must stay optional: %v", gaps)
	}
}

func TestOnlySchoolRegionAcceptsAbroad(t *testing.T) {
	if _, err := clean(t, "education", obj{"education_list": []obj{{"edu_region": "abroad"}}}); err != nil {
		t.Fatalf("school abroad rejected: %v", err)
	}
	if _, err := clean(t, "about", obj{"birth_country": "RU", "birth_region": "abroad"}); err == nil {
		t.Fatal("birth region abroad accepted")
	}
	if _, err := clean(t, "address", obj{"reg_region": "abroad"}); err == nil {
		t.Fatal("registration abroad accepted")
	}
}

func TestOtherLanguageAsksForItsName(t *testing.T) {
	form := autobiography(t)
	record := func(extra obj) []obj {
		r := obj{"language": "other", "degree": "dict", "practice": []string{"native"}}
		for k, v := range extra {
			r[k] = v
		}
		return []obj{r}
	}
	answers := map[string]json.RawMessage{"lang_has": raw(t, "yes"), "lang_list": raw(t, record(nil))}
	if gaps := formMissing(form, answers)["languages"]; !slices.Equal(gaps, []string{"lang_list.0.language_other"}) {
		t.Fatalf("other language without a name: %v", gaps)
	}
	answers["lang_list"] = raw(t, record(obj{"language_other": "Хинди-урду"}))
	if gaps := formMissing(form, answers)["languages"]; gaps != nil {
		t.Fatalf("other language with a name: %v", gaps)
	}
	got, err := clean(t, "languages", obj{"lang_has": "yes", "lang_list": record(obj{"language": "angliyskiy", "language_other": "Лишнее"})})
	if err != nil {
		t.Fatal(err)
	}
	var saved []map[string]json.RawMessage
	_ = json.Unmarshal(got["lang_list"], &saved)
	if saved[0]["language_other"] != nil {
		t.Fatal("name of another language kept for a listed language")
	}
}
