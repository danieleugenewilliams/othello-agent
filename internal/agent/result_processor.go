package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
)

// ToolResultProcessor processes raw tool results into user-friendly summaries
type ToolResultProcessor struct {
	// Can add configuration here later (e.g., verbosity level)
	Logger *log.Logger
}


// keys returns the keys of a map for logging purposes
func keys(m map[string]interface{}) []string {
	var k []string
	for key := range m {
		k = append(k, key)
	}
	return k
}

// logf logs with the configured logger or falls back to standard log
func (p *ToolResultProcessor) logf(format string, args ...interface{}) {
	if p.Logger != nil {
		p.Logger.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

// ProcessToolResult takes a raw MCP tool result and presents it according to the MCP specification
// This is completely universal and works with any MCP server by respecting the Content type system
// Backward compatible version without conversation context
func (p *ToolResultProcessor) ProcessToolResult(ctx context.Context, toolName string, rawResult interface{}, userQuery string) (string, error) {
	// Use the enhanced version with empty context for backward compatibility
	context := &model.ConversationContext{
		UserQuery:   userQuery,
		SessionType: "chat",
	}
	return p.ProcessToolResultWithContext(ctx, toolName, rawResult, context)
}

// ProcessToolResultWithContext processes tool results with conversation context for intelligent responses
func (p *ToolResultProcessor) ProcessToolResultWithContext(ctx context.Context, toolName string, rawResult interface{}, convContext *model.ConversationContext) (string, error) {
	p.logf("[PROCESSOR] Processing MCP tool result for: '%s' with conversation context", toolName)
	p.logf("[PROCESSOR] Raw result type: %T", rawResult)
	p.logf("[PROCESSOR] Conversation history length: %d", len(convContext.History))

	// Handle nil result
	if rawResult == nil {
		p.logf("[PROCESSOR] Raw result is nil")
		return p.generateContextualResponse("The tool returned no results.", convContext), nil
	}

	// Extract metadata from the tool result before formatting
	p.extractAndStoreMetadata(rawResult, convContext)

	// The rawResult should be a ToolResult from the MCP server
	// Try to extract it as a ToolResult struct or map representation
	if toolResult := p.extractMCPToolResult(rawResult); toolResult != nil {
		p.logf("[PROCESSOR] Successfully extracted MCP ToolResult with %d content items", 0)
		baseResult := p.formatMCPContent(toolResult)
		return p.generateContextualResponse(baseResult, convContext), nil
	}

	// Fallback: treat as raw content if not in MCP ToolResult format
	p.logf("[PROCESSOR] Not an MCP ToolResult format, using fallback presentation")
	baseResult := p.formatFallbackContent(rawResult)
	return p.generateContextualResponse(baseResult, convContext), nil
}

// checkForError checks if result contains an error
func (p *ToolResultProcessor) checkForError(result map[string]interface{}) (string, bool) {
	if isError, ok := result["error"].(bool); ok && isError {
		return "I was unable to complete that action. Please try again.", true
	}
	
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return "I encountered an issue while processing that request.", true
	}
	
	return "", false
}

// processSearchResults formats search results concisely
func (p *ToolResultProcessor) processSearchResults(result map[string]interface{}, query string) string {
	p.logf("[PROCESSOR] Processing search results, map keys: %v", keys(result))

	results, ok := result["results"].([]interface{})
	if !ok {
		p.logf("[PROCESSOR] No 'results' field found or not an array")
		return "I didn't find any memories matching your search."
	}

	if len(results) == 0 {
		p.logf("[PROCESSOR] Results array is empty")
		return "I didn't find any memories matching your search."
	}

	p.logf("[PROCESSOR] Found %d search results", len(results))

	var summaries []string
	for i, r := range results {
		if i >= 5 { // Limit to 5 results for conciseness
			summaries = append(summaries, fmt.Sprintf("...and %d more results", len(results)-i))
			break
		}

		resultMap, ok := r.(map[string]interface{})
		if !ok {
			p.logf("[PROCESSOR] Result %d is not a map, skipping", i)
			continue
		}

		p.logf("[PROCESSOR] Result %d keys: %v", i, keys(resultMap))

		// Extract content - handle both 'content' and 'summary' fields (MCP compatibility)
		var content string
		if summary, ok := resultMap["summary"].(string); ok {
			content = summary
			p.logf("[PROCESSOR] Result %d: extracted summary field", i)
		} else if contentField, ok := resultMap["content"].(string); ok {
			content = contentField
			p.logf("[PROCESSOR] Result %d: extracted content field", i)
		} else {
			p.logf("[PROCESSOR] Result %d: no summary or content field found, skipping", i)
			continue // Skip if neither field is found
		}

		// Build rich formatted result with MCP-specific fields
		var resultText strings.Builder

		// Extract importance for priority indication
		importance, _ := resultMap["importance"].(float64)
		if importance > 7 {
			resultText.WriteString("🔥 **")
		} else if importance > 5 {
			resultText.WriteString("⭐ **")
		} else {
			resultText.WriteString("• **")
		}

		// Try to extract a title from the summary (first sentence or line)
		title := content
		if idx := strings.Index(content, ":"); idx > 0 && idx < 80 {
			title = content[:idx]
			content = strings.TrimSpace(content[idx+1:])
		} else if idx := strings.Index(content, "."); idx > 0 && idx < 80 {
			title = content[:idx]
			content = strings.TrimSpace(content[idx+1:])
		}

		resultText.WriteString(title)
		resultText.WriteString("**")

		// Add importance indicator
		if importance > 0 {
			resultText.WriteString(fmt.Sprintf(" (Importance: %.0f/10)", importance))
		}
		resultText.WriteString("\n  ")

		// Truncate long content but be more generous for rich results
		if len(content) > 200 {
			content = content[:197] + "..."
		}
		resultText.WriteString(content)

		// Add tags if available
		if tagsInterface, ok := resultMap["tags"].([]interface{}); ok && len(tagsInterface) > 0 {
			var tags []string
			maxTags := 3
			if len(tagsInterface) < maxTags {
				maxTags = len(tagsInterface)
			}
			for _, tag := range tagsInterface[:maxTags] { // Limit to 3 tags
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
			if len(tags) > 0 {
				resultText.WriteString(fmt.Sprintf("\n  🏷️ %s", strings.Join(tags, ", ")))
			}
		}

		summaries = append(summaries, resultText.String())
	}
	
	if len(summaries) == 0 {
		p.logf("[PROCESSOR] No summaries extracted from %d results", len(results))
		return "I found some results but couldn't extract the content."
	}

	count := len(results)
	header := fmt.Sprintf("I found %d relevant memor", count)
	if count == 1 {
		header += "y:\n\n"
	} else {
		header += "ies:\n\n"
	}

	finalResult := header + strings.Join(summaries, "\n")
	p.logf("[PROCESSOR] Search processing complete, returning %d characters", len(finalResult))
	return finalResult
}

// processStoreMemoryResult formats memory storage confirmation
func (p *ToolResultProcessor) processStoreMemoryResult(result map[string]interface{}) string {
	if success, ok := result["success"].(bool); ok && success {
		return "I've successfully stored that memory."
	}
	
	return "Memory has been stored."
}

// processAnalysisResult formats analysis results
func (p *ToolResultProcessor) processAnalysisResult(result map[string]interface{}) string {
	// Extract answer for question-based analysis
	if answer, ok := result["answer"].(string); ok && answer != "" {
		return answer
	}
	
	// Extract summary for summarization
	if summary, ok := result["summary"].(string); ok && summary != "" {
		return summary
	}
	
	// Extract patterns for pattern analysis
	if patterns, ok := result["patterns"].([]interface{}); ok && len(patterns) > 0 {
		return p.formatPatterns(patterns)
	}
	
	return "Analysis complete. The results are available."
}

// processStatsResult formats statistics concisely
func (p *ToolResultProcessor) processStatsResult(result map[string]interface{}) string {
	var parts []string
	
	// Handle both int and float64 types
	if memCount := p.getNumericValue(result, "memory_count"); memCount > 0 {
		parts = append(parts, fmt.Sprintf("%.0f memories", memCount))
	}
	
	if domainCount := p.getNumericValue(result, "domain_count"); domainCount > 0 {
		parts = append(parts, fmt.Sprintf("%.0f domains", domainCount))
	}
	
	if catCount := p.getNumericValue(result, "category_count"); catCount > 0 {
		parts = append(parts, fmt.Sprintf("%.0f categories", catCount))
	}
	
	if len(parts) == 0 {
		return "Statistics retrieved successfully."
	}
	
	return "You have " + strings.Join(parts, ", ") + "."
}

// getNumericValue extracts a numeric value from result, handling both int and float64
func (p *ToolResultProcessor) getNumericValue(result map[string]interface{}, key string) float64 {
	val, ok := result[key]
	if !ok {
		return 0
	}
	
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// processRelationshipsResult formats relationship results
func (p *ToolResultProcessor) processRelationshipsResult(result map[string]interface{}) string {
	if related, ok := result["related_memories"].([]interface{}); ok {
		count := len(related)
		if count == 0 {
			return "I didn't find any related memories."
		}
		return fmt.Sprintf("I found %d related memories.", count)
	}
	
	if connections, ok := result["connections"].([]interface{}); ok {
		count := len(connections)
		if count == 0 {
			return "I didn't find any connections."
		}
		return fmt.Sprintf("I found %d connections between memories.", count)
	}
	
	return "Relationship analysis complete."
}

// processListResult formats list-type results (domains, categories, sessions)
func (p *ToolResultProcessor) processListResult(result map[string]interface{}, toolName string) string {
	// Try to find the list in result
	var items []interface{}
	
	// Check common list field names
	for _, key := range []string{toolName, "results", "items", "list"} {
		if list, ok := result[key].([]interface{}); ok {
			items = list
			break
		}
	}
	
	if len(items) == 0 {
		return fmt.Sprintf("No %s found.", toolName)
	}
	
	singular := strings.TrimSuffix(toolName, "s")
	return fmt.Sprintf("Found %d %s.", len(items), singular)
}

// formatPatterns formats pattern analysis results
func (p *ToolResultProcessor) formatPatterns(patterns []interface{}) string {
	if len(patterns) == 0 {
		return "No patterns found."
	}
	
	var formatted []string
	for i, pattern := range patterns {
		if i >= 3 { // Limit to 3 patterns
			formatted = append(formatted, fmt.Sprintf("...and %d more patterns", len(patterns)-i))
			break
		}
		
		if patternStr, ok := pattern.(string); ok {
			formatted = append(formatted, fmt.Sprintf("• %s", patternStr))
		} else if patternMap, ok := pattern.(map[string]interface{}); ok {
			if name, ok := patternMap["pattern"].(string); ok {
				formatted = append(formatted, fmt.Sprintf("• %s", name))
			}
		}
	}
	
	return "I found these patterns:\n\n" + strings.Join(formatted, "\n")
}

// normalizeMCPToolName extracts the base tool name from MCP prefixed tools
// Examples: "mcp__local-memory__search" -> "search", "mcp__filesystem__read" -> "read"
func (p *ToolResultProcessor) normalizeMCPToolName(toolName string) string {
	// Check if it's an MCP tool (format: mcp__server__tool)
	if strings.HasPrefix(toolName, "mcp__") {
		parts := strings.Split(toolName, "__")
		if len(parts) >= 3 {
			// Return the tool name part (everything after the second __)
			return strings.Join(parts[2:], "__")
		}
	}
	// Return original name if not MCP format
	return toolName
}

// detectContentType analyzes the result structure to determine the best processing approach
// This allows any MCP server to work regardless of tool naming
func (p *ToolResultProcessor) detectContentType(result map[string]interface{}) string {
	p.logf("[PROCESSOR] Detecting content type from keys: %v", keys(result))

	// Search-type results (lists of items with content/memories)
	if results, hasResults := result["results"].([]interface{}); hasResults {
		if len(results) > 0 {
			// Check if first result looks like a memory/search result
			if firstResult, ok := results[0].(map[string]interface{}); ok {
				if _, hasContent := firstResult["content"]; hasContent {
					p.logf("[PROCESSOR] Detected search-type result (results array with content)")
					return "search"
				}
				if _, hasSummary := firstResult["summary"]; hasSummary {
					p.logf("[PROCESSOR] Detected search-type result (results array with summary)")
					return "search"
				}
			}
		} else {
			// Empty results array still counts as a search result
			p.logf("[PROCESSOR] Detected search-type result (empty results array)")
			return "search"
		}
	}

	// Memory storage results
	if _, hasSuccess := result["success"].(bool); hasSuccess {
		if _, hasMemoryId := result["memory_id"]; hasMemoryId {
			p.logf("[PROCESSOR] Detected store_memory result (success + memory_id)")
			return "store_memory"
		}
	}

	// Analysis results
	if answer, hasAnswer := result["answer"].(string); hasAnswer && answer != "" {
		p.logf("[PROCESSOR] Detected analysis result (answer field)")
		return "analysis"
	}

	// Statistics results
	if _, hasMemoryCount := result["memory_count"]; hasMemoryCount {
		p.logf("[PROCESSOR] Detected stats result (memory_count field)")
		return "stats"
	}
	if _, hasTotalResults := result["total_results"]; hasTotalResults {
		p.logf("[PROCESSOR] Detected stats result (total_results field)")
		return "stats"
	}

	// Relationship results
	if _, hasRelated := result["related_memories"]; hasRelated {
		p.logf("[PROCESSOR] Detected relationships result (related_memories field)")
		return "relationships"
	}
	if _, hasConnections := result["connections"]; hasConnections {
		p.logf("[PROCESSOR] Detected relationships result (connections field)")
		return "relationships"
	}

	// List-type results (domains, categories, sessions, etc.)
	for _, listKey := range []string{"domains", "categories", "sessions", "servers", "tools"} {
		if list, ok := result[listKey].([]interface{}); ok && len(list) > 0 {
			p.logf("[PROCESSOR] Detected list result type: %s", listKey)
			return listKey
		}
	}

	p.logf("[PROCESSOR] No specific content type detected")
	return ""
}

// formatSmartGenericResult provides enhanced fallback formatting with better structure detection
func (p *ToolResultProcessor) formatSmartGenericResult(result map[string]interface{}) string {
	p.logf("[PROCESSOR] formatSmartGenericResult called with keys: %v", keys(result))

	// Try to find the most important content to display
	var content strings.Builder

	// Look for common success indicators
	if success, ok := result["success"].(bool); ok {
		if success {
			content.WriteString("✅ Operation completed successfully")
		} else {
			content.WriteString("❌ Operation failed")
		}

		// Add any message if available
		if msg, hasMsg := result["message"].(string); hasMsg && msg != "" {
			content.WriteString(": " + msg)
		}
		content.WriteString("\n")
	}

	// Look for result counts or summaries
	if count := p.getNumericValue(result, "count"); count > 0 {
		content.WriteString(fmt.Sprintf("Processed %.0f items\n", count))
	}
	if total := p.getNumericValue(result, "total"); total > 0 {
		content.WriteString(fmt.Sprintf("Total: %.0f\n", total))
	}

	// Look for any descriptive fields
	for _, field := range []string{"description", "summary", "message", "result", "output"} {
		if value, ok := result[field].(string); ok && value != "" {
			content.WriteString(value)
			content.WriteString("\n")
			break // Use first found description field
		}
	}

	// If we found meaningful content, return it
	if content.Len() > 0 {
		resultText := strings.TrimSpace(content.String())
		p.logf("[PROCESSOR] Returning smart formatted result: %d chars", len(resultText))
		return resultText
	}

	// Fall back to the original generic formatter
	return p.formatGenericResult(result)
}

// formatGenericResult provides a fallback for unknown result types
func (p *ToolResultProcessor) formatGenericResult(result interface{}) string {
	p.logf("[PROCESSOR] formatGenericResult called with type: %T", result)

	// Try to marshal to JSON and extract key information
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		p.logf("[PROCESSOR] Failed to marshal result to JSON: %v", err)
		return "The tool completed successfully."
	}

	jsonStr := string(jsonBytes)
	p.logf("[PROCESSOR] JSON result length: %d", len(jsonStr))
	if len(jsonStr) < 500 { // Show more in logs for debugging
		p.logf("[PROCESSOR] JSON content: %s", jsonStr)
	}

	// If the JSON is small enough, show a cleaned version
	if len(jsonStr) < 200 {
		// Remove technical fields
		jsonStr = strings.ReplaceAll(jsonStr, `"id":`, ``)
		jsonStr = strings.ReplaceAll(jsonStr, `"timestamp":`, ``)
		p.logf("[PROCESSOR] Returning cleaned JSON result")
		return "Result: " + jsonStr
	}

	p.logf("[PROCESSOR] Returning generic fallback message")
	return "The tool completed successfully. Results are available."
}

// extractMCPToolResult attempts to extract an MCP ToolResult from the raw result
func (p *ToolResultProcessor) extractMCPToolResult(rawResult interface{}) interface{} {
	// Check if it's already a proper MCP ToolResult
	if toolResult, ok := rawResult.(*mcp.ToolResult); ok {
		p.logf("[EXTRACT] Found native MCP ToolResult with %d content items", len(toolResult.Content))
		return toolResult
	}

	// Check if it has the MCP ToolResult structure as a map
	if resultMap, ok := rawResult.(map[string]interface{}); ok {
		if contentField, hasContent := resultMap["content"]; hasContent {
			if contentArray, ok := contentField.([]interface{}); ok {
				p.logf("[EXTRACT] Found MCP-style content array with %d items", len(contentArray))
				return resultMap
			}
		}
	}
	p.logf("[EXTRACT] Could not extract MCP ToolResult structure")
	return nil
}

// formatMCPContent formats MCP Content array according to the MCP specification
func (p *ToolResultProcessor) formatMCPContent(contents interface{}) string {
	var contentArray []interface{}

	// Handle native MCP ToolResult
	if toolResult, ok := contents.(*mcp.ToolResult); ok {
		p.logf("[FORMAT] Processing native MCP ToolResult with %d content items", len(toolResult.Content))

		// Convert mcp.Content to []interface{} for uniform processing
		contentArray = make([]interface{}, len(toolResult.Content))
		for i, content := range toolResult.Content {
			contentArray[i] = map[string]interface{}{
				"type": content.Type,
				"text": content.Text,
				"data": content.Data,
			}
		}
	} else {
		// Handle map format
		contentMap, ok := contents.(map[string]interface{})
		if !ok {
			p.logf("[FORMAT] Invalid content format")
			return "Invalid tool result format"
		}

		contentField, hasContent := contentMap["content"]
		if !hasContent {
			p.logf("[FORMAT] No content field found")
			return "No content in tool result"
		}

		var mapOk bool
		contentArray, mapOk = contentField.([]interface{})
		if !mapOk {
			p.logf("[FORMAT] Content is not an array")
			return "Invalid content format"
		}
	}

	if len(contentArray) == 0 {
		p.logf("[FORMAT] Empty content array")
		return "Tool completed successfully (no content returned)."
	}

	p.logf("[FORMAT] Formatting %d MCP content items", len(contentArray))

	var output strings.Builder

	for i, item := range contentArray {
		contentItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Add separator if multiple content items
		if i > 0 {
			output.WriteString("\n---\n\n")
		}

		contentType, _ := contentItem["type"].(string)
		contentText, _ := contentItem["text"].(string)
		contentData, _ := contentItem["data"].(string)

		p.logf("[FORMAT] Content %d: type='%s', text_len=%d, data_len=%d", 
			i, contentType, len(contentText), len(contentData))

		switch contentType {
		case "text":
			// Text content: check if it's actually JSON that should be parsed
			if contentText != "" {
				// Try to detect if this is JSON masquerading as text
				trimmed := strings.TrimSpace(contentText)
				if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
				   (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
					// Looks like JSON, try to parse and format it intelligently
					if parsed := p.tryParseAndFormatJSON(contentText); parsed != "" {
						output.WriteString(parsed)
					} else {
						// Not valid JSON or failed to parse, display as-is
						output.WriteString(contentText)
					}
				} else {
					// Regular text, display as-is
					output.WriteString(contentText)
				}
			} else {
				output.WriteString("[Empty text content]")
			}

		case "json":
			// JSON content: pretty-print if possible
			var jsonContent string
			if contentData != "" {
				jsonContent = contentData
			} else {
				jsonContent = contentText
			}
			
			if jsonContent != "" {
				if prettyJSON := p.prettyPrintJSON(jsonContent); prettyJSON != "" {
					output.WriteString("```json\n")
					output.WriteString(prettyJSON)
					output.WriteString("\n```")
				} else {
					output.WriteString(jsonContent)
				}
			} else {
				output.WriteString("[Empty JSON content]")
			}

		case "html":
			// HTML content: display as text for now
			var htmlContent string
			if contentText != "" {
				htmlContent = contentText
			} else {
				htmlContent = contentData
			}
			if htmlContent != "" {
				output.WriteString(htmlContent)
			} else {
				output.WriteString("[Empty HTML content]")
			}

		case "image", "binary":
			// Binary content: show metadata
			output.WriteString(fmt.Sprintf("[%s content - %d bytes]", contentType, len(contentData)))

		default:
			// Unknown content type: display what we can
			if contentText != "" {
				output.WriteString(fmt.Sprintf("[%s] %s", contentType, contentText))
			} else if contentData != "" {
				output.WriteString(fmt.Sprintf("[%s] %s", contentType, contentData))
			} else {
				output.WriteString(fmt.Sprintf("[%s content - no data]", contentType))
			}
		}
	}

	result := output.String()
	p.logf("[FORMAT] Final formatted output length: %d", len(result))
	return result
}

// formatFallbackContent handles non-MCP format results
func (p *ToolResultProcessor) formatFallbackContent(rawResult interface{}) string {
	p.logf("[FALLBACK] Formatting non-MCP result of type %T", rawResult)

	// Try to present the content in a useful way
	switch result := rawResult.(type) {
	case string:
		// Plain string: return as-is
		return result

	case map[string]interface{}:
		// Map: try to find meaningful content
		return p.formatMapContent(result)

	case []interface{}:
		// Array: format as list
		return p.formatArrayContent(result)

	default:
		// Unknown type: JSON marshal as fallback
		if jsonBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("Tool result: %v", result)
	}
}

// formatMapContent formats a map in a user-friendly way
func (p *ToolResultProcessor) formatMapContent(result map[string]interface{}) string {
	// First, try to detect content type and use specialized formatters
	contentType := p.detectContentType(result)
	p.logf("[MAP-FORMAT] Detected content type: %s", contentType)
	
	switch contentType {
	case "search":
		return p.processSearchResults(result, "")
	case "store_memory":
		return p.processStoreMemoryResult(result)
	case "analysis":
		return p.processAnalysisResult(result)
	case "stats":
		return p.processStatsResult(result)
	case "relationships":
		return p.processRelationshipsResult(result)
	case "domains", "categories", "sessions":
		return p.processListResult(result, contentType)
	}
	
	// If no specialized formatter, use generic formatting
	
	// Check for errors first
	if errMsg, _ := p.checkForError(result); errMsg != "" {
		return errMsg
	}
	
	// Look for common fields that indicate success/failure
	if success, ok := result["success"].(bool); ok {
		if success {
			if msg, hasMsg := result["message"].(string); hasMsg {
				return fmt.Sprintf("✅ %s", msg)
			}
			return "✅ Operation completed successfully"
		} else {
			if msg, hasMsg := result["message"].(string); hasMsg {
				return fmt.Sprintf("❌ %s", msg)
			}
			return "❌ Operation failed"
		}
	}

	// Look for error indicators
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return fmt.Sprintf("❌ Error: %s", errMsg)
	}

	// Look for descriptive content
	for _, field := range []string{"message", "description", "summary", "result"} {
		if value, ok := result[field].(string); ok && value != "" {
			return value
		}
	}

	// Fallback to JSON
	if jsonBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
		return string(jsonBytes)
	}
	return "Tool completed successfully"
}

