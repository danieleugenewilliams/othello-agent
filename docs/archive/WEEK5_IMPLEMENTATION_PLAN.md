# Week 5 Implementation Plan - Tool Response & UI Improvements

**Status**: Planning Phase
**Created**: 2025-10-12

## Overview

Week 4 successfully implemented tool calling. Week 5 focuses on improving the user experience:
1. Better tool result formatting
2. Improved UI for tool calls (collapsible display)
3. Bug fixes

## Problem Statements

### 1. Tool Results Too Verbose
**Current**: Model shows raw JSON dumps or overly detailed summaries
**Desired**: Model extracts key information and presents it naturally

**Example**:
- Current: "The analysis found: Your keyword search matched 10 documents across different session IDs..."
- Better: "I found 3 relevant memories about API architecture: [summary of key points]"

### 2. Tool Call Display Location
**Current**: Tool execution logs appear below the chat box
**Desired**: Tool calls appear inline in conversation as collapsible dropdowns (like Claude Desktop)

**Reference**: See Claude Desktop screenshot - tool calls show as expandable sections within the conversation thread

### 3. Test Failure: Tools Persist After Stop
**Current**: `TestIntegration_ErrorHandlingAndRecovery` fails because tools remain in registry after agent stops
**Desired**: Registry should be cleared when agent stops

## Implementation Tasks

### Task 1: Improve Tool Result Processing

**Objective**: Add post-processing step to extract key information from tool results

**Approach**:
1. Add `ProcessToolResult()` function that takes raw tool result + original query
2. Use model to summarize/extract relevant information
3. Return concise, natural language response
4. Keep raw result available for debugging

**Files to modify**:
- `internal/agent/agent.go` - Add result processing
- `internal/agent/result_processor.go` (new) - Processing logic

**Success Criteria**:
- Model presents tool results naturally
- User gets actionable information
- Raw results still accessible for debugging

### Task 2: Inline Collapsible Tool Display

**Objective**: Show tool calls as collapsible sections in chat thread (like Claude Desktop)

**Approach**:
1. Add tool call rendering in ChatView
2. Create collapsible component with lipgloss
3. Show: Tool name, parameters, execution time, result preview
4. Expand/collapse with keyboard (e.g., 't' key or Enter)
5. Style similar to Claude Desktop (gray box, rounded corners)

**Files to modify**:
- `internal/tui/chat_view.go` - Add rendering logic
- `internal/tui/tool_call_component.go` (new) - Collapsible component
- `internal/tui/messages.go` - Update message types

**Design**:
```
[12:34] Assistant:
Let me search for that...

┌─ 🔧 search ─────────────────────────────┐
│ query: "API architecture"               │
│ search_type: "semantic"                 │
│ ✓ Completed in 243ms                    │
│ > Press Enter to expand results         │
└─────────────────────────────────────────┘

Based on what I found, here are 3 relevant memories...
```

**Success Criteria**:
- Tool calls appear inline in conversation
- Can expand/collapse to see details
- Doesn't clutter chat with raw data
- Keyboard accessible

### Task 3: Fix Tool Registry Cleanup

**Objective**: Ensure tools are cleared from registry when agent stops

**Approach**:
1. Add `ClearTools()` method to ToolRegistry
2. Call it in `Agent.Stop()`
3. Update test expectations

**Files to modify**:
- `internal/mcp/registry.go` - Add ClearTools()
- `internal/agent/agent.go` - Call on stop
- `internal/agent/integration_test.go` - Update test

**Success Criteria**:
- `TestIntegration_ErrorHandlingAndRecovery` passes
- Tools cleared when agent stops
- No memory leaks from persisted tools

## Implementation Order

1. **Task 3** (Bug fix) - Quick win, unblocks tests
2. **Task 2** (UI) - High impact, improves UX immediately  
3. **Task 1** (Results) - Enhancement, can iterate over time

## Testing Strategy

### Task 1: Result Processing
- Unit tests for ProcessToolResult()
- Compare raw vs processed output
- Test with different tool types (search, store, analysis)

### Task 2: UI Components
- Visual testing in TUI
- Keyboard interaction testing
- Test with long results, short results, errors

### Task 3: Registry Cleanup
- Run existing integration test
- Verify tools cleared after Stop()
- Check for memory leaks

## Success Metrics

- Tools calls are intuitive to users
- Results are actionable, not overwhelming
- All tests pass
- UI feels polished like Claude Desktop

## Future Enhancements (Post-Week 5)

- Streaming tool results
- Multiple tool calls in parallel
- Tool call history view
- Export tool calls/results
- Tool usage analytics
