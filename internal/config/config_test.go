package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaultConfig(t *testing.T) {
	// Test loading default configuration when no config file exists
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check default values
	assert.Equal(t, "ollama", cfg.Model.Type)
	assert.Equal(t, "qwen2.5:3b", cfg.Model.Name)
	assert.Equal(t, 0.7, cfg.Model.Temperature)
	assert.Equal(t, 2048, cfg.Model.MaxTokens)
	assert.Equal(t, 8192, cfg.Model.ContextLength)

	assert.Equal(t, "http://localhost:11434", cfg.Ollama.Host)
	assert.Equal(t, 30*time.Second, cfg.Ollama.Timeout)

	assert.Equal(t, "default", cfg.TUI.Theme)
	assert.True(t, cfg.TUI.ShowHints)
	assert.True(t, cfg.TUI.AutoScroll)

	assert.Equal(t, 1000, cfg.Storage.HistorySize)
	assert.Equal(t, time.Hour, cfg.Storage.CacheTTL)

	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr string
	}{
		{
			name: "empty model type",
			modify: func(c *Config) {
				c.Model.Type = ""
			},
			wantErr: "model.type cannot be empty",
		},
		{
			name: "empty model name",
			modify: func(c *Config) {
				c.Model.Name = ""
			},
			wantErr: "model.name cannot be empty",
		},
		{
			name: "invalid temperature low",
			modify: func(c *Config) {
				c.Model.Temperature = -0.1
			},
			wantErr: "model.temperature must be between 0 and 2",
		},
		{
			name: "invalid temperature high",
			modify: func(c *Config) {
				c.Model.Temperature = 2.1
			},
			wantErr: "model.temperature must be between 0 and 2",
		},
		{
			name: "invalid max tokens",
			modify: func(c *Config) {
				c.Model.MaxTokens = 0
			},
			wantErr: "model.max_tokens must be positive",
		},
		{
			name: "empty ollama host",
			modify: func(c *Config) {
				c.Ollama.Host = ""
			},
			wantErr: "ollama.host cannot be empty",
		},
		{
			name: "invalid timeout",
			modify: func(c *Config) {
				c.Ollama.Timeout = 0
			},
			wantErr: "ollama.timeout must be positive",
		},
		{
			name: "invalid history size",
			modify: func(c *Config) {
				c.Storage.HistorySize = 0
			},
			wantErr: "storage.history_size must be positive",
		},
		{
			name: "invalid cache ttl",
			modify: func(c *Config) {
				c.Storage.CacheTTL = 0
			},
			wantErr: "storage.cache_ttl must be positive",
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			wantErr: "logging.level must be one of: debug, info, warn, error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load()
			require.NoError(t, err)

			tt.modify(cfg)
			err = cfg.validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Save original HOME
	originalHome := os.Getenv("HOME")
	
	// Set temporary HOME
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create default config
	err := CreateDefaultConfig()
	require.NoError(t, err)

	// Check that config file was created
	configFile := filepath.Join(tempDir, ".othello", "config.yaml")
	_, err = os.Stat(configFile)
	assert.NoError(t, err)

	// Test that creating config again fails
	err = CreateDefaultConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file already exists")
}

func TestConfigFileLoading(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Create config file
	configContent := `
model:
  type: "test-model"
  name: "test-name"
  temperature: 0.5
  max_tokens: 1000
  context_length: 4000

ollama:
  host: "http://test:8080"
  timeout: "10s"

tui:
  theme: "dark"
  show_hints: false
  auto_scroll: false

storage:
  history_size: 500
  cache_ttl: "30m"
  data_dir: "/tmp/test"

logging:
  level: "debug"
  file: "/tmp/test.log"
  format: "json"
`

	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp directory to test config loading
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Load config
	cfg, err := Load()
	require.NoError(t, err)

	// Verify loaded values
	assert.Equal(t, "test-model", cfg.Model.Type)
	assert.Equal(t, "test-name", cfg.Model.Name)
	assert.Equal(t, 0.5, cfg.Model.Temperature)
	assert.Equal(t, 1000, cfg.Model.MaxTokens)
	assert.Equal(t, 4000, cfg.Model.ContextLength)

	assert.Equal(t, "http://test:8080", cfg.Ollama.Host)
	assert.Equal(t, 10*time.Second, cfg.Ollama.Timeout)

	assert.Equal(t, "dark", cfg.TUI.Theme)
	assert.False(t, cfg.TUI.ShowHints)
	assert.False(t, cfg.TUI.AutoScroll)

	assert.Equal(t, 500, cfg.Storage.HistorySize)
	assert.Equal(t, 30*time.Minute, cfg.Storage.CacheTTL)
	assert.Equal(t, "/tmp/test", cfg.Storage.DataDir)

	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "/tmp/test.log", cfg.Logging.File)
	assert.Equal(t, "json", cfg.Logging.Format)

	assert.Contains(t, cfg.ConfigFile(), "config.yaml")
}

