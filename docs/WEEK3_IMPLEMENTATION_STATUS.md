# Week 3 Implementation Status
## Test-Driven Development Progress

**Date:** October 11, 2025  
**Phase:** Week 3 - Chat-Tool Integration  
**Approach:** Test-Driven Development (TDD)  

---

## Current Implementation Assessment

### ✅ Phase 1: Agent-MCP Integration (COMPLETED)

**Implemented:**
- ✅ MCP Registry initialized in Agent
- ✅ MCP Manager with server lifecycle management
- ✅ Tool Executor integrated
- ✅ Update callback system for broadcasting changes
- ✅ Server configuration loading from mcp.json

**Evidence:**
- `internal/agent/agent.go` - Lines 45-70: MCP components initialized
- `internal/agent/mcp_manager.go` - Full server lifecycle management
- Agent `Start()` method loads and connects to MCP servers

### ✅ Phase 2: TUI-MCP Integration (COMPLETED)

**Implemented:**
- ✅ Application has agent reference
- ✅ Server view displays real MCP servers
- ✅ Tool view shows actual MCP tools
- ✅ Agent interface exposed to TUI

**Evidence:**
- `internal/tui/application.go` - `NewApplicationWithAgent()` function
- `internal/tui/server_view.go` - Uses agent to get server status
- `internal/tui/tool_view.go` - Displays real tools from agent

### 🔄 Phase 3: Chat-Tool Integration (IN PROGRESS - Week 3)

**What We Need:**

1. **Tool Call Detection** ❌ NOT IMPLEMENTED
   - ChatView needs to detect when model wants to call a tool
   - Parse tool call format from model responses
   
2. **Tool Execution from Chat** ❌ NOT IMPLEMENTED
   - Execute tools asynchronously when model requests them
   - Display "Executing tool..." status in chat
   
3. **Tool Result Display** ⚠️ PARTIALLY IMPLEMENTED
   - Display tool execution results in chat
   - Format results in user-friendly way
   - Current: Can display messages, but no tool-specific formatting

4. **Tool Result Synthesis** ❌ NOT IMPLEMENTED
   - Send tool results back to model
   - Model synthesizes natural language response
   - Display final response to user

### ❌ Phase 4: Model-Tool Integration (NOT STARTED - Week 4)

**What We Need:**

1. **GenerateWithTools Method** ❌ NOT IMPLEMENTED
   - Extend model interface to accept tool descriptions
   - Build system prompts with tool information
   
2. **Tool Call Parsing** ❌ NOT IMPLEMENTED
   - Parse tool calls from model responses
   - Extract tool name and arguments
   
3. **Tool Format System Prompt** ❌ NOT IMPLEMENTED
   - Create standardized format for tool calling
   - Teach model how to request tools

---

## Week 3 TDD Implementation Plan

### Goal: Enable Tool Execution from Chat Messages

Following TDD principles: **Red → Green → Refactor**

### Task 1: Add MCP Message Types to TUI ✅ COMPLETED

**Test File:** `internal/tui/messages_test.go` (CREATED)

**Implemented Messages:**
```go
type MCPToolExecutingMsg struct {
    ToolName string
    Params   map[string]interface{}
}

type MCPToolExecutedMsg struct {
    ToolName string
    Result   *mcp.ExecuteResult
    Error    error
}
```

**Completed Steps:**
1. ✅ Wrote test: `TestMCPMessages_ToolExecution`
2. ✅ Implemented message types in `messages.go`
3. ✅ Tests pass → GREEN

---

### Task 2: ChatView Handles Tool Execution Messages ✅ COMPLETED

**Test File:** `internal/tui/chat_view_mcp_test.go` (CREATED)

**Implemented Test Cases:**
```go
func TestChatView_HandlesMCPToolExecutingMsg(t *testing.T)
func TestChatView_HandlesMCPToolExecutedMsg_Success(t *testing.T)
func TestChatView_HandlesMCPToolExecutedMsg_Error(t *testing.T)
func TestChatView_HandlesMCPToolExecutedMsg_MCPError(t *testing.T)
func TestChatView_StoresToolMessages(t *testing.T)
```

