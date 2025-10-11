package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Model   ModelConfig   `mapstructure:"model" yaml:"model"`
	Ollama  OllamaConfig  `mapstructure:"ollama" yaml:"ollama"`
	TUI     TUIConfig     `mapstructure:"tui" yaml:"tui"`
	MCP     MCPConfig     `mapstructure:"mcp" yaml:"mcp"`
	Storage StorageConfig `mapstructure:"storage" yaml:"storage"`
	Logging LoggingConfig `mapstructure:"logging" yaml:"logging"`

	configFile string // Track which config file was loaded
}

// ModelConfig contains model-specific settings
type ModelConfig struct {
	Type          string  `mapstructure:"type" yaml:"type"`
	Name          string  `mapstructure:"name" yaml:"name"`
	Temperature   float64 `mapstructure:"temperature" yaml:"temperature"`
	MaxTokens     int     `mapstructure:"max_tokens" yaml:"max_tokens"`
	ContextLength int     `mapstructure:"context_length" yaml:"context_length"`
}

// OllamaConfig contains Ollama-specific settings
type OllamaConfig struct {
	Host    string        `mapstructure:"host" yaml:"host"`
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`
}

// TUIConfig contains terminal UI settings
type TUIConfig struct {
	Theme      string `mapstructure:"theme" yaml:"theme"`
	ShowHints  bool   `mapstructure:"show_hints" yaml:"show_hints"`
	AutoScroll bool   `mapstructure:"auto_scroll" yaml:"auto_scroll"`
}

// MCPConfig contains MCP server settings
type MCPConfig struct {
	Servers []ServerConfig `mapstructure:"servers" yaml:"servers"`
	Timeout time.Duration  `mapstructure:"timeout" yaml:"timeout"`
}

// ServerConfig represents an MCP server configuration
type ServerConfig struct {
	Name      string            `mapstructure:"name" yaml:"name"`
	Command   string            `mapstructure:"command" yaml:"command"`
	Args      []string          `mapstructure:"args" yaml:"args"`
	Env       map[string]string `mapstructure:"env" yaml:"env"`
	Transport string            `mapstructure:"transport" yaml:"transport"`
	Timeout   time.Duration     `mapstructure:"timeout" yaml:"timeout"`
}

// StorageConfig contains storage settings
type StorageConfig struct {
	HistorySize int           `mapstructure:"history_size" yaml:"history_size"`
	CacheTTL    time.Duration `mapstructure:"cache_ttl" yaml:"cache_ttl"`
	DataDir     string        `mapstructure:"data_dir" yaml:"data_dir"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level" yaml:"level"`
	File   string `mapstructure:"file" yaml:"file"`
	Format string `mapstructure:"format" yaml:"format"`
}

// ConfigFile returns the path to the configuration file that was loaded
func (c *Config) ConfigFile() string {
	return c.configFile
}

// Load loads the configuration from various sources
func Load() (*Config, error) {
	v := viper.New()

	// Set configuration file properties
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add search paths for configuration files
	v.AddConfigPath(".")
	
	// Add ~/.othello directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(homeDir, ".othello"))
	}
	
	// Add system config directory
	v.AddConfigPath("/etc/othello")

	// Set defaults
	setDefaults(v)

	// Set environment variable support
	v.SetEnvPrefix("OTHELLO")
	v.AutomaticEnv()

	// Read configuration file
	var configFile string
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, will use defaults
		configFile = "defaults (no config file found)"
	} else {
		configFile = v.ConfigFileUsed()
	}

	// Unmarshal configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	config.configFile = configFile

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Model defaults
	v.SetDefault("model.type", "ollama")
	v.SetDefault("model.name", "qwen2.5:3b")
	v.SetDefault("model.temperature", 0.7)
	v.SetDefault("model.max_tokens", 2048)
	v.SetDefault("model.context_length", 8192)

	// Ollama defaults
	v.SetDefault("ollama.host", "http://localhost:11434")
	v.SetDefault("ollama.timeout", "30s")

	// TUI defaults
	v.SetDefault("tui.theme", "default")
	v.SetDefault("tui.show_hints", true)
	v.SetDefault("tui.auto_scroll", true)

	// Storage defaults
	v.SetDefault("storage.history_size", 1000)
	v.SetDefault("storage.cache_ttl", "1h")
	
	// Set default data directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		v.SetDefault("storage.data_dir", filepath.Join(homeDir, ".othello"))
	} else {
		v.SetDefault("storage.data_dir", ".othello")
	}

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	
	// Set default log file path
	if homeDir, err := os.UserHomeDir(); err == nil {
		v.SetDefault("logging.file", filepath.Join(homeDir, ".othello", "logs", "othello.log"))
	} else {
		v.SetDefault("logging.file", "othello.log")
	}

	// MCP defaults (empty servers list)
	v.SetDefault("mcp.servers", []ServerConfig{})
}

