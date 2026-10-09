package usecase

import "vucdiagnostic/internal/domain/model"

// Repository owns persistence and transaction isolation. A failed update must
// leave both the persisted database and subsequent snapshots unchanged.
// The callback must not call back into the repository.
type Repository interface {
	Snapshot() model.Database
	Update(func(*model.Database) error) error
	ReplaceWithBackup(model.Database) (string, error)
}