**Completed Steps:**
1. ✅ Wrote comprehensive failing tests (RED)
2. ✅ Implemented tool message handling in ChatView.Update()
3. ✅ Added tool execution display logic:
   - MCPToolExecutingMsg: Shows "Executing tool: X..."
   - MCPToolExecutedMsg (success): Displays result text
   - MCPToolExecutedMsg (error): Shows error message
   - MCPToolExecutedMsg (MCP error): Shows MCP-level errors
4. ✅ All tests pass → GREEN
5. ✅ Code is clean and well-structured

---

### Task 3: Hierarchical MCP/Tools Navigation ⏭️ REVISED

**Goal:** Make /tools view accessible from /mcp server view, not standalone

**User Feedback:** 
> "The /tools view should be under the /mcp view. When I use /mcp to view the list of MCP servers, if I click on that MCP server, I should see the /tools view (list of tools for that server)."

**Implementation Plan:**

1. **ServerView Enhancement**
   - Add server selection functionality (Enter key or click)
   - Send ServerSelectedMsg when server is selected
   - Navigate to ToolView with selected server context

2. **ToolView Enhancement**
   - Accept selected server from navigation
   - Filter tools to show only those from selected server
   - Add breadcrumb showing "MCP > {server-name} > Tools"
   - Add back navigation (Esc returns to ServerView)

3. **Application Coordination**
   - Handle ServerSelectedMsg to switch views
   - Pass server context to ToolView
   - Maintain navigation state

4. **Remove Manual Tool Execution**
   - Tool execution will come through model (Week 4)
   - No need for `/tool` command implementation

---

### Task 4: Agent ExecuteTool Returns TUI-Friendly Result

**Test File:** `internal/agent/agent_test.go`

**Test Case:**
```go
func TestAgent_ExecuteTool_ReturnsTUIResult(t *testing.T) {
    // GIVEN: Agent with mock MCP server
    // WHEN: ExecuteTool called
    // THEN: Returns ToolExecutionResult with formatted content
}
```

**Steps:**
1. Write test (RED)
2. Implement ExecuteTool method
3. Add result formatting
4. Run test → GREEN

---

## Implementation Order

**Session 1: Foundation (30 min)**
1. ✅ Review Week 3/4 plan
2. ✅ Create this status document
3. ⏭️ Add MCP message types
4. ⏭️ Write message type tests

**Session 2: Chat-Tool Display (1 hour)**
1. Write ChatView tool execution tests
2. Implement tool execution message handling
3. Add tool result formatting
4. Test with mock agent

**Session 3: Tool Execution Integration (1 hour)**
1. Implement `/tool` command
2. Connect to agent.ExecuteTool()
3. Add async tool execution
4. Test end-to-end flow

**Session 4: Polish & Validation (30 min)**
1. Run all tests
2. Manual testing with real MCP server
3. Fix any issues
4. Document learnings

---

## Success Criteria for Week 3

- [ ] All tests pass (>80% coverage)
- [ ] Can manually execute tools via `/tool` command
- [ ] Tool execution displays in chat with status
- [ ] Tool results formatted and readable
- [ ] Tool errors displayed gracefully
- [ ] No regression in existing functionality

---

## Notes

**The `concept` Parameter Issue:**
This is likely a Week 4 problem where the model is incorrectly generating tool call parameters. We'll address this when we implement Phase 4: Model-Tool Integration. For now, we'll focus on manual tool execution to prove the infrastructure works.

**Testing Strategy:**
- Use mock agent for ChatView tests
- Use mock MCP server for Agent tests
- Integration test with real local-memory server
- Manual testing in TUI for UX validation

**Next Steps After Week 3:**
Week 4 will focus on teaching the model how to properly call tools, which will fix the parameter issues we're seeing.
