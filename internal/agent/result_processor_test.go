package agent

import (
	"context"
	"testing"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
	"github.com/danieleugenewilliams/othello-agent/internal/model"
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
	assert.Contains(t, processed, "found", "Should indicate results were found")
	assert.Contains(t, processed, "memor", "Should mention memories")
	
	// Should NOT contain raw JSON structures
	assert.NotContains(t, processed, "mem123", "Should not include internal IDs")
	assert.NotContains(t, processed, "score", "Should not include technical details")
	
	// Should be reasonably concise (not more than 10x a single content length including formatting)
	singleContentLen := len(rawResult["results"].([]interface{})[0].(map[string]interface{})["content"].(string))
	assert.Less(t, len(processed), singleContentLen*10,
		"Processed result should be reasonably concise")
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

// TestMetadataExtraction_MemoryID tests extraction of memory_id from store_memory results
func TestMetadataExtraction_MemoryID(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"success":   true,
		"memory_id": "550e8400-e29b-41d4-a716-446655440000",
		"message":   "Memory stored successfully",
	}
	
	convContext := &model.ConversationContext{
		UserQuery:         "Store this information",
		SessionType:       "chat",
		ExtractedMetadata: make(map[string]interface{}),
	}
	
	_, err := processor.ProcessToolResultWithContext(context.Background(), "store_memory", rawResult, convContext)
	require.NoError(t, err)
	
	// Should extract memory_id into context
	assert.Contains(t, convContext.ExtractedMetadata, "memory_id", "Should extract memory_id")
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", convContext.ExtractedMetadata["memory_id"])
}

// TestMetadataExtraction_SearchResults tests extraction of ID from first search result
func TestMetadataExtraction_SearchResults(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"id":         "first-result-id",
				"memory_id":  "mem-123",
				"content":    "Some content",
				"importance": 8.0,
			},
			map[string]interface{}{
				"id":      "second-result-id",
				"content": "More content",
			},
		},
		"total": 2,
	}
	
	convContext := &model.ConversationContext{
		UserQuery:         "Find related memories",
		SessionType:       "chat",
		ExtractedMetadata: make(map[string]interface{}),
	}
	
	_, err := processor.ProcessToolResultWithContext(context.Background(), "search", rawResult, convContext)
	require.NoError(t, err)
	
	// Should extract first result's IDs
	assert.Contains(t, convContext.ExtractedMetadata, "first_id", "Should extract first result ID")
	assert.Equal(t, "first-result-id", convContext.ExtractedMetadata["first_id"])
	assert.Contains(t, convContext.ExtractedMetadata, "first_memory_id", "Should extract first result memory_id")
	assert.Equal(t, "mem-123", convContext.ExtractedMetadata["first_memory_id"])
}

// TestMetadataExtraction_JSONText tests extraction from JSON embedded in text content
func TestMetadataExtraction_JSONText(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	// Simulate MCP ToolResult with JSON in text content
	rawResult := &mcp.ToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: `{"success": true, "memory_id": "extracted-from-json", "count": 5}`,
			},
		},
	}
	
	convContext := &model.ConversationContext{
		UserQuery:         "Store memory",
		SessionType:       "chat",
		ExtractedMetadata: make(map[string]interface{}),
	}
	
	_, err := processor.ProcessToolResultWithContext(context.Background(), "store_memory", rawResult, convContext)
	require.NoError(t, err)
	
	// Should parse JSON from text and extract metadata
	assert.Contains(t, convContext.ExtractedMetadata, "memory_id", "Should extract memory_id from JSON text")
	assert.Equal(t, "extracted-from-json", convContext.ExtractedMetadata["memory_id"])
	assert.Contains(t, convContext.ExtractedMetadata, "count", "Should extract count from JSON text")
}

// TestMetadataContext_Generation tests that metadata is extracted into context
func TestMetadataContext_Generation(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	rawResult := map[string]interface{}{
		"success":   true,
		"memory_id": "uuid-12345",
	}
	
	convContext := &model.ConversationContext{
		UserQuery:         "Store this",
		SessionType:       "chat",
		ExtractedMetadata: make(map[string]interface{}),
	}
	
	processed, err := processor.ProcessToolResultWithContext(context.Background(), "store_memory", rawResult, convContext)
	require.NoError(t, err)
	
	// Verify response is user-friendly (not raw data)
	assert.Contains(t, processed, "stored", "Should confirm storage")
	assert.NotEmpty(t, processed, "Should return a response")
	
	// Verify metadata was extracted into context for model to use
	assert.Equal(t, "uuid-12345", convContext.ExtractedMetadata["memory_id"], "Metadata should be extracted into context")
}

