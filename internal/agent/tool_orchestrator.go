package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
)

// ToolOrchestrationResult represents the result of a multi-tool operation
type ToolOrchestrationResult struct {
	PrimaryResult    string
	ToolResults      []ToolExecutionResult
	TotalDuration    time.Duration
	Success          bool
	Error            string
	Recommendations  []string
}

// ToolExecutionResult represents the result of executing a single tool
type ToolExecutionResult struct {
	ToolName   string
	Success    bool
	Result     string
	Error      string
	Duration   time.Duration
	Parameters map[string]interface{}
}

// OrchestrationPlan represents a plan for executing multiple tools
type OrchestrationPlan struct {
	Steps       []OrchestrationStep
	Description string
	Confidence  float64
}

// OrchestrationStep represents a single step in a multi-tool operation
type OrchestrationStep struct {
	ToolName     string
	Parameters   map[string]interface{}
	Dependencies []string // Names of tools that must complete before this step
	Optional     bool     // Whether this step can be skipped if it fails
	Reasoning    string   // Why this step is needed
}

// ToolOrchestrator manages complex multi-tool operations
type ToolOrchestrator struct {
	executor    *mcp.ToolExecutor
	classifier  *IntentClassifier
	discovery   *ToolDiscovery
	logger      mcp.Logger
}

// NewToolOrchestrator creates a new tool orchestrator
func NewToolOrchestrator(executor *mcp.ToolExecutor, classifier *IntentClassifier, discovery *ToolDiscovery, logger mcp.Logger) *ToolOrchestrator {
	return &ToolOrchestrator{
		executor:   executor,
		classifier: classifier,
		discovery:  discovery,
		logger:     logger,
	}
}

// OrchestrateTasks analyzes user input and executes appropriate tools in sequence
func (to *ToolOrchestrator) OrchestrateTasks(ctx context.Context, userInput string, sessionContext map[string]interface{}) (*ToolOrchestrationResult, error) {
	startTime := time.Now()

	// Analyze the input to create an orchestration plan
	plan, err := to.createOrchestrationPlan(ctx, userInput, sessionContext)
	if err != nil {
		return &ToolOrchestrationResult{
			Success:      false,
			Error:        fmt.Sprintf("Failed to create orchestration plan: %v", err),
			TotalDuration: time.Since(startTime),
		}, err
	}

	if len(plan.Steps) == 0 {
		return &ToolOrchestrationResult{
			PrimaryResult:   "No tools needed for this request.",
			Success:         true,
			TotalDuration:   time.Since(startTime),
			Recommendations: []string{"This appears to be a conversational request that doesn't require tool usage."},
		}, nil
	}

	to.logger.Info("Executing orchestration plan with %d steps for input: %s", len(plan.Steps), userInput)

	// Execute the plan
	result := to.executePlan(ctx, plan, userInput)
	result.TotalDuration = time.Since(startTime)

	return result, nil
}

// createOrchestrationPlan analyzes input and creates an execution plan
func (to *ToolOrchestrator) createOrchestrationPlan(ctx context.Context, userInput string, sessionContext map[string]interface{}) (*OrchestrationPlan, error) {
	// Get tool suggestions from the classifier
	suggestions, err := to.classifier.SuggestTools(ctx, userInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool suggestions: %w", err)
	}

	if len(suggestions) == 0 {
		return &OrchestrationPlan{
			Steps:       []OrchestrationStep{},
			Description: "No tools needed",
			Confidence:  0.0,
		}, nil
	}

	// Analyze if this is a complex request requiring multiple tools
	plan := to.analyzeComplexity(userInput, suggestions, sessionContext)

	return plan, nil
}

