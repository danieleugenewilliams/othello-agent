package mcp

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// SimpleLogger implements the Logger interface for testing
type SimpleLogger struct {
	*log.Logger
}

func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{
		Logger: log.New(os.Stdout, "[MCP-TEST] ", log.LstdFlags),
	}
}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	l.Printf("INFO: "+msg, args...)
}

func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	l.Printf("ERROR: "+msg, args...)
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	l.Printf("DEBUG: "+msg, args...)
}

func TestNewSTDIOClient(t *testing.T) {
	logger := NewSimpleLogger()
	
	server := Server{
		Name:      "test-server",
		Transport: "stdio",
		Command:   []string{"echo"},
		Args:      []string{"hello"},
		Timeout:   time.Second * 30,
	}
	
	client := NewSTDIOClient(server, logger)
	
	assert.NotNil(t, client)
	assert.Equal(t, server, client.server)
	assert.Equal(t, logger, client.logger)
	assert.NotNil(t, client.responses)
	assert.False(t, client.IsConnected())
}

func TestSTDIOClient_ConnectWithLocalMemory(t *testing.T) {
	logger := NewSimpleLogger()
	
	// Test with the actual local-memory server (without --mcp flag)
	server := Server{
		Name:      "local-memory",
		Transport: "stdio",
		Command:   []string{"local-memory"},
		Args:      []string{}, // No --mcp flag needed
		Timeout:   time.Second * 30,
	}
	
	client := NewSTDIOClient(server, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	// Try to connect - this will test if local-memory is available
	err := client.Connect(ctx)
	if err != nil {
		t.Logf("Could not connect to local-memory server: %v", err)
		t.Skip("local-memory server not available, skipping integration test")
		return
	}
	
	// Verify connection
	assert.True(t, client.IsConnected())
	
	// Clean up
	defer func() {
		disconnectErr := client.Disconnect(ctx)
		assert.NoError(t, disconnectErr)
	}()
	
	// Test getting server info
	info, err := client.GetInfo(ctx)
	if err != nil {
		t.Logf("GetInfo failed: %v", err)
	} else {
		t.Logf("Server info: %+v", info)
		assert.NotEmpty(t, info.Name)
	}
	
	// Test listing tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Logf("ListTools failed: %v", err)
	} else {
		t.Logf("Found %d tools", len(tools))
		for _, tool := range tools {
			t.Logf("Tool: %s - %s", tool.Name, tool.Description)
		}
	}
}

func TestSTDIOClient_ConnectWithInvalidCommand(t *testing.T) {
	logger := NewSimpleLogger()
	
	server := Server{
		Name:      "invalid-server",
		Transport: "stdio",
		Command:   []string{"nonexistent-command-12345"},
		Timeout:   time.Second * 5,
	}
	
	client := NewSTDIOClient(server, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	
	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.False(t, client.IsConnected())
}

func TestSTDIOClient_ConnectWithoutCommand(t *testing.T) {
	logger := NewSimpleLogger()
	
	server := Server{
		Name:      "test-server",
		Transport: "stdio",
		Command:   []string{}, // Empty command should fail
		Timeout:   time.Second * 30,
	}
	
	client := NewSTDIOClient(server, logger)
	
	ctx := context.Background()
	err := client.Connect(ctx)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no command specified")
	assert.False(t, client.IsConnected())
}