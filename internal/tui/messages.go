package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// ModelResponseMsg represents a message from the model
type ModelResponseMsg struct {
	Response *model.Response
	Error    error
	ID       string // to track which request this response is for
}

// ModelRequestMsg represents a request to send to the model
type ModelRequestMsg struct {
	Message string
	ID      string
}

// ServerStatusUpdateMsg represents a server status change notification
type ServerStatusUpdateMsg struct {
	ServerName string
	Connected  bool
	ToolCount  int
	Error      string
}

// ToolUpdateMsg represents a tool availability change notification  
type ToolUpdateMsg struct {
	ServerName string
	Tools      []Tool
	Added      []string // names of newly added tools
	Removed    []string // names of removed tools
}

// RefreshDataMsg signals views to refresh their data
type RefreshDataMsg struct {
	ViewType string // "servers", "tools", or "all"
}

// ToolExecutionMsg represents a tool execution notification
type ToolExecutionMsg struct {
	ToolName string
	Success  bool
	Result   interface{}
	Error    string
}

// GenerateResponse sends a message to the model and returns a command
func GenerateResponse(m model.Model, message, id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		response, err := m.Generate(ctx, message, model.GenerateOptions{
			Temperature: 0.7,
			MaxTokens:   2048,
		})
		
		return ModelResponseMsg{
			Response: response,
			Error:    err,
			ID:       id,
		}
	}
}