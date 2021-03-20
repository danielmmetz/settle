package main

import (
	"context"
	"encoding/json"
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
	dumpConfig := flag.String("dump-config", "", "if specified, prints the parsed config file in the specified format (json or yaml) then exits")
	flag.Parse()

	configBytes, err := ioutil.ReadFile("settle.yaml")
	if err != nil {
		return fmt.Errorf("error reading settle.yaml: %w", err)
	}
	var c config
	if err := yaml.Unmarshal(configBytes, &c); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	if *dumpConfig != "" {
		var bDump []byte
		var err error
		switch *dumpConfig {
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

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
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
