package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

const ensureTableStatement = `
CREATE TABLE IF NOT EXISTS inventory (
	path TEXT NOT NULL,
	generation INTEGER NOT NULL,
	PRIMARY KEY (path, generation)
);
`

const selectOldPaths = `
SELECT path
FROM inventory
WHERE generation < ?
`

func New(db *sql.DB) Store {
	return Store{db: db}
}

type Store struct {
	db *sql.DB

	once       sync.Once
	generation struct {
		err   error
		value int
	}
}

func (i *Store) EnsureTable(ctx context.Context) error {
	_, err := i.db.ExecContext(ctx, ensureTableStatement)
	return err
}

func (i *Store) determineGeneration(ctx context.Context) {
	var result sql.NullInt64
	err := i.db.QueryRowContext(ctx, "SELECT MAX(generation) FROM inventory").Scan(&result)
	if err != nil {
		i.generation.err = err
	}
	i.generation.value = int(result.Int64)
}

func (i *Store) Log(ctx context.Context, path string) error {
	i.once.Do(func() { i.determineGeneration(ctx) })
	if i.generation.err != nil {
		return fmt.Errorf("unable to determine generation: %w", i.generation.err)
	}
	_, err := i.db.ExecContext(ctx, i.insertNStatement(1), path, i.generation.value)
	return err
}

func (i *Store) OldPaths(ctx context.Context) ([]string, error) {
	i.once.Do(func() { i.determineGeneration(ctx) })
	if i.generation.err != nil {
		return nil, fmt.Errorf("unable to determine generation: %w", i.generation.err)
	}

	rows, err := i.db.QueryContext(ctx, selectOldPaths, i.generation.value)
	if err != nil {
		return nil, err
	}

	var paths []string
	defer rows.Close()
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}

	return paths, rows.Err()
}

func (i *Store) insertNStatement(n int) string {
	var sb strings.Builder
	_, _ = sb.WriteString("INSERT INTO inventory (path, generation) VALUES")
	for i := 0; i < n; i++ {
		sb.WriteString(" (?, ?)")
	}
	return sb.String()
}
