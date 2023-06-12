package zsh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

type Zsh struct {
	History    struct {
		Size          int  `json:"size"`
		ShareHistory  bool `json:"share_history"`
		IncAppend     bool `json:"inc_append"`
		IgnoreAllDups bool `json:"ignore_all_dups"`
		IgnoreSpace   bool `json:"ignore_space"`
	} `json:"history"`
	Paths     []string `json:"paths"`
	Variables []KV     `json:"variables"`
	Aliases   []KV     `json:"aliases"`
	Functions []KV     `json:"functions"`
	Prefix    string   `json:"prefix"`
	Suffix    string   `json:"suffix"`
}

type KV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (z *Zsh) Ensure(ctx context.Context) error {
	if z == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	fmt.Println("writing .zshrc")
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(z.String()), 0o644); err != nil {
		return fmt.Errorf("error writing .zshrc: %w", err)
	}
	return nil
}

func (z *Zsh) String() string {
	var sb strings.Builder

	// extra prefix
	sb.WriteString(z.Prefix)
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
		if !slices.Contains(z.Paths, "$PATH") {
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
	sb.WriteString(z.Suffix)
	sb.WriteString("\n")
	return sb.String()
}
