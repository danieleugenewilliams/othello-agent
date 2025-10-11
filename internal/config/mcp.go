package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MCPServerConfig represents the standard MCP server configuration
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPStandardConfig represents the standard MCP configuration format
type MCPStandardConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// LoadMCPConfig loads MCP configuration from ~/.othello/mcp.json
func LoadMCPConfig() (*MCPStandardConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	mcpConfigPath := filepath.Join(homeDir, ".othello", "mcp.json")
	
	// If mcp.json doesn't exist, return empty config
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		return &MCPStandardConfig{
			MCPServers: make(map[string]MCPServerConfig),
		}, nil
	}

	return loadMCPJSON(mcpConfigPath)
}

func loadMCPJSON(path string) (*MCPStandardConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcp.json: %w", err)
	}

	var mcpConfig MCPStandardConfig
	if err := json.Unmarshal(data, &mcpConfig); err != nil {
		return nil, fmt.Errorf("failed to parse mcp.json: %w", err)
	}

	return &mcpConfig, nil
}

// SaveMCPConfig saves the MCP configuration to ~/.othello/mcp.json
func SaveMCPConfig(mcpConfig *MCPStandardConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".othello")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	mcpConfigPath := filepath.Join(configDir, "mcp.json")

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mcp config: %w", err)
	}

	if err := os.WriteFile(mcpConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write mcp.json: %w", err)
	}

	return nil
}

// AddMCPServer adds a server to mcp.json
func AddMCPServer(name string, server MCPServerConfig) error {
	mcpConfig, err := LoadMCPConfig()
	if err != nil {
		return fmt.Errorf("failed to load mcp config: %w", err)
	}

	// Check if server already exists
	if _, exists := mcpConfig.MCPServers[name]; exists {
		return fmt.Errorf("server with name '%s' already exists", name)
	}

	mcpConfig.MCPServers[name] = server
	return SaveMCPConfig(mcpConfig)
}

// RemoveMCPServer removes a server from mcp.json
func RemoveMCPServer(name string) error {
	mcpConfig, err := LoadMCPConfig()
	if err != nil {
		return fmt.Errorf("failed to load mcp config: %w", err)
	}

	if _, exists := mcpConfig.MCPServers[name]; !exists {
		return fmt.Errorf("server with name '%s' not found", name)
	}

	delete(mcpConfig.MCPServers, name)
	return SaveMCPConfig(mcpConfig)
}

// ListMCPServers returns all servers from mcp.json
func ListMCPServers() (map[string]MCPServerConfig, error) {
	mcpConfig, err := LoadMCPConfig()
	if err != nil {
		return nil, err
	}

	return mcpConfig.MCPServers, nil
}

// ConvertMCPToServerConfigs converts MCP standard format to internal ServerConfig format
func ConvertMCPToServerConfigs(mcpConfig *MCPStandardConfig) []ServerConfig {
	servers := make([]ServerConfig, 0, len(mcpConfig.MCPServers))

	for name, mcpServer := range mcpConfig.MCPServers {
		server := ServerConfig{
			Name:      name,
			Command:   mcpServer.Command,
			Args:      mcpServer.Args,
			Env:       mcpServer.Env,
			Transport: "stdio", // Default transport for MCP
			Timeout:   30 * time.Second, // Default timeout
		}
		servers = append(servers, server)
	}

	return servers
}