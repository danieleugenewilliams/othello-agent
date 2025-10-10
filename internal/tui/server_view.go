package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ServerItem represents a server in the list
type ServerItem struct {
	name      string
	status    string
	toolCount int
	connected bool
}

// Title returns the title for the list item
func (s ServerItem) Title() string {
	return s.name
}

// Description returns the description for the list item
func (s ServerItem) Description() string {
	status := "❌ Disconnected"
	if s.connected {
		status = "✅ Connected"
	}
	return fmt.Sprintf("%s • %d tools", status, s.toolCount)
}

// FilterValue returns the value to filter on
func (s ServerItem) FilterValue() string {
	return s.name
}

// ServerView handles the server management interface
type ServerView struct {
	width   int
	height  int
	styles  Styles
	keymap  KeyMap
	list    list.Model
	servers []ServerItem
}

// NewServerView creates a new server view
func NewServerView(styles Styles, keymap KeyMap) *ServerView {
	// Create some mock servers for now
	servers := []ServerItem{
		{name: "filesystem", status: "connected", toolCount: 8, connected: true},
		{name: "web-search", status: "disconnected", toolCount: 5, connected: false},
		{name: "calculator", status: "connected", toolCount: 3, connected: true},
	}
	
	items := make([]list.Item, len(servers))
	for i, server := range servers {
		items[i] = server
	}
	
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "MCP Servers"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styles.ViewHeader
	
	return &ServerView{
		styles:  styles,
		keymap:  keymap,
		list:    l,
		servers: servers,
	}
}

// Init initializes the server view
func (v *ServerView) Init() tea.Cmd {
	return nil
}

// Update handles updates for the server view
func (v *ServerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Toggle server connection
			if selected := v.list.SelectedItem(); selected != nil {
				if server, ok := selected.(ServerItem); ok {
					// TODO: Implement server connection toggle
					_ = server
				}
			}
			return v, nil
		case "r":
			// Refresh servers
			// TODO: Implement server refresh
			return v, nil
		case "a":
			// Add new server
			// TODO: Implement add server dialog
			return v, nil
		case "d":
			// Delete server
			// TODO: Implement delete server
			return v, nil
		}
	}
	
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

// View renders the server view
func (v *ServerView) View() string {
	if v.width == 0 {
		return "Loading servers..."
	}
	
	// Header
	header := v.styles.ViewHeader.
		Width(v.width).
		Render("🖥️  MCP Servers")
	
	// List content
	listContent := v.list.View()
	
	// Help text
	helpText := v.styles.DimmedStyle.Render(
		"enter: toggle • r: refresh • a: add • d: delete",
	)
	
	// Calculate heights
	headerHeight := lipgloss.Height(header)
	helpHeight := lipgloss.Height(helpText)
	listHeight := v.height - headerHeight - helpHeight - 2
	
	if listHeight < 1 {
		listHeight = 1
	}
	
	v.list.SetHeight(listHeight)
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		listContent,
		helpText,
	)
}

// SetSize sets the size of the server view
func (v *ServerView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetWidth(width)
}

// AddServer adds a server to the list
func (v *ServerView) AddServer(server ServerItem) {
	v.servers = append(v.servers, server)
	items := make([]list.Item, len(v.servers))
	for i, s := range v.servers {
		items[i] = s
	}
	v.list.SetItems(items)
}

// RemoveServer removes a server from the list
func (v *ServerView) RemoveServer(name string) {
	for i, server := range v.servers {
		if server.name == name {
			v.servers = append(v.servers[:i], v.servers[i+1:]...)
			break
		}
	}
	
	items := make([]list.Item, len(v.servers))
	for i, s := range v.servers {
		items[i] = s
	}
	v.list.SetItems(items)
}

// UpdateServerStatus updates the status of a server
func (v *ServerView) UpdateServerStatus(name string, connected bool, toolCount int) {
	for i, server := range v.servers {
		if server.name == name {
			v.servers[i].connected = connected
			v.servers[i].toolCount = toolCount
			if connected {
				v.servers[i].status = "connected"
			} else {
				v.servers[i].status = "disconnected"
			}
			break
		}
	}
	
	items := make([]list.Item, len(v.servers))
	for i, s := range v.servers {
		items[i] = s
	}
	v.list.SetItems(items)
}

// GetSelectedServer returns the currently selected server
func (v *ServerView) GetSelectedServer() *ServerItem {
	if selected := v.list.SelectedItem(); selected != nil {
		if server, ok := selected.(ServerItem); ok {
			return &server
		}
	}
	return nil
}

// GetServers returns all servers
func (v *ServerView) GetServers() []ServerItem {
	return v.servers
}