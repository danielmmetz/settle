package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/danielmmetz/settle/pkg/brew"
	"github.com/danielmmetz/settle/pkg/files"
	"github.com/danielmmetz/settle/pkg/log"
	"github.com/danielmmetz/settle/pkg/nvim"
	"github.com/danielmmetz/settle/pkg/store"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
)

func main() {
	fVerbose := flag.Bool("verbose", false, "enable verbose logging")
	fDumpConfig := flag.Bool("dump-config", false, "pretty print config then exit without applying changes")
	fSkipBrew := flag.Bool("skip-brew", false, "skip applying brew changes")
	flag.Parse()

	var logger log.Log
	if *fVerbose {
		logger.Level = log.LevelDebug
	}

	db, err := sql.Open("sqlite3", "inventory.db")
	if err != nil {
		logger.Fatal("error opening db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	e := ensurer{log: logger, store: store.New(db)}
	if err := e.loadConfig(*fSkipBrew); err != nil {
		logger.Fatal("error loading config: %v", err)
	}
	if *fDumpConfig {
		logger.Info(e.dumpConfig())
		os.Exit(0)
	}

	if err := e.Ensure(ctx); err != nil {
		logger.Fatal("error applying config: %v", err)
	}
}

func (e *ensurer) loadConfig(skipBrew bool) error {
	bytes, err := ioutil.ReadFile("settle.yaml")
	if err != nil {
		return fmt.Errorf("error loading settle.yaml: %w", err)
	}
	var config struct {
		Files files.Files
		Nvim  nvim.Nvim
		Brew  *brew.Brew
	}
	if err = yaml.Unmarshal(bytes, &config); err != nil {
		return err
	}

	e.files = config.Files
	e.nvim = config.Nvim
	if !skipBrew {
		e.brew = config.Brew
	}
	return err
}

type ensurer struct {
	log   log.Log
	store store.Store
	files files.Files
	nvim  nvim.Nvim
	brew  *brew.Brew
}

func (e *ensurer) Ensure(ctx context.Context) error {
	if err := e.store.Ensure(ctx); err != nil {
		return fmt.Errorf("error ensuring table: %w", err)
	}
	if err := e.files.Ensure(ctx, e.log, e.store); err != nil {
		return err
	}
	if err := e.nvim.Ensure(ctx, e.log, e.store); err != nil {
		return err
	}
	if err := e.brew.Ensure(ctx, e.log, e.store); err != nil {
		return err
	}
	if err := e.store.Cleanup(); err != nil {
		return fmt.Errorf("error during garbage collection: %w", err)
	}
	return nil
}

func (e *ensurer) dumpConfig() string {
	c := struct {
		Files files.Files
		Nvim  nvim.Nvim
		Brew  *brew.Brew
	}{
		Files: e.files,
		Nvim:  e.nvim,
		Brew:  e.brew,
	}
	pretty, _ := json.MarshalIndent(c, "", "  ")
	return string(pretty)
}
