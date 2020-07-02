package nvim

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielmmetz/settle/pkg/log"
	"github.com/go-git/go-git/v5"
	"golang.org/x/sync/errgroup"
)

type Nvim struct {
	PluginDir string `yaml:"plugin_dir"`
	Plugins   []Plugin
	Config    NvimConfig
}

func (v Nvim) Ensure(logger log.Log) error {
	var wg errgroup.Group
	for _, plugin := range v.Plugins {
		plugin := plugin
		wg.Go(func() error { return v.ensurePlugin(logger, plugin) })
	}
	if err := wg.Wait(); err != nil {
		return err
	}
	return v.ensureInitVim(logger)

}

func (v Nvim) ensurePlugin(logger log.Log, p Plugin) error {
	dst := v.destDir(p)
	_, err := git.PlainOpen(dst)
	if err == git.ErrRepositoryNotExists {
		// do nothing
	} else if err != nil {
		return fmt.Errorf("unable to check for existing repo for %v: %w", p, err)
	} else if err == nil {
		logger.Debug("repo exists, skipping clone for %v", p)
		return nil
	}

	logger.Info("cloning repo %v into %s", p, dst)
	_, err = git.PlainClone(dst, false, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s/%s.git", p.Owner, p.Repo),
	})
	return err
}

func (v Nvim) ensureInitVim(logger log.Log) error {
	if len(v.Plugins) == 0 && v.Config == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	cfgPath := filepath.Join(home, ".config", "nvim", "init.vim")
	logger.Info("writing vim config to %s", cfgPath)
	return ioutil.WriteFile(cfgPath, []byte(v.initVim()), 0755)
}

func (v Nvim) destDir(p Plugin) string {
	return fmt.Sprintf("%s/%s", v.PluginDir, p.Repo)
}

func (v Nvim) initVim() string {
	var vimPlugLines []string
	vimPlugLines = append(vimPlugLines, fmt.Sprintf("call plug#begin('%s')", v.PluginDir))
	for _, plugin := range v.Plugins {
		vimPlugLines = append(vimPlugLines, plugin.toVimPlug())
	}
	vimPlugLines = append(vimPlugLines, "call plug#end()")
	vimPlugLines = append(vimPlugLines, "\n")
	vimPlugLines = append(vimPlugLines, string(v.Config))
	vimPlugLines = append(vimPlugLines, "\n")
	return strings.Join(vimPlugLines, "\n")
}

type Plugin struct {
	Owner string
	Repo  string
}

func (p Plugin) toVimPlug() string {
	return fmt.Sprintf("Plug '%s'", p.String())
}

func (p Plugin) String() string {
	return fmt.Sprintf("%s/%s", p.Owner, p.Repo)
}

func (p *Plugin) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intermediary string
	if err := unmarshal(&intermediary); err != nil {
		return err
	}
	components := strings.Split(intermediary, "/")
	if len(components) != 2 {
		return fmt.Errorf(`expected plugin to resemble "owner/repo": got %s`, intermediary)
	}
	p.Owner = components[0]
	p.Repo = components[1]
	return nil
}

type NvimConfig string

func (c NvimConfig) MarshalJSON() ([]byte, error) {
	if len(c) > 0 {
		return json.Marshal("(present but omitted)")
	}
	return []byte("(empty)"), nil
}
