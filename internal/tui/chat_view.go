package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role      string // "user", "assistant", "tool"
	Content   string
	Timestamp string
	ToolCall  *ToolCallInfo
	Error     string
}

// ToolCallInfo contains information about a tool call
type ToolCallInfo struct {
	Name   string
	Args   map[string]interface{}
	Result string
}

// ChatView handles the chat interface
type ChatView struct {
	width    int
	height   int
	styles   Styles
	keymap   KeyMap
	viewport viewport.Model
	input    textinput.Model
	messages []ChatMessage
	focused  bool
	model    model.Model
	agent    AgentInterface // Add agent for tool access
	waitingForResponse bool
	requestID string
	// Conversation context for tool calling
	conversationHistory []model.Message
	currentUserMessage  string
	availableTools      []model.ToolDefinition
}

// NewChatView creates a new chat view
func NewChatView(styles Styles, keymap KeyMap, m model.Model) *ChatView {
	return NewChatViewWithAgent(styles, keymap, m, nil)
}

// NewChatViewWithAgent creates a new chat view with agent support
func NewChatViewWithAgent(styles Styles, keymap KeyMap, m model.Model, agent AgentInterface) *ChatView {
	input := textinput.New()
	input.Placeholder = "Type a message..."
	input.Focus()
	input.CharLimit = 1000
	input.Width = 50

	vp := viewport.New(0, 0)
	vp.SetContent("")

	chatView := &ChatView{
		styles:   styles,
		keymap:   keymap,
		viewport: vp,
		input:    input,
		model:    m,
		agent:    agent,
		focused:  true,
	}
	
	// Add welcome message with command hints
	welcomeMsg := ChatMessage{
		Role:      "assistant",
		Content:   "Welcome to Othello AI Agent! 🤖\n\nQuick commands:\n• /mcp - View MCP servers\n• /tools - Browse tools\n• /help - Show help\n• /history - View chat history\n• /exit - Exit application\n\nNavigation:\n• Tab - Switch views\n• Esc - Go back\n\nOr just type naturally to chat!",
		Timestamp: time.Now().Format("15:04:05"),
	}
	chatView.AddMessage(welcomeMsg)
	
	return chatView
}

