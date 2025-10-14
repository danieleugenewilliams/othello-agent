package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
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

// ToolExecutionMsg removed - replaced with ToolExecutedUnifiedMsg

// CommandMsg represents a command execution request
type CommandMsg struct {
	Command string
	Args    []string
}

// ViewSwitchMsg requests switching to a specific view
type ViewSwitchMsg struct {
	ViewType ViewType
}

// ToolCallDetectedMsg represents when the model wants to call tools
type ToolCallDetectedMsg struct {
	ToolCalls           []model.ToolCall
	RequestID           string
	Response            *model.Response
	UserMessage         string              // Original user message
	ConversationHistory []model.Message     // Conversation history up to this point
	Tools               []model.ToolDefinition // Available tools
}

// ToolExecutionResultMsg removed - replaced with ToolExecutedUnifiedMsg

// MCPToolExecutingMsg represents a tool execution starting
type MCPToolExecutingMsg struct {
	ToolName string
	Params   map[string]interface{}
}

// MCPToolExecutedMsg represents a tool execution completion
type MCPToolExecutedMsg struct {
	ToolName string
	Result   *mcp.ExecuteResult
	Error    error
}

// ToolExecutedUnifiedMsg represents a unified tool execution result
type ToolExecutedUnifiedMsg struct {
	ToolName string
	Result   string // Already processed natural language result
	Success  bool
}

// ServerSelectedMsg represents a server being selected in the ServerView
type ServerSelectedMsg struct {
	ServerName string
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

// GenerateResponseWithTools sends a message to the model with tool support
func GenerateResponseWithTools(m model.Model, message string, tools []model.ToolDefinition, id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Create conversation with user message
		messages := []model.Message{
			{Role: "user", Content: message},
		}
		
		response, err := m.ChatWithTools(ctx, messages, tools, model.GenerateOptions{
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