// analyzeComplexity determines if the request requires multiple tools
func (to *ToolOrchestrator) analyzeComplexity(userInput string, suggestions []ToolSuggestion, sessionContext map[string]interface{}) *OrchestrationPlan {
	inputLower := strings.ToLower(userInput)

	// Check for complex request patterns
	complexPatterns := []string{
		"and then", "after that", "also", "additionally", "plus",
		"as well as", "followed by", "then", "next", "finally",
	}

	isComplex := false
	for _, pattern := range complexPatterns {
		if strings.Contains(inputLower, pattern) {
			isComplex = true
			break
		}
	}

	// Check for multiple verbs/actions
	actionWords := []string{
		"search", "find", "create", "store", "update", "delete",
		"analyze", "show", "list", "save", "remember", "connect",
	}

	actionCount := 0
	for _, action := range actionWords {
		if strings.Contains(inputLower, action) {
			actionCount++
		}
	}

	if actionCount > 1 {
		isComplex = true
	}

	if !isComplex && len(suggestions) > 0 {
		// Simple single-tool operation
		primary := suggestions[0]
		return &OrchestrationPlan{
			Steps: []OrchestrationStep{
				{
					ToolName:   primary.Tool.Tool.Name,
					Parameters: primary.Parameters,
					Optional:   false,
					Reasoning:  primary.Reasoning,
				},
			},
			Description: fmt.Sprintf("Single tool operation: %s", primary.Tool.Tool.Name),
			Confidence:  primary.Confidence,
		}
	}

	// Complex multi-tool operation
	return to.createComplexPlan(userInput, suggestions, sessionContext)
}

// createComplexPlan creates a plan for complex multi-tool operations
func (to *ToolOrchestrator) createComplexPlan(userInput string, suggestions []ToolSuggestion, sessionContext map[string]interface{}) *OrchestrationPlan {
	var steps []OrchestrationStep

	// Analyze the input for different types of operations
	operations := to.identifyOperations(userInput)

	// Create steps based on identified operations and available tools
	for _, operation := range operations {
		step := to.createStepForOperation(operation, suggestions)
		if step != nil {
			steps = append(steps, *step)
		}
	}

	// If no specific operations identified, use the top suggestions
	if len(steps) == 0 && len(suggestions) > 0 {
		// Take the top 2-3 most confident suggestions
		maxSteps := 3
		if len(suggestions) < maxSteps {
			maxSteps = len(suggestions)
		}

		for i := 0; i < maxSteps; i++ {
			if suggestions[i].Confidence > 0.3 { // Only include reasonably confident suggestions
				steps = append(steps, OrchestrationStep{
					ToolName:   suggestions[i].Tool.Tool.Name,
					Parameters: suggestions[i].Parameters,
					Optional:   i > 0, // First step is required, others are optional
					Reasoning:  suggestions[i].Reasoning,
				})
			}
		}
	}

	// Calculate overall plan confidence
	totalConfidence := 0.0
	for _, step := range steps {
		// Find confidence for this tool
		for _, suggestion := range suggestions {
			if suggestion.Tool.Tool.Name == step.ToolName {
				totalConfidence += suggestion.Confidence
				break
			}
		}
	}

	avgConfidence := totalConfidence / float64(len(steps))
	if len(steps) == 0 {
		avgConfidence = 0.0
	}

	return &OrchestrationPlan{
		Steps:       steps,
		Description: fmt.Sprintf("Multi-tool operation with %d steps", len(steps)),
		Confidence:  avgConfidence,
	}
}

