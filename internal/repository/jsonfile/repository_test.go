package jsonfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestUpdateRollbackAndSnapshotIsolation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.json")
	repo, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Update(func(db *model.Database) error {
		db.Students = append(db.Students, model.Account{ID: "teacher", Name: "Original"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rejected := errors.New("rejected")
	err = repo.Update(func(db *model.Database) error { db.Students[0].Name = "Changed"; return rejected })
	if !errors.Is(err, rejected) {
		t.Fatalf("expected rejection, got %v", err)
	}
	snapshot := repo.Snapshot()
	snapshot.Students[0].Name = "Outside mutation"
	reopened, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	if repo.Snapshot().Students[0].Name != "Original" || reopened.Snapshot().Students[0].Name != "Original" {
		t.Fatal("failed transaction or mutable snapshot changed state")
	}
}
func TestWriteFailureDoesNotPublishChanges(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.json")
	repo, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Update(func(db *model.Database) error {
		db.Profiles = append(db.Profiles, model.Profile{ID: "original"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// A directory at the temporary-file path forces persistence to fail on all platforms.
	if err = os.Mkdir(file+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	err = repo.Update(func(db *model.Database) error { db.Profiles[0].ID = "changed"; return nil })
	if err == nil {
		t.Fatal("expected write failure")
	}
	reopened, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	if repo.Snapshot().Profiles[0].ID != "original" || reopened.Snapshot().Profiles[0].ID != "original" {
		t.Fatal("failed write published changes")
	}
}

func TestPortraitsSurviveReopen(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.json")
	repo, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	portrait := model.Portrait{ID: "key", Task: "autobiography", Input: `{"б":1,"а":2}`, Output: []byte(`{"summary":"текст"}`)}
	if err = repo.Update(func(db *model.Database) error { db.Portraits = append(db.Portraits, portrait); return nil }); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Snapshot().Portraits
	if len(got) != 1 || got[0].Input != portrait.Input || string(got[0].Output) != string(portrait.Output) {
		t.Fatalf("portrait changed: %+v", got)
	}
}
