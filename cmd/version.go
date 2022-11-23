package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func Version(version, commit, date string) *ffcli.Command {
	versionFlagSet := flag.NewFlagSet("settle version", flag.ExitOnError)
	verbose := versionFlagSet.Bool("verbose", false, "")

	return &ffcli.Command{
		Name:    "version",
		FlagSet: versionFlagSet,
		Exec: func(_ context.Context, _ []string) error {
			if *verbose {
				fmt.Printf("version: %s\ncommit: %s\ndate: %s\n", version, commit, date)
				return nil

			}
			fmt.Println(version)
			return nil
		},
	}
}
