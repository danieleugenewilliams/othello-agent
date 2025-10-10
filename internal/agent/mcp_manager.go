package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
)

// Update message types for notifications
type ServerStatusUpdate struct {
	ServerName string
	Connected  bool
	ToolCount  int
	Error      string
}

type ToolUpdate struct {
	ServerName string
	ToolCount  int
	Added      []string
	Removed    []string
}

// Logger interface for manager logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// ServerInfo contains information about an MCP server
type ServerInfo struct {
	Name      string
	Status    string
	Connected bool
	ToolCount int
	Transport string
	Error     string
}

// MCPManager manages MCP server connections and lifecycle
type MCPManager struct {
	registry     *mcp.ToolRegistry
	clients      map[string]mcp.Client
	factory      *mcp.DefaultClientFactory
	logger       Logger
	mutex        sync.RWMutex
	updateCallback func(interface{}) // Callback for status updates
}

// NewMCPManager creates a new MCP manager
func NewMCPManager(registry *mcp.ToolRegistry, logger Logger) *MCPManager {
	return &MCPManager{
		registry: registry,
		clients:  make(map[string]mcp.Client),
		factory:  mcp.NewClientFactory(logger),
		logger:   logger,
	}
}

// SetUpdateCallback sets the callback for status updates
func (m *MCPManager) SetUpdateCallback(callback func(interface{})) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.updateCallback = callback
}

// notifyUpdate sends an update if callback is set (call with mutex held)
func (m *MCPManager) notifyUpdate(update interface{}) {
	if m.updateCallback != nil {
		go m.updateCallback(update) // Send in goroutine to avoid blocking
	}
}

// AddServer adds and connects to an MCP server
func (m *MCPManager) AddServer(ctx context.Context, cfg config.ServerConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check for duplicate
	if _, exists := m.clients[cfg.Name]; exists {
		return fmt.Errorf("server already exists: %s", cfg.Name)
	}

	// Create client using factory
	client, err := m.factory.CreateClient(cfg)
	if err != nil {
		m.logger.Error("Failed to create client", "server", cfg.Name, "error", err)
		return fmt.Errorf("create client: %w", err)
	}

	// Connect to server
	if err := client.Connect(ctx); err != nil {
		m.logger.Error("Failed to connect to server", "server", cfg.Name, "error", err)
		return fmt.Errorf("connect to server: %w", err)
	}

	// Register with registry
	if err := m.registry.RegisterServer(cfg.Name, client); err != nil {
		client.Disconnect(ctx)
		m.logger.Error("Failed to register server", "server", cfg.Name, "error", err)
		return fmt.Errorf("register server: %w", err)
	}

	m.clients[cfg.Name] = client
	m.logger.Info("Added MCP server", "name", cfg.Name, "transport", cfg.Transport)

	// Notify of successful connection
	toolCount := len(m.registry.ListToolsForServer(cfg.Name))
	m.notifyUpdate(ServerStatusUpdate{
		ServerName: cfg.Name,
		Connected:  true,
		ToolCount:  toolCount,
		Error:      "",
	})

	return nil
}

// RemoveServer disconnects and removes an MCP server
func (m *MCPManager) RemoveServer(ctx context.Context, name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("server not found: %s", name)
	}

	// Disconnect client
	if err := client.Disconnect(ctx); err != nil {
		m.logger.Error("Error disconnecting from server", "server", name, "error", err)
	}

	// Unregister from registry
	m.registry.UnregisterServer(name)

	// Remove from map
	delete(m.clients, name)

	// Notify of disconnection
	m.notifyUpdate(ServerStatusUpdate{
		ServerName: name,
		Connected:  false,
		ToolCount:  0,
		Error:      "",
	})

	m.logger.Info("Removed MCP server", "name", name)
	return nil
}

// ListServers returns information about all registered servers
func (m *MCPManager) ListServers() []ServerInfo {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	servers := make([]ServerInfo, 0, len(m.clients))
	for name, client := range m.clients {
		connected := client.IsConnected()
		status := "disconnected"
		if connected {
			status = "connected"
		}

		// Get tool count from registry
		tools := m.registry.GetToolsByServer(name)

		info := ServerInfo{
			Name:      name,
			Status:    status,
			Connected: connected,
			ToolCount: len(tools),
			Transport: client.GetTransport(),
		}
		servers = append(servers, info)
	}

	return servers
}

// GetServer retrieves a server client by name
func (m *MCPManager) GetServer(name string) (mcp.Client, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	client, exists := m.clients[name]
	return client, exists
}

// RefreshTools refreshes tools from all connected servers
func (m *MCPManager) RefreshTools(ctx context.Context) error {
	return m.registry.RefreshTools(ctx)
}

// Close disconnects all servers
func (m *MCPManager) Close(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var errors []error
	for name, client := range m.clients {
		if err := client.Disconnect(ctx); err != nil {
			m.logger.Error("Error disconnecting from server", "server", name, "error", err)
			errors = append(errors, err)
		}
	}

	m.clients = make(map[string]mcp.Client)

	if len(errors) > 0 {
		return fmt.Errorf("errors disconnecting from %d servers", len(errors))
	}

	return nil
}