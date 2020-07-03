# Settle

Settle is a dotfiles manager. Configure `settle.yaml`, run `settle`, and be on your way.

Please note that settle is in its very early development.
There are no guarantees of any functionality or stability at this time.

## Current feature set

* connect to sqlite (not yet used)
* parse configuration file


**File Symlinking**:
* ensure symlinks within configuration file

**Neovim Support**:
* clone (neo)vim plugins into a specified directory
* install vim-plug
* generate `init.vim` from specified plugins and additional config

**Brew Support**:
* supports taps, ordinary packages, and casks
* cleans up packages no longer specified

## Intended feature set

* garbage collection: files created by a previous invocation but no longer preset in config are deleted
  - in this sense, the resulting system should not be "workable" if a depedency is removed from config
  - it also means that settle can clean-up after itself, removing crumbs from previous runs
* multi-platform package management
* zsh is configurable with plugins
* support across macos and at least a single linux distro (one of arch, fedora, or solus)
* make bootstrapping easy (e.g. `settle init github.com/<user>/<dofiles-repo>`)
* config "profiles" by allowing config to reference other config

## Stretch goals

* a way to represent `$HOME` whether as a singular allowed substitution or via full template support
* support for retrieving secure tokens (e.g. github token or ssh keys)


### TODOs for existing features

#### Files

* make the whole process atomic (google may have a library for this, chezmoi may use it)
* register established mappings in db
* garbage collect for db contents no longer in mapping

### TODOs for docs

* add a "Why?" section
* add examples

### For the project

* add linting
* add tests
