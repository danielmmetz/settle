package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/danielmmetz/settle/cmd"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func mainE(ctx context.Context) error {
	ensure := cmd.Ensure()
	var root ffcli.Command
	root = ffcli.Command{
		ShortUsage: "settle <subcommand>",
		ShortHelp:  "Pass -h to see other subcommands. Defaults to `ensure` if no subcommand is provided.",
		Subcommands: []*ffcli.Command{
			cmd.Init(),
			ensure,
			cmd.DumpConfig(),
			cmd.Version(version, commit, date),
		},
		Exec: func(ctx context.Context, args []string) error {
			command := root.FlagSet.Arg(0)
			if command != "" && command != "ensure" {
				return fmt.Errorf("unknown subcommand %s", command)
			}
			return ensure.Exec(ctx, args)
		},
	}

	return root.ParseAndRun(ctx, os.Args[1:])
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := mainE(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if ctx.Err() == nil {
			os.Exit(1)
		}
	}
}
