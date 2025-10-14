package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

// ToolExecutor handles tool execution with parameter validation and result processing
type ToolExecutor struct {
	registry *ToolRegistry
	logger   Logger
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(registry *ToolRegistry, logger Logger) *ToolExecutor {
	return &ToolExecutor{
		registry: registry,
		logger:   logger,
	}
}

// ExecuteResult represents the result of a tool execution
type ExecuteResult struct {
	Tool     Tool        `json:"tool"`
	Result   *ToolResult `json:"result,omitempty"`
	Error    error       `json:"error,omitempty"`
	Duration string      `json:"duration"`
}

// Execute executes a tool with the given parameters
func (e *ToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) (*ExecuteResult, error) {
	start := ctx.Value("start_time")
	if start == nil {
		start = "unknown"
	}
	
	// Get the tool from registry
	tool, exists := e.registry.GetTool(toolName)
	if !exists {
		return &ExecuteResult{
			Error:    fmt.Errorf("tool '%s' not found", toolName),
			Duration: "0ms",
		}, fmt.Errorf("tool '%s' not found", toolName)
	}
	
	e.logger.Info("Executing tool %s from server %s", toolName, tool.ServerName)
	
	// Validate parameters against schema
	if err := e.validateParameters(tool, params); err != nil {
		return &ExecuteResult{
			Tool:     tool,
			Error:    fmt.Errorf("parameter validation failed: %w", err),
			Duration: "0ms",
		}, err
	}
	
	// Get the server client
	client, exists := e.registry.GetServer(tool.ServerName)
	if !exists {
		return &ExecuteResult{
			Tool:     tool,
			Error:    fmt.Errorf("server '%s' not found", tool.ServerName),
			Duration: "0ms",
		}, fmt.Errorf("server '%s' not found", tool.ServerName)
	}
	
	// Ensure server is connected
	if !client.IsConnected() {
		if err := client.Connect(ctx); err != nil {
			return &ExecuteResult{
				Tool:     tool,
				Error:    fmt.Errorf("failed to connect to server: %w", err),
				Duration: "0ms",
			}, err
		}
	}
	
	// Execute the tool
	result, err := client.CallTool(ctx, toolName, params)
	if err != nil {
		e.logger.Error("Tool execution failed %s: %v", toolName, err)
		return &ExecuteResult{
			Tool:     tool,
			Error:    err,
			Duration: fmt.Sprintf("%v", start),
		}, err
	}
	
	e.logger.Info("Tool executed successfully %s content_count %d", toolName, len(result.Content))
	
	return &ExecuteResult{
		Tool:     tool,
		Result:   result,
		Duration: fmt.Sprintf("%v", start),
	}, nil
}

// validateParameters validates tool parameters against the JSON schema
func (e *ToolExecutor) validateParameters(tool Tool, params map[string]interface{}) error {
	schema := tool.InputSchema
	if schema == nil {
		// No schema means no validation required
		return nil
	}
	
	// Get the properties from the schema
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil // No properties to validate
	}
	
	// Get required fields
	required := make(map[string]bool)
	if reqFields, ok := schema["required"].([]interface{}); ok {
		for _, field := range reqFields {
			if fieldName, ok := field.(string); ok {
				required[fieldName] = true
			}
		}
	}
	
	// Validate required fields are present
	for fieldName := range required {
		if _, exists := params[fieldName]; !exists {
			return fmt.Errorf("required parameter '%s' is missing", fieldName)
		}
	}
	
	// Validate each parameter
	for paramName, paramValue := range params {
		propSchema, exists := properties[paramName]
		if !exists {
			return fmt.Errorf("unknown parameter '%s'", paramName)
		}
		
		if err := e.validateParameter(paramName, paramValue, propSchema); err != nil {
			return err
		}
	}
	
	return nil
}

// validateParameter validates a single parameter against its schema
func (e *ToolExecutor) validateParameter(name string, value interface{}, schema interface{}) error {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil // Can't validate without proper schema
	}
	
	// Get the expected type
	expectedType, ok := schemaMap["type"].(string)
	if !ok {
		return nil // No type specified
	}
	
	// Validate type
	if err := e.validateType(name, value, expectedType); err != nil {
		return err
	}
	
	// Validate enum constraints
	if enum, ok := schemaMap["enum"].([]interface{}); ok {
		if err := e.validateEnum(name, value, enum); err != nil {
			return err
		}
	}
	
	// Validate string constraints
	if expectedType == "string" {
		if err := e.validateStringConstraints(name, value, schemaMap); err != nil {
			return err
		}
	}
	
	// Validate number constraints
	if expectedType == "number" || expectedType == "integer" {
		if err := e.validateNumberConstraints(name, value, schemaMap); err != nil {
			return err
		}
	}
	
	return nil
}

