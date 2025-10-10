package agent

import (
	"context"
	"fmt"
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
			name: "successfully adds local-memory server",
			serverCfg: config.ServerConfig{
				Name:      "local-memory",
				Command:   "local-memory",
				Args:      []string{"--mcp"},
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
				Command:   "local-memory",
				Transport: "stdio",
			},
			wantErr:     true,
			errContains: "server name cannot be empty",
		},
		{
			name: "fails with duplicate server name",
			serverCfg: config.ServerConfig{
				Name:      "duplicate",
				Command:   "local-memory",
				Args:      []string{"--mcp"},
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

			// For duplicate test, add server first
			if tt.name == "fails with duplicate server name" {
				err := manager.AddServer(ctx, tt.serverCfg)
				require.NoError(t, err)
			}

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
			Name:      "local-memory",
			Command:   "local-memory",
			Args:      []string{"--mcp"},
			Transport: "stdio",
		}
		require.NoError(t, manager.AddServer(ctx, cfg))
		
		// Remove server
		err := manager.RemoveServer(ctx, "local-memory")
		require.NoError(t, err)
		
		// Verify removed
		servers := manager.ListServers()
		for _, s := range servers {
			assert.NotEqual(t, "local-memory", s.Name)
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
			{Name: "server1", Command: "local-memory", Args: []string{"--mcp"}, Transport: "stdio"},
			{Name: "server2", Command: "local-memory", Args: []string{"--mcp"}, Transport: "stdio"},
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
			Name:      "local-memory",
			Command:   "local-memory",
			Args:      []string{"--mcp"},
			Transport: "stdio",
		}
		require.NoError(t, manager.AddServer(ctx, cfg))
		
		client, exists := manager.GetServer("local-memory")
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
			Command:   "local-memory",
			Args:      []string{"--mcp"},
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
		_, exists = manager.GetServer("lifecycle-server")
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
					Command:   "local-memory",
					Args:      []string{"--mcp"},
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