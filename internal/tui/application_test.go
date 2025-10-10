package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestApplication_ESCKeyNavigation(t *testing.T) {
	// Create application without agent for testing navigation
	app := NewApplication(nil)
	
	// Start in chat view
	if app.currentView != ChatViewType {
		t.Errorf("Expected to start in ChatViewType, got %v", app.currentView)
	}
	
	// Switch to server view
	app.currentView = ServerViewType
	
	// Press ESC key
	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	
	updatedApp, _ := app.Update(escKey)
	app = updatedApp.(*Application)
	
	// Should navigate back to chat view
	if app.currentView != ChatViewType {
		t.Errorf("Expected ESC to navigate to ChatViewType, got %v", app.currentView)
	}
}

func TestApplication_ESCInChatView(t *testing.T) {
	// Create application without agent for testing navigation
	app := NewApplication(nil)
	
	// Start in chat view (default)
	if app.currentView != ChatViewType {
		t.Errorf("Expected to start in ChatViewType, got %v", app.currentView)
	}
	
	// Press ESC key while in chat view
	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	
	updatedApp, cmd := app.Update(escKey)
	app = updatedApp.(*Application)
	
	// Should stay in chat view and not quit
	if app.currentView != ChatViewType {
		t.Errorf("Expected to stay in ChatViewType, got %v", app.currentView)
	}
	
	// Should not return quit command
	if cmd != nil {
		msg := cmd()
		if _, isQuit := msg.(tea.QuitMsg); isQuit {
			t.Error("ESC in chat view should not quit the application")
		}
	}
}

func TestKeyMap_ESCBinding(t *testing.T) {
	keymap := DefaultKeyMap()
	
	// Test that ESC is bound to Back, not Quit
	escKey := tea.KeyMsg{Type: tea.KeyEsc}
	
	if key.Matches(escKey, keymap.Quit) {
		t.Error("ESC should not match Quit binding")
	}
	
	if !key.Matches(escKey, keymap.Back) {
		t.Error("ESC should match Back binding")
	}
	
	// Test that Ctrl+C still quits
	ctrlCKey := tea.KeyMsg{Type: tea.KeyCtrlC}
	
	if !key.Matches(ctrlCKey, keymap.Quit) {
		t.Error("Ctrl+C should match Quit binding")
	}
}