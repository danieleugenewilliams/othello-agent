# Test-Driven Development Implementation Plan
## Othello AI Agent - MCP-TUI Integration

**Version:** 1.0  
**Date:** October 10, 2025  
**Status:** Active Implementation  
**Owner:** Development Team  

---

## Table of Contents

1. [Overview](#overview)
2. [Week 1: Agent-MCP Integration](#week-1-agent-mcp-integration)
3. [Week 2: TUI-MCP Integration](#week-2-tui-mcp-integration)
4. [Week 3: Chat-Tool Integration](#week-3-chat-tool-integration)
5. [Week 4: Model-Tool Integration](#week-4-model-tool-integration)
6. [Week 5: Real-time Notifications & Polish](#week-5-real-time-notifications--polish)
7. [Testing Guidelines](#testing-guidelines)
8. [Acceptance Criteria](#acceptance-criteria)

---

## Overview

### Purpose

This document provides a detailed, test-driven implementation plan for integrating MCP tool capabilities into the Othello TUI. Following TDD principles, we write tests first, then implement the functionality to make tests pass.

### Principles

1. **Red-Green-Refactor**: Write failing test → Implement minimal code → Refactor
2. **Test First**: No production code without a failing test
3. **Incremental**: Small, testable changes over large rewrites
4. **Integration**: Test component integration early and often
5. **Documentation**: Tests serve as living documentation

### Success Metrics

- ✅ All tests pass before moving to next task
- ✅ Code coverage > 80% for new code
- ✅ Integration tests validate end-to-end flows
- ✅ Manual testing checklist completed each week

---

## Week 1: Agent-MCP Integration

**Goal**: Wire Agent to use MCP Registry, Executor, and Manager

### Day 1: MCP Manager Component

#### Task 1.1: Create MCP Manager Test File

**File**: `internal/agent/mcp_manager_test.go`

```go
package agent

import (
	"context"
	"testing"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPManager(t *testing.T) {
	t.Run("creates manager with valid parameters", func(t *testing.T) {
		registry := mcp.NewToolRegistry(newTestLogger())
		logger := newTestLogger()
		
		manager := NewMCPManager(registry, logger)
		
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.registry)
		assert.NotNil(t, manager.logger)
		assert.NotNil(t, manager.clients)
		assert.NotNil(t, manager.factory)
	})
}

func TestMCPManager_AddServer(t *testing.T) {
	tests := []struct {
		name        string
		serverCfg   config.ServerConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "successfully adds stdio server",
			serverCfg: config.ServerConfig{
				Name:      "test-server",
				Command:   "echo",
				Args:      []string{"test"},
				Transport: "stdio",
			},
			wantErr: false,
		},
		{
			name: "fails with invalid command",
			serverCfg: config.ServerConfig{
				Name:      "invalid-server",
				Command:   "nonexistent-command-xyz",
				Transport: "stdio",
			},
			wantErr:     true,
			errContains: "connect to server",
		},
		{
			name: "fails with empty name",
			serverCfg: config.ServerConfig{
				Name:      "",
				Command:   "echo",
				Transport: "stdio",
			},
			wantErr:     true,
			errContains: "server name cannot be empty",
		},
		{
			name: "fails with duplicate server name",
			serverCfg: config.ServerConfig{
				Name:      "duplicate",
				Command:   "echo",
				Transport: "stdio",
			},
			wantErr:     true,
			errContains: "server already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := setupTestManager(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := manager.AddServer(ctx, tt.serverCfg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				
				// Verify server was added
				servers := manager.ListServers()
				found := false
				for _, s := range servers {
					if s.Name == tt.serverCfg.Name {
						found = true
						assert.True(t, s.Connected)
						break
					}
				}
				assert.True(t, found, "Server should be in list")
			}
		})
	}
}

func TestMCPManager_RemoveServer(t *testing.T) {
	t.Run("removes existing server", func(t *testing.T) {
		manager := setupTestManager(t)
		ctx := context.Background()
		
		// Add server first
		cfg := config.ServerConfig{
			Name:      "test-server",
			Command:   "echo",
			Transport: "stdio",
		}
		require.NoError(t, manager.AddServer(ctx, cfg))
		
		// Remove server
		err := manager.RemoveServer(ctx, "test-server")
		require.NoError(t, err)
		
		// Verify removed
		servers := manager.ListServers()
		for _, s := range servers {
			assert.NotEqual(t, "test-server", s.Name)
		}
	})

	t.Run("fails to remove non-existent server", func(t *testing.T) {
		manager := setupTestManager(t)
		ctx := context.Background()
		
		err := manager.RemoveServer(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server not found")
	})
}

func TestMCPManager_ListServers(t *testing.T) {
	t.Run("lists all servers with status", func(t *testing.T) {
		manager := setupTestManager(t)
		ctx := context.Background()
		
		// Add multiple servers
		servers := []config.ServerConfig{
			{Name: "server1", Command: "echo", Transport: "stdio"},
			{Name: "server2", Command: "echo", Transport: "stdio"},
		}
		
		for _, cfg := range servers {
			require.NoError(t, manager.AddServer(ctx, cfg))
		}
		
		list := manager.ListServers()
		assert.Len(t, list, 2)
		
		// Verify server info structure
		for _, info := range list {
			assert.NotEmpty(t, info.Name)
			assert.NotEmpty(t, info.Status)
			assert.NotEmpty(t, info.Transport)
		}
	})
}

func TestMCPManager_GetServer(t *testing.T) {
	t.Run("retrieves existing server client", func(t *testing.T) {
		manager := setupTestManager(t)
		ctx := context.Background()
		
		cfg := config.ServerConfig{
			Name:      "test-server",
			Command:   "echo",
			Transport: "stdio",
		}
		require.NoError(t, manager.AddServer(ctx, cfg))
		
		client, exists := manager.GetServer("test-server")
		assert.True(t, exists)
		assert.NotNil(t, client)
	})

	t.Run("returns false for non-existent server", func(t *testing.T) {
		manager := setupTestManager(t)
		
		client, exists := manager.GetServer("non-existent")
		assert.False(t, exists)
		assert.Nil(t, client)
	})
}

func TestMCPManager_ServerLifecycle(t *testing.T) {
	t.Run("handles full server lifecycle", func(t *testing.T) {
		manager := setupTestManager(t)
		ctx := context.Background()
		
		cfg := config.ServerConfig{
			Name:      "lifecycle-server",
			Command:   "echo",
			Transport: "stdio",
		}
		
		// Add
		require.NoError(t, manager.AddServer(ctx, cfg))
		servers := manager.ListServers()
		require.Len(t, servers, 1)
		assert.True(t, servers[0].Connected)
		
		// Get
		client, exists := manager.GetServer("lifecycle-server")
		require.True(t, exists)
		assert.True(t, client.IsConnected())
		
		// Remove
		require.NoError(t, manager.RemoveServer(ctx, "lifecycle-server"))
		servers = manager.ListServers()
		assert.Len(t, servers, 0)
		
		// Verify cleanup
		client, exists = manager.GetServer("lifecycle-server")
		assert.False(t, exists)
	})
}

func TestMCPManager_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent server operations", func(t *testing.T) {
		manager := setupTestManager(t)
		ctx := context.Background()
		
		// Add multiple servers concurrently
		errChan := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				cfg := config.ServerConfig{
					Name:      fmt.Sprintf("server-%d", id),
					Command:   "echo",
					Transport: "stdio",
				}
				errChan <- manager.AddServer(ctx, cfg)
			}(i)
		}
		
		// Collect errors
		for i := 0; i < 10; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
		
		// Verify all added
		servers := manager.ListServers()
		assert.Len(t, servers, 10)
	})
}

// Test helpers

func setupTestManager(t *testing.T) *MCPManager {
	t.Helper()
	registry := mcp.NewToolRegistry(newTestLogger())
	logger := newTestLogger()
	return NewMCPManager(registry, logger)
}

func newTestLogger() Logger {
	return &testLogger{}
}

type testLogger struct{}

func (l *testLogger) Info(msg string, args ...interface{})  {}
func (l *testLogger) Error(msg string, args ...interface{}) {}
func (l *testLogger) Debug(msg string, args ...interface{}) {}
```

**Action**: Run `go test ./internal/agent -v -run TestMCPManager` (should fail)

#### Task 1.2: Implement MCP Manager

**File**: `internal/agent/mcp_manager.go`

```go
package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
)

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
	registry *mcp.ToolRegistry
	clients  map[string]mcp.Client
	factory  *mcp.ClientFactory
	logger   Logger
	mutex    sync.RWMutex
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
```

**Action**: Run `go test ./internal/agent -v -run TestMCPManager` (should pass)

### Day 2: Integrate MCP Manager into Agent

#### Task 2.1: Agent Tests with MCP

**File**: `internal/agent/agent_test.go` (add to existing file)

```go
func TestAgent_MCPIntegration(t *testing.T) {
	t.Run("creates agent with MCP components", func(t *testing.T) {
		cfg := &config.Config{
			Model: config.ModelConfig{
				Type: "ollama",
				Name: "qwen2.5:3b",
			},
			Ollama: config.OllamaConfig{
				Host: "http://localhost:11434",
			},
			MCP: config.MCPConfig{
				Servers: []config.ServerConfig{},
				Timeout: 5 * time.Second,
			},
		}

		agent, err := New(cfg)
		require.NoError(t, err)
		assert.NotNil(t, agent.mcpManager)
		assert.NotNil(t, agent.mcpRegistry)
		assert.NotNil(t, agent.mcpExecutor)
	})
}

func TestAgent_StartWithMCPServers(t *testing.T) {
	t.Run("connects to configured MCP servers on start", func(t *testing.T) {
		cfg := &config.Config{
			Model: config.ModelConfig{
				Type: "ollama",
				Name: "qwen2.5:3b",
			},
			Ollama: config.OllamaConfig{
				Host: "http://localhost:11434",
			},
			MCP: config.MCPConfig{
				Servers: []config.ServerConfig{
					{
						Name:      "test-server",
						Command:   "echo",
						Args:      []string{"test"},
						Transport: "stdio",
					},
				},
				Timeout: 5 * time.Second,
			},
		}

		agent, err := New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = agent.Start(ctx)
		require.NoError(t, err)

		// Verify server was connected
		servers := agent.GetServerStatus()
		assert.Len(t, servers, 1)
		assert.Equal(t, "test-server", servers[0].Name)
	})

	t.Run("continues with other servers if one fails", func(t *testing.T) {
		cfg := &config.Config{
			Model: config.ModelConfig{
				Type: "ollama",
				Name: "qwen2.5:3b",
			},
			Ollama: config.OllamaConfig{
				Host: "http://localhost:11434",
			},
			MCP: config.MCPConfig{
				Servers: []config.ServerConfig{
					{
						Name:      "invalid-server",
						Command:   "nonexistent-command",
						Transport: "stdio",
					},
					{
						Name:      "valid-server",
						Command:   "echo",
						Transport: "stdio",
					},
				},
				Timeout: 5 * time.Second,
			},
		}

		agent, err := New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = agent.Start(ctx)
		require.NoError(t, err)

		// Should have at least the valid server
		servers := agent.GetServerStatus()
		hasValid := false
		for _, s := range servers {
			if s.Name == "valid-server" {
				hasValid = true
				break
			}
		}
		assert.True(t, hasValid)
	})
}

func TestAgent_GetAvailableTools(t *testing.T) {
	t.Run("returns tools from all connected servers", func(t *testing.T) {
		agent := setupTestAgentWithMCP(t)

		tools := agent.GetAvailableTools()
		assert.NotEmpty(t, tools)

		// Verify tool structure
		for _, tool := range tools {
			assert.NotEmpty(t, tool.Name)
			assert.NotEmpty(t, tool.ServerName)
		}
	})
}

func TestAgent_ExecuteTool(t *testing.T) {
	t.Run("executes tool successfully", func(t *testing.T) {
		agent := setupTestAgentWithMCP(t)
		ctx := context.Background()

		// Assuming test server has a test tool
		result, err := agent.ExecuteTool(ctx, "test_tool", map[string]interface{}{
			"param": "value",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("returns error for unknown tool", func(t *testing.T) {
		agent := setupTestAgentWithMCP(t)
		ctx := context.Background()

		result, err := agent.ExecuteTool(ctx, "nonexistent_tool", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.NotNil(t, result)
		assert.NotNil(t, result.Error)
	})
}

func TestAgent_GetServerStatus(t *testing.T) {
	t.Run("returns status of all servers", func(t *testing.T) {
		agent := setupTestAgentWithMCP(t)

		status := agent.GetServerStatus()
		assert.NotEmpty(t, status)

		for _, s := range status {
			assert.NotEmpty(t, s.Name)
			assert.NotEmpty(t, s.Status)
		}
	})
}

func TestAgent_AddMCPServer(t *testing.T) {
	t.Run("adds server at runtime", func(t *testing.T) {
		agent := setupTestAgentWithMCP(t)
		ctx := context.Background()

		cfg := config.ServerConfig{
			Name:      "runtime-server",
			Command:   "echo",
			Transport: "stdio",
		}

		err := agent.AddMCPServer(ctx, cfg)
		require.NoError(t, err)

		servers := agent.GetServerStatus()
		found := false
		for _, s := range servers {
			if s.Name == "runtime-server" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func TestAgent_RemoveMCPServer(t *testing.T) {
	t.Run("removes server at runtime", func(t *testing.T) {
		agent := setupTestAgentWithMCP(t)
		ctx := context.Background()

		// Add server first
		cfg := config.ServerConfig{
			Name:      "temp-server",
			Command:   "echo",
			Transport: "stdio",
		}
		require.NoError(t, agent.AddMCPServer(ctx, cfg))

		// Remove it
		err := agent.RemoveMCPServer(ctx, "temp-server")
		require.NoError(t, err)

		servers := agent.GetServerStatus()
		for _, s := range servers {
			assert.NotEqual(t, "temp-server", s.Name)
		}
	})
}

// Test helpers

func setupTestAgentWithMCP(t *testing.T) *Agent {
	t.Helper()

	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		Ollama: config.OllamaConfig{
			Host: "http://localhost:11434",
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "test-server",
					Command:   "echo",
					Transport: "stdio",
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)

	return agent
}
```

**Action**: Run `go test ./internal/agent -v` (should fail for new tests)

#### Task 2.2: Update Agent Implementation

**File**: `internal/agent/agent.go`

```go
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
	config      *config.Config
	logger      *log.Logger
	model       model.Model
	mcpManager  *MCPManager
	mcpRegistry *mcp.ToolRegistry
	mcpExecutor *mcp.ToolExecutor
}

// Interface defines the agent's public API
type Interface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	StartTUI() error
	GetStatus() *Status
	GetAvailableTools() []mcp.Tool
	ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (*mcp.ExecuteResult, error)
	GetServerStatus() []ServerInfo
	AddMCPServer(ctx context.Context, cfg config.ServerConfig) error
	RemoveMCPServer(ctx context.Context, name string) error
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

	// Create structured logger
	structuredLogger := newStructuredLogger()

	// Initialize MCP components
	registry := mcp.NewToolRegistry(structuredLogger)
	executor := mcp.NewToolExecutor(registry, structuredLogger)
	manager := NewMCPManager(registry, structuredLogger)

	// Create model
	var mdl model.Model
	switch cfg.Model.Type {
	case "ollama":
		mdl = model.NewOllamaModel(cfg.Ollama.Host, cfg.Model.Name)
	default:
		return nil, fmt.Errorf("unsupported model type: %s", cfg.Model.Type)
	}

	agent := &Agent{
		config:      cfg,
		logger:      log.New(log.Writer(), "[AGENT] ", log.LstdFlags),
		model:       mdl,
		mcpManager:  manager,
		mcpRegistry: registry,
		mcpExecutor: executor,
	}

	return agent, nil
}

// Start starts the agent with the given context
func (a *Agent) Start(ctx context.Context) error {
	a.logger.Println("Starting Othello AI Agent")

	// Connect to configured MCP servers
	for _, serverCfg := range a.config.MCP.Servers {
		a.logger.Printf("Connecting to MCP server: %s", serverCfg.Name)
		if err := a.mcpManager.AddServer(ctx, serverCfg); err != nil {
			a.logger.Printf("Warning: Failed to connect to server %s: %v",
				serverCfg.Name, err)
			// Continue with other servers
		}
	}

	// Refresh tool registry
	if err := a.mcpRegistry.RefreshTools(ctx); err != nil {
		a.logger.Printf("Warning: Failed to refresh tools: %v", err)
	}

	a.logger.Printf("Agent started with model: %s", a.config.Model.Name)
	a.logger.Printf("Connected to %d MCP servers", len(a.mcpManager.ListServers()))

	return nil
}

// Stop gracefully stops the agent
func (a *Agent) Stop(ctx context.Context) error {
	a.logger.Println("Stopping Othello AI Agent")

	// Close MCP connections
	if err := a.mcpManager.Close(ctx); err != nil {
		a.logger.Printf("Warning: Error closing MCP connections: %v", err)
	}

	a.logger.Println("Agent stopped")
	return nil
}

// StartTUI starts the terminal user interface
func (a *Agent) StartTUI() error {
	a.logger.Println("Starting TUI mode")

	// Create and start the TUI application with agent reference
	app := tui.NewApplication(a.model, a)

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
	servers := a.mcpManager.ListServers()

	return &Status{
		Running:        true,
		ConfigFile:     "config.yaml",
		ModelConnected: true,
		MCPServers:     len(servers),
	}
}

// GetAvailableTools returns all tools from connected MCP servers
func (a *Agent) GetAvailableTools() []mcp.Tool {
	return a.mcpRegistry.GetAllTools()
}

// ExecuteTool executes an MCP tool with the given parameters
func (a *Agent) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (*mcp.ExecuteResult, error) {
	return a.mcpExecutor.Execute(ctx, name, params)
}

// GetServerStatus returns the status of all MCP servers
func (a *Agent) GetServerStatus() []ServerInfo {
	return a.mcpManager.ListServers()
}

// AddMCPServer adds a new MCP server at runtime
func (a *Agent) AddMCPServer(ctx context.Context, cfg config.ServerConfig) error {
	return a.mcpManager.AddServer(ctx, cfg)
}

// RemoveMCPServer removes an MCP server at runtime
func (a *Agent) RemoveMCPServer(ctx context.Context, name string) error {
	return a.mcpManager.RemoveServer(ctx, name)
}

// GenerateResponse generates a response from the model
func (a *Agent) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	response, err := a.model.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("generate response: %w", err)
	}
	return response.Content, nil
}

// Structured logger implementation
type structuredLogger struct {
	*log.Logger
}

func newStructuredLogger() *structuredLogger {
	return &structuredLogger{
		Logger: log.New(log.Writer(), "[MCP] ", log.LstdFlags),
	}
}

func (l *structuredLogger) Info(msg string, args ...interface{}) {
	l.Printf("INFO: %s %v", msg, args)
}

func (l *structuredLogger) Error(msg string, args ...interface{}) {
	l.Printf("ERROR: %s %v", msg, args)
}

func (l *structuredLogger) Debug(msg string, args ...interface{}) {
	l.Printf("DEBUG: %s %v", msg, args)
}
```

**Action**: Run `go test ./internal/agent -v` (should pass)

### Day 3: Add Missing MCP Methods

Some methods referenced in tests may need to be added to the MCP registry:

**File**: `internal/mcp/registry.go` (add these methods if missing)

```go
// GetToolsByServer returns all tools from a specific server
func (r *ToolRegistry) GetToolsByServer(serverName string) []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var tools []Tool
	for _, tool := range r.tools {
		if tool.ServerName == serverName {
			tools = append(tools, tool)
		}
	}

	return tools
}

// GetAllTools returns all tools from all servers
func (r *ToolRegistry) GetAllTools() []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// GetServer returns a registered server client
func (r *ToolRegistry) GetServer(name string) (Client, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	client, exists := r.servers[name]
	return client, exists
}
```

**File**: `internal/mcp/client_factory.go` (add this if it doesn't exist)

```go
package mcp

import (
	"fmt"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
)

// ClientFactory creates MCP clients based on configuration
type ClientFactory struct {
	logger Logger
}

// NewClientFactory creates a new client factory
func NewClientFactory(logger Logger) *ClientFactory {
	return &ClientFactory{
		logger: logger,
	}
}

// CreateClient creates an MCP client based on the server configuration
func (f *ClientFactory) CreateClient(cfg config.ServerConfig) (Client, error) {
	switch cfg.Transport {
	case "stdio":
		return NewStdioClient(cfg, f.logger), nil
	case "http":
		return NewHTTPClient(cfg, f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", cfg.Transport)
	}
}
```

Add `GetTransport()` method to `Client` interface if missing:

**File**: `internal/mcp/types.go`

```go
type Client interface {
	// ... existing methods ...
	GetTransport() string
}
```

Implement in stdio_client.go and http_client.go:

```go
func (c *StdioClient) GetTransport() string {
	return "stdio"
}

func (c *HTTPClient) GetTransport() string {
	return "http"
}
```

**Action**: Run `go test ./internal/agent ./internal/mcp -v` (all should pass)

### Day 4-5: Integration Testing

**File**: `integration_test.go` (update existing or create)

```go
// +build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/agent"
	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_AgentWithMCP(t *testing.T) {
	t.Run("full agent lifecycle with MCP servers", func(t *testing.T) {
		cfg := &config.Config{
			Model: config.ModelConfig{
				Type: "ollama",
				Name: "qwen2.5:3b",
			},
			Ollama: config.OllamaConfig{
				Host: "http://localhost:11434",
			},
			MCP: config.MCPConfig{
				Servers: []config.ServerConfig{
					{
						Name:      "test-server",
						Command:   "echo",
						Args:      []string{"test"},
						Transport: "stdio",
					},
				},
				Timeout: 5 * time.Second,
			},
		}

		// Create agent
		a, err := agent.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()

		// Start agent
		err = a.Start(ctx)
		require.NoError(t, err)

		// Verify servers connected
		servers := a.GetServerStatus()
		assert.NotEmpty(t, servers)

		// Verify tools available
		tools := a.GetAvailableTools()
		assert.NotEmpty(t, tools)

		// Stop agent
		err = a.Stop(ctx)
		require.NoError(t, err)
	})
}
```

**Action**: Run `go test -tags=integration -v` (should pass)

---

## Week 2: TUI-MCP Integration

**Goal**: Connect TUI views to real MCP data

### Day 6: Define MCP-TUI Message Types

#### Task 6.1: Create Message Types

**File**: `internal/tui/messages.go`

Add comprehensive test first:

```go
package tui

import (
	"testing"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestMCPMessages(t *testing.T) {
	t.Run("MCPServerConnectedMsg structure", func(t *testing.T) {
		msg := MCPServerConnectedMsg{
			Name:      "test-server",
			ToolCount: 5,
		}
		assert.Equal(t, "test-server", msg.Name)
		assert.Equal(t, 5, msg.ToolCount)
	})

	t.Run("MCPServerDisconnectedMsg structure", func(t *testing.T) {
		err := fmt.Errorf("connection failed")
		msg := MCPServerDisconnectedMsg{
			Name:  "test-server",
			Error: err,
		}
		assert.Equal(t, "test-server", msg.Name)
		assert.Equal(t, err, msg.Error)
	})

	t.Run("MCPToolExecutingMsg structure", func(t *testing.T) {
		params := map[string]interface{}{"key": "value"}
		msg := MCPToolExecutingMsg{
			ToolName: "test_tool",
			Params:   params,
		}
		assert.Equal(t, "test_tool", msg.ToolName)
		assert.Equal(t, params, msg.Params)
	})

	t.Run("MCPToolExecutedMsg structure", func(t *testing.T) {
		result := &mcp.ExecuteResult{
			Tool: mcp.Tool{Name: "test_tool"},
		}
		msg := MCPToolExecutedMsg{
			ToolName: "test_tool",
			Result:   result,
		}
		assert.Equal(t, "test_tool", msg.ToolName)
		assert.Equal(t, result, msg.Result)
	})
}
```

Then implement the messages (already shown in MCP_TUI_INTEGRATION.md).

**Action**: Run `go test ./internal/tui -v -run TestMCPMessages` (should pass)

### Day 7-8: Wire ServerView to Real Data

Complete implementation as detailed in the integration plan, with tests for:
- Displaying real server data
- Handling MCP connection messages
- Refreshing server list
- Server action handling

### Day 9-10: Update Application to Pass Agent

Update `internal/tui/application.go` to accept and use the agent reference.

**Tests**: Verify application correctly initializes with agent and displays server data.

---

## Week 3: Chat-Tool Integration

Follow the detailed plan in MCP_TUI_INTEGRATION.md sections for:
- Tool call detection
- Tool execution from chat
- Displaying tool results

**Key Tests**:
- TestChatView_DetectsToolCalls
- TestChatView_ExecutesToolsFromMessage
- TestChatView_DisplaysToolExecution

---

## Week 4: Model-Tool Integration

Extend model interface to handle tool descriptions and parse tool calls.

**Key Tests**:
- TestModel_GenerateWithTools
- TestModel_ParseToolCalls
- TestModel_ToolSystemPrompt

---

## Week 5: Real-time Notifications & Polish

Implement notification subscription and handling.

**Key Tests**:
- TestApplication_ReceivesNotifications
- TestApplication_UpdatesUIOnNotification
- Integration tests for full flow

---

## Testing Guidelines

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/agent -v

# Run specific test
go test ./internal/agent -v -run TestAgent_MCPIntegration

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run integration tests
go test -tags=integration -v

# Run with race detector
go test -race ./...
```

### Test Structure

Each test file should follow this structure:

```go
package packagename

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Test functions
func TestFeature_Scenario(t *testing.T) {
    t.Run("specific behavior", func(t *testing.T) {
        // Arrange
        // Act
        // Assert
    })
}

// Helper functions
func setupTestEnvironment(t *testing.T) *Component {
    t.Helper()
    // Setup code
    return component
}
```

### Coverage Goals

- **Unit Tests**: > 80% coverage for all packages
- **Integration Tests**: Cover main user flows
- **Manual Tests**: UI/UX validation

---

## Acceptance Criteria

### Week 1 Complete When:
- [ ] All Agent MCP tests pass
- [ ] MCPManager tests pass
- [ ] Agent.Start() connects to configured servers
- [ ] Agent.GetAvailableTools() returns real tools
- [ ] Integration test passes

### Week 2 Complete When:
- [ ] ServerView displays real server data
- [ ] ServerView updates on MCP messages
- [ ] Refresh functionality works
- [ ] All TUI tests pass

### Week 3 Complete When:
- [ ] Chat detects tool calls
- [ ] Tools execute from chat messages
- [ ] Tool results display correctly
- [ ] Error handling works

### Week 4 Complete When:
- [ ] Model receives tool descriptions
- [ ] Model can request tools
- [ ] Tool call parsing works
- [ ] Model tests pass

### Week 5 Complete When:
- [ ] Notifications update UI in real-time
- [ ] All integration tests pass
- [ ] Manual test checklist complete
- [ ] Performance benchmarks met

---

## Daily Checklist

Each day:
1. [ ] Write tests first (Red)
2. [ ] Implement minimal code to pass (Green)
3. [ ] Refactor for clarity (Refactor)
4. [ ] Run full test suite
5. [ ] Update documentation
6. [ ] Commit changes with clear message

---

## Notes

- Keep test files close to implementation files
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error paths, not just happy paths
- Keep tests fast and isolated

---

*This TDD plan ensures every feature is validated before moving forward. Follow the Red-Green-Refactor cycle strictly.*
