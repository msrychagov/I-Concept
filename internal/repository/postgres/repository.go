// Package postgres persists runtime entities in PostgreSQL. Static question and
// form banks belong to questionbank/formbank and never enter this repository.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/migrations"
)

const writeLock int64 = 6421093402
const operationTimeout = 60 * time.Second

type Repository struct{ pool *pgxpool.Pool }

func New(ctx context.Context, pool *pgxpool.Pool) (*Repository, error) {
	if err := migrations.Check(ctx, pool); err != nil {
		return nil, err
	}
	return &Repository{pool: pool}, nil
}

func (r *Repository) ReadSnapshot() (model.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return model.Database{}, storageError(err)
	}
	defer tx.Rollback(context.Background())
	db, err := readSnapshot(ctx, tx)
	if err != nil {
		return model.Database{}, storageError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return model.Database{}, storageError(err)
	}
	return db, nil
}

func (r *Repository) Update(fn func(*model.Database) error) error {
	_, err := r.write(fn, false)
	return err
}

func (r *Repository) ReplaceWithBackup(next model.Database) (string, error) {
	return r.write(func(db *model.Database) error { *db = model.Clone(next); return nil }, true)
}

func (r *Repository) write(fn func(*model.Database) error, backup bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return "", storageError(err)
	}
	defer tx.Rollback(context.Background())
	// Lock before reading: multiple app processes cannot overwrite each other's
	// changes, consume the same invitation or create duplicate active sessions.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", writeLock); err != nil {
		return "", storageError(err)
	}
	before, err := readSnapshot(ctx, tx)
	if err != nil {
		return "", storageError(err)
	}
	after := model.Clone(before)
	if err = fn(&after); err != nil {
		return "", err
	}
	backupID := ""
	if backup {
		data, err := json.Marshal(before)
		if err != nil {
			return "", err
		}
		var id int64
		if err = tx.QueryRow(ctx, "INSERT INTO repository_backups(payload) VALUES ($1) RETURNING id", data).Scan(&id); err != nil {
			return "", storageError(err)
		}
		backupID = fmt.Sprintf("postgres:repository_backups:%d", id)
	}
	if err = saveSnapshot(ctx, tx, before, after); err != nil {
		return "", storageError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", storageError(err)
	}
	return backupID, nil
}

func storageError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		// PostgreSQL Detail may contain an entire row, including password hashes.
		return fmt.Errorf("database operation failed (SQLSTATE %s, constraint %s)", pg.Code, pg.ConstraintName)
	}
	return err
}
