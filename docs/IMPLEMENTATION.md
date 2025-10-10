# Implementation Guide
## Othello AI Agent

**Version:** 1.0  
**Date:** October 10, 2025  
**Document Type:** Implementation Guide  

---

## Table of Contents

1. [Development Setup](#development-setup)
2. [Implementation Phases](#implementation-phases)
3. [Core Components](#core-components)
4. [Testing Strategy](#testing-strategy)
5. [Performance Optimization](#performance-optimization)
6. [Deployment Strategy](#deployment-strategy)
7. [Monitoring and Observability](#monitoring-and-observability)

---

## Development Setup

### Prerequisites

```bash
# Go 1.21 or later
go version

# Ollama for testing
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull qwen2.5:3b

# Development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/air-verse/air@latest  # Hot reload
```

### Project Initialization

```bash
# Initialize project
mkdir othello && cd othello
go mod init github.com/danieleugenewilliams/othello-agent

# Create directory structure
mkdir -p {cmd/othello,internal/{agent,mcp,model,tui,config,storage},pkg/types,docs,configs,scripts}

# Initialize Git
git init
echo "# Othello AI Agent" > README.md
```

### Development Dependencies

```go
// go.mod
module github.com/danieleugenewilliams/othello-agent

go 1.21

require (
    github.com/charmbracelet/bubbletea v0.24.2
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.17.0
    github.com/gorilla/rpc v1.2.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/mattn/go-sqlite3 v1.14.17
    github.com/stretchr/testify v1.8.4
)
```

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-4)

#### Week 1: Project Structure and Build System

**Goals:**
- Set up project structure
- Implement basic CLI with Cobra
- Create configuration system with Viper
- Set up testing and CI/CD

**Deliverables:**
```bash
othello --version          # Show version info
othello --help            # Show usage help
othello config show       # Display configuration
```

**Implementation Tasks:**

1. **CLI Framework** (`cmd/othello/main.go`)
```go
package main

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "github.com/danieleugenewilliams/othello-agent/internal/config"
    "github.com/danieleugenewilliams/othello-agent/internal/agent"
)

var rootCmd = &cobra.Command{
    Use:   "othello",
    Short: "Othello AI Agent - Local AI assistant with MCP tool integration",
    Long:  `A high-performance AI agent built in Go that uses local models and MCP tools`,
    RunE:  runTUI,
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func runTUI(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }
    
    agent, err := agent.New(cfg)
    if err != nil {
        return fmt.Errorf("create agent: %w", err)
    }
    
    return agent.StartTUI()
}
```

2. **Configuration System** (`internal/config/config.go`)
```go
package config

import (
    "github.com/spf13/viper"
    "time"
)

type Config struct {
    Model   ModelConfig   `mapstructure:"model"`
    Ollama  OllamaConfig  `mapstructure:"ollama"`
    TUI     TUIConfig     `mapstructure:"tui"`
    MCP     MCPConfig     `mapstructure:"mcp"`
    Storage StorageConfig `mapstructure:"storage"`
    Logging LoggingConfig `mapstructure:"logging"`
}

type ModelConfig struct {
    Type          string  `mapstructure:"type"`
    Name          string  `mapstructure:"name"`
    Temperature   float64 `mapstructure:"temperature"`
    MaxTokens     int     `mapstructure:"max_tokens"`
    ContextLength int     `mapstructure:"context_length"`
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("$HOME/.config/othello")
    viper.AddConfigPath("/etc/othello")
    
    // Set defaults
    setDefaults()
    
    // Read config file
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

#### Week 2: Basic MCP Client

**Goals:**
- Implement JSON-RPC 2.0 client
- Support STDIO transport
- Basic MCP lifecycle (initialize, ready)

**Implementation Tasks:**

1. **MCP Client** (`internal/mcp/client.go`)
```go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
)

type Client struct {
    config    ServerConfig
    transport Transport
    tools     []Tool
    connected bool
}

type Transport interface {
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error)
    Subscribe(ctx context.Context) (<-chan *JSONRPCNotification, error)
}

func NewClient(config ServerConfig) *Client {
    return &Client{
        config: config,
        transport: NewSTDIOTransport(config),
    }
}

func (c *Client) Connect(ctx context.Context) error {
    if err := c.transport.Connect(ctx); err != nil {
        return fmt.Errorf("transport connect: %w", err)
    }
    
    if err := c.initialize(ctx); err != nil {
        return fmt.Errorf("initialize: %w", err)
    }
    
    c.connected = true
    return nil
}

func (c *Client) initialize(ctx context.Context) error {
    req := &JSONRPCRequest{
        ID:     1,
        Method: "initialize",
        Params: InitializeParams{
            ProtocolVersion: "2025-06-18",
            Capabilities: ClientCapabilities{
                Elicitation: &ElicitationCapability{},
            },
            ClientInfo: ClientInfo{
                Name:    "othello",
                Version: "1.0.0",
            },
        },
    }
    
    resp, err := c.transport.Send(ctx, req)
    if err != nil {
        return fmt.Errorf("send initialize: %w", err)
    }
    
    var result InitializeResult
    if err := json.Unmarshal(resp.Result, &result); err != nil {
        return fmt.Errorf("unmarshal result: %w", err)
    }
    
    // Send initialized notification
    notification := &JSONRPCRequest{
        Method: "notifications/initialized",
    }
    
    _, err = c.transport.Send(ctx, notification)
    return err
}
```

#### Week 3: Ollama Integration

**Goals:**
- Implement Ollama HTTP client
- Model loading and inference
- Basic conversation handling

**Implementation Tasks:**

1. **Model Interface** (`internal/model/interface.go`)
```go
package model

import "context"

type Interface interface {
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
    GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateChunk, error)
    ListModels(ctx context.Context) ([]ModelInfo, error)
    LoadModel(ctx context.Context, modelName string) error
}

type GenerateRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Tools       []Tool    `json:"tools,omitempty"`
    Temperature float64   `json:"temperature,omitempty"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
    Stream      bool      `json:"stream,omitempty"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

2. **Ollama Client** (`internal/model/ollama/client.go`)
```go
package ollama

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/danieleugenewilliams/othello-agent/internal/model"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

func (c *Client) Generate(ctx context.Context, req *model.GenerateRequest) (*model.GenerateResponse, error) {
    ollamaReq := convertToOllamaRequest(req)
    
    body, err := json.Marshal(ollamaReq)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()
    
    var ollamaResp OllamaResponse
    if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    return convertToModelResponse(&ollamaResp), nil
}
```

#### Week 4: Basic TUI

**Goals:**
- Implement basic TUI with bubbletea
- Chat interface
- Basic input/output handling

**Implementation Tasks:**

1. **TUI Application** (`internal/tui/app.go`)
```go
package tui

import (
    "context"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/danieleugenewilliams/othello-agent/internal/agent"
)

type App struct {
    agent       agent.Interface
    messages    []Message
    input       string
    width       int
    height      int
    styles      Styles
}

type Message struct {
    Role    string
    Content string
    Time    string
}

func NewApp(agent agent.Interface) *App {
    return &App{
        agent:    agent,
        messages: []Message{},
        styles:   DefaultStyles(),
    }
}

func (a *App) Init() tea.Cmd {
    return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        a.width = msg.Width
        a.height = msg.Height
        return a, nil
    
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c":
            return a, tea.Quit
        case "enter":
            return a.handleSubmit()
        default:
            a.input += msg.String()
            return a, nil
        }
    }
    
    return a, nil
}

func (a *App) View() string {
    chat := a.renderChat()
    input := a.renderInput()
    status := a.renderStatus()
    
    return lipgloss.JoinVertical(lipgloss.Left,
        status,
        chat,
        input,
    )
}
```

### Phase 2: Core Features (Weeks 5-8)

#### Week 5: Tool Discovery and Registry

**Goals:**
- Implement tool discovery from MCP servers
- Build tool registry with caching
- Handle tool list updates

**Implementation Tasks:**

1. **Tool Registry** (`internal/mcp/registry.go`)
```go
package mcp

import (
    "context"
    "sync"
    "time"
)

type ToolRegistry struct {
    tools   map[string]Tool
    servers map[string]*Client
    cache   *ToolCache
    mutex   sync.RWMutex
}

type Tool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"inputSchema"`
    ServerName  string                 `json:"serverName"`
}

func NewToolRegistry() *ToolRegistry {
    return &ToolRegistry{
        tools:   make(map[string]Tool),
        servers: make(map[string]*Client),
        cache:   NewToolCache(time.Hour),
    }
}

func (r *ToolRegistry) RegisterServer(name string, client *Client) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    r.servers[name] = client
    return r.discoverTools(context.Background(), name, client)
}

func (r *ToolRegistry) discoverTools(ctx context.Context, serverName string, client *Client) error {
    tools, err := client.ListTools(ctx)
    if err != nil {
        return fmt.Errorf("list tools from %s: %w", serverName, err)
    }
    
    for _, tool := range tools {
        tool.ServerName = serverName
        r.tools[tool.Name] = tool
    }
    
    return nil
}

func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    
    tool, exists := r.tools[name]
    return tool, exists
}