// formatArrayContent formats an array in a user-friendly way
func (p *ToolResultProcessor) formatArrayContent(result []interface{}) string {
	if len(result) == 0 {
		return "No items returned"
	}

	if len(result) == 1 {
		// Single item: format directly
		return p.formatFallbackContent(result[0])
	}

	// Multiple items: create a list
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d items:\n\n", len(result)))

	for i, item := range result {
		if i >= 10 { // Limit to 10 items
			output.WriteString(fmt.Sprintf("... and %d more items", len(result)-i))
			break
		}

		output.WriteString(fmt.Sprintf("%d. ", i+1))
		if itemStr := p.formatFallbackContent(item); itemStr != "" {
			output.WriteString(itemStr)
		} else {
			output.WriteString("[No content]")
		}
		output.WriteString("\n")
	}

	return output.String()
}

// tryParseAndFormatJSON attempts to parse JSON and format it intelligently for user display
// Returns formatted string if successful, empty string if not JSON or parsing fails
func (p *ToolResultProcessor) tryParseAndFormatJSON(jsonStr string) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		p.logf("[JSON-PARSE] Failed to parse as JSON: %v", err)
		return ""
	}
	
	p.logf("[JSON-PARSE] Successfully parsed JSON, type: %T", parsed)
	
	// If it's a map, try to format it intelligently
	if resultMap, ok := parsed.(map[string]interface{}); ok {
		return p.formatMapContent(resultMap)
	}
	
	// If it's an array, format as list
	if resultArray, ok := parsed.([]interface{}); ok {
		return p.formatArrayContent(resultArray)
	}
	
	// Fallback to pretty-printed JSON
	return p.prettyPrintJSON(jsonStr)
}

