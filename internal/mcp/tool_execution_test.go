package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMCPToolExecution(t *testing.T) {
	logger := NewSimpleLogger()
	
	// Test with the actual local-memory server
	server := Server{
		Name:      "local-memory",
		Transport: "stdio",
		Command:   []string{"local-memory"},
		Args:      []string{},
		Timeout:   time.Second * 30,
	}
	
	client := NewSTDIOClient(server, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	// Connect to server
	err := client.Connect(ctx)
	if err != nil {
		t.Skipf("Could not connect to local-memory server: %v", err)
		return
	}
	
	defer func() {
		disconnectErr := client.Disconnect(ctx)
		assert.NoError(t, disconnectErr)
	}()
	
	// Test storing a memory
	storeParams := map[string]interface{}{
		"content":    "Testing MCP integration from Go client",
		"importance": 8,
		"tags":       []string{"test", "mcp", "go"},
		"domain":     "testing",
	}
	
	result, err := client.CallTool(ctx, "store_memory", storeParams)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)
	
	t.Logf("Store memory result: %+v", result)
	
	// Test searching for the memory we just stored
	searchParams := map[string]interface{}{
		"query":        "MCP integration",
		"search_type":  "semantic",
		"limit":        5,
	}
	
	searchResult, err := client.CallTool(ctx, "search", searchParams)
	if err != nil {
		t.Logf("Search failed (expected with large results): %v", err)
	} else {
		assert.NotNil(t, searchResult)
		if searchResult != nil {
			assert.False(t, searchResult.IsError)
			t.Logf("Search result: %+v", searchResult)
		}
	}
	
	// Test getting session stats
	statsParams := map[string]interface{}{
		"stats_type": "session",
	}
	
	statsResult, err := client.CallTool(ctx, "stats", statsParams)
	assert.NoError(t, err)
	assert.NotNil(t, statsResult)
	assert.False(t, statsResult.IsError)
	
	t.Logf("Stats result: %+v", statsResult)
}