package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// UniversalAgentIntegration provides the main interface for intelligent tool usage
type UniversalAgentIntegration struct {
	discovery      *ToolDiscovery
	promptGen      *SystemPromptGenerator
	classifier     *IntentClassifier
	orchestrator   *ToolOrchestrator
	enhancedModel  *EnhancedModel
	executor       *mcp.ToolExecutor
	registry       *mcp.ToolRegistry
	logger         mcp.Logger
}

// NewUniversalAgentIntegration creates a complete universal agent integration
func NewUniversalAgentIntegration(registry *mcp.ToolRegistry, baseModel model.Model, logger mcp.Logger) *UniversalAgentIntegration {
	// Create tool executor
	executor := mcp.NewToolExecutor(registry, logger)

	// Create discovery system
	discovery := NewToolDiscovery(registry, logger)

	// Create prompt generator
	promptGen := NewSystemPromptGenerator(discovery, logger)

	// Create intent classifier
	classifier := NewIntentClassifier(discovery, logger)

	// Create orchestrator
	orchestrator := NewToolOrchestrator(executor, classifier, discovery, logger)

	// Create enhanced model
	enhancedModel := NewEnhancedModel(baseModel, registry, logger)

	return &UniversalAgentIntegration{
		discovery:     discovery,
		promptGen:     promptGen,
		classifier:    classifier,
		orchestrator:  orchestrator,
		enhancedModel: enhancedModel,
		executor:      executor,
		registry:      registry,
		logger:        logger,
	}
}

// ProcessUserRequest is the main entry point for processing user requests with intelligent tool usage
func (uai *UniversalAgentIntegration) ProcessUserRequest(ctx context.Context, userInput string, conversationHistory []model.Message, sessionType string) (*UniversalAgentResponse, error) {
	uai.logger.Info("Processing user request with universal integration: %s", userInput)

	response := &UniversalAgentResponse{
		UserInput:       userInput,
		SessionType:     sessionType,
		ToolsAvailable:  true,
		ProcessingSteps: make([]ProcessingStep, 0),
	}

	// Step 1: Classify intent
	intent, intentConfidence, err := uai.classifier.ClassifyIntent(ctx, userInput)
	if err != nil {
		return uai.handleError(response, "intent classification", err)
	}

	response.Intent = string(intent)
	response.IntentConfidence = intentConfidence
	response.ProcessingSteps = append(response.ProcessingSteps, ProcessingStep{
		Step:        "Intent Classification",
		Result:      fmt.Sprintf("Classified as '%s' with %.2f confidence", intent, intentConfidence),
		Success:     true,
		Confidence:  intentConfidence,
	})

	// Step 2: Determine if tools are needed
	if intent == IntentConversation || intentConfidence < 0.3 {
		// Handle as regular conversation
		return uai.handleConversationalRequest(ctx, response, userInput, conversationHistory, sessionType)
	}

	// Step 3: Get tool suggestions
	suggestions, err := uai.classifier.SuggestTools(ctx, userInput)
	if err != nil {
		return uai.handleError(response, "tool suggestion", err)
	}

	response.ToolSuggestions = suggestions
	response.ProcessingSteps = append(response.ProcessingSteps, ProcessingStep{
		Step:       "Tool Selection",
		Result:     fmt.Sprintf("Found %d relevant tools", len(suggestions)),
		Success:    true,
		Confidence: uai.calculateAverageConfidence(suggestions),
	})

	if len(suggestions) == 0 {
		// No tools found, handle as conversation
		return uai.handleConversationalRequest(ctx, response, userInput, conversationHistory, sessionType)
	}

	// Step 4: Check if orchestration is needed
	if uai.needsOrchestration(userInput, suggestions) {
		return uai.handleOrchestrationRequest(ctx, response, userInput, conversationHistory, sessionType)
	}

	// Step 5: Execute single tool
	return uai.handleSingleToolRequest(ctx, response, userInput, suggestions[0])
}

