package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ToolCache manages cached tool information with TTL
type ToolCache struct {
	tools map[string]Tool
	ttl   time.Duration
	mutex sync.RWMutex
}

// NewToolCache creates a new tool cache with the specified TTL
func NewToolCache(ttl time.Duration) *ToolCache {
	return &ToolCache{
		tools: make(map[string]Tool),
		ttl:   ttl,
	}
}

// Get retrieves a tool from the cache if it's still valid
func (c *ToolCache) Get(name string) (Tool, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	tool, exists := c.tools[name]
	if !exists {
		return Tool{}, false
	}
	
	// Check if cache entry is still valid
	if time.Since(tool.LastUpdated) > c.ttl {
		return Tool{}, false
	}
	
	return tool, true
}

// Set stores a tool in the cache
func (c *ToolCache) Set(tool Tool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	tool.LastUpdated = time.Now()
	c.tools[tool.Name] = tool
}

// Clear removes all tools from the cache
func (c *ToolCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.tools = make(map[string]Tool)
}

// ToolRegistry manages tool discovery and caching across multiple MCP servers
type ToolRegistry struct {
	tools   map[string]Tool
	servers map[string]Client
	cache   *ToolCache
	mutex   sync.RWMutex
	logger  Logger
}

// Logger interface for registry logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(logger Logger) *ToolRegistry {
	return &ToolRegistry{
		tools:   make(map[string]Tool),
		servers: make(map[string]Client),
		cache:   NewToolCache(time.Hour), // 1 hour cache TTL
		logger:  logger,
	}
}

// RegisterServer registers an MCP server with the registry
func (r *ToolRegistry) RegisterServer(name string, client Client) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.servers[name] = client
	r.logger.Info("Registered MCP server", "name", name)
	
	// Discover tools from the server
	return r.discoverToolsLocked(context.Background(), name, client)
}

// UnregisterServer removes an MCP server from the registry
func (r *ToolRegistry) UnregisterServer(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	delete(r.servers, name)
	
	// Remove tools from this server
	for toolName, tool := range r.tools {
		if tool.ServerName == name {
			delete(r.tools, toolName)
		}
	}
	
	r.logger.Info("Unregistered MCP server", "name", name)
}

// discoverToolsLocked discovers tools from a server (must be called with lock held)
func (r *ToolRegistry) discoverToolsLocked(ctx context.Context, serverName string, client Client) error {
	if !client.IsConnected() {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("connect to server %s: %w", serverName, err)
		}
	}
	
	tools, err := client.ListTools(ctx)
	if err != nil {
		r.logger.Error("Failed to list tools from server", "server", serverName, "error", err)
		return fmt.Errorf("list tools from %s: %w", serverName, err)
	}
	
	r.logger.Info("Discovered tools from server", "server", serverName, "count", len(tools))
	
	// Register tools in the registry
	for _, tool := range tools {
		tool.ServerName = serverName
		tool.LastUpdated = time.Now()
		r.tools[tool.Name] = tool
		r.cache.Set(tool)
		
		r.logger.Debug("Registered tool", "name", tool.Name, "server", serverName)
	}
	
	return nil
}

// RefreshTools refreshes tools from all registered servers
func (r *ToolRegistry) RefreshTools(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	var errors []error
	
	for serverName, client := range r.servers {
		if err := r.discoverToolsLocked(ctx, serverName, client); err != nil {
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to refresh tools from %d servers: %v", len(errors), errors)
	}
	
	return nil
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	// First try cache
	if tool, found := r.cache.Get(name); found {
		return tool, true
	}
	
	// Then try registry
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	tool, exists := r.tools[name]
	if exists {
		// Update cache
		r.cache.Set(tool)
	}
	
	return tool, exists
}

// ListTools returns all available tools
func (r *ToolRegistry) ListTools() []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	
	return tools
}

// ListToolsForServer returns tools available for a specific server
func (r *ToolRegistry) ListToolsForServer(serverName string) []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var tools []Tool
	for _, tool := range r.tools {
		if tool.ServerName == serverName {
			tools = append(tools, tool)
		}
	}
	
	return tools
}

// GetServer returns the client for a specific server
func (r *ToolRegistry) GetServer(name string) (Client, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	client, exists := r.servers[name]
	return client, exists
}

// ListServers returns all registered server names
func (r *ToolRegistry) ListServers() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	servers := make([]string, 0, len(r.servers))
	for name := range r.servers {
		servers = append(servers, name)
	}
	
	return servers
}

// IsServerConnected checks if a server is connected
func (r *ToolRegistry) IsServerConnected(name string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	client, exists := r.servers[name]
	if !exists {
		return false
	}
	
	return client.IsConnected()
}

// GetToolCount returns the total number of registered tools
func (r *ToolRegistry) GetToolCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	return len(r.tools)
}

// GetServerCount returns the total number of registered servers
func (r *ToolRegistry) GetServerCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	return len(r.servers)
}

// GetAllTools returns all tools from all servers
func (r *ToolRegistry) GetAllTools() []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// GetToolsByServer returns all tools from a specific server
func (r *ToolRegistry) GetToolsByServer(serverName string) []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var tools []Tool
	for _, tool := range r.tools {
		if tool.ServerName == serverName {
			tools = append(tools, tool)
		}
	}

	return tools
}