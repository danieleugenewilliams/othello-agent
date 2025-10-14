package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
)

// Intent represents user intent classification
type Intent string

const (
	IntentSearch      Intent = "search"
	IntentCreate      Intent = "create"
	IntentUpdate      Intent = "update"
	IntentDelete      Intent = "delete"
	IntentAnalyze     Intent = "analyze"
	IntentTransform   Intent = "transform"
	IntentConnect     Intent = "connect"
	IntentHelp        Intent = "help"
	IntentConversation Intent = "conversation"
)

// ToolSuggestion represents a tool suggestion with confidence score
type ToolSuggestion struct {
	Tool        ToolMetadata
	Confidence  float64
	Reasoning   string
	Parameters  map[string]interface{}
	Alternatives []string
}

// IntentClassifier classifies user intent and suggests appropriate tools
type IntentClassifier struct {
	discovery *ToolDiscovery
	logger    mcp.Logger
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier(discovery *ToolDiscovery, logger mcp.Logger) *IntentClassifier {
	return &IntentClassifier{
		discovery: discovery,
		logger:    logger,
	}
}

// ClassifyIntent analyzes user input to determine intent
func (ic *IntentClassifier) ClassifyIntent(ctx context.Context, userInput string) (Intent, float64, error) {
	inputLower := strings.ToLower(strings.TrimSpace(userInput))
	words := strings.Fields(inputLower)

	// Intent patterns with associated keywords and confidence weights
	intentPatterns := map[Intent][]string{
		IntentSearch: {
			"search", "find", "look", "show", "list", "get", "retrieve",
			"where", "what", "who", "when", "how", "display", "query",
		},
		IntentCreate: {
			"create", "add", "new", "make", "store", "save", "remember",
			"insert", "build", "generate", "establish",
		},
		IntentUpdate: {
			"update", "edit", "change", "modify", "alter", "revise",
			"fix", "correct", "adjust", "improve",
		},
		IntentDelete: {
			"delete", "remove", "clear", "erase", "drop", "eliminate",
			"destroy", "purge", "clean",
		},
		IntentAnalyze: {
			"analyze", "analysis", "stats", "statistics", "report",
			"summary", "insights", "patterns", "trends", "overview",
		},
		IntentTransform: {
			"convert", "transform", "format", "process", "translate",
			"export", "import", "migrate", "restructure",
		},
		IntentConnect: {
			"connect", "relate", "link", "associate", "relationship",
			"correlate", "tie", "bind", "join",
		},
		IntentHelp: {
			"help", "how", "explain", "what", "guide", "tutorial",
			"instructions", "documentation", "support",
		},
	}

	// Calculate confidence scores for each intent
	intentScores := make(map[Intent]float64)

	for intent, keywords := range intentPatterns {
		score := ic.calculateIntentScore(inputLower, words, keywords)
		if score > 0 {
			intentScores[intent] = score
		}
	}

	// Find the highest scoring intent
	var bestIntent Intent = IntentConversation
	var bestScore float64 = 0.0

	for intent, score := range intentScores {
		if score > bestScore {
			bestIntent = intent
			bestScore = score
		}
	}

	// Normalize score to 0-1 range
	if bestScore > 1.0 {
		bestScore = 1.0
	}

	ic.logger.Debug("Classified intent '%s' with confidence %.2f for input: %s",
		bestIntent, bestScore, userInput)

	return bestIntent, bestScore, nil
}

// calculateIntentScore calculates the confidence score for a specific intent
func (ic *IntentClassifier) calculateIntentScore(inputLower string, words []string, keywords []string) float64 {
	score := 0.0

	// Direct keyword matches
	for _, keyword := range keywords {
		if strings.Contains(inputLower, keyword) {
			score += 1.0
		}

		// Bonus for exact word matches
		for _, word := range words {
			if word == keyword {
				score += 0.5
			}
		}
	}

	// Context-based scoring
	if len(words) > 0 {
		// Higher score for verbs at the beginning
		firstWord := words[0]
		for _, keyword := range keywords {
			if firstWord == keyword {
				score += 1.0
				break
			}
		}
	}

	return score
}

// SuggestTools suggests the best tools for the given user input
func (ic *IntentClassifier) SuggestTools(ctx context.Context, userInput string) ([]ToolSuggestion, error) {
	// Classify intent first
	intent, intentConfidence, err := ic.ClassifyIntent(ctx, userInput)
	if err != nil {
		return nil, fmt.Errorf("failed to classify intent: %w", err)
	}

	// Get all available tools
	allTools, err := ic.discovery.DiscoverAllTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	// Generate suggestions based on intent
	suggestions := ic.generateToolSuggestions(userInput, intent, intentConfidence, allTools)

	// Sort by confidence
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	// Limit to top 5 suggestions
	maxSuggestions := 5
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	ic.logger.Info("Generated %d tool suggestions for intent '%s' (confidence: %.2f)",
		len(suggestions), intent, intentConfidence)

	return suggestions, nil
}

// generateToolSuggestions creates tool suggestions based on intent and input
func (ic *IntentClassifier) generateToolSuggestions(userInput string, intent Intent, intentConfidence float64, allTools []ToolMetadata) []ToolSuggestion {
	var suggestions []ToolSuggestion
	inputLower := strings.ToLower(userInput)

	// Map intent to tool capabilities
	intentCapabilityMap := map[Intent][]ToolCapability{
		IntentSearch:    {CapabilitySearch},
		IntentCreate:    {CapabilityCreate},
		IntentUpdate:    {CapabilityUpdate},
		IntentDelete:    {CapabilityDelete},
		IntentAnalyze:   {CapabilityAnalyze},
		IntentTransform: {CapabilityTransform},
		IntentConnect:   {CapabilityConnect},
		IntentHelp:      {CapabilitySearch, CapabilityAnalyze}, // Help often requires searching/analyzing
	}

	relevantCapabilities := intentCapabilityMap[intent]

	for _, tool := range allTools {
		// Check if tool capability matches intent
		capabilityMatch := false
		for _, capability := range relevantCapabilities {
			if tool.Capability == capability {
				capabilityMatch = true
				break
			}
		}

		// Calculate tool confidence
		confidence := ic.calculateToolConfidence(userInput, inputLower, tool, capabilityMatch, intentConfidence)

		if confidence > 0.1 { // Only suggest tools with reasonable confidence
			suggestion := ToolSuggestion{
				Tool:       tool,
				Confidence: confidence,
				Reasoning:  ic.generateReasoning(tool, intent, capabilityMatch),
				Parameters: ic.extractPotentialParameters(userInput, tool),
				Alternatives: ic.findAlternativeTools(tool, allTools),
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// calculateToolConfidence calculates confidence score for a specific tool
func (ic *IntentClassifier) calculateToolConfidence(userInput, inputLower string, tool ToolMetadata, capabilityMatch bool, intentConfidence float64) float64 {
	confidence := 0.0

	// Base confidence from intent classification
	if capabilityMatch {
		confidence += intentConfidence * 0.5
	}

	// Keyword matching with tool name and description
	toolNameLower := strings.ToLower(tool.Tool.Name)
	_ = strings.ToLower(tool.Tool.Description) // toolDescLower - reserved for future use

	// Tool name matches
	if strings.Contains(inputLower, toolNameLower) {
		confidence += 0.8
	}

	// Partial name matches
	nameWords := strings.Fields(toolNameLower)
	inputWords := strings.Fields(inputLower)
	for _, nameWord := range nameWords {
		for _, inputWord := range inputWords {
			if strings.Contains(inputWord, nameWord) || strings.Contains(nameWord, inputWord) {
				confidence += 0.3
			}
		}
	}

	// Description keyword matches
	for _, keyword := range tool.Keywords {
		if strings.Contains(inputLower, keyword) {
			confidence += 0.2
		}
	}

	// Boost confidence for simpler tools when confidence is low
	if confidence < 0.3 && tool.Complexity <= 2 {
		confidence += 0.2
	}

	// Penalize overly complex tools for simple requests
	if len(inputWords) <= 3 && tool.Complexity > 3 {
		confidence *= 0.7
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// generateReasoning creates human-readable reasoning for tool suggestion
func (ic *IntentClassifier) generateReasoning(tool ToolMetadata, intent Intent, capabilityMatch bool) string {
	if capabilityMatch {
		return fmt.Sprintf("This tool matches your intent to %s. %s",
			intent, tool.UsagePattern)
	}

	return fmt.Sprintf("This tool might be useful because %s", tool.UsagePattern)
}

// extractPotentialParameters attempts to extract parameters from user input
func (ic *IntentClassifier) extractPotentialParameters(userInput string, tool ToolMetadata) map[string]interface{} {
	parameters := make(map[string]interface{})

	if tool.Tool.InputSchema == nil {
		return parameters
	}

	properties, ok := tool.Tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		return parameters
	}

	inputLower := strings.ToLower(userInput)

	// Look for common parameter patterns
	for paramName, paramInfo := range properties {
		paramMap, ok := paramInfo.(map[string]interface{})
		if !ok {
			continue
		}

		paramType, _ := paramMap["type"].(string)

		// Try to extract based on parameter name and type
		switch paramName {
		case "query", "search", "term":
			// Extract potential search terms
			if paramType == "string" {
				if query := ic.extractSearchQuery(userInput); query != "" {
					parameters[paramName] = query
				}
			}
		case "content", "text", "message":
			// Extract content after common phrases
			if paramType == "string" {
				if content := ic.extractContent(userInput); content != "" {
					parameters[paramName] = content
				}
			}
		case "importance", "priority":
			// Extract numeric values
			if paramType == "integer" || paramType == "number" {
				if value := ic.extractNumericValue(inputLower); value > 0 {
					parameters[paramName] = value
				}
			}
		}
	}

	return parameters
}

// extractSearchQuery extracts search terms from user input
func (ic *IntentClassifier) extractSearchQuery(userInput string) string {
	// Remove common command words
	query := userInput
	commonPrefixes := []string{
		"search for", "find", "look for", "show me", "get", "retrieve",
		"search", "look", "query",
	}

	queryLower := strings.ToLower(query)
	for _, prefix := range commonPrefixes {
		if strings.HasPrefix(queryLower, prefix) {
			query = strings.TrimSpace(query[len(prefix):])
			break
		}
	}

	// Clean up quotes and extra whitespace
	query = strings.Trim(query, `"'`)
	query = strings.TrimSpace(query)

	if len(query) > 0 {
		return query
	}

	return ""
}

// extractContent extracts content from user input
func (ic *IntentClassifier) extractContent(userInput string) string {
	// Look for patterns like "remember that...", "store...", etc.
	content := userInput
	contentPrefixes := []string{
		"remember that", "store", "save", "add", "create", "remember",
	}

	contentLower := strings.ToLower(content)
	for _, prefix := range contentPrefixes {
		if strings.HasPrefix(contentLower, prefix) {
			content = strings.TrimSpace(content[len(prefix):])
			break
		}
	}

	// Clean up
	content = strings.TrimSpace(content)
	if len(content) > 0 {
		return content
	}

	return ""
}

// extractNumericValue extracts numeric values from input
func (ic *IntentClassifier) extractNumericValue(input string) int {
	// Look for numeric words or digits
	numericWords := map[string]int{
		"low": 3, "medium": 5, "high": 8, "critical": 10,
		"one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
		"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
	}

	words := strings.Fields(input)
	for _, word := range words {
		if value, exists := numericWords[word]; exists {
			return value
		}

		// Try to parse as digit
		if len(word) == 1 && word >= "1" && word <= "9" {
			return int(word[0] - '0')
		}
	}

	return 0
}

// findAlternativeTools finds similar tools that could also work
func (ic *IntentClassifier) findAlternativeTools(tool ToolMetadata, allTools []ToolMetadata) []string {
	var alternatives []string

	for _, otherTool := range allTools {
		if otherTool.Tool.Name == tool.Tool.Name {
			continue
		}

		// Same capability = potential alternative
		if otherTool.Capability == tool.Capability {
			alternatives = append(alternatives, otherTool.Tool.Name)
		}
	}

	// Limit to 3 alternatives
	if len(alternatives) > 3 {
		alternatives = alternatives[:3]
	}

	return alternatives
}