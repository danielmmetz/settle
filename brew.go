package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Brew struct {
	Taps  Taps  `json:"taps"`
	Pkgs  Pkgs  `json:"pkgs"`
	Casks Casks `json:"casks"`
}

func (b *Brew) Ensure(ctx context.Context) error {
	if b == nil {
		return nil
	}

	if err := ensureBrew(ctx); err != nil {
		return fmt.Errorf("error ensuring brew is installed: %w", err)
	}

	f, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("error creating temporary Brewfile: %w", err)
	}
	defer f.Close()
	fmt.Println("writing temporary Brewfile to:", f.Name())
	if _, err := f.WriteString(b.String()); err != nil {
		return err
	}

	fmt.Println("installing packages with `brew bundle`")
	installCmd := exec.CommandContext(ctx, "brew", "bundle", "--file", f.Name())
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running `brew bundle`: %w\n%s", err, string(output))
	}
	fmt.Println("cleaning up orphan packages with `brew bundle cleanup`")
	cleanupCmd := exec.CommandContext(ctx, "brew", "bundle", "cleanup", "--force", "--file", f.Name())
	if output, err := cleanupCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running `brew bundle cleanup`: %w\n%s", err, string(output))
	}
	return nil
}

const brewInstallURL = "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"

func ensureBrew(ctx context.Context) error {
	if err := exec.CommandContext(ctx, "which", "brew").Run(); err == nil {
		return nil
	}
	f, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("error creating temporary file for brew install script: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", brewInstallURL, nil)
	if err != nil {
		return fmt.Errorf("error building request for brew install script: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching brew install script: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("error writing brew install script: %w", err)
	}
	_ = f.Close()

	cmd := exec.CommandContext(ctx, "bash", "-c", f.Name())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error installing brew: %w", err)
	}
	return nil

}

func (b *Brew) String() string {
	var lines []string
	for _, tap := range b.Taps {
		lines = append(lines, tap.String())
	}
	for _, pkg := range b.Pkgs {
		lines = append(lines, pkg.String())
	}
	for _, cask := range b.Casks {
		lines = append(lines, cask.String())
	}
	return strings.Join(lines, "\n")
}

func (b *Brew) equal(other Brew) bool {
	if !b.Taps.equal(other.Taps) {
		return false
	}
	if !b.Pkgs.equal(other.Pkgs) {
		return false
	}
	if !b.Casks.equal(other.Casks) {
		return false
	}
	return true
}

type Taps []Tap

func (t Taps) equal(other Taps) bool {
	if len(t) != len(other) {
		return false
	}
	seen := map[string]bool{}
	for _, t := range t {
		seen[t.String()] = true
	}

	for _, t := range other {
		if !seen[t.String()] {
			return false
		}
	}
	return true
}

func (t *Taps) UnmarshalJSON(b []byte) error {
	var intermediary []Tap
	if err := json.Unmarshal(b, &intermediary); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, tap := range intermediary {
		if seen[string(tap)] {
			return fmt.Errorf("error: contains duplicate tap %s", string(tap))
		}
		seen[string(tap)] = true
	}
	*t = intermediary
	return nil
}

type Tap string

func (t Tap) String() string { return fmt.Sprintf(`tap "%s"`, string(t)) }

type Pkgs []Pkg

func (p Pkgs) equal(other Pkgs) bool {
	if len(p) != len(other) {
		return false
	}
	seen := map[string]bool{}
	for _, p := range p {
		seen[p.String()] = true
	}

	for _, p := range other {
		if !seen[p.String()] {
			return false
		}
	}
	return true
}

func (p *Pkgs) UnmarshalJSON(b []byte) error {
	var intermediary []Pkg
	if err := json.Unmarshal(b, &intermediary); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, pkg := range intermediary {
		if seen[pkg.Name] {
			return fmt.Errorf("error: contains duplicate package %s", pkg.Name)
		}
		seen[pkg.Name] = true
	}
	*p = intermediary
	return nil
}

type Pkg struct {
	Name string   `json:"name"`
	Args []string `json:"args,omitempty"`
}

func (p Pkg) String() string {
	components := []string{fmt.Sprintf(`brew "%s"`, p.Name)}
	if len(p.Args) > 0 {
		components = append(components, ", args: [")
		for _, arg := range p.Args {
			components = append(components, fmt.Sprintf(`"%s"`, arg))
		}
		components = append(components, "]")
	}
	return strings.Join(components, "")
}

type Casks []Cask

func (c Casks) equal(other Casks) bool {
	if len(c) != len(other) {
		return false
	}
	seen := map[string]bool{}
	for _, c := range c {
		seen[c.String()] = true
	}

	for _, c := range other {
		if !seen[c.String()] {
			return false
		}
	}
	return true
}

func (t *Casks) UnmarshalJSON(b []byte) error {
	var intermediary []Cask
	if err := json.Unmarshal(b, &intermediary); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, cask := range intermediary {
		if seen[string(cask)] {
			return fmt.Errorf("error: contains duplicate tap %s", string(cask))
		}
		seen[string(cask)] = true
	}
	*t = intermediary
	return nil
}

type Cask string

func (c Cask) String() string { return fmt.Sprintf(`cask "%s"`, string(c)) }
