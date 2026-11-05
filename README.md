# shortcuts

A small Go CLI for creating and managing command shortcuts, with a Bubble Tea TUI
for editing them. Shortcuts behave like shell aliases but are shell-agnostic,
managed in one TOML file, and invoked directly by name.

```console
$ shortcuts add deploy -c 'kubectl apply -f {{args}}' -d "Apply a manifest"
$ deploy prod.yaml
```

## How it works

```
~/.config/shortcuts/config.toml          configuration (mode 0600)
~/.local/share/shortcuts/bin/deploy  ->  /usr/local/bin/shortcuts
```

Each shortcut is a symlink in a managed directory. On startup the binary checks
`basename(argv[0])`:

- `shortcuts` → normal CLI / TUI.
- anything else → look up that name in the config and execute it.

Because the symlink carries the name, creating or editing a shortcut takes effect
immediately with no shell re-sourcing and no generated shell code.

## Build

Dependencies are pinned by `go.mod`/`go.sum`. With Go 1.26+:

```console
$ go build -o shortcuts .
$ ./shortcuts version
```

`make build` does the same. Run `go mod tidy` only when dependencies change (it
requires network access). The binary is self-contained for Linux and macOS.

## Quickstart

```console
$ shortcuts init              # show the PATH line to add to your shell rc
$ shortcuts init --write      # or append it automatically (bash/zsh)
$ exec $SHELL                 # or open a new terminal

$ shortcuts add gst --argv 'git status --short' -d "Short git status"
$ shortcuts add deploy -c 'kubectl apply -f {{args}}' -d "Apply manifests"

$ gst
$ deploy prod.yaml
$ shortcuts                  # open the TUI if stdout is a terminal
```

If you do not want to modify `PATH`, `shortcuts run <name> [args...]` always
works.

## CLI

```
shortcuts                         TUI when interactive, otherwise help
shortcuts tui
shortcuts list [--long] [--json]
shortcuts show <name> [--json]
shortcuts run <name> [--dry-run] [--json] [--] [args...]
shortcuts add <name> (-c|--command CMD | --argv ARGS) [-d TEXT] [--force]
shortcuts edit <name> [field flags] [--rename NEW] [--config] [--force]
shortcuts rm <name> [--force]
shortcuts init [--print|--write|--yes] [--file PATH] [--shell NAME] [--repair]
shortcuts doctor [--json] [--fix]
shortcuts version [--json]
```

- `--command` creates a **shell-mode** shortcut (pipes, redirects, `&&` work).
- `--argv` creates a **direct-mode** shortcut; the string is split with POSIX
  shell-like quoting and executed without a shell.
- Flags for `run` must come before the shortcut name; anything after the name is
  passed to the shortcut verbatim.

## TUI

```
a            add a shortcut
e / enter    view/edit details
d / x        delete (confirmation required)
/            filter
r            reload config from disk
t            cycle themes
?            toggle full help
q / ctrl+c   quit
tab          next field in the form
ctrl+t       toggle shell/direct mode in the form
ctrl+s       save
esc          cancel / back
```

## Configuration

See [docs/config.md](docs/config.md). Minimal example:

```toml
version = 1

[shortcuts.deploy]
command = "kubectl apply -f {{args}}"
description = "Apply a manifest"

[shortcuts.gst]
command = ["git", "status", "--short"]
description = "Short git status"
```

A string command runs through `/bin/sh -c`; an array runs directly with no
shell.

## Templating

See [docs/templating.md](docs/templating.md).

- `{{args}}` — all invocation arguments, individually quoted.
- `{{1}}`, `{{2}}`, … — one positional argument (missing → empty string).
- With no placeholder, arguments are appended, like a shell alias.
- Shell-mode placeholders may not appear inside quotes. `{{args}}` must be a
  standalone word.

## Security

See [docs/security.md](docs/security.md). Shortcut definitions are trusted;
invocation arguments are not. Shell-mode and direct-mode arguments are handled
differently by design.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | config, validation, or execution-start error |
| 2 | CLI usage error |
| 127 | unknown shim name |
| 128+N | child terminated by signal N |
| other | child's exit code, passed through |

## Repairing shims

Shims point at the absolute path of the binary. If you move or reinstall the
binary, run:

```console
$ shortcuts doctor --fix
```

This repoints managed links and prunes stale ones. It never deletes regular
files or symlinks that do not point at a `shortcuts` binary.

## Themes

The TUI ships with these palettes:

`default`, `catppuccin-mocha`, `catppuccin-latte`, `dracula`, `nord`,
`gruvbox-dark`, `tokyo-night`, `one-dark`, `solarized-dark`, `solarized-light`.

- Press `t` in the TUI to cycle themes; the choice is saved automatically.
- `shortcuts theme` lists the current and available themes.
- `shortcuts theme dracula` sets it.
- Or set it in config:

```toml
theme = "dracula"
```

Unknown theme names fall back to `default` in the TUI. Dark palettes paint a
dark base background and light palettes (`catppuccin-latte`, `solarized-light`)
paint a light one, including the full alternate screen, list, form fields,
filter input, and help bar. Colors degrade automatically on terminals without
truecolor support.

## Troubleshooting

### `zsh: command not found: <shortcut>`

The shortcut was saved but its shim directory is not on your shell's `PATH`
(creating a shortcut does not modify `PATH`). Diagnose with:

```console
$ shortcuts doctor
$ shortcuts show <shortcut> --json
```

If doctor reports the shim directory is not in `PATH`, run:

```console
$ shortcuts init --write --shell zsh   # or bash
$ exec zsh
```

Or add the printed line to `~/.zshrc` manually:

```sh
export PATH="$HOME/.local/share/shortcuts/bin:$PATH"
```

If doctor reports a missing or stale shim, rebuild/move the binary and run
`shortcuts doctor --fix`. Also check that `add` and your shell use the same
user and `XDG_DATA_HOME` (running `add` under `sudo` writes shims to root's
home).

## Development

```console
$ go test -race ./...
$ go vet ./...
$ gofmt -l .
$ ./scripts/e2e.sh
```

The tests use temporary `HOME`/`XDG_*` directories and never touch your real
configuration. Shell-mode argument handling is covered by injection and quoting
round-trip tests.

## v1 scope

Included: shell/direct modes, argument templating, TUI management, PATH shims,
`doctor`, strict config validation, atomic + locked writes.

Not included yet: Windows, fish shell, importing existing aliases, usage
statistics, shell completion generation, per-shortcut env/cwd overrides.
