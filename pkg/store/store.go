package store

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
)

const ensureTablesStatement = `
CREATE TABLE IF NOT EXISTS inventory (
	path TEXT NOT NULL,
	generation INTEGER NOT NULL,
	PRIMARY KEY (path, generation)
);

CREATE TABLE IF NOT EXISTS content (
	name TEXT UNIQUE NOT NULL,
	content TEXT NOT NULL
);
`

const setContentStatement = `
INSERT INTO content (name, content) VALUES (?, ?)
ON CONFLICT(name) DO UPDATE SET content = ?
`

const selectOldPaths = `
SELECT path
FROM inventory
WHERE generation < ?
`

func New(db *sql.DB) Store {
	return Store{
		db: db,
	}
}

type Store struct {
	db *sql.DB

	once       sync.Once
	generation struct {
		err   error
		value int
	}
}

func (i *Store) Ensure(ctx context.Context) error {
	_, err := i.db.ExecContext(ctx, ensureTablesStatement)
	return err
}

func (i *Store) Cleanup() error {
	// TODO
	return nil
}

func (i *Store) determineGeneration(ctx context.Context) {
	var result sql.NullInt64
	err := i.db.QueryRowContext(ctx, "SELECT MAX(generation) FROM inventory").Scan(&result)
	if err != nil {
		i.generation.err = err
	}
	i.generation.value = int(result.Int64)
}

func (i *Store) log(ctx context.Context, path string) error {
	i.once.Do(func() { i.determineGeneration(ctx) })
	if i.generation.err != nil {
		return fmt.Errorf("unable to determine generation: %w", i.generation.err)
	}
	_, err := i.db.ExecContext(ctx, i.insertNStatement(1), path, i.generation.value)
	return err
}

func (i *Store) oldPaths(ctx context.Context) ([]string, error) {
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

func (i *Store) SetContent(ctx context.Context, name string, content string) error {
	_, err := i.db.ExecContext(ctx, setContentStatement, name, content, content)
	return err
}

func (i *Store) Content(ctx context.Context, name string) (string, error) {
	var result string
	err := i.db.QueryRowContext(ctx, "SELECT content FROM content WHERE name = ?", name).Scan(&result)
	return result, err
}

func (i *Store) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (i *Store) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (i *Store) MkdirAll(path string, perm os.FileMode) error {
	// TODO should this log to the db?
	return os.MkdirAll(path, perm)
}

func (i *Store) Symlink(ctx context.Context, oldname, newname string) error {
	if err := os.Symlink(oldname, newname); err != nil {
		return err
	}
	return i.log(ctx, newname)
}

func (i *Store) WriteFile(ctx context.Context, filename string, content []byte, perm os.FileMode) error {
	if err := ioutil.WriteFile(filename, content, perm); err != nil {
		return err
	}
	return i.log(ctx, filename)
}

func (i *Store) Remove(name string) error {
	return os.Remove(name)
}

func (i *Store) OpenGitRepo(path string) (*git.Repository, error) {
	return git.PlainOpen(path)
}

func (i *Store) GitClone(ctx context.Context, path string, options *git.CloneOptions) (*git.Repository, error) {
	// TODO log to DB
	return git.PlainCloneContext(ctx, path, false, options)
}
