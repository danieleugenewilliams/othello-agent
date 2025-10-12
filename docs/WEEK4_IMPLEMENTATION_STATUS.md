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

### Task 1: Convert MCP Tool Schemas to Model Definitions (TDD)

**Status**: ✅ COMPLETE

**Objective**: Ensure the model receives complete and accurate tool schemas by converting MCP InputSchema (JSON Schema format) directly to model.ToolDefinition format.

**Approach**:
1. ✅ RED: Write tests for schema conversion function
2. ✅ GREEN: Implement conversion that passes MCP InputSchema unchanged
3. ✅ INTEGRATE: Wire up in Agent.GetMCPToolsAsDefinitions()

**Implementation**:
- Created `internal/agent/tool_conversion.go` with schema conversion functions
- Created `internal/agent/tool_conversion_test.go` with 4 passing tests
- Updated `Agent.GetMCPToolsAsDefinitions()` to use new conversion
- All tests pass, build successful

**Validation (TUI Test)**:
- Asked: "Can you see memories for RDS architecture?"
- Result: Model responded "Let me help you with that using the search tool..."
- **Success**: Model now recognizes tools are available
- **Issue**: Tool execution returned no data (likely wrong parameters - addressed in Task 2-3)

---

### Task 2: Improve Tool Prompt Generation (TDD)

**Status**: 🔄 RED Phase - Tests Written

**Objective**: Format tool schemas in human-readable way so model understands parameters clearly.

**Approach**:
1. 🔄 RED: Write tests for improved `createToolPrompt` formatting
2. ⏭️ GREEN: Implement readable schema formatting
3. ⏭️ Include: parameter types, required vs optional, enum values, defaults, descriptions

**Progress**:
- Created `internal/model/tool_prompt_test.go` with comprehensive test cases
- Tests cover: basic formatting, required/optional params, enums, defaults, multiple tools
- Ready for GREEN phase implementation

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
