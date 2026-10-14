// Package migrations embeds and applies versioned SQL atomically under a lock.
package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var files embed.FS

const lockID int64 = 6421093401

func Up(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	entries, err := files.ReadDir("sql")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		sql, err := files.ReadFile("sql/" + entry.Name())
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(sql))
		version := strings.TrimSuffix(entry.Name(), ".up.sql")
		var existing string
		err = tx.QueryRow(ctx, "SELECT checksum FROM schema_migrations WHERE version=$1", version).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration %s was modified after applying", version)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s: %w", version, err)
		}
		if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version,checksum) VALUES ($1,$2)", version, checksum); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Check prevents the application from starting against an incomplete schema.
func Check(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := files.ReadDir("sql")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		sql, err := files.ReadFile("sql/" + entry.Name())
		if err != nil {
			return err
		}
		var checksum string
		err = pool.QueryRow(ctx, "SELECT checksum FROM schema_migrations WHERE version=$1", strings.TrimSuffix(entry.Name(), ".up.sql")).Scan(&checksum)
		if err != nil {
			return fmt.Errorf("database schema is not ready; run migrate: %w", err)
		}
		if checksum != fmt.Sprintf("%x", sha256.Sum256(sql)) {
			return fmt.Errorf("migration %s checksum mismatch", entry.Name())
		}
	}
	return nil
}