// prettyPrintJSON attempts to pretty-print JSON, returns empty string if invalid
func (p *ToolResultProcessor) prettyPrintJSON(jsonStr string) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return "" // Not valid JSON
	}

	if prettyBytes, err := json.MarshalIndent(parsed, "", "  "); err == nil {
		return string(prettyBytes)
	}
	return ""
}

// generateContextualResponse enhances the base result with conversation context and follow-up suggestions
func (p *ToolResultProcessor) generateContextualResponse(baseResult string, convContext *model.ConversationContext) string {
	if convContext == nil {
		return baseResult
	}

	p.logf("[PROCESSOR] Generating contextual response for user query: %s", convContext.UserQuery)

	var response strings.Builder
	response.WriteString(baseResult)

	// Note: We extract metadata and store it in convContext.ExtractedMetadata
	// but we DON'T show it in the user response. The metadata is available
	// in conversation history for the model to reference when needed.
	// This keeps responses clean while maintaining context for follow-up queries.

	// Add contextual follow-up based on conversation history and result type
	followUp := p.generateFollowUpSuggestions(baseResult, convContext)
	if followUp != "" {
		response.WriteString("\n\n")
		response.WriteString(followUp)
	}

	return response.String()
}

// generateFollowUpSuggestions provides intelligent follow-up suggestions based on context
func (p *ToolResultProcessor) generateFollowUpSuggestions(result string, convContext *model.ConversationContext) string {
	// Analyze the result and conversation to suggest relevant follow-ups
	queryLower := strings.ToLower(convContext.UserQuery)

	var suggestions []string

	// Search result follow-ups
	if strings.Contains(result, "I found") && strings.Contains(result, "memor") {
		// This is a search result
		if !p.hasRecentToolUsage(convContext.PreviousTools, "store_memory") {
			suggestions = append(suggestions, "💡 Would you like me to store any new insights from this search?")
		}
		if strings.Contains(queryLower, "relate") || strings.Contains(queryLower, "connect") {
			suggestions = append(suggestions, "🔗 I can also show you relationships between these memories.")
		}
		if len(convContext.History) > 4 { // Longer conversation
			suggestions = append(suggestions, "📊 Want me to analyze patterns across your memories?")
		}
	}

	// Storage result follow-ups
	if strings.Contains(result, "stored") && strings.Contains(result, "memory") {
		suggestions = append(suggestions, "🔍 You can search for this memory later or find related ones.")
		if p.hasRecentSearches(convContext.History) {
			suggestions = append(suggestions, "🔗 I can connect this to your recent searches if helpful.")
		}
	}

	// Analysis result follow-ups
	if strings.Contains(result, "pattern") || strings.Contains(result, "analys") {
		suggestions = append(suggestions, "💾 Would you like me to remember these insights for future reference?")
	}

	// Context-aware suggestions based on conversation flow
	if len(convContext.History) > 0 {
		lastMessage := convContext.History[len(convContext.History)-1]
		if lastMessage.Role == "user" && strings.Contains(strings.ToLower(lastMessage.Content), "help") {
			suggestions = append(suggestions, "ℹ️ Need more specific guidance? Just ask!")
		}
	}

	// Limit to 2 suggestions to avoid overwhelming
	if len(suggestions) > 2 {
		suggestions = suggestions[:2]
	}

	if len(suggestions) > 0 {
		return strings.Join(suggestions, "\n")
	}

	return ""
}