// Init initializes the chat view
func (v *ChatView) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles updates for the chat view
func (v *ChatView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ModelResponseMsg:
		// Handle model response
		if msg.ID == v.requestID {
			v.waitingForResponse = false
			if msg.Error != nil {
				// Add error message
				errorMsg := ChatMessage{
					Role:      "assistant",
					Content:   "",
					Error:     msg.Error.Error(),
					Timestamp: time.Now().Format("15:04"),
				}
				v.AddMessage(errorMsg)
			} else {
				// Add assistant response
				assistantMsg := ChatMessage{
					Role:      "assistant",
					Content:   msg.Response.Content,
					Timestamp: time.Now().Format("15:04"),
				}
				v.AddMessage(assistantMsg)
			}
		}
		return v, nil
		
	case ToolCallDetectedMsg:
		// Handle tool call detection
		if msg.RequestID == v.requestID {
			v.waitingForResponse = false
			
			// Store conversation context for tool result processing
			v.conversationHistory = msg.ConversationHistory
			v.currentUserMessage = msg.UserMessage
			v.availableTools = msg.Tools
			
			// Add a more natural assistant message
			var toolCallContent string
			if len(msg.ToolCalls) == 1 {
				toolCallContent = fmt.Sprintf("Let me help you with that using the %s tool...", msg.ToolCalls[0].Name)
			} else {
				toolNames := make([]string, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					toolNames[i] = tc.Name
				}
				toolCallContent = fmt.Sprintf("I'll use several tools to help: %s", strings.Join(toolNames, ", "))
			}
				
			assistantMsg := ChatMessage{
				Role:      "assistant",
				Content:   toolCallContent,
				Timestamp: time.Now().Format("15:04"),
			}
			v.AddMessage(assistantMsg)
			
			// Execute the tools
			return v, v.executeToolCalls(msg.ToolCalls, msg.RequestID)
		}
		return v, nil
		
	case ToolExecutionResultMsg:
		// Handle tool execution results
		if msg.RequestID == v.requestID {
			// Instead of just showing "tool completed", we need to feed the results 
			// back to the LLM to generate a proper response
			return v, v.generateFollowUpResponse(msg.Results, msg.RequestID)
		}
		return v, nil
	
	case MCPToolExecutingMsg:
		// Add a message indicating tool execution has started
		executingMsg := ChatMessage{
			Role:      "tool",
			Content:   fmt.Sprintf("Executing tool: %s...", msg.ToolName),
			Timestamp: time.Now().Format("15:04:05"),
		}
		v.AddMessage(executingMsg)
		return v, nil
	
	case MCPToolExecutedMsg:
		// Handle tool execution completion
		if msg.Error != nil {
			// Go error occurred during execution
			errorMsg := ChatMessage{
				Role:      "tool",
				Content:   fmt.Sprintf("Tool execution failed: %s", msg.Error.Error()),
				Timestamp: time.Now().Format("15:04:05"),
				Error:     msg.Error.Error(),
			}
			v.AddMessage(errorMsg)
		} else if msg.Result != nil && msg.Result.Result != nil && msg.Result.Result.IsError {
			// MCP-level error
			errorText := "Unknown MCP error"
			if len(msg.Result.Result.Content) > 0 {
				errorText = msg.Result.Result.Content[0].Text
			}
			errorMsg := ChatMessage{
				Role:      "tool",
				Content:   fmt.Sprintf("Tool error: %s", errorText),
				Timestamp: time.Now().Format("15:04:05"),
				Error:     errorText,
			}
			v.AddMessage(errorMsg)
		} else if msg.Result != nil && msg.Result.Result != nil {
			// Success - extract text from result content
			var resultText string
			if len(msg.Result.Result.Content) > 0 {
				resultText = msg.Result.Result.Content[0].Text
			} else {
				resultText = "Tool completed successfully"
			}
			
			successMsg := ChatMessage{
				Role:      "tool",
				Content:   fmt.Sprintf("Tool result from %s:\n%s", msg.ToolName, resultText),
				Timestamp: time.Now().Format("15:04:05"),
			}
			v.AddMessage(successMsg)
		}
		return v, nil
		
	case tea.KeyMsg:
		// Don't accept input if waiting for response
		if v.waitingForResponse && msg.String() == "enter" {
			return v, nil
		}
		
		switch msg.String() {
		case "enter":
			if v.focused {
				userInput := strings.TrimSpace(v.input.Value())
				if userInput == "" {
					return v, nil
				}

				// Check if it's a command (starts with /)
				if strings.HasPrefix(userInput, "/") {
					return v, v.handleCommand(userInput)
				}

				// Regular chat message
				userMsg := ChatMessage{
					Role:      "user",
					Content:   userInput,
					Timestamp: time.Now().Format("15:04:05"),
				}
				v.AddMessage(userMsg)
				
				// Clear input
				v.input.SetValue("")
				
				// Generate ID for this request
				v.requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
				v.waitingForResponse = true
				
				// Send to model
				if v.agent != nil {
					// Use tool-aware response generation
					return v, v.generateResponseWithTools(userInput, v.requestID)
				} else {
					// Fallback to regular model response
					return v, GenerateResponse(v.model, userInput, v.requestID)
				}
			}
		case "ctrl+l":
			v.input.SetValue("")
			return v, nil
		}
	}

	// Update input
	v.input, cmd = v.input.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport
	v.viewport, cmd = v.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return v, tea.Batch(cmds...)
}

// View renders the chat view
func (v *ChatView) View() string {
	if v.width == 0 {
		return "Loading chat..."
	}

	// Header
	header := v.styles.ViewHeader.
		Width(v.width).
		Render("💬 Chat")

	// Messages content
	v.viewport.SetContent(v.renderMessages())

	// Input section
	inputSection := v.renderInput()

	// Calculate heights
	headerHeight := lipgloss.Height(header)
	inputHeight := lipgloss.Height(inputSection)
	viewportHeight := v.height - headerHeight - inputHeight - 2 // padding

	if viewportHeight < 1 {
		viewportHeight = 1
	}

	v.viewport.Height = viewportHeight

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		v.viewport.View(),
		inputSection,
	)
}

