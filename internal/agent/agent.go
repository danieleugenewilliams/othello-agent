package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
	"github.com/danieleugenewilliams/othello-agent/internal/tui"
)

// sanitizeAndParseJSON implements robust JSON parsing with multiple fallback strategies
func sanitizeAndParseJSON(rawJSON string, logger *log.Logger) (interface{}, error) {
	if logger != nil {
		logger.Printf("[JSON-SANITIZE] Starting JSON sanitization, input length: %d", len(rawJSON))
	}

	// Strategy 1: Try parsing as-is first
	var result interface{}
	if err := json.Unmarshal([]byte(rawJSON), &result); err == nil {
		if logger != nil {
			logger.Printf("[JSON-SANITIZE] Strategy 1 success: Direct parsing worked")
		}
		return result, nil
	} else if logger != nil {
		logger.Printf("[JSON-SANITIZE] Strategy 1 failed: %v", err)
	}

	// Strategy 2: Clean UTF-8 and try again
	cleanedJSON := cleanUTF8String(rawJSON)
	if err := json.Unmarshal([]byte(cleanedJSON), &result); err == nil {
		if logger != nil {
			logger.Printf("[JSON-SANITIZE] Strategy 2 success: UTF-8 cleaning worked")
		}
		return result, nil
	} else if logger != nil {
		logger.Printf("[JSON-SANITIZE] Strategy 2 failed: %v", err)
	}

	// Strategy 3: Remove control characters and invalid sequences
	sanitizedJSON := removeInvalidJSONChars(cleanedJSON)
	if err := json.Unmarshal([]byte(sanitizedJSON), &result); err == nil {
		if logger != nil {
			logger.Printf("[JSON-SANITIZE] Strategy 3 success: Character sanitization worked")
		}
		return result, nil
	} else if logger != nil {
		logger.Printf("[JSON-SANITIZE] Strategy 3 failed: %v", err)
	}

	// Strategy 4: Extract JSON from mixed content using regex
	extractedJSON := extractJSONFromMixedContent(sanitizedJSON)
	if extractedJSON != "" && extractedJSON != sanitizedJSON {
		if err := json.Unmarshal([]byte(extractedJSON), &result); err == nil {
			if logger != nil {
				logger.Printf("[JSON-SANITIZE] Strategy 4 success: JSON extraction worked")
			}
			return result, nil
		} else if logger != nil {
			logger.Printf("[JSON-SANITIZE] Strategy 4 failed: %v", err)
		}
	}

	if logger != nil {
		logger.Printf("[JSON-SANITIZE] All strategies failed, returning error")
	}
	return nil, fmt.Errorf("failed to parse JSON after all sanitization attempts")
}

// cleanUTF8String removes invalid UTF-8 sequences and replaces them with valid chars
func cleanUTF8String(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))

	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 sequence, skip this byte
			s = s[1:]
		} else if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			// Skip most control characters but keep newlines and tabs
			s = s[size:]
		} else {
			builder.WriteRune(r)
			s = s[size:]
		}
	}
	return builder.String()
}

// removeInvalidJSONChars removes characters that commonly break JSON parsing
func removeInvalidJSONChars(s string) string {
	// Remove null bytes and other problematic characters
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "\ufffd", "") // Unicode replacement character

	// Remove sequences that look like encoding artifacts
	invalidPatterns := []string{
		"ð", "Ã", "â", "Â", // Common UTF-8 encoding artifacts
	}

	for _, pattern := range invalidPatterns {
		s = strings.ReplaceAll(s, pattern, "")
	}

	return strings.TrimSpace(s)
}

// extractJSONFromMixedContent attempts to extract valid JSON from mixed content
func extractJSONFromMixedContent(s string) string {
	// Look for JSON object boundaries
	openBrace := strings.Index(s, "{")
	if openBrace == -1 {
		return s
	}

	// Find matching closing brace by counting
	braceCount := 0
	inString := false
	escape := false

	for i := openBrace; i < len(s); i++ {
		char := s[i]

		if escape {
			escape = false
			continue
		}

		if char == '\\' {
			escape = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					// Found complete JSON object
					return s[openBrace : i+1]
				}
			}
		}
	}

	// If we couldn't find a complete object, return original
	return s
}

// getMapKeys returns the keys of a map for logging purposes
func getMapKeys(m map[string]interface{}) []string {
	var k []string
	for key := range m {
		k = append(k, key)
	}
	return k
}