// hasRecentToolUsage checks if a tool was used recently in the conversation
func (p *ToolResultProcessor) hasRecentToolUsage(previousTools []string, toolName string) bool {
	for _, tool := range previousTools {
		if strings.Contains(strings.ToLower(tool), strings.ToLower(toolName)) {
			return true
		}
	}
	return false
}

// generateMetadataContext creates a natural language description of extracted metadata
// This makes important IDs and values visible to the model for follow-up requests
func (p *ToolResultProcessor) generateMetadataContext(convContext *model.ConversationContext) string {
	if convContext == nil || len(convContext.ExtractedMetadata) == 0 {
		return ""
	}

	p.logf("[METADATA-CONTEXT] Generating context from %d metadata fields", len(convContext.ExtractedMetadata))

	var contextParts []string

	// Memory ID is the most important for follow-up
	if memoryID, exists := convContext.ExtractedMetadata["memory_id"]; exists {
		contextParts = append(contextParts, fmt.Sprintf("(Memory ID: %v)", memoryID))
		p.logf("[METADATA-CONTEXT] Including memory_id: %v", memoryID)
	}

	// Also check for generic ID field
	if id, exists := convContext.ExtractedMetadata["id"]; exists {
		if _, hasMemoryID := convContext.ExtractedMetadata["memory_id"]; !hasMemoryID {
			contextParts = append(contextParts, fmt.Sprintf("(ID: %v)", id))
			p.logf("[METADATA-CONTEXT] Including id: %v", id)
		}
	}

	// Category and domain for context
	if categoryID, exists := convContext.ExtractedMetadata["category_id"]; exists {
		contextParts = append(contextParts, fmt.Sprintf("Category: %v", categoryID))
	}
	if domain, exists := convContext.ExtractedMetadata["domain"]; exists {
		contextParts = append(contextParts, fmt.Sprintf("Domain: %v", domain))
	}

	// First result ID from searches
	if firstMemoryID, exists := convContext.ExtractedMetadata["first_memory_id"]; exists {
		contextParts = append(contextParts, fmt.Sprintf("(First result ID: %v)", firstMemoryID))
		p.logf("[METADATA-CONTEXT] Including first_memory_id: %v", firstMemoryID)
	} else if firstID, exists := convContext.ExtractedMetadata["first_id"]; exists {
		contextParts = append(contextParts, fmt.Sprintf("(First result ID: %v)", firstID))
		p.logf("[METADATA-CONTEXT] Including first_id: %v", firstID)
	}

	if len(contextParts) > 0 {
		result := strings.Join(contextParts, " • ")
		p.logf("[METADATA-CONTEXT] Generated context: %s", result)
		return result
	}

	return ""
}