// TestMetadataContext_Accumulation tests metadata accumulates across multiple tool calls
func TestMetadataContext_Accumulation(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	convContext := &model.ConversationContext{
		UserQuery:         "First query",
		SessionType:       "chat",
		ExtractedMetadata: make(map[string]interface{}),
	}
	
	// First tool call - store memory
	result1 := map[string]interface{}{
		"success":   true,
		"memory_id": "mem-001",
	}
	_, err := processor.ProcessToolResultWithContext(context.Background(), "store_memory", result1, convContext)
	require.NoError(t, err)
	assert.Equal(t, "mem-001", convContext.ExtractedMetadata["memory_id"])
	
	// Second tool call - get stats
	result2 := map[string]interface{}{
		"memory_count": 42,
		"domain":       "programming",
	}
	_, err = processor.ProcessToolResultWithContext(context.Background(), "stats", result2, convContext)
	require.NoError(t, err)
	
	// Both metadata should be present
	assert.Equal(t, "mem-001", convContext.ExtractedMetadata["memory_id"], "Previous metadata should persist")
	assert.Equal(t, 42, convContext.ExtractedMetadata["memory_count"], "New metadata should be added")
	assert.Equal(t, "programming", convContext.ExtractedMetadata["domain"], "Domain should be extracted")
}

// TestMetadataExtraction_UniversalMCPServer tests metadata extraction works with arbitrary MCP servers
func TestMetadataExtraction_UniversalMCPServer(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	// Simulate a custom MCP server with non-standard field names
	rawResult := map[string]interface{}{
		"success":      true,
		"document_id":  "doc-12345",                    // Custom ID field
		"artifact_key": "artifact-xyz",                 // Custom key field
		"entity_uuid":  "550e8400-e29b-41d4-a716-446655440000", // UUID field
		"record_ref":   "ref-999",                      // Reference field
		"status":       "completed",                    // Status field
		"name":         "important-document",           // Name field
		"created_at":   "2024-01-01T00:00:00Z",        // Timestamp (should not be extracted)
		"metadata": map[string]interface{}{             // Nested object (should not be extracted)
			"author": "user",
		},
	}
	
	convContext := &model.ConversationContext{
		UserQuery:         "Store this document",
		SessionType:       "chat",
		ExtractedMetadata: make(map[string]interface{}),
	}
	
	_, err := processor.ProcessToolResultWithContext(context.Background(), "custom_tool", rawResult, convContext)
	require.NoError(t, err)
	
	// Verify ID-like fields were extracted
	assert.Equal(t, "doc-12345", convContext.ExtractedMetadata["document_id"], "Should extract custom _id fields")
	assert.Equal(t, "artifact-xyz", convContext.ExtractedMetadata["artifact_key"], "Should extract _key fields")
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", convContext.ExtractedMetadata["entity_uuid"], "Should extract _uuid fields")
	assert.Equal(t, "ref-999", convContext.ExtractedMetadata["record_ref"], "Should extract _ref fields")
	assert.Equal(t, "completed", convContext.ExtractedMetadata["status"], "Should extract status fields")
	assert.Equal(t, "important-document", convContext.ExtractedMetadata["name"], "Should extract name fields")
	
	// Verify complex types were NOT extracted
	assert.NotContains(t, convContext.ExtractedMetadata, "metadata", "Should not extract nested objects")
	assert.NotContains(t, convContext.ExtractedMetadata, "created_at", "Should not extract timestamp strings")
	
	t.Logf("Extracted %d metadata fields: %+v", len(convContext.ExtractedMetadata), convContext.ExtractedMetadata)
}

// TestMetadataExtraction_CustomResults tests extraction from custom result structures
func TestMetadataExtraction_CustomResults(t *testing.T) {
	processor := &ToolResultProcessor{}
	
	// Simulate a custom MCP server with non-standard result structure
	rawResult := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"item_id":      "item-001",
				"resource_key": "res-alpha",
				"type":         "document",
			},
			map[string]interface{}{
				"item_id":      "item-002",
				"resource_key": "res-beta",
				"type":         "image",
			},
		},
		"total": 2,
	}
	
	convContext := &model.ConversationContext{
		UserQuery:         "Find resources",
		SessionType:       "chat",
		ExtractedMetadata: make(map[string]interface{}),
	}
	
	_, err := processor.ProcessToolResultWithContext(context.Background(), "custom_search", rawResult, convContext)
	require.NoError(t, err)
	
	// Verify top-level metadata
	assert.Equal(t, 2, convContext.ExtractedMetadata["total"], "Should extract total count")
	
	// Verify first result metadata with prefix
	assert.Equal(t, "item-001", convContext.ExtractedMetadata["first_item_id"], "Should extract custom ID from first result")
	assert.Equal(t, "res-alpha", convContext.ExtractedMetadata["first_resource_key"], "Should extract custom key from first result")
	assert.Equal(t, "document", convContext.ExtractedMetadata["first_type"], "Should extract type from first result")
	
	t.Logf("Extracted %d metadata fields from custom results: %+v", len(convContext.ExtractedMetadata), convContext.ExtractedMetadata)
}
