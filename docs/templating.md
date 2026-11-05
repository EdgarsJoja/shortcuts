# Argument templating

Supported placeholders:

| Placeholder | Meaning |
|---|---|
| `{{args}}` | every argument passed to the shortcut |
| `{{N}}` | the N-th argument (1-based); missing yields an empty value |

Rules:

1. If the command contains **no placeholder**, all invocation arguments are
   appended to the end. This matches shell alias behavior:
   `gst --untracked` → `git status --short --untracked`.
2. In shell mode, every substituted argument is a complete **single-quoted**
   word. Shell metacharacters in arguments are inert.
3. `{{args}}` must be a standalone shell word in shell mode, and the entire argv
   element in direct mode.
4. Placeholders inside single or double quotes are rejected in shell mode
   (`echo "{{1}}"`), because the inserted quoting would be wrong.
5. `{{N}}` may be embedded in an unquoted word: `--file={{1}}`.
6. An escaped placeholder (`\{{1}}`) is treated as literal text.
7. There is no syntax for a literal unescaped `{{` in v1.

Examples:

```toml
[shortcuts.greet]
command = "printf 'hello %s\n' {{1}}"

[shortcuts.copy]
command = "cp {{1}} {{2}}"

[shortcuts.search]
command = "rg -i {{1}} ."
```

Only `{{args}}` and decimal `{{N}}` are supported. Defaults such as
`{{2:-.}}` are rejected rather than silently misinterpreted.

## Direct mode

Each array element is one argv element.

```toml
[shortcuts.co]
command = ["git", "checkout", "{{args}}"]
```

```console
$ co main feature/x
# argv: git checkout main feature/x
```

`{{args}}` must occupy a whole element; `--branch={{args}}` is rejected.
`{{1}}` may be embedded: `"--message={{1}}"`.

## Quoting

`shortcuts run --dry-run <name> <args...>` shows the exact expanded shell
command (or quoted argv) without executing it. Arguments are always quoted even
when they look safe, so the output is easy to audit.
