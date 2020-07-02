package brew

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/danielmmetz/settle/pkg/log"
)

type Brew struct {
	Taps []string
	Pkgs []struct {
		Name string
		Args []string
	}
	Casks []string
}

func (b *Brew) Ensure(ctx context.Context, logger log.Log) error {
	if b == nil {
		return nil
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
	return nil
}

func (b *Brew) brewfile() string {
	var lines []string
	for _, tap := range b.Taps {
		lines = append(lines, fmt.Sprintf(`tap "%s"`, tap))
	}
	for _, pkg := range b.Pkgs {
		lineComponents := []string{fmt.Sprintf(`brew "%s"`, pkg.Name)}
		if len(pkg.Args) > 0 {
			lineComponents = append(lineComponents, ", args: [")
			for _, arg := range pkg.Args {
				lineComponents = append(lineComponents, fmt.Sprintf(`"%s"`, arg))
			}
			lineComponents = append(lineComponents, "]")
		}
		lines = append(lines, strings.Join(lineComponents, ""))
	}
	for _, cask := range b.Casks {
		lines = append(lines, fmt.Sprintf(`cask "%s"`, cask))
	}
	return strings.Join(lines, "\n")
}
