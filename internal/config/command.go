package config

import "fmt"

// commandFromValue converts the decoded TOML value of a command key into the
// internal union representation. A string is shell mode; an array is direct
// mode.
func commandFromValue(v any) (Command, error) {
	switch t := v.(type) {
	case string:
		return Command{Shell: t}, nil
	case []any:
		argv := make([]string, 0, len(t))
		for i, e := range t {
			s, ok := e.(string)
			if !ok {
				return Command{}, fmt.Errorf("command element %d must be a string, got %T", i+1, e)
			}
			argv = append(argv, s)
		}
		return Command{Argv: argv}, nil
	case []string:
		argv := make([]string, len(t))
		copy(argv, t)
		return Command{Argv: argv}, nil
	default:
		return Command{}, fmt.Errorf("command must be a string (shell mode) or an array of strings (direct mode), got %T", v)
	}
}
