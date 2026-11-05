package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ejoja/shortcuts/internal/theme"
)

var (
	activePalette = theme.Default()

	titleStyle  lipgloss.Style
	labelStyle  lipgloss.Style
	textStyle   lipgloss.Style
	hintStyle   lipgloss.Style
	errorStyle  lipgloss.Style
	statusStyle lipgloss.Style
)

func init() {
	applyTheme(theme.Default())
}

func applyTheme(p theme.Palette) {
	activePalette = p
	titleStyle = paint(fg(p.Accent).Bold(true))
	labelStyle = paint(fg(p.Foreground).Bold(true))
	textStyle = paint(fg(p.Foreground))
	hintStyle = paint(fg(p.Muted))
	errorStyle = paint(fg(p.Error))
	statusStyle = paint(fg(p.Success))
}

// applyThemeByName resolves name, applies it, and returns the canonical name.
// Unknown names fall back to the default palette.
func applyThemeByName(name string) string {
	p, ok := theme.Get(name)
	if !ok {
		p = theme.Default()
	}
	applyTheme(p)
	return p.Name
}

// nextTheme advances the palette and returns the canonical name.
func nextTheme(current string) string {
	p := theme.Next(current)
	applyTheme(p)
	return p.Name
}

// paletteActive reports whether colors should be applied at all. The default
// palette leaves Bubbles' own defaults untouched.
func paletteActive() bool {
	return activePalette.Background != "" || activePalette.Accent != ""
}

func fg(hex string) lipgloss.Style {
	s := lipgloss.NewStyle()
	if hex != "" {
		s = s.Foreground(lipgloss.Color(hex))
	}
	return s
}

// paint applies the palette background while preserving the style's other
// attributes (padding, borders, bold, and so on).
func paint(s lipgloss.Style) lipgloss.Style {
	if activePalette.Background == "" {
		return s
	}
	return s.Background(lipgloss.Color(activePalette.Background))
}

// themed applies hex as a foreground color while preserving the style's other
// attributes.
func themed(s lipgloss.Style, hex string) lipgloss.Style {
	if hex == "" {
		return s
	}
	return s.Foreground(lipgloss.Color(hex))
}

// themedCursor renders a solid accent block with the page color as the
// character color, so it stays visible on both dark and light backgrounds.
func themedCursor(existing lipgloss.Style) lipgloss.Style {
	if activePalette.Background == "" || activePalette.Accent == "" {
		return existing
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(activePalette.Background)).
		Background(lipgloss.Color(activePalette.Accent))
}

// titleBandStyle draws the list title as a filled accent band.
func titleBandStyle() lipgloss.Style {
	s := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	if activePalette.Background == "" || activePalette.Accent == "" {
		return s
	}
	return s.
		Foreground(lipgloss.Color(activePalette.Background)).
		Background(lipgloss.Color(activePalette.Accent))
}

func themeHelpStyles(s help.Styles) help.Styles {
	if !paletteActive() {
		return s
	}
	s.ShortKey = paint(fg(activePalette.Accent)).Bold(true)
	s.ShortDesc = paint(fg(activePalette.Muted))
	s.ShortSeparator = paint(fg(activePalette.Muted))
	s.FullKey = paint(fg(activePalette.Accent)).Bold(true)
	s.FullDesc = paint(fg(activePalette.Muted))
	s.FullSeparator = paint(fg(activePalette.Muted))
	s.Ellipsis = paint(fg(activePalette.Muted))
	return s
}

// fillScreen paints the whole alternate screen with the palette background so
// light themes actually look light instead of showing the terminal background.
//
// Nested Lip Gloss/Bubbles components emit SGR resets mid-line, which clear an
// outer background. We therefore put the background sequence in effect at the
// start of every line and re-apply it after every reset, which also covers
// per-component gaps and right-edge padding.
func fillScreen(content string, width, height int) string {
	if activePalette.Background == "" || width <= 0 || height <= 0 {
		return content
	}
	seq := backgroundSequence(activePalette.Background)
	if seq == "" {
		return content
	}

	const reset = "\x1b[0m"

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Truncate before padding: Bubble Tea truncates at the terminal width,
		// and doing it ourselves keeps a trailing reset from being cut off,
		// which could leak SGR state into the next line.
		line = ansi.Truncate(line, width, "") + reset
		if w := ansi.StringWidth(line); w < width {
			line += strings.Repeat(" ", width-w)
		}
		lines[i] = line
	}
	blank := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, blank)
	}

	for i, line := range lines {
		lines[i] = seq + strings.ReplaceAll(line, reset, reset+seq) + reset
	}
	return strings.Join(lines, "\n")
}

// backgroundSequence returns the ANSI sequence Lip Gloss would use for this
// background, respecting the detected terminal color profile.
func backgroundSequence(hex string) string {
	if hex == "" {
		return ""
	}
	probe := lipgloss.NewStyle().Background(lipgloss.Color(hex)).Render(" ")
	if i := strings.Index(probe, " "); i > 0 {
		return probe[:i]
	}
	return ""
}