// identifyOperations identifies different operations within the user input
func (to *ToolOrchestrator) identifyOperations(userInput string) []string {
	var operations []string
	inputLower := strings.ToLower(userInput)

	// Look for common operation patterns
	operationPatterns := map[string][]string{
		"search":    {"search", "find", "look for", "show", "list"},
		"create":    {"create", "add", "store", "save", "remember"},
		"update":    {"update", "edit", "change", "modify"},
		"delete":    {"delete", "remove", "clear"},
		"analyze":   {"analyze", "stats", "summary", "report"},
		"transform": {"convert", "transform", "export", "format"},
		"connect":   {"relate", "connect", "link", "associate"},
	}

	for operation, patterns := range operationPatterns {
		for _, pattern := range patterns {
			if strings.Contains(inputLower, pattern) {
				operations = append(operations, operation)
				break
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, op := range operations {
		if !seen[op] {
			seen[op] = true
			unique = append(unique, op)
		}
	}

	return unique
}

// createStepForOperation creates a step for a specific operation
func (to *ToolOrchestrator) createStepForOperation(operation string, suggestions []ToolSuggestion) *OrchestrationStep {
	// Find the best tool for this operation
	for _, suggestion := range suggestions {
		capability := suggestion.Tool.Capability

		// Match operation to capability
		match := false
		switch operation {
		case "search":
			match = capability == CapabilitySearch
		case "create":
			match = capability == CapabilityCreate
		case "update":
			match = capability == CapabilityUpdate
		case "delete":
			match = capability == CapabilityDelete
		case "analyze":
			match = capability == CapabilityAnalyze
		case "transform":
			match = capability == CapabilityTransform
		case "connect":
			match = capability == CapabilityConnect
		}

		if match {
			return &OrchestrationStep{
				ToolName:   suggestion.Tool.Tool.Name,
				Parameters: suggestion.Parameters,
				Optional:   false,
				Reasoning:  fmt.Sprintf("Best tool for %s operation", operation),
			}
		}
	}

	return nil
}

// executePlan executes the orchestration plan step by step
func (to *ToolOrchestrator) executePlan(ctx context.Context, plan *OrchestrationPlan, userInput string) *ToolOrchestrationResult {
	result := &ToolOrchestrationResult{
		ToolResults:     make([]ToolExecutionResult, 0),
		Success:         true,
		Recommendations: make([]string, 0),
	}

	var primaryResult strings.Builder
	completedSteps := make(map[string]bool)

	for _, step := range plan.Steps {
		// Check dependencies
		if !to.checkDependencies(step.Dependencies, completedSteps) {
			if !step.Optional {
				result.Success = false
				result.Error = fmt.Sprintf("Dependencies not met for step: %s", step.ToolName)
				return result
			}
			// Skip optional step with unmet dependencies
			continue
		}

		// Execute the step
		stepResult := to.executeStep(ctx, step)
		result.ToolResults = append(result.ToolResults, stepResult)

		if stepResult.Success {
			completedSteps[step.ToolName] = true

			// Add to primary result
			if primaryResult.Len() > 0 {
				primaryResult.WriteString("\n\n")
			}
			primaryResult.WriteString(stepResult.Result)

			to.logger.Info("Successfully executed step: %s", step.ToolName)
		} else {
			if !step.Optional {
				result.Success = false
				result.Error = fmt.Sprintf("Required step failed: %s - %s", step.ToolName, stepResult.Error)
				return result
			}

			// Add recommendation for failed optional step
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Optional step '%s' failed but can be retried later", step.ToolName))

			to.logger.Info("Optional step failed: %s - %s", step.ToolName, stepResult.Error)
		}
	}

	result.PrimaryResult = primaryResult.String()

	// Add success recommendations
	if result.Success && len(result.ToolResults) > 1 {
		result.Recommendations = append(result.Recommendations,
			"Multiple tools were used successfully to complete your request")
	}

	return result
}

// checkDependencies checks if all dependencies for a step are met
func (to *ToolOrchestrator) checkDependencies(dependencies []string, completed map[string]bool) bool {
	for _, dep := range dependencies {
		if !completed[dep] {
			return false
		}
	}
	return true
}

// executeStep executes a single orchestration step
func (to *ToolOrchestrator) executeStep(ctx context.Context, step OrchestrationStep) ToolExecutionResult {
	startTime := time.Now()

	// Execute the tool
	executeResult, err := to.executor.Execute(ctx, step.ToolName, step.Parameters)
	duration := time.Since(startTime)

	if err != nil {
		return ToolExecutionResult{
			ToolName:   step.ToolName,
			Success:    false,
			Error:      err.Error(),
			Duration:   duration,
			Parameters: step.Parameters,
		}
	}

	// Format the result
	formattedResult := to.executor.FormatResult(executeResult)

	return ToolExecutionResult{
		ToolName:   step.ToolName,
		Success:    true,
		Result:     formattedResult,
		Duration:   duration,
		Parameters: step.Parameters,
	}
}

// GetOrchestrationSuggestions provides suggestions for complex operations
func (to *ToolOrchestrator) GetOrchestrationSuggestions(ctx context.Context, userInput string) ([]string, error) {
	suggestions, err := to.classifier.SuggestTools(ctx, userInput)
	if err != nil {
		return nil, err
	}

	var orchestrationSuggestions []string

	if len(suggestions) > 1 {
		orchestrationSuggestions = append(orchestrationSuggestions,
			"This request could benefit from using multiple tools in sequence")

		for i, suggestion := range suggestions {
			if i >= 3 { // Limit to top 3
				break
			}
			orchestrationSuggestions = append(orchestrationSuggestions,
				fmt.Sprintf("Step %d: Use %s - %s", i+1, suggestion.Tool.Tool.Name, suggestion.Reasoning))
		}
	}

	return orchestrationSuggestions, nil
}