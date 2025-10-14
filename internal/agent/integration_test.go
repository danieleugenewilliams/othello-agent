package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullAgentMCPLifecycle tests the complete end-to-end flow
func TestIntegration_FullAgentMCPLifecycle(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		Logging: config.LoggingConfig{
			File: logFile,
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "local-memory",
					Command:   "local-memory",
					Args:      []string{"--mcp"},
					Transport: "stdio",
					Timeout:   10 * time.Second,
				},
			},
			Timeout: 15 * time.Second,
		},
		Storage: config.StorageConfig{
			CacheTTL: time.Hour,
		},
	}

	// Test 1: Agent Creation
	agent, err := New(cfg)
	require.NoError(t, err, "Should create agent successfully")
	assert.NotNil(t, agent.mcpManager, "MCP manager should be initialized")
	assert.NotNil(t, agent.mcpRegistry, "MCP registry should be initialized")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 2: Agent Start and Server Connection
	err = agent.Start(ctx)
	require.NoError(t, err, "Agent should start successfully")

	// Allow time for server connection
	time.Sleep(2 * time.Second)

	// Test 3: Server Registration Verification
	servers := agent.GetMCPServers()
	require.Len(t, servers, 1, "Should have one registered server")
	
	server := servers[0]
	assert.Equal(t, "local-memory", server.Name)
	assert.Equal(t, "stdio", server.Transport)
	assert.True(t, server.Connected, "Server should be connected")
	assert.Greater(t, server.ToolCount, 0, "Should have discovered tools")

	// Test 4: Tool Discovery Verification
	tools, err := agent.GetMCPTools(ctx)
	require.NoError(t, err, "Should get tools successfully")
	assert.Greater(t, len(tools), 0, "Should have discovered tools from local-memory server")
	
	// Verify some expected tools from local-memory
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	
	expectedTools := []string{"store_memory", "search", "analysis", "domains", "categories"}
	for _, expectedTool := range expectedTools {
		assert.True(t, toolNames[expectedTool], "Should have tool: %s", expectedTool)
	}

	// Test 5: Agent Status
	status := agent.GetStatus()
	assert.True(t, status.Running)
	assert.Equal(t, 1, status.MCPServers)

	// Test 6: Agent Stop and Cleanup
	err = agent.Stop(ctx)
	assert.NoError(t, err, "Agent should stop successfully")

	// Verify cleanup
	servers = agent.GetMCPServers()
	for _, server := range servers {
		assert.False(t, server.Connected, "Server should be disconnected after stop")
	}
}

// TestIntegration_MultipleServerManagement tests handling multiple MCP servers
func TestIntegration_MultipleServerManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		Logging: config.LoggingConfig{
			File: logFile,
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "local-memory",
					Command:   "local-memory",
					Args:      []string{"--mcp"},
					Transport: "stdio",
					Timeout:   10 * time.Second,
				},
				{
					// This server will fail to connect (testing error handling)
					Name:      "invalid-server",
					Command:   "nonexistent-command",
					Args:      []string{},
					Transport: "stdio",
					Timeout:   2 * time.Second,
				},
			},
			Timeout: 15 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Start agent (should succeed even with failing server)
	err = agent.Start(ctx)
	assert.NoError(t, err, "Agent should start successfully even with some server failures")

	time.Sleep(3 * time.Second) // Allow time for connection attempts

	// Verify server states
	servers := agent.GetMCPServers()
	
	// Should have at least the successful server
	var connectedServers, failedServers int
	for _, server := range servers {
		if server.Connected {
			connectedServers++
			assert.Equal(t, "local-memory", server.Name, "Connected server should be local-memory")
		} else {
			failedServers++
		}
	}

	assert.Greater(t, connectedServers, 0, "Should have at least one connected server")
	// The invalid server may or may not be in the list depending on implementation

	// Clean up
	agent.Stop(ctx)
}

// TestIntegration_ToolRegistryOperations tests tool registry functionality
func TestIntegration_ToolRegistryOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		Logging: config.LoggingConfig{
			File: logFile,
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "local-memory",
					Command:   "local-memory",
					Args:      []string{"--mcp"},
					Transport: "stdio",
					Timeout:   10 * time.Second,
				},
			},
			Timeout: 15 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Start and connect
	err = agent.Start(ctx)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// Test tool discovery
	tools, err := agent.GetMCPTools(ctx)
	require.NoError(t, err)
	require.Greater(t, len(tools), 0, "Should have tools")

	// Test tool details
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name, "Tool should have a name")
		assert.NotEmpty(t, tool.Description, "Tool should have a description")
		// Schema can be empty for some tools
	}

	// Test registry functionality
	registry := agent.mcpRegistry
	assert.NotNil(t, registry, "Registry should be accessible")

	registryTools := registry.ListTools()
	assert.Equal(t, len(tools), len(registryTools), "Registry and agent should return same tools")

	// Clean up
	agent.Stop(ctx)
}

// TestIntegration_ErrorHandlingAndRecovery tests error scenarios
func TestIntegration_ErrorHandlingAndRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Test with completely invalid configuration
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type: "ollama",
			Name: "qwen2.5:3b",
		},
		Logging: config.LoggingConfig{
			File: logFile,
		},
		MCP: config.MCPConfig{
			Servers: []config.ServerConfig{
				{
					Name:      "completely-invalid",
					Command:   "this-command-does-not-exist",
					Args:      []string{"--invalid"},
					Transport: "stdio",
					Timeout:   1 * time.Second,
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err, "Agent creation should succeed even with invalid server config")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start should succeed even if servers fail
	err = agent.Start(ctx)
	assert.NoError(t, err, "Agent.Start() should not fail due to server connection issues")

	// Verify graceful handling
	servers := agent.GetMCPServers()
	// May have no servers or disconnected servers
	for _, server := range servers {
		if !server.Connected {
			assert.NotEmpty(t, server.Error, "Disconnected server should have error message")
		}
	}

	// Tools should return empty list
	tools, err := agent.GetMCPTools(ctx)
	assert.NoError(t, err, "GetMCPTools should not error")
	assert.Len(t, tools, 0, "Should have no tools with no connected servers")

	// Stop should work
	err = agent.Stop(ctx)
	assert.NoError(t, err, "Agent.Stop() should succeed")
}