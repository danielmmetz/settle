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
	Zsh4Humans bool `json:"zsh4humans"`
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

	if err := z.ensureZsh4Humans(ctx); err != nil {
		return fmt.Errorf("error ensuring zinit: %w", err)
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

func (z *Zsh) ensureZsh4Humans(ctx context.Context) error {
	if !z.Zsh4Humans {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	zshenv := filepath.Join(home, ".zshenv")

	_, err = os.Stat(zshenv)
	if err == nil {
		return nil // file already exists
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	fmt.Println("writing zsh4humans .zshenv file")
	url := "https://raw.githubusercontent.com/romkatv/zsh4humans/v5/.zshenv"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("building request to fetch zsh4humans .zshenv: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching zsh4humans .zshenv: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading zsh4humans .zshenv body: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("bad response fetching %s (%d): %s", url, resp.StatusCode, string(body))
	}
	if err := os.WriteFile(zshenv, body, 0o644); err != nil {
		return fmt.Errorf("writing .zshenv: %w", err)
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
