# MCP-TUI Integration Plan
## Othello AI Agent

**Version:** 1.0  
**Date:** October 10, 2025  
**Purpose:** Enable seamless MCP tool integration with local LLMs through TUI

---

## Overview

This document outlines the Test-Driven Development (TDD) approach for integrating MCP tool capabilities into the Othello TUI. The core value proposition of Othello is enabling MCP tool integration for any open-source LLM with an excellent user experience.

## Current State Analysis

### Completed Components

1. **MCP Foundation** (`internal/mcp/`)
   - ✅ STDIO client implementation with JSON-RPC 2.0
   - ✅ HTTP client implementation for remote servers
   - ✅ Tool registry with caching and TTL
   - ✅ Tool executor with parameter validation
   - ✅ Notification handler for real-time events
   - ✅ Client factory for multiple transport types
   - ✅ 213 passing tests with good coverage

2. **TUI Foundation** (`internal/tui/`)
   - ✅ Basic application structure with bubbletea
   - ✅ Chat view with message display
   - ✅ Server view with list display
   - ✅ Help view and history view
   - ✅ Keyboard navigation and styling
   - ⚠️ **NOT CONNECTED TO MCP** - Using mock data

3. **Agent Foundation** (`internal/agent/`)
   - ✅ Basic agent structure
   - ✅ Ollama model integration
   - ⚠️ **NO MCP INTEGRATION** - Agent doesn't use registry/executor

4. **Model Layer** (`internal/model/`)
   - ✅ Ollama HTTP client
   - ✅ Model manager for multiple backends
   - ⚠️ **NO TOOL CALLING** - Doesn't send/receive tool information

### Critical Gaps

1. **Agent ↔ MCP**: Agent doesn't initialize or use MCP components
2. **TUI ↔ MCP**: TUI displays mock data, not real server/tool info
3. **Chat ↔ Tools**: No tool execution triggered from chat messages
4. **Model ↔ Tools**: Model doesn't know about available tools
5. **Notifications**: No real-time updates from MCP servers to TUI

---

## Implementation Strategy

### Phase 1: Agent-MCP Integration (Week 1)

**Goal**: Wire the Agent to use MCP Registry and Executor

#### 1.1 Add MCP Manager to Agent

**Test First** - `internal/agent/agent_test.go`:
```go
func TestAgent_InitializeMCP(t *testing.T) {
    tests := []struct {
        name       string
        config     *config.Config
        wantErr    bool
        wantServers int
    }{
        {
            name: "initialize with configured servers",
            config: &config.Config{
                MCP: config.MCPConfig{
                    Servers: []config.ServerConfig{
                        {
                            Name:    "filesystem",
                            Command: "npx",
                            Args:    []string{"@modelcontextprotocol/server-filesystem", "/tmp"},
                            Transport: "stdio",
                        },
                    },
                    Timeout: 5 * time.Second,
                },
            },
            wantErr:    false,
            wantServers: 1,
        },
        {
            name: "handle server connection failure gracefully",
            config: &config.Config{
                MCP: config.MCPConfig{
                    Servers: []config.ServerConfig{
                        {
                            Name:    "invalid",
                            Command: "nonexistent-command",
                            Transport: "stdio",
                        },
                    },
                },
            },
            wantErr:    false, // Should not fail entirely
            wantServers: 0,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agent, err := New(tt.config)
            require.NoError(t, err)
            
            ctx := context.Background()
            err = agent.Start(ctx)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.wantServers, len(agent.mcpManager.ListServers()))
            }
        })
    }
}

func TestAgent_GetAvailableTools(t *testing.T) {
    agent := setupTestAgent(t)
    
    tools := agent.GetAvailableTools()
    assert.NotEmpty(t, tools)
    
    // Should have tools from filesystem server
    found := false
    for _, tool := range tools {
        if tool.Name == "read_file" {
            found = true
            break
        }
    }
    assert.True(t, found, "Expected to find read_file tool")
}
```

