package main

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danieleugenewilliams/othello-agent/internal/agent"
	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
	"github.com/danieleugenewilliams/othello-agent/internal/storage"
)

// TestIntegration_MCPClientWithOllama tests the integration between MCP client and Ollama model
func TestIntegration_MCPClientWithOllama(t *testing.T) {
	// Skip if Ollama is not running
	if !isOllamaAvailable(t) {
		t.Skip("Ollama not available, skipping integration test")
	}

	// Create temporary config
	cfg := createTestConfig(t)
	
	// Test Ollama model
	ollamaModel := model.NewOllamaModel(cfg.Ollama.Host, cfg.Model.Name)
	
	available := ollamaModel.IsAvailable(context.Background())
	assert.True(t, available, "Ollama should be available for integration test")
	
	// Test basic generation
	response, err := ollamaModel.Generate(context.Background(), "Hello", model.GenerateOptions{
		Temperature: 0.1,
		MaxTokens:   10,
	})
	
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.Content)
}

// TestIntegration_MCPToolExecution tests MCP tool execution in a real scenario
func TestIntegration_MCPToolExecution(t *testing.T) {
	// Skip if local-memory server is not available
	if !isMCPServerAvailable(t) {
		t.Skip("MCP local-memory server not available, skipping integration test")
	}

	logger := &SimpleLogger{}
	
	// Create MCP server config
	server := mcp.Server{
		Name:      "local-memory",
		Transport: "stdio",
		Command:   []string{"local-memory"},
		Args:      []string{"--db-path", ":memory:", "--session-id", "integration-test"},
		Timeout:   30 * time.Second,
	}
	
	// Create MCP client
	client := mcp.NewSTDIOClient(server, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Connect to server
	err := client.Connect(ctx)
	require.NoError(t, err, "Failed to connect to MCP server")
	defer client.Disconnect()
	
	// Test tool execution - store a memory
	params := map[string]interface{}{
		"content":    "Integration test memory from Othello agent",
		"importance": 8,
		"tags":       []string{"integration", "test", "othello"},
		"domain":     "testing",
	}
	
	result, err := client.CallTool(ctx, "store_memory", params)
	require.NoError(t, err, "Failed to execute store_memory tool")
	assert.False(t, result.IsError)
	assert.NotEmpty(t, result.Content)
	
	// Test search functionality
	searchParams := map[string]interface{}{
		"query":       "integration test",
		"search_type": "semantic",
		"limit":       5,
	}
	
	searchResult, err := client.CallTool(ctx, "search", searchParams)
	require.NoError(t, err, "Failed to execute search tool")
	assert.False(t, searchResult.IsError)
	assert.NotEmpty(t, searchResult.Content)
}

// TestIntegration_AgentCreation tests the full agent creation and initialization
func TestIntegration_AgentCreation(t *testing.T) {
	cfg := createTestConfig(t)
	
	// Create agent
	agentInstance, err := agent.New(cfg)
	require.NoError(t, err, "Failed to create agent")
	assert.NotNil(t, agentInstance)
	
	// Test agent status
	status := agentInstance.GetStatus()
	assert.NotNil(t, status)
	assert.True(t, status.Running)
	// Note: ConfigFile may be empty when config is created programmatically in tests
	// This is expected behavior
}

// TestIntegration_ConversationPersistence tests the full conversation workflow
func TestIntegration_ConversationPersistence(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration_test.db")
	
	store, err := storage.NewConversationStore(dbPath)
	require.NoError(t, err, "Failed to create conversation store")
	defer store.Close()
	
	// Create a conversation
	conv, err := store.CreateConversation("integration-conv", "Integration Test Conversation")
	require.NoError(t, err, "Failed to create conversation")
	
	// Add messages simulating a real conversation
	messages := []*storage.Message{
		{
			ConversationID: conv.ID,
			Role:          "user",
			Content:       "Hello, I need help with a Go project",
			Timestamp:     time.Now(),
			TokenCount:    10,
		},
		{
			ConversationID: conv.ID,
			Role:          "assistant",
			Content:       "I'd be happy to help with your Go project! What specific aspect do you need assistance with?",
			Timestamp:     time.Now().Add(1 * time.Second),
			TokenCount:    20,
		},
		{
			ConversationID: conv.ID,
			Role:          "user",
			Content:       "I need to implement MCP tool integration",
			Timestamp:     time.Now().Add(2 * time.Second),
			TokenCount:    8,
		},
		{
			ConversationID: conv.ID,
			Role:          "tool",
			Content:       "",
			ToolCall: &storage.ToolCall{
				ID:   "call-search-123",
				Name: "search",
				Arguments: map[string]interface{}{
					"query": "MCP tool integration Go",
				},
			},
			Timestamp:  time.Now().Add(3 * time.Second),
			TokenCount: 5,
		},
		{
			ConversationID: conv.ID,
			Role:          "tool",
			Content:       "Found relevant documentation about MCP integration",
			ToolResult: &storage.ToolResult{
				ID:      "call-search-123",
				Content: "MCP integration documentation and examples",
				IsError: false,
			},
			Timestamp:  time.Now().Add(4 * time.Second),
			TokenCount: 12,
		},
	}
	
	// Add all messages
	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err, "Failed to add message")
	}
	
	// Update conversation stats
	err = store.UpdateConversationStats(conv.ID)
	require.NoError(t, err, "Failed to update conversation stats")
	
	// Verify conversation stats
	updated, err := store.GetConversation(conv.ID)
	require.NoError(t, err, "Failed to get updated conversation")
	
	assert.Equal(t, 5, updated.MessageCount)
	assert.Equal(t, 55, updated.TotalTokens) // 10+20+8+5+12
	
	// Retrieve and verify messages
	retrievedMessages, err := store.GetMessages(conv.ID, 10, 0)
	require.NoError(t, err, "Failed to retrieve messages")
	
	assert.Len(t, retrievedMessages, 5)
	
	// Verify tool call serialization/deserialization
	toolCallMsg := retrievedMessages[3] // 4th message (0-indexed)
	assert.Equal(t, "tool", toolCallMsg.Role)
	assert.NotNil(t, toolCallMsg.ToolCall)
	assert.Equal(t, "call-search-123", toolCallMsg.ToolCall.ID)
	assert.Equal(t, "search", toolCallMsg.ToolCall.Name)
	
	toolResultMsg := retrievedMessages[4] // 5th message
	assert.Equal(t, "tool", toolResultMsg.Role)
	assert.NotNil(t, toolResultMsg.ToolResult)
	assert.Equal(t, "call-search-123", toolResultMsg.ToolResult.ID)
	assert.False(t, toolResultMsg.ToolResult.IsError)
}

