package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// ViewType represents the different views in the TUI
type ViewType int

const (
	ChatViewType ViewType = iota
	ServerViewType
	ToolViewType
	HelpViewType
	HistoryViewType
)

// KeyMap defines the keybindings for the application
type KeyMap struct {
	Quit       key.Binding
	Back       key.Binding
	Submit     key.Binding
	SwitchView key.Binding
	ClearInput key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		SwitchView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch view"),
		),
		ClearInput: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear input"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.SwitchView}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Submit, k.SwitchView, k.ClearInput, k.Back},
		{k.Quit},
	}
}

// Styles contains all styling definitions
type Styles struct {
	Base          lipgloss.Style
	StatusBar     lipgloss.Style
	ViewHeader    lipgloss.Style
	MessageUser   lipgloss.Style
	MessageBot    lipgloss.Style
	MessageTool   lipgloss.Style
	InputBox      lipgloss.Style
	InputPrompt   lipgloss.Style
	ServerList    lipgloss.Style
	ServerItem    lipgloss.Style
	ErrorStyle    lipgloss.Style
	SuccessStyle  lipgloss.Style
	DimmedStyle   lipgloss.Style
	HighlightStyle lipgloss.Style
}

// DefaultStyles returns the default styling
func DefaultStyles() Styles {
	return Styles{
		Base: lipgloss.NewStyle().
			Padding(0, 1),
		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1),
		ViewHeader: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1),
		MessageUser: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true),
		MessageBot: lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")),
		MessageTool: lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Italic(true),
		InputBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),
		InputPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		ServerList: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1),
		ServerItem: lipgloss.NewStyle().
			PaddingLeft(2),
		ErrorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		SuccessStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),
		DimmedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")),
		HighlightStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")),
	}
}

// Application represents the main TUI application
type Application struct {
	width       int
	height      int
	currentView ViewType
	keymap      KeyMap
	styles      Styles
	help        help.Model
	model       model.Model
	agent       AgentInterface // Optional agent for MCP data
	
	// Views
	chatView    *ChatView
	serverView  *ServerView
	toolView    *ToolView
	helpView    *HelpView
	historyView *HistoryView
	
	// State
	quitting bool
	err      error
}

// NewApplication creates a new TUI application
func NewApplication(m model.Model) *Application {
	keymap := DefaultKeyMap()
	styles := DefaultStyles()
	
	app := &Application{
		currentView: ChatViewType,
		keymap:      keymap,
		styles:      styles,
		help:        help.New(),
		model:       m,
		agent:       nil, // No agent, use mock data
		chatView:    NewChatViewWithAgent(styles, keymap, m, nil),
		serverView:  NewServerView(styles, keymap),
		helpView:    NewHelpView(styles, keymap),
		historyView: NewHistoryView(styles, keymap),
	}
	
	return app
}

// NewApplicationWithAgent creates a new TUI application with agent support
func NewApplicationWithAgent(keymap KeyMap, styles Styles, agent AgentInterface) *Application {
	// Create a model for the ChatView (we can use a dummy model or create one from agent config)
	// For now, create a basic Ollama model instance
	m := model.NewOllamaModel("http://localhost:11434", "qwen2.5:3b")
	
	app := &Application{
		currentView: ChatViewType,
		keymap:      keymap,
		styles:      styles,
		help:        help.New(),
		model:       m,
		agent:       agent,
		chatView:    NewChatViewWithAgent(styles, keymap, m, agent),
		serverView:  NewServerViewWithAgent(styles, keymap, agent),
		toolView:    NewToolViewWithAgent(agent),
		helpView:    NewHelpView(styles, keymap),
		historyView: NewHistoryView(styles, keymap),
	}
	
	return app
}

// Init implements tea.Model
func (a *Application) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textinput.Blink)
	
	// Initialize chat view if available
	if a.chatView != nil {
		cmds = append(cmds, a.chatView.Init())
	}
	
	// Start listening to agent updates if agent is available
	if a.agent != nil {
		cmds = append(cmds, a.listenForAgentUpdates())
	}
	
	return tea.Batch(cmds...)
}

// Update implements tea.Model
func (a *Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		
		// Update all views with new size
		if a.chatView != nil {
			a.chatView.SetSize(msg.Width, msg.Height-3) // Account for status bar
		}
		a.serverView.SetSize(msg.Width, msg.Height-3)
		a.toolView.SetSize(msg.Width, msg.Height-3)
		a.helpView.SetSize(msg.Width, msg.Height-3)
		a.historyView.SetSize(msg.Width, msg.Height-3)
		
		return a, nil

	case ViewSwitchMsg:
		// Handle view switching from commands
		a.currentView = msg.ViewType
		return a, nil
	
	case ServerSelectedMsg:
		// Handle server selection from ServerView - navigate to ToolView for that server
		if a.toolView != nil {
			a.toolView.SetSelectedServer(msg.ServerName)
		}
		a.currentView = ToolViewType
		return a, nil

	// ToolExecutedUnifiedMsg removed from application handler - chat view handles it directly

	default:
		// Handle agent updates by converting them to TUI messages and forwarding
		if a.agent != nil {
			if tuiMsg := a.convertAgentUpdate(msg); tuiMsg != nil {
				// Forward to all relevant views
				cmds = append(cmds, func() tea.Msg { return tuiMsg })
				// Continue listening for more updates
				cmds = append(cmds, a.waitForNextUpdate())
				return a, tea.Batch(cmds...)
			}
		}
		
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, a.keymap.Quit):
			a.quitting = true
			return a, tea.Quit
			
		// Removed global Back/Esc handler - let individual views handle their own back navigation
		
		case key.Matches(msg, a.keymap.SwitchView):
			a.nextView()
			return a, nil
		}
	}
	
	// Update current view
	switch a.currentView {
	case ChatViewType:
		newModel, cmd := a.chatView.Update(msg)
		a.chatView = newModel.(*ChatView)
		cmds = append(cmds, cmd)
		
	case ServerViewType:
		newModel, cmd := a.serverView.Update(msg)
		a.serverView = newModel.(*ServerView)
		cmds = append(cmds, cmd)
		
	case ToolViewType:
		newModel, cmd := a.toolView.Update(msg)
		a.toolView = newModel.(*ToolView)
		cmds = append(cmds, cmd)
		
	case HelpViewType:
		newModel, cmd := a.helpView.Update(msg)
		a.helpView = newModel.(*HelpView)
		cmds = append(cmds, cmd)
		
	case HistoryViewType:
		newModel, cmd := a.historyView.Update(msg)
		a.historyView = newModel.(*HistoryView)
		cmds = append(cmds, cmd)
	}
	
	return a, tea.Batch(cmds...)
}

