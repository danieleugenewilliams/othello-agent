package mcp

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

// mockHTTPServer creates a test HTTP server that implements MCP over HTTP
func createMockHTTPServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	
	// Track session state
	sessions := make(map[string]bool)
	
	// Initialize endpoint
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.Header.Get("Mcp-Session-Id")
		protocolVersion := r.Header.Get("Mcp-Protocol-Version")
		
		// Validate protocol version
		if protocolVersion == "" {
			http.Error(w, "Missing Mcp-Protocol-Version header", http.StatusBadRequest)
			return
		}
		
		switch r.Method {
		case http.MethodPost:
			var req Message
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			
			// Generate session ID for new sessions
			if sessionID == "" && req.Method == "initialize" {
				sessionID = "test-session-123"
				w.Header().Set("Mcp-Session-Id", sessionID)
				sessions[sessionID] = true
			}
			
			// Handle different request methods
			switch req.Method {
			case "initialize":
				resp := Message{
					ID: req.ID,
					Result: map[string]interface{}{
						"protocolVersion": "2024-11-05",
						"capabilities": map[string]interface{}{
							"tools": map[string]interface{}{},
						},
						"serverInfo": map[string]interface{}{
							"name":    "test-server",
							"version": "1.0.0",
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
				
			case "tools/list":
				resp := Message{
					ID: req.ID,
					Result: map[string]interface{}{
						"tools": []map[string]interface{}{
							{
								"name":        "test-tool",
								"description": "A test tool",
								"inputSchema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"input": map[string]interface{}{
											"type": "string",
										},
									},
								},
							},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
				
			case "tools/call":
				params := req.Params.(map[string]interface{})
				toolName := params["name"].(string)
				
				if toolName == "test-tool" {
					resp := Message{
						ID: req.ID,
						Result: map[string]interface{}{
							"content": []map[string]interface{}{
								{
									"type": "text",
									"text": "Hello from test tool",
								},
							},
						},
					}
					json.NewEncoder(w).Encode(resp)
				} else {
					resp := Message{
						ID: req.ID,
						Error: &Error{
							Code:    ErrorMethodNotFound,
							Message: "Tool not found",
						},
					}
					json.NewEncoder(w).Encode(resp)
				}
				
			case "ping":
				resp := Message{
					ID: req.ID,
					Result: map[string]interface{}{
						"status": "ok",
					},
				}
				json.NewEncoder(w).Encode(resp)
				
			default:
				resp := Message{
					ID: req.ID,
					Error: &Error{
						Code:    ErrorMethodNotFound,
						Message: "Method not found",
					},
				}
				json.NewEncoder(w).Encode(resp)
			}
			
		case http.MethodDelete:
			if sessionID == "" {
				http.Error(w, "Missing session ID", http.StatusBadRequest)
				return
			}
			
			if !sessions[sessionID] {
				http.Error(w, "Session not found", http.StatusNotFound)
				return
			}
			
			delete(sessions, sessionID)
			w.WriteHeader(http.StatusNoContent)
			
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	return httptest.NewServer(mux)
}

func TestNewHTTPClient(t *testing.T) {
	logger := NewSimpleLogger()
	
	server := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       "http://localhost:8080/mcp",
		Timeout:   time.Second * 30,
	}
	
	client := NewHTTPClient(server, logger)
	
	assert.NotNil(t, client)
	assert.Equal(t, server, client.server)
	assert.Equal(t, logger, client.logger)
	assert.NotNil(t, client.httpClient)
	assert.False(t, client.IsConnected())
}

func TestHTTPClientConnect(t *testing.T) {
	server := createMockHTTPServer(t)
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	err := client.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, client.IsConnected())
	assert.NotEmpty(t, client.sessionID)
	
	// Test double connect (should be no-op)
	err = client.Connect(ctx)
	assert.NoError(t, err)
}

func TestHTTPClientConnectInvalidURL(t *testing.T) {
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       "http://invalid-url:99999/mcp",
		Timeout:   time.Second * 1,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	
	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.False(t, client.IsConnected())
}

func TestHTTPClientListTools(t *testing.T) {
	server := createMockHTTPServer(t)
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	err := client.Connect(ctx)
	require.NoError(t, err)
	
	tools, err := client.ListTools(ctx)
	assert.NoError(t, err)
	assert.Len(t, tools, 1)
	assert.Equal(t, "test-tool", tools[0].Name)
	assert.Equal(t, "A test tool", tools[0].Description)
	assert.Equal(t, serverConfig.Name, tools[0].ServerName)
}

func TestHTTPClientListToolsNotConnected(t *testing.T) {
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       "http://localhost:8080/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx := context.Background()
	
	_, err := client.ListTools(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestHTTPClientCallTool(t *testing.T) {
	server := createMockHTTPServer(t)
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	err := client.Connect(ctx)
	require.NoError(t, err)
	
	// Test successful tool call
	params := map[string]interface{}{
		"input": "test input",
	}
	
	result, err := client.CallTool(ctx, "test-tool", params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "Hello from test tool", result.Content[0].Text)
}

func TestHTTPClientCallToolNotFound(t *testing.T) {
	server := createMockHTTPServer(t)
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	err := client.Connect(ctx)
	require.NoError(t, err)
	
	// Test tool not found
	params := map[string]interface{}{
		"input": "test input",
	}
	
	result, err := client.CallTool(ctx, "nonexistent-tool", params)
	assert.NoError(t, err) // Should not error, but result should indicate error
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestHTTPClientCallToolNotConnected(t *testing.T) {
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       "http://localhost:8080/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx := context.Background()
	
	params := map[string]interface{}{
		"input": "test input",
	}
	
	_, err := client.CallTool(ctx, "test-tool", params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestHTTPClientGetInfo(t *testing.T) {
	server := createMockHTTPServer(t)
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	err := client.Connect(ctx)
	require.NoError(t, err)
	
	info, err := client.GetInfo(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, serverConfig.Name, info.Name)
	assert.True(t, info.Capabilities.Tools)
}

func TestHTTPClientDisconnect(t *testing.T) {
	server := createMockHTTPServer(t)
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-http-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	err := client.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, client.IsConnected())
	
	err = client.Disconnect()
	assert.NoError(t, err)
	assert.False(t, client.IsConnected())
	
	// Test double disconnect (should be safe)
	err = client.Disconnect()
	assert.NoError(t, err)
}

func TestHTTPClientWithAuthentication(t *testing.T) {
	// Create server that requires authentication
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		// Simple echo for valid auth
		w.Header().Set("Content-Type", "application/json")
		resp := Message{
			ID: 1,
			Result: map[string]interface{}{
				"status": "authenticated",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	// Test with valid auth header
	serverConfig := Server{
		Name:      "test-auth-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
		Timeout: time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	// This should work with proper authentication
	err := client.Connect(ctx)
	assert.NoError(t, err)
}

func TestHTTPClientTimeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-timeout-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Millisecond * 500, // Very short timeout
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	
	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func TestHTTPClientRequestID(t *testing.T) {
	requestIDs := make([]interface{}, 0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Message
		json.NewDecoder(r.Body).Decode(&req)
		requestIDs = append(requestIDs, req.ID)
		
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "test-session")
		
		resp := Message{
			ID: req.ID,
			Result: map[string]interface{}{
				"status": "ok",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()
	
	logger := NewSimpleLogger()
	
	serverConfig := Server{
		Name:      "test-id-server",
		Transport: "http",
		URL:       server.URL + "/mcp",
		Timeout:   time.Second * 5,
	}
	
	client := NewHTTPClient(serverConfig, logger)
	
	ctx := context.Background()
	
	// Make multiple requests
	client.Connect(ctx)
	client.GetInfo(ctx)
	client.GetInfo(ctx)
	
	// Check that request IDs are unique and sequential
	assert.Len(t, requestIDs, 3)
	assert.NotEqual(t, requestIDs[0], requestIDs[1])
	assert.NotEqual(t, requestIDs[1], requestIDs[2])
}