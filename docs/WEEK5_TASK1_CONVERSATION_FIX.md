# Week 5 Task 1: Conversation Flow Fix

## Problem

The original implementation had a broken conversation flow when using tools:

1. User asks: "Can you see memories for Minecraft AI?"
2. LLM decides to use search tool
3. Tool executes and returns processed results
4. System starts **NEW** conversation asking LLM to "analyze results"
5. LLM responds: "Please provide details about what you want to do next..."

This happened because `generateFollowUpResponse()` was creating a completely new conversation with a generic system prompt, losing all context about the original user question.

## Root Cause

The tool execution flow was:
- `generateResponseWithTools()` - Initial LLM call with tools
- `ToolCallDetectedMsg` - Tool calls detected
- `executeToolCalls()` - Tools executed
- `ToolExecutionResultMsg` - Results returned
- `generateFollowUpResponse()` - **NEW CONVERSATION STARTED HERE** ❌

The conversation context (original user message, available tools, conversation history) was being discarded.

## Solution

### 1. Added Conversation Context to ChatView

```go
type ChatView struct {
    // ... existing fields ...
    
    // Conversation context for tool calling
    conversationHistory []model.Message
    currentUserMessage  string
    availableTools      []model.ToolDefinition
}
```

### 2. Enhanced ToolCallDetectedMsg

```go
type ToolCallDetectedMsg struct {
    ToolCalls           []model.ToolCall
    RequestID           string
    Response            *model.Response
    UserMessage         string              // NEW: Original user message
    ConversationHistory []model.Message     // NEW: Conversation up to this point
    Tools               []model.ToolDefinition // NEW: Available tools
}
```

### 3. Store Context When Tools Are Detected

In `generateResponseWithTools()`:
```go
if response != nil && len(response.ToolCalls) > 0 {
    return ToolCallDetectedMsg{
        // ... existing fields ...
        UserMessage:         message,          // Pass original message
        ConversationHistory: messages,         // Pass conversation
        Tools:               tools,            // Pass available tools
    }
}
```

In the `ToolCallDetectedMsg` handler:
```go
// Store conversation context for tool result processing
v.conversationHistory = msg.ConversationHistory
v.currentUserMessage = msg.UserMessage
v.availableTools = msg.Tools
```

### 4. Rewrote generateFollowUpResponse()

**Before:**
```go
// Started NEW conversation
messages := []model.Message{
    {Role: "system", Content: "Analyze the tool results..."},
    {Role: "user", Content: followUpPrompt},
}
response, err := v.model.Chat(ctx, messages, ...)
```

**After:**
```go
// Continue SAME conversation
messages := make([]model.Message, len(v.conversationHistory))
copy(messages, v.conversationHistory)  // Start with original context

// Add assistant acknowledgment
messages = append(messages, model.Message{
    Role:    "assistant",
    Content: "I'll use the available tools...",
})

// Add tool results
messages = append(messages, model.Message{
    Role:    "user",
    Content: toolResultMessage,  // Formatted results
})

// Continue conversation with full context
response, err := v.model.ChatWithTools(ctx, messages, v.availableTools, ...)
```

## Flow Comparison

### Before (Broken)
```
User: "Can you see memories for Minecraft AI?"
  ↓
LLM: [calls search tool]
  ↓
[Execute search → get results]
  ↓
NEW CONVERSATION:
System: "Analyze these results..."
User: "Results: Found 3 memories..."
  ↓
LLM: "What do you want to do with these results?" ❌
```

### After (Fixed)
```
User: "Can you see memories for Minecraft AI?"
  ↓
LLM: [calls search tool]
  ↓
[Execute search → get results]
  ↓
SAME CONVERSATION CONTINUED:
User: "Can you see memories for Minecraft AI?"
Assistant: "I'll use the available tools..."
User: "Tool results: Found 3 memories about..."
  ↓
LLM: "I found 3 memories about Minecraft AI: [natural summary]" ✅
```

## Benefits

1. **Maintains Context**: LLM remembers what the user originally asked
2. **Natural Responses**: LLM can provide direct answers using tool results
3. **Multi-Turn Support**: If more tools are needed, they're still available
4. **Proper Tool Use**: Follows standard tool-calling conversation patterns

## Testing

All existing tests pass:
- `TestChatView_HandlesMCPToolExecutingMsg`
- `TestChatView_HandlesMCPToolExecutedMsg_Success`
- `TestChatView_HandlesMCPToolExecutedMsg_Error`
- `TestChatView_HandlesMCPToolExecutedMsg_MCPError`
- `TestChatView_StoresToolMessages`

Build successful. Ready for manual TUI testing.

## Files Modified

1. `internal/tui/chat_view.go`:
   - Added conversation context fields to ChatView
   - Updated ToolCallDetectedMsg handler to store context
   - Rewrote generateFollowUpResponse to continue conversation
   - Updated generateResponseWithTools to pass context

2. `internal/tui/messages.go`:
   - Enhanced ToolCallDetectedMsg with context fields

## Next Steps

- Manual testing in TUI to verify natural responses
- Verify multi-turn tool calling works correctly
- Consider adding conversation history limits (max turns)
