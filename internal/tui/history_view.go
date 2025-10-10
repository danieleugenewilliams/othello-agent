package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HistoryView handles the conversation history interface
type HistoryView struct {
	width    int
	height   int
	styles   Styles
	keymap   KeyMap
	viewport viewport.Model
}

// NewHistoryView creates a new history view
func NewHistoryView(styles Styles, keymap KeyMap) *HistoryView {
	vp := viewport.New(0, 0)
	vp.SetContent("No conversation history yet.")
	
	return &HistoryView{
		styles:   styles,
		keymap:   keymap,
		viewport: vp,
	}
}

// Init initializes the history view
func (v *HistoryView) Init() tea.Cmd {
	return nil
}

// Update handles updates for the history view
func (v *HistoryView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the history view
func (v *HistoryView) View() string {
	if v.width == 0 {
		return "Loading history..."
	}
	
	// Header
	header := v.styles.ViewHeader.
		Width(v.width).
		Render("📚 Conversation History")
	
	// History content
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		v.viewport.View(),
	)
}

// SetSize sets the size of the history view
func (v *HistoryView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height - 3 // Account for header
}