func (r *ToolRegistry) ListTools() []Tool {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    
    tools := make([]Tool, 0, len(r.tools))
    for _, tool := range r.tools {
        tools = append(tools, tool)
    }
    
    return tools
}
```

#### Week 6: Tool Execution Engine

**Goals:**
- Implement tool execution with parameter validation
- Handle tool responses and errors
- Support concurrent tool execution

**Implementation Tasks:**

1. **Execution Engine** (`internal/agent/executor.go`)
```go
package agent

import (
    "context"
    "fmt"
    "sync"
    "github.com/danieleugenewilliams/othello-agent/internal/mcp"
)

type Executor struct {
    mcpManager  *mcp.Manager
    registry    *mcp.ToolRegistry
    validator   *ParameterValidator
    semaphore   chan struct{} // Limit concurrent executions
}

func NewExecutor(mcpManager *mcp.Manager, registry *mcp.ToolRegistry) *Executor {
    return &Executor{
        mcpManager: mcpManager,
        registry:   registry,
        validator:  NewParameterValidator(),
        semaphore:  make(chan struct{}, 10), // Max 10 concurrent
    }
}

func (e *Executor) ExecuteTools(ctx context.Context, toolCalls []ToolCall) ([]ToolResult, error) {
    results := make([]ToolResult, len(toolCalls))
    var wg sync.WaitGroup
    var mutex sync.Mutex
    
    for i, call := range toolCalls {
        wg.Add(1)
        go func(index int, toolCall ToolCall) {
            defer wg.Done()
            
            // Acquire semaphore
            e.semaphore <- struct{}{}
            defer func() { <-e.semaphore }()
            
            result := e.executeTool(ctx, toolCall)
            
            mutex.Lock()
            results[index] = result
            mutex.Unlock()
        }(i, call)
    }
    
    wg.Wait()
    return results, nil
}

