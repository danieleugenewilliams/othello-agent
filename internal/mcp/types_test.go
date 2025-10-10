package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToolCache(t *testing.T) {
	cache := NewToolCache(time.Hour)
	
	// Test empty cache
	_, exists := cache.Get("nonexistent")
	assert.False(t, exists)
	
	// Test setting and getting a tool
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
		ServerName: "test-server",
	}
	
	cache.Set(tool)
	
	retrieved, exists := cache.Get("test-tool")
	assert.True(t, exists)
	assert.Equal(t, tool.Name, retrieved.Name)
	assert.Equal(t, tool.Description, retrieved.Description)
	assert.Equal(t, tool.ServerName, retrieved.ServerName)
	assert.NotZero(t, retrieved.LastUpdated)
}

func TestToolCacheTTL(t *testing.T) {
	cache := NewToolCache(time.Millisecond * 100) // Very short TTL for testing
	
	tool := Tool{
		Name:        "ttl-test",
		Description: "TTL test tool",
		ServerName:  "test-server",
	}
	
	cache.Set(tool)
	
	// Should exist immediately
	_, exists := cache.Get("ttl-test")
	assert.True(t, exists)
	
	// Wait for TTL to expire
	time.Sleep(time.Millisecond * 150)
	
	// Should not exist after TTL
	_, exists = cache.Get("ttl-test")
	assert.False(t, exists)
}

func TestServer(t *testing.T) {
	server := Server{
		Name:      "test-server",
		Transport: "stdio",
		Command:   []string{"test-command"},
		Args:      []string{"--arg1", "value1"},
		Env:       map[string]string{"TEST_VAR": "test_value"},
		Timeout:   time.Second * 30,
		Connected: false,
	}
	
	assert.Equal(t, "test-server", server.Name)
	assert.Equal(t, "stdio", server.Transport)
	assert.Equal(t, []string{"test-command"}, server.Command)
	assert.Equal(t, []string{"--arg1", "value1"}, server.Args)
	assert.Equal(t, "test_value", server.Env["TEST_VAR"])
	assert.Equal(t, time.Second*30, server.Timeout)
	assert.False(t, server.Connected)
}

func TestMessage(t *testing.T) {
	// Test request message
	request := Message{
		ID:     1,
		Method: "tools/list",
		Params: map[string]interface{}{"test": "value"},
	}
	
	assert.Equal(t, 1, request.ID)
	assert.Equal(t, "tools/list", request.Method)
	assert.NotNil(t, request.Params)
	assert.Nil(t, request.Result)
	assert.Nil(t, request.Error)
	
	// Test response message
	response := Message{
		ID:     1,
		Result: map[string]interface{}{"tools": []interface{}{}},
	}
	
	assert.Equal(t, 1, response.ID)
	assert.Empty(t, response.Method)
	assert.Nil(t, response.Params)
	assert.NotNil(t, response.Result)
	assert.Nil(t, response.Error)
	
	// Test error message
	errorMsg := Message{
		ID: 1,
		Error: &Error{
			Code:    ErrorInvalidRequest,
			Message: "Invalid request",
		},
	}
	
	assert.Equal(t, 1, errorMsg.ID)
	assert.NotNil(t, errorMsg.Error)
	assert.Equal(t, ErrorInvalidRequest, errorMsg.Error.Code)
	assert.Equal(t, "Invalid request", errorMsg.Error.Message)
}

func TestError(t *testing.T) {
	err := &Error{
		Code:    ErrorMethodNotFound,
		Message: "Method not found",
		Data:    map[string]interface{}{"method": "unknown"},
	}
	
	assert.Equal(t, ErrorMethodNotFound, err.Code)
	assert.Equal(t, "Method not found", err.Message)
	assert.NotNil(t, err.Data)
	
	// Test Error interface implementation
	assert.Equal(t, "Method not found", err.Error())
}

func TestToolResult(t *testing.T) {
	// Test successful result
	result := ToolResult{
		Content: []Content{
			{
				Type: "text",
				Text: "Operation completed successfully",
			},
		},
		IsError: false,
	}
	
	assert.Len(t, result.Content, 1)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "Operation completed successfully", result.Content[0].Text)
	assert.False(t, result.IsError)
	
	// Test error result
	errorResult := ToolResult{
		Content: []Content{
			{
				Type: "text",
				Text: "Operation failed",
			},
		},
		IsError: true,
	}
	
	assert.True(t, errorResult.IsError)
	assert.Equal(t, "Operation failed", errorResult.Content[0].Text)
}