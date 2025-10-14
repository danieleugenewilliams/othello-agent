package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
)

// SystemPromptGenerator creates intelligent, context-aware system prompts
type SystemPromptGenerator struct {
	discovery *ToolDiscovery
	logger    mcp.Logger
}

// PromptContext contains context information for prompt generation
type PromptContext struct {
	UserQuery          string
	ConversationLength int
	PreviousToolCalls  []string
	UserPreferences    map[string]interface{}
	SessionType        string // "chat", "analysis", "automation", etc.
}

// NewSystemPromptGenerator creates a new system prompt generator
func NewSystemPromptGenerator(discovery *ToolDiscovery, logger mcp.Logger) *SystemPromptGenerator {
	return &SystemPromptGenerator{
		discovery: discovery,
		logger:    logger,
	}
}

// GenerateToolPrompt creates a dynamic, context-aware system prompt with tool information
func (spg *SystemPromptGenerator) GenerateToolPrompt(ctx context.Context, promptContext PromptContext) (string, error) {
	// Get all available tools
	allTools, err := spg.discovery.DiscoverAllTools(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to discover tools: %w", err)
	}

	if len(allTools) == 0 {
		return spg.generateBasicPrompt(), nil
	}

	// Filter tools based on context
	relevantTools := spg.filterRelevantTools(allTools, promptContext)

	// Generate prompt sections
	prompt := spg.generateHeaderSection(promptContext)
	prompt += spg.generateToolFormatSection()
	prompt += spg.generateToolCatalogSection(relevantTools)
	prompt += spg.generateUsageExamplesSection(relevantTools, promptContext)
	prompt += spg.generateFooterSection(promptContext)

	spg.logger.Info("Generated system prompt with %d tools for session type: %s",
		len(relevantTools), promptContext.SessionType)

	return prompt, nil
}

// generateBasicPrompt returns a basic prompt when no tools are available
func (spg *SystemPromptGenerator) generateBasicPrompt() string {
	return `You are a helpful AI assistant. Respond to user queries with accurate, helpful information.

Be concise but thorough in your responses. If you're unsure about something, say so rather than guessing.`
}

// filterRelevantTools filters tools based on the prompt context
func (spg *SystemPromptGenerator) filterRelevantTools(allTools []ToolMetadata, context PromptContext) []ToolMetadata {
	// If user query is provided, filter by relevance
	if context.UserQuery != "" {
		return spg.filterByQueryRelevance(allTools, context.UserQuery)
	}

	// Filter by session type
	switch context.SessionType {
	case "analysis":
		return spg.filterByCapabilities(allTools, []ToolCapability{CapabilityAnalyze, CapabilitySearch})
	case "automation":
		return spg.filterByCapabilities(allTools, []ToolCapability{CapabilityCreate, CapabilityUpdate, CapabilityTransform})
	default:
		// For general chat, include all tools but prioritize simpler ones
		return spg.prioritizeSimpleTools(allTools)
	}
}

// filterByQueryRelevance filters tools based on query keywords and intent
func (spg *SystemPromptGenerator) filterByQueryRelevance(tools []ToolMetadata, query string) []ToolMetadata {
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	var relevant []ToolMetadata
	for _, tool := range tools {
		score := spg.calculateRelevanceScore(tool, queryWords)
		if score > 0 {
			relevant = append(relevant, tool)
		}
	}

	// If no tools are relevant, return top 5 simplest tools
	if len(relevant) == 0 {
		return spg.getTopSimpleTools(tools, 5)
	}

	return relevant
}

// calculateRelevanceScore calculates how relevant a tool is to the query
func (spg *SystemPromptGenerator) calculateRelevanceScore(tool ToolMetadata, queryWords []string) int {
	score := 0

	// Check tool name
	toolNameWords := strings.Fields(strings.ToLower(tool.Tool.Name))
	for _, queryWord := range queryWords {
		for _, toolWord := range toolNameWords {
			if strings.Contains(toolWord, queryWord) || strings.Contains(queryWord, toolWord) {
				score += 3
			}
		}
	}

	// Check tool description
	descWords := strings.Fields(strings.ToLower(tool.Tool.Description))
	for _, queryWord := range queryWords {
		for _, descWord := range descWords {
			if strings.Contains(descWord, queryWord) || strings.Contains(queryWord, descWord) {
				score += 2
			}
		}
	}

	// Check keywords
	for _, queryWord := range queryWords {
		for _, keyword := range tool.Keywords {
			if strings.Contains(keyword, queryWord) || strings.Contains(queryWord, keyword) {
				score += 1
			}
		}
	}

	return score
}

