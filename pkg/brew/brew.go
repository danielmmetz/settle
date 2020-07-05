package brew

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/danielmmetz/settle/pkg/log"
	"github.com/danielmmetz/settle/pkg/store"
	"gopkg.in/yaml.v2"
)

type Brew struct {
	Taps  Taps
	Pkgs  Pkgs
	Casks Casks
}

func (b *Brew) Ensure(ctx context.Context, logger log.Log, store store.Store) error {
	if b == nil {
		return nil
	}

	previous, err := store.Content(ctx, "Brewfile")
	if err == nil {
		var parsedPrev Brew
		if err := yaml.Unmarshal([]byte(previous), &parsedPrev); err == nil {
			if b.equal(parsedPrev) {
				logger.Debug("skipping brew ensure: no changes")
				return nil
			} else {
				logger.Debug("brew: cache miss: detected changes since last run")
			}
		} else {
			logger.Debug("brew: cache miss: error unmarshaling previous run from store: %w", err)
		}
	} else {
		logger.Debug("brew: cache miss: error checking cache: %w", err)
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("error creating temporary Brewfile: %w", err)
	}
	if _, err := f.WriteString(b.brewfile()); err != nil {
		return err
	}
	logger.Debug("wrote temporary Brewfile to: %s", f.Name())

	logger.Info("installing packages with `brew bundle`")
	installCmd := exec.CommandContext(ctx, "brew", "bundle", "--file", f.Name())
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("error running `brew bundle`: %w", err)
	}
	logger.Info("cleaning up orphan packages with `brew bundle cleanup`")
	cleanupCmd := exec.CommandContext(ctx, "brew", "bundle", "cleanup", "--force", "--file", f.Name())
	if err := cleanupCmd.Run(); err != nil {
		return fmt.Errorf("error running `brew bundle cleanup`: %w", err)
	}

	content, err := yaml.Marshal(b)
	if err != nil {
		logger.Info("error marshaling Brewfile for storage: %w", err)
		return nil
	}
	logger.Debug("writing Brewfile to store")
	if err := store.SetContent(ctx, "Brewfile", string(content)); err != nil {
		logger.Info("error writing Brewfile to store: %w", err)
	}
	return nil
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

func (b *Brew) brewfile() string {
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

func (t *Taps) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intermediary []Tap
	if err := unmarshal(&intermediary); err != nil {
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

func (p *Pkgs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intermediary []Pkg
	if err := unmarshal(&intermediary); err != nil {
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
	Name string
	Args []string
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

func (t *Casks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intermediary []Cask
	if err := unmarshal(&intermediary); err != nil {
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
