# Settle

Settle is a dotfiles manager. Configure `settle.yaml`, run `settle`, and be on your way.

Please note that settle is in its very early development.
There are no guarantees of any functionality or stability at this time.

## Current feature set

**Neovim Support**:
* clone (neo)vim plugins into a specified directory
* generate `init.vim` from specified plugins and additional config
* install vim-plug & install specified plugins

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

## Future features

* support across macos and at least a single linux distro (one of arch, fedora, or solus)
* simplified bootstrapping (e.g. `settle init github.com/<user>/<dofiles-repo>`)
* config "profiles" by allowing config to reference other config (i.e. `include`)
* simplified rollbacks (e.g. `settle history` and `settle rollback <time>`)

### TODOs for existing features

### TODOs for docs

* add a "Why?" section
* add examples

### For the project

* add linting
* add tests
* executable distribution
