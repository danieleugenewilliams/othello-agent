package tui

import (
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
	waitingForResponse bool
	requestID string
}

// NewChatView creates a new chat view
func NewChatView(styles Styles, keymap KeyMap, m model.Model) *ChatView {
	input := textinput.New()
	input.Placeholder = "Type a message..."
	input.Focus()
	input.CharLimit = 1000
	input.Width = 50

	vp := viewport.New(0, 0)
	vp.SetContent("")

	return &ChatView{
		styles:   styles,
		keymap:   keymap,
		viewport: vp,
		input:    input,
		messages: []ChatMessage{},
		focused:  true,
		model:    m,
		waitingForResponse: false,
	}
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
		
	case tea.KeyMsg:
		// Don't accept input if waiting for response
		if v.waitingForResponse && msg.String() == "enter" {
			return v, nil
		}
		
		switch msg.String() {
		case "enter":
			if v.input.Value() != "" {
				// Add user message
				userInput := v.input.Value()
				userMsg := ChatMessage{
					Role:      "user",
					Content:   userInput,
					Timestamp: time.Now().Format("15:04"),
				}
				v.AddMessage(userMsg)
				
				// Clear input
				v.input.SetValue("")
				
				// Generate ID for this request
				v.requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
				v.waitingForResponse = true
				
				// Send to model
				return v, GenerateResponse(v.model, userInput, v.requestID)
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