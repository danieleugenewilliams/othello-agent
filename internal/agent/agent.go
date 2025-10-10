package agent

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
	"github.com/danieleugenewilliams/othello-agent/internal/tui"
)

// Agent represents the core agent instance
type Agent struct {
	config *config.Config
	logger *log.Logger
}

// Interface defines the agent's public API
type Interface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	StartTUI() error
	GetStatus() *Status
}

// Status represents the current agent status
type Status struct {
	Running        bool   `json:"running"`
	ConfigFile     string `json:"config_file"`
	ModelConnected bool   `json:"model_connected"`
	MCPServers     int    `json:"mcp_servers"`
}

// New creates a new agent instance
func New(cfg *config.Config) (*Agent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	agent := &Agent{
		config: cfg,
		logger: log.New(log.Writer(), "[AGENT] ", log.LstdFlags),
	}

	return agent, nil
}

// Start starts the agent with the given context
func (a *Agent) Start(ctx context.Context) error {
	a.logger.Println("Starting Othello AI Agent")
	
	// TODO: Initialize model interface
	// TODO: Initialize MCP client manager
	// TODO: Initialize storage
	// TODO: Start background services
	
	a.logger.Printf("Agent started with model: %s", a.config.Model.Name)
	return nil
}

// Stop gracefully stops the agent
func (a *Agent) Stop(ctx context.Context) error {
	a.logger.Println("Stopping Othello AI Agent")
	
	// TODO: Stop background services
	// TODO: Close MCP connections
	// TODO: Close storage connections
	
	a.logger.Println("Agent stopped")
	return nil
}

// StartTUI starts the terminal user interface
func (a *Agent) StartTUI() error {
	a.logger.Println("Starting TUI mode")
	
	// Create the model instance
	m := model.NewOllamaModel(a.config.Ollama.Host, a.config.Model.Name)
	
	// Create and start the TUI application
	app := tui.NewApplication(m)
	
	// Run the TUI
	program := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	
	return nil
}

// GetStatus returns the current agent status
func (a *Agent) GetStatus() *Status {
	return &Status{
		Running:        true, // TODO: Track actual state
		ConfigFile:     a.config.ConfigFile(),
		ModelConnected: false, // TODO: Check actual model connection
		MCPServers:     len(a.config.MCP.Servers),
	}
}