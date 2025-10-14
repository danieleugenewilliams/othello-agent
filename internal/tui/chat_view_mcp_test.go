package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestChatView_HandlesMCPToolExecutingMsg tests that ChatView displays tool execution status
func TestChatView_HandlesMCPToolExecutingMsg(t *testing.T) {
	// GIVEN: A chat view with an agent
	view := setupChatViewWithMockAgent(t)
	initialMessageCount := len(view.messages)

	// WHEN: Tool executing message is received
	msg := MCPToolExecutingMsg{
		ToolName: "search",
		Params: map[string]interface{}{
			"query":       "test",
			"search_type": "semantic",
		},
	}

	_, cmd := view.Update(msg)

	// THEN: A new message is added showing tool execution
	assert.Len(t, view.messages, initialMessageCount+1, "Should add a message for tool execution")
	
	lastMsg := view.messages[len(view.messages)-1]
	assert.Equal(t, "tool", lastMsg.Role, "Message should have 'tool' role")
	assert.Contains(t, lastMsg.Content, "search", "Message should mention the tool name")
	assert.Contains(t, lastMsg.Content, "Executing", "Message should indicate execution in progress")
	
	// Should not return a command (execution happens elsewhere)
	assert.Nil(t, cmd, "Should not trigger additional commands")
}

// TestChatView_HandlesMCPToolExecutedMsg_Success tests successful tool execution display
func TestChatView_HandlesMCPToolExecutedMsg_Success(t *testing.T) {
	// GIVEN: A chat view with an executing tool
	view := setupChatViewWithMockAgent(t)
	view.AddMessage(ChatMessage{
		Role:      "tool",
		Content:   "Executing tool: search...",
		Timestamp: time.Now().Format("15:04:05"),
	})
	initialMessageCount := len(view.messages)

	// WHEN: Tool executed message with success is received
	result := &mcp.ExecuteResult{
		Tool: mcp.Tool{
			Name:        "search",
			Description: "Search memories",
		},
		Result: &mcp.ToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: "Found 3 results:\n1. First result\n2. Second result\n3. Third result",
				},
			},
			IsError: false,
		},
		Duration: "125ms",
	}

	msg := MCPToolExecutedMsg{
		ToolName: "search",
		Result:   result,
		Error:    nil,
	}

	_, cmd := view.Update(msg)

	// THEN: A new message is added with the result
	assert.Len(t, view.messages, initialMessageCount+1, "Should add a message for tool result")
	
	lastMsg := view.messages[len(view.messages)-1]
	assert.Equal(t, "tool", lastMsg.Role, "Message should have 'tool' role")
	assert.Contains(t, lastMsg.Content, "Found 3 results", "Message should contain result text")
	assert.Empty(t, lastMsg.Error, "Message should not have an error")
	
	// No command is triggered - tool results displayed inline
	assert.Nil(t, cmd, "Should not trigger additional commands")
}

// TestChatView_HandlesMCPToolExecutedMsg_Error tests tool execution error display
func TestChatView_HandlesMCPToolExecutedMsg_Error(t *testing.T) {
	// GIVEN: A chat view with an executing tool
	view := setupChatViewWithMockAgent(t)
	view.AddMessage(ChatMessage{
		Role:    "tool",
		Content: "Executing tool: search...",
	})
	initialMessageCount := len(view.messages)

	// WHEN: Tool executed message with error is received
	msg := MCPToolExecutedMsg{
		ToolName: "search",
		Result:   nil,
		Error:    errors.New("connection timeout to MCP server"),
	}

	_, _ = view.Update(msg)

	// THEN: A new message is added with the error
	assert.Len(t, view.messages, initialMessageCount+1, "Should add a message for tool error")
	
	lastMsg := view.messages[len(view.messages)-1]
	assert.Equal(t, "tool", lastMsg.Role, "Message should have 'tool' role")
	assert.NotEmpty(t, lastMsg.Error, "Message should have an error")
	assert.Contains(t, lastMsg.Error, "connection timeout", "Error should contain error message")
}

// TestChatView_HandlesMCPToolExecutedMsg_MCPError tests MCP-level errors (IsError=true)
func TestChatView_HandlesMCPToolExecutedMsg_MCPError(t *testing.T) {
	// GIVEN: A chat view with an executing tool
	view := setupChatViewWithMockAgent(t)
	initialMessageCount := len(view.messages)

	// WHEN: Tool executed message with MCP error (IsError=true) is received
	result := &mcp.ExecuteResult{
		Tool: mcp.Tool{
			Name: "stats",
		},
		Result: &mcp.ToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: "parameter validation failed: unknown parameter 'concept'",
				},
			},
			IsError: true,
		},
		Duration: "5ms",
	}

	msg := MCPToolExecutedMsg{
		ToolName: "stats",
		Result:   result,
		Error:    nil, // No Go error, but MCP indicates error
	}

	_, _ = view.Update(msg)

	// THEN: The error should be displayed
	assert.Len(t, view.messages, initialMessageCount+1, "Should add a message for MCP error")
	
	lastMsg := view.messages[len(view.messages)-1]
	assert.Equal(t, "tool", lastMsg.Role, "Message should have 'tool' role")
	assert.NotEmpty(t, lastMsg.Error, "Message should indicate error")
	assert.Contains(t, lastMsg.Error, "parameter validation failed", "Error should show MCP error text")
	assert.Contains(t, lastMsg.Error, "concept", "Error should show problematic parameter")
}

