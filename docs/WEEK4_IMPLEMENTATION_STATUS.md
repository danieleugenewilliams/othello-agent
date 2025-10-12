# Week 4 Implementation Status - Model-Tool Integration

**Goal**: Fix model tool calling to use correct parameters and prevent errors like "unknown parameter 'concept'"

## Problem Statement

From user screenshot: Model attempts to call `stats` tool with invalid parameter:
```
Tool execution failed for stats: unknown parameter 'concept'
```

**Root Cause**: Model is generating tool calls without knowing the exact parameter schemas. It's guessing based on tool descriptions alone.

## Current State Analysis

### How Tool Calling Works Now

1. **Tool Definitions** (`internal/model/ollama.go:createToolPrompt`):
   - Tools passed to model as text descriptions
   - Parameters shown as `map[string]interface{}` (not detailed schema)
   - Model has to guess parameter names and types

2. **Tool Call Parsing** (`internal/model/ollama.go:parseToolCalls`):
   - Model responds with `TOOL_CALL:` and `ARGUMENTS:` format
   - Arguments parsed from JSON
   - **NO VALIDATION** against actual tool schema

3. **Tool Execution** (`internal/agent/mcp_manager.go`):
   - Arguments passed directly to MCP server
   - Server rejects invalid parameters
   - Error shown to user (not helpful)

### The Gap

**MCP servers provide detailed schemas** via `Tool.InputSchema`:
```json
{
  "type": "object",
  "properties": {
    "query": {"type": "string", "description": "Search query"},
    "search_type": {"type": "string", "enum": ["semantic", "tags"]}
  },
  "required": ["query"]
}
```

**But we're not using this information!** We just show:
```
- stats: Get statistics
  Parameters: map[inputSchema:map[...]]
```

## Week 4 Implementation Plan (TDD)

### Task 1: Convert MCP Tool Schemas to Model ToolDefinitions ✅

**Goal**: Format tool schemas properly for model consumption

**Test File**: `internal/agent/tool_conversion_test.go` (NEW)

**Test Cases**:
```go
func TestConvertMCPToolToDefinition(t *testing.T) {
    // GIVEN: MCP tool with JSON schema
    mcpTool := mcp.Tool{
        Name: "search",
        Description: "Search memories",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query": map[string]interface{}{
                    "type": "string",
                    "description": "Search query",
                },
                "search_type": map[string]interface{}{
                    "type": "string",
                    "enum": []interface{}{"semantic", "tags", "date_range"},
                },
            },
            "required": []interface{}{"query"},
        },
    }
    
    // WHEN: Convert to model definition
    definition := ConvertMCPToolToDefinition(mcpTool)
    
    // THEN: Should have proper structure
    assert.Equal(t, "search", definition.Name)
    assert.Contains(t, definition.Parameters, "type")
    assert.Contains(t, definition.Parameters, "properties")
    assert.Contains(t, definition.Parameters, "required")
}
```

**Implementation** (`internal/agent/tool_conversion.go` - NEW):
- Extract InputSchema from mcp.Tool
- Pass directly to model.ToolDefinition
- Model sees full JSON Schema specification

---

### Task 2: Improve Tool Prompt with Schema Details

**Goal**: Help model understand exact parameters

**Test File**: `internal/model/ollama_test.go`

**Test Case**:
```go
func TestOllamaModel_CreateToolPrompt_WithSchema(t *testing.T) {
    tools := []ToolDefinition{
        {
            Name: "stats",
            Description: "Get statistics",
            Parameters: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "stats_type": map[string]interface{}{
                        "type": "enum",
                        "enum": []string{"session", "domain", "category"},
                    },
                },
                "required": []string{"stats_type"},
            },
        },
    }
    
    model := NewOllamaModel("http://localhost:11434", "qwen2.5:3b")
    prompt := model.createToolPrompt(tools)
    
    // Should explain parameters clearly
    assert.Contains(t, prompt, "stats_type")
    assert.Contains(t, prompt, "required")
    assert.Contains(t, prompt, "session")
    assert.Contains(t, prompt, "domain")
    assert.Contains(t, prompt, "category")
}
```

