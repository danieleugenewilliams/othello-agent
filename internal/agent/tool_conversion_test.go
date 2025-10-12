package agent

import (
	"testing"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertMCPToolToDefinition tests converting MCP tool to model definition
func TestConvertMCPToolToDefinition(t *testing.T) {
	// GIVEN: MCP tool with JSON schema
	mcpTool := mcp.Tool{
		Name:        "search",
		Description: "Search through stored memories",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query text",
				},
				"search_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of search to perform",
					"enum":        []interface{}{"semantic", "tags", "date_range"},
				},
			},
			"required": []interface{}{"query"},
		},
	}

	// WHEN: Convert to model definition
	definition := ConvertMCPToolToDefinition(mcpTool)

	// THEN: Should preserve name and description
	assert.Equal(t, "search", definition.Name)
	assert.Equal(t, "Search through stored memories", definition.Description)

	// THEN: Should have complete parameter schema
	require.NotNil(t, definition.Parameters)
	assert.Equal(t, "object", definition.Parameters["type"])

	// THEN: Should have properties
	properties, ok := definition.Parameters["properties"].(map[string]interface{})
	require.True(t, ok, "Properties should be a map")
	require.Contains(t, properties, "query")
	require.Contains(t, properties, "search_type")

	// THEN: Should have required fields
	required, ok := definition.Parameters["required"].([]interface{})
	require.True(t, ok, "Required should be an array")
	assert.Contains(t, required, "query")
}

// TestConvertMCPToolToDefinition_StatsExample tests the problematic stats tool
func TestConvertMCPToolToDefinition_StatsExample(t *testing.T) {
	// GIVEN: The stats tool that caused the 'concept' error
	mcpTool := mcp.Tool{
		Name:        "stats",
		Description: "Get statistical information",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"stats_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of statistics to retrieve",
					"enum":        []interface{}{"session", "domain", "category"},
				},
			},
			"required": []interface{}{"stats_type"},
		},
	}

	// WHEN: Convert to model definition
	definition := ConvertMCPToolToDefinition(mcpTool)

	// THEN: Should clearly define stats_type parameter
	properties, ok := definition.Parameters["properties"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, properties, "stats_type")

	// THEN: Should NOT suggest 'concept' as a parameter
	assert.NotContains(t, properties, "concept")

	// THEN: Should have enum values
	statsType, ok := properties["stats_type"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, statsType, "enum")
}

// TestConvertMCPToolsToDefinitions tests batch conversion
func TestConvertMCPToolsToDefinitions(t *testing.T) {
	// GIVEN: Multiple MCP tools
	mcpTools := []mcp.Tool{
		{
			Name:        "search",
			Description: "Search memories",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			Name:        "store_memory",
			Description: "Store a new memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	// WHEN: Convert all tools
	definitions := ConvertMCPToolsToDefinitions(mcpTools)

	// THEN: Should have all tools
	assert.Len(t, definitions, 2)
	assert.Equal(t, "search", definitions[0].Name)
	assert.Equal(t, "store_memory", definitions[1].Name)
}

// TestConvertMCPToolToDefinition_EmptySchema tests tool with no schema
func TestConvertMCPToolToDefinition_EmptySchema(t *testing.T) {
	// GIVEN: Tool with no input schema
	mcpTool := mcp.Tool{
		Name:        "simple_tool",
		Description: "A tool with no parameters",
		InputSchema: nil,
	}

	// WHEN: Convert to definition
	definition := ConvertMCPToolToDefinition(mcpTool)

	// THEN: Should still have name and description
	assert.Equal(t, "simple_tool", definition.Name)
	assert.Equal(t, "A tool with no parameters", definition.Description)

	// THEN: Should have empty parameters or default object schema
	if definition.Parameters != nil {
		assert.Equal(t, "object", definition.Parameters["type"])
	}
}