// extractRawDataFromToolResult extracts the raw JSON data from a ToolResult
// for processing by ToolResultProcessor
func extractRawDataFromToolResult(toolResult *mcp.ToolResult) (interface{}, error) {
	if toolResult == nil {
		log.Printf("[EXTRACTION] Tool result is nil")
		return nil, fmt.Errorf("tool result is nil")
	}

	if len(toolResult.Content) == 0 {
		log.Printf("[EXTRACTION] Tool result has no content")
		return nil, fmt.Errorf("tool result has no content")
	}

	log.Printf("[EXTRACTION] Tool result has %d content items", len(toolResult.Content))

	// Get the first content item (most MCP tools return a single content item)
	content := toolResult.Content[0]
	log.Printf("[EXTRACTION] First content type: %s", content.Type)

	// If the content type is text, try to parse it as JSON
	if content.Type == "text" && content.Text != "" {
		log.Printf("[EXTRACTION] Processing text content, length: %d", len(content.Text))
		if len(content.Text) < 500 {
			log.Printf("[EXTRACTION] Text content: %s", content.Text)
		}

		var rawData interface{}
		if err := json.Unmarshal([]byte(content.Text), &rawData); err != nil {
			log.Printf("[EXTRACTION] Failed to parse JSON, returning text as-is: %v", err)
			// If it's not valid JSON, return the text as-is
			return content.Text, nil
		}

		log.Printf("[EXTRACTION] Successfully parsed JSON, transforming response")
		// Transform MCP response structure to match ProcessToolResult expectations
		transformed := transformMCPResponse(rawData)
		log.Printf("[EXTRACTION] Transformation complete, result type: %T", transformed)
		return transformed, nil
	}

	// If content type is not text or text is empty, try the Data field
	if content.Data != "" {
		log.Printf("[EXTRACTION] Processing data field, length: %d", len(content.Data))
		var rawData interface{}
		if err := json.Unmarshal([]byte(content.Data), &rawData); err != nil {
			log.Printf("[EXTRACTION] Failed to parse Data JSON, returning data as-is: %v", err)
			// If it's not valid JSON, return the data as-is
			return content.Data, nil
		}

		log.Printf("[EXTRACTION] Successfully parsed Data JSON, transforming response")
		// Transform MCP response structure to match ProcessToolResult expectations
		return transformMCPResponse(rawData), nil
	}

	log.Printf("[EXTRACTION] No usable content found, returning entire ToolResult")
	// Fallback: return the entire ToolResult if we can't extract anything meaningful
	return toolResult, nil
}

// transformMCPResponse transforms the actual MCP response structure into what
// ToolResultProcessor expects
func transformMCPResponse(rawData interface{}) interface{} {
	log.Printf("[TRANSFORM] Input data type: %T", rawData)

	dataMap, ok := rawData.(map[string]interface{})
	if !ok {
		log.Printf("[TRANSFORM] Data is not a map, returning as-is")
		return rawData // Return as-is if not a map
	}

	log.Printf("[TRANSFORM] Data map has keys: %v", getMapKeys(dataMap))

	// Handle local-memory search response format
	if data, hasData := dataMap["data"].([]interface{}); hasData {
		log.Printf("[TRANSFORM] Found 'data' field with %d items, transforming to MCP format", len(data))
		// Transform: {"data": [{"memory": {...}}, ...], "total_results": N}
		// To: {"results": [{...}, ...], "total_count": N}
		results := make([]interface{}, len(data))
		for i, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if memory, hasMemory := itemMap["memory"]; hasMemory {
					results[i] = memory
				} else {
					results[i] = itemMap
				}
			} else {
				results[i] = item
			}
		}

		transformed := map[string]interface{}{
			"results": results,
		}

		// Add total count if available
		if totalResults, hasTotalResults := dataMap["total_results"]; hasTotalResults {
			transformed["total_count"] = totalResults
		} else if count, hasCount := dataMap["count"]; hasCount {
			transformed["total_count"] = count
		} else {
			transformed["total_count"] = len(results)
		}

		// Copy over other relevant fields
		for key, value := range dataMap {
			if key != "data" && key != "total_results" && key != "count" {
				transformed[key] = value
			}
		}

		log.Printf("[TRANSFORM] Transformation complete, result keys: %v", getMapKeys(transformed))
		return transformed
	}

	// Handle other MCP response formats (pass through)
	log.Printf("[TRANSFORM] No data field found, passing through as-is")
	return rawData
}

