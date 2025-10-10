package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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