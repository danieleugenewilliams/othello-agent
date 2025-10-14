package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// MockModel implements the model.Model interface for testing
type MockModel struct {
	responses map[string]*model.Response
}

func NewMockModel() *MockModel {
	return &MockModel{
		responses: make(map[string]*model.Response),
	}
}

func (m *MockModel) Generate(ctx context.Context, prompt string, options model.GenerateOptions) (*model.Response, error) {
	return &model.Response{
		Content: "Mock response for: " + prompt,
	}, nil
}

func (m *MockModel) Chat(ctx context.Context, messages []model.Message, options model.GenerateOptions) (*model.Response, error) {
	lastMessage := ""
	if len(messages) > 0 {
		lastMessage = messages[len(messages)-1].Content
	}

	return &model.Response{
		Content: "Mock chat response for: " + lastMessage,
	}, nil
}

func (m *MockModel) ChatWithTools(ctx context.Context, messages []model.Message, tools []model.ToolDefinition, options model.GenerateOptions) (*model.Response, error) {
	lastMessage := ""
	if len(messages) > 0 {
		lastMessage = messages[len(messages)-1].Content
	}

	// Simulate tool usage for specific patterns
	if len(tools) > 0 {
		return &model.Response{
			Content: "TOOL_CALL: " + tools[0].Name + "\nARGUMENTS: {\"query\": \"test\"}",
			ToolCalls: []model.ToolCall{
				{
					Name:      tools[0].Name,
					Arguments: map[string]interface{}{"query": "test"},
				},
			},
		}, nil
	}

	return &model.Response{
		Content: "Mock chat with tools response for: " + lastMessage,
	}, nil
}

func (m *MockModel) IsAvailable(ctx context.Context) bool {
	return true
}

// MockClient implements the mcp.Client interface for testing
type MockClient struct {
	tools []mcp.Tool
}

func NewMockClient() *MockClient {
	return &MockClient{
		tools: []mcp.Tool{
			{
				Name:        "search",
				Description: "Search for information",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
						"search_type": map[string]interface{}{
							"type": "string",
							"enum": []interface{}{"semantic", "keyword"},
						},
					},
					"required": []interface{}{"query"},
				},
			},
			{
				Name:        "store_memory",
				Description: "Store information in memory",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Content to store",
						},
						"importance": map[string]interface{}{
							"type":    "integer",
							"minimum": 1,
							"maximum": 10,
						},
					},
					"required": []interface{}{"content"},
				},
			},
		},
	}
}

func (c *MockClient) Connect(ctx context.Context) error {
	return nil
}

func (c *MockClient) Disconnect(ctx context.Context) error {
	return nil
}

func (c *MockClient) IsConnected() bool {
	return true
}

func (c *MockClient) GetTransport() string {
	return "mock"
}

func (c *MockClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return c.tools, nil
}

func (c *MockClient) CallTool(ctx context.Context, name string, params map[string]interface{}) (*mcp.ToolResult, error) {
	// Mock tool execution
	return &mcp.ToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: "Mock result for tool: " + name,
			},
		},
	}, nil
}

func (c *MockClient) GetInfo(ctx context.Context) (*mcp.ServerInfo, error) {
	return &mcp.ServerInfo{
		Name:    "mock-server",
		Version: "1.0.0",
	}, nil
}

// MockLogger implements the mcp.Logger interface
type MockLogger struct{}

func (l *MockLogger) Info(msg string, args ...interface{})  {}
func (l *MockLogger) Error(msg string, args ...interface{}) {}
func (l *MockLogger) Debug(msg string, args ...interface{}) {}