// TestIntegration_FullWorkflow tests a complete user interaction workflow
func TestIntegration_FullWorkflow(t *testing.T) {
	// Skip if dependencies are not available
	if !isOllamaAvailable(t) {
		t.Skip("Ollama not available, skipping full workflow test")
	}
	
	cfg := createTestConfig(t)
	
	// Create temporary storage
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "workflow_test.db")
	
	store, err := storage.NewConversationStore(dbPath)
	require.NoError(t, err)
	defer store.Close()
	
	// Create agent
	agentInstance, err := agent.New(cfg)
	require.NoError(t, err)
	
	// Simulate workflow: User asks question -> Agent uses tools -> Stores conversation
	conv, err := store.CreateConversation("workflow-test", "Full Workflow Test")
	require.NoError(t, err)
	
	// User message
	userMsg := &storage.Message{
		ConversationID: conv.ID,
		Role:          "user",
		Content:       "What is the best way to implement error handling in Go?",
		Timestamp:     time.Now(),
		TokenCount:    12,
	}
	err = store.AddMessage(userMsg)
	require.NoError(t, err)
	
	// Simulate model response (in real scenario, this would come from Ollama)
	assistantMsg := &storage.Message{
		ConversationID: conv.ID,
		Role:          "assistant",
		Content:       "Go has several approaches to error handling. The most common is explicit error returns. Let me search for best practices.",
		Timestamp:     time.Now().Add(1 * time.Second),
		TokenCount:    25,
	}
	err = store.AddMessage(assistantMsg)
	require.NoError(t, err)
	
	// Update stats and verify
	err = store.UpdateConversationStats(conv.ID)
	require.NoError(t, err)
	
	finalConv, err := store.GetConversation(conv.ID)
	require.NoError(t, err)
	
	assert.Equal(t, 2, finalConv.MessageCount)
	assert.Equal(t, 37, finalConv.TotalTokens)
	
	// Verify agent status
	status := agentInstance.GetStatus()
	assert.True(t, status.Running)
	// Note: ConfigFile may be empty when config is created programmatically in tests
}

// Helper functions

func createTestConfig(t *testing.T) *config.Config {
	return &config.Config{
		Model: config.ModelConfig{
			Type:        "ollama",
			Name:        "qwen2.5:3b",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		Ollama: config.OllamaConfig{
			Host:    "http://localhost:11434",
			Timeout: 30 * time.Second,
		},
		TUI: config.TUIConfig{
			Theme:      "default",
			ShowHints:  true,
			AutoScroll: true,
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:    "local-memory",
					Command: "local-memory",
					Args:    []string{"--db-path", ":memory:", "--session-id", "test"},
				},
			},
		},
		Storage: config.StorageConfig{
			HistorySize: 1000,
			CacheTTL:    30 * time.Minute,
			DataDir:     ":memory:",
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}
}

func isOllamaAvailable(t *testing.T) bool {
	ollamaModel := model.NewOllamaModel("http://localhost:11434", "qwen2.5:3b")
	return ollamaModel.IsAvailable(context.Background())
}

func isMCPServerAvailable(t *testing.T) bool {
	// Try to run local-memory command to see if it's available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	logger := &SimpleLogger{}
	server := mcp.Server{
		Name:      "test",
		Transport: "stdio",
		Command:   []string{"local-memory"},
		Args:      []string{"--help"},
		Timeout:   5 * time.Second,
	}
	client := mcp.NewSTDIOClient(server, logger)
	
	// This is a quick test - if the command exists, it should at least start
	err := client.Connect(ctx)
	if err == nil {
		client.Disconnect()
		return true
	}
	
	return false
}

// SimpleLogger implements the mcp.Logger interface for tests
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, keysAndValues)
}

func (l *SimpleLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, keysAndValues)
}

func (l *SimpleLogger) Debug(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, keysAndValues)
}