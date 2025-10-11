package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Model interface defines the operations for language models
type Model interface {
	Generate(ctx context.Context, prompt string, options GenerateOptions) (*Response, error)
	Chat(ctx context.Context, messages []Message, options GenerateOptions) (*Response, error)
	ChatWithTools(ctx context.Context, messages []Message, tools []ToolDefinition, options GenerateOptions) (*Response, error)
	IsAvailable(ctx context.Context) bool
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// ToolDefinition represents a tool that can be called by the model
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call request from the model
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// GenerateOptions contains options for generation
type GenerateOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// Response represents a model response
type Response struct {
	Content      string        `json:"content"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	FinishReason string        `json:"finish_reason,omitempty"`
	Usage        Usage         `json:"usage,omitempty"`
	Duration     time.Duration `json:"duration,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OllamaModel implements the Model interface for Ollama
type OllamaModel struct {
	host      string
	modelName string
	client    *http.Client
}

// NewOllamaModel creates a new Ollama model instance
func NewOllamaModel(host, modelName string) *OllamaModel {
	return &OllamaModel{
		host:      host,
		modelName: modelName,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Generate generates text from a prompt
func (m *OllamaModel) Generate(ctx context.Context, prompt string, options GenerateOptions) (*Response, error) {
	// Convert to chat format for consistency
	messages := []Message{
		{Role: "user", Content: prompt},
	}
	return m.Chat(ctx, messages, options)
}

// Chat performs a chat completion
func (m *OllamaModel) Chat(ctx context.Context, messages []Message, options GenerateOptions) (*Response, error) {
	start := time.Now()
	
	// Prepare request payload
	payload := map[string]interface{}{
		"model":    m.modelName,
		"messages": messages,
		"stream":   false,
	}
	
	// Add options if provided
	if options.Temperature > 0 {
		payload["temperature"] = options.Temperature
	}
	if options.MaxTokens > 0 {
		payload["max_tokens"] = options.MaxTokens
	}
	if options.TopP > 0 {
		payload["top_p"] = options.TopP
	}
	
	// Marshal request
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	
	// Create HTTP request
	url := fmt.Sprintf("%s/api/chat", m.host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API error %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var ollamaResponse struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done   bool `json:"done"`
		Error  string `json:"error,omitempty"`
	}
	
	if err := json.Unmarshal(body, &ollamaResponse); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	
	if ollamaResponse.Error != "" {
		return nil, fmt.Errorf("ollama error: %s", ollamaResponse.Error)
	}
	
	duration := time.Since(start)
	
	return &Response{
		Content:  ollamaResponse.Message.Content,
		Duration: duration,
		Usage: Usage{
			// Ollama doesn't provide token counts by default
			TotalTokens: len(ollamaResponse.Message.Content) / 4, // Rough estimate
		},
	}, nil
}

// ChatWithTools performs a chat completion with tool calling capabilities
func (m *OllamaModel) ChatWithTools(ctx context.Context, messages []Message, tools []ToolDefinition, options GenerateOptions) (*Response, error) {
	// For now, we'll implement tool calling by including tool descriptions in the system prompt
	// and parsing the response for tool calls. This is a simplified approach that works with
	// models that don't have native tool calling support.
	
	// Create system message with tool descriptions
	toolPrompt := m.createToolPrompt(tools)
	
	// Add system message with tool instructions
	enhancedMessages := []Message{
		{Role: "system", Content: toolPrompt},
	}
	enhancedMessages = append(enhancedMessages, messages...)
	
	// Use regular chat endpoint
	response, err := m.Chat(ctx, enhancedMessages, options)
	if err != nil {
		return nil, err
	}
	
	// Parse response for tool calls
	toolCalls := m.parseToolCalls(response.Content)
	response.ToolCalls = toolCalls
	
	return response, nil
}

// createToolPrompt creates a system prompt that describes available tools
func (m *OllamaModel) createToolPrompt(tools []ToolDefinition) string {
	if len(tools) == 0 {
		return "You are a helpful AI assistant."
	}
	
	prompt := `You are a helpful AI assistant with access to tools. When you need to use a tool, respond with a tool call in this exact format:

TOOL_CALL: tool_name
ARGUMENTS: {"param1": "value1", "param2": "value2"}

Available tools:
`
	
	for _, tool := range tools {
		prompt += fmt.Sprintf("\n- %s: %s", tool.Name, tool.Description)
		if tool.Parameters != nil {
			prompt += fmt.Sprintf("\n  Parameters: %v", tool.Parameters)
		}
	}
	
	prompt += "\n\nOnly use tools when necessary to answer the user's question. If you don't need a tool, respond normally."
	
	return prompt
}

// parseToolCalls extracts tool calls from the model response
func (m *OllamaModel) parseToolCalls(content string) []ToolCall {
	var toolCalls []ToolCall
	
	lines := strings.Split(content, "\n")
	var currentToolCall *ToolCall
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "TOOL_CALL:") {
			if currentToolCall != nil {
				toolCalls = append(toolCalls, *currentToolCall)
			}
			toolName := strings.TrimSpace(strings.TrimPrefix(line, "TOOL_CALL:"))
			currentToolCall = &ToolCall{
				Name:      toolName,
				Arguments: make(map[string]interface{}),
			}
		} else if strings.HasPrefix(line, "ARGUMENTS:") && currentToolCall != nil {
			argsJson := strings.TrimSpace(strings.TrimPrefix(line, "ARGUMENTS:"))
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(argsJson), &args); err == nil {
				currentToolCall.Arguments = args
			}
		}
	}
	
	// Add the last tool call if exists
	if currentToolCall != nil {
		toolCalls = append(toolCalls, *currentToolCall)
	}
	
	return toolCalls
}

// IsAvailable checks if the model is available
func (m *OllamaModel) IsAvailable(ctx context.Context) bool {
	url := fmt.Sprintf("%s/api/tags", m.host)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}
	
	resp, err := m.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false
	}
	
	// Parse response to check if our model is available
	var tagsResponse struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	
	if err := json.Unmarshal(body, &tagsResponse); err != nil {
		return false
	}
	
	// Check if our model is in the list
	for _, model := range tagsResponse.Models {
		if model.Name == m.modelName {
			return true
		}
	}
	
	return false
}