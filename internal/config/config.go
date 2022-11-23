package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/danielmmetz/settle/internal/apt"
	"github.com/danielmmetz/settle/internal/brew"
	"github.com/danielmmetz/settle/internal/files"
	"github.com/danielmmetz/settle/internal/nvim"
	"github.com/danielmmetz/settle/internal/zsh"
	"github.com/ghodss/yaml"
	"github.com/peterbourgon/ff/v3"
)

// Load loads the config at path.
// If path == "", it will attempt to load settle.yaml.
// Note: Load may change the program's working directory
// so that it may correctly handle relative paths.
func Load(path string, opts ...Option) (Config, error) {
	var err error
	if path == "" {
		path = "settle.yaml"
	}

	absConfigPath, err := filepath.Abs(path)
	if err != nil {
		return Config{}, fmt.Errorf("error determing absolute path from %s: %w", path, err)
	}
	configDir := filepath.Dir(absConfigPath)
	if err := os.Chdir(configDir); err != nil {
		return Config{}, fmt.Errorf("error changing directory to %s: %w", configDir, err)
	}

	configBytes, err := os.ReadFile(absConfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file %s: %w", absConfigPath, err)
	}
	var c Config
	if err := yaml.Unmarshal(configBytes, &c); err != nil {
		return Config{}, fmt.Errorf("error parsing config file: %w", err)
	}
	c.absPath = absConfigPath

	for _, o := range opts {
		o(&c)
	}
	return c, nil
}

func WriteBackup(c Config) error {
	settingsBytes, err := yaml.Marshal(settings{ConfigPath: c.absPath})
	if err != nil {
		return fmt.Errorf("error marshaling contents for settings.yaml: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}

	_ = os.MkdirAll(filepath.Join(home, ".config", "settle"), 0o755)
	err = os.WriteFile(
		filepath.Join(home, ".config", "settle", "settings.yaml"),
		settingsBytes,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("error writing settings.yaml: %w", err)
	}
	xdgDataDir := filepath.Join(home, ".local", "share")
	_ = os.MkdirAll(filepath.Join(xdgDataDir, "settle"), 0o755)
	err = os.WriteFile(
		filepath.Join(home, ".local", "share", "settle", fmt.Sprintf("%s.yaml", time.Now().Local().Format("2006-01-02 15:04:05"))),
		c.YAML(),
		0o644,
	)
	if err != nil {
		return fmt.Errorf("error writing settle.yaml copy: %w", err)
	}
	return nil
}

type Config struct {
	Files *files.Files `json:"files"`
	Brew  *brew.Brew   `json:"brew"`
	Apt   *apt.Apt     `json:"apt"`
	Nvim  *nvim.Nvim   `json:"nvim"`
	Zsh   *zsh.Zsh     `json:"zsh"`

	// absPath is the absolute path to where config exists on disk.
	absPath string
}

func (c *Config) UnmarshalJSON(b []byte) error {
	type clone struct {
		Includes []string     `json:"includes"`
		Files    *files.Files `json:"files"`
		Brew     *brew.Brew   `json:"brew"`
		Apt      *apt.Apt     `json:"apt"`
		Nvim     *nvim.Nvim   `json:"nvim"`
		Zsh      *zsh.Zsh     `json:"zsh"`
	}
	var original clone
	if err := json.Unmarshal(b, &original); err != nil {
		return err
	}

	var final Config
	for _, f := range original.Includes {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", f, err)
		}
		if err := yaml.Unmarshal(b, &final); err != nil {
			return fmt.Errorf("error unmarshaling %s: %w", f, err)
		}
	}

	if original.Files != nil {
		final.Files = original.Files
	}
	if original.Brew != nil {
		final.Brew = original.Brew
	}
	if original.Apt != nil {
		final.Apt = original.Apt
	}
	if original.Nvim != nil {
		final.Nvim = original.Nvim
	}
	if original.Zsh != nil {
		final.Zsh = original.Zsh
	}
	*c = final
	return nil
}

func (c *Config) JSON() []byte {
	b, _ := json.MarshalIndent(c, "", "  ")
	return b
}

func (c *Config) YAML() []byte {
	b, _ := yaml.Marshal(c)
	return b
}

func (c *Config) Ensure(ctx context.Context) error {
	if err := c.Files.Ensure(ctx); err != nil {
		return fmt.Errorf("error ensuring files: %w", err)
	}
	if err := c.Apt.Ensure(ctx); err != nil {
		return fmt.Errorf("error ensuring apt: %w", err)
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

type Option func(c *Config)

func OptionFrom(target string) Option {
	switch target {
	case "brew":
		return OnlyBrew()
	case "files":
		return OnlyFiles()
	case "nvim":
		return OnlyNvim()
	case "zsh":
		return OnlyZsh()
	default:
		return func(c *Config) {}
	}
}

func OnlyBrew() Option {
	return func(c *Config) {
		*c = Config{Brew: c.Brew}
	}
}

func OnlyFiles() Option {
	return func(c *Config) {
		*c = Config{Files: c.Files}
	}
}

func OnlyNvim() Option {
	return func(c *Config) {
		*c = Config{Nvim: c.Nvim}
	}
}

func OnlyZsh() Option {
	return func(c *Config) {
		*c = Config{Zsh: c.Zsh}
	}
}

func Parser() ff.ConfigFileParser {
	return func(r io.Reader, set func(name, value string) error) error {
		b, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading: %w", err)
		}
		var s settings
		if err := yaml.Unmarshal(b, &s); err != nil {
			return fmt.Errorf("yaml unmarshal: %w", err)
		}
		if err := set("config", s.ConfigPath); err != nil {
			return fmt.Errorf("set config=%s: %w", s.ConfigPath, err)
		}
		return nil
	}
}
