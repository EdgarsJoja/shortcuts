# Configuration

Config lives at:

1. `$SHORTCUTS_CONFIG` if set (absolute path), else
2. `$XDG_CONFIG_HOME/shortcuts/config.toml`, else
3. `~/.config/shortcuts/config.toml`.

Shims live at `$SHORTCUTS_BIN_DIR`, else `$XDG_DATA_HOME/shortcuts/bin`, else
`~/.local/share/shortcuts/bin`.

## Schema

```toml
version = 1
theme = "dracula"          # optional TUI palette; default if omitted

[shortcuts.<name>]
command = "..."          # string => shell mode
# or
command = ["argv", "..."] # array  => direct mode
description = "..."       # optional, max 500 characters
```

- A **string** command is executed as `/bin/sh -c <expanded>`.
- An **array** command is executed directly; no shell is involved, so pipes,
  redirects, globs, and `&&` are not interpreted.
- Unknown keys are errors, which catches typos.
- `version` must be `1`.

### Names

`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`, unique case-insensitively (macOS
filesystems). CLI command names (`list`, `run`, `add`, `edit`, `rm`, `init`,
`doctor`, `version`, `help`, `tui`, `shortcuts`, …) are reserved.

### Modes

Shell mode:

```toml
[shortcuts.logs]
command = "journalctl -u {{1}} -f"
```

Direct mode:

```toml
[shortcuts.gst]
command = ["git", "status", "--short"]
```

## File handling

- Config mode is `0600`; parent directories are created `0700`.
- Writes use a temp file, `fsync`, and atomic rename.
- Updates take an advisory lock (`config.toml.lock`), so concurrent CLI
  invocations do not lose each other's changes.
- A symlinked config file is updated at its target.

## Directories

`shortcuts init` creates the config and shim directories and prints the PATH
line. Use `--write` (or `--yes`) to append it to `~/.bashrc` or `~/.zshrc`. The
block is marked and the previous file is backed up as `<rc>.bak`.
