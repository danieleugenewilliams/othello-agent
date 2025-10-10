package mcp

import (
	"fmt"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
)

// NewClient creates a new MCP client based on the server transport configuration
func NewClient(server Server, logger Logger) (Client, error) {
	switch server.Transport {
	case "stdio":
		return NewSTDIOClient(server, logger), nil
	case "http":
		return NewHTTPClient(server, logger), nil
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", server.Transport)
	}
}

// NewClientFromConfig creates a new MCP client from a config.ServerConfig
func NewClientFromConfig(cfg config.ServerConfig, logger Logger) (Client, error) {
	server := ServerFromConfig(cfg)
	return NewClient(server, logger)
}

// ServerFromConfig converts a config.ServerConfig to an mcp.Server
func ServerFromConfig(cfg config.ServerConfig) Server {
	// Build command slice
	var command []string
	if cfg.Command != "" {
		command = []string{cfg.Command}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	return Server{
		Name:      cfg.Name,
		Transport: cfg.Transport,
		Command:   command,
		Args:      cfg.Args,
		Env:       cfg.Env,
		Timeout:   timeout,
	}
}

// ClientFactory provides a factory interface for creating MCP clients
type ClientFactory interface {
	CreateClient(cfg config.ServerConfig) (Client, error)
}

// DefaultClientFactory implements ClientFactory with support for stdio and http transports
type DefaultClientFactory struct {
	logger Logger
}

// NewClientFactory creates a new client factory
func NewClientFactory(logger Logger) *DefaultClientFactory {
	return &DefaultClientFactory{
		logger: logger,
	}
}

// CreateClient creates a client using the default factory
func (f *DefaultClientFactory) CreateClient(cfg config.ServerConfig) (Client, error) {
	return NewClientFromConfig(cfg, f.logger)
}