func TestConfig_AddMCPServer(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")
	
	cfg := &Config{
		MCP: MCPConfig{
			Servers: []ServerConfig{},
		},
		configFile: configFile,
	}
	
	// Test adding a new server
	server := ServerConfig{
		Name:      "test-server",
		Command:   "echo",
		Args:      []string{"hello"},
		Transport: "stdio",
		Timeout:   30 * time.Second,
	}
	
	err := cfg.AddMCPServer(server)
	require.NoError(t, err)
	
	// Verify server was added
	assert.Len(t, cfg.MCP.Servers, 1)
	assert.Equal(t, "test-server", cfg.MCP.Servers[0].Name)
	assert.Equal(t, "echo", cfg.MCP.Servers[0].Command)
	assert.Equal(t, []string{"hello"}, cfg.MCP.Servers[0].Args)
	
	// Test adding duplicate server (should fail)
	err = cfg.AddMCPServer(server)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	
	// Verify only one server exists
	assert.Len(t, cfg.MCP.Servers, 1)
}

func TestConfig_RemoveMCPServer(t *testing.T) {
	// Create a config with test servers
	cfg := &Config{
		MCP: MCPConfig{
			Servers: []ServerConfig{
				{Name: "server1", Command: "echo", Transport: "stdio"},
				{Name: "server2", Command: "cat", Transport: "stdio"},
			},
		},
		configFile: filepath.Join(t.TempDir(), "test_config.yaml"),
	}
	
	// Test removing existing server
	err := cfg.RemoveMCPServer("server1")
	require.NoError(t, err)
	
	// Verify server was removed
	assert.Len(t, cfg.MCP.Servers, 1)
	assert.Equal(t, "server2", cfg.MCP.Servers[0].Name)
	
	// Test removing non-existent server
	err = cfg.RemoveMCPServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	
	// Verify remaining server is unchanged
	assert.Len(t, cfg.MCP.Servers, 1)
}

func TestConfig_GetMCPServer(t *testing.T) {
	cfg := &Config{
		MCP: MCPConfig{
			Servers: []ServerConfig{
				{Name: "server1", Command: "echo", Transport: "stdio"},
				{Name: "server2", Command: "cat", Transport: "stdio"},
			},
		},
	}
	
	// Test getting existing server
	server, err := cfg.GetMCPServer("server1")
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "server1", server.Name)
	assert.Equal(t, "echo", server.Command)
	
	// Test getting non-existent server
	server, err = cfg.GetMCPServer("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfig_ListMCPServers(t *testing.T) {
	cfg := &Config{
		MCP: MCPConfig{
			Servers: []ServerConfig{
				{Name: "server1", Command: "echo", Transport: "stdio"},
				{Name: "server2", Command: "cat", Transport: "stdio"},
			},
		},
	}
	
	servers := cfg.ListMCPServers()
	assert.Len(t, servers, 2)
	assert.Equal(t, "server1", servers[0].Name)
	assert.Equal(t, "server2", servers[1].Name)
	
	// Test empty list
	emptyConfig := &Config{MCP: MCPConfig{Servers: []ServerConfig{}}}
	servers = emptyConfig.ListMCPServers()
	assert.Len(t, servers, 0)
}