// View implements tea.Model
func (a *Application) View() string {
	if a.quitting {
		return "Goodbye!\n"
	}
	
	if a.width == 0 {
		return "Loading..."
	}
	
	var content string
	
	// Render current view
	switch a.currentView {
	case ChatViewType:
		content = a.chatView.View()
	case ServerViewType:
		content = a.serverView.View()
	case ToolViewType:
		content = a.toolView.View()
	case HelpViewType:
		content = a.helpView.View()
	case HistoryViewType:
		content = a.historyView.View()
	}
	
	// Render status bar
	statusBar := a.renderStatusBar()
	
	// Combine everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		statusBar,
	)
}

// nextView cycles to the next view
func (a *Application) nextView() {
	switch a.currentView {
	case ChatViewType:
		a.currentView = ServerViewType
	case ServerViewType:
		a.currentView = ToolViewType
	case ToolViewType:
		a.currentView = HistoryViewType
	case HistoryViewType:
		a.currentView = HelpViewType
	case HelpViewType:
		a.currentView = ChatViewType
	}
}

// renderStatusBar renders the status bar
func (a *Application) renderStatusBar() string {
	var viewName string
	switch a.currentView {
	case ChatViewType:
		viewName = "Chat"
	case ServerViewType:
		viewName = "Servers"
	case ToolViewType:
		viewName = "Tools"
	case HelpViewType:
		viewName = "Help"
	case HistoryViewType:
		viewName = "History"
	}
	
	status := fmt.Sprintf(" %s ", viewName)
	helpText := a.help.ShortHelpView(a.keymap.ShortHelp())
	
	// Calculate spacing
	gap := a.width - lipgloss.Width(status) - lipgloss.Width(helpText)
	if gap < 0 {
		gap = 0
	}
	
	line := lipgloss.JoinHorizontal(
		lipgloss.Top,
		a.styles.StatusBar.Render(status),
		strings.Repeat(" ", gap),
		a.styles.DimmedStyle.Render(helpText),
	)
	
	return line
}

// SetError sets an error message to display
func (a *Application) SetError(err error) {
	a.err = err
}

// GetCurrentView returns the current view type
func (a *Application) GetCurrentView() ViewType {
	return a.currentView
}

// GetServerView returns the server view (for testing)
func (a *Application) GetServerView() *ServerView {
	return a.serverView
}

// listenForAgentUpdates creates a command that listens for agent status updates
func (a *Application) listenForAgentUpdates() tea.Cmd {
	return func() tea.Msg {
		if a.agent == nil {
			return nil
		}
		
		updateChan := a.agent.SubscribeToUpdates()
		select {
		case update := <-updateChan:
			// For now, just return the raw update and handle it in Update method
			return update
		}
	}
}

// waitForNextUpdate creates a command to continue listening for updates
func (a *Application) waitForNextUpdate() tea.Cmd {
	if a.agent == nil {
		return nil
	}
	return a.listenForAgentUpdates()
}

// convertAgentUpdate converts raw agent updates to TUI messages
func (a *Application) convertAgentUpdate(update interface{}) tea.Msg {
	// Use reflection to check the type name since we can't import agent package
	switch u := update.(type) {
	case interface{}:
		// Check if it's a ServerStatusUpdate by checking fields
		if serverName, connected, toolCount, errStr, ok := a.extractServerUpdate(u); ok {
			return ServerStatusUpdateMsg{
				ServerName: serverName,
				Connected:  connected,
				ToolCount:  toolCount,
				Error:      errStr,
			}
		}
		// Check if it's a ToolUpdate by checking fields
		if serverName, added, removed, ok := a.extractToolUpdate(u); ok {
			return ToolUpdateMsg{
				ServerName: serverName,
				Tools:      []Tool{}, // Will trigger refresh
				Added:      added,
				Removed:    removed,
			}
		}
	}
	return nil
}

// Helper methods to extract update data using type assertions
func (a *Application) extractServerUpdate(update interface{}) (string, bool, int, string, bool) {
	// Define a temporary struct that matches the agent's ServerStatusUpdate
	type ServerStatusUpdate struct {
		ServerName string
		Connected  bool
		ToolCount  int
		Error      string
	}
	
	if su, ok := update.(ServerStatusUpdate); ok {
		return su.ServerName, su.Connected, su.ToolCount, su.Error, true
	}
	return "", false, 0, "", false
}

func (a *Application) extractToolUpdate(update interface{}) (string, []string, []string, bool) {
	// Define a temporary struct that matches the agent's ToolUpdate
	type ToolUpdate struct {
		ServerName string
		ToolCount  int
		Added      []string
		Removed    []string
	}
	
	if tu, ok := update.(ToolUpdate); ok {
		return tu.ServerName, tu.Added, tu.Removed, true
	}
	return "", nil, nil, false
}