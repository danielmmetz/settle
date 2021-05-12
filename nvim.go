package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Nvim struct {
	PluginDir string     `json:"plugin_dir"`
	Plugins   []Plugin   `json:"plugins"`
	Config    NvimConfig `json:"config"`
}

func (v *Nvim) Ensure(ctx context.Context) error {
	if v == nil {
		return nil
	}

	if err := v.ensureVimPlug(ctx); err != nil {
		return fmt.Errorf("error ensuring vim-plug: %w", err)
	}
	if err := v.ensureInitVim(); err != nil {
		return fmt.Errorf("error ensuring init.vim: %w", err)
	}
	fmt.Println("installing neovim plugins")
	installCmd := exec.CommandContext(ctx, "nvim", "--headless", "+PlugInstall", "+qa")
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running `nvim --headless +PlugInstall +qa`: %w\n%s", err, string(output))
	}
	return nil
}

const (
	vimPlugURL = "https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim"
)

func (v *Nvim) ensureVimPlug(ctx context.Context) error {
	if len(v.Plugins) == 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	dstPath := filepath.Join(home, ".local", "share", "nvim", "site", "autoload", "plug.vim")

	_, err = os.Stat(dstPath)
	if err == nil {
		fmt.Println("skipping vim-plug install: already present")
		return nil // file already exists
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	fmt.Println("installing vim-plug")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, vimPlugURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(dstPath, content, 0755)
}

func (v *Nvim) ensureInitVim() error {
	if len(v.Plugins) == 0 && v.Config == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	cfgPath := filepath.Join(home, ".config", "nvim", "init.vim")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return fmt.Errorf("error making intermediate directories for %s: %w", cfgPath, err)
	}
	fmt.Println("writing vim config to", cfgPath)
	return ioutil.WriteFile(cfgPath, []byte(v.initVim()), 0755)
}

func (v *Nvim) destDir(p Plugin) string {
	return fmt.Sprintf("%s/%s", v.PluginDir, p.Repo)
}

func (v *Nvim) initVim() string {
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
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

func (p Plugin) toVimPlug() string {
	return fmt.Sprintf("Plug '%s'", p.String())
}

func (p Plugin) String() string {
	return fmt.Sprintf("%s/%s", p.Owner, p.Repo)
}

func (p *Plugin) UnmarshalJSON(b []byte) error {
	var intermediary string
	if err := json.Unmarshal(b, &intermediary); err != nil {
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
		return json.Marshal(`"(present but omitted)"`)
	}
	return []byte(`"(empty)"`), nil
}
