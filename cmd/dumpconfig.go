package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/danielmmetz/settle/internal/config"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func DumpConfig() *ffcli.Command {
	fs := flag.NewFlagSet("settle dump-config", flag.ExitOnError)
	path := fs.String("config", "", "use config file at given path")
	format := fs.String("format", "json", "output format (json or yaml)")
	target := fs.String("target", "", "apply only specified stanza of the config")

	return &ffcli.Command{
		Name:       "dump-config",
		ShortUsage: "settle dump-config [-config path] [-format json|yaml] [-target files|brew|apt|nvim|zsh]",
		FlagSet:    fs,
		Exec: func(_ context.Context, _ []string) error {
			c, err := config.Load(*path, config.OptionFrom(*target))
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			var b []byte
			switch *format {
			case "json":
				b = c.JSON()
			case "yaml":
				b = c.YAML()
			default:
				return fmt.Errorf(`invalid format value specified: expected "json" or "yaml"`)
			}
			fmt.Println(string(b))
			return nil
		},
	}
}
