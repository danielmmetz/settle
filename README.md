# Settle

Settle is a dotfiles manager. Configure `settle.yaml`, run `settle`, and be on your way.

Please note that settle is in its very early development.
There are no guarantees of any functionality or stability at this time.

See [here for an example config](./settle.yaml).

## Current feature set

**Neovim Support**:
* generate `init.lua` from specified plugins and additional config
* install paq & install specified plugins

**Apt Support**:
* supports installing packages
* runs autoremove

**Brew Support**:
* supports taps, ordinary packages, and casks
* cleans up packages no longer specified

**Zsh Support**:
* configuring history size and history sharing
* variables
* aliases
* functions
* arbitrary .zshrc prefixes & suffixes

**File Symlinking**
* symlink files using relative or absolute paths

**Includes**
* enable modular config by means of _including_ other files into the main configuration
  * resolution order is the listed files for inclusion (in-order), then content in the main config file.
    The last definition wins. Stanzas are taken as all or nothing--no clever merging happens within stanzas.

### Bootstrapping

Clones the specified repo before settling in.

`settle init -repo danielmmetz/settle`

`settle init -repo danielmmetz/settle -auth basic`

`settle init -repo danielmmetz/settle -auth pubkey -private-key /path/to/key`

Don't yet have settle? Install it with:

```bash
curl -sL https://raw.githubusercontent.com/danielmmetz/settle/master/install.sh | bash
```

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

* simplified rollbacks (e.g. `settle history` and `settle rollback <time>`)

### TODOs for docs

* fabricate a legitimate-sounding reason why I made this, then add a "Why?" section
* add examples

### For the project

* add linting
* add tests