// hasRecentSearches checks if the user has performed searches recently
func (p *ToolResultProcessor) hasRecentSearches(history []model.Message) bool {
	// Look at the last few messages for search-related activity
	searchTerms := []string{"search", "find", "look", "show"}
	for i := len(history) - 1; i >= 0 && i >= len(history)-4; i-- {
		if history[i].Role == "user" {
			content := strings.ToLower(history[i].Content)
			for _, term := range searchTerms {
				if strings.Contains(content, term) {
					return true
				}
			}
		}
	}
	return false
}

// extractAndStoreMetadata extracts important metadata from tool results
// This makes metadata like memory_id, category_id available for follow-up requests
func (p *ToolResultProcessor) extractAndStoreMetadata(rawResult interface{}, convContext *model.ConversationContext) {
	if convContext == nil {
		p.logf("[METADATA-DEBUG] ConvContext is NIL, cannot extract metadata")
		return
	}

	p.logf("[METADATA-DEBUG] ConvContext pointer: %p, current metadata fields: %d", convContext, len(convContext.ExtractedMetadata))

	// Initialize metadata map if needed
	if convContext.ExtractedMetadata == nil {
		convContext.ExtractedMetadata = make(map[string]interface{})
		p.logf("[METADATA-DEBUG] Initialized ExtractedMetadata map")
	}

	// Try to extract metadata from MCP ToolResult format
	if toolResult, ok := rawResult.(*mcp.ToolResult); ok {
		p.logf("[METADATA-DEBUG] Raw result is MCP ToolResult, extracting...")
		p.extractMetadataFromMCPResult(toolResult, convContext)
		p.logf("[METADATA-DEBUG] After MCP extraction, metadata fields: %d", len(convContext.ExtractedMetadata))
		return
	}

	// Try to extract from map format
	if resultMap, ok := rawResult.(map[string]interface{}); ok {
		p.logf("[METADATA-DEBUG] Raw result is map[string]interface{}, extracting...")
		p.extractMetadataFromMap(resultMap, convContext)
		p.logf("[METADATA-DEBUG] After map extraction, metadata fields: %d", len(convContext.ExtractedMetadata))
		return
	}

	p.logf("[METADATA] Unable to extract metadata from result type: %T", rawResult)
}

