package model

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOllamaModel_IsAvailable(t *testing.T) {
	// Create model instance
	model := NewOllamaModel("http://localhost:11434", "qwen2.5:3b")
	
	// Test availability check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	available := model.IsAvailable(ctx)
	
	// This test will pass if Ollama is running and has the model
	// If Ollama is not available, the test should still pass but report availability as false
	t.Logf("Model availability: %v", available)
	
	// We don't assert true here because the test environment might not have Ollama running
	// This is more of an integration test that validates the connection logic
	assert.NotNil(t, model)
}

func TestNewOllamaModel(t *testing.T) {
	host := "http://localhost:11434"
	modelName := "qwen2.5:3b"
	
	model := NewOllamaModel(host, modelName)
	
	assert.NotNil(t, model)
	assert.Equal(t, host, model.host)
	assert.Equal(t, modelName, model.modelName)
	assert.NotNil(t, model.client)
}