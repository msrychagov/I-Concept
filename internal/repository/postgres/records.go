package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// recordSet stores each entity independently while preserving versioned payloads.
type recordSet[T any] struct {
	table string
	key   func(T) string
}

func (s recordSet[T]) list(ctx context.Context, tx pgx.Tx) ([]T, error) {
	rows, err := tx.Query(ctx, "SELECT payload FROM "+pgx.Identifier{s.table}.Sanitize()+" ORDER BY position,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []T{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var value T
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", s.table, err)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s recordSet[T]) sync(ctx context.Context, tx pgx.Tx, before, after []T) error {
	type saved struct {
		position int
		data     []byte
	}
	previous := make(map[string]saved, len(before))
	for i, value := range before {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		previous[s.key(value)] = saved{i, data}
	}
	seen := make(map[string]bool, len(after))
	batch := &pgx.Batch{}
	table := pgx.Identifier{s.table}.Sanitize()
	for i, value := range after {
		id := s.key(value)
		if id == "" || seen[id] {
			return fmt.Errorf("%s: empty or duplicate record ID", s.table)
		}
		seen[id] = true
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		old, exists := previous[id]
		if exists && old.position == i && bytes.Equal(old.data, data) {
			continue
		}
		batch.Queue("INSERT INTO "+table+" (id,position,payload) VALUES ($1,$2,$3) ON CONFLICT (id) DO UPDATE SET position=EXCLUDED.position,payload=EXCLUDED.payload", id, i, data)
	}
	removed := []string{}
	for id := range previous {
		if !seen[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		if _, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE id=ANY($1::text[])", removed); err != nil {
			return err
		}
	}
	if batch.Len() == 0 {
		return nil
	}
	return tx.SendBatch(ctx, batch).Close()
}