// extractMetadataFromMCPResult extracts metadata from MCP ToolResult
func (p *ToolResultProcessor) extractMetadataFromMCPResult(toolResult *mcp.ToolResult, convContext *model.ConversationContext) {
	p.logf("[METADATA-MCP] Extracting from MCP ToolResult with %d content items", len(toolResult.Content))
	
	// MCP results have content array - try to parse JSON from text content
	for i, content := range toolResult.Content {
		p.logf("[METADATA-MCP] Content[%d]: type=%s, text_len=%d", i, content.Type, len(content.Text))
		
		if content.Type == "text" && content.Text != "" {
			trimmed := strings.TrimSpace(content.Text)
			p.logf("[METADATA-MCP] Trimmed text preview (first 200 chars): %s", truncateString(trimmed, 200))
			
			// First, try to parse as JSON for structured responses
			if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			   (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
				p.logf("[METADATA-MCP] Text looks like JSON, attempting to parse...")
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
					p.logf("[METADATA-MCP] Successfully parsed JSON with %d top-level keys", len(parsed))
					p.extractMetadataFromMap(parsed, convContext)
					return
				} else {
					p.logf("[METADATA-MCP] Failed to parse JSON: %v", err)
				}
			}
			
			// If not JSON, try regex extraction for common patterns
			p.logf("[METADATA-MCP] Attempting regex-based extraction from human-readable text...")
			extracted := p.extractMetadataWithRegex(trimmed, convContext)
			if extracted > 0 {
				p.logf("[METADATA-MCP] Extracted %d metadata fields using regex", extracted)
				return
			}
		}
	}
	p.logf("[METADATA-MCP] No extractable metadata found in MCP ToolResult")
}