// validateType validates the basic type of a parameter
func (e *ToolExecutor) validateType(name string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' must be a string, got %T", name, value)
		}
	case "number":
		switch value := value.(type) {
		case float64, float32, int, int32, int64:
			// Valid number types
		default:
			return fmt.Errorf("parameter '%s' must be a number, got %T", name, value)
		}
	case "integer":
		switch value := value.(type) {
		case int, int32, int64:
			// Valid integer types
		case float64:
			// Check if it's actually an integer
			if value != float64(int64(value)) {
				return fmt.Errorf("parameter '%s' must be an integer, got float", name)
			}
		default:
			return fmt.Errorf("parameter '%s' must be an integer, got %T", name, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' must be a boolean, got %T", name, value)
		}
	case "array":
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return fmt.Errorf("parameter '%s' must be an array, got %T", name, value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("parameter '%s' must be an object, got %T", name, value)
		}
	}
	
	return nil
}

// validateEnum validates that a value is in the allowed enum values
func (e *ToolExecutor) validateEnum(name string, value interface{}, enum []interface{}) error {
	for _, allowedValue := range enum {
		if value == allowedValue {
			return nil
		}
	}
	
	return fmt.Errorf("parameter '%s' must be one of %v, got %v", name, enum, value)
}

// validateStringConstraints validates string-specific constraints
func (e *ToolExecutor) validateStringConstraints(name string, value interface{}, schema map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return nil // Type validation should have caught this
	}
	
	// Min length
	if minLen, ok := schema["minLength"].(float64); ok {
		if len(str) < int(minLen) {
			return fmt.Errorf("parameter '%s' must be at least %d characters long", name, int(minLen))
		}
	}
	
	// Max length
	if maxLen, ok := schema["maxLength"].(float64); ok {
		if len(str) > int(maxLen) {
			return fmt.Errorf("parameter '%s' must be at most %d characters long", name, int(maxLen))
		}
	}
	
	// Pattern (basic regex - would need regex package for full support)
	if pattern, ok := schema["pattern"].(string); ok {
		// This is a simplified pattern check - in production, use regexp package
		e.logger.Debug("Pattern validation not fully implemented for parameter %s pattern %s", name, pattern)
	}
	
	return nil
}

// validateNumberConstraints validates number-specific constraints
func (e *ToolExecutor) validateNumberConstraints(name string, value interface{}, schema map[string]interface{}) error {
	var num float64
	
	switch v := value.(type) {
	case float64:
		num = v
	case float32:
		num = float64(v)
	case int:
		num = float64(v)
	case int32:
		num = float64(v)
	case int64:
		num = float64(v)
	default:
		return nil // Type validation should have caught this
	}
	
	// Minimum
	if min, ok := schema["minimum"].(float64); ok {
		if num < min {
			return fmt.Errorf("parameter '%s' must be at least %g", name, min)
		}
	}
	
	// Maximum
	if max, ok := schema["maximum"].(float64); ok {
		if num > max {
			return fmt.Errorf("parameter '%s' must be at most %g", name, max)
		}
	}
	
	// Exclusive minimum
	if min, ok := schema["exclusiveMinimum"].(float64); ok {
		if num <= min {
			return fmt.Errorf("parameter '%s' must be greater than %g", name, min)
		}
	}
	
	// Exclusive maximum
	if max, ok := schema["exclusiveMaximum"].(float64); ok {
		if num >= max {
			return fmt.Errorf("parameter '%s' must be less than %g", name, max)
		}
	}
	
	return nil
}

// FormatResult formats a tool execution result for display
func (e *ToolExecutor) FormatResult(result *ExecuteResult) string {
	if result.Error != nil {
		return fmt.Sprintf("Error: %s", result.Error.Error())
	}
	
	if result.Result == nil {
		return "No result"
	}
	
	var output []string
	for _, content := range result.Result.Content {
		switch content.Type {
		case "text":
			output = append(output, content.Text)
		case "json":
			// Pretty print JSON
			if data, err := json.MarshalIndent(content.Data, "", "  "); err == nil {
				output = append(output, string(data))
			} else {
				output = append(output, content.Data)
			}
		default:
			output = append(output, fmt.Sprintf("[%s] %s", content.Type, content.Text))
		}
	}
	
	if len(output) == 0 {
		return "Empty result"
	}
	
	return fmt.Sprintf("%s", output[0]) // Return first content for now
}