**Implementation** - `internal/agent/agent.go`:
```go
type Agent struct {
    config      *config.Config
    logger      *log.Logger
    mcpManager  *mcp.Manager
    mcpRegistry *mcp.ToolRegistry
    mcpExecutor *mcp.ToolExecutor
    model       model.Model
}

type Manager interface {
    AddServer(ctx context.Context, cfg config.ServerConfig) error
    RemoveServer(ctx context.Context, name string) error
    ListServers() []ServerInfo
    GetServer(name string) (mcp.Client, bool)
}

func New(cfg *config.Config) (*Agent, error) {
    // ... existing code ...
    
    // Initialize MCP components
    logger := NewStructuredLogger()
    registry := mcp.NewToolRegistry(logger)
    executor := mcp.NewToolExecutor(registry, logger)
    manager := NewMCPManager(registry, logger)
    
    agent := &Agent{
        config:      cfg,
        logger:      log.New(log.Writer(), "[AGENT] ", log.LstdFlags),
        mcpRegistry: registry,
        mcpExecutor: executor,
        mcpManager:  manager,
    }
    
    return agent, nil
}

func (a *Agent) Start(ctx context.Context) error {
    // Connect to configured MCP servers
    for _, serverCfg := range a.config.MCP.Servers {
        if err := a.mcpManager.AddServer(ctx, serverCfg); err != nil {
            a.logger.Printf("Warning: Failed to connect to server %s: %v", 
                serverCfg.Name, err)
            // Continue with other servers
        }
    }
    
    // Refresh tool registry
    if err := a.mcpRegistry.RefreshTools(ctx); err != nil {
        a.logger.Printf("Warning: Failed to refresh tools: %v", err)
    }
    
    return nil
}

func (a *Agent) GetAvailableTools() []mcp.Tool {
    return a.mcpRegistry.GetAllTools()
}

func (a *Agent) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (*mcp.ExecuteResult, error) {
    return a.mcpExecutor.Execute(ctx, name, params)
}
```

#### 1.2 Create MCP Manager

**Test First** - `internal/agent/mcp_manager_test.go`:
```go
func TestMCPManager_AddServer(t *testing.T) {
    manager := setupTestManager(t)
    
    cfg := config.ServerConfig{
        Name:      "test-server",
        Command:   "npx",
        Args:      []string{"@modelcontextprotocol/server-filesystem", "/tmp"},
        Transport: "stdio",
    }
    
    err := manager.AddServer(context.Background(), cfg)
    assert.NoError(t, err)
    
    servers := manager.ListServers()
    assert.Len(t, servers, 1)
    assert.Equal(t, "test-server", servers[0].Name)
}

func TestMCPManager_ServerLifecycle(t *testing.T) {
    manager := setupTestManager(t)
    
    // Add server
    cfg := config.ServerConfig{Name: "test", Command: "echo", Transport: "stdio"}
    require.NoError(t, manager.AddServer(context.Background(), cfg))
    
    // Check connected
    servers := manager.ListServers()
    require.Len(t, servers, 1)
    assert.True(t, servers[0].Connected)
    
    // Remove server
    require.NoError(t, manager.RemoveServer(context.Background(), "test"))
    
    // Check removed
    servers = manager.ListServers()
    assert.Len(t, servers, 0)
}
```

**Implementation** - `internal/agent/mcp_manager.go`:
```go
type MCPManager struct {
    registry  *mcp.ToolRegistry
    clients   map[string]mcp.Client
    factory   *mcp.ClientFactory
    logger    Logger
    mutex     sync.RWMutex
}

type ServerInfo struct {
    Name      string
    Status    string
    Connected bool
    ToolCount int
    Transport string
}

func NewMCPManager(registry *mcp.ToolRegistry, logger Logger) *MCPManager {
    return &MCPManager{
        registry: registry,
        clients:  make(map[string]mcp.Client),
        factory:  mcp.NewClientFactory(logger),
        logger:   logger,
    }
}

func (m *MCPManager) AddServer(ctx context.Context, cfg config.ServerConfig) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    // Create client using factory
    client, err := m.factory.CreateClient(cfg)
    if err != nil {
        return fmt.Errorf("create client: %w", err)
    }
    
    // Connect to server
    if err := client.Connect(ctx); err != nil {
        return fmt.Errorf("connect to server: %w", err)
    }
    
    // Register with registry
    if err := m.registry.RegisterServer(cfg.Name, client); err != nil {
        client.Disconnect(ctx)
        return fmt.Errorf("register server: %w", err)
    }
    
    m.clients[cfg.Name] = client
    m.logger.Info("Added MCP server", "name", cfg.Name)
    
    return nil
}
```

---

### Phase 2: TUI-MCP Integration (Week 2)

**Goal**: Connect TUI views to real MCP data

#### 2.1 Define TUI Message Types for MCP

