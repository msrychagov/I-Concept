package formbank

import (
	"testing"
	"testing/fstest"
)

func TestParseAttachesOnlyUsedReferences(t *testing.T) {
	files := fstest.MapFS{
		"demo.json": {Data: []byte(`{"id":"demo","version":"1","sections":[{"id":"s","title":"S","fields":[
			{"id":"a","label":"A","type":"choice","ref":"colors"},
			{"id":"g","label":"G","type":"group","fields":[{"id":"b","label":"B","type":"choice","ref":"sizes"}]}]}]}`)},
		"refs/colors.json": {Data: []byte(`[{"id":"red","text":"Красный"}]`)},
		"refs/sizes.json":  {Data: []byte(`[{"id":"big","text":"Большой"}]`)},
		"refs/unused.json": {Data: []byte(`[{"id":"x","text":"X"}]`)},
	}
	forms, err := Parse(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(forms) != 1 || len(forms[0].Refs) != 2 || forms[0].Refs["colors"][0].Text != "Красный" || forms[0].Refs["sizes"][0].ID != "big" {
		t.Fatalf("unexpected result: %+v", forms)
	}
}

func TestParseRejectsMissingOrEmptyReference(t *testing.T) {
	form := `{"id":"demo","version":"1","sections":[{"id":"s","title":"S","fields":[{"id":"a","label":"A","type":"choice","ref":"colors"}]}]}`
	if _, err := Parse(fstest.MapFS{"demo.json": {Data: []byte(form)}}); err == nil {
		t.Fatal("missing reference list was accepted")
	}
	empty := fstest.MapFS{"demo.json": {Data: []byte(form)}, "refs/colors.json": {Data: []byte(`[]`)}}
	if _, err := Parse(empty); err == nil {
		t.Fatal("empty reference list was accepted")
	}
}
