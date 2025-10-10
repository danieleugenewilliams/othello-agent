package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// STDIOClient implements the Client interface for STDIO-based MCP servers
type STDIOClient struct {
	server     Server
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	connected  int32 // atomic boolean
	responses  map[int64]chan Message
	responsesMu sync.RWMutex
	requestID  int64
	logger     Logger
}

// NewSTDIOClient creates a new STDIO client for an MCP server
func NewSTDIOClient(server Server, logger Logger) *STDIOClient {
	return &STDIOClient{
		server:    server,
		responses: make(map[int64]chan Message),
		logger:    logger,
	}
}

// Connect establishes a connection to the MCP server
func (c *STDIOClient) Connect(ctx context.Context) error {
	if atomic.LoadInt32(&c.connected) == 1 {
		return nil // Already connected
	}
	
	// Prepare command
	if len(c.server.Command) == 0 {
		return fmt.Errorf("no command specified for server %s", c.server.Name)
	}
	
	args := append(c.server.Command[1:], c.server.Args...)
	c.cmd = exec.CommandContext(ctx, c.server.Command[0], args...)
	
	// Set environment variables
	c.cmd.Env = os.Environ()
	for key, value := range c.server.Env {
		c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Set up pipes
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}
	
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	
	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("create stderr pipe: %w", err)
	}
	
	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start MCP server process: %w", err)
	}
	
	// Start reading responses
	go c.readResponses()
	go c.readErrors()
	
	atomic.StoreInt32(&c.connected, 1)
	c.logger.Info("Connected to MCP server", "name", c.server.Name, "pid", c.cmd.Process.Pid)
	
	// Send initialize request
	return c.initialize(ctx)
}

// Disconnect closes the connection to the MCP server
func (c *STDIOClient) Disconnect(ctx context.Context) error {
	if atomic.LoadInt32(&c.connected) == 0 {
		return nil // Already disconnected
	}
	
	atomic.StoreInt32(&c.connected, 0)
	
	// Close pipes
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}
	
	// Terminate process
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			c.logger.Error("Failed to kill MCP server process", "error", err)
		}
		c.cmd.Wait() // Wait for process to exit
	}
	
	c.logger.Info("Disconnected from MCP server", "name", c.server.Name)
	return nil
}

// IsConnected returns true if the client is connected
func (c *STDIOClient) IsConnected() bool {
	return atomic.LoadInt32(&c.connected) == 1
}

// GetTransport returns the transport type for this client
func (c *STDIOClient) GetTransport() string {
	return "stdio"
}

// ListTools lists all available tools from the server
func (c *STDIOClient) ListTools(ctx context.Context) ([]Tool, error) {
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
	var toolsResponse ToolListResponse
	if data, err := json.Marshal(response.Result); err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	} else if err := json.Unmarshal(data, &toolsResponse); err != nil {
		return nil, fmt.Errorf("unmarshal tools response: %w", err)
	}
	
	return toolsResponse.Tools, nil
}

// CallTool executes a tool with the given parameters
func (c *STDIOClient) CallTool(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, error) {
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
		return nil, fmt.Errorf("unmarshal tool result: %w", err)
	}
	
	return &result, nil
}

// GetInfo retrieves server information
func (c *STDIOClient) GetInfo(ctx context.Context) (*ServerInfo, error) {
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
func (c *STDIOClient) initialize(ctx context.Context) error {
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
	
	c.logger.Info("Initialized MCP server", "name", c.server.Name)
	return nil
}

// sendRequest sends a request and waits for a response
func (c *STDIOClient) sendRequest(ctx context.Context, msg Message) (Message, error) {
	// Ensure ID is int64
	requestID := c.nextRequestID()
	msg.ID = requestID
	
	// Create response channel
	responseChan := make(chan Message, 1)
	
	c.responsesMu.Lock()
	c.responses[requestID] = responseChan
	c.responsesMu.Unlock()
	
	// Clean up channel on exit
	defer func() {
		c.responsesMu.Lock()
		delete(c.responses, requestID)
		c.responsesMu.Unlock()
		close(responseChan)
	}()
	
	// Send the message
	data, err := json.Marshal(msg)
	if err != nil {
		return Message{}, fmt.Errorf("marshal message: %w", err)
	}
	
	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return Message{}, fmt.Errorf("write message: %w", err)
	}
	
	// Wait for response
	timeout := c.server.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	case <-time.After(timeout):
		return Message{}, fmt.Errorf("request timeout after %v", timeout)
	}
}

// readResponses reads responses from the server
func (c *STDIOClient) readResponses() {
	scanner := bufio.NewScanner(c.stdout)
	
	// Increase buffer size for large responses
	buf := make([]byte, 64*1024) // 64KB buffer
	scanner.Buffer(buf, 1024*1024) // 1MB max token size
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			c.logger.Error("Failed to unmarshal response", "error", err, "line", line)
			continue
		}
		
		// Handle response
		if msg.ID != nil {
			// Convert ID to int64 for consistent comparison
			var responseID int64
			switch id := msg.ID.(type) {
			case int64:
				responseID = id
			case float64:
				responseID = int64(id)
			case int:
				responseID = int64(id)
			default:
				c.logger.Error("Unexpected ID type", "type", fmt.Sprintf("%T", id), "value", id)
				continue
			}
			
			c.responsesMu.RLock()
			if ch, exists := c.responses[responseID]; exists {
				select {
				case ch <- msg:
				default:
					c.logger.Error("Response channel full", "id", responseID)
				}
			} else {
				c.logger.Debug("No waiting request for response", "id", responseID)
			}
			c.responsesMu.RUnlock()
		} else {
			// Handle notification
			c.logger.Debug("Received notification", "method", msg.Method)
		}
	}
	
	if err := scanner.Err(); err != nil {
		c.logger.Error("Error reading from server", "error", err)
	}
}

// readErrors reads stderr from the server
func (c *STDIOClient) readErrors() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			c.logger.Error("Server stderr", "message", line)
		}
	}
}

// nextRequestID generates the next request ID
func (c *STDIOClient) nextRequestID() int64 {
	return atomic.AddInt64(&c.requestID, 1)
}