**Implementation** - `internal/tui/messages.go`:
```go
// MCP-related messages
type MCPServerConnectedMsg struct {
    Name      string
    ToolCount int
}

type MCPServerDisconnectedMsg struct {
    Name  string
    Error error
}

type MCPToolsRefreshedMsg struct {
    ServerName string
    Tools      []mcp.Tool
}

type MCPToolExecutingMsg struct {
    ToolName string
    Params   map[string]interface{}
}

type MCPToolExecutedMsg struct {
    ToolName string
    Result   *mcp.ExecuteResult
    Error    error
}

type MCPServerStatusMsg struct {
    Servers []ServerStatus
}

type ServerStatus struct {
    Name      string
    Connected bool
    ToolCount int
    LastError string
}

// Helper to create async tool execution command
func ExecuteToolCmd(agent *agent.Agent, toolName string, params map[string]interface{}) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        result, err := agent.ExecuteTool(ctx, toolName, params)
        
        return MCPToolExecutedMsg{
            ToolName: toolName,
            Result:   result,
            Error:    err,
        }
    }
}

// Helper to refresh server status
func RefreshServersCmd(agent *agent.Agent) tea.Cmd {
    return func() tea.Msg {
        servers := agent.GetServerStatus()
        return MCPServerStatusMsg{Servers: servers}
    }
}
```

#### 2.2 Wire ServerView to Real MCP Data

**Test First** - `internal/tui/server_view_test.go`:
```go
func TestServerView_DisplaysRealServers(t *testing.T) {
    agent := setupTestAgent(t)
    styles := DefaultStyles()
    keymap := DefaultKeyMap()
    
    view := NewServerView(styles, keymap, agent)
    
    // Should display actual servers from agent
    servers := view.GetDisplayedServers()
    assert.NotEmpty(t, servers)
    
    // Should match agent's server list
    agentServers := agent.GetServerStatus()
    assert.Len(t, servers, len(agentServers))
}

func TestServerView_HandlesMCPMessages(t *testing.T) {
    view := setupTestServerView(t)
    
    // Simulate server connected
    msg := MCPServerConnectedMsg{
        Name:      "new-server",
        ToolCount: 5,
    }
    
    _, cmd := view.Update(msg)
    assert.NotNil(t, cmd)
    
    // Should have new server in list
    servers := view.GetDisplayedServers()
    found := false
    for _, s := range servers {
        if s.Name == "new-server" {
            found = true
            assert.Equal(t, 5, s.ToolCount)
            break
        }
    }
    assert.True(t, found)
}
```

**Implementation** - `internal/tui/server_view.go`:
```go
type ServerView struct {
    width   int
    height  int
    styles  Styles
    keymap  KeyMap
    list    list.Model
    agent   *agent.Agent
    servers []ServerStatus
}

func NewServerView(styles Styles, keymap KeyMap, agent *agent.Agent) *ServerView {
    l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
    l.Title = "MCP Servers"
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(true)
    l.Styles.Title = styles.ViewHeader
    
    view := &ServerView{
        styles:  styles,
        keymap:  keymap,
        list:    l,
        agent:   agent,
        servers: []ServerStatus{},
    }
    
    return view
}

func (v *ServerView) Init() tea.Cmd {
    // Fetch initial server status
    return RefreshServersCmd(v.agent)
}

func (v *ServerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd
    
    switch msg := msg.(type) {
    case MCPServerStatusMsg:
        // Update server list with real data
        v.servers = msg.Servers
        v.updateListItems()
        return v, nil
        
    case MCPServerConnectedMsg:
        // Add new server to list
        v.servers = append(v.servers, ServerStatus{
            Name:      msg.Name,
            Connected: true,
            ToolCount: msg.ToolCount,
        })
        v.updateListItems()
        return v, nil
        
    case MCPServerDisconnectedMsg:
        // Update server status
        for i, s := range v.servers {
            if s.Name == msg.Name {
                v.servers[i].Connected = false
                if msg.Error != nil {
                    v.servers[i].LastError = msg.Error.Error()
                }
                break
            }
        }
        v.updateListItems()
        return v, nil
        
    case tea.KeyMsg:
        switch msg.String() {
        case "r":
            // Refresh servers from agent
            return v, RefreshServersCmd(v.agent)
        case "enter":
            // Show server details or toggle connection
            if selected := v.list.SelectedItem(); selected != nil {
                if item, ok := selected.(ServerListItem); ok {
                    return v, v.handleServerAction(item.Name)
                }
            }
        }
    }
    
    v.list, cmd = v.list.Update(msg)
    cmds = append(cmds, cmd)
    
    return v, tea.Batch(cmds...)
}

func (v *ServerView) updateListItems() {
    items := make([]list.Item, len(v.servers))
    for i, server := range v.servers {
        items[i] = ServerListItem{
            Name:      server.Name,
            Connected: server.Connected,
            ToolCount: server.ToolCount,
            Error:     server.LastError,
        }
    }
    v.list.SetItems(items)
}

type ServerListItem struct {
    Name      string
    Connected bool
    ToolCount int
    Error     string
}

func (s ServerListItem) Title() string {
    return s.Name
}

func (s ServerListItem) Description() string {
    status := "❌ Disconnected"
    if s.Connected {
        status = "✅ Connected"
    }
    desc := fmt.Sprintf("%s • %d tools", status, s.ToolCount)
    if s.Error != "" {
        desc += fmt.Sprintf("\n   Error: %s", s.Error)
    }
    return desc
}

func (s ServerListItem) FilterValue() string {
    return s.Name
}
```

