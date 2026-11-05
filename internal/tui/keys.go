package tui

import "github.com/charmbracelet/bubbles/key"

// Bindings advertised in the list help bar. The list's default keymap already
// covers navigation, filtering, quitting, and toggling help; these are the
// shortcut-manager actions.
var (
	keyAdd    = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add"))
	keyEdit   = key.NewBinding(key.WithKeys("e", "enter"), key.WithHelp("e/enter", "view/edit"))
	keyDelete = key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d/x", "delete"))
	keyReload = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload"))
	keyTheme  = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme"))
)

func listHelpKeys() []key.Binding {
	return []key.Binding{keyAdd, keyEdit, keyDelete, keyReload, keyTheme}
}
