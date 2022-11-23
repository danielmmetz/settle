package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/danielmmetz/settle/internal/config"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/term"
)

func Init() *ffcli.Command {
	fs := flag.NewFlagSet("settle init", flag.ExitOnError)
	repo := fs.String("repo", "", "clone specified repo, then ensure (format: owner/repo)")
	authMethod := fs.String("auth", "", `use specified auth type for clone ("", "basic", "pubkey")`)
	privateKeyPath := fs.String("private-key", "", `path to PEM encoded private key`)
	configPath := fs.String("config", "", "use config file at given path")
	target := fs.String("target", "", "apply only specified stanza of the config")

	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "settle init -repo owner/repo [-auth basic|pubkey] [-private-key path/to/.pem] [-config path] [-format json|yaml] [-target files|brew|apt|nvim|zsh]",
		FlagSet:    fs,
		Exec: func(ctx context.Context, _ []string) error {
			repoParts := strings.Split(*repo, "/")
			if len(repoParts) != 2 {
				return fmt.Errorf(`expected repo of form "owner/repo", got %s`, *repo)
			}
			owner, repository := repoParts[0], repoParts[1]
			url := fmt.Sprintf("https://github.com/%s/%s", owner, repository)

			var auth transport.AuthMethod
			var err error
			switch {
			case *authMethod == "basic":
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
			case *authMethod == "pubkey":
				url = fmt.Sprintf("git@github.com:%s/%s.git", owner, repository)
				auth, err = ssh.NewPublicKeysFromFile("git", *privateKeyPath, "")
				if err != nil {
					return fmt.Errorf("error creating pubkey auth: %w", err)
				}
			case *authMethod != "":
				return fmt.Errorf(`invalid auth type specified: expected "basic" or "pubkey"`)
			}
			_, err = git.PlainCloneContext(ctx, repository, false, &git.CloneOptions{
				URL:  url,
				Auth: auth,
			})
			if err != nil {
				return fmt.Errorf("error cloning repo: %w", err)
			}

			if err := os.Chdir(repository); err != nil {
				return fmt.Errorf("error chdir-ing into cloned repo")
			}
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
