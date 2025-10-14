package tui

import (
	"context"
	"testing"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAgent is a mock implementation of the Agent interface for testing
type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) GetMCPServers() []ServerInfo {
	args := m.Called()
	return args.Get(0).([]ServerInfo)
}

func (m *MockAgent) GetMCPTools(ctx context.Context) ([]Tool, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Tool), args.Error(1)
}

func (m *MockAgent) SubscribeToUpdates() <-chan interface{} {
	args := m.Called()
	if ch := args.Get(0); ch != nil {
		return ch.(<-chan interface{})
	}
	// Return a nil channel for tests that don't need it
	return nil
}

func (m *MockAgent) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*ToolExecutionResult, error) {
	args := m.Called(ctx, toolName, params)
	return args.Get(0).(*ToolExecutionResult), args.Error(1)
}

func (m *MockAgent) GetMCPToolsAsDefinitions(ctx context.Context) ([]model.ToolDefinition, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.ToolDefinition), args.Error(1)
}

func (m *MockAgent) ExecuteToolUnified(ctx context.Context, toolName string, params map[string]interface{}, userContext string) (string, error) {
	args := m.Called(ctx, toolName, params, userContext)
	return args.String(0), args.Error(1)
}

func (m *MockAgent) ProcessToolResult(ctx context.Context, toolName string, result *mcp.ExecuteResult, userQuery string) (string, error) {
	args := m.Called(ctx, toolName, result, userQuery)
	return args.String(0), args.Error(1)
}

// TestServerView_WithRealMCPData tests that ServerView displays real MCP server data
func TestServerView_WithRealMCPData(t *testing.T) {
	mockAgent := &MockAgent{}
	
	// Set up mock data
	servers := []ServerInfo{
		{
			Name:      "local-memory",
			Status:    "connected",
			Connected: true,
			ToolCount: 11,
			Transport: "stdio",
			Error:     "",
		},
		{
			Name:      "filesystem",
			Status:    "disconnected", 
			Connected: false,
			ToolCount: 0,
			Transport: "stdio",
			Error:     "connection failed",
		},
	}
	
	mockAgent.On("GetMCPServers").Return(servers)
	
	// Create ServerView with agent
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	
	serverView := NewServerViewWithAgent(styles, keymap, mockAgent)
	require.NotNil(t, serverView, "ServerView should be created")
	
	// Test that it loads real server data
	serverView.RefreshServers()
	
	// Verify the servers are loaded
	items := serverView.GetServerItems()
	require.Len(t, items, 2, "Should have 2 servers")
	
	// Check local-memory server
	assert.Equal(t, "local-memory", items[0].Title())
	assert.Contains(t, items[0].Description(), "✅ Connected")
	assert.Contains(t, items[0].Description(), "11 tools")
	
	// Check filesystem server
	assert.Equal(t, "filesystem", items[1].Title())
	assert.Contains(t, items[1].Description(), "❌ Disconnected")
	assert.Contains(t, items[1].Description(), "0 tools")
	
	mockAgent.AssertExpectations(t)
}

// TestServerView_EmptyMCPData tests ServerView with no MCP servers
func TestServerView_EmptyMCPData(t *testing.T) {
	mockAgent := &MockAgent{}
	
	// No servers
	mockAgent.On("GetMCPServers").Return([]ServerInfo{})
	
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	
	serverView := NewServerViewWithAgent(styles, keymap, mockAgent)
	serverView.RefreshServers()
	
	items := serverView.GetServerItems()
	assert.Len(t, items, 0, "Should have no servers")
	
	mockAgent.AssertExpectations(t)
}

// TestServerView_RefreshUpdatesData tests that refresh updates the server list
func TestServerView_RefreshUpdatesData(t *testing.T) {
	mockAgent := &MockAgent{}
	
	// Initial state - no servers (called during construction)
	mockAgent.On("GetMCPServers").Return([]ServerInfo{}).Once()
	
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	
	serverView := NewServerViewWithAgent(styles, keymap, mockAgent)
	
	items := serverView.GetServerItems()
	assert.Len(t, items, 0, "Should start with no servers")
	
	// After refresh - one server appears
	newServers := []ServerInfo{
		{
			Name:      "local-memory",
			Status:    "connected",
			Connected: true,
			ToolCount: 11,
			Transport: "stdio",
		},
	}
	mockAgent.On("GetMCPServers").Return(newServers).Once()
	
	serverView.RefreshServers()
	
	items = serverView.GetServerItems()
	assert.Len(t, items, 1, "Should have one server after refresh")
	assert.Equal(t, "local-memory", items[0].Title())
	
	mockAgent.AssertExpectations(t)
}

// TestApplication_WithAgent tests that Application can be created with an Agent
func TestApplication_WithAgent(t *testing.T) {
	mockAgent := &MockAgent{}
	
	mockAgent.On("GetMCPServers").Return([]ServerInfo{})
	mockAgent.On("GetMCPTools", mock.Anything).Return([]Tool{}, nil)
	
	// This tests the new constructor that accepts an agent
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	
	app := NewApplicationWithAgent(keymap, styles, mockAgent)
	require.NotNil(t, app, "Application should be created with agent")
	
	// Test that server view has access to agent data
	serverView := app.GetServerView()
	require.NotNil(t, serverView, "Should have server view")
	
	mockAgent.AssertExpectations(t)
}