// Agent represents the core agent instance
type Agent struct {
	config       *config.Config
	logger       *log.Logger
	model        model.Model     // For LLM-based metadata extraction
	mcpRegistry  *mcp.ToolRegistry
	mcpManager   *MCPManager
	toolExecutor *mcp.ToolExecutor
	updateChan   chan interface{} // Channel for broadcasting status updates
}

// Interface defines the agent's public API
type Interface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	StartTUI() error
	GetStatus() *Status
	GetMCPServers() []ServerInfo
	GetMCPTools(ctx context.Context) ([]tui.Tool, error)
	SubscribeToUpdates() <-chan interface{}
	ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*tui.ToolExecutionResult, error)
}

// Status represents the current agent status
type Status struct {
	Running        bool   `json:"running"`
	ConfigFile     string `json:"config_file"`
	ModelConnected bool   `json:"model_connected"`
	MCPServers     int    `json:"mcp_servers"`
}

// New creates a new agent instance
func New(cfg *config.Config) (*Agent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	// Set up file-based logging
	logger, err := setupFileLogger(cfg.Logging.File)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}

	// Initialize MCP registry with logger adapter
	mcpLogger := &agentLogger{logger: logger}
	mcpRegistry := mcp.NewToolRegistry(mcpLogger)

	// Initialize MCP manager
	mcpManager := NewMCPManager(mcpRegistry, mcpLogger)

	// Initialize tool executor
	toolExecutor := mcp.NewToolExecutor(mcpRegistry, mcpLogger)

	agent := &Agent{
		config:       cfg,
		logger:       logger,
		mcpRegistry:  mcpRegistry,
		mcpManager:   mcpManager,
		toolExecutor: toolExecutor,
		updateChan:   make(chan interface{}, 100), // Buffered channel for updates
	}

	// Set up the callback for MCP status updates
	mcpManager.SetUpdateCallback(agent.broadcastUpdate)

	return agent, nil
}

// setupFileLogger creates a file-based logger with the specified log file path
func setupFileLogger(logFilePath string) (*log.Logger, error) {
	// Expand tilde to home directory if present
	if len(logFilePath) >= 2 && logFilePath[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		logFilePath = filepath.Join(homeDir, logFilePath[2:])
	}

	// Create the directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Open or create the log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}

	// Create logger that writes to the file
	logger := log.New(logFile, "[AGENT] ", log.LstdFlags)

	return logger, nil
}

// agentLogger adapts standard log.Logger to the MCP Logger interface
type agentLogger struct {
	logger *log.Logger
}

func (a *agentLogger) Info(msg string, args ...interface{}) {
	a.logger.Printf("[INFO] "+msg, args...)
}

func (a *agentLogger) Error(msg string, args ...interface{}) {
	a.logger.Printf("[ERROR] "+msg, args...)
}

func (a *agentLogger) Debug(msg string, args ...interface{}) {
	a.logger.Printf("[DEBUG] "+msg, args...)
}

// Start starts the agent with the given context
// SetModel sets the model for LLM-based metadata extraction
func (a *Agent) SetModel(m model.Model) {
	a.model = m
	a.logger.Printf("Model set for LLM-based metadata extraction")
}

func (a *Agent) Start(ctx context.Context) error {
	a.logger.Println("Starting Othello AI Agent")
	
	// Use the agent's own configuration instead of loading from filesystem
	servers := a.config.MCP.Servers
	
	// Initialize MCP servers
	for _, serverCfg := range servers {
		a.logger.Printf("Connecting to MCP server: %s", serverCfg.Name)
		if err := a.mcpManager.AddServer(ctx, serverCfg); err != nil {
			a.logger.Printf("Failed to connect to MCP server %s: %v", serverCfg.Name, err)
			// Continue with other servers even if one fails
			continue
		}
		a.logger.Printf("Successfully connected to MCP server: %s", serverCfg.Name)
	}
	
	a.logger.Printf("Agent started with model: %s", a.config.Model.Name)
	return nil
}

// Stop gracefully stops the agent
func (a *Agent) Stop(ctx context.Context) error {
	a.logger.Println("Stopping Othello AI Agent")
	
	// Stop MCP connections
	if err := a.mcpManager.Close(ctx); err != nil {
		a.logger.Printf("Error stopping MCP connections: %v", err)
	}
	
	// Clear tool registry
	if a.mcpRegistry != nil {
		a.mcpRegistry.Clear()
	}
	
	a.logger.Println("Agent stopped")
	return nil
}

