// Package tui implements the interactive shortcut manager.
package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/shlex"

	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/shim"
	"github.com/ejoja/shortcuts/internal/template"
)

type screen int

const (
	screenList screen = iota
	screenForm
	screenDetail
	screenConfirm
)

// Run starts the TUI. The caller is responsible for TTY checks.
func Run(store *config.Store, mgr *shim.Manager, in io.Reader, out io.Writer) error {
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	m := newModel(store, mgr, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(in), tea.WithOutput(out))
	_, err = p.Run()
	return err
}

// Model is the root Bubble Tea model.
type Model struct {
	store *config.Store
	mgr   *shim.Manager
	cfg   config.Config

	scr  screen
	list list.Model
	form formModel

	oldName string
	detail  string
	confirm string

	width, height int
	status        string
	err           string
	stamp         fileStamp
	themeName     string
}

func newModel(store *config.Store, mgr *shim.Manager, cfg config.Config) Model {
	themeName := applyThemeByName(cfg.Theme)
	m := Model{
		store:     store,
		mgr:       mgr,
		cfg:       cfg,
		width:     80,
		height:    24,
		themeName: themeName,
	}
	m.list = newList(cfg, m.width, m.height-2)
	m.stamp = statStamp(store.Path())
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, max(1, msg.Height-2))
		m.form.setWidth(msg.Width)
		return m, nil

	case savedMsg:
		if msg.err != nil {
			if m.scr == screenForm {
				m.form.err = msg.err.Error()
			} else {
				m.err = msg.err.Error()
				m.scr = screenList
				m.confirm = ""
			}
			return m, nil
		}
		m.cfg = msg.cfg
		m.err = ""
		m.status = msg.status
		m.stamp = msg.stamp
		m.scr = screenList
		m.rebuildList()
		return m, nil
	}

	switch m.scr {
	case screenForm:
		return m.updateForm(msg)
	case screenDetail:
		if km, ok := msg.(tea.KeyMsg); ok {
			return m.updateDetail(km)
		}
		return m, nil
	case screenConfirm:
		if km, ok := msg.(tea.KeyMsg); ok {
			return m.updateConfirm(km)
		}
		return m, nil
	default:
		return m.updateList(msg)
	}
}

func (m Model) View() string {
	var v string
	switch m.scr {
	case screenForm:
		v = m.formView()
	case screenDetail:
		v = m.detailView()
	case screenConfirm:
		v = m.confirmView()
	default:
		v = m.list.View()
		if m.status != "" {
			v += "\n" + statusStyle.Render(m.status)
		}
		if m.err != "" {
			v += "\n" + errorStyle.Render(m.err)
		}
	}
	return fillScreen(v, m.width, m.height)
}

// --- list screen ---------------------------------------------------------

type shortcutItem struct {
	name string
	sc   config.Shortcut
}

func (i shortcutItem) Title() string {
	return fmt.Sprintf("%s  [%s]", i.name, i.sc.Command.Mode())
}

func (i shortcutItem) Description() string {
	cmd := commandDisplay(i.sc)
	if i.sc.Description == "" {
		return cmd
	}
	return i.sc.Description + " - " + cmd
}

func (i shortcutItem) FilterValue() string {
	return i.name + " " + i.sc.Description
}

func commandDisplay(sc config.Shortcut) string {
	if sc.Command.Argv != nil {
		return template.QuoteArgv(sc.Command.Argv)
	}
	return sc.Command.Shell
}

