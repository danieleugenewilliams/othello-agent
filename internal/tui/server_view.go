package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// AgentInterface defines what the TUI needs from the Agent

type AgentInterface interface {
	GetMCPServers() []ServerInfo
	GetMCPTools(ctx context.Context) ([]Tool, error)
	GetMCPToolsAsDefinitions(ctx context.Context) ([]model.ToolDefinition, error)
	GetUniversalIntegration() interface{} // Returns *UniversalAgentIntegration but using interface{} to avoid import cycle
	SubscribeToUpdates() <-chan interface{} // Channel for receiving status updates
	ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*ToolExecutionResult, error)
	ProcessToolResult(ctx context.Context, toolName string, result *mcp.ExecuteResult, userQuery string) (string, error)
	ExecuteToolUnified(ctx context.Context, toolName string, params map[string]interface{}, userContext string) (string, error)
	ExecuteToolUnifiedWithContext(ctx context.Context, toolName string, params map[string]interface{}, convContext *model.ConversationContext) (string, error)
}

// ServerInfo represents MCP server information
type ServerInfo struct {
	Name      string
	Status    string
	Connected bool
	ToolCount int
	Transport string
	Error     string
}

// Tool represents an MCP tool
type Tool struct {
	Name        string
	Description string
	Server      string
}

// ToolExecutionResult represents the result of executing an MCP tool
type ToolExecutionResult struct {
	ToolName   string
	Success    bool
	Result     interface{}
	Error      string
	Duration   string
}

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
	agent   AgentInterface // Optional agent for real data
}

// NewServerView creates a new server view with mock data (backward compatibility)
func NewServerView(styles Styles, keymap KeyMap) *ServerView {
	return NewServerViewWithAgent(styles, keymap, nil)
}

// NewServerViewWithAgent creates a new server view with real agent data
func NewServerViewWithAgent(styles Styles, keymap KeyMap, agent AgentInterface) *ServerView {
	var servers []ServerItem
	
	if agent != nil {
		// Use real data from agent
		servers = getServerItemsFromAgent(agent)
	} else {
		// Create some mock servers for backward compatibility
		servers = []ServerItem{
			{name: "filesystem", status: "connected", toolCount: 8, connected: true},
			{name: "web-search", status: "disconnected", toolCount: 5, connected: false},
			{name: "calculator", status: "connected", toolCount: 3, connected: true},
		}
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
		agent:   agent,
	}
}

// getServerItemsFromAgent converts agent server info to ServerItem list
func getServerItemsFromAgent(agent AgentInterface) []ServerItem {
	if agent == nil {
		return []ServerItem{}
	}
	
	serverInfos := agent.GetMCPServers()
	items := make([]ServerItem, len(serverInfos))
	
	for i, info := range serverInfos {
		items[i] = ServerItem{
			name:      info.Name,
			status:    info.Status,
			toolCount: info.ToolCount,
			connected: info.Connected,
		}
	}
	
	return items
}

// Init initializes the server view
func (v *ServerView) Init() tea.Cmd {
	return nil
}

// Update handles updates for the server view
func (v *ServerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case ServerStatusUpdateMsg:
		// Handle server status update
		v.handleServerStatusUpdate(msg)
		return v, nil
	case RefreshDataMsg:
		// Handle refresh request
		if msg.ViewType == "servers" || msg.ViewType == "all" {
			v.RefreshServers()
		}
		return v, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Select server to view its tools
			if selected := v.list.SelectedItem(); selected != nil {
				if server, ok := selected.(ServerItem); ok {
					// Send ServerSelectedMsg to navigate to tools view for this server
					return v, func() tea.Msg {
						return ServerSelectedMsg{
							ServerName: server.name,
						}
					}
				}
			}
			return v, nil
		case "esc":
			// Go back to chat view
			return v, func() tea.Msg {
				return ViewSwitchMsg{ViewType: ChatViewType}
			}
		case "r":
			// Refresh servers from agent
			v.RefreshServers()
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

// RefreshServers refreshes the server list from the agent
func (v *ServerView) RefreshServers() {
	if v.agent == nil {
		return // No agent, keep mock data
	}
	
	// Get fresh data from agent
	v.servers = getServerItemsFromAgent(v.agent)
	
	// Update the list
	items := make([]list.Item, len(v.servers))
	for i, server := range v.servers {
		items[i] = server
	}
	v.list.SetItems(items)
}

// GetServerItems returns server items for testing
func (v *ServerView) GetServerItems() []ServerItem {
	return v.servers
}

// handleServerStatusUpdate processes server status update messages
func (v *ServerView) handleServerStatusUpdate(msg ServerStatusUpdateMsg) {
	// Find and update the server in our list
	for i, server := range v.servers {
		if server.name == msg.ServerName {
			// Update the server status
			v.servers[i].connected = msg.Connected
			v.servers[i].toolCount = msg.ToolCount
			if msg.Connected {
				v.servers[i].status = "connected"
			} else {
				v.servers[i].status = "disconnected"
			}
			
			// Update the list items
			items := make([]list.Item, len(v.servers))
			for j, s := range v.servers {
				items[j] = s
			}
			v.list.SetItems(items)
			return
		}
	}
	
	// Server not found, it might be a new server - refresh from agent
	if v.agent != nil {
		v.RefreshServers()
	}
}