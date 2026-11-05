// Package theme defines the built-in TUI color palettes.
//
// Palettes are plain hex strings so this package stays independent of the TUI
// stack and can be used by CLI commands.
package theme

import "strings"

// Palette is a set of semantic colors. Empty strings mean "use the terminal
// default".
type Palette struct {
	Name       string
	Background string
	Foreground string
	Accent     string
	Accent2    string
	Muted      string
	Error      string
	Success    string
	Warning    string
}

var palettes = []Palette{
	{Name: "default"},
	{
		Name:       "catppuccin-mocha",
		Background: "#1e1e2e",
		Foreground: "#cdd6f4",
		Accent:     "#cba6f7",
		Accent2:    "#89b4fa",
		Muted:      "#7f849c",
		Error:      "#f38ba8",
		Success:    "#a6e3a1",
		Warning:    "#f9e2af",
	},
	{
		Name:       "catppuccin-latte",
		Background: "#eff1f5",
		Foreground: "#4c4f69",
		Accent:     "#8839ef",
		Accent2:    "#1e66f5",
		Muted:      "#8c8fa1",
		Error:      "#d20f39",
		Success:    "#40a02b",
		Warning:    "#df8e1d",
	},
	{
		Name:       "dracula",
		Background: "#282a36",
		Foreground: "#f8f8f2",
		Accent:     "#bd93f9",
		Accent2:    "#8be9fd",
		Muted:      "#6272a4",
		Error:      "#ff5555",
		Success:    "#50fa7b",
		Warning:    "#f1fa8c",
	},
	{
		Name:       "nord",
		Background: "#2e3440",
		Foreground: "#d8dee9",
		Accent:     "#88c0d0",
		Accent2:    "#81a1c1",
		Muted:      "#616e88",
		Error:      "#bf616a",
		Success:    "#a3be8c",
		Warning:    "#ebcb8b",
	},
	{
		Name:       "gruvbox-dark",
		Background: "#282828",
		Foreground: "#ebdbb2",
		Accent:     "#fabd2f",
		Accent2:    "#83a598",
		Muted:      "#928374",
		Error:      "#fb4934",
		Success:    "#b8bb26",
		Warning:    "#fe8019",
	},
	{
		Name:       "tokyo-night",
		Background: "#1a1b26",
		Foreground: "#c0caf5",
		Accent:     "#7aa2f7",
		Accent2:    "#bb9af7",
		Muted:      "#565f89",
		Error:      "#f7768e",
		Success:    "#9ece6a",
		Warning:    "#e0af68",
	},
	{
		Name:       "one-dark",
		Background: "#282c34",
		Foreground: "#abb2bf",
		Accent:     "#61afef",
		Accent2:    "#c678dd",
		Muted:      "#5c6370",
		Error:      "#e06c75",
		Success:    "#98c379",
		Warning:    "#e5c07b",
	},
	{
		Name:       "solarized-dark",
		Background: "#002b36",
		Foreground: "#839496",
		Accent:     "#268bd2",
		Accent2:    "#2aa198",
		Muted:      "#586e75",
		Error:      "#dc322f",
		Success:    "#859900",
		Warning:    "#b58900",
	},
	{
		Name:       "solarized-light",
		Background: "#fdf6e3",
		Foreground: "#657b83",
		Accent:     "#268bd2",
		Accent2:    "#6c71c4",
		Muted:      "#93a1a1",
		Error:      "#dc322f",
		Success:    "#859900",
		Warning:    "#b58900",
	},
}

var byName = func() map[string]Palette {
	m := make(map[string]Palette, len(palettes))
	for _, p := range palettes {
		m[strings.ToLower(p.Name)] = p
	}
	return m
}()

// Get looks a palette up case-insensitively.
func Get(name string) (Palette, bool) {
	p, ok := byName[strings.ToLower(strings.TrimSpace(name))]
	return p, ok
}

// Default returns the no-color palette.
func Default() Palette {
	return palettes[0]
}

// Names returns the built-in theme names in display order.
func Names() []string {
	names := make([]string, len(palettes))
	for i, p := range palettes {
		names[i] = p.Name
	}
	return names
}

// Next returns the palette after name, wrapping around. Unknown names start the
// cycle at the first palette.
func Next(name string) Palette {
	current, ok := Get(name)
	if !ok {
		return palettes[0]
	}
	for i, p := range palettes {
		if p.Name == current.Name {
			return palettes[(i+1)%len(palettes)]
		}
	}
	return palettes[0]
}
