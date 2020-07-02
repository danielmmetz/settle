package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielmmetz/settle/pkg/log"
	"github.com/go-git/go-git/v5"
	_ "github.com/mattn/go-sqlite3"
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
	if err := ensureTable(ctx, db); err != nil {
		logger.Fatal("error ensuring table: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		logger.Fatal("error loading config: %v", err)
	}
	logger.Debug("loaded config: %+v", config)

	e := ensurer{log: logger, cfg: config}
	if err := e.ensure(ctx); err != nil {
		logger.Fatal("error applying config: %v", err)
	}
}

func ensureTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ensureTableStatement)
	return err
}

type config struct {
	Files []FileMapping
	Vim   Vim
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

type Vim struct {
	PluginDir string `yaml:"plugin_dir"`
	Plugins   []Plugin
	Config    string
}

func (v Vim) destDir(p Plugin) string {
	return fmt.Sprintf("%s/%s", v.PluginDir, p.Name.Repo)
}

func (v Vim) initVim() string {
	var vimPlugLines []string
	vimPlugLines = append(vimPlugLines, fmt.Sprintf("call plug#begin('%s')", v.PluginDir))
	for _, plugin := range v.Plugins {
		vimPlugLines = append(vimPlugLines, plugin.toVimPlug())
	}
	vimPlugLines = append(vimPlugLines, "call plug#end()")
	vimPlugLines = append(vimPlugLines, "\n")
	vimPlugLines = append(vimPlugLines, v.Config)
	vimPlugLines = append(vimPlugLines, "\n")
	return strings.Join(vimPlugLines, "\n")
}

type Plugin struct {
	Name PluginName
	Do   string
	For  string
}

func (p Plugin) toVimPlug() string {
	plugStatement := fmt.Sprintf("Plug '%v'", p.Name)
	if p.Do == "" && p.For == "" {
		return plugStatement
	}
	options := make(map[string]string)
	if p.Do != "" {
		options["do"] = p.Do
	}
	if p.For != "" {
		options["for"] = p.For
	}
	jsonified, _ := json.Marshal(options)
	formattedOptions := strings.ReplaceAll(string(jsonified), `"`, `'`)
	return fmt.Sprintf("%s, %s", plugStatement, formattedOptions)
}

type PluginName struct {
	Owner string
	Repo  string
}

func (p PluginName) String() string {
	return fmt.Sprintf("%s/%s", p.Owner, p.Repo)
}

func (p *PluginName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intermediary string
	if err := unmarshal(&intermediary); err != nil {
		return err
	}
	components := strings.Split(intermediary, "/")
	if len(components) != 2 {
		return fmt.Errorf(`expected plugin to resemble "owner/repo": got %s`, intermediary)
	}
	p.Owner = components[0]
	p.Repo = components[1]
	return nil
}

func (p Plugin) String() string {
	return p.Name.String()
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
	log log.Log
	cfg config
}

func (e ensurer) ensure(ctx context.Context) error {
	for _, mapping := range e.cfg.Files {
		if err := e.ensureFileMapping(mapping); err != nil {
			return err
		}
	}
	var wg errgroup.Group
	for _, plugin := range e.cfg.Vim.Plugins {
		plugin := plugin
		wg.Go(func() error { return e.ensureVimPlugin(plugin) })
	}
	if err := wg.Wait(); err != nil {
		return err
	}
	return e.ensureInitVim()
}

func (e ensurer) ensureFileMapping(m FileMapping) error {
	_, err := os.Lstat(m.Dst)
	if errors.Is(err, os.ErrNotExist) {
		// do nothing
	} else if err != nil {
		return err
	} else if err == nil {
		e.log.Debug("file exists, deleting it: %s", m.Dst)
		if err := os.Remove(m.Dst); err != nil {
			return err
		}
	}
	e.log.Debug("symlinking %v to %v", m.Src, m.Dst)
	return os.Symlink(m.Src, m.Dst)
}

func (e ensurer) ensureVimPlugin(p Plugin) error {
	dst := e.cfg.Vim.destDir(p)
	_, err := git.PlainOpen(dst)
	if err == git.ErrRepositoryNotExists {
		// do nothing
	} else if err != nil {
		return fmt.Errorf("unable to check for existing repo for %v: %w", p, err)
	} else if err == nil {
		e.log.Debug("repo exists, skipping clone for %v", p)
		return nil
	}

	e.log.Info("cloning repo %v into %s", p, dst)
	_, err = git.PlainClone(dst, false, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s/%s.git", p.Name.Owner, p.Name.Repo),
	})
	return err
}

func (e ensurer) ensureInitVim() error {
	if len(e.cfg.Vim.Plugins) == 0 && e.cfg.Vim.Config == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	cfgPath := filepath.Join(home, ".config", "nvim", "init.vim")
	e.log.Info("writing vim config to %s", cfgPath)
	return ioutil.WriteFile(cfgPath, []byte(e.cfg.Vim.initVim()), 0755)
}