// SetSize sets the size of the chat view
func (v *ChatView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.input.Width = width - 4 // Account for borders and padding
}

// AddMessage adds a message to the chat
func (v *ChatView) AddMessage(msg ChatMessage) {
	v.messages = append(v.messages, msg)
	v.viewport.SetContent(v.renderMessages())
	v.viewport.GotoBottom()
}

// ClearMessages clears all messages
func (v *ChatView) ClearMessages() {
	v.messages = []ChatMessage{}
	v.viewport.SetContent("")
}

// GetInput returns the current input value
func (v *ChatView) GetInput() string {
	return v.input.Value()
}

// handleCommand processes chat commands that start with /
func (v *ChatView) handleCommand(input string) tea.Cmd {
	// Clear input immediately
	v.input.SetValue("")
	
	// Parse command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}
	
	command := strings.ToLower(parts[0])
	// args := parts[1:] // Reserved for future use with command arguments
	
	// Add command to chat history
	commandMsg := ChatMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now().Format("15:04:05"),
	}
	v.AddMessage(commandMsg)
	
	// Process different commands
	switch command {
	case "/mcp", "/servers":
		// Show MCP servers
		return func() tea.Msg {
			return ViewSwitchMsg{ViewType: ServerViewType}
		}
	case "/tools":
		// Show tools
		return func() tea.Msg {
			return ViewSwitchMsg{ViewType: ToolViewType}
		}
	case "/help":
		// Show help
		return func() tea.Msg {
			return ViewSwitchMsg{ViewType: HelpViewType}
		}
	case "/history":
		// Show history
		return func() tea.Msg {
			return ViewSwitchMsg{ViewType: HistoryViewType}
		}
	case "/exit", "/quit":
		// Exit the application
		return tea.Quit
	case "/chat":
		// Stay in chat (no-op but show confirmation)
		responseMsg := ChatMessage{
			Role:      "assistant",
			Content:   "Already in chat view. Available commands:\n• /mcp or /servers - MCP servers\n• /tools - Available tools\n• /help - Detailed help\n• /history - Conversation history\n• /exit or /quit - Exit application",
			Timestamp: time.Now().Format("15:04:05"),
		}
		v.AddMessage(responseMsg)
		return nil
	case "/commands":
		// List all commands
		responseMsg := ChatMessage{
			Role:      "assistant",
			Content:   "Available commands:\n• /mcp, /servers - Switch to MCP servers view\n• /tools - Switch to tools view\n• /help - Switch to help view\n• /history - Switch to history view\n• /chat - Stay in chat view\n• /commands - Show this list\n\nTip: You can also use number keys 1-5 to switch views!",
			Timestamp: time.Now().Format("15:04:05"),
		}
		v.AddMessage(responseMsg)
		return nil
	default:
		// Unknown command
		responseMsg := ChatMessage{
			Role:      "assistant",
			Content:   fmt.Sprintf("Unknown command: %s\nType /commands to see all available commands.", command),
			Timestamp: time.Now().Format("15:04:05"),
		}
		v.AddMessage(responseMsg)
		return nil
	}
}

// SetInput sets the input value
func (v *ChatView) SetInput(value string) {
	v.input.SetValue(value)
}

// renderMessages renders all chat messages
func (v *ChatView) renderMessages() string {
	if len(v.messages) == 0 {
		return v.styles.DimmedStyle.Render("No messages yet. Start a conversation!")
	}

	var lines []string
	for _, msg := range v.messages {
		lines = append(lines, v.renderMessage(msg))
		lines = append(lines, "") // Add spacing between messages
	}

	return strings.Join(lines, "\n")
}