---

### Phase 3: Chat-Tool Integration (Week 3)

**Goal**: Enable tool execution from chat messages

#### 3.1 Tool Call Detection and Execution

**Test First** - `internal/tui/chat_view_test.go`:
```go
func TestChatView_DetectsToolCalls(t *testing.T) {
    view := setupTestChatView(t)
    
    // User asks to use a tool
    userMsg := "Can you read the file /tmp/test.txt?"
    
    // Should detect tool call intent
    toolCalls := view.DetectToolCalls(userMsg)
    assert.NotEmpty(t, toolCalls)
    assert.Equal(t, "read_file", toolCalls[0].Name)
}

func TestChatView_ExecutesToolsFromMessage(t *testing.T) {
    agent := setupTestAgent(t)
    view := NewChatView(DefaultStyles(), DefaultKeyMap(), agent)
    
    // Submit message that requires tool
    view.input.SetValue("List files in /tmp")
    
    // Process submission
    _, cmd := view.handleSubmit()
    
    // Should trigger model query with tool context
    assert.NotNil(t, cmd)
}

func TestChatView_DisplaysToolExecution(t *testing.T) {
    view := setupTestChatView(t)
    
    // Simulate tool execution message
    result := &mcp.ExecuteResult{
        Tool: mcp.Tool{Name: "read_file"},
        Result: &mcp.ToolResult{
            Content: []mcp.Content{
                {Type: "text", Text: "file contents"},
            },
        },
    }
    
    msg := MCPToolExecutedMsg{
        ToolName: "read_file",
        Result:   result,
    }
    
    _, cmd := view.Update(msg)
    
    // Should display tool result in chat
    messages := view.GetMessages()
    found := false
    for _, m := range messages {
        if m.Role == "tool" && m.ToolCall != nil {
            found = true
            assert.Equal(t, "read_file", m.ToolCall.Name)
            break
        }
    }
    assert.True(t, found)
}
```

