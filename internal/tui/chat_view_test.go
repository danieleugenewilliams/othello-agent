package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

func TestChatView_ExitCommand(t *testing.T) {
	// Create chat view with nil model (sufficient for command testing)
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	chatView := NewChatView(styles, keymap, nil)
	
	// Test /exit command
	cmd := chatView.handleCommand("/exit")
	
	// The command should return tea.Quit
	if cmd == nil {
		t.Fatal("Expected command to be returned for /exit")
	}
	
	// Execute the command to get the message
	msg := cmd()
	
	// Should be a quit message
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Expected tea.QuitMsg, got %T", msg)
	}
}

func TestChatView_BuildMetadataContext(t *testing.T) {
	// Import model package for ConversationContext
	// This test is in the same package so we need to use the full import path
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	chatView := NewChatView(styles, keymap, nil)

	tests := []struct {
		name     string
		metadata map[string]interface{}
		want     []string // Expected substrings in the output
	}{
		{
			name:     "no metadata",
			metadata: map[string]interface{}{},
			want:     []string{},
		},
		{
			name: "memory_id only",
			metadata: map[string]interface{}{
				"memory_id": "uuid-12345",
			},
			want: []string{
				"IMPORTANT: Context from previous tool executions",
				"memory_id: uuid-12345",
				"use this value when tools require 'memory_id' parameter",
			},
		},
		{
			name: "multiple priority fields",
			metadata: map[string]interface{}{
				"memory_id":       "uuid-12345",
				"first_memory_id": "uuid-67890",
			},
			want: []string{
				"IMPORTANT: Context from previous tool executions",
				"memory_id: uuid-12345",
				"first_memory_id: uuid-67890",
			},
		},
		{
			name: "universal extraction fields",
			metadata: map[string]interface{}{
				"memory_id":    "uuid-12345",
				"document_id":  "doc-456",
				"artifact_key": "art-xyz",
				"status":       "completed",
			},
			want: []string{
				"IMPORTANT: Context from previous tool executions",
				"memory_id: uuid-12345",
				"document_id: doc-456",
				"artifact_key: art-xyz",
				"status: completed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up conversation context with metadata
			if len(tt.metadata) > 0 {
				chatView.conversationContext = &model.ConversationContext{
					ExtractedMetadata: tt.metadata,
				}
			} else {
				chatView.conversationContext = nil
			}

			result := chatView.buildMetadataContextForModel()

			if len(tt.want) == 0 {
				if result != "" {
					t.Errorf("Expected empty result, got: %s", result)
				}
				return
			}

			// Check that all expected substrings are present
			for _, expectedSubstring := range tt.want {
				if !contains(result, expectedSubstring) {
					t.Errorf("Expected result to contain %q, got: %s", expectedSubstring, result)
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestChatView_QuitCommand(t *testing.T) {
	// Create chat view with nil model (sufficient for command testing)
	styles := DefaultStyles()
	keymap := DefaultKeyMap()
	chatView := NewChatView(styles, keymap, nil)
	
	// Test /quit command (alias for /exit)
	cmd := chatView.handleCommand("/quit")
	
	// The command should return tea.Quit
	if cmd == nil {
		t.Fatal("Expected command to be returned for /quit")
	}
	
	// Execute the command to get the message
	msg := cmd()
	
	// Should be a quit message
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Expected tea.QuitMsg, got %T", msg)
	}
}