// renderMessage renders a single message
func (v *ChatView) renderMessage(msg ChatMessage) string {
	var style lipgloss.Style
	var prefix string

	switch msg.Role {
	case "user":
		style = v.styles.MessageUser
		prefix = "You"
	case "assistant":
		style = v.styles.MessageBot
		prefix = "Assistant"
	case "tool":
		style = v.styles.MessageTool
		prefix = "Tool"
	default:
		style = v.styles.Base
		prefix = "System"
	}

	// Format timestamp (simplified for now)
	timeStr := v.styles.DimmedStyle.Render(fmt.Sprintf("[%s]", msg.Timestamp))

	// Header line
	header := fmt.Sprintf("%s %s:",
		timeStr,
		style.Render(prefix),
	)

	// Content - wrap long lines
	content := v.wrapText(msg.Content, v.width-4)
	
	// Add error if present
	if msg.Error != "" {
		content += "\n" + v.styles.ErrorStyle.Render("Error: "+msg.Error)
	}

	// Add tool call info if present
	if msg.ToolCall != nil {
		toolInfo := fmt.Sprintf("\n%s Called tool: %s",
			v.styles.DimmedStyle.Render("🔧"),
			v.styles.HighlightStyle.Render(msg.ToolCall.Name),
		)
		if msg.ToolCall.Result != "" {
			toolInfo += "\n" + v.styles.DimmedStyle.Render("Result: ") + msg.ToolCall.Result
		}
		content += toolInfo
	}

	return header + "\n" + content
}

// renderInput renders the input section
func (v *ChatView) renderInput() string {
	prompt := v.styles.InputPrompt.Render("❯ ")
	
	// Show different prompt when waiting for response
	if v.waitingForResponse {
		prompt = v.styles.DimmedStyle.Render("⏳ ")
	}
	
	input := v.styles.InputBox.
		Width(v.width-lipgloss.Width(prompt)-2).
		Render(v.input.View())

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		prompt,
		input,
	)
}

// wrapText wraps text to fit within the specified width
func (v *ChatView) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) <= width {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// generateResponseWithTools generates a response using available tools
func (v *ChatView) generateResponseWithTools(message, id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Get available tools from agent
		tools, err := v.agent.GetMCPToolsAsDefinitions(ctx)
		if err != nil {
			// Fallback to regular generation if tool fetch fails
			response, err := v.model.Generate(ctx, message, model.GenerateOptions{
				Temperature: 0.7,
				MaxTokens:   2048,
			})
			return ModelResponseMsg{
				Response: response,
				Error:    err,
				ID:       id,
			}
		}
		
		// Use tool-aware generation
		messages := []model.Message{
			{Role: "user", Content: message},
		}
		
		response, err := v.model.ChatWithTools(ctx, messages, tools, model.GenerateOptions{
			Temperature: 0.7,
			MaxTokens:   2048,
		})
		
		// If tools were called, execute them
		if response != nil && len(response.ToolCalls) > 0 {
			return ToolCallDetectedMsg{
				ToolCalls:           response.ToolCalls,
				RequestID:           id,
				Response:            response,
				UserMessage:         message,
				ConversationHistory: messages,
				Tools:               tools,
			}
		}
		
		return ModelResponseMsg{
			Response: response,
			Error:    err,
			ID:       id,
		}
	}
}

// executeToolCalls executes the detected tool calls
func (v *ChatView) executeToolCalls(toolCalls []model.ToolCall, requestID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		var results []string
		for _, toolCall := range toolCalls {
			if v.agent != nil {
				result, err := v.agent.ExecuteTool(ctx, toolCall.Name, toolCall.Arguments)
				if err != nil {
					results = append(results, fmt.Sprintf("❌ **%s** failed: %v", toolCall.Name, err))
				} else if result.Success {
					// Capture the ACTUAL result instead of just "success"
					resultText := fmt.Sprintf("✅ **%s**: %v", toolCall.Name, result.Result)
					results = append(results, resultText)
				} else {
					results = append(results, fmt.Sprintf("❌ **%s**: %s", toolCall.Name, result.Error))
				}
			} else {
				results = append(results, fmt.Sprintf("❌ **%s**: no agent available", toolCall.Name))
			}
		}
		
		return ToolExecutionResultMsg{
			RequestID: requestID,
			Results:   results,
		}
	}
}

// formatToolResult formats tool results in a user-friendly way
func (v *ChatView) formatToolResult(toolName string, result interface{}) string {
	switch toolName {
	case "store_memory":
		// For memory storage, just confirm success
		return "Memory stored successfully"
		
	case "search":
		// For search results, format nicely
		return v.formatSearchResult(result)
		
	case "get_memory_by_id":
		// For memory retrieval, show the content
		return v.formatMemoryResult(result)
		
	case "analysis", "relationships", "stats", "sessions":
		// For analytical tools, provide a summary
		return v.formatAnalysisResult(result)
		
	default:
		// For unknown tools, provide a clean fallback
		return v.formatGenericResult(result)
	}
}

