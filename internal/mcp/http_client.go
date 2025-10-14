package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPClient implements the Client interface for HTTP-based MCP servers
type HTTPClient struct {
	server     Server
	httpClient *http.Client
	sessionID  string
	connected  int32 // atomic boolean
	requestID  int64
	logger     Logger
	mu         sync.RWMutex
}

// NewHTTPClient creates a new HTTP client for an MCP server
func NewHTTPClient(server Server, logger Logger) *HTTPClient {
	return &HTTPClient{
		server: server,
		httpClient: &http.Client{
			Timeout: server.Timeout,
		},
		logger: logger,
	}
}

// Connect establishes a connection to the MCP server via HTTP
func (c *HTTPClient) Connect(ctx context.Context) error {
	if atomic.LoadInt32(&c.connected) == 1 {
		return nil // Already connected
	}

	if c.server.URL == "" {
		return fmt.Errorf("no URL specified for HTTP server %s", c.server.Name)
	}

	// Send initialize request
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	atomic.StoreInt32(&c.connected, 1)
	c.logger.Info("Connected to HTTP MCP server", "name", c.server.Name, "url", c.server.URL)

	return nil
}

// Disconnect closes the connection to the MCP server
func (c *HTTPClient) Disconnect(ctx context.Context) error {
	if atomic.LoadInt32(&c.connected) == 0 {
		return nil // Already disconnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send DELETE request to terminate session if we have a session ID
	if c.sessionID != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.server.URL, nil)
		if err != nil {
			c.logger.Error("Failed to create disconnect request: %v", err)
		} else {
			c.setHeaders(req)
			resp, err := c.httpClient.Do(req)
			if err != nil {
				c.logger.Error("Failed to send disconnect request: %v", err)
			} else {
				resp.Body.Close()
			}
		}
	}

	atomic.StoreInt32(&c.connected, 0)
	c.sessionID = ""
	c.logger.Info("Disconnected from HTTP MCP server", "name", c.server.Name)

	return nil
}

// IsConnected returns true if the client is connected
func (c *HTTPClient) IsConnected() bool {
	return atomic.LoadInt32(&c.connected) == 1
}

// GetTransport returns the transport type for this client
func (c *HTTPClient) GetTransport() string {
	return "http"
}

// ListTools lists all available tools from the server
func (c *HTTPClient) ListTools(ctx context.Context) ([]Tool, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to server")
	}

	msg := Message{
		Method: "tools/list",
		Params: map[string]interface{}{},
	}

	response, err := c.sendRequest(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("send tools/list request: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", response.Error.Message)
	}

	// Parse the response
	var result struct {
		Tools []Tool `json:"tools"`
	}
	if data, err := json.Marshal(response.Result); err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	} else if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Set server name for each tool
	for i := range result.Tools {
		result.Tools[i].ServerName = c.server.Name
		result.Tools[i].LastUpdated = time.Now()
	}

	return result.Tools, nil
}

// CallTool executes a tool with the given parameters
func (c *HTTPClient) CallTool(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to server")
	}

	msg := Message{
		Method: "tools/call",
		Params: ToolCallParams{
			Name:      name,
			Arguments: params,
		},
	}

	response, err := c.sendRequest(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("send tools/call request: %w", err)
	}

	if response.Error != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: response.Error.Message,
			}},
			IsError: true,
		}, nil
	}

	// Parse the response
	var result ToolResult
	if data, err := json.Marshal(response.Result); err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	} else if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// GetInfo retrieves server information
func (c *HTTPClient) GetInfo(ctx context.Context) (*ServerInfo, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to server")
	}

	msg := Message{
		Method: "ping",
		Params: map[string]interface{}{},
	}

	response, err := c.sendRequest(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("send ping request: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("ping error: %s", response.Error.Message)
	}

	// For now, return basic info
	info := &ServerInfo{
		Name:     c.server.Name,
		Version:  "unknown",
		Protocol: "mcp/1.0",
	}
	info.Capabilities.Tools = true

	return info, nil
}

// initialize sends the initialize request
func (c *HTTPClient) initialize(ctx context.Context) error {
	msg := Message{
		Method: "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"roots": map[string]interface{}{
					"listChanged": true,
				},
			},
			"clientInfo": map[string]interface{}{
				"name":    "othello",
				"version": "1.0.0",
			},
		},
	}

	response, err := c.sendRequest(ctx, msg)
	if err != nil {
		return fmt.Errorf("send initialize request: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("initialize error: %s", response.Error.Message)
	}

	c.logger.Info("Initialized HTTP MCP server", "name", c.server.Name)
	return nil
}

// sendRequest sends an HTTP request and returns the response
func (c *HTTPClient) sendRequest(ctx context.Context, msg Message) (Message, error) {
	// Generate request ID
	requestID := c.nextRequestID()
	msg.ID = requestID

	// Marshal the message
	data, err := json.Marshal(msg)
	if err != nil {
		return Message{}, fmt.Errorf("marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.server.URL, bytes.NewReader(data))
	if err != nil {
		return Message{}, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	c.setHeaders(req)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for session ID in response
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		c.mu.Lock()
		c.sessionID = sessionID
		c.mu.Unlock()
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response Message
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Message{}, fmt.Errorf("decode response: %w", err)
	}

	return response, nil
}

// setHeaders sets the required HTTP headers for MCP
func (c *HTTPClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Mcp-Protocol-Version", "2024-11-05")

	// Set session ID if we have one
	c.mu.RLock()
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.mu.RUnlock()

	// Set custom headers from server config
	for key, value := range c.server.Headers {
		req.Header.Set(key, value)
	}
}

// nextRequestID generates the next request ID
func (c *HTTPClient) nextRequestID() int64 {
	return atomic.AddInt64(&c.requestID, 1)
}