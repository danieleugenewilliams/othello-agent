package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolResultProcessor processes raw tool results into user-friendly summaries
type ToolResultProcessor struct {
	// Can add configuration here later (e.g., verbosity level)
}

// ProcessToolResult takes a raw tool result and formats it for user consumption
func (p *ToolResultProcessor) ProcessToolResult(ctx context.Context, toolName string, rawResult interface{}, userQuery string) (string, error) {
	// Handle nil result
	if rawResult == nil {
		return "The tool returned no results for your query.", nil
	}
	
	// Convert to map for easier processing
	resultMap, ok := rawResult.(map[string]interface{})
	if !ok {
		// If not a map, try to stringify intelligently
		return p.formatGenericResult(rawResult), nil
	}
	
	// Check for error in result
	if errMsg, hasError := p.checkForError(resultMap); hasError {
		return errMsg, nil
	}
	
	// Process based on tool type
	switch toolName {
	case "search":
		return p.processSearchResults(resultMap, userQuery), nil
	case "store_memory":
		return p.processStoreMemoryResult(resultMap), nil
	case "analysis":
		return p.processAnalysisResult(resultMap), nil
	case "stats":
		return p.processStatsResult(resultMap), nil
	case "relationships":
		return p.processRelationshipsResult(resultMap), nil
	case "domains", "categories", "sessions":
		return p.processListResult(resultMap, toolName), nil
	default:
		return p.formatGenericResult(resultMap), nil
	}
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
	results, ok := result["results"].([]interface{})
	if !ok || len(results) == 0 {
		return "I didn't find any memories matching your search."
	}
	
	var summaries []string
	for i, r := range results {
		if i >= 5 { // Limit to 5 results for conciseness
			summaries = append(summaries, fmt.Sprintf("...and %d more results", len(results)-i))
			break
		}
		
		resultMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		
		content, ok := resultMap["content"].(string)
		if !ok {
			continue
		}
		
		// Truncate long content
		if len(content) > 150 {
			content = content[:147] + "..."
		}
		
		summaries = append(summaries, fmt.Sprintf("• %s", content))
	}
	
	if len(summaries) == 0 {
		return "I found some results but couldn't extract the content."
	}
	
	count := len(results)
	header := fmt.Sprintf("I found %d relevant memor", count)
	if count == 1 {
		header += "y:\n\n"
	} else {
		header += "ies:\n\n"
	}
	
	return header + strings.Join(summaries, "\n")
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

// formatGenericResult provides a fallback for unknown result types
func (p *ToolResultProcessor) formatGenericResult(result interface{}) string {
	// Try to marshal to JSON and extract key information
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "The tool completed successfully."
	}
	
	jsonStr := string(jsonBytes)
	
	// If the JSON is small enough, show a cleaned version
	if len(jsonStr) < 200 {
		// Remove technical fields
		jsonStr = strings.ReplaceAll(jsonStr, `"id":`, ``)
		jsonStr = strings.ReplaceAll(jsonStr, `"timestamp":`, ``)
		return "Result: " + jsonStr
	}
	
	return "The tool completed successfully. Results are available."
}