**Implementation** (`internal/model/ollama.go:createToolPrompt`):
- Parse `Parameters` field (JSON Schema)
- Format each parameter with type, description, enum values
- Show required vs optional clearly
- Give examples for complex parameters

---

### Task 3: Validate Tool Calls Before Execution

**Goal**: Catch invalid parameters before sending to MCP server

**Test File**: `internal/agent/tool_validation_test.go` (NEW)

**Test Cases**:
```go
func TestValidateToolCall_ValidParameters(t *testing.T) {
    schema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "query": map[string]interface{}{"type": "string"},
        },
        "required": []interface{}{"query"},
    }
    
    args := map[string]interface{}{
        "query": "test search",
    }
    
    err := ValidateToolCall("search", args, schema)
    assert.NoError(t, err)
}

func TestValidateToolCall_MissingRequired(t *testing.T) {
    schema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "query": map[string]interface{}{"type": "string"},
        },
        "required": []interface{}{"query"},
    }
    
    args := map[string]interface{}{} // Missing required param
    
    err := ValidateToolCall("search", args, schema)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "required parameter 'query'")
}

func TestValidateToolCall_UnknownParameter(t *testing.T) {
    schema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "query": map[string]interface{}{"type": "string"},
        },
    }
    
    args := map[string]interface{}{
        "query": "test",
        "concept": "invalid",  // Unknown parameter!
    }
    
    err := ValidateToolCall("search", args, schema)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unknown parameter 'concept'")
}
```

**Implementation** (`internal/agent/tool_validation.go` - NEW):
- Check required parameters present
- Reject unknown parameters
- Validate parameter types
- Return helpful error messages

---

### Task 4: Integration Test

**Goal**: Verify model makes valid tool calls end-to-end

**Test File**: `internal/agent/agent_integration_test.go`

**Test Case**:
```go
func TestAgent_ModelMakesValidToolCalls(t *testing.T) {
    // GIVEN: Agent with real local-memory MCP server
    agent := setupTestAgent(t)
    
    // WHEN: Ask model to search memories
    response := agent.ProcessQuery(context.Background(), 
        "Search for memories about Go programming")
    
    // THEN: Should make valid tool call
    assert.NotNil(t, response.ToolCalls)
    assert.Len(t, response.ToolCalls, 1)
    
    toolCall := response.ToolCalls[0]
    assert.Equal(t, "search", toolCall.Name)
    
    // Should have valid parameters
    assert.Contains(t, toolCall.Arguments, "query")
    assert.Contains(t, toolCall.Arguments, "search_type")
    
    // Should NOT have invalid parameters
    assert.NotContains(t, toolCall.Arguments, "concept")
}
```

---

## Success Criteria

✅ Task 1: Tool schemas properly converted from MCP to model format  
✅ Task 2: Model prompt includes detailed parameter information  
✅ Task 3: Invalid tool calls caught before execution  
✅ Task 4: Integration test shows model making valid calls  
✅ Manual test: Ask model to use stats tool, no "unknown parameter" errors  

## Implementation Order

1. **Session 1** (45 min): Task 1 - Schema conversion (TDD)
2. **Session 2** (45 min): Task 2 - Improved prompts (TDD)
3. **Session 3** (45 min): Task 3 - Validation (TDD)
4. **Session 4** (30 min): Task 4 - Integration test
5. **Session 5** (30 min): Manual testing and refinement

## Current Status

- [x] Week 4 planning complete
- [ ] Task 1: Schema conversion
- [ ] Task 2: Improved prompts
- [ ] Task 3: Validation
- [ ] Task 4: Integration test
- [ ] Manual verification
