package mcp

import (
	"fmt"
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

// ClientFactory provides a factory interface for creating MCP clients
type ClientFactory interface {
	CreateClient(server Server, logger Logger) (Client, error)
}

// DefaultClientFactory implements ClientFactory with support for stdio and http transports
type DefaultClientFactory struct{}

// CreateClient creates a client using the default factory
func (f *DefaultClientFactory) CreateClient(server Server, logger Logger) (Client, error) {
	return NewClient(server, logger)
}