func TestUniversalAgentIntegration(t *testing.T) {
	// Setup mock components
	logger := &MockLogger{}
	registry := mcp.NewToolRegistry(logger)
	mockClient := NewMockClient()
	mockModel := NewMockModel()

	// Register mock MCP server
	err := registry.RegisterServer("mock-server", mockClient)
	if err != nil {
		t.Fatalf("Failed to register mock server: %v", err)
	}

	// Create universal integration
	integration := NewUniversalAgentIntegration(registry, mockModel, logger)

	ctx := context.Background()

	t.Run("Test Intent Classification", func(t *testing.T) {
		testCases := []struct {
			input           string
			expectedIntent  string
			minConfidence   float64
		}{
			{"search for python tutorials", "search", 0.5},
			{"store this information", "create", 0.5},
			{"nice weather today", "conversation", 0.0},
			{"analyze my data", "analyze", 0.5},
		}

		for _, tc := range testCases {
			analysis, err := integration.AnalyzeUserIntent(ctx, tc.input)
			if err != nil {
				t.Errorf("Failed to analyze intent for '%s': %v", tc.input, err)
				continue
			}

			if analysis.Intent != tc.expectedIntent {
				t.Errorf("Expected intent '%s' for input '%s', got '%s'",
					tc.expectedIntent, tc.input, analysis.Intent)
			}

			if analysis.Confidence < tc.minConfidence {
				t.Errorf("Expected confidence >= %.2f for input '%s', got %.2f",
					tc.minConfidence, tc.input, analysis.Confidence)
			}
		}
	})

	t.Run("Test Tool Discovery", func(t *testing.T) {
		discovery := integration.discovery
		tools, err := discovery.DiscoverAllTools(ctx)
		if err != nil {
			t.Fatalf("Failed to discover tools: %v", err)
		}

		if len(tools) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(tools))
		}

		// Check that tools are categorized correctly
		searchFound := false
		createFound := false
		for _, tool := range tools {
			if tool.Tool.Name == "search" && tool.Capability == CapabilitySearch {
				searchFound = true
			}
			if tool.Tool.Name == "store_memory" && tool.Capability == CapabilityCreate {
				createFound = true
			}
		}

		if !searchFound {
			t.Error("Search tool not found or incorrectly categorized")
		}
		if !createFound {
			t.Error("Store memory tool not found or incorrectly categorized")
		}
	})

	t.Run("Test System Prompt Generation", func(t *testing.T) {
		promptContext := PromptContext{
			UserQuery:   "search for something",
			SessionType: "chat",
		}

		prompt, err := integration.promptGen.GenerateToolPrompt(ctx, promptContext)
		if err != nil {
			t.Fatalf("Failed to generate system prompt: %v", err)
		}

		if prompt == "" {
			t.Error("Generated prompt is empty")
		}

		// Check that prompt contains expected elements
		expectedElements := []string{
			"TOOL_CALL:",
			"ARGUMENTS:",
			"search",
			"store_memory",
		}

		for _, element := range expectedElements {
			if !strings.Contains(prompt, element) {
				t.Errorf("Generated prompt missing expected element: %s", element)
			}
		}
	})

	t.Run("Test User Request Processing", func(t *testing.T) {
		testCases := []struct {
			input        string
			expectTools  bool
			responseType string
		}{
			{"search for python tutorials", true, "single_tool"},
			{"hello world", false, "conversation"},
			{"store this and then search for it", true, "orchestration"},
		}

		for _, tc := range testCases {
			response, err := integration.ProcessUserRequest(
				ctx,
				tc.input,
				[]model.Message{{Role: "user", Content: tc.input}},
				"chat",
			)

			if err != nil && tc.expectTools {
				t.Errorf("Failed to process request '%s': %v", tc.input, err)
				continue
			}

			if !response.Success {
				t.Errorf("Request processing failed for '%s': %s", tc.input, response.Error)
				continue
			}

			if tc.expectTools && len(response.ToolSuggestions) == 0 {
				t.Errorf("Expected tool suggestions for '%s', got none", tc.input)
			}

			if response.FinalResponse == "" {
				t.Errorf("Expected non-empty final response for '%s'", tc.input)
			}

			if len(response.ProcessingSteps) == 0 {
				t.Errorf("Expected processing steps for '%s'", tc.input)
			}
		}
	})

	t.Run("Test Tool Capability Summary", func(t *testing.T) {
		summary, err := integration.GetToolCapabilitySummary(ctx)
		if err != nil {
			t.Fatalf("Failed to get tool capability summary: %v", err)
		}

		if len(summary) == 0 {
			t.Error("Expected non-empty capability summary")
		}

		// Should have search and creation capabilities
		if count, exists := summary["Search & Retrieval"]; !exists || count == 0 {
			t.Error("Expected search capability in summary")
		}
		if count, exists := summary["Creation & Storage"]; !exists || count == 0 {
			t.Error("Expected creation capability in summary")
		}
	})
}

func TestToolOrchestration(t *testing.T) {
	// Setup
	logger := &MockLogger{}
	registry := mcp.NewToolRegistry(logger)
	mockClient := NewMockClient()

	err := registry.RegisterServer("mock-server", mockClient)
	if err != nil {
		t.Fatalf("Failed to register mock server: %v", err)
	}

	executor := mcp.NewToolExecutor(registry, logger)
	discovery := NewToolDiscovery(registry, logger)
	classifier := NewIntentClassifier(discovery, logger)
	orchestrator := NewToolOrchestrator(executor, classifier, discovery, logger)

	ctx := context.Background()

	t.Run("Test Complex Request Orchestration", func(t *testing.T) {
		sessionContext := map[string]interface{}{
			"sessionType": "chat",
		}

		result, err := orchestrator.OrchestrateTasks(
			ctx,
			"search for python and then store what you find",
			sessionContext,
		)

		if err != nil {
			t.Fatalf("Orchestration failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Expected successful orchestration, got error: %s", result.Error)
		}

		if len(result.ToolResults) == 0 {
			t.Error("Expected tool results from orchestration")
		}

		if result.PrimaryResult == "" {
			t.Error("Expected non-empty primary result")
		}
	})
}

// Benchmark tests
func BenchmarkIntentClassification(b *testing.B) {
	logger := &MockLogger{}
	registry := mcp.NewToolRegistry(logger)
	mockClient := NewMockClient()
	registry.RegisterServer("mock-server", mockClient)

	discovery := NewToolDiscovery(registry, logger)
	classifier := NewIntentClassifier(discovery, logger)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = classifier.ClassifyIntent(ctx, "search for something interesting")
	}
}

func BenchmarkToolSuggestion(b *testing.B) {
	logger := &MockLogger{}
	registry := mcp.NewToolRegistry(logger)
	mockClient := NewMockClient()
	registry.RegisterServer("mock-server", mockClient)

	discovery := NewToolDiscovery(registry, logger)
	classifier := NewIntentClassifier(discovery, logger)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = classifier.SuggestTools(ctx, "search for python tutorials")
	}
}