// validate validates the configuration
func (c *Config) validate() error {
	// Validate model configuration
	if c.Model.Type == "" {
		return fmt.Errorf("model.type cannot be empty")
	}
	if c.Model.Name == "" {
		return fmt.Errorf("model.name cannot be empty")
	}
	if c.Model.Temperature < 0 || c.Model.Temperature > 2 {
		return fmt.Errorf("model.temperature must be between 0 and 2")
	}
	if c.Model.MaxTokens <= 0 {
		return fmt.Errorf("model.max_tokens must be positive")
	}

	// Validate Ollama configuration
	if c.Ollama.Host == "" {
		return fmt.Errorf("ollama.host cannot be empty")
	}
	if c.Ollama.Timeout <= 0 {
		return fmt.Errorf("ollama.timeout must be positive")
	}

	// Validate storage configuration
	if c.Storage.HistorySize <= 0 {
		return fmt.Errorf("storage.history_size must be positive")
	}
	if c.Storage.CacheTTL <= 0 {
		return fmt.Errorf("storage.cache_ttl must be positive")
	}

	// Validate logging configuration
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}

	return nil
}

// Save writes the current configuration to the config file
func (c *Config) Save() error {
	if c.configFile == "" || c.configFile == "defaults (no config file found)" {
		// No config file exists, create one
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		
		configDir := filepath.Join(homeDir, ".othello")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		
		c.configFile = filepath.Join(configDir, "config.yaml")
	}
	
	// Create viper instance and marshal the config
	v := viper.New()
	v.SetConfigType("yaml")
	
	// Set all values from current config
	v.Set("model", c.Model)
	v.Set("ollama", c.Ollama)
	v.Set("tui", c.TUI)
	v.Set("mcp", c.MCP)
	v.Set("storage", c.Storage)
	v.Set("logging", c.Logging)
	
	// Write to file
	if err := v.WriteConfigAs(c.configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// AddMCPServer adds a new MCP server to the configuration
func (c *Config) AddMCPServer(server ServerConfig) error {
	// Check if server with same name already exists
	for _, existing := range c.MCP.Servers {
		if existing.Name == server.Name {
			return fmt.Errorf("server with name '%s' already exists", server.Name)
		}
	}
	
	// Add the server
	c.MCP.Servers = append(c.MCP.Servers, server)
	
	// Save the configuration
	return c.Save()
}

// RemoveMCPServer removes an MCP server from the configuration
func (c *Config) RemoveMCPServer(name string) error {
	found := false
	newServers := make([]ServerConfig, 0, len(c.MCP.Servers))
	
	for _, server := range c.MCP.Servers {
		if server.Name != name {
			newServers = append(newServers, server)
		} else {
			found = true
		}
	}
	
	if !found {
		return fmt.Errorf("server with name '%s' not found", name)
	}
	
	c.MCP.Servers = newServers
	
	// Save the configuration
	return c.Save()
}

// ListMCPServers returns all configured MCP servers
func (c *Config) ListMCPServers() []ServerConfig {
	return c.MCP.Servers
}

// GetMCPServer returns a specific MCP server by name
func (c *Config) GetMCPServer(name string) (*ServerConfig, error) {
	for _, server := range c.MCP.Servers {
		if server.Name == name {
			return &server, nil
		}
	}
	return nil, fmt.Errorf("server with name '%s' not found", name)
}

// CreateDefaultConfig creates a default configuration file in the user's home directory
func CreateDefaultConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".othello")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	
	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists: %s", configFile)
	}

	defaultConfig := `# Othello AI Agent Configuration

# Model configuration
model:
  type: "ollama"           # Model provider (ollama)
  name: "qwen2.5:3b"       # Model name
  temperature: 0.7         # Response creativity (0.0-2.0)
  max_tokens: 2048         # Maximum response length
  context_length: 8192     # Context window size

# Ollama configuration
ollama:
  host: "http://localhost:11434"  # Ollama server URL
  timeout: "30s"                  # Request timeout

# Terminal UI configuration
tui:
  theme: "default"         # UI theme
  show_hints: true         # Show keyboard hints
  auto_scroll: true        # Auto-scroll to new messages

# MCP server configuration
mcp:
  servers: []              # List of MCP servers (empty by default)
  # Example server configuration:
  # - name: "filesystem"
  #   command: "mcp-filesystem"
  #   args: ["--root", "/home/user"]
  #   transport: "stdio"
  #   timeout: "10s"

# Storage configuration
storage:
  history_size: 1000       # Maximum conversation history
  cache_ttl: "1h"          # Tool cache time-to-live
  data_dir: "~/.othello"   # Data directory

# Logging configuration
logging:
  level: "info"            # Log level (debug, info, warn, error)
  file: "~/.othello/logs/othello.log"  # Log file path
  format: "text"           # Log format (text, json)
`

	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Default configuration created: %s\n", configFile)
	return nil
}