func newList(cfg config.Config, w, h int) list.Model {
	items := make([]list.Item, 0, len(cfg.Shortcuts))
	for _, name := range cfg.Names() {
		items = append(items, shortcutItem{name: name, sc: cfg.Shortcuts[name]})
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = paint(themed(delegate.Styles.NormalTitle, activePalette.Foreground))
	delegate.Styles.NormalDesc = paint(themed(delegate.Styles.NormalDesc, activePalette.Muted))
	delegate.Styles.SelectedTitle = paint(themed(delegate.Styles.SelectedTitle, activePalette.Accent))
	delegate.Styles.SelectedDesc = paint(themed(delegate.Styles.SelectedDesc, activePalette.Accent2))
	if activePalette.Accent != "" {
		delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.BorderForeground(lipgloss.Color(activePalette.Accent))
	}
	delegate.Styles.DimmedTitle = paint(themed(delegate.Styles.DimmedTitle, activePalette.Muted))
	delegate.Styles.DimmedDesc = paint(themed(delegate.Styles.DimmedDesc, activePalette.Muted))
	delegate.Styles.FilterMatch = paint(themed(delegate.Styles.FilterMatch, activePalette.Warning))

	l := list.New(items, delegate, w, max(1, h))
	l.Title = "Shortcuts"
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)

	if paletteActive() {
		l.Styles.Title = titleBandStyle()
	} else {
		// The default theme is intentionally neutral: drop Bubbles' fixed
		// blue title band and purple filter prompt.
		l.Styles.Title = lipgloss.NewStyle().Bold(true).Padding(0, 1)
		l.Styles.FilterPrompt = lipgloss.NewStyle()
		l.Styles.FilterCursor = lipgloss.NewStyle()
		l.FilterInput.PromptStyle = lipgloss.NewStyle()
		l.FilterInput.Cursor.Style = lipgloss.NewStyle()
	}
	l.Styles.TitleBar = paint(l.Styles.TitleBar)
	l.Styles.Spinner = paint(themed(l.Styles.Spinner, activePalette.Accent))
	l.Styles.FilterPrompt = paint(themed(l.Styles.FilterPrompt, activePalette.Accent))
	l.Styles.FilterCursor = themed(l.Styles.FilterCursor, activePalette.Accent2)
	l.Styles.DefaultFilterCharacterMatch = paint(themed(l.Styles.DefaultFilterCharacterMatch, activePalette.Warning))
	l.Styles.StatusBar = paint(themed(l.Styles.StatusBar, activePalette.Muted))
	l.Styles.StatusEmpty = paint(themed(l.Styles.StatusEmpty, activePalette.Muted))
	l.Styles.StatusBarActiveFilter = paint(themed(l.Styles.StatusBarActiveFilter, activePalette.Accent))
	l.Styles.StatusBarFilterCount = paint(themed(l.Styles.StatusBarFilterCount, activePalette.Muted))
	l.Styles.NoItems = paint(themed(l.Styles.NoItems, activePalette.Muted))
	l.Styles.PaginationStyle = paint(themed(l.Styles.PaginationStyle, activePalette.Muted))
	l.Styles.HelpStyle = paint(l.Styles.HelpStyle)
	l.Styles.ActivePaginationDot = paint(themed(l.Styles.ActivePaginationDot, activePalette.Accent))
	l.Styles.InactivePaginationDot = paint(themed(l.Styles.InactivePaginationDot, activePalette.Muted))
	l.Styles.ArabicPagination = paint(themed(l.Styles.ArabicPagination, activePalette.Muted))
	l.Styles.DividerDot = paint(themed(l.Styles.DividerDot, activePalette.Muted))

	// Both are exported in Bubbles v0.21, so the built-in help and filter
	// input can follow the theme too.
	l.Help.Styles = themeHelpStyles(l.Help.Styles)
	if paletteActive() {
		l.FilterInput.PromptStyle = paint(fg(activePalette.Accent))
		l.FilterInput.TextStyle = paint(fg(activePalette.Foreground))
		l.FilterInput.PlaceholderStyle = paint(fg(activePalette.Muted))
		l.FilterInput.CompletionStyle = paint(fg(activePalette.Muted))
		l.FilterInput.Cursor.Style = themedCursor(l.FilterInput.Cursor.Style)
		l.FilterInput.Cursor.TextStyle = l.FilterInput.TextStyle
	}

	l.AdditionalShortHelpKeys = listHelpKeys
	// Bubbles v1.0.0 changed this hook to a flat []key.Binding.
	l.AdditionalFullHelpKeys = listHelpKeys
	return l
}

