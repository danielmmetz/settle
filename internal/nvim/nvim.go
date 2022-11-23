package nvim

import (
	"context"
	"encoding/json"
	"fmt"
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
	installCmd := exec.CommandContext(ctx, "nvim", "--headless", "+PaqInstall", "+qa")
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
	return os.WriteFile(cfgPath, []byte(v.initLua()), 0755)
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
		pluginLines = append(pluginLines, fmt.Sprintf(`  %v;`, plugin))
	}
	pluginLines = append(pluginLines, "}", "\n")
	lines := append(pluginLines, string(v.Config))
	return strings.Join(lines, "\n")
}

type Plugin struct {
	Name string `json:"name"`
	Opt  bool   `json:"opt,omitempty"`
	Run  string `json:"run,omitempty"`
}

func (p Plugin) String() string {
	components := []string{fmt.Sprintf(`"%s"`, p.Name)}
	if p.Opt {
		components = append(components, "opt=true")
	}
	if p.Run != "" {
		components = append(components, fmt.Sprintf(`run="%s"`, p.Run))
	}
	return fmt.Sprintf("{%s}", strings.Join(components, ", "))
}

type NvimConfig string

func (c NvimConfig) MarshalJSON() ([]byte, error) {
	if len(c) > 0 {
		return json.Marshal(`"(omitted for brevity)"`)
	}
	return []byte(`"(empty)"`), nil
}
