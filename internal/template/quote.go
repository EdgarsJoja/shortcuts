package template

import "strings"

// ShellQuote returns s as a single-quoted POSIX shell word. It always quotes,
// even for strings that look safe, so the result is easy to audit.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// QuoteArgv joins argv into a single-quoted shell-like string. It is used for
// display and to pre-fill direct-mode shortcuts in the TUI; shlex parses the
// result back into the same argv.
func QuoteArgv(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = ShellQuote(a)
	}
	return strings.Join(parts, " ")
}
