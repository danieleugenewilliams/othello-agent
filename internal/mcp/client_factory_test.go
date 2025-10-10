package mcp

import (
	"testing"
	"time"

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
	factory := &DefaultClientFactory{}
	logger := NewSimpleLogger()

	server := Server{
		Name:      "test-factory",
		Transport: "stdio",
		Command:   []string{"echo"},
		Timeout:   time.Second * 30,
	}

	client, err := factory.CreateClient(server, logger)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify it creates a STDIO client
	_, ok := client.(*STDIOClient)
	assert.True(t, ok)
}

func TestClientFactoryWithHTTP(t *testing.T) {
	factory := &DefaultClientFactory{}
	logger := NewSimpleLogger()

	server := Server{
		Name:      "test-http-factory",
		Transport: "http",
		URL:       "http://localhost:8080/mcp",
		Timeout:   time.Second * 30,
	}

	client, err := factory.CreateClient(server, logger)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify it creates an HTTP client
	_, ok := client.(*HTTPClient)
	assert.True(t, ok)
}