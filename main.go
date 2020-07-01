package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
)

const ensureTableStatement = `
CREATE TABLE IF NOT EXISTS inventory (
	src TEXT NOT NULL UNIQUE,
	dst TEXT NOT NULL UNIQUE
);
`

func main() {
	fVerbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	logCfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			LevelKey:      "L",
			MessageKey:    "M",
			StacktraceKey: "S",
			LineEnding:    zapcore.DefaultLineEnding,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	if *fVerbose {
		logCfg.Development = true
		logCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	logger, err := logCfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building logger: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("sqlite3", "inventory.db")
	if err != nil {
		logger.Fatal(fmt.Sprintf("error opening db: %v", err))
	}
	defer db.Close()

	ctx := context.Background()
	if err := ensureTable(ctx, db); err != nil {
		logger.Fatal(fmt.Sprintf("error ensuring table: %v", err))
	}

	config, err := loadConfig()
	if err != nil {
		logger.Fatal(fmt.Sprintf("error loading config: %v", err))
	}
	logger.Debug(fmt.Sprintf("loaded config: %+v", config))

	e := ensurer{log: logger, cfg: config}
	if err := e.ensure(ctx); err != nil {
		logger.Fatal(fmt.Sprintf("error applying config: %v", err))
	}
}

func ensureTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ensureTableStatement)
	return err
}

type config struct {
	Files []FileMapping
	Repos []RepoMapping
}

type FileMapping struct {
	Src string
	Dst string
}

func (m *FileMapping) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

type RepoMapping struct {
	Src string
	Dst string
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

type ensurer struct {
	log *zap.Logger
	cfg config
}

func (e ensurer) ensure(ctx context.Context) error {
	for _, mapping := range e.cfg.Files {
		if err := e.ensureFileMapping(mapping); err != nil {
			return err
		}
	}
	var wg errgroup.Group
	for _, mapping := range e.cfg.Repos {
		mapping := mapping
		wg.Go(func() error { return e.ensureRepoMapping(mapping) })
	}
	return wg.Wait()
}

func (e ensurer) ensureFileMapping(m FileMapping) error {
	_, err := os.Lstat(m.Dst)
	if errors.Is(err, os.ErrNotExist) {
		// do nothing
	} else if err != nil {
		return err
	} else if err == nil {
		e.log.Debug(fmt.Sprintf("file exists, deleting it: %s", m.Dst))
		if err := os.Remove(m.Dst); err != nil {
			return err
		}
	}
	e.log.Debug(fmt.Sprintf("symlinking %v to %v", m.Src, m.Dst))
	return os.Symlink(m.Src, m.Dst)
}

func (e ensurer) ensureRepoMapping(m RepoMapping) error {
	_, err := git.PlainOpen(m.Dst)
	if err == git.ErrRepositoryNotExists {
		// do nothing
	} else if err != nil {
		return fmt.Errorf("unable to check for existing repo: %w", err)
	} else if err == nil {
		e.log.Debug(fmt.Sprintf("repo exists, skipping clone: %s", m.Src))
		return nil
	}

	e.log.Info(fmt.Sprintf("cloning repo %s into %s", m.Src, m.Dst))
	_, err = git.PlainClone(m.Dst, false, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s.git", m.Src),
	})
	return err
}
