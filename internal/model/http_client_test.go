package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test HTTP API client for open source model backends (LM Studio, LocalAI, llama.cpp, vLLM, Text Generation WebUI)

func TestNewHTTPClient(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		apiKey    string
		provider  string
		wantError bool
	}{
		{
			name:      "valid LM Studio client",
			baseURL:   "http://localhost:1234/v1",
			apiKey:    "",
			provider:  "lmstudio",
			wantError: false,
		},
		{
			name:      "valid LocalAI client",
			baseURL:   "http://localhost:8080/v1",
			apiKey:    "",
			provider:  "localai",
			wantError: false,
		},
		{
			name:      "valid llama.cpp client",
			baseURL:   "http://localhost:8080",
			apiKey:    "",
			provider:  "llama-cpp",
			wantError: false,
		},
		{
			name:      "valid vLLM client",
			baseURL:   "http://localhost:8000/v1",
			apiKey:    "",
			provider:  "vllm",
			wantError: false,
		},
		{
			name:      "valid Text Generation WebUI client",
			baseURL:   "http://localhost:5000",
			apiKey:    "",
			provider:  "textgen-webui",
			wantError: false,
		},
		{
			name:      "empty base URL",
			baseURL:   "",
			apiKey:    "",
			provider:  "lmstudio",
			wantError: true,
		},
		{
			name:      "unsupported provider",
			baseURL:   "http://localhost:8080",
			apiKey:    "",
			provider:  "unknown",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHTTPClient(tt.baseURL, tt.apiKey, tt.provider)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestHTTPClient_Generate_LMStudio(t *testing.T) {
	// Mock LM Studio API server (OpenAI-compatible)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "local-model",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "This is a test response from LM Studio",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL+"/v1", "", "lmstudio")
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Generate(ctx, "Test prompt", GenerateOptions{
		Temperature: 0.7,
		MaxTokens:   100,
	})

	require.NoError(t, err)
	assert.Equal(t, "This is a test response from LM Studio", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 20, resp.Usage.CompletionTokens)
	assert.Equal(t, 30, resp.Usage.TotalTokens)
	assert.Greater(t, resp.Duration, time.Duration(0))
}

func TestHTTPClient_Chat_LocalAI(t *testing.T) {
	// Mock LocalAI API server (OpenAI-compatible)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		// Verify request structure
		messages := requestBody["messages"].([]interface{})
		assert.Equal(t, 2, len(messages))
		
		response := map[string]interface{}{
			"id":      "chatcmpl-456",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "local-model",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Multi-turn response from LocalAI",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     20,
				"completion_tokens": 10,
				"total_tokens":      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL+"/v1", "", "localai")
	require.NoError(t, err)

	ctx := context.Background()
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}

	resp, err := client.Chat(ctx, messages, GenerateOptions{Temperature: 0.7})
	require.NoError(t, err)
	assert.Equal(t, "Multi-turn response from LocalAI", resp.Content)
}

func TestHTTPClient_Generate_LlamaCpp(t *testing.T) {
	// Mock llama.cpp HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/completion", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		// llama.cpp expects "prompt" field, not "messages"
		assert.Contains(t, requestBody, "prompt")
		
		response := map[string]interface{}{
			"content": "This is a test response from llama.cpp",
			"stop":    true,
			"tokens_predicted": 25,
			"tokens_evaluated": 15,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, "", "llama-cpp")
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := client.Generate(ctx, "Test prompt", GenerateOptions{
		Temperature: 0.7,
		MaxTokens:   100,
	})

	require.NoError(t, err)
	assert.Equal(t, "This is a test response from llama.cpp", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	// llama.cpp doesn't provide token usage in the same format, so these may be 0
	assert.Equal(t, 0, resp.Usage.PromptTokens)  // llama.cpp response doesn't have exact token counts
	assert.Equal(t, 0, resp.Usage.CompletionTokens)
	assert.Equal(t, 0, resp.Usage.TotalTokens)
}

func TestHTTPClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   map[string]interface{}
		wantError  bool
	}{
		{
			name:       "rate limit error",
			statusCode: http.StatusTooManyRequests,
			response: map[string]interface{}{
				"error": map[string]string{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
				},
			},
			wantError: true,
		},
		{
			name:       "authentication error",
			statusCode: http.StatusUnauthorized,
			response: map[string]interface{}{
				"error": map[string]string{
					"message": "Invalid API key",
					"type":    "authentication_error",
				},
			},
			wantError: true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response: map[string]interface{}{
				"error": "Internal server error",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client, err := NewHTTPClient(server.URL+"/v1", "", "lmstudio")
			require.NoError(t, err)

			ctx := context.Background()
			_, err = client.Generate(ctx, "Test", GenerateOptions{})

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_ContextCancellation(t *testing.T) {
	// Slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Response"}},
			},
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL+"/v1", "", "vllm")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = client.Generate(ctx, "Test", GenerateOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestHTTPClient_IsAvailable(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		wantAvailable bool
	}{
		{
			name:         "server available",
			statusCode:   http.StatusOK,
			wantAvailable: true,
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			wantAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client, err := NewHTTPClient(server.URL+"/v1", "", "localai")
			require.NoError(t, err)

			ctx := context.Background()
			available := client.IsAvailable(ctx)
			assert.Equal(t, tt.wantAvailable, available)
		})
	}
}

func TestHTTPClient_RequestOptions(t *testing.T) {
	// Test Text Generation WebUI API format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		// Text Generation WebUI uses different parameter names and formats prompts
		assert.Equal(t, "user: Test\n", requestBody["prompt"])  // Our implementation formats prompts this way
		assert.Equal(t, 0.8, requestBody["temperature"])
		assert.Equal(t, float64(150), requestBody["max_new_tokens"])
		assert.Equal(t, 0.9, requestBody["top_p"])

		// Text Generation WebUI response format
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"text": "Response from Text Generation WebUI",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, "", "textgen-webui")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = client.Generate(ctx, "Test", GenerateOptions{
		Temperature: 0.8,
		MaxTokens:   150,
		TopP:        0.9,
	})

	require.NoError(t, err)
}
