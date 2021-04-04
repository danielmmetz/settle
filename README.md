# Settle

Settle is a dotfiles manager. Configure `settle.yaml`, run `settle`, and be on your way.

Please note that settle is in its very early development.
There are no guarantees of any functionality or stability at this time.

See [my personal settle.yaml](https://github.com/danielmmetz/dotfiles/blob/master/settle.yaml)
for an example config.

## Current feature set

**Neovim Support**:
* generate `init.vim` from specified plugins and additional config
* install vim-plug & install specified plugins

**Apt Support**:
* supports installing packages
* runs autoremove

**Brew Support**:
* supports taps, ordinary packages, and casks
* cleans up packages no longer specified

**Zsh Support**:
* plugins using zinit
* configuring history size and history sharing
* variables
* aliases
* functions
* arbitrary .zshrc prefixes & suffixes

**File Symlinking**
* symlink files using relative or absolute paths

### Run history

After a successful run, a copy of that run's `settle.yaml` is backed up to `~/.local/share/settle`.
This enables a relatively easy process to restore a prior good config.

### Sticky config files

After a successful run, `settle` remembers the config file it used.
When run without specifying a `--config` argument,
`settle` will default to using the path to the last successfully applied config file.

This allows users to update and re-apply their config without needing to worry about their working directory,
and allows a user to more easily maintain multiple config files in a single directory.

## Future features

* simplified bootstrapping (e.g. `settle init github.com/<user>/<dofiles-repo>`)
* modular config by allowing config files to reference other config (i.e. `include`)
* simplified rollbacks (e.g. `settle history` and `settle rollback <time>`)

### TODOs for docs

* add a "Why?" section
* add examples

### For the project

* add linting
* add tests
