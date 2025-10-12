package agent

import (
	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// ConvertMCPToolToDefinition converts an MCP tool to a model tool definition
// This ensures the model receives complete JSON schema information about tool parameters
func ConvertMCPToolToDefinition(mcpTool mcp.Tool) model.ToolDefinition {
	definition := model.ToolDefinition{
		Name:        mcpTool.Name,
		Description: mcpTool.Description,
		Parameters:  make(map[string]interface{}),
	}

	// If the tool has an input schema, use it directly
	if mcpTool.InputSchema != nil {
		// MCP tools use JSON Schema for InputSchema
		// Pass it through to the model as-is
		definition.Parameters = mcpTool.InputSchema
	} else {
		// Default to empty object schema if no schema provided
		definition.Parameters = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	return definition
}

// ConvertMCPToolsToDefinitions converts a slice of MCP tools to model definitions
func ConvertMCPToolsToDefinitions(mcpTools []mcp.Tool) []model.ToolDefinition {
	definitions := make([]model.ToolDefinition, len(mcpTools))
	for i, tool := range mcpTools {
		definitions[i] = ConvertMCPToolToDefinition(tool)
	}
	return definitions
}
