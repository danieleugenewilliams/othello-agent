package agent

import (
	"testing"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateToolCall_RequiredParameters tests that required parameters are enforced
func TestValidateToolCall_RequiredParameters(t *testing.T) {
	tool := mcp.Tool{
		Name:        "search",
		Description: "Search for items",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"query"},
		},
	}
	
	tests := []struct {
		name      string
		toolCall  model.ToolCall
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid - has required parameter",
			toolCall: model.ToolCall{
				Name: "search",
				Arguments: map[string]interface{}{
					"query": "test",
				},
			},
			wantError: false,
		},
		{
			name: "invalid - missing required parameter",
			toolCall: model.ToolCall{
				Name:      "search",
				Arguments: map[string]interface{}{},
			},
			wantError: true,
			errorMsg:  "missing required parameter: query",
		},
		{
			name: "invalid - nil arguments",
			toolCall: model.ToolCall{
				Name:      "search",
				Arguments: nil,
			},
			wantError: true,
			errorMsg:  "missing required parameter: query",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolCall(tt.toolCall, tool)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateToolCall_UnknownParameters tests that unknown parameters are rejected
func TestValidateToolCall_UnknownParameters(t *testing.T) {
	tool := mcp.Tool{
		Name: "search",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{},
		},
	}
	
	toolCall := model.ToolCall{
		Name: "search",
		Arguments: map[string]interface{}{
			"query":   "test",
			"invalid": "parameter", // This parameter doesn't exist in schema
		},
	}
	
	err := ValidateToolCall(toolCall, tool)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown parameter: invalid")
}

// TestValidateToolCall_TypeValidation tests basic type checking
func TestValidateToolCall_TypeValidation(t *testing.T) {
	tool := mcp.Tool{
		Name: "search",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type": "integer",
				},
				"query": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}
	
	tests := []struct {
		name      string
		arguments map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid types",
			arguments: map[string]interface{}{
				"query": "test",
				"limit": 10,
			},
			wantError: false,
		},
		{
			name: "invalid - string instead of integer",
			arguments: map[string]interface{}{
				"query": "test",
				"limit": "not a number",
			},
			wantError: true,
			errorMsg:  "parameter 'limit' should be integer",
		},
		{
			name: "invalid - integer instead of string",
			arguments: map[string]interface{}{
				"query": 123,
			},
			wantError: true,
			errorMsg:  "parameter 'query' should be string",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCall := model.ToolCall{
				Name:      "search",
				Arguments: tt.arguments,
			}
			
			err := ValidateToolCall(toolCall, tool)
			
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateToolCall_EnumValidation tests that enum constraints are enforced
func TestValidateToolCall_EnumValidation(t *testing.T) {
	tool := mcp.Tool{
		Name: "stats",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"stats_type": map[string]interface{}{
					"type": "string",
					"enum": []interface{}{"session", "domain", "category"},
				},
			},
		},
	}
	
	tests := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{
			name:      "valid enum value",
			value:     "session",
			wantError: false,
		},
		{
			name:      "invalid enum value",
			value:     "invalid",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCall := model.ToolCall{
				Name: "stats",
				Arguments: map[string]interface{}{
					"stats_type": tt.value,
				},
			}
			
			err := ValidateToolCall(toolCall, tool)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "must be one of")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateToolCall_NoSchema tests graceful handling when schema is missing
func TestValidateToolCall_NoSchema(t *testing.T) {
	tool := mcp.Tool{
		Name:        "test",
		InputSchema: nil,
	}
	
	toolCall := model.ToolCall{
		Name: "test",
		Arguments: map[string]interface{}{
			"anything": "goes",
		},
	}
	
	// Should not error when no schema is present
	err := ValidateToolCall(toolCall, tool)
	assert.NoError(t, err)
}

// TestValidateToolCall_ArrayType tests array parameter validation
func TestValidateToolCall_ArrayType(t *testing.T) {
	tool := mcp.Tool{
		Name: "search",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}
	
	tests := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{
			name:      "valid array",
			value:     []interface{}{"tag1", "tag2"},
			wantError: false,
		},
		{
			name:      "valid empty array",
			value:     []interface{}{},
			wantError: false,
		},
		{
			name:      "invalid - not an array",
			value:     "not an array",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCall := model.ToolCall{
				Name: "search",
				Arguments: map[string]interface{}{
					"tags": tt.value,
				},
			}
			
			err := ValidateToolCall(toolCall, tool)
			
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
