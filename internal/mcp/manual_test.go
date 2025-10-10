package mcp

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManualMCPCommunication(t *testing.T) {
	// This test manually sends JSON-RPC messages to local-memory to verify it works
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	
	// Start the local-memory process
	cmd := exec.CommandContext(ctx, "local-memory", "--mcp")
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin: %v", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout: %v", err)
	}
	
	err = cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}
	
	defer func() {
		stdin.Close()
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	
	// Send initialize request
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test",
				"version": "1.0",
			},
		},
	}
	
	// Send the request
	requestBytes, err := json.Marshal(initRequest)
	assert.NoError(t, err)
	
	t.Logf("Sending: %s", string(requestBytes))
	
	_, err = stdin.Write(append(requestBytes, '\n'))
	assert.NoError(t, err)
	
	// Read response
	response := make([]byte, 4096)
	n, err := stdout.Read(response)
	if err != nil {
		t.Logf("Read error: %v", err)
	} else {
		t.Logf("Received: %s", string(response[:n]))
		
		// Parse the response
		var respObj map[string]interface{}
		err = json.Unmarshal(response[:n], &respObj)
		assert.NoError(t, err)
		
		// Verify it's a valid response
		assert.Equal(t, "2.0", respObj["jsonrpc"])
		assert.Equal(t, float64(1), respObj["id"])
		assert.Contains(t, respObj, "result")
	}
}