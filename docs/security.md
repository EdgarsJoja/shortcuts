# Security notes

Threat model: the **shortcut definition is trusted user configuration**;
**invocation arguments are untrusted input**.

## Shell mode

Arguments are expanded as complete single-quoted words:

```sh
# template
kubectl apply -f {{args}}
# invocation with one argument: ; rm -rf ~
kubectl apply -f '; rm -rf ~'
```

Command substitution, backticks, globs, `&&`, newlines, and quotes inside an
argument are inert because they remain inside single quotes.

The quoting strategy can only fail if the template puts a placeholder in a
context where a single-quoted word is not one word. Validation rejects:

- placeholders inside single or double quotes;
- `{{args}}` embedded inside a larger word;
- escaped placeholders are treated as literal text (they are not expanded).

Here-docs containing placeholders are not supported.

## Direct mode

Direct mode never invokes a shell. Each argv element is passed verbatim to
`execve`, so shell syntax in an argument has no special meaning. It is the safer
choice for commands where shell features are unnecessary.

## Environment

Shortcuts inherit the caller's environment. Shell mode therefore expands `$HOME`
and similar expressions at run time, as a normal shell command would. Do not put
secrets in shortcut definitions: the config file is mode `0600`, but
`shortcuts show` and `--dry-run` will print command text.

## Recursion and shims

A shortcut that invokes its own name through `PATH` would recurse. A
`SHORTCUTS_DEPTH` counter is set for each invocation and capped at 16.

Shim management will not overwrite regular files or foreign symlinks without
`--force`. `doctor --fix` only removes symlinks whose target is named
`shortcuts` and that are no longer configured.
