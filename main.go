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
	"os/exec"
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
	fDumpConfig := flag.Bool("dump-config", false, "pretty print config then exit without applying changes")
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
	if *fDumpConfig {
		logger.Info("%+v", config)
		os.Exit(0)
	}

	e := NewEnsurer(config)
	if err := e.Ensure(ctx, logger); err != nil {
		logger.Fatal("error applying config: %v", err)
	}
}

func ensureTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ensureTableStatement)
	return err
}

type config struct {
	Files Files
	Vim   Vim
	Brew  *Brew
}

func (c config) String() string {
	pretty, _ := json.MarshalIndent(c, "", "  ")
	return string(pretty)
}

type Files []FileMapping

func (f Files) Ensure(logger log.Log) error {
	for _, mapping := range f {
		if err := mapping.ensure(logger); err != nil {
			return err
		}
	}
	return nil
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
	Config    VimConfig
}

func (v Vim) Ensure(logger log.Log) error {
	var wg errgroup.Group
	for _, plugin := range v.Plugins {
		plugin := plugin
		wg.Go(func() error { return v.ensurePlugin(logger, plugin) })
	}
	if err := wg.Wait(); err != nil {
		return err
	}
	return v.ensureInitVim(logger)

}

func (v Vim) destDir(p Plugin) string {
	return fmt.Sprintf("%s/%s", v.PluginDir, p.Repo)
}

func (v Vim) initVim() string {
	var vimPlugLines []string
	vimPlugLines = append(vimPlugLines, fmt.Sprintf("call plug#begin('%s')", v.PluginDir))
	for _, plugin := range v.Plugins {
		vimPlugLines = append(vimPlugLines, plugin.toVimPlug())
	}
	vimPlugLines = append(vimPlugLines, "call plug#end()")
	vimPlugLines = append(vimPlugLines, "\n")
	vimPlugLines = append(vimPlugLines, string(v.Config))
	vimPlugLines = append(vimPlugLines, "\n")
	return strings.Join(vimPlugLines, "\n")
}

type Plugin struct {
	Owner string
	Repo  string
}

func (p Plugin) toVimPlug() string {
	return fmt.Sprintf("Plug '%s'", p.String())
}

func (p Plugin) String() string {
	return fmt.Sprintf("%s/%s", p.Owner, p.Repo)
}

func (p *Plugin) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

type VimConfig string

func (c VimConfig) MarshalJSON() ([]byte, error) {
	if len(c) > 0 {
		return json.Marshal("(present but omitted)")
	}
	return []byte("(empty)"), nil
}

type Brew struct {
	Taps []string
	Pkgs []struct {
		Name string
		Args []string
	}
	Casks []string
}

func (b *Brew) Ensure(ctx context.Context, logger log.Log) error {
	if b == nil {
		return nil
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("error creating temporary Brewfile: %w", err)
	}
	if _, err := f.WriteString(b.brewfile()); err != nil {
		return err
	}
	logger.Debug("wrote temporary Brewfile to: %s", f.Name())

	logger.Info("installing packages with `brew bundle`")
	installCmd := exec.CommandContext(ctx, "brew", "bundle", "--file", f.Name())
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("error running `brew bundle`: %w", err)
	}
	logger.Info("cleaning up orphan packages with `brew bundle cleanup`")
	cleanupCmd := exec.CommandContext(ctx, "brew", "bundle", "cleanup", "--force", "--file", f.Name())
	if err := cleanupCmd.Run(); err != nil {
		return fmt.Errorf("error running `brew bundle cleanup`: %w", err)
	}
	return nil
}

func (b *Brew) brewfile() string {
	var lines []string
	for _, tap := range b.Taps {
		lines = append(lines, fmt.Sprintf(`tap "%s"`, tap))
	}
	for _, pkg := range b.Pkgs {
		lineComponents := []string{fmt.Sprintf(`brew "%s"`, pkg.Name)}
		if len(pkg.Args) > 0 {
			lineComponents = append(lineComponents, ", args: [")
			for _, arg := range pkg.Args {
				lineComponents = append(lineComponents, fmt.Sprintf(`"%s"`, arg))
			}
			lineComponents = append(lineComponents, "]")
		}
		lines = append(lines, strings.Join(lineComponents, ""))
	}
	for _, cask := range b.Casks {
		lines = append(lines, fmt.Sprintf(`cask "%s"`, cask))
	}
	return strings.Join(lines, "\n")
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
	log   log.Log
	files Files
	vim   Vim
	brew  *Brew
}

func NewEnsurer(cfg config) ensurer {
	return ensurer{
		files: cfg.Files,
		vim:   cfg.Vim,
		brew:  cfg.Brew,
	}
}

func (e *ensurer) Ensure(ctx context.Context, logger log.Log) error {
	if err := e.files.Ensure(logger); err != nil {
		return err
	}
	if err := e.vim.Ensure(logger); err != nil {
		return err
	}
	return e.brew.Ensure(ctx, logger)
}

func (m FileMapping) ensure(logger log.Log) error {
	_, err := os.Lstat(m.Dst)
	if errors.Is(err, os.ErrNotExist) {
		// do nothing
	} else if err != nil {
		return err
	} else if err == nil {
		logger.Debug("file exists, deleting it: %s", m.Dst)
		if err := os.Remove(m.Dst); err != nil {
			return err
		}
	}
	logger.Debug("symlinking %v to %v", m.Src, m.Dst)
	return os.Symlink(m.Src, m.Dst)
}

func (v Vim) ensurePlugin(logger log.Log, p Plugin) error {
	dst := v.destDir(p)
	_, err := git.PlainOpen(dst)
	if err == git.ErrRepositoryNotExists {
		// do nothing
	} else if err != nil {
		return fmt.Errorf("unable to check for existing repo for %v: %w", p, err)
	} else if err == nil {
		logger.Debug("repo exists, skipping clone for %v", p)
		return nil
	}

	logger.Info("cloning repo %v into %s", p, dst)
	_, err = git.PlainClone(dst, false, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s/%s.git", p.Owner, p.Repo),
	})
	return err
}

func (v Vim) ensureInitVim(logger log.Log) error {
	if len(v.Plugins) == 0 && v.Config == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	cfgPath := filepath.Join(home, ".config", "nvim", "init.vim")
	logger.Info("writing vim config to %s", cfgPath)
	return ioutil.WriteFile(cfgPath, []byte(v.initVim()), 0755)
}