// extractMetadataWithRegex extracts metadata from human-readable text using regex patterns
func (p *ToolResultProcessor) extractMetadataWithRegex(text string, convContext *model.ConversationContext) int {
	extracted := 0
	
	// Common patterns for IDs and metadata in human-readable text
	patterns := map[string]*regexp.Regexp{
		// Match "ID: <uuid>" or "with ID: <uuid>" or "memory_id: <uuid>"
		"memory_id": regexp.MustCompile(`(?i)(?:memory[_\s-]?)?(?:with\s+)?ID:\s*([a-f0-9\-]{36})`),
		// Match "document ID: <uuid>" or "document_id: <uuid>"
		"document_id": regexp.MustCompile(`(?i)document[_\s-]?ID:\s*([a-f0-9\-]{36})`),
		// Match "session ID: <uuid>" or "session_id: <uuid>"
		"session_id": regexp.MustCompile(`(?i)session[_\s-]?ID:\s*([a-f0-9\-]{36})`),
		// Match "category ID: <uuid>" or "category_id: <uuid>"
		"category_id": regexp.MustCompile(`(?i)category[_\s-]?ID:\s*([a-f0-9\-]{36})`),
		// Match "total: <number>" or "count: <number>"
		"total": regexp.MustCompile(`(?i)(?:total|count):\s*(\d+)`),
	}
	
	for key, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(text); len(matches) > 1 {
			value := matches[1]
			// Skip if already exists
			if _, exists := convContext.ExtractedMetadata[key]; exists {
				continue
			}
			convContext.ExtractedMetadata[key] = value
			extracted++
			p.logf("[METADATA-REGEX] Extracted %s = %v", key, value)
		}
	}
	
	return extracted
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractMetadataFromMap extracts metadata from a map result
func (p *ToolResultProcessor) extractMetadataFromMap(resultMap map[string]interface{}, convContext *model.ConversationContext) {
	// Priority metadata keys to extract (these are most useful for follow-up requests)
	priorityKeys := []string{
		"memory_id", "id",
		"category_id", "domain", "session_id", "relationship_id",
		"count", "total", "memory_count", "domain_count", "category_count",
	}

	extracted := 0
	
	// Extract priority keys first
	for _, key := range priorityKeys {
		if value, exists := resultMap[key]; exists && value != nil {
			convContext.ExtractedMetadata[key] = value
			extracted++
			p.logf("[METADATA] Extracted %s = %v", key, value)
		}
	}

	// Then extract any other ID-like fields (universal approach for any MCP server)
	for key, value := range resultMap {
		if value == nil {
			continue
		}
		
		// Skip if already extracted
		if _, exists := convContext.ExtractedMetadata[key]; exists {
			continue
		}
		
		// Extract fields that look like identifiers or important metadata
		keyLower := strings.ToLower(key)
		if strings.HasSuffix(keyLower, "_id") || 
		   strings.HasSuffix(keyLower, "id") || 
		   strings.HasSuffix(keyLower, "_uuid") ||
		   strings.HasSuffix(keyLower, "_key") ||
		   strings.HasSuffix(keyLower, "_ref") ||
		   strings.HasSuffix(keyLower, "_handle") ||
		   strings.HasSuffix(keyLower, "_type") ||
		   keyLower == "name" ||
		   keyLower == "type" ||
		   keyLower == "status" {
			// Only extract simple types (strings, numbers, bools)
			switch value.(type) {
			case string, int, int64, float64, bool:
				convContext.ExtractedMetadata[key] = value
				extracted++
				p.logf("[METADATA] Extracted %s = %v (identifier-like field)", key, value)
			}
		}
	}

	// Also check nested results array for metadata
	if results, ok := resultMap["results"].([]interface{}); ok && len(results) > 0 {
		// Extract IDs from the first result
		if firstResult, ok := results[0].(map[string]interface{}); ok {
			// Extract priority keys with "first_" prefix
			for _, key := range priorityKeys {
				if value, exists := firstResult[key]; exists && value != nil {
					prefixedKey := "first_" + key
					convContext.ExtractedMetadata[prefixedKey] = value
					extracted++
					p.logf("[METADATA] Extracted %s = %v", prefixedKey, value)
				}
			}
			
			// Extract other ID-like fields from first result
			for key, value := range firstResult {
				if value == nil {
					continue
				}
				
				prefixedKey := "first_" + key
				if _, exists := convContext.ExtractedMetadata[prefixedKey]; exists {
					continue
				}
				
				// Apply the same universal extraction logic as the main loop
				keyLower := strings.ToLower(key)
				if strings.HasSuffix(keyLower, "_id") || 
				   strings.HasSuffix(keyLower, "id") || 
				   strings.HasSuffix(keyLower, "_uuid") ||
				   strings.HasSuffix(keyLower, "_key") ||
				   strings.HasSuffix(keyLower, "_ref") ||
				   strings.HasSuffix(keyLower, "_handle") ||
				   strings.HasSuffix(keyLower, "_type") ||
				   keyLower == "name" ||
				   keyLower == "type" ||
				   keyLower == "status" {
					switch value.(type) {
					case string, int, int64, float64, bool:
						convContext.ExtractedMetadata[prefixedKey] = value
						extracted++
						p.logf("[METADATA] Extracted %s = %v (from first result)", prefixedKey, value)
					}
				}
			}
		}
	}

	if extracted > 0 {
		p.logf("[METADATA] Successfully extracted %d metadata fields", extracted)
	}
}
