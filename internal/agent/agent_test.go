package agent

import (
	"context"
	"testing"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	cfg := &config.Config{
		Model: config.ModelConfig{
			Type:          "ollama",
			Name:          "test-model",
			Temperature:   0.7,
			MaxTokens:     2048,
			ContextLength: 8192,
		},
		Ollama: config.OllamaConfig{
			Host:    "http://localhost:11434",
			Timeout: 30 * time.Second,
		},
		TUI: config.TUIConfig{
			Theme:      "default",
			ShowHints:  true,
			AutoScroll: true,
		},
		Storage: config.StorageConfig{
			HistorySize: 1000,
			CacheTTL:    time.Hour,
			DataDir:     "/tmp/test",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			File:   "/tmp/test.log",
			Format: "text",
		},
	}

	agent, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, agent)

	assert.Equal(t, cfg, agent.config)
	assert.NotNil(t, agent.logger)
}

func TestNewAgentNilConfig(t *testing.T) {
	agent, err := New(nil)
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "configuration cannot be nil")
}

func TestAgentStartStop(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	agent, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Test start
	err = agent.Start(ctx)
	assert.NoError(t, err)

	// Test stop
	err = agent.Stop(ctx)
	assert.NoError(t, err)
}

func TestAgentGetStatus(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	agent, err := New(cfg)
	require.NoError(t, err)

	status := agent.GetStatus()
	require.NotNil(t, status)

	assert.True(t, status.Running)
	assert.NotEmpty(t, status.ConfigFile)
	assert.False(t, status.ModelConnected) // Not yet implemented
	assert.Equal(t, 0, status.MCPServers)  // No servers configured by default
}

func TestAgentStartTUI(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	agent, err := New(cfg)
	require.NoError(t, err)

	// TUI start should not error (even though it's not fully implemented)
	err = agent.StartTUI()
	assert.NoError(t, err)
}