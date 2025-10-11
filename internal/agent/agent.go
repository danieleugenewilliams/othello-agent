package agent

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
	"github.com/danieleugenewilliams/othello-agent/internal/tui"
)

// Agent represents the core agent instance
type Agent struct {
	config       *config.Config
	logger       *log.Logger
	mcpRegistry  *mcp.ToolRegistry
	mcpManager   *MCPManager
	toolExecutor *mcp.ToolExecutor
	updateChan   chan interface{} // Channel for broadcasting status updates
}

// Interface defines the agent's public API
type Interface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	StartTUI() error
	GetStatus() *Status
	GetMCPServers() []ServerInfo
	GetMCPTools(ctx context.Context) ([]tui.Tool, error)
	SubscribeToUpdates() <-chan interface{}
	ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*tui.ToolExecutionResult, error)
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

	logger := log.New(log.Writer(), "[AGENT] ", log.LstdFlags)

	// Initialize MCP registry with logger adapter
	mcpLogger := &agentLogger{logger: logger}
	mcpRegistry := mcp.NewToolRegistry(mcpLogger)

	// Initialize MCP manager
	mcpManager := NewMCPManager(mcpRegistry, mcpLogger)

	// Initialize tool executor
	toolExecutor := mcp.NewToolExecutor(mcpRegistry, mcpLogger)

	agent := &Agent{
		config:       cfg,
		logger:       logger,
		mcpRegistry:  mcpRegistry,
		mcpManager:   mcpManager,
		toolExecutor: toolExecutor,
		updateChan:   make(chan interface{}, 100), // Buffered channel for updates
	}

	// Set up the callback for MCP status updates
	mcpManager.SetUpdateCallback(agent.broadcastUpdate)

	return agent, nil
}

// agentLogger adapts standard log.Logger to the MCP Logger interface
type agentLogger struct {
	logger *log.Logger
}

func (a *agentLogger) Info(msg string, args ...interface{}) {
	a.logger.Printf("[INFO] "+msg, args...)
}

func (a *agentLogger) Error(msg string, args ...interface{}) {
	a.logger.Printf("[ERROR] "+msg, args...)
}

func (a *agentLogger) Debug(msg string, args ...interface{}) {
	a.logger.Printf("[DEBUG] "+msg, args...)
}

// Start starts the agent with the given context
func (a *Agent) Start(ctx context.Context) error {
	a.logger.Println("Starting Othello AI Agent")
	
	// Load MCP servers from standard mcp.json or fallback to config.yaml
	mcpConfig, err := config.LoadMCPConfig()
	if err != nil {
		a.logger.Printf("Failed to load MCP config: %v", err)
		return fmt.Errorf("failed to load MCP config: %w", err)
	}
	
	// Convert MCP standard format to internal ServerConfig format
	servers := config.ConvertMCPToServerConfigs(mcpConfig)
	
	// Initialize MCP servers
	for _, serverCfg := range servers {
		a.logger.Printf("Connecting to MCP server: %s", serverCfg.Name)
		if err := a.mcpManager.AddServer(ctx, serverCfg); err != nil {
			a.logger.Printf("Failed to connect to MCP server %s: %v", serverCfg.Name, err)
			// Continue with other servers even if one fails
			continue
		}
		a.logger.Printf("Successfully connected to MCP server: %s", serverCfg.Name)
	}
	
	a.logger.Printf("Agent started with model: %s", a.config.Model.Name)
	return nil
}

// Stop gracefully stops the agent
func (a *Agent) Stop(ctx context.Context) error {
	a.logger.Println("Stopping Othello AI Agent")
	
	// Stop MCP connections
	if err := a.mcpManager.Close(ctx); err != nil {
		a.logger.Printf("Error stopping MCP connections: %v", err)
	}
	
	a.logger.Println("Agent stopped")
	return nil
}

// StartTUI starts the terminal user interface
func (a *Agent) StartTUI() error {
	a.logger.Println("Starting TUI mode")
	
	// Create TUI application with agent integration
	keymap := tui.DefaultKeyMap()
	styles := tui.DefaultStyles()
	app := tui.NewApplicationWithAgent(keymap, styles, a)
	
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

// GetMCPServers returns information about all registered MCP servers
func (a *Agent) GetMCPServers() []tui.ServerInfo {
	mcpServers := a.mcpManager.ListServers()
	
	// Convert agent.ServerInfo to tui.ServerInfo
	servers := make([]tui.ServerInfo, len(mcpServers))
	for i, mcpServer := range mcpServers {
		servers[i] = tui.ServerInfo{
			Name:      mcpServer.Name,
			Status:    mcpServer.Status,
			Connected: mcpServer.Connected,
			ToolCount: mcpServer.ToolCount,
			Transport: mcpServer.Transport,
			Error:     mcpServer.Error,
		}
	}
	
	return servers
}

// GetMCPTools returns all available tools from registered MCP servers
func (a *Agent) GetMCPTools(ctx context.Context) ([]tui.Tool, error) {
	mcpTools := a.mcpRegistry.ListTools()
	
	// Convert mcp.Tool to tui.Tool
	tools := make([]tui.Tool, len(mcpTools))
	for i, mcpTool := range mcpTools {
		tools[i] = tui.Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			Server:      mcpTool.ServerName,
		}
	}
	
	return tools, nil
}

// GetMCPToolsAsDefinitions converts MCP tools to model.ToolDefinition format
func (a *Agent) GetMCPToolsAsDefinitions(ctx context.Context) ([]model.ToolDefinition, error) {
	mcpTools := a.mcpRegistry.ListTools()
	
	// Convert mcp.Tool to model.ToolDefinition
	definitions := make([]model.ToolDefinition, len(mcpTools))
	for i, mcpTool := range mcpTools {
		// Create basic parameters structure
		parameters := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
			"required": []string{},
		}
		
		// TODO: In a more sophisticated implementation, we would parse
		// the actual MCP tool schema to create proper parameters
		
		definitions[i] = model.ToolDefinition{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			Parameters:  parameters,
		}
	}
	
	return definitions, nil
}

// SubscribeToUpdates returns a channel for receiving status updates
func (a *Agent) SubscribeToUpdates() <-chan interface{} {
	return a.updateChan
}

// ExecuteTool executes an MCP tool with the given parameters
func (a *Agent) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*tui.ToolExecutionResult, error) {
	a.logger.Printf("Executing tool: %s with params: %+v", toolName, params)
	
	// Execute the tool using the tool executor
	result, err := a.toolExecutor.Execute(ctx, toolName, params)
	if err != nil {
		a.logger.Printf("Tool execution failed for %s: %v", toolName, err)
		return &tui.ToolExecutionResult{
			ToolName: toolName,
			Success:  false,
			Error:    err.Error(),
		}, nil
	}
	
	a.logger.Printf("Tool %s executed successfully", toolName)
	
	// Broadcast tool execution update
	a.broadcastUpdate(tui.ToolExecutionMsg{
		ToolName: toolName,
		Success:  true,
		Result:   result.Result,
	})
	
	return &tui.ToolExecutionResult{
		ToolName: toolName,
		Success:  true,
		Result:   result.Result,
		Duration: result.Duration,
	}, nil
}

// broadcastUpdate sends an update to all subscribers (non-blocking)
func (a *Agent) broadcastUpdate(update interface{}) {
	select {
	case a.updateChan <- update:
		// Update sent successfully
	default:
		// Channel is full, drop the update to avoid blocking
		a.logger.Printf("Warning: Update channel full, dropping update")
	}
}