func (e *Executor) executeTool(ctx context.Context, call ToolCall) ToolResult {
    // Validate tool exists
    tool, exists := e.registry.GetTool(call.Name)
    if !exists {
        return ToolResult{
            IsError: true,
            Content: []Content{{
                Type: "text",
                Text: fmt.Sprintf("Tool %s not found", call.Name),
            }},
        }
    }
    
    // Validate parameters
    if err := e.validator.Validate(tool.InputSchema, call.Parameters); err != nil {
        return ToolResult{
            IsError: true,
            Content: []Content{{
                Type: "text", 
                Text: fmt.Sprintf("Parameter validation failed: %v", err),
            }},
        }
    }
    
    // Execute tool
    result, err := e.mcpManager.ExecuteTool(ctx, call.Name, call.Parameters)
    if err != nil {
        return ToolResult{
            IsError: true,
            Content: []Content{{
                Type: "text",
                Text: fmt.Sprintf("Tool execution failed: %v", err),
            }},
        }
    }
    
    return *result
}
```

#### Week 7: Conversation Management

**Goals:**
- Implement conversation state management
- History persistence with SQLite
- Context window management

**Implementation Tasks:**

1. **Conversation Manager** (`internal/agent/conversation.go`)
```go
package agent

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"
    _ "github.com/mattn/go-sqlite3"
)

