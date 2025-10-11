package mcp

import (
	"testing"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	logger := NewSimpleLogger()

	tests := []struct {
		name          string
		server        Server
		expectedType  string
		expectError   bool
	}{
		{
			name: "stdio client",
			server: Server{
				Name:      "test-stdio",
				Transport: "stdio",
				Command:   []string{"echo"},
				Timeout:   time.Second * 30,
			},
			expectedType: "*mcp.STDIOClient",
			expectError:  false,
		},
		{
			name: "http client",
			server: Server{
				Name:      "test-http",
				Transport: "http",
				URL:       "http://localhost:8080/mcp",
				Timeout:   time.Second * 30,
			},
			expectedType: "*mcp.HTTPClient",
			expectError:  false,
		},
		{
			name: "unsupported transport",
			server: Server{
				Name:      "test-unsupported",
				Transport: "websocket",
				Timeout:   time.Second * 30,
			},
			expectedType: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.server, logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), "unsupported transport type")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				
				// Verify correct type is returned
				switch tt.expectedType {
				case "*mcp.STDIOClient":
					_, ok := client.(*STDIOClient)
					assert.True(t, ok, "Expected STDIOClient but got %T", client)
				case "*mcp.HTTPClient":
					_, ok := client.(*HTTPClient)
					assert.True(t, ok, "Expected HTTPClient but got %T", client)
				}
			}
		})
	}
}

func TestDefaultClientFactory(t *testing.T) {
	logger := NewSimpleLogger()
	factory := NewClientFactory(logger)

	serverCfg := config.ServerConfig{
		Name:      "test-factory",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{},
	}

	client, err := factory.CreateClient(serverCfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify it creates a STDIO client
	_, ok := client.(*STDIOClient)
	assert.True(t, ok)
}

func TestClientFactoryWithHTTP(t *testing.T) {
	logger := NewSimpleLogger()
	factory := NewClientFactory(logger)

	serverCfg := config.ServerConfig{
		Name:      "test-http-factory",
		Transport: "http",
		Command:   "http://localhost:8080/mcp", // For HTTP, the command might be the URL
	}

	client, err := factory.CreateClient(serverCfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify it creates an HTTP client
	_, ok := client.(*HTTPClient)
	assert.True(t, ok)
}