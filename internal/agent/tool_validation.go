package agent

import (
	"fmt"
	"reflect"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// ValidateToolCall validates a tool call against the tool's JSON schema
func ValidateToolCall(toolCall model.ToolCall, tool mcp.Tool) error {
	// If no schema, accept anything
	if tool.InputSchema == nil {
		return nil
	}
	
	schema := tool.InputSchema
	
	// Get properties and required fields from schema
	properties, _ := schema["properties"].(map[string]interface{})
	if properties == nil {
		properties = make(map[string]interface{})
	}
	
	required, _ := schema["required"].([]interface{})
	requiredMap := make(map[string]bool)
	for _, req := range required {
		if reqStr, ok := req.(string); ok {
			requiredMap[reqStr] = true
		}
	}
	
	// Check arguments exist
	if toolCall.Arguments == nil {
		toolCall.Arguments = make(map[string]interface{})
	}
	
	// Validate required parameters are present
	for paramName := range requiredMap {
		if _, exists := toolCall.Arguments[paramName]; !exists {
			return fmt.Errorf("missing required parameter: %s", paramName)
		}
	}
	
	// Validate no unknown parameters
	for paramName := range toolCall.Arguments {
		if _, exists := properties[paramName]; !exists {
			return fmt.Errorf("unknown parameter: %s (not in tool schema)", paramName)
		}
	}
	
	// Validate parameter types
	for paramName, paramValue := range toolCall.Arguments {
		paramSchema, exists := properties[paramName]
		if !exists {
			continue // Already checked above
		}
		
		paramSchemaMap, ok := paramSchema.(map[string]interface{})
		if !ok {
			continue
		}
		
		// Check type
		if err := validateType(paramName, paramValue, paramSchemaMap); err != nil {
			return err
		}
		
		// Check enum constraints
		if err := validateEnum(paramName, paramValue, paramSchemaMap); err != nil {
			return err
		}
	}
	
	return nil
}

// validateType checks if the value matches the expected type
func validateType(paramName string, value interface{}, schema map[string]interface{}) error {
	expectedType, ok := schema["type"].(string)
	if !ok {
		return nil // No type specified
	}
	
	actualType := reflect.TypeOf(value)
	if actualType == nil {
		return fmt.Errorf("parameter '%s' is null", paramName)
	}
	
	switch expectedType {
	case "string":
		if actualType.Kind() != reflect.String {
			return fmt.Errorf("parameter '%s' should be string, got %s", paramName, actualType.Kind())
		}
		
	case "integer", "number":
		kind := actualType.Kind()
		if kind != reflect.Int && kind != reflect.Int8 && kind != reflect.Int16 &&
			kind != reflect.Int32 && kind != reflect.Int64 &&
			kind != reflect.Uint && kind != reflect.Uint8 && kind != reflect.Uint16 &&
			kind != reflect.Uint32 && kind != reflect.Uint64 &&
			kind != reflect.Float32 && kind != reflect.Float64 {
			return fmt.Errorf("parameter '%s' should be integer, got %s", paramName, actualType.Kind())
		}
		
	case "boolean":
		if actualType.Kind() != reflect.Bool {
			return fmt.Errorf("parameter '%s' should be boolean, got %s", paramName, actualType.Kind())
		}
		
	case "array":
		if actualType.Kind() != reflect.Slice && actualType.Kind() != reflect.Array {
			return fmt.Errorf("parameter '%s' should be array, got %s", paramName, actualType.Kind())
		}
		
	case "object":
		if actualType.Kind() != reflect.Map {
			return fmt.Errorf("parameter '%s' should be object, got %s", paramName, actualType.Kind())
		}
	}
	
	return nil
}

// validateEnum checks if the value is one of the allowed enum values
func validateEnum(paramName string, value interface{}, schema map[string]interface{}) error {
	enumValues, ok := schema["enum"].([]interface{})
	if !ok || len(enumValues) == 0 {
		return nil // No enum constraint
	}
	
	// Check if value is in enum
	for _, allowed := range enumValues {
		if reflect.DeepEqual(value, allowed) {
			return nil
		}
	}
	
	// Value not in enum
	return fmt.Errorf("parameter '%s' must be one of %v, got %v", paramName, enumValues, value)
}