// TestChatView_StoresToolMessages tests that tool messages are stored correctly
func TestChatView_StoresToolMessages(t *testing.T) {
	// GIVEN: A chat view with several messages
	view := setupChatViewWithMockAgent(t)
	
	view.AddMessage(ChatMessage{
		Role:    "user",
		Content: "Hello",
	})
	view.AddMessage(ChatMessage{
		Role:    "assistant",
		Content: "Hi there!",
	})
	view.AddMessage(ChatMessage{
		Role:    "tool",
		Content: "Tool result",
	})

	// THEN: Should store all messages
	// Note: Welcome message is added by default, so we have 4 total
	assert.GreaterOrEqual(t, len(view.messages), 3, "Should store at least the added messages")
	
	// Find our messages (skip welcome message)
	hasUser := false
	hasAssistant := false
	hasTool := false
	
	for _, msg := range view.messages {
		if msg.Role == "user" && msg.Content == "Hello" {
			hasUser = true
		}
		if msg.Role == "assistant" && msg.Content == "Hi there!" {
			hasAssistant = true
		}
		if msg.Role == "tool" && msg.Content == "Tool result" {
			hasTool = true
		}
	}
	
	assert.True(t, hasUser, "Should have user message")
	assert.True(t, hasAssistant, "Should have assistant message")
	assert.True(t, hasTool, "Should have tool message")
}

// Helper function to set up a chat view with a mock agent
func setupChatViewWithMockAgent(t *testing.T) *ChatView {
	mockModel := &MockModel{}
	mockAgent := &MockAgentForChat{
		servers: []ServerInfo{
			{Name: "local-memory", Connected: true, ToolCount: 26},
		},
		tools: []Tool{
			{Name: "search", Description: "Search memories"},
			{Name: "stats", Description: "Get statistics"},
		},
	}
	
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	
	return NewChatViewWithAgent(styles, keymap, mockModel, mockAgent)
}

// MockModel implements the model interface for testing
type MockModel struct {
	generateFunc func(ctx context.Context, prompt string, opts model.GenerateOptions) (*model.Response, error)
}

func (m *MockModel) Generate(ctx context.Context, prompt string, opts model.GenerateOptions) (*model.Response, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt, opts)
	}
	return &model.Response{
		Content: "Mock response",
	}, nil
}

func (m *MockModel) Chat(ctx context.Context, messages []model.Message, opts model.GenerateOptions) (*model.Response, error) {
	return &model.Response{
		Content: "Mock chat response",
	}, nil
}

func (m *MockModel) ChatWithTools(ctx context.Context, messages []model.Message, tools []model.ToolDefinition, opts model.GenerateOptions) (*model.Response, error) {
	return &model.Response{
		Content: "Mock chat with tools response",
	}, nil
}

func (m *MockModel) IsAvailable(ctx context.Context) bool {
	return true
}

func (m *MockModel) Name() string {
	return "mock-model"
}

// MockAgentForChat implements the AgentInterface for chat tests
type MockAgentForChat struct {
	servers []ServerInfo
	tools   []Tool
}

func (m *MockAgentForChat) GetMCPServers() []ServerInfo {
	return m.servers
}

func (m *MockAgentForChat) GetMCPTools(ctx context.Context) ([]Tool, error) {
	return m.tools, nil
}

func (m *MockAgentForChat) GetMCPToolsAsDefinitions(ctx context.Context) ([]model.ToolDefinition, error) {
	defs := make([]model.ToolDefinition, len(m.tools))
	for i, tool := range m.tools {
		defs[i] = model.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
		}
	}
	return defs, nil
}

func (m *MockAgentForChat) SubscribeToUpdates() <-chan interface{} {
	ch := make(chan interface{})
	return ch
}

func (m *MockAgentForChat) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*ToolExecutionResult, error) {
	return &ToolExecutionResult{
		ToolName: toolName,
		Success:  true,
		Result:   "Mock tool execution result",
	}, nil
}

func (m *MockAgentForChat) ExecuteToolUnified(ctx context.Context, toolName string, params map[string]interface{}, userContext string) (string, error) {
	return "Mock unified tool execution result", nil
}

func (m *MockAgentForChat) ProcessToolResult(ctx context.Context, toolName string, result *mcp.ExecuteResult, userQuery string) (string, error) {
	return "Mock processed result", nil
}
