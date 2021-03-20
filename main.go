package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ghodss/yaml"
)

func mainE(ctx context.Context) error {
	var (
		dumpConfig string
		configPath string
	)
	flag.StringVar(&dumpConfig, "dump-config", "", "if specified, prints the parsed config file in the specified format (json or yaml) then exits")
	flag.StringVar(&configPath, "config", "", "if specified, uses config file at given path (default: previous value, then settle.yaml)")
	flag.Parse()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	if configPath == "" {
		settingsPath := filepath.Join(home, ".config", "settle", "settings.yaml")
		b, err := ioutil.ReadFile(settingsPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error reading %s: %w", settingsPath, err)
		}
		if err == nil {
			var s settings
			if err := yaml.Unmarshal(b, &s); err != nil {
				return fmt.Errorf("error parsing %s: %w", settingsPath, err)
			}
			configPath = s.ConfigPath
		}
		if configPath == "" {
			configPath = "settle.yaml"
		}
	}

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("error determing absolute path from %s: %w", configPath, err)
	}
	configDir := filepath.Dir(absConfigPath)
	if err := os.Chdir(configDir); err != nil {
		return fmt.Errorf("error changing directory to %s: %w", configDir, err)
	}

	configBytes, err := ioutil.ReadFile(absConfigPath)
	if err != nil {
		return fmt.Errorf("error reading config file %s: %w", absConfigPath, err)
	}
	var c config
	if err := yaml.Unmarshal(configBytes, &c); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	if dumpConfig != "" {
		var bDump []byte
		var err error
		switch dumpConfig {
		case "json":
			bDump, err = json.MarshalIndent(c, "", "  ")
		case "yaml":
			bDump, err = yaml.Marshal(c)
		default:
			return fmt.Errorf(`invalid dump-config value specified: expected "json" or "yaml"`)
		}
		if err != nil {
			return fmt.Errorf("error marshaling config: %w", err)
		}
		fmt.Println(string(bDump))
		return nil
	}

	if err := c.Ensure(ctx); err != nil {
		return err
	}

	settingsBytes, err := yaml.Marshal(settings{ConfigPath: absConfigPath})
	if err != nil {
		return fmt.Errorf("error marshaling contents for settings.yaml: %w", err)
	}
	_ = os.MkdirAll(filepath.Join(home, ".config", "settle"), 0755)
	err = ioutil.WriteFile(
		filepath.Join(home, ".config", "settle", "settings.yaml"),
		settingsBytes,
		0644,
	)
	if err != nil {
		return fmt.Errorf("error writing settings.yaml: %w", err)
	}
	xdgDataDir := filepath.Join(home, ".local", "share")
	_ = os.MkdirAll(filepath.Join(xdgDataDir, "settle"), 0755)
	err = ioutil.WriteFile(
		filepath.Join(home, ".local", "share", "settle", fmt.Sprintf("%s.yaml", time.Now().Local().Format("2006-01-02 15:04:05"))),
		configBytes,
		0644,
	)
	if err != nil {
		return fmt.Errorf("error writing settle.yaml copy: %w", err)
	}
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
		<-sigs
		os.Exit(1)
	}()
	if err := mainE(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type config struct {
	Files *Files `json:"files"`
	Brew  *Brew  `json:"brew"`
	Apt   *Apt   `json:"apt"`
	Nvim  *Nvim  `json:"nvim"`
	Zsh   *Zsh   `json:"zsh"`
}

func (c config) Ensure(ctx context.Context) error {
	if err := c.Files.Ensure(ctx); err != nil {
		return fmt.Errorf("error ensuring files: %w", err)
	}
	if err := c.Brew.Ensure(ctx); err != nil {
		return fmt.Errorf("error ensuring brew: %w", err)
	}
	if err := c.Nvim.Ensure(ctx); err != nil {
		return fmt.Errorf("error ensuring nvim: %w", err)
	}
	if err := c.Zsh.Ensure(ctx); err != nil {
		return fmt.Errorf("error ensuring zsh: %w", err)
	}
	return nil
}

type settings struct {
	ConfigPath string `json:"configPath"`
}
