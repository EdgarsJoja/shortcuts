// Package template implements the small argument language used by shortcuts.
//
// Supported placeholders are {{args}} and {{1}}..{{N}}.  Parsing is
// deliberately strict: ambiguous placements are rejected rather than guessed
// at, because arguments are inserted into a shell command.
package template

import (
	"fmt"
	"strconv"
	"strings"
)

// Kind identifies a parsed template part.
type Kind int

const (
	// Literal is text copied verbatim.
	Literal Kind = iota
	// Args represents {{args}}.
	Args
	// Positional represents {{N}}.
	Positional
)

// Part is one piece of a parsed template.
type Part struct {
	Kind   Kind
	Text   string // Literal text
	Index  int    // 1-based index for Positional
	Start  int    // byte offset in the source template
	End    int    // exclusive byte offset in the source template
	Quoted bool   // placeholder started inside a shell quote
}

// HasPlaceholder reports whether parts contains any placeholder.
func HasPlaceholder(parts []Part) bool {
	for _, p := range parts {
		if p.Kind != Literal {
			return true
		}
	}
	return false
}

// Parse splits tmpl into parts. When shell is true, shell quoting rules are
// enforced: placeholders may not appear inside quotes, and {{args}} must be a
// standalone word. When shell is false, {{args}} must be an entire argv
// element.
func Parse(tmpl string, shell bool) ([]Part, error) {
	var parts []Part
	literalStart := 0
	inSingle, inDouble := false, false

	for i := 0; i < len(tmpl); {
		c := tmpl[i]

		// Backslash escapes the next byte everywhere except inside single
		// quotes. Escaped braces therefore stay literal.
		if c == '\\' && !inSingle {
			i += 2
			continue
		}
		if c == '\'' && !inDouble {
			inSingle = !inSingle
			i++
			continue
		}
		if c == '"' && !inSingle {
			inDouble = !inDouble
			i++
			continue
		}

		if c == '{' && i+1 < len(tmpl) && tmpl[i+1] == '{' {
			rel := strings.Index(tmpl[i+2:], "}}")
			if rel < 0 {
				return nil, fmt.Errorf("unmatched '{{' at byte %d", i)
			}
			inner := tmpl[i+2 : i+2+rel]
			end := i + 2 + rel + 2

			if i > literalStart {
				parts = append(parts, Part{
					Kind:  Literal,
					Text:  tmpl[literalStart:i],
					Start: literalStart,
					End:   i,
				})
			}

			p := Part{Start: i, End: end, Quoted: inSingle || inDouble}
			switch {
			case inner == "args":
				p.Kind = Args
			case inner == "":
				return nil, fmt.Errorf("empty placeholder at byte %d", i)
			default:
				n, err := strconv.Atoi(inner)
				if err != nil || n <= 0 || n > 9999 {
					return nil, fmt.Errorf("invalid placeholder {{%s}} at byte %d", inner, i)
				}
				p.Kind = Positional
				p.Index = n
			}

			if shell && p.Quoted {
				return nil, fmt.Errorf("placeholder {{%s}} at byte %d is inside quotes", inner, i)
			}

			if p.Kind == Args {
				if shell {
					before := byte(' ')
					if i > 0 {
						before = tmpl[i-1]
					}
					after := byte(' ')
					if end < len(tmpl) {
						after = tmpl[end]
					}
					if !isSpace(before) || !isSpace(after) {
						return nil, fmt.Errorf("{{args}} at byte %d must be a standalone word", i)
					}
				} else if i != 0 || end != len(tmpl) {
					return nil, fmt.Errorf("{{args}} must be the entire command element")
				}
			}

			parts = append(parts, p)
			i = end
			literalStart = end
			continue
		}
		i++
	}

	if literalStart < len(tmpl) {
		parts = append(parts, Part{
			Kind:  Literal,
			Text:  tmpl[literalStart:],
			Start: literalStart,
			End:   len(tmpl),
		})
	}
	return parts, nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
