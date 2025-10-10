package mcp

import (
	"context"
	"time"
)

// Tool represents an MCP tool with its metadata and schema
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	ServerName  string                 `json:"serverName"`
	LastUpdated time.Time              `json:"lastUpdated"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError"`
}

// Content represents a piece of content in a tool result
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}

// Server represents an MCP server configuration
type Server struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"` // "stdio" or "http"
	Command   []string          `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Timeout   time.Duration     `json:"timeout"`
	Connected bool              `json:"connected"`
}

// Client interface for MCP server communication
type Client interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool

	// Tool operations
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, error)

	// Server information
	GetInfo(ctx context.Context) (*ServerInfo, error)
}

// ServerInfo contains information about an MCP server
type ServerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Protocol    string `json:"protocol"`
	Capabilities struct {
		Tools        bool `json:"tools"`
		Resources    bool `json:"resources"`
		Prompts      bool `json:"prompts"`
		Notifications bool `json:"notifications"`
	} `json:"capabilities"`
}

// Message represents an MCP protocol message
type Message struct {
	ID     interface{} `json:"id,omitempty"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

// Error represents an MCP protocol error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// Common MCP error codes
const (
	ErrorParseError     = -32700
	ErrorInvalidRequest = -32600
	ErrorMethodNotFound = -32601
	ErrorInvalidParams  = -32602
	ErrorInternalError  = -32603
)

// Tool execution request parameters
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// Tool list response
type ToolListResponse struct {
	Tools []Tool `json:"tools"`
}