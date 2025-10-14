package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// EnhancedModel wraps a regular model with intelligent tool integration
type EnhancedModel struct {
	baseModel       model.Model
	promptGenerator *SystemPromptGenerator
	toolDiscovery   *ToolDiscovery
	registry        *mcp.ToolRegistry
	logger          mcp.Logger
}

// NewEnhancedModel creates a new enhanced model with tool integration
func NewEnhancedModel(baseModel model.Model, registry *mcp.ToolRegistry, logger mcp.Logger) *EnhancedModel {
	discovery := NewToolDiscovery(registry, logger)
	promptGenerator := NewSystemPromptGenerator(discovery, logger)

	return &EnhancedModel{
		baseModel:       baseModel,
		promptGenerator: promptGenerator,
		toolDiscovery:   discovery,
		registry:        registry,
		logger:          logger,
	}
}

// ChatWithIntelligentTools performs chat with context-aware tool integration
func (em *EnhancedModel) ChatWithIntelligentTools(ctx context.Context, messages []model.Message, sessionType string) (*model.Response, error) {
	// Determine prompt context from the conversation
	promptContext := em.analyzePromptContext(messages, sessionType)

	// Generate intelligent system prompt
	systemPrompt, err := em.promptGenerator.GenerateToolPrompt(ctx, promptContext)
	if err != nil {
		em.logger.Error("Failed to generate system prompt: %v", err)
		// Fallback to basic chat
		return em.baseModel.Chat(ctx, messages, model.GenerateOptions{})
	}

	// Get tool definitions for the model
	tools, err := em.getToolDefinitions(ctx)
	if err != nil {
		em.logger.Error("Failed to get tool definitions: %v", err)
		// Fallback to basic chat
		return em.baseModel.Chat(ctx, messages, model.GenerateOptions{})
	}

	// Prepare enhanced messages with system prompt
	enhancedMessages := []model.Message{
		{Role: "system", Content: systemPrompt},
	}
	enhancedMessages = append(enhancedMessages, messages...)

	// Use the model's ChatWithTools method if available
	if len(tools) > 0 {
		response, err := em.baseModel.ChatWithTools(ctx, enhancedMessages, tools, model.GenerateOptions{})
		if err != nil {
			em.logger.Error("ChatWithTools failed, falling back to regular chat: %v", err)
			return em.baseModel.Chat(ctx, enhancedMessages, model.GenerateOptions{})
		}
		return response, nil
	}

	// Fallback to regular chat
	return em.baseModel.Chat(ctx, enhancedMessages, model.GenerateOptions{})
}

// analyzePromptContext analyzes the conversation to determine the appropriate context
func (em *EnhancedModel) analyzePromptContext(messages []model.Message, sessionType string) PromptContext {
	context := PromptContext{
		ConversationLength: len(messages),
		SessionType:        sessionType,
		PreviousToolCalls:  make([]string, 0),
		UserPreferences:    make(map[string]interface{}),
	}

	// Extract the latest user query
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			context.UserQuery = messages[i].Content
			break
		}
	}

	// Analyze conversation for previous tool calls
	for _, message := range messages {
		if message.Role == "assistant" && em.containsToolCall(message.Content) {
			toolName := em.extractToolName(message.Content)
			if toolName != "" {
				context.PreviousToolCalls = append(context.PreviousToolCalls, toolName)
			}
		}
	}

	return context
}

// containsToolCall checks if a message contains a tool call
func (em *EnhancedModel) containsToolCall(content string) bool {
	return strings.Contains(content, "TOOL_CALL:")
}

// extractToolName extracts the tool name from a tool call
func (em *EnhancedModel) extractToolName(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TOOL_CALL:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "TOOL_CALL:"))
		}
	}
	return ""
}

// getToolDefinitions converts MCP tools to model tool definitions
func (em *EnhancedModel) getToolDefinitions(ctx context.Context) ([]model.ToolDefinition, error) {
	// Get tool metadata from discovery
	toolMetadata, err := em.toolDiscovery.DiscoverAllTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	// Convert to model tool definitions
	definitions := make([]model.ToolDefinition, len(toolMetadata))
	for i, meta := range toolMetadata {
		definitions[i] = ConvertMCPToolToDefinition(meta.Tool)
	}

	return definitions, nil
}

// AnalyzeToolIntent analyzes user intent to suggest appropriate tools
func (em *EnhancedModel) AnalyzeToolIntent(ctx context.Context, userQuery string) ([]ToolMetadata, error) {
	// Get all tools
	allTools, err := em.toolDiscovery.DiscoverAllTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	// Create a simple prompt context for analysis
	promptContext := PromptContext{
		UserQuery:   userQuery,
		SessionType: "chat",
	}

	// Filter relevant tools
	relevant := em.promptGenerator.filterRelevantTools(allTools, promptContext)

	em.logger.Info("Analyzed intent for query '%s', found %d relevant tools", userQuery, len(relevant))

	return relevant, nil
}

// RefreshToolCache refreshes the tool discovery cache
func (em *EnhancedModel) RefreshToolCache() {
	em.toolDiscovery.InvalidateCache()
	em.logger.Info("Enhanced model tool cache refreshed")
}

// GetAvailableCapabilities returns the capabilities available through tools
func (em *EnhancedModel) GetAvailableCapabilities(ctx context.Context) (map[ToolCapability]int, error) {
	tools, err := em.toolDiscovery.DiscoverAllTools(ctx)
	if err != nil {
		return nil, err
	}

	capabilities := make(map[ToolCapability]int)
	for _, tool := range tools {
		capabilities[tool.Capability]++
	}

	return capabilities, nil
}

// Implement the base Model interface by delegating to the base model
func (em *EnhancedModel) Generate(ctx context.Context, prompt string, options model.GenerateOptions) (*model.Response, error) {
	return em.baseModel.Generate(ctx, prompt, options)
}

func (em *EnhancedModel) Chat(ctx context.Context, messages []model.Message, options model.GenerateOptions) (*model.Response, error) {
	return em.baseModel.Chat(ctx, messages, options)
}

func (em *EnhancedModel) ChatWithTools(ctx context.Context, messages []model.Message, tools []model.ToolDefinition, options model.GenerateOptions) (*model.Response, error) {
	return em.baseModel.ChatWithTools(ctx, messages, tools, options)
}

func (em *EnhancedModel) IsAvailable(ctx context.Context) bool {
	return em.baseModel.IsAvailable(ctx)
}