func (m *Model) rebuildList() {
	selected := ""
	if it, ok := m.list.SelectedItem().(shortcutItem); ok {
		selected = it.name
	}
	m.list = newList(m.cfg, m.width, max(1, m.height-2))
	if selected == "" {
		return
	}
	for i, it := range m.list.Items() {
		if si, ok := it.(shortcutItem); ok && si.name == selected {
			m.list.Select(i)
			return
		}
	}
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && m.list.FilterState() != list.Filtering {
		switch km.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a":
			return m.startForm("")
		case "enter", "e":
			if it, ok := m.list.SelectedItem().(shortcutItem); ok {
				m.detail = it.name
				m.scr = screenDetail
				return m, nil
			}
		case "d", "x":
			if it, ok := m.list.SelectedItem().(shortcutItem); ok {
				m.confirm = it.name
				m.scr = screenConfirm
				return m, nil
			}
		case "r":
			cfg, err := m.store.Load()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.cfg = cfg
			m.themeName = applyThemeByName(cfg.Theme)
			m.rebuildList()
			m.stamp = statStamp(m.store.Path())
			m.status = "reloaded"
			m.err = ""
			return m, nil
		case "t":
			name := nextTheme(m.themeName)
			m.themeName = name
			m.cfg.Theme = name
			status := "theme: " + name
			if err := m.store.Update(func(c *config.Config) error {
				c.Theme = name
				return nil
			}); err != nil {
				status += " (not saved: " + err.Error() + ")"
			}
			m.stamp = statStamp(m.store.Path())
			m.rebuildList()
			m.status = status
			m.err = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// --- form screen ---------------------------------------------------------

type formModel struct {
	name        textinput.Model
	description textinput.Model
	command     textinput.Model

	focus   int
	direct  bool
	editing bool
	width   int
	err     string
}

func newInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	if paletteActive() {
		ti.PromptStyle = paint(fg(activePalette.Accent))
		ti.TextStyle = paint(fg(activePalette.Foreground))
		ti.PlaceholderStyle = paint(fg(activePalette.Muted))
		ti.CompletionStyle = paint(fg(activePalette.Muted))
		ti.Cursor.Style = themedCursor(ti.Cursor.Style)
		ti.Cursor.TextStyle = ti.TextStyle
	} else {
		// Neutral default: no purple prompt, keep text/placeholder at the
		// terminal's own colors.
		ti.PromptStyle = lipgloss.NewStyle()
		ti.TextStyle = lipgloss.NewStyle()
	}
	return ti
}

func newForm() formModel {
	name := newInput("deploy")
	name.CharLimit = 64

	desc := newInput("Apply a Kubernetes manifest")
	desc.CharLimit = 500

	command := newInput("kubectl apply -f {{args}}")

	return formModel{name: name, description: desc, command: command, width: 80}
}

func (f *formModel) setWidth(w int) {
	f.width = w
	f.resize()
}

func (f *formModel) resize() {
	w := f.width - 4
	if w < 20 {
		w = 20
	}
	f.name.Width = w
	f.description.Width = w
	f.command.Width = w
}

func (f *formModel) applyFocus() tea.Cmd {
	f.name.Blur()
	f.command.Blur()
	f.description.Blur()
	switch f.focus {
	case 0:
		return f.name.Focus()
	case 1:
		return f.command.Focus()
	default:
		return f.description.Focus()
	}
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			m.scr = screenList
			m.form.err = ""
			return m, nil
		case "tab":
			m.form.focus = (m.form.focus + 1) % 3
			return m, m.form.applyFocus()
		case "shift+tab":
			m.form.focus = (m.form.focus + 2) % 3
			return m, m.form.applyFocus()
		case "ctrl+t":
			m.form.direct = !m.form.direct
			return m, nil
		case "ctrl+s":
			return m.saveForm()
		}
	}

	var cmd tea.Cmd
	switch m.form.focus {
	case 0:
		m.form.name, cmd = m.form.name.Update(msg)
	case 1:
		m.form.command, cmd = m.form.command.Update(msg)
	default:
		m.form.description, cmd = m.form.description.Update(msg)
	}
	return m, cmd
}

func (m Model) startForm(name string) (tea.Model, tea.Cmd) {
	f := newForm()
	f.width = m.width
	f.resize()

	if name != "" {
		sc := m.cfg.Shortcuts[name]
		f.name.SetValue(name)
		f.description.SetValue(sc.Description)
		if sc.Command.Argv != nil {
			f.direct = true
			f.command.SetValue(template.QuoteArgv(sc.Command.Argv))
		} else {
			f.command.SetValue(sc.Command.Shell)
		}
		f.editing = true
	}

	m.form = f
	m.oldName = name
	m.scr = screenForm
	m.err = ""
	m.status = ""
	return m, m.form.name.Focus()
}

func (m Model) saveForm() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.form.name.Value())
	desc := strings.TrimSpace(m.form.description.Value())
	raw := strings.TrimSpace(m.form.command.Value())

	if err := config.ValidateName(name); err != nil {
		m.form.err = err.Error()
		return m, nil
	}
	if _, exists := m.cfg.Shortcuts[name]; exists && name != m.oldName {
		m.form.err = fmt.Sprintf("shortcut %q already exists", name)
		return m, nil
	}

	sc := config.Shortcut{Description: desc}
	if m.form.direct {
		parts, err := shlex.Split(raw)
		if err != nil {
			m.form.err = "cannot parse argv: " + err.Error()
			return m, nil
		}
		if parts == nil {
			parts = []string{}
		}
		sc.Command = config.Command{Argv: parts}
	} else {
		sc.Command = config.Command{Shell: raw}
	}
	if err := config.ValidateShortcut(name, sc); err != nil {
		m.form.err = err.Error()
		return m, nil
	}

	return m, saveCmd(m.store, m.mgr, m.oldName, name, sc, m.stamp)
}

