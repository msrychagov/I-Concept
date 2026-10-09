package usecase

import (
	"path/filepath"
	"testing"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/repository/jsonfile"
)

func TestStoreRevisionAndPersistence(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "data.json")
	s, err := jsonfile.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	a := New(model.Bank{}, s)
	v := model.Session{ID: "one", Answers: map[string]int{}, Notes: map[string]string{}}
	if err := a.addSession(model.SessionRecord{TokenHash: "hash", Session: v}); err != nil {
		t.Fatal(err)
	}
	v.Answers["x"] = 1
	saved, err := a.updateSession(v, 0)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 1 {
		t.Fatal("revision not incremented")
	}
	if _, err := a.updateSession(v, 0); err == nil {
		t.Fatal("revision conflict was not detected")
	}
	s2, err := jsonfile.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := New(model.Bank{}, s2).sessionByID("one")
	if !ok || got.Answers["x"] != 1 {
		t.Fatal("data did not persist")
	}
}
