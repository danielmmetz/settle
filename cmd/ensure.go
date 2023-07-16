package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/danielmmetz/settle/internal/config"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func Ensure(settingsPath string) *ffcli.Command {
	fs := flag.NewFlagSet("settle ensure", flag.ExitOnError)
	configPath := fs.String("config", "", "use config file at given path")
	target := fs.String("target", "", "apply only specified stanza of the config")

	return &ffcli.Command{
		Name:       "ensure",
		ShortUsage: "settle ensure [-config path] [-target files|brew|apt|pacman|nvim|zsh]",
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithConfigFile(settingsPath),
			ff.WithConfigFileParser(config.Parser()),
			ff.WithAllowMissingConfigFile(true),
		},
		Exec: func(ctx context.Context, _ []string) error {
			c, err := config.Load(*configPath, config.OptionFrom(*target))
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			if err := c.Ensure(ctx); err != nil {
				return err
			}

			if *target != "" {
				fmt.Println("skipping writing of settings.yaml and creating settle.yaml backup: non-zero target specified:", *target)
				return nil
			}

			return config.WriteBackup(c)
		},
	}
}
