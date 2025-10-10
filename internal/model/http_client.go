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

// HTTPClient implements the Model interface for generic HTTP API providers
type HTTPClient struct {
	baseURL  string
	apiKey   string
	provider string // "lmstudio", "localai", "llama-cpp", "vllm", "textgen-webui"
	client   *http.Client
}

// NewHTTPClient creates a new HTTP API client
func NewHTTPClient(baseURL, apiKey, provider string) (*HTTPClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	// API key is optional for local servers
	// if apiKey == "" {
	// 	return nil, fmt.Errorf("apiKey cannot be empty")
	// }

	// Validate provider - focus on open source models
	validProviders := map[string]bool{
		"lmstudio":       true, // LM Studio local server
		"localai":        true, // LocalAI (OpenAI-compatible)
		"llama-cpp":      true, // llama.cpp HTTP server
		"vllm":           true, // vLLM inference server
		"textgen-webui":  true, // Text Generation WebUI (Oobabooga)
		"openai-compat":  true, // Generic OpenAI-compatible endpoint
	}
	if !validProviders[provider] {
		return nil, fmt.Errorf("unsupported provider: %s (supported: lmstudio, localai, llama-cpp, vllm, textgen-webui, openai-compat)", provider)
	}

	return &HTTPClient{
		baseURL:  baseURL,
		apiKey:   apiKey,
		provider: provider,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Generate generates text from a prompt
func (c *HTTPClient) Generate(ctx context.Context, prompt string, options GenerateOptions) (*Response, error) {
	messages := []Message{
		{Role: "user", Content: prompt},
	}
	return c.Chat(ctx, messages, options)
}

// Chat performs a chat completion
func (c *HTTPClient) Chat(ctx context.Context, messages []Message, options GenerateOptions) (*Response, error) {
	start := time.Now()

	switch c.provider {
	case "lmstudio", "localai", "openai-compat":
		// These use OpenAI-compatible API
		return c.chatOpenAICompatible(ctx, messages, options, start)
	case "llama-cpp":
		return c.chatLlamaCpp(ctx, messages, options, start)
	case "vllm":
		return c.chatVLLM(ctx, messages, options, start)
	case "textgen-webui":
		return c.chatTextGenWebUI(ctx, messages, options, start)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.provider)
	}
}

// chatOpenAICompatible handles OpenAI-compatible API calls (LM Studio, LocalAI, etc.)
func (c *HTTPClient) chatOpenAICompatible(ctx context.Context, messages []Message, options GenerateOptions, start time.Time) (*Response, error) {
	// Build request payload
	payload := map[string]interface{}{
		"model":    "gpt-4", // Default model
		"messages": messages,
	}

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
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.client.Do(req)
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
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResponse.Error.Message)
	}

	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	duration := time.Since(start)

	return &Response{
		Content:      apiResponse.Choices[0].Message.Content,
		FinishReason: apiResponse.Choices[0].FinishReason,
		Duration:     duration,
		Usage: Usage{
			PromptTokens:     apiResponse.Usage.PromptTokens,
			CompletionTokens: apiResponse.Usage.CompletionTokens,
			TotalTokens:      apiResponse.Usage.TotalTokens,
		},
	}, nil
}

// chatLlamaCpp handles llama.cpp HTTP server API calls
func (c *HTTPClient) chatLlamaCpp(ctx context.Context, messages []Message, options GenerateOptions, start time.Time) (*Response, error) {
	// llama.cpp uses /completion endpoint with prompt string
	// Convert messages to prompt
	var prompt string
	for _, msg := range messages {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	payload := map[string]interface{}{
		"prompt": prompt,
	}

	if options.Temperature > 0 {
		payload["temperature"] = options.Temperature
	}
	if options.MaxTokens > 0 {
		payload["n_predict"] = options.MaxTokens
	}
	if options.TopP > 0 {
		payload["top_p"] = options.TopP
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/completion", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Content string `json:"content"`
		Stop    bool   `json:"stop"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	duration := time.Since(start)

	return &Response{
		Content:      apiResponse.Content,
		FinishReason: "stop",
		Duration:     duration,
	}, nil
}

// chatVLLM handles vLLM inference server API calls (OpenAI-compatible)
func (c *HTTPClient) chatVLLM(ctx context.Context, messages []Message, options GenerateOptions, start time.Time) (*Response, error) {
	// vLLM is OpenAI-compatible
	return c.chatOpenAICompatible(ctx, messages, options, start)
}

// chatTextGenWebUI handles Text Generation WebUI (Oobabooga) API calls
func (c *HTTPClient) chatTextGenWebUI(ctx context.Context, messages []Message, options GenerateOptions, start time.Time) (*Response, error) {
	// Convert messages to prompt
	var prompt string
	for _, msg := range messages {
		if msg.Role == "system" {
			prompt += msg.Content + "\n\n"
		} else {
			prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
	}

	payload := map[string]interface{}{
		"prompt": prompt,
		"max_new_tokens": 200,
	}

	if options.Temperature > 0 {
		payload["temperature"] = options.Temperature
	}
	if options.MaxTokens > 0 {
		payload["max_new_tokens"] = options.MaxTokens
	}
	if options.TopP > 0 {
		payload["top_p"] = options.TopP
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/generate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResponse struct {
		Results []struct {
			Text string `json:"text"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(apiResponse.Results) == 0 {
		return nil, fmt.Errorf("no results in response")
	}

	duration := time.Since(start)

	return &Response{
		Content:      apiResponse.Results[0].Text,
		FinishReason: "stop",
		Duration:     duration,
	}, nil
}

// IsAvailable checks if the API is available
func (c *HTTPClient) IsAvailable(ctx context.Context) bool {
	// Simple health check - try to make a minimal request
	var url string
	var req *http.Request
	var err error

	switch c.provider {
	case "lmstudio", "localai", "openai-compat", "vllm":
		// OpenAI-compatible: check /models or /v1/models endpoint
		url = fmt.Sprintf("%s/models", c.baseURL)
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return false
		}
		if c.apiKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}

	case "llama-cpp":
		// llama.cpp: check /health endpoint
		url = fmt.Sprintf("%s/health", c.baseURL)
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return false
		}

	case "textgen-webui":
		// Text Generation WebUI: check /api/v1/model endpoint
		url = fmt.Sprintf("%s/api/v1/model", c.baseURL)
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return false
		}

	default:
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
