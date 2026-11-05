package template

import "strings"

// ExpandShell renders parts for execution by a POSIX shell. Every substituted
// argument is a complete single-quoted word, so shell metacharacters in user
// input are inert.
func ExpandShell(parts []Part, args []string) string {
	var b strings.Builder
	for _, p := range parts {
		switch p.Kind {
		case Literal:
			b.WriteString(p.Text)
		case Args:
			for i, a := range args {
				if i > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(ShellQuote(a))
			}
		case Positional:
			if p.Index-1 < len(args) {
				b.WriteString(ShellQuote(args[p.Index-1]))
			} else {
				b.WriteString("''")
			}
		}
	}
	return b.String()
}

// ExpandDirectElement renders one argv element for direct mode. A {{args}}
// part must be the entire element and expands to zero or more elements; any
// other element produces exactly one element, including when the template is
// empty.
func ExpandDirectElement(parts []Part, args []string) []string {
	for _, p := range parts {
		if p.Kind == Args {
			out := make([]string, len(args))
			copy(out, args)
			return out
		}
	}

	var b strings.Builder
	for _, p := range parts {
		switch p.Kind {
		case Literal:
			b.WriteString(p.Text)
		case Positional:
			if p.Index-1 < len(args) {
				b.WriteString(args[p.Index-1])
			}
		}
	}
	return []string{b.String()}
}

// AppendShell renders a suffix for commands without placeholders, preserving
// classic alias behavior: extra arguments are appended to the command.
func AppendShell(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = ShellQuote(a)
	}
	return command + " " + strings.Join(quoted, " ")
}
