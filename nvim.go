package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Nvim struct {
	Plugins []Plugin   `json:"plugins"`
	Config  NvimConfig `json:"config"`
}

func (v *Nvim) Ensure(ctx context.Context) error {
	if v == nil {
		return nil
	}

	if err := v.ensureInitVim(); err != nil {
		return fmt.Errorf("error ensuring init.lua: %w", err)
	}
	fmt.Println("installing neovim plugins")
	installCmd := exec.CommandContext(ctx, "nvim", "--headless", "+PaqSync", "+qa")
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running neovim plugin sync commands: %w\n%s", err, string(output))
	}
	return nil
}

func (v *Nvim) ensureInitVim() error {
	if len(v.Plugins) == 0 && v.Config == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home dir: %w", err)
	}
	cfgPath := filepath.Join(home, ".config", "nvim", "init.lua")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return fmt.Errorf("error making intermediate directories for %s: %w", cfgPath, err)
	}
	fmt.Println("writing vim config to", cfgPath)
	return ioutil.WriteFile(cfgPath, []byte(v.initLua()), 0755)
}

const paqBootstrap = `-- boostrap paq
local fn = vim.fn
local install_path = fn.stdpath('data') .. '/site/pack/paqs/start/paq-nvim'
if fn.empty(fn.glob(install_path)) > 0 then
  fn.system({'git', 'clone', '--depth=1', 'https://github.com/savq/paq-nvim.git', install_path})
end
`

func (v *Nvim) initLua() string {
	var pluginLines []string
	pluginLines = append(pluginLines, paqBootstrap)
	pluginLines = append(pluginLines, `require "paq" {`)
	for _, plugin := range v.Plugins {
		pluginLines = append(pluginLines, fmt.Sprintf(`  "%v";`, plugin))
	}
	pluginLines = append(pluginLines, "}", "\n")
	lines := append(pluginLines, string(v.Config))
	return strings.Join(lines, "\n")
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
		return json.Marshal(`"(omitted for brevity)"`)
	}
	return []byte(`"(empty)"`), nil
}
