package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Zsh struct {
	Zinit   []string `json:"zinit"`
	History struct {
		Size          int  `json:"size"`
		ShareHistory  bool `json:"share_history"`
		IncAppend     bool `json:"inc_append"`
		IgnoreAllDups bool `json:"ignore_all_dups"`
		IgnoreSpace   bool `json:"ignore_space"`
	} `json:"history"`
	Paths       []string `json:"paths"`
	Variables   []KV     `json:"variables"`
	Aliases     []KV     `json:"aliases"`
	Functions   []KV     `json:"functions"`
	ExtraPrefix string   `json:"extra_prefix"`
	ExtraSuffix string   `json:"extra_suffix"`
}

type KV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (z *Zsh) Ensure(ctx context.Context) error {
	if z == nil {
		return nil
	}

	if err := z.ensureZinit(ctx); err != nil {
		return fmt.Errorf("error ensuring zinit: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	fmt.Println("writing .zshrc")
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(z.String()), 0644); err != nil {
		return fmt.Errorf("error writing .zshrc: %w", err)
	}
	return nil
}

const (
	zinitURL = "https://raw.githubusercontent.com/zdharma-continuum/zinit/master/zinit.zsh"
)

func (z *Zsh) ensureZinit(ctx context.Context) error {
	if len(z.Zinit) == 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	dstPath := filepath.Join(home, ".zinit", "bin", "zinit.zsh")

	_, err = os.Stat(dstPath)
	if err == nil {
		return nil // file already exists
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	fmt.Println("installing Zinit")
	_, err = git.PlainCloneContext(ctx, filepath.Join(home, ".zinit", "bin"), false, &git.CloneOptions{URL: "https://github.com/zdharma-continuum/zinit.git"})
	if err != nil {
		return fmt.Errorf("error cloning zinit: %w", err)
	}
	return nil
}

func (z *Zsh) String() string {
	var sb strings.Builder

	// extra prefix
	sb.WriteString(z.ExtraPrefix)
	sb.WriteString("\n")

	// zinit
	for _, line := range z.Zinit {
		sb.WriteString(fmt.Sprintf("zinit %s\n", line))
	}
	sb.WriteString("\n")

	// history
	if z.History.Size != 0 {
		sb.WriteString("HISTFILE=~/.zsh_history\n")
		sb.WriteString(fmt.Sprintf("HISTSIZE=%d\n", z.History.Size))
		sb.WriteString(fmt.Sprintf("SAVEHIST=%d\n", z.History.Size))
	}
	if z.History.ShareHistory {
		sb.WriteString("setopt SHARE_HISTORY\n")
	}
	if z.History.IncAppend {
		sb.WriteString("setopt INC_APPEND_HISTORY\n")
	}
	if z.History.IgnoreAllDups {
		sb.WriteString("setopt HIST_IGNORE_ALL_DUPS\n")
	}
	if z.History.IgnoreSpace {
		sb.WriteString("setopt HIST_IGNORE_SPACE\n")
	}
	sb.WriteString("\n")

	// path
	if len(z.Paths) > 0 {
		if !contains(z.Paths, "$PATH") {
			z.Paths = append(z.Paths, "$PATH")
		}
		sb.WriteString(fmt.Sprintf("export PATH=%s\n", strconv.Quote(strings.Join(z.Paths, ":"))))
	}

	// variables
	for _, kv := range z.Variables {
		sb.WriteString(fmt.Sprintf("export %s=%s\n", kv.Name, kv.Value))
	}
	sb.WriteString("\n")

	// aliases
	for _, kv := range z.Aliases {
		sb.WriteString(fmt.Sprintf("alias %s=\"%s\"\n", kv.Name, kv.Value))
	}
	sb.WriteString("\n")

	// functions
	for _, kv := range z.Functions {
		sb.WriteString(fmt.Sprintf("function %s() {\n", kv.Name))
		for _, l := range strings.Split(kv.Value, "\n") {
			sb.WriteString(fmt.Sprintf("\t%s\n", l))
		}
		sb.WriteString("}\n")
	}
	sb.WriteString("\n")

	// extra suffix
	sb.WriteString(z.ExtraSuffix)
	sb.WriteString("\n")
	return sb.String()
}

func contains(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}