type ConversationManager struct {
    db           *sql.DB
    maxHistory   int
    contextLimit int
}

type Conversation struct {
    ID       string    `json:"id"`
    Messages []Message `json:"messages"`
    Created  time.Time `json:"created"`
    Updated  time.Time `json:"updated"`
}

func NewConversationManager(dbPath string, maxHistory, contextLimit int) (*ConversationManager, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }
    
    cm := &ConversationManager{
        db:           db,
        maxHistory:   maxHistory,
        contextLimit: contextLimit,
    }
    
    if err := cm.initDB(); err != nil {
        return nil, fmt.Errorf("initialize database: %w", err)
    }
    
    return cm, nil
}

func (cm *ConversationManager) initDB() error {
    query := `
    CREATE TABLE IF NOT EXISTS conversations (
        id TEXT PRIMARY KEY,
        messages TEXT NOT NULL,
        created DATETIME NOT NULL,
        updated DATETIME NOT NULL
    );
    
    CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated);
    `
    
    _, err := cm.db.Exec(query)
    return err
}

func (cm *ConversationManager) AddMessage(ctx context.Context, conversationID string, message Message) error {
    conversation, err := cm.GetConversation(ctx, conversationID)
    if err != nil {
        // Create new conversation
        conversation = &Conversation{
            ID:      conversationID,
            Created: time.Now(),
        }
    }
    
    conversation.Messages = append(conversation.Messages, message)
    conversation.Updated = time.Now()
    
    // Trim to context limit
    if len(conversation.Messages) > cm.contextLimit {
        conversation.Messages = conversation.Messages[len(conversation.Messages)-cm.contextLimit:]
    }
    
    return cm.SaveConversation(ctx, conversation)
}
```

#### Week 8: Enhanced TUI

**Goals:**
- Multi-view TUI with server management
- Real-time status updates
- Keyboard shortcuts and navigation

**Implementation Tasks:**

1. **Enhanced TUI** (`internal/tui/views.go`)
```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type ViewType int

const (
    ChatView ViewType = iota
    ServerView
    HelpView
)

type Views struct {
    current ViewType
    chat    *ChatModel
    server  *ServerModel
    help    *HelpModel
}

func NewViews(agent agent.Interface) *Views {
    return &Views{
        current: ChatView,
        chat:    NewChatModel(agent),
        server:  NewServerModel(agent),
        help:    NewHelpModel(),
    }
}

