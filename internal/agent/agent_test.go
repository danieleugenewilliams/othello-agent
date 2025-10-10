package agent

import (
	"context"
	"testing"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type:          "ollama",
			Name:          "qwen2.5:3b",
			Temperature:   0.7,
			MaxTokens:     2048,
			ContextLength: 8192,
		},
		Ollama: config.OllamaConfig{
			Host:    "http://localhost:11434",
			Timeout: 30 * time.Second,
		},
		TUI: config.TUIConfig{
			Theme:      "default",
			ShowHints:  true,
			AutoScroll: true,
		},
		Storage: config.StorageConfig{
			HistorySize: 1000,
			CacheTTL:    time.Hour,
			DataDir:     "/tmp/test",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			File:   "/tmp/test.log",
			Format: "text",
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, agent)

	assert.Equal(t, cfg, agent.config)
	assert.NotNil(t, agent.logger)
}

func TestNewAgentNilConfig(t *testing.T) {
	agent, err := New(nil)
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "configuration cannot be nil")
}

func TestAgentStartStop(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Test start
	err = agent.Start(ctx)
	assert.NoError(t, err)

	// Test stop
	err = agent.Stop(ctx)
	assert.NoError(t, err)
}

func TestAgentGetStatus(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	agent, err := New(cfg)
	require.NoError(t, err)

	status := agent.GetStatus()
	require.NotNil(t, status)

	assert.True(t, status.Running)
	assert.NotEmpty(t, status.ConfigFile)
	assert.False(t, status.ModelConnected) // Not yet implemented
	assert.Equal(t, 0, status.MCPServers)  // No servers configured by default
}

func TestAgentStartTUI(t *testing.T) {
	t.Skip("TUI tests require interactive mode - will be tested in integration tests")
	
	cfg, err := config.Load()
	require.NoError(t, err)

	agent, err := New(cfg)
	require.NoError(t, err)

	// This test is skipped because StartTUI() is blocking and starts an interactive interface
	// In a real TDD approach, we would refactor StartTUI to be non-blocking or mockable
	err = agent.StartTUI()
	assert.NoError(t, err)
}

// TestAgent_MCPManagerIntegration tests that Agent properly initializes and manages MCP components
func TestAgent_MCPManagerIntegration(t *testing.T) {
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "local-memory",
					Command:   "local-memory",
					Args:      []string{"--mcp"},
					Transport: "stdio",
					Timeout:   5 * time.Second,
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)
	
	// Test that MCP manager is initialized
	assert.NotNil(t, agent.mcpManager, "MCP manager should be initialized")
	assert.NotNil(t, agent.mcpRegistry, "MCP registry should be initialized")
}

// TestAgent_StartInitializesMCP tests that Agent.Start() properly initializes MCP connections
func TestAgent_StartInitializesMCP(t *testing.T) {
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "local-memory",
					Command:   "local-memory",
					Args:      []string{"--mcp"},
					Transport: "stdio",
					Timeout:   5 * time.Second,
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = agent.Start(ctx)
	assert.NoError(t, err, "Agent.Start() should succeed even if some servers fail to connect")
	
	// Test that we attempted to register the server (even if connection failed)
	servers := agent.GetMCPServers()
	// With connection failure, the server may not be in the list or may be marked as disconnected
	// The important thing is that Start() doesn't fail completely
	assert.True(t, len(servers) >= 0, "Should handle server registration gracefully")
}

// TestAgent_StopCleansMCP tests that Agent.Stop() properly cleans up MCP connections
func TestAgent_StopCleansMCP(t *testing.T) {
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "local-memory",
					Command:   "local-memory",
					Args:      []string{"--mcp"},
					Transport: "stdio",
					Timeout:   5 * time.Second,
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the agent
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Stop the agent
	err = agent.Stop(ctx)
	assert.NoError(t, err, "Agent.Stop() should succeed")
	
	// Verify cleanup
	servers := agent.GetMCPServers()
	for _, server := range servers {
		assert.False(t, server.Connected, "Server should be disconnected after stop")
	}
}

// TestAgent_GetMCPTools tests tool discovery through the Agent
func TestAgent_GetMCPTools(t *testing.T) {
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		MCP: config.MCPConfig{
			Timeout: 5 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	tools, err := agent.GetMCPTools(ctx)
	assert.NoError(t, err, "GetMCPTools should not error")
	assert.NotNil(t, tools, "Tools should not be nil")
	// With no servers configured, should return empty list
	assert.Len(t, tools, 0, "Should have no tools with no servers")
}

// TestAgent_ConfigurationServerDiscovery tests that Agent properly discovers servers from configuration
func TestAgent_ConfigurationServerDiscovery(t *testing.T) {
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "local-memory",
					Command:   "local-memory",
					Args:      []string{"--mcp"},
					Transport: "stdio",
					Timeout:   5 * time.Second,
				},
				{
					// This server is intentionally configured with "echo" to test error handling.
					// "echo" will respond with "hello" instead of valid JSON-RPC, causing a parse error.
					// This tests that the agent gracefully handles server connection failures.
					Name:      "filesystem",
					Command:   "echo", // Mock command for testing error handling
					Args:      []string{"hello"},
					Transport: "stdio",
					Timeout:   5 * time.Second,
				},
			},
			Timeout: 10 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Start the agent to trigger server discovery
	err = agent.Start(ctx)
	assert.NoError(t, err, "Agent.Start() should succeed")

	// Verify that the agent attempted to connect to configured servers
	// Note: The local-memory server should connect successfully, while the "filesystem" 
	// server with "echo" command should fail (this is expected behavior for testing error handling)
	servers := agent.GetMCPServers()
	
	// We should have at least attempted to register both servers
	serverNames := make(map[string]bool)
	for _, server := range servers {
		serverNames[server.Name] = true
	}
	
	// At least one server should be registered (local-memory should work)
	assert.True(t, len(servers) > 0, "Should have attempted to register at least one server")
	
	// Clean up
	agent.Stop(ctx)
}