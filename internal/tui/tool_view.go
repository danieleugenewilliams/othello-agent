package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToolView represents the tools display view
type ToolView struct {
	table          table.Model
	filter         textinput.Model
	tools          []Tool
	agent          AgentInterface
	width          int
	height         int
	filterMode     bool
	selectedServer string // Filter tools by this server when set
}

// NewToolView creates a new tool view with mock data (backward compatibility)
func NewToolView() *ToolView {
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Server", Width: 15},
		{Title: "Description", Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	// Create filter input
	filter := textinput.New()
	filter.Placeholder = "Filter tools..."
	filter.CharLimit = 50

	tv := &ToolView{
		table:  t,
		filter: filter,
		tools:  []Tool{},
	}

	tv.loadMockData()
	return tv
}

// NewToolViewWithAgent creates a new tool view with real agent data
func NewToolViewWithAgent(agent AgentInterface) *ToolView {
	tv := NewToolView()
	tv.agent = agent
	tv.refreshTools()
	return tv
}

// loadMockData loads sample tool data for testing
func (tv *ToolView) loadMockData() {
	mockTools := []Tool{
		{
			Name:        "search_memory",
			Description: "Search through stored memories",
			Server:      "local-memory",
		},
		{
			Name:        "store_memory",
			Description: "Store a new memory",
			Server:      "local-memory",
		},
		{
			Name:        "file_read",
			Description: "Read file contents",
			Server:      "filesystem",
		},
	}

	tv.tools = mockTools
	tv.updateTable()
}

// refreshTools loads tools from the agent
func (tv *ToolView) refreshTools() {
	if tv.agent == nil {
		tv.loadMockData()
		return
	}

	ctx := context.Background()
	tools, err := tv.agent.GetMCPTools(ctx)
	if err != nil {
		// Fallback to mock data on error
		tv.loadMockData()
		return
	}

	tv.tools = tools
	tv.updateTable()
}

// SetSelectedServer sets the server filter and refreshes the tool list
func (tv *ToolView) SetSelectedServer(serverName string) {
	tv.selectedServer = serverName
	tv.refreshTools() // Refresh to ensure we have latest tools
	tv.updateTable()
}

// updateTable updates the table with current tools data
func (tv *ToolView) updateTable() {
	filterText := strings.ToLower(tv.filter.Value())
	var filteredTools []Tool

	// First filter by selected server if set
	serverFilteredTools := tv.tools
	if tv.selectedServer != "" {
		serverFilteredTools = []Tool{}
		for _, tool := range tv.tools {
			if tool.Server == tv.selectedServer {
				serverFilteredTools = append(serverFilteredTools, tool)
			}
		}
	}

	// Then apply text filter if any
	if filterText == "" {
		filteredTools = serverFilteredTools
	} else {
		for _, tool := range serverFilteredTools {
			if strings.Contains(strings.ToLower(tool.Name), filterText) ||
				strings.Contains(strings.ToLower(tool.Description), filterText) ||
				strings.Contains(strings.ToLower(tool.Server), filterText) {
				filteredTools = append(filteredTools, tool)
			}
		}
	}

	rows := make([]table.Row, len(filteredTools))
	for i, tool := range filteredTools {
		description := tool.Description
		if len(description) > 47 {
			description = description[:47] + "..."
		}
		rows[i] = table.Row{tool.Name, tool.Server, description}
	}

	tv.table.SetRows(rows)
}

// Init initializes the tool view
func (tv *ToolView) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the tool view
func (tv *ToolView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ToolUpdateMsg:
		// Handle tool update
		tv.handleToolUpdate(msg)
		return tv, nil
	case RefreshDataMsg:
		// Handle refresh request
		if msg.ViewType == "tools" || msg.ViewType == "all" {
			tv.refreshTools()
		}
		return tv, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return tv, tea.Quit
		case "/":
			tv.filterMode = true
			tv.filter.Focus()
			return tv, textinput.Blink
		case "esc":
			if tv.filterMode {
				// Exit filter mode
				tv.filterMode = false
				tv.filter.Blur()
				tv.filter.SetValue("")
				tv.updateTable()
				return tv, nil
			} else {
				// Always go back to server view
				tv.selectedServer = ""
				return tv, func() tea.Msg {
					return ViewSwitchMsg{ViewType: ServerViewType}
				}
			}
		case "enter":
			if tv.filterMode {
				// Exit filter mode when in filter
				tv.filterMode = false
				tv.filter.Blur()
				tv.updateTable()
				return tv, nil
			}
			// Otherwise, do nothing - tool execution will come from model in Week 4
			return tv, nil
		case "x":
			if !tv.filterMode {
				// Execute selected tool with 'x' key
				return tv, tv.executeSelectedTool()
			}
		case "r":
			if !tv.filterMode {
				tv.refreshTools()
				return tv, nil
			}
		}

		if tv.filterMode {
			tv.filter, cmd = tv.filter.Update(msg)
			tv.updateTable()
			return tv, cmd
		} else {
			tv.table, cmd = tv.table.Update(msg)
			return tv, cmd
		}

	case tea.WindowSizeMsg:
		tv.width = msg.Width
		tv.height = msg.Height
		tv.table.SetWidth(msg.Width - 4)
		tv.table.SetHeight(msg.Height - 8)
		return tv, nil
	}

	return tv, cmd
}