// StartTUI starts the terminal user interface
func (a *Agent) StartTUI() error {
	a.logger.Println("Starting TUI mode")
	
	// Create TUI application with agent integration
	keymap := tui.DefaultKeyMap()
	styles := tui.DefaultStyles()
	app := tui.NewApplicationWithAgent(keymap, styles, a)
	
	// Run the TUI
	program := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	
	return nil
}

// GetStatus returns the current agent status
func (a *Agent) GetStatus() *Status {
	return &Status{
		Running:        true, // TODO: Track actual state
		ConfigFile:     a.config.ConfigFile(),
		ModelConnected: false, // TODO: Check actual model connection
		MCPServers:     len(a.config.MCP.Servers),
	}
}

// GetMCPServers returns information about all registered MCP servers
func (a *Agent) GetMCPServers() []tui.ServerInfo {
	mcpServers := a.mcpManager.ListServers()
	
	// Convert agent.ServerInfo to tui.ServerInfo
	servers := make([]tui.ServerInfo, len(mcpServers))
	for i, mcpServer := range mcpServers {
		servers[i] = tui.ServerInfo{
			Name:      mcpServer.Name,
			Status:    mcpServer.Status,
			Connected: mcpServer.Connected,
			ToolCount: mcpServer.ToolCount,
			Transport: mcpServer.Transport,
			Error:     mcpServer.Error,
		}
	}
	
	return servers
}

// GetMCPTools returns all available tools from registered MCP servers
func (a *Agent) GetMCPTools(ctx context.Context) ([]tui.Tool, error) {
	mcpTools := a.mcpRegistry.ListTools()
	
	// Convert mcp.Tool to tui.Tool
	tools := make([]tui.Tool, len(mcpTools))
	for i, mcpTool := range mcpTools {
		tools[i] = tui.Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			Server:      mcpTool.ServerName,
		}
	}
	
	return tools, nil
}

// GetMCPToolsAsDefinitions converts MCP tools to model.ToolDefinition format
func (a *Agent) GetMCPToolsAsDefinitions(ctx context.Context) ([]model.ToolDefinition, error) {
	mcpTools := a.mcpRegistry.ListTools()
	
	// Use our new conversion function that properly handles JSON schemas
	definitions := ConvertMCPToolsToDefinitions(mcpTools)
	
	return definitions, nil
}

// SubscribeToUpdates returns a channel for receiving status updates
func (a *Agent) SubscribeToUpdates() <-chan interface{} {
	return a.updateChan
}

// ExecuteTool executes an MCP tool with the given parameters
func (a *Agent) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (*tui.ToolExecutionResult, error) {
	a.logger.Printf("Executing tool: %s with params: %+v", toolName, params)
	
	// Get the tool schema for validation
	tool, exists := a.mcpRegistry.GetTool(toolName)
	if !exists {
		err := fmt.Errorf("tool '%s' not found", toolName)
		a.logger.Printf("Tool not found: %s", toolName)
		return &tui.ToolExecutionResult{
			ToolName: toolName,
			Success:  false,
			Error:    err.Error(),
		}, nil
	}
	
	// Validate the tool call before execution
	toolCall := model.ToolCall{
		Name:      toolName,
		Arguments: params,
	}
	if err := ValidateToolCall(toolCall, tool); err != nil {
		a.logger.Printf("Tool validation failed for %s: %v", toolName, err)
		return &tui.ToolExecutionResult{
			ToolName: toolName,
			Success:  false,
			Error:    fmt.Sprintf("Invalid parameters: %v", err),
		}, nil
	}
	
	// Execute the tool using the tool executor
	result, err := a.toolExecutor.Execute(ctx, toolName, params)
	if err != nil {
		a.logger.Printf("Tool execution failed for %s: %v", toolName, err)
		return &tui.ToolExecutionResult{
			ToolName: toolName,
			Success:  false,
			Error:    err.Error(),
		}, nil
	}
	
	a.logger.Printf("Tool %s executed successfully", toolName)
	
	// Process the result into a natural language summary
	processor := &ToolResultProcessor{}

	// Use universal MCP processor directly with the ToolResult
	processedResult, err := processor.ProcessToolResult(ctx, toolName, result.Result, "")
	if err != nil {
		// Log error but don't fail - use original result as fallback
		a.logger.Printf("Warning: Failed to process result for %s: %v", toolName, err)
		processedResult = fmt.Sprintf("%v", result.Result)
	}
	
	// Note: Broadcasting moved to ExecuteToolUnified - this method is deprecated
	
	return &tui.ToolExecutionResult{
		ToolName: toolName,
		Success:  true,
		Result:   processedResult,
		Duration: result.Duration,
	}, nil
}

