package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessToolResult_SearchResults tests processing of search tool results
func TestProcessToolResult_SearchResults(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	// Simulate raw search result from MCP server
	rawResult := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"id":      "mem123",
				"content": "Python uses list comprehensions for concise iteration",
				"tags":    []interface{}{"python", "programming"},
				"score":   0.95,
			},
			map[string]interface{}{
				"id":      "mem456",
				"content": "API design best practices: use RESTful endpoints",
				"tags":    []interface{}{"api", "rest"},
				"score":   0.87,
			},
		},
		"total_count": 2,
	}
	
	userQuery := "What are best practices for Python?"
	
	processed, err := processor.ProcessToolResult(context.Background(), "search", rawResult, userQuery)
	require.NoError(t, err)
	
	// Should extract key information
	assert.Contains(t, processed, "Python", "Should mention Python")
	assert.Contains(t, processed, "list comprehensions", "Should extract key info")
	
	// Should NOT contain raw JSON structures
	assert.NotContains(t, processed, "mem123", "Should not include internal IDs")
	assert.NotContains(t, processed, "score", "Should not include technical details")
	
	// Should be concise
	assert.Less(t, len(processed), len(rawResult["results"].([]interface{})[0].(map[string]interface{})["content"].(string))*3,
		"Processed result should be concise, not verbose")
}

// TestProcessToolResult_EmptyResults tests handling of empty search results
func TestProcessToolResult_EmptyResults(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"results":     []interface{}{},
		"total_count": 0,
	}
	
	processed, err := processor.ProcessToolResult(context.Background(), "search", rawResult, "test query")
	require.NoError(t, err)
	
	// Should indicate no results found
	assert.Contains(t, processed, "didn't find", "Should indicate no results")
	assert.Contains(t, processed, "memories", "Should mention memories")
}

// TestProcessToolResult_StoreMemory tests processing of store_memory results
func TestProcessToolResult_StoreMemory(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"success":   true,
		"memory_id": "mem789",
		"message":   "Memory stored successfully",
	}
	
	userQuery := "Store this information about Go programming"
	
	processed, err := processor.ProcessToolResult(context.Background(), "store_memory", rawResult, userQuery)
	require.NoError(t, err)
	
	// Should confirm action without technical details
	assert.Contains(t, processed, "stored", "Should confirm storage")
	assert.NotContains(t, processed, "mem789", "Should not include memory ID")
}

// TestProcessToolResult_AnalysisResults tests processing of analysis tool results
func TestProcessToolResult_AnalysisResults(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"analysis_type": "question",
		"answer":        "The key differences are: supervised learning uses labeled data, unsupervised learning finds patterns in unlabeled data.",
		"confidence":    0.92,
		"sources":       []interface{}{"mem1", "mem2", "mem3"},
	}
	
	userQuery := "What are the differences between supervised and unsupervised learning?"
	
	processed, err := processor.ProcessToolResult(context.Background(), "analysis", rawResult, userQuery)
	require.NoError(t, err)
	
	// Should extract the answer
	assert.Contains(t, processed, "supervised learning", "Should include answer content")
	assert.Contains(t, processed, "labeled data", "Should include key concepts")
	
	// Should not include technical metadata
	assert.NotContains(t, processed, "confidence", "Should not include confidence scores")
	assert.NotContains(t, processed, "mem1", "Should not include source IDs")
}

// TestProcessToolResult_ErrorResult tests handling of error results
func TestProcessToolResult_ErrorResult(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"error":   true,
		"message": "Database connection failed",
	}
	
	processed, err := processor.ProcessToolResult(context.Background(), "search", rawResult, "test query")
	require.NoError(t, err)
	
	// Should convey error in user-friendly way
	assert.Contains(t, processed, "unable", "Should indicate failure")
	assert.NotContains(t, processed, "Database connection", "Should not expose technical errors")
}

// TestProcessToolResult_Stats tests processing of statistics results
func TestProcessToolResult_Stats(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"stats_type":    "session",
		"memory_count":  42,
		"domain_count":  5,
		"category_count": 8,
	}
	
	processed, err := processor.ProcessToolResult(context.Background(), "stats", rawResult, "How many memories do I have?")
	require.NoError(t, err)
	
	// Should present stats clearly
	assert.Contains(t, processed, "42", "Should include memory count")
	assert.Contains(t, processed, "memories", "Should use natural language")
	
	// Should be conversational
	assert.NotContains(t, processed, "memory_count", "Should not use technical field names")
}

// TestProcessToolResult_UnknownTool tests fallback for unknown tool types
func TestProcessToolResult_UnknownTool(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"some_field": "some value",
	}
	
	processed, err := processor.ProcessToolResult(context.Background(), "unknown_tool", rawResult, "test query")
	require.NoError(t, err)
	
	// Should provide reasonable default formatting
	assert.NotEmpty(t, processed, "Should return non-empty result")
	assert.NotContains(t, processed, "map[", "Should not dump Go map structure")
}

// TestProcessToolResult_ComplexNestedData tests handling of deeply nested structures
func TestProcessToolResult_ComplexNestedData(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"content": "Main content",
				"metadata": map[string]interface{}{
					"created_at": "2024-01-01",
					"author":     "system",
					"tags": []interface{}{
						map[string]interface{}{"name": "tag1", "category": "cat1"},
						map[string]interface{}{"name": "tag2", "category": "cat2"},
					},
				},
			},
		},
	}
	
	processed, err := processor.ProcessToolResult(context.Background(), "search", rawResult, "test")
	require.NoError(t, err)
	
	// Should extract only relevant information
	assert.Contains(t, processed, "Main content", "Should include main content")
	
	// Should not overwhelm with nested structures
	maxDepth := 0
	for _, char := range processed {
		if char == '{' || char == '[' {
			maxDepth++
		}
	}
	assert.Less(t, maxDepth, 3, "Should not have deeply nested structures in output")
}

// TestProcessToolResult_NilResult tests handling of nil results
func TestProcessToolResult_NilResult(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	processed, err := processor.ProcessToolResult(context.Background(), "search", nil, "test query")
	require.NoError(t, err)
	
	// Should handle gracefully
	assert.NotEmpty(t, processed, "Should return non-empty message")
	assert.Contains(t, processed, "no results", "Should indicate no results")
}