// handleConversationalRequest handles requests that don't need tools
func (uai *UniversalAgentIntegration) handleConversationalRequest(ctx context.Context, response *UniversalAgentResponse, userInput string, conversationHistory []model.Message, sessionType string) (*UniversalAgentResponse, error) {
	response.ProcessingSteps = append(response.ProcessingSteps, ProcessingStep{
		Step:    "Conversational Response",
		Result:  "Handling as conversational request",
		Success: true,
	})

	// Use enhanced model for intelligent conversation
	modelResponse, err := uai.enhancedModel.ChatWithIntelligentTools(ctx, conversationHistory, sessionType)
	if err != nil {
		return uai.handleError(response, "conversation generation", err)
	}

	response.FinalResponse = modelResponse.Content
	response.Success = true
	response.ResponseType = "conversation"

	return response, nil
}

// handleSingleToolRequest handles requests needing a single tool
func (uai *UniversalAgentIntegration) handleSingleToolRequest(ctx context.Context, response *UniversalAgentResponse, userInput string, suggestion ToolSuggestion) (*UniversalAgentResponse, error) {
	response.ProcessingSteps = append(response.ProcessingSteps, ProcessingStep{
		Step:    "Single Tool Execution",
		Result:  fmt.Sprintf("Executing tool: %s", suggestion.Tool.Tool.Name),
		Success: true,
	})

	// Execute the tool
	executeResult, err := uai.executor.Execute(ctx, suggestion.Tool.Tool.Name, suggestion.Parameters)
	if err != nil {
		return uai.handleError(response, "tool execution", err)
	}

	// Format the result
	formattedResult := uai.executor.FormatResult(executeResult)

	response.ToolResults = []ToolExecutionResult{
		{
			ToolName:   suggestion.Tool.Tool.Name,
			Success:    true,
			Result:     formattedResult,
			Parameters: suggestion.Parameters,
		},
	}

	response.FinalResponse = formattedResult
	response.Success = true
	response.ResponseType = "single_tool"

	return response, nil
}

// handleOrchestrationRequest handles complex requests needing multiple tools
func (uai *UniversalAgentIntegration) handleOrchestrationRequest(ctx context.Context, response *UniversalAgentResponse, userInput string, conversationHistory []model.Message, sessionType string) (*UniversalAgentResponse, error) {
	response.ProcessingSteps = append(response.ProcessingSteps, ProcessingStep{
		Step:    "Multi-Tool Orchestration",
		Result:  "Executing orchestrated tool sequence",
		Success: true,
	})

	// Execute orchestration
	sessionContext := map[string]interface{}{
		"sessionType": sessionType,
		"historyLength": len(conversationHistory),
	}

	orchResult, err := uai.orchestrator.OrchestrateTasks(ctx, userInput, sessionContext)
	if err != nil {
		return uai.handleError(response, "orchestration", err)
	}

	response.OrchestrationResult = orchResult
	response.ToolResults = orchResult.ToolResults
	response.FinalResponse = orchResult.PrimaryResult
	response.Success = orchResult.Success
	response.ResponseType = "orchestration"

	if !orchResult.Success {
		response.Error = orchResult.Error
	}

	return response, nil
}

// needsOrchestration determines if a request needs multiple tools
func (uai *UniversalAgentIntegration) needsOrchestration(userInput string, suggestions []ToolSuggestion) bool {
	// Check for multiple high-confidence suggestions
	highConfidenceCount := 0
	for _, suggestion := range suggestions {
		if suggestion.Confidence > 0.6 {
			highConfidenceCount++
		}
	}

	if highConfidenceCount > 1 {
		return true
	}

	// Check for complex language patterns
	inputLower := strings.ToLower(userInput)
	complexPatterns := []string{
		"and then", "after that", "also", "additionally",
		"as well as", "followed by", "then", "next",
	}

	for _, pattern := range complexPatterns {
		if strings.Contains(inputLower, pattern) {
			return true
		}
	}

	return false
}