// filterByCapabilities filters tools by their capabilities
func (spg *SystemPromptGenerator) filterByCapabilities(tools []ToolMetadata, capabilities []ToolCapability) []ToolMetadata {
	capabilityMap := make(map[ToolCapability]bool)
	for _, cap := range capabilities {
		capabilityMap[cap] = true
	}

	var filtered []ToolMetadata
	for _, tool := range tools {
		if capabilityMap[tool.Capability] {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}

// prioritizeSimpleTools prioritizes simpler tools for general use
func (spg *SystemPromptGenerator) prioritizeSimpleTools(tools []ToolMetadata) []ToolMetadata {
	// Sort by complexity (simple first) and limit to reasonable number
	simple := make([]ToolMetadata, 0)
	complex := make([]ToolMetadata, 0)

	for _, tool := range tools {
		if tool.Complexity <= 3 {
			simple = append(simple, tool)
		} else {
			complex = append(complex, tool)
		}
	}

	// Return simple tools first, then complex ones, but limit total
	result := simple
	maxTools := 10
	if len(result) < maxTools {
		remaining := maxTools - len(result)
		if remaining > len(complex) {
			remaining = len(complex)
		}
		result = append(result, complex[:remaining]...)
	}

	return result
}

// getTopSimpleTools returns the N simplest tools
func (spg *SystemPromptGenerator) getTopSimpleTools(tools []ToolMetadata, n int) []ToolMetadata {
	simple := make([]ToolMetadata, 0)
	for _, tool := range tools {
		if tool.Complexity <= 2 {
			simple = append(simple, tool)
		}
	}

	if len(simple) > n {
		return simple[:n]
	}
	return simple
}

// generateHeaderSection creates the header of the system prompt
func (spg *SystemPromptGenerator) generateHeaderSection(context PromptContext) string {
	header := `You are an intelligent AI assistant with access to powerful tools that extend your capabilities. `

	switch context.SessionType {
	case "analysis":
		header += `You excel at analyzing data and providing insights. `
	case "automation":
		header += `You focus on automating tasks and managing data efficiently. `
	default:
		header += `You help users accomplish their goals efficiently and accurately. `
	}

	header += `

CRITICAL TOOL USAGE RULES:
1. **ALWAYS use tools when the user's request requires them** - don't try to answer from memory when tools can provide current/accurate data
2. **Use the EXACT format specified below** - any deviation will cause tool calls to fail
3. **Include ALL required parameters** with correct names and types
4. **One tool call per response** - if you need multiple tools, make one call and explain what additional tools might be needed

`
	return header
}

// generateToolFormatSection creates the tool calling format section
func (spg *SystemPromptGenerator) generateToolFormatSection() string {
	return `TOOL CALLING FORMAT (use exactly as shown):
TOOL_CALL: exact_tool_name
ARGUMENTS: {"parameter_name": "parameter_value", "another_param": "value"}

IMPORTANT FORMAT NOTES:
- Tool name must match exactly (case-sensitive)
- Arguments must be valid JSON
- Use double quotes for all strings
- Include all required parameters
- Use correct data types (string, number, boolean, array)

`
}

// generateToolCatalogSection creates the main tool catalog
func (spg *SystemPromptGenerator) generateToolCatalogSection(tools []ToolMetadata) string {
	if len(tools) == 0 {
		return ""
	}

	catalog := "AVAILABLE TOOLS:\n"

	// Group tools by capability
	byCapability := make(map[ToolCapability][]ToolMetadata)
	for _, tool := range tools {
		byCapability[tool.Capability] = append(byCapability[tool.Capability], tool)
	}

	// Generate sections for each capability
	capabilities := []ToolCapability{
		CapabilitySearch, CapabilityCreate, CapabilityUpdate,
		CapabilityDelete, CapabilityAnalyze, CapabilityTransform,
		CapabilityConnect, CapabilityUnknown,
	}

	for _, capability := range capabilities {
		toolsInCap := byCapability[capability]
		if len(toolsInCap) == 0 {
			continue
		}

		catalog += fmt.Sprintf("\n## %s\n", GetCapabilityName(capability))

		for _, tool := range toolsInCap {
			catalog += fmt.Sprintf("**%s**: %s\n", tool.Tool.Name, tool.Tool.Description)
			catalog += spg.formatToolParameters(tool.Tool)
			catalog += fmt.Sprintf("  Usage: %s\n\n", tool.UsagePattern)
		}
	}

	return catalog
}

// formatToolParameters formats the parameters for a tool in a readable way
func (spg *SystemPromptGenerator) formatToolParameters(tool mcp.Tool) string {
	if tool.InputSchema == nil {
		return "  Parameters: None\n"
	}

	properties, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok || len(properties) == 0 {
		return "  Parameters: None\n"
	}

	// Get required fields
	requiredFields := make(map[string]bool)
	if required, ok := tool.InputSchema["required"].([]interface{}); ok {
		for _, field := range required {
			if fieldName, ok := field.(string); ok {
				requiredFields[fieldName] = true
			}
		}
	}

	result := "  Parameters:\n"
	for paramName, paramInfo := range properties {
		paramMap, ok := paramInfo.(map[string]interface{})
		if !ok {
			continue
		}

		// Format parameter
		if requiredFields[paramName] {
			result += fmt.Sprintf("    - %s* (required)", paramName)
		} else {
			result += fmt.Sprintf("    - %s (optional)", paramName)
		}

		if paramType, ok := paramMap["type"].(string); ok {
			result += fmt.Sprintf(": %s", paramType)
		}

		if desc, ok := paramMap["description"].(string); ok {
			result += fmt.Sprintf(" - %s", desc)
		}

		if enum, ok := paramMap["enum"].([]interface{}); ok && len(enum) > 0 {
			result += " (options: "
			for i, val := range enum {
				if i > 0 {
					result += ", "
				}
				result += fmt.Sprintf("%v", val)
			}
			result += ")"
		}

		result += "\n"
	}

	return result
}

// generateUsageExamplesSection creates contextual examples
func (spg *SystemPromptGenerator) generateUsageExamplesSection(tools []ToolMetadata, context PromptContext) string {
	if len(tools) == 0 {
		return ""
	}

	examples := "\nTOOL USAGE EXAMPLES:\n"

	// Generate examples for the top 3 most relevant tools
	maxExamples := 3
	if len(tools) < maxExamples {
		maxExamples = len(tools)
	}

	for i := 0; i < maxExamples; i++ {
		tool := tools[i]
		examples += spg.generateToolExample(tool)
	}

	return examples
}

// generateToolExample creates a realistic usage example for a tool
func (spg *SystemPromptGenerator) generateToolExample(tool ToolMetadata) string {
	toolName := tool.Tool.Name

	// Generate example based on tool capability
	switch tool.Capability {
	case CapabilitySearch:
		return fmt.Sprintf(`User: "Find information about machine learning"
Assistant: TOOL_CALL: %s
ARGUMENTS: {"query": "machine learning", "search_type": "semantic"}

`, toolName)

	case CapabilityCreate:
		return fmt.Sprintf(`User: "Remember that Redis is used for caching"
Assistant: TOOL_CALL: %s
ARGUMENTS: {"content": "Redis is used for caching", "importance": 8}

`, toolName)

	case CapabilityAnalyze:
		return fmt.Sprintf(`User: "Analyze my recent memories"
Assistant: TOOL_CALL: %s
ARGUMENTS: {"analysis_type": "summarize", "timeframe": "week"}

`, toolName)

	default:
		// Generate a generic example based on the first parameter
		if tool.Tool.InputSchema != nil {
			if properties, ok := tool.Tool.InputSchema["properties"].(map[string]interface{}); ok {
				// Find the first parameter for a simple example
				for paramName, paramInfo := range properties {
					paramMap, ok := paramInfo.(map[string]interface{})
					if !ok {
						continue
					}

					paramType, _ := paramMap["type"].(string)
					exampleValue := spg.getExampleValue(paramType)

					return fmt.Sprintf(`User: "Use %s tool"
Assistant: TOOL_CALL: %s
ARGUMENTS: {"%s": %s}

`, toolName, toolName, paramName, exampleValue)
				}
			}
		}

		return fmt.Sprintf(`User: "Use %s"
Assistant: TOOL_CALL: %s
ARGUMENTS: {}

`, toolName, toolName)
	}
}

// getExampleValue returns an example value for a parameter type
func (spg *SystemPromptGenerator) getExampleValue(paramType string) string {
	switch paramType {
	case "string":
		return `"example text"`
	case "number", "integer":
		return "10"
	case "boolean":
		return "true"
	case "array":
		return `["item1", "item2"]`
	case "object":
		return `{"key": "value"}`
	default:
		return `"example"`
	}
}

// generateFooterSection creates the footer with final instructions
func (spg *SystemPromptGenerator) generateFooterSection(context PromptContext) string {
	footer := `
IMPORTANT REMINDERS:
- **Always prioritize tool usage** when the user's request matches tool capabilities
- **Use exact tool names and parameter names** as shown above
- **Provide helpful responses** even when not using tools
- **Explain your reasoning** when choosing which tool to use
- **Ask for clarification** if the user's request is ambiguous

If you don't need a tool for a query, respond normally with helpful information.`

	if context.SessionType == "analysis" {
		footer += "\n- **Focus on data-driven insights** and use analysis tools when appropriate"
	} else if context.SessionType == "automation" {
		footer += "\n- **Emphasize efficiency** and suggest automation opportunities"
	}

	return footer
}