func (v *Views) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            v.switchView()
            return v, nil
        case "ctrl+s":
            v.current = ServerView
            return v, nil
        case "ctrl+h":
            v.current = HelpView
            return v, nil
        case "esc":
            v.current = ChatView
            return v, nil
        }
    }
    
    // Delegate to current view
    switch v.current {
    case ChatView:
        model, cmd := v.chat.Update(msg)
        v.chat = model.(*ChatModel)
        return v, cmd
    case ServerView:
        model, cmd := v.server.Update(msg)
        v.server = model.(*ServerModel)
        return v, cmd
    case HelpView:
        model, cmd := v.help.Update(msg)
        v.help = model.(*HelpModel)
        return v, cmd
    }
    
    return v, nil
}
```

### Phase 3: Advanced Features (Weeks 9-12)

#### Week 9: HTTP Transport and Remote Servers

**Goals:**
- Implement HTTP transport for MCP
- Support remote MCP servers
- Authentication and security

#### Week 10: Real-time Notifications

**Goals:**
- Implement MCP notification handling
- Real-time UI updates
- Server status monitoring

#### Week 11: Multiple Model Backends

**Goals:**
- GGUF direct loading support
- Generic HTTP API client
- Model switching and configuration

#### Week 12: Advanced Storage and Caching

**Goals:**
- Conversation search and filtering
- Tool result caching
- Configuration migration

---

## Core Components

### Error Handling Strategy

```go
// Custom error types
type MCPError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func (e MCPError) Error() string {
    return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// Error wrapping
func WrapError(err error, operation string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s: %w", operation, err)
}

// Result pattern
type Result[T any] struct {
    Value T
    Error error
}

func (r Result[T]) Unwrap() (T, error) {
    return r.Value, r.Error
}
```

### Concurrency Patterns

```go
// Worker pool for tool execution
type WorkerPool struct {
    workers    int
    jobQueue   chan Job
    resultQueue chan Result
    quit       chan bool
}

func NewWorkerPool(workers int) *WorkerPool {
    return &WorkerPool{
        workers:     workers,
        jobQueue:    make(chan Job, workers*2),
        resultQueue: make(chan Result, workers*2),
        quit:        make(chan bool),
    }
}

func (wp *WorkerPool) Start() {
    for i := 0; i < wp.workers; i++ {
        go wp.worker()
    }
}

func (wp *WorkerPool) worker() {
    for {
        select {
        case job := <-wp.jobQueue:
            result := job.Execute()
            wp.resultQueue <- result
        case <-wp.quit:
            return
        }
    }
}
```

### Configuration Validation

```go
// Schema validation
type ConfigValidator struct {
    schemas map[string]*jsonschema.Schema
}

func (v *ConfigValidator) Validate(config *Config) error {
    // Validate model configuration
    if config.Model.Temperature < 0.0 || config.Model.Temperature > 1.0 {
        return fmt.Errorf("temperature must be between 0.0 and 1.0")
    }
    
    if config.Model.MaxTokens <= 0 {
        return fmt.Errorf("max_tokens must be positive")
    }
    
    // Validate MCP configuration
    if config.MCP.MaxServers <= 0 {
        return fmt.Errorf("max_servers must be positive")
    }
    
    return nil
}
```

---

## Testing Strategy

### Unit Testing

```go
// Test structure
func TestMCPClient_Connect(t *testing.T) {
    tests := []struct {
        name    string
        config  ServerConfig
        mock    func(*MockTransport)
        wantErr bool
    }{
        {
            name: "successful connection",
            config: ServerConfig{
                Name:    "test-server",
                Command: "test-command",
            },
            mock: func(mt *MockTransport) {
                mt.On("Connect", mock.Anything).Return(nil)
                mt.On("Send", mock.Anything, mock.MatchedBy(func(req *JSONRPCRequest) bool {
                    return req.Method == "initialize"
                })).Return(&JSONRPCResponse{
                    ID: 1,
                    Result: json.RawMessage(`{
                        "protocolVersion": "2025-06-18",
                        "capabilities": {}
                    }`),
                }, nil)
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockTransport := &MockTransport{}
            if tt.mock != nil {
                tt.mock(mockTransport)
            }
            
            client := &Client{
                config:    tt.config,
                transport: mockTransport,
            }
            
            err := client.Connect(context.Background())
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            mockTransport.AssertExpectations(t)
        })
    }
}
```

### Integration Testing

```go
// Integration test setup
func TestAgentIntegration(t *testing.T) {
    // Start test MCP server
    server := startTestMCPServer(t)
    defer server.Stop()
    
    // Create test configuration
    config := &Config{
        Model: ModelConfig{
            Type: "mock",
            Name: "test-model",
        },
        MCP: MCPConfig{
            Timeout: 5 * time.Second,
        },
    }
    
    // Create agent with mock model
    agent, err := NewAgent(config)
    require.NoError(t, err)
    
    // Add test server
    err = agent.AddMCPServer(ServerConfig{
        Name:    "test-server",
        Command: server.Command(),
    })
    require.NoError(t, err)
    
    // Test tool execution
    response, err := agent.ProcessQuery(context.Background(), "test query")
    require.NoError(t, err)
    assert.NotEmpty(t, response.Content)
}
```

### Performance Testing

```go
// Benchmark tool execution
func BenchmarkToolExecution(b *testing.B) {
    agent := setupTestAgent(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := agent.ProcessQuery(context.Background(), "list files")
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Load testing
func TestConcurrentToolExecution(t *testing.T) {
    agent := setupTestAgent(t)
    
    const concurrency = 10
    const requests = 100
    
    var wg sync.WaitGroup
    errChan := make(chan error, concurrency)
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < requests/concurrency; j++ {
                _, err := agent.ProcessQuery(context.Background(), "test query")
                if err != nil {
                    errChan <- err
                    return
                }
            }
        }()
    }
    
    wg.Wait()
    close(errChan)
    
    for err := range errChan {
        t.Error(err)
    }
}
```

---

## Performance Optimization

### Memory Management

```go
// Object pooling for frequent allocations
var messagePool = sync.Pool{
    New: func() interface{} {
        return &Message{}
    },
}

func getMessage() *Message {
    return messagePool.Get().(*Message)
}

func putMessage(msg *Message) {
    msg.Reset()
    messagePool.Put(msg)
}

// Memory monitoring
type MemoryMonitor struct {
    maxMemory int64
    interval  time.Duration
}

func (m *MemoryMonitor) Start() {
    ticker := time.NewTicker(m.interval)
    go func() {
        for range ticker.C {
            var stats runtime.MemStats
            runtime.ReadMemStats(&stats)
            
            if int64(stats.Alloc) > m.maxMemory {
                log.Warn("Memory usage exceeds threshold",
                    "current", stats.Alloc,
                    "threshold", m.maxMemory)
                runtime.GC()
            }
        }
    }()
}
```

### Connection Pooling

```go
// Connection pool for MCP clients
type ConnectionPool struct {
    clients   map[string][]*Client
    maxPerServer int
    mutex     sync.RWMutex
}

func (cp *ConnectionPool) Get(serverName string) (*Client, error) {
    cp.mutex.Lock()
    defer cp.mutex.Unlock()
    
    clients := cp.clients[serverName]
    if len(clients) > 0 {
        client := clients[len(clients)-1]
        cp.clients[serverName] = clients[:len(clients)-1]
        return client, nil
    }
    
    // Create new client if pool is empty
    return cp.createClient(serverName)
}

func (cp *ConnectionPool) Put(serverName string, client *Client) {
    cp.mutex.Lock()
    defer cp.mutex.Unlock()
    
    clients := cp.clients[serverName]
    if len(clients) < cp.maxPerServer {
        cp.clients[serverName] = append(clients, client)
    } else {
        client.Disconnect(context.Background())
    }
}
```

### Caching Strategies

```go
// LRU cache for tool results
type ToolCache struct {
    cache    *lru.Cache
    ttl      time.Duration
    mutex    sync.RWMutex
}

type CacheEntry struct {
    Value     interface{}
    ExpiresAt time.Time
}

func (tc *ToolCache) Get(key string) (interface{}, bool) {
    tc.mutex.RLock()
    defer tc.mutex.RUnlock()
    
    if value, ok := tc.cache.Get(key); ok {
        entry := value.(*CacheEntry)
        if time.Now().Before(entry.ExpiresAt) {
            return entry.Value, true
        }
        tc.cache.Remove(key)
    }
    
    return nil, false
}

func (tc *ToolCache) Set(key string, value interface{}) {
    tc.mutex.Lock()
    defer tc.mutex.Unlock()
    
    entry := &CacheEntry{
        Value:     value,
        ExpiresAt: time.Now().Add(tc.ttl),
    }
    
    tc.cache.Add(key, entry)
}
```

---

## Deployment Strategy

### Build System

```makefile
# Makefile
.PHONY: build test clean install

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/othello cmd/othello/main.go

build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/othello-linux-amd64 cmd/othello/main.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/othello-darwin-amd64 cmd/othello/main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/othello-darwin-arm64 cmd/othello/main.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/othello-windows-amd64.exe cmd/othello/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	go test -v -tags=integration ./test/integration/...

install:
	go install $(LDFLAGS) cmd/othello/main.go

clean:
	rm -rf bin/ dist/ coverage.out
```

### Docker Support

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o othello cmd/othello/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

COPY --from=builder /app/othello .
COPY configs/docker.yaml config.yaml

CMD ["./othello"]
```

### Release Automation

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run tests
      run: make test
    
    - name: Build binaries
      run: make build-all
    
    - name: Create release
      uses: goreleaser/goreleaser-action@v4
      with:
        version: latest
        args: release --clean
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Monitoring and Observability

### Logging

```go
// Structured logging setup
func setupLogging(level string) *slog.Logger {
    var logLevel slog.Level
    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "info":
        logLevel = slog.LevelInfo
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }
    
    opts := &slog.HandlerOptions{
        Level: logLevel,
    }
    
    handler := slog.NewJSONHandler(os.Stdout, opts)
    return slog.New(handler)
}

// Contextual logging
func (a *Agent) logWithContext(ctx context.Context, level slog.Level, msg string, args ...any) {
    logger := a.logger.With(
        "trace_id", getTraceID(ctx),
        "session_id", getSessionID(ctx),
    )
    logger.Log(ctx, level, msg, args...)
}
```

### Metrics Collection

```go
// Prometheus metrics
var (
    toolExecutions = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "othello_tool_executions_total",
            Help: "Total number of tool executions",
        },
        []string{"tool_name", "server_name", "status"},
    )
    
    responseTime = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "othello_response_time_seconds",
            Help: "Response time for queries",
            Buckets: prometheus.DefBuckets,
        },
        []string{"query_type"},
    )
    
    activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "othello_active_mcp_connections",
            Help: "Number of active MCP connections",
        },
    )
)

func init() {
    prometheus.MustRegister(toolExecutions)
    prometheus.MustRegister(responseTime)
    prometheus.MustRegister(activeConnections)
}
```

### Health Checks

```go
// Health check system
type HealthChecker struct {
    checks map[string]HealthCheck
    mutex  sync.RWMutex
}

type HealthCheck interface {
    Check(ctx context.Context) error
    Name() string
}

type MCPHealthCheck struct {
    manager *mcp.Manager
}

func (h *MCPHealthCheck) Check(ctx context.Context) error {
    servers := h.manager.ListServers()
    for _, server := range servers {
        if !server.Connected {
            return fmt.Errorf("server %s is disconnected", server.Name)
        }
    }
    return nil
}

func (h *MCPHealthCheck) Name() string {
    return "mcp_connections"
}

func (hc *HealthChecker) RegisterCheck(check HealthCheck) {
    hc.mutex.Lock()
    defer hc.mutex.Unlock()
    hc.checks[check.Name()] = check
}

func (hc *HealthChecker) CheckAll(ctx context.Context) map[string]error {
    hc.mutex.RLock()
    defer hc.mutex.RUnlock()
    
    results := make(map[string]error)
    for name, check := range hc.checks {
        results[name] = check.Check(ctx)
    }
    
    return results
}
```

This implementation guide provides a comprehensive roadmap for building the Othello AI agent with clear phases, detailed code examples, and best practices for Go development.