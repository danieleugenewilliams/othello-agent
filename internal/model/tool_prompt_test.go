package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateToolPrompt_BasicTool tests that the prompt includes essential tool information
func TestCreateToolPrompt_BasicTool(t *testing.T) {
	model := &OllamaModel{}
	
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Search for items in the database",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query text",
					},
				},
				"required": []interface{}{"query"},
			},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should include tool name and description
	assert.Contains(t, prompt, "search", "Prompt should include tool name")
	assert.Contains(t, prompt, "Search for items in the database", "Prompt should include tool description")
	
	// Should include parameter information
	assert.Contains(t, prompt, "query", "Prompt should include parameter name")
	assert.Contains(t, prompt, "string", "Prompt should include parameter type")
	assert.Contains(t, prompt, "The search query text", "Prompt should include parameter description")
	
	// Should indicate which parameters are required
	assert.Contains(t, prompt, "required", "Prompt should indicate required parameters")
	
	// Should include instructions on how to call tools
	assert.Contains(t, prompt, "TOOL_CALL", "Prompt should include tool call format")
	assert.Contains(t, prompt, "ARGUMENTS", "Prompt should include arguments format")
}

// TestCreateToolPrompt_RequiredVsOptional tests that required and optional parameters are distinguished
func TestCreateToolPrompt_RequiredVsOptional(t *testing.T) {
	model := &OllamaModel{}
	
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Search for items",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Max results",
						"default":     10,
					},
				},
				"required": []interface{}{"query"},
			},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should clearly distinguish required vs optional
	// Check that query is marked as required
	assert.Contains(t, prompt, "query", "Should include query parameter")
	assert.Contains(t, prompt, "required", "Should mark query as required")
	
	// Check that limit is shown as optional (with default)
	assert.Contains(t, prompt, "limit", "Should include limit parameter")
	assert.Contains(t, prompt, "optional", "Should indicate optional parameters")
	assert.Regexp(t, `(?i)default`, prompt, "Should show default values")
}

// TestCreateToolPrompt_EnumValues tests that enum constraints are displayed
func TestCreateToolPrompt_EnumValues(t *testing.T) {
	model := &OllamaModel{}
	
	tools := []ToolDefinition{
		{
			Name:        "stats",
			Description: "Get statistics",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"stats_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of statistics to retrieve",
						"enum":        []interface{}{"session", "domain", "category"},
					},
				},
				"required": []interface{}{},
			},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should show all enum values
	assert.Contains(t, prompt, "stats_type", "Should include parameter name")
	assert.Contains(t, prompt, "session", "Should include enum value 'session'")
	assert.Contains(t, prompt, "domain", "Should include enum value 'domain'")
	assert.Contains(t, prompt, "category", "Should include enum value 'category'")
	
	// Should indicate these are the only valid values
	assert.Regexp(t, `(?i)(allowed values|enum|must be|choices|options)`, prompt, "Should indicate enum constraint")
}

// TestCreateToolPrompt_MultipleTools tests formatting with multiple tools
func TestCreateToolPrompt_MultipleTools(t *testing.T) {
	model := &OllamaModel{}
	
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Search for items",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		{
			Name:        "store_memory",
			Description: "Store a memory",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should include both tools
	assert.Contains(t, prompt, "search", "Should include first tool")
	assert.Contains(t, prompt, "store_memory", "Should include second tool")
	
	// Should be clearly separated (we use **tool_name** format)
	count := strings.Count(prompt, "**search**")
	assert.Equal(t, 1, count, "Each tool should appear exactly once in tool list")
}

// TestCreateToolPrompt_ComplexNestedSchema tests handling of nested objects and arrays
func TestCreateToolPrompt_ComplexNestedSchema(t *testing.T) {
	model := &OllamaModel{}
	
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Search with filters",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
					},
					"tags": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Array of tag strings",
					},
				},
			},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should indicate array type
	assert.Contains(t, prompt, "tags", "Should include array parameter")
	assert.Contains(t, prompt, "array", "Should indicate array type")
	assert.Contains(t, prompt, "Array of tag strings", "Should include array description")
}

// TestCreateToolPrompt_NoTools tests behavior with empty tool list
func TestCreateToolPrompt_NoTools(t *testing.T) {
	model := &OllamaModel{}
	
	prompt := model.createToolPrompt([]ToolDefinition{})
	
	// Should return a basic assistant prompt
	assert.Contains(t, prompt, "assistant", "Should have basic assistant prompt")
	assert.NotContains(t, prompt, "TOOL_CALL", "Should not mention tools")
}

// TestCreateToolPrompt_Example tests that the prompt includes usage examples
func TestCreateToolPrompt_Example(t *testing.T) {
	model := &OllamaModel{}
	
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Search for items",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should include an example of how to call a tool
	assert.Contains(t, prompt, "TOOL_CALL:", "Should show tool call format")
	assert.Contains(t, prompt, "ARGUMENTS:", "Should show arguments format")
	assert.Contains(t, prompt, `"`, "Should show JSON format with quotes")
}

// TestCreateToolPrompt_ClearInstructions tests that the prompt gives clear guidance
func TestCreateToolPrompt_ClearInstructions(t *testing.T) {
	model := &OllamaModel{}
	
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Search for items",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should tell model when to use tools
	assert.Regexp(t, `(?i)(when|if you need|use.*tool)`, prompt, "Should explain when to use tools")
	
	// Should tell model it can respond normally without tools
	assert.Regexp(t, `(?i)(don't need|not necessary|respond normally)`, prompt, "Should allow normal responses")
	
	// Should emphasize the model has access to these capabilities
	assert.Regexp(t, `(?i)(you have access|available|can use)`, prompt, "Should emphasize tool availability")
}

// TestCreateToolPrompt_SearchExample tests the specific search tool that user is experiencing issues with
func TestCreateToolPrompt_SearchExample(t *testing.T) {
	model := &OllamaModel{}
	
	// This is the actual search tool from local-memory MCP server
	tools := []ToolDefinition{
		{
			Name:        "search",
			Description: "Unified search for memories with semantic, tag, date-range, and hybrid capabilities",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The text to search for in memory contents",
					},
					"search_type": map[string]interface{}{
						"type":    "string",
						"enum":    []interface{}{"semantic", "tags", "date_range", "hybrid"},
						"default": "semantic",
					},
					"use_ai": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable vector embeddings for semantic similarity search",
						"default":     false,
					},
				},
				"required": []interface{}{},
			},
		},
	}
	
	prompt := model.createToolPrompt(tools)
	
	// Should make it obvious this tool can search memories
	assert.Contains(t, prompt, "search", "Should include search tool")
	assert.Contains(t, prompt, "memories", "Should mention memories")
	
	// Should show all parameter options clearly
	assert.Contains(t, prompt, "query", "Should show query parameter")
	assert.Contains(t, prompt, "search_type", "Should show search_type parameter")
	assert.Contains(t, prompt, "semantic", "Should show semantic option")
	
	// When user asks about memories, model should recognize it can use this tool
	// The prompt should make the connection clear
	require.NotEmpty(t, prompt, "Prompt should not be empty")
}