// formatSearchResult formats search results nicely
func (v *ChatView) formatSearchResult(result interface{}) string {
	// Extract meaningful information from search results
	if resultStr, ok := result.(string); ok {
		// Try to parse if it's JSON-like
		if strings.Contains(resultStr, "memories") && strings.Contains(resultStr, "total") {
			// This looks like a search result summary
			lines := strings.Split(resultStr, "\n")
			var summary []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "total") || strings.Contains(line, "found") || strings.Contains(line, "results") {
					summary = append(summary, line)
					if len(summary) >= 3 { // Limit to first few lines
						break
					}
				}
			}
			if len(summary) > 0 {
				return strings.Join(summary, " • ")
			}
		}
	}
	return "Search completed successfully"
}

// formatMemoryResult formats memory retrieval results
func (v *ChatView) formatMemoryResult(result interface{}) string {
	if resultStr, ok := result.(string); ok {
		// Extract content from memory result
		if strings.Contains(resultStr, "content") {
			return "Memory retrieved successfully"
		}
	}
	return "Memory operation completed"
}

// formatAnalysisResult formats analysis tool results
func (v *ChatView) formatAnalysisResult(result interface{}) string {
	return "Analysis completed successfully"
}

// formatGenericResult provides a fallback for unknown tools
func (v *ChatView) formatGenericResult(result interface{}) string {
	if resultStr, ok := result.(string); ok {
		// If it's a short string, show it
		if len(resultStr) < 100 {
			return resultStr
		}
		// If it's long, show a summary
		return "Operation completed successfully"
	}
	return "Tool executed successfully"
}

// generateFollowUpResponse generates an LLM response based on tool results
// This continues the SAME conversation by adding tool results to the history
func (v *ChatView) generateFollowUpResponse(toolResults []string, requestID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Build the conversation history with tool results
		// Start with the original user message (already in conversationHistory)
		messages := make([]model.Message, len(v.conversationHistory))
		copy(messages, v.conversationHistory)
		
		// Format tool results cleanly - strip the checkmarks and formatting
		// to make it easier for the LLM to parse
		var cleanResults []string
		for _, result := range toolResults {
			// Remove markdown formatting and emoji
			cleaned := strings.TrimPrefix(result, "✅ **")
			cleaned = strings.TrimPrefix(cleaned, "❌ **")
			// Remove the tool name prefix (e.g., "search**: ")
			if idx := strings.Index(cleaned, "**: "); idx != -1 {
				cleaned = cleaned[idx+4:]
			} else if idx := strings.Index(cleaned, "**: "); idx != -1 {
				cleaned = cleaned[idx+3:]
			}
			cleanResults = append(cleanResults, cleaned)
		}
		resultsText := strings.Join(cleanResults, "\n\n")
		
		// Add an assistant message saying it used tools
		// This mimics the flow the LLM expects
		messages = append(messages, model.Message{
			Role:    "assistant",
			Content: "Let me use the available tools to help answer your question.",
		})
		
		// Add a user message with the tool results
		// Frame it as the user providing feedback on what the tools returned
		userFollowUp := fmt.Sprintf("The tools returned the following information:\n\n%s\n\nPlease use this information to answer my original question.", resultsText)
		
		messages = append(messages, model.Message{
			Role:    "user",
			Content: userFollowUp,
		})
		
		// Now continue the conversation - use regular Chat (not ChatWithTools)
		// since we already executed the tools
		response, err := v.model.Chat(ctx, messages, model.GenerateOptions{
			Temperature: 0.7,
			MaxTokens:   1024,
		})
		
		return ModelResponseMsg{
			Response: response,
			Error:    err,
			ID:       requestID,
		}
	}
}

// Focus sets focus to the input
func (v *ChatView) Focus() {
	v.focused = true
	v.input.Focus()
}

// Blur removes focus from the input
func (v *ChatView) Blur() {
	v.focused = false
	v.input.Blur()
}