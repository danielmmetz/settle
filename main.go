package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
)

const ensureTableStatement = `
CREATE TABLE IF NOT EXISTS inventory (
	src TEXT NOT NULL UNIQUE,
	dst TEXT NOT NULL UNIQUE
);
`

func main() {
	log.SetFlags(0)

	db, err := sql.Open("sqlite3", "inventory.db")
	if err != nil {
		log.Fatal("error opening db:", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := ensureTable(ctx, db); err != nil {
		log.Fatal("error ensuring table:", err)
	}

	config, err := loadConfig()
	if err != nil {
		log.Fatal("error loading config:", err)
	}

	if err := ensure(config); err != nil {
		log.Fatal(err)
	}
}

func ensureTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ensureTableStatement)
	return err
}

type config struct {
	Files []Mapping
}

type Mapping struct {
	Src string
	Dst string
}

func (m *Mapping) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intermediary struct {
		Src string
		Dst string
	}
	if err := unmarshal(&intermediary); err != nil {
		return err
	}

	absSrc, err := filepath.Abs(intermediary.Src)
	if err != nil {
		return fmt.Errorf("unable to resolve to absolute path: %w", err)
	}
	m.Src = absSrc
	m.Dst = intermediary.Dst
	return nil
}

func loadConfig() (config, error) {
	bytes, err := ioutil.ReadFile("settle.yaml")
	if err != nil {
		return config{}, fmt.Errorf("error loading settle.yaml: %w", err)
	}
	var result config
	err = yaml.Unmarshal(bytes, &result)
	return result, err
}

func ensure(c config) error {
	for _, mapping := range c.Files {
		if err := ensureMapping(mapping); err != nil {
			return err
		}
	}
	return nil
}

func ensureMapping(m Mapping) error {
	_, err := os.Lstat(m.Dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err == nil {
		if err := os.Remove(m.Dst); err != nil {
			return err
		}
	}
	return os.Symlink(m.Src, m.Dst)
}