**Implementation** - `internal/tui/chat_view.go`:
```go
type ChatView struct {
    // ... existing fields ...
    agent *agent.Agent
    toolExecutionInProgress map[string]bool
}

func NewChatView(styles Styles, keymap KeyMap, agent *agent.Agent) *ChatView {
    // ... existing initialization ...
    
    return &ChatView{
        // ... existing fields ...
        agent: agent,
        toolExecutionInProgress: make(map[string]bool),
    }
}

func (v *ChatView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    
    switch msg := msg.(type) {
    case MCPToolExecutingMsg:
        // Show tool execution in progress
        v.toolExecutionInProgress[msg.ToolName] = true
        v.AddMessage(ChatMessage{
            Role:      "tool",
            Content:   fmt.Sprintf("Executing tool: %s...", msg.ToolName),
            Timestamp: time.Now().Format("15:04"),
            ToolCall: &ToolCallInfo{
                Name: msg.ToolName,
                Args: msg.Params,
            },
        })
        return v, nil
        
    case MCPToolExecutedMsg:
        // Remove from in-progress
        delete(v.toolExecutionInProgress, msg.ToolName)
        
        if msg.Error != nil {
            // Display error
            v.AddMessage(ChatMessage{
                Role:      "tool",
                Content:   "",
                Error:     msg.Error.Error(),
                Timestamp: time.Now().Format("15:04"),
                ToolCall: &ToolCallInfo{
                    Name: msg.ToolName,
                },
            })
        } else {
            // Display result
            resultText := formatToolResult(msg.Result)
            v.AddMessage(ChatMessage{
                Role:      "tool",
                Content:   resultText,
                Timestamp: time.Now().Format("15:04"),
                ToolCall: &ToolCallInfo{
                    Name:   msg.ToolName,
                    Result: resultText,
                },
            })
            
            // Send result to model for synthesis
            cmds = append(cmds, v.synthesizeToolResult(msg.ToolName, resultText))
        }
        return v, tea.Batch(cmds...)
        
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            if !v.input.Focused() {
                return v, nil
            }
            return v, v.handleSubmit()
        }
    }
    
    return v, tea.Batch(cmds...)
}

func (v *ChatView) handleSubmit() tea.Cmd {
    userInput := strings.TrimSpace(v.input.Value())
    if userInput == "" {
        return nil
    }
    
    // Add user message
    v.AddMessage(ChatMessage{
        Role:      "user",
        Content:   userInput,
        Timestamp: time.Now().Format("15:04"),
    })
    
    v.input.SetValue("")
    v.waitingForResponse = true
    
    // Get available tools
    tools := v.agent.GetAvailableTools()
    
    // Create model request with tool context
    return v.generateResponseWithTools(userInput, tools)
}

func (v *ChatView) generateResponseWithTools(prompt string, tools []mcp.Tool) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
        defer cancel()
        
        // Format prompt with tool information
        systemPrompt := buildSystemPromptWithTools(tools)
        fullPrompt := fmt.Sprintf("%s\n\nUser: %s", systemPrompt, prompt)
        
        // Call model
        response, err := v.agent.GenerateResponse(ctx, fullPrompt)
        if err != nil {
            return ModelResponseMsg{Error: err}
        }
        
        // Check if response requests tool execution
        if toolCall := parseToolCall(response); toolCall != nil {
            // Execute tool
            return MCPToolExecutingMsg{
                ToolName: toolCall.Name,
                Params:   toolCall.Args,
            }
        }
        
        return ModelResponseMsg{
            Response: response,
        }
    }
}

func buildSystemPromptWithTools(tools []mcp.Tool) string {
    var sb strings.Builder
    sb.WriteString("You are a helpful AI assistant with access to the following tools:\n\n")
    
    for _, tool := range tools {
        sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
        if tool.InputSchema != nil {
            sb.WriteString(fmt.Sprintf("  Parameters: %v\n", tool.InputSchema))
        }
    }
    
    sb.WriteString("\nWhen you need to use a tool, respond with: USE_TOOL: <tool_name>(<params>)\n")
    sb.WriteString("Example: USE_TOOL: read_file({\"path\": \"/tmp/test.txt\"})\n\n")
    
    return sb.String()
}

func parseToolCall(response string) *ToolCallRequest {
    // Simple parser for tool calls
    // Format: USE_TOOL: tool_name(params_json)
    if !strings.Contains(response, "USE_TOOL:") {
        return nil
    }
    
    // Extract tool name and params
    // TODO: Implement robust parsing
    
    return nil
}

func formatToolResult(result *mcp.ExecuteResult) string {
    if result.Result == nil {
        return "Tool executed successfully (no result)"
    }
    
    var sb strings.Builder
    for _, content := range result.Result.Content {
        if content.Type == "text" {
            sb.WriteString(content.Text)
        } else if content.Type == "image" {
            sb.WriteString(fmt.Sprintf("[Image: %s]", content.Data))
        } else if content.Type == "resource" {
            sb.WriteString(fmt.Sprintf("[Resource: %s]", content.Resource.URI))
        }
        sb.WriteString("\n")
    }
    
    return sb.String()
}
```

---

### Phase 4: Model-Tool Integration (Week 4)

**Goal**: Enable model to understand and request tools

#### 4.1 Extend Model Interface for Tool Awareness

