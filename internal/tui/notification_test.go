package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test that ServerView handles status update messages correctly
func TestServerView_HandleStatusUpdate(t *testing.T) {
	mockAgent := &MockAgent{}
	
	// Initial server state
	initialServers := []ServerInfo{
		{Name: "test-server", Status: "disconnected", Connected: false, ToolCount: 0},
	}
	mockAgent.On("GetMCPServers").Return(initialServers)
	
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	
	serverView := NewServerViewWithAgent(styles, keymap, mockAgent)
	
	// Verify initial state
	items := serverView.GetServerItems()
	assert.Len(t, items, 1)
	assert.Equal(t, "test-server", items[0].name)
	assert.False(t, items[0].connected)
	assert.Equal(t, 0, items[0].toolCount)
	
	// Send a status update message
	updateMsg := ServerStatusUpdateMsg{
		ServerName: "test-server",
		Connected:  true,
		ToolCount:  5,
		Error:      "",
	}
	
	// Update the server view with the message
	updatedModel, cmd := serverView.Update(updateMsg)
	updatedView := updatedModel.(*ServerView)
	
	// Verify the server status was updated
	items = updatedView.GetServerItems()
	assert.Len(t, items, 1)
	assert.Equal(t, "test-server", items[0].name)
	assert.True(t, items[0].connected)
	assert.Equal(t, 5, items[0].toolCount)
	assert.Equal(t, "connected", items[0].status)
	
	// No additional commands should be returned
	assert.Nil(t, cmd)
	
	mockAgent.AssertExpectations(t)
}

// Test that ToolView handles tool update messages correctly
func TestToolView_HandleToolUpdate(t *testing.T) {
	mockAgent := &MockAgentForTools{}
	
	// Initial tools
	initialTools := []Tool{
		{Name: "tool1", Description: "First tool", Server: "server1"},
	}
	
	// Updated tools after refresh (what will be returned when refreshTools is called)
	updatedTools := []Tool{
		{Name: "tool1", Description: "First tool", Server: "server1"},
		{Name: "tool2", Description: "Second tool", Server: "server1"},
	}
	
	// Mock initial call
	mockAgent.On("GetMCPTools", mock.Anything).Return(initialTools, nil).Once()
	
	toolView := NewToolViewWithAgent(mockAgent)
	
	// Verify initial state
	assert.Len(t, toolView.tools, 1)
	assert.Equal(t, "tool1", toolView.tools[0].Name)
	
	// Mock the refresh call that will happen during update handling
	mockAgent.On("GetMCPTools", mock.Anything).Return(updatedTools, nil).Once()
	
	// Send a tool update message
	updateMsg := ToolUpdateMsg{
		ServerName: "server1",
		Tools:      []Tool{}, // Will trigger refresh
		Added:      []string{"tool2"},
		Removed:    []string{},
	}
	
	// Update the tool view with the message
	updatedModel, cmd := toolView.Update(updateMsg)
	updatedToolView := updatedModel.(*ToolView)
	
	// Verify the tools were refreshed
	assert.Len(t, updatedToolView.tools, 2)
	assert.Equal(t, "tool1", updatedToolView.tools[0].Name)
	assert.Equal(t, "tool2", updatedToolView.tools[1].Name)
	
	// No additional commands should be returned
	assert.Nil(t, cmd)
	
	mockAgent.AssertExpectations(t)
}