// View renders the tool view
func (tv *ToolView) View() string {
	var s strings.Builder

	// Show breadcrumb if viewing tools for a specific server
	if tv.selectedServer != "" {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("MCP > "))
		s.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Render(tv.selectedServer))
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(" > Tools"))
		s.WriteString("\n\n")
	} else {
		s.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Render("MCP Tools"))
		s.WriteString("\n\n")
	}

	if tv.filterMode {
		s.WriteString("Filter: ")
		s.WriteString(tv.filter.View())
		s.WriteString("\n\n")
	} else {
		if tv.selectedServer != "" {
			s.WriteString("Press '/' to filter, 'r' to refresh, 'esc' to go back to servers, 'q' to quit\n\n")
		} else {
			s.WriteString("Press '/' to filter, 'r' to refresh, 'x' to execute, 'enter' to go back, 'q' to quit\n\n")
		}
	}

	s.WriteString(tv.table.View())

	if !tv.filterMode && len(tv.tools) > 0 {
		selected := tv.table.SelectedRow()
		if len(selected) > 0 {
			s.WriteString("\n\n")
			s.WriteString(lipgloss.NewStyle().
				Bold(true).
				Render("Selected Tool Details:"))
			s.WriteString("\n")
			s.WriteString(fmt.Sprintf("Name: %s\n", selected[0]))
			s.WriteString(fmt.Sprintf("Server: %s\n", selected[1]))
			s.WriteString(fmt.Sprintf("Description: %s\n", selected[2]))
		}
	}

	return s.String()
}

// GetSelectedTool returns the currently selected tool
func (tv *ToolView) GetSelectedTool() *Tool {
	if len(tv.tools) == 0 {
		return nil
	}

	selectedRow := tv.table.SelectedRow()
	if len(selectedRow) == 0 {
		return nil
	}

	// Find the tool by name
	for _, tool := range tv.tools {
		if tool.Name == selectedRow[0] {
			return &tool
		}
	}

	return nil
}

// handleToolUpdate processes tool update messages
func (tv *ToolView) handleToolUpdate(msg ToolUpdateMsg) {
	// For simplicity, just refresh all tools when there's an update
	// In a more sophisticated implementation, we could handle Added/Removed lists
	tv.refreshTools()
}

// executeSelectedTool executes the currently selected tool
func (tv *ToolView) executeSelectedTool() tea.Cmd {
	selectedTool := tv.GetSelectedTool()
	if selectedTool == nil || tv.agent == nil {
		return nil
	}
	
	return func() tea.Msg {
		ctx := context.Background()
		
		// For now, execute with empty parameters
		// In a more sophisticated implementation, we would prompt for parameters
		result, err := tv.agent.ExecuteTool(ctx, selectedTool.Name, make(map[string]interface{}))
		if err != nil {
			return ToolExecutionMsg{
				ToolName: selectedTool.Name,
				Success:  false,
				Error:    err.Error(),
			}
		}
		
		return ToolExecutionMsg{
			ToolName: selectedTool.Name,
			Success:  result.Success,
			Result:   result.Result,
			Error:    result.Error,
		}
	}
}

// SetSize updates the view dimensions
func (tv *ToolView) SetSize(width, height int) {
	tv.width = width
	tv.height = height
	tv.table.SetWidth(width - 4)
	tv.table.SetHeight(height - 8)
}