// ProcessToolResult processes tool results using the intelligent result processor
func (a *Agent) ProcessToolResult(ctx context.Context, toolName string, result *mcp.ExecuteResult, userQuery string) (string, error) {
	// Use universal MCP processor directly with the ToolResult
	processor := &ToolResultProcessor{Logger: a.logger}
	return processor.ProcessToolResult(ctx, toolName, result.Result, userQuery)
}

// ExecuteToolUnified provides a single, consistent pathway for tool execution
// This method replaces the dual pathways (direct + chat) with unified processing
func (a *Agent) ExecuteToolUnified(ctx context.Context, toolName string, params map[string]interface{}, userContext string) (string, error) {
	// Use the enhanced version with empty conversation context for backward compatibility
	convContext := &model.ConversationContext{
		UserQuery:   userContext,
		SessionType: "chat",
	}
	return a.ExecuteToolUnifiedWithContext(ctx, toolName, params, convContext)
}

// ExecuteToolUnifiedWithContext provides tool execution with conversation context for intelligent responses
func (a *Agent) ExecuteToolUnifiedWithContext(ctx context.Context, toolName string, params map[string]interface{}, convContext *model.ConversationContext) (string, error) {
	a.logger.Printf("Executing tool (unified with context): %s with params: %+v", toolName, params)
	a.logger.Printf("Conversation context: %d history messages, query: %s", len(convContext.History), convContext.UserQuery)
	log.Printf("🚀 UNIFIED EXECUTION STARTED (with context): %s", toolName)

	// Get the tool schema for validation
	tool, exists := a.mcpRegistry.GetTool(toolName)
	if !exists {
		err := fmt.Errorf("tool '%s' not found", toolName)
		a.logger.Printf("Tool not found: %s", toolName)
		return "", err
	}

	// Validate the tool call before execution
	toolCall := model.ToolCall{
		Name:      toolName,
		Arguments: params,
	}
	if err := ValidateToolCall(toolCall, tool); err != nil {
		a.logger.Printf("Tool validation failed for %s: %v", toolName, err)
		return "", fmt.Errorf("invalid parameters: %v", err)
	}

	// Execute the tool using the tool executor
	result, err := a.toolExecutor.Execute(ctx, toolName, params)
	if err != nil {
		a.logger.Printf("Tool execution failed for %s: %v", toolName, err)
		return "", err
	}

	a.logger.Printf("Tool %s executed successfully (unified with context)", toolName)

	// Use enhanced MCP processor with conversation context and model for LLM-based extraction
	processor := &ToolResultProcessor{
		Logger: a.logger,
		Model:  a.model,
	}
	a.logger.Printf("[UNIFIED] About to call processor with toolName=%s and conversation context", toolName)
	processedResult, err := processor.ProcessToolResultWithContext(ctx, toolName, result.Result, convContext)
	a.logger.Printf("[UNIFIED] Context-aware processor returned result length=%d, error=%v", len(processedResult), err)
	if err != nil {
		// Log error but don't fail - use a basic fallback
		a.logger.Printf("Warning: Failed to process result for %s: %v", toolName, err)
		if result.Result != nil && len(result.Result.Content) > 0 {
			processedResult = result.Result.Content[0].Text
		} else {
			processedResult = "Tool executed successfully but couldn't process the result."
		}
	}

	// Update conversation context with this tool usage
	if convContext.PreviousTools == nil {
		convContext.PreviousTools = make([]string, 0)
	}
	convContext.PreviousTools = append(convContext.PreviousTools, toolName)

	// Broadcast unified tool execution update
	a.broadcastUpdate(tui.ToolExecutedUnifiedMsg{
		ToolName: toolName,
		Result:   processedResult,
		Success:  true,
	})

	return processedResult, nil
}

// broadcastUpdate sends an update to all subscribers (non-blocking)
func (a *Agent) broadcastUpdate(update interface{}) {
	select {
	case a.updateChan <- update:
		// Update sent successfully
	default:
		// Channel is full, drop the update to avoid blocking
		a.logger.Printf("Warning: Update channel full, dropping update")
	}
}