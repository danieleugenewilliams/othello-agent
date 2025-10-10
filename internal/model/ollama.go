package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Model interface defines the operations for language models
type Model interface {
	Generate(ctx context.Context, prompt string, options GenerateOptions) (*Response, error)
	Chat(ctx context.Context, messages []Message, options GenerateOptions) (*Response, error)
	IsAvailable(ctx context.Context) bool
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
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