package tui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpView handles the help interface
type HelpView struct {
	width  int
	height int
	styles Styles
	keymap KeyMap
	help   help.Model
}

// NewHelpView creates a new help view
func NewHelpView(styles Styles, keymap KeyMap) *HelpView {
	h := help.New()
	h.Styles.FullKey = styles.HighlightStyle
	h.Styles.FullDesc = styles.Base
	h.Styles.ShortKey = styles.HighlightStyle
	h.Styles.ShortDesc = styles.DimmedStyle
	
	return &HelpView{
		styles: styles,
		keymap: keymap,
		help:   h,
	}
}

// Init initializes the help view
func (v *HelpView) Init() tea.Cmd {
	return nil
}

// Update handles updates for the help view
func (v *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return v, nil
}

// View renders the help view
func (v *HelpView) View() string {
	if v.width == 0 {
		return "Loading help..."
	}
	
	// Header
	header := v.styles.ViewHeader.
		Width(v.width).
		Render("❓ Help")
	
	// Help content
	helpContent := v.help.FullHelpView(v.keymap.FullHelp())
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		helpContent,
		"",
		v.styles.DimmedStyle.Render("Othello AI Agent - Local AI assistant with MCP tool integration"),
	)
}

// SetSize sets the size of the help view
func (v *HelpView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.help.Width = width
}