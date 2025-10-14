package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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
func (p *ToolResultProcessor) ProcessToolResult(ctx context.Context, toolName string, rawResult interface{}, userQuery string) (string, error) {
	p.logf("[PROCESSOR] Processing MCP tool result for: '%s'", toolName)
	p.logf("[PROCESSOR] Raw result type: %T", rawResult)

	// Handle nil result
	if rawResult == nil {
		p.logf("[PROCESSOR] Raw result is nil")
		return "The tool returned no results.", nil
	}

	// The rawResult should be a ToolResult from the MCP server
	// Try to extract it as a ToolResult struct or map representation
	if toolResult := p.extractMCPToolResult(rawResult); toolResult != nil {
		p.logf("[PROCESSOR] Successfully extracted MCP ToolResult with %d content items", 0)
		return p.formatMCPContent(toolResult), nil
	}

	// Fallback: treat as raw content if not in MCP ToolResult format
	p.logf("[PROCESSOR] Not an MCP ToolResult format, using fallback presentation")
	return p.formatFallbackContent(rawResult), nil
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
	if results, hasResults := result["results"].([]interface{}); hasResults && len(results) > 0 {
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
	// Check if it has the MCP ToolResult structure
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

	contentArray, ok := contentField.([]interface{})
	if !ok {
		p.logf("[FORMAT] Content is not an array")
		return "Invalid content format"
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
			// Text content: display as-is (server has already formatted it)
			if contentText != "" {
				output.WriteString(contentText)
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
