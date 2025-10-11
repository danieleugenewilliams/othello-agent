package tui

import (
	"testing"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestMCPMessages_ToolExecuting(t *testing.T) {
	// Test that MCPToolExecutingMsg can be created and carries the right data
	msg := MCPToolExecutingMsg{
		ToolName: "search",
		Params: map[string]interface{}{
			"query":       "test search",
			"search_type": "semantic",
		},
	}

	assert.Equal(t, "search", msg.ToolName)
	assert.Equal(t, "test search", msg.Params["query"])
	assert.Equal(t, "semantic", msg.Params["search_type"])
}

func TestMCPMessages_ToolExecuted_Success(t *testing.T) {
	// Test that MCPToolExecutedMsg can carry successful results
	result := &mcp.ExecuteResult{
		Tool: mcp.Tool{
			Name:        "search",
			Description: "Search memories",
		},
		Result: &mcp.ToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: "Found 3 results",
				},
			},
			IsError: false,
		},
		Duration: "100ms",
	}

	msg := MCPToolExecutedMsg{
		ToolName: "search",
		Result:   result,
		Error:    nil,
	}

	assert.Equal(t, "search", msg.ToolName)
	assert.NotNil(t, msg.Result)
	assert.NoError(t, msg.Error)
	assert.False(t, msg.Result.Result.IsError)
	assert.Equal(t, "Found 3 results", msg.Result.Result.Content[0].Text)
}

func TestMCPMessages_ToolExecuted_Error(t *testing.T) {
	// Test that MCPToolExecutedMsg can carry errors
	msg := MCPToolExecutedMsg{
		ToolName: "search",
		Result:   nil,
		Error:    assert.AnError,
	}

	assert.Equal(t, "search", msg.ToolName)
	assert.Nil(t, msg.Result)
	assert.Error(t, msg.Error)
}

func TestMCPMessages_ToolExecuted_MCPError(t *testing.T) {
	// Test that MCPToolExecutedMsg can carry MCP-level errors
	result := &mcp.ExecuteResult{
		Tool: mcp.Tool{
			Name: "search",
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
		ToolName: "search",
		Result:   result,
		Error:    nil, // No Go error, but MCP returned an error
	}

	assert.Equal(t, "search", msg.ToolName)
	assert.NotNil(t, msg.Result)
	assert.NoError(t, msg.Error)
	assert.True(t, msg.Result.Result.IsError, "MCP result should indicate an error")
}