**Test First** - `internal/model/ollama_test.go`:
```go
func TestOllamaModel_GenerateWithTools(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return response with tool call
        response := map[string]interface{}{
            "response": "USE_TOOL: read_file({\"path\": \"/tmp/test.txt\"})",
        }
        json.NewEncoder(w).Encode(response)
    }))
    defer server.Close()
    
    model := NewOllamaModel(server.URL, "qwen2.5:3b")
    
    tools := []Tool{
        {Name: "read_file", Description: "Read a file"},
    }
    
    response, err := model.GenerateWithTools(context.Background(), "Read test.txt", tools)
    require.NoError(t, err)
    
    // Should detect tool call in response
    assert.Contains(t, response.Content, "USE_TOOL:")
    assert.NotNil(t, response.ToolCall)
    assert.Equal(t, "read_file", response.ToolCall.Name)
}
```

**Implementation** - `internal/model/types.go`:
```go
type Tool struct {
    Name        string
    Description string
    Parameters  map[string]interface{}
}

type GenerateOptions struct {
    Temperature float64
    MaxTokens   int
    Tools       []Tool
}

type ToolCall struct {
    Name      string
    Arguments map[string]interface{}
}

type Response struct {
    Content  string
    ToolCall *ToolCall
}
```

**Implementation** - `internal/model/ollama.go`:
```go
func (m *OllamaModel) GenerateWithTools(ctx context.Context, prompt string, tools []Tool, opts ...GenerateOption) (*Response, error) {
    // Build system prompt with tool descriptions
    systemPrompt := buildToolSystemPrompt(tools)
    
    // Combine with user prompt
    fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, prompt)
    
    // Generate response
    response, err := m.Generate(ctx, fullPrompt, opts...)
    if err != nil {
        return nil, err
    }
    
    // Parse for tool calls
    toolCall := parseToolCallFromResponse(response.Content)
    if toolCall != nil {
        response.ToolCall = toolCall
    }
    
    return response, nil
}

func buildToolSystemPrompt(tools []Tool) string {
    var sb strings.Builder
    sb.WriteString("You have access to the following tools:\n\n")
    
    for _, tool := range tools {
        sb.WriteString(fmt.Sprintf("Tool: %s\n", tool.Name))
        sb.WriteString(fmt.Sprintf("Description: %s\n", tool.Description))
        if len(tool.Parameters) > 0 {
            paramsJSON, _ := json.MarshalIndent(tool.Parameters, "", "  ")
            sb.WriteString(fmt.Sprintf("Parameters: %s\n", paramsJSON))
        }
        sb.WriteString("\n")
    }
    
    sb.WriteString("To use a tool, respond with this exact format:\n")
    sb.WriteString("TOOL_CALL: {\"name\": \"tool_name\", \"arguments\": {...}}\n")
    
    return sb.String()
}
```

---

### Phase 5: Real-time Notifications (Week 5)

**Goal**: Implement live updates from MCP servers

#### 5.1 Notification Handler Integration

**Implementation** - `internal/tui/application.go`:
```go
type Application struct {
    // ... existing fields ...
    notificationChan chan mcp.Notification
    agent            *agent.Agent
}

func NewApplication(agent *agent.Agent) *Application {
    // ... existing initialization ...
    
    app := &Application{
        // ... existing fields ...
        agent:            agent,
        notificationChan: make(chan mcp.Notification, 100),
    }
    
    // Subscribe to MCP notifications
    agent.SubscribeToNotifications(app.notificationChan)
    
    return app
}

func (a *Application) Init() tea.Cmd {
    return tea.Batch(
        a.chatView.Init(),
        a.serverView.Init(),
        a.listenForNotifications(),
    )
}

func (a *Application) listenForNotifications() tea.Cmd {
    return func() tea.Msg {
        select {
        case notif := <-a.notificationChan:
            return convertNotificationToMsg(notif)
        }
    }
}

func convertNotificationToMsg(notif mcp.Notification) tea.Msg {
    switch notif.Method {
    case "notifications/tools/list_changed":
        return MCPToolsRefreshedMsg{
            ServerName: notif.ServerName,
        }
    case "notifications/resources/list_changed":
        return MCPResourcesChangedMsg{
            ServerName: notif.ServerName,
        }
    default:
        return MCPNotificationMsg{
            Notification: notif,
        }
    }
}
```

---

## Testing Strategy

### Unit Tests

Each component should have comprehensive unit tests:

```go
// Agent tests
- TestAgent_InitializeMCP
- TestAgent_GetAvailableTools
- TestAgent_ExecuteTool
- TestAgent_HandleServerFailure

// MCP Manager tests
- TestMCPManager_AddServer
- TestMCPManager_RemoveServer
- TestMCPManager_ListServers
- TestMCPManager_ServerLifecycle

// TUI tests
- TestServerView_DisplaysRealServers
- TestServerView_HandlesMCPMessages
- TestChatView_DetectsToolCalls
- TestChatView_ExecutesTools
- TestChatView_DisplaysToolResults

// Model tests
- TestModel_GenerateWithTools
- TestModel_ParseToolCalls
```

