package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAgentForTools extends the existing mock to handle GetMCPTools
type MockAgentForTools struct {
	mock.Mock
}

func (m *MockAgentForTools) GetMCPServers() []ServerInfo {
	args := m.Called()
	return args.Get(0).([]ServerInfo)
}

func (m *MockAgentForTools) GetMCPTools(ctx context.Context) ([]Tool, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Tool), args.Error(1)
}

func (m *MockAgentForTools) SubscribeToUpdates() <-chan interface{} {
	args := m.Called()
	if ch := args.Get(0); ch != nil {
		return ch.(<-chan interface{})
	}
	// Return a nil channel for tests that don't need it
	return nil
}

func (m *MockAgentForTools) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*ToolExecutionResult, error) {
	args := m.Called(ctx, toolName, params)
	return args.Get(0).(*ToolExecutionResult), args.Error(1)
}

func TestToolView_NewToolView(t *testing.T) {
	tv := NewToolView()
	
	assert.NotNil(t, tv)
	assert.NotNil(t, tv.table)
	assert.NotNil(t, tv.filter)
	assert.Len(t, tv.tools, 3) // Should have mock data
	assert.False(t, tv.filterMode)
}

func TestToolView_NewToolViewWithAgent(t *testing.T) {
	mockAgent := &MockAgentForTools{}
	
	// Mock tools data
	expectedTools := []Tool{
		{Name: "search", Description: "Search memories", Server: "local-memory"},
		{Name: "store", Description: "Store memory", Server: "local-memory"},
	}
	
	mockAgent.On("GetMCPTools", mock.Anything).Return(expectedTools, nil)
	
	tv := NewToolViewWithAgent(mockAgent)
	
	assert.NotNil(t, tv)
	assert.Equal(t, mockAgent, tv.agent)
	assert.Len(t, tv.tools, 2)
	assert.Equal(t, "search", tv.tools[0].Name)
	assert.Equal(t, "local-memory", tv.tools[0].Server)
	
	mockAgent.AssertExpectations(t)
}

func TestToolView_WithRealMCPTools(t *testing.T) {
	mockAgent := &MockAgentForTools{}
	
	// Mock real MCP tools data
	expectedTools := []Tool{
		{Name: "store_memory", Description: "Store a new memory", Server: "local-memory"},
		{Name: "search", Description: "Search through memories", Server: "local-memory"},
		{Name: "analysis", Description: "Analyze memories", Server: "local-memory"},
	}
	
	mockAgent.On("GetMCPTools", mock.Anything).Return(expectedTools, nil)
	
	tv := NewToolViewWithAgent(mockAgent)
	
	// Verify tools are loaded
	assert.Len(t, tv.tools, 3)
	assert.Equal(t, "store_memory", tv.tools[0].Name)
	assert.Equal(t, "Store a new memory", tv.tools[0].Description)
	assert.Equal(t, "local-memory", tv.tools[0].Server)
	
	mockAgent.AssertExpectations(t)
}

func TestToolView_EmptyMCPTools(t *testing.T) {
	mockAgent := &MockAgentForTools{}
	
	// Mock empty tools
	mockAgent.On("GetMCPTools", mock.Anything).Return([]Tool{}, nil)
	
	tv := NewToolViewWithAgent(mockAgent)
	
	// Should have no tools
	assert.Len(t, tv.tools, 0)
	
	mockAgent.AssertExpectations(t)
}

func TestToolView_RefreshUpdatesTools(t *testing.T) {
	mockAgent := &MockAgentForTools{}
	
	// Initial tools
	initialTools := []Tool{
		{Name: "tool1", Description: "First tool", Server: "server1"},
	}
	
	// Updated tools after refresh
	updatedTools := []Tool{
		{Name: "tool1", Description: "First tool", Server: "server1"},
		{Name: "tool2", Description: "Second tool", Server: "server2"},
	}
	
	// First call during construction
	mockAgent.On("GetMCPTools", mock.Anything).Return(initialTools, nil).Once()
	
	// Second call during refresh
	mockAgent.On("GetMCPTools", mock.Anything).Return(updatedTools, nil).Once()
	
	tv := NewToolViewWithAgent(mockAgent)
	
	// Initial state
	assert.Len(t, tv.tools, 1)
	
	// Refresh tools
	tv.refreshTools()
	
	// Should have updated tools
	assert.Len(t, tv.tools, 2)
	assert.Equal(t, "tool2", tv.tools[1].Name)
	
	mockAgent.AssertExpectations(t)
}

func TestToolView_FilterTools(t *testing.T) {
	tv := NewToolView()
	
	// Set up test tools
	tv.tools = []Tool{
		{Name: "search_memory", Description: "Search through memories", Server: "local-memory"},
		{Name: "store_memory", Description: "Store a memory", Server: "local-memory"},
		{Name: "file_read", Description: "Read file contents", Server: "filesystem"},
	}
	
	// Test filter by name
	tv.filter.SetValue("search")
	tv.updateTable()
	
	// Should only show search tool
	rows := tv.table.Rows()
	assert.Len(t, rows, 1)
	assert.Equal(t, "search_memory", rows[0][0])
	
	// Test filter by server
	tv.filter.SetValue("filesystem")
	tv.updateTable()
	
	// Should only show filesystem tool
	rows = tv.table.Rows()
	assert.Len(t, rows, 1)
	assert.Equal(t, "file_read", rows[0][0])
	
	// Test no filter
	tv.filter.SetValue("")
	tv.updateTable()
	
	// Should show all tools
	rows = tv.table.Rows()
	assert.Len(t, rows, 3)
}

func TestToolView_Update(t *testing.T) {
	tv := NewToolView()
	
	// Test entering filter mode
	updatedModel, cmd := tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updatedTv := updatedModel.(*ToolView)
	
	assert.True(t, updatedTv.filterMode)
	assert.NotNil(t, cmd)
	
	// Test exiting filter mode with Esc
	updatedModel, _ = updatedTv.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedTv = updatedModel.(*ToolView)
	
	assert.False(t, updatedTv.filterMode)
	assert.Empty(t, updatedTv.filter.Value())
}

func TestToolView_GetSelectedTool(t *testing.T) {
	tv := NewToolView()
	
	// Set up test tools
	tv.tools = []Tool{
		{Name: "tool1", Description: "First tool", Server: "server1"},
		{Name: "tool2", Description: "Second tool", Server: "server2"},
	}
	tv.updateTable()
	
	// Get selected tool (should be first by default)
	selected := tv.GetSelectedTool()
	assert.NotNil(t, selected)
	assert.Equal(t, "tool1", selected.Name)
}

func TestToolView_GetSelectedToolEmpty(t *testing.T) {
	tv := NewToolView()
	tv.tools = []Tool{} // Empty tools
	tv.updateTable()
	
	selected := tv.GetSelectedTool()
	assert.Nil(t, selected)
}

