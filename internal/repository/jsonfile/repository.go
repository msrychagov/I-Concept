package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"vucdiagnostic/internal/domain/model"
)

type Repository struct {
	mu   sync.RWMutex
	path string
	db   model.Database
}

func Open(path string) (*Repository, error) {
	r := &Repository{path: path, db: model.Database{Profiles: []model.Profile{}, Sessions: []model.SessionRecord{}, Submissions: []model.FormSubmission{}}}
	data, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(data, &r.db); err != nil {
			return nil, fmt.Errorf("cannot read data file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return r, nil
}
func (r *Repository) Snapshot() model.Database {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return model.Clone(r.db)
}

func (r *Repository) ReadSnapshot() (model.Database, error) { return r.Snapshot(), nil }
func (r *Repository) Update(fn func(*model.Database) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	next := model.Clone(r.db)
	if err := fn(&next); err != nil {
		return err
	}
	if err := r.persist(next); err != nil {
		return err
	}
	r.db = next
	return nil
}
func (r *Repository) persist(next model.Database) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err = os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}
func (r *Repository) ReplaceWithBackup(next model.Database) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	backup := ""
	if data, err := os.ReadFile(r.path); err == nil {
		backup = r.path + ".backup-" + time.Now().UTC().Format("20060102-150405.000000000")
		if err = os.WriteFile(backup, data, 0600); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := r.persist(next); err != nil {
		return backup, err
	}
	r.db = model.Clone(next)
	return backup, nil
}