### Integration Tests

End-to-end scenarios:

```go
func TestIntegration_FullToolExecution(t *testing.T) {
    // Start test MCP server
    server := startTestMCPServer(t)
    defer server.Stop()
    
    // Create agent with MCP
    config := testConfig(server.Address())
    agent, err := agent.New(config)
    require.NoError(t, err)
    
    // Start agent
    ctx := context.Background()
    require.NoError(t, agent.Start(ctx))
    
    // Verify tools are available
    tools := agent.GetAvailableTools()
    assert.NotEmpty(t, tools)
    
    // Execute tool
    result, err := agent.ExecuteTool(ctx, "list_files", map[string]interface{}{
        "path": "/tmp",
    })
    require.NoError(t, err)
    assert.NotNil(t, result.Result)
    
    // Verify result content
    assert.NotEmpty(t, result.Result.Content)
}

func TestIntegration_TUIWithMCP(t *testing.T) {
    // This would test the full TUI flow
    // Using a headless bubbletea test harness
}
```

### Manual Testing Checklist

- [ ] Start Othello with filesystem server configured
- [ ] Verify server appears in Server View (Ctrl+S)
- [ ] Check that tools are listed correctly
- [ ] Send chat message: "List files in /tmp"
- [ ] Verify tool execution appears in chat
- [ ] Verify tool result is displayed
- [ ] Verify model synthesizes result into response
- [ ] Test server disconnection handling
- [ ] Test tool execution errors
- [ ] Test real-time notification updates

---

## Success Criteria

### Functional Requirements

1. ✅ Agent initializes MCP registry and executor on startup
2. ✅ Agent connects to all configured MCP servers
3. ✅ Server View displays real server status and tool counts
4. ✅ Chat can trigger tool execution based on user intent
5. ✅ Tool execution results are displayed in chat
6. ✅ Model receives tool descriptions and can request tools
7. ✅ Real-time notifications update UI automatically

### Performance Requirements

1. Server connection: < 2 seconds per server
2. Tool discovery: < 1 second per server
3. Tool execution: < 5 seconds (depends on tool)
4. UI remains responsive during tool execution
5. Memory usage: < 150MB with 5 servers and 50 tools

### User Experience Requirements

1. Clear visual feedback during tool execution
2. Error messages are user-friendly and actionable
3. Server status is always current and accurate
4. Tool results are formatted for readability
5. Keyboard shortcuts work consistently

---

## Timeline

- **Week 1**: Agent-MCP Integration (Days 1-5)
- **Week 2**: TUI-MCP Integration (Days 6-10)
- **Week 3**: Chat-Tool Integration (Days 11-15)
- **Week 4**: Model-Tool Integration (Days 16-20)
- **Week 5**: Real-time Notifications + Polish (Days 21-25)

**Total**: 5 weeks to complete MCP-TUI integration

---

## Risk Mitigation

### Technical Risks

1. **Tool call parsing from LLM responses**
   - Risk: Models don't reliably output parseable tool calls
   - Mitigation: Use structured prompts, provide clear examples, implement fallbacks

2. **MCP server reliability**
   - Risk: Servers crash or hang during operation
   - Mitigation: Timeouts, health checks, automatic reconnection

3. **UI performance with many tools**
   - Risk: Slow rendering with 100+ tools
   - Mitigation: Lazy loading, pagination, virtual scrolling

### Process Risks

1. **Scope creep**
   - Risk: Adding too many features during integration
   - Mitigation: Stick to core MCP integration, defer enhancements

2. **Testing complexity**
   - Risk: Hard to test TUI interactions
   - Mitigation: Use bubbletea testing utils, focus on logic tests

---

## Next Steps

1. **Immediate**: Start Week 1 implementation (Agent-MCP Integration)
2. **Review**: Update this document based on implementation learnings
3. **Validation**: Run integration tests after each week
4. **Documentation**: Update user docs with MCP usage examples

---

*This plan prioritizes getting MCP tools working in the TUI with a solid TDD approach. Each phase builds on the previous one, ensuring incremental progress with continuous validation.*
