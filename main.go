package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/term"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func mainE(ctx context.Context) error {
	var (
		command        string
		configPath     string
		format         string
		target         string
		repo           string
		authMethod     string
		privateKeyPath string
	)
	flag.StringVar(&command, "command", "ensure", "sub-command for settle (init, ensure, dump-config, version, version-verbose)")
	flag.StringVar(&configPath, "config", "", "if specified, uses config file at given path (default: previous value, then settle.yaml)")
	flag.StringVar(&target, "target", "", "if specified, applies only specified stanza of the config")
	flag.StringVar(&format, "format", "json", "dump-config specific: output format (json or yaml)")
	flag.StringVar(&repo, "repo", "", "init specific: clone specified repo, then ensure (format: owner/repo)")
	flag.StringVar(&authMethod, "auth", "", `init specific: use specified auth type for clone ("", "basic", "pubkey")`)
	flag.StringVar(&privateKeyPath, "private-key", "", `init specific: path to PEM encoded private key`)
	flag.Parse()

	var opts []option
	switch target {
	case "files":
		opts = append(opts, withOnlyFiles())
	case "nvim":
		opts = append(opts, withOnlyNvim())
	case "zsh":
		opts = append(opts, withOnlyZsh())
	case "":
	default:
		return fmt.Errorf("unsupported target specified: %s", target)
	}

	var err error
	switch command {
	case "version":
		fmt.Println(version)
		return nil
	case "version-verbose":
		fmt.Printf("version: %s\ncommit: %s\ndate: %s\n", version, commit, date)
		return nil
	case "dump-config":
		c, err := loadConfig(configPath, opts...)
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		var b []byte
		switch format {
		case "json":
			b = c.JSON()
		case "yaml":
			b = c.YAML()
		default:
			return fmt.Errorf(`invalid format value specified: expected "json" or "yaml"`)
		}
		fmt.Println(string(b))
		return nil
	case "init":
		repoParts := strings.Split(repo, "/")
		if len(repoParts) != 2 {
			return fmt.Errorf(`expected repo of form "owner/repo", got %s`, repo)
		}
		owner, repository := repoParts[0], repoParts[1]
		url := fmt.Sprintf("https://github.com/%s/%s", owner, repository)

		var auth transport.AuthMethod
		switch {
		case authMethod == "basic":
			fmt.Print("username: ")
			username, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading username: %w", err)
			}
			fmt.Print("password or token: ")
			password, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("error reading password: %w", err)
			}
			auth = &http.BasicAuth{Username: username, Password: string(password)}
		case authMethod == "pubkey":
			url = fmt.Sprintf("git@github.com:%s/%s.git", owner, repository)
			auth, err = ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
			if err != nil {
				return fmt.Errorf("error creating pubkey auth: %w", err)
			}
		case authMethod != "":
			return fmt.Errorf(`invalid auth type specified: expected "" or "basic"`)
		}
		_, err := git.PlainCloneContext(ctx, repository, false, &git.CloneOptions{
			URL:  url,
			Auth: auth,
		})
		if err != nil {
			return fmt.Errorf("error cloning repo: %w", err)
		}

		if err := os.Chdir(repository); err != nil {
			return fmt.Errorf("error chdir-ing into cloned repo")
		}
		fallthrough
	case "ensure":
		c, err := loadConfig(configPath, opts...)
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		if err := c.Ensure(ctx); err != nil {
			return err
		}

		if target != "" {
			fmt.Println("skipping writing of settings.yaml and creating settle.yaml backup: non-zero target specified:", target)
			return nil
		}

		return writeBackup(c)
	default:
		return fmt.Errorf("invalid command specified: %s", command)
	}
}

// loadConfig loads the config at path.
// If path == "", it will attempt to infer the config path.
// Note: loadConfig may change the program's working directory
// so that it may correctly handle relative paths.
func loadConfig(path string, opts ...option) (config, error) {
	var err error
	if path == "" {
		path, err = inferConfigPath()
		if err != nil {
			return config{}, fmt.Errorf("error inferring config file path: %w", err)
		}
	}

	absConfigPath, err := filepath.Abs(path)
	if err != nil {
		return config{}, fmt.Errorf("error determing absolute path from %s: %w", path, err)
	}
	configDir := filepath.Dir(absConfigPath)
	if err := os.Chdir(configDir); err != nil {
		return config{}, fmt.Errorf("error changing directory to %s: %w", configDir, err)
	}

	configBytes, err := os.ReadFile(absConfigPath)
	if err != nil {
		return config{}, fmt.Errorf("error reading config file %s: %w", absConfigPath, err)
	}
	var c config
	if err := yaml.Unmarshal(configBytes, &c); err != nil {
		return config{}, fmt.Errorf("error parsing config file: %w", err)
	}
	c.absPath = absConfigPath

	for _, o := range opts {
		o(&c)
	}
	return c, nil
}

// inferConfigPath infers the config file path using the following priorities:
// 1. last used config file path (as stored in settings.yaml)
// 2. settle.yaml
func inferConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home dir: %w", err)
	}
	settingsPath := filepath.Join(home, ".config", "settle", "settings.yaml")
	b, err := os.ReadFile(settingsPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("error reading %s: %w", settingsPath, err)
	}
	var configPath string
	if err == nil {
		var s settings
		if err := yaml.Unmarshal(b, &s); err != nil {
			return "", fmt.Errorf("error parsing %s: %w", settingsPath, err)
		}
		configPath = s.ConfigPath
	}
	if configPath == "" {
		configPath = "settle.yaml"
	}
	return configPath, nil
}

func writeBackup(c config) error {
	settingsBytes, err := yaml.Marshal(settings{ConfigPath: c.absPath})
	if err != nil {
		return fmt.Errorf("error marshaling contents for settings.yaml: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}

	_ = os.MkdirAll(filepath.Join(home, ".config", "settle"), 0755)
	err = os.WriteFile(
		filepath.Join(home, ".config", "settle", "settings.yaml"),
		settingsBytes,
		0644,
	)
	if err != nil {
		return fmt.Errorf("error writing settings.yaml: %w", err)
	}
	xdgDataDir := filepath.Join(home, ".local", "share")
	_ = os.MkdirAll(filepath.Join(xdgDataDir, "settle"), 0755)
	err = os.WriteFile(
		filepath.Join(home, ".local", "share", "settle", fmt.Sprintf("%s.yaml", time.Now().Local().Format("2006-01-02 15:04:05"))),
		c.YAML(),
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

	// absPath is the absolute path to where config exists on disk.
	absPath string
}

func (c *config) UnmarshalJSON(b []byte) error {
	type clone struct {
		Includes []string `json:"includes"`
		Files    *Files   `json:"files"`
		Brew     *Brew    `json:"brew"`
		Apt      *Apt     `json:"apt"`
		Nvim     *Nvim    `json:"nvim"`
		Zsh      *Zsh     `json:"zsh"`
	}
	var original clone
	if err := json.Unmarshal(b, &original); err != nil {
		return err
	}

	var final config
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

func (c *config) JSON() []byte {
	b, _ := json.MarshalIndent(c, "", "  ")
	return b
}

func (c *config) YAML() []byte {
	b, _ := yaml.Marshal(c)
	return b
}

func (c *config) Ensure(ctx context.Context) error {
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

type option func(c *config)

func withOnlyFiles() option {
	return func(c *config) {
		*c = config{Files: c.Files}
	}
}

func withOnlyNvim() option {
	return func(c *config) {
		*c = config{Nvim: c.Nvim}
	}
}

func withOnlyZsh() option {
	return func(c *config) {
		*c = config{Zsh: c.Zsh}
	}
}