func (m Model) formView() string {
	f := m.form
	title := "Add shortcut"
	if f.editing {
		title = "Edit shortcut"
	}
	mode := "shell"
	if f.direct {
		mode = "direct"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel(0, "Name"))
	b.WriteString("\n")
	b.WriteString(f.name.View())
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Mode: "))
	b.WriteString(textStyle.Render(mode))
	b.WriteString(labelStyle.Render("   (ctrl+t toggles)"))
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel(1, "Command"))
	b.WriteString("\n")
	b.WriteString(f.command.View())
	b.WriteString("\n")
	if f.direct {
		if parts, err := shlex.Split(f.command.Value()); err == nil && len(parts) > 0 {
			b.WriteString(hintStyle.Render("argv: " + template.QuoteArgv(parts)))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(m.fieldLabel(2, "Description"))
	b.WriteString("\n")
	b.WriteString(f.description.View())
	b.WriteString("\n\n")

	b.WriteString(hintStyle.Render("tab next - shift+tab previous - ctrl+t shell/direct - ctrl+s save - esc cancel"))
	b.WriteString("\n")
	if f.err != "" {
		b.WriteString(errorStyle.Render(f.err))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) fieldLabel(index int, text string) string {
	if m.form.focus == index {
		return labelStyle.Render("> " + text)
	}
	return labelStyle.Render("  " + text)
}

// --- detail and confirm screens -----------------------------------------

func (m Model) detailView() string {
	sc, ok := m.cfg.Shortcuts[m.detail]
	if !ok {
		return "This shortcut no longer exists. Press esc to return."
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(m.detail))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Mode"))
	b.WriteString(textStyle.Render("  " + string(sc.Command.Mode())))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Description"))
	b.WriteString(textStyle.Render("  " + sc.Description))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Command"))
	b.WriteString("\n")
	b.WriteString(textStyle.Render("  " + commandDisplay(sc)))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("e edit - d delete - esc back"))
	return b.String()
}

func (m Model) updateDetail(km tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch km.String() {
	case "esc", "q":
		m.scr = screenList
		m.detail = ""
	case "e":
		name := m.detail
		m.scr = screenList
		return m.startForm(name)
	case "d", "x":
		m.confirm = m.detail
		m.scr = screenConfirm
	}
	return m, nil
}

func (m Model) confirmView() string {
	return titleStyle.Render("Confirm delete") + "\n\n" +
		textStyle.Render("Remove shortcut ") +
		textStyle.Render(fmt.Sprintf("%q", m.confirm)) +
		textStyle.Render("? This cannot be undone.") +
		"\n\n" +
		hintStyle.Render("y confirm - n/esc cancel")
}

func (m Model) updateConfirm(km tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch km.String() {
	case "y", "enter":
		name := m.confirm
		m.confirm = ""
		return m, deleteCmd(m.store, m.mgr, name, m.stamp)
	case "n", "esc", "q":
		m.scr = screenList
		m.confirm = ""
	}
	return m, nil
}

// --- commands ------------------------------------------------------------

type savedMsg struct {
	cfg    config.Config
	status string
	stamp  fileStamp
	err    error
}

func saveCmd(store *config.Store, mgr *shim.Manager, oldName, name string, sc config.Shortcut, expected fileStamp) tea.Cmd {
	return func() tea.Msg {
		if expected != (fileStamp{}) {
			if cur := statStamp(store.Path()); cur != expected {
				return savedMsg{err: fmt.Errorf("config changed on disk; press esc and then r to reload")}
			}
		}
		err := store.Update(func(c *config.Config) error {
			if oldName != "" && oldName != name {
				delete(c.Shortcuts, oldName)
			}
			c.Shortcuts[name] = sc
			return nil
		})
		if err != nil {
			return savedMsg{err: err}
		}

		var syncErr error
		if oldName != "" && oldName != name {
			syncErr = mgr.Rename(oldName, name)
		} else {
			syncErr = mgr.Create(name)
		}

		cfg, loadErr := store.Load()
		if loadErr != nil {
			return savedMsg{err: loadErr}
		}
		status := "saved " + name
		if syncErr != nil {
			status = fmt.Sprintf("config saved; shim sync failed: %v (run 'shortcuts doctor --fix')", syncErr)
		}
		return savedMsg{cfg: cfg, status: status, stamp: statStamp(store.Path())}
	}
}

func deleteCmd(store *config.Store, mgr *shim.Manager, name string, expected fileStamp) tea.Cmd {
	return func() tea.Msg {
		if expected != (fileStamp{}) {
			if cur := statStamp(store.Path()); cur != expected {
				return savedMsg{err: fmt.Errorf("config changed on disk; press esc and then r to reload")}
			}
		}
		if err := store.Update(func(c *config.Config) error {
			delete(c.Shortcuts, name)
			return nil
		}); err != nil {
			return savedMsg{err: err}
		}
		syncErr := mgr.Remove(name)
		cfg, loadErr := store.Load()
		if loadErr != nil {
			return savedMsg{err: loadErr}
		}
		status := "removed " + name
		if syncErr != nil {
			status = fmt.Sprintf("config saved; shim cleanup failed: %v (run 'shortcuts doctor --fix')", syncErr)
		}
		return savedMsg{cfg: cfg, status: status, stamp: statStamp(store.Path())}
	}
}