// calculateAverageConfidence calculates average confidence from suggestions
func (uai *UniversalAgentIntegration) calculateAverageConfidence(suggestions []ToolSuggestion) float64 {
	if len(suggestions) == 0 {
		return 0.0
	}

	total := 0.0
	for _, suggestion := range suggestions {
		total += suggestion.Confidence
	}

	return total / float64(len(suggestions))
}

// handleError handles errors and creates appropriate response
func (uai *UniversalAgentIntegration) handleError(response *UniversalAgentResponse, step string, err error) (*UniversalAgentResponse, error) {
	response.Success = false
	response.Error = err.Error()
	response.ProcessingSteps = append(response.ProcessingSteps, ProcessingStep{
		Step:    step,
		Result:  fmt.Sprintf("Error: %v", err),
		Success: false,
	})

	uai.logger.Error("Universal integration error in %s: %v", step, err)

	return response, err
}

// UniversalAgentResponse represents the complete response from universal agent processing
type UniversalAgentResponse struct {
	UserInput             string                      `json:"user_input"`
	SessionType           string                      `json:"session_type"`
	Intent                string                      `json:"intent"`
	IntentConfidence      float64                     `json:"intent_confidence"`
	ToolsAvailable        bool                        `json:"tools_available"`
	ToolSuggestions       []ToolSuggestion           `json:"tool_suggestions,omitempty"`
	ToolResults           []ToolExecutionResult      `json:"tool_results,omitempty"`
	OrchestrationResult   *ToolOrchestrationResult   `json:"orchestration_result,omitempty"`
	ProcessingSteps       []ProcessingStep           `json:"processing_steps"`
	FinalResponse         string                      `json:"final_response"`
	ResponseType          string                      `json:"response_type"` // "conversation", "single_tool", "orchestration"
	Success               bool                        `json:"success"`
	Error                 string                      `json:"error,omitempty"`
	Recommendations       []string                    `json:"recommendations,omitempty"`
}

// ProcessingStep represents a step in the processing pipeline
type ProcessingStep struct {
	Step       string  `json:"step"`
	Result     string  `json:"result"`
	Success    bool    `json:"success"`
	Confidence float64 `json:"confidence,omitempty"`
	Duration   string  `json:"duration,omitempty"`
}

// GetToolCapabilitySummary returns a summary of available tool capabilities
func (uai *UniversalAgentIntegration) GetToolCapabilitySummary(ctx context.Context) (map[string]int, error) {
	capabilities, err := uai.enhancedModel.GetAvailableCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	summary := make(map[string]int)
	for capability, count := range capabilities {
		summary[GetCapabilityName(capability)] = count
	}

	return summary, nil
}

// RefreshToolCache refreshes all tool caches
func (uai *UniversalAgentIntegration) RefreshToolCache() {
	uai.discovery.InvalidateCache()
	uai.enhancedModel.RefreshToolCache()
	uai.logger.Info("Universal agent integration caches refreshed")
}

// AnalyzeUserIntent provides detailed intent analysis for debugging
func (uai *UniversalAgentIntegration) AnalyzeUserIntent(ctx context.Context, userInput string) (*IntentAnalysis, error) {
	intent, confidence, err := uai.classifier.ClassifyIntent(ctx, userInput)
	if err != nil {
		return nil, err
	}

	suggestions, err := uai.classifier.SuggestTools(ctx, userInput)
	if err != nil {
		return nil, err
	}

	return &IntentAnalysis{
		Intent:          string(intent),
		Confidence:      confidence,
		ToolSuggestions: suggestions,
		RequiresTools:   len(suggestions) > 0 && confidence > 0.3,
		ComplexRequest:  uai.needsOrchestration(userInput, suggestions),
	}, nil
}

// IntentAnalysis provides detailed analysis of user intent
type IntentAnalysis struct {
	Intent          string             `json:"intent"`
	Confidence      float64            `json:"confidence"`
	ToolSuggestions []ToolSuggestion   `json:"tool_suggestions"`
	RequiresTools   bool               `json:"requires_tools"`
	ComplexRequest  bool               `json:"complex_request"`
}