package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDirectSTDIOCommunication(t *testing.T) {
	// Test our STDIO implementation step by step
	
	logger := NewSimpleLogger()
	
	// Set up command exactly like in STDIOClient
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "local-memory", "--mcp")
	
	stdin, err := cmd.StdinPipe()
	assert.NoError(t, err)
	
	stdout, err := cmd.StdoutPipe()
	assert.NoError(t, err)
	
	stderr, err := cmd.StderrPipe()
	assert.NoError(t, err)
	
	// Start the process
	err = cmd.Start()
	assert.NoError(t, err)
	
	defer func() {
		stdin.Close()
		stdout.Close()  
		stderr.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	
	logger.Info("Process started with PID: %d", cmd.Process.Pid)
	
	// Start reading responses like STDIOClient does
	responses := make(chan Message, 10)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Info("Received line: %s", line)
			
			if line == "" {
				continue
			}
			
			var msg Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				logger.Error("Failed to unmarshal: %v", err)
				continue
			}
			
			responses <- msg
		}
		
		if err := scanner.Err(); err != nil {
			logger.Error("Scanner error: %v", err)
		}
		close(responses)
	}()
	
	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				logger.Error("Server stderr: %s", line)
			}
		}
	}()
	
	// Give the goroutines a moment to start
	time.Sleep(time.Millisecond * 100)
	
	// Send initialize request
	initMsg := Message{
		ID:     1,
		Method: "initialize",
		Params: map[string]interface{}{
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
	
	data, err := json.Marshal(initMsg)
	assert.NoError(t, err)
	
	logger.Info("Sending: %s", string(data))
	
	data = append(data, '\n')
	_, err = stdin.Write(data)
	assert.NoError(t, err)
	
	// Wait for response
	select {
	case response := <-responses:
		logger.Info("Got response: %+v", response)
		// JSON unmarshaling makes numbers float64 by default
		assert.Equal(t, float64(1), response.ID)
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)
	case <-time.After(time.Second * 3):
		t.Fatal("Timeout waiting for response")
	case <-ctx.Done():
		t.Fatal("Context cancelled")
	}
}